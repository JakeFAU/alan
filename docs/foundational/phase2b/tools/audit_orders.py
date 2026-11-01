#!/usr/bin/env python3
"""Independently check Phase 2B causal cuts against retained SQLite and ingress bytes."""
import collections
import copy
import hashlib
import json
from pathlib import Path
import sys

PHASE = Path(__file__).resolve().parents[1]
EVIDENCE = PHASE / "evidence"
REPO = PHASE.parents[2]
sys.dont_write_bytecode = True
sys.path.insert(0, str(PHASE.parent / "phase2a" / "tools"))
# Historical independent reader and unchanged structural joins. No Go runtime or fixture calls.
import audit_observers as prior

read, db, one = prior.read, prior.db, prior.one
effect, selected, projection = prior.effect, prior.selected, prior.projection
R = "R confirmed against durable source/scope"
FREE = "release held opportunity"
Q = "successful Q after Run handling"
Q0 = "successful Q while consideration held before A"
LIVE = {"A-source", "A-scope", "R-source", "R-scope", "R-abandon", "R-retire",
        "drop-after-commit", "queued-result", "recorded-result"}


def strings(tables):
    return {k: [{k: v.decode() if isinstance(v, bytes) else v for k, v in r.items()} for r in rows]
            for k, rows in tables.items()}


def assertion_order(sequence, *stages):
    positions = [sequence.index(s) for s in stages]
    assert sorted(set(positions)) == positions, (stages, sequence)


def action(row, target, name):
    req = row["Request"]
    assert req["Intent"]["Run"] == {"Subject": "retry" if name == "Retry" else "operator", "Generation": 1}
    assert req["Intent"]["Target"] == target and req["Operation"] == name
    assert row["Accepted"] and req["ExpectedRevision"] == req["SourceRevision"] == 1


def decision(t, target, name):
    d = one([json.loads(r["result"]) for r in t["lifecycle_results"]],
            DecisionSequence=2 if name == "Retry" else 1)
    action(d, target, name)
    one(t["permissions"], source=d["Request"]["Intent"]["Run"]["Subject"],
        target=target, revision=1, operation=name, kind="lifecycle")
    if name == "Retry":
        assert d["Successor"] == 2
    else:
        assert d["Before"]["Projection"]["Status"] == "open"
        assert d["After"]["Projection"]["Status"] == ("abandoned" if name == "AbandonRun" else "cancelled")
        assert d["After"]["Retired"] == (name == "RetireSubject")
    return d


def identity(t, route, op):
    attempt, grant, wire = effect(t, route, op)
    assert attempt["Allowed"] and grant["Allowed"] and attempt["Reason"] == grant["Reason"] == "authorized"
    intent = attempt["Intent"]
    assert json.loads(op) == [intent["Run"]["Subject"], intent["Run"]["Generation"], intent["Sequence"], intent["Ordinal"]]
    if route == "external":
        req = json.loads(wire)
        assert req["Subject"] == intent["Run"]["Subject"] and req["Generation"] == intent["Run"]["Generation"]
        assert req["Authorization"] == grant["ID"] and req["Sequence"] == intent["Sequence"]
        assert req["Epoch"] == grant["Source"]["Epoch"]
    return attempt, grant, wire


def outcome(t, route, op, receipt=None, *, disposition="validated", settled=False, consumed=False):
    table = "adapter_results" if route == "adapter" else "external_receipts"
    results = [r for r in t[table] if r["intent_id"] == op]
    assert len(results) == int(receipt is not None)
    if results:
        prior.joined_receipt(t, route, op, receipt, disposition)
    ss = [r for r in t["settlements"] if r["intent_id"] == op]
    obs = [json.loads(r["descriptor"]) for r in t["observations"] if json.loads(r["descriptor"])["Operation"] == op]
    assert len(ss) == len(obs) == int(settled) and len(selected(t, op)) == int(consumed)
    if settled:
        assert ss[0]["observation_id"] == obs[0]["ID"]
        assert obs[0]["Run"] == json.loads(one(t["intents"], id=op)["descriptor"])["Run"]
        v = json.loads(ss[0]["descriptor"])
        if route == "external":
            assert v["Receipt"] == receipt and v["Disposition"] == "applied"
        else:
            assert v == json.loads(results[0]["descriptor"])


def contact_rows(t, journal, wires, *, pending=False):
    """Check original wire, controller-independent ingress PID/token and durable endpoint rows."""
    records = [json.loads(line) for line in journal.splitlines()]
    contacts = [json.loads(r["request"]) for r in t["contacts"]]
    apps = [json.loads(r["receipt"]) for r in t["applications"]]
    assert len(contacts) == len(wires) - int(pending)
    assert len(records) == len(wires) + len(contacts)
    for i, wire in enumerate(wires):
        ingress = records[2*i]
        assert (ingress["Event"], ingress["Method"], ingress["Path"], ingress["Contact"]) == ("ingress", "POST", "/apply", i+1)
        assert ingress["Body"] == wire and ingress["PID"] > 0 and ingress["Process"]
        if i == len(contacts):
            continue
        saved = contacts[i]
        assert all(saved[k] == v for k, v in json.loads(wire).items())
        commit = records[2*i+1]
        assert (commit["Event"], commit["Contact"], commit["PID"], commit["Process"]) == (
            "committed", i+1, ingress["PID"], ingress["Process"])
        receipt = commit["Receipt"]
        assert receipt in apps and receipt["Key"] == saved["Key"] and receipt["Delta"] == saved["Delta"]
        assert receipt["Process"] == ingress["Process"] and receipt["PID"] == ingress["PID"]
    assert len(t["keys"]) == len({r["Key"] for r in contacts})
    for row in t["keys"]:
        req = json.loads(row["descriptor"])
        assert req in contacts and req["Key"] == row["key"] and req["Delta"] == row["delta"]
    assert [r["Mutation"] for r in apps] == list(range(1, len(apps)+1))
    total = 0
    for r in apps:
        total += r["Delta"]
        assert r["Total"] == total
    return apps


def dump(prefix, label):
    def path(s): return prefix.with_name(prefix.name + "-" + label + "-" + s)
    return db(path("local.sqlite.gz")), db(path("world.sqlite.gz")), path("ingress.jsonl").read_text()


def labels(events, op):
    return [e if isinstance(e, str) else e["Boundary"] + "/" + e["Phase"]
            for e in events if isinstance(e, str) or e["Intent"] == op or e["Boundary"] == "dispatch.waiting"]


def live(x, case, saved):
    raw, w, log = saved
    held, revoked, end = [x[s]["Tables"] for s in ("held", "revoked", "final")]
    assert strings(raw) == end and log == x["ingress"]
    route = "adapter" if x["route"] == "adapter" else "external"
    op = x["operation"]
    a, au, wire = identity(held, route, op)
    assert x["key"] == json.loads(wire)["Key"]
    assert len(held["composition_charges"]) == 1
    charge = json.loads(held["composition_charges"][0]["descriptor"])
    assert charge["Kind"] == "effect" and charge["OperationID"] == op and charge["Effects"] == 1
    for t in (revoked, end):
        assert (a, au, wire) == identity(t, route, op)
        assert t["composition_charges"] == held["composition_charges"]
        assert len(t[route+"_attempts"]) == 1
    domain = "scope" if case.endswith("-scope") else "client"
    name = {"R-retire": "RetireSubject", "R-abandon": "AbandonRun"}.get(case, "CancelRun")
    assert x["target"] == {"Namespace": held["metadata"][0]["namespace"], "Run": a["Intent"]["Run"], "OwningScope": domain == "scope"}
    d = decision(revoked, domain, name)
    assert decision(end, domain, name) == d and not held["lifecycle_results"]
    assert projection(revoked, domain) == d["After"]["Projection"] and projection(held, domain) == d["Before"]["Projection"]
    assert (projection(revoked, "client")["Status"] == "open") == (domain == "scope")
    denied = case.startswith("R-")
    apps = contact_rows(w, log, [] if denied else [wire])
    assert apps == (x["world_final"]["Applications"] or []) and len(apps) == int(not denied)
    late = case in ("queued-result", "recorded-result")
    assert (x["world_held"]["Applications"] or []) == (apps if late else [])
    assert x["world_held"]["Contacts"] == int(late)
    original = apps[0] if apps else None
    evidence = original if case != "returned-unknown" and not (route == "external" and case == "drop-after-commit") else None
    for t in (held, revoked):
        outcome(t, route, op, original if case == "recorded-result" else None)
    outcome(end, route, op, evidence, disposition="transport_error" if case == "drop-after-commit" else "validated",
            settled=bool(evidence), consumed=bool(evidence) and domain == "scope")
    seq = labels(x["events"], op)
    consider, admit, waiting = "dispatch.consider/pre-admission", "dispatch.admitted/pre-call", "dispatch.waiting/in-flight"
    base = ["intent.authorized/authorized", consider]
    if denied:
        reason = "scope_closed" if domain == "scope" else "source_retired" if name == "RetireSubject" else "source_closed"
        assert x["wait_error"] == "" and admit not in seq and waiting not in seq
        assert not any(s.startswith(route+".") or s.startswith("runner.received") for s in seq)
        assertion_order(seq, *base, R, Q0, FREE, "dispatch.refused/"+reason, Q)
    else:
        assert x["wait_error"] == "context canceled" and seq.count(admit) == 1 and not any(s.startswith("dispatch.refused") for s in seq)
        response = route+".response/pre-record"
        rec = route+".result-recorded/pre-settlement"
        h = [route+".finished/transport-ended", "runner.received/pre-account"]
        if not (case == "drop-after-commit" and route == "external"):
            h.insert(0, response)
        if evidence:
            h += [rec, route+".settled/"+("pre-consumption" if route == "adapter" else "post-record")]
        if case == "returned-unknown":
            h.append("adapter.unknown/unresolved")
        cut = (h.index(rec)+1 if case == "recorded-result" else h.index("runner.received/pre-account")+1 if late else 0)
        assertion_order(seq, *base, route+".authorized/pre-transport", admit, *h[:cut], R, waiting, FREE, *h[cut:], Q)
    assert seq[-1] == Q


def checked_replay(f):
    x = read(f)
    assert x["before"] == x["after"] and x["endpoint_before"] == x["endpoint_after"]
    for side in ("local", "world"):
        paths = [f.parent / Path(x[k+"_"+side+"_store"]).name for k in ("before", "after")]
        a, b = (db(p) for p in paths)
        assert a == b
        if side == "local":
            assert strings(a) == x["before"]["Tables"]
            assert len(a["history"]) == x["report"]["HistoryCount"] and len(a["intents"]) == x["report"]["IntentCount"]
    assert x["ingress_before"] == x["ingress_after"]


def owners(prefix, x, signals, race, cut, end, known):
    b = x.get("barrier") or x["executor_barrier"]
    first, fresh = x["crashed"], x["reopened"]
    assert [r["Command"] for r in first] == ["open", "start", "revoke-manual" if known else "revoke-automatic"]
    commands = ["open", "quiesce"] + ([] if known else ["start", "finish", "quiesce"]) + ["exit"]
    assert [r["Command"] for r in fresh] == commands
    pid, next_pid = first[0]["PID"], fresh[0]["PID"]
    assert pid == b["PID"] and pid != next_pid and x["signal"] == "SIGKILL"
    sig = one(signals, pid=pid)
    assert sig["exit_signal"] == "killed" and sig["race_children"] == race
    assert all(sig["barrier_config"][k] == b[k] for k in ("Token", "Store", "Boundary", "Scenario", "Phase"))
    assert b["Subject"] == "client" and b["Generation"] == 1 and b["Epoch"] == prior.run(cut, "client")["epoch"]
    for replies, ident, rows in ((first, pid, cut), (fresh, next_pid, end)):
        start = read(prefix.with_name(f"{prefix.name}-process-start-{ident}.json"))
        assert start["PID"] == ident and start["executable_sha256"] == sig["binary_sha256"]
        assert start["environment"]["ALAN_PHASE2_DB"] == b["Store"] and start["argv"] == sig["command"]
        for r in replies:
            assert r["PID"] == ident and r["AnchorEpoch"] == prior.run(rows,"anchor")["epoch"]
            assert r["Client"]["ID"] == {"Subject":"client","Generation":1}
            assert r["Client"]["Epoch"] == prior.run(rows,"client")["epoch"]
    assert first[-1]["Client"]["Projection"] == projection(cut,"client") == projection(end,"client")
    assert all(r["Client"]["Projection"] == projection(end,"client") for r in fresh)
    assert fresh[0]["AnchorEpoch"] == first[0]["AnchorEpoch"]+1 and fresh[0]["Client"]["Epoch"] == first[0]["Client"]["Epoch"]
    finished = read(prefix.with_name(f"{prefix.name}-process-{next_pid}.json"))
    assert finished["exit_code"] == 0 and finished["race_children"] == race and finished["binary_sha256"] == sig["binary_sha256"]


def closed(prefix, x, branch, route, signals, race):
    known, delayed = branch == "known", branch == "delayed"
    tag = branch+"-" if branch.startswith("U") else ""
    cut, wc, jc = dump(prefix, tag+"cut")
    end, wa, ja = dump(prefix, tag+"after")
    owners(prefix, x, signals, race, cut, end, known)
    b = x.get("barrier") or x["executor_barrier"]
    op = '["client",1,1,0]'
    a, grant, wire = identity(cut, route, op)
    assert identity(end, route, op) == (a, grant, wire)
    assert cut[route+"_attempts"] == end[route+"_attempts"] and len(cut[route+"_attempts"]) == 1
    assert not cut.get("composition_charges") and not end.get("composition_charges")
    assert decision(cut,"client","CancelRun") == decision(end,"client","CancelRun")
    if delayed:
        assert b["Boundary"] == "fixture.revoked" and b["Phase"] == "pre-kill" and x["live_open_refusal"] == "store already active"
        ing = x["endpoint_ingress"]
        assert ing["Boundary"] == "resource.ingress" and ing["Phase"] == "pre-apply" and ing["Intent"] == x["key"] == json.loads(wire)["Key"]
        assert ing["Attempt"] == grant["ID"] and ing["Epoch"] == b["Epoch"] and ing["PID"] == json.loads(jc)["PID"] != b["PID"]
        middle, wm, jm = dump(prefix,"acknowledged")
        assert (wm,jm) == (wc,jc) and ja.startswith(jc)
        assert not contact_rows(wm,jm,[wire],pending=True)
        (app,) = contact_rows(wa,ja,[wire])
        assert app["PID"] == ing["PID"] and app["Process"] == json.loads(jm)["Process"]
        assert middle == end
        outcome(middle,route,op)
    else:
        expected = (route+".result-recorded","pre-settlement") if known else (
            ("dispatch.admitted","pre-call") if branch == "U0" else ("external.response","pre-record"))
        assert (b["Boundary"],b["Phase"]) == expected and b["Intent"] == op and b["Attempt"] == grant["ID"]
        assert (wc,jc) == (wa,ja)
        apps = contact_rows(wc,jc,[wire] if known or branch == "U1" else [])
        assert len(apps) == int(known or branch == "U1")
        if known:
            result_table = "adapter_results" if route == "adapter" else "external_receipts"
            assert end[result_table] == cut[result_table]
            outcome(cut,route,op,apps[0]); outcome(end,route,op,apps[0],settled=True)
            assert cut["history"] == end["history"]
    if not known:
        outcome(cut,route,op); outcome(end,route,op)
        (report,) = [r["Report"] for r in x["reopened"] if r["Command"] == "finish"]
        assert report["UnknownExternal"] == [op]
    for label, journal in ((tag+"cut", jc),(tag+"after", ja)):
        replay = read(prefix.with_name(prefix.name+"-replay-"+label+".json"))
        assert replay["ingress_before"] == journal
    return cut, end


def applications(x, wires):
    """Link a composition fixture's journal bytes to its independent endpoint inspection."""
    w = x["world"]
    log = [json.loads(line) for line in x.get("journal", "").splitlines()]
    apps, requests = w["Applications"] or [], w["Requests"] or []
    assert len(wires) == w["Contacts"] == len(apps) == len(requests) and len(log) == 2*len(wires)
    for i, (wire, req, app) in enumerate(zip(wires, requests, apps)):
        p, c = log[2*i:2*i+2]
        assert (p["Event"], p["Method"], p["Path"], p["Body"]) == ("ingress", "POST", "/apply", wire)
        assert p["Contact"] == c["Contact"] == i+1 and p["PID"] == c["PID"] == app["PID"]
        assert p["Process"] == c["Process"] == app["Process"] and c["Event"] == "committed" and c["Receipt"] == app
        assert all(req[k] == v for k, v in json.loads(wire).items())
        assert req["Key"] == app["Key"] and req["Delta"] == app["Delta"] and app["Mutation"] == i+1
    return apps


def generations(prefix, x):
    raw, w, j = dump(prefix,"final")
    end = strings(raw)
    assert end == x["after"]["Tables"]
    old = one(end["external_attempts"], intent_id='["client",1,1,0]')["intent_id"]
    new = '["client",2,1,0]'
    assert identity(end,"external",old) == identity(x["before"]["Tables"],"external",old)
    _, _, wire = identity(end,"external",new)
    first = json.loads(identity(end,"external",old)[2])
    assert first["Key"] == x["old_key"] != json.loads(wire)["Key"]
    assert decision(end,"client","CancelRun") == x["cancel"]
    assert decision(end,"client","Retry") == x["retry"]
    assert projection(end,"client",1)["Status"] == "cancelled" and projection(end,"client",2)["Status"] == "succeeded"
    (app,) = contact_rows(w,j,[wire])
    assert x["world"]["Applications"] == [app]
    outcome(end,"external",old)
    outcome(end,"external",new,app,settled=True,consumed=True)
    marker = "Q on G1 only; G2 still eligible; release G1 consideration"
    assertion_order(labels(x["events"],old), "dispatch.consider/pre-admission", marker, "dispatch.refused/source_generation_changed")
    assert "dispatch.admitted/pre-call" not in labels(x["events"],old)
    assertion_order(labels(x["events"],new), marker, "dispatch.admitted/pre-call", "external.response/pre-record")


def open_success(prefix, route):
    x = read(prefix.with_name(prefix.name+"-success.json"))
    t,w,j = dump(prefix,"final")
    (r,) = t[route+"_attempts"]
    op = r["intent_id"]
    _, _, wire = identity(t,route,op)
    (app,) = contact_rows(w,j,[wire])
    assert json.loads(wire)["Key"] == x["key"] == app["Key"]
    outcome(t,route,op,app,settled=True,consumed=True)
    (obs,) = selected(t,op)
    assert obs["ID"] == x["selected"] and projection(t,"client") == x["client"] and x["client"]["Status"] == "succeeded"


def composition(f):
    x = read(f)
    end = x["after"]["Tables"]
    name = f.name
    if "ExactPriorStores" in name:
        route = "adapter" if "-adapter" in name else "external"
        before = x["before"]["Tables"]
        source = (REPO/"internal/durabletest"/x["archive"]).resolve()
        assert source.parent == PHASE.parent/"phase2a/evidence/process-normal"
        assert strings(db(source)) == before
        (r,) = before[route+"_attempts"]
        op = r["intent_id"]
        assert identity(before,route,op) == identity(end,route,op)
        result = "adapter_results" if route == "adapter" else "external_receipts"
        assert before[result] == end[result] and len(before[result]) == 1 and not before["settlements"]
        assert not selected(before,op) and len(selected(end,op)) == 1 and projection(end,"client")["Status"] == "succeeded"
        for cut,t in (("before",before),("after",end)):
            assert x["replay_"+cut]["HistoryCount"] == len(t["history"]) and x["replay_"+cut]["IntentCount"] == len(t["intents"])
        return
    if "SlowEffect" in name:
        before = x["before"]["Tables"]
        assert projection(before,"ordinary")["Status"] == "succeeded" and projection(before,"peer")["Status"] == "open"
        assert decision(before,"client","AbandonRun") and x["q_interrupted"]
        ordered = [one(end["intents"],subject=who, sequence=seq)["id"] for who,seq in (("client",1),("peer",13))]
        wires = [identity(end,"adapter",op)[2] for op in ordered]
        ack = x["ingress_ack"]
        assert (ack["Boundary"], ack["Phase"],ack["Intent"]) == ("resource.ingress","pre-apply",json.loads(wires[0])["Key"])
        apps = applications(x,wires)
        assert ack["PID"] == apps[0]["PID"] and [a["Total"] for a in apps] == [1,3]
        for i,op in enumerate(ordered):
            outcome(before,"adapter",op)
            outcome(end,"adapter",op,apps[i],settled=True,consumed=i==1)
        return
    op = x["cut"]["Intent"]
    _, _, wire = identity(end,"adapter",op)
    root, = x["after"]["Roots"]
    assert root["Charges"]["Effects"] == 1
    (charge,) = [r for r in end["composition_charges"] if r["kind"] == "effect"]
    assert charge["operation_id"] == op and json.loads(charge["descriptor"])["Effects"] == 1
    stages = labels(x["events"],op)
    if "SuccessfulSource" in name:
        assert projection(end,"root")["Status"] == "succeeded" and root["UnknownExternal"] == [op]
        assert root["Outcome"] is None and root["Pending"] == "external_outcome"
        assert "dispatch.refused/source_closed" in stages and "dispatch.admitted/pre-call" not in stages
        outcome(end,"adapter",op)
        applications(x,[])
        return
    before = x["before"]["Tables"]
    child = x["cut"]["Subject"]
    assert identity(before,"adapter",op) == identity(end,"adapter",op)
    outcome(before,"adapter",op)
    if "NestedDrain" in name:
        source, frame = x["source"], x["scope"]["Run"]
        assert x["scope_pending"] and x["scope"]["OwningScope"] and x["original"] == op
        bindings = [json.loads(r["descriptor"]) for r in before["composition_bindings"]]
        outer = one(bindings,Depth=1)
        inner = one(bindings,Depth=2)
        assert outer["Members"]["entry"] == inner["Caller"] == frame and inner["Members"]["entry"] == source
        assert projection(before,frame["Subject"])["Status"] == "failed" and projection(before,child)["Status"] == "open"
        assert source == {"Subject":child,"Generation":1} and projection(end,child)["Status"] == "succeeded"
        (receipt,) = applications(x,[wire])
        outcome(end,"adapter",op,receipt,settled=True,consumed=True)
        assertion_order(stages,"dispatch.admitted/pre-call","adapter.response/pre-record","adapter.settled/pre-consumption")
        return
    if "PrivateFailureOrders" in name:
        allowed = x["case"] == "dispatch.admitted"
        assert projection(before,child)["Status"] == projection(end,child)["Status"] == "failed"
        prior.parent(end,"failed")
        if allowed:
            prior.parent(before,"failed")
    else:
        assert "NestedAuthority" in name
        status = x["case"]
        allowed = status == "succeeded"
        frozen = x["prepared"]["Freeze"]
        assert frozen["Run"] == x["target"]["Run"] and frozen["Status"] == status
        assert not before["composition_publications"] and x["q_open"] == allowed
        assert projection(before,child)["Status"] == "open" and projection(end,child)["Status"] == ("succeeded" if allowed else "open")
        root_state = projection(end,"root")
        outer = root_state["Data"]["outer"]["Text"]
        (choice,) = [o for o in selected(end,outer) if o["Run"]["Subject"] == "root"]
        assert choice["Run"]["Subject"] == "root" and choice["Kind"] == "invocation"
        published = json.loads(one(end["composition_publications"],operation_id=outer)["descriptor"])
        assert published["Status"] == choice["Payload"]["status"]["Text"] == status
        assert published["InvocationID"] == frozen["InvocationID"] and published["ResponsePresent"] == allowed
        assert root_state["Data"]["present"]["Flag"] == allowed
    apps = applications(x,[wire] if allowed else [])
    if allowed:
        assertion_order(stages, "dispatch.admitted/pre-call","adapter.response/pre-record","adapter.result-recorded/pre-settlement","adapter.settled/pre-consumption")
        outcome(end,"adapter",op,apps[0],settled=True,consumed="NestedAuthority" in name)
    else:
        reason = "scope_closed" if "NestedAuthority" in name else "source_closed"
        assert "dispatch.refused/"+reason in stages and "dispatch.admitted/pre-call" not in stages
        outcome(end,"adapter",op)


def logs(command, key):
    return [json.loads("{"+line.split(": {",1)[1]) for line in command["output"].splitlines()
            if ": {" in line and '"'+key+'":' in line]


def direct(x):
    op, target = x["operation"], x["target"]
    assert x["total_posts"] == 2 and x["q_pending_with_one_direct_left"]
    a,b = [x[k]["Tables"] for k in ("first_H","second_H")]
    assert a == b and len(a["external_attempts"]) == 1
    _, _, wire = identity(a,"external",op)
    (r,) = a["external_receipts"]
    receipt = json.loads(r["descriptor"])
    assert receipt["Key"] == json.loads(wire)["Key"]
    assert projection(a,target["Run"]["Subject"])["Status"] == "succeeded" and not target["OwningScope"]
    assert target["Namespace"] == a["metadata"][0]["namespace"]
    outcome(a,"external",op,receipt,settled=True)


def consumer(x):
    mode = x["mode"]
    t = x["inspection"]["Tables"]
    parent = {"success":"succeeded","selected-error":"failed","unknown":"open"}[mode]
    prior.parent(t,parent)
    assert x["parent"]["Projection"] == projection(t,"root")
    (r,) = t["adapter_attempts"]
    op = r["intent_id"]
    identity(t,"adapter",op)
    assert x["target"] == {"Namespace":t["metadata"][0]["namespace"],"Run":{"Subject":"root","Generation":1},"OwningScope":True}
    if mode == "unknown":
        assert x["host_output"] == "" and x["run"]["UnknownAdapters"] == [op]
        outcome(t,"adapter",op)
    else:
        assert json.loads(x["host_output"]) == {"target":x["target"],"local_quiescent":True}
        result = json.loads(one(t["adapter_results"], intent_id=op)["descriptor"])
        receipt = json.loads(result["Candidate"]["Response"])
        outcome(t,"adapter",op,receipt,disposition="validated" if mode=="success" else "selected_error",settled=True,consumed=True)


def historical_authority(x, race):
    assert x["race"] == race
    c,op,t = x["case"],x["operation"],x["final"]["Tables"]
    if c["Cut"] != "relaunch" and (c["Cut"] != "after-permission" or c["Member"]):
        prior.authority(x)
        return
    route = c["Route"]
    a, au, wire = identity(t,route,op)
    assert projection(t,"client")["Status"] == "cancelled" and not t.get("composition_charges")
    decision(t,"client","CancelRun")
    before = (x.get("closed_before_relaunch") or x["closed_before_run"])["local"]["Tables"]
    assert json.loads(one(before["attempts"],intent_id=op)["result"]) == au
    if c["Cut"] == "after-permission":
        assert not before[route+"_attempts"] and x["world"]["Contacts"] == 0 and not x["world"]["Applications"]
        assert x["ingress_journal"] == ""
    else:
        assert identity(before,route,op) == (a,au,wire) and x["world"]["Contacts"] == 1
        post,commit = map(json.loads,x["ingress_journal"].splitlines())
        assert post["Body"] == wire and post["Event"] == "ingress" and commit["Event"] == "committed"
        assert commit["Receipt"] == x["world"]["Applications"][0]
        outcome(before,route,op)
    outcome(t,route,op)


def provenance(directory, command, race):
    manifests, endpoints = {}, {}
    sources = list(directory.glob("*-sources-*.json"))
    assert sources
    for f in sources:
        hashes = read(f)
        key = f.stem.rpartition("-")[2]
        if key in manifests:
            assert hashes == manifests[key]
        else:
            ordered = "".join(f"{hashes[k]}  {k}\n" for k in sorted(hashes))
            assert hashlib.sha256(ordered.encode()).hexdigest() == key
            assert all(command["inputs"][k] == value for k,value in hashes.items())
            manifests[key] = hashes
    starts = list(directory.glob("*-process-start-*.json"))
    servers = list(directory.glob("*-provenance.json"))
    for f in starts+servers:
        x = read(f)
        assert x["source_manifest_sha256"] in manifests and len(x["executable_sha256"]) == 64
        settings = {a["Key"]:a.get("Value") for a in x["child_build_info"]["Settings"]}
        assert settings["CGO_ENABLED"] == "1" and (settings.get("-race") == "true") == race
        assert x["environment"]["GOPROXY"] == "off" and x["environment"]["GOFLAGS"] == "-mod=readonly"
        if f in servers:
            r = read(f.with_name(f.name.removesuffix("-provenance.json")+".json"))
            ready = r["ready"]
            assert r["binary_sha256"] == x["executable_sha256"] and r["race_children"] == race
            assert ready["PID"] == x["PID"] and r["command"] == x["argv"]
            args = x["argv"]
            assert args[args.index("--db")+1] == ready["Store"] and args[args.index("--token")+1] == ready["Token"]
            assert (args[args.index("--response-mode")+1] == "finish-accepted") == f.name.startswith("TestPhase2BDelayedApplication-")
            endpoints[ready["Token"]] = (x["PID"],f.name.split("-phase2-endpoint-")[0])
    for f in directory.glob("*.jsonl"):
        for row in map(json.loads,f.read_text().splitlines()):
            pid,case = endpoints[row["Process"]]
            assert pid == row["PID"] and f.name.startswith(case+"-")
    return len(starts), len(servers)


def controlled_errors(directory, signals):
    f = directory/"TestPhase2BLiveOrders_idempotent_A-source-live.json"
    x = read(f)
    saved = dump(f.with_name(f.name.removesuffix("-live.json")), "final")
    def check_live(v): live(v, "A-source", saved)
    def premature_q(v):
        e = v["events"]
        e.remove(Q)
        e.insert(e.index(FREE),Q)
    prefix = directory/"TestPhase2BDelayedApplication"
    delayed = read(prefix.with_name(prefix.name+"-delayed.json"))
    t,w,log = dump(prefix,"acknowledged")
    op = '["client",1,1,0]'
    wire = identity(t,"external",op)[2]
    xdir = read(EVIDENCE/"commands/direct-final-normal.json.gz")
    (two,) = logs(xdir,"first_H")
    checks = [
        ("Q before released H",x,premature_q,check_live),
        ("successful wait while held",x,lambda v:v.update(wait_error=""),check_live),
        ("missing accepted packet", [log],lambda v:v.clear(),lambda v:contact_rows(w,"".join(v),[wire],pending=True)),
        ("wrong original wire", [wire],lambda v:v.__setitem__(0,wire+" "),lambda v:contact_rows(w,log,v,pending=True)),
        ("different kernel signal",signals,lambda v:one(v,pid=delayed["executor_barrier"]["PID"]).update(exit_signal="terminated"),
         lambda v:closed(prefix,delayed,"delayed","external",v,False)),
        ("reusable process owner",delayed,lambda v:v["reopened"][0].update(PID=v["crashed"][0]["PID"]),
         lambda v:closed(prefix,v,"delayed","external",signals,False)),
        ("merged duplicate lifetime",two,lambda v:v.update(q_pending_with_one_direct_left=False),direct),
    ]
    for name,original,mutate,check in checks:
        check(original)
        corrupt = copy.deepcopy(original)
        mutate(corrupt)
        try:
            check(corrupt)
        except (AssertionError, ValueError):
            continue
        raise AssertionError("escaped mutation: "+name)
    return len(checks)


def audit(regime):
    import audit_withholding
    race = regime == "race"
    d = EVIDENCE/("process-"+regime)
    out = {}
    commands = {}
    for family in ("world","composition","consumer","direct"):
        cmd = read(EVIDENCE/"commands"/f"{family}-final-{regime}.json.gz")
        assert cmd["exit"] == 0 and ("-race" in cmd["argv"]) == race and "--- SKIP" not in cmd["output"]
        commands[family] = cmd
    sigs = [json.loads(l.split(" DURABLE ",1)[1]) for l in commands["world"]["output"].splitlines()
            if " DURABLE " in l and '"exit_signal"' in l]
    assert len(sigs) == len({s["pid"] for s in sigs}) == 17
    out["executor_starts"], out["world_endpoint_starts"] = provenance(d,commands["world"],race)
    _, out["composition_endpoint_starts"] = provenance(EVIDENCE/("composition-process-"+regime),commands["composition"],race)
    for route in ("adapter", "idempotent"):
        for case in sorted(LIVE | ({"returned-unknown"} if route == "adapter" else set())):
            prefix = d/f"TestPhase2BLiveOrders_{route}_{case}"
            live(read(prefix.with_name(prefix.name+"-live.json")),case,dump(prefix,"final"))
        open_success(d/f"TestPhase2BOpenSuccess_{route}","adapter" if route == "adapter" else "external")
    out.update(live_orders=19,open_consumers=2)
    g = d/"TestPhase2BSupersededOpportunity"
    generations(g,read(g.with_name(g.name+"-generations.json")))
    out["old_and_new_generation"] = 1
    prefix = d/"TestPhase2BRevokedCrashPair"
    pair = [closed(prefix,read(d/(prefix.name+"-"+b+".json")),b,"external",sigs,race) for b in ("U0","U1")]
    assert pair[0] == pair[1]
    for kind in ("adapter","idempotent"):
        prefix = d/("TestPhase2BRecordedRecovery_"+kind)
        closed(prefix,read(d/(prefix.name+"-known.json")),"known","external" if kind == "idempotent" else "adapter",sigs,race)
    prefix = d/"TestPhase2BDelayedApplication"
    closed(prefix,read(d/(prefix.name+"-delayed.json")),"delayed","external",sigs,race)
    open_pairs = collections.defaultdict(dict)
    crash_files = [*d.glob("TestPhase2KnownEvidenceCrashCuts*-known.json"),
                   *d.glob("*-parent-crash.json"), *d.glob("TestPhase2PairedCrashCuts*-U?.json")]
    for f in crash_files:
        x = read(f)
        cut = prior.crash(f,x,sigs,race)
        if cut:
            open_pairs[x["route"]][x["branch"]] = cut
    assert len(crash_files) == 12 and set(open_pairs) == {"idempotent","non-idempotent","adapter"}
    assert all(p["U0"] == p["U1"] for p in open_pairs.values())
    out.update(sigkill_recoveries=17,open_pairs=3,revoked_pair=1,q_alone_known=2,late_packet=1)
    replays = list(d.glob("*-replay-*.json"))
    for f in replays:
        checked_replay(f)
    assert len(replays) == 38
    out["replays"] = len(replays)
    adapter_modes = set()
    for f in d.glob("TestPhase2AdapterRealErrorAndSelectedUnsuccessful*-result.json"):
        x = read(f)
        adapter_modes.add((x["adapter_mode"],x["endpoint_mode"]))
        t, w = [db(d/Path(x[k]).name) for k in ("local_store","world_store")]
        (r,) = t["adapter_attempts"]
        op = r["intent_id"]
        _, _, wire = identity(t,"adapter",op)
        (app,) = contact_rows(w,(d/Path(x["journal"]).name).read_text(),[wire])
        mode = x["adapter_mode"]
        kept = mode != "unknown"
        disposition = "transport_error" if x["endpoint_mode"]=="drop-after-commit" else "selected_error" if mode=="selected-error" else "validated"
        outcome(t,"adapter",op,app if kept else None,disposition=disposition,settled=kept,consumed=kept)
    assert len(adapter_modes) == 4
    out["inherited_adapter_results"] = 4
    inherited = EVIDENCE/("authority-"+regime)
    auth_files = list(inherited.glob("TestPhase2AAuthority-*.gz"))
    assert len(auth_files) == 30
    for f in auth_files: historical_authority(read(f),race)
    withheld = list(inherited.glob("TestPhase2AClosedWithholding-*.gz"))
    assert len(withheld) == 2
    for f in withheld: audit_withholding.verify(read(f),regime)
    out["inherited_authority"] = 30
    out["inherited_withholding"] = 2
    comp = list((EVIDENCE/("composition-"+regime)).glob("*.json.gz"))
    assert len(comp) == 9
    for f in comp: composition(f)
    out["composition_and_compatibility"] = len(comp)
    observed = logs(commands["consumer"],"host_output")
    assert sorted(x["mode"] for x in observed) == ["selected-error","success","unknown"]
    for x in observed: consumer(x)
    out["public_observers"] = len(observed)
    (two,) = logs(commands["direct"],"first_H")
    direct(two)
    for test in ("TwoDirectOpportunities","MixedDuplicatesAndQuiescence","CloseBeforeDirectAdmission",
                 "ClosePinsDirectAndReopenKeepsAuthority","StickyAccountingAndRecoveredSettlement",
                 "RecoveredAccounting/external-Q","RecoveredAccounting/external-Run","RecoveredAccounting/external-direct",
                 "RecoveredAccounting/adapter-Q","RecoveredAccounting/adapter-Run",
                 "RecoveredSourceMismatch/external","RecoveredSourceMismatch/adapter"):
        assert "--- PASS: TestPhase2B"+test+" " in commands["direct"]["output"]
    assert commands["direct"]["output"].count("genuine recovery settlement error, repair, sticky same-owner Q error") == 5
    out["two_direct_snapshots"] = 2
    if not race:
        out["rejected_mutations"] = controlled_errors(d,sigs)
    return out


if __name__ == "__main__":
    if not __debug__:
        raise SystemExit("run with Python assertions enabled")
    for regime in sys.argv[1:] or ["normal","race"]:
        assert regime in {"normal","race"}
        print(regime, json.dumps(audit(regime),sort_keys=True))
