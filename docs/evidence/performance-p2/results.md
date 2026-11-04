# P2: one-call membership reuse reduces measured retained-execution cost

The paired B256 median Execute time fell from **104.629680458s to
57.7476138125s (44.8076%)**, while separate SQL diagnostics fell from
11,770,629 to 3,382,149 executions. Cumulative Go allocation fell 16.6612%,
from 62.633890876GB to 52.198326736GB. Process peak RSS rose from 99.265625
to 102.5390625MiB. This is one bounded workload on one shared host.

## Exact evidence boundary

| Role | Identity |
| --- | --- |
| Production baseline | `76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315` |
| Frozen contract | `34409c56a37a5f5917913ac56a4807479f145375` |
| Production implementation | `2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0` |
| Frozen public timing workload | `9d1209a8069a52fbcf71f2176061eef66d6b9fe8` |
| Probe tooling | `14ea2effa91ad0a1f101dcf59fd400717508dc4f` |
| Corrected HTTP differential tooling | `d11e896c7b6efebd427f577613664d10b919417e` |

The draft PR evidence record binds the final delivered tree, delivery commit,
sealed raw archive and final qualification/review command exits. Those delivery
identities are distinct from the production implementation measured here.
[contract.md](contract.md) governs the experiment;
[invariants.md](invariants.md) records the selected change and prediction;
[baseline-attribution.md](baseline-attribution.md) retains the original 41-process
diagnosis, fifteen-category coverage map and its limitations unchanged.

Raw inputs are retained under `/private/tmp/alan-p2-tf46z4pm` and subsequently
bound by the delivery archive. [results-data.json](results-data.json) contains
seven-bound paired values/ranges, CPU/GC/resources, phase summaries, all SQL
family metrics, work counters, exact command argv and input/source hashes.
Large profiles, traces, stores and raw logs remain external. The ordinary paired
binaries are SHA256 `6ce049ef1778925ec4b7e3d5c4ddbdbf2e559646b3c74c7a7b8ba89983b0e0df`
(baseline) and `e7567ffa07d34d8b73723a8efa033dc5040d48851ce9f7e6a1bd264063340c95`
(candidate); counter/coarse binaries retain their separate schedule identities.

## Direct answers to the ten final questions

1. **Largest SQL source:** the baseline membership family contributes 8,429,051
   executions (71.61%); 8,421,376 belong specifically to `finishRun`'s repeated
   root × intent lookup. The candidate removes exactly 8,388,480 of those
   executions, leaving 32,896 in that scope and 7,675 in unchanged other scopes.
2. **Most CPU:** the baseline B128 profile's largest flat frame is
   `runtime.cgocall`, 9.76s/27.87s sampled CPU (35.02%). Root reconstruction is
   5.34s cumulative, with 4.52s at membership lookup, mostly SQLite descendants.
   These identify substantial repeated native/SQL work; they do not establish
   a complete exclusive application-component ranking. The candidate's largest
   flat frame is `runtime.pthread_cond_signal`, 5.19s/19s (27.32%), followed by
   `runtime.cgocall`, 4.87s. Runtime/native and asynchronous worker samples
   prevent an exclusive attribution of all CPU to semantic categories.
3. **Most allocation:** among the compared application components, B128
   `selectReady` has the largest cumulative sampled allocation: baseline
   5,106.66MB and candidate 5,064.17MB. Candidate typed inspection is 4,674.78MB
   and root reconstruction 569.92MB (baseline 1,773.12MB). Larger overlapping
   candidate totals include parent `Store.Run` 7,364.69MB and cross-cutting
   `decode` 6,452.23MB. They are not disjoint allocation owners and cannot be
   added. These are literal pprof **MB**, not exact MemStats bytes or peak RSS.
4. **Superlinear growth:** for fixed declaration B and retained position k,
   baseline membership queries are Bk per report and Bk(k+1)/2 cumulatively.
   Full-series membership is B²(B+1)/2 under this successful schedule.
   Repeated root traversal, sorting, readiness and growing approval snapshots
   add other work. The candidate changes membership queries to k and
   k(k+1)/2, while preserving the Go root × intent loop. This proves the
   schedule-specific query recurrence, not total-runtime asymptotics.
5. **Necessary marginal work:** a separately approved observation must retain
   its original operation, valid authorization/approval, reservation and debit,
   physical dispatch accounting, original result or uncertainty, settlement,
   selection and durable continuation. Current ownership/generation/epoch,
   authority and remaining resources must still be checked at their owning
   cuts. The protocol requires these facts and decisions; it does not establish
   a minimum SQL statement count, exact write count or asymptotic lower bound.
6. **Selected optimization:** lazily reuse successful membership-root strings
   in a local `map[RunID]string` within one `readRootCompletions` call. Full
   subject+generation keys, successful empty membership and first-use errors
   preserve their meanings. Supported callers provide a stable mutex/owner or
   transaction-snapshot cut. The map ends with the call and carries no mutable
   authority, balance, result or receipt state. No other production optimization
   was made; every subsequent report reconstructs mutable facts afresh.
7. **Prediction:** yes, within this campaign. The exact predicted SQL reduction
   occurred, six B256 paired differences are positive, and median wall/allocation
   reductions exceed the predeclared 30%/10% forecasts. The small-bound guard
   passes. This does not establish tiny B1/B8 improvements or a universal gain.
8. **Exact preservation:** the inspected same-seed differential histories retain
   exact original authority, IDs, epochs, accounting and causal evidence under
   the explicitly narrow normalization below. Named actual crash/recovery and
   historical-reader cases also preserve their bounded contracts. No finite
   qualification establishes equality for every program or fault schedule;
   elided read opportunities can change elapsed-cancellation/I/O fault timing.
9. **Remaining dominant costs:** candidate B256 coarse `selectReady` is
   24.117235288s and fresh approval `InspectExecution` 15.027944045s, versus
   5.688005073s root reconstruction. Readiness, snapshot decoding/validation and
   other repeated root preparation remain. Total fullscan steps and source
   loops are unchanged; fewer queries do not imply less work in every category.
10. **Smallest justified next hypothesis:** separately investigate call-local
    reuse of validated unbound programs keyed by ProgramID in `selectReady`.
    Its mutex-held loop currently calls `loadProgram` for each eligible open
    run before per-run `boundCompositionProgram`; B128 `loadProgram` allocation
    is 2,815.75MB inclusive. First measure duplicate loads and prove immutability,
    lifetime and absence of mutation through binding. Retain per-run binding,
    mutable authority/resource checks and complete fresh approval validation.
    This motivates a narrow study, not an approved change or promised saving.

## Paired ordinary execution and resources

The frozen 98-process schedule has 14 retained warmups and 84 measured rows:
six pairs per bound, alternating source order. No measured row was excluded.
All 98 reports/progress/meta identities validated strictly. Wall, CPU and Go
allocation refer to Execute; RSS is the whole-process high-water, including
later inspection/reopen/Replay. GB means 10⁹ bytes; MiB means 2²⁰ bytes.

| B | Baseline median s [min,max] | Candidate median s [min,max] | Median reduction | Alloc GB base→candidate | CPU s base→candidate | RSS MiB base→candidate |
| ---: | --- | --- | ---: | --- | --- | --- |
| 1 | .004791 [.004567,.005384] | .004689 [.004489,.004868] | 2.13% | .001881→.001887 | .006629→.006513 | 24.312→24.320 |
| 8 | .086616 [.086125,.090713] | .085253 [.084258,.089420] | 1.57% | .057615→.057296 | .117934→.116367 | 27.875→27.633 |
| 16 | .285366 [.281935,.317963] | .274630 [.272622,.292200] | 3.76% | .212298→.209782 | .397999→.380015 | 30.211→30.086 |
| 32 | 1.036385 [1.028977,1.116591] | .970492 [.948739,1.090750] | 6.36% | .812097→.791855 | 1.483682→1.375244 | 35.078→34.805 |
| 64 | 4.233596 [4.214102,4.342950] | 3.526160 [3.508315,3.687958] | 16.71% | 3.246926→3.084141 | 6.220233→5.080044 | 46.000→45.984 |
| 128 | 19.856983 [19.828497,22.498978] | 14.082595 [13.924846,16.657743] | 29.08% | 13.857930→12.550770 | 29.744307→20.384763 | 65.547→66.977 |
| 256 | 104.629680 [104.069834,109.370687] | 57.747614 [57.519847,64.097245] | 44.81% | 62.633891→52.198327 | 157.964058→82.513374 | 99.266→102.539 |

B256 baseline−candidate paired wall differences are
43.988443500, 45.978078083, 46.393522334, 47.195222458,
46.316870000 and 46.995710125 seconds. The primary candidate/baseline
ratio of medians is **0.5519238284941606**; the median of the six paired ratios
is **0.5547172195735479**. They summarize different quantities. The first two
B256 pairs are slower for both sources and remain included; observed shared-host
load spans 2.9067–5.8218. Six positive signs have one-sided
probability 1/64 **conditional on six independent fair null signs**. Shared-host
drift, thermal conditions, caches and uncertain CPU placement limit that
interpretation. Small-bound medians do not cross the frozen max(10%,1ms)
regression guard. The observed B1/B8 changes are descriptive, not established
1–2% improvements. The exploratory baseline 105.268848s is a separate single
sample, not the paired median.

B256 median mallocs fall 784,841,646.5→547,370,977; GC cycles
18,121.5→15,227; accumulated GC pause 502.976824→418.2450975ms.
GC CPU-class accumulation is 12.7864383655→11.0560574555s; asynchronous
runtime accounting must not be added to user/system CPU. No forced GC occurs.

Provisioning, Create, original-state verification, final inspection/view/proof,
reopen/resume and Replay remain separate. B256 median Create is
.246582→.260451s, final inspection .108424→.118108s,
Close/Open/resume 1.427660→.732622s and Replay .859283→.567265s.
Inside Execute, approval snapshot basis is 15.418708→15.515521s and approval
commitment .204025→.199179s. These phase medians are independently calculated;
their sum need not equal a median total. None crosses its predeclared phase
regression flag. The phase/resources vectors and all paired values are retained.

## SQL prediction, top families and unchanged work

| B | Execute SQL baseline→candidate | finishRun membership baseline→candidate |
| ---: | --- | --- |
| 1 | 331→331 | 1→1 |
| 8 | 6,274→6,022 | 288→36 |
| 16 | 20,706→18,666 | 2,176→136 |
| 32 | 79,330→62,962 | 16,896→528 |
| 64 | 358,395→227,355 | 133,120→2,080 |
| 128 | 1,916,389→867,877 | 1,056,768→8,256 |
| 256 | 11,770,629→3,382,149 | 8,421,376→32,896 |

The following is the union of the B256 top tens. Every family other than
membership has exactly unchanged values for all nine SQL metrics at all seven
bounds. Normalized templates and metrics are retained via one JSON registry.
All listed families are reads. Predicate shape and repetition do not establish
per-family latency or prove an index/query plan.

| Family | Baseline Q | Candidate Q |
| --- | ---: | ---: |
| Static/dynamic membership by subject+generation | 8,429,051 | 40,571 |
| Root descriptor by ID | 526,846 | 526,846 |
| Run/program/authority projection join | 496,507 | 496,507 |
| Root publication by scope | 459,264 | 459,264 |
| Subject generation/retirement | 362,110 | 362,110 |
| Settlement presence | 262,656 | 262,656 |
| Composition closure | 230,783 | 230,783 |
| Composition freeze | 230,272 | 230,272 |
| Member/binding descriptor | 181,925 | 181,925 |
| Schema version | 181,158 | 181,158 |
| Program semantics/bytes (candidate top ten) | 180,647 | 180,647 |

Candidate B256 SQL executions/prepares/attempts are each 3,382,149;
step calls 6,303,195, row returns 5,390,841, VM steps 99,581,674.
Fullscan steps remain 11,502,540. Bind/step errors are zero. All partition
checks are complete with no overflow/invalid events, while semantic scopes
still leave 3,069 COMMIT executions unknown. Row returns/fullscan steps are
not exact rows examined; connection setup's internal SQL remains excluded.
All 3,069 begins, 3,069 commits and 255 opens/closes remain, independently of the
changed query calls. No per-family CPU/latency or transaction snapshot IDs are
inferred from these counts.

All source invocation/work counters and semantic counts match exactly at all
seven bounds, including encoded text/blob bytes in these actual rows. B256
still has 8,421,376 root × intent pairs, 65,536 root-loop entries, 360,320
readiness run entries and 29,358,102 observation-loop entries. Typed approval
scans still materialize 1,403,520 rows. Original effects, contacts, charges,
results and selections remain 256, protected approvals 255, reductions 1,023.
Ordinary Execute full-ledger validation counters remain zero. Necessary durable
writes, mutable authority checks, resource reconstruction and full public
approval snapshots were not removed by membership reuse.

## Actual adjacent costs and bounded scaling

Sampling rule was fixed before extraction: take **pair1-candidate-B**, the first
measured pair at each bound, and all its prefixes; take SQL from the distinct
**counters-bB** process. No interval is selected by measured performance.
[candidate-marginals.csv](candidate-marginals.csv) contains all 505 fixed-B
adjacent/cumulative positions, ordinary CPU/allocation/GC and separate SQL.
Both old Bk/Bk(k+1)/2 and new k/k(k+1)/2 membership recurrences reconcile at
every prefix with the unchanged source-pair counter.

| Fixed B=256, k | Adjacent ordinary ms | Adjacent alloc MB (10⁶ bytes) | Separate adjacent SQL | Membership SQL |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 141.165 | 91.240 | 12,543 | 1 |
| 8 | 214.200 | 150.363 | 15,995 | 8 |
| 32 | 227.633 | 165.412 | 15,273 | 32 |
| 128 | 245.372 | 207.818 | 12,870 | 128 |
| 256 | 284.833 | 236.028 | 10,978 | 256 |

This particular ordinary B256 sample totals 64.097244875s; it is not the
57.7476138125s campaign median. T(256,8)=1.603642833s differs from standalone
T(8,8). The final same-process prefix leaves a small outer Execute residual
retained in JSON. These intervals include required approvals and are dependent
descriptions, not 505 independent trials or an exact cross-campaign speedup.

| B | Candidate T/B ms | T/B² ms | T/B³ µs | Q/B² | Q/B³ |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 32 | 30.328 | .947746 | 29.6171 | 61.4863 | 1.92145 |
| 64 | 55.096 | .860879 | 13.4512 | 55.5066 | .867290 |
| 128 | 110.020 | .859533 | 6.71510 | 52.9710 | .413836 |
| 256 | 225.577 | .881159 | 3.44203 | 51.6075 | .201592 |

T/B rises, T/B² is relatively stable over the larger candidate bounds, and
T/B³ falls. Candidate allocated bytes/B² range .753–.796MB at B32–256.
The removed membership-query recurrence has a lower degree for this schedule;
the Go pair loop remains cubic-leading. These descriptive ratios and seven
finite bounds do not prove a quadratic total runtime or unrestricted scaling.
No regression fit or exact per-query cost is claimed.

## Post-change diagnostics and unresolved attribution

| Separate inclusive B256 coarse scope | Baseline s | Candidate s | Candidate calls |
| --- | ---: | ---: | ---: |
| Execute | 104.086892 | 56.394376 | 1 |
| finishRun | 54.650857 | 7.363364 | 256 |
| readRootCompletions, nested in finishRun | 52.959427 | 5.688005 | 256 |
| selectReady | 24.346996 | 24.117235 | 1,535 |
| Approval InspectExecution | 15.095352 | 15.027944 | 255 |
| settleRoots | 7.046835 | 6.974144 | 1,535 |

These inclusive diagnostics do not sum and are not ordinary timing samples.
At B128 the candidate root CPU sample is .720s cumulative, with .030s at its
membership callsite; root allocation is 569.92MB, including 11MB at that
callsite. Entire membership-helper allocation across callers is 18.50MB
(baseline 1,254.11MB); the baseline root-focused callsite was 1,247.61MB.
Profile collection
scope/publication lag, sampling and native allocation limits remain; B128
profile fractions must not be applied to B256 MemStats bytes.

Candidate B32 `b32-cpu-membership-source` exited 2 with “no matches found”:
sample absence is not proof of zero CPU. Four initial B16 trace-top commands
also exited 2 because the generated profile path had moved. All four corrected
path commands exited 0 while reinterpreting the same trace; the failures remain
in command history. Corrected delay totals are scheduler 73.04074ms, sync
1,676.01ms, syscall 126.61ms and network 1,273.32ms. They overlap concurrent
goroutines and include idle server/background waits. They do not measure
additive Execute latency, remote provider latency or establish pool contention.

Host/toolchain remain Apple M4 Mac16,10, 10 CPUs/16GiB, macOS 27.2/26B123,
Go 1.27.3 darwin/arm64, CGO=1/go-sqlite3 v1.14.52, GOMAXPROCS=4,
GOGC=100 and GOMEMLIMIT=off. The endpoint is controlled numeric loopback;
its independent handler journal shares process resources. There is no live
provider. Shared-host conditions and unknown native/runtime ownership remain
limits; wall reduction is measured independently of statement reduction.
The paired controller argv records `python3`, without a contemporaneous
resolved interpreter/version/hash for that campaign. The baseline record and
later `postcampaign-python-identity.json` report Python 3.9.6; the latter is
explicitly not retrospective proof of the paired interpreter. Measured Go
binaries/runtime and recorded tool hashes retain their own bound identities.

## Semantic evidence, reviews and qualifications

`task5-direct-campaign` and `task5-candidate-archive-campaign` each retain four
same-seed execution pairs and 21 same-history crash-reader pairs (101 commands,
all exit 0, per campaign). Normalization removes only source label, invocation
store path and physical SQL row ordering. Original IDs, namespaces, epochs,
authority decisions, grants/debits, request/result bytes, approvals, selection
and uncertainty are not normalized. Database schema/typed cells, public cuts,
history, Replay and contacts compare exactly. The generic archived reader only
captures and Open/Close-renews; it does not Run/resume or contact an endpoint.

Corrected HTTP success/reopen pairs pass at B8 and B16 with exact complete
cuts and independent request/response journals. The earlier malformed matching
outputs accepted by the first tool and genuine B16 ordering rejection remain
preserved. `reviews/task5-http-tool-final-review.md` returns APPROVE for the
corrected 16-file tool snapshot after two preserved REJECT reviews; it approves
that semantic tool scope only. Matching controls retain two positive outcomes
and 51 structured semantic rejections with nonempty reasons.

`reviews/task5-recovery-review.md` returns APPROVE for implementation
`2fe47cb7…` and its bounded authority/accounting/recovery/compatibility gate.
It independently copied/inspected 20 named main/WAL history cuts and verified
original hashes unchanged. Unknown accepted response, retained-but-unsettled
result, settled-before-selection and fourth approval/authorization/reservation/
gate cuts remain distinct. Both baseline and candidate each retain 21 actual
weather process archives. Four additional CLI opposite-build ACK/SIGKILL/reap
and resume cases pass; their temporary DB directories were deleted by their
controller, so only their source/commands/output/executable identities remain.
No retained reservation is treated as proof of physical contact; no external
exactly-once guarantee is claimed.

The independent external module records 11 commands at exit 0, ordinary/race
consumers without skips, historical readers and genuine writer-created 11→12
conflicting-receipt promotion. The original receipt/digest and selected effects
remain; conflict uncertainty is separately visible. The initial candidate
qualification's 12 commands all exit 0, including full ordinary/integration/
integration-race, build/vet, modules/tidy and four root ALAN Staticcheck
configurations. Copied SQLite Staticcheck findings retain the same 47 upstream
matches and nonzero disposition; they are not counted as a root ALAN pass.

Baseline intermittent competing-read assertion failures remain recorded;
a later full rerun pass does not erase them. Their exact current cause is
unknown because the failing DB/full joined error was not retained. Earlier
conditional DDL/locking explanations are not proof for these events. No
production/test weakening was used to eliminate the failure.

## Reproducing the compact extraction

Run the extractor against the retained inputs and an explicit read-only
interpretation checkout, choosing a fresh output directory disjoint from both:

```sh
python3 /absolute/source/docs/evidence/performance-p2/extract_results.py \
  --inputs /absolute/retained-p2-evidence \
  --source /absolute/source \
  --out /absolute/fresh-results
```

It reads JSON/text/log artifacts, verifies all 98 raw paired input identities,
summary hashes and command-output hashes, checks all 505 recurrences and every
nonmembership SQL family, then emits compact JSON and LF CSV. It hashes every
input it reads, records its actual argv and source identities, refuses existing
or overlapping outputs, and opens no database or network connection. Output
JSON embeds the chosen paths/argv, so moving inputs changes its provenance
bytes; numeric results and CSV follow the same deterministic sampling rule.
`--source` supplies files used to interpret the campaign, whose hashes are
recorded; it does not certify or derive the measured revision identities.
The JSON identities block contains fixed archived campaign labels. Extraction
performs no new measurement and does not qualify a different checkout as those
measured revisions.
