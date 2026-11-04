# Semantic, recovery and compatibility qualification

The production source under initial qualification is immutable commit
`e32d0e2f020638f301dcd0f0687c5c444180225c`, compared with baseline
`af6943a5805a9a6c0e8fe77f804264f4765d07ec`. Later delivery verification is
recorded with its own exact revision; it must not relabel these earlier runs.

## New reuse-premise attacks

The private mode projection has a smaller premise than a validated-history
cache: the exact private profile object's protocol version cannot change during
its lifetime. New tests exercise ordinary nil, foundation and invocation modes;
reconciliation/dispatch profiles; caller and decoded-descriptor alias mutation;
independent same-name Stores; and fresh-owner object reconstruction.

Disposable retained stores are corrupted with truncated JSON, column/raw version
mismatch, coherent foundation downgrade and physical schema downgrade. Open,
Replay, raw Inspect and typed InspectExecution reject each mismatch. Independent
epoch reads verify failed reopening has not partially renewed the runs. Existing
tests still reject unapproved/stale/foreign continuations, source revocation,
wrong approval/action/namespace, dispatch admission forgery, receipt conflicts,
impossible chronology and corrupt causal references. The flag never stands in
for those dynamic facts.

The mode-query allocation test fails on the old implementation at 27
allocations/query and passes at zero on the candidate. Semantic preservation
tests appropriately pass both sources. Two assumptions in the new tests were
corrected during development: foundation stores use their existing private
opening path, and a fixture already at format 4 needs a real downgrade to 3 for
a corruption test. Original failures and corrections are retained. No existing
assertion or accepted-history rule was weakened.

## Exact differential evidence

Four controlled eight-position execution pairs begin from byte copies of one
genuinely provisioned, closed, untouched seed. They cover successful results and
a retained third refusal, each with ordinary ownership and reopening after every
selected position. Both builds receive the same schedule and namespace.

The [independent external module](differential/README.md) collects full typed
snapshots, every History page, Replay, SeriesView, complete Prepared requests and
Candidate responses at every cut. The comparator requires exact logical
equality of every public field and every SQLite schema object/typed cell. Only
the caller's source label, invocation path and physical SQL row ordering are
ignored. IDs, epochs, revisions, grants, charges, payload bytes, digests,
selected observations, causal order and uncertainty are not normalized.

The four pairs pass, with 18, 25, 18 and 25 public cuts. The same comparator also
passes 21 baseline-written real process-cut reader/renewal pairs, for 25 total
comparisons and 101 successful commands. A separate reviewer independently
recomputed all public-object and typed-SQL comparisons without importing the
author's comparator. Eleven disposable report/SQL mutations reject, including the
request key and original approval predecessor; four clone safety guards reject.

The retained third refusal is a local unsuccessful result and is selected. Its
remote consequence remains unresolved under the existing runtime contract. The
fourth through eighth observations use distinct, protected approvals and paid
opportunities. This test does not reinterpret refusal as proven nonapplication.

## Actual process death and recovery

The unchanged weather-series suite uses acknowledged barriers, SIGKILL and reap,
not sleeps, at ten boundaries: before and after fourth approval, after its
selection, after authorization, after payment before admission, after admission
before contact, accepted third request with response lost, third result retained,
third settlement and third terminal completion. Each source exports 21 main/WAL
pairs, including stopped/final cuts and an ordinary reference history.

Baseline ordinary/integration and candidate ordinary/integration/integration-race
suites execute the actual process protocol. An independent reviewer inspected eight decisive baseline
cuts with the candidate reader, then all 21 candidate cuts with the baseline
reader, using disposable byte copies. Original archive main/WAL hashes remain
unchanged. Every candidate namespace ties to its actual test log and acknowledged
kill history. Open changes only the expected epochs of open runs; all other
typed cells and schema entries remain exact.

| Cut | Required retained fact, verified in actual stores |
| --- | --- |
| Before fourth approval | No fourth approval/effect; three paid opportunities |
| After approval, before selection | Original fourth approval exists; no fourth effect |
| Fourth paid, before admission | Fourth charged and attempted; no admission/result; contact bounds [0,0] |
| Fourth admitted, before call | Original admission exists; no result; possible contact bounds [0,1] |
| Third accepted, response lost | Original third remains paid, admitted, unresolved, unsettled |
| Approved fourth after lost third | Same entire third effect; fourth has its own approval/result/charge |
| Third result retained, before settlement | Original retained result exists; local settlement pending |
| Third result recovered | Exact authorization, prepared attempt and result survive; selection needs no repeat contact |

Generic archive readers never Run or register an adapter; they qualify readers
and owner renewal. Actual continuation/resume behavior is exercised separately
by the unchanged process suites. The existing `tools/compare_stores.py` also
passes all four baseline/candidate writer-reader combinations at pre/post-commit
SIGKILL boundaries: the opposite binary resumes the actual killed store, both
reach identical 26-commit/4-intent histories, Inspect/Replay preserve durable
bytes, and both reject a corrupted successor on a disposable copy.

The contract remains single-host cooperative ownership. Same-store exclusion,
independent Stores and concurrent readers remain covered. Unsupported direct
SQLite mutation behind an active owner is not added to the supported failure
model. Neither crash nor local quiescence establishes remote finality.

## Public consumer and storage formats

The unchanged external eight-operation consumer is copied from the preceding
bounded-series qualification and built against the frozen candidate. Ordinary
and race-enabled tests pass for fresh bounds 1/8/32/256, independent manual
lowering equality, the real eight-effect execution, historical format 4 lifecycle
and format 9 weather archives, and read-only durable-byte preservation. The live
Open-Meteo artifact is used offline; no new live request is made.

A real competing receipt offered through the existing writer promotes a fresh
series from format 11 to 12. No SQL version is fabricated. The baseline reader
accepts both fixtures through public inspection, Replay and Open. Independent
review verifies exact original profile bytes, eight first results, 31 history
rows, seven approvals and eight charges across promotion. Only the additional
losing receipt and expected physical-format evidence differ; both original and
losing digest preimages are checked. Full existing compatibility suites continue
to cover earlier formats. No supported version, schema, dependency, public API
or BoundedSeries behavior changes in this optimization.

## Initial immutable-source verification

Commands run from the exported candidate source, with CGO 1, local Go 1.27.3,
GOWORK=off, GOPROXY=off and pinned modules. Exact environment, command, duration,
exit and log hash are retained per command. Counts below are named passing test
events, including parent tests and subtests, not distinct top-level test functions.
Each suite has zero skipped or failing named tests; packages with no tests emit
three package-level skip records.

| Command | Exit | Named pass events |
| --- | ---: | ---: |
| `go test -json -p=3 -count=1 -timeout=20m ./...` | 0 | 1,186 |
| `go test -json -p=3 -tags integration -count=1 -timeout=30m ./...` | 0 | 2,130 |
| `go test -json -p=3 -race -tags integration -count=1 -timeout=30m ./...` | 0 | 2,130 |
| `go build ./...` | 0 | — |
| `go vet ./...` and `go vet -tags integration ./...` | 0 | — |
| `go mod verify` and `go mod tidy -diff` | 0 | — |
| Formatting and whitespace checks | 0 | — |
| Staticcheck ordinary, integration, race, integration+race | 0 | — |

The baseline ordinary and integration suites also pass (1,172 and 2,116 named
events). Both baseline Staticcheck configurations pass. Thus no known
pre-existing Staticcheck diagnostic is carried forward. One initial targeted
Staticcheck attempt failed solely because its default cache was not writable;
the unchanged-source rerun with an explicit writable cache passed. Both actual
results are retained. There are no native `func Fuzz` targets in this source;
the existing finite-model, clock and adversarial regression tests ran in the
suites. This is not reported as a fuzz campaign.

The baseline and candidate filename/content manifests match before and after
initial qualification. Final delivery qualification will additionally bind the
complete delivered tree. No timeout, interruption or cache-setup failure is
relabeled as a passing run.
