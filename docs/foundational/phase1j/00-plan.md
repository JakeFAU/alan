# Phase 1J retained joins implementation plan

Question: does keeping the ordinary root at a retained join until its static peer
settles S eliminate the Phase 1I lifetime failure within the original eight
positive configurations? A new counterexample or concrete API obstruction is a
valid result. Scope and protocol are frozen in [J1](01-join-contract.md), subject
to independent review before implementation.

Baseline supplied by the controller: commit
`d68ec667d276a4bd9e1be11613905c628cfea704`, tree
`25c2e22f21181fe70d8f2b2f8bc7e44c512ef948`, branch
`codex/phase1j-retained-joins`. Controller records ancestry/clean-state evidence
including merged Phase 1I PR18 and intervening PR19 TeX fixes. Manuscript textual
synchronization is required; run Tectonic once at phase end as authorized, owned
by final qualification rather than this contract-writing workstream.

Inspected inputs: Phase 1I contract, qualification/decisions/runbook and final
review; `examples/substitution/deployment.go`, `probes.go`, `protocol/`;
`author/author.go`, `application.go`, `compile.go`, `invocations.go`;
`internal/executor/invocation.go`, `invocation_payload.go`, `messages.go` and
`program.go`; reviewer inspection additionally covers `invocation_outcome.go`;
`research/substitution/{oracle,explorer,runtime_boundary}.go`; and retained
fixture/decoder/process helpers under `internal/substitutiontest`.

## Staged work and independent ownership

- [ ] **Freeze and attack J1.** Independently review report truth/correlation,
   strict failure joining, lifetime and resource obligations, finite products,
   allowed intervention, raw mapping and process cuts. Preserve every rejected
   revision and finding. Record approved exact contract digest before authoring.
- [ ] **Ordinary context implementation.** Add `examples/join/deployment.go` and
   focused package tests. Obtain unchanged S from the public
   `substitution.New(...parallel...).Peer` result, then provision new root/peer
   artifacts with public author APIs; this avoids exporting/recompiling an
   accidental variant of S or editing frozen substitution sources. Preserve
   original 32 Set instructions and S Invoke/Await, replacing only the reporting
   suffix and root decision. Add separately named mutants/control S templates.
   Add `examples/join/consumer/` as a separate module using the existing public
   consumer pattern; no dependency changes to root module.
- [ ] **Independent model/oracle.** Separate worker owns `research/join/` and model
   evidence. Specify reference K1 product first, then independent actual artifact
   mappings for A/B and context. Enumerate the eight configurations per product;
   retain exact counts/rank and any counterexamples. The oracle imports no
   compiler/reducer/runtime. Model and context workers can proceed independently
   after the reviewed freeze; correspondence requires frozen actual artifacts.
- [ ] **Runtime/decoder/process qualification.** Add only
   `internal/substitutiontest/join_*_test.go` to reuse existing reviewed private
   row-check/process utilities. If current process child cannot select the new
   cuts, add a join-specific child/observer rather than modifying old fixtures.
   First reproduce C0 at capacities3/4, then run the distinguishing J1 matrix,
   decisive intermediate lifetime state and four actual SIGKILL/resume cases.
   Validate raw selected observations before oracle checks; retain RED decoder
   mutations and mutant rejection reasons. Fixture identity comparisons must
   reject altered A/B/S bytes or changed static topology.
- [ ] **Reconcile and review.** Write qualification, review, runbook, decisions and
   compact evidence index under `docs/foundational/phase1j`; synchronize relevant
   Markdown/LaTeX passages (not universal claims). Keep bibliography unchanged
   unless sources actually change. Obtain independent substantive review, fix and
   requalify findings, inspect complete diff, then narrowly scoped commits and a
   draft PR. Do not merge/release or message people.

## Context builder interface

The implementation/harness handoff uses:

```go
type Options struct {
    SMode string         // empty or "answer" preserves original S; controls: fail, absent, pending
    PrematureReport bool // control: report after S Invoke but before its AwaitInvocation
    WrongRequest bool    // control: report the other id in {1,2}
}
func New(candidate author.Template, config substitution.Config, options Options) (substitution.Deployment, error)
```

`New` accepts parallel shape (or empty normalized to parallel), retains delay32
and static root/peer provisioning, and returns original candidate in
`Candidate`/`Entry` and actual selected S in `Peer`. Empty options define primary
C_join. Controls never change A/B or primary S. The premature mutant commits S
and reports a plausible successful outcome with its actual handle before awaiting
that outcome; the oracle must reject this even if the root trusts it. No delay
override is needed. Freeze source/artifact identities per option.

## Gates and reproducibility

Go/toolchain/module values were read from root `go.mod`: Go1.27.3,
`github.com/mattn/go-sqlite3 v1.14.52`, CGO required. No Makefile is present.
Use `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly`, a writable
external GOCACHE, and no provider credential. Commands below are planned, not
reported executions. Capture command/environment, exit status, exact selected
cases, source/artifact hashes and baseline/final identities. Freeze actual test
names before final selectors; zero selected tests are not a passing gate.

```sh
go test -count=1 -timeout=10m ./research/join ./research/substitution
go test -tags integration -count=1 -timeout=10m ./examples/join/...
go test -tags integration -count=1 -timeout=10m -run '^TestJoin' ./internal/substitutiontest
go test -race -tags integration -count=1 -timeout=30m -run '^TestJoin' ./internal/substitutiontest
go test -tags integration -count=1 -run '^TestSubstitutionRootSuccessFence$' ./internal/substitutiontest
go test -count=1 ./author ./schema ./research/semantics
go vet -tags integration ./examples/join/... ./research/join ./internal/substitutiontest
go mod verify
git diff --check
```

Run separate consumer normal/race tests and build from its nested module. Run
focused existing invocation/closure regressions using actually discovered test
names. Confirm gofmt output and package discovery. Extend gates only for remaining
risk or required repository checks. Preserve Phase 1I effect/unknown evidence
under original source labels; no fresh effect campaign is required.

## Feasibility limits to resolve with evidence

- Public static messaging hardcodes generation1 and matches peer/obligation,
  not application request id. J1 is one request per generation; explicit payload
  check and independent exact-handle truth check are necessary.
- Outcome metadata is bindable, but receipt authenticity is not transferred by
  ordinary Send. Root trusts peer; causal oracle must reject premature reporting.
- Wrong-request rejection parks rather than terminates. This invalid-peer control
  lies outside the positive profile; no general retry/registry mechanism is added.
- Existing raw validation may assume every message is an invocation reply. Extend
  only join observations additively and preserve old decoder tests.
- `execution.committed` boundaries appear suitable for exact consume/Send cuts;
  verify event emission and retained state instead of claiming success from API
  inspection. Store.Run200-step bounds may need documented reporting overhead.
- A committed report can precede peer local terminal; qualification must preserve
  this distinction and independently establish final peer/aggregate results.
- Ordinary report source authorization after peer Success can be refused as
  source_closed. Preserve this fair schedule as a potential negative; do not add
  handshake or assume successful delivery. Peer terminal is Success regardless
  of S result, because peer Failure could itself freeze shared authority early.
- Absent AwaitInvocation responses delete bound response variables. Construct
  absent report payloads with explicit zero literals after checking presence.

Nothing in this plan reports a completed experiment. The completion criterion is
the reviewed bounded result (positive or falsified), full evidence/limitations,
manuscript reconciliation and draft PR required by the assignment.
