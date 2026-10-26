package investigation_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Identical lexical queries initially select the earlier tied spans. Only
// ordinary revision exclusions permit the later, previously unseen facts.
func TestRevisionExcludesExactlyPublicInitialSpans(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%t", partial), func(t *testing.T) {
			asset, err := effects.NewAsset("packet", "fixture", []byte("shared x common header\nshared y known scope\nshared x new safety fact\nshared y new rollback fact\n"))
			if err != nil {
				t.Fatal(err)
			}
			c := investigation.DefaultConfig()
			c.Model = "controlled"
			c.Nested = true
			provision, err := investigation.Provision("Find both missing facts.", c, []effects.Asset{asset})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "excluding.sqlite")
			store, err := runtime.Create(t.Context(), path, provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.Run(t.Context(), runtime.RunOptions{MaxSteps: 10000, MaxDispatch: 10000, MaxInFlight: 2, Adapters: map[string]effects.Adapter{investigation.ModelCapability: noveltyModel{partial: partial}, investigation.LookupCapability: lookup.New()}})
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			view, err := investigation.Inspect(t.Context(), path)
			if err != nil {
				t.Fatal(err)
			}
			wantModels := 13
			if partial {
				wantModels = 12
			}
			if view.Result["disposition"].Text != "answer" || view.Result["revision_count"].Number != 1 || view.ModelRequests != wantModels || view.LookupRequests != 4 {
				t.Fatalf("ordinary revision failed: result=%v models=%d lookups=%d", view.Result, view.ModelRequests, view.LookupRequests)
			}
			wantThird := "shared x new safety fact"
			if partial {
				wantThird = "shared x common header"
			}
			if view.Result["e3_snippet"].Text != wantThird || view.Result["e4_snippet"].Text != "shared y new rollback fact" {
				t.Fatal("revision did not acquire next eligible evidence", view.Result)
			}
			if view.Result["e2_snippet"].Text != "shared y known scope" {
				t.Fatal("initial evidence overwritten")
			}
			seen := map[string]effects.Request{}
			pending := map[string]string{}
			for _, run := range view.Runs {
				if strings.HasSuffix(run.ID.Subject, "/retrieval") {
					pending[run.ID.Subject] = run.Projection.Data["pending_lookup"].Text
					if run.Projection.Data["lookup_spent"].Number != 1 {
						t.Fatal("lookup counter not durable")
					}
				}
			}
			for _, row := range view.Snapshot.Tables["adapter_attempts"] {
				var attempt runtime.AdapterAttempt
				if err = json.Unmarshal([]byte(row["descriptor"].(string)), &attempt); err != nil {
					t.Fatal(err)
				}
				r := attempt.Prepared.Request
				if r.Capability != investigation.LookupCapability {
					continue
				}
				var id []any
				if err = json.Unmarshal([]byte(r.ID), &id); err != nil {
					t.Fatal(err)
				}
				subject := id[0].(string)
				if r.ID != pending[subject] {
					t.Fatal("pending lookup lost committed identity")
				}
				seen[subject] = r
				revised := strings.Contains(subject, "helper3/") || strings.Contains(subject, "helper4/")
				if !revised {
					if len(r.Payload) != 1 || attempt.Prepared.Protocol != "lookup/tokens/v1" {
						t.Fatal("initial helper contract changed")
					}
					continue
				}
				expectedSchema, _ := lookup.ExcludingInputSchema().Retain()
				if r.Input.ID != expectedSchema.ID || attempt.Prepared.Protocol != "lookup/tokens-excluding/v1" {
					t.Fatal("revision omitted explicit profile")
				}
				for i := 1; i <= 2; i++ {
					for _, field := range []string{"document", "version", "start", "end"} {
						expected := view.Result[fmt.Sprintf("e%d_%s", i, field)]
						if view.Result[fmt.Sprintf("e%d_disposition", i)].Text != "answer" {
							expected = schema.Value{Kind: expected.Kind}
						}
						if r.Payload[fmt.Sprintf("exclude%d_%s", i, field)] != expected {
							t.Fatal("revision input is not the exact public initial tuple", r.Payload)
						}
					}
				}
			}
			if len(seen) != 4 {
				t.Fatal("wrong lookup identities", len(seen))
			}
			if !partial {
				for _, pair := range [][2]string{{"investigation/helper1/retrieval", "investigation/helper3/retrieval"}, {"investigation/helper2/retrieval", "investigation/helper4/retrieval"}} {
					if seen[pair[0]].Payload["query"] != seen[pair[1]].Payload["query"] || seen[pair[0]].ID == seen[pair[1]].ID {
						t.Fatal("not a fresh bounded effect with the same lexical query")
					}
				}
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			record, _ := json.Marshal(map[string]any{"case": "revision-exclusions", "partial_initial": partial, "lookups": seen, "result": view.Result, "model_requests": view.ModelRequests, "lookup_requests": view.LookupRequests})
			t.Logf("INVESTIGATION %s", record)
		})
	}
}

type noveltyModel struct{ partial bool }

func (noveltyModel) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "controlled/v1", MaxResponseBytes: 65536}, nil
}
func (m noveltyModel) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	r := p.Request
	in := r.Payload
	out := map[string]schema.Value{}
	switch r.Effect {
	case "decompose":
		q := "shared x"
		if m.partial {
			q = "unfindable"
		}
		out = map[string]schema.Value{"decision": schema.Text("investigate"), "subquestion1": schema.Text(q), "subquestion2": schema.Text("shared y"), "message": schema.Text("")}
	case "plan":
		out = map[string]schema.Value{"decision": schema.Text("lookup"), "tool": schema.Text("lookup"), "query": in["question"], "message": schema.Text("")}
	case "synthesize":
		out["answer"] = in["snippet"]
		for _, k := range []string{"document", "version", "snippet", "start", "end"} {
			out[k] = in[k]
		}
	case "compose":
		candidate := "Initial evidence lacks the requested facts."
		if in["e3_disposition"].Text != "unused" {
			candidate = "Repeated evidence still lacks facts."
			want := "shared x new safety fact"
			if m.partial {
				want = "shared x common header"
			}
			if in["e3_snippet"].Text == want && in["e4_snippet"].Text == "shared y new rollback fact" {
				candidate = "New evidence acquired from distinct eligible spans."
			}
		}
		out = map[string]schema.Value{"decision": schema.Text("answer"), "candidate": schema.Text(candidate)}
	case "review":
		decision := "revise"
		if in["candidate"].Text == "New evidence acquired from distinct eligible spans." {
			decision = "accept"
		}
		out = map[string]schema.Value{"decision": schema.Text(decision), "reason": schema.Text("Check the missing facts in distinct evidence."), "subquestion1": schema.Text("shared x"), "subquestion2": schema.Text("shared y")}
	default:
		panic("unexpected effect " + r.Effect)
	}
	return effects.Candidate{ID: r.ID, Key: r.Key, Disposition: "validated", Payload: out}
}
