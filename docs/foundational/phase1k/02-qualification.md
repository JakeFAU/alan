# Phase 1K qualification — required communication and safe termination

Status: all substantive task and whole-phase reviews approved. Complete runtime
normal/race qualification is bound to the R2 source; the final fixture-isolation
correction passed focused decoder normal/race checks. Scoped final rereview approved the correction; the single end-of-phase Tectonic
build succeeded with warnings. The frozen contract is in
[01-communication-contract.md](01-communication-contract.md); independent verdicts
and preserved rejections are in [03-review.md](03-review.md).

## Question and baseline

When may a participant terminate without stranding required ordinary communication,
and which completion facts can retained public programs observe? The boundary under
study separates Send commitment, first source authorization, destination admission,
observation retention, actual selection/consumption, local terminal and aggregate
publication.

Verified baseline: `94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`, tree
`a20478fa31b333c6ff0be43fc9b43826b54bbb3a`. PR21 was merged; its reviewed head
`1d13bc238aada500c3b9608d400435e235fedc0b` has the same tree. The original J1
experimental baseline and integration base remain distinct historical identities.
PR19 formatting and PR20 mathematical correction are inherited work.

## Concrete J1 profiles and authorization pair

**Concretely tested public transitions:** the original B/J1 artifacts run at
id 1/value 0/answer, I=R3/depth2/E0, two fixed runs and physical capacity1.
The driver executes original B/helper/S invocations and replies, genuine local
instructions and actual selected observations. It does not run concurrently with
Store.Run or substitute host-computed branches/projections.

| Public ordering | Original report and source facts | Receiver/aggregate result |
| --- | --- | --- |
| LATE | Report `["peer",1,37,0]` commits at sequence37; genuine peer jump/End reaches Success at39; first authorization at39 refuses `source_closed`. Original transport admission refuses `source_denied:source_closed`; dispatch ACK links that refusal. | No report observation or consumption; root open at required Receive, aggregate pending. |
| EARLY | First authorization at37 records open/allowed; peer genuinely terminates at39 before admission. The same first record is reused for the original message. | Admission accepts; root consumes the original report at sequence3 and reaches local/aggregate Success. |

Actual spending in both cases is three dynamic invocation/created-run charges,
five total runs including the fixed pair, and zero effects. These observations
support H-auth under the fixed receiver/route/schema assumptions. They establish
legal public transition reachability and are not an automatic-runner failure.
The opt-in false Send-is-enough completion assertion is retained as expected RED.

**Automatic sample plus conditional source result:** the fresh B/J1 execution
pauses at its genuine post-Send sequence37 event. The actual retained queue has
only the original report: all three invocation outcomes are delivered, all reply
messages acknowledged, terminal routes empty; root waits without an unconsumed
report, and peer's pure continuation is ready. The runner services one internal
delivery before its next local commit. Under this exact predicate it cannot
first terminate the peer and only then authorize the report. The sample completes.
No identity search, general drain-all barrier or arbitrary-queue exclusion is claimed.

## Exact process recovery

**Concretely tested:** each of the following B/J1 cuts was achieved with an
acknowledged child checkpoint, actual SIGKILL and a different fresh-process PID,
in both corrected-source normal and race executions. Initial records remain
separately archived under their original source labels.

| Retained pre-kill predicate | Reopen profile and observed result |
| --- | --- |
| Peer open post-Send; report committed; no authorization/admission/ACK; root open; sole-report queue | Unchanged automatic Run completes. First authorization uses renewed peer epoch3 at sequence37 although that sequence's history row retains epoch2. |
| Allowed first authorization; peer genuinely terminal; no report admission; root open | Public driver reuses the byte-identical first attempt/snapshot, admits the original report and completes. |
| Refused first authorization; refused original transport admission and dispatch ACK; peer terminal | Public driver explicitly retries the same authorization/delivery identities. Original refusal/ACK remain; no report observation/consumption or aggregate completion appears. |

Root activation renews from epoch2 to3; logical generations, programs, invocation
handles, message/authorization identities, charges and permissions remain. The
refused case additionally preserves attempts, admissions, ACK links, observations,
history, intents, facts and root publications exactly across its explicit retries.
Dispatcher acknowledgment already removed that ordinary message from automatic
queue eligibility; the refused retry is deliberately labeled a public-driver action.

The process gates use safe signaling without Store reentry inside its mutex-held
hook. Parent inspection verifies committed predicates before kill; checkpoint
names alone are insufficient. Fresh-process replay leaves retained ledgers
unchanged. A finite pending observation is not a proof of infinite waiting, and
single-host process-crash samples do not prove general distributed recovery.

The raw decoder binds original intent, first authorization and its source snapshot,
admission, observation, dispatcher ACK, selected consumption and root summary.
Standalone Python audits checked the recorded instructions/authority/history and
all process cuts independently of the ALAN reducer. Independent runtime review
accepted the actual execution/cut evidence but rejected a decoder no-report
early-return bypass; both wrong-target/obligation witnesses are preserved and
the correction now runs that check on both successful return paths. Its two
malformed-report regressions reject and the genuine pre-report control passes.
Independent scoped rereview approved the correction with no new findings.

## Independent J2 result

**Bounded model-checked:** the single id1/value0/answer, I=R3/depth2/E0 K1
local/helper product contains 1,408 states and 2,886 edges. All four maximal
states were examined: two complete and two retain an open peer after ACK source
authorization is refused. Dynamic invocation/created-run charges are two or three,
with two fixed runs separate and zero effects. These are abstract charges, not
measurements from a compiled J2 deployment.

The negative path consumes the original report, commits the ACK, reaches root
local Success, then first authorizes that ACK as refused `source_closed`. The
peer waits and aggregate completion is absent. The paired complete path retains
allowed ACK authorization before root termination, then admits and consumes the
same ACK. This refutes the frozen all-schedule completion claim in the finite
product; it does not establish a concrete J2 runtime failure or an impossibility
theorem for all finite handshakes.

Every progressing edge, including edges to previously seen states, increases
the intrinsic rank. Observed ranks are 800..8541 within the declared cap23202;
there were no rank/domain/resource/lifetime/protocol violations, cycles, cutoffs
or unexpected maximal categories. Retry bookkeeping is modeled separately as
idempotent stutter. Fair service never converts refusal into permission. The
recorded path count is an abstract interleaving count, not a probability.

The [retained census](evidence/model/census-K1.json.gz) contains all four maxima
and decisive traces. The semantic RED assertion that all maxima complete exits1
as expected; it remains separate from initial implementation tests against empty
stubs. Original J1 eight-configuration results are unchanged. The small H-auth
product has eight states/eight edges/two maxima under an always-open receiver and
stable routes/schema/grants; this is an abstract source-lifetime result.

The independent oracle initially accepted two cyclic peer histories. Review
retained both witnesses; corrected source requires report Send and first report
authorization to precede the peer's ACK consumption. The current approved
reference freeze is
`7ef775362562faa6b7ac1c4a364638545160be57b75daed81577b5417e0e7868`;
the first rejected source/archive remains under `evidence/model/`, with corrected
checks separately under `evidence/model/revision2/`.

## Concrete J2 construction boundary

**Concretely tested compile obstruction:** the exact peer definition fails with
`steps/37:if/then/0:if: branch type mismatch: string and request:invocation:peer`.
Ordinary ACK receive binds a string; the original retained invocation handle
keeps its declaration-specific kind. Their required equality comparison fails.
The full attempted definitions and failed command are retained under
`evidence/context/attempted/` and `evidence/context/checks/`.

Both roots compile. All sixteen available frozen artifact hashes and the original
A/B/S identities were independently checked; 36 inherited candidate/helper mapping
entries and the available root fragments agree with their model mappings. Root
ACK commitment is at PC7. These are fragment/source checks, not complete J2
instruction correspondence. See the [independent mapping audit](evidence/model/correspondence.md).

There is no compiled J2 peer, complete Provision, J2 public-driver or automatic
trace, measured J2 spending, or observed J2 negative-root peer freeze. Desired
runtime-policy tests are preserved as unexecuted source. Passing construction
regressions establish the specific error, not those intended runtime behaviors.
The example returns an empty Provision on failure, preventing accidental fallback
to J1.

**Source-inspected scope:** Set retains the request kind. Typed payload/list
boundaries erase it to a string; a declared collection would prepend initialization
and change this frozen peer prefix. That alternative was not implemented. The
result is a precise obstruction for the chosen construction and constraints, not
universal non-expressibility of ACK protocols or a missing raw-runtime capability.
No correlation check, prefix, compiler or public facade was changed to obtain a
passing complete J2 example.

## Observation surfaces

| Surface | Result | Qualification |
| --- | --- | --- |
| Current ordinary author constructors | Send supplies no retained message-result binding; Receive matches ordinary message/local input. No constructor exposes ordinary authorization/admission/bookkeeping or a terminal wait. Await and AwaitInvocation expose their distinct runtime results. | Source-inspected constructor scope, not a theorem about all programs. |
| Raw artifact through public runtime | A two-instruction receiver matches an authentic runtime-terminal observation from a genuinely terminated peer and consumes that original observation. | Concretely tested in a separate external module with no direct internal imports. |
| Host methods/inspection | Public methods expose authorization, admission, acknowledgment, snapshots, invocation settlement and root publication. | Source-inspected methods; each concrete usage is qualified by its own experiment. A host result is not automatically program-visible. |

The terminal probe retains peer terminal sequence1, frozen receiver/done route,
accepted admission and an initially unconsumed observation, then receiver
`history.selected` and matching `consumed_sequence` at sequence1, followed by
receiver Success at sequence2. Normal/race tests and the built external executable
passed. Final-source records are in `evidence/surfaces/terminal-probe-reviewed/`.
Replay was an internal-history check and did not reissue an effect.

An authentic terminal observation is stable evidence of that source run's local
terminal outcome. It alone does not prove an earlier ordinary message was consumed.
A future protocol could relate peer successful End to prior correlated ACK
consumption and require the matching authentic terminal observation with outcome `succeeded`, but neither that complete protocol nor its
liveness argument was implemented here. Invoking the peer would instead change
topology/accounting and use the separately justified invocation-outcome path.

## Current source and validation

The R2 five runtime/test-source aggregate is
`e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1`.
Its complete focused normal and race commands both passed five qualifying
top-level tests and 27 leaf cases on identical bytes; the subprocess-entry skip
is excluded. Normal took 36.533 seconds and race took 535.773 seconds. Combined
integration-tagged vet passed. The R2-source Send-is-enough RED exits1.
Standalone audits checked 20 journals/998 genuine commits and all six new process
cuts. All 328 original runtime evidence files remain unchanged. Exact commands,
source/archive identities, selected cases, exits and hashes are in
[evidence/runtime/revision2/evidence-index.json](evidence/runtime/revision2/evidence-index.json).
The unchanged separate public-method consumer retains its earlier normal/race
visibility qualification; it is not presented as the full process campaign.

Model/oracle normal/race/vet, construction normal/race, external terminal-probe
normal/race/build/execution, applicable unchanged-core tests and module verification
also passed within their documented scopes. Historical REDs and rejected versions
remain distinct from the corrected passing source.

The final review additionally found that filtering one decoder attack skipped
sibling fixture setup and panicked. The final correction constructs genuine
fixtures lazily for each selected case and preserves copied attacks and controls.
Its exact single-leaf RED exits1 and GREEN exits0; focused decoder normal/race
commands pass 21 leaves across both decoder top-level tests (18.190 and 253.591
seconds), and focused integration-tagged vet passes. The latest five-source
aggregate is `e174c4b7ca2e256259045ecdc117378111ecbb3d3de14b0ca5f7d88affb5eb04`;
only the decoder test file changed from R2. Driver, process, runtime, evidence
decoder and oracle bytes remain unchanged. Full 27-case normal/race evidence
remains bound to R2; no full-suite rerun on this final aggregate is claimed.
All 621 prior runtime/review artifacts remain unchanged. See the
[final correction record](evidence/reviews/final-fix-report.md) and
[focused source/gate index](evidence/runtime/final-fix/evidence-index.json).

## Final gates and remaining obligations

Final scoped review approved N1 closure with no open actionable findings. The
controller ran `tectonic alan-dissertation.tex` once from `paper/` after substantive
review: exit0, warnings accepted, 18-page PDF. Pages6/13/14 were rendered with
Poppler and visually inspected; new content is legible without clipping or overlap.
The TeX source remained at its independently reviewed hash. Exact build inputs,
output/log hashes and visual-check scope are in `evidence/gates/tectonic.json` and
`evidence/gates/pdf-inspection.json`. This qualifies renderability only.

Complete J2 artifact/runtime correspondence remains unavailable for this frozen
construction.
General compiler correctness, arbitrary-context scheduling, distributed recovery
and universal protocol expressibility remain outside this bounded investigation.
