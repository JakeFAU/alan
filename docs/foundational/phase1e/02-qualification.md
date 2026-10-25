# Phase 1E qualification (in progress)

The completed historical baseline, namespace and authoring slices are recorded below. Provider, application and process qualification remain in progress; this document is not yet the final acceptance report.

## Historical baseline reproduction

Source commit `d708493646aeae3c1e70a03cf2bb2d7db974c127` has the same tracked tree as the published Phase 1D.1 source. The immutable archived source reproduced its retained 58-file manifest, aggregate SHA-256 `9fef40370632bdf7416d5bdb2e8472f31ae2adc80ae5145cd7c9c90ccfe63e63`. Full normal and race suites each passed 357 test/subtest events. All ordered finite-semantics (15), composition (20) and supervision (32) qualification payloads matched the historical final evidence. See [baseline report](evidence/baseline/report.md), commands, raw logs and comparison records alongside it. Historical files were not rewritten.

## Namespace correction

The original two-store input produced four endpoint contacts but one mutation: identical local intent identities collided at the shared endpoint. With namespace/key version 2, the same scenario produces four contacts and two mutations. Reopening each original store preserves its key. Rename/copy preserves identity; a copy is not a separately provisioned store. The retained pre-fix source and databases are in [task1 evidence](evidence/task1/manifest.json). Full current-source tests at this boundary passed 368 test/subtest events, separately from historical reproduction; focused races also passed. See [slice report](evidence/task1/report.md).

## Compiler review counterexamples

The first compiler implementation passed its supplied default and integration suites, but independent public-consumer probes found two wrong results. A true branch ending in a reachable forward label fell into the false branch; distinct invalid UTF-8 variable names became the same serialized identifier. Run and Replay both reproduced each wrong result, illustrating why replay consistency alone is not compiler conformance. The [review](evidence/task2/review-round1.md), original inputs/artifacts and complete pre-fix Go/module source are retained in [task2 evidence](evidence/task2/manifest.json). Both defects are corrected and [scoped re-review](evidence/task2/rereview-round1.md) accepts R1/R2. The original public probes and new executable regressions verify the intended branch result and UTF-8 rejection; valid Unicode remains distinct. Full suites before the fix passed 405 default/414 integration test events; the fix then passed 72 focused author/runtime/schema events and targeted metadata/legacy checks. These intermediate counts are not the final Phase 1E qualification.

## Live provider gate

**NOT RUN.** No configured OpenAI credential or explicit model ID was available during initial inspection. No live result or provider usage is claimed. Controlled protocol/application qualification will be reported separately, with a runnable opt-in live command.

## Remaining gates

Narrow adapter and retained corpus protocol; ordinary nested application; acknowledged process-crash recovery and offline replay; independent whole-branch review; final source/dependency manifest and credential-free reproducible archive. No final acceptance claim is made until those gates are recorded.
