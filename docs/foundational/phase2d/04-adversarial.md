# Composed authority, outstanding settlements and competing evidence

The four tests in [accounting and binding controls](../../../internal/durabletest/phase2d_adversarial_test.go) and [world controls](../../../internal/durabletest/phase2d_adversarial_world_test.go) execute the public A/B Fork with two issuer invocations in one Store. The pure leaves use duplicate literal id 1 and distinct values 0 and 1. The additional actor in the competing history is an explicitly provisioned static resolver. That actor receives only the local operation selector from the controller. Both designated resolvers get their own original IDs through the declared ordinary handoffs.

## A published original can still owe settlement

Two real successful mutation responses reached the executor. Both workers acknowledged the pre-record cut, both issuers selected their ACKs and ended, and SIGKILL removed the owner before either result was recorded. On reopening, a targeted Q succeeded for each original. R then adopted and settled its original. For L, the designated resolver and its competitor both passed current dispatch and received independent snapshots before either returned to the Store. Both snapshots identified the same original receipt. The operator revoked the competitor while its return was held.

The competitor returned first and durably adopted L. An exact-operation SQLite trigger aborted one subsequent settlement. The failed Run's existing worker cleanup handled the designated resolver's other, already returned physical candidate. It retained and settled a duplicate without obtaining another read. Two cuts distinguish the remaining debts.

| Faulted INSERT | State at `published-unpaid` |
|---|---|
| L original settlement | Original result recovered, original settlement missing, winning query settlement missing, duplicate query settled. |
| Winning query settlement | Original result recovered and settled, winning query settlement missing, duplicate query settled. |

With the fault still installed, supported invocation service published the original issuance outcome and admitted it to P. Supported ordinary Commit selected the duplicate response and advanced the designated resolver through its report Send. Delivery and acknowledgment admitted this genuine report at P. Inspection and effect-free replay at this cut preserve both the publication and the outstanding debt. The trace distinguishes report delivery from P's later selection. The final parent decision in these experiments occurs after repair.

Removing the trigger did not repair the first owner's Q. Two successive Q calls on the revoked competitor returned the real SQLite rejection. A fresh owner's Q settled the outstanding original and/or competitor locally. Mutation applications, accepted business contacts and read contacts stayed at 2, 2 and 3. The resumed F(T) selected the authentic L duplicate and R report and took its applied branch. The sequential rule has one designated resolver per original. Extending it to preadmitted competitors would require inspecting the first importer's actual outstanding work.

## Operand, binding and world boundaries

| Control | Observation |
|---|---|
| Authentic R report offered to P's L receive | L and R had equal delta and distinct original keys. R's delivered found report remained unconsumed after a rejected supported Commit. P later selected both reports on their correct ports and completed. |
| Denied adoption or wrong target grant | Read-only R selected found/denied and retained application knowledge while its missing original stayed unpublished. The mismatched grant denied R before any physical ordinal. P could not pass its outstanding join. |
| Unsupported original policy | The L mutation applied under an unknown-return policy. Its found receipt could be selected as evidence, but its missing original could not be reconstructed. |
| Same address with new ledger | An independently created ledger reused the address. Both readers retained unsuccessful results and neither original was reconstructed. The replacement recorded zero successful lookup snapshots and zero applications. |
| Two genuine applications for one original key | Before the original network call, an independent client submitted two non-idempotent packets with the real committed L key. The honest endpoint applied both. L's differently encoded idempotent mutation was refused. Lookup returned conflict and no unique receipt. The composed parent selected invalid evidence. Count ingress attempts from the journal separately from accepted mutation rows. |
| Incompatible selected positives | As an explicit endpoint-trust ablation, an intermediary returned the honest first snapshot and altered the second snapshot's receipt numbers. The upstream ledger retained its genuine bytes for both snapshots. The ordinary resolver kept the first query/receipt, retained both per-query readings and set conflict. P actually rejected the joined report. A trusted unchanged idempotent ledger cannot itself supply that pair of different positive receipts. |

The normal development runs and their first failed build are [archived separately](evidence/adversarial-development). The failed build had an unused test import. The recorded runtime faults and invalid program decisions are expected observations in passing tests. Use the phase qualification for the immutable source, normal/race commands and final raw captures. The two source files offered for freeze have SHA-256 `1d312d5622714e2254e0341f5263820c9bf171b1ffd1422b31812a2728250a87` and `1060ec3be6d54dd0ef2288ec9bc048967f1e4a50824e0d4a45fe4d288cd9e677`, respectively.
