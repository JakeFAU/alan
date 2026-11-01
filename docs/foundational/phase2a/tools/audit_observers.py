#!/usr/bin/env python3
"""Independent observer checks. Run without -O. Requires only Python's stdlib."""
import argparse
import collections
import copy
import gzip
import json
from pathlib import Path
import sqlite3
import tempfile

EVIDENCE = Path(__file__).resolve().parents[1] / "evidence"


def read(path):
    data = gzip.decompress(path.read_bytes()) if path.suffix == ".gz" else path.read_bytes()
    return json.loads(data)


def db(path):
    with tempfile.NamedTemporaryFile() as tmp:
        tmp.write(gzip.decompress(path.read_bytes()))
        tmp.flush()
        cx = sqlite3.connect(f"file:{tmp.name}?immutable=1", uri=True)
        cx.row_factory = sqlite3.Row
        assert cx.execute("pragma integrity_check").fetchone()[0] == "ok"
        names = cx.execute("select name from sqlite_master where type='table' order by name")
        result = {}
        for (name,) in names.fetchall():
            result[name] = [dict(row) for row in cx.execute(f'SELECT * FROM "{name}" ORDER BY rowid')]
        cx.close()
        return result


def one(rows, **match):
    (row,) = [r for r in rows if all(r[k] == v for k, v in match.items())]
    return row


def run(t, who, gen=1):
    return one(t["runs"], subject=who, generation=gen)


def projection(t, who, gen=1):
    return json.loads(run(t, who, gen)["projection"])


def selected(t, op):
    """Admitted envelope and matching committed history; join target and sequence."""
    observations = [o for o in t["observations"] if json.loads(o["descriptor"])["Operation"] == op]
    taken = []
    for o in observations:
        history = [h for h in t["history"] if h["selected_observation"] == o["id"]]
        if not history:
            assert o["consumed_sequence"] is None
        else:
            (h,) = history
            assert (h["subject"], h["generation"], h["sequence"]) == (
                o["subject"], o["generation"], o["consumed_sequence"])
            taken.append(json.loads(o["descriptor"]))
    return taken


def effect(t, route, op):
    intent = json.loads(one(t["intents"], id=op)["descriptor"])
    emit = one(t["history"], subject=intent["Run"]["Subject"], generation=intent["Run"]["Generation"],
               sequence=intent["Sequence"])
    assert intent in json.loads(emit["intents"])
    auth = json.loads(one(t["attempts"], intent_id=op)["result"])
    a = json.loads(one(t[route + "_attempts"], intent_id=op)["descriptor"])
    assert a["Intent"] == intent and a["AuthorizationID"] == auth["ID"] and auth["IntentID"] == op
    if route == "adapter":
        req = a["Prepared"]["Request"]
        assert req["ID"] == op
        wire = a["Prepared"]["Wire"]
    else:
        req = a["Request"]
        wire = json.dumps(req, separators=(",", ":"))
    metadata, = t["metadata"]
    assert json.loads(req["Key"]) == [metadata["key_version"], metadata["namespace"], op]
    return a, auth, wire


def joined_receipt(t, route, op, receipt, mode="validated"):
    if route == "external":
        assert json.loads(one(t["external_receipts"], intent_id=op)["descriptor"]) == receipt
        return
    result = json.loads(one(t["adapter_results"], intent_id=op)["descriptor"])
    c = result["Candidate"]
    assert c["ID"] == op and c["Key"] == receipt["Key"] and c["Disposition"] == mode
    assert result["Values"]["valid"]["Flag"] == (mode == "validated")
    if mode == "transport_error":
        assert c["HTTPStatus"] == 0 and "EOF" in c["Response"]
    else:
        assert c["HTTPStatus"] == 200 and json.loads(c["Response"]) == receipt


def parent(t, expected):
    root = projection(t, "root")
    d = root["Data"]
    op = d["child_operation"]["Text"]
    selection = [o for o in selected(t, op) if o["Run"]["Subject"] == "root"]
    if expected == "open":
        assert selection == [] and root["Status"] == "open" and root["PC"] == 2
        assert "child_status" not in d and d["knowledge"]["Text"] == "may_have_applied"
        return
    (o,) = selection
    payload = o["Payload"]
    assert o["Run"] == {"Subject": "root", "Generation": 1} and o["Kind"] == "invocation"
    assert o["Provenance"] == "runtime-invocation" and o["Obligation"] == "child"
    assert one(t["composition_outcome_deliveries"], operation_id=op)["observation_id"] == o["ID"]
    assert root["Status"] == "succeeded" and (root["PC"] == 6 if expected == "succeeded" else root["PC"] == 9)
    assert payload["status"]["Text"] == d["child_status"]["Text"] == expected
    assert payload["reason"]["Text"] == d["child_reason"]["Text"] == "member_" + expected
    assert payload["response_present"]["Flag"] == d["child_response_present"]["Flag"] == (expected == "succeeded")
    if expected == "succeeded":
        assert d["knowledge"]["Text"] == "historical_application_receipt"
        for name in ["total", "mutation"]:
            assert d["child_" + name]["Number"] == payload["response." + name]["Number"] == 1
    else:
        assert set(payload) == {"status", "reason", "response_present"}
        assert d["knowledge"]["Text"] == "local_failure_world_unresolved"


def parent_case(f):
    x = read(f)
    if "before" in x:
        t, end = x["before"]["Tables"], x["after"]["Tables"]
        parent(t, "failed")
        parent(end, "failed")
        (attempt,) = t["adapter_attempts"]
        op = attempt["intent_id"]
        child = json.loads(op)[0]
        a, _, wire = effect(t, "adapter", op)
        assert a["Allowed"] and not t["adapter_results"] and not selected(end, op)
        assert projection(t, child) == projection(end, child) and projection(t, child)["Status"] == "failed"
        assert projection(t, "root") == projection(end, "root")
        assert [json.loads(wire)] == x["oracle_requests"]
        joined_receipt(end, "adapter", op, x["oracle_receipts"][0])
        return
    t = x["inspection"]["Tables"]
    requests = x["oracle_requests"]
    receipts = x["oracle_receipts"]
    assert [r["Delta"] for r in requests] == [1, 2] and [r["Total"] for r in receipts] == [1, 3]
    assert x["physical_cap"] == 1
    for i, request in enumerate(requests):
        receipt = receipts[i]
        assert all(receipt[k] == v for k, v in request.items()) and receipt["Mutation"] == i + 1
        op = json.loads(request["Key"])[2]
        a, au, wire = effect(t, "adapter", op)
        assert a["Allowed"] and au["Allowed"] and json.loads(wire) == request
        if i == 1:
            assert json.loads(op)[0] == "peer"
            assert projection(t, "peer")["Status"] == "succeeded"
            assert len(selected(t, op)) == 1
            joined_receipt(t, "adapter", op, receipt)
            continue
        mode = {"completed": "validated", "real_postcommit_connection_loss": "transport_error",
                "selected_unsuccessful": "selected_error", "unrecorded_unknown": "unknown"}[x["boundary"]]
        if mode == "unknown":
            assert not any(r["intent_id"] == op for r in t["adapter_results"]) and not selected(t, op)
            parent(t, "open")
        else:
            joined_receipt(t, "adapter", op, receipt, mode)
            (o,) = selected(t, op)
            assert o["Payload"]["valid"]["Flag"] == (mode == "validated")
            if mode != "validated":
                assert all(o["Payload"][k]["Number"] == 0 for k in ("total", "mutation"))
            parent(t, "succeeded" if mode == "validated" else "failed")


def authority(x):
    case = x["case"]
    route, cut, target, operation, member = [case[k] for k in ("Route", "Cut", "Target", "Operation", "Member")]
    t = x["final"]["Tables"]
    op = x["operation"]
    a, auth, wire = effect(t, route, op)
    assert projection(x["initial"]["Tables"], "client")["Status"] == "open"
    pre = cut == "before-permission"
    close_reason = "source_retired" if operation == "RetireSubject" else "source_closed"
    reason = (close_reason if pre else "effect_limit" if cut == "budget" else
              "scope_closed" if cut == "after-permission" and member else "authorized")
    assert (auth["Reason"], auth["Allowed"]) == ((close_reason, False) if pre else ("authorized", True))
    assert (a["Reason"], a["Allowed"]) == (reason, reason == "authorized")
    wanted_apps = 0 if reason != "authorized" or cut == "context" else 2 if cut == "retry" else 1
    wanted_posts = 2 if cut == "relaunch" else wanted_apps
    events = [json.loads(line) for line in x["ingress_journal"].splitlines()]
    posts = [e for e in events if e["Event"] == "ingress"]
    commits = [e for e in events if e["Event"] == "committed"]
    assert len(posts) == len(commits) == x["world"]["Contacts"] == wanted_posts
    apps = x["world"]["Applications"] or []
    assert len(apps) == wanted_apps
    for i, (post, commit) in enumerate(zip(posts, commits)):
        assert (post["Method"], post["Path"]) == ("POST", "/apply")
        assert (post["Contact"], post["PID"]) == (commit["Contact"], commit["PID"])
        assert commit["Receipt"] == apps[min(i, len(apps)-1)]
        if cut != "retry" or i == 0:
            assert post["Body"] == wire
    assert bool(t.get("composition_static_members")) == member
    if member:
        assert {r["subject"] for r in t["composition_static_members"]} == {"client", "scope"}
        charges = t["composition_charges"]
        assert len(charges) == wanted_apps
        if charges:
            assert charges[0]["operation_id"] == op and json.loads(charges[0]["descriptor"])["Effects"] == 1
    if cut in ("budget", "context"):
        assert not t["lifecycle_results"]
    else:
        decisions = [json.loads(r["result"]) for r in t["lifecycle_results"]]
        d = one(decisions, DecisionSequence=1)
        assert d["Accepted"] and d["Request"]["Operation"] == operation and d["Request"]["ExpectedRevision"] == 1
        one(t["permissions"], source="operator", target=target, operation=operation, revision=1, kind="lifecycle")
        assert d["Request"]["Intent"]["Run"]["Subject"] == "operator" and d["Request"]["Intent"]["Target"] == target
        assert d["Before"]["Run"] == d["After"]["Run"] == {"Subject": target, "Generation": 1}
        assert d["Before"]["Projection"]["Status"] == "open" and d["After"]["Projection"]["Status"] == "cancelled"
        assert d["After"]["Retired"] == (operation == "RetireSubject")
        snap = x.get("closed_before_run") or x.get("closed_before_relaunch") or x["closed_held"]
        st = snap["local"]["Tables"]
        assert projection(st, target)["Status"] == "cancelled"
        if target == "scope":
            assert projection(st, "client")["Status"] == "open"
        if cut in ("before-permission", "after-permission"):
            assert not st[route + "_attempts"] and not snap["world"]["Contacts"]
            perms = [v for v in st["attempts"] if v["intent_id"] == op]
            assert len(perms) == (cut == "after-permission")
            if perms:
                assert json.loads(perms[0]["result"]) == auth
        if cut in ("worker", "response", "retry", "ingress"):
            assert not st["external_receipts"] and not st["adapter_results"]
            assert a == json.loads(one(st[route + "_attempts"], intent_id=op)["descriptor"])
            assert len(snap["world"]["Applications"] or []) == (cut in ("response", "retry"))
            order = [(v["Boundary"], v["Phase"], v["Intent"]) if isinstance(v, dict) else v for v in x["order"]]
            release = order.index("release after committed closure")
            decided = order.index(("lifecycle.decided", "accepted", d["Request"]["Intent"]["ID"]))
            pre_call = order.index((route + ".authorized", "pre-transport", op))
            assert pre_call < decided < release
            response = order.index((route + ".response", "pre-record", op))
            if cut in ("worker", "ingress"):
                assert response > release
            else:
                assert response < decided
            if cut == "ingress":
                assert x["endpoint_barrier"]["Intent"] == json.loads(wire)["Key"]
                assert x["endpoint_barrier"]["Boundary"] == "resource.ingress"
        if cut == "relaunch":
            assert snap["world"]["Contacts"] == 1 and len(snap["world"]["Applications"]) == 1
            assert not st["external_receipts"] and not selected(st, op)
            assert posts[0]["Body"] == posts[1]["Body"] and apps == snap["world"]["Applications"]
    if wanted_apps:
        joined_receipt(t, route, op, apps[0])
    if cut == "retry":
        d = one(decisions, DecisionSequence=2)
        assert d["Accepted"] and d["Request"]["Operation"] == "Retry" and d["Successor"] == 2
        successor = one(t["intents"], subject="client", generation=2)["id"]
        a2, _, w2 = effect(t, route, successor)
        assert a2["Request"]["Generation"] == 2 and a2["Request"]["Delta"] == a["Request"]["Delta"]
        assert posts[1]["Body"] == w2 and w2 != wire and apps[0]["Key"] != apps[1]["Key"]
        assert len(selected(t, successor)) == 1
    assert len(selected(t, op)) == int(target == "scope" or cut == "budget")


def process_identity(directory, x, signals, race):
    b = x["barrier"]
    pid, fresh = b["PID"], x["recovery_PID"]
    assert pid == x["crash_PID"] and pid != fresh
    signal = one(signals, pid=pid)
    assert signal["exit_signal"] == "killed" and signal["race_children"] == race
    assert signal["barrier_config"]["Subject"] in ("", b["Subject"])
    for k in ("Token", "Scenario", "Store", "Boundary", "Phase"):
        assert signal["barrier_config"][k] == b[k]
    for p in (pid, fresh):
        start = read(next(directory.glob(f"*-process-start-{p}.json")))
        assert start["PID"] == p and start["executable_sha256"] == signal["binary_sha256"]
        assert start["environment"]["ALAN_PHASE2_DB"] == b["Store"]
        assert start["argv"] == signal["command"]
        settings = {r["Key"]: r.get("Value") for r in start["build_info"]["Settings"]}
        assert (settings.get("-race") == "true") == race
    finish = read(next(directory.glob(f"*-process-{fresh}.json")))
    assert finish["PID"] == fresh and finish["exit_code"] == 0 and finish["race_children"] == race
    report, = [json.loads(line.removeprefix("PHASE2_RESULT ")) for line in finish["output"].splitlines()
               if line.startswith("PHASE2_RESULT ")]
    assert report["PID"] == fresh and report["Error"] == "<nil>"
    return report["Report"]


def world(t, log, wire):
    requests = [json.loads(row["request"]) for row in t["contacts"]]
    apps = [json.loads(row["receipt"]) for row in t["applications"]]
    records = [json.loads(line) for line in log.splitlines()]
    ingresses = [r for r in records if r["Event"] == "ingress"]
    done = [r for r in records if r["Event"] == "committed"]
    assert len(records) == 2 * len(requests) and len(ingresses) == len(requests) == len(done)
    for i, (req, ingress, event) in enumerate(zip(requests, ingresses, done), 1):
        assert ingress["Body"] == wire and ingress["Path"] == "/apply" and ingress["Method"] == "POST"
        assert all(req[k] == v for k, v in json.loads(wire).items())
        assert event["Contact"] == ingress["Contact"] == i and event["PID"] == ingress["PID"]
        assert event["Receipt"] in apps and event["Receipt"]["Key"] == req["Key"]
    return apps


def crash(f, x, signals, race):
    report = process_identity(f.parent, x, signals, race)
    t, end, wc, wa = [db(f.parent / Path(x[k]).name) for k in
                      ("cut_local_store", "after_local_store", "cut_world_store", "after_world_store")]
    b = x["barrier"]
    op = b["Intent"]
    route = b["Boundary"].split(".")[0]
    attempt, authorization, wire = effect(t, route, op)
    assert attempt["Allowed"] and b["Attempt"] == authorization["ID"]
    assert attempt["Intent"]["Run"] == {"Subject": b["Subject"], "Generation": b["Generation"]}
    assert b["Epoch"] == run(t, b["Subject"])["epoch"] and b["Sequence"] == attempt["Intent"]["Sequence"]
    assert run(end, b["Subject"])["epoch"] == b["Epoch"] + 1 and not selected(t, op)
    assert effect(end, route, op) == (attempt, authorization, wire)
    initial = x.get("cut") or x["stopped"]
    # Both the physical SQLite images and the complete public inspections must match between cuts.
    for snap, table in ((initial, t), (x["after"], end)):
        assert set(snap["Tables"]) == set(table)
        for name, rows in table.items():
            display = [{k: v.decode() if isinstance(v, bytes) else v for k, v in r.items()} for r in rows]
            assert display == snap["Tables"][name]
    branch = x.get("branch")
    is_parent = f.name.endswith("-parent-crash.json")
    stage = x.get("stage")
    prefix = b["Scenario"].replace("/", "_")
    label = branch if branch else "parent" if is_parent else "known"
    cut_log = read(f.parent / f"{prefix}-replay-{label}-cut.json")["ingress_before"]
    end_log = (f.parent / Path(x["journal"]).name).read_text()
    assert end_log.startswith(cut_log)
    before_apps = world(wc, cut_log, wire)
    after_apps = world(wa, end_log, wire)
    table = "external_receipts" if route == "external" else "adapter_results"
    if branch:
        n = int(branch[-1])
        assert b["Boundary"] == route + (".response" if n else ".authorized")
        assert len(before_apps) == len(wc["contacts"]) == n
        assert not t["observations"] and not t["settlements"] and not t[table]
        assert not t.get("composition_charges") and len(t["history"]) == len(t["intents"]) == 1
        assert not x["endpoint_initial"]["Contacts"] and not x["endpoint_initial"]["Applications"]
        assert len(wa["contacts"]) == n + (x["route"] == "idempotent")
        if x["route"] != "idempotent":
            assert t["observations"] == end["observations"] == [] and not end[table]
            assert wc == wa and projection(end, "client") == projection(t, "client")
            unknown = report["UnknownAdapters" if route == "adapter" else "UnknownExternal"]
            assert len(unknown) == 1
            return (t, initial, x["initial"])
        assert len(after_apps) == 1 and (not n or before_apps == after_apps)
    elif is_parent:
        parent(t, "open")
        assert wc == wa and run(end, "root")["epoch"] == run(t, "root")["epoch"] + 1
        if stage == "unknown":
            assert not before_apps and not t[table] and not end[table] and not selected(end, op)
            parent(end, "open")
            return
        assert stage == "recorded" and not t["settlements"]
        parent(end, "succeeded")
    else:
        assert wc == wa and len(wc["contacts"]) == len(before_apps) == 1
        assert len(t["settlements"]) == len(t["observations"]) == (stage == "settled")
    if not branch:
        joined_receipt(t, route, op, before_apps[0])
    joined_receipt(end, route, op, after_apps[0])
    (o,) = selected(end, op)
    assert o["Payload"]["total"]["Number"] == after_apps[0]["Total"]
    assert projection(end, b["Subject"])["Status"] == "succeeded" and len(end[table]) == 1
    if branch:
        return (t, initial, x["initial"])


def endpoint_control(directory, x):
    t = db(directory / Path(x["world_store"]).name)
    ledger = [json.loads(row["receipt"]) for row in t["applications"]]
    journal = [json.loads(line) for line in (directory / Path(x["journal"]).name).read_text().splitlines()]
    posts, replies = journal[::2], journal[1::2]
    assert len(posts) == len(replies) == 5 and len(t["contacts"]) == 4
    assert all(p["Event"] == "ingress" and p["Method"] == "POST" and p["Path"] == "/apply" for p in posts)
    assert [r["Event"] for r in replies] == ["committed", "committed", "rejected", "committed", "committed"]
    requests = [json.loads(r["Body"]) for r in posts]
    first, duplicate, altered, later, last = requests
    assert first == duplicate == last and altered == dict(first, Delta=6) and first["Delta"] == 5
    assert first["Key"] != later["Key"] and dict(first, Key=later["Key"]) == later
    assert [json.loads(row["request"]) for row in t["contacts"]] == [first, duplicate, later, last]
    assert replies[2]["Status"] == 409
    assert [r["Total"] for r in ledger] == [5, 10]
    assert replies[0]["Receipt"] == replies[1]["Receipt"] == replies[4]["Receipt"] == ledger[0]
    assert replies[3]["Receipt"] == ledger[1]
    assert posts[0]["PID"] != posts[1]["PID"] and len({p["PID"] for p in posts[1:]}) == 1
    stopped = read(next(directory.glob("*-endpoint-stopped-" + str(posts[0]["PID"]) + ".json")))
    assert stopped["signal"] == "killed"


def audit(regime):
    directory = EVIDENCE / ("process-" + regime)
    cmd = read(EVIDENCE / "commands" / f"process-{regime}-reviewfix.json.gz")
    signals = [json.loads(line.split(" DURABLE ", 1)[1]) for line in cmd["output"].splitlines()
               if " DURABLE " in line and '"exit_signal"' in line]
    assert len(signals) == len({r["pid"] for r in signals}) == 12
    pairs = collections.defaultdict(dict)
    counts = collections.Counter()
    recorder = read(EVIDENCE / "commands" / f"inherit-{regime}.json.gz")
    altered = set()
    for line in recorder["output"].splitlines():
        if " RECORDER " in line:
            result = json.loads(line.split(" RECORDER ", 1)[1])
            assert result["error"] == "identity conflict" and result["original"] == result["retained"]["Candidate"]
            (field,) = [k for k, v in result["original"].items() if result["offered"][k] != v]
            altered.add(field)
            counts["recorder_refusals"] += 1
    assert altered == {"Key", "Payload"}
    for f in sorted(directory.glob("*.json")):
        x = read(f)
        if "crash_PID" in x:
            cut = crash(f, x, signals, regime == "race")
            counts["killed_fresh_recoveries"] += 1
            if cut:
                pairs[x["route"]][x["branch"]] = cut
        elif f.name.endswith("-endpoint-controls.json"):
            endpoint_control(directory, x)
            counts["endpoint_controls"] += 1
        elif "ingress_before" in x:
            for kind in ("world", "local"):
                assert db(directory / Path(x[f"before_{kind}_store"]).name) == db(directory / Path(x[f"after_{kind}_store"]).name)
            assert x["ingress_before"] == x["ingress_after"]
            counts["replays"] += 1
        elif f.name.endswith("-result.json") and "adapter_mode" in x:
            t, w = [db(directory / Path(x[k]).name) for k in ("local_store", "world_store")]
            (r,) = t["adapter_attempts"]
            op = r["intent_id"]
            _, _, wire = effect(t, "adapter", op)
            (receipt,) = world(w, (directory / Path(x["journal"]).name).read_text(), wire)
            mode = x["adapter_mode"]
            if mode == "unknown":
                assert not t["adapter_results"] and not selected(t, op)
            else:
                joined_receipt(t, "adapter", op, receipt, "transport_error" if x["endpoint_mode"] == "drop-after-commit"
                               else "selected_error" if mode == "selected-error" else "validated")
                assert len(selected(t, op)) == 1
            counts["adapter_result_modes"] += 1
    assert set(pairs) == {"idempotent", "non-idempotent", "adapter"}
    for pair in pairs.values():
        assert pair["U0"] == pair["U1"]
        counts["equal_pairs"] += 1
    expected = set()
    for f in (EVIDENCE / ("authority-" + regime)).glob("*.gz"):
        x = read(f)
        authority(x)
        c = x["case"]
        expected.add(tuple(c[k] for k in ("Route", "Cut", "Target", "Operation", "Member")))
        counts["authority_cases"] += 1
    schedules = {("relaunch", "client", "CancelRun", False), ("retry", "client", "CancelRun", False)}
    shared = {
        ("after-permission", "client", "CancelRun", False), ("after-permission", "client", "CancelRun", True),
        ("after-permission", "scope", "CancelRun", True), ("after-permission", "scope", "RetireSubject", True),
        ("before-permission", "client", "CancelRun", False), ("before-permission", "client", "RetireSubject", False),
        ("budget", "client", "CancelRun", True), ("context", "client", "CancelRun", False),
        ("ingress", "client", "CancelRun", False), ("response", "client", "CancelRun", False),
        ("worker", "client", "CancelRun", False), ("worker", "client", "RetireSubject", False),
        ("worker", "client", "CancelRun", True), ("worker", "scope", "CancelRun", True),
    }
    population = {("external", *v) for v in schedules | shared} | {("adapter", *v) for v in shared}
    assert expected == population
    parent_dir = EVIDENCE / "parent" / ("normal" if regime == "normal" else "race-final")
    for f in parent_dir.glob("*.gz"):
        parent_case(f)
        counts["parent_probes"] += 1
    assert counts == dict(killed_fresh_recoveries=12, equal_pairs=3, replays=28,
                          recorder_refusals=2, endpoint_controls=1, adapter_result_modes=4, authority_cases=30, parent_probes=5), counts
    return counts


def controls():
    """In-memory mutations of genuine records challenge important observer joins."""
    a = read(next((EVIDENCE / "authority-normal").glob("*adapter-worker-CancelRun-client-member-true*")))
    p = read(EVIDENCE / "parent/normal/selected_unsuccessful.json.gz")["inspection"]["Tables"]

    def late_release(x):
        x["order"].insert(0, x["order"].pop(x["order"].index("release after committed closure")))

    def wrong_ingress(x):
        x["ingress_journal"] = x["ingress_journal"].replace('"Path":"/apply"', '"Path":"/other"')

    def forged_parent(t):
        for h in t["history"]:
            if h["subject"] == "root":
                h["selected_observation"] = None

    f = next((EVIDENCE / "process-normal").glob("*idempotent-U0.json"))
    record = read(f)
    cmd = read(EVIDENCE / "commands/process-normal-reviewfix.json.gz")
    signals = [json.loads(line.split(" DURABLE ", 1)[1]) for line in cmd["output"].splitlines()
               if " DURABLE " in line and '"exit_signal"' in line]
    def wrong_signal(rows):
        one(rows, pid=record["crash_PID"])["exit_signal"] = "terminated"

    for name, value, change, check in (
        ("different operating-system signal", signals, wrong_signal,
         lambda rows: process_identity(f.parent, record, rows, False)),
        ("release before closure", a, late_release, authority),
        ("different endpoint path", a, wrong_ingress, authority),
        ("missing selection despite matching projection", p, forged_parent, lambda t: parent(t, "failed")),
    ):
        value = copy.deepcopy(value)
        change(value)
        try:
            check(value)
        except (AssertionError, ValueError):
            continue
        raise AssertionError("undetected counter-control: " + name)


if __name__ == "__main__":
    if not __debug__:
        raise SystemExit("Python assertions must be enabled")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("regimes", nargs="*", default=["normal", "race"])
    for regime in parser.parse_args().regimes:
        print(regime, json.dumps(audit(regime), sort_keys=True))
    controls()
    print("Rejected four deliberate in-memory observer corruptions.")
