# Task 4 independent review, round 1

Review scope: C_reason construction and representative public preflight, against
assignment sections 2, 4 and 6 and task-4-brief.md. Base:
`1f71b56148f2fd6ef41261b7caf2996e8eeb2533`. Read-only implementation review;
no subagents, commits, production edits, model-source inspection or test reruns.
The Go orchestration, code-style and testing skills informed the code/test review.

Question: does the ordinary owning parent classify the actual original envelopes
only after candidate-then-S commitments and S-then-candidate consumptions, while
preserving original components, correlation, guarded payload access and resource
scope?

## Verdicts

- **Specification: APPROVE for Task 4 bounded construction/preflight.** The authored
  and compiled parent realizes the required policy. This is not approval of the
  independent model, G_outcome census, all-mode conformance or correspondence.
- **Quality/reporting: REJECT pending finding R1 below.** No actionable Go-code,
  original-identity or captured-execution defect was found. The bound explanation
  repeats a previously corrected causal error.
- **Overall Task 4: REJECT pending the reporting correction only.** Preserve the
  finding and record its disposition. A report-only correction does not require
  rerunning the passing suites or recapturing source-bound preflight artifacts.

## R1 — [P2] Correct the causal derivation of the dispatch bound

Location: `.superpowers/sdd/00-plan-phase1m/task-4-report.md:100–101`.
The report says “At most one invocation, request and reply dispatch per admitted
singleton gives 6/9”. Request insertion is not a separate counted dispatch.
`internal/executor/invocation.go:480–497` builds/inserts the request observation and
request row in the admission transaction. `internal/executor/runner.go:152–159`
counts protocol service; `runner.go:385–399` schedules an invocation initially
for admission and later when its outcome is ready, omitting admitted unready
invocations and already delivered outcomes. Task 2b explicitly preserved this
correction in `.superpowers/sdd/00-plan-phase1m/task-2b-report.md:41–47`.

Required correction: explain the three possible counted dispatches per original
admitted singleton as **initial invocation admission (including atomic request
observation insertion), original reply-message delivery, and ready original
outcome settlement/delivery**. Under the declared stable eligible service, this
still yields upper bounds A+S = 6 and B+helper+S = 9. A failure/absence can omit
reply delivery, and pending can omit outcome settlement; root settlement adds no
report.Dispatches charge. The existing numeric bounds and captured results do not
need changing. This matters because the research evidence must preserve actual
admission/request/reply/publication/delivery distinctions, rather than treating
a correct total as validation of an incorrect causal account.

## Specification and implementation checks

- `examples/consumingpolicy/reason.go:85–90` commits the original candidate handle,
  then original S handle, before either await, then consumes S and candidate using
  those handles. Compiled PCs 0/1 are invocation emits and 2/3 are matching
  receives, with runtime-invocation provenance, original template source,
  generation, obligation and operation correlation. S receives request_id and
  original request_value, not the candidate response.
- `reason.go:69–80` dominates candidate interpretation with S status succeeded,
  reason member_succeeded, presence true, original id and value zero. Presence
  precedes either S payload read. The expected complement is calculated only
  after these guards; both joins already completed.
- `reason.go:51–66,75` accepts exactly succeeded/member_succeeded/present/correct
  id/complement as answer; failed/absent/member_failed as candidate_failed; and
  failed/absent/missing_reply as candidate_missing_reply. All other completed
  evidence, including invalid S, reaches invalid_outcome and Failure. Candidate
  payload access is dominated by its success, reason and presence guards. The
  retained candidate_reason is never overwritten. Request mode is passed to the
  candidate at line 86 and never used by a parent branch.
- `reason.go:39–47` emits ordinary committed Set then Success/Failure instructions.
  There is no host classification, report Send, extra parent/invocation, retry,
  fallback or runtime primitive. All terminal paths follow the two receives.
  Compiled parent has 52 instructions / 21703 bytes for each binding. Answer uses
  Set/end PCs 18/19, candidate_failed 31/32 and candidate_missing_reply 34/35;
  all defensive exits also set invalid_outcome before end. All targets are forward.
- Pending cannot pass the required receive or reach any classifier/end. The
  public archive shows A/B candidate pending at PC 3, S consumed, candidate
  unpublished, no classification/end/root publication. Existing captured normal
  and race integration leaves additionally cover both S-pending directions and
  S failure with candidate pending (`consumer/reason_integration_test.go:76–132`).
- `reason.go:95–112` provides a distinct named control with exactly Receive then
  Failure. An absent request genuinely yields member_failed and candidate_failed;
  ordinary A/B absent yields missing_reply and candidate_missing_reply. This is
  an executable component mutation, not decoder corruption or a conforming K1
  absent case. Actual component/profile rejection remains explicitly deferred.
- Public histories bind classification/end to original selected observations
  (`consumer/reason_integration_test.go:134–198`, capture.py:115–137,244–274).
  A/B absent use setter PC 34 then end PC 35; deliberate-failure absent uses
  31 then 32. Their raw child publications retain missing_reply/member_failed
  respectively, while each parent/root publication is failed/member_failed.
  An additional review-only check evaluated all 97 captured branch comparisons
  from retained predecessor operands and confirmed each actual successor target.
  This strengthens branch-evidence inspection; it is not independent compiler
  correctness or a new reference model.

## Reuse, identity and scope

`NewReason` reuses author-time NewSFirst template selection/config validation,
replaces its unused compiled parent and provision, then rebuilds resource roots
from the new catalogue (`reason.go:13–36`). This does compile unused historical
construction, but is transparent minimal reuse within this additive slice and
runs no historical provision. One fixed root, original common limits and zero
effects remain. A/B parent differences are exactly candidate emit target,
candidate receive source and author candidate TemplateID; B also brings its
original helper catalogue closure. This is an independently authored parent
relative to the separately delegated model, with shared public compiler/runtime
and capture infrastructure, not independent compiler/runtime validation.

Review independently compared all 21 protected source/evidence paths to git base
bytes, including historical C_se builder/tests/preflight, original A/B/helper/S
sources and historical invocation-join builders. Every comparison matched.
All 127 current captured source hashes match the manifest, and all 126 base64
snapshots hash to that manifest; historical gzip is hash-bound. Captured builder
normal/race tests preserve C_se IDs and candidate-only rebinding. No original
component, runtime/compiler, manuscript or historical evidence change appears in
the actual Task 4 worktree delta. The README accurately limits construction
independence and marks future qualification separately.

The public consumer uses only public imports and exposes the reason policy plus
the named control (`consumer/main.go:31–32,46–71`); normal run/recover still uses
MaxSteps 200, MaxDispatch 200 and MaxInFlight 1 at line 95. The capture keeps exact
artifacts, commands, public outputs, complete raw tables and actual selected
history rather than synthesizing expected handles. Expectations are recorded in
a separate preflight file; their pre-execution provenance and withholding from
the model author are author/process statements, not facts established by archive
content alone. No model outputs were inspected during this review.

Bounds are appropriately partial at this gate: compiled forward path sums give
A answer 28 and B conservative 38 (observed B 35), versus 200 steps. Dispatch
upper 6/9 is sound with R1's corrected causal explanation. Max sampled projection,
observation and output-array bytes are 1252/797/537; these are explicitly measured
construction samples, not an all-input or arbitrary-template proof. The report
keeps full-domain byte/progress/resource qualification, artifact correspondence,
all-mode local conformance and synchronized manuscript reconciliation open. The
Task 4 gate must not silently promote these partial bounds to phase completion.

## Evidence examined and commands actually run for this review

Read assignment sections 2/4/6, relevant remaining phase obligations, applicable
AGENTS.md, task brief/report/diff, parent/builder/public consumer/tests, capture,
expectations, archived evidence and the preserved Task 2b dispatch correction.
Used `rg`, `nl`/`sed`, `git status --short`, and scoped `git diff --stat` for actual
scope; read runner/admission sources directly. Used review-only Python checks to
load/decompress existing records, recompute SHA256/snapshot/base comparisons,
inspect compiled instructions, count archived test leaves, extract joins and
setter/end sequences, and validate 97 actual recorded branch targets. These
checks made no runtime calls and did not rerun passing suites. No focused runtime
probe was necessary because the remaining doubt was a source-verifiable report
claim.

Verified archive SHA256:
`00a56416238d858dfcc6a46a378fec7c27a8f571d2b2122f384161c32275a22c`.
Verified source-manifest SHA256:
`f3cf8a55ca257230d54217655ae6097540c3ef42ae606051e34f9b13a34ef8ed`.
Archive status passed; 81 recorded commands, all exit 0; 13 public samples.
Archived normal/race builder runs each contain 8 passing leaves; public consumer
integration runs each contain 29 passing leaves, with no failing/skipped leaves.
These are attributed to the inspected Task 4 archive, not newly executed tests.

Representative raw evidence: both commitments are parent sequences 1/2 and both
consumptions are 3/4 (S/candidate). A/B answer setter/end are sequences 18/19;
fail 17/18; absent 18/19; deliberate-failure absent 17/18. S-fail/S-absent
classify only at sequences 6/7 after both consumptions. Pending samples have no
write/end. This confirms the recorded construction behavior and authentic child
versus root reasons, within the archive's stated domain.

Only this review file was written. Remaining obligations are the report correction
R1, independent reason reference review/freeze, full audited answer inputs,
negative component/profile rejection, runtime-model correspondence, all-mode
local/progress/resource qualification and final manuscript reconciliation.


## R1 scoped re-review disposition — correction accepted

The original specification APPROVE and quality/overall REJECT above are preserved
verbatim as historical review evidence. This disposition supersedes their pending
status only for the bounded Task 4 construction/preflight gate.

Read `task-4-fix-r1.md`, its exact one-hunk diff, and the corrected report. Verified
that reversing only the dispatch paragraph reproduces the recorded original
report SHA256 `d6841471541b70477cb36a1b27416081beec075c5b9b4247ba064fd5141c37ea`;
corrected report SHA256 is
`ce01da78bcb59aabf736dcf3d38dc90d6c910934afad5a5a5812ea3ac491267b`.
Thus the correction has exactly the requested scope. The original review SHA256
before this append was
`b67ec1a703abdeb356c2a11ff89b58409cfe8042c5bc83eaa691ffd5dc0396b1`.

The corrected `task-4-report.md:100–107` explicitly distinguishes initial
admission with atomic request insertion, original reply-message delivery, and
ready outcome settlement/delivery. It preserves the valid 6/9 upper bounds and
correctly explains omitted reply/outcome service for failure/absence/pending and
uncounted root settlement. Re-read cited executor source: invocation.go:480–497,
runner.go:268–288,385–399,241–245. Those citations support the correction; no
directly introduced issue was found. **R1 is RESOLVED.**

Bounded read-only hash checks reconfirmed the unchanged archive digest, all 127
current manifest-bound source files and all 126 archived snapshots. No code edit,
suite, consumer/model run, recapture, commit or subagent was performed. Only this
append was written for re-review.

- **Specification: APPROVE for bounded Task 4 construction/preflight.**
- **Quality/reporting: APPROVE for the reviewed construction/preflight scope.**
- **Overall Task 4: APPROVE.**

No new finding. This does not qualify the independent reason reference,
G_outcome census, all-mode component conformance, negative profile rejection,
runtime-model correspondence or phase-wide bounds/manuscript obligations. Those
remain separate gates as stated in the original review and corrected report.
