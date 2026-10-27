# ALAN computational foundations

Status: research candidate, 24 October 2026. ALAN means Agentic Lambda Abstraction Network. The candidate is an open transition system with deterministic local steps, explicit interaction requests, and independently scheduled computations. This is a foundation to test, not a proved minimal architecture or a runtime implementation.

## Scope and repository baseline

The repository inspected for this investigation was on `main` at `c41961040f4392b2d615306f7a24cb211ecb2ec7`, with a clean worktree. It contained `.gitignore`, an Apache 2.0 `LICENSE`, and `go.mod` declaring `github.com/JakeFAU/alan` with Go `1.27.3`. The existing `docs/foundational` directory was empty. There were no source files, tests, runtime, SDK, provider integrations, or existing architecture documents to inherit. The Go module does not determine the abstract semantics or commit this phase to a language implementation.

The investigation plan was to compare primary foundations, specify a candidate with explicit assumptions, construct ordinary recursive compositions, expose the runtime obligations, challenge ten concrete failure cases, and obtain independent adversarial review. The deliverables are [Composition](02-composition.md), [Runtime boundary](03-runtime-boundary.md), [Adversarial cases](04-adversarial-cases.md), and [Decisions](05-decisions.md), alongside this document. Review findings and their disposition are preserved in Decisions.

The phase is research and design only. No runtime, SDK, persistence layer, provider integration, infrastructure, or external dependency is introduced. Proposed semantic rules below are ALAN design choices unless explicitly attributed to an established result.

## Competing foundations

| Foundation | What it contributes | What it leaves unresolved for ALAN |
| --- | --- | --- |
| Lambda calculus or universal machine | A basis for arbitrary effectively computable internal transformations | Interaction, scheduling, failure, and durability are additional semantics |
| Register machine | A small instruction model with explicit control and unbounded abstract storage | An instruction minimum says little about a usable concurrent interface |
| Actor configurations | Asynchronous communication, dynamic creation, private state, explicit network boundaries | Durable history and authority cannot be inferred from actor identity |
| CSP | Composition through communication and explicit synchronization | Rendezvous and buffered asynchronous admission have different behavior |
| π-calculus | Changing topology through transmission of names, with scoped connectivity | A name is not automatically a checked security capability or lifecycle right |
| I/O automata | Inputs, outputs, internal actions, compatibility, and conditional composition results | ALAN's bounded admission and failures require a deliberate correspondence |
| Interaction Trees | A compositional account of return, internal steps, and requests with response continuations | Concurrent branches, scheduling, persistence, and lifecycle need additional rules |

These are alternative perspectives, not a parts list. A useful candidate can use operational rules to specify an interactive machine without embedding every calculus or introducing an agent class for every perspective.

### Computability and small machines

Church and Turing establish formal accounts of effective computation and equivalences among particular formalisms. Turing's universal machine simulates a described machine. The identification of those formal accounts with the informal notion of effective calculability is the Church–Turing thesis, not a proved statement about all possible physical systems. Neither universality nor that thesis establishes an adequate contract for open, concurrent, durable agents. [Church, 1936](https://www.cis.upenn.edu/~cis5110/Church-UnsolvableProblemElementary-1936.pdf); [Turing, 1936](https://trhvidsten.com/docs/classics/Turing-1936.pdf).

Minsky's 1961 Theorem Ia supplies a two-register construction using increment/jump and decrement/zero-test branching, with unbounded integer registers and encoded input/output. It is a precise sufficiency result under those assumptions. It neither proves that ALAN needs two primitives nor makes finite bounded computation universally powerful by itself. [Minsky, Recursive Unsolvability of Post's Problem of “Tag” and Other Topics in Theory of Turing Machines, section 3](https://www.wolframscience.com/prizes/tm23/images/Minsky.pdf).

For ALAN, a credible universality argument would exhibit an encoding of a universal machine into repeated finite transitions, with unbounded abstract state and an unbounded number of steps. A permanently finite state space cannot supply that property. Finite real deployments instead approximate the abstract machine subject to quotas and exhaustion outcomes.

### Interaction and concurrency

Hewitt, Bishop, and Steiger propose uniform message-based computation. Agha and colleagues make actor configurations explicit: actors, messages in transit, visible receptionist names, and external names. Their composition results and fairness assumptions apply to their defined model. The useful lesson for ALAN is that the compositional unit needs network state and a boundary, not merely the signature of an individual actor callback. [Hewitt et al., 1973](https://www.ijcai.org/Proceedings/73/Papers/027B.pdf); [Agha et al., A Foundation for Actor Computation, 1997](https://osl.cs.illinois.edu/media/papers/agha-1997-jfp-a_foundation_for_actor_computation.pdf).

Hoare's original CSP uses matching input/output as synchronization; a participant can wait for its counterpart. ALAN's proposed durable asynchronous mailbox admission makes a different choice. A rendezvous protocol can be built through acknowledgments, but that encoding must be checked against the intended observations and failure behavior. [Hoare, Communicating Sequential Processes, 1978](https://www.cs.cmu.edu/~crary/819-f09/Hoare78.pdf).

Milner, Parrow, and Walker model changing connectivity by communicating names. Their equivalence theory also warns against informal replacement arguments: a behavioral equivalence may fail to be preserved by an operator unless suitable restrictions or a stronger congruence are used. ALAN must separately define name scope, capability authority, and the contexts in which replacement is promised. [A Calculus of Mobile Processes, Part I](https://www.pure.ed.ac.uk/ws/files/16426053/A_Calculus_of_Mobile_Processes_I.pdf); [Part II](https://www.pure.ed.ac.uk/ws/files/16426068/A_Calculus_of_Mobile_Processes_II.pdf).

Lynch and Tuttle's I/O automata distinguish environment-controlled input actions from locally controlled output and internal actions. Their automata are input-enabled, and composition requires compatibility conditions. ALAN cannot reject an input due to capacity and then inherit those theorems unchanged. One possible correspondence treats every attempted send as an input and acceptance/rejection as an output, but the resulting automaton and fairness conditions still need definition. [Lynch and Tuttle, An Introduction to Input/Output Automata, 1989](https://groups.csail.mit.edu/tds/papers/Lynch/CWI89.pdf).

### Operational specification and expressibility

Plotkin's structural operational semantics provides a method for defining behavior through transition rules over configurations. It motivates specifying admission, local reduction, dispatch, observation delivery, and lifecycle changes separately. Writing a function-shaped tuple is not a substitute for those rules. [Plotkin, A Structural Approach to Operational Semantics, 2004 presentation of the foundational notes](https://www.cs.cmu.edu/~crary/819-f09/Plotkin81.pdf).

Interaction Trees use `Ret`, `Tau`, and `Vis`; the visible-event continuation is indexed by the environment's response. This separates describing an interaction from interpreting it. A family of possible response continuations does not mean that every continuation executes concurrently. The original paper leaves further reasoning about concurrency and nondeterminism as future work. Its results support an interactive foundation but do not prove ALAN's proposed concurrent kernel. [Xia et al., Interaction Trees, 2020](https://arxiv.org/pdf/1906.00046).

Felleisen makes expressibility relative to an encoding discipline and observed behavior. A universal whole-program interpreter is not automatically a local elimination of a language construct. For ALAN, any claim that a construct is derived must state the retained host operations and preserve the relevant interaction, divergence, failure, and cancellation behavior. [Felleisen, On the Expressive Power of Programming Languages, 1991, definitions 3.3–3.4](https://www2.ccs.neu.edu/racket/pubs/scp91-felleisen.pdf).

### Recovery traditions

OTP supplies concrete policies for restart, escalation, and repeated-failure limits. Durable workflow systems such as Temporal distinguish deterministic workflow computation from external activities and use event history for recovery. State-machine replication literature connects deterministic processing with ordered commands. These traditions address different problems: supervisor restart is not durable state recovery, and replicated state is not atomic external execution. [OTP supervision](https://www.erlang.org/doc/system/sup_princ.html); [Temporal workflow determinism](https://docs.temporal.io/workflow-definition); [Schneider's state-machine survey](https://www.cs.cornell.edu/fbs/publications/SMSurvey.pdf).

The architectural inference is to separate policy, deterministic computation, durable decisions, and effect protocols. It is not to import an existing workflow orchestrator or supervisor hierarchy into ALAN.

## Proposed candidate semantics

Start with an open labeled transition system:

```text
C --label--> C'
label ∈ {internal} ∪ Inputs ∪ Outputs
```

`C` is the global semantic configuration: a map of executions and their private states, messages/inboxes, pending intents and outcomes, capability bindings, and lifecycle authority records. This is a mathematical description, not a proposal for one global database or globally serialized loop.

Each execution has an immutable semantic version binding `v` and state `X = (D, F)`. `D` contains application data and correlation/obligation records. `F` is a finite map of uniquely identified live continuations. Each continuation contains a versioned program label, serializable environment, and readiness or wait condition. Distinct identifiers preserve two identical tasks; an unlabelled mathematical set would collapse them.

The executable local presentation is:

```text
δ_v(X, o) = (X', E)
```

`o` is one admitted observation, including a selected continuation when selection is not otherwise deterministic. `E` is a finite sequence of immutable intents; its ordinal gives each intent a stable identity. Order in `E` does not imply serial completion. `δ_v` is deterministic for fixed inputs and performs a finite amount of internal computation. Overall execution can be nonterminating through an unbounded sequence of such steps.

The proposed `Step(S, O) -> (S', E, K)` is adequate as a presentation only if `S` includes the current residual control and `K` explicitly denotes the next continuation frontier. Normalize its output as `X' = (D', K)`, avoiding two independently authoritative copies of control state. `K` can contain zero, one, or many live continuations; it is not a single tail call. Alternatively, keep all control in `S'` and omit the redundant `K` field. This is a representational choice, not a change in computational power.

### Continuation and completion rules

1. A ready continuation may take a bounded internal step when scheduled. A waiting continuation becomes eligible only for an observation matching its recorded wait/correlation condition.
2. Selection is deterministic from persisted state and observation, or recorded as part of `o`. Multiple eligible waiters do not all consume the same event by accident.
3. A step may replace its continuation, introduce several fresh ones, wait, or remove a completed one. Other live continuations remain unless the program explicitly updates them.
4. State changes within one execution serialize. The frontier permits overlapping pending work; it does not permit ungoverned parallel writes to `D`.
5. Distinct executions can advance independently. Dynamic creation is requested as an effect and becomes a new execution only when the runtime admits it.
6. Empty `F` alone is not a terminal result. Completion is an explicit outcome with a declared disposition for pending effects, messages, and owned work. An empty ready set can instead mean waiting, deadlock, or deliberate suspension.

A terminal decision closes ordinary computation according to the lifecycle contract. Late external receipts can still be retained for reconciliation without reviving the completed program. Long-lived services need not produce a terminal success at all.

A defined program fault is a deterministic terminal outcome under `v`, committed with the observation and any permitted final intents. A physical interruption before commit leaves the prior state intact and does not itself fail the logical run. Suspected worker failure instead permits an authorized epoch change or an explicit decision to abandon the run; the distinction and durable failure notifications are specified in [Runtime boundary](03-runtime-boundary.md).

An arbitrary host callback that loops forever violates the finite-step assumption. A future implementation must constrain the language, instrument evaluation with deterministic fuel, or otherwise expose resumable internal steps. A timeout killing an opaque callback is an activation failure, not a deterministic continuation checkpoint or a terminal program failure. The run remains recoverable unless an authorized abandonment decision ends it. This is a substantial implementation obligation, not solved by naming a construct OneShot.

### Global transition families

| Rule family | Premise | Effect on semantic configuration |
| --- | --- | --- |
| Admit | Valid capability, valid target binding, and available capacity | Record input identity and inbox admission, or return an explicit rejection |
| Reduce and commit | Eligible observation, bounded `δ_v`, valid activation and sequence | Atomically advance one execution and retain its outgoing intents |
| Dispatch | Committed unresolved intent, permitted attempt | Attempt interaction; leave its result unresolved until evidence is recorded |
| Complete effect | Valid correlated receipt or authoritative reconciliation | Record a result/error/unknown outcome for later consumption |
| Schedule | Runnable work and allowed scheduling policy | Select an enabled local step without changing application data outside `δ_v` |
| Lifecycle | Current authority and matching revision/generation | Serialize creation, cancellation, transfer, retry, or reactivation outcomes |

These are a semantic inventory. Whether any can be eliminated from a future surface language depends on what the remaining interpreter provides. The rules are intentionally more explicit than “the runtime handles it”; their atomicity and failure obligations are in Runtime boundary.

## OneShot and Conditional

The names alone do not settle independence. Consider three definitions:

| Definition of the available constructs | Expressibility conclusion |
| --- | --- |
| `OneShot(f)` executes one bounded pure reducer that can select residual control | For a total bounded pure predicate `p`, `Conditional(p, A, B)` is derived by returning only the selected continuation |
| OneShot leaves always emit their complete fixed observable traces and terminate normally, with no input-dependent failure, divergence, or early exit; composition has only fixed sequencing and no dynamic dispatch | A state-dependent branch between distinct observable actions cannot generally be derived |
| `Conditional(p, A, B)` selects between already available computations | A constant-true guard embeds an existing unconditional computation; it does not create arbitrary computations or effects from Boolean selection alone |

In the expressive first case:

```text
Conditional(p, A, B)
  = OneShot(s -> (s, noEffects, selectedContinuation(p(s), A, B)))
```

This is a relative encoding. Selection is available in the host reducer and continuation interpreter. It has not vanished from the semantics. The predicate runs once, the unchosen branch is not evaluated, and the guard cannot introduce hidden I/O.

For the restricted second case, total, normally terminating leaves emit their complete fixed traces. Induction over fixed sequencing therefore shows that the sequence of effect labels is input-independent. A conditional that emits `left` for one input and `right` for another violates that invariant. Allowing input-dependent failure or early exit invalidates this particular induction because observed prefixes can differ. This is a small non-expressibility argument under the stated restrictions, not a theorem about every possible OneShot API.

An effectful predicate is a computation that requests an effect, waits, then selects using the recorded result. A partial or expensive predicate needs internal steps and may diverge. Replacing it with an atomic pure test would change behavior. Conditional selection alone also cannot provide mutation, creation, communication, or recursion unless its branches or interpreter already have those abilities.

The recommendation is to treat OneShot and Conditional as possible programming conveniences. Neither is the preferred foundational agent type. An interactive transition model states the assumptions that both names would otherwise conceal.

## Explicit assumptions

| Assumption | Consequence if it fails |
| --- | --- |
| Local transitions are finite and deterministic under a complete version binding | A run may starve peers or reconstruct a different history |
| Abstract state and execution length are unbounded for computability claims | Universality claims must be weakened for a fixed finite machine |
| Every finite transition creates only finitely many intents and continuations | One step could require infinite work before yielding |
| Nondeterministic inputs and relevant scheduling choices become durable observations | Replay can choose a different branch or race winner |
| State commits and lifecycle authority serialize at a declared subject boundary | Split-brain workers or competing supervisors can both act as current |
| Trusted runtime/storage/capability enforcement follows its declared crash model | Byzantine behavior or corruption needs a different fault model |
| External guarantees are specific to each adapter and resource | Internal replay correctness cannot imply exactly-once effects |
| Fairness, eventual recovery, connectivity, and resource admission are additional progress premises | Safety may hold while the system never makes progress |

No global wall-clock order, perfect failure detector, globally atomic composite, fixed number of agent types, or synchronous network is assumed.

## Properties that require separate arguments

| Property | What would establish it | What does not establish it |
| --- | --- | --- |
| Turing completeness | An effective universal-machine encoding under explicit resource assumptions | A state-transition signature or a recursive API |
| Recursive compositional closure | A construction of composite state/transitions/boundary plus preservation of well-formedness | Giving a network the same method name as a leaf |
| Observational equivalence | A chosen observation relation and a contextual or bisimulation argument | Equal final values or matching schemas |
| Concurrency expressiveness | An encoding preserving the chosen independence, communication, and fairness observations | Sequentially simulating some interleavings |
| Safety | Invariants preserved by every admitted transition, including failure/recovery | Successful examples or eventual retries |
| Liveness | Progress proof under named fairness and environment assumptions | Turing completeness, replay, or absence of a safety violation |
| Durable replay correctness | Complete ordered observations, stable semantics, and a replay correspondence | Logging outputs, restarting a process, or replicating storage alone |

The documents offer a coherent candidate, explicit constructions, conditional arguments, and falsifying traces. The remaining proofs and architectural choices are tracked in [Decisions](05-decisions.md).
