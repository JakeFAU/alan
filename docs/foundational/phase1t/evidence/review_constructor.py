"""Probe the archived same-template Fork after typed selections. Run from repository root."""

import base64
import collections
import gzip
import hashlib
import itertools
import json

if not __debug__:
    raise RuntimeError("enable assertions for this audit")

m = json.load(gzip.open("docs/foundational/phase1t/evidence/consumer-manifest.json.gz"))
assert (
    m["exit"] == 0
    and hashlib.sha256(m["output"].encode()).hexdigest() == m["output_sha256"]
)
p = json.loads(m["output"])["Provision"]
programs = {}
for a in p["Programs"]:
    b = base64.b64decode(a["Bytes"])
    assert hashlib.sha256(b).hexdigest() == a["ID"]
    programs[a["ID"]] = json.loads(b)
base = dict(id=1, value=1, mode="pending")
good = {
    "status": "succeeded",
    "reason": "member_succeeded",
    "response_present": True,
    "response.id": 1,
    "response.classification": "answer",
    "response.answers": [dict(id=1, value=0)],
}


def sample(tag):
    result = good.copy()
    if tag != "answer":
        result.update(
            {
                "response.classification": tag,
                "response.answers": [],
                "status": "failed",
                "reason": "member_failed",
            }
        )
    return result


def changed(d, **updates):
    return dict(d, **updates)


def run(code, children):
    regs, curs, commits, selects, sends = {}, {}, {}, [], []
    pc = 0

    def val(op):
        if op.get("Ref"):
            return regs[op["Ref"]]
        v = op["Literal"]
        return v[{"int": "Number", "bool": "Flag", "string": "Text"}[v["Kind"]]]

    while True:
        i = code[pc]
        nxt = pc + 1
        if i["Op"] == "receive":
            m = i["Match"]
            name = val(m["Obligation"])
            if m["Kind"] == "invocation":
                assert (
                    list(commits) == ["left", "right"]
                    and val(m["Operation"]) == commits[name]
                    and m["Provenance"] == "runtime-invocation"
                )
                env = children[name]
                if env is None:
                    return tuple(selects), tuple(commits), None, sends, name
                selects.append(name)
            else:
                env = {
                    prefix + field: v
                    for prefix in ("l_", "r_")
                    for field, v in base.items()
                }
            for k, d in i["Bind"].items():
                if k not in env:
                    assert k.startswith("response.") and not env["response_present"]
                    regs.pop(d, None)
                else:
                    regs[d] = env[k]
        elif i["Op"] == "set":
            regs[i["Dst"]] = val(i["A"])
        elif i["Op"] == "jump":
            nxt = i["Target"]
        elif i["Op"] == "branch":
            nxt = i["Target"] if val(i["A"]) == val(i["B"]) else i["Otherwise"]
        elif i["Op"] == "emit":
            e = i["Effect"]
            pay = {k: val(v) for k, v in e["Payload"].items()}
            if e["Kind"] == "invocation":
                assert pay == base and not selects
                name = val(e["Obligation"])
                assert name not in commits
                commits[name] = ("op", pc)
                regs[i["Dst"]] = commits[name]
            else:
                assert val(e["Target"]) == "caller" and val(e["Obligation"]) == "work"
                sends.append(pay)
        elif i["Op"] == "foreach":
            curs[pc] = iter(val(i["A"]))
        elif i["Op"] == "next":
            start = i["Target"]
            item = next(curs[start], None)
            if item is None:
                del curs[start]
                nxt = code[start]["Otherwise"]
            else:
                for k, d in i["Bind"].items():
                    regs[d] = item[k]
        elif i["Op"] == "loop":
            nxt = i["Target"] + 1
        elif i["Op"] == "end":
            return (
                tuple(selects),
                tuple(commits),
                (i["Outcome"], regs["decision"]),
                sends,
                None,
            )
        else:
            raise AssertionError(i)
        pc = nxt


code = programs["9f4c3c4cb8dba6c69eddac76768960ca2a1bf877ca79241b0cafb05195570829"]["Code"]
assert len(code) == 124 and code[1]["Effect"]["Target"] == code[2]["Effect"]["Target"]
fail = sample("candidate_failed")
missing = sample("candidate_missing_reply")
cases = [
    ("answer", good),
    ("diagnosed", fail),
    ("diagnosed", missing),
    (
        "invalid",
        {"status": "failed", "reason": "member_failed", "response_present": False},
    ),
    (
        "invalid",
        {"status": "failed", "reason": "missing_reply", "response_present": False},
    ),
    ("invalid", changed(good, status="failed", reason="member_failed")),
    ("invalid", changed(fail, status="succeeded", reason="member_succeeded")),
    ("invalid", changed(fail, reason="missing_reply")),
    ("invalid", changed(good, **{"response.id": 2})),
    ("invalid", changed(good, **{"response.answers": [dict(id=2, value=0)]})),
    ("invalid", changed(good, **{"response.answers": [dict(id=1, value=1)]})),
    ("invalid", changed(good, **{"response.answers": []})),
    ("invalid", changed(fail, **{"response.answers": [dict(id=1, value=0)]})),
    ("invalid", sample("invalid_outcome")),
    ("pending", None),
]
counts = collections.Counter()
for (expectl, left), (expectr, right) in itertools.product(cases, repeat=2):
    selected, invokes, decision, sends, awaiting = run(
        code, {"left": left, "right": right}
    )
    assert invokes == ("left", "right")
    if left is None or right is None:
        assert (
            awaiting == ("left" if left is None else "right")
            and selected == (() if left is None else ("left",))
            and not sends
            and decision is None
        )
        counts["await_" + awaiting] += 1
    else:
        assert selected == ("left", "right")
        if "invalid" in (expectl, expectr):
            assert decision == ("failed", "invalid_tree_outcome") and not sends
        else:
            decision_name = (
                "diagnosed" if "diagnosed" in (expectl, expectr) else "answer"
            )
            assert decision == (
                "failed" if decision_name == "diagnosed" else "succeeded",
                decision_name,
            )
            assert sends == [
                {
                    prefix + k.removeprefix("response."): v
                    for prefix, envelope in [("l_", left), ("r_", right)]
                    for k, v in envelope.items()
                    if k.startswith("response.")
                }
            ]
        counts[decision[1]] += 1
assert dict(counts) == dict(
    answer=1, diagnosed=8, invalid_tree_outcome=187, await_right=14, await_left=15
)
print(
    "verified",
    len(programs),
    "program digests;",
    sum(counts.values()),
    "retained fork control probes",
    dict(counts),
)
