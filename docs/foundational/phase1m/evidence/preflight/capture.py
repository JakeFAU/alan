#!/usr/bin/env python3
"""Phase 1M C_se feasibility capture, adapted from Phase 1L preflight.
Not an oracle: expectations.md is recorded before execution.
"""
import base64
import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[5]
DEST = Path(__file__).resolve().parent
PINNED = dict(GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
              GOFLAGS="-mod=readonly", GOCACHE="/private/tmp/alan-phase1l-go-build")
ENV = dict(os.environ, **PINNED)
COMMANDS = []


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def run(argv):
    print("RUN", " ".join(argv), flush=True)
    start = time.monotonic()
    result = subprocess.run(argv, cwd=ROOT, env=ENV, capture_output=True, text=True, timeout=120)
    record = dict(argv=argv, cwd=str(ROOT), environment=PINNED,
                  exit=result.returncode, seconds=round(time.monotonic()-start, 3),
                  stdout=result.stdout, stderr=result.stderr)
    COMMANDS.append(record)
    print("EXIT", result.returncode, record["seconds"], flush=True)
    if result.returncode:
        raise RuntimeError(record)
    return result.stdout


def selected(snapshot):
    """Bind original handles to raw rows without reconstructing operation IDs."""
    tables = snapshot["Tables"]
    parent = next(row for row in tables["runs"] if row["subject"] == "root")
    projection = json.loads(parent["projection"])
    joins = {}
    for name, field in (("candidate", "candidate_operation"), ("s", "s_operation")):
        operation = projection["Data"][field]["Text"]
        intent = next(row for row in tables["intents"] if row["id"] == operation)
        admission = next(row for row in tables["composition_invocations"]
                         if row["operation_id"] == operation)
        publication = next(row for row in tables["composition_publications"]
                           if row["operation_id"] == operation)
        delivery = next(row for row in tables["composition_outcome_deliveries"]
                        if row["operation_id"] == operation)
        observation = next(row for row in tables["observations"]
                           if row["id"] == delivery["observation_id"])
        history = next(row for row in tables["history"]
                       if row["subject"] == "root"
                       and row["selected_observation"] == observation["id"])
        joins[name] = dict(handle_from_parent=operation, intent=intent,
                           admission=admission, publication=publication,
                           delivery=delivery, observation=observation, history=history)
    return dict(parent=parent, joins=joins, roots=snapshot["Roots"])


if not (DEST/"expectations.md").is_file():
    raise RuntimeError("specification expectations must exist before executions")

with tempfile.TemporaryDirectory(prefix="alan-phase1m-preflight-", dir="/private/tmp") as tmp:
    binary = str(Path(tmp)/"consumer")
    bundle = dict(started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  purpose="C_se construction feasibility only; not independent oracle", commands=COMMANDS, samples={})
    try:
        run(["go", "version"])
        run(["go", "env", "GOOS", "GOARCH", "CGO_ENABLED", "GOVERSION", "GOTOOLCHAIN", "GOPROXY", "GOSUMDB", "GOFLAGS", "GOCACHE"])
        run(["go", "test", "-list", ".", "./examples/consumingpolicy"])
        run(["go", "test", "-json", "-count=1", "./examples/consumingpolicy"])
        run(["go", "test", "-json", "-count=1", "-race", "./examples/consumingpolicy"])
        prefix = ["go", "-C", "examples/consumingpolicy/consumer"]
        run(prefix+["test", "-tags", "integration", "-list", ".", "./..."])
        run(prefix+["test", "-json", "-count=1", "-tags", "integration", "./..."])
        run(prefix+["test", "-json", "-count=1", "-race", "-tags", "integration", "./..."])
        run(["go", "vet", "./examples/consumingpolicy"])
        run(prefix+["vet", "-tags", "integration", "./..."])
        run(["go", "mod", "verify"])
        run(prefix+["mod", "verify"])
        run(["gofmt", "-l", "examples/consumingpolicy/deployment.go", "examples/consumingpolicy/deployment_test.go", "examples/consumingpolicy/consumer/main.go", "examples/consumingpolicy/consumer/main_test.go", "examples/consumingpolicy/consumer/main_integration_test.go"])
        run(prefix+["build", "-o", binary, "."])
        run(prefix+["list", "-f", "{{join .Imports \"\\n\"}}", "."])
        bundle["binary_sha256"] = sha(Path(binary).read_bytes())
        for candidate in ("a", "b"):
            path = str(Path(tmp)/(candidate+".db"))
            sample = {}
            sample["artifacts"] = json.loads(run([binary, "artifacts", "-policy", "s-first", "-candidate", candidate]))
            sample["init"] = json.loads(run([binary, "init", "-db", path, "-policy", "s-first", "-candidate", candidate, "-id", "1", "-value", "0", "-mode", "answer"]))
            sample["run"] = json.loads(run([binary, "run", "-db", path]))
            sample["inspection"] = json.loads(run([binary, "inspect", "-db", path]))
            sample["replay"] = json.loads(run([binary, "replay", "-db", path]))
            sample["selected_raw_rows"] = selected(sample["inspection"])
            # These assertions use the already frozen specification, not a model.
            projection = json.loads(sample["selected_raw_rows"]["parent"]["projection"])
            data = projection["Data"]
            assert projection["Status"] == "succeeded"
            for name, want in (("candidate_id", 1), ("candidate_value", 1), ("s_id", 1), ("s_value", 0)):
                assert data[name]["Number"] == want
            joins = sample["selected_raw_rows"]["joins"]
            assert joins["candidate"]["intent"]["sequence"] < joins["s"]["intent"]["sequence"] < joins["s"]["observation"]["consumed_sequence"] < joins["candidate"]["observation"]["consumed_sequence"]
            for name, join in joins.items():
                descriptor = json.loads(join["intent"]["descriptor"])
                assert descriptor["Run"] == {"Subject": "root", "Generation": 1}
                assert descriptor["Payload"]["id"]["Number"] == 1
                assert descriptor["Payload"]["value"]["Number"] == 0
                outcome = json.loads(join["observation"]["descriptor"])
                assert outcome["Payload"]["status"]["Text"] == "succeeded"
                assert outcome["Payload"]["response_present"]["Flag"]
                assert join["history"]["sequence"] == join["observation"]["consumed_sequence"]
            root = sample["inspection"]["Roots"][0]
            expected = 2 if candidate == "a" else 3
            assert root["Charges"]["Invocations"] == root["Charges"]["CreatedRuns"] == expected
            assert root["Charges"]["Effects"] == 0
            assert root["Outcome"]["Status"] == "succeeded"
            programs = [base64.b64decode(p["Bytes"]) for p in sample["artifacts"]["Provision"]["Programs"]]
            def acyclic_path_bound(program):
                code = json.loads(program)["Code"]
                longest = {}
                for pc in reversed(range(len(code))):
                    instruction = code[pc]
                    if instruction["Op"] == "end":
                        longest[pc] = 1
                        continue
                    targets = ([instruction["Target"], instruction["Otherwise"]]
                               if instruction["Op"] == "branch" else
                               [instruction["Target"]] if instruction["Op"] == "jump" else [pc+1])
                    assert all(pc < target < len(code) for target in targets)
                    longest[pc] = 1+max(longest[target] for target in targets)
                return longest[0]
            tables = sample["inspection"]["Tables"]
            sample["operational_metrics"] = dict(
                code_instructions=[len(json.loads(p)["Code"]) for p in programs],
                acyclic_instruction_path_upper=sum(acyclic_path_bound(p) for p in programs),
                finite_dispatch_bound=3*expected,
                program_bytes=max(map(len, programs)),
                projection_bytes=max(len(row["successor"].encode()) for row in tables["history"]),
                observation_bytes=max(len(row["descriptor"].encode()) for row in tables["observations"]),
                all_intent_descriptor_bytes=sum(len(row["descriptor"].encode()) for row in tables["intents"]),
                steps=sample["run"]["Steps"], dispatches=sample["run"]["Dispatches"])
            assert sample["operational_metrics"]["steps"] <= sample["operational_metrics"]["acyclic_instruction_path_upper"] < 200
            assert sample["operational_metrics"]["dispatches"] <= sample["operational_metrics"]["finite_dispatch_bound"] < 200
            assert sample["operational_metrics"]["dispatches"] < 200
            assert all(sample["operational_metrics"][k] < 1048576 for k in ("program_bytes", "projection_bytes", "observation_bytes", "all_intent_descriptor_bytes"))
            bundle["samples"][candidate] = sample
    finally:
        sources = sorted((ROOT/"examples/consumingpolicy").rglob("*.go"))
        for directory in ("author", "runtime", "internal/executor", "internal/boundary", "compose", "schema", "effects"):
            sources += sorted((ROOT/directory).rglob("*.go"))
        sources += [ROOT/"examples/consumingpolicy/consumer/go.mod",
                    ROOT/"examples/consumingpolicy/consumer/go.sum",
                    ROOT/"examples/consumingpolicy/README.md", Path(__file__).resolve()]
        originals = ["examples/substitution/a/candidate.go", "examples/substitution/b/b.go",
                     "examples/substitution/deployment.go", "examples/substitution/probes.go", "examples/substitution/protocol/protocol.go",
                     "examples/join/deployment.go", "author/invocations.go", "author/compile.go",
                     "compose/collection.go", "go.mod", "go.sum", "examples/invocationjoin/deployment.go",
                     "examples/invocationjoin/consumer/main.go",
                     "docs/foundational/phase1l/evidence/preflight/capture.py"]
        sources += [ROOT/p for p in originals]
        sources += [DEST/"expectations.md"]
        source_hashes = {str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in sorted(set(sources))}
        bundle["source_hashes"] = source_hashes
        bundle["base_commit"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        bundle["original_source_unchanged_from_base"] = {
            p: sha(subprocess.check_output(["git", "show", "HEAD:"+p], cwd=ROOT)) == source_hashes[p]
            for p in originals}
        raw = json.dumps(bundle, separators=(",", ":")).encode()+b"\n"
        archive = gzip.compress(raw, mtime=0)
        (DEST/"records.json.gz").write_bytes(archive)
        index = dict(purpose=bundle["purpose"], base_commit=bundle["base_commit"],
                     source_hashes=source_hashes,
                     source_manifest_sha256=sha(json.dumps(source_hashes, sort_keys=True,
                                                          separators=(",", ":")).encode()),
                     original_source_unchanged_from_base=bundle["original_source_unchanged_from_base"],
                     records=dict(file="records.json.gz", sha256=sha(archive)),
                     commands=[{k: v for k, v in c.items() if k not in ("stdout", "stderr")}
                               for c in COMMANDS], samples={})
        for candidate, sample in bundle["samples"].items():
            joins = sample["selected_raw_rows"]["joins"]
            index["samples"][candidate] = dict(identities=sample["init"],
                root=sample["inspection"]["Roots"][0],
                operational_metrics=sample["operational_metrics"],
                original_joins={k: dict(handle=v["handle_from_parent"],
                    commit_sequence=v["intent"]["sequence"],
                    consumed_sequence=v["observation"]["consumed_sequence"],
                    selected_observation=v["history"]["selected_observation"])
                    for k, v in joins.items()},
                program_sha256={p["ID"]: sha(base64.b64decode(p["Bytes"]))
                    for p in sample["artifacts"]["Provision"]["Programs"]})
        def differences(a, b, path=""):
            if isinstance(a, dict) and isinstance(b, dict) and a.keys() == b.keys():
                return sum((differences(a[k], b[k], path+"/"+k) for k in a), [])
            if isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
                return sum((differences(x, y, path+"/"+str(i)) for i, (x, y) in enumerate(zip(a, b))), [])
            return [] if a == b else [dict(path=path, a=a, b=b)]
        if set(bundle["samples"]) == {"a", "b"}:
            index["bound_parent_differences"] = differences(bundle["samples"]["a"]["artifacts"]["Parent"], bundle["samples"]["b"]["artifacts"]["Parent"])
        (DEST/"evidence-index.json").write_text(json.dumps(index, indent=2)+"\n")
        print("Evidence saved:", DEST/"evidence-index.json", flush=True)
