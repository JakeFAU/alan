# Actual probe correspondence correction

The P2 finding is corrected at the original frozen profile bounds. The earlier
B-derived surrogate and weak-helper complement response are preserved under
`docs/foundational/phase1i/evidence/model/pre-probe-review-fix/`. No contract,
entitlement, public artifact source, runtime implementation, A/B source, or
frozen K1 CheckTrace oracle was changed.

## Actual transcription and independent evidence

`probe_graph.go` now hand-transcribes six actual retained programs: W_A5,
W_B8, P8, Q9, constant-zero helper4, complement helper7 =41 instructions.
Public constructor byte extraction is preserved as `probe-artifacts/extract.go.txt`;
it constructs frozen template bytes without executing programs or deriving
expected outputs from a reducer. JSON decoding and artifact construction are
explicit trust boundaries, not a compiler proof. `correspondence.json` binds
actual public source hash, four templates, six program byte hashes, every
instruction and operand, both helper branch arms, original `helper_handle`
consumption and all receive provenance/generation/obligation guards.

Q's additional `scratch=0` Set has a distinct model transition. A private bit
records definedness of zero, and canonical instruction locations align with P
only after accounting for the insertion. No environment service is postponed:
local weak-relation matching continues to require direct bilateral service at
every instruction boundary. No fabricated success-with-absent helper observation
is in the declared helper environment; actual P/Q contain no presence or mode
branch, and none is added to their transcription.

Direct regressions first failed RED: old weak helper emitted1 rather than0;
old P surrogate reachedPC5 afterPC4, while actual P jumps to Send atPC6. The
preserved RED log is `probe-artifacts/regression-red.log`. New mapping tests check
all41 instructions, opcode/control coverage, both helper branch arms, exact
receive guards and source hashes. The value1 helper arm is an instruction-level
structural test only; it does not broaden the fixed input0 product census.
A separate mutated-helper test requires the explicit weak profile monitor to
reject helper1. K1 CheckTrace stays unchanged, with its original SHA256.

## Qualified results and scope

All three local weak relations run **before** any contextual language test.
For capacities1/2/3, original-protocol local paired states are58/123/160,
checks142/328/472, zero failures. Exact original-protocol parallel trace-language
comparison has124/223/231 paired subsets and224/429/474 labeled transitions,
with equality in all three cases. P/Q state counts are136/142,273/284,297/308.
These equal the old surrogate counts only coincidentally: actual P removes one
successful entry instruction and its input0 helper adds one branch/join step.
Equal counts are not evidence of artifact correspondence.

These local/language results concern the **original declared protocol only**.
No runtime-boundary trace-language equality is claimed. `CheckProbeContext`
separately checks maximal-outcome membership, optionally with `boundary=true`
for the inspected root-terminal/peer-lifecycle extension. The runtime workstream
uses that API for actual weak8/PQ16 samples; its executions remain separate.

Actual weak state/edge counts, W_A then W_B:
- I=R1:19/19 and15/14, both fail.
- I=R2:24/25 and30/31, W_A succeeds and W_B fails S admission.
- I=R3:24/25 and35/37, both succeed.
- I3/R2:24/25 and30/31, W_A succeeds and W_B fails.
Both actual weak helper and candidate reply0. The explicit weak monitor is
separate from K1's complement oracle. Every listed positive probe has zero
stuck/rank failures/cutoffs; original K1/A/B/boundary full censuses remain unchanged.
The eight parallel completion failures and76 unaffected entitled completions
per graph are neither removed nor repaired by this correction.

## Verification, files and remaining review

Focused entire model package PASS5.965s, `go vet ./research/substitution` exit0,
`gofmt -l research/substitution/*.go` empty. Full exact commands/source hashes are
in `probe-artifacts/final-source-manifest.json`; current top-level model manifest
points to the same source. Logs and new probe outputs are under probe-artifacts;
old sources/results/mapping are preserved under pre-probe-review-fix.

Changed model sources: probes.go, probes_test.go, new probe_graph.go,
probe_graph_test.go, probe_correspondence_test.go, README. New exact membership
API: `CheckProbeContext(Config, profile string, ContextObservation, boundary bool)
(ProbeResult,error)`. Profiles weak-a/weak-b require sequential; p/q parallel;
all require id1/value0/answer/depth2 and frozen capacities, including weak I3/R2.
No production oracle imports, paid calls, source A/B/runtime/contract edits,
subagents or commits. Independent final reviewer rereview remains required.
