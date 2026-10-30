# Task 1 independent review — round 1

Reviewer: /root/m1_consumer_review; separate from the consumer and model authors.
Scope: Task 1 authoring/public construction preflight against assignment sections
1–3, the plan Global Constraints and Task 1 brief. Base:
`f5b52853ce3cf2b93563eed5eb8268234386e691`.

**Specification verdict: APPROVE for Task 1.**

**Code-quality verdict: APPROVE.**

No actionable findings. These verdicts accept the construction/preflight slice;
they do not accept a reference premise, substitution result or milestone-wide
qualification. Review agreement is not proof of the semantics.

## Question and inspected boundary

Does the new ordinary owning parent commit original candidate e then original S,
consume S then e, and decide only after both consumptions while preserving the
strict guarded result policy and common resource entitlement?

I read the supplied task-1 brief, report and full delta, the assignment and plan,
applicable AGENTS instructions, all new Go/README/module files, capture code and
archived records, original A/B/helper/S/control constructors, public author
invocation APIs, historical parent construction and the relevant historical
contracts. The original parent was consulted as reusable context, not as an
independent semantic oracle. No new model sources or outputs were used.

## Specification and implementation checks

- `examples/consumingpolicy/deployment.go:29–67` exposes the requested API and
  retained deployment values. Historical provisions are used only to obtain
  public original templates. The executable provision is rebuilt with exactly
  one fixed `root`; the resource root covers that owner and the original
  catalogue closure. Neither historical provision is passed to a store.
- `deployment.go:90–93` commits e, then S with original request id/value, before
  the S and candidate awaits. Both awaits bind the original retained handle
  registers and preserve status/reason/presence. No error branch precedes the
  second await. There is no extra report peer, wrapper, retry, release request,
  effect or runtime primitive.
- `deployment.go:72–80,94–97` checks candidate and S status/presence before their
  payloads, preserves exact correlation, computes the Boolean complement from
  the original input and requires S zero. Guards apply after both consumptions;
  the original Phase 1L strict result predicate is preserved. Reading the total
  request input for `expected_value` does not read an absent response.
- A/B use the same builder and policy. Captured parent JSON differs at exactly
  candidate invoke target, candidate receive source and author template metadata.
  B's catalogue includes its unchanged helper. A/B parent artifact identities
  are honestly reported as different. The API allows configured limits; the
  preflight CLI and captured executions supply common I=R3/depth2/E0/fixed1 and
  concrete runner capacity1.
- The separate consumer module pins Go1.27.3 and a local root-module replacement.
  Its imports are public. Store/CLI mechanics are attributed to the old consumer;
  new program structure is explicit rather than a renamed historical artifact.
  Authorship claims are appropriately limited: the author inspected the old
  parent and reused compiler/runtime, templates and infrastructure. Separate
  author identity does not imply independent reinvention of those components.

The constructor tests check the bound instruction prefix, original template bytes,
fixed ownership, common limits and candidate-only differences. Public integration
controls check both-side failure/absence, wrong-id and dual pending behavior.
The candidate-failure helper has a narrower identity check than a full audited raw
adapter; the report correctly reserves that adapter audit for the later task.
This is not a missing Task 1 gate.

## Execution, identity and resource evidence

Evidence reviewed: `evidence/preflight/expectations.md`, `capture.py`,
`evidence-index.json`, `records.json.gz`, `construction-red.txt`, preserved
`failed-metrics/` and `capture-completion.txt` relative to this phase directory.

Readonly Python archive inspection independently checked the current 108-file
source manifest, its digest, archive digest, original-source/base equality,
artifact-byte SHA256 identities, command exits/test-event leaves, original
intent/admission/request/binding/publication/delivery/observation/selected-history
links and strictly forward compiled control-flow bounds. No stores or programs
were mutated. The manifest is
`ce93fdffdf390ad7356aa42bd6a23d964f36f8c9fb8668437a7d824965fc5763`;
archive SHA256 is
`e389dda446d2cf76ede45ecc7b5432220383de7d932f925bd714e047bd67414c`.
All current manifest entries match; all protected source entries equal the base.
All seven retained program blobs across A/B have IDs matching their actual bytes.

For each A/B capture, e commits at parent sequence1 and S at2; the original S
observation is consumed at3 and the original e observation at4. Each selected
observation is selected by exactly one authentic history row. Operation,
invocation/template/caller identities agree across admissions, bindings and
publications; delivery IDs identify the actual selected observations. Request
rows retain id1/value0, including original S input. Envelopes retain
succeeded/member_succeeded/present and candidate id1/value1, S id1/value0. Parent
and root succeed with no unresolved intents, pending deliveries or open runs.

| Binding | Dynamic I/R | Effects | Actual steps/dispatches | Forward path upper | Dispatch upper |
| --- | ---: | ---: | ---: | ---: | ---: |
| A + S | 2/2 | 0 | 25/6 | 25 | 6 |
| B + helper + S | 3/3 | 0 | 32/9 | 35 | 9 |

I independently recomputed per-program path bounds: parent16/A6/S3 and
parent16/B10/helper6/S3. Each nonterminal target is strictly forward. The complete
catalogue contains one instance of each relevant singleton program; summing is
valid for this profile. Dispatch counts correspond to one invocation, request and
reply dispatch per accepted singleton under stable eligible service. These are
finite pure-profile bounds, not arbitrary-template or transient-failure bounds.
Captured byte maxima and enforced DefaultLimits provide adequate margin below
the four 1MiB caps; the reported projection/observation figures remain sample
measurements rather than a general compiler byte theorem.

The archive records 25 commands, all exit0. Its JSON test records contain four
builder leaves and thirteen consumer leaves in each normal/race gate, with no
fail/skip events. The environment, source paths, exact argv and temporary paths
are retained. I inspected these prior task executions; I did not rerun their
passing Go suites. Review probes were readonly `python3` heredocs and source/git
inspection. One initial archive-inspection probe incorrectly assumed a publication
column named `outcome` and exited with KeyError; corrected inspection used the
actual `descriptor` column and passed. This reviewer probe failure is not a
consumer defect or a substituted execution result.

The initial scaffold RED and later metrics KeyError are preserved. The final zsh
wrapper failure is also explicitly preserved and distinguished from the successful
Python capture and its enclosed commands. This reporting is truthful.

## Remaining obligations and review scope

The authored expectation file states its pre-execution provenance; its current
hash is bound into the capture. File existence/hashes alone do not independently
prove the historical write ordering, so this review accepts the recorded workflow
provenance rather than asserting a cryptographic temporal proof.

Task 1 appropriately leaves the independently frozen reference, K1-first census,
full four-input qualification, postfreeze artifact/model correspondence, both
public-driver publication orders, positive intermediate-state witness, shared raw
adapter audit and manuscript reconciliation to the assigned later tasks/controller.
The pending controls do not substitute for the positive intermediate witness.
No crash/recovery theorem, arbitrary-context congruence, equal-debit result,
C_reason classification or phase-wide paper consistency is accepted here.

Only this review report was written. No subagents, commits, suite reruns or other
repository changes were made by this reviewer.
