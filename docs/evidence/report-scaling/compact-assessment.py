#!/usr/bin/env python3
"""Reproduce the compact-report coverage witness without editing production files."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shlex
import subprocess


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True, help="new evidence directory")
    parser.add_argument("--gocache", type=Path, required=True)
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[3]
    root = args.root.resolve()
    root.mkdir(parents=True, exist_ok=False)
    source = repo / "internal/executor/runner.go"
    original = source.read_text()
    call = "report, runErr = s.finishRun(cleanup, report, quiescent && runErr == nil, runErr)"
    if original.count(call) != 1:
        raise RuntimeError("finishRun call changed; re-audit the witness overlay")
    # Workers, cleanup, accounting, scheduling and all transaction paths remain.
    bypass = root / "bypass-runner.go"
    bypass.write_text(original.replace(call, "report.Quiescent = quiescent && runErr == nil"))
    overlay = root / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(source): str(bypass)}}, indent=2) + "\n")
    paths = subprocess.check_output(
        ["git", "ls-files", "-z", "*.go", "go.mod", "go.sum"], cwd=repo
    ).decode().split("\0")
    paths = [p for p in paths if p and not p.startswith("docs/")]
    # Include the new test even when running before its first commit.
    paths += ["internal/executor/runner_compact_assessment_test.go", str(Path(__file__).relative_to(repo))]
    hashes = {p: digest(repo / p) for p in sorted(set(paths)) if p and (repo / p).is_file()}
    freeze = {
        "head": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo).decode().strip(),
        "source_sha256": hashes,
        "overlay_sha256": digest(overlay),
        "bypass_source_sha256": digest(bypass),
        "go_version": subprocess.check_output(["go", "version"], cwd=repo).decode().strip(),
    }
    (root / "source-freeze.json").write_text(json.dumps(freeze, indent=2) + "\n")
    env = dict(os.environ, GOCACHE=str(args.gocache.resolve()))
    focused = "^Test(RunCompactAssessment|AsyncRunReportRejectsMalformedRetainedReceipt|StaticExecutionUntrustedWriteBeforeTransaction)"
    integration = focused[:-1] + "|FinishRunReportFreshEffectEvidence)"
    cases = [
        ("ordinary", [], focused, 0),
        ("integration", ["-tags=integration"], integration, 0),
        ("race", ["-race", "-tags=integration"], integration, 0),
        ("bypass-positive", ["-overlay=" + str(overlay)], "^TestRunCompactAssessmentGenuineHistory$", 0),
        ("bypass-red", ["-overlay=" + str(overlay)], "^TestRunCompactAssessmentRetainedAdmission$", 1),
    ]
    results = []
    for name, flags, pattern, expected in cases:
        cmd = ["go", "test", "-mod=readonly", "-count=1", "-v", *flags, "-run", pattern, "./internal/executor"]
        result = subprocess.run(cmd, cwd=repo, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        log = "GOCACHE=" + shlex.quote(env["GOCACHE"]) + " " + shlex.join(cmd) + "\n"
        log += result.stdout + "\nactual_exit=" + str(result.returncode) + "\n"
        log_path = root / (name + ".log")
        log_path.write_text(log)
        valid = result.returncode == expected
        if name == "bypass-red":
            valid = valid and "retained admission diagnostic lost after valid unrelated execution:" in result.stdout and "error=<nil>" in result.stdout
        results.append({"name": name, "command": cmd, "actual_exit": result.returncode,
                        "expected_exit": expected, "valid": valid, "log_sha256": digest(log_path)})
        print(f"{name}: actual_exit={result.returncode}, expected_exit={expected}, valid={valid}", flush=True)
    (root / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    return 0 if all(result["valid"] for result in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
