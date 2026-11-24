# Reproduce the integrated SQLite and retained-store checks

The public pre-release runtime and the first release candidate link go-sqlite3 v1.14.52 (SQLite 3.53.4). The integrated runtime links v1.14.53 (SQLite 3.54.1). Each evidence record identifies the commits it compares. The historical research tree is read at its extraction commit and compiled with its own original module and sums. All commands build locally with a CGO compiler and run on a local filesystem with SQLite WAL, exclusive locks, subprocesses and loopback. The recorded host used Darwin/arm64, Go 1.27.4 and Python 3.15 with sqlite3 3.54.1.

Start with local full-history Git checkouts of the public runtime and its separate research repository. In a checkout containing this record and its companion verification script, set `harness` to their directory and `candidate_commit` to the integrated commit reported in `storage-results.json`. Ensure both driver versions are available to the Go toolchain with their checked sums.

```sh
work=$(mktemp -d)
repo=/path/to/alan
research=/path/to/alan-research
harness=/path/to/this/evidence/directory
candidate_commit=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["integrated"]["commit"])' "$harness/storage-results.json")
export GOWORK=off GOTOOLCHAIN=local CGO_ENABLED=1
for side in base rc integrated; do
    git clone --quiet --shared --no-checkout "$repo" "$work/$side"
done
git -C "$work/base" checkout --quiet --detach 64a692b6f82ae10353f50c8741377166bc6cf854
git -C "$work/rc" checkout --quiet --detach bea0acc86d9a7302ccc3511327455922503c4a57
git -C "$work/integrated" checkout --quiet --detach "$candidate_commit"
go -C "$work/base" mod download
go -C "$work/integrated" mod download
go -C "$work/base" mod verify
go -C "$work/integrated" mod verify
export GOPROXY=off
cd "$work/integrated"
python3 -B docs/qualification/sqlite-1.14.53/compare.py \
    "$work/base" "$work/integrated" "$work/cross"
python3 -B docs/qualification/phase3b3/qualify-old-reader.py "$work/pre-authority"
python3 -B tools/prepare_release_stores.py "$work/readers"
python3 -B "$harness/verify-cross.py" \
    "$work/base" "$work/rc" "$work/integrated" "$research" "$work/cross" "$work/extra"
```

The original cross-binding campaign runs the native SQLite trace and both candidate/baseline writer directions at two acknowledged crash boundaries. It builds and uses the frozen K01 owner, checks outputs for formats 3–16, and runs database integrity and foreign-key checks on 18 inputs. The independent pre-authority CLI accepts its format 3 control and rejects format 13/14 inspection and replay. For each paired inspection/replay, the comparator verifies the actual engine's full version and source ID first. It compares every other byte. Retained rows, epoch rules, scopes and adapter evidence participate in the comparison.

The companion script compares the frozen research and previous candidate binaries with the integrated writer and reader. Its eight additional kill schedules each verify the resumed outcome, original and control ledger, read-only database/WAL hashes, and corruption refusal. Both release-capable binaries also write each scoped format 17–30. The capture wrapper retains only disposable fixtures from the unmodified storage-format tests and preserves the pre-scope CLI's stdout, stderr and exit code. It checks a pre-scope reader's refusal and compares the two supported engines on every resulting fixture. The original public corpus quickstart produces a legacy-format waiting store that the integrated reader resumes with one effect. The RC and integrated asset-scoped quickstarts resume one another's waiting stores. A pre-scope reader rejects the integrated form through both inspection and replay.

For the exact 14 historical inputs from the first candidate, restore 13–16 from its retained archive and authenticate every database against its manifest. The preparer supplies the original 3–12 byte images and both frozen authority readers. Use a fresh directory for promoted outputs on each tagged invocation.

```sh
cp -R "$work/readers/matrix" "$work/historical"
tar -xzf docs/qualification/v0.1.0/historical-authority-inputs.tar.gz -C "$work/historical"
python3 - "$work/historical" <<'PY'
from pathlib import Path
import hashlib, json, sys
root = Path(sys.argv[1])
expected = json.loads(Path("docs/qualification/v0.1.0/historical-inputs.json").read_text())
assert {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in root.iterdir()} == expected
PY
export ALAN_RELEASE_STORES="$work/historical"
export ALAN_PHASE3B3_OLD_READER="$work/readers/old13"
export ALAN_PHASE3B4_OLD_READER="$work/readers/old15"
export ALAN_V010_OLD_READER="$work/readers/pre-release-reader"
export ALAN_PHASE3B4_PUBLIC_PARENTS="$work/readers/legacy-authority"
export ALAN_PHASE3B4_PROMOTED_OUTPUT="$work/rechecked-promotions"
go test -count=1 -tags integration -run 'Test(ReleaseHistoricalStores|ScopedAsset|ScopedContinuation|Phase3B4PromotedPublicParents|RequestAuthorityFormatsConflictPromotionAndOldReader|RequestBindingConflictAndOldReader)' ./runtime ./internal/executor
```

The retained hashes identify the individual images used during the recorded run. Fresh generations of authority stores or scoped stores can allocate new namespaces. Compare the verified formats, projections and outcomes on each independently reproduced fixture.
