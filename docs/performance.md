# Execution and performance evidence

These measurements apply to the workloads, source revisions and hosts in the
linked reports. Each report records the timing window and checks that accompanied
the measurement.

| Work | Result and evidence |
| --- | --- |
| Finite observation series | The [series qualification](bounded-series-qualification.md) records an eight-observation run, process recovery, independent review and measured scaling. The [authoring guide](bounded-series.md) describes the finite declaration. |
| Profile decoding | The controlled 256-operation median fell from 347.99 s to 105.52 s with the same authorization and recovery checks. [Execution scaling, raw provenance and memory results](series-scaling.md) |
| Membership lookups | Report-local reuse reduced the 256-operation median from 104.63 s to 57.75 s and SQL statements from 11.77 million to 3.38 million in a paired single-host campaign. Cumulative allocation fell and peak memory rose slightly. [Comparison and recovery qualification](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/performance-p2/results.md) |
| Readiness | Reuse of validated unbound programs within one readiness call reduced the 256-operation median from 60.56 s to 47.19 s against a separate integrated baseline. Per-run checks remained. Allocation fell and peak memory rose slightly. [Source qualification and decision](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/performance-p3/decision.md) |
| Bounded execution work | The campaign measured guarded owner discovery, grouped full reports and a separately tested targeted continuation-basis read. [Recovery qualification and remaining full-report costs](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/execution-scaling/results.md) |
| Full reports | Report-local batching reduced median Execute elapsed time from 7.031 s to 4.613 s on a frozen 256-operation series. [SQL work, allocation, peak RSS, compatibility and recovery](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/report-scaling/results.md) |
| Execution progress | On the same frozen candidate as full Run, the opt-in progress call reduced median execution-window time from 4.585 s to 1.298 s at 256 operations. The lifecycle included a final full report, inspection, reopen and Replay. [Progress scaling results](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/progress-scaling/results.md) |
| Retained history insertion | The P7 pilot missed its latency threshold at 256 operations. Production runtime stayed unchanged. [Retained-query scaling results](https://github.com/JakeFAU/alan/blob/64a692b6f82ae10353f50c8741377166bc6cf854/docs/evidence/retained-query-scaling/results.md) |
