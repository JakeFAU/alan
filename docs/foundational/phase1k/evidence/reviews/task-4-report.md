# Task 4 — public J1 communication and exact recovery

## Question and result

The frozen B/J1 programs admit a concrete public transition ordering in which
Send commitment does not preserve first authorization after peer termination.
The paired early authorization survives genuine peer termination and subsequent
original-message admission. These are concretely tested public-API executions;
they are not an automatic-runner failure or an all-schedule completion theorem.

J2 has no complete executable artifact: the independently reviewed Task 3 peer
construction fails `branch type mismatch: string and request:invocation:peer`.
No weaker correlation, prefix change, raw peer, J2 runtime pair or J2 automatic
sample was substituted. See `evidence/context/frozen/availability.json`,
`b-peer-error.txt`, and `evidence/reviews/context-r1.md` under Phase1K. The pure
model's J2 result retains its model-only status and incomplete artifact mapping.

## Concrete observations

All primary cases use unchanged B, id 1/value 0/answer, parallel, I=R3/depth2/E0,
physical capacity 1, two static runs. Genuine B/helper/S invocations and replies
execute before the truthful original report. Actual spending is three invocation
and created-run charges, five total runs, zero effects.

| Case | Actual retained result |
|---|---|
| J1 public LATE | Peer report `["peer",1,37,0]` commits at sequence 37; original jump/End suffix reaches sequence 39 Success; first authorization records sequence 39/source_closed; original transport admission refuses source_denied:source_closed and dispatcher ACK links that refusal. No report observation/consumption; root open at its report Receive; aggregate pending. |
| J1 public EARLY | First authorization records peer sequence 37/open/allowed; genuine peer suffix terminates before report admission. Same authorization is reused; original admission accepts; root consumes original report at sequence 3 and reaches actual local/aggregate Success. |
| Fresh unchanged automatic B/J1 | Actual post-Send sequence 37 cut has no report attempt/admission/ACK, root waiting with no unconsumed observation, peer pure continuation ready, all invocation outcomes delivered, empty terminal routes. The sole pending internal key is the original report. Same child resumes and completes. |
| report-no-authorization SIGKILL | Genuine automatic post-Send cut; fresh automatic Open resumes original DB and completes. Root epoch 2→3. First report authorization source epoch 3/sequence 37 coexists with the sequence 37 history epoch 2; decoder correctly does not require their equality. |
| report-authorized-peer-terminal SIGKILL | Allowed first disposition plus genuine peer terminal and no report admission before kill. Fresh public driver reuses byte-identical attempt/snapshot, delivers original report and completes; root epoch 2→3. |
| report-refused-peer-terminal SIGKILL | Refused auth, refused original-runner-transport admission and dispatcher ACK all precede kill. Fresh public driver explicitly repeats the same authorization/delivery identities. Attempts, admissions, ACK linkage, observations, history, intents, facts and root publications stay identical; no replacement permission/charge or consumption; root/aggregate remain pending. |

The automatic result's queue premise supports only this frozen boundary: the
runner services its one pending report before its next local commit. It is not
a general drain-all-deliveries barrier. Refused recovery is explicitly public
retry after bookkeeping removed ordinary automatic queue eligibility. A finite
pending observation does not prove infinite waiting.

Each checkpoint has actual pipe acknowledgment token/PID, retained predicate,
SIGKILL exit and a different fresh-process PID. All three cuts were achieved in
both final normal and race gates, with immutable original row prefixes, unchanged
charges/permissions and normal activation renewal. No synthetic stop/redeploy,
reconstructed work, authoritative SQL edit or reentrant Store hook call is used.
Fresh-process replay leaves each recovered ledger unchanged. Replay establishes
historical internal consistency, not independent compiler semantics.

## Apparatus and raw mapping

Only five new integration test files were added:
`communication_{driver,evidence,runtime,decoder,process}_test.go` under
`internal/substitutiontest`. No production executor/compiler/schema, dependency,
frozen example/model or old test changed; the recorded preserved-source diff
command exited 0. There were no commits, TeX invocations or nested agents.

The driver uses exported Snapshot/Authority/Commit, original invocation
admission/preparation/settlement, message authorization/delivery/acknowledgment,
and root preparation/settlement. It inspects the original instruction and lets
Commit execute branches and resolve original composition bindings. Receives
select actual unconsumed observations only. A rejected probe would be separately
recorded and required to leave Inspection and Snapshot unchanged; none occurred
in the qualified decisive paths.

An eagerly appended/flushed JSONL journal retains each manual pre-PC, projection,
current Authority, raw original instruction, selected observation, returned state
and new history row. Original program bytes are saved by content ID outside the
per-step records. Exact decisive snapshots are complete. The standalone Python
journal audit passed for 18 final journals and 888 genuine commit records,
checking program bytes/hash, pre-instruction, authority, selected original
observation and successor/history linkage without calling an ALAN reducer.

The decoder reuses validateRetained/decodeJoin and the unchanged join truth
oracle under explicit J1 source/obligation identity, then binds the original
intent, first authorization request/result and source evidence digest, original
runner transport/admission, dispatch ACK, observation, actual selected
consumption and aggregate. Eleven raw attacks reject denial-as-delivery,
admission-as-consumption, swapped intent/auth/selected identity, changed source
sequence/epoch/descriptor revision, wrong ACK linkage, wrong report correlation
and a false root summary. Honest refused/admitted-unconsumed prefixes and
reordered unrelated rows remain valid. Original premature/wrong-request J1
controls are rejected using their actual unchanged source artifacts.

A separate external module in evidence/runtime/public-consumer has no internal
imports. It compiles every required public method and inferred unaliased result
field and executes the original root Invoke plus actual invocation authorization
and admission. Its sole commit retains original instruction/prestate/history.
This visibility probe is not a second full join campaign or the terminal probe
already qualified by Task 3.

## Executed qualification and identities

Final five-source aggregate SHA256:
`07ee2145f258069aaaa169eeded371f2e1ff1cf48b344340e4681b9b5072a2d6`.
`evidence/runtime/source-identity.json` defines the aggregate encoding, all five
file hashes, baseline 94e97d5, Go 1.27.3 darwin/arm64, and frozen oracle v2 hash
`7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4`.
`qualified-source.tar.gz` preserves the exact qualified apparatus input with SHA256
`19f009553b0dfc555211fbba049daf66e9845e2bd3292b4ea82f2bbeab7e7a4e`.

Both complete focused gates used identical final five source hashes:

```
go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunication' ./internal/substitutiontest
go test -race -tags integration -v -count=1 -timeout=30m -run '^TestCommunication' ./internal/substitutiontest
```

Normal exited 0 in 31.643 s; race exited 0 in 490.034 s. Each selected four qualifying
top-level tests and 24 passing leaf cases. The subprocess-entry top-level skip is
explicitly excluded. Exact leaf names, commands, outputs, exits and source
bindings are in `evidence/runtime/evidence-index.json` and
`checks/{normal-initial,race-initial}.json`; the labels' word initial does not
mean earlier bytes: these records are checked equal to the final five sources.
Earlier pair/process development stages keep their original separate labels.

The following also executed and exited 0:

```
go test -C docs/foundational/phase1k/evidence/runtime/public-consumer -v -count=1 ./...
go test -C docs/foundational/phase1k/evidence/runtime/public-consumer -race -v -count=1 ./...
go vet -tags integration ./internal/substitutiontest
python3 docs/foundational/phase1k/evidence/runtime/audit-journals.py normal-initial race-initial
python3 docs/foundational/phase1k/evidence/runtime/audit-process.py normal-initial race-initial
```

The audit outputs/records are `checks/journal-final.{stdout,json}` and
`checks/process-audit-all-prefixes.{stdout,json}`. The latter explicitly records all six
final kills and epoch 3/history2 facts. The opt-in false Send-is-enough assertion
is retained: ALAN_K_RED=send-is-enough with the late pair selector exits 1 with
`first authorization source_closed`; `checks/red-send-current.json` binds that
expected RED to the same final five source hashes.

Every Go gate used GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
GOFLAGS=-mod=readonly CGO_ENABLED=1, GOCACHE=/private/tmp/alan-phase1k-runtime-cache,
no OPENAI_API_KEY, and a fresh absolute Phase1K ALAN_I_ARTIFACTS directory. The
command runner is archived as evidence/runtime/run-check.py. The evidence index
hashes raw artifacts; it does not promote standalone audits into reviewer approval.
Formatting produced no gofmt differences; git diff --check passed.

## Remaining obligations

Independent public-legality/raw-causal/process review is pending. Parent owns the
combined phase-wide vet/build/regression gates, manuscript reconciliation and PR.
J2 complete artifact/public-runtime correspondence remains unavailable, not fixed.
All communication findings are bounded to this single-host fixture and stable
identity/routing/schema assumptions; no universal completion, distributed
recovery or ordinary-protocol impossibility theorem is claimed.


## Revision 2 — R1 decoder lookup correction and complete requalification

The first independent runtime review returned REJECT for P2 R1. Its exact finding
and copied-inspection counterexamples remain in evidence/reviews/runtime-r1.md.
The earlier sections describe the original source snapshot and its historical
runs; their old decoder qualification is superseded by this appendix. The review
did not falsify the concrete paired/process records, and no old result is relabeled
as a run of the corrected sources.

The strict report lookup could miss an original peer message with changed Target
or Obligation while decodeJoin had already retained its malformed report_commit
in Truth. The early no-report return skipped communication.CheckTrace, accepting
that malformed trace as a zero-message prefix. R1 was verified before fixing it:
checks/red-lookup-r2 records exit 1 for both wrong-Target and wrong-Obligation,
while the genuine pre-report control passed. Returned traces had report_provenance
and report_mapping violations. These are decoder attacks on copied inspections,
not new runtime histories or authoritative database edits.

The fix adds three lines to run CheckTrace before the early successful return;
the existing main successful return already runs it. Both successful decoder
paths now validate the relevant truth/message mapping. The only other change is
TestCommunicationDecoderReportIdentity in communication_decoder_test.go: a real
public root Invoke establishes a genuine pre-report valid prefix, and two copies
of a genuine post-Send/no-authorization prefix exercise wrong Target and wrong
Obligation in both the original intent and matching history representation.
The exact scoped change is revision2/r1-fix.patch.txt. Driver, process and runtime
source bytes are unchanged; no production/old test/model/context source changed.

All new records live under docs/foundational/phase1k/evidence/runtime/revision2/.
The pre-fix RED input is retained in red-qualified-source.tar.gz, SHA256
`a66c1c03a9e7d874636853f29ac467086849ef0ab81221c9883371eb92aed1c2`,
with its exact five hashes in red-source-manifest.json. lookup-green-r2 exits 0
for all three regression/control leaves after the fix.

Corrected five-source aggregate SHA256:
`e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1`.
The corrected decoder-test hash is
`59e3751bacfd411ec6857ea18a3d4251a0e4865a20365aaca5194840d64ed8b2`;
the evidence-decoder hash is
`231136367ced2f1682c031f6636a52b415c30e3c9d4286570fef42f7ea9e87f6`.
The other three hashes match the original qualification. revision2/source-identity.json
records all five, the unchanged oracle v2, complete gate scope and prior-source
identity. revision2/qualified-source.tar.gz preserves the corrected input, SHA256
`58412ebb92f85c2ca173126902d088fd68bf16d2f263ea843504a524e66db0af`.

The complete focused commands were rerun on those exact same corrected five bytes:

```
go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunication' ./internal/substitutiontest
go test -race -tags integration -v -count=1 -timeout=30m -run '^TestCommunication' ./internal/substitutiontest
go vet -tags integration ./examples/communication/... ./research/communication ./internal/substitutiontest
```

normal-r2 exited 0 in 36.533 s; race-r2 exited 0 in 535.773 s. Each selected five
qualifying top-level tests and 27 passing leaf cases, including the new three
prefix/identity cases, all original raw controls, both public orderings, unchanged
automatic B/J1 and all three real ACK/SIGKILL/fresh-Open cuts. The subprocess-entry
skip is excluded. vet-r2 exited 0. There is no latest-source race coverage caveat:
the whole focused runtime/decoder/process selection ran on the corrected bytes.
The original semantic Send-is-enough RED remains false; red-send-r2 records its
current-source exit 1 for source_closed. Exact commands, selected leaf names,
environment, exits, output hashes and source hashes are in revision2/checks/ and
revision2/evidence-index.json. The unchanged external consumer/dependencies were
not rerun; their original source-bound visibility result retains its earlier scope.

The same offline local Go 1.27.3/CGO/readonly settings, absent provider credential
and temporary GOCACHE were used, with fresh absolute revision2 artifact roots.
Standalone revised-run audits also exited 0:

```
python3 docs/foundational/phase1k/evidence/runtime/revision2/audit-journals.py normal-r2 race-r2
python3 docs/foundational/phase1k/evidence/runtime/revision2/audit-process.py normal-r2 race-r2
```

checks/journal-final-r2 binds 20 journals and 998 genuine commit records to exact
original instruction bytes, authority, actual selection and retained history.
checks/process-final-r2 inspects all six new process cuts, every original immutable
table prefix and exact refused auth/admission/ACK reuse. It again observes first
authorization epoch 3 at sequence 37 alongside historical sequence 37 epoch 2,
unchanged charges/permissions, root epoch renewal, actual killed/new PIDs, original
allowed delivery/consumption completion and truthful refused root/aggregate pending.
Replay remains read-only consistency evidence. None of these audit counts is a
compiler proof, probability or reviewer approval.

All 328 preexisting runtime evidence files were checked byte-identical against
revision2/prior-evidence-sha256.json. In particular the original qualified-source
archive, source-identity.json, evidence-index.json and all initial command/raw
records remain immutable. This report was appended; its original bytes remain
an unchanged prefix. Formatting and diff checks passed. No commits, TeX,
production changes, protocol workaround or nested agents were used.

Independent scoped rereview of R1 and its correction remains required. The J2
construction obstruction and all execution-profile/finite-evidence limitations
remain unchanged; this decoder correction makes no new completion theorem claim.
