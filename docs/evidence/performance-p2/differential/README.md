# P2 same-seed HTTP differential qualification

This separate public Go module is semantic qualification, not timing evidence.
It leaves the frozen performance driver and P1 qualification tools unchanged.
Report protocol: `alan/performance-p2-http-differential/v1`. Comparison protocol:
`alan/performance-p2-http-differential-comparison/v1`. Campaign protocol:
`alan/performance-p2-http-differential-campaign/v1`. These are distinct from the
P1 differential and P2 timing report protocols.

`main.go` reuses P1's genuine public Create/Close seed, Open, public continuation
basis/approval, one Run per selected position, complete snapshot, every History
page, Replay and SeriesView flow. It imports only ALAN's public packages. No Go
SQL queries, internal imports, new failure protocol, retry or timing collection.
The only scenario is success. Existing unchanged P1 direct refusal cases and real
weather-process accepted/lost/retained/crash fixtures supply failure qualification.

`transport.go` is a byte-for-byte copy of the frozen P2 driver's reviewed HTTP
transport: canonical complete Prepared envelope, original ID/key, frozen digest,
stable action Wire, deterministic exact response bytes, independent handler
journal, finite request/response bounds, no retries and drain-before-final-report,
including failures. Both journals retain every field exactly. Handler values are
derived from incoming HTTP bytes, independently of the adapter-entry journal.
The server runs in each consumer invocation, owns its listener, and fully drains
before either final journal is copied. This is not a process-resource experiment.

`proof.go` adapts the frozen driver's complete final identity/provenance proof to
each already selected prefix. Its assertion-only view is restricted to selected
positions; the full original snapshot and SeriesView stay unchanged in the
report. `prefix_proof.go` copies the driver's returned-root verifier, omitting
only its unused compact-record revalidation function. It validates original root
membership, charges/outcomes, earlier completed roots, future open roots,
OpenRuns, steps, dispatches, pending and unknown evidence on every returned Run.
Complete RunReport, including every root and pending/unknown list, is captured
rather than replaced by the compact verifier result. The no-work Run after each
reopen is also captured and validated.

Every cut includes full adapter and independent handler journal prefixes. The
producer reconciles retained Prepared/Candidate, request and response identities,
digests, byte strings, charge/selection and protected predecessor evidence.
The comparator retains all report fields and typed SQL schema/cells, ignoring
only `source_label`, `store` and physical SQL row order. Array order, IDs,
namespace, source revisions, epochs, generations, root IDs, authorities,
approvals, scoped grants/debits, prepared/result digests and contact fields are
never normalized. `compare.py` copies P1's exact equality/table-dump mechanics,
removes its unrelated timing normalization mode, versions the protocol and adds
fail-closed complete HTTP/cut/RunReport/journal/provenance shape validation. A
matching final weather value cannot substitute for these records. It checks
returned roots also equal that cut's complete inspection roots.

`clone_seed.py` is the unchanged P1 clone guard. It requires a genuine closed,
unexecuted seed, holds the existing advisory owner lock, rejects nonempty WAL,
checks SQLite and empty executed history, rejects existing outputs/sidecars and
retains byte hashes. Run it only against the new seed, never archived originals.
SQL comparison/mutation tools operate on closed disposable copies only. Original
P1 crash archives and archive reader tools remain unchanged.

## Serial reproduction

Root must serialize these commands with other qualifications and timing work.
Use immutable selected source exports, exact source hashes and fresh absolute
output paths. Source labels are caller supplied, not self-verifying claims.

```sh
TOOLS=/absolute/alan/docs/evidence/performance-p2/differential
"$TOOLS/build.sh" "$BASE_SOURCE" "$OUT/base-http-differential"
"$TOOLS/build.sh" "$CANDIDATE_SOURCE" "$OUT/candidate-http-differential"
python3 "$TOOLS/campaign.py" \
  --baseline-binary "$OUT/base-http-differential" \
  --candidate-binary "$OUT/candidate-http-differential" \
  --baseline-source "$BASE_ID" --candidate-source "$CANDIDATE_ID" \
  --output "$OUT/http-campaign"
python3 "$TOOLS/verify_tools.py" --binary "$OUT/candidate-http-differential" \
  "$OUT/http-campaign" "$OUT/http-tool-proof"
```

The controller reserves one numeric loopback port once, releases it, records the
address, and passes that exact fixed address to each serial source invocation.
Both ordinary8 and reopen8 clone the same genuine seed. Reopen8 closes/opens and
runs the no-work resumption after every selected position, including position 8.
If a bind race or any runner/reader/comparison fails, preserve all outputs and
stop; there is no endpoint/sample replacement. Individual execute-existing calls
require `-endpoint 127.0.0.1:PORT` with a nonzero numeric port. Create-seed starts no
server or adapter. Both source runs retain identical endpoints without mapping.

The controller records argv, exits, binary/tool hashes, source labels, Python
version, fixed endpoint and all output hashes. Use the campaign environment from
contract.md; no builds/tests here have yet been executed. Build.sh uses a
throwaway local replacement modfile; dependencies remain pinned and portable.

## Required negative proof

`verify_tools.py` consumes successful completed campaign output. It runs an
unchanged positive comparator control, one-sided report mutants for original ID,
key, Prepared digest, result digest, response identity, handler/adapter mismatch,
missing/duplicate contact, RunReport root/charge/pending/unknown, epoch, approval
predecessor, selected observation, missing returned report and missing/reordered
prefixes. Matching malformed journals/prefixes/value-only reports must also
reject. SQL mutations independently change scoped grants, debit descriptors and
epochs in disposable copies. Four unchanged clone guards and four new interface
refusals verify owner/WAL/executed/existing safety and no ephemeral endpoint,
nonloopback endpoint, omitted endpoint or invented refusal scenario.

Run Python syntax checks, Go build/Staticcheck against selected temporary source
modfiles, both positive schedules, all mutants and clone/interface guards.
No executed qualification claim follows from this implementation. Root records
exact execution artifacts and obtains independent differential-integrity review.

## H1/H2 correction and matching-output proof

The initial independent review rejected the first comparator for accepting
matching noncanonical HTTP bytes, forged Prepared digests and wrong returned
work. Initial source/reports/RED evidence remain preserved separately. Root
reported the two initial positive schedules passed; those positives did not
resolve the rejecting-mutant failures. This correction has not yet been run.

`finite_validation.py` adds only this fixed producer's bounded contract. It
copies the reviewed Go JSON escaping primitive and frozen configuration values
from `../campaign_validation.py`; it builds Go struct fields in known order,
retains sorted fixed map keys and recomputes actual schema, Prepared and result
digests. Exact canonical request/action/response bytes, complete fixed request,
Candidate and transport policy are checked with strict scalar types. It adds
exact Steps/Dispatches/OpenRuns and initial root membership/debit/outcome checks,
History count/order, Replay namespace/count/all runs and roots, SeriesView counts
and retained observation/approval/History selection correlations. It neither
opens SQL nor interprets a generic program, authority graph or recovery policy.
Existing full public/SQL equality and its three normalization allowances remain
unchanged. No frozen transport, Go producer or production source was altered.

`prove_comparison.py` creates matching malformed reports and takes an explicit
`--comparator` argument. Run it once against preserved initial source for actual
old RED, then against corrected source for positive/rejection evidence. It
retains argv/exits/input/output hashes and rejection reasons. A rejected fixture
counts only when the comparator emits its structured failed-comparison result;
an import/setup error is not a successful negative proof. Both ordinary and
reopen reports have positive controls, including a specific no-work reopen
mutation. `test_finite_validation.py` supplies three small cross-language golden
tests using one genuine initial Go-produced contact copied to
`fixtures/go-http-contact.json`; the original report remains untouched.

Additional root-owned commands, with fresh output directories:

```sh
cd "$TOOLS"
python3 -m unittest -v test_finite_validation
python3 "$TOOLS/prove_comparison.py" "$INITIAL_CAMPAIGN" "$OUT/http-old-additional-red" \
  --comparator "$INITIAL_TOOLS/compare.py"
# Expected exit 1 for old comparator because malformed matching reports pass.
python3 "$TOOLS/prove_comparison.py" "$NEW_CAMPAIGN" "$OUT/http-matching-proof" \
  --comparator "$TOOLS/compare.py"
# Expected exit 0 only if both positives and every semantic rejection agree.
```

The new matching proof supplements the original one-sided report, typed SQL,
clone and interface proof command; all remain required. The golden contact's
source is initial `success-reopen0-candidate.json` (SHA256
`6ff3c8798584625e3036ffc47d1967d9c8a35728eda25065580511304c6efea9`),
contact ordinal 1. Its fixture hash is
`924ba85cf93c72d55e5a31670676deed3723b9f279e6a6fd8be458f6f981c3d1`.

## Bound-16 ordering regression

The initial H1/H2 correction was separately rejected after root's genuine
bound-16 positive exposed an incorrect numeric expected approval order. Public
`Continuations` uses `runLess` (lexical Subject, then Generation), so `series-10`
precedes `series-2`. The corrected validator sorts only its expected registry
using that rule. It compares the actual retained approval array unchanged;
there is no evidence normalization or weakened order check.

`campaign.py --bound 16` provides the reproducible genuine larger-bound
regression with the same two ordinary/reopen schedules. `--bound` defaults to 8;
the primary eight-position schedules remain unchanged. The controller records
the selected bound in inputs and success/failure summaries. Use a fresh output:

```sh
python3 "$TOOLS/campaign.py" --bound 16 \
  --baseline-binary "$OUT/base-http-differential" \
  --candidate-binary "$OUT/candidate-http-differential" \
  --baseline-source "$BASE_ID" --candidate-source "$CANDIDATE_ID" \
  --output "$OUT/http-bound16"
```

Root owns execution and re-review; no run is claimed by this source correction.
