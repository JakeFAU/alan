# P3 readiness reuse: frozen comparison

The single call-local program reuse change passes this bounded experiment's
predeclared performance and memory guards. B256 median Execute fell from
60.558126s to 47.193826s (22.07%), all six paired differences are positive, and
median cumulative Go allocation fell 20.41%. No primary row failed or was
replaced. This is performance evidence for the fixed direct-adapter workload,
not general composition, hostile-writer, distributed-owner or universal latency
qualification. Independent Task 5 review approved this evidence. The final-source
gate protocol and decision are in [decision.md](decision.md); its external attestation
records actual completion. This record is not merge approval.

Baseline: `926a899b651fccc48be52a054612a92afcf1ddfd`;
candidate executable source: `4f5fe2cc83316900a2f387180dd9f856db5303a0`.
The sole production change is the runner cache introduced at `a298332f`.
Tooling checkpoint `59bc43a` was independently REJECTED; the exact guard and
approval-chronology fixes at `b2b9cc9` received independent APPROVE before the
first warmup. Both original findings and counterexamples remain retained.

The approved binding is
`/private/tmp/alan-p3-j216kxoz/task5/fix1/checkpoint-binding.json`, SHA256
`66a8c3a4d2d596ba53c3c1e35930eb4c706f747c52cb402e7584efd4fdfd6116`.
A binary SHA256: `729677568eb86c6d90483a8f2c513f147d34787abe0bdc7d21f1aa82478efe99`;
B binary SHA256: `f0df8f5989efb5478515406962bbcf7096c9afb87624a35ec1b515f79f45ed77`.
The unchanged P2 driver, explicit `-mode=direct -prefix=true`, sources, module
replacements, tools, whitelist and all 98 commands are bound there. Go 1.27.3,
CGO 1, SQLite 1.14.52, GOMAXPROCS 4, GOGC 100 and GOMEMLIMIT off are unchanged.

## Primary results

Exactly 14 retained excluded warmups and 84 measured rows ran serially in the
frozen order: six adjacent AB/BA pairs at each bound. All 98 passed the strict
proof and custody checks. Execute includes N Run calls and N−1 fresh public
snapshot/basis/approval cycles, progress and prefix proof. Other phases stayed
separate. The inherited raw outer resource label still mentions HTTP; actual
validated transport is `measurement/v1`, `deterministic-local-no-http`,
`process-no-server`, with no server journal. These are direct adapter entries.

| Bound | Baseline median Execute s | Candidate median s | Reduction | Baseline range s | Candidate range s |
|---:|---:|---:|---:|---:|---:|
|1|0.003973|0.004091|−2.97%|0.003768–0.004232|0.003899–0.004398|
|8|0.086038|0.078296|9.00%|0.084642–0.089162|0.077184–0.079005|
|16|0.275608|0.238797|13.36%|0.274581–0.276285|0.236102–0.241302|
|32|0.962550|0.804218|16.45%|0.958634–0.975145|0.801607–0.805790|
|64|3.545137|2.899992|18.20%|3.536887–3.557425|2.882651–2.909961|
|128|14.244499|11.343695|20.36%|14.212649–14.290706|11.319077–11.417155|
|256|60.558126|47.193826|22.07%|60.362435–60.725706|47.129507–47.366623|

B256 baseline-minus-candidate paired Execute seconds, in pair order:
`13.342555500, 13.409476208, 13.546315124, 13.192766792, 13.474601458, 13.122908041`.
Median paired candidate/baseline ratio is 0.778961947. The frozen greater-than 5%
criterion uses 1−median(candidate)/median(baseline), not the median paired ratio;
its result is 0.220685490. All six signs are positive. A 1/64 tail requires six
independent fair null signs; shared-host scheduling, caches, thermals and CPU
placement limit that inference. There was no adaptive sample expansion,
exclusion of slow rows, cross-campaign speedup multiplication or subtraction of
instrumentation overhead.

| Bound | Baseline allocated GB | Candidate allocated GB | Baseline peak RSS MiB | Candidate peak RSS MiB |
|---:|---:|---:|---:|---:|
|1|0.001588|0.001597|23.453|23.578|
|8|0.056976|0.049472|26.336|26.805|
|16|0.210598|0.172661|28.398|28.422|
|32|0.799326|0.641657|32.570|32.438|
|64|3.121955|2.496071|40.805|40.906|
|128|12.727908|10.073282|55.078|55.320|
|256|53.008366|42.191147|82.492|83.141|

Allocation is cumulative Execute traffic (decimal GB); RSS is the entire fresh
process's high-water gauge (MiB), including other phases. B256 median Execute
CPU is 88.397500s→69.319975s, allocation count 558,422,867.5→459,852,681, and GC
cycles 21,495→14,945. Actual values, GC pauses/runtime metrics and process CPU
appear in [results-data.json](results-data.json). Lower allocation traffic does
not imply lower peak memory: B256 median RSS increased 0.648MiB, within the
unchanged max(10%,8MiB) guard. The map retains O(distinct reached reusable
ProgramIDs) program objects until each readiness call ends. There is no
universal peak-memory improvement claim.

B1 is 0.1181045ms slower in median (2.97%), within the frozen max(10%,1ms) guard;
B8 improves 9.00%. B16/B32 pass the same small-bound guard. Every separately
measured non-Execute phase and summed approval phase at every bound passes the
unchanged exact guard. In particular, B256 approval snapshot/basis is
15.599079s→15.611185s, approval establishment 0.206476s→0.200974s, and summed
Run time 44.669245s→31.304999s. All other phase samples and guard dispositions
are retained in the JSON; unchanged work was not moved outside Execute.

## Actual mechanism and remaining work

Fresh separate instrumented processes ran once per source at all seven bounds.
The initial diagnostic protocol fixed 1/8/128/256; an explicitly frozen coverage
supplement added 16/32/64 once each at the controller's request. Neither changed
primary rows, control samples, workload or thresholds. No instrumented timing
is used as primary evidence.

| Bound | Eligible visits / baseline loads | Distinct IDs reached summed per call / candidate loads | Eliminated loads | Baseline Execute SQL | Candidate Execute SQL |
|---:|---:|---:|---:|---:|---:|
|1|4|4|0|332|332|
|8|172|49|123|6,443|6,197|
|16|720|97|623|19,951|18,705|
|32|2,776|193|2,583|67,127|61,961|
|64|10,651|385|10,266|241,811|221,279|
|128|44,297|769|43,528|923,883|836,827|
|256|178,601|1,537|177,064|3,600,555|3,246,427|

At every bound the successful load, decode and unbound-validation counts match
those columns. Exactly two SQL executions disappear per eliminated load.
Actual visited ProgramID counts, per-call visit/distinct distributions and
selected-return distributions agree between sources. All recorded `work.*`
counters remain equal. At B256 there are still 178,601 per-run bindings,
1,023 readiness reductions and 1,023 committed reductions, 511 observation
attempts/selections, 256 effects/direct entries/charges/results/selections and
255 later public approvals. The two actual retained ProgramIDs hash correctly,
agree between sources and have no HandoffPeer port; closed-store copies preserve
main/WAL/SHM custody for that check.

Remaining B256 work includes 29,358,102 observation considerations, 360,320 runs
considered and 8,421,376 root-intent checks across 65,536 roots. These counts are
unchanged. Reuse removes duplicate unbound loads inside one call; it does not
cache per-run binding, projections, generations, scope, authority or observations.
The existing ownership/alias/error boundaries remain those in
[invariants.md](invariants.md), under supported Store mutex and cooperative
single-owner use, excluding hostile unsynchronized SQL writers.

## Shared and distinct program controls

The source-frozen 32-run fixtures use genuine retained programs and the independent
baseline selector. Before execution, the control protocol fixed three cases,
six fresh-process AB/BA pairs per case, one excluded selector call then exactly
20 measured calls per process. No Commit occurs between calls; each selection
must match the excluded call. Counter instrumentation is a separate binary and
never enters those timings. All 36 timed processes and the separate probe pass.

| Case | Actual visits / distinct per call | Baseline→candidate loads | Median ns per call, baseline→candidate | Allocated bytes per call, baseline→candidate |
|---|---:|---:|---:|---:|
|Shared waiting|32 /4|32→4|1,422,962.5→678,903.1|819,943.8→295,023.6|
|Distinct waiting|32 /32|32→32|1,422,090.6→1,425,321.9|819,602.2→829,687.0|
|Shared, last ready|32 /4|32→4|1,449,460.4→715,334.4|835,882.0→310,721.8|

Each shared ID is actually reached eight times; each distinct ID once. Both
implementations still bind 32 runs; waiting cases reduce zero observations and
the late-ready case reduces one. The distinct control's point estimate is
+0.227% time and +1.23% allocation; it demonstrates bounded no-sharing overhead,
not universal overhead bounds or a second acceptance campaign. Full sample
variation and paired ratios are retained. Shared controls save about 51–52% per
selector call, while whole-workload savings remain 22.07% because other work
continues.

## Fresh candidate profile and limits

A separate B128 candidate process produced CPU and allocation profiles with the
same ordinary binary and default 524,288-byte allocation sampling. It is not a
primary row. CPU profile duration 12.56s contains 16.16s sampled CPU across
threads. Inclusive cumulative allocations include InspectExecution 4758.84MB,
selectReady 2630.26MB, readObservations 1197.41MB, finishRun 778.99MB,
readRootCompletions 575.95MB and loadProgram 345.90MB. Inclusive scopes overlap
and must not be added. Full CPU/allocation top tables and selectReady source
annotations are retained. Runtime/cgo/scheduler work occupies substantial CPU;
sampled CPU is not exclusive wall-time attribution.

The largest separately measured remaining B256 phase within Execute is the
summed Run work 31.305s; fresh approval snapshot/basis remains 15.611s. The profile
and exact counters keep observation scanning, root traversal and snapshot
construction visible as the next investigation targets. No second optimization
was implemented. The program cache still ends on every selectReady return;
per-call distinctness does not establish a general complexity theorem.

## Evidence and disposition

[results-data.json](results-data.json) contains all paired metrics, every phase,
all 98 row identities/exits/hashes, mechanism distributions, control samples,
profile hashes and raw index pointers. The raw index is
`/private/tmp/alan-p3-j216kxoz/task5b/raw-index.json`, SHA256
`d09049881f10edf8d631e7c9d9658ac37b75f1e63212a87c10e110b24a52d420`.
It covers the full primary campaign and Task 5b raw files. Original Task 5a failures,
REJECT, correction and binding remain untouched. Task 5b has no failed workload,
build or diagnostic process and no skipped rows. The only intentionally failing
child remains the earlier tooling exit 23 negative test.

The independently reviewed experiment supports retaining this single optimization
subject to the final qualification and whole-change review recorded with delivery.
It establishes neither unrestricted correctness nor merge/release readiness. No production withdrawal is triggered
by these performance results; this timing report does not claim later gates ran.
See [reproduce.md](reproduce.md) for exact routes and source identities.
