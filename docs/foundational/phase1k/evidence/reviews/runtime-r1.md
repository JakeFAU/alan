# Task 4 independent runtime, raw mapping and process review, round 1

Spec verdict: **REJECT**. Code and research quality verdict: **REJECT**.
One actionable finding, R1 below. The concrete B/J1 paired executions, automatic
boundary and six process records were not falsified by this review. Rejection
concerns a demonstrated false acceptance in the raw decoder, not a claim that
those actual records are wrong. No phase completion, J2 runtime result or
universal liveness/recovery result is approved.

Question: Do genuine public transitions over frozen B/J1 support the claimed
first-authorization and recovery distinction, with an independent raw adapter
that rejects incorrect original-message correlation before abstraction?

## R1 — P2: validate truth before accepting the no-report branch

Location: `internal/substitutiontest/communication_evidence_test.go:65-68`;
related lookup: `communication_driver_test.go:69-76`.

`communicationReport` recognizes only a peer message whose target is `root`
and whose obligation is `phase1j_s_join`. If retained report evidence has an
incorrect target or obligation, this lookup returns an empty intent.
`decodeCommunication` then returns success before `communication.CheckTrace`
(at lines 198-200). Its reused `decodeJoin` adapter has already retained the
incorrect `report_commit` in `out.Truth`, but has not run the independent truth
checker. The malformed report is therefore accepted as an empty message prefix.

A narrow overlay diagnostic reproduced both cases using the archived genuine
preauthorization checkpoint at
`evidence/runtime/normal-initial/87b28e995978/before.json.gz`. The unmodified
checkpoint decodes successfully. For each case the diagnostic changes only the
original report's `Target` or `Obligation`, in its intent descriptor and matching
history intent representation, to `foreign-report-context`. It never changes an
authoritative database, program, production source, or runtime transition.
Both mutated inspections return a nil decoder error and zero messages. Calling
the independently frozen `communication.CheckTrace` on the returned trace then
reports both `report_provenance` and `report_mapping`. This demonstrates the
skipped validation rather than relying on an inferred threat or on a fabricated
runtime schedule.

Required correction: make every successful decoder return apply the relevant
truth/message mapping checks, including this early branch. Reject an unexpected
retained report identity rather than silently dropping it. Add regressions for
both wrong target and wrong obligation, and retain a genuinely pre-report valid
prefix so the correction does not simply forbid all prefixes. Preserve this RED
witness, rerun the affected checks, and request scoped rereview. Existing 11 raw
mutations do not cover this lookup-miss path.

## Reviewed source and evidence identities

Reviewed AGENTS.md, binding assignment, approved contract, Task 4 brief/report and
review package, all five added integration files, public consumer, journal and
process audits, evidence index/manifests/archive, inherited evidence validators,
public facade and relevant runner/message/commit/boundary behavior. The J2
construction ruling and context review were read as explicit scope limits.

Current five-source aggregate SHA256:
`07ee2145f258069aaaa169eeded371f2e1ff1cf48b344340e4681b9b5072a2d6`.
The encoding is sorted relative path, NUL, lowercase file SHA256, LF.

| Reviewed source | SHA256 |
|---|---|
| `internal/substitutiontest/communication_decoder_test.go` | `9c874a2108e787f877d6e48fd45c99eb548e45178da4e71d45afe5aecc4782e7` |
| `internal/substitutiontest/communication_driver_test.go` | `453db8e8207e9e876e3883f270110dd8af184b3e929ab309798b5406ec78efd4` |
| `internal/substitutiontest/communication_evidence_test.go` | `c8455ad20ad4f9c73de6864d7a23823ccf782ffbbbdc868d37b93a3d4eccd4ee` |
| `internal/substitutiontest/communication_process_test.go` | `60968566192b697da6b534409dba13be0934046bb71ac04db8c722d22be4a22c` |
| `internal/substitutiontest/communication_runtime_test.go` | `c3d7d4c7bfc47d0c00cb27a378a908cb8a094f8435f3018e29ef498364da8d13` |

| Reviewed authority/evidence | SHA256 |
|---|---|
| `assignment.md` | `dc9af808296a9516382c7b286f02b8e65b00042f8934ec4f208efcff2261654d` |
| `01-communication-contract.md` | `075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544` |
| `.superpowers/sdd/00-plan-phase1k/task4-review-package.txt` | `2b61ef0363ba55cdb1210c1f81cb1bb752a371e01c879d96a23619dc5eacbf3d` |
| `.superpowers/sdd/00-plan-phase1k/task-4-report.md` | `84b359605df28ba46e2e1f37c727e5da71ce7ee73c9b31d523900d1bafddff1c` |
| `evidence/runtime/source-identity.json` | `4f4cb602262bd556d1d515a1e97ecb9aaf3f98c44e95420ef8f9bd160e3e201f` |
| `evidence/runtime/evidence-index.json` | `d988867a81238ccc3fd9739f05ab2e57cf9e8aa135b964131cb6d4219bad8727` |
| `evidence/runtime/qualified-source.tar.gz` | `19f009553b0dfc555211fbba049daf66e9845e2bd3292b4ea82f2bbeab7e7a4e` |
| `research/communication/oracle.go` | `7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4` |

Independent read-only Python checks recomputed the five hashes and aggregate,
verified all 327 indexed artifact digests, all archive members against current
files, every indexed command exit/output digest, and all 131 protected baseline
file hashes. A baseline-relative `git diff --name-status` over the protected
production/frozen context/model/dependency and existing integration-test paths
was empty. This is preservation evidence; it does not treat untracked additions
as nonexistent.

The recorded complete normal and race gates bind to these exact five sources.
Their outputs contain the claimed 24 passing leaf cases each; the subprocess
entry skip is excluded. Normal elapsed 31.643 s and race 490.034 s are prior
inspectable executions, not reruns by this reviewer. The separately scoped public
normal/race visibility probe, vet, independent Python audit outputs, and semantic
Send-is-enough RED have correct command/exit/hash records. The RED actually fails
on `source_closed`. The index does not label its own audits reviewer approval.

## Public transitions and causal evidence

The driver obtains `Snapshot` and current `Authority`, reads the original
instruction bytes, and calls exported `Commit`. It supplies only actual
unconsumed observations to a receive, lets Commit check eligibility and choose
branches, and records rejected probes separately without accepting mutations.
The qualified paths have no rejected probes. No successor/branch/register or
SQL state is fabricated. Invocation admissions, real child programs, replies,
settlement and root publication use exported operations. There is no concurrent
public driver and Store.Run. The external module has no internal imports and
compiles the required inferred return types/fields; its executed original Invoke
probe is accurately distinguished from a second full scheduling campaign.

The original report is `["peer",1,37,0]`. Its truthful S operation, status,
reason, presence and response match retained S publication evidence. Genuine
peer jump/End reaches sequence 39. LATE first authorization at sequence 39 is
`source_closed`, original transport admission is `source_denied:source_closed`,
and dispatcher acknowledgment links that actual refusal. There is no report
observation/consumption, root remains open at its report Receive, and aggregate
completion is absent. EARLY authorizes at sequence 37 while open, terminates the
peer before original report admission, then consumes the accepted observation
and actually publishes success. Actual charges remain three dynamic invocations
and created runs, two static runs, five total runs and zero effects.

The main decoding path verifies original intent/authorization request identity,
source revision, retained source projection and authorization fact digest,
runner transport/admission identity, dispatch linkage, authentic observation,
selected history and consumption, report truth and aggregate state. The source
projection is selected by run/sequence, not by demanding equal activation and
historical commit epochs. These checks and the admitted/unconsumed, reordered
unrelated rows and original J1 negative controls are useful; R1 is a separate
unvalidated success path.

The journal audit is a standalone Python reader, without an ALAN reducer/model
import. Its inspected source/output binds 18 final journals and 888 genuine
commit records to archived program hashes, original PCs/instructions, authority,
selected observations and returned history. It is an independent identity/link
check, not a proof of arbitrary compiler semantics. Replay remains a historical
consistency check, not a substitute for this independent mapping.

## Automatic boundary and process evidence

Source inspection confirms that the runner services one internal delivery before
one local commit. It does not drain all pending deliveries. The actual fresh
post-Send and preauthorization restart records have only the original report
pending, all three invocation outcomes already delivered, empty terminal routes,
root waiting without an unconsumed observation, and peer's pure continuation
ready. Thus the stated exclusion is conditional on this frozen queue predicate;
it is not promoted to a general runner guarantee or a concrete automatic LATE.

Independent direct JSON inspection of all six final before/after pairs verified
the three exact predicates, matching report intent/transport/authorization,
actual ACK token and peer sequence/epoch, killed exit, different new PID, same DB
path, unchanged charges/permissions, original immutable row prefixes and root
activation 2 to 3. No hook reentry occurs: automatic pauses use the existing
signaling observer; public pauses follow completed API calls and safe inspection.

| Gate | Cut | Killed PID → fresh PID | Retained/result distinction |
|---|---|---|---|
| normal | no authorization | 78747 → 78748 | automatic succeeds; auth epoch 3/sequence 37, historical sequence 37 epoch 2 |
| normal | allowed, peer terminal | 78750 → 78763 | byte-identical allowed attempt; original delivery/consumption succeeds |
| normal | refused, peer terminal | 78768 → 78781 | exact refused auth/admission/ACK retained; root/aggregate pending |
| race | no authorization | 81204 → 81210 | automatic succeeds; auth epoch 3/sequence 37, historical sequence 37 epoch 2 |
| race | allowed, peer terminal | 81222 → 81490 | byte-identical allowed attempt; original delivery/consumption succeeds |
| race | refused, peer terminal | 81581 → 81856 | exact refused auth/admission/ACK retained; root/aggregate pending |

For refused cases, source authorization, refused original admission and dispatcher
ACK all precede the kill. Fresh public recovery explicitly repeats authorization,
delivery and acknowledgment. Original descriptor/result/source snapshot and
ACK linkage remain identical, as do observations, history, intents, facts and
root publications. There is no replacement permission, report, consumption or
charge. Public retry is correctly distinguished from automatic queue service
after dispatcher acknowledgment has removed ordinary queue eligibility.
Fresh-process replay checks the existing database without issuing effects.

J2 complete authored/runtime/automatic correspondence remains unavailable due
to the reviewed frozen branch type mismatch. No raw peer, weaker correlation,
changed prefix, or terminal probe substitutes for J2. The model-only ACK race,
available artifact fragments and public terminal probe retain their separate
classifications. Pending observations are finite observations, not proofs of
infinite waiting. Manuscript reconciliation and phase-wide gates remain the
controller's obligations.

## R1 reproduction retained in this review

Only the following narrow diagnostic was run by this reviewer. No routine Go
suite was rerun; no implementation fixes, commits, TeX runs or nested agents
were performed. The source below was placed in a temporary file and added only
through Go's overlay, at the virtual repository path
`internal/substitutiontest/communication_review_overlay_test.go`.

Temporary test SHA256:
`2e668ce488ce8dcd3a9e158d8f38aa1e08c607342f6f2562875ad4bf4a95c0f2`.
Input gzip SHA256:
`2edc6f53269aeb85640057378d53922dfca709a8a0f047f1c00e057f7f79697a`.
The test source is preserved verbatim below so reproduction does not depend on
retaining the temporary directory.

```go
//go:build integration
package substitutiontest
import (
 "compress/gzip"
 "encoding/json"
 "os"
 "testing"
 communication "github.com/JakeFAU/alan/research/communication"
 model "github.com/JakeFAU/alan/research/substitution"
 "github.com/JakeFAU/alan/runtime"
)
func TestCommunicationReviewWrongReportIdentity(t *testing.T) {
 for _, field := range []string{"Target", "Obligation"} {
  t.Run(field,func(t *testing.T) {
   f,e:=os.Open("../../docs/foundational/phase1k/evidence/runtime/normal-initial/87b28e995978/before.json.gz"); must(t,e); defer f.Close()
   z,e:=gzip.NewReader(f); must(t,e); defer z.Close()
   var raw struct {Inspection runtime.Inspection `json:"inspection"`};must(t,json.NewDecoder(z).Decode(&raw));s:=raw.Inspection
   q:=model.Request{ID:1,Value:0,Mode:"answer"}
   _,e=decodeCommunication(s,q,communication.J1);must(t,e)
   in:=communicationReport(s)
   row:=communicationRow(s,"intents","id",in.ID)
   editEvidenceJSON(t,row,"descriptor",func(v map[string]any){v[field]="foreign-report-context"})
   for _,h:=range s.Tables["history"] {
    var intents []map[string]any;must(t,json.Unmarshal([]byte(h["intents"].(string)),&intents))
    for _,v:=range intents {if v["ID"]==in.ID {v[field]="foreign-report-context"; b,e:=json.Marshal(intents);must(t,e);h["intents"]=string(b)}}
   }
   tr,e:=decodeCommunication(s,q,communication.J1)
   if e==nil {t.Fatalf("accepted mismatched original report %s as empty prefix; messages=%d; skipped oracle violations=%v",field,len(tr.Messages),communication.CheckTrace(tr))}
  })
 }
}
```

The overlay JSON maps the absolute virtual path under this worktree to
`/private/tmp/alan-phase1k-runtime-review-r1/decoder_review_test.go`.
Command, from the worktree root:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1 GOCACHE=/private/tmp/alan-phase1k-runtime-cache go test -overlay /private/tmp/alan-phase1k-runtime-review-r1/overlay.json -tags integration -count=1 -timeout=2m -run '^TestCommunicationReviewWrongReportIdentity$' -v ./internal/substitutiontest
```

Observed exit: **1**; package elapsed **0.238 s**. Both leaf cases failed:

```text
accepted mismatched original report Target as empty prefix; messages=0; skipped oracle violations=[{6 report_provenance ["peer",1,37,0]} {-1 report_mapping truth event has no original message record}]
accepted mismatched original report Obligation as empty prefix; messages=0; skipped oracle violations=[{6 report_provenance ["peer",1,37,0]} {-1 report_mapping truth event has no original message record}]
```

These are decoder counterexamples over copied inspections, not new executed
protocol histories. The original final sources and raw evidence remain intact.
