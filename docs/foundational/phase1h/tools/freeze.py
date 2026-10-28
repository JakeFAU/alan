#!/usr/bin/env python3
"""Freeze complete contributing current sources, never generated self-hashes."""

import json, subprocess
from common import ROOT, DOC, inventory, source_id, environment, sha

out = DOC / "source-manifest.json"
assert not out.exists(), "refuse to replace frozen manifest"
files = inventory()
env = environment()
cache = subprocess.check_output(
    ["go", "env", "GOMODCACHE"], cwd=ROOT, env=env, text=True
).strip()
from pathlib import Path

proxy = Path(cache) / "cache/download/github.com/mattn/go-sqlite3/@v"
dependency = dict(
    module="github.com/mattn/go-sqlite3",
    version="v1.14.52",
    zip_sha256=sha((proxy / "v1.14.52.zip").read_bytes()),
)
manifest = dict(
    SourceID=source_id(files),
    Files=files,
    HeadAtFreeze=subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=ROOT, text=True
    ).strip(),
    OriginalBaselineCommit="4550e0f4ba4c57cae8e84d1afe9fbf47e16786d8",
    MainBaseCommit="56892cb20c4f4add4bcf786d146b0aed82fbc5d1",
    Task3BaseCommit="19065f30d4135c8710faed3db5c7311a01c2530e",
    RebaseIdentity="docs/foundational/phase1h/evidence/rebase-20261028/identity.json",
    Dependency=dependency,
    Scope="Complete current package/test/testdata/assets and both separate consumer modules, build configuration, Phase1H and reused historical tools/fixtures/semantic comparison input. Generated source manifests/results and historical source archives excluded; separately archive-manifested. Toolchains not bundled.",
)
out.write_text(json.dumps(manifest, indent=2) + "\n")
print(json.dumps({k: v for k, v in manifest.items() if k != "Files"}, indent=2))
