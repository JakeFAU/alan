# Structural scaling qualification apparatus

The [structural correction contract](../structural-contract.md) governs these
routes. The integrated baseline is `ef38ca47179d86e7544885471ae43e86e1d6351e`.
The reviewed executable source is `102c62062c61985bcbe5ea8c8c9d12fcf385891a`.
The campaign can export a descendant that includes this apparatus after the
executable source bridge passes. Its actual commit and tree identify the export.

Use an Apple M4 host with the bound Go 1.27.4 darwin/arm64 toolchain, Python
3.9.6, Staticcheck 2026.2.2, native Clang and SDK, and the already downloaded
go-sqlite3 v1.14.52 module. Use a new persistent evidence root outside every
frozen source tree. Local loopback access is required for the real HTTP and
process-cut fixtures. Keep the host otherwise idle while running timing. The
source freeze records a sanitized hardware identity, OS, native toolchain,
whitelisted environment and complete source/dependency inventory. Source
archives, standalone Git metadata, builds, caches, gates and captures are
siblings under the new root. Build inputs use disposable overlays.

## Freeze and run

Run these from the repository root after apparatus review. Replace the two
example paths and use the exact reviewed apparatus commit for `candidate`.
Retain the three printed digests externally and compare them when resuming.

```sh
python=/usr/bin/python3
task5=docs/evidence/execution-scaling/task5/controller.py
root=/absolute/fresh/persistent/task5/run-id
candidate=$(git rev-parse HEAD)
baseline=ef38ca47179d86e7544885471ae43e86e1d6351e

"$python" -B "$task5" freeze --root "$root" --baseline "$baseline" --candidate "$candidate"
source_digest=$(shasum -a 256 "$root/source-freeze.json" | cut -d' ' -f1)
"$python" -B "$task5" build-timing --root "$root" --source-digest "$source_digest"
timing_digest=$(shasum -a 256 "$root/timing-freeze.json" | cut -d' ' -f1)
"$python" -B "$task5" run-timing --root "$root" --source-digest "$source_digest" --build-digest "$timing_digest"
"$python" -B "$task5" analyze-timing --root "$root" --source-digest "$source_digest" --build-digest "$timing_digest"

"$python" -B "$task5" build-diagnostics --root "$root" --source-digest "$source_digest"
diagnostics_digest=$(shasum -a 256 "$root/diagnostics-freeze.json" | cut -d' ' -f1)
"$python" -B "$task5" run-diagnostics --root "$root" --source-digest "$source_digest" --build-digest "$diagnostics_digest"
"$python" -B "$task5" analyze-diagnostics --root "$root" --source-digest "$source_digest" --build-digest "$diagnostics_digest"

"$python" -B "$task5" build-qualification --root "$root" --source-digest "$source_digest"
qualification_digest=$(shasum -a 256 "$root/qualification-freeze.json" | cut -d' ' -f1)
"$python" -B "$task5" qualification --root "$root" --source-digest "$source_digest" --build-digest "$qualification_digest"
"$python" -B "$task5" differential --root "$root" --source-digest "$source_digest" --build-digest "$qualification_digest"
"$python" -B "$task5" recovery --root "$root" --source-digest "$source_digest" --build-digest "$qualification_digest"
"$python" -B "$task5" reproduce --root "$root" --source-digest "$source_digest" --outside /private/tmp/alan-task5-reproduction-run-id
```

Each attempted gate has an exclusive directory. The recorder retains exact
commands, logs, child status, before/after identities and test events.
Qualification writes a failure index even when a required test or static gate
fails. A timing or diagnostic row failure stays in its assigned position, and
analysis requires all original rows to validate. Use a new labeled evidence
root for any separately authorized repeat. The outside reproduction mirror
contains the exact separately rebuilt files, its original paths and binary hash.

## Timing and counter interpretation

There are 21 excluded warmups and 168 measured rows. Each comparison has six
adjacent pairs at each of seven bounds in AB/BA/AB/BA/AB/BA order. Candidate
snapshot is measured independently in each comparison. All timing rows use
direct transport, no interval reopening, no profiling, and no work counters.
The unchanged driver obtains and verifies every full public RunReport inside
the measured execution window and retains its prefix proofs. Exact differential
routes separately retain and compare all actual RunReport fields.

Use `execute_window` for elapsed, process CPU, cumulative allocation bytes
and counts, and GC deltas for the complete run/approval window. The process
high-water RSS is a cumulative gauge in bytes on this host. The nested
`execute_sum` accounts for ordinary Run, basis and approval stages, including
the actual targeted basis stage when selected. Keep construct, Create,
initial/final audits, completed reopen and Replay separate. Profile captures
are whole-process CPU and allocation-space data. Their stage labels may
support an explicitly filtered follow-up.

The augmented baseline and candidate Go probes count reached work at exact
anchors. The profiles document source-specific names in `work-*.json`.
`work.report.*` uses finishRun scope 8 where work could also run during
inspection. Scalar per-root charge queries and grouped charge rows have
different units. Use the matching SQL family's FULLSCAN_STEP and VM_STEP
values for scanned charge work. Use raw stage counter deltas to distinguish
approval snapshot validation from approval transaction work and from the
candidate's predecessor-range and point reads. Keep total SQL executions,
returned rows, VM steps and full-scan steps distinct. The same-source B1/B8
calibration compares the original and compact vocabulary on the work and SQL
counters both actually record. New work counters are supported by probe
placement and separate controls. An absent path-specific key means the probe
did not run or is absent from that source profile, as specified by the
declared vocabulary.

## Apparatus controls

An excluded development freeze uses `freeze --excluded` with the reviewed
Task 4 commit. It cannot run confirmatory commands. With that freeze, run
`smoke` and `build-qualification`, followed by the three optional excluded
controls `excluded-differential-smoke`, `excluded-historical-smoke` and
`excluded-weather-smoke` with the qualification build digest. Run the strict
report mutation tests against its raw samples.

```sh
ALAN_TASK5_SMOKE_ROOT="$root" /usr/bin/python3 -B -m unittest discover -s docs/evidence/execution-scaling/task5 -v
```
