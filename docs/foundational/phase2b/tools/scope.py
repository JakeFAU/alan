"""Verify historical evidence, frozen contracts, protected implementation and delivery scope."""
import hashlib
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
BASE = "72e406b73b05a3ed4a6e8b37801ab6e0f0a73209"
HISTORICAL = "24485a2b4f5bb3af45027995c6e6d8f34e0acb2d"
PHASE = "docs/foundational/phase2b/"
EXACT = {
    "cmd/alan-resource/main.go", "research/semantics/runtime_conformance_test.go",
    "internal/executor/adapters.go", "internal/executor/dispatch.go", "internal/executor/external.go",
    "internal/executor/runner.go", "internal/executor/store.go", "runtime/runtime.go",
    "internal/executor/dispatch_gate_test.go", "internal/executor/quiescence_test.go",
    "internal/executor/recovered_accounting_test.go", "internal/executor/external_test.go", "internal/durabletest/phase2_process_test.go",
    "internal/durabletest/phase2a_authority_test.go",
    "paper/alan-dissertation.md", "paper/alan-dissertation.tex", "paper/alan-dissertation.pdf",
}

def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT)

def check():
    assert git("rev-parse", BASE+"^{tree}") == git("rev-parse", HISTORICAL+"^{tree}")
    historical_index = "docs/foundational/phase2a/evidence/index.sha256"
    assert hashlib.sha256(git("show", HISTORICAL+":"+historical_index)).hexdigest() == "c5fc2c55087cdc206fc4af578a90654b82f027714ef02985d0e40081d8ce6855"
    for name in ("assignment.md","01-contract.md","01-source-and-matrix.md"):
        assert (ROOT/PHASE/name).read_bytes() == git("show", "656f84b:"+PHASE+name), name
    paths = (git("diff","--name-only","-z",BASE) + git("ls-files","--others","--exclude-standard","-z")).decode().split("\0")
    for p in set(paths) - {""}:
        is_phase_test = p.startswith(("internal/durabletest/phase2b_","internal/executor/phase2b_")) and p.endswith("_test.go")
        assert is_phase_test or p in EXACT or p.startswith((PHASE, "examples/revocation/", "research/dispatchauthority/")), p
    assert not git("diff", BASE, "--", "docs/foundational/phase2a","research/effectprotocol","paper/references.bib","go.mod","go.sum")
    print("PASS frozen contract, historical index/trees, previous archives and repository path scope")

if __name__ == "__main__":
    check()
