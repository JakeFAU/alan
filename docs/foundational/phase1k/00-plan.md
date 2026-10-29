# Phase 1K required communication and safe termination implementation plan

> **For agentic workers:** Use `superpowers:subagent-driven-development` task by
> task, with independent reference and substantive review work. Autonomous work
> is authorized; independent design review is the next gate.

**Goal:** Determine when ordinary-message source termination strands required
communication, and which completion facts existing public surfaces expose.
**Architecture:** Additive public-driver experiments, one ordinary J2 ACK context,
an independent finite model/oracle, three process cuts and one external raw
terminal-observation probe. Preserve production semantics and historical evidence.
**Tech stack:** Go1.27.3, CGO, pinned SQLite v1.14.52, existing local fixtures.
**Spec:** [assignment.md](assignment.md), [communication contract](01-communication-contract.md).
**Baseline:** Controller-verified merged main
`94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`, tree
`a20478fa31b333c6ff0be43fc9b43826b54bbb3a`; reviewed PR21 head
`1d13bc238aada500c3b9608d400435e235fedc0b` has the same tree. Preserve J1's
experimental/integration identities and inherited PR19 formatting/PR20 correction.

## Global constraints

- Production executor/compiler/schema/dependencies/accounting/scheduler stay unchanged.
- Preserve C0/K1/J1, A/B/S artifacts, original peer prefix and historical source labels.
- Primary witness B/id1/value0/answer/I=R3/depth2/E0, fixed root+peer2.
- Public steps execute original instructions using current authority and actual
  observations; no private reducer, register edit or concurrent Store.Run.
- Freeze/review profiles and J2 before experiments; one ACK candidate, no repair rounds.
- Three exact B/J1 cuts use real ACK/SIGKILL/fresh Open with original identities.
- Separate external terminal probe, no internal imports or third join protocol.
- One final Tectonic invocation after manuscript review; accepted warning policy.
- Narrow reviewed commits and draft PR against main; no merge/release/paid calls/messages.

## Review focus

- Dispatcher acknowledgment of denial must not become delivery or consumption.
- Original message/first-authorization/selected-observation identities cannot be swapped.
- Public manual reachability must not be promoted to automatic-runner failure.
- Negative J2 root Failure can freeze peer before ACK consumption.
- Mutex-held hooks must not reenter Store; prove cuts from committed records.

## Inspected inputs

Task1 read AGENTS.md, assignment, relevant J1 plan/contract/review/runbook/decisions,
model and retained B process record; join context and separate consumer; public
runtime/author surfaces; executor messages, runner, store, inspection, invocation
outcome/root settlement; join evidence/process helpers, boundary facilities and
scoped manuscript. Separate surface/model preparation reports live under
`.superpowers/sdd/00-plan-phase1k/`. Inspection establishes API signatures and a
conditional sole-pending-report automatic exclusion, not a fresh protocol result.

## Task 1: Freeze profiles and protocol

**Files:** this plan, `01-communication-contract.md`, task1 report. Controller owns
review records and digests in `03-review.md` and `evidence/reviews/`.
**Produces:** J1 pair, full J2 branch/correlation policy, finite bounds, public
step legality, source boundary table and exact process predicates.

- [ ] Independently review source map, manual legality, exact queue predicate,
  ACK negative/pending paths, raw terminal probe and process cuts.
- [ ] Preserve REJECT findings, fix and re-review; record approved plan/contract
  hashes before model/context experiments or interpreting new results.

## Task 2: Independent model/oracle

**Create:** `research/communication/{model,explorer,oracle}.go`, focused tests,
`README.md`; `docs/foundational/phase1k/evidence/model/`.
**Interfaces:** `Explore(Config, substitution.Variant) Census` and
`CheckTrace(Trace) []substitution.Violation`; Trace combines explicitly mapped
`join.Trace` truth facts with original message/first-auth snapshot/admission/
observation/dispatcher-ACK/selected-consumption records. Worker publishes exact
Go record types before Task4 handoff. Only pure research/std-library imports.

- [ ] Write negative prefix-oracle checks and preserve the deliberately false
  completion assertion as RED; freeze reference source before new J2 artifacts.
- [ ] Extend unchanged J1 primitives with ACK0..4 and terminal guards; check frozen
  domains/rank. Enumerate all maximal states in one id1/value0/answer/I=R3/depth2
  K1 local/helper product, no pruning/cutoffs; retain counts and decisive traces.
- [ ] After Task3 artifact freeze, independently map A/B/context instructions and
  finite stutters. Preserve original J1 eight-configuration evidence unchanged.
- [ ] Run focused normal/race model/oracle checks and independent graph review.

## Task 3: J2 context and public terminal probe

**Create:** `examples/communication/deployment.go`, `deployment_test.go`, separate
`consumer/{go.mod,go.sum,main.go,main_test.go,terminal_test.go}`; context/surface evidence.
**Interfaces:** `New(candidate author.Template, config substitution.Config,
options Options) (substitution.Deployment, error)`; Options includes SMode
(empty/answer original; fail/absent/pending selected existing J1 controls) and
separately identified wrong-report/ACK request/handle controls as needed. Return
existing Deployment fields Candidate/Entry/Peer/FixedRuns/Provision. Root module
dependencies remain unchanged; nested consumer uses local replace and existing pins.

- [ ] `TestCommunicationArtifacts` pins original A/B/S bytes, topology/limits,
  32 Sets/Invoke/Await and new-only suffixes; compare catalogue by ID, code in order.
- [ ] Implement exact J2 names, ports, obligations and full branch policy.
  `TestCommunicationPolicy` checks negative/absent/pending and wrong correlation
  paths, ACK-before-decision, peer wait and no extra dynamic/effect work.
- [ ] Export compact artifact bytes/hashes and obtain independent correspondence review.
- [ ] `TestTerminalObservationPublicConsumer` provisions Success peer plus raw
  alan/1d/v1 receiver and frozen terminal grant, uses real terminal delivery and
  actual selected consumption with anonymous destination struct. Retain successful
  access or exact compile/provision/execution obstruction; no constructor repair.
- [ ] Run package and separate-module normal/race checks and consumer build.

## Task 4: Public pairs, automatic profile, decoder and process cuts

**Create:** `internal/substitutiontest/communication_{driver,evidence,runtime,
decoder,process}_test.go` only, reusing existing integration helpers; runtime/process evidence.
**Interfaces:** `driveCommunication(t *testing.T, path, protocol, order string)
runtime.Inspection` for J1/J2 late/early; `decodeCommunication(runtime.Inspection,
substitution.Request, string) (communication.Trace, error)`;
`communicationCut(runtime.Inspection, string) bool`. Finalize trace types with Task2.
Runtime mutations use public APIs exclusively; private imports only signal/read apparatus.

- [ ] `TestCommunicationJ1PublicPair` executes genuine candidate/S paths, original
  peer report and remaining instructions, first-authorization LATE/EARLY pair and
  actual admission. Record every pre-PC/authority/selection/post-history step.
- [ ] `TestCommunicationJ2PublicPair` realizes decisive B ACK LATE and preauthorized
  control, preserving root local vs aggregate status, peer wait and actual charges.
- [ ] `TestCommunicationAutomatic` runs unchanged B/J1 and J2 separately, retaining
  exact report/ACK queue and readiness facts. No identity search or drain-barrier claim.
- [ ] `TestCommunicationDecoder` rejects denial-as-delivery, admission-as-consumption,
  swapped original identities, changed auth snapshot and wrong correlation before
  normalization; reuse existing report truth checks under explicit source identity.
- [ ] `TestCommunicationRecovery` and `TestCommunicationProcessChild` implement
  the three named B/J1 cuts, actual ACK/SIGKILL/fresh Open, original immutable-row
  prefix/charges and renewed epoch; no mutex reentry. Automatic resumes first cut,
  public driver resumes authorized/refused terminal cuts. Before the refused cut,
  retain refused authorization, refused admission under the original runner
  transport and dispatcher acknowledgment. After reopen, repeat the same
  authorization/delivery; require identical authorization descriptor/result/source
  snapshot and admission descriptor/result, unchanged acknowledgment linkage, no
  replacement rows/observations/consumption/charges/permissions, and actual root/
  aggregate pending. Retain unsupported cuts.
- [ ] Run focused normal/race/public/process gates, read-only replay and independent
  legality/raw-mapping/process review. Stop J2 after its decisive bounded result.

## Task 5: Reconcile, review and draft PR

**Create:** `02-qualification.md` through `05-decisions.md`, compact evidence index,
source/artifact manifests and decisive trace archive. **Modify:** scoped Markdown
and LaTeX manuscript; BibTeX only if references actually change.

- [ ] Answer all five assignment criteria with profile/evidence classifications,
  preserved failures and precise remaining gaps; no universal impossibility claim.
- [ ] Reconcile scoped manuscript claims, preserve inherited corrections, obtain
  independent final APPROVE after correcting/rechecking retained REJECT findings.
- [ ] Check actual gates, full diff, unchanged production/frozen sources and hashes;
  run one `tectonic alan-dissertation.tex` from paper after substantive review.
- [ ] Narrow commits, push codex/phase1k-safe-termination, draft PR against main,
  attach URL; report baseline/final identities, validation and next question. No merge.

## Planned verification

Root go.mod pins Go1.27.3 and sqlite3 v1.14.52; no Makefile/CI test workflow found.
Use `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
GOCACHE=/tmp/alan-phase1k-go-build`, with OPENAI_API_KEY absent and absolute evidence
paths. Record actual commands/cwd/env/exits/source hashes and nonzero leaf selection.
The commands below are planned, not performed by the design worker.

```sh
go test -count=1 -timeout=10m ./research/communication ./research/join
go test -tags integration -count=1 -timeout=10m ./examples/communication/...
go test -tags integration -count=1 -timeout=10m -run '^TestCommunication' ./internal/substitutiontest
go test -race -count=1 -timeout=10m ./research/communication
go test -race -tags integration -count=1 -timeout=30m -run '^TestCommunication' ./internal/substitutiontest
go test -count=1 -timeout=10m ./author ./schema ./research/semantics
go test -count=1 -timeout=10m -run '^(TestInvocationRootClosureSerializesWithAdmission|TestInvocationReplyExceptionDoesNotAuthorizeOtherEffects|TestInvocationSuccessWaitsForCommittedUnadmittedChild|TestReplayRejectsChangedSelectedObservation)$' ./internal/executor
go vet -tags integration ./examples/communication/... ./research/communication ./internal/substitutiontest
go mod verify
git diff --check
```

Run `go test -count=1 ./...`, `go test -race -count=1 ./...`, and `go build ./...`
from the consumer module. Check test discovery/gofmt. Reuse old evidence only in
its exact scope; controller baseline checks are attributed to their own logs.
