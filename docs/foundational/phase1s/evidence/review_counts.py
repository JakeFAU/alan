"""Independent fixed-B state/edge census. Run at the repo root with python3."""

import base64
import gzip
import json
import re
from collections import namedtuple
from pathlib import Path

archive = Path("docs/foundational/phase1i/evidence/authorship/b/template.json")
bundle = json.loads(archive.read_bytes())
streams = {
    a["ID"]: json.loads(base64.b64decode(a["Bytes"]))["Code"]
    for a in bundle["Programs"]
}
programs = {
    "e": streams[bundle["Descriptor"]["Members"]["entry"]["ProgramID"]],
    "h": streams[bundle["Dependencies"][0]["Members"]["entry"]["ProgramID"]],
}

# For each operation retain protocol phase, PC, frozen bit and reply stage.
State = namedtuple("State", "closed q eb hb ep hp ef hf er hr")
initial = State(False, False, 0, 0, -1, -1, False, False, 0, 0)
untouched, committed, admitted, refused, published, delivered, selected = range(7)
no_reply, sent, accepted_reply, refused_reply = range(4)


def successors(s, mode, value, allow_close):
    if not s.closed:
        if allow_close:
            yield s._replace(closed=True)
        if not s.q:
            yield s._replace(q=True)
        if s.eb == untouched:
            yield s._replace(eb=committed)
        if s.eb == delivered:
            yield s._replace(eb=selected)

    for actor, code in programs.items():
        def update(**fields):
            return s._replace(**{actor + key: val for key, val in fields.items()})

        phase, pc, frozen, reply = (getattr(s, actor + key) for key in "bpfr")
        if phase == committed:
            yield update(b=refused) if s.closed else update(b=admitted, p=0)
        if reply == sent:
            yield update(r=refused_reply if s.closed else accepted_reply)
        # Reaching an e End already requires consuming the published h.
        if phase == refused or (phase == admitted and frozen and reply != sent):
            yield update(b=published)
        if phase == published:
            yield update(b=delivered)
        if pc < 0 or frozen:
            continue

        ins = code[pc]
        if ins["Op"] == "receive":
            if pc == 0:
                yield update(p=1)
            elif pc == 2 and s.hb == delivered:
                yield s._replace(ep=3, hb=selected)
        elif ins["Op"] == "emit":
            if ins["Effect"]["Kind"] == "invocation":
                assert actor == "e" and pc == 1 and s.hb == untouched
                yield s._replace(ep=2, hb=committed)
            else:
                yield update(p=pc + 1, r=sent)
        elif ins["Op"] == "branch":
            # On this fixed bundle h receipt bits are functions of its own
            # reply and End. No private operand reads q or the root's closure.
            status = "succeeded" if s.hr == accepted_reply else "failed" if s.hf else "rejected"
            register = {
                "helper_status": status, "helper_present": s.hr == accepted_reply,
                "requested_mode": mode, "input_value": value,
            }[ins["A"]["Ref"]]
            literal = ins["B"]["Literal"]
            field = {"int": "Number", "string": "Text", "bool": "Flag"}[literal["Kind"]]
            yield update(p=ins["Target"] if register == literal[field] else ins["Otherwise"])
        elif ins["Op"] == "end":
            yield update(f=True)
        elif ins["Op"] == "set":
            yield update(p=pc + 1)
        elif ins["Op"] == "jump":
            yield update(p=ins["Target"])
        else:
            raise ValueError(ins["Op"])


def census(mode, value, allow_close):
    queue, seen, maximum = [initial], {initial}, []
    total = 0
    for s in queue:
        edges = list(successors(s, mode, value, allow_close))
        total += len(edges)
        if not edges:
            maximum.append(s)
        for nxt in edges:
            if nxt not in seen:
                seen.add(nxt)
                queue.append(nxt)
    return len(seen), total, maximum


def main():
    record = Path("docs/foundational/phase1s/evidence/local-normal.json.gz")
    trace = json.loads(gzip.decompress(record.read_bytes()))["output"]
    terminal_pc = {"answer": 7, "fail": 9, "absent": 11, "pending": 12}
    for closure in (True, False):
        nodes = edges = 0
        for ident in (1, 2):
            for value in (0, 1):
                for mode, pc in terminal_pc.items():
                    n, m, maxima = census(mode, value, closure)
                    name = f"local/id{ident}/value{value}/{mode}/I3/R3/D3"
                    label = f"{name} closure={str(closure).lower()} B=true"
                    observed = re.search(label + r" states=(\d+) edges=(\d+)", trace)
                    assert observed and (n, m) == tuple(map(int, observed.groups())), label
                    if not closure:
                        end, = maxima
                        assert end.q and end.hb == selected and end.hf and end.hr == accepted_reply
                        assert end.ep == pc and end.ef == (mode != "pending")
                        assert end.er == (accepted_reply if mode == "answer" else no_reply)
                    nodes += n
                    edges += m
        print(f"closure={closure} states={nodes} edges={edges}")


if __name__ == "__main__":
    main()
