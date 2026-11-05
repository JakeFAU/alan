# Task 1 independent review

## Spec compliance

**APPROVE — Task 1 is spec compliant at its preliminary attribution/invariant gate.**

Reviewed exact package `review-926a899..5e2045d.diff`, baseline `926a899b651fccc48be52a054612a92afcf1ddfd`, head `5e2045d1556083548e2926a965c43695322388c0`. The package contains only the eleven declared `docs/evidence/performance-p3` additions. Its prose and tools were read in bounded passes; the large JSON additions were parsed in full and their sample phase/resource/counter values independently reconciled against raw data. No changed file was separately reread, no Git command or test suite was rerun, and no production edit was made.

**Quality: APPROVE. Attribution: PROCEED to Task 2.** This approves the preliminary supported-owner hypothesis and evidence sufficiency for the invariant/RED work. It does not approve an optimized production path, unconditional shallow reuse, direct paired tooling, or a performance result. Task 2's independent invariant/alias/RED gate and Task 5's independent strict direct-tooling gate remain mandatory (`plan.md:52`, `plan.md:90`; `contract.md:40`, `contract.md:102`).

The baseline source identity is independently supported: every one of the 607 entries in `/private/tmp/alan-p3-j216kxoz/task1/baseline-manifest.json` matches both the current workspace file and the frozen `task1/baseline/` export. All 88 artifact/sample entries in the diff's `raw-index.json` match actual sizes and SHA-256 hashes. This corroborates the no-production-change claim, beyond the diff's file list.

⚠️ Historical-reader execution, the full final qualification, a complete alias matrix, and candidate paired results are subsequent obligations, correctly not claimed here (`baseline-attribution.md:125`, `contract.md:158`). PR states were checked against the retained initial records, not newly queried; current publication state must still be checked by the controller at delivery. No assertion is made that a review can prove which documents the implementer privately read.

## Strengths and decisive evidence

### Fresh mechanism and counter arithmetic

The diagnostic anchors count a visit immediately after current projection/retirement/generation eligibility and immediately before the actual load. The named-return defer records every exit. `loadProgram` success/failure, decode, and unbound validation are scoped to readiness scope 9; bindings remain separate (`instrument.py:30`, `instrument.py:42`, `instrument.py:58`; actual generated `/private/tmp/alan-p3-j216kxoz/task1/counters-scoped/p3-instrument.patch`; generated `alan/internal/executor/runner.go:323`, `:368`; generated `alan/measurecounter/counter.go:12`). This agrees with the original selector at `internal/executor/runner.go:320–391`, including early returns and eligibility filtering.

I independently streamed the Execute stage of each raw counter report, recomputed histogram call/visit/distinct sums, checked every histogram bin against the committed JSON, and compared every exported counter against raw values:

| Bound | Calls | Visits / successful loads / decodes / validations / bindings | Distinct sum | Repeats | Observation attempts/selections | Raw Execute stage |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | 5 | 4 | 4 | 0 | 1 | `task1/pilot-results/counters-b1.json:21842–25999` |
| 8 | 47 | 172 | 49 | 123 | 15 | `task1/pilot-results/counters-b8.json:185504–190123` |
| 128 | 767 | 44,297 | 769 | 43,528 | 255 | `task1/pilot-results/counters-b128.json:3050169–3055028` |
| 256 | 1,535 | 178,601 | 1,537 | 177,064 | 511 | `task1/pilot-results/counters-b256.json:6271351–6276466` |

All table raw paths are rooted at `/private/tmp/alan-p3-j216kxoz/`. The cited ranges begin at stage `name`; GC/runtime fields occur immediately before that key and were included in the subsequent full-stage check. No load/error counter contradicts the successful samples. B256 SQL executions are 3,600,555. Source `internal/executor/store.go:564`, `:583` confirms two omitted SQL reads per reusable successful load, giving the conditional prediction `3,600,555 - 2*(178,601-1,537) = 3,246,427`. Neither binding nor Reduce is included in that subtraction.

I independently opened the B256 diagnostic database read-only with `mode=ro&immutable=1`, checked each retained byte hash against ProgramID, and confirmed 2,493 bytes/no ports for `654c2f...d4b1c` and 4,083 bytes/a single `local` peer for `d743c2...33e07`. Neither meets the handoff exclusion. Database SHA-256 remained `82ed61e8bc0e57a0eb98abf9ddcbd0e7d7610ef5af48c47c4e88642551233c37` before/after; this independently agrees with `/private/tmp/alan-p3-j216kxoz/task1/program-summary.json`. Actual counter visits are four for the first ID and 178,597 for the approval ID at B256. Thus the measured opportunity survives the proposed exclusion; it is not inferred from provisioned run count.

### Attribution and stopwatch

`baseline-attribution.md:3`, `:92–117` appropriately separates a material allocation mechanism from a latency claim. Raw `/private/tmp/alan-p3-j216kxoz/task1/gates/alloc-selectReady/stdout.log` attributes 2.50GB to source line 362 (the readiness load callsite). `gates/alloc-loadProgram/stdout.log` reports 2.78GB inclusive overall, 1.63GB decode and 715.30MB validation. `gates/cpu-selectReady/stdout.log` reports only 440ms at that load callsite out of 24.12 sampled CPU seconds; `gates/cpu-loadProgram/stdout.log` reports 510ms inclusive across callers. This is sufficient allocation/validation attribution to test reuse, not a defensible numerical elapsed-time prediction.

Every committed sample's full singleton phase/resource fields and exported counters were independently checked against raw stage objects, including coarse scopes; all ten matched. Raw stage multiplicities independently show N Run stages and N−1 snapshot/basis and approval-establishment stages. The actual public driver places those operations inside Execute (`docs/evidence/performance-p2/driver/main.go:323–411`), and retains final inspection/reopen/Replay afterward. All ten sample metadata files record exit zero. Four ordinary samples remain single observations; instrumented/profile/coarse rows are correctly separated and scopes are explicitly nonadditive (`contract.md:76–92`, `baseline-attribution.md:27–43`, `:90`).

### Preliminary invariant challenge

1. **Validation context and immutable-artifact risk:** Inspected `internal/executor/store.go:519–590`, `request_authority.go:60`, and the exact source writes to `programs`/`s.limits`. The production insertion validates and compares existing bytes/semantics; the only production programs write is INSERT at `store.go:558`. Limits are assigned at create/open (`:327`, `:630`). ProgramID equality alone does not establish format/limit equivalence. The contract correctly confines reuse to one fixed Store/queryer/limits/format call (`contract.md:12–28`). It does not invent a read transaction or exclude arbitrary external SQL writers.
2. **Ownership/close risk:** `runner.go:321–322` holds the mutex throughout traversal. `store.go:161–181` provides cooperative flock ownership; `store.go:693–727` cancels/joins Run and owner users before taking the mutex and closing connection/flock. This supports the stated legitimate-owner scope, not hostile-store concurrency. Returning from the selector ends any proposed map before later commit.
3. **Handoff mutation risk:** `invocation.go:98–121` routes a missing composition member to `bindHandoffCaller`; `invocation_handoff.go:154–162` invokes mutation only after finding an author port with `Peer == HandoffPeer`. `:189–202` writes nested Effect/Match and Author.Ports. Excluding every such unbound program before retaining it is conservative and matches this branch. Retaining a shallow copy and checking the flag only after binding would be unsafe because binding changes the port peer. Task 2/3 must test the actual eligibility placement, repeated visits, and errors, not merely detect a marker in a synthetic object.
4. **Member-copy risk:** `composition_binding.go:290–368` validates/encodes then decodes into `out` before endpoint/port writes. The proposed cache must retain the original successful unbound program and must still call binding on each run. Sharing a bound result or bypassing the revalidation here would invalidate this review.
5. **Reducer/data/schema mutation risk:** Inspected `program.go:299–498`, `:211–239`, `collections.go:158–223`, `author_metadata.go:47–135`, `schema/schema.go:65–114`, `schema/collections.go:85–110`, `:306–335`. The paths read program fields, clone projection/cursor/value outputs, copy AuthorityUse, and clone schema enum slices/nested records before sorting. No additional program mutation was found in those paths. This supports the preliminary exclusion proposal, but is not a substitute for Task 2's nested alias probes and full branch matrix (`contract.md:40–46`).

### Frozen design and tool quality

The full schedule has 98 ordered rows, with A/B excluded warmups followed by AB, BA, AB, BA, AB, BA at each of 1,8,16,32,64,128,256. It enables direct mode/prefixes. The >5% B256 and six-positive-pair criterion, conditional sign inference, max(10%,1ms) small/non-Execute/approval guards, RSS review guard, shared-few/distinct controls, and no adaptive replacement are all frozen before any candidate (`contract.md:94–155`). Actual P2 strict validation is HTTP-specific (`performance-p2/campaign_validation.py:20`, `:113`, `:143`); Task 5 explicitly requires a separate frozen and independently reviewed direct adaptation before any confirmatory row. This deferral is correctly scoped, not an implicit validation pass.

`instrument.py` delegates the existing overlay and adds a narrow disposable transformation with fail-closed unique anchors; `summarize.py:10–43` reconciles distribution arithmetic and rejects failed positive workloads. The tests exercise mismatched distribution counters, empty counters, failed child self-deletion with exit preservation, synthetic credential exclusion, and schedule balance (`test_tooling.py:18–68`). Retained test outputs are clean: 22 P2 campaign tests, three P2 anchor tests, four P3 tests, SQL ordinary/race proofs with intended mutant rejection (`task1/gates/{campaign-tool-tests,p2-anchor-tests,p3-tool-tests-final,sql-overlay-proofs}/{result.json,stdout.log,stderr.log}`). No test suite was rerun during this review.

The retained ordinary baseline log independently contains 1,295 pass and five skip events, no fail events. The two actual test skips are the frozen old readers for formats 13/14; the other three are no-test packages. Exact source: `/private/tmp/alan-p3-j216kxoz/task1/gates/ordinary-baseline/stdout.log`; result exit zero and 62.421672833 seconds in adjacent `result.json`. Corrected authorized loopback/subprocess proof says `ACK loopback and subprocess` in `gates/permission-probe-corrected-authorized/stdout.log`. Historical failures remain separate.

## Issues and retained concerns

**Critical: none. Important: none for this Task 1 gate.**

**Minor / measurement caution:** The SQL-overlay diagnostic is expensive, not representative of ordinary memory use. B256 raw diagnostic JSON is 958,824,533 bytes and process peak RSS is 4,364,206,080 bytes (`raw-index.json`, `task1/pilot-results/counters-b256.meta.json`); ordinary peak RSS is 85,622,784 bytes. The current documents already warn against using diagnostic timings and correctly report ordinary RSS (`source-map.md:70–74`, `baseline-attribution.md:27`). Preserve that separation through final analysis and do not describe the entire overlay as low-overhead. This does not invalidate count attribution or require rerunning Task 1.

**Mandatory subsequent gates:** The exclusion is approved only as a preliminary proposal; exact pre-binding eligibility, unbound retention, error ordering, later-call/context changes, and alias probes must be independently approved in Task 2. Strict direct-mode adaptation must be reviewed before Task 5 timing. Unknown speedup, potential distinct-program overhead, unsupported arbitrary writers, and old-reader skips remain unresolved rather than averaged away.

## Review operation outcomes

Read-only evidence scripts completed the hash, histogram, stage, source-custody, and retained-program checks above. No runtime/semantic/timing workload was run. The only file written is this review.

Retained reviewer apparatus failures: an initial hash-check script exited 1 because the system Python lacks `hashlib.file_digest`; its replacement used chunked SHA-256 and verified all 88 entries. A partial-stage extraction then exited 1 because it started at the `name` property, omitting preceding GC/runtime properties. Inspection showed those properties immediately above `name`; the final independent streaming check parsed each whole stage object and all ten complete phase/resource/counter comparisons passed. These were reviewer extraction errors, not failed workload rows or defects in submitted evidence. One intermediate read-only extraction was yielded without its session identifier being surfaced; its output is not relied upon. The final bounded streaming verification exited zero in 2.9216395 seconds.

## Assessment

**Spec APPROVE. Quality APPROVE. PROCEED to Task 2 only.** The actual repeated-load mechanism is independently supported, source premises are accurately bounded, and the proposed handoff exclusion preserves the measured opportunity. Production safety and usefulness still depend on the explicitly retained subsequent gates.
