# Ordinary context implementation report

Question: can the public retained root await the predeclared static peer's report,
while preserving C0's candidate and S invocation paths? This workstream implements
J1 and checks concrete public-API samples. It does not establish an all-schedule
completion guarantee or authenticate a peer assertion as a runtime receipt.

The approved contract digest checked before implementation was SHA256
`a950450d33ab928ccb875f8c98de83a83c956d8759fa0f39a71afe1c06c52a81`.
The final frozen `examples/join/deployment.go` digest is SHA256
`0bcbff9113b9e11b148f076e839b5279b28c833e77fbe0069fab14902ce3249e`.
Other final source digests are in `source-sha256.json`. No context source changes
are planned after this freeze unless qualification or review requests a fix.

## Implementation and preserved boundary

`examples/join.New(candidate, config, Options)` returns the original public
`substitution.Deployment` type. Empty shape normalizes to parallel; other shapes
are rejected. Empty options or SMode=answer obtains original S directly from
`substitution.New(...).Peer`. A/B and S are never reauthored in this context.
Root and peer remain two statically provisioned generation-1 members; limits,
allowed templates and candidate inputs remain unchanged. The sole new grant is
ordinary peer-to-root messaging on obligation `phase1j_s_join`.

The primary root's first two compiled instructions are exactly C0's Invoke and
AwaitInvocation, including original pending handle and outcome response bindings.
The primary peer's first 34 compiled instructions are exactly C0's 32 Sets,
S Invoke and S AwaitInvocation. The structural test compares these compiled
instructions and the entire resource-root declaration. Candidate and S template
bytes are equal between C0 and C_join for both A/B. These checks were executed;
they are bounded identity/prefix evidence, not a compiler-correctness proof.

Root then receives the report before checking either result. It checks request
identity, candidate status/presence/id/complement, and S status/presence/id/zero.
Wrong-request consumption parks at a further Receive in the frozen one-report
profile. Peer reports actual bound S status/reason/presence/response and original
handle. Absent branches Send literal zero fillers because AwaitInvocation deletes
absent response bindings. Peer always ends with local Success after reporting.
Send commitment remains separate from source authorization and peer termination.
The observed primary S handle was `["peer",1,33,0]`, below the 256-byte field bound.

Control options retain separately identified S failure/absence/pending artifacts,
a wrong-request report, and an ordinary premature report sent after S Invoke but
before S AwaitInvocation. The premature mutant later awaits S and terminates;
its early report still violates J1 even when the trusting root succeeds.

## Frozen artifacts and exact comparisons

`artifacts/` contains 45 exports: raw template/program bytes and public provisions
for C0 A/B, C_join A/B, and the five separate controls. `artifacts/sha256.json`
identifies exact exported bytes. `identities.json` records original/new static
program IDs and unchanged candidate/S template IDs. The primary new static peer
program ID is `6a77bf8322f8bf227e7c78597d6e753d8d9390f3c981fec7f2bdc59814ea71ef`.
New root IDs are A `d7fa07a0913dc035c5a1e6b47a1d663064184f4818bb1e38e91d9d3ca157e216`
and B `4c5ea75d4e08467042f3f25491ac341a5e0347652207d9970ffc6be3b4220292`.

`a-root-compiled.patch`, `a-peer-compiled.patch`, `b-root-compiled.patch` and
`b-peer-compiled.patch` give exact normalized JSON diffs of compiled artifacts
against C0. Normalization only pretty-prints and sorts JSON object keys; compiled
instruction arrays and metadata remain intact. `context-source.patch` compares
explicit original construction functions to new construction functions; adjacent
source excerpts preserve their exact text. This source comparison names the
factorization difference rather than claiming linewise source equivalence.

Re-export command, from repository root:

```sh
ALAN_JOIN_EXPORT="$(mktemp -d /tmp/alan-phase1j-context-export-XXXXXX)" go test -count=1 -run '^TestJoinArtifactExport$' ./examples/join
```

The exporter only runs when explicitly selected with the output environment
variable. Its default skip is intentional and does not stand in for qualification.

## Executed checks and observations

All Go commands used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1j-go-build`. Tests/builds made no
network/provider calls. Every final command listed below exited zero.

- RED: the structural preservation test failed with `Phase 1J context not
  implemented` using the initial New placeholder; `red.log` preserves it.
- `go test -tags integration -count=1 -v ./examples/join`: final output is
  `runtime-final.log`. Ten focused runtime controls, the C0 A/B negative and
  C_join A/B accounting samples, structural preservation and invalid configuration
  checks pass. The default artifact-export test is explicitly skipped.
- `go test -race -tags integration -count=1 ./examples/join`: passed;
  `context-race.log`.
- `go vet -tags integration ./examples/join/...`: passed with no diagnostics;
  `context-vet.log`.
- `go test -count=1 -run '^TestSubstitutionRootSuccessFence$' -tags integration
  ./internal/substitutiontest`: passed while recognizing the unchanged historical
  failure; `historical-negative.log`.
- Separate module `examples/join/consumer`: `go test -count=1 -v ./...`,
  `go test -race -count=1 ./...`, and
  `go build -o /tmp/alan-phase1j-join-consumer .` passed. Normal/race logs are
  `consumer-normal.log` and `consumer-race.log`. The consumer uses only exported
  APIs, exercises A/B init/run/inspect/replay/recover for id2/value1, and preserves
  a wrong-request root as open across recover/replay.
- `gofmt -w examples/join` was applied; `git diff --check` passed.

At physical capacity one and unchanged operational ceilings MaxSteps=200 and
MaxDispatch=200, concrete C_join A/B samples succeeded with dynamic charges
A=(2 invocations,2 created runs,0 effects), B=(3,3,0). Fixed runs remain 2.
The A primary sample took 65 steps and 7 dispatches. C0 samples retain local root
Success, peer Failure and aggregate Failure. Known candidate failure/absence and
S failure/absence produced root Failure after report consumption, peer Success;
S absence binds no peer_id and reports response_present=false with zero fillers.
Candidate pending leaves root open even when peer reports. S pending leaves root
and peer open, including when the candidate already failed. Wrong request leaves
root open and peer locally succeeded. The premature report can produce local and
aggregate success; this is the intended causal-oracle negative, not a passing J1
peer contract.

## Scope and remaining obligations

Only `examples/join/` and this evidence directory were changed by this worker.
No old source, executor/compiler/schema, module dependency, closure policy or
accounting code was modified. Root-module go.mod/go.sum remain untouched.
The separate consumer repeats existing pinned requirements in its own module.

This workstream does not claim the decisive retained lifetime cut, causal raw
adapter validation, resource refusal matrix, crash/restart witnesses, stale or
mutated transport evidence, or model completeness. Those belong to the independent
model and runtime apparatus workstreams. The ordinary report source authorization
race is deliberately preserved: a fair first authorization after peer terminal
can refuse source_closed. The concrete completion samples do not erase that
all-schedule failure boundary. Final substantive review and manuscript
reconciliation remain with the controller.


## Review correction: provision catalogue ordering

Independent context review v1 returned specification APPROVE and quality REJECT
on P2: `TestJoinConfigurationControls` compared independently constructed
Deployments using the incidental order of `Provision.Programs`. Existing public
provisioning gathers invocation catalogues from application maps, so equivalent
program catalogues can have different slice orders. The reviewer reproduced nine
failures in 50 repetitions; this worker independently reproduced eleven failures
in 50 repetitions (Go test exit 1), retained in
`review-v1/reproduced-order-red.log`. The prior green sample did not qualify
repeatability of this assertion. The rejected source, report and all three
manifests were preserved under `review-v1/` before correction.

The test now sorts copied Programs slices by exact ID, then applies the same
whole-Deployment `reflect.DeepEqual`. All other fields, exact artifact bytes,
semantics and identities remain checked; instruction order is never normalized.
`TestJoinDeploymentComparison` additionally verifies acceptance of reordered
catalogues without mutating either input, and rejection of changed program IDs,
bytes, semantics, a missing program and changed fixed-run accounting.

Executed with the previously recorded offline Go environment:

```sh
go test -count=50 -run '^TestJoin(ConfigurationControls|DeploymentComparison)$' ./examples/join
go test -tags integration -count=1 ./examples/join
```

Both corrected commands exited zero; logs are `order-fix-repeat50.log` and
`order-fix-integration.log`. This correction changes test apparatus only.
`deployment.go` and all frozen compiled artifacts retain their original hashes.
The source/evidence manifests were refreshed; historical v1 manifests remain
inspectable. Independent re-review is required before approval.

The raw frozen provision snapshots are exact historical serialized bytes.
Re-exporting can change only the incidental Programs array order, so fresh raw
provision hashes need not match those snapshots. Re-export into a fresh temporary
directory as shown above; compare program/template bytes and IDs exactly, and
compare provisions after sorting only Programs by ID. Preserve every other field
and all instruction arrays. Do not overwrite the frozen exports to conceal this
variation. The frozen artifact manifest was not regenerated for this correction.
