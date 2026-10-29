# Task 3 result — frozen J2 authoring obstruction; terminal access succeeds

Question: can the frozen ordinary report/ACK context express its exact retained
S-handle check through current author constructors, and can an external public
consumer consume an authentic terminal observation?

## Result and scope ruling

**Concretely tested construction obstruction:** the full intended J2 peer fails
`author.Compile` with the exact diagnostic:

```
steps/37:if/then/0:if: branch type mismatch: string and request:invocation:peer
```

The ordinary ACK receive binds `ack_s_operation` as string, while the genuinely
retained `peer_pending` keeps static kind `request:invocation:peer`. The exact
required IfEqual cannot compile. Source inspection locates strict branch-kind
comparison at `author/compile.go:674-687`, handle assignment at `:734`, and handle
serialization erasure only at typed payload/list boundaries at `:528-533`.
Ordinary Set preserves the request kind. The existing list roundtrip would need
compiler-prepended collection initialization, violating this frozen peer prefix;
that alternative was source-inspected, not implemented or experimentally tested.
This is one exact frozen construction obstruction, not universal non-expressibility.

The controller explicitly ruled to preserve this obstruction: no collection
workaround, prefix revision, weakened correlation or compiler repair. Accordingly,
`examples/communication/deployment.go` retains the complete attempted root/peer
public definitions. `New` returns the actual compile error and an empty Provision,
so callers cannot accidentally execute the old J1 provision as J2. The initial
J1-forwarding harness RED and first real peer-compile failure are retained separately.

**Concretely tested public terminal access:** a separate external module, with
no direct internal imports, provisions an authored one-Success peer and an
independently encoded `alan/1d/v1` receiver. The peer genuinely terminates at
sequence1, its `receiver/done` terminal route is retained, anonymous-destination
`DeliverTerminal` returns Accepted, and the receiver selects that original
observation at sequence1 and reaches Success at sequence2. The observation's
runtime-terminal provenance, exact source/generation/route/outcome, unconsumed
admission state, consumed_sequence and history.selected linkage are checked.
Both normal and race executions passed. The built terminal-only executable also
ran successfully. This demonstrates the lower-level public runtime path only.
It neither adds an author terminal constructor nor establishes an ordinary ACK
protocol, composition completion, general delivery or impossibility result.

## Available and unavailable evidence

- `evidence/context/frozen`: exact A/B candidate, original S/control templates,
  independently compiled J2 root bytes, original J1 peer bytes, sorted-ID original
  catalogue/resource-root manifests, exact peer errors and SHA256 manifest.
  Root candidate Invoke/Await instructions match J1 in order. Original template
  bytes are obtained through public `join.New`; no A/B/S/control reconstruction.
- `evidence/context/attempted`: full first-attempt builder and the desired artifact
  and runtime-policy test source. The proposed runtime checks are evidence of
  intended tests, **not executed full-J2 policy qualification**.
- `evidence/context/source-identity.json` pins the seven current context/consumer
  source/module files, available frozen artifacts, baseline HEAD and consumer
  executable. `evidence-index.json` links all commands/environments/exits and traces.
- `evidence/surfaces/terminal-probe-reviewed`: final-source peer/receiver bytes,
  actual Provision, frozen terminal route and before/admitted/consumed/final
  inspections, AdmissionResult, replay result and per-file hashes. Earlier direct
  probe/executable-refactor evidence remains separately identified.

No complete J2 peer artifact, complete J2 Provision, J2 runtime trace, measured
J2 resource charges, negative-root peer freeze observation, or full instruction
correspondence exists. Those branch policies remain intended/source-inspected,
not passed by the expected-failure regression. A/B successes and J1 historical
results are not relabeled. No new model source/output was inspected by this worker.

## Checks actually run

Toolchain: `go version go1.27.3 darwin/arm64`. Every command used
`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
GOCACHE=/tmp/alan-phase1k-go-build`; OPENAI_API_KEY was removed. Exact argv, cwd,
filtered environment, source hashes, stdout/stderr and exit codes are retained
under `evidence/context/checks`.

| Final applicable check | Recorded label | Exit |
| --- | --- | --- |
| `go test -count=1 -v ./examples/communication` | context-normal-final | 0 |
| `go test -race -count=1 -v ./examples/communication` | context-race-final | 0 |
| consumer `go test -tags=integration -count=1 -v .` | consumer-normal-reviewed | 0 |
| consumer `go test -race -tags=integration -count=1 -v .` | consumer-race-reviewed | 0 |
| consumer `go build -o /tmp/alan-phase1k-terminal-consumer .` | consumer-build-final | 0 |
| explicit final-source terminal test with absolute export path | terminal-final-freeze | 0 |
| built external executable with a fresh retained DB | consumer-executable | 0 |
| formatting / original source diff / copied go.sum comparison | formatting / preserved-sources / consumer-pins | 0 |

Artifact export was separately selected and passed at context-obstruction-freeze;
its default skip in ordinary package runs is not counted as runtime qualification.
Final context tests contain nonzero artifact/obstruction/input subtests; final
consumer gates select actual terminal delivery/consumption. Initial harness RED
was exit1 for missing J2 artifact/ports/ACK; first-green was exit1 for the genuine
compiler obstruction; consumer-cli-red was exit1 before executable implementation.
Those failed attempts remain retained. No repository-wide gate is claimed here;
controller owns broader validation and manuscript reconciliation.

## Changes and remaining obligations

Changed only new `examples/communication` context/tests and its separate
terminal-only consumer, this task's context evidence and terminal-probe evidence.
The root dependency files and original author/runtime/schema/executor/J1/A/B/S
sources were unchanged (`git diff --exit-code` passed on those paths). Consumer
uses existing Go/sqlite3 pins and identical existing consumer go.sum bytes.
No production/compiler/facade/schema change, TeX edit, commit or nested agent.

Independent review must challenge the obstruction, root-only artifact mapping
and terminal evidence classification. Complete J2 model/artifact correspondence
and concrete ACK-race realization remain unavailable at this boundary. Parent
must reconcile this authoring limitation and successful lower-level terminal
access into phase documents/manuscript without promoting the independent model
result to concrete public or automatic J2 behavior.

## Final source identity

Current deployment.go SHA256 is `b0f68dd7ad34c193251cfe9a7d2450d907d91e3b9fd3d3661025378f677d671e`. The full seven-file manifest is `evidence/context/source-identity.json`; `final-source-check.json` verifies every final normal/race/build/export record against those actual current bytes. Current source reproduces all sixteen available frozen artifact hashes exactly (`available-artifacts-final.json`).

The earlier available-artifact-freeze builder SHA256 was `7f0cdabf070fb1898613fe178a28f449d1def11a098ab09576085dc2d7705e89`; the separately retained original full attempted builder is `51ffabbcaa1eacd24537ef05d6dfcdb51cf1931b835820dde2f4f695b03fbd2a`. These historical labels are preserved, not relabeled as final source. Formatting, explanatory comments and the empty-Provision error return account for the later current example; the available compiled root/original artifact bytes did not change. No code edits followed the final context/consumer gates; only evidence and this report were written afterward.
