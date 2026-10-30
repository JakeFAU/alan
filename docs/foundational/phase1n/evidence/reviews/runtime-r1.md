Specification verdict: APPROVE.

Code/evidence quality verdict: APPROVE.

## Scope and method

Independent Task4 review against base `cb678321324437d9d5c7247c8a139fc5dc401201`.
The complete 3,618-line scoped diff was read once, SHA256
`74349822773f46e16554d94408a2f36c23de6c66613aa1cd3105c8d9b28c3639`.
The task brief, resume, global constraints, and implementer report were read;
the report was treated as unverified claims. The task-reviewer prompt and Go
testing Review-mode guidance governed this review. Constructor, reference and
qualification reviews remain approved inputs; this review does not reopen them.

Only named concrete risks justified inspecting unchanged helpers, as recorded
below. No Go suite, race suite, model census, construction preflight, or Python
audit was rerun. Reviewer execution consisted of read-only `sed`/`rg` inspection
and Python decompression, hashing, archive/source comparison, saved-row
inspection, and independent metric recalculation. No production, historical,
model, manuscript, dependency, or capture file was changed by this reviewer.

## Specification checks and strengths

- Separate genuine controls preserve their actual identities and public reply
  schema (`examples/callablejoin/controls.go:12`,
  `examples/callablejoin/controls_test.go:12`). The false-report program guards
  on actual consumed missing-reply evidence and valid S before its deliberately
  false Set/Send. The mode-disagreement fixture reuses historical
  `DeliberateFailure`, with O rebuilt only for the actual J template
  (`internal/substitutiontest/callable_join_runtime_test.go:23`).
- New raw mapping authenticates supplied artifact bytes, original commitments,
  source revisions, admissions/bindings, original selected observations, and
  emitted handles. Roles come from the emitting instruction and its destination
  register; handles are not reconstructed from an expected tuple
  (`internal/substitutiontest/callable_join_evidence_test.go:44`, `:510`, `:527`).
  Report, classification and O decision evidence refer to their actual committed
  steps and original consumptions (`:549`, `:613`, `:634`).
- Every static PC receives an explicit mapping and source position, and the
  complete fixed instruction subset is checked against actual pre/post states.
  O cursor snapshot, row binding/index advance, loop return and exhaustion are
  authenticated, including intermediate commits
  (`internal/substitutiontest/callable_join_instruction_test.go:23`, `:83`).
  J's classification/ghost-decision expansion is explicitly one commit; Send,
  End, authorization, admission and publication are kept distinct
  (`docs/foundational/phase1n/evidence/runtime/README.md:51`).
- Automatic sampling covers all eight A/B answer inputs and representative
  A/B fail, absent and pending paths at actual limits 512/64/capacity1,
  I=R4/depth3/E0. Pending checks retain the original unmatched receives, B's
  helper consumption, J's consumed S and original e-await, and O's J-await
  (`internal/substitutiontest/callable_join_runtime_test.go:54`, `:131`, `:157`;
  `internal/substitutiontest/callable_join_instruction_test.go:375`).
- Genuine controls pass raw authentication and O public policy while failing
  their appropriate semantic premise. Omission Success yields J
  failed/missing_reply/absent; omission Failure yields failed/member_failed/absent;
  O commits invalid_join_outcome in both. The authentic false report yields
  candidate_failed in O while report truth fails. DeliberateFailure under absent
  yields a truthful candidate_failed report while component mode conformance
  fails (`internal/substitutiontest/callable_join_runtime_test.go:183`).
  The documentation explicitly withholds the conforming K_join premise; these
  are not conforming substitution counterexamples
  (`docs/foundational/phase1n/evidence/runtime/README.md:87`, `:96`).
- Seven detached mutations exercise the new handle/selection/classification/
  report/branch/cursor boundary, with history facts rehashed. Classification-only
  and Send-only original prefixes pass raw/public-prefix checks and fail the
  detached completion claim, without rewriting the original snapshot
  (`internal/substitutiontest/callable_join_decoder_test.go:13`, `:103`).
- The Python audit independently walks instructions and cursors, checks raw
  digests and selected-evidence links, and computes separate child-observation
  truth and O public policy without Go oracles
  (`docs/foundational/phase1n/evidence/runtime/audit.py:60`, `:133`, `:141`,
  `:148`, `:265`, `:331`, `:381`). Its saved final PASS includes both profiles,
  all seven detached attacks and the retained prefixes. Shared formats and
  artifact identities remain an explicit independence limit.

## Genuine late-first-authorization witness

The supported driver uses public operations and preserves original prefix rows
(`internal/substitutiontest/callable_join_runtime_test.go:241`). In final normal
capture record `360b706714c9/result.json.gz`:

| Retained fact | Order |
| --- | ---: |
| J consumes original S / e outcomes | 41 / 42 |
| J classification Set | 56 |
| J diagnostic Send, sequence 29 | 65 |
| J Failure, sequence 31 / freeze | 67 / 68 |
| First reply authorization, source sequence 31 | 69 |
| Reply admission | 70 |
| Original J publication and outcome delivery | 71 |
| O consumes original J observation | 72 |

The original J handle is `["root",1,1,0]`. Its admitted member is
`7b66d0420abba4a84c96511b13b2c61c82290231bcb7959a07329ab55a3bc3eb`,
and its report intent is that member's original sequence-29 intent. The selected
J observation is
`c5a58258b4b870aa6a72d231730a518143df2c028b0ea41e2185f84c8bacb338`.
It contains failed/member_failed with the present candidate_missing_reply
report. O actually consumes it and commits candidate_missing_reply.

Companion `360b706714c9/end-cut.json.gz` has no authorization attempt or reply
admission for the report; preparation is not ready with `reply_admission`.
The first authorization is Allowed/authorized with generation/revision/epoch1
and source sequence31, whose retained projection is failed. The bound reply
exception is checked against the original entry member, actual caller/generation,
work obligation, protocol/template and schema; it does not extend to invocation
intents (`internal/substitutiontest/callable_join_evidence_test.go:664`).
The race capture and saved Python audit retain the same causal ordering.
This meets the requested first authorization after End, rather than merely late
admission of an earlier authorization.

## Evidence and source checks actually performed

All indexed file byte sizes/hashes, every command's exact argv/exit/leaf count,
all source tar hashes and every archived source member were verified. Final
normal/race and historical normal/race have identical 405-source maps, all
matching current bytes. Aggregate SHA256:
`8b0c3e9b628dfbda57822df8e0005369635831ff99ed76ae9c3d075630d92a7d`.
Their 400 Go/module entries also match the controller's final-static source map.
All four report no source change during execution and have empty stderr, no
failed/skipped leaves. Saved validation, not reviewer reruns:

| Capture | Exit | Leaves | Command-record SHA256 |
| --- | ---: | ---: | --- |
| normal | 0 | 29 | `a676754f06b996aea91055f0ef62040202987c8535c58fa1c7ff7086af2f7922` |
| race | 0 | 29 | `66702b4ad62280c43bb047b1d617d5b2579bc065492e686ef8848286c9eff08e` |
| historical-normal | 0 | 82 | `7b8a2736c774a3a460dc2fd96dc00b44351825a2ec1f30a2229f7988af475ede` |
| historical-race | 0 | 82 | `4ee23efb3e66caa90dae3331727167315abaa34accb347e67c251c0a84d1e95a` |
| audit-final | 0 | Python | `c1b6f9fbd71fa62e76341c1c7a40a097c867531c211c4c2dea95d45cae3de0db` |

The index SHA256 is
`a932031e049e78f908f248d1ad6af9454f5f9fe43f599ea3afaf59ba8011089e`.
The final Python audit source SHA256 is
`821a1f70f8f4e5f0d512492f8b48ec865948ca59f96018bfc29601ba84bf1c3f`.
The controller final-static record SHA256 is
`a645a99a179067c9a06a8944b2d19a07f98237aa0ee8653b200d73db9d2dba58`;
its eight commands all exit0, including empty gofmt output, root normal/integration
vet and root/consumer module verification. Those checks are attributed to the
controller, not this reviewer.

For both profiles, reviewer calculations from the saved raw tables independently
match all 19 principal metric rows, all static PC/instruction/source-position
inventories, declared actual commit/dispatch counts, bounds and byte maxima.
This includes control bounds, not just positive examples. Static-plus-one local
ceilings are A139/B149, omission88, false-report124, mode-disagreement128;
dispatch ceilings are respectively9/12/9/12/9. Supported-driver RunReport is null.

Both historical captures contain the identical 48-entry equivalence record
`3da1c6672f80/equivalence.json.gz`, with full-value SHA256
`d0b8826d5f037dfafb18f902f987e3ace0a1233590c6b162fe4a0a14e81ae221`,
referring to the original Phase1L archive. A first reviewer lookup incorrectly
expected a top-level list and raised an assertion; inspecting the actual record
located its dictionary of count/keys/trace_hashes, after which the count and full
hash were verified. No evidence was regenerated or changed. The preserved
red-controls compile failure remains clearly distinguished from semantic RED
and from the passing final source identity.

## Focused unchanged-source risk checks

- Risk: shared structural validation could fail to tie O's selection to its
  consumed_sequence and original publication, or could accept missing terminal
  evidence. Checked `internal/substitutiontest/evidence_test.go:432`, `:520`,
  `:555`, `:601`, `:750`: inverse observation/history links, original receive
  guard/handle, member/freeze/End linkage and accepted frozen response publication
  are explicitly checked. The new full instruction walk adds the N-specific
  transition mapping rather than renaming a historical root-PC map.
- Risk: the supported schedule might compute successor states on the host,
  fabricate observations, or authorize J's reply during setup. Checked
  `internal/substitutiontest/communication_driver_test.go:86`, `:163`, `:218`,
  `:236` and `internal/substitutiontest/invocation_join_runtime_test.go:188`,
  `:194`: Commit, admission, settlement, authorization, delivery and acknowledgment
  are public calls; setup services descendants before J's Send exists. The new
  suffix then commits J locally without service until after End. No hook/held-lock
  reentry, sleep, authoritative edit or host-computed successor is used.

## Issues and remaining limits

No Critical, Important or Minor findings for this task-scoped gate.

The result is bounded tested implementation and saved-record evidence. It is
not a universal scheduler/compiler theorem, new recovery result, arbitrary
context congruence, or unrestricted weak-K1 lifting result. ARG-2 and the
approved reference/qualification restrictions remain unchanged. Runtime pending
quiescence does not independently prove finite-model maximality. Manuscript
reconciliation and final branch review are separate controller obligations;
this Task4 approval does not assert they are already complete.
