# Bounded data-dependent work runbook

This is a provisional single-host composition profile. Read the [contract](01-composition-contract.md) and [qualification report](evidence/task4/report.md) for exact evidence and remaining limits. Process tests use the test executable with public Open/Run, while the CLI has its own application tests; these are separate evidence sources.

## Run a deterministic collection

From the repository root, with Go1.27.3, CGO and a C compiler:

```sh
export GOTOOLCHAIN=local
alan_work_dir=$(mktemp -d)
go build -buildvcs=false -o "$alan_work_dir/alan-compose" ./cmd/alan-compose
"$alan_work_dir/alan-compose" init --db "$alan_work_dir/work.sqlite" --nested --capacity 1
cat > "$alan_work_dir/items.json" <<'JSON'
[{"item_id":"item-a","number":2},{"item_id":"item-b","number":-3},{"item_id":"item-c","number":2}]
JSON
"$alan_work_dir/alan-compose" submit --db "$alan_work_dir/work.sqlite" --id work-1 --input "$alan_work_dir/items.json"
"$alan_work_dir/alan-compose" run --db "$alan_work_dir/work.sqlite"
"$alan_work_dir/alan-compose" inspect --db "$alan_work_dir/work.sqlite"
"$alan_work_dir/alan-compose" replay --db "$alan_work_dir/work.sqlite"
"$alan_work_dir/alan-compose" recover --db "$alan_work_dir/work.sqlite"
```

Expected ordered doubled values are4,-6,4 under the original three IDs. `submit` acknowledges transport admission. The collection accepts the work only when its own Receive commit retains the typed list. The static driver's intake and the enclosing invocation admission are different boundaries. A successful command exit means the command returned normally; inspect `Snapshot.Roots[].Outcome` for public completion and `Result.results` for ordered item outcomes. Local quiescence or a local terminal is not sufficient.

One root handles one work obligation. Use a new store for new root work; recovery opens the original store and never resubmits its input. Duplicate transport delivery reuses the original binding. Physical activation epochs advance on recovery, while item IDs, invocation operations, accepted artifacts, root limits and spent charges remain. Recorded outcomes can still await ordinary consumption. An unknown external operation stays unknown and spent; recovery does not authorize an automatic repeat or replace it with a fresh business operation.

## Limits and identity

The default input maximum is8 records, maximum invocation depth3 with the resource root at depth0,32 admitted invocations,64 created program runs,96 external-effect reservations, and physical capacity1 or2. The runner's lower physical ceiling also applies. These limits are not paid-provider authorization. Depth3 supports root→wrapper→collection→item. Each template member remains generation1; scoped Retry is unsupported. Fresh Invoke creates different work with a fresh operation.

One static driver is fixed provisioning cost. Deterministic direct work of size n charges n+1 invocations/runs; nested charges n+2. The unchanged evidence leaf has two member programs, so created runs are2n+1 direct or2n+2 nested, with unchanged invocation counts. Empty work still incurs enclosing calls. Duplicate IDs or a ninth item reject before collection acceptance/child work. A later exhausted root budget can reject individual invocations after list acceptance, producing ordered non-success outcomes.

The public API is [`compose.New`](../../../compose/collection.go) plus ordinary author Invoke/Await and finite record lists. [`examples/worktree`](../../../examples/worktree/README.md) defines the deployment and the separate external consumer. It uses the same collector artifact and request/result schemas directly or nested. Application code receives public results; qualification inspection does not supply application policy.

Schema3 and legacy scalar programs retain their prior meaning. Invocation-enabled provisioning uses schema4, `alan/composition/v1`, versioned bindings, and new collection/invocation artifact semantics `alan/1g/v1` with author compiler v2. Legacy author compilation still emits the original v1 bytes. Open/Inspect/Replay select the proper version; there is no migration or repair of inconsistent descriptors. `DispatchExternal` refuses the composition profile; physical dispatch belongs to the existing Run owner.

## Controlled evidence and closure

For reproducible effectful work, initialize with `--leaf evidence`, submit `question` fields and use an explicitly controlled loopback Responses server with `run --controlled-endpoint http://127.0.0.1:PORT`. The CLI accepts no arbitrary remote endpoint or ambient provider credential. The retained helper's planner, lookup, synthesis and diagnostic policy remain ordinary programs. Controlled responses establish protocol behavior, not answer quality or actual provider cost.

Cancellation is operator-injected in qualification. There is one live Store owner: kill/close it before a separate operator opens the store, or issue supported calls through the same owner. Root closure prevents new descendant admissions and effect reservations. Already admitted children and authorized uncertain work retain their records; late evidence cannot reopen the root. This is not group cancellation, supervision policy, compensation or cleanup.

## Reproduce exact qualification

The current [source manifest](source-manifest.json) identifies metadata-tool source `ce4bbf6f9e6a0c7e94ccdfec09d97f9355b5e9bb1a6f49fcca2b17fe7256482a`; its Go/runtime/module/fixture inputs exactly match fully qualified source `43b86781f819c6bf7a957956aceaf717a5808873114e8213bf1147620bdf2ec3`. The [final-fix report](evidence/final-fix/report.md) records the focused metadata regression and fresh extracted consumer checks; it does not claim a repeat of the broad suites. New runs capture their executing platform and `go version` with the selected command environment. `--parse-existing` preserves the recorded execution environment; it cannot recover metadata absent from an old record.

The source manifest uses sorted uint64-length-delimited paths and bytes. The [archive manifest](archive-manifest.json) additionally identifies documentation, original control inputs and the local SQLite module proxy. The archive excludes credentials, temporary databases, build caches and itself. Toolchains are not bundled. Historical Phase1F source remains a separate embedded archive for historical claims.

```sh
# In the original checkout, retain new output under a fresh directory:
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
export GOCACHE=/tmp/alan-phase1f-go-build GOFLAGS=-mod=readonly
export ALAN_QUALIFICATION_OUT=/tmp/alan-phase1g-my-verification
python3 docs/foundational/phase1g/tools/qualify.py default
python3 docs/foundational/phase1g/tools/qualify.py normal
python3 docs/foundational/phase1g/tools/qualify.py race
python3 docs/foundational/phase1g/tools/qualify.py consumer
python3 docs/foundational/phase1g/tools/qualify.py consumer-race
python3 docs/foundational/phase1g/tools/checks.py
python3 docs/foundational/phase1g/tools/compare.py normal
```

Collectors remove `OPENAI_API_KEY` without reading or printing it, verify source before/after, retain actual numeric exits, raw Go JSON and separate structured marker `WORKTREE`. Root `./...` excludes `examples/worktree/consumer`; its commands are mandatory and use a240s race watchdog. `normal` and `race` include integration. Existing Phase1A/B/C/conformance markers remain separate;15/20/32/6 record counts are checked and semantic payloads compared. A pass count is not proof of an unbounded property.

`tools/reproduce.py` verifies every archive member, extracts to a fresh directory, starts with an empty GOMODCACHE, sets GOPROXY to the bundled file proxy, and reruns all five suite modes plus comparisons and auxiliary checks. Retained full-suite output for source43b86781 remains under `evidence/task4/extracted` and is also snapshotted under `evidence/final-fix/qualified-source-43b86781/task4/extracted`. The original source/archive manifests and all original Task4 outputs are preserved in that qualified-source directory. For current focused verification, run `python3 docs/foundational/phase1g/evidence/final-fix/verify_focused.py` in a fresh copy with no existing `evidence/final-fix/extracted` result; it runs Python metadata tests and consumer test/vet/module verification only. The script refuses to replace its existing result. To reproduce the original full qualification from its retained archive, use a fresh host workspace with this exact layout:

```sh
alan_repro_root=$(mktemp -d)
mkdir -p "$alan_repro_root/docs/foundational/phase1g/tools"
cp docs/foundational/phase1g/evidence/final-fix/qualified-source-43b86781/source.tar.gz \
   docs/foundational/phase1g/evidence/final-fix/qualified-source-43b86781/archive-manifest.json \
   "$alan_repro_root/docs/foundational/phase1g/"
cp docs/foundational/phase1g/tools/reproduce.py \
   "$alan_repro_root/docs/foundational/phase1g/tools/"
env -u OPENAI_API_KEY python3 \
   "$alan_repro_root/docs/foundational/phase1g/tools/reproduce.py"
# Results: "$alan_repro_root/docs/foundational/phase1g/evidence/task4/extracted"
```

The host needs the stated Go/Python/C toolchains; the archive supplies the source and dependency proxy. Generated source/qualification outputs are not folded recursively into their own identities.

Process and provider deadlines are hang watchdogs only: parent acknowledgment/transport-entry/connection-close300s, child process360s, process controlled external300s, capacity fixture context180s/external150s, final root suite20m and consumer240s. Ordering is established by durable boundary/transport acknowledgments. On the qualified macOS host, Replay children additionally run under `sandbox-exec` with `deny network*`, without adapter registry or credentials. The generic test reports when that platform-specific OS denial is unavailable rather than claiming it elsewhere.
