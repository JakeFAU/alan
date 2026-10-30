#!/usr/bin/env python3
"""C_reason construction capture, adapted from Phase1M C_se preflight.
Not the independent model, its oracle, or the Phase1M audited full-input adapter.
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
GO = "/opt/homebrew/bin/go"
BASE = "1f71b56"
PINNED = dict(GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
              GOFLAGS="-mod=readonly", CGO_ENABLED="1",
              GOCACHE="/private/tmp/alan-phase1l-go-build")
ENV = dict(os.environ, **PINNED)
ENV = {k: v for k, v in ENV.items() if k != "OPENAI_API_KEY" and not k.startswith("ALAN_")}
COMMANDS = []
ROOT_RUN = dict(Subject="root", Generation=1)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def run(argv):
    print("RUN", " ".join(argv), flush=True)
    start = time.monotonic()
    result = subprocess.run(argv, cwd=ROOT, env=ENV, capture_output=True,
                            text=True, timeout=120)
    record = dict(argv=argv, cwd=str(ROOT), environment=PINNED,
                  removed_environment=["OPENAI_API_KEY", "ALAN_*"],
                  exit=result.returncode, seconds=round(time.monotonic()-start, 3),
                  stdout=result.stdout, stderr=result.stderr)
    COMMANDS.append(record)
    print("EXIT", result.returncode, record["seconds"], flush=True)
    if result.returncode:
        raise RuntimeError(record)
    return result.stdout


def one(rows, field, value, optional=False):
    matches = [r for r in rows if r[field] == value]
    assert len(matches) == 1 or (optional and not matches), (field, value, matches)
    return matches[0] if matches else None


def decode(row, column):
    return json.loads(row[column])


def selected(snapshot):
    """Read real handles; retain every source row, including pending prefixes."""
    tables = snapshot["Tables"]
    parent = one(tables["runs"], "subject", "root")
    projection = decode(parent, "projection")
    joins = {}
    for name in ("candidate", "s"):
        operation = projection["Data"][name+"_operation"]["Text"]
        intent = one(tables["intents"], "id", operation)
        invocation = one(tables["composition_invocations"], "operation_id", operation)
        admission = decode(invocation, "admission")
        assert admission["Accepted"]
        descriptor = decode(invocation, "descriptor")
        assert descriptor["OperationID"] == operation and descriptor["Caller"] == ROOT_RUN
        assert decode(intent, "descriptor")["Run"] == ROOT_RUN
        assert descriptor["Inputs"] == decode(intent, "descriptor")["Payload"]
        binding = one(tables["composition_bindings"], "id", admission["BindingID"])
        assert binding["invocation_id"] == invocation["id"]
        request = one(tables["composition_requests"], "invocation_id", invocation["id"])
        request_observation = one(tables["observations"], "id", request["observation_id"])
        publication = one(tables["composition_publications"], "operation_id", operation, True)
        delivery = one(tables["composition_outcome_deliveries"], "operation_id", operation, True)
        observation = None
        history = None
        if publication:
            assert delivery is not None
            outcome = decode(publication, "descriptor")
            assert outcome["OperationID"] == operation and outcome["TemplateID"] == descriptor["TemplateID"]
            assert outcome["Caller"] == ROOT_RUN and outcome["InvocationID"] == invocation["id"]
            observation = one(tables["observations"], "id", delivery["observation_id"])
            observed = decode(observation, "descriptor")
            assert observed["Operation"] == operation and observed["Run"] == ROOT_RUN
            assert observed["Kind"] == "invocation" and observed["Provenance"] == "runtime-invocation"
            payload = observed["Payload"]
            assert payload["status"]["Text"] == outcome["Status"]
            assert payload["reason"]["Text"] == outcome["Reason"]
            assert payload["response_present"]["Flag"] == outcome["ResponsePresent"]
            if observation["consumed_sequence"] is not None:
                history = one([h for h in tables["history"] if h["subject"] == "root"],
                              "selected_observation", observation["id"])
                assert history["sequence"] == observation["consumed_sequence"]
                assert decode(history, "successor")["Data"][name+"_status"]["Text"] == outcome["Status"]
                for field, outfield in (("status", "Status"), ("reason", "Reason"), ("present", "ResponsePresent")):
                    value = projection["Data"][name+"_"+field]
                    assert (value["Flag"] if field == "present" else value["Text"]) == outcome[outfield]
        else:
            assert delivery is None
        joins[name] = dict(handle_from_parent=operation, intent=intent, invocation=invocation,
                           binding=binding, request=request, request_observation=request_observation,
                           publication=publication, delivery=delivery,
                           observation=observation, selected_history=history)
    assert joins["candidate"]["handle_from_parent"] != joins["s"]["handle_from_parent"]
    assert joins["candidate"]["intent"]["sequence"] < joins["s"]["intent"]["sequence"]
    return dict(parent=parent, joins=joins, roots=snapshot["Roots"])


def history_evidence(snapshot, parent_program):
    """Locate the actual executed setter/end by each predecessor's retained PC."""
    rows = sorted((r for r in snapshot["Tables"]["history"] if r["subject"] == "root"),
                  key=lambda r: r["sequence"])
    previous = dict(PC=0, Status="open")
    evidence = dict(classification_writes=[], terminals=[], branches=[])
    code = parent_program["Code"]
    for row in rows:
        successor = decode(row, "successor")
        pc = previous["PC"]
        instruction = code[pc]
        item = dict(pc=pc, instruction=instruction, previous=previous, committed_history=row)
        if instruction["Op"] == "set" and instruction["Dst"] == "classification":
            assert successor["Data"]["classification"] == instruction["A"]["Literal"]
            evidence["classification_writes"].append(item)
        if instruction["Op"] == "end":
            assert successor["Status"] == instruction["Outcome"]
            evidence["terminals"].append(item)
        if instruction["Op"] == "branch":
            assert successor["PC"] in (instruction["Target"], instruction["Otherwise"])
            evidence["branches"].append(item)
        previous = successor
    return evidence


def acyclic_path_bound(program):
    code = program["Code"]
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


def differences(a, b, path=""):
    if isinstance(a, dict) and isinstance(b, dict) and a.keys() == b.keys():
        return sum((differences(a[k], b[k], path+"/"+k) for k in a), [])
    if isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
        return sum((differences(x, y, path+"/"+str(i)) for i, (x, y) in enumerate(zip(a, b))), [])
    return [] if a == b else [dict(path=path, a=a, b=b)]


assert (DEST/"expectations.md").is_file()
sources = []
for directory in ("examples/consumingpolicy", "examples/substitution", "examples/join",
                  "author", "runtime", "internal/executor", "internal/boundary",
                  "compose", "schema", "effects"):
    sources += sorted((ROOT/directory).rglob("*.go"))
originals = [
    "examples/consumingpolicy/deployment.go", "examples/consumingpolicy/deployment_test.go",
    "examples/consumingpolicy/consumer/main_test.go",
    "examples/consumingpolicy/consumer/main_integration_test.go",
    "examples/substitution/a/candidate.go", "examples/substitution/b/b.go",
    "examples/substitution/deployment.go", "examples/substitution/probes.go",
    "examples/substitution/protocol/protocol.go", "examples/join/deployment.go",
    "author/invocations.go", "author/compile.go", "compose/collection.go", "go.mod", "go.sum",
    "examples/invocationjoin/deployment.go", "examples/invocationjoin/consumer/main.go",
    "docs/foundational/phase1m/evidence/preflight/capture.py",
    "docs/foundational/phase1m/evidence/preflight/expectations.md",
    "docs/foundational/phase1m/evidence/preflight/records.json.gz",
    "docs/foundational/phase1m/evidence/preflight/evidence-index.json",
]
sources += [ROOT/p for p in originals]
sources += [ROOT/"examples/consumingpolicy/consumer/go.mod", ROOT/"examples/consumingpolicy/consumer/go.sum",
            ROOT/"examples/consumingpolicy/README.md", Path(__file__).resolve(), DEST/"expectations.md",
            DEST/"construction-red.json", DEST/"construction-attempt1.json"]
source_hashes = {str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in sorted(set(sources))}
source_snapshots = {str(p.relative_to(ROOT)): base64.b64encode(p.read_bytes()).decode()
                    for p in sorted(set(sources)) if p.suffix != ".gz"}
bundle = dict(started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
              purpose="C_reason public construction preflight; not independent model or full audited domain",
              base_commit=subprocess.check_output(["git", "rev-parse", BASE], cwd=ROOT, text=True).strip(),
              source_hashes=source_hashes, source_snapshots_base64=source_snapshots,
              commands=COMMANDS, samples={}, status="in_progress")

with tempfile.TemporaryDirectory(prefix="alan-phase1m-reason-preflight-", dir="/private/tmp") as tmp:
    binary = str(Path(tmp)/"consumer")
    try:
        run([GO, "version"])
        run([GO, "env", "GOOS", "GOARCH", "CGO_ENABLED", "GOVERSION", "GOTOOLCHAIN", "GOPROXY", "GOSUMDB", "GOFLAGS", "GOCACHE"])
        run([GO, "test", "-list", ".", "./examples/consumingpolicy"])
        run([GO, "test", "-json", "-count=1", "-timeout=2m", "./examples/consumingpolicy"])
        run([GO, "test", "-json", "-count=1", "-race", "-timeout=3m", "./examples/consumingpolicy"])
        prefix = [GO, "-C", "examples/consumingpolicy/consumer"]
        run(prefix+["test", "-tags", "integration", "-list", ".", "./..."])
        run(prefix+["test", "-json", "-count=1", "-timeout=2m", "-tags", "integration", "./..."])
        run(prefix+["test", "-json", "-count=1", "-race", "-timeout=3m", "-tags", "integration", "./..."])
        run([GO, "vet", "./examples/consumingpolicy"])
        run(prefix+["vet", "-tags", "integration", "./..."])
        run([GO, "mod", "verify"])
        run(prefix+["mod", "verify"])
        go_files = sorted(str(p.relative_to(ROOT)) for p in (ROOT/"examples/consumingpolicy").rglob("*.go"))
        assert run(["/opt/homebrew/bin/gofmt", "-l"]+go_files) == ""
        run(["git", "diff", "--check", "--", "examples/consumingpolicy"])
        run(prefix+["build", "-o", binary, "."])
        imports = run(prefix+["list", "-f", "{{join .Imports \"\\n\"}}", "."])
        assert "/internal/" not in imports
        bundle["binary_sha256"] = sha(Path(binary).read_bytes())
        cases = [(c+"-"+mode, c, mode, "answer", tag) for c in ("a", "b")
                 for mode, tag in (("answer", "answer"), ("fail", "candidate_failed"),
                                   ("absent", "candidate_missing_reply"), ("pending", None))]
        cases += [("wrong-id", "wrong-id", "answer", "answer", "invalid_outcome"),
                  ("wrong-complement", "weak-a", "answer", "answer", "invalid_outcome"),
                  ("deliberate-failure-absent", "deliberate-failure", "absent", "answer", "candidate_failed"),
                  ("s-fail", "a", "answer", "fail", "invalid_outcome"),
                  ("s-absent", "a", "answer", "absent", "invalid_outcome")]
        for name, candidate, mode, s_mode, want_tag in cases:
            sample = dict(candidate=candidate, mode=mode, s_mode=s_mode,
                          predeclared_expected_classification=want_tag)
            bundle["samples"][name] = sample
            path = str(Path(tmp)/(name+".db"))
            args = ["-policy", "reason", "-candidate", candidate, "-id", "1",
                    "-value", "0", "-mode", mode, "-s-mode", s_mode]
            sample["artifacts"] = json.loads(run([binary, "artifacts"]+args))
            sample["init"] = json.loads(run([binary, "init", "-db", path]+args))
            sample["run"] = json.loads(run([binary, "run", "-db", path]))
            sample["inspection"] = json.loads(run([binary, "inspect", "-db", path]))
            sample["replay"] = json.loads(run([binary, "replay", "-db", path]))
            sample["selected_raw_rows"] = selected(sample["inspection"])
            projection = decode(sample["selected_raw_rows"]["parent"], "projection")
            data = projection["Data"]
            joins = sample["selected_raw_rows"]["joins"]
            evidence = history_evidence(sample["inspection"], sample["artifacts"]["Parent"])
            sample["committed_policy_history"] = evidence
            s_seq = joins["s"]["observation"]["consumed_sequence"]
            assert joins["s"]["intent"]["sequence"] < s_seq
            s_input = decode(joins["s"]["intent"], "descriptor")["Payload"]
            assert s_input["id"]["Number"] == 1 and s_input["value"]["Number"] == 0 and "mode" not in s_input
            expected_charges = 3 if candidate == "b" else 2
            root = sample["inspection"]["Roots"][0]
            assert root["Charges"]["Invocations"] == root["Charges"]["CreatedRuns"] == expected_charges
            assert root["Charges"]["Effects"] == 0
            if want_tag is None:
                assert projection["Status"] == "open" and projection["PC"] == 3
                assert "classification" not in data and root["Outcome"] is None
                assert joins["candidate"]["publication"] is None
                assert not evidence["classification_writes"] and not evidence["terminals"]
            else:
                e_seq = joins["candidate"]["observation"]["consumed_sequence"]
                assert s_seq < e_seq
                assert len(evidence["classification_writes"]) == len(evidence["terminals"]) == 1
                write = evidence["classification_writes"][0]
                terminal = evidence["terminals"][0]
                assert e_seq < write["committed_history"]["sequence"] < terminal["committed_history"]["sequence"]
                assert data["classification"]["Text"] == want_tag
                assert write["instruction"]["A"]["Literal"]["Text"] == want_tag
                status = "succeeded" if want_tag == "answer" else "failed"
                assert projection["Status"] == root["LocalStatus"] == root["Outcome"]["Status"] == status
                assert root["Outcome"]["Reason"] == ("member_succeeded" if status == "succeeded" else "member_failed")
                assert terminal["instruction"]["Outcome"] == status
                expected_reason = ("member_failed" if candidate == "deliberate-failure" or mode == "fail"
                                   else "missing_reply" if mode == "absent" else "member_succeeded")
                assert data["candidate_reason"]["Text"] == expected_reason
                if want_tag == "answer":
                    assert data["candidate_id"]["Number"] == 1 and data["candidate_value"]["Number"] == 1
            for side in ("candidate", "s"):
                if side+"_present" in data and not data[side+"_present"]["Flag"]:
                    assert side+"_id" not in data and side+"_value" not in data
            program_raw = [base64.b64decode(p["Bytes"]) for p in sample["artifacts"]["Provision"]["Programs"]]
            programs = [json.loads(raw) for raw in program_raw]
            tables = sample["inspection"]["Tables"]
            metrics = dict(code_instructions=[len(p["Code"]) for p in programs],
                           acyclic_instruction_path_upper=sum(acyclic_path_bound(p) for p in programs),
                           finite_dispatch_bound=3*expected_charges,
                           program_bytes=max(map(len, program_raw)),
                           provision_json_bytes=len(json.dumps(sample["artifacts"]["Provision"], separators=(",", ":")).encode()),
                           projection_bytes=max(len(r["successor"].encode()) for r in tables["history"]),
                           observation_bytes=max(len(r["descriptor"].encode()) for r in tables["observations"]),
                           output_array_bytes=max(len(r["intents"].encode()) for r in tables["history"]),
                           steps=sample["run"]["Steps"], dispatches=sample["run"]["Dispatches"])
            assert metrics["steps"] <= metrics["acyclic_instruction_path_upper"] < 200
            assert metrics["dispatches"] <= metrics["finite_dispatch_bound"] < 200
            assert all(metrics[k] < 1048576 for k in ("program_bytes", "projection_bytes", "observation_bytes", "output_array_bytes"))
            sample["operational_metrics"] = metrics
            parent = sample["artifacts"]["Parent"]
            sample["compiled_inventory"] = [
                dict(pc=pc, position=parent["Author"]["Positions"][pc], instruction=in_)
                for pc, in_ in enumerate(parent["Code"])]
        bundle["status"] = "passed"
    except BaseException as exc:
        bundle["status"] = "failed"
        bundle["failure"] = repr(exc)
        raise
    finally:
        after = {str(p.relative_to(ROOT)): sha(p.read_bytes()) for p in sorted(set(sources))}
        bundle["source_stable_through_capture"] = after == source_hashes
        bundle["original_source_unchanged_from_base"] = {
            p: sha(subprocess.check_output(["git", "show", BASE+":"+p], cwd=ROOT)) == source_hashes[p]
            for p in originals}
        bundle["bound_parent_differences"] = differences(
            bundle["samples"].get("a-answer", {}).get("artifacts", {}).get("Parent"),
            bundle["samples"].get("b-answer", {}).get("artifacts", {}).get("Parent"))
        archive = gzip.compress(json.dumps(bundle, separators=(",", ":")).encode()+b"\n", mtime=0)
        (DEST/"records.json.gz").write_bytes(archive)
        index = {k: v for k, v in bundle.items() if k not in ("commands", "samples", "source_snapshots_base64")}
        index["source_manifest_sha256"] = sha(json.dumps(source_hashes, sort_keys=True, separators=(",", ":")).encode())
        index["records"] = dict(file="records.json.gz", sha256=sha(archive))
        index["commands"] = []
        for command in COMMANDS:
            c = {k: v for k, v in command.items() if k not in ("stdout", "stderr")}
            tests = []
            for line in command["stdout"].splitlines():
                try:
                    event = json.loads(line)
                    if isinstance(event, dict) and event.get("Action") in ("pass", "fail", "skip") and "Test" in event:
                        tests.append(dict(test=event["Test"], action=event["Action"]))
                except ValueError:
                    pass
            c["tests"] = tests
            c["test_leaves"] = [x for x in tests if not any(y["test"].startswith(x["test"]+"/") for y in tests)]
            index["commands"].append(c)
        index["samples"] = {}
        for name, sample in bundle["samples"].items():
            if "operational_metrics" not in sample:
                continue
            rows = sample["selected_raw_rows"]
            policy = sample["committed_policy_history"]
            index["samples"][name] = dict(identities=sample["init"], root=sample["inspection"]["Roots"][0],
                operational_metrics=sample["operational_metrics"],
                classification=decode(rows["parent"], "projection")["Data"].get("classification"),
                original_joins={k: dict(handle=v["handle_from_parent"],
                    commit_sequence=v["intent"]["sequence"],
                    consumed_sequence=v["observation"]["consumed_sequence"] if v["observation"] else None,
                    selected_observation=v["selected_history"]["selected_observation"] if v["selected_history"] else None,
                    outcome=decode(v["publication"], "descriptor") if v["publication"] else None)
                    for k, v in rows["joins"].items()},
                classification_writes=[dict(pc=h["pc"], sequence=h["committed_history"]["sequence"], instruction=h["instruction"])
                    for h in policy["classification_writes"]],
                terminal_decisions=[dict(pc=h["pc"], sequence=h["committed_history"]["sequence"], instruction=h["instruction"])
                    for h in policy["terminals"]])
        (DEST/"evidence-index.json").write_text(json.dumps(index, indent=2)+"\n")
        print("Evidence saved:", DEST/"evidence-index.json", flush=True)
        assert bundle["source_stable_through_capture"]
        assert all(bundle["original_source_unchanged_from_base"].values())
