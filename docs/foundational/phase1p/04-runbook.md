# Phase 1P — Reproducing the qualification

Run commands from the repository root. The integration base is `d0def139632c73936a3e821d93271a62267fa46d`. The [baseline](evidence/baseline.json) records original tree and blob identities, PR26 ancestry, exact local Go version, CGO, platform and offline/readonly module settings. [The second fetch and protected-input comparison](evidence/before-paper.json) were completed before paper edits.

## Historical reference and public policy

```sh
python3 docs/foundational/phase1p/evidence/check_reference.py
```

This reads two compressed Phase 1N result payloads and their command manifests directly from the immutable integration commit. It verifies compressed and raw hashes, the reference domain and recorded failure counters, then reports the policy correspondence coverage for each binding. The [captured output](evidence/reference-audit.json) is an execution of this record inspection in Phase 1P. The model enumeration and policy evaluations described in that output were executed in Phase 1N. The [qualification](03-qualification.md) separates those results from this phase's analytic challenges.

## Source preservation and manuscript alignment

```sh
python3 docs/foundational/phase1p/evidence/verify_sources.py HEAD
```

Use the Phase 1P delivery commit in place of HEAD when reproducing from a later checkout. The command reads that commit directly. It checks the change set against the integration base and compares the full inherited manuscripts byte for byte after removing the new Phase 1P paragraphs. It compares those paragraphs across formats after normalizing just the heading and file-path markup. The pre-build [source record](evidence/source-audit.json) ran the same checker with WORKTREE to bind the proposed manuscript files to their reviewed bytes.

The proof, guard tables and analytic counterexample attempts are the new research artifacts. Qualification of those arguments used independent semantic review. No new Go or runtime code was introduced and no Go test, race run, model search or public execution was performed in this phase.

## PDF

The phase-end build uses `tectonic alan-dissertation.tex` from `paper/` after independent source review. The [delivery record](05-review.md) gives the build result, warning disposition and inspected pages. Poppler page extraction and raster inspection use the finished PDF.
