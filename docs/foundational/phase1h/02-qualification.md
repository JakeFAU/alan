# Phase1H qualification — current status

**Required local qualification and packaging gates are complete. Independent whole-branch review returned APPROVE at `ed3006859dcdf075e92192b0722afc6f29ec2c43`.** The full default/integration/race, both consumers, applicable static checks and archive/extracted gates below have completed. These are finite controlled single-host implementation results, not general correctness, liveness, distributed or power-loss proofs.

Frozen source is `770df465af2bfc3bfa78547b6ad821427fbe69a01cd5ad284ff9030e82269912`,277 contributing inputs, head at freeze `680be378e0104d5fcf7d045fced8123b0909f189`, main base `56892cb20c4f4add4bcf786d146b0aed82fbc5d1`. [Source association](evidence/task3/final/frozen-source-association.json) and [selected-input proof](evidence/task3/final/guard-source-equivalence.json) separate earlier actual results from the final source. Source remained unchanged in every completed final capture.

## Completed commands

| Gate | Actual source | Exit / seconds | Go pass events |
| --- | --- | --- | ---: |
| Fresh full default |4a92f319|0 /1688.792|855 across18 packages|
| Full integration |770df465|0 /1899.128|1146 across24 packages|
| Full root race |770df465|0 /21481.340|1146 across24 packages|
| Worktree consumer normal |4a92f319|0 /10.633|20|
| Worktree consumer race |4a92f319|0 /104.896|20|
| Adaptive consumer normal |4a92f319|0 /123.637|12|
| Adaptive consumer covering race |770df465|0 /2476.348|12|

[Machine-readable command index](qualification.jsonl) points to exact argv/environment/platform, raw logs, source archives, hashes and records. Default source4a92 remains its actual label. The complete Go-selected default identity is unchanged through the integration-only watchdog/tool/prose edits: SHA2565e034ddcfc1809a05a04232fa42a1a696fb69d17026048d5c3b6548723d71f93,1509files/288packages, including recorded settings, module and frozen fixture inputs. Runtime integration tests are not inferred from the default command. Consumer production/module/test inputs also remain unchanged; original results are not relabeled.

Both vet modes, both consumer vets, all three module verifications, root/new-consumer builds, correct old-consumer test build, final Go formatting, three collector contracts, five semantic-provenance controls,13 Python tool formatting and actual documented loopback CLI smoke passed as individually captured. The integration-watchdog delta received fresh integration vet and actual normal/race companion checks. The unsuitable normal build of the test-only worktree module failed and remains alongside the corrected successful test-build command. No live provider credential or request was used.

## Adaptive boundary evidence

Full integration contains224 ADAPTIVE records, all eight named qualification groups, all six process cut/recovery cases, two next-frontier root denials, two failed-plus-unknown schedules, six capacity/out-of-order cases, two primary closures, the separate real late-transport companion, isolated replay and authentic faulty-application rejection. Named-control checks passed. Actual process SIGKILL/fresh reopening, original identities/charges, full accepted count before first intent, unfinished sibling at partial consumption, after-contact ordinary progress, conserved unknown reservations and source-versus-scope denial are independently inspectable in retained states.

Runtime/replay protocol validity and application-policy validity remain distinct. The faulty ordinary program consumes authenticated gamma0, substitutes2 and actually admits forbidden10/11/31; the unchanged paired oracle rejects it while Run/Replay accept its protocol. Closure is two disclosed schedules, not a complete adaptive barrier containing a required unknown result. Completion cuts use the separately identified128-iteration pure fixture. Capacity adds a fixed ordinary progress member; static root+witness cost2 is separate from dynamic charges.

All12 byte/profile records passed in each actual default, integration and race run. Actual measured maxima are adaptive Projection285270B, root111158B, wrapper111218B, unchanged helper151321B and program117295B; largest canonical decision array54327B. These fit the unchanged declared limits. The [fresh default peak records](evidence/task3/final/default-byte-summary.json) retain exact scope, variable and transition addresses; old fd7 measurements keep their historical identity.

## Historical comparisons

Actual default/integration/race15/20/32/6 comparisons passed:15 qualification payloads exactly;20 composition/32 supervision records with old/new complete manifests and every inner source identity validated first, then19/31 exactly equal ordered behavior records; six unchanged public conformance projections/model streams and Go pass events. Only the pinned user-merged one-line input permission addition `Obligation: "control"` is mapped. The exact pre-main bridge bytes are retained as a `.go.txt` input. This is not historical raw-source, raw-metadata or whole-runtime-trace equality. Manifest/inner-source tamper and behavior state/order/count changes are rejected by controls. Strict race comparison passed. Its separate named-control check additionally required actual child race instrumentation and watchdog scale20, and passed.

## Immutable archive and actual offline extraction

Archive SHA256 `2e9efb6136f3de728cff2a09c3a7a3d0686375e6054577e4bb384414102d1d36`,5,258,134bytes/309 safe regular members, contains all277 source inputs,27 separately declared prose snapshots, source manifest and four pinned sqlite proxy members. [Member manifest](archive-manifest.json), [source manifest](source-manifest.json) and [package-time context](evidence/task3/archive-context.json) define the exact bytes. Generated archive prose retains its package-time status; these adjacent current reports may advance later without rewriting the archive or changing source inputs.

[Actual extracted reproduction](evidence/task3/extracted/reproduction.json) passed all six commands in259.806s, with initially empty module cache, bundled file proxy, credential removed and unchanged source. Root/both consumers compiled; all seven exact required focused control paths and eight public-consumer paths passed;15/20/32/6 semantic comparison passed. The Go build cache is deliberately reused and the installed Go/C toolchain is not bundled. Whole-reproduction OS network isolation is not claimed; isolated replay separately uses its actual deny-network policy. This is focused extracted-source reproduction, not an extracted full default/integration/race suite.

## Failures, review and open obligations

Original first-controls inspector defects, whole-frontier counterexample and RED/GREEN, rebased capacity snapshot-order failures, the15m adaptive-consumer race timeout, unsuitable old-consumer build and naive raw-provenance comparison remain failed evidence. The consumer timeout's complete stack and partial CLI passes are preserved separately from the successful60m covering command. The selector-omission hypothesis was disproved by actual Go matcher output; its diagnostic assertion failure remains distinct from ALAN evidence. No original failed result is relabeled or test weakened.

[Review scope](03-review.md) retains contract/application/harness/scoped approvals and the same whole-branch reviewer's stage1/stage2 inspection. The same independent reviewer completed the final race and metadata inspection and returned [APPROVE](evidence/task3/final-review/alan-phase1h-final-verdict.md) for specification compliance and code quality, with no unresolved actionable findings. This is bounded review evidence, not a proof of universal correctness. [Controller rulings](evidence/task3/controller-rulings-snapshot.md) preserve the detailed decisions. No GitHub Actions workflow exists in this repository; local commands are not remote CI results, and draft publication/actual available checks belong to the controller's later gate.

Open research limits include individual cancelled/abandoned child-envelope permutations, automatic eventual repair of unknown non-idempotent work, unbounded fairness/liveness, distributed ownership, power-loss recovery, compiler correctness and business exactly-once. The manuscript audit leaves broader loop-ranking presentation, fair-schedule confluence, wider Markdown–LaTeX alignment and overbroad proof rhetoric unresolved. Initial full tests-first chronology was not achieved; constructor RED and later meaningful regressions do not retroactively establish it.

## Final integrity inspection

The [post-capture inspection](evidence/task3/final-integrity/result.json) verified all277 frozen inputs,309 archive members,41 captured commands,138 retained artifact references, all six extracted commands with required7/8 actual pass names and13 separately hashed review outputs. Race completed exit0/21481.340s with1146 passing test paths/24 passing packages, zero failed events and no emitted data-race warning. Its raw SHA256 is `f8a43edd490831e84d00587bcf3365f28e88b3c74eee5baa317cd17acab8730e`. The generated inspector’s first attempt compared Encoder JSON including its newline to a Marshal value digest; its compressed hash matched, and the corrected inspector removes exactly that one terminating newline. The original attempted inspector and diagnosis remain separate inspection evidence, not an ALAN failure. No Go source or test was changed or rerun for this correction.
