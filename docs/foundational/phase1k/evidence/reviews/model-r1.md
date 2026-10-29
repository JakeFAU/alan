# Phase 1K Task 2 model/oracle review r1 — REJECT

Spec compliance: **REJECT**. Code/research quality: **REJECT**.

The bounded reference graph and interpretation of its completion counterexample
are supported by the inspected source and recorded evidence. The new causal
oracle has a blocking false-negative: it accepts impossible same-peer histories.
This review does not invalidate the graph census. Correct the oracle, retain this
rejection and diagnostic, and obtain a focused rereview before treating Task 2's
model/oracle gate as approved.

## Scope and question

Reviewed the initial frozen reference implementation, before new J2 artifacts.
The question is whether the single ordinary ACK permits root termination that
strands its peer, and whether the raw prefix oracle correctly distinguishes
commitment, first authorization, admission, selected consumption and completion.
The declared domain is id1/value0/answer, I=R3/depth2/E0, the K1 local/helper
product, truthful participants and stable routing/generations/grants, with no
external closure. Eventual service does not convert refused authorization into
permission.

Read repository AGENTS.md, the Task 2 brief/report/review package, the binding
assignment and approved contract, original join/substitution model and oracle,
new communication sources/tests/README, reference archive, census/witnesses,
H-auth state evidence and command outputs. Inspected the relevant inherited
Markdown/LaTeX Phase 1J discussion. This is a source/evidence review, with one
narrow diagnostic; it is not a fresh run of the normal/race census suites.

A/B and new-context instruction/stutter correspondence is explicitly deferred
to the Task 3 artifact gate. No examples/communication directory existed at
review time. This legitimate dependency is not a finding. Raw adapter validation,
public-driver legality, unchanged automatic scheduling, process recovery and
final manuscript reconciliation remain later gates.

## R1 — reject cyclic ACK histories using the peer's own sequences

**Blocking finding:** `research/communication/oracle.go:217-224` validates ACK
consumption identity and its position before receiver termination, but never its
position after the same peer's original report commitment or first report
authorization. The cross-message checks at lines 246-264 impose
root report-consume < root ACK-commit < root terminal and ACK-consume < peer
terminal, leaving the opposite side of the round trip unchecked.

Two minimal mutations of the existing valid `goodTrace()` demonstrate this:

1. Set `Messages[1].Consumption.Sequence = 1`. The peer now consumes the ACK at
   sequence 1, consumes S at 33, sends the report at 34, and terminates at 36.
   All original intent/auth/admission/observation links remain unchanged.
   `CheckTrace` returns no violations and accepts aggregate success.
2. Leave ACK consumption at peer sequence 35, but set the report's first source
   authorization sequence to 35 and update its retry copy identically. Again
   `CheckTrace` returns no violations. That source snapshot asserts the peer has
   already committed sequence 35 before the original report is first authorized,
   although the same sequence 35 consumes the ACK whose creation depends on that
   report's later admission and root consumption.

Both are impossible under the frozen protocol: report Send → first report
authorization → report admission → root report consumption → root ACK Send →
ACK admission → peer ACK consumption. Closing that path with the supplied peer
sequence order creates a causal cycle. This is entirely a same-actor ordering
obligation plus existing identity links; no fabricated total order across
unrelated histories is required. The first authorization snapshot is allowed to
equal its Send sequence, but must precede the subsequent ACK-consumption sequence.

The diagnostic tests assert rejection and both fail (actual exit 1). The valid
control is checked first and passes. These are malformed normalized-history
witnesses exposing oracle weakness, not evidence that the runtime generated an
impossible history. A raw adapter's obligation to authenticate retained records
does not make a causally inconsistent normalized trace safe for the independent
oracle to endorse; the extra check also detects a faulty sequence mapping.

Required correction: validate the peer side of the round trip, including
`report.CommitSequence < ack.Consumption.Sequence` and, when that first report
authorization exists,
`report.FirstAuthorization.Sequence < ack.Consumption.Sequence` (or an equivalent
causal-acyclicity check). Preserve honest pending prefixes and arbitrary array
ordering. Add focused regressions for both counterexamples and a valid control;
keep original J1 sources unchanged. Re-freeze the corrected reference source
before new J2 artifacts and rerun only affected required checks.

## Graph and evidence assessment

- `model.go:37-53` retains every successor from unchanged J1 primitives except
  the two approved terminal guards. Report obligation remapping does not change
  successor states. ACK stages at lines 57-98 separately commit, authorize,
  admit/retain an observation, and consume. Root Success remains enabled before
  first ACK authorization; peer Success requires ACK consumption.
- The primary finite domain and intrinsic rank agree with contract lines 261-282.
  Control/resource/sequence coordinates are checked by `InDomain`; payload and
  identity values derive from fixed inputs and unchanged finite primitives.
  Rank is unchanged `join.Rank + AckStage`, not a path/sequence rank. The
  conservative arithmetic is 100*231+60+1+32+8+1=23202.
- `explorer.go:62-147` enumerates every queued state without success pruning or
  cutoffs; rank is checked for every edge before seen-state lookup. Every
  no-successor state is retained and categorized. Strict increase gives a
  topological order for the path-count calculation at lines 155-172 and excludes
  cycles in this finite checked graph. The later witness-only deterministic
  suffix does not prune the graph.
- Recorded normal/race outputs and `census-K1.json.gz` agree: 1,408 states, 2,886
  edges, four maxima, two complete and two ACK-refusal/open-peer pending; no
  reported rank/domain/resource/lifetime/protocol violations, cycles, cutoffs or
  unexpected stuck states. Observed rank is 800..8541. All four maximal states
  are retained; the two local/helper branches spend two/three I and R, E0.
  These counts are attributed to inspected prior execution, not a reviewer rerun.
- The retained negative orders report consumption, ACK commitment, root Success,
  refused first ACK authorization and refused admission. The peer stays open and
  aggregate is absent. The paired witness has allowed first ACK authorization,
  then root Success, then admission/peer consumption and aggregate success.
  Root local success is kept distinct from composition completion.
- H-auth's eight states/eight progressing edges/two maxima match its small
  transition definition (`model.go:129-159`). Retained allowed and refused
  dispositions survive source closure. Its always-open receiver and stable
  route/schema/grants assumptions are explicit. Retry stutters are excluded
  from progressing path counts. This establishes an abstract source-lifetime
  property, not end-to-end delivery or crash evidence.
- The raw oracle distinguishes original intent, first authorization snapshot,
  admission, retained observation, dispatcher bookkeeping and selected
  consumption. Retry comparisons preserve full normalized records; allowed auth
  is checked against source status/terminal sequence. Existing negative tests
  cover wrong identities/payloads, refusal-as-delivery, admission-as-consumption,
  changed retry records and aggregate publication with an open member. Honest
  refusal and admitted-but-unconsumed prefixes are accepted. These strengths do
  not discharge R1's omitted causal edge.
- The reference package imports only pure research packages and standard library.
  All 26 reused join/substitution source digests match both the freeze and HEAD.
  No tracked diff exists in original J1 source/evidence, examples/join or the
  manuscript. The unchanged J1 oracle is called directly; its truth/correlation
  tests have separately recorded exit-0 evidence under the current freeze.

## RED interpretation and recorded checks

`oracle-red.txt` and `model-red.txt` are implementation REDs against empty stubs;
they are not semantic counterexamples. `completion-hypothesis-red.txt` is the
separate semantic RED against the frozen implementation: the executable
all-maximal-states-complete assertion fails because only two of four maxima
complete. Its expected and actual exits are 1. The passing regression suite
correctly expects the two pending maxima and does not erase that rejected claim.

Verified all 18 evidence-manifest entries, all command-output hashes, all six
new source hashes and all six archived source members, with no mismatches.
`commands.json` binds the same reference-freeze digest to these recorded commands:

- `go test -v -count=1 -timeout=2m ./research/communication`: exit 0.
- `go test -race -v -count=1 -timeout=2m ./research/communication`: exit 0.
- `go vet ./research/communication`: exit 0.
- `go test -v -count=1 -run '^TestOracle' ./research/join`: exit 0.
- `python3 docs/foundational/phase1k/evidence/model/reproduce-red.py`: expected and
  actual exit 1.

No routine suite was rerun during this review. Read-only file/source-manifest
inspection and SHA256 verification exited 0. The only added executable diagnostic
is retained verbatim below, including exact command, environment, source, output,
UTC times and exit. It used a Go overlay in /tmp; implementation files remained
unchanged. The Python wrapper that recorded it exited 0 while preserving the
child Go test's exit 1. No implementation fix, commit, TeX build or nested agent
was performed. Only this review report was added to the worktree.

## Exact reviewed identities

- Baseline: `94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`.
- Assignment: `dc9af808296a9516382c7b286f02b8e65b00042f8934ec4f208efcff2261654d`.
- Approved contract: `075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544`.
- Reference freeze: `afa248134d846a0891bf3ec7caa1bca180f2f17fb0d66ebf193bd6db41715802`.
- Frozen archive: `9d81edbfbb8dcb19ffa619f1fbf8e4ea8d1e99ef102e25dc294982e830323c26`.
- Task brief: `cf69922f0bf22f15a2d7836c38c4f5146cfeae265d53788038b0f78c1212a440`.
- Task report: `90f7e0c18f1f46aa2c41e9dabe3a3121bcc3b8bcea6d6ea499f1dbd34f3910c6`.
- Review package: `318574f11482d077da91fb4e9e11e90a43f63452eb5f06afd6768966d123db9d`.
- Reused J1 oracle: `dbf22396cee24e59b0ac5311815ffc036be356f16f84bc192e1dc3c1c31b0b30`.

| Frozen new source | SHA256 |
|---|---|
| `research/communication/explorer.go` | `ba5f401b26754b8591a64fa5acfaa27bce30a32b8b52519c195c79eaaf057222` |
| `research/communication/model.go` | `e8771d0c88e59472d9e6296e1dc348c29b4e48f1cfa33f72cad1771fe4fdd4ca` |
| `research/communication/model_test.go` | `9a62c49e34655a3a56db9e9c9af7c949a634738e5d9f2f2fd69b693f31210146` |
| `research/communication/oracle.go` | `2ec81e819c6a73ca08f885ad3693566b44fd42f31d6926bf750bab68c4b4466e` |
| `research/communication/oracle_test.go` | `51203136defedfa85c6575ddc6ebd3a9bb994078df8174e4fd788bbc520b8c50` |
| `research/communication/README.md` | `36f3daac4562ab983d806358e0455291f78a004488470b00cbaed0304d29bff2` |

The freeze records the exact remaining reused-source digests. All 32 current
source files still matched after the diagnostic. Original sources and failed
hypotheses are preserved.

## Focused diagnostic record

To reproduce without changing repository source, write `overlay_source` below to
`/tmp/alan-phase1k-model-r1/model_test.go`, map the absolute repository
`research/communication/model_test.go` to it in a Go `Replace` overlay, and invoke
the recorded argv in the recorded cwd/environment. The unchanged oracle test
file supplies `goodTrace()`.

```json
{
  "argv": [
    "go",
    "test",
    "-overlay=/tmp/alan-phase1k-model-r1/overlay.json",
    "-v",
    "-count=1",
    "-run",
    "^TestReviewAckCausalOrder$",
    "./research/communication"
  ],
  "cwd": "/Users/jacob/.codex/worktrees/phase1j-retained-joins/alan",
  "started_utc": "2026-10-29T15:29:20.051210+00:00",
  "finished_utc": "2026-10-29T15:29:20.863095+00:00",
  "actual_exit": 1,
  "expected_exit": 1,
  "environment": {
    "GOTOOLCHAIN": "local",
    "GOPROXY": "off",
    "GOSUMDB": "off",
    "GOFLAGS": "-mod=readonly",
    "GOCACHE": "/tmp/alan-phase1k-go-build"
  },
  "OPENAI_API_KEY": "absent",
  "source_freeze_sha256": "afa248134d846a0891bf3ec7caa1bca180f2f17fb0d66ebf193bd6db41715802",
  "overlay_source": "package communication\nimport \"testing\"\nfunc TestReviewAckCausalOrder(t *testing.T) {\n cases := []struct{name string; mutate func(*Trace)}{\n  {\"ack_consumed_before_report_send\", func(tr *Trace) { tr.Messages[1].Consumption.Sequence=1 }},\n  {\"report_authorized_after_ack_consumed\", func(tr *Trace) { tr.Messages[0].FirstAuthorization.Sequence=35; tr.Messages[0].Reauthorizations=[]Authorization{*tr.Messages[0].FirstAuthorization} }},\n }\n for _,tc:=range cases { t.Run(tc.name,func(t *testing.T){\n  tr:=goodTrace(); if v:=CheckTrace(tr);len(v)!=0 { t.Fatalf(\"control: %v\",v) }\n  tc.mutate(&tr)\n  if v:=CheckTrace(tr);len(v)==0 { t.Fatal(\"impossible cyclic communication history accepted\") } else { t.Log(v) }\n }) }\n}\n",
  "overlay_source_sha256": "0f3e0efeb02abc6be5304e40d291f28277dbabc869131365210e67775c217a78",
  "output": "=== RUN   TestReviewAckCausalOrder\n=== RUN   TestReviewAckCausalOrder/ack_consumed_before_report_send\n    model_test.go:11: impossible cyclic communication history accepted\n=== RUN   TestReviewAckCausalOrder/report_authorized_after_ack_consumed\n    model_test.go:11: impossible cyclic communication history accepted\n--- FAIL: TestReviewAckCausalOrder (0.00s)\n    --- FAIL: TestReviewAckCausalOrder/ack_consumed_before_report_send (0.00s)\n    --- FAIL: TestReviewAckCausalOrder/report_authorized_after_ack_consumed (0.00s)\nFAIL\nFAIL\tgithub.com/JakeFAU/alan/research/communication\t0.183s\nFAIL\n",
  "output_sha256": "4b9de41091c12868a30c622f790cad091407cfcecc886e02bb070ec5d9d93247"
}
```
