## Final narrow fix — N1 decoder fixture isolation

Scope: only the fixture setup in internal/substitutiontest/communication_decoder_test.go.
The final whole-branch review approved the bounded specification/research result
and recorded nonblocking P3 N1: selecting wrong-dispatch-link alone skipped its
sibling source setup and panicked before the intended decoder rejection. The
original review remains unchanged at evidence/reviews/final-r1.md, with an exact
copy in runtime/final-fix/final-review-before.md.

The exact single-leaf command was reproduced before editing:

```
env -u OPENAI_API_KEY GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=1 GOCACHE=/private/tmp/alan-phase1k-runtime-cache go test -tags integration -count=1 -timeout=2m -run '^TestCommunicationDecoder$/^wrong-dispatch-link$' ./internal/substitutiontest
```

single-leaf-red exited 1 in 0.219 s with assignment to entry in nil map at old
line 86. Source hash was 59e3751bacfd411ec6857ea18a3d4251a0e4865a20365aaca5194840d64ed8b2.
The retained before source, five-file archive, current review and command records
precede the change; no historical gate or review was rewritten.

The test now obtains late/early/admitted sources through one lazy genuine-fixture
helper. Source controls and every dependent selected attack, reordered-input
control and J2-unavailable guard request their required fixture directly. Cached
snapshots avoid duplicate construction during a complete decoder selection.
The actual public B/S paths, admitted-unconsumed control, copied inspections,
mutations, nonempty-mutation assertion and decoder-error assertion are retained.
There is no skip or empty substitute when a fixture is absent. No production,
model, context, decoder implementation, driver or process source changed.

The same exact single-leaf command now exits 0 in 4.054 s. A separate -v proof
exits 0 in 4.779 s and lists exactly one leaf:
TestCommunicationDecoder/wrong-dispatch-link. No source-* setup sibling ran in
that proof. Its genuine late fixture was constructed on demand, and its actual
retained rejection is report acknowledgment original admission mismatch.

Affected checks on the same frozen final five-source bytes:

```
go test -tags integration -v -count=1 -timeout=10m -run '^TestCommunicationDecoder' ./internal/substitutiontest
go test -race -tags integration -v -count=1 -timeout=30m -run '^TestCommunicationDecoder' ./internal/substitutiontest
go vet -tags integration ./internal/substitutiontest
```

Normal exited 0 in 18.190 s; race exited 0 in 253.591 s. Each selected both
TestCommunicationDecoder and TestCommunicationDecoderReportIdentity with all 21
leaf cases passing: the 18 original decoder leaves and the genuine-pre-report,
wrong-Target and wrong-Obligation controls. Focused vet exited 0. The code remained
frozen throughout these checks. Commands used Go 1.27.3, local/offline/readonly
CGO settings, the existing temporary cache, absent provider credential and fresh
absolute ALAN_I_ARTIFACTS destinations under runtime/final-fix/.

Latest qualification is explicitly decoder-only. The full 27-leaf normal/race
campaign remains the immutable R2 evidence on aggregate
e40b4b699b1682cce99718ca95d7d12292a6637025635843cf9ffba72e7b64c1.
Those results are not relabeled as runs of the final fixture version. Unchanged
driver/process/automatic campaigns were not repeated; no observed regression
required their rerun. The bounded research outcomes and J2 construction limit
remain as reviewed.

Current five-source aggregate SHA256:
e174c4b7ca2e256259045ecdc117378111ecbb3d3de14b0ca5f7d88affb5eb04.
Changed decoder-test SHA256:
81513baaf1c9c8ee9359c3924436e004af0868cc54968f6f3ef458634699e1ba.
The other four test-source hashes match R2. Exact patch SHA256:
ad26f7ec9bcd2eadcdc17de2ed83bd2560533f093b8a5e31456fc1ed3a26a46d.

runtime/final-fix/source-identity.json records before/after hashes and source
archives; fixture-isolation.patch.txt is the complete scoped diff. The separate
evidence-index.json binds command/environment/output/source hashes and exact
selected leaves. All 621 preexisting runtime/review files, including revision2,
were verified byte-identical against prior-artifacts-sha256.json. The prior Task4
report is preserved as an unchanged prefix and this new report is appended to it.
Only this new report is copied to evidence/reviews/final-fix-report.md; the
historical copied Task4 report is not overwritten.

Formatting and diff checks passed. No manuscript/narrative source edits, protocol
change, nested agents, commits or TeX were performed. Scoped final rereview of
this N1-only correction remains pending.
