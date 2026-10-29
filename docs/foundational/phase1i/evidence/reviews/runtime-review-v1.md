# Phase 1I independent runtime/public-consumer/process review

Verdict: **REJECT** pending the decoder correction below and independent rereview.

Question: do the bounded runtime witnesses actually execute ordinary retained programs, preserve observed identities/presence/charges across their declared boundaries, and support the stated sampled results without using replay as the semantic oracle?

## Source and scope

Reviewed workdir `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`, HEAD `82573b1eaf349f9567dc34d91f6d8ddcbb6a3177`, committed tree `158d2a24ece0b14f99018921797a7f6f6a45a465`; Phase 1I changes were uncommitted. Read AGENTS.md, assignment, frozen 01-substitution-contract.md, complete runtime stream report, all `internal/substitutiontest/*.go`, `examples/substitution/{probes,effect}.go`, standalone consumer source/module, fixture manifests, underlying Store.Run, ACK, commit, admission, outcome-delivery, lifecycle, final reservation and root-fence boundaries. Candidate/context authorship and complete model review are separate review scopes.

Qualification manifest SHA256: `4334306c968543bd7c5101e618b693d51fe10645588019300415c12893acfb80` (`evidence/runtime/fixtures-qualification-v1.json`). Every runtime/consumer implementation hash matched this manifest. The sole full-manifest mismatch was `research/substitution/probes_test.go`, a separately changed model test not compiled into this harness dependency. No production runtime/executor/boundary/author/schema/effects diff exists relative to the stated base.

## Actionable finding

**[P2] Decode logical root cancellation from its actual retained evidence** — `internal/substitutiontest/fixtures_test.go:283`.

`normalize` defines `Context.Closed` as `len(s.Tables["composition_closures"]) > 0`. That is false in every one of the nine real root CancelRun schedules in both normal-v1 and race-v1. Their records contain an accepted lifecycle result with reason `closed`, a root projection with status `cancelled`, and an immutable cancelled root outcome, but no `composition_closures` row. For example `normal-v1/fb5320350148/after.json.gz` (TestSubstitutionClosure/b/helper-commit) and the corresponding race record report `RootStatus:"cancelled", Closed:false`.

The frozen model's `Closed` flag denotes logical root cancellation, not presence in that particular table. This is an actual raw-to-normalized disagreement, and the root-success-fence test's `!n.Context.Closed` guard is consequently not a reliable check that no logical cancellation occurred. The existing nine closure tests only run CheckTrace and do not assert this field, so all 236 leaves can pass despite it. This does not invalidate the actual retained cancellation/endpoint results, but it prevents approving the current normalization evidence.

Required correction: derive logical cancellation from the accepted lifecycle transition/root state (and any other declared logical-closure facts), retaining the distinction between root **success** fencing later work and explicit cancellation. Add assertions on the real cancelled records and on the successful root-fence controls; retain these original incorrect normalized records as pre-review evidence. Do not change production semantics or rewrite the original counterexample. Requalify the affected exact source and regenerate new-version evidence rather than replacing v1.

Reproduction independent of the Go checker: `/tmp/alan-phase1i-review-records.py` reads the original gzip records. Its output `/tmp/alan-phase1i-review-records-result.json` identifies all 18 incorrect closure flags (nine each normal/race). Both real records and unchanged decoder source establish the discrepancy; no invented invalid program is involved.

The controller also relayed a separate model-review finding about consumption observations being reconstructed from publications (`fixtures_test.go:248–253`). I inspected the source and agree that this substitution occurs, but the model reviewer owns the mutation reproduction and detailed finding. It must also be resolved before combined approval; this report does not count their experiment as one I executed.

## Checks actually performed

- Independently parsed original normal-v1.jsonl.gz and race-v1.jsonl.gz. Each has exactly **236 passed leaf cases**: matrix190, probes26, pure recovery4, closure9, effects5, root fence2. Neither has selected skips or failures. A/helper-commit is explicitly inapplicable and absent, and ProcessChild is a subprocess entry helper, not a claimed extra experiment.
- Read and checked actual original records in both profiles, not only summaries: all190 matrix identity/parent/publication/response-presence/charge records; all16 boundary ACKs; all16 retained before/after prefixes; six actual OS-network-denied replay children per profile. Crosschecked operations, parent links, member identities, artifact/program identity, namespace/root identity, nondecreasing generation-preserving activation/sequence and spent counters, original rows, PID/token/store/scenario/phase matches and real SIGKILL exit signals.
- Four pure recoveries use different fresh PIDs, unchanged test executable digest, retained program IDs, increasing root epoch, original unconsumed outcome at the pre-consumption cut, and Open+Run without resubmission. All race children record `race_instrumented=true` and use the parent race binary. Replay children actually launch `/usr/bin/sandbox-exec -p '(version 1)(allow default)(deny network*)'` with absent endpoint/registry; source removes OPENAI_API_KEY.
- Endpoint application is acknowledged by the separate parent process's HTTP handler before kill/closure, with original operation/key. Unknown-reopen records preserve one allowed original attempt/reservation and effect debit1, no receipt, original unknown handle, no new contact and open continuation. The old connection closes; the fresh executor has no surviving transport callback. Live late evidence uses a separate same-worker schedule. Original receipt is validated HTTP200; the second operation has source authorization allowed and final reservation denied scope_closed, no second application, and root outcome stays cancelled.
- Wrong-id and extra-helper are valid ordinary programs that actually run successfully and pass replay while the independent application monitor rejects application_id and extra_helper respectively. Extra-helper has three actual accepted charges. Thus replay agreement is not presented as semantic correctness.
- Root-fence A/B records retain initial entitlement true at I=R3/D2, 32 ordinary acyclic peer Sets, correct candidate success, peer S scope_closed refusal, peer failure, and rejection by frozen optimistic membership. Boundary-extension membership preserves completion counterexamples. This is a failure of the optimistic product/correspondence premise; no universal equivalence or stronger-theorem refutation follows.
- Inspected preserved original compile RED, failed matrix normalization/membership, failed live ACK, failed live result-count assertions and successful later live runs. Historical source capture limitations are explicitly disclosed; original failures were not relabeled as passes.

## Independent execution

Used Go1.27.3 and environment `env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/tmp/alan-phase1f-go-build`.

1. Focused runtime command: `ALAN_I_ARTIFACTS=/tmp/alan-phase1i-review-runtime-records-unrestricted go test -tags integration -run '^TestSubstitution(Runtime|Probes|Recovery|Closure|Effects|RootSuccessFence)$' -count=1 -json -timeout=10m ./internal/substitutiontest` — exit0, **236 actual passed leaves**, package elapsed15.184s. Log `/tmp/alan-phase1i-review-runtime-unrestricted.jsonl`; retained fresh records at the named /tmp directory. The initial restricted attempt failed because nested sandbox-exec and local TCP bind were denied; its log and records remain under `/tmp/alan-phase1i-review-runtime*`. The same authorized test was rerun with approved host access. This environment failure is not a semantic failure or a skipped gate.
2. In `examples/substitution/consumer`, `go test -race -count=1 -json -timeout=10m ./...` — exit0, **two actual passed leaves**, package elapsed3.905s. Log `/tmp/alan-phase1i-review-consumer-race.jsonl`.
3. `python3 /tmp/alan-phase1i-review-records.py` — exit0, audits both original normal/race record sets and explicitly lists closure-decoder defects rather than silently accepting them.

I did not rerun the full Phase1H campaign or the complete root race suite; the supplied focused236 race evidence was inspected directly. No paid calls, source edits, commits or subagents were made by this review. Passing executions establish bounded tested behavior only. The unresolved decoder findings block approval; broad formal equivalence, generalized closure-model refinement and distributed recovery remain outside this review's claims.
