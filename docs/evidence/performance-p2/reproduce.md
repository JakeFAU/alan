# P2 reproduction

These are commands for serialized execution after the timing campaign finishes;
this file reports no execution or result. Read the [contract](contract.md),
[selection/invariant](invariants.md), [driver](driver/README.md),
[instrumentation](instrument/README.md) and
[HTTP qualification](differential/README.md) for their owning definitions.
Use fresh absolute outputs, preserve failures, and retain exact commands, exits,
environment, source manifests, executable/tool hashes and all raw artifacts.
Local listeners and real process/termination tests require explicit host
permissions in restricted environments. Never overlap timing with task-launched
builds, tests, profiles, diagnostics, scans or compression.

## Identities and environment

| Role | Commit |
| --- | --- |
| Production baseline | `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315` |
| Frozen implementation | `2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0` |
| HTTP comparison tooling | `d11e896c7b6efebd427f577613664d10b919417e` |
| Original baseline extraction source | `0fd0f2fd78e23f86320b25befb3b650db840e305` |

The measured host is Apple M4/macOS 27.2, Go 1.27.3 darwin/arm64, CGO enabled,
and the pinned Go SQLite module is `github.com/mattn/go-sqlite3@v1.14.52`.
Use the existing local dependency cache; offline commands require all pinned
dependencies already present. No installer or dependency upgrade is part of
reproduction. Choose `REPO` as the delivery checkout, `BASE_SOURCE` and
`CANDIDATE_SOURCE` as immutable exports of the commits above, and `OUT` as a new
absolute campaign directory. Set cache paths to their existing local locations.

```sh
export BASE_ID=76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315
export CANDIDATE_ID=2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0
export GOTOOLCHAIN=local GOWORK=off GOPROXY=off CGO_ENABLED=1
export GOMAXPROCS=4 GOGC=100 GOMEMLIMIT=off
export GODEBUG= GORACE= GOFLAGS= GOTRACEBACK=single
export PYTHONDONTWRITEBYTECODE=1
P2="$REPO/docs/evidence/performance-p2"
SQLITE="$GOMODCACHE/github.com/mattn/go-sqlite3@v1.14.52"
mkdir "$OUT"
go version
go env -json GOVERSION GOOS GOARCH CGO_ENABLED GOMODCACHE GOCACHE
```

The recorded cache used `/private/tmp/alan-runtime-health.18e1DyxD/{cache,modcache}`.
Those paths are historical provenance, not dependencies to download or rewrite.
The retained campaign scratch is `/private/tmp/alan-p2-tf46z4pm`, with immutable
`baseline/` and `candidate/` exports. Its `baseline-record.json` and
`candidate-source-identity.json` identify those exports; filename-to-SHA-256 maps
are `baseline-source-before.json` and `candidate-source-before.json`.

## Retained evidence versus a new campaign

An old schedule binds its original absolute executable/source labels and tool
hashes. Retain them byte-for-byte. Analyze old samples with the corresponding
archived tools; do not edit their labels/hashes to satisfy current validation.
Archive relocation does not change the original recorded commands. Rebuilding
or changing tools creates a **new campaign**, with new schedule, source/tool/
binary hashes, paths and outputs; it does not reproduce old elapsed times.

To build a new campaign, invoke the **same complete public driver directory**
against each frozen source. Do not build each source's incidental driver copy:

```sh
"$P2/driver/build.sh" "$BASE_SOURCE" "$OUT/base-driver"
"$P2/driver/build.sh" "$CANDIDATE_SOURCE" "$OUT/candidate-driver"
shasum -a 256 "$OUT/base-driver" "$OUT/candidate-driver"
python3 "$P2/make_schedule.py" "$OUT/base-driver" "$OUT/candidate-driver" \
  "$BASE_ID" "$CANDIDATE_ID" "$OUT/paired-schedule.json"
python3 "$P2/run_schedule.py" "$OUT/paired-schedule.json" "$OUT/paired-results"
python3 "$P2/analyze_pairs.py" "$OUT/paired-schedule.json" \
  "$OUT/paired-results" "$OUT/paired-analysis.json"
```

The frozen protocol has seven bounds, seed 20261104, 14 excluded warmups and
84 measured rows: six adjacent balanced AB/BA pairs per bound. All 98 rows are
mandatory, prefixes are enabled, and failed rows are retained without replacement.
The analyzer verifies schedule/tool/configuration/artifact bindings. Its six-sign
B256 check is conditional on independent fair null signs; dependent prefixes and
shared-host drift remain limitations. This document makes no matrix result claim.
The retained frozen98 artifacts are `paired-schedule.json`, `paired-results/`,
`paired-plan.json` and `paired-command-results/` under the scratch directory.
Only after completion, analyze that schedule/results pair with its frozen tools;
never regenerate its schedule in place.

## Separate twelve-row diagnostic campaign

Keep SQL counters, coarse timers and ordinary sampled profiles separate from
headline timing. Build disposable overlays using the exact interfaces below;
review generated patches and retain their manifests/proofs as the instrumentation
README requires. Repeat for each source under separate output paths when
collecting post-change attribution.
The prepared candidate12 artifacts are `candidate-diagnostic-schedule.json`,
`candidate-diagnostic-results/`, `candidate-profiles/` and
`candidate-diagnostic-plan.json`. Their existence as a plan is not a run claim.

```sh
ALAN="$BASE_SOURCE"
DRIVER="$P2/driver"
"$P2/instrument/build.sh" "$ALAN" "$SQLITE" "$DRIVER" "$OUT/counters"
python3 "$P2/instrument/timing.py" --alan "$ALAN" --sqlite "$SQLITE" \
  --driver "$DRIVER" --out "$OUT/coarse" --build
```

Construct a new twelve-row schedule with the existing runner's diagnostic row
interface. The snippet freezes actual new binary and controller hashes; it does
not modify any retained schedule. Profiles use the ordinary driver, not overlays.

```sh
python3 - "$P2" "$OUT" "$BASE_ID" <<'PY'
import json, sys
from pathlib import Path
p2, out, label = Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3]
sys.path.insert(0, str(p2))
from campaign_validation import ENVIRONMENT, sha, tooling_hashes
rows = []
def row(name, bound, binary, args=(), instrumented=False, profiled=False):
    binary = binary.resolve(strict=True)
    rows.append(dict(name=name, bound=bound, binary=str(binary), source=label,
                     args=list(args), instrumented=instrumented, profiled=profiled,
                     expected_binary_sha256=sha(binary),
                     expected_environment=ENVIRONMENT.copy(),
                     expected_tooling_sha256=tooling_hashes()))
for b in (1, 8, 16, 32, 64, 128, 256):
    row(f'counters-b{b}', b, out/'counters/driver.bin', instrumented=True)
for b in (128, 256):
    row(f'coarse-b{b}', b, out/'coarse/driver.bin', instrumented=True)
for b in (32, 128):
    row(f'cpu-alloc-b{b}', b, out/'base-driver',
        ('-cpu-profile', str(out/f'b{b}.cpu'),
         '-alloc-profile', str(out/f'b{b}.alloc')), profiled=True)
row('trace-b16', 16, out/'base-driver',
    ('-trace', str(out/'b16.trace')), profiled=True)
with (out/'diagnostic-schedule.json').open('x') as f:
    json.dump(rows, f, indent=2)
    f.write('\n')
PY
python3 "$P2/run_schedule.py" "$OUT/diagnostic-schedule.json" "$OUT/diagnostic-results"
python3 "$P2/summarize_diagnostics.py" --schedule "$OUT/diagnostic-schedule.json" \
  --results "$OUT/diagnostic-results" --out "$OUT/diagnostic-summary"
```

These 7 counter + 2 coarse + 2 CPU/allocation + 1 trace rows never enter paired
medians. Retained original baseline evidence instead uses
`baseline-diagnostic-{schedule.json,results/,summary-initial/}` and the original
analysis source. Original preliminary timing has 29 rows (7 ordinary, 6 direct
bridge and 16 prefix calibration), separate from the 98-row comparison.

## Exact direct and HTTP qualification

For existing direct success/refusal/reopen schedules and crash readers, follow
[the unchanged direct qualification README](../series-scaling/differential/README.md).
Build its module against both sources. For each combination of `success`/
`refusal3` and reopen interval `0`/`1`, clone the same genuine closed seed into
new paths, execute both builds and compare every public record and typed SQL cell:

```sh
DIRECT="$REPO/docs/evidence/series-scaling/differential"
"$DIRECT/build.sh" "$BASE_SOURCE" "$OUT/base-direct"
"$DIRECT/build.sh" "$CANDIDATE_SOURCE" "$OUT/candidate-direct"
mkdir "$OUT/direct-seed" "$OUT/direct-pair"
"$OUT/base-direct" -mode create-seed -bound 8 -db "$OUT/direct-seed/store.sqlite" \
  -source-label "$BASE_ID" > "$OUT/direct-seed.json"
python3 "$DIRECT/clone_seed.py" "$OUT/direct-seed/store.sqlite" \
  "$OUT/direct-pair/base.sqlite" "$OUT/direct-pair/candidate.sqlite" > "$OUT/direct-clones.json"
"$OUT/base-direct" -mode execute-existing -bound 8 -scenario success -reopen-every 0 \
  -db "$OUT/direct-pair/base.sqlite" -source-label "$BASE_ID" > "$OUT/direct-base.json"
"$OUT/candidate-direct" -mode execute-existing -bound 8 -scenario success -reopen-every 0 \
  -db "$OUT/direct-pair/candidate.sqlite" -source-label "$CANDIDATE_ID" > "$OUT/direct-candidate.json"
python3 "$DIRECT/compare.py" same-seed "$OUT/direct-base.json" "$OUT/direct-candidate.json" \
  --baseline-db "$OUT/direct-pair/base.sqlite" --candidate-db "$OUT/direct-pair/candidate.sqlite"
```

HTTP tools are separate. Their default bound 8 and explicit bound 16 must both
retain identical endpoints and the full genuine same-seed evidence. Bound 16
guards lexical public continuation ordering. Do not normalize actual arrays.

```sh
HTTP="$P2/differential"
"$HTTP/build.sh" "$BASE_SOURCE" "$OUT/base-http"
"$HTTP/build.sh" "$CANDIDATE_SOURCE" "$OUT/candidate-http"
python3 "$HTTP/campaign.py" --baseline-binary "$OUT/base-http" \
  --candidate-binary "$OUT/candidate-http" --baseline-source "$BASE_ID" \
  --candidate-source "$CANDIDATE_ID" --output "$OUT/http8"
python3 "$HTTP/campaign.py" --bound 16 --baseline-binary "$OUT/base-http" \
  --candidate-binary "$OUT/candidate-http" --baseline-source "$BASE_ID" \
  --candidate-source "$CANDIDATE_ID" --output "$OUT/http16"
python3 "$HTTP/verify_tools.py" --binary "$OUT/candidate-http" "$OUT/http8" "$OUT/http8-proof"
python3 "$HTTP/prove_comparison.py" "$OUT/http8" "$OUT/http8-matching-proof" --comparator "$HTTP/compare.py"
python3 "$HTTP/prove_comparison.py" "$OUT/http16" "$OUT/http16-matching-proof" --comparator "$HTTP/compare.py"
```

Keep original HTTP rejected tools/RED evidence separate from corrected tools and
new positive/negative controls. Crash archives remain immutable: clone main+WAL
only after acknowledged termination/reap, hash before any reader, and inspect/
Open only disposable copies. The direct README owns those exact clone/read and
process-test commands, historical-format fixtures and independent external consumer.

## Original baseline extraction

`extract_baseline.py` is a byte-for-byte packaging of historical
`task3-report-fix2/extract-v3.py`, SHA-256
`c8d7689b46e04e1c3075336cbdd8d63d71caf9041541fa5cbd404eb56c7bbfd2`.
Its internal script identifier deliberately retains that historical v3 name;
the repository pathname is now `docs/evidence/performance-p2/extract_baseline.py`.
The emitted command records the actual invocation path and explicit paths.
Do not edit the extractor or relabel the original `baseline-data.json`.

Use archived original inputs and source export
`0fd0f2fd78e23f86320b25befb3b650db840e305`: its production helper predates the
optimization. Candidate source is invalid for this extraction. Set
`ORIGINAL_INPUTS` and `ORIGINAL_SOURCE` to those immutable directories. Outputs
must be absent, distinct, non-nested and outside both inputs and source:

```sh
python3 "$P2/extract_baseline.py" --inputs "$ORIGINAL_INPUTS" \
  --source "$ORIGINAL_SOURCE" --out "$OUT/extracted-data" --aux-out "$OUT/extracted-aux"
cmp "$P2/baseline-marginals.csv" "$OUT/extracted-data/baseline-marginals.csv"
python3 - "$P2/baseline-data.json" "$OUT/extracted-data/baseline-data.json" <<'PY'
import json, sys
old, new = [json.load(open(p)) for p in sys.argv[1:]]
assert set(old['extraction']) == set(new['extraction']) == {'script', 'sha256', 'command'}
old.pop('extraction'); new.pop('extraction')
assert old == new
PY
```

The published LF CSV must be byte-identical (SHA-256
`b6a0cbcda0bf9237d2bd716c9d8d5a11869c9c4d51beeba5f312c53036d74f15`).
JSON may differ **only** in `extraction.script`, `extraction.sha256` and
`extraction.command`; all measured values and input hashes must remain exact.
Historical v2/CRLF bytes, failed initial reproduction and whitespace failure
remain in `task3-report-fix1/`, `task3-reproduction-red/` and
`task3-staged-whitespace-red/`; v3 formatting records are in `task3-report-fix2/`.

## Qualification and artifact sealing

Run the contract's full qualification on an immutable final export after timing:
ordinary, integration and integration-race suites; focused process/replay/
inspection/receipt/continuation/compatibility cases; external consumer; both
driver/tool modules and negative controls. Run `go build ./...`, `go vet ./...`,
`go vet -tags=integration ./...`, `go mod verify`, `go mod tidy -diff`, and
Staticcheck for ordinary, integration, race and integration/race build tags.
Preserve command exits and raw output. Baseline failures are a separate record:
retain the intermittent competing-read assertion failure under its original
source identity. Preserve the failed run and unchanged rerun separately; the
exact current cause remains unestablished. Never alter baseline source or
assertions to pass it.

For example, from each selected source root, record these commands separately
with their exits and raw logs, using a distinct external evidence directory:

```sh
go test -json -p=3 -count=1 -timeout=30m ./...
go test -json -p=3 -count=1 -timeout=30m -tags=integration ./...
go test -json -p=3 -count=1 -timeout=30m -race -tags=integration ./...
go test -json -count=1 -tags=integration \
  -run 'Composition|Root|BoundedSeries|Series|CausalRecovery|Inspect|Replay|Receipt|Continuation|Historical|SIGKILL|WeatherProcessRecovery' \
  ./internal/executor ./runtime ./author ./examples/weather-agent
```

Before and after qualification, compare every source file against its recorded
SHA-256 manifest and verify the final export against the published commit. Keep
baseline, implementation, tooling, final tested export and delivery identities
distinct. Source labels alone do not verify bytes. Check whitespace/formatting,
module differences and final source scope before committing; qualification
results and independent review verdicts belong in the final research summary.
There is no standalone qualification checker assumed here. For the retained
filename-to-SHA-256 source maps, this read-only recipe checks exact files and
bytes; run it before and after qualification with the appropriate map/export:

```sh
python3 - "$SOURCE_MANIFEST" "$SOURCE_EXPORT" <<'PY'
import hashlib, json, sys
from pathlib import Path
expected = json.loads(Path(sys.argv[1]).read_text())
root = Path(sys.argv[2])
actual = {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest()
          for p in root.rglob('*') if p.is_file()}
assert actual == expected, 'source file set or bytes differ'
PY
```

Original profile interpretation command records are in
`baseline-profile-analysis-plan.json` and `baseline-profile-analysis/commands.json`;
the prepared post-change counterpart is `candidate-profile-analysis-plan.json`.
Keep all original script/path labels and actual interpreter/tool identities.

Raw artifact directories include `baseline-{timing,diagnostic}-results/`, their
`*-summary-initial/` folders, `baseline-profile-analysis/`, `reviews/`,
`task3-report-fix2/`, the same-seed/crash/compatibility outputs and the frozen
paired campaign's schedule/results/analysis. Raw publication links are not yet
available. Final checksummed archive links will live in the final research
summary and draft PR once sealed; no upload is claimed here.
