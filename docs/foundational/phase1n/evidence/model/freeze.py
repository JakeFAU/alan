"""Freeze only approved inputs and independently authored N model sources."""
import gzip
import hashlib
import io
import json
from pathlib import Path
import subprocess
import tarfile

root = Path(__file__).resolve().parents[5]
output = Path(__file__).parent
baseline = "3c5118ea9763a27cca432ca91aac667bd59e21e7"
head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
historical = subprocess.check_output(
    ["git", "ls-tree", "-r", "--name-only", baseline, "research/substitution"],
    cwd=root, text=True).splitlines()
historical += [
    "go.mod", "go.sum", "AGENTS.md",
    "docs/foundational/phase1i/01-substitution-contract.md",
    "docs/foundational/phase1m/01-reason-contract.md",
    "docs/foundational/phase1m/02-qualification.md",
    "docs/foundational/phase1m/03-review.md",
    "docs/foundational/phase1m/05-decisions.md",
    "examples/substitution/protocol/protocol.go",
    "examples/substitution/a/candidate.go", "examples/substitution/b/b.go",
    "examples/substitution/deployment.go", "internal/executor/invocation.go",
    "internal/executor/messages.go", "internal/executor/invocation_outcome.go",
    "paper/alan-dissertation.md", "paper/alan-dissertation.tex",
]
for name in historical:
    original = subprocess.check_output(["git", "show", baseline + ":" + name], cwd=root)
    if (root / name).read_bytes() != original:
        raise SystemExit("historical source changed: " + name)
authored = [str(p.relative_to(root)) for p in sorted((root / "research/substitution").glob("callable_join*.go"))]
authored += ["docs/foundational/phase1n/02-lifting.md",
             "docs/foundational/phase1n/evidence/model/capture.py",
             "docs/foundational/phase1n/evidence/model/freeze.py"]
inputs = ["docs/foundational/phase1n/01-contract.md",
          ".superpowers/sdd/00-plan-phase1n/task-3-brief.md",
          ".superpowers/sdd/00-plan-phase1n/constraints.md",
          ".superpowers/sdd/00-plan-phase1n/task-3-preparation.md"]
paths = sorted(set(historical + authored + inputs))
sources = {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in paths}
commands = {}
for label in ["construction-normal-freeze", "construction-race-freeze", "vet-freeze", "toolchain"]:
    p = output / (label + ".json.gz")
    record = json.loads(gzip.decompress(p.read_bytes()))
    if record["exit"] != 0:
        raise SystemExit("failed freeze command: " + label)
    for name in authored:
        if name.endswith(".go") and record["sources"].get(name) != sources[name]:
            # The version command is informational and predates final formatting.
            if label != "toolchain":
                raise SystemExit("test/source mismatch: " + name)
    commands[label] = dict(sha256=hashlib.sha256(p.read_bytes()).hexdigest(),
                           exit=record["exit"], leaves=len(record["leaves"]), argv=record["argv"])
buffer = io.BytesIO()
with tarfile.open(fileobj=buffer, mode="w") as archive:
    for name in paths:
        data = (root / name).read_bytes()
        info = tarfile.TarInfo(name)
        info.size, info.mode, info.mtime = len(data), 0o644, 0
        archive.addfile(info, io.BytesIO(data))
compressed = gzip.compress(buffer.getvalue(), mtime=0)
archive_path = output / "reference-r1.tar.gz"
manifest_path = output / "reference-r1.json"
if archive_path.exists() or manifest_path.exists():
    raise SystemExit("refusing to overwrite reference freeze")
aggregate = hashlib.sha256("".join(p + "\0" + sources[p] + "\n" for p in paths).encode()).hexdigest()
record = dict(
    version="N-reference-r1", baseline=baseline, ambient_head=head,
    ambient_head_note="informational only; new construction files/results were not read or used",
    author="/root/n_model, distinct independent reference author",
    input_boundary="approved contract, brief/constraints/preparation, shared historical semantics only",
    forbidden_inputs_unopened=["examples/callablejoin", "new construction representation/reports/preflight",
                               "new runtime output", "A/B replacement qualification outputs"],
    authored=authored, approved_inputs=inputs, historical_sources_unchanged=historical,
    sources=sources, source_aggregate_sha256=aggregate,
    archive=dict(path=archive_path.name, sha256=hashlib.sha256(compressed).hexdigest()),
    commands=commands,
    not_run=["exhaustive K1 product", "A/B products", "aligned A/B qualification",
             "exhaustive finite lifting", "artifact correspondence", "runtime samples"],
    status="construction freeze awaiting independent reference review",
)
archive_path.write_bytes(compressed)
manifest_path.write_text(json.dumps(record, sort_keys=True, indent=2) + "\n")
print(json.dumps(dict(manifest=manifest_path.name,
                     manifest_sha256=hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
                     source_aggregate_sha256=aggregate, archive_sha256=record["archive"]["sha256"],
                     source_files=len(paths), commands=commands), sort_keys=True))
