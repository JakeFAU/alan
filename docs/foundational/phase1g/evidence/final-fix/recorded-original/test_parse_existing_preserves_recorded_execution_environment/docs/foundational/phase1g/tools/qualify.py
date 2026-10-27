#!/usr/bin/env python3
"""Run the frozen-source qualification and retain raw Go JSON ordering."""
from pathlib import Path
import collections
import datetime
import gzip
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / "docs/foundational/phase1g"
MODE = sys.argv[1] if len(sys.argv) > 1 else "normal"
assert MODE in ("normal", "race", "default", "consumer", "consumer-race")
MANIFEST = json.loads((DOC / "source-manifest.json").read_text())
OUT = Path(os.environ.get("ALAN_QUALIFICATION_OUT", str(DOC / "evidence/task4/final")))
OUT.mkdir(exist_ok=True, parents=True)

def verify():
    for item in MANIFEST["files"]:
        raw = (ROOT / item["path"]).read_bytes()
        assert len(raw) == item["bytes"] and hashlib.sha256(raw).hexdigest() == item["sha256"], item["path"]

verify()
cmd = ["go", "test", "-json", "-count=1", "-timeout", "20m"] + (["-race"] if MODE == "race" else []) + ([] if MODE == "default" else ["-tags", "integration"]) + ["./..."]
if MODE.startswith("consumer"):
    cmd = ["go", "-C", "examples/worktree/consumer", "test", "-json", "-count=1", "-timeout=240s"] + (["-race"] if MODE == "consumer-race" else []) + ["./..."]
env = dict(os.environ)
env.pop("OPENAI_API_KEY", None)
env.update(GOTOOLCHAIN="local", GOFLAGS="-mod=readonly")
env.setdefault("GOCACHE", "/tmp/alan-phase1f-go-build")
env.setdefault("GOPROXY", "off")
env.setdefault("GOSUMDB", "off")
raw_path = OUT / ("final-" + MODE + ".go-test.jsonl.gz")
meta_path = OUT / ("final-" + MODE + ".command.json")
if "--parse-existing" in sys.argv:
    previous = json.loads(meta_path.read_text())
    assert previous["SourceID"] == MANIFEST["source_sha256"]
    raw = gzip.decompress(raw_path.read_bytes())
    assert hashlib.sha256(raw).hexdigest() == previous["RawSHA256"]
    started, elapsed = previous["StartedUTC"], previous["ElapsedSeconds"]
    returncode, errors = previous["ExitCode"], previous["Stderr"]
    cmd, env = previous["Command"], previous["Environment"]
else:
    assert not meta_path.exists() and not raw_path.exists(), "refuse to overwrite qualification evidence"
    start = time.monotonic()
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
        proc = subprocess.run(cmd, cwd=ROOT, env=env, stdout=stdout, stderr=stderr)
        stdout.seek(0); raw = stdout.read()
        stderr.seek(0); errors = stderr.read().decode()
    returncode = proc.returncode
    elapsed = round(time.monotonic() - start, 3)
    verify()
    raw_path.write_bytes(gzip.compress(raw, mtime=0))
pattern = re.compile(r"^\s*(?:[^\s:]+:\d+:\s*)?(QUALIFICATION|COMPOSITION|SUPERVISION|CONFORMANCE|DURABLE|ASYNC|EVIDENCE|INVESTIGATION|WORKTREE) (.*)$")
buffers, records, packages, failures = {}, [], [], []
passes = 0
for line in raw.decode().splitlines():
    e = json.loads(line)
    package, test = e.get("Package", ""), e.get("Test", "")
    if e.get("Action") == "output":
        key = (package, test)
        value = buffers.get(key, "") + e.get("Output", "")
        while "\n" in value:
            one, _, value = value.partition("\n")
            match = pattern.match(one)
            # Executor unit tests also emit plain JSON objects with type=ASYNC.
            plain = re.sub(r"^\s*[^\s:]+:\d+:\s*", "", one).strip()
            if not match and plain.startswith("{"):
                value_json = json.loads(plain)
                if isinstance(value_json, dict) and value_json.get("type") == "ASYNC":
                    records.append({"SourceID": MANIFEST["source_sha256"], "Mode": MODE,
                                    "Package": package, "Test": test, "RecordIndex": len(records) + 1,
                                    "Marker": "ASYNC", "Evidence": value_json})
            if match:
                records.append({"SourceID": MANIFEST["source_sha256"], "Mode": MODE,
                                "Package": package, "Test": test, "RecordIndex": len(records) + 1,
                                "Marker": match[1], "Evidence": (json.loads(match[2]) if match[2].startswith(("{", "[")) else {"Kind": "text_trace", "Text": match[2]})})
        buffers[key] = value
    elif e.get("Action") == "pass" and test:
        passes += 1
    elif e.get("Action") == "fail":
        failures.append(e)
    if e.get("Action") in ("pass", "fail", "skip") and not test:
        packages.append({"Package": package, "Result": e["Action"], "ElapsedSeconds": e.get("Elapsed")})
assert not any(v.strip() for v in buffers.values())
meta = {"Kind": "verification_command", "SourceID": MANIFEST["source_sha256"], "Mode": MODE,
        "Command": cmd, "Directory": str(ROOT), "StartedUTC": started, "ElapsedSeconds": elapsed,
        "Environment": {k: env[k] for k in ("GOTOOLCHAIN", "GOFLAGS", "GOCACHE", "GOPROXY", "GOSUMDB")},
        "ExitCode": returncode, "Stderr": errors, "Packages": packages,
        "PassingEvents": passes, "Failures": failures,
        "Markers": dict(collections.Counter(r["Marker"] for r in records)),
        "RawLog": os.path.relpath(raw_path, ROOT), "RawSHA256": hashlib.sha256(raw).hexdigest(),
        "GzipSHA256": hashlib.sha256(raw_path.read_bytes()).hexdigest(), "SourceUnchanged": True, "Platform": MANIFEST["platform"], "GoVersion": MANIFEST["go_version"], "CredentialAbsent": "OPENAI_API_KEY" not in env}
(OUT / ("final-" + MODE + ".command.json")).write_text(json.dumps(meta, indent=2) + "\n")
(OUT / ("final-" + MODE + ".records.jsonl.gz")).write_bytes(gzip.compress(
    "".join(json.dumps(r, separators=(",", ":")) + "\n" for r in records).encode(), mtime=0))
print(json.dumps(meta, indent=2), flush=True)
assert returncode == 0 and not failures
if not MODE.startswith("consumer"):
    assert meta["Markers"]["QUALIFICATION"] == 15
    assert meta["Markers"]["COMPOSITION"] == 20
    assert meta["Markers"]["SUPERVISION"] == 32
    assert meta["Markers"]["CONFORMANCE"] == 6
    assert meta["Markers"]["DURABLE"] >= 87
    assert meta["Markers"].get("ASYNC", 0) > 0
    if MODE != "default":
        assert meta["Markers"].get("WORKTREE", 0) > 0
        events=[json.loads(line) for line in raw.decode().splitlines()]
        required={"TestProcessRecoveryCuts", "TestProcessPartialWithOutstandingTransport", "TestOperatorClosureAtCompositionCuts", "TestLegacyFactoryRetainsDevelopmentArtifactIdentity"}
        passed={e.get("Test") for e in events if e.get("Action")=="pass"}
        assert required <= passed, required-passed
else:
    events=[json.loads(line) for line in raw.decode().splitlines()]
    assert any(e.get("Action")=="pass" and e.get("Test")=="TestFaultyOrdinaryCollectorReachesIndependentMappingMonitor" for e in events)
