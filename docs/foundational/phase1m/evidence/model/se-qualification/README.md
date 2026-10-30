# C_se qualification and postfreeze correspondence

The approved reference is unchanged: all36 frozen inputs still match
`../reference-r1.json.gz`, source aggregate
`cdfe2a191641129dc591b92aabd680d238a80a1e027cfb77a82edc84fd465cd1`.
Qualification began after independent reference APPROVE. K1 completed before A/B
in each normal/race profile; recorded completion/start times check this ordering.
No runtime qualification is claimed by this slice.

| Product | Configurations | Model/product states | Edges | Terminal maxima | Largest rank |
| --- | ---: | ---: | ---: | ---: | ---: |
| C_se[K1] | 4 | 1996 | 4376 | 8 | 7800 |
| C_se[A] | 4 | 944 | 2046 | 4 | 5706 |
| C_se[B] | 4 | 1784 | 3842 | 4 | 7813 |

Inputs are id∈{1,2}, value∈{0,1}, mode=answer, I=R3/depth2/E0,
one fixed parent. Initial rank100, cap10621. All rank/state/prefix/component/G
failures, rejections, pending/stuck maxima, cycles, cutoffs and counterexamples
are zero. K1 preserves both branches: terminal dynamic spending is
(I_spent,R_spent)=(2,2) or (3,3); A spends (2,2), B spends (3,3). The common
limits remain I=R=3 throughout. All have E0 and fixed-runs1. Every edge is checked
before product-state deduplication. The archive retains every reached maximal
state, observation and representative trace, not every equivalent interleaving.
Normal and race census structures are identical.

These establish bounded, conditional results separately: C_se[K1] satisfies
G_answer; the current answer-profile local relation/progress accepts A and B;
and C_se[A/B] satisfies G_answer. Eventual eligible service is assumed; infinite
idle/retry-only suffixes are outside the model. No general context congruence,
all-mode conformance, raw-record authentication or compiler theorem follows.
S projects only to an arbitrary-timing one-unit q debit for the local relation;
its payload/lifetime and the parent's original joins need the separate product.

The focused C_es regression has K1/A/B states1952/926/1706,
edges4276/2006/3678 and maxima8/4/4 over four inputs each. The unchanged historical
public wrappers still choose candidate-first; the frozen before/after source
argument is now supplemented by these current product checks. Current local
A/B relations cover300/632 pairs,156/344 component checks,600/1172 environment
checks,52/104 stutters. Separate Closure=false progress covers120/220 states,
184/338 edges,4/4 terminal outcomes; all failures/pending/stuck/cycles/cutoffs zero.
Each normal/race profile passes85 leaves:4 new K1,8 new replacements,23 historical
product/local/instruction and50 construction/oracle. Vet/format/discovery pass.

The postfreeze mapping uses the reviewed Phase1L checker with five explicit,
assert-count-one substitutions for the S-first receive order and new source
paths. The original checker is source-hashed, not copied into another framework.
It checks all89 instruction occurrences: inherited A13/B16/helper7, S3 and both
25-instruction parents. It reconstructs actual guarded control, keeps setters,
branches and jumps as private stutters, and places parent_decide at the actual
resolved-End boundary. The parent sources use original operation, template,
provenance, generation and obligation matching. Candidate templates/programs,
helper and S are byte-identical to frozen originals. All eight request/binding
constructions differ only in the initial request, candidate binding and its
required catalogue closure. Artifact identities and exact bytes are archived.

| Concrete bound | A | B |
| --- | ---: | ---: |
| Longest forward local commits | 25 | 35 |
| Dispatch calls under stable service | 6 | 9 |
| Maximum retained program bytes | 11719 | 11719 |
| Conservative bound-program bytes | 37319 | 37319 |
| Conservative projection bytes | 26400 | 26400 |
| Conservative checked envelope bytes | 53056 | 53056 |

Compiled paths are parent16/A6/S3 and parent16/B10/helper6/S3; all targets advance.
Configured caps are200 steps,200 dispatches,capacity1 and four byte caps1048576.
The conservative64KiB common byte bound uses at most16 scalar registers, at most5
payload fields,75-byte original handles, fixed reasons/hashes, and a256-character
scalar text allowance with worst JSON escaping. The named protocol structs have
at most16 scalar slots in this profile; the envelope calculation allows32 plus
key/integer margin. At most one intent is emitted per reduction. Bound-program
growth conservatively allows four256-byte substitutions per instruction. These
are source arguments plus finite artifact checks for pure positive executions,
not sampled runtime maxima or arbitrary payload/adapter/error bounds. They are
separate from the model's finite-state rank.

**Preserved dispatch correction:** `.superpowers/sdd/00-plan-phase1m/task-1-report.md`
lines137–139 and `../../reviews/task1-r1.md` lines101–102 described invocation,
request and reply dispatches. The request observation is inserted atomically by
admission (`internal/executor/invocation.go:479`). The three possible runner calls
are initial invocation admission, original reply delivery, and ready original
outcome settlement/delivery (`runner.go:268,385`). An admitted unready invocation
is omitted from pending deliveries, and a delivered one is removed. Root
settlement is not report.Dispatches. The numeric6/9 bound and prior measured6/9
remain valid. Earlier report/review wording is preserved; no runtime/code defect
or rerun is implied by this correction.

`records.json.gz` contains15 completed command records with exact argv, pinned
environment, source hashes, stdout/stderr/exits and selected leaves; normal/race
censuses and local checks; compiled artifacts; mapping; capture sources and
history. `index.json` supplies hashes and aggregates. One initial exporter/capture
filename collision produced valid artifacts but prevented its command record
from being saved; history records this limitation. Repeating under separate
names produced identical artifact bytes and a complete successful command
record. Initial/final checker versions preserve the prose clarification.

Reproduce from repository root, with a fresh output directory (capture refuses
existing command labels). The source was Go1.27.3 darwin/arm64, CGO1. `capture.py`
sets local/offline/readonly Go, removes ALAN_* and OPENAI_API_KEY, and records the
explicit ALAN_MODEL_EVIDENCE override for census commands. It does not call a live
model or run a store.

```sh
q=docs/foundational/phase1m/evidence/model/se-qualification
o=$(mktemp -d /private/tmp/alan-phase1m-se-recheck.XXXXXX)
cp "$q/extract.go.txt" "$o/extract.go"
for profile in normal race; do
  race=
  if [ "$profile" = race ]; then race=-race; fi
  python3 "$q/capture.py" "$o" "reference-$profile" "$o/reference-$profile-census" /opt/homebrew/bin/go test -json $race -count=1 -timeout=5m -run '^TestConsumerPolicyReferenceQualification$' ./research/substitution
  python3 "$q/capture.py" "$o" "replacements-$profile" "$o/replacements-$profile-census" /opt/homebrew/bin/go test -json $race -count=1 -timeout=5m -run '^TestConsumerPolicyReplacementQualification$' ./research/substitution
  python3 "$q/capture.py" "$o" "local-historical-$profile" "$o/local-historical-$profile-census" /opt/homebrew/bin/go test -json $race -count=1 -timeout=5m -run '^(TestInvocationJoinReferenceQualification|TestInvocationJoinReplacementQualification|TestInvocationJoinLocalRefinementSubset|TestFrozenInstructionCorrespondenceCoverage)$' ./research/substitution
  python3 "$q/capture.py" "$o" "construction-$profile" - /opt/homebrew/bin/go test -json $race -count=1 -timeout=3m -run '^(TestJoin.*|TestConsumerPolicy(SFirstOracle|RejectsCandidateFirst|FirstSelectionTransition|PendingPrefix|OracleRequiresOriginalJoins|HonestFailureIsNotConformance|AbsentPayloadAndFailedS|DirectedReferenceBranches|DomainAndStateGuards))$' ./research/substitution
done
python3 "$q/capture.py" "$o" artifact-extraction - /opt/homebrew/bin/go run "$o/extract.go" "$o/artifacts-inputs.json"
python3 "$q/capture.py" "$o" correspondence-final - python3 "$q/check_correspondence.py" "$o/artifacts-inputs.json" "$o/correspondence-final-output.json"
python3 "$q/capture.py" "$o" vet - /opt/homebrew/bin/go vet ./research/substitution
python3 "$q/capture.py" "$o" format - /opt/homebrew/bin/gofmt -l research/substitution/consumer_policy_qualification_test.go
python3 "$q/capture.py" "$o" discovery - /opt/homebrew/bin/go test -list '^(TestConsumerPolicy|TestJoin|TestInvocationJoin|TestFrozenInstructionCorrespondenceCoverage)' ./research/substitution
python3 "$q/capture.py" "$o" toolchain - /opt/homebrew/bin/go env GOVERSION GOOS GOARCH CGO_ENABLED GOTOOLCHAIN GOPROXY GOSUMDB GOFLAGS GOCACHE GOMOD
```

`python3 "$q/bundle.py" "$o"` regenerates the archive/index in this evidence folder;
copy them aside first if preserving the original capture. The original archive's
failed-attempt history is included only when those observed inputs exist.
Stop at Task2b review. Task3 must authenticate actual original retained protocol
records and consumption, both completion orders, S-consumed/e-required state,
and pending controls. C_reason and manuscript reconciliation remain later work.
