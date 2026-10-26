# Existing Phase 1E live gate closure

PASS on merged source `e2157a8fb2837ea3d7d0232683072182a0ee832b`, executable/input digest `8f95bda980504caec8c29cbe196297f61abafe7941d11ddc51effc19c9321caa`. This is new Phase 1F evidence from the unchanged Phase 1E application. Earlier published NOT RUN evidence remains unchanged.

The user supplied a rotated project credential and authorized a $10 total budget for this phase. The selected, account-available API model is `gpt-6.1-sol`. The direct adapter contacted the official Responses endpoint; no controlled response was substituted. Configuration selected maximum three logical model requests, two lookups, 4,096 output tokens per request, 65,536 response bytes and a 120-second request timeout. This one-shot helper can actually make at most two model calls and one lookup. A conservative $2 reservation preceded inference. No unsuccessful or unknown operation was repeated.

Question: “How does WAL work?” The final answer was “SQLite WAL works by recording changes in a write-ahead log before updating the main database.” Its citation is the exact retained `durability` document bytes [0,82), with content version `c2d44001abf76e7220188c766c6ca6b4080dd56afbff64ac2bcc48510af84437`. The cited statement entails this answer; this single simple question is a live functionality gate, not evidence of general answer quality.

Both model results were validated HTTP 200 responses. Planning request `req_e1a34f470b54469fb58c15c7c01650b6` / response `resp_0e0579781ccf4f6c016adf577590ac87d28fe79d83c765fb73` used 168 input and 135 output tokens. Synthesis request `req_9aeb4c3142ae476eb572184e57e50584` / response `resp_0ab267d249500f72016adf577c87b087d2a50c6331c34006eb` used 233 input and 130 output tokens. Both returned `gpt-6.1-sol`, service tier `default`, and zero cached/cache-write tokens. Total 401 input +265 output =666 tokens. The run took 11.891 seconds, including executor/provider work. No unknown outcomes or denied attempts remained.

At the published standard rates of $2/M input and $10/M output, the token-cost estimate is **$0.003452**, not an invoice or account billing verification. Prices/model behavior can change; this is the selected mutable API alias, not a claimed immutable provider model artifact. Primary references checked for this run: [model documentation](https://developers.openai.com/api/docs/models/gpt-6.1-sol) and [pricing](https://developers.openai.com/api/docs/pricing). Local program/artifact identities are immutable independently of the provider alias.

## Replay and retained evidence

Replay exit status0 with `OPENAI_API_KEY` absent and macOS sandbox profile `(version 1) (allow default) (deny network*)`. A socket control under the same profile returned EPERM, confirming effective network denial. Exact inspection before/after replay matched, including lifecycle, intents, attempts and adapter outcomes. Replay constructs no adapter registry. This establishes historical verification of this retained run, not live recovery or a general operating-system sandbox claim.

`result.json` records full outcome, IDs, usage, replay control and binary/store hashes. `commands.json` records actual commands/exits/times. `inspect.json`, `run.json`, `replay.json` and `inspect-after-replay.json` retain public output and records. `store.sqlite` is a read-only SQLite backup of the completed store, including exact local programs/assets/provider outcomes. Original raw output and DB files were scanned for literal and encoded forms of the supplied credential before publication; none were found. Credentials were read only for execution and never stored in configuration.

Reproduction uses a new store and incurs new billable work, requiring an explicit model, credential and spending cap. With an authorized key in the environment:

```sh
go build -o /tmp/alan-evidence ./cmd/alan-evidence
/tmp/alan-evidence init --db /tmp/alan-live-new.sqlite --question 'How does WAL work?' --model gpt-6.1-sol --max-model-requests 3 --max-lookup-requests 2 --max-output-tokens 4096 --max-response-bytes 65536 --request-timeout 120s
/tmp/alan-evidence run --db /tmp/alan-live-new.sqlite
/tmp/alan-evidence inspect --db /tmp/alan-live-new.sqlite
env -u OPENAI_API_KEY /usr/bin/sandbox-exec -p '(version 1) (allow default) (deny network*)' /tmp/alan-evidence replay --db /tmp/alan-live-new.sqlite
```

To verify the retained result offline, use the same replay command with `docs/foundational/phase1f/evidence/live-phase1e/store.sqlite`. No credential, live call or fresh store is required. Other operating systems should use their own network-isolation mechanism; the retained command and EPERM control qualify macOS only.
