# Prospective integration with the advanced runtime main

This is supplemental integration evidence. The original Phase 3B.3 executable
freeze remains `759df6e3d2505f27e1287b2cfeb07bc4290afc0a`; its experiments and
the independent research checker inputs remain unchanged.

During delivery, remote main advanced to
`76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315`. Its series-scaling work caches the
validated composition protocol mode. The two overlapping production files are
`composition_store.go` and `composition_types.go`, in distinct hunks from this
delegation change. A prospective tree combining that main with task delivery
`c7fe6d87cdd49878558d676a301bec2d217ae3a6` is
`bdbbae5f89f0c76d48618b3d8891dc40199ca4ed`. No original branch was merged,
rebased or rewritten.

The tree was produced with:

```sh
git merge-tree --write-tree 76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315 c7fe6d87cdd49878558d676a301bec2d217ae3a6
```

It was archived into a disposable directory. All Go/module input hashes are in
`source.json`; three successful checks ran directly on those bytes:

- Core integration: 1,611 test/subtest pass events.
- Authority and retained-mode race tests: 119 pass events.
- Full root-module integration Staticcheck: exit 0.

The first reciprocal race run failed during evidence export because
`reciprocalSources` requires `git rev-parse HEAD`, while a source archive has no
Git metadata. `apparatus-failure.json`, the original command/log and all its raw
outputs are retained. This is an apparatus failure, not a semantic counterexample
or a passing qualification.

A local Git repository was initialized only in the disposable source directory.
Its exact tree was checked against the prospective tree above, then committed as
`f29e41b7532eaa0cbfd2b1f99f5171986a8d9a90` to provide an honest source identity.
This synthetic snapshot has no claim to be a published commit or a merge of an
original branch. The affected reciprocal race command was repeated into a new
output directory, with no source change: 53 pass events, zero skips/failures and
48 indexed cases. Its 1,761 per-case artifact digests were verified. The initial
failed run was not overwritten.

`commands.json` contains the initial four actual commands, including the failed
export run. `reciprocal-race-corrected-command.json` contains the replacement
qualification. `scratch-source.json` relates its local commit to the exact input
tree. The Python runners preserve the original execution paths; to reproduce,
construct the same tree in a fresh directory with Git provenance, use the recorded
toolchain/environment and frozen old reader, and give the fixture a fresh evidence
root. A different local commit timestamp does not change the asserted tree.

The archive includes both runs under `prospective/`. Verify its SHA256 and every
member against `manifest.json` before using it. Hash checks opened no SQLite.
Any database reader must use a disposable complete database/WAL/SHM copy. This
supplement checks the named combination, not future main changes, a new finite
model, or general compositional closure.
