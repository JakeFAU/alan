# Phase 1H independent contract and Task 1 review

Reviewed head: `bb9ff0e` against `4550e0f4ba4c57cae8e84d1afe9fbf47e16786d8`, including `.superpowers/sdd/phase1h-plan/review-4550e0f..bb9ff0e.diff`, assignment, Task 1 brief/report, literal oracle, and existing public author/schema/compose/runtime code. Read-only checkout; this report is the only written file. No subagents and no broad suite reruns.

**Spec verdict: Issues. Task Quality: Needs fixes.** The ordinary adaptive design is supported by existing machinery, but the proposed evidence payload profile does not fit its promised retained-list bounds for legitimate schema-valid inputs/results. Resolve that contract gap before implementation. No missing generic runtime capability was established; no generic repair is authorized by this review. Task 2/3 implementation and qualification are not approved or claimed complete.

## Findings

### P2 — Universal 262144-byte lists cannot preserve all allowed evidence frontiers/results

Contract `docs/foundational/phase1h/01-adaptive-contract.md:17` fixes every list at 262144 canonical bytes; lines 27–33 require full remaining candidates up to sixteen, exact evidence responses, and copied questions up to 4096 bytes. The existing public evidence schema allows answer/message values up to 8192 bytes (`examples/evidence/evidence.go:20`, `:32`, `:105`). The helper's clarification/insufficient-evidence paths only require a nonempty message (`:121`, `:135–136`); NUL characters are valid UTF-8 and accepted by these schemas.

Two concrete counterexamples, using a bounded Python JSON-size probe matching the current exported tagged `schema.Value` representation (`schema/schema.go:14–19`):

1. Eight seeds IDs 1..8, each with a question consisting of 4096 NUL bytes, encode to **197601 bytes**, below the request list ceiling. Eight valid `insufficient_evidence` results derive sixteen copied-question candidate rows. Those rows encode to **398117 bytes**, above the candidate/remaining ceiling. The policy must retain that entire frontier and report duplicate IDs 4/5/7/8 before work-limit checking. Instead, the required Append fails on size before that decision can be represented.
2. With tiny question `q`, six valid clarification responses whose answer is 8192 NUL bytes produce **301105 bytes** of minimally sized result rows; five produce 250921. The six-item round is within the work bound and each leaf response is individually valid (49592 encoded bytes). It cannot finish the required round evidence/results collection.

The exact byte counts use short operation IDs; realistic IDs only enlarge them. These are size/source probes, not claims that a nonexistent adaptive implementation was run. The fundamental lower bounds are already decisive: 16 × 4096 × 6 = 393216 escaped question bytes, and 6 × 8192 × 6 = 294912 escaped answer bytes, before any record overhead.

This is enforced today: strings are checked by UTF-8 byte length (`schema/collections.go:247`), while lists count JSON bytes (`:256–289`), and Append validates the entire new list (`:294–313`). Independently, the complete runtime projection, including copied cursor snapshots, must fit the default 1 MiB `MaxDataBytes` (`internal/executor/program.go:95–100`, `:267–287`; snapshots are copied at `internal/executor/collections.go:170–202`). Per-list validation therefore would not prove the aggregate projection safe even after the immediate list problem is addressed.

Required correction: choose and document an enforceable byte-safe evidence application profile or retention design within the assignment's existing finite limits, preserving exact required evidence and complete remaining frontiers. Do not silently truncate values/frontiers, change leaf policy, raise runtime ceilings, or turn this into an unreviewed generic length/repair feature. Add a named Task 2 worst-byte test covering escaped strings, candidate16, exact response nesting, complete projection plus simultaneous round/total evidence/cursor snapshots, and serialized program bounds. `phase1h-plan.md:70–79` currently tests item counts but not these byte ceilings. The contract must identify what happens for otherwise schema-valid payloads outside any newly declared narrower profile. Merely noting that the small fixture fits does not repair the advertised public contract.

### P3 — Clarify wrapper status propagation

Contract line 55 says wrappers “faithfully forward presence, payload and status.” Existing ordinary wrappers forward the payload when present, then select Success only for child `succeeded`, otherwise Failure (`examples/worktree/worktree.go:120–123`). Public authoring exposes Success/Failure (`author/author.go:79–80`), not an operation that sets arbitrary terminal statuses. Thus, for example, an independently cancelled adaptive invocation can be observed as a failed enclosing wrapper; an admission rejection likewise does not make the wrapper itself rejected.

Clarify the intended guarantee as exact application payload/leaf statuses plus success-versus-nonsuccess propagation, with each enclosing scope retaining its own runtime status/reason. Root/ancestor cancellation may still produce the existing closure envelope. If exact arbitrary envelope-status equality is truly intended, it is a separate unproven capability, not supplied by the proposed copied wrapper. No runtime extension is needed for the narrower ordinary forwarding interpretation.

## Verified contract/API fit outside the findings

- Public `author.Definition` supports initial values, declared collections, invocations, ports and steps (`author/author.go:18–27`); ForEach/Append/Set/Add/IfEqual are public (`:62–78`). `Invoke`/`AwaitInvocation` expose operation handles and status/reason/presence/typed response bindings (`author/invocations.go:94–108`), and template schemas are public and copied (`:150–163`). Existing `compose.New` demonstrates all intents before ordered consumption and zero-or-one exact typed response lists (`compose/collection.go:76–103`). Reusing these ordinary steps inside adaptive avoids an extra invoked collection/coordinator depth.
- The compiler emits one ForEach body and structured cursor loop (`author/compile.go:214–228`); runtime cursor snapshots and iteration position survive as projection state (`internal/executor/collections.go:127–155`, `:170–207`). A three-row control list is ordinary finite durable iteration. Nested duplicate scans, counters, equality conditions and Add-derived integer IDs need no scheduler, concat, greater-than, host frontier builder or new runtime ledger.
- Integer seed IDs 1..8 and child IDs 3p+1/3p+2 are a documented restricted application profile. Three expansions from seed8 have maximum242, including the unperformed fourth-round frontier. Arithmetic cannot overflow inside that profile. The lack of public concat is real: Add rejects noninteger operands (`author/compile.go:462–468`). `UniqueKey` is restricted to string fields (`schema/collections.go:232–237`), so application integer duplicate scans are necessary. No generic repair is required for these restrictions.
- Numeric lineage is injective for distinct parent/slot pairs, but seed/descendant collisions are real: seeds2 and7 with observations1 and0 produce candidate7. The oracle correctly rejects the complete candidate frontier before any descendant item invocation. Eight observation2 seeds produce [4,5,7,8,10,11,13,14,16,17,19,20,22,23,25,26]; duplicate history correctly wins over the work limit, retaining all sixteen when their bytes fit. Sixteen is an item bound, not permission to admit more than eight.
- Complete barrier then first nonsuccess, invalid successful response, duplicate, exhaustion, work bound, round bound is coherent (`contract:43–51`, literal ledger `:1903–1911`). Successful siblings' candidates remain visible after known failure. Unknown required outcomes cannot be bypassed by early hard failure. Equality-before-increment can detect work overflow while scanning a bounded candidate list; overflow must remain sticky.
- A one-row decision Append is one ordinary committed transition; clearing the old row first is recoverable because normal history and instruction position remain durable. Store commits projection, selected outcome and intents transactionally before `execution.committed` (`internal/executor/store.go:800–849`). Individual Invoke updates retain their operation identity atomically (`author/compile.go:312–317`). Existing `execution.committed` and `invocation.admitted` (`internal/executor/invocation.go:533`) can acknowledge the three requested cuts without adding runtime hooks. The plan names actual SIGKILL/fresh-process tests and identity/charge comparisons (`phase1h-plan.md:88–95`); none has yet been executed for Phase1H.
- Depth and charges are sound: root0/adaptive1/leaf2 directly and root0/wrapper1/adaptive2/leaf3 nested, regardless of rounds. The evidence helper returns two ordinary members (`examples/evidence/evidence.go:162`) retained by `worktree.Evidence` (`examples/worktree/worktree.go:50–61`) in one template. N admitted leaves cost N+1/N+2 invocations, scalar N+1/N+2 runs or evidence 2N+1/2N+2 runs. At N8 nested evidence is10 invocations/18 runs/at most24 external reservations, within32/64/96; capacity1/2 affects concurrency only. The fixed root is separately provisioned. Admission checks count actual template members (`internal/executor/composition_prefix.go:158–164`); a rejected Invoke consumes no admitted invocation/member cost.
- Evidence mapping is consistent with unchanged helper policy: clarification and insufficient_evidence finish after one model request; no_match requires planning plus lookup; answer requires planning, lookup and synthesis (`examples/evidence/evidence.go:130–157`). Therefore clarification then answer costs4 reservations total, direct3 invocations/5 runs or nested4/6. Paired answer-only costs3 reservations; no_match costs2. Copying the question avoids the independent 8192-answer-to-4096-question mismatch but does not avoid the aggregate-byte finding above. Known failed invalid_result must be inspected before its diagnostic payload.
- Required unknown work remains pending/spent. Existing reopening coverage explicitly retains the projection, makes zero repeated calls and preserves the unknown reservation (`internal/executor/composition_limits_integration_test.go:452–500`); this is existing evidence, not new execution. Admission prefix checks fence source/scope closure (`composition_prefix.go:125–138`); successful publication requires known external results and descendant outcomes (`:219–237`). `runtime.Open`, Inspect and Replay are public (`runtime/runtime.go:43–54`). Replay opens read-only and has no dispatch capability (`internal/executor/replay.go:10–41`); Task3 separately requires the established OS network isolation, no credentials/registry, unchanged ledger and zero contacts. Unknowns/closure/replay and all three new crash cuts remain planned, not qualified.

## Task 1 artifacts and independent oracle

The exact change is documentation/evidence only (29 files); no Go, runtime, persistence or module implementation changed. The checkout was clean at reviewed head. The ledger is labelled pre-implementation and does not derive its expectations from application code, which does not yet exist. Its two recorded SHA256 identities match `expected-work.json` and `oracle.md`. The literal primary frontiers [1,2,3]/[7,10,11]/[31] and paired [1,2,3]/[7] have correct inputs/observations/parent ordering, seven/four leaves, charges8/9 plus7 effects and5/6 plus4 effects. Empty/all-zero/eight-total/duplicate/ancestry collision/limits/failure/invalid observation/denial/unknown/effectful cases are present and self-consistent for their compact payloads. IDs and operations are correctly distinguished; actual runtime IDs are required to remain injective and stable across recovery rather than invented ahead of execution. The genuine faulty paired variant must admit forbidden gamma descendants and fail the unchanged semantic oracle while remaining protocol-valid.

The broad generic `charges_for_each_n` table includes hypothetical evidence all-clarification/all-answer costs; these are cost formulas, not claims that those dispositions produce the scalar primary frontier. Tests should consume the explicit evidence cases for evidence semantics.

I verified all 2494 historical inventory entries against current bytes. The only changes are the three authorized current Phase1G documents. Their saved original copies exactly match `git show 4550e0f:<path>`. The historical/pending labels and supported direct CLI sequence are present beside nested usage. Frozen historical artifacts are untouched. Baseline command records identify actual source `ce4bbf6f...`, exits0 and764/1006/20 passing events. I verified each retained default/normal/consumer compressed log hash and decompressed raw hash against metadata. No new baseline race or extracted historical execution is claimed, and I did not rerun those suites.

Task 1 is a substantial, carefully scoped preparation slice. The remaining review gate is real: repair the byte-safe contract and clarify wrapper status wording, independently retain revised expectations as needed, then proceed to Task 2. This review neither validates future implementation nor approves expanding runtime/resource scope.

## Reproducible focused byte probe

Run the following read-only Python probe. It writes no application/source artifacts and imports no future adaptive builder. It reproduces the current exported `schema.Value` JSON field representation; ASCII and NUL are used so Python's compact encoding matches the relevant Go JSON bytes. The shorter operation IDs deliberately give the proposed design the benefit of a lower size.

```python
import json

def encoded_size(v):
    return len(json.dumps(v, separators=(',', ':'), ensure_ascii=False).encode())

def value(kind, x):
    return {
        'Kind': kind,
        'Number': x if kind == 'int' else 0,
        'Text': x if kind == 'string' else '',
        'Flag': x if kind == 'bool' else False,
    }

def s(x): return value('string', x)
def i(x): return value('int', x)

question = '\0' * 4096
seeds = [{'work_id': i(p), 'question': s(question)} for p in range(1, 9)]
candidates = [
    {'work_id': i(3*p+c), 'parent_id': i(p),
     'edge': s('left' if c == 1 else 'right'), 'round': i(2),
     'question': s(question)}
    for p in range(1, 9) for c in [1, 2]
]
response = {
    'disposition': s('clarification'), 'answer': s('\0' * 8192),
    'document': s(''), 'version': s(''), 'snippet': s(''),
    'start': i(0), 'end': i(0),
}
results = [
    {'work_id': i(p), 'parent_id': i(0), 'edge': s('seed'),
     'round': i(1), 'question': s('q'), 'operation': s('op' + str(p)),
     'status': s('succeeded'), 'reason': s('member_succeeded'),
     'responses': dict(value('list', None), List=[response])}
    for p in range(1, 7)
]
print(encoded_size(seeds))       # 197601
print(encoded_size(candidates))  # 398117
print(encoded_size(response))    # 49592
print(encoded_size(results[:5])) # 250921
print(encoded_size(results))     # 301105
```

No proposed remediation is preapproved by this report. A narrower question profile or a deliberate public result projection may be a local application solution, but it changes the currently advertised exact-response contract and needs explicit documentation, revised independent expectations, and a bound analysis/targeted test that includes complete projection state. Full authenticated leaf responses retained in existing outcome history can remain separately inspectable by original operation; that is distinct from forwarding full leaf answer payloads in the adaptive application's public result.
