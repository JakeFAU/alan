# Phase 1H baseline

Actual starting commit4550e0f4ba4c57cae8e84d1afe9fbf47e16786d8, tree dc809d50d699baa6ef86e5023046ac97e7126349. PR13 is merged; no dependent unmerged branch is required. This is the current merged checkout including restored historical Phase1D source. `baseline-identity.json` retains hashes for2494 historical documentation/evidence files. Working tree was clean before new Phase1H preparation.

The original Phase1G collector verified all258 frozen contributing inputs before and after each command: source ce4bbf6f9e6a0c7e94ccdfec09d97f9355b5e9bb1a6f49fcca2b17fe7256482a. No Go/runtime/module/fixture changes precede these tests. Commands were executed from the repository root, with OPENAI_API_KEY removed, GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, GOCACHE=/tmp/alan-phase1f-go-build, GOFLAGS=-mod=readonly, ALAN_QUALIFICATION_OUT=/tmp/alan-phase1h-baseline.

| Command | Exit | Test-pass events | Seconds |
| --- | --- | --- | --- |
| python3 docs/foundational/phase1g/tools/qualify.py default |0|764|53.249|
| python3 docs/foundational/phase1g/tools/qualify.py normal |0|1006|62.648|
| python3 docs/foundational/phase1g/tools/qualify.py consumer |0|20|6.444|
| python3 docs/foundational/phase1g/tools/compare.py normal |0|15/20/32 exact semantic records and6 declared conformance comparisons|not timed|

The command metadata and compressed raw Go JSON retain actual macOS27.2 arm64 / Go1.27.3 execution, numeric exits, source identity and commands. Raw paths in those immutable command records refer to the original /tmp location; copies here have identical bytes. This is fresh current-source regression against the frozen source inputs, not a claim that archived source was extracted and rerun.

`archive-integrity.json` separately verifies SHA256, exact members and every member byte against both retained Phase1G archives: source43b86781 (499members) and ce4bbf6f (613members). That is historical archive integrity verification, not new historical execution. Original broad qualification remains associated with43b86781. No new baseline race result is claimed; the final changed-source qualification owns the next required race run.

`phase1g-current-docs/` preserves original00-plan,03-review and04-runbook guidance before the explicitly requested historical-label/direct-example cleanup. Frozen source/archive manifests, archived docs and raw historical evidence are untouched.
