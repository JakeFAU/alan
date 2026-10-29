# Task 4 scoped R1 rereview, round 2

Specification verdict: **APPROVE**.
Code and research quality verdict: **APPROVE**.
R1 is addressed. New actionable findings: **none** within the correction and
regression scope. The earlier REJECT and counterexamples in `runtime-r1.md`
remain valid historical evidence and are preserved unchanged.

## Scope and result

Reviewed the appended Task 4 report, fix review package, exact patch, current
five runtime test sources, revised manifests/archive, regression RED/GREEN
records, complete focused normal/race records, standalone audits, and copied
regression/process inspections. This is a scoped rereview of R1 and the
correction's effect on the previously reviewed Task 4 result. It does not approve
phase completion, a complete J2 runtime correspondence, or universal liveness.

At `internal/substitutiontest/communication_evidence_test.go:65-70`, the
no-report branch now runs `communication.CheckTrace` before returning success.
This closes the exact path demonstrated in R1: an original peer report with an
incorrect target or obligation remains in `out.Truth`, so the independent oracle
rejects its provenance and absent message mapping instead of accepting an empty
prefix. The main successful return already applies the same oracle. The change
is exactly three added lines; it does not change lookup identity, normalization,
public transitions, original programs, or the oracle.

`communication_decoder_test.go:163-222` adds the requested three cases. A real
public root Invoke establishes a valid pre-report prefix and still decodes with
zero messages. Two independently copied genuine post-Send/no-authorization
inspections alter the original report's Target or Obligation in both the intent
and matching history representation. The tests assert an actual lookup miss and
changed evidence before requiring rejection. Raw results preserve the mutated
inspection, decoder error, returned trace and independent oracle violations.
These are decoder attacks on copies, not authoritative edits or executed malformed
protocol histories.

The preserved pre-fix RED ran with the new tests and the old decoder, rejected
both attack assertions, and passed the genuine prefix control. The corrected
GREEN and complete normal/race artifacts retain both expected oracle violation
codes in decoder errors. Direct inspection confirmed that each attack changes
exactly one intent row and one history row, with no other table changes.
This avoids passing merely because the test damaged unrelated evidence.

## Independent confirmation

I reran only the exact original R1 overlay diagnostic against the original
archived checkpoint, using the unchanged witness source retained verbatim in
`runtime-r1.md`. Both Target and Obligation cases now pass. Command:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1 GOCACHE=/private/tmp/alan-phase1k-runtime-cache go test -overlay /private/tmp/alan-phase1k-runtime-review-r1/overlay.json -tags integration -count=1 -timeout=2m -run '^TestCommunicationReviewWrongReportIdentity$' -v ./internal/substitutiontest
```

Observed exit **0**, two passing leaves, package elapsed **0.239 s**. Witness
source SHA256 remains
`2e668ce488ce8dcd3a9e158d8f38aa1e08c607342f6f2562875ad4bf4a95c0f2`;
original input gzip SHA256 remains
`2edc6f53269aeb85640057378d53922dfca709a8a0f047f1c00e057f7f79697a`.
No broad or race suite was rerun by this reviewer.

Read-only independent Python checks recomputed all five current hashes and their
aggregate, verified every one of the 280 indexed revision2 artifact digests,
all eight corrected archive members against current files, pre-fix RED archive
hashes, every indexed command exit/output digest, and all 328 original runtime
evidence files. The original R1 report hash also matches. Comparing the original
qualified-source archive with current source confirms exactly two changed files:
the three-line evidence guard and the appended regression/control test.
Driver/process/runtime source bytes and the oracle are unchanged.

## Corrected-source qualification and regression risk

The recorded full normal and race commands select `^TestCommunication` and bind
to the same corrected five source hashes. Each output contains 27 passing leaf
cases; the subprocess-entry skip is excluded. Normal elapsed **36.533 s**, race
**535.773 s**. These are inspectable worker executions, not reviewer reruns.
Combined vet also exited 0. Local/offline/readonly Go and CGO settings, absent
provider credential, separate revision2 artifact roots, commands and exits are
recorded. The unchanged external consumer retains its earlier source-bound
visibility result and is not relabeled as rerun.

The standalone journal/process audit scripts are byte-identical to the previously
reviewed versions. Their recorded revised-run results cover 20 journals, 998
genuine commits and all six fresh real SIGKILL/reopen cuts. Direct inspection of
those six raw before/after pairs independently confirmed killed exits, distinct
new PIDs, matching acknowledgment tokens, original immutable prefixes, unchanged
programs/permissions/charges/admissions, and root epoch 2 to 3. The refused cases
retain identical attempts, admissions, dispatcher acknowledgments, observations,
history, intents, facts and root publications with root/aggregate pending.
The allowed and no-authorization cases reach actual consumption and success.
No-authorization recovery again has authorization epoch 3 at sequence 37 while
that sequence's original history has epoch 2.

The source change only adds validation to a previously unchecked success path;
it does not weaken existing validation or change scheduling/recovery. The genuine
prefix control and full corrected-source selection support the intended bounded
compatibility. The original public-driver/automatic/process conclusions retain
their prior classification and assumptions. J2 remains unavailable under the
reviewed construction obstruction. Replay is consistency evidence, and neither
audit counts nor this approval constitute a compiler or universal safety proof.

## Reviewed identities

Corrected five-source aggregate:
`e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1`.
Historical R1 aggregate remains
`07ee2145f258069aaaa169eeded371f2e1ff1cf48b344340e4681b9b5072a2d6`.

| Source/evidence | SHA256 |
|---|---|
| `internal/substitutiontest/communication_decoder_test.go` | `59e3751bacfd411ec6857ea18a3d4251a0e4865a20365aaca5194840d64ed8b2` |
| `internal/substitutiontest/communication_driver_test.go` | `453db8e8207e9e876e3883f270110dd8af184b3e929ab309798b5406ec78efd4` |
| `internal/substitutiontest/communication_evidence_test.go` | `231136367ced2f1682c031f6636a52b415c30e3c9d4286570fef42f7ea9e87f6` |
| `internal/substitutiontest/communication_process_test.go` | `60968566192b697da6b534409dba13be0934046bb71ac04db8c722d22be4a22c` |
| `internal/substitutiontest/communication_runtime_test.go` | `c3d7d4c7bfc47d0c00cb27a378a908cb8a094f8435f3018e29ef498364da8d13` |
| `research/communication/oracle.go` | `7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4` |
| `.superpowers/sdd/00-plan-phase1k/task-4-report.md` | `5fe7b7f91a014b1825c86bede2f6b95aff5000e556ddc7ae5b08ac4b3ac59b8a` |
| `.superpowers/sdd/00-plan-phase1k/task4-fix1-review-package.txt` | `85449b55104045ae45b4a4c9ef2bf6436da091b66f6d8efe5f9bfac7198faa2f` |
| `evidence/runtime/revision2/r1-fix.patch.txt` | `29430399db7dd88e80f6cb021273e0b4ca1730010bce3a26093a2cd050de10a5` |
| `evidence/runtime/revision2/source-identity.json` | `95a391acc12790c565851f5bb5513772b1b512ab0ae3a031c02a373944348795` |
| `evidence/runtime/revision2/evidence-index.json` | `9dc2b3aa9f8f86f878a00a0a5d0524ad9f65b7f6dea621ffd1dd3402546d2f87` |
| `evidence/runtime/revision2/prior-evidence-sha256.json` | `3b09682e4db5a554319898749a1294f546b254ebc7b8b53a1ef1b1acf3971229` |
| `evidence/runtime/revision2/qualified-source.tar.gz` | `58412ebb92f85c2ca173126902d088fd68bf16d2f263ea843504a524e66db0af` |
| `evidence/runtime/revision2/red-qualified-source.tar.gz` | `a66c1c03a9e7d874636853f29ac467086849ef0ab81221c9883371eb92aed1c2` |
| `evidence/reviews/runtime-r1.md` | `5bfe319cfbef4ecef745fbc238f347cb879e7489f7c292d780c4f5c756aee509` |

No source fixes, commits, TeX runs or nested agents were performed in this
rereview. Phase-wide manuscript/gate/PR obligations remain with the controller.
