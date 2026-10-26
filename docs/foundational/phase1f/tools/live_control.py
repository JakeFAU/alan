#!/usr/bin/env python3
"""Qualification-only spending gate around the unchanged CLI/adapter protocol.

Every adapter.authorized boundary pauses before transport. The controller validates
its durable identity, reserves spend for model work, then releases. It never
chooses a program transition or supplies a provider response. One controller owns
the budget file. The 64 one-shot filters exceed the application's 17-attempt bound.
"""
import argparse
import base64
import fcntl
import hashlib
import json
import os
from pathlib import Path
import selectors
import shlex
import sqlite3
import subprocess
import sys
import time
import urllib.parse

MODEL = "gpt-6.1-sol"
ENDPOINT = "https://api.openai.com/v1/responses"
MAX_ATTEMPTS = 17
FILTERS = 64

def require(value, message):
    if not value:
        raise ValueError(message)


def write_json(path, value):
    path = Path(path)
    temp = path.with_suffix(path.suffix + ".tmp")
    with temp.open("w") as f:
        json.dump(value, f, indent=2)
        f.write("\n")
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp, path)
    fd = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def credential(path):
    if path:
        found = []
        for line in Path(path).read_text().splitlines():
            text = line.strip()
            if text.startswith("export "):
                text = text[7:].lstrip()
            name, sep, value = text.partition("=")
            if sep and name.strip() == "OPENAI_API_KEY":
                parts = shlex.split(value, comments=True)
                require(len(parts) == 1, "invalid credential assignment")
                found.append(parts[0])
        require(len(found) == 1, "credential assignment missing or duplicated")
        key = found[0]
    else:
        key = os.environ.get("OPENAI_API_KEY", "")
    require(bool(key) and "\n" not in key, "credential absent or invalid")
    return key


def clean_env():
    return {k: v for k, v in os.environ.items()
            if k not in ("OPENAI_API_KEY", "ALAN_BARRIER")}


def inspect(binary, db):
    result = subprocess.run([binary, "inspect", "--db", db], env=clean_env(),
                            capture_output=True, timeout=30)
    require(result.returncode == 0, "CLI inspection failed (raw stderr retained only by run controller)")
    return json.loads(result.stdout)


def snapshot(view):
    require("Snapshot" in view and "Tables" in view["Snapshot"], "missing public snapshot")
    return view["Snapshot"]


def validate_boundary(event, view, pid, db, tokens):
    require(event["PID"] == pid and event["Store"] == db, "wrong process/store")
    require(event["Boundary"] == "adapter.authorized" and event["Phase"] == "pre-transport", "wrong boundary")
    require(event["Token"] in tokens, "unknown or repeated token")
    tables = snapshot(view)["Tables"]
    rows = [json.loads(r["descriptor"]) for r in tables["adapter_attempts"]
            if r["intent_id"] == event["Intent"]]
    require(len(rows) == 1, "attempt identity missing or ambiguous")
    attempt = rows[0]
    intent, prepared = attempt["Intent"], attempt["Prepared"]
    require(attempt["Allowed"] and attempt["AuthorizationID"] == event["Attempt"], "authorization mismatch")
    require(intent["ID"] == event["Intent"] == prepared["Request"]["ID"], "intent mismatch")
    require(intent["Run"] == {"Subject": event["Subject"], "Generation": event["Generation"]}, "run mismatch")
    require(intent["Sequence"] == event["Sequence"], "intent sequence mismatch")
    auths = [json.loads(r["result"]) for r in tables["attempts"] if r["id"] == event["Attempt"]]
    require(len(auths) == 1 and auths[0]["Allowed"], "missing retained authorization")
    auth = auths[0]
    require(auth["IntentID"] == event["Intent"] and auth["Source"]["Run"] == intent["Run"], "source mismatch")
    require(auth["Source"]["Epoch"] == event["Epoch"] and auth["Source"]["Sequence"] == event["Sequence"], "epoch/sequence mismatch")
    runs = [r for r in tables["runs"] if r["subject"] == event["Subject"] and r["generation"] == event["Generation"]]
    require(len(runs) == 1 and runs[0]["epoch"] == event["Epoch"] and runs[0]["sequence"] >= event["Sequence"], "current execution mismatch")
    tokens.remove(event["Token"])
    return attempt


def reservation(prepared):
    if prepared["Protocol"] != "openai/responses/v1":
        # Lookup is the only other permitted adapter in this bounded application.
        require(prepared["Request"]["Capability"] == "evidence-lookup", "unexpected adapter")
        return 0.0
    require(prepared["Model"] == MODEL and prepared["Endpoint"] == ENDPOINT, "unapproved model/endpoint")
    wire = json.loads(prepared["Wire"])
    require(wire["model"] == MODEL and wire["max_output_tokens"] == prepared["MaxOutputTokens"], "wire/config mismatch")
    require(wire.get("service_tier", "default") == "default", "unapproved service tier")
    require(1 <= prepared["MaxOutputTokens"] <= 4096, "output ceiling")
    # Conservative planning bound: byte-level text token bound plus 8192 tokens
    # for protocol overhead, highest documented input/cache-write rate $5/M,
    # output $10/M, and 25% margin. This is a local spending assumption, not an
    # invoice guarantee or provider-side hard billing cap.
    return round(1.25 * ((len(prepared["Wire"].encode()) + 8192) * 5
                         + prepared["MaxOutputTokens"] * 10) / 1_000_000, 8)


def charge(ledger):
    previous = sum(r.get("known_estimated_token_cost_usd", 0)
                   + r.get("unknown_spend_reservation_usd", 0)
                   for r in ledger.get("reservations", []))
    return previous + sum(op.get("budget_charge_usd", op["reserved_usd"])
                          for run in ledger.get("controlled_runs", [])
                          for op in run["operations"])


def add_operation(ledger, run, operation):
    require(charge(ledger) + operation["reserved_usd"] <= ledger["maximum_total_usd"], "total spending limit reached")
    require(not any(o["id"] == operation["id"] for o in run["operations"]), "duplicate transport release")
    run["operations"].append(operation)


def unresolved_reservations(run):
    return sum(op["reserved_usd"] for op in run["operations"]
               if op.get("status") not in ("usage-known", "local"))


def usage_counts(usage):
    require(type(usage) is dict, "usage must be an object")
    for field in ("InputTokens", "OutputTokens", "TotalTokens"):
        require(type(usage[field]) is int and usage[field] >= 0,
                field + " must be a nonnegative integer")
    require(usage["TotalTokens"] == usage["InputTokens"] + usage["OutputTokens"],
            "total tokens must equal input plus output")
    require(usage["TotalTokens"] > 0, "successful result has zero billable usage")
    detail = usage.get("Details")
    if detail is None:
        detail = {}
    require(type(detail) is dict, "usage details must be an object")
    ins = detail.get("input_tokens_details")
    if ins is None:
        ins = {}
    require(type(ins) is dict, "input token details must be an object")
    cached = ins.get("cached_tokens", 0)
    written = ins.get("cache_write_tokens", 0)
    require(type(cached) is int and cached >= 0,
            "cached tokens must be a nonnegative integer")
    require(type(written) is int and written >= 0,
            "cache write tokens must be a nonnegative integer")
    require(cached + written <= usage["InputTokens"], "cache counts exceed input tokens")
    return cached, written


def reconcile(run, view):
    results = {r["Candidate"]["ID"]: r["Candidate"] for r in
               (json.loads(row["descriptor"]) for row in snapshot(view)["Tables"]["adapter_results"])}
    for op in run["operations"]:
        if not op["reserved_usd"]:
            op.update(status="local", budget_charge_usd=0)
            continue
        # A previous reconciliation must never discount newly unresolved usage.
        op.pop("budget_charge_usd", None)
        op.pop("estimated_token_cost_usd", None)
        op.pop("usage", None)
        op.update(status="unknown-cost-reservation-retained", cost_reason="result absent")
        result = results.get(op["id"])
        if result:
            op.update(request_id=result.get("RequestID"), response_id=result.get("ResponseID"))
        usage = result.get("Usage") if result else None
        if usage is None or (type(usage) is dict and any(
                usage.get(k) is None for k in ("InputTokens", "OutputTokens", "TotalTokens"))):
            op["cost_reason"] = "usage absent or token counts incomplete"
            continue
        try:
            cached, written = usage_counts(usage)
        except ValueError as error:
            op.update(status="malformed-usage-reservation-retained", cost_reason=str(error))
            continue
        op["cost_reason"] = "result validation incomplete; full reservation retained"
        require(result["ReturnedModel"] == MODEL, "unexpected returned model; keep full reservation")
        body = json.loads(result["Response"])
        require(body.get("service_tier", "default") == "default", "unexpected returned tier; keep reservation")
        ordinary = usage["InputTokens"] - cached - written
        # Provider-reported usage is an accounting assumption, not invoice proof.
        # Cache write duration not selected: charge maximum documented rate.
        estimate = (ordinary * 2 + cached * .1 + written * 5 + usage["OutputTokens"] * 10) / 1_000_000
        budget = 1.25 * (usage["InputTokens"] * 5 + usage["OutputTokens"] * 10) / 1_000_000
        require(budget <= op["reserved_usd"], "usage exceeded conservative reservation")
        op.pop("cost_reason", None)
        op.update(status="usage-known", estimated_token_cost_usd=round(estimate, 8),
                  budget_charge_usd=round(budget, 8), usage=usage,
                  request_id=result["RequestID"], response_id=result["ResponseID"],
                  returned_model=result["ReturnedModel"], service_tier=body.get("service_tier"))


def run(args):
    out = Path(args.output)
    out.mkdir(parents=True, exist_ok=False)
    ledger_path = Path(args.ledger)
    with ledger_path.with_suffix(".lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        ledger = json.loads(ledger_path.read_text())
        require(ledger["maximum_total_usd"] == 10, "unexpected total authorization")
        key = credential(args.credential_file)
        before = inspect(args.binary, args.db)
        require(not snapshot(before)["Tables"]["adapter_attempts"], "evaluation run must start fresh; no outcome-dependent rerun")
        record = dict(id=args.run_id, operations=[], status="running", max_attempts=MAX_ATTEMPTS,
                      binary_sha256=hashlib.sha256(Path(args.binary).read_bytes()).hexdigest())
        require(not any(r["id"] == args.run_id for r in ledger.get("controlled_runs", [])), "duplicate run identity")
        ledger.setdefault("controlled_runs", []).append(record)
        write_json(ledger_path, ledger)
        write_json(out/"before.json", before)
        filters = [dict(Scenario=args.run_id, Token=f"spend-{n}", Store=args.db,
                        Boundary="adapter.authorized", Phase="pre-transport") for n in range(FILTERS)]
        tokens = {f["Token"] for f in filters}
        ack_r, ack_w = os.pipe(); rel_r, rel_w = os.pipe()
        env = clean_env(); env["OPENAI_API_KEY"] = key; env["ALAN_BARRIER"] = json.dumps(filters)
        wrapper = "import os,sys; os.dup2(int(sys.argv[1]),3); os.dup2(int(sys.argv[2]),4); os.execv(sys.argv[3],sys.argv[3:])"
        start = time.monotonic()
        with (out/"run.stdout").open("wb") as stdout, (out/"run.stderr").open("wb") as stderr:
            proc = subprocess.Popen([sys.executable,"-c",wrapper,str(ack_w),str(rel_r),args.binary,"run","--db",args.db],
                                    env=env,pass_fds=(ack_w,rel_r),stdout=stdout,stderr=stderr)
            os.close(ack_w);os.close(rel_r)
            selector=selectors.DefaultSelector();selector.register(ack_r,selectors.EVENT_READ)
            pending=b""; events=[]; failure=None
            try:
                while proc.poll() is None:
                    require(time.monotonic()-start < args.timeout, "whole-run watchdog expired")
                    for _,_ in selector.select(timeout=.25):
                        raw=os.read(ack_r,65536)
                        if not raw:
                            selector.unregister(ack_r)
                            continue
                        pending+=raw
                        while b"\n" in pending:
                            line,pending=pending.split(b"\n",1);event=json.loads(line)
                            require(len(events)<MAX_ATTEMPTS,"application exceeded attempt ceiling")
                            view=inspect(args.binary,args.db)
                            attempt=validate_boundary(event,view,proc.pid,args.db,tokens)
                            amount=reservation(attempt["Prepared"])
                            op=dict(id=event["Intent"],reserved_usd=amount,status="released",ack=event,
                                    prepared=attempt["Prepared"])
                            add_operation(ledger,record,op)
                            # Reserve durably before allowing a request to leave.
                            write_json(ledger_path,ledger)
                            events.append(event);write_json(out/"boundaries.json",events)
                            os.write(rel_w,b"c")
                proc.wait(timeout=10)
            except BaseException as error:
                failure=type(error).__name__+": "+str(error)
                if proc.poll() is None: proc.kill()
                proc.wait(timeout=10)
            finally:
                selector.close();os.close(ack_r);os.close(rel_w)
            record.update(exit_code=proc.returncode, elapsed_seconds=round(time.monotonic()-start,3),controller_error=failure)
        after=inspect(args.binary,args.db);write_json(out/"inspect.json",after)
        reconcile(record,after)
        record["status"]="finished" if proc.returncode==0 and not failure else "failed-retained"
        ledger["conservative_charge_and_reservations_usd"]=round(charge(ledger),8)
        write_json(ledger_path,ledger);write_json(out/"run-record.json",record)
        # Offline replay has neither credentials nor network. macOS-only measured profile.
        profile="(version 1) (allow default) (deny network*)"
        control=subprocess.run(["/usr/bin/sandbox-exec","-p",profile,sys.executable,"-c",
          "import socket,errno; s=socket.socket(); n=s.connect_ex(('127.0.0.1',9)); print(n); assert n==errno.EPERM"],
          env=clean_env(),capture_output=True,timeout=30)
        require(control.returncode==0,"network denial positive control failed")
        replay=subprocess.run(["/usr/bin/sandbox-exec","-p",profile,args.binary,"replay","--db",args.db],
                              env=clean_env(),capture_output=True,timeout=60)
        (out/"replay.json").write_bytes(replay.stdout);(out/"replay.stderr").write_bytes(replay.stderr)
        after_replay=inspect(args.binary,args.db)
        write_json(out/"replay-control.json",dict(exit_code=replay.returncode,credential_present=False,
            network_profile=profile,network_denial_control=control.stdout.decode().strip(),inspection_unchanged=after==after_replay))
        require(replay.returncode==0 and after==after_replay,"offline replay mismatch")
        with sqlite3.connect(f"file:{args.db}?mode=ro",uri=True) as source, sqlite3.connect(out/"store.sqlite") as dest:
            source.backup(dest)
        needles=[key.encode(),json.dumps(key)[1:-1].encode(),urllib.parse.quote(key,safe="").encode(),base64.b64encode(key.encode())]
        for path in out.iterdir():
            if path.is_file(): require(not any(n in path.read_bytes() for n in needles),"secret scan failed; do not publish output")
        write_json(out/"summary.json",dict(status=record["status"],exit_code=record["exit_code"],
            elapsed_seconds=record["elapsed_seconds"],operations=len(record["operations"]),
            estimated_token_cost_usd=round(sum(o.get("estimated_token_cost_usd",0) for o in record["operations"]),8),
            unresolved_cost_reservations=unresolved_reservations(record),
            replay_pass=True,credential_scan_pass=True,store_sha256=hashlib.sha256((out/"store.sqlite").read_bytes()).hexdigest()))
        print((out/"summary.json").read_text())
        require(record["status"]=="finished","run failed; all outcomes and reservations retained")

if __name__=="__main__":
    parser=argparse.ArgumentParser()
    parser.add_argument("--binary",required=True);parser.add_argument("--db",required=True)
    parser.add_argument("--output",required=True);parser.add_argument("--ledger",required=True)
    parser.add_argument("--run-id",required=True);parser.add_argument("--credential-file")
    parser.add_argument("--timeout",type=int,default=1800)
    run(parser.parse_args())
