"""Audit the immutable I/N/R input bytes and original local census. Run at repo root."""
import base64
import gzip
import hashlib
import itertools
import json
import subprocess
from pathlib import Path

BASE = "61f44fc2c62772065ff3d483f52c0235670f586c"
R_HEAD = "0cb8835574e5cf0e710c3e56442254a6555e46f9"
FOUND = Path("docs/foundational")
ARCHIVE = FOUND / "phase1i/evidence/authorship/b"
N = FOUND / "phase1n/evidence/model/qualification"
R = FOUND / "phase1r/evidence/runs/relocation"
source_hashes = {}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(["git", *args])


def pinned(path):
    current = Path(path).read_bytes()
    assert current == git("show", f"{BASE}:{path}"), path
    source_hashes[str(path)] = sha(current)
    return current


def packed(path):
    return json.loads(gzip.decompress(pinned(path)))


def compact(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode()


def descriptor_id(descriptor):
    plain = dict(descriptor, ID="")
    return sha(compact(["alan/template/v1", plain]))


def audit():
    assert git("rev-parse", BASE+"^2").decode().strip() == R_HEAD
    assert git("rev-parse", BASE+"^{tree}") == git("rev-parse", R_HEAD+"^{tree}")
    for path in """
docs/foundational/phase1i/01-substitution-contract.md
docs/foundational/phase1n/01-contract.md
docs/foundational/phase1n/02-lifting.md
docs/foundational/phase1n/03-qualification.md
docs/foundational/phase1o/01-weak-contract.md
docs/foundational/phase1o/03-finite-contract.md
docs/foundational/phase1p/02-preservation.md
docs/foundational/phase1q/03-relay-rule.md
docs/foundational/phase1r/02-transport.md
docs/foundational/phase1r/03-qualification.md
research/substitution/model.go
research/substitution/candidates.go
research/substitution/relation.go
research/substitution/callable_join_local.go
research/substitution/weak_refinement.go
research/nominal/certificate.go
research/nominal/operands.go
internal/executor/program.go
internal/executor/runner.go
internal/executor/invocation.go
internal/executor/invocation_outcome.go
internal/executor/messages.go
examples/substitution/b/b.go
""".split():
        pinned(path)
    raw = pinned(ARCHIVE/"template.json")
    bundle = json.loads(raw)
    b = bundle["Descriptor"]
    h, = bundle["Dependencies"]
    assert [descriptor_id(v) for v in (b,h)] == [b["ID"],h["ID"]]
    assert b["Templates"] == [h["ID"]] and not h["Templates"]
    descs = [b,h]
    programs = {}
    for desc in descs:
        member = desc["Members"]["entry"]
        assert list(desc["Members"]) == ["entry"] and desc["EntryMember"] == "entry"
        assert desc["EntryPort"] == "public" and desc["CallerPeer"] == "caller"
        assert member["Initial"] == {"PC":0, "Data":{}, "Status":"open"}
        pid = member["ProgramID"]
        program = pinned(ARCHIVE/f"program-{pid}.json")
        assert sha(program) == pid
        archive_program, = [a for a in bundle["Programs"] if a["ID"] == pid]
        assert base64.b64decode(archive_program["Bytes"]) == program
        programs[pid] = program
    cert = sha(b"alan/research/nomcert/v1\0template\0" + raw)
    cases = set()
    for path in sorted(R.glob("*/relocation.json.gz")):
        rec = packed(path)
        tables = rec["inspection"]["Tables"]
        assert (rec["certificate"]["Digest"],rec["certificate"]["ID"]) == (cert,b["ID"])
        for desc in descs:
            pid = desc["Members"]["entry"]["ProgramID"]
            row, = [t for t in tables["composition_templates"] if t["id"] == desc["ID"]]
            assert json.loads(row["descriptor"]) == desc
            row, = [p for p in tables["programs"] if p["id"] == pid]
            assert row["bytes"].encode() == programs[pid]
            run, = [r for r in tables["runs"] if r["program_id"] == pid]
            assert json.loads(run["initial"]) == desc["Members"]["entry"]["Initial"]
        cases.add((rec["k"],rec["mode"]))
    modes = ("answer","fail","absent","pending")
    assert cases == set(itertools.product(range(2),modes))
    records = packed(N/"normal-local/callable-join-aligned-local.json.gz")
    rows = [r for r in records if r["Variant"]=="helper"]
    assert len(rows)==16 and {(q["ID"],q["Value"],q["Mode"]) for r in rows for q in [r["Config"]["Request"]]} == set(itertools.product((1,2),(0,1),modes))
    assert all(r["Relation"]["Failures"] == r["Progress"]["Cutoffs"] == r["Progress"]["RankFailures"] == 0 for r in rows)
    census = (sum(r["Relation"]["Pairs"] for r in rows),sum(r["Progress"]["States"] for r in rows),sum(r["Progress"]["Transitions"] for r in rows))
    assert census == (2320,824,1244)
    correspondence = packed(N/"normal-correspondence-r2/correspondence.json.gz")
    assert set(programs) <= {p["program_id"] for p in correspondence["bindings"]["b"]["programs"]}
    assert all(d["ID"] in correspondence["bindings"]["b"]["template_ids"] for d in descs)
    for pid in programs:
        matches = [r for r in correspondence["instruction_records"] if r["program_id"]==pid]
        assert matches
        for match in matches:
            assert [r["instruction"] for r in sorted(match["instructions"],key=lambda r:r["pc"])] == json.loads(programs[pid])["Code"]
    return {"base":BASE,"reviewed_R":R_HEAD,"base_tree":git("rev-parse",BASE+"^{tree}").decode().strip(),
            "template_descriptors":descs,"program_ids":sorted(programs),"nominal_digest":cert,
            "R_placements":sorted(cases),"N_historical_pairs_open_states_edges":census,
            "N_mapping_scope":correspondence["scope"],"source_sha256":source_hashes}


if __name__ == "__main__":
    print(json.dumps(audit(), sort_keys=True, indent=2))
