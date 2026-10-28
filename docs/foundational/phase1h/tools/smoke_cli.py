#!/usr/bin/env python3
"""Focused actual CLI smoke of the documented loopback provider, retaining exact IO."""

from pathlib import Path
import gzip, json, os, subprocess, tempfile
from common import ROOT, environment, sha

out = Path(os.environ["ALAN_H_ARTIFACTS"])
out.mkdir(parents=True, exist_ok=True)
env = environment()
commands = []
with tempfile.TemporaryDirectory(prefix="alan-phase1h-cli-") as tmp:
    binary = Path(tmp) / "alan-adaptive"

    def command(argv):
        p = subprocess.run(
            [str(x) for x in argv], cwd=ROOT, env=env, capture_output=True
        )
        index = len(commands) + 1
        (out / f"{index:02d}.stdout.gz").write_bytes(gzip.compress(p.stdout, mtime=0))
        (out / f"{index:02d}.stderr.gz").write_bytes(gzip.compress(p.stderr, mtime=0))
        commands.append(
            dict(
                argv=[str(x) for x in argv],
                exit=p.returncode,
                stdout_sha256=sha(p.stdout),
                stderr_sha256=sha(p.stderr),
            )
        )
        assert p.returncode == 0, (commands[-1], p.stderr.decode())
        return json.loads(p.stdout) if p.stdout else None

    command(["go", "-C", "examples/adaptive/consumer", "build", "-o", binary, "."])
    for case, nested, leaf, ids, invocations, runs, effects in [
        ("scalar-paired", False, "controlled", [1, 2, 3, 7], 5, 5, 4),
        ("evidence", True, "evidence", [1, 4], 4, 6, 4),
    ]:
        ledger = out / (case + ".provider.jsonl")
        provider = subprocess.Popen(
            [
                "python3",
                str(ROOT / "docs/foundational/phase1h/tools/controlled_provider.py"),
                "--case",
                case,
                "--port",
                "0",
                "--ledger",
                str(ledger),
            ],
            cwd=ROOT,
            env=env,
            stdout=subprocess.PIPE,
            text=True,
        )
        try:
            endpoint = json.loads(provider.stdout.readline())["endpoint"]
            db = Path(tmp) / (case + ".sqlite")
            request = {
                "seeds": [
                    {"work_id": 1, "value": 0},
                    {"work_id": 2, "value": 1},
                    {"work_id": 3, "value": 2},
                ]
                if leaf == "controlled"
                else [
                    {
                        "work_id": 1,
                        "question": "How does the retained WAL record relate to database changes?",
                    }
                ],
                "max_rounds": 3,
                "max_work": 8,
            }
            requestPath = out / (case + ".input.json")
            requestPath.write_text(json.dumps(request) + "\n")
            init = command(
                [
                    binary,
                    "init",
                    "--db",
                    db,
                    "--leaf",
                    leaf,
                    "--capacity",
                    "2",
                    *(["--nested"] if nested else []),
                ]
            )
            command(
                [
                    binary,
                    "submit",
                    "--db",
                    db,
                    "--id",
                    "frozen-cli",
                    "--input",
                    requestPath,
                ]
            )
            report = command(
                [binary, "run", "--db", db, "--controlled-endpoint", endpoint]
            )
            inspection = command([binary, "inspect", "--db", db])
            command([binary, "replay", "--db", db])
            command([binary, "recover", "--db", db, "--controlled-endpoint", endpoint])
            q = inspection["Result"]
            actual = [r["work_id"]["Number"] for r in q["results"]["List"]]
            assert (
                actual == ids
                and q["complete"]["Flag"]
                and q["stop_reason"]["Text"] == "exhausted"
            )
            ch = report["Roots"][0]["Charges"]
            assert (ch["Invocations"], ch["CreatedRuns"], ch["Effects"]) == (
                invocations,
                runs,
                effects,
            )
            print(
                json.dumps(
                    dict(
                        case=case,
                        init=init,
                        ids=actual,
                        charges=ch,
                        provider_contacts=len(ledger.read_text().splitlines()),
                    )
                )
            )
        finally:
            provider.terminate()
            provider.wait(timeout=10)
(out / "commands.json").write_text(json.dumps(commands, indent=2) + "\n")
