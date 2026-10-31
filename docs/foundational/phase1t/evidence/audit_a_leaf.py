"""Audit genuine A public Store histories against the 13-instruction leaf argument."""

import gzip
import hashlib
import itertools
import json
import pathlib
import sys


PROGRAM = "74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7"
TEMPLATE = "a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5"


def singleton(rows):
    (row,) = rows
    return row


def check(raw, ident, value, mode):
    record = json.loads(raw)
    tables = record["Inspection"]["Tables"]
    member = json.loads(singleton(tables["composition_members"])["descriptor"])
    run = member["Run"]
    assert member["ProgramID"] == PROGRAM
    assert member["Initial"] == {"PC": 0, "Data": {}, "Status": "open"}
    program = singleton(p for p in tables["programs"] if p["id"] == PROGRAM)
    assert hashlib.sha256(program["bytes"].encode()).hexdigest() == PROGRAM
    admission = json.loads(singleton(tables["composition_admission_decisions"])["descriptor"])
    inv = admission["Invocation"]
    assert admission["Accepted"] and admission["Members"] == [run]
    assert inv["TemplateID"] == TEMPLATE and inv["ID"] == member["InvocationID"]
    assert (inv["Inputs"]["id"]["Number"], inv["Inputs"]["value"]["Number"],
            inv["Inputs"]["mode"]["Text"]) == (ident, value, mode)

    req = json.loads(singleton(tables["composition_requests"])["descriptor"])
    assert req["Operation"] == inv["OperationID"] and req["Run"] == run
    assert req["Provenance"] == "runtime-invocation-request" and req["Kind"] == "message"
    assert req["Source"] == inv["Caller"]["Subject"] and req["SourceGeneration"] == inv["Caller"]["Generation"]
    assert req["Obligation"] == "substitution" and req["Payload"] == inv["Inputs"]
    paths = {"answer": [0, 1, 2, 3, 4, 6] if value == 0 else [0, 1, 2, 5, 6],
             "fail": [0, 1, 7, 8], "absent": [0, 1, 7, 9, 10], "pending": [0, 1, 7, 9]}
    path = paths[mode]
    final = {"answer": (6, "succeeded"), "fail": (8, "failed"),
             "absent": (10, "succeeded"), "pending": (11, "open")}[mode]
    history = sorted((r for r in tables["history"] if r["subject"] == run["Subject"]),
                     key=lambda row: row["sequence"])
    assert len(path) == len(history)
    data = {"request_" + k: v for k, v in inv["Inputs"].items()}
    for i, (pc, row) in enumerate(zip(path, history)):
        assert row["sequence"] == i + 1 and row["generation"] == run["Generation"]
        assert row["selected_observation"] == (req["ID"] if i == 0 else None)
        post = json.loads(row["successor"])
        expected = (path[i + 1], "open") if i + 1 < len(path) else final
        assert (post["PC"], post["Status"]) == expected and post["Data"] == data and not post.get("Cursors")
        intents = json.loads(row["intents"])
        if pc in (3, 5):
            reply = singleton(intents)
            assert reply["Kind"] == "message" and reply["Obligation"] == req["Obligation"]
            assert reply["Target"] == req["Source"] and reply["Generation"] == req["SourceGeneration"]
            assert reply["Payload"]["id"] == data["request_id"] and reply["Payload"]["value"]["Number"] == 1 - value
        else:
            assert not intents
    assert record["Report"]["Quiescent"] and record["Child"]["Projection"] == post
    events = tables["composition_publications"]
    if mode == "pending":
        assert not events and not tables["composition_replies"] and not tables["composition_freezes"]
    else:
        event = json.loads(singleton(events)["descriptor"])
        status, reason = {"answer": ("succeeded", "member_succeeded"), "fail": ("failed", "member_failed"),
                          "absent": ("failed", "missing_reply")}[mode]
        assert event["OperationID"] == inv["OperationID"] and (event["Status"], event["Reason"]) == (status, reason)
        assert event["ResponsePresent"] == (mode == "answer")
        if mode == "answer":
            assert event["Response"] == reply["Payload"]
            disposition = json.loads(singleton(tables["composition_replies"])["descriptor"])
            assert disposition["Intent"] == reply and disposition["Accepted"] and disposition["Reason"] == "recorded_reply"
        else:
            assert not event["Response"] and not tables["composition_replies"]
    return dict(path=path, final=final, sha256=hashlib.sha256(raw.encode()).hexdigest())


def main():
    if not __debug__:
        raise RuntimeError("enable assertions for this audit")
    samples = json.loads(gzip.decompress(pathlib.Path(sys.argv[1]).read_bytes()))
    result = {}
    for ident, value, mode in itertools.product((1, 2), (0, 1), ("answer", "fail", "absent", "pending")):
        name = f"id{ident}-value{value}-{mode}.json"
        result[name] = check(samples[name], ident, value, mode)
    assert samples.keys() == result.keys()
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
