#!/usr/bin/env python3
"""H-specific modes use a complete H inventory; old collectors are not executed."""

import os, subprocess, sys
from common import ROOT

mode = sys.argv[1]
assert mode in (
    "default",
    "normal",
    "race",
    "old-consumer",
    "old-consumer-race",
    "consumer",
    "consumer-race",
    "focused",
)
cmd = ["go", "test", "-json", "-count=1", "-timeout=60m"]
if mode == "focused":
    cmd += ["-tags", "integration", "./internal/adaptivetest"]
elif "consumer" in mode:
    folder = "worktree" if mode.startswith("old-") else "adaptive"
    cmd = (
        [
            "go",
            "-C",
            f"examples/{folder}/consumer",
            "test",
            "-json",
            "-count=1",
            "-timeout=60m" if mode.endswith("race") else "-timeout=15m",
        ]
        + (["-race"] if mode.endswith("race") else [])
        + ["./..."]
    )
else:
    if mode == "race":
        cmd[cmd.index("-timeout=60m")] = "-timeout=600m"
    cmd += (
        (["-race"] if mode == "race" else [])
        + ([] if mode == "default" else ["-tags", "integration"])
        + ["./..."]
    )
env = dict(os.environ, ALAN_H_MODE=mode)
raise SystemExit(
    subprocess.run(
        [
            sys.executable,
            str(ROOT / "docs/foundational/phase1h/tools/capture.py"),
            "final-" + mode,
            "--",
            *cmd,
        ],
        cwd=ROOT,
        env=env,
    ).returncode
)
