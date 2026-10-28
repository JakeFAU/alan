# Phase 1H contract fix round 1 — P2/P3

Status: documentation fix complete; independent scoped rereview required before Task2. Original contract/plan and full findings/probe remain immutable at `docs/foundational/phase1h/evidence/{contract-v0.md,plan-v0.md,contract-review.md}`. This fix changes no application/runtime/Go/module/provider code and runs no repeated Go suites.

P2 accepted as a real counterexample. The original4096NUL request may pass a4096-byte question schema yet its16 copied candidates exceed262144 bytes; full8192NUL response accumulation independently fails. The controller's application-local repair narrows adaptive question fields to512 UTF-8 bytes, rejects over-profile input before itemwork, and summarizes exact evidence disposition instead of copying full answers/citations. Full authenticated outcomes remain retained and inspectable by original operation. Scalar outcomes still carry actual observation. No truncation, helper policy change or runtime ceiling increase.

Read `author/invocations.go:94-108`, `author/compile.go:599-620`, and `internal/executor/program.go:412-422`: public outcome Response bindings are selective; compiler validates requested keys/types without requiring all fields and reducer copies only selected fields. No concat/length/new runtime operation is required. The concrete interpretation of schema strings vs reachable producer strings is explicit: generated invoked-member subjects are SHA256hex64 (`composition_types.go:120-122`, `program.go:117`), intent IDs are tuple JSON (`program.go:437-438`), and outcome reason sources are finite literals (`invocation.go:298-342`, `messages.go:154-165`, `invocation_outcome.go:375-436`, `invocation_closure.go:120`). Reason is not provider text.

Retained read-only Python tagged-Value probe: a 19-digit sequence gives92-byte operation /96-byte encoded JSON string; longest relevant prefixed reason is39 bytes. With512NUL questions, largest observed candidate row3378 bytes, full16 candidate array54053 bytes; eight disposition-only summary rows30929 bytes, largest3866 bytes. A maximally escaped legal helper response is65373 bytes. These numbers are source/schema calculations, not execution of a future adaptive program. Conservative row caps3968/5500 and the full621KiB adaptive state inventory include retained lists, three simultaneous data-bearing cursor snapshots, scratch/current bindings and metadata; actual complete projections and program/observation/output bytes remain named Task2 acceptance tests. A failed gate requires an application-local redesign and independent review, not a raised limit.

P3 corrected: exact application payload and recorded leaf status/reason are forwarded when present; child success maps to ordinary wrapper Success, other child outcomes to Failure unless ancestor closure determines its own envelope. A wrapper does not set arbitrary child cancelled/rejected status. Each scope retains its own reason/status. Task2 now names explicit scope-mapping assertions.

Validation performed: inspected the stated public/compiler/runtime sources; ran the focused Python size probe; `git diff --check` passed. No implementation, qualified runtime behavior, race result or existing suite rerun is claimed. The independent oracle author must review/update expectations separately; this commit excludes frozen oracle changes.

## Reproduce the focused calculation

This is an upper-size tagged-value calculation. The long reason paired with succeeded status and maximal citation fields paired with clarification are conservative schema-size combinations, not claims that the unchanged helper produces those semantic combinations. Task2 exercises actual reachable helper paths separately.

```python
import json

def size(x):
    return len(json.dumps(x, separators=(',', ':'), ensure_ascii=False).encode())

def value(kind, x):
    return {'Kind': kind, 'Number': x if kind == 'int' else 0,
            'Text': x if kind == 'string' else '', 'Flag': False}

def s(x): return value('string', x)
def i(x): return value('int', x)
def listing(x): return dict(value('list', None), List=x)

question = '\0' * 512
operation = json.dumps(['f' * 64, 1, 9223372036854775807, 0], separators=(',', ':'))
reason = 'source_denied:source_generation_changed'
frontier = [dict(work_id=i(3*p+c), parent_id=i(p),
                 edge=s('left' if c == 1 else 'right'), round=i(4), question=s(question))
            for p in range(1, 9) for c in (1, 2)]
results = [dict(frontier[j], operation=s(operation), status=s('succeeded'), reason=s(reason),
                responses=listing([{'disposition': s('insufficient_evidence')}]))
           for j in range(8)]
response = {k: s('\0' * n) for k, n in [('answer', 8192), ('document', 512),
                                       ('version', 64), ('snippet', 2048)]}
response.update(disposition=s('clarification'), start=i(-9223372036854775808),
                end=i(9223372036854775807))
print(json.dumps({'operation_bytes': len(operation), 'operation_encoded_bytes': size(operation),
                 'reason_bytes': len(reason), 'frontier_row_bytes': max(map(size, frontier)),
                 'candidate16_bytes': size(frontier), 'summary_result_row_bytes': max(map(size, results)),
                 'results8_bytes': size(results), 'full_helper_response_bytes': size(response),
                 'inventory_kib': sum([32, 128, 65, 8, 2, 96, 2, 192, 64, 32])}, indent=2))
```

Observed output:

```json
{
  "operation_bytes": 92,
  "operation_encoded_bytes": 96,
  "reason_bytes": 39,
  "frontier_row_bytes": 3378,
  "candidate16_bytes": 54053,
  "summary_result_row_bytes": 3866,
  "results8_bytes": 30929,
  "full_helper_response_bytes": 65373,
  "inventory_kib": 621
}
```
