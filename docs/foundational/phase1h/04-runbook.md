# Phase 1H runbook

This ordinary application receives work after initialization. Returned typed results determine the next ordered frontier inside one retained adaptive member. No live provider credential is required or used by these examples. The same component works directly or through one ordinary wrapper. Current final qualification/review identities will be added after the Task3 review gate; earlier Task2/Phase1G evidence retains its original source identity.

Run from repository root using the pinned local Go toolchain and dependency. Use a fresh database path for each `init`. Recover by opening the existing store; never initialize a replacement root or raise persisted limits.

## Direct and nested scalar

```sh
unset OPENAI_API_KEY
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
cat >/tmp/adaptive-request.json <<'JSON'
{"seeds":[{"work_id":1,"value":0},{"work_id":2,"value":1},{"work_id":3,"value":2}],"max_rounds":3,"max_work":8}
JSON
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-direct.sqlite --capacity 1
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-direct.sqlite --id work --input /tmp/adaptive-request.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . replay --db /tmp/adaptive-direct.sqlite
go -C examples/adaptive/consumer run . recover --db /tmp/adaptive-direct.sqlite

go -C examples/adaptive/consumer run . init --db /tmp/adaptive-nested.sqlite --nested --capacity 2
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-nested.sqlite --id work --input /tmp/adaptive-request.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . replay --db /tmp/adaptive-nested.sqlite
go -C examples/adaptive/consumer run . recover --db /tmp/adaptive-nested.sqlite
```

Pure scalar output exhausts frontiers `[1,2,3]`, `[7,10,11]`, `[31]`: `complete:true`, `stop_reason:exhausted`, rounds3, work_count7, remaining empty. Scalar direct/nested enclosing-inclusive invocations and created runs are8/9, effects0. Fixed root1 is separate. Maximum depth is2/3 and does not increase for later rounds.

## Paired controlled scalar observations

In a separate terminal start a local fixture; retain its printed endpoint and ledger. Stop it after the run. It returns observations only; ALAN derives all work identities and admissions.

```sh
python3 docs/foundational/phase1h/tools/controlled_provider.py --case scalar-primary --port 8765 --ledger /tmp/adaptive-primary-provider.jsonl
# In another terminal:
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-controlled.sqlite --leaf controlled --nested --capacity 2
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-controlled.sqlite --id work --input /tmp/adaptive-request.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-controlled.sqlite --controlled-endpoint http://127.0.0.1:8765
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-controlled.sqlite
```

Repeat with a fresh store, identical request and `--case scalar-paired` on another port8766. Change only the endpoint argument to that controlled provider. Gamma returns0 instead of2; frontiers become `[1,2,3]`, `[7]`, forbidden10/11/31 are absent, rounds2/work_count4/exhausted. The application and leaf artifacts remain identical between paired controlled runs. Controlled primary effects7 versus paired4; direct/nested invocation+scalar-run charges8/9 versus5/6. Omitting `--nested` gives the same controlled component directly.

## Direct and nested unchanged evidence helper

The evidence profile accepts questions up to512 raw UTF8 bytes. Its result summary binds only disposition, while the unchanged helper's exact full authenticated answers/citations remain inspectable by the original result operation. This is an explicit narrower adaptive question profile; no truncation occurs. A513- or4096-NUL question rejects before leaf work. Result absence is an empty responses list.

```sh
cat >/tmp/adaptive-evidence.json <<'JSON'
{"seeds":[{"work_id":1,"question":"How does the retained WAL record relate to database changes?"}],"max_rounds":3,"max_work":8}
JSON
# Separate terminal; fresh server per controlled evidence execution:
python3 docs/foundational/phase1h/tools/controlled_provider.py --case evidence --port 8767 --ledger /tmp/adaptive-evidence-provider.jsonl
# Main terminal, direct:
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-evidence-direct.sqlite --leaf evidence --capacity 1
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-evidence-direct.sqlite --id evidence --input /tmp/adaptive-evidence.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-evidence-direct.sqlite --controlled-endpoint http://127.0.0.1:8767
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-evidence-direct.sqlite
go -C examples/adaptive/consumer run . replay --db /tmp/adaptive-evidence-direct.sqlite

# Restart provider (fresh counter), or use a second fresh server on8768.
go -C examples/adaptive/consumer run . init --db /tmp/adaptive-evidence-nested.sqlite --leaf evidence --nested --capacity 2
go -C examples/adaptive/consumer run . submit --db /tmp/adaptive-evidence-nested.sqlite --id evidence --input /tmp/adaptive-evidence.json
go -C examples/adaptive/consumer run . run --db /tmp/adaptive-evidence-nested.sqlite --controlled-endpoint http://127.0.0.1:8768
go -C examples/adaptive/consumer run . inspect --db /tmp/adaptive-evidence-nested.sqlite
go -C examples/adaptive/consumer run . recover --db /tmp/adaptive-evidence-nested.sqlite --controlled-endpoint http://127.0.0.1:8768
```

The fixture returns clarification then a real lookup/synthesis answer. Frontiers `[1]`, `[4]`, work_count2/rounds2/exhausted. Direct3invocations/5runs and nested4/6, effects4. Both helper members occupy the same leaf invocation depth. `--case evidence-terminal-answer` supplies a paired actual answer path: frontier `[1]`, no4, direct2/3 or nested3/4, effects3. These controlled responses do not retune helper prompts, corpus, grading or policy.

## Bounded stopping and unresolved work

Change the complete input JSON before submit, retaining the original variant bytes. For the primary scalar fixture: max_rounds1 returns `complete:false`, `round_limit`, rounds1/work_count3, remaining7/10/11; max_work5 returns `work_limit` with the same complete remainder and count; max_work6 permits round2 and returns work_limit with remaining31 and count6. Bounds use protocol success for an orderly incomplete reply. Empty next frontier wins over an exactly reached bound. Duplicate IDs, invalid observations, known leaf failure and root-budget denial send a diagnostic payload then protocol Failure.

`work_count` counts the complete accepted frontier before its first item intent, including later runtime-denied work. Emission-history prefixes and actual admitted runtime charges are separate. The root envelope32invocations/64runs/96reservations/depth3 persists across all rounds and recovery. A reservation made before interruption stays spent. Unknown required work blocks the barrier and public success, including when another sibling is known failed. Recovery never authorizes replacement or repeated provider contact for that unknown operation; this increment has no automatic eventual-result repair for it.

Cancelling a Run context is physical interruption, not logical cancellation. Environment-granted lifecycle closure may prevent any application reply; callers see the retained cancelled/abandoned/failed envelope and partial evidence. Wrappers forward present payloads and map child success/nonsuccess to their own Success/Failure; they do not impersonate arbitrary child cancellation/abandonment envelopes. Closure fences new admissions/reservations while already admitted children can drain locally. Original decision/evidence/charges remain retained.

## Process boundaries and reproduction

```sh
qualification_dir="$(mktemp -d /tmp/alan-phase1h-focused.XXXXXX)"
ALAN_H_OUT="$qualification_dir" python3 docs/foundational/phase1h/tools/qualify.py focused
python3 docs/foundational/phase1h/tools/checks.py "$qualification_dir/final-focused"
```

Focused tests use actual ACK/SIGKILL/fresh-process Open/Run for decision-before-intent, partial intents and partial consumption with an unfinished pure sibling. A bounded128-iteration pure test leaf exposes the last boundary without external uncertainty; its artifact is distinguished from the production controlled leaf. A read-only test-only SQLite probe chooses candidate cuts; full public Inspect and independent original-source/observation/intent/charge monitors validate the ACK. It never changes application state or schedules work. A Snapshot callback under Commit would deadlock because the Store mutex remains held, so it is not used there.

Capacity deployments add one fixed ordinary static run in the same root: Receive(start), Set(marker1), Success. After confirmed held HTTP contact, the parent records its untouched marker0/no-input projection, then signals an inherited pipe. The live child owner admits that environmental input through Store.AdmitInput. Automatic Run must consume it and finish marker1 before the waiting ACK while the HTTP response remains held. Dynamic charges stay unchanged; capacity fixtures report fixed root+witness2 separately. Previously admitted frontier items alone do not establish this causal progress.

Closure evidence is transparently split: the primary adaptive decision/kill/closure/recovery schedule tests successor admission fencing and duplicate known settlement; the separate live static-peer companion proves real late HTTP result settlement and a following scope_closed reservation denial. The companion leaves adaptive intake idle, keeps source peer open, uses one automatic Store.Run for its granted operator, and charges0invocations/0runs/1effect plus fixed root/peer/operator3. No same-execution late required round result is claimed.

Read-only replay uses a subprocess with no adapter registry/credential and the exact supported macOS sandbox-exec deny-network policy; ledger digests and contacts must remain equal. Unsupported OS isolation is reported honestly. Runtime replay validity and the independent application oracle are separate verdicts. An authentic ordinary faulty source overlay overwrites consumed gamma0 with2; real forbidden10/11/31 admissions must occur and the unchanged oracle must reject them even while Run/Replay remain valid. A race parent builds that variant with race instrumentation.

Final H source freeze, current-source broad integration/race, both consumer modules and verified extracted offline focused reproduction follow independent harness review. Historical G/F archives remain untouched alongside; the current H archive includes current required source/tools/inputs and pinned sqlite proxy without recursively copying old archives. Extracted focused reproduction is explicitly narrower than the full current-source suite. The rebase onto main56892cb changes compiler/executor inputs, so the earlier Task2 default result cannot qualify this source; a fresh changed-source default run is required. The original selected-input inventory and result remain historical. Task2's initial full behavioral/CLI matrix did not all predate implementation; the constructor-first red and later tests, original harness failures, byte fix and whole-frontier accepted-count RED→GREEN remain preserved. No full initial TDD chronology is claimed.
