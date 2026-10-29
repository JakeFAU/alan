#!/usr/bin/env python3
"""Check fixed weak/closure obligations on original JSON rows, without ALAN imports."""
import gzip
import hashlib
import json
import pathlib
import sys

base = pathlib.Path(sys.argv[1])

def record(test, name):
    token = hashlib.sha256(json.dumps(test, separators=(",", ":")).encode()).hexdigest()[:12]
    with gzip.open(base / token / (name + ".json.gz")) as stream:
        return json.load(stream)

def rows(inspection, table):
    return [json.loads(row["descriptor"]) for row in inspection["Tables"][table]]

checks = []
for candidate in ("weak-a", "weak-b"):
    for i, r in ((1, 1), (2, 2), (3, 3), (3, 2)):
        test = f"TestSubstitutionProbes/{candidate}-I{i}-R{r}"
        result = record(test, "result")
        decoded = result["decoded"]
        operations = decoded["Operations"]
        candidate_ops = [op for op, facts in operations.items() if facts["Role"] == "candidate"]
        assert len(candidate_ops) == 1, test
        outcome = next(o for o in rows(result["inspection"], "composition_publications") if o["OperationID"] == candidate_ops[0])
        if candidate == "weak-a" or i >= 2:
            assert outcome["Status"] == "succeeded" and outcome["ResponsePresent"], test
            assert outcome["Response"]["id"]["Number"] == 1, test
            assert outcome["Response"]["value"]["Number"] == 0, test
        else:
            assert outcome["Status"] == "failed" and not outcome["ResponsePresent"], test
        s = result["inspection"]
        admissions = rows(s, "composition_admission_decisions")
        reasons = [a["Reason"] for a in admissions if not a["Accepted"]]
        if i == 2:
            assert (reasons == [] if candidate == "weak-a" else reasons == ["invocation_limit"]), test
        if i == 3 and r == 2:
            assert (reasons == [] if candidate == "weak-a" else reasons == ["created_run_limit"]), test
        checks.append(test)

expected = {
    ("a", "entry-commit"): (0, ["source_denied:source_closed"], [("rejected", "source_denied:source_closed", False)]),
    ("a", "entry-admitted"): (1, [], [("succeeded", "member_succeeded", True)]),
    ("a", "reply-commit"): (1, [], [("succeeded", "member_succeeded", True)]),
    ("a", "published"): (1, [], [("succeeded", "member_succeeded", True)]),
    ("b", "entry-commit"): (0, ["source_denied:source_closed"], [("rejected", "source_denied:source_closed", False)]),
    ("b", "entry-admitted"): (1, ["scope_closed"], [("rejected", "scope_closed", False), ("failed", "member_failed", False)]),
    ("b", "helper-commit"): (1, ["scope_closed"], [("rejected", "scope_closed", False), ("failed", "member_failed", False)]),
    ("b", "reply-commit"): (2, [], [("succeeded", "member_succeeded", True), ("succeeded", "member_succeeded", True)]),
    ("b", "published"): (2, [], [("succeeded", "member_succeeded", True), ("succeeded", "member_succeeded", True)]),
}
for (candidate, cut), (debit, denials, envelopes) in expected.items():
    test = f"TestSubstitutionClosure/{candidate}/{cut}"
    result = record(test, "after")
    inspection = result["inspection"]
    root = inspection["Roots"][0]
    assert root["LocalStatus"] == "cancelled" and root["Outcome"]["Status"] == "cancelled", test
    assert (root["Charges"]["Invocations"], root["Charges"]["CreatedRuns"], root["Charges"]["Effects"]) == (debit, debit, 0), test
    assert [a["Reason"] for a in rows(inspection, "composition_admission_decisions") if not a["Accepted"]] == denials, test
    actual = [(o["Status"], o["Reason"], o["ResponsePresent"]) for o in rows(inspection, "composition_publications")]
    assert actual == envelopes, (test, actual)
    checks.append(test)
print(json.dumps({"scope": "fixed weak payload and closure envelope obligations over original records", "records": str(base), "checked": checks, "count": len(checks), "result": "PASS"}, indent=2))
