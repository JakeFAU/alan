# ALAN Phase 1E: OpenAI Responses adapter notes

Checked 2026-10-25 against current official OpenAI documentation. This is read-only protocol research for implementation; no credentials or live request were used.

## Recommended narrow design

Use one nonstreaming `POST https://api.openai.com/v1/responses` per ALAN model effect. A thin `net/http` client is sufficient for this fixed request/response subset and makes the single physical attempt auditable. Resolve an explicitly configured model ID, API key, and fixed endpoint from deployment configuration; no model-chosen URL/header, SDK orchestration, or remote conversation. The effect request retains the full instructions, actual input text (including retained tool evidence on synthesis), selected model, schema/version, and all selected generation parameters. Keep the adapter's output a completion candidate; ALAN persists and validates it before ordinary program branching. This is a design recommendation grounded in the [Responses create reference](https://developers.openai.com/api/reference/resources/responses/methods/create) and the [structured-output guide](https://developers.openai.com/api/docs/guides/structured-outputs).

For the first evidence assistant, prefer structured **text** over native function calling: the model returns a bounded JSON *proposal* such as `{kind: "lookup", query: "..."}` or `{kind: "clarify", ...}`. ALAN validates the schema and action-specific fields, commits that observation, and separately issues its own authorized lookup effect. The final synthesis call uses another strict text schema and explicit retained evidence. This keeps model/tool/model as separate durable operations. The docs recommend native [function calling](https://developers.openai.com/api/docs/guides/function-calling) when connecting models to functions; it is also viable, but then the adapter must parse `output[]` `function_call` items, preserve `call_id`, handle zero/multiple calls (or set `parallel_tool_calls:false`), and construct a subsequent model input containing the call and a matching `function_call_output`. For this ALAN increment, structured text has less protocol state and does not grant execution authority to provider tool calls. This tradeoff is an implementation inference, not a claim that function calling is unsupported.

## Exact request envelope

Minimal JSON shape (the model value is deployment-provided; the schema shown is illustrative, not a new fixed runtime opcode):

```json
{
  "model": "<explicit configured model ID>",
  "instructions": "<retained developer instructions>",
  "input": [{"role":"user","content":"<retained question and stage input>"}],
  "max_output_tokens": 512,
  "store": false,
  "truncation": "disabled",
  "text": {"format": {
    "type": "json_schema",
    "name": "alan_lookup_decision_v1",
    "strict": true,
    "schema": {
      "type": "object",
      "properties": {
        "kind": {"type":"string","enum":["lookup","clarify","insufficient_evidence"]},
        "query": {"type":["string","null"]},
        "message": {"type":["string","null"]}
      },
      "required": ["kind","query","message"],
      "additionalProperties": false
    }
  }}
}
```

Send `Authorization: Bearer <runtime credential>` and `Content-Type: application/json`; optionally send one unique ASCII `X-Client-Request-Id` (<=512 characters) **per physical attempt** for tracing. Do not treat that header as an idempotency key. `stream` and `background` can be omitted (or set false), and `previous_response_id`/conversation IDs are omitted so ALAN retains the whole model-visible input itself. The `store:false` setting avoids depending on provider retrieval; it is not a deduplication mechanism. `max_output_tokens` includes visible and reasoning tokens, so a too-small bound can yield `status:"incomplete"`. The [structured-output REST example](https://developers.openai.com/api/docs/guides/structured-outputs) shows `text.format` with `type`, `name`, `schema`, `strict`; the [create reference](https://developers.openai.com/api/reference/resources/responses/methods/create) specifies `instructions`, `input`, `max_output_tokens`, `store`, and `truncation`.

Keep schemas inside the documented strict subset: object root (not root `anyOf`), every property required, `additionalProperties:false` on every object, nullable union for optional semantics, and no unsupported JSON Schema composition keywords. Have a distinct stable schema identity for planning and final answer. Independently check semantic bounds (query length, known document ID, exact snippet/offset, citation membership), since schema adherence alone does not prove evidence accuracy. [Structured-output schema rules](https://developers.openai.com/api/docs/guides/structured-outputs).

## Decode and classify one response

Bound the HTTP body and retain the bounded wire response/body, HTTP status, `x-request-id` header if present, client request ID, and timing/attempt identity without authorization data. For a 2xx response, decode the JSON envelope and retain `id`, returned `model`, `status`, `error`, `incomplete_details`, `output`, and `usage` (if available). The response output is an array: inspect assistant `message` content for `output_text` versus `refusal`; do not assume output position zero is text (reasoning and tool items can appear). For the narrow text-schema adapter, require `status:"completed"`, no refusal, one unambiguous output-text result, valid JSON, and local schema plus program validation before publishing success. Retain anomalous or additional output items as evidence and classify them explicitly rather than silently concatenating or ignoring them. [Responses create reference](https://developers.openai.com/api/reference/resources/responses/methods/create), [structured-output edge cases](https://developers.openai.com/api/docs/guides/structured-outputs).

The failure/uncertainty distinctions to preserve are:

| Observed state | ALAN disposition |
|---|---|
| Completed response **durably recorded** | Reuse it on recovery/replay; no provider contact. |
| Received response before durable commit | Candidate only; if process dies first, recovery cannot infer completion from local state. |
| HTTP error with response body/status (e.g. rejected schema/auth/rate limit), or response `status:"failed"`/`error` | Retain provider error classification and request ID. Do not turn into a valid model decision. For 5xx, avoid asserting no remote work or billing occurred. |
| `status:"incomplete"` and `incomplete_details.reason` (e.g. `max_output_tokens`, `content_filter`) | Retain partial output and reason; never consume as schema success. |
| `refusal` content | Retain refusal and return explicit non-success disposition even if envelope says completed. Refusal may not match schema. |
| Completed but missing/ambiguous text, malformed JSON, local schema or semantic validation failure | Retain raw observation and validation error; ordinary program chooses explicit failure/repair branch. |
| Dispatch sent, then timeout/EOF/connection loss/cancellation before durable result | Unknown remote outcome. Preserve uncertainty; no automatic retransmission as historical replay. |

Capture actual `usage.input_tokens`, `output_tokens`, `total_tokens` and nested details when present; use optional fields so absent usage remains unknown. Store the requested model and returned `model` separately. `x-request-id` is an HTTP **header**, separate from JSON response `id`; the [API overview](https://developers.openai.com/api/reference/overview) recommends request IDs for troubleshooting and explains `X-Client-Request-Id` for cases with no server header. Do not infer cost from token limits or unobserved attempts.

## Retry, secrecy, fixture implications

Use exactly one `http.Client.Do` for an ALAN physical attempt with a context deadline and no adapter retry loop. A Go SDK alternative must set `option.WithMaxRetries(0)` because its [Go reference](https://developers.openai.com/api/reference/go) states it retries connection errors and HTTP 408/409/429/5xx twice by default. Neither low temperature nor the client request ID establishes idempotence. A deliberate repeat after an unknown outcome is a **new counted physical attempt**, explicitly selected by policy, with possible duplicate remote work. A model repair is another visible bounded model effect.

Construct requests from structured values rather than logging raw `http.Request`; never persist bearer tokens, auth headers, SDK error dumps, or URLs containing credentials. The [Go SDK error guidance](https://developers.openai.com/api/reference/go) warns raw diagnostics can expose headers and bodies. Keep the endpoint fixed by trusted runtime configuration (production OpenAI HTTPS; controlled loopback only in tests), reject redirects or otherwise constrain them to the trusted origin, and make the transport injection point test-only/configuration-only. Store bounded response bytes and a sanitized error summary so replay and inspection preserve decisions without secrets.

Controlled HTTP fixtures should exercise the real serializer/parser with: successful lookup proposal, clarification and insufficient-evidence branches; final answer with valid and invalid citations; refusal; incomplete max-token and content-filter responses; malformed or schema-invalid text; `failed`/`error`; 400/401/429/500 bodies and request ID headers; missing usage; returned model different from requested; extra/reordered output items; lost response after server acceptance; and a failpoint after received response but before durable commit. Assert body `model`, `instructions`, `input`, `text.format`, `max_output_tokens`, no provider tools/previous-response state, no credentials in retained evidence, no hidden retries, no model-triggered lookup before ALAN commits and validates its decision, and zero provider contacts on offline replay. This fixture list follows the project spec plus the documented envelope/edge cases; it is a qualification recommendation, not a list of provider guarantees.

## Sources

- [OpenAI Structured model outputs](https://developers.openai.com/api/docs/guides/structured-outputs), checked 2026-10-25.
- [OpenAI Function calling](https://developers.openai.com/api/docs/guides/function-calling), checked 2026-10-25.
- [OpenAI Responses create reference](https://developers.openai.com/api/reference/resources/responses/methods/create), checked 2026-10-25.
- [OpenAI Go API reference](https://developers.openai.com/api/reference/go), checked 2026-10-25.
- [OpenAI API overview / debugging requests](https://developers.openai.com/api/reference/overview), checked 2026-10-25.
