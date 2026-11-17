# Delegated Weather Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace direct Weather with one explicitly awaited retained helper and independently check bounded completion, costs and recovery evidence for the same Outing caller.

**Architecture:** Ordinary retained programs share the existing proposal/assessment/decision boundary. Strict application validation follows original invocation, forecast and selection identities. A separate Python finite model and complete raw-Store checker qualify the declared fixture domain without using the production compiler, reducer or verifier.

**Tech Stack:** Existing Go/SQLite runtime and hourly fixture adapter; Python standard library for independent qualification; Staticcheck.

**Spec:** `docs/superpowers/specs/2026-11-17-delegated-weather-design.md`.

## Global Constraints

- Work after separately reviewed scheduler, compiler and rejected-pending inspection ports. Root owns branch transitions, reviews and publication; no automatic merge.
- Use only existing public runtime/author APIs in application code. No new primitive, dependency, format, composition-v2 behavior, runtime scheduling hook or retry policy.
- Delegated mode supports zero through two eligible candidates and one helper per proposal, including the empty proposal. Legacy modes keep their existing limits and artifacts.
- Outing source/artifact/inputs, public schemas, canonical configuration bytes and legacy report behavior stay unchanged. Defaults remain `assess-first` and delegated helper allowance one.
- Preserve every original operation, result, charge and pending obligation; cross-run causality uses identities and explicit references, never numeric sequence comparison between runs.
- Admission allowance zero is a visible refusal, not completion, no-match or circular dependency. Unknown dispatched effects remain unknown and are never retried.
- Independent qualification enumerates its finite domain and separately validates complete recorded concrete cuts. Neither result is universal model/runtime refinement or exact-debit equivalence.
- Ordinary messages have durable source authorization before admission; recorded allow survives later closure and recorded refusal cannot turn into allow. Preserve broad-interleaving source-closure counterexamples. Positive model checks use an explicit pre-close-authorization domain; any automatic-runner correspondence is limited to proven sole-pending-message cuts with open endpoints and no concurrent mutators.
- Read research AGENTS.md and read-only adversarial reference. Do not edit research or read `.envrc`. Keep findings and exact evidence; never weaken checks to pass.
- Implementers do not spawn subagents. Production tasks execute sequentially; each has an independent review gate before dependent work.

## Review Focus

- Empty proposals and allowance zero must still exercise exactly one original invocation with truthful costs and refusal evidence.
- Repeated start must distinguish omitted allowance (one) from explicit zero; resume uses retained options.
- An unchanged final answer cannot hide a wrong invocation, missing second forecast, extra child/effect or altered charge.
- Parent, helper and Outing local sequence numbers are incomparable across runs; original request/result/selection links must drive evidence.
- A lost external response blocks the original await across fresh processes without new contacts or invented observations.

### Task 1: Author and validate the retained helper deployment

**Files:**
- Create: `examples/outing-planner/delegated.go`, `delegated_deployment.go`, `delegated_program_test.go`, `delegated_deployment_test.go`.
- Modify narrowly: `examples/outing-planner/deployment.go`.

**Interfaces:**
- Add constant `delegatedWeather = "delegated"`.
- Add `weatherHelperTemplate() (author.Template, error)` and `delegatedWeatherDefinition(helper author.Template) author.Definition`.
- Add `deploymentOptions { Weather string; HelperAdmission int64 }`, `defaultDeploymentOptions(variant string) deploymentOptions`, and `(c configuration) provisionDeployment(options deploymentOptions) (runtime.Provision, error)`. Default helper allowance is one only for delegated mode; legacy options carry zero. CLI explicit-flag rejection belongs to Task 2.
- Add `retainedDeploymentInfo { Configuration configuration; Options deploymentOptions; HelperTemplateID string; HelperRun *runtime.RunID }` and `retainedDeployment(*runtime.ExecutionSnapshot) (retainedDeploymentInfo, error)`.
- Preserve `c.provision(variant)` and `retainedConfiguration(snapshot)` signatures as wrappers. This task produces the exact deployment classifier consumed by Task 2; it need not expose delegated CLI flags yet.

- [ ] Capture direct/confirm-first/Outing artifact and canonical configuration identities at the reviewed prerequisite base; retain the pre-port Store compatibility evidence. Verify the legacy identities again after constructing delegated programs to detect accidental mutation of shared schema/definition maps.
- [ ] Add failing program and deployment tests: n=0,1,2, accepted/refused invocation, complete ordered response, malformed response/identity/coverage, exact resource roots and no unsupported member/grant/template. Record RED.
- [ ] Implement one helper whose work request/response and sequential hourly behavior follow the spec. Parent must await the original published result, validate it, then use the unchanged public assessment/decision protocol. Guard optional response fields before reads and preserve genuine original forecast handles.
- [ ] Extend exact deployment validation with original catalogue, root limits, initial inputs, accepted dynamic member/binding/grants and zero extra work. Preserve the existing legacy validator's effective checks. Exclude admission rejection from descendant debt without hiding its outcome delivery or caller wait.
- [ ] Exercise through public runtime APIs with deterministic local fixtures: unchanged Outing bytes; exactly n original helper effects; one invocation and one created run on acceptance; zero member/contact/debit on refusal; reject more than two eligible candidates before Store creation.
- [ ] Run focused new tests normal/race, existing outing tests and Staticcheck for the example. Inspect complete diff and commit only this task. Record results, exact artifacts/base and public interfaces in the supplied task report.

### Task 2: Expose delegated planning with complete causal evidence

**Files:**
- Create: `examples/outing-planner/delegated_evidence.go`, `delegated_evidence_test.go`, `delegated_cli_test.go`, `delegated_replacement_test.go`.
- Modify narrowly: `examples/outing-planner/main.go`, `evidence.go`, `report.go`, `replacement.go`; existing test helpers only where needed.

**Interfaces:**
- Consume Task 1 deployment APIs. Existing CLI/start/operate wrappers remain available to legacy tests.
- Add `--weather delegated` and explicit-presence-aware `--helper-admission 0|1` only to start/check-replacement. Omission means one for delegated; reject explicit allowance for legacy modes. Repeating start compares configuration and options; resume derives retained options.
- Add optional `plannerReport.Delegated` JSON field, omitted for legacy modes, with the retained allowance/template/call/helper identity plus complete invocation and root views. Existing top-level report arrays continue to cover all runs, operations, effects, observations, admissions, pending obligations and full histories.
- Retained replacement output directory keeps existing configuration/fixture/Store/replacement files. For delegated comparisons additionally emit `deployment-manifest.json`: version one, exact canonical artifacts/program IDs/role and template identities, canonical configuration hash, complete frozen normalized candidate records (including exclusions and order), expected original root initial projections and options, and source base provenance. This trusted manifest binds programs and the normalized-input boundary for the independent checker; it is not a proof of those programs or calendar normalization. Create it from the frozen deployment inputs before execution, not by copying a later claimed answer.

- [ ] Record RED for delegated CLI, option idempotence, causal report/replacement and hostile snapshot controls. Cover legacy flag/JSON behavior and exact old cycle witness.
- [ ] Implement full invocation/request/member/reply/publication/selection and forecast causality, including complete operation coverage and same-run ordering. Preserve direct verifier behavior. Report acceptance, refusal, failure, unknown, pending and bound exhaustion separately.
- [ ] Require actual root publication and discharged debt for positive delegated completion. A refused helper leaves Outing's wait visible; never synthesize an assessment to force progress.
- [ ] Compare fresh direct/delegated Stores against frozen identical input and fixture bytes, unchanged Outing artifact, explicit candidate-aligned cross-Store identity map and separately reported costs `(0,0,n)` versus `(1,1,n)`. Do not erase IDs or treat unavailable/unknown/refused outcomes as positive replacement.
- [ ] Test n=0,1,2 across suitable/unsuitable/unavailable mixtures and allowance0, unsupported bound/option rejection before work, stale start options, and malformed catalogue/grant/extra-run rejection. Test wrong original forecast/helper result, omitted pending operation and numerically misleading cross-run selection.
- [ ] Address the Task1 nonblocking test-strength finding with well-formed equal-cardinality mutations: change an original forecast's owner together with its view, and separately alter original-effect linkage/sequence without empty IDs or extra entries masking the intended check. Include a focused mutation or exact diagnostic proving the ownership/link checks are exercised.
- [ ] Run focused tests normal/race, all outing tests, Staticcheck and build the example. Commit this task; report exact APIs, export fields and commands for the independent-checker task.

### Task 3: Implement the independent finite protocol model

**Files:**
- Create: `examples/outing-planner/model/check.py`, `model/run.py`, `model/test_model.py`, `model/README.md`.

**Interfaces:**
- `python3 -I -B examples/outing-planner/model/run.py enumerate --output FILE` writes one versioned JSON result. The launcher reads check.py once into bytes, hashes those exact bytes and executes compile(buffer, path, "exec") in its isolated process. It records both source/launcher hashes and Python/SQLite versions; no local imports or generated graph input.
- `check.py` owns independent protocol state/transition rules, reachable-state exploration, reverse completion reachability and shortest counterexamples. Task 4 extends this same script with raw-Store correspondence commands. Keep functions separated by responsibility inside this one source-only module.
- Symbolic event alphabet, obligations and state fields follow the design, including original invocation/forecast identities, distinct commit/source-authorization/admission/acknowledgment/dispatch/result/settlement/selection stages, public messages, child/root completion and explicit costs. Enumerate both broad and pre-close-authorization scheduling domains, labeling their results separately.
- Match the fixed Store's atomic boundaries: invocation publication, accepted result copy when applicable, outcome observation and delivery link are one retained transition; parent selection is later. Rejected publication creates no helper/member/result row. Do not add an impossible runtime crash cut between publication and delivery.

- [ ] Start from the design/protocol contract, not a translation of production control flow. Add independent failing tests that detect a deliberately bad transition and an invalid completion mark.
- [ ] Enumerate n=0,1,2, direct/delegated, helper allowance0/1, all suitable/unsuitable/unavailable vectors and no-result original-dispatch alternatives, plus fixed confirm-first control. Keep exploration finite and report actual counts only after execution.
- [ ] Compute possible successful completion and nonblocking from the entire reachable graph in each declared scheduling domain. Preserve refusal, unknown, cycle and decision-Send/Outing-End/first-authorization-source-closed counterexamples; diagnostic termination is distinct from successful planning completion. Test durable allow after closure, durable deny on retry and the bound helper reply exception with exact original binding. Never silently make every committed message deliverable.
- [ ] Compare semantic answer sets only under adequate admission and known fixture outcomes. Expose different internal costs and refusal behavior; do not label these variants exact-debit bisimilar or claim the research continuation-monitor theorem applies.
- [ ] Validate every state invariant and exercise model mutations that preserve the apparent answer but omit an obligation/cost. Test exact source execution hashing and refusal of unsupported inputs. Run source-only unit tests and exhaustive enumeration once on final bytes.
- [ ] Record source hashes, commands, finite domain/counts/limits and RED/GREEN. Inspect and commit only this task; report the stable model API for Task 4.

### Task 4: Check complete concrete Store correspondence independently

**Files:**
- Extend: `examples/outing-planner/model/check.py`, `model/test_model.py`, `model/README.md`.
- Create: `examples/outing-planner/model/test_evidence.py`, `examples/outing-planner/delegated_model_test.go` and small reproducible test fixture helpers if needed.

**Interfaces:**
- `model/run.py check --store FILE --configuration FILE --fixture FILE --manifest FILE [--report FILE] --output FILE` independently checks a complete consistent cut.
- `model/run.py compare --baseline FILE --candidate FILE --configuration FILE --fixture FILE --manifest FILE [--replacement FILE] --output FILE` reads/checks both raw Stores, then compares the bounded semantic answers, original-obligation mapping and explicit costs. No production verifier or prefiltered event export is an authority.
- Outputs include source identities, full row inventory/coverage, reconstructed events/debts/charges, legal model path or specific witness, concrete completion classification and the claim's bounded domain. The Go harness only creates/captures fixtures and invokes the checker; it cannot supply its verdict.

- [ ] Add RED controls for report-only omissions and copied-Store semantic mutations before implementing their correspondence rules.
- [ ] Read SQLite in one read-only transaction; validate the exact supported schema manifest and enumerate every table/row with primary-key and byte digests. Reject nonempty unsupported authority/reconciliation/lifecycle/receipt/asset categories. Explicitly classify all remaining structural and semantic records; never silently ignore a nonempty table.
- [ ] Reconstruct full histories, intents, original invocation and adapter stages, message authorization/admission/acknowledgment/selection, complete pending debt and all ordered charges directly from raw records. Verify bidirectional coverage and every causal link against pinned artifacts/configuration/fixtures; never use the Go compiler/reducer/evidence-verifier/native-response verifier. Validate the authorization premises separately before attributing a concrete cut to the constrained model. Helper freeze must retain committed original reply intent and settlement must await its admission.
- [ ] Independently derive fixture request/response and suitability, then require a legal model path consuming every observable event and matching final state/debts. Hidden pure steps cannot hide operations or selections. Compare optional report/export coverage to raw inventory.
- [ ] Treat source-bound composition facts as ordered and ordinary admission/acknowledgment/settlement placement as partially ordered. Check authorization Source.Sequence against its source projection and global fact prefix, not the fact's original-intent Sequence. Report compatible model placement rather than a fabricated observed total chronology. Pin schema9/all42 tables and the complete pre-created program manifest; require captured decision-commit evidence for the sole-pending premise, and independent endpoint evidence for actual contact.
- [ ] Test all same-answer negative controls in the design: hidden second forecast/charge/child, extra inert program row, wrong owner/request/original response, changed same-shaped candidate/handle, premature public assessment, duplicate reply/observation, removed result, cross-run sequence misuse and altered pending row. Require specific invariant failures rather than parser failures. Mutate only copied Stores and preserve originals. The raw program inventory check is separate from the public snapshot's referenced-artifact validation because that API does not expose inert program rows.
- [ ] Check complete/pending/refused/unknown direct/delegated fixture cuts and unchanged confirm-first cycle independently. Add legitimate public-runtime controls for late first decision authorization (`source_closed`) and allowed-before-End/admitted-after-End; the former is valid incomplete evidence, not corruption. The raw checker must preserve that distinction. Run source-only tests plus focused Go harness tests; record concrete coverage separately from finite enumeration and document trusted capture/manifest boundaries.
- [ ] Inspect and commit only this task; report complete commands/results, supported table manifest and any unresolved discrepancy. A runtime defect must become a separately scoped fix, never a skipped correspondence check.

### Task 5: Qualify process recovery and document the bounded result

**Files:**
- Create: `examples/outing-planner/delegated_process_test.go`, `delegated_qualification_test.go`, `testdata/delegated-walkthrough.sh`. Reuse the existing two-park `testdata/configuration.json` and `testdata/fixture.json`; do not create duplicate fixture files.
- Extend existing process/barrier fixture helpers narrowly if needed; no production scheduling callbacks.
- Modify: `examples/outing-planner/QUALIFICATION.md`, `examples/outing-planner/README.md`, `docs/outing-planner.md`, and `model/README.md` for final tested commands.
- Create a compact source-bound qualification record under `docs/qualification/delegated-weather/` only after the campaign runs, including a manifest and a controlled archive of the raw commands/results, negative witnesses and necessary complete Store/source snapshots. No unrelated build caches or binaries.

**Interfaces:**
- Consume the public app commands, raw Store/report/manifest outputs and independent checker commands from Tasks 1–4.
- Qualification output is written outside the checkout first, with exact source/tree hashes, commands/exits, frozen configuration/fixtures, complete safe Store snapshots, checker results and retained counterexamples. Document a repeatable controlled archive, not fabricated historical counts.

- [ ] Add tests for each specified retained cut around proposal, invocation admission/request, result persistence/settlement/selection, helper reply/outcome publication, parent assessment and public decision. Use existing barriers or ordinary step/settlement APIs in a test subprocess to prove the actual retained cut.
- [ ] Exercise helper Send/End/prepare-before-reply-admission: the frozen original reply intent remains pending at `reply_admission`, with no published empty success. Authorize/record that bound reply after local End, then publish/select the exact complete result. At automatic decision-commit cuts, verify the decision is the sole pending internal delivery with open sender/receiver before asserting the constrained-model premise.
- [ ] Preserve a genuine separate-process SIGKILL/reopen witness after helper result publication but before parent consumption; verify same original IDs/history/charges and one-time consumption. Check all positive cuts with the independent raw reader.
- [ ] Exercise acknowledged original GET with withheld response and owner kill. Two fresh resumes keep original invocation/effect IDs and unknown/debt without further endpoint contact or later candidate work. Status/inspect/replay remain read-only and contact-free.
- [ ] Compare before/after namespace, logical generations, artifacts, frozen inputs, operation prefix and costs. Only open activation epochs renew. Keep source-bound compatibility with pre-port legacy Stores and exact old cycle witness.
- [ ] Run focused process/qualification normal/race tests and independent model/checker campaigns. Then run `go mod verify`, `go mod tidy -diff`, `go build ./...`, `staticcheck ./...`, `staticcheck -tags integration ./...`, `go test -count=1 -timeout=60m ./...`, `go test -count=1 -timeout=60m -tags integration ./...`, and `go test -count=1 -timeout=60m -race -tags integration ./...` once after final source changes. Root runs the committed older-reader gate and final independent whole-stack review.
- [ ] Document actual outcomes, counts, refusal/unknown/cycle witnesses, extra costs, empirical correspondence limits, single-host/finite-domain assumptions and research-source implications. Include an approachable two-park walkthrough. Inspect status/full diff, exclude unrelated/generated scratch output and commit this task.
