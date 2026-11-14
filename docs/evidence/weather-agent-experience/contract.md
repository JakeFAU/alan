# Weather agent experience contract

Baseline: `42122cd69742d8a231d24cd497a95f6952163e27` (merged PR #18),
tree `1f8f8fa27838165b93756bbeb31811b0ea99821f`. Frozen before implementation.

Return to the ordinary agent experience: author a weather request, let ALAN
retain its execution, and understand the result after restarting the process.
The public runtime already supplies the necessary execution and recovery
mechanics. This slice improves the example and its entry-point documentation.

Two exploratory trials on the baseline found concrete friction. A separate
197-line public-package application completed a single reading, preserved it
on fresh-process reopen, and left a lost response unresolved without another
GET. An operator trial ran 42 CLI invocations with five loopback GETs, including
an acknowledged SIGKILL. Bare `inspect` and `inspect --json` produced identical
10,886-byte JSON for one completed reading in a three-position deployment.
The human report described both the next eligible read and later ineligible
reads as awaiting approval. Premature approval returned only `ineligible
observation`. These are bounded observations, not an API or performance study.

## Decision

Add `weather-agent status --db FILE`: a human, read-only view of the existing
validated snapshot, with original operation identity, retained response and
selection, uncertainty, expenditure, and concrete next-action guidance.
`status --json` uses the existing detailed inspection report. Preserve bare
`inspect` and `inspect --json`, their data fields, and existing exit behavior.
Correct misleading human readiness text and contextualize approval errors.

A short authoring walkthrough will show the actual request/await program,
finite provisioning, trusted adapter setup, Create/Open/RunProgress, and
typed inspection. It will use the existing runnable weather example. The
README will lead with that path and link to the preserved performance record.

Documentation alone would leave the misleading status and opaque error.
A second executable would duplicate host policy and compatibility code.
A runtime helper is not justified: both trials worked through public APIs.

## Boundaries

- No runtime, author, adapter, schema, dependency, or public Go API changes.
- No new execution, approval, retries, authority, or provider contacts from
  status. It must not open an owner, renew epochs, or settle retained work.
- Use existing validated identities and approval-policy functions. A paid
  predecessor alone does not make a next read approvable when the weather
  policy still requires local result processing. A suggestion is not a grant.
- Preserve original results, approvals, prices remaining unknown, independent
  expenditure, historical weather deployments, and the archived live Store.
- A locally admitted request without a response remains uncertain; a completed
  successor does not settle it. Local contact bounds are not endpoint logs.
- Commands containing a Store path must quote it safely for a POSIX shell.
- No live provider traffic during qualification. Controlled loopback and the
  existing offline archive suffice.

## Acceptance

Test provisioning, success, refusal, paid but undispatched, admitted unknown,
retained but unfinished results, explicit and duplicate approvals, skipped
approval, exhausted allowance, historical deployments, corrupt evidence, and
one acknowledged ordinary-executable SIGKILL followed by fresh reopen.
Status must make zero additional contacts and preserve database/WAL bytes
(absent and empty WAL are equivalent; SQLite SHM is outside that guarantee).
Keep detailed JSON unchanged for matched retained cuts. Check useful guidance,
not a duplicate implementation expressed as a test.

Run the weather tests, relevant process and race qualification, repository
ordinary/integration suites, build, vet, module verification, formatting and
Staticcheck from recorded source revisions. Preserve failures and limitations.
Obtain independent task and whole-branch review, then deliver a draft PR.
Do not merge or release. Stop scope growth if presentation requires changing
the execution protocol; report the concrete gap instead.
