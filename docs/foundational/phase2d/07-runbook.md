# Reproduce the frozen gates

The executable release snapshot is `8ad4b3cffb5185ec236f152c87fc7b9ce3676c36`, tree `14bb82f82eeabc1f4ce6d640aa9c736ce1da1a5a`. The detached qualification worktree was `/private/tmp/alan-phase2d-qualification`. The editable branch, manuscript and evidence assembly used a different path. Reproduction needs an empty destination outside the checkout, loopback sockets, SQLite/CGO, the local Go toolchain and its already cached modules. Our Go 1.27.3 darwin/arm64 environment used CGO, local toolchain selection, readonly modules, and disabled Go proxy and checksum-database network traffic. The gate manifest records the actual PATH, cache, working directory, argument vector, executable identity, exit code and output.

```sh
git worktree add --detach /private/tmp/alan-2d-reproduce 8ad4b3cffb5185ec236f152c87fc7b9ce3676c36
cd /private/tmp/alan-2d-reproduce
packet=/private/tmp/alan-2d-reproduce-output
scripts=docs/foundational/phase2d/tools
python3 -B "$scripts/batch.py" "$packet" normal dep-start
python3 -B "$scripts/batch.py" "$packet" normal broad
python3 -B "$scripts/batch.py" "$packet" normal static
for regime in normal race
do
  for group in build focus adverse prior model grid cases-a cases-b cases-c
  do
    python3 -B "$scripts/batch.py" "$packet" "$regime" "$group" || exit 1
  done
  python3 -B "$scripts/gate.py" "$packet/commands/$regime-independent-model.json.gz" -- python3 -B "$scripts/audit_model.py" "$packet/$regime-model" || exit 1
  python3 -B "$scripts/gate.py" "$packet/commands/$regime-independent-raw.json.gz" -- python3 -B "$scripts/audit_raw.py" --process "$packet/$regime-process" --grid "$packet/$regime-grid" --consumer "$packet/$regime-bin/consumer" || exit 1
  python3 -B "$scripts/gate.py" "$packet/commands/$regime-independent-adverse.json.gz" -- python3 -B "$scripts/audit_adverse.py" "$packet/$regime-adverse" || exit 1
done
python3 -B "$scripts/batch.py" "$packet" normal dep-end
```

The standalone pre-query public construction is an additional command in both modes. Run its script with a Python whose sqlite3 connection supports serialize. The recorded invocations give the local executable.

```sh
python3 -B "$scripts/gate.py" "$packet/commands/normal-construction.json.gz" -- python3 -B examples/effectfulcomposition/consumer/construction.py normal "$packet/normal-construction"
python3 -B "$scripts/gate.py" "$packet/commands/race-construction.json.gz" -- python3 -B examples/effectfulcomposition/consumer/construction.py race "$packet/race-construction"
```

Each command recorder checks the Git worktree before execution, compares HEAD and the per-file hashes afterwards and checks final Git cleanliness. The manifest includes Go, native, module, shell and Python source, including the historical checkers, and the embedded corpus. The clean committed Git tree additionally binds retained bytecode, old SQLite evidence and other package test inputs. The external dependency inventory binds the actual cached sqlite module, Go sources/compiler/tools, SDK and compiler headers, tool executables and system link stubs at both batch boundaries. Prebuilt test/process executables have distinct normal/race identities in their traces. Shared Go build-cache reuse is permitted; `-count=1` disables reuse of test results.

The ordinary broad gate is exactly `go test -p=2 -count=1 -timeout=60m ./...` at the repository module root. Sixty minutes is a per-package watchdog because the earlier adaptive test took about 25 minutes on this host. The separate public consumer has its own module. The focused command includes tagged executor and authoring checks, finite models, endpoint, both reference examples and the external consumer. The historical process selection covers Phase 2B revocation, Q, delayed application, prior Stores and Phase 2C process/authority/competing/handoff/strict-guard controls. The Series 1 selection exercises resource and readiness boundaries. Exact `-run` expressions and package lists are in the command records and batch source. Frozen-version unit, integration, process, model and archive-audit evidence are separate units.

The controlled owner is a tagged Go test binary entered with its dedicated selector and explicit environment. Directly running that helper without its environment performs no experiment. Several negative cases deliberately produce a denied operation, failed authored run, open resolver or no root publication. Verify each against its retained expected frontier. Child SIGKILL is recorded separately from the wrapper's pass/fail exit. Absence of an unrecorded network contact relies on the controlled routing and complete endpoint journal. [Driver](tools/batch.py) · [Process evidence field guide](03-experiments.md)

## Delivery checks

The read-only final inventory verifier runs in the delivery checkout, which includes the paper, evidence and reviews added after the executable freeze. Execute `python3 -B docs/foundational/phase2d/tools/delivery.py --verify`. The inventory excludes itself and the whole-phase review that signs its digest. To compare command manifests against delivery source, give the verifier the directory of *successful* frozen qualification records via `--commands DIRECTORY`. Prior attempts, if present, are preserved separately. The Phase 2C and Series 1 historical inventories must be regenerated in their respective historical checkouts. [Baseline identities and results](evidence/baseline.json)

The single final manuscript command was run from the paper directory using the unchanged Phase 2C command recorder on the reviewed source commit. The command was `tectonic alan-dissertation.tex`. The corresponding output includes engine warnings and internal TeX passes. A separate source and Poppler render review verifies the delivered artifact. [Pre-build review](evidence/manuscript-source-review.md) · [Build record](evidence/manuscript/phase2d-tectonic.json.gz) · [PDF review](evidence/manuscript-render-review.md)
