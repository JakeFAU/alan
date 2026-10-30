# Phase 1N Task 3b independent qualification review r1

**Specification: APPROVE. Quality: APPROVE.**

Reviewer: `/root/n_qualification_review`. Review date: 2026-10-30.
This is the Task3b qualification gate against approved Task3a base
`c3c5b81938d177d73bca28532c3f8886ad02211b`. No blocking specification or quality
defect was found. Approval covers the finite model/local/lifting qualifications,
static artifact correspondence and declared operational-bound argument below.
It does not approve Task4 runtime evidence, raw retained-history authentication,
manuscript reconciliation, or a universal weak-refinement theorem.

## Inspected identities and scope

The task brief, binding constraints, implementation report, complete scoped diff
and `reference-r1.md` were read. The diff was read once, in contiguous portions;
no hunk was truncated mid-function. No git command or broad branch review was
performed. Task4's concurrent implementation and evidence were excluded.

| Input | SHA-256 |
| --- | --- |
| `.superpowers/sdd/00-plan-phase1n/task-3b-diff.txt` | `39b8f6f52829a82347850407717a81f33bda678686ce2cdb851acf822adf05ff` |
| `.superpowers/sdd/00-plan-phase1n/task-3b-report.md` | `6d89366d36b61fe45a69a734a3872995a6cde01e0c8e232013f35c80475b1abf` |
| `evidence/model/qualification/index.json` | `0487869a81edac47283086492da876ad921567660819453290672f70ceca7f7b` |
| `evidence/model/reference-r1.json` | `4603206f1eccf3bb471ca1df821aa56c613633d98dfdd9d84016786db137023e` |
| `research/substitution/callable_join_qualification_test.go` | `646ceba875c36ee4238bf641c6c2729b18ea6105ee3619b8ee609e84dc0c273b` |
| `research/substitution/callable_join_artifact_test.go` | `022d600520ce0adfd835f8e911e12f224fe89bf1ed74333873ae5c87a05d7040` |
| `qualification/check_correspondence.py` | `1fec84e9a96ce7c8691c8705d5ea9a86aa702b9c9e664fa09e50bb06a4c802ac` |
| `qualification/audit.py` | `8105d8402188e2cdb726c5a445d0430f8ef7f9788c2e5268902b6dd3e00c9382` |
| `qualification/capture.py` | `cafb58d697703484dadc557da2e966d62ded48d430bfd735b154e114cdc21545` |
| `qualification/audit-r2/audit.json.gz` | `6a61c530d59abe27582f76f0a9b51e840894a75be2820aa1f1388730cb8392f5` |
| `qualification/normal-correspondence-r2/correspondence.json.gz` | `0bc86332b9df7705e84e189accdf7bb62e36ad54ebec246af29c9f2118e0dbc6` |
| `qualification/race-correspondence-r2/correspondence.json.gz` | `23f0088e5742c3dfdf0b446753377945aa183010fe1197c7ae38535ca7fbfd92` |

Here `qualification/` abbreviates
`docs/foundational/phase1n/evidence/model/qualification/`; other evidence paths
are relative to `docs/foundational/phase1n/`. The implementation report's final
identity was checked again immediately before this review was written.

Independent read-only Python checks verified all 70 diff entries' lengths/hashes,
all 67 indexed files, all eight authored-source entries, and all 66 frozen
reference source/input hashes. For all 21 command records, archive member names
and bytes matched the recorded source map, compressed/raw result hashes matched,
and recorded exit, stderr, changes-during-run and test events were inspected.
The current audit contains 20 earlier command records; its own capture is the
21st, not an omitted behavioral run. All 25 current correspondence source hashes
also match. These checks establish identity, not independent correctness of every
historical input.

## Specification checks and strengths

**QUAL-1 — Required ordering, discovery and actual execution: satisfactory.**
`callable_join_qualification_test.go:8-126` qualifies the declared configurations
without hard-coding state, edge or maximum counts. The archived exact anchored
selectors and actual leaf events show, in each normal/race profile: reference
16, aligned local 32, replacements 32, finite lifting 32, then artifact extraction
2. Recorded timestamps confirm K1-first ordering. All 228 leaves passed, none
failed or skipped; successful commands have empty stderr. The focused vet record
has exit 0 and empty stderr. Initial/final discovery records identify the intended
entrypoints. The eight earlier model qualifier snapshots become exactly the
current qualifier bytes under read-only `gofmt`; later artifact-test source is
not falsely attributed to those runs (`qualification/audit.py:1`,
`qualification/README.md:1`, respective `command.json.gz` and source archives).

**QUAL-2 — Product maxima, progress and relation are distinct: satisfactory.**
The archived result arrays contain exactly the id {1,2}, value {0,1}, four-mode
domain: 16 configurations each for K1/A/B, 48 total. Recomputed product totals
are K1 6,376 states/13,232 edges, A 2,864/5,622, B 6,644/13,844. All 64 maxima
and their traces are present: K1 24 terminal/8 pending, A and B each 12/4.
All recorded refusal, stuck, cutoff, cycle, invariant and property failure fields
are zero. Normal/race full result bytes agree. These are finite results, not a
universal conclusion from the counts.

The named risk was accepting a pending prefix or deduplicating away monitor
history. Focused inspection of frozen `callable_join_explorer.go:29-37,55-66,99-174`
confirms the model-plus-monitor key, actual zero-successor maxima and edge checks
before deduplication. `callable_join_oracle.go:334-430` independently separates
K_join and public G_outer predicates. Their projections are not counted as two
censuses. Inspected pending states retain original candidate request consumption,
completed/consumed chosen helper, actual original S consumption, J at original
e await and O at original J await; diagnosis, report, End and publication remain
absent. There is no release or fabricated Pending response.

The second named risk was treating safety simulation as all-mode progress.
Frozen `callable_join_local.go:104-165,181-219` checks the physical depth2/3,
residual I/R3 environment and a separate open progress graph. Its all-mode
maximum predicate does not reuse the legacy answer-only `Entitled` count.
Archived local relation totals are A 884/B 2,320 pairs, with separate progress
A 376 states/544 edges and B 824/1,244; each has 12 terminal/4 pending maxima.
Frozen `callable_join_relation.go:22-120` checks candidate event matching/private
stutters and bilateral environment readiness. The finite lifted results have
A 2,864/B 6,644 pairs, 16 initial pairs each, zero failures/counterexamples.
Matching full O control supplies the corollary; erasing its private fields gives
the constructor relation. No generic weak-path normalization is checked by this
finite algorithm.

**QUAL-3 — Schema eligibility and independent finite policy: satisfactory.**
`callable_join_artifact_test.go:1-195` extracts actual retained artifacts and uses
only the original Receive for schema eligibility; the independent Python
evaluator supplies policy judgments (`qualification/check_correspondence.py:1`).
The named risk was defaulting absent payload fields or counting schema rejection
as a defensive policy branch. Focused checks of
`internal/executor/invocation_payload.go:9-52` and
`internal/executor/program.go:389-415` support the recorded eligibility and
omission rules. Candidate/S output schemas agree in the actual archived inputs.

Each binding has 140 inner envelopes, 136 accepted, and 1,703 outer envelopes,
1,696 accepted. Inspected rejection records identify four successful-absent
envelopes and, for O, unknown tag, two rows and missing answers. The actual finite
schema-valid adverse domain includes inconsistent status/reason/presence and
wrong ids/values; it is not limited to deployed outputs. The policy tables match
the frozen contract's J and O tables (`01-contract.md:97-106,138-155`). Per binding,
the archived evaluator checks 73,984 J and 6,784 O cases. Complete witnesses cover
22 J paths and 54 O paths, including cursor snapshots/exhaustion. Recomputed
witness-PC unions match the reported visited-PC sets. Missing data is never
silently made zero. This is finite policy evidence, not a proof for arbitrary
strings, schemas or compiler inputs.

**QUAL-4 — Complete instruction mapping and atomic groups: satisfactory.**
The correspondence record contains all 283 instruction records: both J58/O64
bindings plus original A13/B16/helper7/S3. The checker compares complete code,
operands, branch targets, setters, Send/End, metadata and collections; checks the
six exact binding differences; and checks all 16 input builds per binding
(`qualification/check_correspondence.py:1`; archived `instruction_records`,
`difference_paths` and `bindings`). Original component byte identities and the
shared historical mapping are explicit. Unreachable policy setters remain in
the static map rather than disappearing from coverage.

The named risk was equating ghost decisions or cursor transitions with durable
runtime commits. Classification Set maps to classify plus a ghost decision fixed
by the immutable tag and the checked later branch; no intermediate checkpoint
is claimed. Invoke/handle/intent, Receive/selection/binding/consumption and
End/freeze groups are explicit. Focused comparison with
`internal/executor/collections.go:168-207` confirms the evaluator's snapshot,
advance, exhaustion and loop-back behavior. Frozen
`callable_join.go:178-209` retains Send/End before first reply authorization;
the qualification does not add a source-open guard to remove that schedule.
Exact original binding authentication remains an explicit Task4 obligation.

**QUAL-5 — Operational bounds have a concrete derivation: satisfactory.**
The named risk was confusing model rank with runtime budgets or retaining the
historical 200/200 metadata. The current checker derives local commit bounds
A139/B149 from the finite code, including both sides of mutually exclusive
branches, and one additional O `next` for exhaustion. The single-row schema
bounds the only loop. The dispatch bound A9/B12 charges admission/request,
optional reply delivery and ready settlement/delivery per original singleton.
Focused checks of `internal/executor/runner.go:152-160,222-239,268-295,378-440`
support that accounting under the declared no-error/no-retry/no-closure profile.
`examples/callablejoin/consumer/main.go:84` has the asserted actual 512 steps,
64 dispatches and capacity1; both bounds fit.

Default byte caps are 1 MiB (`internal/executor/program.go:95-100`). The recorded
largest retained program is 25,387 bytes, maximum bound-program allowance
416,142, projection/observation/single-intent allowance 55,616, finite payload
571 and original handle allowance 76. The finite register counts, forward
control, one intent per Emit and fixed identity assumptions make these
conservative bounds intelligible. Focused checks of
`internal/executor/program.go:66-89,437-469` and
`internal/executor/composition_binding.go:246-339` support the serialized-field
and binding-substitution allowances. They do not cover arbitrary values,
additional adapters, retries, closure, recovery or arbitrary candidate code.

**QUAL-6 — Correction history and argument status: satisfactory.**
Archived source comparisons confirm that `normal-correspondence-1` failed on
the new checker's incorrect unreachable-PC expectation `[27,28,49,50]`; the
correction is `[27,49]` plus its explanatory text. The shared jumps are reachable.
The failed argv, source archive and traceback remain. The later metadata fix
changes 200/200 to actual 512/64 and adds a source assertion and fit check;
superseded successful captures and `audit-final` remain identifiable. Neither
correction changes the frozen model, policy tables or public construction
(`qualification/index.json:1`, `qualification/README.md:1`, archived checker
sources and current r2 records).

`qualification/README.md:9-37` accurately carries the independent
`reference-r1.md` disposition: ARG-1 is a restricted conditional constructor
argument with supplied eventwise/guard-stable R, bilateral environment readiness,
caller-owned delayed selection, equal authorization sidecars, resource/depth/
future-demand alignment and separate progress/nondivergence. The finite A/B
checks instantiate those stronger premises. ARG-2 remains open: unchanged
arbitrary weak K1 refinement has no established normalization/extension bridge
to them. The constructor relation and transfer cases in frozen
`02-lifting.md:110-190` are not silently rewritten after qualification.

## Quality assessment and remaining obligations

No Critical, Important or Minor issue requiring a change was identified.
The two Go files have distinct responsibilities; qualification failures remain
test failures, and the extractor does not double as the policy oracle. The
research scripts preserve exact source-bound evidence and failed captures.
Historical component mapping and shared semantics remain disclosed independence
limitations. Exact instruction expectations are appropriate to this frozen
correspondence task; they are not presented as a general compiler validation.

The reviewer ran read-only Python hash/archive/event/result/correction checks
and read-only `gofmt` on eight archived byte streams. No Go tests, race tests,
vet command, Python correspondence run or model search were rerun. The successful
test/census/policy results above are inspected author-run evidence. No new
behavioral probe was necessary. No code, index, branch, frozen source, runtime
evidence or manuscript was changed; this report is the sole written file.

Still outside this approval: actual runtime execution and all-history auditing,
raw source/generation/revision/caller/route/schema/binding authentication, actual
post-End first authorization, manuscript synchronization, ARG-2, arbitrary-context
congruence and recovery. Task3b is complete within its declared finite/static
scope; none of these remaining obligations is implicitly discharged by counts.
