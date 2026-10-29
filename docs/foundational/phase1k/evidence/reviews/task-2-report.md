# Task 2 independent model/oracle report

Status: initial reference implementation complete and ready for independent
review. Post-Task3 new J2 artifact correspondence is deferred as instructed.
No new J2 compiled artifact or runtime output was inspected. No production,
research/join, research/substitution, compiler, schema, dependency, manuscript or
historical Phase1J file changed. No commits, paid calls or nested agents.

Question: can the single ordinary ACK establish all-schedule completion, and
which causal facts distinguish retained allowed authorization from consumption?
Result: bounded model-checked counterexample. The reference product has 1,408
states, 2,886 edges, four maximal states (two complete, two ACK-refusal pending),
zero rank/domain/resource/lifetime/protocol failures, cycles, cutoffs or unexpected
stuck states. Every maximal state is retained. The exact abstract path count is
245247050897402400, not a probability or concrete instruction count. Model root
local Success with an open peer remains distinct from aggregate completion.

The original id1/value0/answer/I=R3/depth2/E0 K1 local/helper product was executed
first and is the only candidate product executed so far. Both complete and
pending maxima spend two or three dynamic invocations/created runs, zero effects,
with two fixed provisioned runs separate. Rank is unchanged join.Rank+AckStage,
observed800..8541 within the frozen conservative cap23202. Every edge, including
revisits, was checked. A separate eight-state/eight-edge/two-maximal-state H-auth
abstraction preserves first allowed or refused authorization through source
closure, with an open destination and stable route/schema/grants assumed.

The shortest retained negative explicitly orders original report consumption,
ACK commitment, root Success, first refused ACK authorization, refused admission.
Peer remains open and aggregate absent. A paired complete witness retains allowed
ACK authorization before root Success, then original ACK admission and consumption.
The rejected all-maximal-states-complete assertion is executable via an overlay
and exits1 with the exact 2-of-4 completion result. No abstract trace is labeled
an automatic-runner failure or concrete public-driver witness.

API: research/communication/oracle.go exports CheckTrace(Trace)
[]substitution.Violation. Trace has Profile, Truth join.Trace, AckObligation,
Messages and Aggregate. Message records hold the original intent, actor/generation,
commit sequence, exact normalized Report/Ack payload, optional first authorization,
actual admission/observation/dispatch-ack/selected consumption, and immutable retry
copies. First Authorization stores original Epoch/Revision/Sequence/SourceStatus;
new activation epoch need not equal Send's epoch. Admission.ID is the original
transport/admission identity and accepted admission Reason is "accepted".
Consumption includes exact original IntentID, ObservationID, Receiver, Generation
and successful receive Sequence. The raw adapter must establish those from actual
history.Selected and consumed_sequence before normalization. Dispatcher ack of a
refusal is valid bookkeeping; it does not discharge receiver consumption.

Unchanged join.CheckTrace supplies truthful-report/S-handle/absence/candidate/
report ordering and final decision checks. Its source and all reused Go sources
have exact hashes in reference-freeze.json. Additional focused adversarial tests
reject denial-as-delivery, admission-as-consumption, foreign message/auth/selected
observation, changed first source sequence or retained admission, stale generation,
wrong payload/correlation, invalid terminal order and aggregate with an open
member. Honest admitted-but-unconsumed and source-refusal prefixes remain valid;
independent actor arrays may be reordered. Oracle applicability is the frozen
positive communication profiles, not a complete model of negative root-Failure
shared-scope cancellation behavior.

Freeze identities:
- Contract: 075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544.
- reference-freeze.json: afa248134d846a0891bf3ec7caa1bca180f2f17fb0d66ebf193bd6db41715802.
- oracle.go: 2ec81e819c6a73ca08f885ad3693566b44fd42f31d6926bf750bab68c4b4466e.
- Exact frozen bytes: evidence/model/reference-source.tar.gz; hashes in freeze.

Executed checks, all with OPENAI_API_KEY absent, GOTOOLCHAIN=local,
GOPROXY=off, GOSUMDB=off, GOFLAGS=-mod=readonly,
GOCACHE=/tmp/alan-phase1k-go-build, Go1.27.3 darwin/arm64:
- go test -v -count=1 -timeout=2m ./research/communication: exit0.
- go test -race -v -count=1 -timeout=2m ./research/communication: exit0.
- go vet ./research/communication: exit0.
- go test -v -count=1 -run '^TestOracle' ./research/join: exit0, unchanged checks.
- python3 docs/foundational/phase1k/evidence/model/reproduce-red.py: expected exit1.

commands.json records exact post-freeze arguments, environment, UTC starts/ends,
expected/actual exits, outputs and hashes. oracle-red.txt and model-red.txt retain
preimplementation failing checks; reference-green-v2.txt is the last pre-freeze
normal run and normal/race outputs bind the same frozen source. Census and
source-lifetime JSON, summary.json and evidence-manifest.json are compact evidence.

Remaining: independent graph/oracle review, new J2 instruction/stutter mapping
once Task3 artifacts freeze, selected A/B product/resource correspondence, raw
adapter validation and public/manual/automatic/process evidence. A/B and J2
context correspondence must not be claimed completed from these K1 counts.

Pre-freeze iteration disclosure: oracle-red.txt ran the 23 initial adversarial
checks against an intentionally empty CheckTrace stub and failed because checks
were absent. model-red.txt ran the census/paired-witness checks against the empty
Explore stub. These are implementation REDs, not semantic counterexamples.
reference-green.txt then passed the implemented graph and initial oracle. Before
freeze, self-inspection added accepted/refused admission-reason consistency,
required nonempty distinct trace identities, an explicit renewed-epoch/J1 reuse
check, and an honest admitted-but-unconsumed prefix check; reference-green-v2.txt
passed those additions. Types did not change. The source freeze includes these
changes. No correction followed the freeze. The separate completion-hypothesis
RED is the rejected semantic claim, executed against the frozen implementation
via overlay; its expected exit1 has a different meaning from the stub failures.
No further source edit is planned before independent review.

## Independent R1 correction — revision2

Independent model-r1 review returned REJECT: the initial oracle accepted two
impossible peer histories, with ACK consumed at sequence1 before report Send34,
or first report authorization at sequence35 equal to ACK consumption35. Existing
identity checks did not close the peer side of the round-trip causal order.
The retained model graph/census was not invalidated. Initial source archive,
freeze, command records and every original evidence-manifest entry were verified
unchanged and preserved. The REJECT and its diagnostic remain in model-r1.md.

Before implementation, TestOraclePeerRoundTripCausality reproduced both failures
and the Send-equality boundary (ACK consumed at34). The valid control passed;
all three malformed cases failed the rejection assertion, exit1, retained as
revision2/peer-causality-red.txt with exact test source and red-command.json.
The oracle now requires report.CommitSequence < ack.Consumption.Sequence and,
when present, report.FirstAuthorization.Sequence < ack.Consumption.Sequence.
These are same-peer sequence comparisons joined through the original message
identities, not a cross-run total order. The record types and graph are unchanged.
The regression checks reversed actor/message arrays too. Existing honest pending,
admitted-unconsumed, reordered-actor, epoch-renewal and J1 truth checks remain.

Corrected source was frozen before any new J2 artifact/output inspection:
- revision2/reference-freeze.json SHA256:
  7ef775362562faa6b7ac1c4a364638545160be57b75daed81577b5417e0e7868.
- Corrected oracle.go SHA256:
  7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4.
- Exact corrected bytes: revision2/reference-source.tar.gz, member hashes in freeze.

Actual corrected-source results: focused regressions exit0; communication normal
and race suites exit0; communication vet exit0; semantic false-completion overlay
expected/actual exit1. Commands, exact pinned/offline environment, UTC times,
outputs, exits and source identities are in revision2/{red-command,green-command,
commands}.json. The final normal/race tests still enumerate 1408 states, 2886 edges,
four maxima (two complete/two ACK-refusal pending) with zero invariant failures.
No duplicate census campaign or historical relabeling was introduced; revision2
summary binds the unchanged original compressed census by hash. No production,
J1/substitution, artifact, compiler, schema, manuscript, dependency or API change.
No further source edit is planned before the required scoped independent rereview.
