#!/usr/bin/env python3
"""Reproduce the rejected J2 all-maximal-states-complete claim; expected exit 1."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[5]
source = Path(__file__).with_name("completion-hypothesis-red_test.go.txt")
env = os.environ.copy()
env.pop("OPENAI_API_KEY", None)
env.update(GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off", GOFLAGS="-mod=readonly", GOCACHE="/tmp/alan-phase1k-go-build")
with tempfile.TemporaryDirectory(prefix="alan-phase1k-red-") as temporary:
    overlay = Path(temporary) / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "research/communication/model_test.go"): str(source)}}))
    raise SystemExit(subprocess.call(["go", "test", "-overlay=" + str(overlay), "-v", "-count=1", "-run", "^TestAllMaximalStatesComplete$", "./research/communication"], cwd=root, env=env))
