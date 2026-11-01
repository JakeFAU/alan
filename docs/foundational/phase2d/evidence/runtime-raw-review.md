# Independent reconstruction of F(T) histories

**APPROVE the bounded runtime and raw-evidence claims for the qualified histories.** Each regime passed independent reconstruction of 18 named process cases (63 paired local/world checkpoints, 25 owner SIGKILLs after acknowledged barriers), the common-prefix pair, all 72 grid histories and nine dedicated adversarial cases (34 paired checkpoints). Public Replay rejected all four altered Store copies in each regime. The conclusions cover these captured executions and the stated endpoint and authority assumptions.

The audit starts at the two invocation occurrences P committed at sequences 1 and 2. It derives each bound member from the Store namespace, invocation and member name, then derives that member's original Request at sequence 2. The equal-delta control still has two different IDs, namespace-qualified keys, grants, frozen replies, preparations and observed receipt histories. Python decodes the archived SQLite rows, joins each lookup and ordinal to its retained grant, matches the exact original body against the independent endpoint journal and ledger, and reconstructs the receipt numbers from actual mutation order. Both auditors use the documented formats. The composition auditor reuses the independently authored SQLite, schema, ordinal and identity helpers from Phase 2C. Neither auditor imports the Phase 2D controller, grid grader, executor or model transition code. [Composition auditor](../tools/audit_raw.py) · [Adversarial auditor](../tools/audit_adverse.py)

* In the qualified forward histories the left and right roles respectively received (mutation 1, total 1) and (mutation 2, total 3). Reversing original service gave L (2,3) and R (1,2) while the final total stayed 3. With two delta-one originals the reverse order gave L (2,2) and R (1,1). Each accepted lookup preserved the original wire, ID and its own distinct key. Every application in these three histories joined to an original POST on the mutation route.
* The stable-error history retained two distinct right queries. The resolver selected a transport error on its second query. Its ordinary report retained the ID, target, adoption, mutation and total from the first selected found/adopted query. The Store retained that query’s full receipt. In the mixed case the observed original transport error and its earlier selected defaults coexist with a later found/existing read. The parent followed the executed equality branches of its own retained program and selected its five inputs in order.
* At a committed final ACK cut, the ID-bearing last message was still unauthorized, every earlier message had delivery bookkeeping, every invocation had an outcome delivery and eligible terminal subscriptions were exhausted. Stock service and fresh-owner service delivered that message while P was open. Manual End before service produced a retained authorization refusal with reason `source_closed`. The parent's decision was `applied` in all three histories. The manual history finished with two published roots, and the right resolver never selected its ACK.
* The coupled branches started with the same frozen local/world images and prefix journal. Exact per-role original keys, receipts including the original PID/token, selected outcome and semantic debits agreed. The right read cost changed from one spent ordinal and one lookup contact to two spent ordinals and two lookup contacts in the recovered branch.
* The two unpaid-settlement cuts separate the original reservation, original result, original publication, the first adopter's accounting and the later duplicate. At each cut the twin's successful import and the real designated resolver's duplicate result both existed, the latter's settlement and admitted report existed, and the original invocation was public. The original's own settlement was absent in the original-settlement fault and present in the adopter-query fault. The twin's own settlement was absent in both. Its same-owner Q returned the directed error twice. Successor local accounting preserved the evidence and endpoint rows.
* The actual wrong-role refusal presented the genuine right observation at the left receive. The parent rejected it before subsequently selecting L then R. A wrong read grant produced no right ordinal. Proper read-only authority and unknown original policy could observe genuine found snapshots but left incompatible original obligations unresolved. A fresh endpoint on the original address returned HTTP 404 on the pinned old ledger route. In the multiplicity history two explicit external non-idempotent applications competed with L's own refused application, and the parent rejected the joined evidence. In the trust-ablation history, stored upstream and delivered bytes differ for the second L found response. The resolver retained its first witness, detected conflicting values and the parent rejected the join.

## Raw refusals and resources

In `atomic/ordinal-1` the Store already contained the reconstructed R result and the first physical ordinal. Its original settlement, publication and resolver selection were still absent. In `spent`, two R reservations survived and there were zero R lookup ingresses. The resolver eventually selected `read_exhausted` and P selected only L. In `finite-reads` both real R snapshots returned absence before R's subsequent application. In `capacity-one`, R's original had `source_denied`, produced zero application contacts and P reached none of the five outer selections. Under the R read-only grant the resolver selected an authenticated found result with adoption `denied`. P selected L and remained at R.

Across the 64 base leaf placements there were four applied answers, 32 diagnoses and 28 pending joins. In each pending join P had genuinely selected L and R, the retained PC was waiting for its tree invocation, both resolver reports had reached P's inbox without selection and both resolvers lacked a selected ACK. The eight extra placements contain three Relay positions, a slow Relay run and four one-less resource runs. Across all 72 the counts were six applied, 36 diagnosed, 29 pending and one `invalid_evidence`. In the all-B answering case the actual parent root spent 11 invocations, 11 created runs and two original effects at depth bound four. Each resolver paid for one logical query against its own root. The four one-less runs record separate `invocation_limit`, `created_run_limit`, `depth_limit` and `effect_limit` refusals. The first three produce diagnosed pure trees. The effect refusal produces `invalid_evidence`.

In both `BB-answer-answer-above-slow` runs I queried the two SQLite images at `held-read` and `pure-before-read-return`. Rows in `history` increased from 224 to 280. `composition_publications` increased from nine to ten, adding the succeeded invocation at parent sequence 3. Local `adapter_results` stayed empty, `receipt_read_ordinals` stayed at one and both world images contained two applications and zero receipt reads. The proxy had recorded the lookup ingress at event 18. The backups were events 20 and 77 and its first lookup reply was event 79, using zero-based indices in `processes.json.gz`.

## Rehashed corruption attacks

For each execution regime, the auditor started with the same history for its baseline and all four copies. It updated the stored row, its result or reservation digests, the associated composition fact and matching settlement or effect charge. The swap supplied the genuine other role's receipt from the same ledger. The status attack changed a found response to absence. The import attack named the other resolver's real query as its source. The cost attack refunded a real query debit. Public `consumer replay -db …` accepted the baseline without changing its image hash. Each modified copy failed in both the independent checker and public Replay. The public errors distinguished a missing original result for the swapped receipt, an unexplained reconciled original for the false status and false import, and an adapter attempt binding failure for the refunded debit. The command output records every mutant image's SHA-256, reject reason and process exit.

## Qualification records

All four command records contain a zero exit, the same starting and ending executable revision, an empty Git status and 718 unchanged executable input hashes. The output includes the exact PID-qualified receipts, decoder results, per-case debits, hashes of the actual baseline and mutant images, and public Replay errors. The composition and adversarial auditor SHA-256 values are `c5483915c8da91f14c3ee13b7936ca9b816af96b2c8da4ab93e183076b75afc8` and `f2c2d1a195b2f05999d5bc1461dff06e2f8013bad448802e37ccda20c3d7628b`. The Phase 2C helper is `cc5470695ad13bbc7035d8c642d1a44a6c715ad4867f8e6413050b2d048ec336`.

| Gate | Compressed record SHA-256 | Exact stdout SHA-256 |
|---|---|---|
| `normal-independent-raw.json.gz` | `b89ce0b9b827fdac3e37eac4610c7798c446ffcc6351368bba4bddc3bf402243` | `60158c3327627a3ba5d42dea0f894001f2af43ecce893ead2e6f904b4c2ec8ea` |
| `normal-independent-adverse.json.gz` | `135bf5f934d78c705c6802e88d96cdf5f44c707b588c716f01159981ac363724` | `95159f14c4c9359a2924aa134196c41b85aefdc6be6e19be06178b37ea98114f` |
| `race-independent-raw.json.gz` | `93457d8b195d8257690abed99e6941ef3ed377ac4e8021e78f7700d534c3a91d` | `1f585472c29086fa99893a89bca0bc8db8d265d87b2f07e5a502123343723225` |
| `race-independent-adverse.json.gz` | `68ae9f1cc5e090e4c768e2ccb8658e7bc62bb6a60291246a5dc67369cd705bad` | `71e7f524455bfb6144ae60aed0a7d4f58536e2cba8c1e25c65c59441bf053404` |

I reopened each delivered process, grid and adversarial tar member and recomputed the SHA-256 of the sorted `hexhash  relative-path` lines, including their newlines. Each regime contained 198 process members, 372 grid members and 146 adversarial members. Their manifests matched the unarchived audit inputs. [Archived histories](qualified)

| Corpus | Normal | Race-instrumented |
|---|---|---|
| process | `cdd337e4ae37f8974366a2e2da0974e0aea40f3adb974366209f47e09d568109` | `0431bd9c7ac5af17606c11db93bde6d6496948889ba83b19a2ffe125942c603f` |
| grid | `c69bf51ad861c5ab7697971a576b7e5f001cda0f45aff5187313293e9a38a30d` | `9d5c641c4fc2f978dadb0dfb3c5c7193ac3a4e2a141df531e97e49d5a7b5f8f4` |
| adverse | `f4d223b7d97fddabed72cf92a9bc871f791e5f4eaa810bf05f8618b27fd9e371` | `2febb87e41886e07bfe20cf1e1094e71dcf06c3d0b83b032d667a275fee9d158` |

The recorded audit commands ran from `/private/tmp/alan-phase2d-qualification`. `python3` resolved to the Homebrew 3.15.2 interpreter shown in each manifest. Run the following with the qualified corpus and a fresh destination for the records. [Gate environment and producer commands](../07-runbook.md)

```sh
scripts=docs/foundational/phase2d/tools
corpus=/private/tmp/phase2d-qualified
records=$(mktemp -d)
for regime in normal race
do
  python3 -B "$scripts/gate.py" "$records/$regime-independent-raw.json.gz" -- python3 -B "$scripts/audit_raw.py" --process "$corpus/$regime-process" --grid "$corpus/$regime-grid" --consumer "$corpus/$regime-bin/consumer" || exit 1
  python3 -B "$scripts/gate.py" "$records/$regime-independent-adverse.json.gz" -- python3 -B "$scripts/audit_adverse.py" "$corpus/$regime-adverse" || exit 1
done
```

## Limits of the verdict

The qualification checkout pinned the source, protocol and process tools at `8ad4b3cffb5185ec236f152c87fc7b9ce3676c36`. These are finite implementation observations under the inspected local/wire formats, acknowledged cuts and branch-specific endpoint assumptions. Replay establishes internal consistency only to the extent checked by its code. It cannot establish the honesty of an external receipt or the observation history of an unrecorded schedule. The archived process replay windows compare the local image and endpoint journal hashes. The adversarial replay window also compares all captured world tables. Timing does not establish scheduler fairness.
