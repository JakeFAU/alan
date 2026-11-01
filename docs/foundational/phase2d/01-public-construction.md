# Public two-edge construction gate

**Result.** On one retained schema-5 Store, an ordinary caller received two different original operation IDs from its invoked issuers. Each separately rooted resolver selected its own ID and sent it back. Each issuer selected the caller's authentic ACK and reached successful End while its original lacked a local result. Both physical originals had reached one independently persistent ledger. Neither resolver had issued a read. These statements were checked before the principal campaign in normal and race builds. [Normal history](evidence/construction/normal/frontier.json) · [Race history](evidence/construction/race/frontier.json)

## The rejected old construction and the selected extension

The original public two-edge attempt used the Phase 2C handoff declaration twice. Create rejected the actual deployment with `ambiguous caller handoff peer`. Each single edge's public template was individually constructible. A new pair profile requires exactly two fixed caller/sequence/port bindings. The old single-edge template and repository's older qualification remain bound to their earlier semantics. The new test retains the genuine rejection from the integration base. [Original RED](evidence/binding-commands/binding-public-red.json.gz) · [Binding proposal and controls](01-binding.md)

The separate module imports the public product, adapter, schema, effects and runtime packages. It provisions an A/B Series 1 Fork with literal request IDs 1 and 1, values 0 and 1 and two answer modes. The caller's two distinct issuer invocations commit at sequences 1 and 2. Its pure invocation commits at sequence 3. Each dynamic issuer selects its bound request and commits its counter intent at sequence 2. The caller's common root limits are 11 invocations, 11 created runs, depth 4, two effects and two in-flight. Each separate resolver has two effect units, one slot and its own exact grant for the original at member `source`, sequence 2, under the specified caller invocation and ledger endpoint. The two effect deltas are 1 and 2. [Standalone consumer](../../../examples/effectfulcomposition/consumer/main.go) · [Provision builder](../../../examples/effectfulcomposition/deployment.go)

The consumer driver sends each empty `start` or `refresh` service input only after reading the actual handoff and obtaining Q for that source. It passes no original target, receipt, world status or ACK to the authored programs. `service` can admit the input without starting Run. `resolve` and `refresh` service one explicitly named role and run the ordinary scheduler. For the construction cut the driver used none of these commands. It invoked only init, issue, frontier and read-only replay.

## Observable cut and how to falsify it

1. The independent ledger process recorded two genuine POSTs and two committed receipts. Its two post-commit hold filters applied before response delivery. The controller received the first endpoint barrier identity. The endpoint's barrier serialization kept the other response behind the same hold. At kill both keys bound the actual namespace and distinct `[subject,1,2,0]` originals, with the expected per-role delta. Each key had one original application. The complete journal and SQLite backups distinguish this claim from a report that a worker merely started.
2. Before the kill the caller had selected both genuine IDs, both resolvers had selected and echoed their own targets, and both issuers had consumed matching observations from `parent/1` and reached sequence 6, status succeeded. The checker joins each consumed observation to its actual source history selection, payload and role-specific obligation. The parent had committed the pure invocation and remained open. Each local original had one retained physical preparation, zero results and no reconstructed success. There were zero read ordinals.
3. The controller sent SIGKILL to the original issuing process only after those real held responses and ordinary terminals were established. The receiving fresh owner obtained Q for each exact terminal source. Parent and resolver Qs each returned open authority. Fresh preparation of each original invocation reported `external_outcome`, with one captured original and one captured reply. Neither original invocation outcome had been selected by the parent. The pure or remote facts do not stand in for those missing selections. The issued program used the real controlled counter type, allowing subsequent eligible reads to satisfy the same original policy as in the experiment.
4. The public Replay command succeeded and the ledger journal was unchanged across it. The two source Qs disclosed local drain. The available authenticated grant binding and the still-open resolver are separate facts from executing a lookup, paying for one, adopting a result or publishing and selecting an original. Those later stages require their own evidence.

| Build | Ledger PID | Killed executor PID | New frontier owner PID | Endpoint calls | Program observation |
| --- | ---: | ---: | ---: | --- | --- |
| Normal | 74769 | 74786 | 74787 | Two original applications, zero lookups | Both source ACKs selected and both local original results absent |
| Race | 74852 | 74881 | 74886 | Two original applications, zero lookups | Same |

The checkpoint files include per-role source and resolver states, the two complete authentic ACK observations, the exact grants, roots, operation bindings and fresh-owner responses. The packet also includes both pre-kill databases and the reopened local database. The command records identify actual executable digests, all argv, cwd, selected environment, PIDs, exit statuses, endpoint token and embedded Go build data. Both binaries in the race history have `-race=true`. Each of the six compressed SQLite backups passed a separate SQLite integrity check. [Normal records](evidence/construction/normal) · [Race records](evidence/construction/race)

## Input identities and reproduction

The complete input inventories for the two construction histories matched, SHA-256 `c7e4a3cb91bf68dea4ddfdb14dd2455c05c7eb578423ffe655d82462bd8f5ccb`. This was a construction gate during development. The final frozen-source qualification is a separate gate. In these captures the retained parent artifact ID was `042cfd9169fcba98e6bbd662e8a20dca5d27b7439356634578ad082f32fbd779`. The L and R template IDs were `deb75c99d9c7ada0ea79f7bd1dc22846b17f5538e33139b0a6d39aa1ea71cc9c` and `4d8ed52bfb065b59e9dcfd08a9b60e4fe292ffc09d894b9c125af1b317b02c0e`. The resolver artifact IDs were `5de95a6e40ba86a37548457b8902f287159e41b8567a0915e00375dcb9f59021` and `ee5338252117bbfcc4bf7911f987ac254162b94064c7ff8b3db394bb40bb427b`.

From the repository root, point GOCACHE at a writable directory and use cached Go 1.27.3, CGO and readonly pinned modules. Supply two fresh output paths. The driver builds both the public executable and the actual endpoint and checks their binary settings.

```sh
out=$(mktemp -d)
python3 -B examples/effectfulcomposition/consumer/construction.py normal "$out/normal"
python3 -B examples/effectfulcomposition/consumer/construction.py race "$out/race"
(cd examples/effectfulcomposition/consumer && go test -count=1 -timeout=2m -v ./...)
(cd examples/effectfulcomposition/consumer && go test -race -count=1 -timeout=2m -v ./...)
```

The two construction module test commands also passed, including the public rejection of servicing a role before its genuine original invocation exists. The broader composition, stable evidence selection, world-coupling and lifetime/progress claims are addressed separately. [Normal module run](evidence/construction/phase2d-construction-normal.json.gz) · [Race module run](evidence/construction/phase2d-construction-race.json.gz)
