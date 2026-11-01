#!/usr/bin/env python3
"""Drive the actual external consumer executable and the separate controlled ledger."""
import argparse
import contextlib
import fcntl
import gzip
import hashlib
import json
import os
from pathlib import Path
import select
import signal
import sqlite3
import subprocess
import sys
import tempfile
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def write(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def readline(file):
    if not select.select([file], [], [], 30)[0]:
        raise RuntimeError("process acknowledgment timeout")
    return json.loads(file.readline())


def archive(db, path):
    with sqlite3.connect(f"file:{db}?mode=ro", uri=True) as src, sqlite3.connect(path) as dst:
        src.backup(dst)
    path.with_suffix(".sqlite.gz").write_bytes(gzip.compress(path.read_bytes(), mtime=0))
    path.unlink()


def run_case(name, out, work, consumer, resource, builds):
    d = out / name
    d.mkdir()
    wd = work / name
    wd.mkdir()
    db, world = wd / "retained.sqlite", wd / "world.sqlite"
    old = name == "immutable"
    held = not old
    adopt = name != "read-only"
    records = []
    log = d / "journal.jsonl"
    endpoint_args = [str(resource), "-receipt-ledger", "-db", str(world), "-token", "public-"+name, "-journal", str(log),
                     "-response-mode", "drop-after-commit" if old else "normal"]
    records.append({"argv": endpoint_args, "cwd": str(HERE), "sha256": digest(resource)})
    with contextlib.ExitStack() as stack:
        env = os.environ.copy()
        if held:
            ackr, ackw = os.pipe()
            relr, relw = os.pipe()
            # Allocate non-overlapping source fds before the child maps them to 3/4.
            aw = fcntl.fcntl(ackw, fcntl.F_DUPFD, 10)
            rr = fcntl.fcntl(relr, fcntl.F_DUPFD, 10)
            for fd in (ackw, relr, aw, rr, relw):
                stack.callback(os.close, fd)
            ack = stack.enter_context(os.fdopen(ackr, "rb"))
            env["ALAN_BARRIER"] = json.dumps({"Token": name, "Scenario": "public-qualification", "Store": str(world),
                                            "Boundary": "resource.applied", "Phase": "post-commit"})
            wrapper = "import os,sys;os.dup2(int(sys.argv[1]),3);os.dup2(int(sys.argv[2]),4);os.execv(sys.argv[3],sys.argv[3:])"
            launch = [sys.executable, "-c", wrapper, str(aw), str(rr)] + endpoint_args
            extra = (aw, rr)
        else:
            launch, extra = endpoint_args, ()
        ep = subprocess.Popen(launch, cwd=HERE, env=env, pass_fds=extra, stdout=subprocess.PIPE)
        def stop(proc):
            if proc.poll() is None:
                proc.kill()
                proc.wait(10)
        stack.callback(stop, ep)
        ready = readline(ep.stdout)
        assert ready["PID"] == ep.pid and ready["Store"] == str(world)
        records[-1].update(ready=ready, pid=ep.pid)
        def cmd(op, *args, async_=False):
            argv = [str(consumer), op, "-db", str(db)]
            if op != "replay":
                argv += ["-endpoint", ready["URL"]]
            argv += list(args)
            child = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE, cwd=HERE)
            stack.callback(stop, child)
            record = {"argv": argv, "pid": child.pid, "cwd": str(HERE), "sha256": digest(consumer)}
            records.append(record)
            if async_:
                return child, record
            stdout, stderr = child.communicate(timeout=60)
            record.update(exit=child.returncode, stderr=stderr.decode(), stdout=stdout.decode())
            write(d/"commands.json", records)
            if child.returncode:
                raise RuntimeError(f"{op} failed: {stderr.decode()}")
            return json.loads(stdout)
        cmd("init", "-adopt="+str(adopt).lower(), "-observe-original="+str(old).lower())
        if held:
            owner, identity = cmd("issue", async_=True)
            event = readline(ack)
            assert event["PID"] == ep.pid and event["Boundary"] == "resource.applied" and event["Phase"] == "post-commit"
            deadline = time.monotonic() + 12
            while True:
                with sqlite3.connect(f"file:{db}?mode=ro", uri=True) as sql:
                    row = sql.execute("SELECT projection FROM runs WHERE subject NOT IN ('parent','resolver')").fetchone()
                if row and json.loads(row[0])["Status"] == "succeeded":
                    break
                if time.monotonic() > deadline:
                    raise RuntimeError("issuer did not close while original response held")
                time.sleep(.01)
            owner.kill()
            stdout, stderr = owner.communicate(timeout=10)
            assert owner.returncode == -signal.SIGKILL
            identity.update(exit=owner.returncode, stderr=stderr.decode(), stdout=stdout.decode(), applied_barrier=event)
            os.write(relw, b'c')
        else:
            cmd("issue")
        archive(db, d/"before.sqlite")
        with sqlite3.connect(f"file:{db}?mode=ro", uri=True) as sql:
            source, = sql.execute("SELECT subject FROM runs WHERE subject NOT IN ('parent','resolver')").fetchone()
            original = json.dumps([source, 1, 2, 0], separators=(",",":"))
            def historical(conn):
                return [list(conn.execute(q,args)) for q,args in [
                    ("SELECT descriptor FROM adapter_results WHERE intent_id=?", (original,)),
                    ("SELECT descriptor FROM composition_freezes WHERE json_extract(descriptor,'$.Run.Subject')=?", (source,)),
                    ("SELECT descriptor FROM composition_publications", ()),
                    ("SELECT selected_observation,successor,intents FROM history WHERE subject=? ORDER BY sequence", (source,)),
                    ("SELECT sequence,projection FROM runs WHERE subject=?", (source,)),
                    ("SELECT descriptor,consumed_sequence FROM observations WHERE subject=? ORDER BY id", (source,)),
                    ("SELECT descriptor,consumed_sequence FROM observations WHERE id IN (SELECT observation_id FROM composition_outcome_deliveries)", ())]]
            prior = historical(sql)
            assert bool(prior[0]) == old
        res = cmd("resolve")
        write(d/"selected.json", res)
        vals = lambda who: res["states"][who]["Projection"]["Data"]
        parent, resolver = vals("parent"), vals("resolver")
        assert resolver["query1_status"]["Text"] == "found" and resolver["query1_target"]["Text"] == original
        assert resolver["query1_mutation"]["Number"] == 1 and resolver["query1_total"]["Number"] == 1
        assert resolver["query1_adoption"]["Text"] == ("existing" if old else "adopted" if adopt else "denied")
        if adopt:
            assert res["states"]["parent"]["Projection"]["Status"] == "succeeded" and parent["receipt_target"]["Text"] == original
            assert parent["decision"]["Text"] == ("applied_after_original_failure" if old else "applied")
        else:
            assert res["after"]["Pending"] == "external_outcome" and res["states"]["parent"]["Projection"]["Status"] == "open"
        if old:
            assert vals("source")["original_disposition"]["Text"] == "transport_error" and not vals("source")["original_valid"]["Flag"]
            assert not vals("source")["original_mutation"]["Number"] and parent["source_status"]["Text"]=="failed"
            with sqlite3.connect(f"file:{db}?mode=ro", uri=True) as sql:
                assert prior == historical(sql) and all(prior), "original failure facts changed"
        elif adopt:
            assert res["before"]["Pending"] == "external_outcome" and res["after"]["Ready"]
        before = log.read_bytes()
        cmd("replay")
        assert log.read_bytes() == before
        ep.send_signal(signal.SIGTERM)
        assert ep.wait(timeout=10) == 0
        records[0]["exit"] = ep.returncode
        archive(db, d/"after.sqlite")
        archive(world, d/"world.sqlite")
        journal = [json.loads(line) for line in before.splitlines()]
        assert [r["Path"].rsplit("/",1)[-1] for r in journal if r["Event"]=="ingress"] == ["apply","lookup"]
        write(d/"commands.json", records)
        write(d/"result.json", {"builds":builds, "original":original, "adoption":resolver["query1_adoption"]["Text"], "parent":res["states"]["parent"]["Projection"]["Status"],"decision":parent.get("decision"), "contacts":2})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("regime", choices=["normal","race"])
    parser.add_argument("output", type=Path)
    parser.add_argument("cases", nargs="*", default=["positive","immutable","read-only"])
    args = parser.parse_args()
    args.output.mkdir(parents=True)
    with tempfile.TemporaryDirectory() as temp:
        work = Path(temp)
        binaries=[]
        builds=[]
        for name,package,cwd in [("consumer",".",HERE),("resource","./cmd/alan-resource",ROOT)]:
            binary=work/name
            argv=["go","build"] + (["-race"] if args.regime=="race" else []) + ["-o",str(binary),package]
            subprocess.run(argv,cwd=cwd,check=True)
            builds.append({"argv":argv,"cwd":str(cwd),"sha256":digest(binary),"version":subprocess.check_output(["go","version","-m",binary],text=True)})
            binaries.append(binary)
        for name in args.cases:
            if name not in {"positive","immutable","read-only"}:
                raise ValueError(name)
            run_case(name,args.output,work,*binaries,builds)
            print(f"{name}: verified",flush=True)


if __name__ == "__main__":
    main()
