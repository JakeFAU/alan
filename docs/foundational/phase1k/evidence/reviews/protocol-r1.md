# Phase 1K protocol/profile review r1 — REJECT

Spec compliance: **REJECT**. Design quality: **REJECT**.

Review scope: pre-experiment plan and protocol only. One blocking recovery-design
finding remains. This review neither approves an implementation nor claims any
new model, runtime, process-recovery, or external-consumer result.

## Reviewed identities

- Baseline HEAD: `94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`.
- Baseline tree: `a20478fa31b333c6ff0be43fc9b43826b54bbb3a`.
- `1d13bc238aada500c3b9608d400435e235fedc0b^{tree}` independently resolves to
  that same tree. GitHub merge status is attributed to the controller's
  `evidence/baseline/baseline.json`, not a new remote check by this reviewer.
- Plan SHA256: `9e89954ad0d327b9c2be7441b9a9f27c4924a7943351710192ef9f502da78346`.
- Contract SHA256: `0b9013128dd84790da1d3fe9466fcb9fb4b4ecbfb1cf850576d50064076ca112`.
- Assignment SHA256: `dc9af808296a9516382c7b286f02b8e65b00042f8934ec4f208efcff2261654d`.
- Review-package SHA256: `95984b3c06ddd78d65b5681d64af70c1c2b9b372271c673e842febca3a31de3e`.
- Worker-report SHA256: `2f46eb383c507323194de9576e863ecab06997ddc59f2b30c2e76f52699fa95d`.

Hashes were computed from actual files. Both new-file diffs reconstructed from
`task1-review-package.txt` equal the reviewed plan/contract bytes. The worker
report does not itself contain freeze digests; the identities above are the
reviewer's verification of the exact current files.

## Blocking finding R1 — the refused cut does not retry a retained delivery

**[P1] `01-communication-contract.md:312`; propagated by `00-plan.md:125–128`.**
The pre-kill predicate explicitly requires “no report admission yet”, while the
reopen column calls the operation a reattempt of “same authorization and transport
delivery”. With that predicate, only the authorization has been attempted before
the crash. The post-reopen `DeliverMessage` is the first delivery/admission under
the chosen transport identity. It cannot establish persistence of an already
refused delivery identity across the kill, as required by `assignment.md:113–117`.

This distinction is executable in the source, not terminology alone:

- `internal/executor/messages.go:112–123` returns the prior authorization row.
- `messages.go:180–225` independently authorizes, constructs the message
  descriptor, and calls admission with `source_denied:source_closed`.
- `messages.go:408–415` returns the existing admission result for an identical
  transport/descriptor. The current cut never exercises this retained-admission
  branch after reopen because no such row exists before the crash.
- `messages.go:231–260` records dispatcher acknowledgment of an admission even
  when that admission was refused; this must stay separate from delivery success.

**Required correction before design approval:** retain the same original report's
first refused authorization and an actual refused admission before the kill,
using the fixed original runner transport identity. Freeze whether dispatcher
acknowledgment is already present or deliberately absent. After a real SIGKILL
and fresh Open, call authorization and delivery again using those exact identities,
compare the old authorization descriptor/result/source snapshot and admission
descriptor/result, and verify no replacement attempt/admission/observation,
consumption, charge, or permission was invented. If acknowledgment is delayed,
record it after the reattempt as refused-admission bookkeeping. Root and aggregate
pending status remain separately observed.

The author-only refused cut could be retained as an additional labeled prefix,
but must not substitute for this same-delivery retry case. Three focused cases
remain sufficient; no new payload/candidate campaign is needed. Preserve this
REJECT record and independently rereview the revised plan/contract hashes.

## Other challenged boundaries

| Area | Review result and source basis |
|---|---|
| Three profiles | Acceptable at design level. Contract lines 40–54 distinguishes abstract reachability, exported-API scheduling, and unchanged automatic scheduling; fairness does not override a refusal. It preserves model-only and source-inspected labels. |
| Automatic exclusion | Acceptable only under the frozen queue predicate. `runner.go:131–160` handles one internal delivery before `runner.go:222–239` commits at most one ready run. `runner.go:378–472` derives/sorts pending work; `runner.go:268–295` dispatches and acknowledges a single item. There is no drain-all barrier. Restart resets `last` at line 113; the sole-item premise makes rotation irrelevant. |
| Historical queue facts | Independently inspected the retained B `before.json.gz`: peer open PC37/seq37, root open PC2/seq2, original report `["peer",1,37,0]`, no matching authorization/admission/ack, three delivered invocation outcomes, all three reply messages acknowledged, and three empty terminal routes. These facts support the conditional sole-pending-report inference. They are historical bytes, not a new execution or stronger relabeling of J1's crash result. Fresh J1/J2 queue qualification remains required. |
| Source boundary map | Acceptable. Commit records selected consumption, history, intent and successor together (`store.go:754–828`); message authorization, admission, and observation retention are separate (`messages.go:92–225,385–492`). Source closure is checked only on first authorization; target checks remain at first admission. Invocation replies have a distinct binding exception (`invocation.go:123–140`). Root publication is separate (`composition_root.go:23–85`). |
| Public-driver legality | Acceptable as a protocol to implement and qualify. `runtime/runtime.go:15–31,44–65` aliases the Store and required public values; `store.go:715–779` enforces current authority and actual eligible observations. The plan forbids private reduction, fabricated observations, branch/register edits, replacement instructions, and concurrent Store.Run. Inferred return types and external compilation remain explicit implementation gates, not established tests. |
| Full J2 branch policy | Acceptable. Contract lines 201–227 preserve candidate-before-report, truthful known negative/absent S reports, literal absent fillers, pending discipline, wrong-report parking, one ACK before final decision, exact request/S-handle validation, and no extra rounds. This agrees with the retained J1 branch shape (`examples/join/deployment.go:84–142`). Root Failure's shared-scope freeze is separately qualified; successful root termination cannot freeze the open peer (`invocation_outcome.go:239–256`). |
| Moved ACK race | A sound falsifiable design prediction. Peer report lifetime is protected by the peer's wait, while root ACK commitment still precedes its own terminal boundary and first authorization may be delayed. Aggregate completion waits for all static members; root local success alone is insufficient (`invocation_outcome.go:247–256,324–352`). Actual public realizability remains unexecuted. |
| Finite product/rank | Acceptable at design level with the clarification below. ACK stage and terminal guards separate the intended edges; source identity and actor sequence are not replaced by an invented global ordering. The rank extends `research/join/model.go:181–192` and `research/substitution/model.go:431–454`; all-edge monotonicity and maximal-state coverage are required future checks. No census has been run by this review. |
| Terminal accessibility | Correctly framed as an external probe. Author Send/Receive constructors expose no authorization receipt or terminal matching (`author/author.go:81–93`, `author/compile.go:285–305`). Raw Program receive accepts explicit kind/provenance (`program.go:161–171,362–387`). Frozen terminal routes and authenticated terminal descriptors are available through Store (`store.go:815–825,854–855`, `messages.go:263–307,381–383`). The anonymous destination struct matches the fields at `messages.go:29`; actual compile/provision/consumption is still a required probe. It does not prove consumption of a prior ordinary message. |
| Other two process cuts | Predicates distinguish committed/no-auth from allowed-auth/terminal/no-admission. The safe-boundary plan is appropriate: Store.Commit retains `s.mu` through its committed callback (`store.go:715–717,841–850`), and authorization does likewise (`messages.go:92–94,857–865`). Parent read-only inspection uses a distinct connection (`inspect.go:27–40`). The pipe acknowledgment is apparatus, never a protocol ACK. R1 is the remaining cut blocker. |

## Nonblocking precision note N1

At contract lines 261–276, “base PC0..8” should distinguish operation PCs (0..8)
from context RootPC/PeerPC (0..7) used in the claimed base-rank cap231. The actual
context assignments inspected in `research/substitution/model.go:234–237,393–414`
and `research/join/model.go:77–89,144–168` never assign context PC8. With that
source-supported restriction, the conservative arithmetic is
`6 + 7 + 7 + 3 + 100 + 4*(3+2+2+8+2+3+2+5) = 231`, then
`100*231 + 60 + 1 + 32 + 8 + 1 = 23202`.
If both context PCs were allowed to range to8 merely from the currently listed
interval, that field-wise derivation would yield233 and23402 instead. Spell out
the narrower reachable context bound so the frozen derivation is self-contained.
This is a precision correction, not evidence of a reachable rank violation.

## Evidence scope and remaining obligations

Executed only read-only source/document inspection, Git identity/status/diff
checks, SHA256 hashing, review-package reconstruction, and Python gzip/JSON
inspection of the historical B record. `git diff --check` exited0 and tracked
`git diff --stat` was empty before writing this report. Some discovery reads
referenced nonexistent guessed paths; those reads supplied no evidence and were
replaced by actual discovered files. No new model/context/runtime experiment,
Go test/build, process execution, dependency change, TeX build, or commit ran.

Inspected scoped manuscript claims at `paper/alan-dissertation.md:528–532` retain
the correct J1 model/runtime distinction. Phase1K manuscript reconciliation is
a later implementation obligation; this design review does not claim it complete.
The historical B compressed record SHA256 is
`cb4b91bc335813ed9c9886e6ee86d705558e4be20f703eb7110ce9a606bd8301`.

Only this review report was written by this reviewer. Production, compiler,
schema, dependency, accounting, scheduler and historical sources were unchanged.
The controller must correct R1, retain the rejected revision, obtain approval of
new design hashes, and only then begin the new experiments. Later correspondence,
raw causal mapping, actual public compilation, exact process predicates and final
claims still require the planned independent review.

## Source identity appendix

Selected source sections above were inspected from these full-file identities.
Aggregate SHA256 of UTF-8 compact sorted-key JSON (no trailing newline):
`b40423df6cc3ed7c3befee5fdc841ab246f7c4a74a79806ba886aa1dc61a3f46`.

```json
{
  "author/author.go": "92db0faa7441fc1e4dd8864651627fbc478bc14251078cca225324aeb9d86662",
  "author/compile.go": "45e5ede143a60159d3beeb970c2e17adf1989f5c788fb4107e506034fbf58307",
  "examples/join/deployment.go": "0bcbff9113b9e11b148f076e839b5279b28c833e77fbe0069fab14902ce3249e",
  "internal/boundary/boundary.go": "7064a8b7afdf279a9c49065b865ca476e8d7f375d40ee140c8394e968ea52d8e",
  "internal/executor/composition_facts.go": "883a150eab80c32206f81efa69b9cc6f13453626ddbcb13360328c389e68c090",
  "internal/executor/composition_root.go": "a0e04ee2b1713caacc8b48cf648d783a60365c2b020a1513aa6b7a22000cc2cf",
  "internal/executor/inspect.go": "ef642165e9ba6731d1ad892116ac17e055b0bac586707bd28a3044eee96822aa",
  "internal/executor/invocation.go": "b40c1edc17249a31ccff84408a7855098e3a51ad28579ea856efc34bf4fa6746",
  "internal/executor/invocation_outcome.go": "adeaba4f04843867c5405a6bfb5a8716e64654a3e96bc58ad190ec23cbfcb53d",
  "internal/executor/messages.go": "1186fd9220d44352dcfd77ec5e7209dadd78d9956304a931593b5de6c6b8ae39",
  "internal/executor/program.go": "4a6813c69d509dff545c75a05410919e182e73c9f06e5fd359112e64df568374",
  "internal/executor/runner.go": "ad9fe52c4f17adb4c528b9397fb2e90338f9f9e63c5015160946daa593c1b2f4",
  "internal/executor/store.go": "4100327b3d95b44c061dad51f656fd193f26bc28c58093ecd2b385f9f76895bb",
  "internal/substitutiontest/join_process_test.go": "72fe13b2d5f5735284c748390cd3d77b3f444ea84ecb6fa0adfe1c9f61a6595a",
  "paper/alan-dissertation.md": "6e77aa02aa151f5457a478e8d17cd278ce35d47c2cbea575639844cc55c4f231",
  "research/join/model.go": "d614256652dedbb616dbd02fa1414c26609ab4295575825dfa28bb521a5c9154",
  "research/substitution/model.go": "1faef47e9ef17401512aba14239fc07ad7031a6db249600844752b35db342ddf",
  "research/substitution/runtime_boundary.go": "b22089368f93bea5f3e471292f0b7308419b937508e78942343cdb001d362e44",
  "runtime/runtime.go": "01af8ad5279bde9b6fe5d51158edd492452a3235ec1d2372d6f36996a48d6a4d"
}
```
