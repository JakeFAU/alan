"""Reconstruct the bounded relation from the protocol, public programs and Store guards.

This checker consumes only JSON. An edge under check never supplies an action,
an owner or a target to the expected relation. A profile selects the environment.
"""
import argparse
from collections import Counter, deque
from copy import deepcopy
import gzip
import json
from pathlib import Path

BASE = Path(__file__).resolve().parents[1] / "evidence/model"
OP = "N/source/1/emit/0"
REPLY = "N/source/1/reply/0"
INV = "N/parent/1/invoke/0"
PAIR = "unknown-plus-responsive-single-slot"
YES = "authorized"
RECEIPT = "receipt:first-application"
SUCCESS = "succeeded/member_succeeded/confirmed"
FAILURE = "failed/member_failed/no-response"
CLOSED = {"cancelled", "retired", "failed", "ended"}

LOCAL = dict.fromkeys(("ResultDetail", "ParentDecision", "Permission", "Reservation",
                       "Result", "Reply", "Published", "ParentSelected"), "")
LOCAL.update(Program="emit", Ancestor="open", Epoch=1, PermissionEpoch=0, Charge=0,
             Settled=False, Selected=False, PeerDone=False)
START = dict(Local=LOCAL, Volatile=dict(Process=1, Run=1, Work=""),
             World=dict(Ingress=0, ValidRequests=0, Applications=0),
             Oracle=dict.fromkeys(("WorkerEntries", "Executes", "Launches", "HTTPCalls",
                                   "Records", "Kills"), 0))


class Rejected(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise Rejected(message)


def profiles():
    result = {}
    flags = "Idempotent Member NoGrant ZeroLimit Crash SecondRun Peer Parent PrivateFault".split()

    def add(name, route, closure="", reply="", **changes):
        result[name] = dict.fromkeys(flags, False)
        result[name].update(Name=name, Route=route, Closure=closure, Reply=reply, **changes)

    for route in ("raw", "adapter"):
        for target in ("source", "retire-source", "ancestor", "retire-ancestor"):
            add(f"{route}-member-{target}", route, target, Member=True)
            if "source" in target:
                add(f"{route}-unscoped-{target}", route, target)
        add(route+"-recovery-plain", route, Crash=True)
        add(route+"-recovery-keyed", route, Crash=True, Idempotent=True)
        add(route+"-no-grant", route, Member=True, NoGrant=True)
        add(route+"-zero-limit", route, Member=True, ZeroLimit=True)
        add(route+"-peer", route, Idempotent=True, Peer=True)
    add("raw-second-run", "raw", "source", Idempotent=True, SecondRun=True)
    parent = dict(Member=True, Peer=True, Parent=True)
    for outcome in ("ok", "early-error", "loss", "chosen-error", "unknown"):
        add("parent-"+outcome, "adapter", reply=outcome, **parent)
    add("parent-recovery", "adapter", Crash=True, **parent)
    add("parent-private-fault", "adapter", reply="unknown", PrivateFault=True, **parent)
    add("parent-private-fault-receipt", "adapter", reply="ok", PrivateFault=True,
        Idempotent=True, **parent)
    return result


def single(p, s):
    """Each alternative has a new successor copied from the unchanged input state."""
    l, v, w, o = (s[k] for k in ("Local", "Volatile", "World", "Oracle"))
    pc, work, perm, attempt = l["Program"], v["Work"], l["Permission"], l["Reservation"]
    raw = p["Route"] == "raw"
    live = pc not in CLOSED
    alternatives = {}

    def offer(name, obj=OP, **patches):
        target = deepcopy(s)
        for section, changes in patches.items():
            target[section].update(changes)
        key = (None, name, obj)
        require(key not in alternatives, f"duplicate rule {key}")
        alternatives[key] = target

    def move(name, stage, counters=None, **fields):
        offer(name, Volatile={"Work": stage}, Oracle={
            k: o[k]+1 for k in counters or ()}, **fields)

    if v["Process"] == 0:
        offer("host.reopen", "executor", Volatile=dict(Process=2, Run=2, Work=""),
              Local=dict(Epoch=l["Epoch"]+int(live)))
        return alternatives

    if p["Peer"] and not l["PeerDone"]:
        offer("peer.end", "peer/1", Local=dict(PeerDone=True))
    if pc == "emit":
        offer("source.emit", Local=dict(Program="after-emit"))
        return alternatives

    target = p["Closure"]
    if target.endswith("source") and live:
        status = "retired" if target.startswith("retire") else "cancelled"
        offer("operator."+target, "source/1", Local=dict(Program=status))
    if target.endswith("ancestor") and l["Ancestor"] == "open":
        offer("operator."+target, "ancestor/1", Local=dict(Ancestor=target))

    # The first decision samples the actual source. A saved allowance survives closure.
    if not perm:
        reason = "source_retired" if pc == "retired" else YES if live else "source_closed"
        offer("store.permission", Local=dict(Permission=reason, PermissionEpoch=l["Epoch"]))
    elif not attempt:
        reason = perm
        if perm == YES:
            if p["NoGrant"]:
                reason = "grant_denied"
            elif p["Member"] and (not live or l["Ancestor"] != "open"):
                reason = "scope_closed"
            elif p["Member"] and p["ZeroLimit"]:
                reason = "effect_limit"
        debit = int(reason == YES and p["Member"])
        offer("store.reserve", Local=dict(Reservation=reason, Charge=debit),
              Volatile=dict(Work="reserved" if reason == YES else ""))

    denied = bool(attempt and attempt != YES)
    if denied and not raw and not l["Result"]:
        result = "grant_denied" if attempt == "grant_denied" else "source_denied"
        offer("store.record-denial", Local=dict(Result=result), Oracle=dict(Records=o["Records"]+1))

    # Process death is a finite choice at the four named cuts only. It clears
    # the candidate and worker, not the local DB, any first permission, or world.
    kill_cut = work in {"entered", "good"} or (l["Result"] and not l["Selected"])
    if p["Crash"] and v["Process"] == 1 and kill_cut:
        offer("environment.kill", "executor", Volatile=dict(Process=0, Run=0, Work=""),
              Oracle=dict(Kills=o["Kills"]+1))
    if p["SecondRun"] and v["Run"] == 1 and not live and work == "unanswered":
        offer("host.new-run", "executor", Volatile=dict(Run=2, Work=""))
    if (v["Run"] == 2 and not work and attempt == YES and not l["Result"]
            and p["Idempotent"] and raw):
        move("runner.reexpose", "reserved")

    mode = p["Reply"]
    if work == "reserved":
        move("worker.enter", "entered", ["WorkerEntries"])
    if work == "entered":
        if raw:
            move("worker.http", "sent", ["HTTPCalls", "Launches"])
        else:
            move("worker.execute", "executing", ["Executes", "Launches"])
    if work == "executing":
        move("adapter.http", "sent", ["HTTPCalls"])
    if work == "sent":
        if mode == "early-error":
            move("transport.no-response", "error")
        else:
            move("endpoint.ingress", "arrived", World=dict(Ingress=w["Ingress"]+1))
    if work == "arrived":
        repeat = p["Idempotent"] and w["ValidRequests"] > 0
        move("endpoint.apply", "applied", World=dict(ValidRequests=w["ValidRequests"]+1,
                                                     Applications=w["Applications"]+int(not repeat)))
    if work == "applied":
        if mode == "loss" or (p["SecondRun"] and v["Run"] == 1):
            move("transport.no-response", "unanswered" if raw else "error")
        elif mode == "chosen-error":
            move("adapter.choose-error", "chosen-error")
        elif mode == "unknown":
            move("adapter.return-unknown", "unknown")
        else:
            move("transport.obtain-receipt", "good")
    if work == "unknown":
        move("store.leave-unresolved", "unanswered")
    candidates = {"good": (RECEIPT, "validated-raw-receipt"),
                  "chosen-error": ("selected_error", "validated-raw-receipt"),
                  "error": ("transport_error", "no-authentic-receipt")}
    if work in candidates:
        result, detail = candidates[work]
        move("store.record", "accounted", ["Records"], Local=dict(Result=result, ResultDetail=detail))
    if (l["Result"] or (raw and denied)) and not l["Settled"]:
        offer("store.settle", Local=dict(Settled=True))

    if pc == "after-emit":
        if p["PrivateFault"]:
            offer("source.fault", "source/1", Local=dict(Program="failed"))
        elif l["Settled"]:
            offer("source.select", Local=dict(Program="ending" if raw else "guard", Selected=True))
    if pc == "guard":
        k = "failing"
        if l["Result"] == RECEIPT:
            k = "emit-reply" if p["Parent"] else "ending"
        offer("source.guard", Local=dict(Program=k))
    if pc == "emit-reply":
        offer("source.reply", REPLY, Local=dict(Reply="committed", Program="ending"))
    if pc == "failing":
        offer("source.fail", "source/1", Local=dict(Program="failed"))
    if pc == "ending":
        offer("source.end", "source/1", Local=dict(Program="ended"))
    if l["Reply"] == "committed":
        offer("store.reply-permission", REPLY, Local=dict(Reply="allowed"))
    if l["Reply"] == "allowed":
        offer("store.admit-reply", REPLY, Local=dict(Reply="admitted"))

    if p["Parent"] and not l["Published"]:
        if pc == "failed" and not l["Reply"]:
            offer("store.publish-deliver", INV, Local=dict(Published=FAILURE))
        if pc == "ended" and l["Reply"] == "admitted" and l["Result"] == RECEIPT:
            offer("store.publish-deliver", INV, Local=dict(Published=SUCCESS))
    if l["Published"] and not l["ParentSelected"] and l["Ancestor"] == "open":
        offer("parent.select", INV, Local=dict(ParentSelected=l["Published"]))
    if l["ParentSelected"] and not l["ParentDecision"]:
        # Reads only the envelope on the retained original invocation.
        decision = ("historical_application_receipt" if l["ParentSelected"] == SUCCESS
                    else "local_failure_world_unresolved")
        offer("parent.finish", "parent/1", Local=dict(ParentDecision=decision))
    return alternatives


def coupled(s, actor_profiles):
    offers = {}
    for i, (actor, p) in enumerate(zip(s["Actors"], actor_profiles)):
        other = s["Actors"][1-i]
        free = (other["Volatile"]["Work"] in {"", "unanswered"} or other["Local"]["Settled"])
        for (_, name, obj), successor in single(p, actor).items():
            if name == "store.reserve" and not free:
                continue
            target = deepcopy(s)
            target["Actors"][i] = successor
            if name == "endpoint.apply":
                applications = [a["World"]["Applications"] for a in target["Actors"]]
                target["Ordinal"][i] = sum(applications)
                target["Total"][i] = sum((j+1)*n for j, n in enumerate(applications))
            offers[(i, name, obj)] = target
    return offers


def load(path):
    graphs = {}
    with gzip.open(path, "rt") as f:
        for line in f:
            r = json.loads(line)
            if "Profile" in r or "PairProfile" in r:
                p = r.get("Profile")
                name = p["Name"] if p else r["PairProfile"]
                require(name not in graphs, f"duplicate profile {name}")
                rows = []
                graphs[name] = (p, rows)
            else:
                require(r["ID"] == len(rows), "missing or repeated row")
                rows.append(r)
    return graphs


def identity(s):
    return json.dumps(s, sort_keys=True, separators=(",", ":"))


def audit_graph(name, header, rows, manifest):
    pair = name == PAIR
    expected_start = dict(Actors=[START, START], Ordinal=[0, 0], Total=[0, 0]) if pair else START
    require(rows and rows[0]["State"] == expected_start, f"{name} initial state")
    if not pair:
        require(identity(header) == identity(manifest[name]), f"{name} environment differs")
    # Make no assumptions about row numbering beyond the designated initial state.
    by_state = {identity(r["State"]): r["ID"] for r in rows}
    require(len(by_state) == len(rows), f"{name} duplicated state")
    visited = {0}
    queue = deque([0])
    edges = maxima = 0
    actors = [dict(manifest["adapter-peer"], Peer=False, Reply=m) for m in ("unknown", "ok")]
    indegree = [0]*len(rows)
    while queue:
        i = queue.popleft()
        r = rows[i]
        expected = coupled(r["State"], actors) if pair else single(header, r["State"])
        actual = {}
        for e in r["Outgoing"]:
            require(set(e) == ({"Actor", "ID", "Action"} if pair else {"ID", "Action"}),
                    f"{name}[{i}] edge schema")
            if pair:
                require(type(e["Actor"]) is int and e["Actor"] in (0, 1), f"{name}[{i}] actor")
            require(type(e["ID"]) is int and set(e["Action"]) == {"Name", "On"}, f"{name}[{i}] action")
            key = (e.get("Actor"), e["Action"]["Name"], e["Action"]["On"])
            require(key not in actual, f"{name}[{i}] duplicate {key}")
            actual[key] = e["ID"]
        require(actual.keys() == expected.keys(),
                f"{name}[{i}] missing={expected.keys()-actual.keys()} extra={actual.keys()-expected.keys()}")
        for key, target in expected.items():
            dest = by_state.get(identity(target))
            require(dest is not None and actual[key] == dest,
                    f"{name}[{i}] {key} expected target {dest}, got {actual[key]}")
            indegree[dest] += 1
            if dest not in visited:
                visited.add(dest)
                queue.append(dest)
        edges += len(expected)
        maxima += int(not expected)
    require(len(visited) == len(rows), f"{name} unreachable row(s)")
    queue.extend(i for i, d in enumerate(indegree) if d == 0)
    removed = 0
    while queue:
        removed += 1
        for e in rows[queue.popleft()]["Outgoing"]:
            k = e["ID"]
            indegree[k] -= 1
            if not indegree[k]:
                queue.append(k)
    require(removed == len(rows), f"{name} infinite nonstuttering path")
    return len(rows), edges, maxima


def follow(rows, seq, start=0):
    i = start
    for name in seq.split():
        matches = [e["ID"] for e in rows[i]["Outgoing"] if e["Action"]["Name"] == name]
        require(len(matches) == 1, f"missing/ambiguous trace step {name} at {i}")
        i = matches[0]
    return i


def terminals(rows, start):
    stack, seen = [start], set()
    while stack:
        i = stack.pop()
        if i not in seen:
            seen.add(i)
            if not rows[i]["Outgoing"]:
                yield rows[i]["State"]
            stack.extend(e["ID"] for e in rows[i]["Outgoing"])


def paired_cuts(graphs):
    for name in ("raw-recovery-keyed", "raw-recovery-plain", "adapter-recovery-keyed",
                 "adapter-recovery-plain", "parent-recovery"):
        p, rows = graphs[name]
        first = follow(rows, "source.emit store.permission store.reserve worker.enter")
        call = "worker.http" if p["Route"] == "raw" else "worker.execute adapter.http"
        committed = follow(rows, call+" endpoint.ingress endpoint.apply transport.obtain-receipt", first)
        cuts = [follow(rows, "environment.kill", i) for i in (first, committed)]
        a, b = (rows[i]["State"] for i in cuts)
        require(a["Local"] == b["Local"] and a["Volatile"] == b["Volatile"]
                == dict(Process=0, Run=0, Work=""), f"{name} U0/U1 recovery input")
        require([a["World"]["Applications"], b["World"]["Applications"]] == [0, 1],
                f"{name} U0/U1 world")
        for branch, cut in enumerate(cuts):
            reopened = follow(rows, "host.reopen", cut)
            for final in terminals(rows, reopened):
                retry = p["Route"] == "raw" and p["Idempotent"]
                require(final["Local"]["Selected"] == retry, f"{name} consumer progress")
                require(final["Oracle"]["Launches"] == branch + int(retry), f"{name} repeats")
                require(final["World"]["Applications"] == (1 if retry else branch), f"{name} result")
                require(final["Local"]["Result"] == (RECEIPT if retry else ""), f"{name} fabricated result")


def private_receipt(graphs):
    _, rows = graphs["parent-private-fault-receipt"]
    # Held Execute precedes even the first outgoing HTTP call in the fixture.
    held = follow(rows, "source.emit store.permission store.reserve worker.enter worker.execute "
                  "source.fault store.publish-deliver parent.select parent.finish")
    before = rows[held]["State"]
    require(before["Local"]["ParentSelected"] == FAILURE and not before["Local"]["Result"]
            and not before["World"]["Applications"], "parent observation before private late return")
    done = follow(rows, "adapter.http endpoint.ingress endpoint.apply transport.obtain-receipt "
                  "store.record store.settle", held)
    after = rows[done]["State"]
    require(after["Local"]["Result"] == RECEIPT and not after["Local"]["Selected"]
            and after["Local"]["Program"] == "failed" and after["World"]["Applications"] == 1,
            "private late result lost or consumed")


def negatives(graphs, manifest):
    # Change a copy of exported observations. Eligibility remains a function of the
    # independent source-state rules, even when other alternatives still progress.
    cases = [
        ("omit-private-fault", "parent-private-fault", "source.fault", "remove"),
        ("omit-keyed-recovery", "raw-recovery-keyed", "runner.reexpose", "remove"),
        ("omit-alternative-peer", PAIR, "source.emit", "remove"),
        ("wrong-generation", "raw-member-source", "worker.http", "identity"),
        ("wrong-owner", "adapter-member-source", "worker.execute", "owner"),
        ("wrong-peer", PAIR, "endpoint.apply", "actor"),
        ("retarget-peer-counter", PAIR, "endpoint.apply", "target"),
        ("reservation-as-launch", "raw-peer", "store.reserve", "launch"),
        ("infer-nonexecution", "parent-loss", "store.record", "absence"),
        ("leak-oracle-to-parent", "parent-loss", "parent.finish", "world"),
        ("invent-old-adapter-retry", "adapter-recovery-keyed", "host.reopen", "blind"),
    ]
    for label, profile, action, change in cases:
        header, original = graphs[profile]
        rows = deepcopy(original)
        row, edge = next((r, e) for r in rows for e in r["Outgoing"]
                         if e["Action"]["Name"] == action)
        target = rows[edge["ID"]]["State"]
        if change == "remove":
            row["Outgoing"].remove(edge)
        elif change == "identity":
            edge["Action"]["On"] = "N/source/2/emit/0"
        elif change == "owner":
            edge["Action"]["Name"] = "store.execute"
        elif change == "actor":
            edge["Actor"] = 1-edge["Actor"]
        elif change == "target":
            target["Total"][edge["Actor"]] += 1
        elif change == "launch":
            target["Oracle"]["Launches"] += 1
        elif change == "absence":
            target["Local"]["ResultDetail"] = "proved-nonapplication"
        elif change == "world":
            target["Local"]["ParentDecision"] = "applied"
        elif change == "blind":
            rows[edge["ID"]]["Outgoing"].append(
                dict(Action=dict(Name="runner.reexpose", On=OP), ID=row["ID"]))
        try:
            audit_graph(profile, header, rows, manifest)
        except Rejected as e:
            print(f"REJECT {label}: {e}")
        else:
            raise Rejected(f"accepted mutant {label}")


def frontier(name, state):
    if name == PAIR:
        a, b = state["Actors"]
        require(not a["Local"]["Result"] and not a["Local"]["Selected"]
                and b["Local"]["Program"] == "ended" and b["Local"]["Selected"],
                "two-actor serviced boundary")
        return "unresolved-and-completed"
    l = state["Local"]
    if l["Reservation"] == YES and not l["Result"]:
        if l["Program"] == "failed":
            return "failed-source-unresolved"
        if l["Program"] in CLOSED:
            return "closed-source-unresolved"
        return "unresolved"
    require(l["Program"] in CLOSED, f"{name} withheld responsive service")
    return "completed" if l["Selected"] else "closed-source-unselected"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("export", nargs="?", type=Path, default=BASE/"graphs.jsonl.gz")
    args = parser.parse_args()
    graphs = load(args.export)
    manifest = profiles()
    expected = set(manifest) | {PAIR}
    require(set(graphs) == expected,
            f"profile inventory missing={expected-set(graphs)} extra={set(graphs)-expected}")
    summaries = json.loads(args.export.with_name("summary.json").read_text())
    summaries = {r["Profile"]: r for r in summaries}
    require(set(summaries) == expected, "summary inventory")
    totals = [0, 0, 0]
    for name, (header, rows) in graphs.items():
        counts = audit_graph(name, header, rows, manifest)
        groups = Counter(frontier(name, r["State"]) for r in rows if not r["Outgoing"])
        summary = dict(Profile=name, States=counts[0], Edges=counts[1], Frontiers=groups,
                       MaximumOutgoing=max(len(r["Outgoing"]) for r in rows))
        require(summary == summaries[name], f"{name} summary")
        totals = [a+b for a, b in zip(totals, counts)]
        print(f"PASS {name}: states={counts[0]} edges={counts[1]} maximal={counts[2]}")
    paired_cuts(graphs)
    private_receipt(graphs)
    negatives(graphs, manifest)
    print(f"PASS {len(graphs)} profiles, {totals[0]} states, {totals[1]} edges, "
          f"{totals[2]} maximal; five paired cuts, private-fault receipt, 11 mutants")


if __name__ == "__main__":
    main()
