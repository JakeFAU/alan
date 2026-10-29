# Task2b evidence and reproduction

`index.json` binds the approved five-file reference freeze, source hashes, exact
command records and `qualification.json.gz`. The gzip JSON has `census` (all
maximal states/traces and local subset records), `artifact_extraction_json` (exact
source-built bytes) and `correspondence` (checked finite instruction mapping,
actual IDs/differences and syntactic bounds). Nothing here is a new retained
runtime execution. Prior reference-r1/r2 archives remain unchanged one level up.

Run from the repository root with its recorded current source. No network or
provider is used. These commands produce fresh outputs, leaving archived records
unchanged. `capture_check.py` records exact commands for this task and deliberately
refuses to overwrite a command label. Its stored workspace path must be adjusted
for a different checkout; the tests themselves have no workspace-path dependency.

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export GOCACHE=/tmp/alan-phase1f-go-build
unset OPENAI_API_KEY
phase1l_repro=$(mktemp -d /tmp/alan-phase1l-reproduce.XXXXXX)
export ALAN_MODEL_EVIDENCE="$phase1l_repro/normal"
go test -run '^TestInvocationJoinReferenceQualification$' -count=1 -v -timeout=10m ./research/substitution
go test -run '^TestInvocationJoinReplacementQualification$' -count=1 -v -timeout=10m ./research/substitution
go test -run '^(TestInvocationJoinLocalRefinementSubset|TestFrozenInstructionCorrespondenceCoverage)$' -count=1 -v -timeout=10m ./research/substitution
export ALAN_MODEL_EVIDENCE="$phase1l_repro/race"
go test -race -run '^(TestInvocationJoin.*|TestFrozenInstructionCorrespondenceCoverage)$' -count=1 -v -timeout=10m ./research/substitution
unset ALAN_MODEL_EVIDENCE
go vet ./research/substitution
cp docs/foundational/phase1l/evidence/model/task2b/extract_inputs.go.txt "$phase1l_repro/extract_inputs.go"
go run "$phase1l_repro/extract_inputs.go" "$phase1l_repro/artifacts-inputs.json"
python3 docs/foundational/phase1l/evidence/model/task2b/check_correspondence.py "$phase1l_repro/artifacts-inputs.json" "$phase1l_repro/correspondence.json"
```

The census `States` key is model × prefix-monitor identity; `ModelStates` is also
reported. Each generated edge is checked before deduplication, no failure pruning
or cutoff. Maximal-state traces are representatives, not all schedules. The rank
cap is10,621 and initial rank100; eventual service excludes infinite stuttering.
The artifact checker compares every actual A/B/helper instruction to the inherited
mapping, explicitly reconstructs parent/S instructions, checks all forward targets,
and computes longest syntactic paths. Dispatch6/9 is a source-inspection protocol
bound, not a Python enumeration of the production runner. Dynamic binding,
authentic raw selected observations, publication/delivery atomicity and process
recovery require the separately gated Task3 qualification.
