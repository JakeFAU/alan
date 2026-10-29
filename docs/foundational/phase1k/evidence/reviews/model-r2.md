# Phase 1K Task 2 model/oracle rereview r2 — APPROVE

Spec compliance: **APPROVE**. Code/research quality: **APPROVE**.

**R1 is addressed.** This is a scoped rereview of the peer round-trip causality
correction and any new breakage introduced by its diff. Together with the
unchanged graph/evidence assessment in model-r1, it approves the initial Task 2
reference model/oracle gate on the exact corrected freeze below. It does not
approve the deferred A/B/new-context instruction correspondence, raw adapter,
public-driver, automatic-runner or recovery qualification.

## R1 resolution

`research/communication/oracle.go:257-266` now requires both the original report
Send sequence and its first authorization source-snapshot sequence to precede
ACK consumption in the same peer. The report, authorization and ACK-consumption
records already carry the checked original identities. These two strict
comparisons close the peer side of the causal round trip without introducing a
total order across different actors.

The exact R1 mutations are covered at `oracle_test.go:172-177`: ACK consumption
at peer sequence 1 before report Send34, and first report authorization at
sequence35 equal to ACK consumption35. A third case checks consumption at the
Send34 equality boundary. Each case first checks the valid control and then
requires its specific violation code; reversing both event and message arrays
must preserve the rejection (`oracle_test.go:179-194`). This tests the missing
semantic obligations rather than accepting an unrelated failure.

The guard requires an actual ACK-consumption record before applying these bounds.
Consequently ordinary J1 traces without an ACK and honest J2 refusal or
admitted-but-unconsumed prefixes remain unaffected. The old source-authorization,
retry, payload, identity, terminal and aggregate checks are unchanged. The new
comparisons reject equality as required: a source snapshot that has reached an
ACK-consumption commit cannot precede the authorization needed to create that
same ACK's causal antecedent. No new issue was found in the fix.

## Evidence and preservation

Independently compared current source with the initial frozen archive. Only
`oracle.go`, `oracle_test.go` and the README changed: two guarded causal checks,
three regression cases with reordering checks, and an accurate disclosure of the
original rejection. Record types/API, model.go, explorer.go and model_test.go are
byte-identical. All 26 reused join/substitution sources match both freezes and
HEAD. No new J2 artifact directory existed during this rereview.

Verified every entry of the original 18-entry evidence manifest and revision2's
13-entry manifest, all six members of each source archive, all current source
hashes, and all recorded command-output hashes. There were no mismatches.
The initial REJECT report, freeze, source archive and failing diagnostic remain
unchanged; this approval does not rewrite them.

Inspected execution evidence, without rerunning routine suites:

- `red-command.json` binds the three regression failures (exit1) to the original
  oracle hash and the exact new test-source hash. `regression-source.go.txt`
  equals the current complete oracle test file.
- `green-command.json` binds the same focused regressions to the corrected
  oracle and records exit0. The original valid control and both array orders are
  exercised by those tests.
- Post-freeze `commands.json` records communication normal and race suites and
  vet with exit0, each bound to the corrected freeze. The normal/race logs include
  the three regression subtests plus the preserved honest-pending/reordering
  checks; this is not a zero-selection gate.
- The semantic all-maximal-states-complete assertion still exits1 for its intended
  reason: two of four maximal states complete, with two ACK-refusal pending.
  Normal/race logs retain 1,408 states, 2,886 edges, rank800..8541 and exact path
  count245247050897402400. The original compressed census is reused under its
  unchanged hash, not presented as a new independently generated artifact.

Recorded checks use OPENAI_API_KEY absent, GOTOOLCHAIN=local, GOPROXY=off,
GOSUMDB=off, GOFLAGS=-mod=readonly and GOCACHE=/tmp/alan-phase1k-go-build.
Only read-only source/evidence inspection and SHA256 comparisons were executed
by this reviewer; they exited0. No additional diagnostic was needed. Only this
report was written: no implementation edit, test rerun, commit, TeX build or
nested agent.

## Exact reviewed identities

- Corrected freeze: `7ef775362562faa6b7ac1c4a364638545160be57b75daed81577b5417e0e7868`.
- Corrected archive: `03615bb85188f443c195a418744d2f77f14943c1bbf0a91448a9614d86adba34`.
- Revision2 evidence manifest: `b37b28fe58bbbb7710780346a6f96ddebaeb0cc02da3fa7f739d9ddc83e0ceb7`.
- Approved contract: `075debb7c6596d70fb8438367f30399feefde326a6005e2be63603583178d544`.
- Fix review package: `2db6662756cf7899070cbc423b1a0cf9d058b551a267d58538a47a5beb11fce6`.
- Appended Task2 report: `c458ce81d21e3f424e58a72255563c806e9eb9403c3d28bdd71c2a94b9e8dbd7`.
- Preserved model-r1 report: `1aedd762aa4bee61a604038656d207f86ba8433ea73620406857b6b37431155b`.
- Preserved initial freeze: `afa248134d846a0891bf3ec7caa1bca180f2f17fb0d66ebf193bd6db41715802`.

| Current frozen source | SHA256 |
|---|---|
| `research/communication/explorer.go` | `ba5f401b26754b8591a64fa5acfaa27bce30a32b8b52519c195c79eaaf057222` |
| `research/communication/model.go` | `e8771d0c88e59472d9e6296e1dc348c29b4e48f1cfa33f72cad1771fe4fdd4ca` |
| `research/communication/model_test.go` | `9a62c49e34655a3a56db9e9c9af7c949a634738e5d9f2f2fd69b693f31210146` |
| `research/communication/oracle.go` | `7cf146dc425bcff31baedec31059eb0bc21c24f9738bd8c19dea9b12e3dd9bb4` |
| `research/communication/oracle_test.go` | `90a20734065c60cea0661e30e4595e9dac29f040df839fed5448b700eeaef9fe` |
| `research/communication/README.md` | `a7de619fc9373df3896b807879912a34c33b6a19d2f9efee6e6ca3896667d834` |

The Task 3 instruction/stutter mapping must still independently preserve enabled
service and terminal transitions for the actual new artifacts. This approval
establishes neither automatic-runner failure nor a general completion or
non-expressibility theorem.
