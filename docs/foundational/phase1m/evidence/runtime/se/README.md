# C_se concrete qualification

Task 3 implementation result: the eight automatic A/B answer samples satisfy
G_answer; both selected original invocation publication orders complete with
S-first parent consumption. This is bounded runtime evidence under the approved
stable pure profile, not model enumeration or universal scheduling. Independent
Task 3 review remains a separate gate.

Normal and race each pass 19 new leaves: eight positives, four controls, two
public schedules, two authentic cross-policy order negatives, two detached
raw-record negatives, and one historical-wrapper equality check over 48 archived
inspections. The affected historical Phase 1L selection passes all 39 leaves
in each profile, including its existing recovery predicate and all 19 decoder
corruptions. No new process/crash campaign was added.

The original e/S intents commit before either await. In the S-publication-first
schedule, S is actually consumed while e is still unproduced and its admitted
member/request untouched; dynamic spending at that cut is 2. The original B work
then admits/finishes its helper, reaches spending 3, and completes. In the other
schedule e publishes first, and is retained-unconsumed when S is consumed; dynamic
spending is already 3. Both cuts retain open parent PC3, original handles, no root
outcome, a checked acknowledgment and the public-operation journal. Completion
preserves old consumed rows and selects each original result once. Separate
causal fields retain member/helper terminal freeze, original publication,
delivery observation, and actual history consumption sequence/order. Current
SettleInvocation makes publication/delivery atomic; no published-undelivered
checkpoint is claimed.

Controls show candidate-pending with S consumed and e required, S-pending at the
first await with completed e unconsumed, truthful consumed wrong-id with defensive
failure plus component/G rejection, and budget2 refusal with zero refused debit.
No second request releases pending. Authentic C_es and C_se histories pass raw
validation but are rejected by the opposite consumption-policy checker. Detached
selection swapping (with coherent consumed sequence and rehashed history) and
promotion of delivery to consumption are both rejected before RawValid.

The sole shared Go change extracts decodeInvocationJoinRaw. Its validation body
through RawValid is text-identical; the historical decodeInvocationJoin wrapper
retains the same three old checker calls. wrapper-before and each final profile
produce identical complete traces/verdicts/handles on the same 48 historical raw
inspections. red-order preserves the expected initial old-checker consume_order
failure on an authentic new execution; it is construction RED, not a substitution
counterexample. No historical archive was copied or rewritten.

All final test/static command records bind 367 source paths at aggregate
ab2db041175840a7dea8c6175aeb7bc24cedcb44555d7aefad0e125f64a6ecbf.
The executed platform is Go1.27.3, darwin/arm64, CGO1. Operational limits remain
MaxSteps200/MaxDispatch200/capacity1, justified by the independently reviewed
Task 2b artifact bounds (A25/B35 commits, 6/9 admission/reply/outcome dispatches,
64KiB conservative byte bound below 1MiB caps), not inferred from these samples.

## Reproduction and retained records

Run from the repository root with the pinned local toolchain. Exact argv, cwd,
environment overrides, source hashes/snapshots, exits, leaves, stdout/stderr,
raw inspections and public journals are in the named .json.gz command records.
capture.py uses exclusive labels and will refuse to overwrite these records.
Use a fresh label or a disposable checkout for reproduction. It strips ALAN_*
and OPENAI_API_KEY, then sets GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off,
GOFLAGS=-mod=readonly, CGO_ENABLED=1, GOCACHE=/private/tmp/alan-phase1l-go-build
and a fresh temporary ALAN_I_ARTIFACTS directory.

```sh
python3 docs/foundational/phase1m/evidence/runtime/se/capture.py rerun-se-normal /opt/homebrew/bin/go test -json -tags integration -run '^TestConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)$' -count=1 -timeout=10m ./internal/substitutiontest
python3 docs/foundational/phase1m/evidence/runtime/se/capture.py rerun-se-race /opt/homebrew/bin/go test -json -race -tags integration -run '^TestConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)$' -count=1 -timeout=15m ./internal/substitutiontest
python3 docs/foundational/phase1m/evidence/runtime/se/capture.py rerun-l-normal /opt/homebrew/bin/go test -json -tags integration -run '^TestInvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes|Recovery)$' -count=1 -timeout=10m ./internal/substitutiontest
python3 docs/foundational/phase1m/evidence/runtime/se/capture.py rerun-l-race /opt/homebrew/bin/go test -json -race -tags integration -run '^TestInvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes|Recovery)$' -count=1 -timeout=15m ./internal/substitutiontest
```

The scoped integration vet, test discovery, three-file gofmt check, go mod verify,
and toolchain inspection also exit0. Formatting output is empty. Final normal/race
qualification has no failed or skipped leaves. The RED exit1 remains explicitly
recorded and excluded from that passing claim.

`audit.py` independently reconstructs original handle→commit→admission/request/
binding→publication→retained observation→actual selected receive/consumed-sequence
links. It extracts only the pinned historical audit's three inert function
definitions (descriptor, links, source_authority) through Python AST; none of that
script's top-level archive reads/writes execute. Added checks cover S-first
consumption, both cuts, distinct original publication order, unchanged consumed
rows, controls, exact old-wrapper equality and the historical process regression.
The audit does not call Go checkers/reducers and does not claim a second runtime.

```sh
python3 docs/foundational/phase1m/evidence/runtime/se/audit.py
```

This audit verifies the saved named command records, then rewrites only this
package's audit.json/index.json. Its executed command/source/output record is
`audit-command-1.json.gz`; audit passed. Source-bound files and timings are indexed
in index.json. The old archive remains at its Phase 1L location and identity.

No production/compiler/runtime, original A/B/helper/S, model/reference, manuscript
or dependency source changed in Task 3. C_reason, reason classification, all-mode
conformance and manuscript reconciliation remain later controller gates. No
C_se process-recovery claim is made.
