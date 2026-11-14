# Introduction to ALAN

## 1. Why Would I Need ALAN?

Picture a Go service that investigates an engineering incident at 02:14 AM:
1. Search a runbook corpus for snippets matching a subsystem name.
2. Ask a language model to draft a root-cause summary citing exact character spans from those snippets.
3. Verify that the model did not invent or alter the quoted text.
4. Send an HTTP request to an external service to record the finding and update an incident counter.

Halfway through step 4—just after the HTTP request is handed to a background worker, or while one child agent has finished and its sibling awaits the model API—the container hits its memory limit or a deploy script sends `SIGKILL` (`kill -9`).

What survives when the OS process disappears?

In an ordinary Go program—or a typical agent framework built around in-memory loops and tool callbacks—almost nothing useful survives. Next steps lived on the Go call stack; retrieved snippets and spent budgets lived in heap structs; in-flight child workers and HTTP calls lived in goroutines and sockets; and permissions lived in closures. Restarting from step 0 either repeats paid model calls and risks double-applying the HTTP mutation, or halts because it cannot tell whether the interrupted request reached the server.

If you have used Temporal, you know one way to survive process death: log completed activity events and replay the Go workflow function from the top until the goroutine rebuilds its stack. That works well for service orchestration, but leaves live agent state implicit inside a replayed coroutine—and models an external call as a coarse activity rather than separating local program decisions, permission and budget checks, network dispatch, and unknown post-crash outcomes.

**ALAN (Agentic Lambda Abstraction Network)** explores a different foundation:

> *What if an interactive computation did not live on the host call stack at all? What if the program, its current step, its variables, its budgets, its permissions, and the state of every in-flight external operation were plain, inspectable data in SQLite?*

---

## 2. What Is Different About a Retained Computation?

ALAN is a Go library and single-host engine for **retainable computation**—programs whose control and state are explicit values on disk rather than active stack frames in RAM.

Instead of registering Go closures and replaying code from step 0, you author a program declaratively with [`author`](author/author.go) and compile it ahead of time into an immutable artifact. At runtime, its SQLite state tracks its instruction index, typed variables, loop snapshots, and status (`"open"`, `"succeeded"`, `"failed"`, `"abandoned"`, or `"cancelled"`):
- **Inspect without running:** Read a coherent snapshot of a live, paused, or crashed database—variables, budgets, and pending calls—without taking executor ownership or registering network adapters.
- **Resume in place:** Reopening after a crash continues directly from each `"open"` run's committed instruction index instead of re-executing earlier steps.
- **Audit history offline:** A read-only `Replay` command re-evaluates every committed transition from initial state through ALAN's pure reducer without touching the network.

To keep that state trustworthy across untrusted model outputs and mid-flight crashes, ALAN separates four concerns that ordinary agent loops blur together.

### 1. Untrusted Observations vs. Deterministic Steps vs. World Commitments

In a conventional agent loop, a language model can invoke tools and steer control flow directly. In ALAN's [Recoverable Document Investigator](examples/investigator/README.md), those concerns are separated:
- **Untrusted observations:** Model outputs and HTTP responses cannot mutate variables or call tools. Validated against a closed schema, they enter the inbox as **observations**; malformed JSON, incomplete streams, or refusals become zero defaults with `valid = false`.
- **Deterministic step reduction:** ALAN's reducer is a pure function with zero I/O. It reads the current state and one observation, verifies byte-for-byte that a model's quoted snippet and character offsets match the local corpus lookup, and emits the next state plus any outgoing **intents**. If a single character of a quote was altered, the program branches deterministically to `"failed"`.
- **Serialized world commitments:** Separate SQLite transactions commit the new state, check permissions, reserve budget slots, pass a local dispatch gate before network I/O, and record responses.

### 2. Physical Recovery vs. Logical Retry

Suppose a worker has a supervisor allowed to retry it once (`limit = 1`) on failure:
- **Event A (`SIGKILL`):** The worker is at step 2 when `kill -9` kills the OS process.
- **Event B (Program failure):** The worker receives bad data, executes `Failure()`, and commits status `"failed"`.

In Event A, the program did not fail—its OS process died. Counting a process restart as a retry would burn the retry budget and discard steps 0 and 1:
- **Physical recovery (Event A):** Reopening fences out the dead process by advancing the run's *activation epoch* while keeping the same logical *generation* and step sequence. The `"open"` worker resumes at step 2 with its variables and operation IDs intact, and the retry budget stays unspent.
- **Logical retry (Event B):** Physical recovery leaves the finished generation closed. The supervisor receives an authenticated terminal notification, increments its retry counter, and requests a retry—leaving generation 1 `"failed"` and starting **generation 2** at step 0 from initial state.

Because outgoing operation keys are scoped to a logical generation, resuming generation 1 after a crash refers to the *same* operation, whereas retrying in generation 2 creates a *new* operation identity that even an idempotent endpoint treats as a distinct request unless generation 1 was reconciled first.

### 3. What Durability Cannot Tell You

Persisting every step locally still leaves one hard boundary: **knowing what your process committed locally is not the same as knowing what happened on a remote server.**

In [Phase 3B.7 Matched Concrete Effect Histories](https://github.com/JakeFAU/alan-research/blob/main/docs/foundational/phase3b7/05-results.md), two executions start from copies of the same closed SQLite seed database and advance until an authorized worker passes the local dispatch gate to increment an external HTTP counter:
- **Zero branch:** A barrier holds the worker *just before* opening the HTTP connection, and `SIGKILL` kills the process (**0 requests, 0 increments**).
- **Applied branch:** The worker sends the HTTP request, the server commits the increment (**1 request, 1 increment**), and a barrier holds the worker *just before* writing the HTTP response to SQLite. `SIGKILL` kills the process.

When both databases are reopened and exported via `runtime.InspectExecution`, **their complete local exports (`inspect.json` across 41 exported SQLite tables and 94 rows) and final `run.json` reports match byte for byte** (while the checker treats raw `.sqlite` files as opaque bytes).

Because no local store can tell whether a post-admission crash happened right before the packet left the machine or right before the response hit disk, ALAN refuses to guess: auto-retrying an ordinary call would double-apply the effect in the applied branch, while fabricating a timeout error would run fallback logic on a false premise. Instead, ALAN distinguishes four recovery regimes:
1. **Original non-idempotent and ordinary adapter attempts:** If a response is missing after local dispatch admission, recovery never automatically re-sends that original attempt or fabricates a result; the budget slot stays spent and the operation remains `outcome_unknown`.
2. **Idempotent raw external attempts (`--profile idempotent`):** When the endpoint deduplicates by ALAN's wire key, recovery re-presents the original attempt's immutable key—applying it in the zero branch or retrieving the cached receipt in the applied branch.
3. **Controlled receipt reads (`adapters/receiptquery`):** If the external service exposes an application log, a separate resolver run can be granted permission to read the ledger for the original key and—under a separate grant—adopt a matching receipt.
4. **Newly approved bounded observations (`author.BoundedSeries` / `Store.ApproveContinuation`):** A failed or unresolved Open-Meteo request does **not** permanently block future weather observations. While `weather-1` remains attached to its original operation ID as `outcome_unknown`, an operator can approve the next slot (`weather-2`) within the allowance for a fresh, separately charged reading.

### 4. Supervision, Composition, and Two-Agent Coordination

- **Orthogonal permissions:** Supervisors are ordinary programs, and creation, messaging, terminal notifications, and lifecycle authority (`Retry`, `CancelRun`, `RetireSubject`) are independent grants. Watching a worker fail does not grant permission to retry it; stopping a worker that another supervisor might retry requires retiring the subject (`RetireSubject`), not just cancelling its current run (`CancelRun`).
- **Root budgets and zero-slot waiting parents:** Call trees share a fixed root budget for depth, invocations, created runs, and effect reservations. A parent waiting at `AwaitInvocation` is an `"open"` row in SQLite holding **zero** goroutines or HTTP worker slots, so leaf workers never deadlock behind waiting ancestors.
- **Result custody and reciprocal servicing:** ALAN commits incoming messages to SQLite before recording delivery acknowledgments. Under concurrent peer requests, a *serial* agent admits incoming requests to its inbox and processes receipts and acknowledgments, but withholds **incoming request application** while its own outgoing request is unresolved—so two serial peers wait on each other forever unless at least one is configured to *service* incoming requests while waiting (using explicit request-bound authority grants when servicing requires the caller's permissions).

---

## 3. Can I Run Something That Demonstrates the Difference?

You can see retained state, allowances, original operation IDs, offline replay, and `SIGKILL` recovery using the [Durable Weather Agent](examples/weather-agent/README.md) (`./examples/weather-agent`).

Build the binary and unpack the archived SQLite store from a live Open-Meteo reading—no network required:

```sh
work=$(mktemp -d)
go build -o "$work/weather-agent" ./examples/weather-agent
gzip -dc examples/weather-agent/testdata/live.sqlite.gz > "$work/saved.sqlite"

"$work/weather-agent" status --db "$work/saved.sqlite"
"$work/weather-agent" replay --db "$work/saved.sqlite"
```

`status` reads the retained state from SQLite:

```text
Read 1. Previously retained. At 2026-11-03T20:15:00Z, Open-Meteo reports 80.3 °F with 77% relative humidity for (26.93, -80.08), using grid (26.93391, -80.07091). Original operation ["weather-1",1,1,0]. Response retained (HTTP 200). Observation selected. Opportunity spent.
Location 26.93, -80.08. Units fahrenheit. Opportunities spent 1 of 1. Provider cost unknown.
No remaining read slot.
```

`replay` opens that `.sqlite` file read-only with no HTTP client and verifies all 15 committed reductions (`Replay verified 15 committed reductions.`).

Next, reproduce a `SIGKILL` interruption and explicit continuation on a two-allowance store (`--allowance 2`):

```sh
go test -count=1 -tags integration \
  -run '^TestWeatherProcessRecovery$/^accepted_and_lost_then_explicitly_reobserved$' \
  -v ./examples/weather-agent
```

1. `weather-agent run` sends Read 1 (`["weather-1",1,1,0]`). The loopback fixture accepts the `GET`, withholds the response, and sends `SIGKILL`.
2. Two `weather-agent resume` runs reopen the store. Because `["weather-1",1,1,0]` already passed local dispatch admission and its response is missing, `resume` **does not re-send** `weather-1`; provider contacts remain `1`, and `weather-1` stays `"open"` in `reserved_unknown`.
3. Approving Read 2 (`weather-agent refresh --db ... --read 2`) and running `resume` preserves `["weather-1",1,1,0]` as unresolved (`new_read_after_unknown_response`) and executes a second operation `["weather-2",1,2,0]`, reporting `76.1 °F with 77% relative humidity at 2026-11-03T10:00:00Z` (`contacts=2 charged=2/2`).

*(For a 50-line Go program, run [`examples/quickstart/main.go`](examples/quickstart/main.go); clean up with `rm -rf "$work"`.)*

---

## 4. What Guarantees Do I Get, and What Don’t I Get?

- **Model-level guarantees:** Within the monograph's formal model, ALAN proves deterministic pure reduction, bounded loop termination, orthogonal permissions, suppression of intents cancelled before first authorization, root budget conservation, waiting-parent slot freedom, side-effect-free `Replay`, and the impossibility of local-only recovery guaranteeing both at-most-once execution and identical continuation after a lost non-idempotent response. Exhaustive finite-state model checks verify component substitution in owning trees, local dispatch quiescence after revocation, two-agent result custody, deadlock freedom when one peer services incoming requests while waiting, and one-hop request-bound authority delegation.
- **Source-level guarantees:** The single-host SQLite WAL runtime (`github.com/JakeFAU/alan`) enforces activation-epoch fencing on `runtime.Open` and verifies `SIGKILL` recovery at specific instrumented cuts—12 local/wire cuts plus repeated-interruption and refresh-after-loss cases in [`examples/weather-agent/process_test.go`](examples/weather-agent/process_test.go), worker crash cuts in [`examples/investigator`](examples/investigator/README.md), and matched 41-table, 94-row local exports in [Phase 3B.7](https://github.com/JakeFAU/alan-research/blob/main/docs/foundational/phase3b7/05-results.md).
- **What you don’t get today:** ALAN is a single-host SQLite runtime, not a distributed store or hosted control plane. It cannot resolve lost non-idempotent calls without an idempotent endpoint, receipt ledger, or approved continuation; and general congruence under peer budget contention, multihop delegation, and mechanized compiler verification remain open problems.

---

## Going Deeper

- **[Technical Guide (`technical-guide.md`)](technical-guide.md)** — `Projection` internals, identity tiers, admission gates, SQLite storage formats (3–16), and exact theorem and counterexample mappings.
- **Examples & Contracts:** **[Weather Agent](examples/weather-agent/README.md)** · **[Quickstart](examples/quickstart/main.go)** · **[Document Investigator](examples/investigator/README.md)** · **[Reciprocal Delegation](examples/delegation/README.md)** · **[Inspection](docs/inspection.md)** · **[Continuations](docs/continuations.md)**
- **[ALAN Research Monograph (`paper/alan-dissertation.md`)](https://github.com/JakeFAU/alan-research/blob/main/paper/alan-dissertation.md)** and **[Foundational Notes (`docs/foundational/`)](https://github.com/JakeFAU/alan-research/blob/main/docs/foundational/01-foundations.md)**
