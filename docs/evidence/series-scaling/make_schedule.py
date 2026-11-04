#!/usr/bin/env python3
"""Freeze complete paired schedule, executable bytes and controlled configuration."""
import argparse
import json
import os
from pathlib import Path
import subprocess
from campaign_validation import CONTRACT, ENVIRONMENT, layout, sha, tooling_hashes, validate_schedule


def runtime_configuration():
    env = dict(os.environ, GOTOOLCHAIN="local", GOWORK="off", GOPROXY="off")
    info = json.loads(subprocess.check_output(["go", "env", "-json", "GOVERSION", "GOOS", "GOARCH"], env=env))
    return {"go_version": info["GOVERSION"], "goos": info["GOOS"], "goarch": info["GOARCH"],
            "gomaxprocs": 4, "mem_profile_rate": 524288, "profiled": False,
            "instrumented": False, "schedule": "ordinary", "reopen_every": 0}


def build_schedule(baseline_binary, candidate_binary, baseline_source, candidate_source, runtime=None):
    binaries = {"baseline": str(Path(baseline_binary).resolve(strict=True)),
                "candidate": str(Path(candidate_binary).resolve(strict=True))}
    sources = {"baseline": baseline_source, "candidate": candidate_source}
    hashes = {v: sha(p) for v, p in binaries.items()}
    runtime = runtime if runtime is not None else runtime_configuration()
    tools = tooling_hashes()
    rows = []
    for identity in layout():
        variant = identity["variant"]
        rows.append(dict(identity, campaign_contract=CONTRACT, binary=binaries[variant],
                         source=sources[variant], expected_binary_sha256=hashes[variant],
                         expected_runtime=runtime.copy(), expected_environment=ENVIRONMENT.copy(),
                         expected_tooling_sha256=tools.copy()))
    validate_schedule(rows)
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline_binary")
    parser.add_argument("candidate_binary")
    parser.add_argument("baseline_source")
    parser.add_argument("candidate_source")
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    if args.output.exists():
        raise SystemExit("schedule already exists")
    rows = build_schedule(args.baseline_binary, args.candidate_binary, args.baseline_source, args.candidate_source)
    with args.output.open("x") as output:
        output.write(json.dumps(rows, indent=2) + "\n")
    print("Frozen executable/configuration identities, 14 excluded warmups and 84 measured processes.")


if __name__ == "__main__":
    main()
