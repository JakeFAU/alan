# Delegated Weather qualification record

The [qualification narrative](../../../examples/outing-planner/QUALIFICATION.md)
records the actual process cuts, commands, results, corrections and limits.
The [controller decision record](DECISIONS.md) preserves scope/rework rulings;
it is not an independent evidence verdict.

The controlled [archive](qualification.tar.xz) contains 9,624 hashed
source/evidence files and 884 distinct independent replay jobs.
Its size is 13,403,752 bytes and SHA-256 is
`d56e614c9f2134d6f403216a5d7ed8053652add0084872c8a8cc0a3563c95aab`. [manifest.json](manifest.json) binds the archive, selected
409-file source map and companion records. It does not claim to contain its own
final commit hash. Executables, caches, live Stores and unrelated scratch are
excluded; complete certified raw Stores and explicit negative mutations remain.

Extract into a fresh directory and execute the independent checks:

```sh
work=$(mktemp -d)
tar -xf docs/qualification/delegated-weather/qualification.tar.xz -C "$work"
python3 -I -B "$work/delegated-weather/reproduce.py" --output /tmp/delegated-reproduction
```

The output directory must be new and outside the extracted bundle. Existing
paths and paths resolving into archived inputs are rejected. Reproduction
verifies all archived input hashes before and after, runs each job against its
actual archived checker sources, then runs 24 model methods, all 14 fixture-backed
evidence methods and finite enumeration. The observed reproduction matched all
expected outcomes, including separately labeled historical wrong acceptance,
required rejections, superseded captures and diagnostic trials. Those categories
are not counted as final passing captures. The archive contains no missing replay
inputs. Interpreter/stdlib/SQLite, source/capture integrity and the other declared
trusted boundaries remain explicit.

[reproduction.json](reproduction.json) records post-build chronology: a premature
startup failed before opening the script; the later full replay passed; normalized
repacking reproduced exactly the archive SHA-256 above. The [full replay summary](reproduction-summary.json.gz)
is a separately hashed compressed companion. These post-build records were not
inside the already-hashed archive. The inner README describes source-set history,
legacy/seed capture provenance and optional diagnostic reenactment.
