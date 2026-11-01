# Which observations survive recovery?

The eight coordinates below are accepted dynamic invocations, created runs, original mutation debits, logical-read debits, spent physical ordinals, original endpoint ingresses, lookup ingresses and business applications. The first four come from the resource roots, the fifth from durable reservations, and the last three from the independent endpoint journal and application ledger. These are the final **ordinary-regime** raw decoded counts for the fixed AB fork. Read the paired race qualification separately. A credit of two for P's effects permits both legal debits but charges only actual allowed reservations. A committed Request can be denied before it incurs a debit.

| Schedule | Eight coordinates | Actual parent join and completion |
|---|---|---|
| Forward or reverse | (10,10,2,2,2,2,2,2) | Five selections, both application classifications, three root publications. |
| Stable second duplicate or transport error | (10,10,2,3,3,2,3,2) | Five selections. R retains the first witness and the exact later query disposition. |
| Capacity one | (10,10,1,0,0,1,0,1) | Both original requests committed. R's first authorization denied after its End. P is still at its first original Await in this service schedule. |
| Read-only R | (10,10,2,2,2,2,2,2) | P selected L, waits on R. Real R found evidence and a report cannot supply the missing original publication. |
| Two pre-transmission R losses | (10,10,2,2,3,2,1,2) | R spent both ordinals with zero R lookup ingress. The one lookup in the total was L's. P is still at R. |
| Both R absences retained, eventual application | (10,10,2,3,3,2,3,2) | Both applications exist by the final backup, yet R never selected a positive and P cannot select R. |
| R lost absence then finds | (10,10,2,2,3,2,3,2) | Same logical R request on another ordinal observes the later ledger. P selects all five. |
| R retained absence then refreshes | (10,10,2,3,3,2,3,2) | A distinct logical R request observes the later ledger. P selects all five. |
| R ACK serviced automatically or on reopen | (10,10,2,2,2,2,2,2) | Five selections, both resolver ACK selections, three root publications. |
| R ACK first attempted after manual P End | (10,10,2,2,2,2,2,2) | Same private decision, two root publications. The R resolver stays open and never selects the ACK. |
| Matched continuous / lost response | (10,10,2,2,2,2,2,2) / (10,10,2,2,3,2,3,2) | Identical historical original rows and selected business projection. R used a different physical ordinal and snapshot in the recovered continuation. |

A zero in the parent join here counts the five required outer selections only. It does not erase the earlier ordinary identity handshakes. The application-order control has L=(1,1), R=(2,3) in the forward case and L=(2,3), R=(1,2) in reverse. With both deltas one and R first, R=(1,1), L=(2,2) under distinct keys. The parent checks the original invocation and report for each of those roles. The scalar final total and the number of published roots alone cannot recover those histories. [Case construction](03-experiments.md) · [Independent raw decoder](tools/audit_raw.py)

## Pure scope and capacity

A fully funded all-B two-leaf fork at P uses eleven invocation and eleven created-run slots. Under the same bounds AA uses nine, and AB/BA use ten. The four all-B deficits exhibit actual cause. An invocation limit of ten gives an invocation-limit refusal; a created-run limit of ten gives a created-run-limit refusal. Depth three excludes two helper admissions. Each of these three pure deficits allows both applications and yields a diagnosed pure product. An original effect limit of one leaves the second role without valid required evidence and the joined decision is invalid. The slow experiment first admits both originals with capacity two. A later lookup held under a one-worker runner does not block ordinary pure commits and publication. The two capacity settings describe distinct parts of that history.

In the 64 full two-leaf placements there are four applied answers, 32 completed diagnoses, and 28 intentionally pending placements. A pending fork waits on left if its left original is unpublished; a completed right can already have been delivered there. With completed left and pending right it has selected only left. In either case P has selected L and R and is at its T Await after full service, while each designated resolver waits for ACK and both reports are delivered without a parent selection. This is checked by admitted operations, retained programs, selections and leaf/frontier states. A timer is insufficient. The three relay, four one-less and one slow case bring the full grid to 72. Their diverse outputs are recorded separately from the original 64. [Pure induction](../phase1t/03-tree-theorem.md) · [Grid source](tools/grid.py)

## What an unchanged Replay establishes

On a cold copy a successful public Replay can verify an immutable internal history, including binding, reconstructed wire/proof link, committed ordinals and selections. The before/after file and journal hashes prove the observed replay window was read-only. In the controls, changing a real receipt's role, changing a normalized status and its matching facts, falsely attributing the import, or refunding expenditure fails both Replay and a raw decoder. Neither pure replay nor a preexisting local publication timestamps a network ingress, the return of an unrecorded handler, or payment of someone else's settlement. The actual process acknowledgments, independent journal and SQLite cut provide those different observations. [Adversarial settlement experiment](04-adversarial.md)
