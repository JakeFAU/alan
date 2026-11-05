# P3 decision and evidence map

Retain the single call-local readiness optimization, subject to the final-source
attestation and independent whole-change review accompanying the delivery draft
PR. The independently approved bounded experiment passes its frozen guards;
this is not a merge, release, unrestricted composition or arbitrary-writer claim.

1. **Was repeated loading material?** Yes on integrated baseline `926a899b`:
   B256 reached 178,601 eligible visits but only 1,537 summed distinct reusable
   IDs per call. The candidate removes 177,064 successful loads/decodes/unbound
   validations. See [results](results.md), including all seven measured bounds.
2. **What makes reuse safe?** Within one `selectReady`, Store mutex, pinned
   connection, cooperative single-owner exclusion and immutable retained-program
   rules keep bytes, semantics, limits and format context stable. The local map
   ends before the next commit. There is no SQL snapshot or hostile-writer claim;
   pre-call corruption still fails on first reach. See [invariants](invariants.md).
3. **Did binding/reduction alias data?** Handoff binding has a real shallow-copy
   alias hazard, witnessed by a test. Programs with any HandoffPeer port are
   excluded before binding and reload every visit. Genuine same-call members,
   nested metadata/schema/literal mutation, collection reduction and authority
   copy probes support the remaining ownership boundary. These are bounded
   controls, not exhaustive pointer proof.
4. **What disappeared and remained?** Two SQL executions per eliminated load;
   B256 Execute SQL 3,600,555→3,246,427. All 178,601 bindings, 1,023 readiness and
   1,023 committed reductions, complete identity/current-generation checks,
   observation order, approvals and actual ready selections remain. The test-only
   reference is the baseline selector body, independent of the new helper.
5. **What changed in resources?** B256 median Execute 60.558126→47.193826s
   (22.07%); cumulative allocation 53.008→42.191GB (20.41%). Whole-process RSS
   82.492→83.141MiB (+0.648MiB) passes the unchanged guard. Allocation traffic is
   not retained memory. The map retains O(distinct reached reusable programs)
   until return; the 32-distinct control does not bound arbitrary unique cases.
6. **Did one/eight observations regress?** B1 median is 0.1181045ms slower
   (+2.97%), within max(10%,1ms); B8 improves 9.00%. All small-bound, other-phase,
   summed-approval and peak-RSS guards pass. All 98 scheduled outcomes remain,
   including 14 excluded warmups; no replacement, retry or sample expansion.
7. **Did external contracts hold?** Independently approved Task 4 records genuine
   SIGKILL/fresh-owner cuts, authority/revocation/retained grants, public consumers,
   genuine 11→12 promotion and actual historical-reader rejection of 13/14.
   Its tested source is `49df59f1f767dd63fec5e238bf565d023261a75b`, explicitly bridged by unchanged Go/module bytes
   to the timed and final sources. The final attestation separately records fresh
   full suites, consumers, formats and protocol tests on the final source.
   [Recovery limits](recovery-compatibility.md) retain the absent archived DB for
   the request-commit cut, distinct namespaces and immediate-selection refusal.
8. **What remains costly?** B256 summed Run is 31.305s and fresh approval
   snapshot/basis 15.611s. Counters retain 29.36 million observation considerations
   and 8.42 million root-intent checks. The separate B128 profile identifies
   InspectExecution, readiness, observation/root reads and finishRun allocation;
   inclusive scopes overlap. No second optimization is included.

## Exact identities and review custody

| Role | Identity |
| --- | --- |
| Integrated baseline | `926a899b651fccc48be52a054612a92afcf1ddfd`, tree `a58fe3b0ef0b9a7525c46c4f0a662774e6d4221d` |
| Contract/attribution freeze | `5e2045d1556083548e2926a965c43695322388c0` |
| Invariant/RED freeze | `a3d7a395c325ed7009705fd0f7c4ad35b1cedbb1`; same-call correction `93998c9065a5ceab7f33db06f1938bc8f87873c2` |
| Sole production implementation | `a298332f7e8507a64d4ec5dc00cd18599bb4c361`; test-helper convention `6e9b65276adff40184cbc86b140c492b551002dd`, counter-placement `49df59f1f767dd63fec5e238bf565d023261a75b` |
| Task 4 tested source | `49df59f1f767dd63fec5e238bf565d023261a75b`, tree `b86590205169e661a1bf14ad5d72701b4e0a684a` |
| Timed candidate | `4f5fe2cc83316900a2f387180dd9f856db5303a0`, tree `83ce5c2662dc73b9acc413ed87c94ef4511822ab` |
| Strict tool rejection → approval | `59bc43a0e37d3b6b10b24a2090d64cc664a46970` → `b2b9cc9f8dc1b0af7f8f38ac684136b53709469e` |
| Task 5 evidence/review head | `6934e41aed749a14d2f68c26603518b8dfd57839` |
| Timed A binary SHA256 | `729677568eb86c6d90483a8f2c513f147d34787abe0bdc7d21f1aa82478efe99` |
| Timed B binary SHA256 | `f0df8f5989efb5478515406962bbcf7096c9afb87624a35ec1b515f79f45ed77` |
| Approved timing binding SHA256 | `66a8c3a4d2d596ba53c3c1e35930eb4c706f747c52cb402e7584efd4fdfd6116` |

[reviews/index.json](reviews/index.json) hashes byte-identical copies of each
original independent report, including the Task 2 and Task 5a REJECT findings and
subsequent explicit APPROVEs. No original report was rewritten. Task 4's
reviewer artifacts are under `task4/review-task4/` (66 copies and
`sqlite-review.json`), as recorded by the original report's `R` shorthand.
Task 5 review raw checks also live at
`/private/tmp/alan-p3-performance-review-6934e41/`; Task 5a original/re-review
raw checks are `/private/tmp/alan-p3-tooling-review-59bc43a/` and
`/private/tmp/alan-p3-tooling-rereview-b2b9cc9/`.

The source for the complete qualification was committed **before** that run. `task6/fix2/attestation.json` and its source/tool inventories
identify the complete run at `4184c5129b9d7b9e62f155c41b86763f7a103417`.
The later external-input custody correction has its own
`task6/review-fix1/external-attestation.json`;
`task6/delivery-attestation.json` links both exact source trees, gate sets and
the 330-file Go/module byte bridge. These external records avoid self-reference
and do not relabel the earlier complete run as the later correction commit.
The delivery PR supplies both tested identities, the delivery SHA, final
independent review, actual verified attachment URLs and target-main/integration disposition.
The evidence bootstrap/index names and reproduction route are in
[reproduce.md](reproduce.md). Earlier gate identities are never relabeled as the
final commit; any later source change requires affected requalification or an
explicit evidence-only executable-input byte bridge.
