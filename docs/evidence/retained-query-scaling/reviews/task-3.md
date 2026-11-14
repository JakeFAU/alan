# P7 Task 3 independent review

Reviewed 2026-11-14 against the negative-result Task 3 brief. Base
`c19785d5ca2f3a6cc2ea4efe3ca539c76a0a6b80`; head
`c1c475cb81c59b5388e83b8960ce1029f3159613`, tree
`5ef2d7b151a0cc6fb478376614d9d9ba5dc03d71`.

## Verdicts

- **Specification: PASS.** The stopped P7 experiment delivers the frozen
  apparatus, bounded negative result, retained evidence, reproduction and scope
  record required by Task 3. Task 2 remains skipped.
- **Quality: APPROVE.** No blocking or medium-severity finding. The source bridge,
  observed results, failures and qualification boundaries agree with the artifacts
  independently checked. Whole-branch review and draft publication are separate gates.

## Finding

**Minor, non-blocking — cache location.**
`docs/evidence/retained-query-scaling/qualification.md:66` describes a
“checkout-local cache.” The successful retry command records put
`STATICCHECK_CACHE` in the temporary reproduction root as a sibling of
`checkout/`. “Reproduction-local cache” would describe the verified path
precisely. Both successful lint outcomes stand.

## Source and delivery scope

The supplied patch matches the actual Git delta with ten lines of context.
The base-to-head diff contains exactly the six paths in the delivery bridge:
root README navigation, results, qualification, the Task 1 review copy,
the reproduction record and the bridge. The baseline-to-head diff is confined
to root README and the new P7 evidence directory. The five harness inputs are
byte-identical to the corrected `ca65abb857a75f3677ce88a74cbeb94a2251450b`
commit. Runtime, old tests, public API, dependencies, schema, indexes, ownership,
retries, BoundedSeries and nested consumer sources retain the baseline bytes.
Verified both bridge links, the changed-file digest set against actual Git
blobs, and the exact Task 1 review copy against the ignored workspace original.
`git diff --check c19785d c1c475c` exited 0. Read-only validation parsed all
17 JSON/JSONL files and checked 38 relative Markdown links and fragments.

## Detached apparatus reproduction

The actual shared local clone in the temporary reproduction root was detached
and clean at corrected commit `ca65abb857a75f3677ce88a74cbeb94a2251450b`,
tree `bb51e53578fda278c7b02a08c50c6efc9b56f7d8`. Independently hashed its 829
tracked files and matched the maps in all six before/after inventory files.
Verified the four raw index records, indexed file presence and sizes, and
local clone/checkout logs. Compared all 35 outer command records with raw
commands, exit records and stdout/stderr hashes. The seven focused Python
tests passed in the retained log. The generated Go control command and
before/after bound inputs agree with the compact record, and its raw test
log reports 35 passing test events, one passing package and zero
failures/skips. Go 1.27.4 Darwin arm64, Python 3.9.6, Apple clang 21.0.0 and
stock SQLite v1.14.52 match recorded version and build outputs.

The fresh stock smoke contains all four B1/B8 arm rows. The raw row and gate
hashes, sample schedule, actual binaries, manifests and before/after inputs
agree with the reproduction record. Each execution passed invariants and
preserved the bound's result, expenditure, contact, root and evidence counts.
The smoke uses separate output roots and supplies no primary timing data.
Root build, vet and module verification exited 0. Both initial Staticcheck
runs exited 1 and emitted five default-cache write errors per tag setup.
Both retries with a writable cache exited 0 with empty diagnostic streams.
Both formatting checks exited 0 with empty output.

## Decisive results and limits

Selected fresh read-only checks corroborated the implementer's Task 3
recomputation. The raw pilot has 14 warmups and 84 measured rows. From the
raw B256 reports, baseline and overlay Execute medians were 1,290.6471465
and 1,294.1518545 ms. Median paired reduction was −0.326281254%, three of
six pairs were faster, and median savings of −4.2163545 ms were below the
17.4148125 ms MAD. AB was +1.827720% and BA was −1.131850%. Each B256 pilot
report has 256 completed and selected results, charges, direct contacts,
roots, original evidence entries and replay intents, 255 protected
approvals, and 1,023 reductions and replay history entries. The main
results table and retained resource vectors distinguish summed call time,
Execute, lifecycle, audit stages, allocation volume and cumulative peak RSS.

The raw B256 counter lines give 227,970 → 97,665 Execute full-scan steps,
164,428 SQL executions and 131,184 row returns in each arm. The removed
130,305 steps are history insertion scans. Remaining result-lookup and
charge scans sum to 65,280 + 32,385. The raw clock reports give B256
history on-mode median 5.854341 ms, empty median 0.024032 ms and zero
off-mode totals. The method and decision appropriately limit the structural
diagnosis to the tested workload and pinned SQLite access paths, with
separate diagnostic, stock timing and profile evidence. The measured
latency screen fails all three requirements.

The two ordinary baseline exports have matching archive digests and
before/after source maps, with 822 files each. The raw ordinary runs confirm
exit 1 with loopback bind errors on the first attempt, and exit 0 with 1,420
passing test events, 18 passing packages, three without tests and four
explicit optional frozen-reader skips on the loopback-enabled run. The
results and qualification distinguish original baseline recovery coverage
from skipped historical-reader, nested-consumer, same-seed recovery,
selected-write SIGKILL/weather-cut, full integration/race-integration and
focused race campaigns. Local raw data is disclosed as unpublished. Early
controller/packaging failures with partial custody are distinguished from
retained terminal failure records. The recorded non-overlap for campaign
work carries the stated limitation regarding unrelated background work.

I checked existing command outputs and selectively recalculated the
decisive metrics. I did not repeat Task 1's full custody audit, rerun suites
or build/benchmark anything. This report is the only review write.
