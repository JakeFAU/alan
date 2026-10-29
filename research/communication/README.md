# Phase 1K independent communication model and oracle

The frozen single J2 ACK protocol moves the ordinary-message source-lifetime
race from report to ACK. The reference K1 product at id1/value0/answer,
I=R3/depth2/E0 has 1,408 reachable states, 2,886 edges and four maximal states:
two complete and two retain root local Success with an open peer waiting for
an ACK whose first authorization was refused as source_closed. A valid finite
trace falsifies the proposed all-maximal-states-complete claim. This does not
establish failure of automatic Store.Run or impossibility of other protocols.

The model reuses unchanged research/join and research/substitution primitives.
It changes only the two context terminal guards and adds ACK commit, first
authorization, admission/observation and selected consumption. The peer may
terminate only after ACK consumption; the root may terminate after ACK commitment.
Report commitment, authorization, admission and consumption remain separate.
The original 32 Set prefix remains explicit. No production package is imported.
All child invocations and charges remain in the unchanged reference graph.

The intrinsic rank is join.Rank(state.Join)+AckStage, namely
100*substitution.Rank(base)+sum(privatePC)+privateRegister[e][7]+delay+
reportStage+ackStage+aggregatePublished. Its frozen conservative cap is 23202:
base rank <=231, private PCs <=60, private register <=1, delay <=32,
report/ACK stages <=8 and aggregate <=1. The observed K1 range is 800..8541.
All edges, including revisits, strictly increase rank. Finite coordinate domains
are checked on every reachable state. Histories and actor sequence counters are
not a rank. The exact abstract maximal path count is 245247050897402400; this is
not a concrete instruction-schedule count or a probability. All four maximal
states are retained in the census. There are no cutoffs, rank/domain/resource/
lifetime/protocol failures, unexpected stuck states or cycles in this product.

The source-lifetime abstraction begins with one committed ordinary message and
an always-open destination with valid stable route/generation/schema. Its eight
states/eight edges have two maximal outcomes: first allowed authorization stays
allowed after source closure and admits; first source_closed refusal stays refused.
Reattempts are explicit identity-preserving stutters excluded from progressing
path counts. This is an abstraction under stated assumptions, not crash evidence.

CheckTrace(Trace) is a prefix-safety oracle. It calls unchanged join.CheckTrace
on explicitly mapped truth facts, then validates message, first authorization,
admission, observation, dispatch acknowledgement and selected-consumption links.
Types are in oracle.go. A Consumption requires the original selected observation
and receiver execution sequence. The raw adapter must authenticate those facts
against successful history.Selected and consumed_sequence before normalization.
Dispatch acknowledgement of refused admission is valid bookkeeping; it is not
accepted delivery or consumption. Omitted optional records mean absent evidence.
First authorization includes original epoch/revision/sequence/status; retries
must preserve the entire record. A renewed epoch may exceed the Send epoch.
Only same-actor sequences and retained identity links impose order; independent
actor arrays may be reordered without changing the verdict. Honest pending
prefixes remain valid. Aggregate completion requires both local terminal facts.

Scope: the primary positive answer product and associated causal prefixes. J2
negative branches involving root Failure and shared-scope freeze are not modeled
as this product. No claim about their eventual peer Success is made. New J2
compiled instruction correspondence remains a separate post-freeze obligation;
A/B candidate primitives retain their prior independent mapping. No new context
artifact/runtime output was used to derive the frozen reference transitions.

Run with Go1.27.3:

```sh
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1k-go-build \
  go test -v -count=1 -timeout=2m ./research/communication
```

Set ALAN_COMMUNICATION_MODEL_EVIDENCE to an existing absolute directory to emit
compact census and source-lifetime JSON. Add -race for apparatus race checks.
Run python3 docs/foundational/phase1k/evidence/model/reproduce-red.py to reproduce
the rejected completion hypothesis (expected exit1). Its source/output are
retained separately from the passing regression suite. Exact commands, source
freeze, reuse lineage and evidence hashes are retained in that evidence directory.

The initial reference oracle freeze was rejected in model-r1 review: it omitted
peer-local report-Send/first-authorization ordering before ACK consumption and
accepted two impossible cyclic normalized histories. The graph census was not
invalidated. Revision2 adds those strict causal comparisons, retains regressions
for both witnesses and the equal-Send boundary, and preserves pending prefixes
and independent-array reordering. Original source/evidence remain immutable;
corrected source, RED/GREEN checks and commands are under evidence/model/revision2.
The record types and model graph are unchanged. Review status is tracked in the
phase review document; passing tests alone do not constitute reviewer approval.
