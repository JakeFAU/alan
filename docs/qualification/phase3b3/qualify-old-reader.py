#!/usr/bin/env python3
"""Qualify storage formats 13 and 14 against the pinned pre-authority CLI."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
from tempfile import TemporaryDirectory


OLD_READER = "af6943a5805a9a6c0e8fe77f804264f4765d07ec"
FORMAT_TEST = "TestRequestAuthorityFormatsConflictPromotionAndOldReader"
COMMANDS = ("inspect", "replay")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path, help="new evidence directory outside the checkout")
    output = parser.parse_args().output.resolve()
    repo = Path(__file__).resolve().parents[3]
    if output == repo or repo in output.parents:
        parser.error("the evidence directory must be outside the checkout")
    output.mkdir()

    env = dict(os.environ, GOFLAGS="", GOWORK="off", GOTOOLCHAIN="local")
    env.pop("ALAN_PHASE3B3_OLD_READER", None)

    def run(name, argv, cwd=repo, extra=None):
        argv = list(map(str, argv))
        command_env = dict(env, **(extra or {}))
        result = subprocess.run(argv, cwd=cwd, env=command_env, capture_output=True)
        (output / f"{name}.stdout").write_bytes(result.stdout)
        (output / f"{name}.stderr").write_bytes(result.stderr)
        command = {"argv": argv, "cwd": str(cwd), "exit": result.returncode,
                   "environment": {key: command_env[key] for key in (
                       "GOFLAGS", "GOWORK", "GOTOOLCHAIN", *(extra or {}))}}
        (output / f"{name}.json").write_text(json.dumps(command, indent=2) + "\n")
        if result.returncode != 0:
            raise SystemExit(f"{name} exited {result.returncode}; see {output / (name + '.stderr')}")
        return result.stdout

    with TemporaryDirectory(prefix="alan-old-reader-") as temp:
        scratch = Path(temp)
        candidate = run("candidate-head", ["git", "rev-parse", "HEAD"]).decode().strip()
        sources = {}
        for role, revision in (("historical", OLD_READER), ("candidate", candidate)):
            tree = run(f"{role}-tree", ["git", "rev-parse", f"{revision}^{{tree}}"]).decode().strip()
            archive = scratch / f"{role}.tar"
            source = scratch / role
            source.mkdir()
            run(f"{role}-archive", ["git", "archive", "--format=tar", "-o", archive, revision])
            run(f"{role}-extract", ["tar", "-xf", archive, "-C", source])
            sources[role] = {"commit": revision, "tree": tree}

        reader = scratch / "historical-alan"
        run("go-version", ["go", "version"])
        run("historical-build", ["go", "build", "-buildvcs=false", "-mod=readonly", "-o", reader,
                                 "./cmd/alan"], scratch / "historical")
        run("historical-build-info", ["go", "version", "-m", reader])
        sources["historical"]["binary_sha256"] = hashlib.sha256(reader.read_bytes()).hexdigest()
        sources["runner_sha256"] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
        (output / "sources.json").write_text(json.dumps(sources, indent=2) + "\n")

        supported = scratch / "historical-supported.sqlite"
        run("historical-init", [reader, "init", "--db", supported, "--demo", "nested"])
        inspect = json.loads(run("historical-inspect-supported", [reader, "inspect", "--db", supported]))
        replay = json.loads(run("historical-replay-supported", [reader, "replay", "--db", supported]))
        version = inspect["Tables"]["metadata"][0]["schema_version"]
        if not (3 <= version < 13 and inspect["Tables"]["runs"] and replay["Runs"]
                and inspect["Namespace"] == replay["Namespace"]):
            raise SystemExit("historical reader did not decode its own supported Store")

        log = run("format-rejections", ["go", "test", "-buildvcs=false", "-mod=readonly", "-count=1",
                                        "-timeout=2m", "-json", f"-run=^{FORMAT_TEST}$",
                                        "./internal/executor"], scratch / "candidate",
                  {"ALAN_PHASE3B3_OLD_READER": str(reader)})
        events = [json.loads(line) for line in log.splitlines()]
        passed = {event.get("Test") for event in events if event["Action"] == "pass"}
        required = {f"{FORMAT_TEST}/{version}/frozen_old_reader/{command}"
                    for version in (13, 14) for command in COMMANDS}
        if missing := required - passed:
            raise SystemExit(f"missing required rejection passes: {', '.join(sorted(missing))}")
        print(f"Historical reader accepted format {version} and rejected formats 13 and 14 "
              f"through inspect and replay. Evidence saved to {output}")


if __name__ == "__main__":
    main()
