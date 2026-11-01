"""Exercise two real public handoffs on one Store before either receipt query."""
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


def save(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def ack(stream):
    if not select.select([stream], [], [], 30)[0]:
        raise RuntimeError("missing process acknowledgment")
    return json.loads(stream.readline())


def stop(process):
    if process.poll() is None:
        process.kill()
        process.wait(timeout=10)


def backup(db, destination):
    with sqlite3.connect(f"file:{db}?mode=ro", uri=True) as source:
        with sqlite3.connect(":memory:") as copy:
            source.backup(copy)
            destination.write_bytes(gzip.compress(copy.serialize(), mtime=0))


def commands(mode, out, tmp):
    env = {k:os.environ[k] for k in ("PATH","HOME","TMPDIR","GOCACHE") if k in os.environ}
    env.update(GOTOOLCHAIN="local", CGO_ENABLED="1", GOPROXY="off", GOSUMDB="off", GOFLAGS="-mod=readonly", LC_ALL="C", PYTHONDONTWRITEBYTECODE="1")
    records = []
    def run(argv, directory, binary, **options):
        rec = {"argv":list(map(str,argv)),"cwd":str(directory),"executable_sha256":sha(binary),"env":env}
        records.append(rec)
        proc = subprocess.Popen(argv, cwd=directory, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, **options)
        rec["pid"] = proc.pid
        save(out/"commands.json",records)
        return proc,rec

    def finish(proc, rec, timeout=30):
        stdout, stderr = proc.communicate(timeout=timeout)
        rec.update(exit=proc.returncode, stdout=stdout.decode(), stderr=stderr.decode())
        save(out/"commands.json",records)
        if proc.returncode:
            raise RuntimeError(f"{rec['argv']}: {proc.returncode}: {rec['stderr']}")
        return stdout

    import shutil
    go=Path(shutil.which("go")).resolve()
    inputs = set(subprocess.check_output(["git","ls-files","-co","--exclude-standard","-z"],cwd=ROOT).decode().split("\0"))
    save(out/"inputs.json",{n:sha(ROOT/n) for n in sorted(inputs) if n.endswith((".go",".mod",".sum",".c",".h")) or n=="examples/effectfulcomposition/consumer/construction.py"})
    consumer,endpoint=tmp/"consumer",tmp/"alan-resource"
    for path,cwd,package in ((consumer,HERE,"."),(endpoint,ROOT,"./cmd/alan-resource")):
        argv=[go,"build","-buildvcs=false"] + (["-race"] if mode=="race" else []) + ["-o",path,package]
        finish(*run(argv,cwd,go),timeout=120)
        info=finish(*run([go,"version","-m",path],cwd,go))
        assert (b"-race=true" in info)==(mode=="race"),info

    db,world=tmp/"store.sqlite",tmp/"ledger.sqlite"
    journal=out/"ledger.jsonl"
    with contextlib.ExitStack() as cleanup:
        ar,aw=os.pipe()
        rr,rw=os.pipe()
        ackfd=fcntl.fcntl(aw,fcntl.F_DUPFD,10)
        relfd=fcntl.fcntl(rr,fcntl.F_DUPFD,10)
        for fd in (aw,rr,rw,ackfd,relfd):
            cleanup.callback(os.close,fd)
        events=cleanup.enter_context(os.fdopen(ar,"rb"))
        wrapper="import os,sys;os.dup2(int(sys.argv[1]),3);os.dup2(int(sys.argv[2]),4);os.execv(sys.argv[3],sys.argv[3:])"
        filters=[{"Scenario":"two-public-handoffs","Token":token,"Store":str(world),"Boundary":"resource.applied","Phase":"post-commit"} for token in ("first-applied","second-applied")]
        endpoint_env={**env,"ALAN_BARRIER":json.dumps(filters)}
        argv=[str(endpoint),"-receipt-ledger","-db",str(world),"-token","construction-"+mode,"-journal",str(journal)]
        launch=[sys.executable,"-c",wrapper,str(ackfd),str(relfd)]+argv
        ep=subprocess.Popen(launch,cwd=HERE,env=endpoint_env,pass_fds=(ackfd,relfd),stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        cleanup.callback(stop,ep)
        ready=ack(ep.stdout)
        assert ready["PID"]==ep.pid and ready["Store"]==str(world)
        ep_rec={"argv":argv,"launch":launch,"cwd":str(HERE),"pid":ep.pid,"env":endpoint_env,"executable_sha256":sha(endpoint),"ready":ready}
        records.append(ep_rec)
        def cli(command,*flags):
            return run([consumer,command,"-db",db,*flags],HERE,consumer)
        provision=json.loads(finish(*cli("init","-endpoint",ready["URL"])))
        assert len(provision["grants"])==2 and len(provision["roots"])==3
        old,entry=cli("issue","-endpoint",ready["URL"])
        cleanup.callback(stop,old)
        barrier=ack(events)
        assert barrier["PID"]==ep.pid and barrier["Boundary"]=="resource.applied" and barrier["Phase"]=="post-commit"
        # Require committed independent world rows and durably selected echoes on
        # both terminal source projections before killing the executor.
        end=time.monotonic()+60
        while True:
            if old.poll() is not None:
                raise RuntimeError("issuer unexpectedly exited: "+old.stderr.read().decode())
            raw=[json.loads(s) for s in journal.read_text().splitlines()]
            with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as conn:
                rows={k:json.loads(v) for k,v in conn.execute("SELECT subject,projection FROM runs")}
            closed={k:v for k,v in rows.items() if v["Status"]=="succeeded" and v.get("Data",{}).get("echo",{}).get("Text")}
            if len(closed)==2 and sum(row["Event"]=="committed" for row in raw)==2:
                break
            if time.monotonic()>end:
                raise RuntimeError(f"unreached construction cut: {[(k,v['Status'],v.get('Data',{}).get('echo')) for k,v in rows.items()]}, {raw}")
            time.sleep(.02)
        phase=rows["parent"]["Data"]
        ids={}
        for name,seq in (("l",1),("r",2)):
            target=phase[name+"_target"]["Text"]
            source,generation,sequence,ordinal=json.loads(target)
            assert (generation,sequence,ordinal)==(1,2,0)
            assert closed[source]["Data"]["original"]["Text"]==closed[source]["Data"]["echo"]["Text"]==target
            assert phase[name+"_resolver_ack"]["Text"]==rows["resolver_"+name]["Data"]["received"]["Text"]==target
            inv=phase[name+"_invocation"]["Text"]
            assert json.loads(inv)==["parent",1,seq,0]
            # Require the actual retained source selection of the parent's echo.
            with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as conn:
                selected=[json.loads(z[0]) for z in conn.execute(
                    "SELECT o.descriptor FROM history h JOIN observations o ON o.id=h.selected_observation AND o.consumed_sequence=h.sequence WHERE h.subject=? AND h.generation=1",(source,))]
            echoes=[z for z in selected if z["Obligation"]==name+"_identity" and z["Source"]=="parent" and z["Payload"]["target"]["Text"]==target]
            assert len(echoes)==1,(name,selected)
            grant=next(g for g in provision["grants"] if g["Resolver"]["Subject"]=="resolver_"+name)
            assert (grant["Invocation"],grant["Member"],grant["Sequence"],grant["Endpoint"],grant["Adopt"])==(inv,"source",2,ready["URL"],True)
            ids[name]={"target":target,"issuer":source,"invocation":inv,"selected_ack":echoes[0],"grant":grant}
        assert ids["l"]["target"]!=ids["r"]["target"]
        wires=[json.loads(r["Body"]) for r in raw if r["Event"]=="ingress"]
        assert len(wires)==2
        for name,delta in (("l",1),("r",2)):
            original=ids[name]["target"]
            matches=[w for w in wires if json.loads(w["Key"])==[2,provision["namespace"],original]]
            assert len(matches)==1 and matches[0]["Delta"]==delta and matches[0]["Profile"]=="idempotent"
        assert all(r["Path"].endswith("/apply") for r in raw if r["Event"]=="ingress")
        backup(db,out/"held-local.sqlite.gz")
        backup(world,out/"held-world.sqlite.gz")
        old.kill()
        stdout,stderr=old.communicate(timeout=10)
        entry.update(exit=old.returncode,stdout=stdout.decode(),stderr=stderr.decode(),first_endpoint_barrier=barrier)
        assert old.returncode==-signal.SIGKILL
        os.write(rw,b"cc")
        with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as conn:
            assert set(row[0] for row in conn.execute("SELECT intent_id FROM adapter_attempts"))=={r["target"] for r in ids.values()}
            assert not list(conn.execute("SELECT * FROM adapter_results"))
            assert not list(conn.execute("SELECT * FROM receipt_read_ordinals"))
        new,rec=cli("frontier")
        result=json.loads(finish(new,rec))
        assert new.pid!=old.pid
        before=journal.read_bytes()
        finish(*cli("replay"))
        assert journal.read_bytes()==before
        backup(db,out/"reopened-local.sqlite.gz")
        ep.send_signal(signal.SIGTERM)
        ep_out,ep_err=ep.communicate(timeout=15)
        assert ep.returncode==0,ep_err
        ep_rec.update(exit=ep.returncode,stdout=ep_out.decode(),stderr=ep_err.decode())
        save(out/"commands.json",records)
        save(out/"frontier.json",{"public":result,"ids":ids,"grants":provision["grants"],"roots":provision["roots"],"first_barrier":barrier,"fresh_owner_pid":new.pid,"killed_pid":old.pid,"signal":old.returncode,"world":ready,"wires":wires})
        print(f"{mode}: both handoffs, original physical admissions, revoked sources, two open resolver roots, no read, fresh Q and replay passed")


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("regime",choices=("normal","race"))
    p.add_argument("output",type=Path)
    a=p.parse_args()
    a.output.mkdir(parents=True,exist_ok=False)
    with tempfile.TemporaryDirectory(prefix="phase2d-public-") as d:
        commands(a.regime,a.output.resolve(),Path(d))


if __name__=="__main__":
    main()
