# Phase 1D.1 independent whole-branch review

**Observed decisive sequence:** endpoint holding the exact authorized request -> independent peer committed -> ordinary caller consumed the peer result -> controller released the endpoint response -> original client result settled and was consumed.

**Specification verdict: CHANGES REQUIRED for one reporting defect. Quality verdict: CHANGES REQUIRED for the same P2; no additional findings.** The asynchronous progress repair, its durable recovery behavior, and the source/evidence package are otherwise supported by the inspected implementation and evidence. This review assesses completion for the authorized draft PR, not merge, release, or production readiness. Correct and re-review the finding before declaring this phase fully qualified. Opening a draft to discuss unfinished work is distinct from declaring the implementation complete.

## Reviewed identity and scope

- Frozen Go/module source: `ee144b08e6abac10c704c990e83a80314326a1eac3956bc5ecf9489f8f0afd5a`, 57 files.
- Base: `6effdc02259af5335633bfc41e9786386430fa7c`.
- Source archive: SHA256 `4ea1db7953fdc754941ea45b35fa2c5d806a36309cadd1a149af7185af5c38fd`, 94 members.
- Checkout: `/Users/jacob/.codex/worktrees/alan-phase1d1-dispatch/alan`.
- Read the whole-branch review package, requirements, plan, dispatch contract, qualification/review/runbook, root README, changed runtime and test paths, qualification tools, source/archive manifests, historical comparison report, scoped review and retained probes, final evidence and offline-reproduction result.
- This was a fresh independent review. I made no checkout edits, commits, or subagents. All new probes and audit scripts reside in `/tmp/phase1d1-final-review/`.

## Ranked finding

### P2 — Do not classify a durably known applied result as UnknownExternal

**Location:** `internal/executor/runner.go:422–425` (`finishRun`). Relevant boundary: `internal/executor/external.go:210–218` (`accountExternal` persists the receipt before separately settling it).

When a valid receipt commits successfully but the subsequent observation/settlement transaction fails, the final report still appends the operation to `UnknownExternal`. The report checks only that settlement is absent and the attempt was allowed; it never asks whether a valid receipt already proves the external outcome. This collapses the required distinction between an unknown physical result and a known result awaiting local settlement.

A deterministic overlay probe inserts a temporary SQLite trigger rejecting insertion into `settlements`, then runs a non-idempotent operation. The endpoint is contacted exactly once; `external_receipts` contains one valid receipt, `settlements` contains zero rows, the client remains open, and Run returns the explicit settlement error. Nevertheless its report contains `UnknownExternal=[["client",1,1,0]]`. The focused assertion that this field be empty fails on the frozen source. The independent recovery portion passes five race-enabled runs: reopening settles and consumes the existing receipt without another request or mutation.

The underlying classification predates this patch, but it is within this phase's explicit accurate uncertainty reporting and separate-boundary requirements. The revised cancellation/cleanup path and receipt-accounting architecture must preserve that distinction too. This is a reporting defect; the probe does **not** demonstrate loss of evidence, unauthorized dispatch, duplicate application, or failed recovery.

**Correction:** Preserve the intent in `UnresolvedIntents`/`PendingDeliveries` until settlement, preserve the error and `Quiescent=false`, but populate `UnknownExternal` only when an authorized external outcome genuinely lacks retained receipt evidence. Use the existing durable outcome validation so a malformed retained receipt surfaces as corruption instead of silently being called known or unknown. Add a regression covering the receipt-committed/settlement-failed boundary and non-idempotent reopen without redispatch. Refresh qualification identity/evidence after changing runtime source.

**Exact retained input/output:**

- `probe_test.go`, `overlay.json`, `probe-authorized-output.log`: five passing race runs establish the boundary and safe recovery while logging the incorrect field.
- `reporting_red_test.go`, `red-overlay.json`, `reporting-red-output.log`: unchanged explicit expected-report assertion fails, exit 1, with `durable valid receipt is known applied; UnknownExternal=...`.
- `probe-output.log`: initial environmental failure (`bind: operation not permitted`) is retained separately and is not behavioral RED.

No other concrete revision request was found. No finding is parked or deferred.

## Strengths and examined failure paths

1. **Transport ownership and boundedness.** The worker receives `externalWork` and has no Store reference. HTTP request/body reading occurs outside the database mutex and transaction. `active` counts launched-but-unaccounted work, not merely network-active goroutines. Each launch reserves capacity and one dispatch unit before spawning; one completion per launch fits in the equally bounded buffer, so cancellation followed by join cannot deadlock on completion sends. Extra intent work remains durable. The invocation-local attempted map prevents duplicate automatic claims.
2. **Scheduling and waiting.** Each pass offers a completion, an internal delivery/lifecycle action, an eligible external launch, and a round-robin ordinary commit. Saturation does not stop peer computation. With owned work but no ready action, Run waits on completion, cancellation, or a coalesced authoritative-write wake hint. It does not declare quiescence while its transports remain owned. MaxSteps and MaxDispatch are charged by the single owner; a post-commit observer error still charges the durable commit.
3. **Shutdown/store lifecycle.** Close sets closing/cancels under `ownerMu`, releases that mutex, then joins Run before closing storage or relinquishing exclusivity. No reverse nested acquisition counters the write path's `mu` then short `ownerMu` wake section. Received shutdown candidates are accounted under a distinct cleanup context; an already-dequeued candidate that cannot persist leaves explicit error and durable unresolved state. The context deadline is not falsely presented as a hard wall-clock shutdown guarantee.
4. **Authorization and old evidence.** Transport starts only after retained intent authorization and attempt preparation. Receipt persistence reconstructs the request and checks retained attempt, intent, authorization and receipt binding. The original authorization epoch remains authoritative for late evidence even after current activation changes. Settlement does not reopen a closed source run or create a new generation. Ordinary lifecycle programs and existing grants continue to determine policy.
5. **Causal oracle.** The actual CLI is launched with a real separately stored endpoint. A staged peer transaction is released only after the exact endpoint hold acknowledgment. Real peer and caller commit acknowledgments precede the endpoint live-hold probe and release. The controller does not call Commit. Successful original receipt consumption after release additionally prevents a timeout-first result from masquerading as the successful decisive execution. The trace's release entry is correctly documented as a controller command using the verified request identity, not another endpoint acknowledgment.
6. **Concurrent process crash recovery.** Both endpoint profiles cover SIGKILL after peer/caller progress while held and after response reception before durable recording. Inspected assertions and retained records require unresolved authorization and durable peer history, preserve the separate endpoint, enforce original-key idempotent recovery with one application/two contacts, and preserve non-idempotent uncertainty with one contact. Replay is checked without endpoint contact. Earlier accepted/rejected Retry, repeated-recovery crash, terminal delivery and replay tests are preserved.
7. **Documentation and evidence discipline.** README commands match the CLI and explain process success versus logical success, Unknown, replay versus live recovery, fresh database pairs, and the cross-store key limitation. Qualification distinguishes current-source checks from historical archive reproduction, marks reconstructed failure inputs, and retains original failures rather than counting fixture errors as passing evidence. No dependency/schema/language/model expansion is hidden in the repair.

## Independent checks performed

- `audit.py` independently recomputed the length-framed aggregate and every source-file length/hash, plus qualification-tool and scoped-probe hashes. All matched before and after probes.
- Verified the archive hash and exact 94-member inventory, every member length/hash, every Go/module member against the qualified source manifest, and archived files still present in the checkout against current bytes.
- Verified raw normal/race compressed and decompressed log hashes; independently counted 353 passing test/subtest events and no failures in each. Checked all qualification index pointers against exact saved record bytes and identifying fields: 431 index rows, 214 full records per mode.
- Independently examined all 12 saved CLI causal traces (six normal, six race): exact event order, request identity, active authorization at both ordinary commits, live-hold phase, and child-race marker.
- Checked exact ordered QUALIFICATION/COMPOSITION/SUPERVISION payload equality against retained baseline records; confirmed existing phase directories have no tracked differences against base.
- Read the offline archive reproduction record: initially empty module cache, bundled file proxy, all 94 members verified, full normal tests/vet/module verification passed. The parent ran that reproduction; I independently verified its artifact identity, not a second full build.
- Read the saved complete normal/race, static-gate, README smoke and scoped probe results. I did not rerun full suites on unchanged code. The new targeted recovery probe ran with `-race -count=5` and passed; the reporting assertion ran with `-race -count=1` and failed as expected.

Commands:

```sh
python3 /tmp/phase1d1-final-review/audit.py
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off \
  go test -race -overlay=/tmp/phase1d1-final-review/overlay.json \
  -count=5 -timeout=60s -run '^TestFinalReview' -v ./internal/executor
GOTOOLCHAIN=local GOCACHE=/tmp/alan-phase1d1-go-build GOPROXY=off GOSUMDB=off \
  go test -race -overlay=/tmp/phase1d1-final-review/red-overlay.json \
  -count=1 -timeout=30s -run '^TestFinalReviewKnownReceipt' -v ./internal/executor
```

## Explicit limits and declined judgments

- **Unbounded fairness, eventual recovery, arbitrary-depth equivalence:** finite fixtures and source reasoning cannot prove these; the package does not claim them.
- **Power loss, disk failure, distributed partitions/fencing, hostile storage mutation:** only the stated single-host SQLite/process-kill profile is qualified. No new claims were inferred from SIGKILL results.
- **Hard real-time shutdown:** mandatory worker join, driver busy timeout/commit behavior, OS scheduling, and context-respecting observers prevent treating the cleanup context as a wall-clock theorem. The limitation is explicit.
- **Concurrent direct DispatchExternal plus Run:** automatic Run owns the documented bound; simultaneous independently invoked synchronous dispatch is outside it and explicitly excluded.
- **Provider protocols, cryptographic remote authenticity, generic exactly-once business effects, cross-store key repair:** these are outside this local fixture protocol and repair scope. Fresh database filenames still do not establish business namespaces.
- **Exhaustive scheduler/interleaving or leak proof:** focused race tests, source ownership analysis and actual process fixtures provide evidence, not exhaustive search. No full-suite repetition was used as a substitute for a new targeted question.
- **Historical execution claimed as my own:** baseline/scoped-review and offline-reproduction executions were retained evidence inspected and hash-checked, not rerun by this reviewer. I verified historical bytes remain unchanged; I did not re-certify all old phase claims beyond relevant regressions.
- **Other OS/architecture/toolchains:** the manifest and saved qualification identify macOS/arm64, Go 1.27.3, CGO/clang. No cross-platform assurance is implied.
- **Final publication state:** commit/push/draft PR and inclusion of this report remain parent work. The plan's unfinished publication checklist and pending whole-review placeholder reflect that stage, rather than a claim of already completed publication. Any corrected source needs a new freeze and targeted re-review before final approval.
