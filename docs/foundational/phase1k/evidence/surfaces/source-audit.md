# Phase 1K preliminary public-surface audit

Status: source-inspected design input; no new probe, build, test, model exploration, or runtime experiment has been executed. Profile freeze and independent review must precede those actions.

Inspected checkout: `94e97d5ba8b7ed2b3e22d0e5ab86fa91b7a2adc5`, branch `codex/phase1k-safe-termination`. This note does not reverify remote PR ancestry/status; that is controller-owned baseline work. It preserves Phase 1J's distinction between a model counterexample and an automatic-runner failure.

Question: Which closure-stable completion facts can an unchanged ordinary authored program observe, which can a lower-level artifact provisioned through the public runtime interface observe, and which remain host-side facts? The smallest boundary is the separation among committed ordinary message intent, first source authorization, destination admission, retained observation, and actual consumption.

## Three separate interfaces

| Interface | Source-inspected ability | Scope / limitation |
| --- | --- | --- |
| Ordinary `author` constructors | `Send` commits an ordinary-message intent; `Receive` waits for an ordinary message or local input; `Await` consumes a runtime adapter result; `AwaitInvocation` consumes a runtime invocation outcome. | No constructor for an authentic terminal observation, ordinary message authorization result, admission result, dispatch acknowledgment, or aggregate-root outcome. This is an inspected constructor gap, not language-wide non-expressibility. |
| Lower-level public runtime artifacts | `runtime.Artifact{ID, Semantics, Bytes}` accepts original `alan/1d/v1` JSON programs with no Author metadata. A raw `receive` can match `Kind="terminal"`, `Provenance="runtime-terminal"`, exact source/generation/obligation and empty operation. Public `Permission` can install the terminal-observation route. | Public facade omits aliases for `Program`, `Instruction`, `Match`, `Operand`, `Observation`, `Authorization`, and `TerminalDestination`. Independent raw JSON and an anonymous destination struct provide a planned route around naming omissions. External compilation/provision/execution remain untested. |
| Host-side Store dispatch/inspection | Exported Store methods expose snapshots, genuine single-instruction commitment, first authorization, delivery, dispatch acknowledgments, invocation settlement, root preparation/publication; inspection exposes retained tables. | A returned Go value or inspected row is not automatically an observation available to the retained program. Host-selected commitments must still execute the actual artifact/PC with an eligible retained observation. |

### Ordinary author surface examined

Production public author files are `author/author.go`, `compile.go`, `invocations.go`, `application.go`, and `identifiers.go`. The complete constructor declarations in `author.go:45-94` and `invocations.go:94-108` expose scalar/collection/control steps, `Send`, `Receive`, `Request`, `Await`, `Invoke`, and `AwaitInvocation`; none takes a generic observation kind/provenance or arbitrary raw instruction. `Step` and `Expr` representation fields are private (`author.go:45,53-61`), as are Artifact bytes/identity fields (`author.go:97-101`). Public Metadata aliases do not make compiled artifacts mutable through the author package; Bytes returns a copy.

Exact relevant signatures:

```go
func Send(port string, values map[string]Expr) Step
func Receive(port string, bindings map[string]string) Step
func Request(request, effect string, values, updates map[string]Expr) Step
func Await(effect, request string, bindings map[string]string) Step
func Invoke(pending, template string, values, updates map[string]Expr) Step
func AwaitInvocation(template, pending string, b InvocationOutcomeBindings) Step
func (a Application) Provision() (runtime.Provision, error)
```

The compiler initializes instruction Dst from `s.destination` (`compile.go:266-267`), but `Send` sets only kind/declaration/values (`author.go:81-83`). Send therefore does not retain its intent identity in an authored register. Its lowering is an emit of Kind message with fixed peer, generation 1, obligation, and payload (`compile.go:280-293`). Receive lowering fixes Kind/Provenance to message/ordinary-message or input/local-input (`compile.go:294-305`). A payload field named status, receipt, authorization, or terminal cannot change those runtime metadata matches.

`Request`/`Await` and `Invoke`/`AwaitInvocation` have distinct destination handles and runtime-adapter/runtime-invocation observation matches (`compile.go:307-359`). Their existence prevents a claim that author programs lack every runtime-originated observation; the gap concerns the relevant ordinary-message and static-peer-terminal facts.

`Application.Provision` derives only used ordinary message/input and adapter permissions from ports/effects (`application.go:51-94`); its public Port has Peer, Obligation, Send, Receive, Local (`author.go:36-43`). There is no terminal-observation port mode. A host can augment the returned `runtime.Provision` before creation, but the ordinary Receive instruction still matches ordinary-message, so adding a terminal route alone does not create an authored terminal wait.

## Boundary map

| Boundary | Durable implementation / evidence | Retained-program visibility |
| --- | --- | --- |
| 1. Send intent committed | `program.go:423-469` creates Intent; `store.go:781-814` commits history/intents/successor atomically. Original identity binds run/generation/sequence/ordinal. | Author Send advances without an identity/result register. Raw emit may set Dst to intent ID (`program.go:466-467`), but that alone is not authorization or delivery evidence. |
| 2. First source authorization | `messages.go:90-177`: first attempts row retains Authorization ID, original IntentID, Allowed/Reason, current Source Authority, optional ClosureOperation. Existing row returned before current source checks (`112-123`). | Host receives inferred Authorization fields. No observation insertion occurs in this path. Ordinary author Receive has no corresponding runtime result to select. |
| 3. Destination accepted/refused | `DeliverMessage` first calls retained `AuthorizeIntent`, then records ordinary admission or composition reply (`messages.go:180-225`). `admit` retains accepted/refused result and bound target (`385-492`). | Host gets `runtime.AdmissionResult`. Sender does not receive a result observation from ordinary delivery. |
| 4. Destination observation retained | Only accepted ordinary admission inserts Observation and sets ObservationID (`messages.go:474-479`). Metadata is runtime-stamped by `observationFor` (`381-383`). | Available in durable inbox; not yet read into registers. Destination need not currently be at matching Receive. |
| 5. Observation consumed | `Commit` validates original, unconsumed, run-bound selected ID, runs genuine reducer, and atomically records `observations.consumed_sequence`, `history.selected`, successor (`store.go:754-814`). Reducer checks full kind/provenance/source/generation/obligation/operation match (`program.go:362-421`). | Only this matching Receive binds payload into program data. Admission or dispatch acknowledgment is insufficient. |
| 6. Local terminal | End sets final projection (`program.go:470-477`); Commit freezes final projection plus terminal destinations derived from source-revision observation permissions (`store.go:815-824,854-866`). | Terminal source stops reducing. A separate recipient can consume an authentic terminal observation if its raw program matches it and the runtime admits delivery. |
| 7. Aggregate completion publication | `PrepareRoot` computes retained structured readiness; `SettleRoot` inserts `composition_root_publications` and a composition fact (`composition_root.go:10-85`). `Inspection.Roots` exposes local/aggregate/pending information (`composition_inspect.go:9-22,49-65`). | Host fact/publication; this path does not insert an observation into the root's inbox. Local Success and aggregate completion remain distinct. |

### Ordinary source lifetime and bookkeeping

`AuthorizeIntent` first reuses a retained authorization (`messages.go:112-123`); a new ordinary authorization refuses changed revision/generation, retirement, or non-open source (`154-165`). The invocation-reply binding is checked separately (`147-153`), and only that exception bypasses source retired/closed checks. `DeliverMessage` calls the same authorization API, so a prior retained allowed result is reused (`180-184`); the ordinary admit path does not recheck current source status. It still checks destination existence, retirement, generation, open status, route permission, and schema (`422-465`). This source inspection supports only the proposed source-lifetime hypothesis under those fixed destination/routing assumptions, pending actual paired evidence and recovery cuts.

`AcknowledgeMessage(ctx, intentID, transportID)` loads an existing message admission and checks its descriptor linkage, deliberately discarding its AdmissionResult (`messages.go:231-260`, especially `242`). It does not require Accepted, insert an observation, or consult receiver consumption. The acknowledgment table is dispatch/admission bookkeeping, including for a refusal. An ordinary protocol ACK is instead a newly committed Kind message intent with its own first source authorization and admission/consumption obligations.

`AcknowledgeTerminal` likewise validates a retained route/admission and marks delivery_status (`messages.go:309-351`); it is not receiver consumption. The minimal probe must preserve the accepted AdmissionResult and selected observation independently of this optional bookkeeping call.

## Raw terminal path and exact public method accessibility

The facade is `runtime/runtime.go:15-32,43-67`: Store and provisioning/state/result types are aliases to executor definitions, so exported Store methods remain in the method set even where their parameter/result names have no facade alias. Relevant actual declarations:

```go
func (s *Store) Snapshot(ctx context.Context, id RunID) (RunState, error)
func (s *Store) Commit(ctx context.Context, a Authority, selected string) (RunState, error)
func (s *Store) AuthorizeIntent(ctx context.Context, id string) (Authorization, error)
func (s *Store) DeliverMessage(ctx context.Context, intentID, transportID string) (AdmissionResult, error)
func (s *Store) AcknowledgeMessage(ctx context.Context, intentID, transportID string) error
func (s *Store) DeliverTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) (AdmissionResult, error)
func (s *Store) AcknowledgeTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) error
func (s *Store) AdmitInvocation(ctx context.Context, operation string) (compositionAdmission, error)
func (s *Store) PrepareInvocation(ctx context.Context, operation string) (InvocationPreparation, error)
func (s *Store) SettleInvocation(ctx context.Context, operation string) (AdmissionResult, error)
func (s *Store) PrepareRoot(ctx context.Context, rootID string) (InvocationPreparation, error)
func (s *Store) SettleRoot(ctx context.Context, rootID string) (InvocationPreparation, error)
```

Source locations: `store.go:650,715`; `messages.go:92,180,231,288,309`; `invocation.go:220`; `invocation_outcome.go:445,459`; `composition_root.go:12,49`. `RunState.Authority()` is exported (`store.go:69`). Type names above are the actual executor signatures, not invented facade exports. Inferred values such as `auth, err := s.AuthorizeIntent(...)` permit field inspection without naming Authorization.

`TerminalDestination` has exactly exported fields `Target, Obligation string` (`messages.go:29`). Planned external call:

```go
destination := struct{ Target, Obligation string }{Target: "receiver", Obligation: "done"}
admitted, err := s.DeliverTerminal(ctx, runtime.RunID{Subject: "peer", Generation: 1}, destination, "phase1k-terminal-probe")
```

By the Go assignment rule for identical underlying types with at least one unnamed type, this should avoid any import of executor. This is a source/language assessment; no successful external compile is claimed. `runtime.Inspection` has Tables, not a typed terminal-route accessor (`inspect.go:17-23`), so do not promise inference from an accessor that does not exist.

Raw artifact viability is based on: exported Artifact Bytes and Provision fields (`store.go:29-57`); acceptance of original SemanticVersion `alan/1d/v1` with nil Author (`program.go:16`; `author_metadata.go:44-52`); generic receive validation and match (`program.go:130-185,362-421`); hash validation of exact submitted bytes (`store.go:474-491`). It does not require an exported EncodeProgram, Program, or Match alias. Submit independently authored JSON and standard-library SHA256/hex identity; do not modify existing compiler output.

Terminal delivery authenticates a retained terminal obligation and exact frozen destination (`messages.go:263-286`). Its payload is only `outcome` equal to retained local status; Source/SourceGeneration are authentic run metadata. Operation is empty, and target Generation is 0 in the descriptor, binding at first admission to the current target generation. `admit` still requires an open/current non-retired receiver; `admissionPermitted` trusts the already verified frozen terminal route (`messages.go:494-497`). Stable receiver generation is an explicit probe assumption. No first ordinary-message source authorization is used for terminal delivery.

Historical internal test source `TestTerminalDeliveryAfterClosureAndLostAcknowledgment` (`messages_test.go:187-228`) already provisions this raw receiver/route and checks terminal delivery/consumption after reopening. It is an inspected historical test, not a test run in this task and not evidence that external consumers can name all required values.

## Minimal external-consumer probe to execute only after profile freeze

Suggested isolated consumer module: `examples/termination/consumer` (final placement controller-owned), modeled on the existing separate `examples/join/consumer/go.mod` boundary. Import standard library plus public `author`, `runtime`, and `schema` as needed; no `internal` imports, test-only internal hooks, reflection, unsafe operations, SQL edits, or existing artifact rewriting. Keep it one terminal accessibility probe, not a third join protocol.

1. Produce a fresh peer with ordinary `author.Compile` of one `author.Success()` instruction. Produce an independent raw original-semantics receiver artifact with exactly two instructions: terminal Receive then succeeded End. Receive match is terminal/runtime-terminal, source peer, generation 1, obligation done, empty operation, Bind outcome to peer_outcome. All four match operands must use explicit Literal values; empty operation is an explicit empty-string literal.
2. SHA256 each submitted raw artifact's exact bytes and use lowercase hex ID; for the authored peer use its exact returned bytes/ID/Semantics. Provision two subjects/runs at generation/revision 1 with nonempty owners, open projections and empty non-nil data maps. Add `runtime.Permission{Kind:"observation", Source:"peer", Target:"receiver", Operation:"terminal", Revision:1, Obligation:"done"}`. No composition root, invocation, dynamic run, external effect, or scheduler change is needed for this accessibility question.
3. `runtime.Create`, then `Snapshot(peer)` and `Commit(peer.Authority(), "")`. Retain its actual terminal RunState. `runtime.Inspect` must show the peer terminal history and terminal_obligations row with original sequence/status and the exact frozen receiver/done route. Preserve these records before delivery.
4. Call `DeliverTerminal` with the anonymous destination struct above and a fixed transport identity. Retain Accepted/Reason/Bound/ObservationID. Fail the probe on refusal rather than replacing identities. Inspect admissions and the authenticated terminal observation; require source peer/generation1, runtime-terminal provenance, exact route, actual outcome and no consumed_sequence yet.
5. `Snapshot(receiver)` then `Commit(receiver.Authority(), admitted.ObservationID)`. Require genuine receiver PC advance and peer_outcome equal the source terminal status. Inspect the exact history.selected and observation consumed_sequence linkage. Then execute the receiver's genuine End using its returned Authority and empty selected ID; verify final actual status. Optional `AcknowledgeTerminal` is separately labeled bookkeeping.
6. Close and retain the DB/raw inspection. `runtime.Replay` is an additional historical consistency check, not an independent proof of semantics. Archive external compile/provision/execution failure verbatim if any stage is blocked; do not replace this with an internal-package probe or call its absence an impossibility.

Keep this probe's supported public-driver execution label explicit. It proves accessibility of the lower-level interface only if successful. It says nothing by itself about J1 automatic scheduling, whole-protocol completion, crashes, retry generations, or finite-protocol impossibility.

## Completion fact and invoked-peer distinction

An authentic peer terminal observation is a closure-stable fact of the original source run's retained local outcome. It is not itself evidence that all of that run's ordinary messages were admitted or consumed. In a separately justified report/ordinary-ACK protocol where peer End is reachable only after consuming the correlated ACK, observing that peer's authentic terminal outcome could support an argument that the ACK was consumed. The argument requires the exact program/control/correlation mapping and stable identities; it is only a candidate for later study. The current author constructors cannot express that terminal wait; the raw-runtime probe may show the underlying capability is accessible. Do not implement that third protocol here.

Invoking the peer is a separate alternative: author Invoke retains a handle and AwaitInvocation consumes runtime-invocation evidence (`compile.go:307-329`). Composition replies are recorded separately and can be authorized after source closure under the reply exception (`messages.go:147-162,203-213`; `invocation_outcome.go:121-217`); the final structured outcome observation is inserted by `SettleInvocation` (`invocation_outcome.go:457-567`). This changes topology, invocation/run accounting and settlement obligations. It is not the unchanged static-peer solution and needs no new campaign in this phase.

Manuscript implication to reconcile after actual results: existing statements about ordinary supervisors consuming terminal observations concern the core/raw runtime calculus (`paper/alan-dissertation.tex:91,155`), while Phase 1J's report-source lifetime qualification is at `tex:347-349` / `md:528-532`. State explicitly which authoring/interface level is supported by new evidence. Do not promote the inspected author gap into a universal theorem or erase J1's eight-configuration model failure, positive automatic samples, or historical recovery qualifications.

Remaining obligations: independently review and freeze the profile map/probe; execute an external no-internal-import compile and authentic terminal delivery/consumption probe; preserve any obstruction at the exact stage; incorporate the qualified result into communication-contract, qualification and synchronized manuscript text. No production change is proposed or made by this audit.
