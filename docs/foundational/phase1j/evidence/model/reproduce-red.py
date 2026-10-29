#!/usr/bin/env python3
"""Recheck the deliberately false all-maximal-paths-complete hypothesis (exit 1)."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[5]
source = Path(__file__).with_name("completion-hypothesis-red_test.go.txt")
env = os.environ.copy()
env.pop("OPENAI_API_KEY", None)
env.update(GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off", GOFLAGS="-mod=readonly", GOCACHE="/tmp/alan-phase1f-go-build")
with tempfile.TemporaryDirectory(prefix="alan-phase1j-red-") as temporary:
    overlay = Path(temporary) / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "research/join/model_test.go"): str(source)}}))
    raise SystemExit(subprocess.call(["go", "test", "-overlay=" + str(overlay), "-v", "-count=1", "-run", "^TestReferenceCensus$", "./research/join"], cwd=root, env=env))
