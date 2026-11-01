"""Independently check every Phase 2C graph successor, maximal classification and falsifier."""

import argparse
from contextlib import contextmanager
from collections import Counter, deque
from copy import deepcopy
import gzip
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile


PHASE = Path(__file__).resolve().parents[1]
GRAPH = PHASE / "evidence/model/graphs.jsonl.gz"


class Rejection(ValueError):
    pass


def require(ok, reason):
    if not ok:
        raise Rejection(reason)


def key(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def profiles():
    # These bounds are independent inputs. A modified graph cannot rename or
    # omit entire scenarios and thereby remove their negative obligations.
    options = {
        "admission-race": dict(Origin="race"),
        "matched-original-loss": dict(Origin="paired"),
        "held-packet-refresh": dict(Origin="delayed", Queries=2),
        "lost-negative-later-found": dict(Origin="delayed", Loss="snapshot"),
        "spent-before-send": dict(Loss="exhaust"),
        "retention-cut": dict(Loss="retained"),
        "pre-import-cut": dict(Loss="returned"),
        "existing-interaction-failure": dict(Origin="failed"),
        "selected-error-policy": dict(Policy="selected-error"),
        "unknown-policy": dict(Policy="unknown"),
        "read-without-import": dict(ReadOnly=True),
        "raw-keyed": dict(Policy="raw-keyed"),
        "raw-non-idempotent": dict(Policy="raw-plain"),
        "ambiguous-raw": dict(Origin="twice", Policy="raw-plain"),
        "unbound-target": dict(BadTarget=True),
        "endpoint-unavailable": dict(Service="unavailable"),
        "foreign-snapshot": dict(Service="foreign"),
        "malformed-response": dict(Service="malformed"),
        "two-resolvers": dict(Queries=2, Concurrent=True),
        "resolver-revocation-and-peer": dict(CloseReader=True, Peer=True),
    }
    defaults = dict(
        Origin="applied",
        Policy="",
        Service="",
        Loss="",
        Queries=1,
        Concurrent=False,
        ReadOnly=False,
        BadTarget=False,
        CloseReader=False,
        Peer=False,
    )
    return {name: dict(defaults, Name=name) | settings for name, settings in options.items()}


def initial():
    blank = dict(Target=0, Query=0, Ordinal=0, Applications=0, Status="")
    read = dict(
        Stage="",
        Spent=0,
        BoundTo=0,
        Acquired=0,
        Retained=0,
        Outcome="",
        Adoption="",
        Closed=False,
        Drained=False,
        Selected=False,
    )
    return dict(
        Original="consider",
        Interaction="",
        Revoked=False,
        Quiescent=False,
        Held=False,
        Dead=False,
        OldOwnerLost=False,
        Losses=0,
        Applications=0,
        SnapshotCount=0,
        Log=[blank.copy() for _ in range(4)],
        Reads=[read.copy(), read.copy()],
        Target=1,
        OriginalDebits=1,
        ReadDebits=0,
        ReconciliationSource=0,
        Settled=False,
        OriginalFailureSelected=False,
        Published=False,
        OutcomeSelected=False,
        Used=False,
        PeerRan=False,
        Rejected=False,
    )


def results(s, prefix="", index=None):
    """Produce immutable successors, preserving every field outside the specified change."""

    def change(label, read=None, snapshot=None, **delta):
        n = deepcopy(s)
        n.update(delta)
        if read is not None:
            n["Reads"][index].update(read)
        if snapshot is not None:
            n["Log"][s["SnapshotCount"]] = snapshot
        return prefix + label, n

    return change


def original(p, s):
    emit = results(s)
    cut, mode = s["Original"], p["Origin"]
    early_close = {
        "race": {"consider", "admitted", "entered", "packet", "returned"},
        "paired": {"admitted", "returned"},
        "delayed": {"packet"},
    }
    if not s["Revoked"] and (cut == "handled" or cut in early_close.get(mode, set())):
        if mode != "failed" or s["OriginalFailureSelected"]:
            yield emit("original.revoke", Revoked=True)
    if cut == "consider":
        if s["Revoked"]:
            yield emit("original.refuse", Original="handled")
        else:
            yield emit("original.admit", Original="admitted")
    elif cut == "admitted" and not (mode == "paired" and s["Revoked"]):
        yield emit("original.enter", Original="entered")
    elif cut == "entered":
        yield emit("world.accept", Held=True, Original="packet")
    elif cut == "packet" and not s["Held"] and mode != "delayed":
        if mode == "twice" and s["Applications"] == 1:
            yield emit("world.accept-second-original", Held=True)
        else:
            yield emit("original.return", Original="returned")
    elif cut == "returned" and mode != "paired":
        if mode == "failed":
            yield emit("original.handle", Original="failure-recorded", Interaction="transport_error")
        else:
            yield emit("original.handle", Original="handled")
    elif cut == "failure-recorded":
        yield emit("original.settle-failure", Original="handled", Settled=True)
    elif cut == "handled" and mode == "failed" and not s["OriginalFailureSelected"]:
        yield emit("issuer.select-failure", OriginalFailureSelected=True)

    permissible_cut = mode == "paired" and cut in {"admitted", "returned"} or mode == "delayed" and cut == "packet"
    if permissible_cut and s["Revoked"] and not s["OldOwnerLost"]:
        yield emit("owner.kill-original", OldOwnerLost=True, Dead=True, Original="handled", Quiescent=False)
    if s["Revoked"] and not s["Quiescent"]:
        if cut == "handled" and (not s["Interaction"] or s["Settled"]):
            yield emit("original.Q", Quiescent=True)


def authenticated_result(p, s, i):
    """Derive authentic evidence *at the acquired snapshot*, not current Applications."""
    q = s["Reads"][i]
    snap = s["Log"][q["Acquired"] - 1] if q["Acquired"] else None
    delta = dict(Stage="retained", Acquired=0)
    if snap is None:
        return delta | {"Outcome": "transport_error"}, {}
    if p["Service"] == "malformed" or snap["Target"] != s["Target"]:
        return delta | {"Outcome": "protocol_invalid"}, {}
    require((snap["Query"], snap["Ordinal"]) == (i + 1, q["Spent"]), "acquired a foreign read/ordinal")
    delta.update(Outcome=snap["Status"], Retained=q["Acquired"])
    if snap["Status"] != "found":
        return delta, {}
    if p["ReadOnly"]:
        decision = "denied"
    elif p["Policy"].startswith("raw-"):
        decision = "evidence_only"
    elif s["Interaction"] == "transport_error":
        decision = "existing"
    elif p["Policy"]:
        decision = "unsupported_policy"
    elif s["ReconciliationSource"]:
        decision = "duplicate"
    else:
        return delta | {"Adoption": "adopted"}, dict(ReconciliationSource=i + 1, Interaction="reconciled")
    return delta | {"Adoption": decision}, {}


def query(p, s, i):
    q = s["Reads"][i]
    stage = q["Stage"]
    emit = results(s, f"q{i + 1}.", i)
    if p["CloseReader"] and stage and not q["Closed"]:
        yield emit("revoke", read={"Closed": True})
    if q["Closed"] and not q["Drained"] and stage in {"ready", "handled"}:
        yield emit("Q", read={"Drained": True})
    if not stage:
        pred = s["Reads"][i - 1]
        if i == 0 or (bool(pred["Stage"]) if p["Concurrent"] else pred["Selected"]):
            yield emit("request", read={"Stage": "ready"}, ReadDebits=s["ReadDebits"] + 1)
    elif stage == "ready" and not q["Closed"]:
        if q["Spent"] == 2 or p["BadTarget"]:
            yield emit(
                "refuse",
                read=dict(
                    Stage="retained", Outcome="refused", Adoption="exhausted" if q["Spent"] == 2 else "unbound"
                ),
            )
        else:
            yield emit("spend-and-admit", read=dict(Stage="admitted", Spent=q["Spent"] + 1, BoundTo=s["Target"]))
    elif stage == "admitted":
        yield emit("enter", read={"Stage": "entered"})
    elif stage == "entered":
        if p["Service"] == "unavailable":
            yield emit("transport-fails", read={"Stage": "returned"})
        else:
            a = s["Applications"]
            snap = dict(
                Target=2 if p["Service"] == "foreign" else s["Target"],
                Query=i + 1,
                Ordinal=q["Spent"],
                Applications=a,
                Status=("not_found", "found", "conflict")[a],
            )
            serial = s["SnapshotCount"] + 1
            yield emit(
                "snapshot", snapshot=snap, SnapshotCount=serial, read=dict(Stage="snapshot", Acquired=serial)
            )
    elif stage == "snapshot":
        yield emit("return", read={"Stage": "returned"})
    elif stage == "returned":
        r_delta, original_delta = authenticated_result(p, s, i)
        yield emit("retain", read=r_delta, **original_delta)
    elif stage == "retained" and (q["Adoption"] != "adopted" or s["Settled"]):
        yield emit("handle", read={"Stage": "handled"})
    elif stage == "handled" and not (q["Closed"] or q["Selected"]):
        yield emit("select-evidence", read={"Selected": True})

    cuts = {
        "exhaust": ("admitted", 2),
        "snapshot": ("snapshot", 1),
        "returned": ("returned", 1),
        "retained": ("retained", 1),
    }
    if p["Loss"] in cuts:
        where, limit = cuts[p["Loss"]]
        if stage == where and s["Losses"] < limit:
            yield emit(
                "kill-owner",
                Dead=True,
                Quiescent=False,
                Losses=s["Losses"] + 1,
                read=dict(Acquired=0, Stage=stage if stage == "retained" else "ready"),
            )


def outgoing(p, s):
    emit = results(s)
    if s["Held"]:
        yield emit("world.apply", Applications=s["Applications"] + 1, Held=False)
    if s["Dead"]:
        yield emit("owner.open", Dead=False)
        return
    yield from original(p, s)
    if s["Interaction"] == "reconciled" and not s["Settled"]:
        yield emit("original.settle", Settled=True)
    if not s["Quiescent"]:
        return
    if p["Peer"] and not s["PeerRan"]:
        yield emit("peer.commit", PeerRan=True)
    for i in range(p["Queries"]):
        yield from query(p, s, i)
    if s["Interaction"] and not s["Published"]:
        yield emit("aggregate.publish", Published=True)
    if s["Published"] and not s["OutcomeSelected"]:
        yield emit("parent.select-original", OutcomeSelected=True)
    if s["OutcomeSelected"] and not (s["Used"] or s["Rejected"]):
        eligible = s["Reads"][: p["Queries"]] if p["Concurrent"] else [s["Reads"][p["Queries"] - 1]]
        reports = [q["Adoption"] for q in eligible if q["Selected"] and q["Outcome"] == "found"]
        if "adopted" in reports or "existing" in reports and s["Interaction"] == "transport_error":
            yield emit("parent.use-correlated-evidence", Used=True)
        if not p["Concurrent"] and p["Queries"] == 2 and "duplicate" in reports:
            yield emit("parent.reject-final-duplicate", Rejected=True)


def classification(s):
    if s["Dead"]:
        return "owner-lost"
    if s["Used"]:
        return (
            "applied-after-failed-interaction" if s["Interaction"] == "transport_error" else "ordinary-continuation"
        )
    if s["Rejected"]:
        return "final-duplicate-guard-failed"
    for q in reversed(s["Reads"]):
        stage = q["Stage"]
        if stage:
            if stage == "ready" and q["Closed"]:
                return "revoked-before-admission"
            if stage == "retained":
                return "accounting-owed"
            if stage == "handled":
                return q["Adoption"] or q["Outcome"]
            return "read-service"
    if s["Quiescent"]:
        return "drained-with-live-packet" if s["Held"] else "original-drained"
    return "original-service"


def check_profile(g, p):
    require(key(g["Profile"]) == key(p), f"wrong declared bounds for {p['Name']}")
    rows = g["Rows"]
    require(rows and key(rows[0]["State"]) == key(initial()), "wrong initial state")
    encoded = [key(r["State"]) for r in rows]
    require(len(set(encoded)) == len(rows), "duplicate state")
    sinks = Counter()
    edges = 0
    for i, r in enumerate(rows):
        context = f"{p['Name']}/{i}"
        s = r["State"]
        require(r["Class"] == classification(s), context + " classification")
        want = {label: key(n) for label, n in outgoing(p, s)}
        seen = set()
        for edge in r["Edges"]:
            label, to = edge["Action"], edge["To"]
            require(type(to) is int and 0 <= to < len(rows), context + " bad edge index")
            require(label not in seen, context + " repeated edge " + label)
            require(label in want, context + " unexpected action " + label)
            require(encoded[to] == want[label], context + " incorrect successor " + label)
            seen.add(label)
        require(seen == want.keys(), context + " missing " + ",".join(sorted(want.keys() - seen)))
        edges += len(seen)
        if not seen:
            sinks[r["Class"]] += 1
    reached, pending = {0}, deque([0])
    while pending:
        for e in rows[pending.popleft()]["Edges"]:
            if e["To"] not in reached:
                reached.add(e["To"])
                pending.append(e["To"])
    require(len(reached) == len(rows), "export contains unreachable states")
    return len(reached), edges, sinks


def mutate(graph, label, condition, effect):
    """Modify one reachable exported edge, preserving the remainder of the source graph."""
    g = deepcopy(graph)
    for i, row in enumerate(g["Rows"]):
        for edge in row["Edges"]:
            if edge["Action"] == label and condition(row["State"]):
                if effect is None:
                    row["Edges"].remove(edge)
                else:
                    alternate = deepcopy(g["Rows"][edge["To"]])
                    effect(alternate["State"])
                    alternate["Class"] = classification(alternate["State"])
                    edge["To"] = len(g["Rows"])
                    g["Rows"].append(alternate)
                return g, i
    raise Rejection("mutant never reached intended transition " + label)


def mutants(graphs, declared):
    checks = [
        (
            "Q forbids future application",
            "lost-negative-later-found",
            "world.apply",
            lambda s: s["Quiescent"],
            None,
        ),
        (
            "absence seals the pending packet",
            "held-packet-refresh",
            "q1.retain",
            lambda s: s["Held"] and s["Log"][s["Reads"][0]["Acquired"] - 1]["Status"] == "not_found",
            lambda n: n.update(Held=False),
        ),
        (
            "read permission imports",
            "read-without-import",
            "q1.retain",
            lambda s: True,
            lambda n: n.update(Interaction="reconciled", ReconciliationSource=1),
        ),
        (
            "receipt validates any adapter policy",
            "selected-error-policy",
            "q1.retain",
            lambda s: True,
            lambda n: n.update(Interaction="reconciled", ReconciliationSource=1),
        ),
        (
            "owner loss resets physical allowance",
            "spent-before-send",
            "q1.kill-owner",
            lambda s: s["Reads"][0]["Spent"] == 2,
            lambda n: n["Reads"][0].update(Spent=0),
        ),
        (
            "query completion alone publishes original",
            "read-without-import",
            "q1.handle",
            lambda s: True,
            lambda n: n.update(Published=True),
        ),
        (
            "omit independent peer service",
            "resolver-revocation-and-peer",
            "peer.commit",
            lambda s: s["Reads"][0]["Stage"] == "entered",
            None,
        ),
    ]
    for desc, name, label, condition, effect in checks:
        g, i = mutate(graphs[name], label, condition, effect)
        try:
            check_profile(g, declared[name])
        except Rejection as e:
            expected = "missing" if effect is None else "incorrect successor"
            require(
                str(e).startswith(f"{name}/{i} {expected}") and label in str(e),
                f"{desc}: unexpected checker failure: {e}",
            )
            print(f"REJECT {desc}: {e}")
        else:
            raise Rejection("accepted mutant " + desc)
    for tag in ("extra-edge", "bad-class"):
        name = "read-without-import"
        g = deepcopy(graphs[name])
        row = g["Rows"][0]
        if tag == "extra-edge":
            row["Edges"].append({"Action": "invented.dispatch", "To": 0})
        else:
            row["Class"] = "ordinary-continuation"
        try:
            check_profile(g, declared[name])
        except Rejection as e:
            require(("unexpected action" if tag == "extra-edge" else "classification") in str(e), str(e))
            print(f"REJECT {tag}: {e}")
        else:
            raise Rejection("accepted mutant " + tag)


@contextmanager
def database(case, filename):
    source = PHASE / "evidence" / case / filename
    archive = source.read_bytes()
    print("sqlite archive", str(source.relative_to(PHASE)), hashlib.sha256(archive).hexdigest())
    with tempfile.NamedTemporaryFile() as f:
        f.write(gzip.decompress(archive))
        f.flush()
        db = sqlite3.connect("file:" + f.name + "?immutable=1", uri=True)
        try:
            yield db
        finally:
            db.close()


def table_count(db, table):
    return db.execute("select count(*) from " + table).fetchone()[0]


def stored_results(db):
    return {k: json.loads(v) for k, v in db.execute("select intent_id,descriptor from adapter_results")}


def history(db, run):
    sql = (
        "select o.descriptor from history as h join observations as o "
        "on h.selected_observation=o.id where h.subject=? order by h.sequence"
    )
    return [json.loads(raw) for (raw,) in db.execute(sql, (run,))]


def application_snapshots(db):
    out = []
    for serial, request, response in db.execute(
        "select number,request,response from receipt_reads order by number"
    ):
        target, snapshot = json.loads(request), json.loads(response)
        require(snapshot["Number"] == serial and snapshot["Query"] == target, "endpoint query correlation")
        require(hashlib.sha256(target["Wire"].encode()).hexdigest() == target["WireSHA256"], "wire preimage")
        if snapshot["Status"] == "found":
            require(
                snapshot["AppliedWire"] == target["Wire"]
                and snapshot["Receipt"]["Key"] == target["TargetKey"]
                and snapshot["Count"] == 1,
                "found receipt correlation",
            )
        out.append((target["QueryID"], target["Ordinal"], snapshot["Status"]))
    return out


def check_histories():
    for regime in ("normal", "race"):
        public = "public-" + regime + "/"
        process = "process-" + regime + "/"
        with database("authority-" + regime + "/before-admission", "final-local.sqlite.gz") as db:
            require(
                table_count(db, "receipt_read_ordinals") == 0 and not stored_results(db),
                "R before A produced a result",
            )
            state = json.loads(db.execute("select projection from runs where subject='resolver'").fetchone()[0])
            require(state["Status"] == "cancelled", "wrong revoked resolver")
        with database(public + "read-only", "after.sqlite.gz") as db:
            (result,) = stored_results(db).values()
            require(
                result["Read"]["Adoption"] == "denied" and result["Values"]["status"]["Text"] == "found",
                "read-only result",
            )
            require(not table_count(db, "composition_publications"), "read without adoption published original")
        with (
            database(public + "immutable", "before.sqlite.gz") as before,
            database(public + "immutable", "after.sqlite.gz") as after,
        ):
            before_rows = dict(before.execute("select intent_id,descriptor from adapter_results"))
            (old_id,) = before_rows
            require(json.loads(before_rows[old_id])["Disposition"] == "transport_error", "wrong historical result")
            require(
                dict(after.execute("select intent_id,descriptor from adapter_results"))[old_id]
                == before_rows[old_id],
                "rewritten historical interaction",
            )
            (later,) = [r for k, r in stored_results(after).items() if k != old_id]
            require(
                later["Read"]["Adoption"] == "existing" and later["Values"]["status"]["Text"] == "found",
                "missing independent later application evidence",
            )
        for case, expected_ids, expected_ordinals in [("held-lost", 1, [1, 2]), ("held-retained", 2, [1, 1])]:
            with database(process + case, "final-world.sqlite.gz") as db:
                snaps = application_snapshots(db)
                require(
                    len({x[0] for x in snaps}) == expected_ids
                    and [x[1] for x in snaps] == expected_ordinals
                    and [x[2] for x in snaps] == ["not_found", "found"],
                    "lost/retained refresh distinction",
                )
            with database(process + case, "final-local.sqlite.gz") as db:
                local = [r for r in stored_results(db).values() if r.get("Read")]
                require(
                    len(local) == expected_ids and sum(r["Read"]["Adoption"] == "adopted" for r in local) == 1,
                    "wrong retained read/import count",
                )
        with database(process + "exhausted", "final-local.sqlite.gz") as db:
            (result,) = stored_results(db).values()
            require(
                table_count(db, "receipt_read_ordinals") == 2
                and result["Disposition"] == "read_exhausted"
                and not result.get("Reconciled"),
                "physical allowance reset or invented original",
            )
        with database(process + "atomic-import", "atomic-committed-local.sqlite.gz") as db:
            states = stored_results(db)
            require(
                len(states) == 2
                and sum(bool(r.get("Reconciled")) for r in states.values()) == 1
                and table_count(db, "settlements") == 0
                and table_count(db, "composition_publications") == 0,
                "incorrect post-import prefix",
            )
        case = "refresh-" + regime + "/final-duplicate"
        with database(case, "final-world.sqlite.gz") as db:
            snaps = application_snapshots(db)
            require(
                table_count(db, "applications") == 1
                and table_count(db, "contacts") == 1
                and len(snaps) == 2
                and all(x[1:] == (1, "found") for x in snaps),
                "duplicate application or missing refresh",
            )
        with database(case, "final-local.sqlite.gz") as db:
            reads = [o for o in history(db, "resolver") if o["Provenance"] == "runtime-adapter"]
            require(
                [o["Payload"]["adoption"]["Text"] for o in reads] == ["adopted", "duplicate"],
                "resolver did not select both real outcomes",
            )
            reports = [
                o
                for o in history(db, "parent")
                if o["Provenance"] == "ordinary-message" and o["Obligation"] == "receipt"
            ]
            outcome = [o for o in history(db, "parent") if o["Provenance"] == "runtime-invocation"]
            require(
                len(outcome) == len(reports) == 1
                and reports[0]["Payload"]["adoption"]["Text"] == "duplicate"
                and not reports[0]["Payload"]["adopted"]["Flag"],
                "wrong parent selections",
            )
            require(
                json.loads(db.execute("select projection from runs where subject='parent'").fetchone()[0])["Status"]
                == "failed",
                "parent used the earlier adoption",
            )
        print("PASS actual SQLite comparisons", regime)


def challenge_prior():
    # Re-examine specifically the semantic disagreement in the older export;
    # its missing Rejected field alone would fail the current schema check.
    rel = str(GRAPH.relative_to(PHASE.parents[2]))
    source = subprocess.check_output(["git", "show", "9f108d2b1eaf37cb585982f6c26d9830abff6fb7:" + rel], cwd=PHASE)
    graphs = {
        g["Profile"]["Name"]: g["Rows"] for g in (json.loads(line) for line in gzip.decompress(source).splitlines())
    }
    bad = {
        "early report": (
            "held-packet-refresh",
            lambda s, label, n: label == "parent.use-correlated-evidence" and not s["Reads"][1]["Selected"],
        ),
        "fabricated gate refusal": (
            "resolver-revocation-and-peer",
            lambda s, label, n: (
                s["Reads"][0]["Closed"] and label == "q1.refuse" and n["Reads"][0]["Outcome"] == "refused"
            ),
        ),
        "premature successor Q": (
            "retention-cut",
            lambda s, label, n: label == "original.Q" and s["Interaction"] == "reconciled" and not n["Settled"],
        ),
    }
    print("prior graph SHA-256", hashlib.sha256(source).hexdigest())
    for reason, (name, predicate) in bad.items():
        rows = graphs[name]
        hit = next(
            (
                (i, e)
                for i, r in enumerate(rows)
                for e in r["Edges"]
                if predicate(r["State"], e["Action"], rows[e["To"]]["State"])
            ),
            None,
        )
        require(hit is not None, "incorrect prior diagnosis " + reason)
        print(f"REJECT prior graph {reason}: {name}/{hit[0]} {hit[1]}")


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--graph", type=Path, default=GRAPH)
    parser.add_argument("--mutants", action="store_true")
    parser.add_argument("--histories", action="store_true")
    parser.add_argument("--challenge-prior", action="store_true")
    args = parser.parse_args()
    data = args.graph.read_bytes()
    declared = profiles()
    docs = [json.loads(line) for line in gzip.decompress(data).splitlines()]
    graphs = {g["Profile"]["Name"]: g for g in docs}
    require(len(docs) == len(declared) and graphs.keys() == declared.keys(), "wrong profile inventory")
    print("graph SHA-256", hashlib.sha256(data).hexdigest())
    states = edges = maximal = 0
    for name, p in declared.items():
        count, transitions, sinks = check_profile(graphs[name], p)
        print(f"{name}: {count} states, {transitions} edges; sinks {dict(sorted(sinks.items()))}")
        states += count
        edges += transitions
        maximal += sinks.total()
    print(f"Total {len(declared)} profiles, {states} states, {edges} labelled edges, {maximal} sinks")
    if args.mutants:
        mutants(graphs, declared)
    if args.challenge_prior:
        challenge_prior()
    if args.histories:
        check_histories()


if __name__ == "__main__":
    main()
