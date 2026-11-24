# Consumer release qualification

Run on a local Unix filesystem with Go 1.27.3 or newer, a C compiler, CGO, Git and Python 3. The producer needs a verified copy of the SQLite dependency or connectivity to obtain it. From the runtime checkout, select the exact committed revision for the candidate.

```sh
python3 tools/qualify_consumer.py --candidate HEAD --output /tmp/alan-v0.1.0-candidate.json
```

The producer resolves v0.1.0 inside a disposable local Git repository, verifies the resolved commit, and asks Go to build the standard distribution ZIP. It serves the module and the pinned SQLite dependency through a directory implementing the Go proxy protocol. The driver ZIP and module descriptor must match the candidate's checked-in go.sum. This local candidate uses those hashes and the local Git revision for provenance. The public checksum database first records the released version after publication.

The consumer starts outside the checkout with empty, private module and build caches, workspace mode off and a new application module. It executes the install, build and four separate process invocations from the packaged README. The application checks the stored state, result and call count. The driver checks actual output, public documentation, imports, absence of replacements, selected module checksums and module verification. CI runs the candidate path for proposed changes. A manifest identifies the commit, compressed ZIP, Go module sums, tested platform and outputs.

Keep the passing candidate JSON with the release decision. After authorized publication, run the following command with that exact manifest.

```sh
python3 tools/qualify_consumer.py --published /tmp/alan-v0.1.0-candidate.json --output /tmp/alan-v0.1.0-published.json
```

The published check resolves the tag through the public Go proxy with the public checksum database, checks the public Git tag's commit, compares both module sums with the candidate, and runs the same consumer. Tag the commit recorded in the manifest. If the intended release commit changes during integration, qualify that revision and retain its manifest before publication. Creating a public tag is a separate maintainer action.

## Evidence distribution

### Reproduce the historical reader gates

The following commands use the same Go toolchain and require local loopback and subprocess support. Supply a fresh path. The preparer builds the historical readers, checks the archive and member hashes for formats 3–12, and generates four authority stores with the pre-release runtime for formats 13–16. Their promoted copies retain conflicting receipt evidence. Generation assigns fresh store namespaces. Each run records its input hashes and checks the same format and replay outcomes.

```sh
inputs=$(mktemp -d)/readers
python3 tools/prepare_release_stores.py "$inputs"
export ALAN_RELEASE_STORES="$inputs/matrix" ALAN_V010_OLD_READER="$inputs/pre-release-reader"
export ALAN_PHASE3B3_OLD_READER="$inputs/old13" ALAN_PHASE3B4_OLD_READER="$inputs/old15"
export ALAN_PHASE3B4_PUBLIC_PARENTS="$inputs/legacy-authority"
export ALAN_PHASE3B4_PROMOTED_OUTPUT="$inputs/new-promotions"
go test -count=1 -tags integration -run 'Test(ReleaseHistoricalStores|ScopedAsset|ScopedContinuation|Phase3B4PromotedPublicParents|RequestAuthorityFormatsConflictPromotionAndOldReader|RequestBindingConflictAndOldReader)' ./runtime ./internal/executor
```

### Archive provenance

The runtime Git repository retains the campaign source, manifests, database fixtures and archives under their original paths. The Go module boundaries in `docs/evidence` and `docs/qualification` exclude those two directories from the library ZIP. Developer reproduction starts from the corresponding Git commit and each campaign's manifest. The fixtures under `internal/storetest/testdata` and `runtime/testdata` still travel with the library. Source-specific historical measurements and review verdicts retain the scope of their original commits. [Original qualification archive](https://github.com/JakeFAU/alan/tree/64a692b6f82ae10353f50c8741377166bc6cf854/docs/qualification) · [Original performance evidence](https://github.com/JakeFAU/alan/tree/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence)
