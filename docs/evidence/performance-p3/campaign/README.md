# Strict direct campaign checkpoint

These tools adapt the unchanged P2 vector/cut validator for P3's actual direct
adapter. The P2 driver and original HTTP tools remain unchanged. No diagnostic
acceptance path exists in this runner or analyzer. `freeze.py` exports baseline
926a899b and candidate source 4f5fe2cc, builds the same original driver bytes with
retained local module replacements, and binds the exact 98 ordered design rows,
absolute commands/cwd, whitelisted settings (including GOSUMDB=off), source,
SQLite module, tool, interpreter and executable hashes. A source snapshot covers
all exported files, so new/deleted/modified files invalidate custody. Before/after
checks happen outside the child's timing. Source-only 32-run shared/distinct
controls are bound through the existing selector probe and candidate test source.

The raw report's outer `resource_scope` string still mentions a same-process HTTP
server because the frozen driver emits that string for both modes. This validator
requires that unchanged raw string, while separately requiring the complete direct
transport configuration (`measurement/v1`, `deterministic-local-no-http`,
`process-no-server`, all HTTP settings zero/false), zero server journal entries,
zero-valued HTTP fields in adapter contacts, the full retained Prepared digest,
fixed request schemas/payload, scalar contact identities, actual direct Candidate,
retained result digest, original runs/selections/approvals and prefix arithmetic.
Direct adapter entries are never called HTTP contacts. No report is rewritten.

`run.py BINDING_JSON REVIEWED_BINDING_SHA256` is the eventual campaign command;
it is **not authorized at this checkpoint**. The controller must obtain an
independent APPROVE against the exact committed tools and frozen binding first.
The runner creates the campaign output exclusively, launches serially, saves
start and child-outcome records before post-exit reads, and stops on failure.
It refuses existing outputs. There is no resume/overwrite/until-green option.
`analyze.py BINDING_JSON OUTPUT_JSON` requires every warmup and measured outcome,
its original bytes, correct process order, all proof checks and unchanged custody.
The digest argument is a custody check; independent approval remains a controller
responsibility rather than a self-issued machine assertion.

The primary greater-than-5% rule is `1 - median(candidate Execute) /
median(baseline Execute) > 0.05`, implemented with exact nanosecond-median
cross-multiplication at the boundary. Median paired ratios and every difference
are separately reported, and all six B256 differences must be positive. Guards
cover B1/B8/B16/B32 and every separate non-Execute phase, including summed
approval snapshot/basis and approval establishment, using max(10%,1ms). The
summed Run phase is additionally reported/guarded. RSS above max(10%,8MiB)
requires explicit resolution. Timing and allocation flags do not assert overall
acceptance: independent mechanism/semantic/recovery qualifications remain needed.

Tests require explicit paths so historical archives never silently qualify P3:

```sh
P3_TOOLING_BINDING=/absolute/binding.json \
P3_TOOLING_FIXTURE=/absolute/tooling-fixture-b8-B.json \
P3_TOOLING_FIXTURE_PLAN=/absolute/tooling-fixture-plan.json \
PYTHONDONTWRITEBYTECODE=1 python3 test_campaign.py
```

Small fresh B1/B8 fixtures are labelled `tooling-fixture-*`, outside the 98-row
schedule, and never enter timing analysis. Tests also mutate those reports and
metadata, check schedule/custody/phase/guard failures, and use a tiny exit-23 child
with an injected post-exit metadata failure. These are tooling qualifications,
not confirmatory samples or performance estimates.
