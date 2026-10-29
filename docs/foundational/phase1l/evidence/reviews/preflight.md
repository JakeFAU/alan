# Phase 1L Task 1 independent review

**Specification verdict: APPROVE.**

**Code and evidence quality verdict: APPROVE.**

**Overall disposition: APPROVE for Task 1 construction feasibility only.**
No actionable finding requires a change to this frozen slice. This approval does
not establish the K1 reference premise, bounded substitution, all-schedule
completion, or recovery.

## Scope and identity

Reviewed `AGENTS.md`, the Phase 1L assignment, Task 1 of `00-plan.md`, the
preflight `01-contract.md`, implementation report and review diff before assessing
the implementation. Then inspected all new example/consumer sources, public
invocation operations, original A/B/helper/S construction, capture script, compact
index and decompressed raw records. The eight new source/module/README files in
`task-1-review.diff` match their actual files exactly. Capture/evidence were
reviewed separately.

Base/HEAD is `d31bf4dee65d4a5a63ef5c149e58ccfc1995cd37`.
Verified all 19 listed source digests against disk, all ten inherited listed files
against HEAD, and equality of index and archive manifests. Manifest SHA256:
`e1f1137e4763bec9b2a49dd573be2084bbb094e11c925ab2e400c26baf83ab83`.
Archive SHA256:
`26282d56e1d9a597cc968718b13977d0d5f389695e17f1e153acc82375c8d7d7`.
No implementation or evidence was modified during review.

## Specification checks

- **One fixed parent, original templates:** `deployment.go:31` obtains the original
  S template through the existing public builder and discards the old provision;
  `deployment.go:57` provisions only `root`. Both archived provisions contain one
  fixed run. The complete exported A/B/S template records equal the corresponding
  frozen Phase 1K records, and every provisioned program's bytes hash to its ID.
- **Commit both, then join both:** `deployment.go:98` emits candidate and S
  invocations before either await. The actual compiled prefix is emit, emit,
  receive, receive, with the receives matching their respective retained handle
  variables, template sources, obligations and runtime-invocation provenance.
  Both archived histories commit at parent sequences 1/2 and consume at 3/4;
  terminal success occurs only later, at sequence 16.
- **Strict outcome policy:** all normal terminal branches follow both awaits.
  Compiled PCs 8/9 guard candidate status/presence before response reads at 10/11;
  PCs 12/13 guard S status/presence before reads at 14/15. The program checks the
  required id/complement and S id/zero relation. Candidate fail/absent tests consume
  both original outcomes once, retain S's successful present outcome, leave absent
  candidate response bindings absent, and finish failed. Pending cannot skip the
  required receive in this control flow; broader pending/S controls remain Task 3.
- **Same context and allowance:** the recursive compiled-program comparison finds
  exactly `Author.Invocations.candidate.TemplateID`,
  `Code[0].Effect.Target.Literal.Text`, and
  `Code[2].Match.Source.Literal.Text` differ. This matches the report's three
  enclosing binding locations. Parent IDs and catalogues may differ, as documented.
  Both samples use I=R3, depth2, E0, physical capacity1 and the same checks.
- **Public execution and observation:** the separate consumer has public direct
  imports and a separate module identity. It calls Create/Open/Run/Inspect/Replay;
  no replacement result, synthetic observation, projection edit or host-made
  parent decision is present. The root exposes retained data and aggregate status;
  its absent public response is accurately reported rather than invented.

## Evidence checks

Independently traversed the raw inspection tables from each parent's retained
handle values. Checked unique original intent/admission/delivery/selected-history
rows, source/caller/generation/template/operation agreement, admission input and
source-sequence agreement, accepted depth1 original invocations, publication to
observation response agreement, consumed-sequence to selected-history agreement,
and equality of typed selected response fields with the successor projection and
final parent fields. No operation identifier was reconstructed from an expected
sequence tuple. Both selected successor projections are still open.

| Binding | Selected candidate | Selected S | Dynamic I/R | Admission depths | Fixed |
| --- | --- | --- | --- | --- | --- |
| A | id1/value1 | id1/value0 | 2/2 | 1,1 | 1 |
| B | id1/value1 | id1/value0 | 3/3 | 1,1,2 | 1 |

Each reply is successful and present. Raw charge rows agree with the aggregate
counts, and effects are zero. There are 3/4 retained runs respectively. Both local
and root outcomes succeed with no pending invocation or external obligation.
The captured automatic runs report 25/32 steps and 6/9 dispatches respectively.

The archive contains twenty commands, all exit0. Inspected test JSON confirms
actual A/B construction leaves and A/B positive plus fail/absent consumer leaves
under normal and race invocations; they are not empty test selections. The archive
also retains build/vet/module/import checks and separate CLI runs. These are
attributed to the capture, not presented as fresh reviewer race runs.

Fresh reviewer commands, both exit0:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/private/tmp/alan-phase1l-go-build go test -count=1 ./examples/invocationjoin
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/private/tmp/alan-phase1l-go-build go -C examples/invocationjoin/consumer test -count=1 -tags integration ./...
```

Fresh read-only Python assertions verified the source/archive hashes, snapshot
file equality, frozen template equality, exact compiled differences, artifact byte
hashes and raw links/results described above. No broad suites, TeX build, commit,
delegation, or live provider calls were performed.

## Limits and next obligations

The evidence is adequate for the explicitly scoped two-sample construction
milestone. It remains coupled to the implementation for replay and is not an
independent semantic oracle. The separate reference author must still establish
C_inv[K1], finite bounds/rank/fairness and actual artifact correspondence before
substitution qualification. Four positive payloads, full raw negative controls,
meaningful partial joins, wrong-id conformance rejection, insufficient allowance,
pending/S controls and the acknowledged SIGKILL/fresh-open witness remain Tasks
2–3. Manuscript reconciliation and final phase gates remain Task 4. These planned
obligations are not Task 1 blockers, and this review promotes none of them to a
completed or proved result.
