# Independent validator and authoring-boundary review

**APPROVE** the reviewed validator, its controls and the relay implementation for the restricted source claim. This verdict covers inputs returned by the public author builders, the template closure named by the certificate and the explicit frame premises in the [transport argument](../02-transport.md). It supplies one of the pre-execution gates. The public relocation experiment has a separate verdict.

The question was whether an author-valid, certified artifact could distinguish the spelling of two paired original handles at the same private instruction under adequate limits. I examined the lowering of inputs, values and collections, simultaneous emit updates, typed payloads, successful End outputs, loop snapshots and exhausted edges, the optional invocation bindings and the binder's actual rewriting of message endpoints. I also looked for a way to substitute a callee with different code, initial data or schemas. I found no accepted counterexample.

* **Initial state and closure.** Artifact and Template keep private immutable serializations. The artifact check recomputes the canonical program bytes and ID under one of two fixed compiler/semantics pairs. For templates, RetainTemplate checks program identity and semantics plus the initial projection against the entire closed input record. Actual lexical defaults execute in set instructions. Every member of the checked singleton-template catalogue has the sole bound caller port. The catalogue traversal compares the declaration schema IDs and child IDs with the actual descriptors and covers all retained programs. A mutable Certificate struct is a description of that check. Reliance requires rechecking/matching the actual inputs.
* **Attacks on operand flow.** Each accepted program has at most one sort per register over its entire lifetime. At a merge, a back edge or exhaustion of an empty snapshot, a preserved register therefore cannot carry the wrong kind of value. Collections accept literal data recursively, and row binding comes from actual schema-validated snapshot data. Simultaneous invoke updates read the old store even if another update overwrites the source. Selection needs the original operation with its declaration and also checks the runtime structural fields. A receive binding cannot overwrite a handle with an optional reply string. Absent responses actually delete the bound reply registers. Equality between two handles is safe only because fresh commitments are distinct within a history and the growing pairing preserves aliases. Identically spelled ordinary strings remain in the data sort. The original Q escape, branch/reassignment and dependency controls exercise separate paths through these checks.
* **Rebinding and failure.** resolveCompositionProgram matches the member's original bytes to the bound descriptor, then rewrites declared ordinary message endpoints, port metadata and the caller-entry match. Application literals/payloads, output declarations and invocation-handle operands retain their interpretation. Representability, noncolliding bound identities, authentic service and adequate per-frame byte and numerical limits are substantive premises. I raised a distinction between precommit reduction faults and an unmatched receive. The revised proof at SHA-256 `bc9298dbf7dfe1e71953024733c959446e19fa0b7282006df515f8c90b60ba00` addresses it. Store.selectReady propagates a non-ineligibility error; a Store progress claim needs original correspondence covering faults, and the validator alone supplies no such premise. The four-mode claim still requires W, Progress and both reference/frame arguments.
* **Relay implementation.** Independent construction reproduced the original 22,926-byte O for A and B. Their SHA-256 digests were respectively `272c68dfe7c2b422c9819a8caf80b0277f35fb91859c9ddf80915cf46fdc4725` and `5e8fee04fd8855c501c1c20b87fdc4c2a1f96d6f4152baba3a7b5a48acb45621`, exactly the [Phase 1N preflight](../../phase1n/evidence/preflight/artifacts-and-samples.json). A separate pure-reducer probe started at the actual relay Await with selected, schema-valid envelopes. Absent evidence cleared three stale bindings. Absence, bad ID, bad complement, empty answer, bad status and bad reason each produced one reply (request ID, invalid_outcome, empty list) and failed End. A valid invalid_outcome report, valid answer and valid missing-reply diagnosis forwarded their data and took the expected End. This isolates the relay's private control and fallback. It bypasses authentication, store scheduling and publication, which are separate obligations.

I reran the following commands with Go 1.27.3 darwin/arm64 from the Phase 1R checkout, with `GOCACHE=/private/tmp/alan-phase1q-go-cache GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly`. All exited zero. I did not execute the gated relocation test.

```sh
go test -count=1 -timeout=3m -run '^TestNominal' ./research/nominal
go test -tags=integration -count=1 -timeout=3m -run '^(TestNominalAliasFreshExecution|TestNominalWrongAnswerCertifies|TestDiagnosticRelayHandleObservation)$' ./internal/substitutiontest
go test -count=1 -timeout=2m -overlay=/private/tmp/alan-r-validator-review/overlay.json -run '^TestReviewRelay$' -v ./examples/callablejoin
```

Reviewed source hashes (SHA-256, against freeze commit `8547d919062af6b3dd0d97e1fc7aa86967c78220`):

```text
75edb3f58e1c94928a4f7bcd4f9e867aceb9339fcaf393c4423de6b8324994df  research/nominal/certificate.go
c279f86260e1e245c9fa3bc11c0c3f8af1c8ac39f780f79937e363715b561657  research/nominal/operands.go
86cce651515c61ee00ae7082e30b5119708b9a36ee7a859de4f8912ed1095ac2  research/nominal/controls_test.go
6f41fd79af532ca6ae53c25a665a9484c445865cbc68063dcc6a3a0c9845e34a  internal/substitutiontest/nominal_controls_test.go
6aa1756d1159ae58a2e0fc1913323a4c900572287e7f7ea21b50fffc33a82382  examples/callablejoin/relay.go
c53b7866890e35ee8735c8f265465fcf7ab0460887961535487af6e34ed6345a  examples/callablejoin/outer.go
```

<details>
<summary>Independent pure-reducer probe for reproduction</summary>

The Go overlay maps an unused file name in the checkout, `examples/callablejoin/reviewer_only_test.go`, to the following file (both paths absolute in the overlay's Replace object). This is a private instruction test using a seeded state, not either of the public placements.

```go
package callablejoin_test

import (
	"encoding/json"
	"testing"

	"github.com/JakeFAU/alan/author"
	cj "github.com/JakeFAU/alan/examples/callablejoin"
	sub "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	ex "github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

func TestReviewRelay(t *testing.T) {
	q := sub.Config{Request: sub.Request{ID: 1, Value: 0, Mode: "answer"}}
	for _, tc := range []struct {
		build func() (author.Template, error)
		want  string
	}{
		{a.New, "272c68dfe7c2b422c9819a8caf80b0277f35fb91859c9ddf80915cf46fdc4725"},
		{b.New, "5e8fee04fd8855c501c1c20b87fdc4c2a1f96d6f4152baba3a7b5a48acb45621"},
	} {
		c, err := tc.build()
		if err != nil {
			t.Fatal(err)
		}
		d, err := cj.New(c, q)
		if err != nil {
			t.Fatal(err)
		}
		if d.Outer.ID() != tc.want || len(d.Outer.Bytes()) != 22926 {
			t.Fatal("O changed", d.Outer.ID())
		}
	}
	c, err := b.New()
	if err != nil {
		t.Fatal(err)
	}
	d, err := cj.New(c, q)
	if err != nil {
		t.Fatal(err)
	}
	relay, err := cj.NewRelay(d.Join)
	if err != nil {
		t.Fatal(err)
	}
	var bundle struct{ Programs []ex.Artifact }
	if err = json.Unmarshal(relay.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	var p ex.Program
	for _, a := range bundle.Programs {
		if err = json.Unmarshal(a.Bytes, &p); err != nil {
			t.Fatal(err)
		}
		if p.Author.Name == "phase1r-diagnostic-relay" {
			break
		}
	}
	pc := -1
	for i, in := range p.Code {
		if in.Op == "receive" && in.Match.Kind == "invocation" {
			pc = i
			break
		}
	}
	if pc < 0 {
		t.Fatal("missing Await")
	}
	for _, tc := range []struct {
		name, status, reason, tag string
		id, value                 int64
		present, one              bool
		want                      string
	}{
		{"absent", "failed", "missing_reply", "", 0, 0, false, false, "invalid_outcome"},
		{"bad-id", "succeeded", "member_succeeded", "answer", 2, 1, true, true, "invalid_outcome"},
		{"bad-answer", "succeeded", "member_succeeded", "answer", 1, 0, true, true, "invalid_outcome"},
		{"empty-answer", "succeeded", "member_succeeded", "answer", 1, 0, true, false, "invalid_outcome"},
		{"bad-status", "failed", "member_failed", "answer", 1, 1, true, true, "invalid_outcome"},
		{"bad-reason", "failed", "missing_reply", "candidate_failed", 1, 0, true, false, "invalid_outcome"},
		{"invalid-pass", "failed", "member_failed", "invalid_outcome", 1, 0, true, false, "invalid_outcome"},
		{"answer-pass", "succeeded", "member_succeeded", "answer", 1, 1, true, true, "answer"},
		{"diagnosis-pass", "failed", "member_failed", "candidate_missing_reply", 1, 0, true, false, "candidate_missing_reply"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := ex.RunID{Subject: "relay", Generation: 1}
			op := "[\u0022relay\u0022,1,3,0]"
			s := ex.Projection{PC: pc, Status: "open", Data: map[string]schema.Value{
				"request_id": schema.Int(1), "request_value": schema.Int(0), "empty_answers": schema.List(), "child_operation": schema.Text(op),
				"report_id": schema.Int(42), "report_answers": schema.List(), "report_classification": schema.Text("answer"),
			}}
			obs := ex.Observation{Run: run, Kind: "invocation", Provenance: "runtime-invocation", Source: d.Join.ID(), SourceGeneration: 1, Obligation: "child", Operation: op, Payload: map[string]schema.Value{
				"status": schema.Text(tc.status), "reason": schema.Text(tc.reason), "response_present": schema.Bool(tc.present),
			}}
			answers := schema.List()
			if tc.one {
				answers = schema.List(map[string]schema.Value{"id": schema.Int(tc.id), "value": schema.Int(tc.value)})
			}
			if tc.present {
				obs.Payload["response.id"] = schema.Int(tc.id)
				obs.Payload["response.classification"] = schema.Text(tc.tag)
				obs.Payload["response.answers"] = answers
			}
			var input *ex.Observation = &obs
			var messages []ex.Intent
			for i := 0; i < 2*len(p.Code) && s.Status == "open"; i++ {
				r, err := ex.Reduce(p, s, input, run, int64(i+4), ex.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				s = r.Projection
				messages = append(messages, r.Intents...)
				input = nil
				if i == 0 && !tc.present {
					for _, k := range []string{"report_id", "report_answers", "report_classification"} {
						if _, ok := s.Data[k]; ok {
							t.Fatal("stale after absence", k)
						}
					}
				}
			}
			wantAnswers := schema.List()
			wantStatus := "failed"
			if tc.want == "answer" {
				wantAnswers = answers
				wantStatus = "succeeded"
			}
			if s.Status != wantStatus || len(messages) != 1 || messages[0].Payload["id"] != schema.Int(1) || messages[0].Payload["classification"] != schema.Text(tc.want) || !messages[0].Payload["answers"].Equal(wantAnswers) {
				t.Fatal(s, messages)
			}
		})
	}
}
```
</details>
