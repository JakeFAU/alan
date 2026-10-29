# Phase 1J final independent review v1

**Verdict: APPROVE for the bounded research result and draft-PR preparation.**
There are no unresolved actionable source or claim findings in this review.
Approval does not assert all-schedule completion: simple J1 has the preserved
report-source-closure counterexample. Final evidence-index refresh, the requested
single TeX compilation, commit identity and draft-PR publication remain controller
completion gates; none is reported here as already completed.

Reviewer: independent `final_review` agent, 2026-10-29. No implementation/model
edits, paid calls, new process campaign, commits, subagents or TeX compilation
were performed by this reviewer. The reviewer read the complete source/top-document/
manuscript change represented by `.superpowers/sdd/00-plan/final-review.diff`,
plus the model README, relevant unchanged runtime/model/validator/process sources,
workstream reports, frozen artifacts, raw records and review lineage.

## Reviewed identity and authority

- Experimental baseline: `d68ec667d276a4bd9e1be11613905c628cfea704`.
- Integration base: `5764ec0b5679c72c80e56c44fd9e4198c986b91c`. The intervening
  tracked diff is confined to `.gitignore` and the separate manuscript correction.
- Frozen J1: SHA256 `a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81`.
- Frozen plan: SHA256 `f773775b5ac65a21d63180eec7862c703590f6ea6c279d6669c9d58be9f506dd`.
- All 151 source-manifest entries and its independently recomputed aggregate
  matched: `55260ec410e9b63393b5c24a211bdb70866228a8edb75d2ec4bc28345af8da3e`.
- The 530 evidence-index entries and all qualification-record references matched
  when checked before the metadata correction below. Final index refresh must
  incorporate that correction and this review; this is not approval of stale
  final metadata.
- Reviewed Markdown manuscript SHA256:
  `6e77aa02aa151f5457a478e8d17cd278ce35d47c2cbea575639844cc55c4f231`.
- Reviewed LaTeX manuscript SHA256:
  `c86d0c79d5337fb6dc038c2deba286616f5ad1a6d4bd6e966ec9feb865fc844d`.

The prior context quality **REJECT** remains authoritative historical evidence,
followed by the independently approved ordering correction in
`context-rereview-v2.md`. This review does not overwrite or reinterpret that
failure. The reference-freeze source bytes match their recorded hashes. The
post-artifact, pre-runtime-output oracle strengthening is separately disclosed
in `oracle-review-correction.json`; its older bytes and REDs remain retained.
These records establish inspectable provenance, not a proof of compiler semantics
or an independently observable history of the authors' knowledge.

## Resolved review finding

**FJ1 — inaccurate recorded runtime output environment (resolved).**
`evidence/runtime/execution.json` and its duplicate run records in `index.json`
originally recorded relative `ALAN_I_ARTIFACTS` values. The retained logs show
absolute values, and `retain` does not resolve relative paths against the repository
root. The controller corrected the actual environment values, recorded invocation
cwd, and preserved the prior hashes/reason in
`runtime-path-metadata-correction.json`. I checked all four corrected paths against
the archived log lines and checked that the duplicated run records are equal.
No source, executed result, raw record or scientific conclusion changed.

Verified corrected execution record SHA256:
`a3031ffee5c817a3825f63dea62aedcc91c1de32dfb157e52795e470511a7669`;
corrected runtime index:
`8f0b608e0c5424b04b8a7a0ca8ab4fb139a729d3bf1a90e9684271c62c43c3cd`;
correction record:
`75263e9d1a71b36a8871698b7400561a3e816c87ee64b80e78193038e7aec58f`.

## Substantive assessment

**Intervention and strict joining.** The original candidate and S templates,
root Invoke/Await prefix, peer's 32 Sets and original S Invoke/Await, static
root/peer topology and primary resource limits are preserved. The only new
application behavior is ordinary report wiring and the permitted root/peer
suffixes. Absent response fields are omitted by Await and reported with explicit
non-asserting zero fillers. Root checks candidate and S success conditions only
after consuming both outcomes. Known failure still joins; pending remains open.
Wrong request parks after consuming an invalid assertion. The designated peer's
ordinary message is trusted application evidence, not a transferred receipt.
Peer local Success means its reporting program finished; aggregate success is
checked separately. No production scheduler/compiler/schema/accounting change
is present.

**Model and correspondence.** The model keeps normal terminal transitions,
committed but unadmitted demand, optional-helper branch exclusion, root terminal
admission fencing, separate outcome publication/delivery/consumption, and separate
report commitment/authorization/delivery/consumption. Its positive scope is exactly
eight configurations per K1/A/B graph, not arbitrary contexts. All 32 delay Sets
are explicit. The inspected compiled root/peer suffixes are finite and forward;
the JSON evaluator checks their outcome decisions and exact report construction
independently of the runtime/compiler, and the full artifact mapping retains all
136 instructions. Finite pure suffix stuttering does not introduce protected work
or move a terminal before its required receive/report.

A separately written breadth-first graph collection and Kahn topological traversal
verified the explorer without using its rank as the topological oracle. It reproduced
all state/edge/maximal counts and exact maximal-path counts for all 24 configurations:

| Product | States | Edges | Maximal success | Maximal pending | Longest abstract path |
| --- | ---: | ---: | ---: | ---: | ---: |
| K1 | 12,320 | 25,648 | 16 | 16 | 67 |
| A | 6,160 | 12,620 | 8 | 8 | 60–61 |
| B | 11,260 | 23,124 | 8 | 8 | 72–73 |

Every collected node was removed by the topological traversal: no graph cycles
were found independently of the implementation's rank check. This is a check of
the census over the same transition generator, not a second independent semantic
model. Model-source inspection and the preserved rank checks support the bounded
lifetime/resource claims. All pending maxima in this product have consumed
candidate/S, terminal peer, refused report, open root and no aggregate publication.

The negative has a concrete semantic basis in `AuthorizeIntent`: first ordinary
source authorization after local terminal is `source_closed`; the composition-reply
exception does not cover the join port. A fair attempt can be refused. The automatic
runner dispatches internal work before further local steps, so its successful
samples are compatible with this broader interleaving negative. The manuscript
and qualification make that distinction explicitly; no concrete automatic-runner
report-refusal witness is claimed.

**Raw adapter and controls.** The new adapter extends the existing retained-record
validator at ordinary reports. It follows original operation IDs, selected
observation IDs, per-actor history sequences, original receive guards and bound
payloads. It does not replace consumed observations with final registers. It
links report authorization, delivery and acknowledgment records, and reconciles
actual spending with accepted admissions. The oracle then checks S-consumed-before-
report, exact reported outcome/handle, correlation, report identity, candidate-
before-report consumption and final root decision. This is a bounded apparatus
with explicit static-generation assumptions, not a general corruption verifier.

A separate Python inspection of the archived v2 raw records confirmed all 13
matrix cases and all four C0 records. For ordinary truthful reports, the original
S outcome is consumed at peer sequence34 and the report is committed at sequence37.
The premature mutant instead reports at41 before actual S consumption at43 and
can produce root/aggregate success: the oracle correctly rejects its contract.
The wrong-request report is delivered/consumed but the root stays open. Both
primary cases contain the decisive candidate-consumed/S-unadmitted/root-open
state and finish with dynamic charges A=(2,2,0), B=(3,3,0). Both capacity refusals,
known absence/failure, and strict pending controls retain their actual statuses
and charges. C0 A/B at capacities3/4 retain scope_closed and failed/member_failed.

**Process evidence.** Independently inspected all four v2 ACK/before/after/exit
record groups. Candidate-consumed cuts are root sequence2 before S admission;
report-committed cuts are peer sequence37 with no root report consumption.
Each recorded acknowledged PID actually exits by SIGKILL, two distinct later
PIDs complete resume and replay, root epochs renew2→3, original records remain
an immutable prefix, and the resumed root/peer/aggregate succeed. Prefix checks
include programs, runs/generations, initial projections, operations, admissions,
report-if-present, observations, history and charges. This verifies retained
single-host process samples; it does not establish power-loss, distributed or
all-schedule recovery.

**Claims and validation lineage.** Markdown/LaTeX additions agree in mathematical
content and preserve the separately inherited structured-loop correction. No
citation change requires a BibTeX edit. The preserved v1 full race campaign is
not relabeled as v2: v2 has full normal36 leaves and focused runtime/decoder race26;
unchanged process machinery has the earlier full race coverage plus current
normal reruns. The correction is sequential oracle logic and does not create a
new concurrency boundary. This coverage statement is accurate; this review does
not claim a latest-oracle full race run or TeX success.

## Reviewer checks actually executed

All reviewer Go commands used Go1.27.3 with `GOTOOLCHAIN=local GOPROXY=off
GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1j-go-build`, and removed
`OPENAI_API_KEY` from the child environment.

1. `go test -v -count=1 -timeout=10m -run '^(TestReportAuthorizationCounterexample|TestConsumptionDoesNotTerminate|TestContextInstructionCorrespondence|TestOriginalFrozenComponentHashes|TestCorrespondenceManifest|TestOracleAdverseReports|TestOracleIndependentActorOrdering)$' ./research/join` — exit0, package0.228s.
2. `go test -tags integration -v -count=1 -timeout=10m -run '^TestJoinDecoder$' ./internal/substitutiontest` — exit0, all13 mutation leaves pass, package0.407s.
3. `go run /tmp/phase1j-independent-graph.go` — exit0; independent counts/topology/path results above. Exact script retained as `final-review-v1-graph.go.txt`.
4. `python3 /tmp/phase1j-independent-raw.py` — exit0; all13 matrix, four C0 and four process groups checked. Exact script retained as `final-review-v1-raw.py`.
5. Read-only Python hash checks verified the source/evidence/qualification manifests,
   reference-freeze bytes, manuscript hashes and corrected runtime record paths.
   An initial reviewer helper mistyped the manuscript relative path after finishing
   the manifest checks; the subsequent corrected direct paths verified both paper
   hashes. That helper error is not a repository or qualification failure.

To reproduce the additional reviewer graph check, copy the retained `.go.txt`
script to a temporary `.go` file and run it from repository root with the environment
above. Run the retained Python script from repository root directly. Neither
probe writes authoritative application state or changes the transition generator.
The larger existing campaigns were inspected in their source-bound archived
records rather than gratuitously rerun.

## Remaining obligations and publication boundary

The all-schedule completion hypothesis for simple J1 is **falsified** in the declared
finite interleaving model. A sufficient lifetime condition for required ordinary
communication's source authorization, or another explicitly specified ordinary
protocol, remains open. No universal impossibility, weakest lifetime theorem,
compiler-correctness proof, arbitrary-context/exact-debit congruence, general
confluence, distributed recovery or effectful/unknown-outcome repair is established.

The controller must refresh final hashes/index entries, record the actual one-shot
TeX result, preserve the prior review dispositions, inspect the final staged scope,
and create the authorized draft PR without merging. Those mechanical completion
gates do not require silently promoting the bounded research result to a stronger
claim. New substantive source or claim changes would require further review.
