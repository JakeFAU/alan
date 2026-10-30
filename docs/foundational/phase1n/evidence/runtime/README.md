# Phase 1N actual runtime qualification

This folder contains additive Task4 runtime evidence. It does not rerun the model
census or change the approved J/O construction, original A/B/helper/S, production
executor, historical decoders, or historical archives. Reference and finite
correspondence evidence remain separately identified under `../model/`.

## Reproduction and capture format

`capture.py LABEL ARGV...` creates an exclusive directory. Every command retains
its exact argv/cwd, sanitized environment, Go toolchain/effective environment,
stdout/stderr/exit, selected leaves, raw exported records, all Go/module source
hashes and a complete source tar archive. It rejects label reuse. New copies
must use fresh labels or a disposable checkout. ALAN_* and OPENAI_API_KEY are
removed, then ALAN_I_ARTIFACTS is set to a fresh capture directory. No provider
calls are used. Modules are readonly; proxy and sumdb are disabled.

The focused commands are:

```sh
python3 docs/foundational/phase1n/evidence/runtime/capture.py rerun-normal /opt/homebrew/bin/go test -json -tags integration -run '^TestCallableJoin(Runtime|Modes|Controls|LateReply|Decoder|Prefixes|ControlIdentities)$' -count=1 -timeout=5m ./internal/substitutiontest ./examples/callablejoin
python3 docs/foundational/phase1n/evidence/runtime/capture.py rerun-race /opt/homebrew/bin/go test -json -race -tags integration -run '^TestCallableJoin(Runtime|Modes|Controls|LateReply|Decoder|Prefixes|ControlIdentities)$' -count=1 -timeout=10m ./internal/substitutiontest ./examples/callablejoin
python3 docs/foundational/phase1n/evidence/runtime/capture.py rerun-historical /opt/homebrew/bin/go test -json -tags integration -run '^Test(ConsumerReason(Runtime|Modes|Controls|Decoder|ClassificationCut)|ConsumerPolicy(Runtime|Controls|Orders|Decoder|HistoricalWrapper)|InvocationJoin(Runtime|Controls|Orders|Decoder|IncompletePrefixes))$' -count=1 -timeout=5m ./internal/substitutiontest
```

The historical race selection adds `-race` and uses an8m watchdog. It executes
the existing checks and records their outputs here; it never rewrites historical
capture destinations. The historical wrapper compares48 archived inspections
and their full trace hashes. Discovery uses the same anchored Task4 test names.
No broad root test or new crash/recovery cross-product ran for Task4.

`normal/` and `race/` are the final Task4 sample captures; `historical-normal/`
and `historical-race/` are the unchanged M/L selections. Their actual statuses,
leaf counts and hashes are indexed in `index.json`. `audit-final/` contains the
independent audit of both final sample captures; no Go classifier/reducer runs
inside that audit. Controller static verification is separately retained as
`../core/final-static.json.gz`.

## Actual boundary and mapping

The new Go decoder derives original j/e/S/helper handles from emitted intents
and their committed destination registers, then follows actual admission,
template/binding/request, observation, selected history and consumed_sequence
links. It does not create handles from expected tuples or call the old root-PC
decoder. Historical common row/identity helpers and the stable structural
`validateRetained` checks are reused explicitly. The additive N authentication
pass follows the historical record-link algorithm with new topology, depths,
report payload and public O mapping; that shared infrastructure limits
independence.

Every supplied program has an explicit exported PC/instruction/source-position
mapping. Exact step checking authenticates all executed original component and
J/O instructions, including intermediate list and cursor state. Both positive J
programs have58 PCs and both O programs64. Control J programs have7,7,33 PCs;
mode disagreement rebinds the58-PC positive policy. The pure branches, jumps,
list construction and cursor operations are checked stutters in the abstract
projection, not ignored evidence. J's classification Set expands to classification
and immutable-tag ghost decision at one order; O's decision is an actual Set.
Neither expansion invents a resumable internal checkpoint.

Invoke+handle+intent, selected Receive+binding+consumption, and End+freeze are
atomic groups. Publication+outcome delivery share a settlement transaction.
Classification, Send, source authorization, reply admission, End, publication,
selected observation and outer decision remain distinct facts. The raw audit
independently walks exact instructions, including cursor snapshot/index/removal,
then compares actual selected evidence, report Send and O decision. Program
digests are explicitly pinned for positive and separately named control artifacts.

The supported B/absent schedule commits J's candidate_missing_reply Send,
executes its branch and Failure, verifies the frozen reply is awaiting admission
with no prior authorization, then first authorizes and admits that original reply.
Its observed fact orders are Send65, End execution67/freeze68, first
authorization69, reply admission70, J publication/delivery71 and O consumption72.
Authorization records the failed source authority, not a pre-End permit. The
exception is the existing original entry-member reply binding to the actual
caller/generation/work obligation with stable revision/generation. Invocation
authorization receives no closed-source exception. All calls are sequential
public Store methods; no hooks, sleeps or authoritative-state writes are used.

## Samples, controls and interpretation

Each final normal/race profile includes eight automatic answers (A/B × id1/2 ×
value0/1), six representative A/B fail/absent/pending cases, one supported
late-reply case, four genuine controls, seven detached corruption cases, two
incomplete-prefix cases, and one control-construction identity leaf. That is29
test leaves:28 runtime/decoder/prefix leaves and one construction leaf. Runtime
samples are not the full finite configuration census or all-schedule coverage.

| Genuine control | Raw authentication and O public policy | Report truth/K_join | Component/profile |
| --- | --- | --- | --- |
| Valid children, no report, J Success | pass; O commits invalid_join_outcome from failed/missing_reply/absent | completed diagnostic requirement fails | children conform; constructor fails |
| Valid children, no report, J Failure | pass; O commits invalid_join_outcome from failed/member_failed/absent | completed diagnostic requirement fails | children conform; constructor fails |
| Actual candidate missing_reply, false candidate_failed report | pass; O faithfully commits candidate_failed | truthful classification/report and K_join fail | candidate conforms; constructor fails |
| Historical DeliberateFailure under absent request | pass; truthful candidate_failed and matching O policy | observation truth passes; mode-specific K_join not discharged | K1 mode conformance fails |

The outer completion predicate is false for these controls, with the conforming
K_join premise absent. They are not conforming substitution counterexamples.
The semantic lie is an authored genuine execution, not a detached payload edit.
O never observes J's private child rows. Independent truth checking does.

Detached attacks change original J handle, selected J observation,
classification Set, report successor, J branch, O branch, or a cursor snapshot.
History fact hashes are recomputed so semantic checks face the changes. All are
separate from genuine controls. The classification-only and Send-only prefixes
retain valid raw/policy prefixes but fail claimed End/publication/outer completion.
Neither a final-looking register nor Send commitment is publication evidence.

Pending samples retain the byte-identified original unmatched candidate receive,
B's real helper consumption when applicable, S consumed by J, J at original e
await PC5, and O at original J await PC1. They have no diagnostic, J publication,
O decision or root publication. No second request/release is supplied. Runtime
quiescence is sample evidence; exhaustive maximality belongs to the separately
qualified finite model.

## Bounds and source-stage distinctions

All actual automatic runs use512 steps,64 dispatches, capacity1 and common
I=R4/depth3/E0/fixed1. Positive artifact ceilings remain A139/B149 local commits
and9/12 protocol dispatches. Actual A/B charges are3/4, with e/S at depth2 and
B helper at3. Controls have their own static-plus-one ceilings: omission88,
false-report124, mode-disagreement128. The single possible repeated O next
instruction accounts for the added one. No bound is inferred from model rank.

Supported-driver captures have no fabricated automatic RunReport: report is null
and actual committed histories/protocol transitions are counted separately.
Every main result retains complete raw records, static mapping, identities,
bounds and layer-separated verdicts. Observed normal sample maxima are program
25,387 bytes, projection1,548, observation880, intent array605, and diagnostic312.
These are observed maxima, not replacements for the approved conservative bounds.

`red-controls/` preserves an intentional build failure for the initially absent
NewControl API; it is not a runtime semantic failure. `implementation-1/` passed
20 leaves before detached attacks; `implementation-2/` passed29 before formatting
and the final static-map/export additions. Each has its own source snapshot.
`audit-attempt1/` passed against implementation-2; `audit-normal/` passed against
normal. These earlier captures are preserved and are not attributed to later
source bytes. The first sandboxed capture attempt failed to create its output
directory before execution; the authorized scoped capture then succeeded. No
runtime or production obstruction was encountered.

Source-level-independent Python checks still share schema/protocol definitions,
retained artifact identities and the existing runtime format with Go. Replay is
shared-runtime consistency evidence. No universal compiler/scheduler theorem,
arbitrary-context congruence, new recovery claim or unrestricted weak K1 lifting
claim follows. Independent Task4 review and manuscript reconciliation remain
controller gates.
