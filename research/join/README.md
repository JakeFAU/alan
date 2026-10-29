# J1 independent bounded join model

Simple J1 preserves the root's protected-work lifetime invariant in all eight
frozen configurations, but does **not** guarantee completion over the declared
fair interleavings. The ordinary report can be committed, followed by peer local
Success, then receive its first source authorization. That authorization is
refused as `source_closed`; message delivery records
`source_denied:source_closed`. The root remains open at its mandatory join,
although candidate and S outcomes have already been consumed. No cancellation,
resource exhaustion, or unfair withheld service is needed.

The model imports only the previously reviewed `research/substitution` primitives
and standard library. `Successors` uses unchanged K1 and independently transcribed
A/B graphs, retains the root terminal admission fence, separates root/peer outcome
consumption from terminal instructions, and adds ordinary report commitment,
source disposition, admission/delivery, consumption and aggregate publication.
The 32 peer Set steps remain explicit. Fixed topology costs two static runs;
dynamic singleton admissions charge (1,1,0), report communication charges none.

Each product has exactly eight initial states: id1/2 × value0/1 × depth2/3,
answer mode, I=R3, E0. Across those states:

| Product | Reachable states | Edges | Maximal success | Maximal report-refusal pending |
|---|---:|---:|---:|---:|
| K1 (both permitted choices) | 12,320 | 25,648 | 16 | 16 |
| A | 6,160 | 12,620 | 8 | 8 |
| B | 11,260 | 23,124 | 8 | 8 |

These are graph-state counts, not failure frequencies. Exact abstract maximal
path counts are separately recorded as decimal strings. They are not concrete
runtime instruction-schedule counts: the new root/peer finite pure suffix is
mapped by stuttering. `correspondence_test.go` independently evaluates inspected
compiled branch operands against the already frozen J1 oracle, checks all 136
new-context instructions in the mapping, exact unchanged Invoke/Await prefixes,
forward-only suffixes, and original candidate/S bytes. The source and artifact
identities are in `docs/foundational/phase1j/evidence/model/`.

The rank is `100*substitution.Rank(base) + sum(private candidate PCs) +
private-register7 + delay + reportStage + aggregatePublished`. It excludes
history length and Sequence coordinates. Every graph edge, including revisits,
strictly increases this bounded intrinsic rank. The census has zero rank
failures, cycles, cutoffs, unexpected stuck states, resource/depth/ledger
violations or joined-context lifetime violations. The optional K1 local branch
excludes its unused helper ticket; prior spending is never refunded. Root normal
Success/Failure remain actual explicit edges enabled by the program's join
state; no environmental lifetime assumption removes them. C0 all-eight products
still retain the original premature-root fence and failed aggregate witnesses.

The finite stutter map preserves terminal eligibility and enabled protocol
service: root guards after its Receive have no new protected work, peer prefix
Set/branch instructions precede report commitment, and the post-report jump has
no effects before peer Success. Arbitrary service can occur before/after those
finite steps. This inspected bounded mapping is not a compiler correctness proof
or a general simulation theorem. Model Sequence fields provide monotone actor
coordinates; runtime normalized events retain actual execution sequence numbers.
Only their causal ordering is related.

`CheckTrace(Trace)` is a prefix safety oracle, not an eventual-completion test.
The raw adapter must establish provenance, original operation/selected observation
identity, source authorization and storage integrity before normalization. The
oracle checks consumed-S-before-report, exact status/reason/presence/payload,
report correlation, exact committed/delivered/consumed report identity, strict
candidate-before-report consumption, final root decision and duplicate/extra
work. It is independent of cross-actor event array ordering. It permits the
conforming refused-report pending prefix and rejects premature or wrong-request
peer assertions. External root cancellation is a separate interruption.

Reproduce from the repository root with Go1.27.3:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build \
  ALAN_JOIN_MODEL_EVIDENCE=/tmp/alan-phase1j-model \
  go test -v -count=1 -timeout=10m ./research/join
```

The evidence variable must be an absolute path. Tests write gzip JSON with exact
per-configuration censuses and shortest retained witnesses. The same command
with `-race` qualifies this pure apparatus for data races. The intentionally
false all-maximal-states-complete assertion is reconstructible with
`python3 docs/foundational/phase1j/evidence/model/reproduce-red.py` and must exit1.
Its failed output is retained separately; ordinary tests expect and preserve the
counterexample. This is a bounded negative result, not a universal impossibility
of joins or a claim that dispatch-first Store.Run samples must fail.
