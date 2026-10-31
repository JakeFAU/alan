"""Independently decode the Phase 1T archived Store inspections (stdlib only)."""

import argparse
import base64
from collections import Counter, defaultdict
import gzip
import hashlib
import itertools
import json
from pathlib import Path


HERE = Path(__file__).resolve().parent
MODES = ("answer", "fail", "absent", "pending")
OLD = {
    "A": "74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7",
    "B": "ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67",
    "h": "698256075f67b360cacb0b20e16c5aeafc561c76255aeb3fd1ca7481ef9e9a6b",
    "S": "d19cc8d312c2d1f16e10fe71d9b891400bf49b5d31f0d63226c77521efe63cf8",
    "JA": "fd9d3e82946594e8114a84a156f56594d4f9dac6448656ff86e090aadb1b32eb",
    "JB": "e87d0f268da24f572e9f8fd14a194a8dd96551838708c186dadd9ac5a3b68773",
}


def read(path):
    with gzip.open(path, "rt") as f:
        return json.load(f)


def compact(x):
    return json.dumps(x, separators=(",", ":"), ensure_ascii=False)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def identity(tag, data):
    return digest(compact([tag, data]).encode())


def run_id(subject):
    return {"Subject": subject, "Generation": 1}


def check(condition, label):
    if not condition:
        raise ValueError(label)


def index(rows, key):
    result = {row[key]: row for row in rows}
    check(len(result) == len(rows), f"duplicate {key}")
    return result


def value(v):
    k = v["Kind"]
    if k == "list":
        return [record(row) for row in v["List"]]
    x = v[{"string": "Text", "int": "Number", "bool": "Flag"}[k]]
    check(type(x) is {"string": str, "int": int, "bool": bool}[k], "malformed typed value")
    return x


def record(d):
    return {k: value(v) for k, v in d.items()}


def tagged(x):
    k = {str: "string", bool: "bool", int: "int", list: "list"}[type(x)]
    d = dict(Kind=k, Number=0, Text="", Flag=False)
    d[{"string": "Text", "int": "Number", "bool": "Flag", "list": "List"}[k]] = x
    return d


def dec(row):
    return json.loads(row["descriptor"])


def get(operand, data):
    check(operand.get("Offset") is None, "unexpected offset")
    return data[operand["Ref"]] if operand["Ref"] else operand["Literal"]


def fields(d, **expected):
    for k, v in expected.items():
        check(d[k] == v and type(d[k]) is type(v), f"{k}: {d[k]!r} != {v!r}")


class Store:
    def __init__(self, raw):
        self.raw = raw
        self.t = t = raw["Tables"]
        (self.root,) = raw["Roots"]
        self.rid = self.root["RootID"]
        check(self.rid == identity("alan/resource-profile/v1", [raw["Namespace"], run_id("root")]), "root ID")
        self.runs = index(t["runs"], "subject")
        self.progs = {}
        for r in t["programs"]:
            p = json.loads(r["bytes"])
            check(digest(r["bytes"].encode()) == r["id"] and p["Semantics"] == r["semantics"], "program bytes")
            self.progs[r["id"]] = p
            author = p.get("Author")
            if author:
                schemas = [author["Inputs"], author["Outputs"]]
                for d in author.get("Invocations", {}).values():
                    schemas += [d["Input"], d["Output"]]
                for port in author["Ports"].values():
                    schemas += [port["Receive"], port["Send"]]
                for schema in schemas:
                    if schema["ID"]:
                        check(digest(compact(schema["Record"]).encode()) == schema["ID"], "schema digest")
        self.tmpl = {r["id"]: dec(r) for r in t["composition_templates"]}
        for tid, tmpl in self.tmpl.items():
            check(identity(tmpl["Version"], dict(tmpl, ID="")) == tid == tmpl["ID"], "template ID")
        profile = dec(t["composition_profile"][0])
        check(
            index(profile["Templates"], "ID") == self.tmpl
            and set(profile["Roots"][0]["Templates"]) == self.tmpl.keys(),
            "installed root catalogue",
        )
        for tmpl in self.tmpl.values():
            metadata = self.progs[tmpl["Members"]["entry"]["ProgramID"]]["Author"]
            children = {d["TemplateID"] for d in metadata.get("Invocations", {}).values()}
            check(set(tmpl["Templates"] or []) == children, "local catalogue/declaration mismatch")
        self.ad = {r["operation_id"]: dec(r) for r in t["composition_admission_decisions"]}
        self.pub = {r["operation_id"]: dec(r) for r in t["composition_publications"]}
        self.reply = {r["operation_id"]: dec(r) for r in t["composition_replies"]}
        self.auth = index(t["attempts"], "intent_id")
        self.it = {r["id"]: dec(r) for r in t["intents"]}
        self.obs = index(t["observations"], "id")
        self.deliv = index(t["composition_outcome_deliveries"], "operation_id")
        self.freeze = {r["scope_id"]: dec(r) for r in t["composition_freezes"]}
        root_out = {r["scope_id"]: dec(r) for r in t["composition_root_publications"]}
        kinds = {
            "admission": self.ad,
            "reply": self.reply,
            "outcome": {**self.pub, **root_out},
            "freeze": self.freeze,
            "authorization": {k: json.loads(r["result"]) for k, r in self.auth.items()},
            "lifecycle": {r["id"]: [dec(r), json.loads(r["result"])] for r in t["lifecycle_results"]},
        }
        self.orders, self.execs = {}, {}
        for n, r in enumerate(sorted(t["composition_facts"], key=lambda r: r["decision_order"]), 1):
            f = dec(r)
            fields(f, Order=n, Kind=r["kind"], ID=r["id"], RootIDs=[self.rid])
            check(n == r["decision_order"], "fact order")
            if f["Kind"] == "execution":
                key = (f["Run"]["Subject"], f["Sequence"])
                check(key not in self.execs, "repeated execution")
                self.execs[key] = f
            else:
                key = (f["Kind"], f["OperationID"])
                check(key not in self.orders, "repeated first decision")
                self.orders[key] = n
                kind, oid = key
                fields(
                    f,
                    ID=identity("alan/fact/" + kind + "/v1", oid),
                    RecordDigest=identity("alan/" + kind + "-row/v1", kinds[kind][oid]),
                )
        self.children, self.parents, self.charged = defaultdict(dict), {}, {}
        self.join_calls()
        self.check_history()
        self.check_publications()

    def program(self, subject):
        return self.progs[self.runs[subject]["program_id"]]

    def projection(self, subject):
        return json.loads(self.runs[subject]["projection"])

    def join_calls(self):
        t = self.t
        invs = index(t["composition_invocations"], "operation_id")
        charges = index(t["composition_charges"], "operation_id")
        members = index(t["composition_members"], "invocation_id")
        bindings = index(t["composition_bindings"], "invocation_id")
        requests = index(t["composition_requests"], "invocation_id")
        accepted = set()
        for op, d in self.ad.items():
            i, it = d["Invocation"], self.it[op]
            parent, role = it["Run"]["Subject"], it["Obligation"]
            decl = self.program(parent)["Author"]["Invocations"][role]
            fields(it, ID=op, Kind="invocation", Target=decl["TemplateID"])
            fields(
                i,
                ID=identity("alan/invocation/v1", [self.raw["Namespace"], op]),
                OperationID=op,
                TemplateID=decl["TemplateID"],
                Inputs=it["Payload"],
                Caller=it["Run"],
                RootID=self.rid,
                SourceRevision=1,
                SourceSequence=it["Sequence"],
            )
            check(role not in self.children[parent], "duplicate original role")
            self.children[parent][role] = op
            auth = json.loads(self.auth[op]["result"])
            fields(auth, IntentID=op, Allowed=True, Reason="authorized")
            check(dec(self.auth[op])["Intent"] == it, "first permission on unrelated intent")
            check(
                self.execs[parent, it["Sequence"]]["Order"]
                < self.orders["authorization", op]
                < self.orders["admission", op],
                "first admission order",
            )
            if not d["Accepted"]:
                check(not d["Members"] and not d["BindingID"], "denied created member")
                continue
            fields(d, Accepted=True, Reason="accepted")
            accepted.add(op)
            iid = i["ID"]
            who = identity("alan/invocation-member/v1", [iid, "entry"])
            fields(d, Members=[run_id(who)])
            b, m = dec(bindings[iid]), dec(members[iid])
            fields(
                b,
                ID=identity(b["Version"], dict(b, ID="")),
                Caller=it["Run"],
                TemplateID=it["Target"],
                OperationID=op,
                InvocationID=iid,
                RootID=self.rid,
                Namespace=self.raw["Namespace"],
                Members={"entry": run_id(who)},
                Programs={"entry": self.runs[who]["program_id"]},
                Depth=i["Depth"],
            )
            check(b["ID"] == d["BindingID"] and b["Protocol"] == "alan/composition/v1", "admission binding")
            template = self.tmpl[it["Target"]]
            entry = template["Members"]["entry"]
            fields(
                m,
                Run=run_id(who),
                ProgramID=entry["ProgramID"],
                Initial=entry["Initial"],
                RootID=self.rid,
                InvocationID=iid,
                Member="entry",
            )
            fields(self.runs[who], program_id=m["ProgramID"], generation=1)
            check(
                json.loads(self.runs[who]["initial"])
                == m["Initial"]
                == {"PC": 0, "Data": {}, "Status": "open"},
                "initializer",
            )
            check(dec(invs[op]) == i and json.loads(invs[op]["admission"]) == d, "immutable admission")
            port = self.program(who)["Author"]["Ports"][template["EntryPort"]]
            fields(decl, Input=port["Receive"], Output=port["Send"])
            req = dec(requests[iid])
            fields(
                req,
                ID=requests[iid]["observation_id"],
                Operation=op,
                Run=run_id(who),
                Payload=i["Inputs"],
                Source=parent,
                SourceGeneration=1,
                Provenance="runtime-invocation-request",
                Kind="message",
                Obligation=port["Obligation"],
            )
            check(
                req["ID"] == identity("alan/invocation-request/v1", iid) and req == dec(self.obs[req["ID"]]),
                "original request observation",
            )
            charge = dec(charges[op])
            fields(
                charge,
                Invocations=1,
                CreatedRuns=1,
                Effects=0,
                RootID=self.rid,
                OperationID=op,
                Kind="invocation",
                DecisionOrder=self.orders["admission", op],
            )
            self.parents[who] = op
            self.charged[op] = charge
        check(accepted == invs.keys() == charges.keys(), "accepted/charged set")
        accepted_ids = {self.ad[op]["Invocation"]["ID"] for op in accepted}
        check(
            accepted_ids == members.keys() == bindings.keys() == requests.keys(),
            "member/binding/request surplus",
        )
        fields(self.root["Charges"], Invocations=len(accepted), CreatedRuns=len(accepted), Effects=0)
        for op, d in self.ad.items():
            i = d["Invocation"]
            parent = self.parents.get(i["Caller"]["Subject"])
            pi = self.ad[parent]["Invocation"] if parent else None
            fields(i, ParentInvocationID=pi["ID"] if pi else "", Depth=pi["Depth"] + 1 if pi else 1)
            if pi:
                check(i["TemplateID"] in self.tmpl[pi["TemplateID"]]["Templates"], "parent catalogue")

    def check_history(self):
        picked, emitted = set(), set()
        by = defaultdict(list)
        for row in self.t["history"]:
            by[row["subject"]].append(row)
        for who, r in self.runs.items():
            state = json.loads(r["initial"])
            code = self.program(who)["Code"]
            for n, h in enumerate(sorted(by[who], key=lambda x: x["sequence"]), 1):
                check((h["sequence"], h["generation"], state["Status"]) == (n, 1, "open"), "history continuity")
                pc, data = state["PC"], state["Data"]
                x = code[pc]
                op, dst = x["Op"], x["Dst"]
                nextpc, status = pc + 1, "open"
                cursors = {k: dict(v) for k, v in state.get("Cursors", {}).items()}
                new = data.copy()
                intents = json.loads(h["intents"])
                check(len(intents) == (op == "emit"), "intent arity")
                obsid = h["selected_observation"]
                check(bool(obsid) == (op == "receive"), "observation at wrong instruction")

                def operand(name):
                    return get(x[name], data)

                if op == "receive":
                    o = dec(self.obs[obsid])
                    fields(o, ID=obsid, Run=run_id(who))
                    check(self.obs[obsid]["consumed_sequence"] == n and obsid not in picked, "selected row")
                    picked.add(obsid)
                    m = x["Match"]
                    parent = self.parents.get(who)
                    bound = parent and m["Kind"] == "message" and value(get(m["Source"], data)) == "caller"
                    fields(
                        o,
                        Kind=m["Kind"],
                        Provenance="runtime-invocation-request" if bound else m["Provenance"],
                        Source=self.it[parent]["Run"]["Subject"] if bound else value(get(m["Source"], data)),
                        Operation=parent if bound else value(get(m["Operation"], data)),
                        SourceGeneration=value(get(m["Generation"], data)),
                        Obligation=value(get(m["Obligation"], data)),
                    )
                    for field, reg in (x["Bind"] or {}).items():
                        if field.startswith("response.") and value(o["Payload"]["response_present"]) is False:
                            check(field not in o["Payload"], "absent response payload")
                            new.pop(reg, None)
                        else:
                            new[reg] = o["Payload"][field]
                elif op == "branch":
                    nextpc = x["Target"] if operand("A") == operand("B") else x["Otherwise"]
                elif op == "jump":
                    nextpc = x["Target"]
                elif op == "set":
                    new[dst] = operand("A")
                elif op == "append":
                    row = {k: get(v, data) for k, v in x["Updates"].items()}
                    check(value(data[dst]) == [], "singleton list overflow")
                    new[dst] = tagged([row])
                elif op == "foreach":
                    check(x["Collection"]["MaxItems"] == 1 and len(value(operand("A"))) <= 1, "foreach bound")
                    cursors[str(pc)] = {"Snapshot": operand("A"), "Index": 0}
                elif op == "next":
                    start = str(x["Target"])
                    c = cursors[start]
                    rows, pos = c["Snapshot"]["List"], c["Index"]
                    if pos == len(rows):
                        nextpc = code[x["Target"]]["Otherwise"]
                        del cursors[start]
                    else:
                        new.update({v: rows[pos][k] for k, v in x["Bind"].items()})
                        c["Index"] += 1
                elif op == "loop":
                    check(str(x["Target"]) in cursors, "loop cursor")
                    nextpc = x["Target"] + 1
                elif op == "emit":
                    (it,) = intents
                    oid = compact([who, 1, n, 0])
                    eff = x["Effect"]
                    target = value(get(eff["Target"], data))
                    if eff["Kind"] == "message" and target == "caller":
                        target = self.it[self.parents[who]]["Run"]["Subject"]
                    fields(
                        it,
                        ID=oid,
                        Run=run_id(who),
                        Sequence=n,
                        Ordinal=0,
                        Target=target,
                        Kind=eff["Kind"],
                        Generation=value(get(eff["Generation"], data)),
                        Obligation=value(get(eff["Obligation"], data)),
                        Payload={k: get(v, data) for k, v in eff["Payload"].items()},
                    )
                    check(it == self.it[oid] and oid not in emitted, "original emission")
                    emitted.add(oid)
                    if dst:
                        new[dst] = tagged(oid)
                elif op == "end":
                    nextpc, status = pc, x["Outcome"]
                else:
                    raise ValueError(f"unsupported instruction {op}")
                state = {"PC": nextpc, "Data": new, "Status": status}
                if cursors:
                    state["Cursors"] = cursors
                check(state == json.loads(h["successor"]), f"incorrect successor {who} sequence {n} pc {pc}")
                if who != "operator":
                    f = self.execs[who, n]
                    ev = dict(
                        Run=run_id(who),
                        Sequence=n,
                        Epoch=h["epoch"],
                        Revision=h["source_revision"],
                        Selected=obsid or "",
                        Successor=json.loads(h["successor"]),
                        Intents=intents,
                    )
                    check(f["RecordDigest"] == identity("alan/history-row/v1", ev), "execution fact digest")
            final = self.projection(who)
            if final["Status"] == "cancelled":
                targets = [json.loads(y["result"]) for y in self.t["lifecycle_results"]]
                check(
                    any(
                        d["After"]["Run"] == run_id(who)
                        and d["After"]["Projection"] == final
                        and d["Before"]["Projection"] == state
                        and d["Accepted"]
                        for d in targets
                    ),
                    "cancellation transition",
                )
            else:
                check(state == final, "last projection")
        check(
            emitted == self.it.keys()
            and picked == {k for k, v in self.obs.items() if v["consumed_sequence"] is not None},
            "orphaned intent/selection",
        )

    def check_publications(self):
        results = index(self.t["composition_results"], "invocation_id")
        check(self.pub.keys() == self.deliv.keys(), "non-atomic delivery")
        by_scope = defaultdict(list)
        for oid, reply in self.reply.items():
            it = self.it[oid]
            who = it["Run"]["Subject"]
            orig = self.parents[who]
            fields(
                reply,
                Intent=it,
                Accepted=True,
                Reason="recorded_reply",
                InvocationID=self.ad[orig]["Invocation"]["ID"],
                RootID=self.rid,
                Authorization=json.loads(self.auth[oid]["result"]),
            )
            fields(reply["Authorization"], IntentID=oid, Allowed=True, Reason="authorized")
            fields(it, Target=self.it[orig]["Run"]["Subject"])
            check(
                self.execs[who, it["Sequence"]]["Order"]
                < self.orders["authorization", oid]
                < self.orders["reply", oid],
                "first reply order",
            )
            by_scope[orig].append(oid)
        for op, out in self.pub.items():
            i = self.ad[op]["Invocation"]
            fields(
                out,
                ID=identity("alan/invocation-outcome/v1", i["ID"]),
                OperationID=op,
                InvocationID=i["ID"],
                TemplateID=i["TemplateID"],
                Caller=i["Caller"],
                DecisionOrder=self.orders["outcome", op],
            )
            check(out == dec(results[i["ID"]]), "different published result")
            when = out["DecisionOrder"]
            who = self.ad[op]["Members"][0]["Subject"]
            freeze = self.freeze[i["ID"]]
            fields(freeze, Run=run_id(who), DecisionOrder=self.orders["freeze", i["ID"]])
            check(
                self.execs[who, freeze["Sequence"]]["Order"] < freeze["DecisionOrder"] < when,
                "End/freeze/publication order",
            )
            replies = by_scope[op]
            check(
                set(freeze["ReplyIntents"]) == set(replies) and len(replies) <= 1, "captured replies unsettled"
            )
            if replies:
                (reply,) = replies
                check(self.orders["reply", reply] < when, "reply after first publication")
                fields(out, Response=self.it[reply]["Payload"], ResponsePresent=True)
            else:
                fields(out, Response=None, ResponsePresent=False)
            children = self.children[who]
            check(set(freeze["InvocationIntents"]) == set(children.values()), "captured child obligations")
            for child in children.values():
                check(child in self.pub and self.pub[child]["DecisionOrder"] < when, "early parent publication")
                seq = self.selected(child)
                check(
                    seq is not None and self.execs[who, seq]["Order"] < when,
                    "parent published before selecting child",
                )
            ob = dec(self.obs[self.deliv[op]["observation_id"]])
            p = {
                "status": tagged(out["Status"]),
                "reason": tagged(out["Reason"]),
                "response_present": tagged(out["ResponsePresent"]),
            }
            p.update({"response." + k: v for k, v in (out["Response"] or {}).items()})
            fields(
                ob,
                Operation=op,
                ID=identity("alan/invocation-result-observation/v1", out["ID"]),
                Run=i["Caller"],
                Source=i["TemplateID"],
                SourceGeneration=1,
                Kind="invocation",
                Provenance="runtime-invocation",
                Obligation=self.it[op]["Obligation"],
                Payload=p,
            )
            if self.selected(op) is not None:
                check(
                    when < self.execs[i["Caller"]["Subject"], self.selected(op)]["Order"],
                    "selection predates outcome",
                )

    def selected(self, op):
        if op in self.deliv:
            return self.obs[self.deliv[op]["observation_id"]]["consumed_sequence"]
        return None

    def child(self, op):
        return self.ad[op]["Members"][0]["Subject"]


# A term is a leaf ordinal or the ordered children of a unary/binary node.
SHAPES = {
    "fork": (0, 1),
    "left-nested": ((0, 1), 2),
    "right-nested": (0, (1, 2)),
    "relay-above": ((0, 1),),
    "relay-left": ((0,), 1),
    "consumer": ((0,), (1, 2)),
}


def oracle(term, qs):
    if type(term) is int:
        q = qs[term]
        mode, row = q["mode"], dict(id=q["id"], value=1 - q["value"])
        if mode == "pending":
            return q, None, 4, 3
        tag = {"answer": "answer", "fail": "candidate_failed", "absent": "candidate_missing_reply"}[mode]
        return q, dict(id=q["id"], classification=tag, answers=[row] if mode == "answer" else []), 4, 3
    req, report, slots, depths, pending = {}, {}, 1, [], False
    for j, ch in enumerate(term):
        prefix = ("l_", "r_")[j] if len(term) == 2 else ""
        qr, out, b, h = oracle(ch, qs)
        req.update({prefix + k: v for k, v in qr.items()})
        report.update({prefix + k: v for k, v in (out or {}).items()})
        pending |= out is None
        slots += b
        depths.append(h)
    return req, None if pending else report, slots, 1 + max(depths)


def boundary(s, op, expected, selected, reason="member_succeeded"):
    check((s.selected(op) is not None) == selected, "incorrect original selection")
    if reason == "pending":
        check(
            op not in s.pub and not selected and s.projection(s.child(op))["Status"] == "open",
            "invented publication",
        )
        return
    out = s.pub[op]
    fields(
        out,
        Reason=reason,
        ResponsePresent=expected is not None,
        Status="succeeded" if reason == "member_succeeded" else "failed",
    )
    check(
        (record(out["Response"]) if expected is not None else out["Response"]) == expected,
        "wrong published payload",
    )


def waiting(s, who, op):
    p = s.projection(who)
    x = s.program(who)["Code"][p["PC"]]
    fields(p, Status="open")
    check(
        x["Op"] == "receive"
        and x["Match"]["Kind"] == "invocation"
        and value(get(x["Match"]["Operation"], p["Data"])) == op
        and op not in s.pub,
        "wrong pending frontier",
    )


def walk(s, op, term, assignment, qs, selected):
    who = s.child(op)
    req, out, _, _ = oracle(term, qs)
    check(record(s.it[op]["Payload"]) == req, "path request mismatch")
    diagnosed = out is not None and any(v != "answer" for k, v in out.items() if k.endswith("classification"))
    reason = "pending" if out is None else "member_failed" if diagnosed else "member_succeeded"
    boundary(s, op, out, selected, reason)
    children = s.children[who]
    if type(term) is int:
        letter = assignment[term]
        check(
            s.runs[who]["program_id"] == OLD["J" + letter] and set(children) == {"candidate", "successor"},
            "wrong J occurrence",
        )
        e, side = children["candidate"], children["successor"]
        check(
            s.runs[s.child(side)]["program_id"] == OLD["S"] and s.runs[s.child(e)]["program_id"] == OLD[letter],
            "substituted leaf byte identity",
        )
        check(
            record(s.it[e]["Payload"]) == req
            and record(s.it[side]["Payload"]) == {k: req[k] for k in ("id", "value")},
            "J call arguments",
        )
        check(s.it[e]["Sequence"] < s.it[side]["Sequence"], "J emit order")
        boundary(s, side, {"id": req["id"], "value": 0}, True)
        expected = dict(id=req["id"], value=1 - req["value"])
        mode = req["mode"]
        rawreason = {
            "answer": "member_succeeded",
            "fail": "member_failed",
            "absent": "missing_reply",
            "pending": "pending",
        }[mode]
        boundary(s, e, expected if mode == "answer" else None, mode != "pending", rawreason)
        if mode == "pending":
            waiting(s, who, e)
            ep = s.projection(s.child(e))
            check(
                ep["PC"] == (11 if letter == "A" else 12)
                and s.program(s.child(e))["Code"][ep["PC"]]["Match"]["Kind"] == "message",
                "candidate not at designated unmatched receive",
            )
        else:
            check(s.selected(side) < s.selected(e), "J outcome order")
        if letter == "B":
            helper = s.children[s.child(e)]["complement"]
            check(
                s.runs[s.child(helper)]["program_id"] == OLD["h"]
                and record(s.it[helper]["Payload"]) == dict(req, mode="answer"),
                "helper identity/request",
            )
            boundary(s, helper, expected, True)
        return
    roles = ("child",) if len(term) == 1 else ("left", "right")
    check(tuple(sorted(children)) == roles, "wrong constructor children")
    expected_name = "phase1t-relay" if len(term) == 1 else "phase1t-fork"
    fields(s.program(who)["Author"], Name=expected_name)
    blocked, lastseq = False, 0
    for role, ch in zip(roles, term):
        child = children[role]
        _, expected, _, _ = oracle(ch, qs)
        if expected is None and not blocked:
            waiting(s, who, child)
            blocked = True
        walk(s, child, ch, assignment, qs, not blocked)
        if not blocked:
            seq = s.selected(child)
            check(
                max(s.it[o]["Sequence"] for o in children.values()) < seq and lastseq < seq,
                "commit/select order",
            )
            lastseq = seq
    if len(roles) == 2:
        check(s.it[children["left"]]["Sequence"] < s.it[children["right"]]["Sequence"], "fork commitment order")


def replay(s, rep):
    fields(
        rep,
        Namespace=s.raw["Namespace"],
        Roots=s.raw["Roots"],
        HistoryCount=len(s.t["history"]),
        IntentCount=len(s.it),
    )
    check(
        len(rep["Runs"]) == len(s.runs) and {r["ID"]["Subject"] for r in rep["Runs"]} == s.runs.keys(),
        "replay run set",
    )
    for r in rep["Runs"]:
        who = r["ID"]["Subject"]
        fields(
            r,
            ID=run_id(who),
            ProgramID=s.runs[who]["program_id"],
            Epoch=s.runs[who]["epoch"],
            Sequence=s.runs[who]["sequence"],
            Revision=1,
            Projection=s.projection(who),
        )


def automatic(s, shape, assignment, qs, report):
    term = SHAPES[shape]
    req, out, b, h = oracle(term, qs)
    top = s.children["root"]["tree"]
    decision = (
        "pending"
        if out is None
        else "diagnosed"
        if any(v != "answer" for k, v in out.items() if k.endswith("classification"))
        else "answer"
    )
    fields(s.program("root")["Author"], Name="phase1t-outer")
    check(record(json.loads(s.runs["root"]["initial"])["Data"]) == req, "outer initializer")
    walk(s, top, term, assignment, qs, out is not None)
    p = s.projection("root")
    if out is None:
        waiting(s, "root", top)
        check(
            s.root["Outcome"] is None
            and "decision" not in p["Data"]
            and not s.t["composition_root_publications"],
            "invented root boundary",
        )
    else:
        check(
            value(p["Data"]["decision"]) == decision
            and {k[7:]: value(v) for k, v in p["Data"].items() if k.startswith("report_")} == out,
            "outer report",
        )
        (r,) = s.t["composition_root_publications"]
        o = s.root["Outcome"]
        check(o == dec(r), "root result row")
        fields(
            o,
            Status="failed" if decision == "diagnosed" else "succeeded",
            Reason="member_failed" if decision == "diagnosed" else "member_succeeded",
            ResponsePresent=False,
            DecisionOrder=s.orders["outcome", "root:" + s.rid],
        )
        rootfreeze = s.freeze["root:" + s.rid]
        check(
            rootfreeze["InvocationIntents"] == [top]
            and s.execs["root", s.selected(top)]["Order"] < rootfreeze["DecisionOrder"] < o["DecisionOrder"],
            "root published before scope settled",
        )
    spend = b - assignment.count("A")
    fields(s.root["Limits"], MaxInvocations=b, MaxCreatedRuns=b, MaxDepth=h, MaxEffects=0, MaxInFlight=1)
    check(len(s.charged) == spend and len(s.runs) == spend + 1, "per-occurrence spend")
    check(
        {r["Subject"] for r in report["OpenRuns"]}
        == {who for who in s.runs if s.projection(who)["Status"] == "open"},
        "unreported open runs",
    )
    check(s.auth.keys() == s.it.keys() and len(s.it) == len(s.ad) + len(s.reply), "unserviced original intent")
    check(
        len(s.obs) == len(s.charged) + len(s.pub)
        and all(s.obs[dec(r)["ID"]]["consumed_sequence"] is not None for r in s.t["composition_requests"]),
        "unserviced original entry",
    )
    fields(
        report,
        Quiescent=True,
        RejectedAdmissions=0,
        Roots=s.raw["Roots"],
        UnknownExternal=[],
        UnknownAdapters=[],
        PendingDeliveries=[],
    )
    step = (b + 1) * max(len(p["Code"]) + sum(x["Op"] == "next" for x in p["Code"]) for p in s.progs.values())
    check(len(s.t["history"]) <= step and report["Steps"] <= step and report["Dispatches"] <= 3 * b, "watchdog")
    check(
        not any(
            s.t[k]
            for k in (
                "external_attempts",
                "external_receipts",
                "adapter_attempts",
                "adapter_results",
                "composition_reservations",
                "lifecycle_results",
            )
        ),
        "unexpected effect or closure",
    )
    return decision, b, h, step


def matrix():
    keys = set()
    for a, i, j in itertools.product(("AA", "AB", "BA", "BB"), range(4), range(4)):
        keys.add(("fork", a, ((1, 0, MODES[i]), (1 + (i + j) % 2, 1, MODES[j]))))
    for shape in ("left-nested", "right-nested", "relay-above", "relay-left"):
        n = 2 if shape.startswith("relay") else 3
        for modes in (
            ("answer", "fail", "absent"),
            ("pending", "answer", "fail"),
            ("fail", "pending", "answer"),
            ("answer", "absent", "pending"),
        ):
            keys.add((shape, "BAB"[:n], tuple(zip((1, 1, 2)[:n], (0, 1, 0)[:n], modes[:n]))))
    return keys


def manual(s, a):
    kind = a["kind"]
    top = s.children["root"]["tree"]
    fork = s.child(top)
    roles = s.children[fork]
    if kind.startswith(("tight-", "late-", "delayed-")):
        base_ops = {top, *roles.values()}
        for j in roles.values():
            kids = s.children[s.child(j)]
            check(
                set(kids) == {"candidate", "successor"} and s.runs[s.child(j)]["program_id"] == OLD["JB"],
                "manual all-B fixture",
            )
            base_ops.update(kids.values())
        ordered = sorted(s.charged, key=lambda op: s.charged[op]["DecisionOrder"])
        check(len(base_ops) == 7 and set(ordered[:7]) == base_ops, "seven baseline admissions")
    if kind.startswith("tight-"):
        depth = kind == "tight-depth"
        right = "left" if depth else "right"
        e = s.child(s.children[s.child(roles[right])]["candidate"])
        op = s.children[e]["complement"]
        reason = {"tight-I": "invocation_limit", "tight-R": "created_run_limit", "tight-depth": "depth_limit"}[
            kind
        ]
        fields(s.ad[op], Accepted=False, Reason=reason)
        fields(a, denied=op, reason=reason)
        fields(
            s.root["Limits"],
            MaxInvocations=8 if kind == "tight-I" else 9,
            MaxCreatedRuns=8 if kind == "tight-R" else 9,
            MaxDepth=3 if depth else 4,
            MaxInFlight=1,
            MaxEffects=0,
        )
        check(
            len(s.charged) == (7 if depth else 8)
            and s.ad[op]["Invocation"]["Depth"] == 4
            and op not in s.charged,
            "wrong refusal scope",
        )
        left_e = s.child(s.children[s.child(roles["left"])]["candidate"])
        right_e = s.child(s.children[s.child(roles["right"])]["candidate"])
        helpers = [value(s.projection(e)["Data"]["helper_operation"]) for e in (left_e, right_e)]
        check(
            len(set(helpers)) == 2 and (depth or ordered[7] == helpers[0]), "unexpected prior helper admission"
        )
        check(
            all(c["DecisionOrder"] < s.orders["admission", op] for c in s.charged.values()), "refusal prestate"
        )
    elif kind.startswith("delayed-"):
        win = kind.split("-")[1]
        lose = "right" if win == "left" else "left"
        prefix = Store(a["prefix"])
        op = roles[win]
        opposite = s.child(s.children[s.child(roles[lose])]["candidate"])
        fields(a, winning=op, blocked_member=run_id(opposite))
        check(
            prefix.projection(opposite)["PC"] == 1 and len(prefix.charged) == 8 and op in prefix.pub,
            "sibling not serviced at B1 cut",
        )
        credit = s.children[s.child(s.children[s.child(op)]["candidate"])]["complement"]
        check(
            prefix.execs[opposite, 1]["Order"]
            < prefix.charged[credit]["DecisionOrder"]
            < prefix.pub[op]["DecisionOrder"],
            "independent sibling debit",
        )
        qs = [
            dict(id=1, value=0, mode="pending" if win == "right" else "fail"),
            dict(id=1, value=1, mode="answer" if win == "right" else "pending"),
        ]
        automatic(s, "fork", "BB", qs, a["report"])
        if win == "right":
            rej = a["rejected"]
            fields(
                rej,
                error="ineligible observation",
                after=rej["before"],
                observation_id=s.deliv[op]["observation_id"],
            )
            pre = rej["before"]
            fields(pre, ID=run_id(fork), ProgramID=s.runs[fork]["program_id"])
            waiting(s, fork, roles[lose])
            check(
                pre["Projection"] == s.projection(fork) and pre["Sequence"] == s.runs[fork]["sequence"],
                "wrong-selection trial changed fork",
            )
    elif kind.startswith("late-"):
        cancel = kind == "late-cancel"
        e = s.child(s.children[s.child(roles["left"])]["candidate"])
        helper = s.children[e]["complement"]
        h, reply = s.child(helper), a["reply"]
        fields(a, helper=helper)
        fields(s.it[reply], Run=run_id(h), Target=e)
        check(
            record(s.it[reply]["Payload"]) == {"id": 1, "value": 1} and s.runs[h]["program_id"] == OLD["h"],
            "late bound payload",
        )
        end = s.execs[h, s.runs[h]["sequence"]]["Order"]
        send = s.execs[h, s.it[reply]["Sequence"]]["Order"]
        auth = s.reply[reply]["Authorization"]
        fields(auth, Allowed=True, ClosureOperation="")
        fields(auth["Source"], Run=run_id(h), Revision=1, Sequence=s.runs[h]["sequence"])
        check(
            s.projection(h)["Status"] == "succeeded"
            and send < end < s.orders["authorization", reply] < s.orders["reply", reply],
            "late first authorization",
        )
        check(
            s.projection(e)["Status"] == "open" and s.projection(fork)["Status"] == "open",
            "actual caller/ancestor closure",
        )
        ops = s.t["lifecycle_results"]
        if cancel:
            (row,) = ops
            lifecycle = json.loads(row["result"])
            fields(lifecycle, Accepted=True, Reason="closed")
            fields(lifecycle["Before"], Run=run_id("root"), Revision=1)
            fields(lifecycle["After"], Run=run_id("root"), Revision=1)
            check(
                lifecycle["Before"]["Projection"]["Status"] == "open"
                and lifecycle["After"]["Projection"]["Status"] == "cancelled",
                "lifecycle pre/post",
            )
            cid = a["cancel"]
            check(
                lifecycle["Request"]["Intent"] == s.it[cid]
                and s.it[cid]["Run"] == run_id("operator")
                and json.loads(s.auth[cid]["result"])["Allowed"] is True,
                "operator authorization",
            )
            check(
                any(
                    p["kind"] == "lifecycle"
                    and p["source"] == "operator"
                    and p["target"] == "root"
                    and p["operation"] == "CancelRun"
                    and p["revision"] == 1
                    for p in s.t["permissions"]
                ),
                "operator grant",
            )
            check(
                send < s.orders["lifecycle", cid] < end and s.projection("root")["Status"] == "cancelled",
                "ancestor cancellation order",
            )
        else:
            check(not a["cancel"] and not ops and s.projection("root")["Status"] == "open", "open late control")
    elif kind == "optional-fault":
        (who,) = (k for k, v in s.runs.items() if v["program_id"] == a["bad_program"])
        p = s.projection(who)
        ins = s.program(who)["Code"][p["PC"]]
        fields(a["report"], Quiescent=False)
        fields(a, error='missing or invalid data "selected_id"')
        fields(ins, Op="set", Dst="unguarded")
        original = value(p["Data"]["saved"])
        selected = s.selected(original)
        pre = next(
            json.loads(h["successor"])
            for h in s.t["history"]
            if h["subject"] == who and h["sequence"] == selected - 1
        )
        check(
            value(pre["Data"]["selected_id"]) == 73
            and p["Status"] == "open"
            and s.parents[who] not in s.pub
            and s.root["Outcome"] is None,
            "fault did not clear seeded register or fabricated a completion",
        )
        check(
            ins["A"]["Ref"] == "selected_id"
            and "selected_id" not in p["Data"]
            and value(p["Data"]["present"]) is False
            and s.selected(original) == s.runs[who]["sequence"]
            and s.pub[original]["Reason"] == "missing_reply",
            "fault was not absent-field read",
        )
    else:
        invalid, mode = kind.rsplit("-", 1)
        rc = roles["child"]
        owner = s.child(rc)
        bad, good = (s.children[owner][k] for k in ("left", "right"))
        if invalid == "wrong-answer":
            boundary(s, bad, {"id": 1, "classification": "answer", "answers": [{"id": 1, "value": 0}]}, True)
        else:
            boundary(s, bad, None, True, "missing_reply" if invalid == "absent-success" else "member_failed")
        check(value(s.projection(owner)["Data"]["tree_valid"]) is False, "invalid evidence accepted")
        if mode == "pending":
            waiting(s, owner, good)
            check(
                not any(o in s.pub for o in (rc, top)) and s.root["Outcome"] is None,
                "early invalid publication",
            )
        else:
            boundary(s, good, {"id": 1, "classification": "answer", "answers": [{"id": 1, "value": 0}]}, True)
            boundary(s, rc, None, True, "member_failed")
            boundary(s, top, None, True, "member_failed")
            check(
                s.selected(bad) < s.selected(good)
                and value(s.projection("root")["Data"]["decision"]) == "invalid_tree_outcome"
                and s.root["Outcome"]["Status"] == "failed",
                "invalid propagation",
            )


def consumer(directory):
    m = json.loads(read(directory / "consumer-manifest.json.gz")["output"])
    x = json.loads(read(directory / "consumer-execution.json.gz")["output"])
    tree, provision = m["Tree"], m["Provision"]
    s = Store(x["Inspection"])
    qs = [
        dict(id=2, value=1, mode="fail"),
        dict(id=2, value=0, mode="absent"),
        dict(id=2, value=1, mode="answer"),
    ]
    decision, b, _, steps = automatic(s, "consumer", "ABB", qs, x["Report"])
    fields(
        x,
        Term="fork(relay(A),fork(B,B))",
        Template=tree["Descriptor"]["ID"],
        MaxSteps=steps + 1,
        MaxDispatch=3 * b + 1,
    )
    replay(s, x["Replay"])
    check(
        x["Root"] == next(r for r in x["Replay"]["Runs"] if r["ID"] == run_id("root")), "consumer root snapshot"
    )
    check(
        tree["Descriptor"] == s.tmpl[x["Template"]]
        and s.it[s.children["root"]["tree"]]["Target"] == x["Template"],
        "consumer tree",
    )
    templates = index(provision["Composition"]["Templates"], "ID")
    check(
        templates == s.tmpl == index([tree["Descriptor"]] + tree["Dependencies"], "ID"),
        "template manifest coverage",
    )
    pids = index(provision["Programs"], "ID")
    check(
        pids.keys() == s.progs.keys()
        and index(tree["Programs"], "ID").keys() == pids.keys() - {x["Root"]["ProgramID"]},
        "program manifest coverage",
    )
    for r in provision["Programs"] + tree["Programs"]:
        raw = base64.b64decode(r["Bytes"], validate=True)
        check(digest(raw) == r["ID"] and json.loads(raw) == s.progs[r["ID"]], "consumer retained byte identity")
    (dep,) = s.t["composition_deployment"]
    deployed = dec(dep)
    check(
        provision["Runs"] == deployed["Runs"] and provision["Subjects"] == deployed["Subjects"],
        "consumer fixed deployment",
    )
    req = s.program("root")["Author"]["Invocations"]["tree"]
    fields(req, Input=tree["Input"], Output=tree["Output"])
    (root,) = provision["Composition"]["Roots"]
    fields(root, Limits=s.root["Limits"], Members=[run_id("root")], Run=run_id("root"))
    check(set(root["Templates"]) == set(templates), "consumer root catalogue")
    return dict(
        decision=decision,
        programs=len(pids),
        templates=len(templates),
        charges=len(s.charged),
        steps=x["Report"]["Steps"],
        dispatches=x["Report"]["Dispatches"],
        outer_program=x["Root"]["ProgramID"],
        tree_template=x["Template"],
    )


def rejected(label, action, reason):
    try:
        action()
    except ValueError as err:
        check(reason in str(err), f"{label} rejected for unrelated reason: {err}")
        return label
    raise ValueError(f"accepted detached mutation {label}")


def detached(cases):
    from copy import deepcopy

    # Alter decoded material in memory. Never submit these mutations to the Store.
    a = next(
        a
        for a in cases
        if a["shape"] == "fork"
        and a["assignment"] == "BB"
        and [q["Mode"] for q in a["requests"]] == ["fail", "answer"]
    )
    s = Store(a["inspection"])
    fork = s.child(s.children["root"]["tree"])
    left, right = (s.children[fork][k] for k in ("left", "right"))
    results = []
    s.deliv[left], s.deliv[right] = s.deliv[right], s.deliv[left]
    results.append(rejected("swapped sibling deliveries", s.check_publications, "Operation:"))
    s.deliv[left], s.deliv[right] = s.deliv[right], s.deliv[left]
    qs = [{k.lower(): v for k, v in q.items()} for q in a["requests"]]

    def case():
        automatic(s, "fork", "BB", qs, a["report"])

    saved = deepcopy(s.pub[left])
    s.pub[left]["Response"]["classification"] = tagged("candidate_missing_reply")
    results.append(rejected("substituted failure diagnosis", case, "wrong published payload"))
    s.pub[left] = saved
    # Move a real captured-reply disposition past the original publication.
    (oid,) = s.freeze[s.ad[left]["Invocation"]["ID"]]["ReplyIntents"]
    s.orders["reply", oid] = s.pub[left]["DecisionOrder"] + 1
    results.append(
        rejected("publication before captured reply", s.check_publications, "reply after first publication")
    )
    fault, late, delay = (
        next(a for a in cases if a.get("kind") == k) for k in ("optional-fault", "late-cancel", "delayed-right")
    )
    f = Store(fault["inspection"])
    altered = dict(fault, report=dict(fault["report"], Quiescent=True))
    results.append(rejected("operand fault relabelled quiescent", lambda: manual(f, altered), "Quiescent:"))
    late_store = Store(late["inspection"])
    late_store.reply[late["reply"]]["Authorization"]["Allowed"] = False
    results.append(
        rejected("denied first permission labelled accepted", lambda: manual(late_store, late), "Allowed:")
    )
    late_store = Store(late["inspection"])
    late_store.orders["lifecycle", late["cancel"]] = late_store.orders["reply", late["reply"]] + 1
    results.append(
        rejected(
            "ancestor closure moved after reply",
            lambda: manual(late_store, late),
            "ancestor cancellation order",
        )
    )
    d = Store(delay["inspection"])
    top = d.children["root"]["tree"]
    d.pub[top] = deepcopy(s.pub[s.children["root"]["tree"]])
    results.append(
        rejected("product invented behind pending left", lambda: manual(d, delay), "invented publication")
    )
    return results


def records(directory):
    repo = directory.parents[3]
    idx = json.loads((directory / "run-index.json").read_text())
    check(
        {p.relative_to(directory).as_posix() for p in (directory / "runs").rglob("*.json.gz")}
        == set(index(idx, "file")),
        "raw index coverage",
    )
    logs = {}
    for r in idx:
        path = directory / r["file"]
        raw = gzip.decompress(path.read_bytes())
        check(raw.endswith(b"\n") and digest(raw[:-1]) == r["value_sha256"], "retained JSON value digest")
        name = r["command"]
        if name not in logs:
            logs[name] = read(directory / name)
        expected_line = "/" + path.parent.name + "/" + path.name + " value_sha256=" + r["value_sha256"]
        check(logs[name]["output"].count(expected_line) == 1, "command omitted/duplicated retained hash")
    for name in ("consumer-build", "consumer-manifest", "consumer-execution"):
        logs[name + ".json.gz"] = read(directory / (name + ".json.gz"))
    for name, log in logs.items():
        fields(log, exit=0)
        check(
            log["base"] == "153e981879cee32a67fd4784e9c162542e2dc421"
            and digest(log["output"].encode()) == log["output_sha256"],
            "command output identity",
        )
        for k, v in log["source_sha256"].items():
            if k in ("examples/owningtree/tree.go", "examples/owningtree/validate.go") or (
                name.startswith("consumer-")
                and k.startswith("examples/owningtree/consumer/")
                and not k.endswith("_test.go")
            ):
                check(
                    digest((repo / k).read_bytes()) == v and k not in log["changed_during"],
                    f"{name} executable input changed: {k}",
                )
    # Dir keys are first 12 hex digits of SHA-256 of the JSON string test.Name().
    for group, names in (("automatic", "automatic-leaves.txt"), ("manual", "manual-leaves.txt")):
        names = (directory / names).read_text().splitlines()
        expected = {digest(compact(n).encode())[:12]: n for n in names}
        check({p.name for p in (directory / "runs" / group).iterdir()} == expected.keys(), "leaf/test identity")
        for r in idx:
            p = Path(r["file"])
            if p.parts[1] == group:
                check(
                    "--- PASS: " + expected[p.parent.name] + " (" in logs[r["command"]]["output"],
                    "named test did not pass",
                )
    return idx


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", type=Path, default=HERE)
    directory = parser.parse_args().evidence.resolve()
    idx = records(directory)
    kinds, matrix_seen, cases = set(), set(), []
    counts = Counter()
    programs, templates, placements = set(), set(), {}
    for f in sorted((directory / "runs" / "automatic").glob("*/case.json.gz")):
        a = read(f)
        s = Store(a["inspection"])
        qs = [{k.lower(): v for k, v in q.items()} for q in a["requests"]]
        key = (a["shape"], a["assignment"], tuple((q["id"], q["value"], q["mode"]) for q in qs))
        check(key not in matrix_seen, "duplicate automatic configuration")
        matrix_seen.add(key)
        decision, b, h, step = automatic(s, a["shape"], a["assignment"], qs, a["report"])
        fields(a, decision=decision, expected_slots=b, depth=h, steps_bound=step, dispatch_bound=3 * b)
        replay(s, read(f.parent / "replay.json.gz"))
        counts.update(
            {
                decision: 1,
                "committed_steps": len(s.t["history"]),
                "accepted_invocations": len(s.charged),
                "published_child_outcomes": len(s.pub),
                "bound_reply_dispositions": len(s.reply),
            }
        )
        placement = a["shape"] + "/" + a["assignment"]
        pin = dict(
            outer_program=s.runs["root"]["program_id"], tree_template=s.it[s.children["root"]["tree"]]["Target"]
        )
        check(placements.setdefault(placement, pin) == pin, "constructor bytes changed across requests")
        templates.update(s.tmpl)
        programs.update(s.progs)
        if a["shape"] == "fork" and a["assignment"] == "BB" and [q["mode"] for q in qs] == ["fail", "answer"]:
            cases.append(a)
    check(matrix_seen == matrix(), "incomplete frozen automatic matrix")
    for f in sorted((directory / "runs" / "manual").glob("*/manual.json.gz")):
        a = read(f)
        manual(Store(a["inspection"]), a)
        check(a["kind"] not in kinds, "duplicate manual control")
        kinds.add(a["kind"])
        if a["kind"] in ("optional-fault", "late-cancel", "delayed-right"):
            cases.append(a)
    expected = {
        "tight-I",
        "tight-R",
        "tight-depth",
        "delayed-left",
        "delayed-right",
        "late-open",
        "late-cancel",
        "optional-fault",
    }
    expected.update(
        f"{k}-{m}" for k in ("wrong-answer", "absent-failure", "absent-success") for m in ("answer", "pending")
    )
    check(kinds == expected and len(idx) == 174, "manual/raw coverage")
    print(
        json.dumps(
            dict(
                automatic_cases=len(matrix_seen),
                automatic=dict(counts),
                automatic_distinct_programs=len(programs),
                automatic_distinct_templates=len(templates),
                placements=placements,
                manual=sorted(kinds),
                raw_files=len(idx),
                consumer=consumer(directory),
                detached_rejected=detached(cases),
            ),
            indent=2,
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
