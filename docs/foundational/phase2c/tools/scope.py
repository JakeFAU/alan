"""Verify the verified integration tree, frozen protocol, historical archives and change scope."""
from hashlib import sha256
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[4]
BASE = "a461f25fb42d1663735f68296460eceda4270d47"
PRIOR = "41fc50b3b821dcc4b3700920de3109fb3204315d"
FROZEN = "77c253e0c72ab77217b1e87c266a941a91fe2354"
P = "docs/foundational/phase2c/"
CORE = {
    "adapters.go", "composition_binding.go", "composition_facts.go", "composition_replay.go",
    "composition_reservation.go", "composition_store.go", "composition_types.go", "dispatch.go",
    "inspect.go", "invocation.go", "invocation_handoff.go", "receipt_read.go", "receipt_replay.go",
    "receipt_result.go", "receipt_redelivery_test.go", "reconciliation_profile.go", "reconciliation_test.go",
    "replay.go", "runner.go", "store.go",
}
EXACT = {
    "author/application.go", "author/invocations.go", "runtime/runtime.go",
    "cmd/alan-resource/main.go", "cmd/alan-resource/main_test.go",
    "internal/fixture/resource.go", "internal/fixture/ledger.go", "internal/fixture/ledger_test.go",
    "paper/alan-dissertation.md", "paper/alan-dissertation.tex", "paper/alan-dissertation.pdf",
} | {"internal/executor/" + n for n in CORE}


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT)


def main():
    assert git("rev-parse", BASE + "^{tree}") == git("rev-parse", PRIOR + "^{tree}")
    git("merge-base", "--is-ancestor", BASE, FROZEN)
    git("merge-base", "--is-ancestor", FROZEN, "HEAD")
    assert not git("diff", BASE, "--", "docs/foundational/phase2a", "docs/foundational/phase2b",
                   "go.mod", "go.sum", "paper/references.bib", "effects", "schema", "research/semantics")
    for old, expected in (("phase2a", "c5fc2c55087cdc206fc4af578a90654b82f027714ef02985d0e40081d8ce6855"),
                          ("phase2b", "3e4b4c392dcd9340f778333ed25d932d83d3173e87a2e68a951859fd6a4627c1")):
        name = f"docs/foundational/{old}/evidence/index.sha256"
        assert sha256(git("show", PRIOR + ":" + name)).hexdigest() == expected
    for path in git("ls-tree", "-r", "--name-only", FROZEN, P).decode().splitlines():
        if Path(path).name.startswith(("00-", "01-")) or path.endswith(("assignment.md", "spec-review.md")):
            assert (ROOT/path).read_bytes() == git("show", FROZEN + ":" + path), path
    changed = set((git("diff", "--name-only", "-z", BASE) +
                   git("ls-files", "--others", "--exclude-standard", "-z")).decode().split("\0")) - {""}
    for path in changed:
        assert (path in EXACT or path.startswith((P, "research/reconciliation/", "examples/reconciliation/",
                                                  "adapters/receiptquery/"))
                or path.startswith("internal/durabletest/phase2c_") and path.endswith("_test.go")), path
    print("PASS historical tree/index identities, frozen specification, unchanged protected paths and scoped changes")


if __name__ == "__main__":
    main()
