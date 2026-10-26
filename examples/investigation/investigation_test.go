package investigation_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// These cases catch wrong review branches, lost earlier evidence, missing guards,
// and an unauthorized second revision. Provider decisions depend on actual inputs.
func TestInvestigationDecisions(t *testing.T) {
	for _, tc := range []struct {
		name, mode, want           string
		revisions, models, lookups int
	}{
		{"accept", "accept", "answer", 0, 7, 2},
		{"revise-accept", "revise", "answer", 1, 13, 4},
		{"second-revise", "again", "revision_limit", 1, 13, 4},
		{"disabled-revision", "revise", "revision_limit", 0, 7, 2},
		{"stop", "stop", "stopped", 0, 7, 2},
		{"clarify", "clarify", "clarification", 0, 1, 0},
		{"missing", "missing", "insufficient_evidence", 0, 4, 2},
		{"bad-plan", "bad-plan", "invalid_result", 0, 1, 0},
		{"empty-subquestion", "empty-plan", "invalid_result", 0, 1, 0},
		{"bad-citation", "bad-citation", "invalid_result", 0, 5, 2},
		{"bad-review", "bad-review", "invalid_result", 0, 7, 2},
		{"bad-review-query", "bad-review-query", "invalid_result", 0, 7, 2},
		{"empty-candidate", "empty-candidate", "invalid_result", 0, 6, 2},
		{"invalid-synthesis", "invalid-synthesis", "invalid_result", 0, 6, 2},
		{"false-support", "false-support", "insufficient_evidence", 0, 4, 2},
	} {
		for _, nested := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/nested=%t", tc.name, nested), func(t *testing.T) {
				c := investigation.DefaultConfig()
				c.Model = "controlled"
				c.Nested = nested
				if tc.name == "disabled-revision" {
					c.MaxRevisions = 0
				}
				a, _ := effects.NewAsset("packet", "fixture", []byte("alpha release supports Linux.\nbeta rollback requires version 2.\ngamma approval is recorded.\ndelta window is seven minutes.\n"))
				p, err := investigation.Provision("What is supported and when do we roll back?", c, []effects.Asset{a})
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "run.sqlite")
				s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				provider := &controlled{mode: tc.mode}
				_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 10000, MaxDispatch: 10000, MaxInFlight: 2, Adapters: map[string]effects.Adapter{evidence.ModelCapability: provider, evidence.LookupCapability: lookup.New()}})
				if err != nil {
					t.Fatal(err)
				}
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				v, err := investigation.Inspect(t.Context(), path)
				if err != nil {
					t.Fatal(err)
				}
				if nested {
					wantOutcome := "needs_attention"
					if tc.want == "answer" {
						wantOutcome = "publish"
					}
					if v.Result["report_outcome"].Text != wantOutcome {
						t.Fatalf("outer report outcome=%q want %q", v.Result["report_outcome"].Text, wantOutcome)
					}
				}
				if got := v.Result["disposition"].Text; got != tc.want {
					t.Fatalf("disposition=%q want %q", got, tc.want)
				}
				if v.Result["revision_count"].Number != int64(tc.revisions) || v.ModelRequests != tc.models || v.LookupRequests != tc.lookups {
					t.Fatalf("counts revision=%v model=%d lookup=%d", v.Result["revision_count"], v.ModelRequests, v.LookupRequests)
				}
				if tc.want == "answer" || tc.want == "revision_limit" || tc.want == "stopped" {
					if v.Result["initial_candidate"].Text != "Linux supported; rollback version 2." {
						t.Fatal("initial candidate not retained", v.Result)
					}
					if v.Result["review_reason"].Text == "" {
						t.Fatal("review reason missing")
					}
					for i := 1; i <= 2+tc.revisions*2; i++ {
						prefix := fmt.Sprintf("e%d_", i)
						start, end := v.Result[prefix+"start"].Number, v.Result[prefix+"end"].Number
						if v.Result[prefix+"document"].Text != a.ID || v.Result[prefix+"version"].Text != a.Version || start < 0 || end > int64(len(a.Content)) || end <= start || v.Result[prefix+"snippet"].Text != a.Content[start:end] {
							t.Fatal("citation provenance lost", prefix, v.Result)
						}
					}
					if tc.revisions == 1 && v.Result["final_candidate"].Text != "Linux supported; rollback version 2; approval recorded; seven minutes." {
						t.Fatal("revision did not use retained plus new evidence")
					}
				}
				if tc.want != "answer" && v.Result["answer"].Text != "" && tc.want != "clarification" && tc.want != "insufficient_evidence" {
					t.Fatal("non-answer presented as answer")
				}
				var modelSpent, lookupSpent int64
				for _, r := range v.Runs {
					data := r.Projection.Data
					modelSpent += data["model_spent"].Number
					lookupSpent += data["lookup_spent"].Number
					if data["model_spent"].Number > data["model_limit"].Number || data["lookup_spent"].Number > data["lookup_limit"].Number {
						t.Fatal("participant exceeded retained budget", r.ID)
					}
					if spent, ok := data["model_spent"]; ok && spent.Number > 0 {
						pending := false
						for k, val := range data {
							if strings.HasPrefix(k, "pending_") && val.Text != "" {
								pending = true
							}
						}
						if !pending {
							t.Fatal("spent counter without retained request identity", r.ID)
						}
					}
				}
				if modelSpent != int64(v.ModelRequests) || lookupSpent != int64(v.LookupRequests) {
					t.Fatal("ordinary counters differ from committed intents", modelSpent, lookupSpent)
				}
				if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
					t.Fatal(err)
				}
				t.Logf("INVESTIGATION {\"case\":%q,\"nested\":%t,\"disposition\":%q,\"revision_count\":%d,\"model_requests\":%d,\"lookup_requests\":%d}", tc.name, nested, tc.want, tc.revisions, v.ModelRequests, v.LookupRequests)
			})
		}
	}
}

type controlled struct {
	mu      sync.Mutex
	mode    string
	reviews int
}

func (a *controlled) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "controlled/v1", MaxResponseBytes: 65536}, nil
}
func (a *controlled) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	a.mu.Lock()
	defer a.mu.Unlock()
	r := p.Request
	in := r.Payload
	out := map[string]schema.Value{}
	if (a.mode == "unknown-helper" && r.Effect == "plan" && in["question"].Text == "alpha") || (a.mode == "unknown-review" && r.Effect == "review") || (a.mode == "unknown-review2" && r.Effect == "review" && a.reviews == 1) {
		return effects.Candidate{ID: r.ID, Key: r.Key, Disposition: "unknown"}
	}
	switch r.Effect {
	case "decompose":
		out = map[string]schema.Value{"decision": schema.Text("investigate"), "subquestion1": schema.Text("alpha"), "subquestion2": schema.Text("beta"), "message": schema.Text("")}
		if a.mode == "clarify" {
			out["decision"] = schema.Text("clarification")
			out["message"] = schema.Text("Which release?")
		}
		if a.mode == "bad-plan" {
			out["decision"] = schema.Text("shell")
		}
		if a.mode == "empty-plan" {
			out["subquestion2"] = schema.Text("")
		}
	case "plan":
		q := in["question"].Text
		if a.mode == "missing" || a.mode == "false-support" {
			q = "unfindable"
		}
		out = map[string]schema.Value{"decision": schema.Text("lookup"), "tool": schema.Text("lookup"), "query": schema.Text(q), "message": schema.Text("")}
	case "synthesize":
		out = map[string]schema.Value{"answer": schema.Text(in["snippet"].Text)}
		for _, k := range []string{"document", "version", "snippet", "start", "end"} {
			out[k] = in[k]
		}
		if a.mode == "bad-citation" {
			out["version"] = schema.Text("fabricated")
		}
	case "compose":
		candidate := "Linux supported; rollback version 2."
		if in["e1_snippet"].Text != "alpha release supports Linux." || in["e2_snippet"].Text != "beta rollback requires version 2." {
			candidate = "INVALID INPUT"
		}
		if candidate != "INVALID INPUT" && in["e3_snippet"].Text == "gamma approval is recorded." && in["e4_snippet"].Text == "delta window is seven minutes." {
			candidate = "Linux supported; rollback version 2; approval recorded; seven minutes."
		}
		out = map[string]schema.Value{"decision": schema.Text("answer"), "candidate": schema.Text(candidate)}
		if a.mode == "empty-candidate" {
			out["candidate"] = schema.Text("")
		}
		if a.mode == "invalid-synthesis" {
			out["decision"] = schema.Text("fabricated")
		}

		if a.mode == "missing" {
			out["decision"] = schema.Text("insufficient_evidence")
			out["candidate"] = schema.Text("The packet does not establish support.")
		}
	case "review":
		a.reviews++
		decision := "accept"
		missingRevisionFacts := in["e3_snippet"].Text != "gamma approval is recorded." || in["e4_snippet"].Text != "delta window is seven minutes."
		if ((a.mode == "revise" || a.mode == "unknown-review2") && missingRevisionFacts) || a.mode == "again" {
			decision = "revise"
		}
		if a.mode == "stop" {
			decision = "stop"
		}
		if in["candidate"].Text == "INVALID INPUT" || in["e1_snippet"].Text != "alpha release supports Linux." || in["e2_snippet"].Text != "beta rollback requires version 2." {
			decision = "stop"
		}
		out = map[string]schema.Value{"decision": schema.Text(decision), "reason": schema.Text("Checked candidate against supplied evidence."), "subquestion1": schema.Text("gamma"), "subquestion2": schema.Text("delta")}
		if a.mode == "bad-review" {
			out["reason"] = schema.Text("")
		}
		if a.mode == "bad-review-query" {
			out["decision"] = schema.Text("revise")
			out["subquestion1"] = schema.Text("")
		}
	default:
		panic("unexpected effect " + r.Effect)
	}
	return effects.Candidate{ID: r.ID, Key: r.Key, Disposition: "validated", Payload: out}
}

func TestPublicFactoryAndInvalidLimits(t *testing.T) {
	for _, limit := range []int{-1, 2} {
		if _, err := investigation.New("study", "client", "work", investigation.Budgets{MaxRevisions: limit}); err == nil {
			t.Fatal("accepted invalid revision limit", limit)
		}
	}
	for _, nested := range []bool{false, true} {
		component, err := investigation.New("study", "client", "work", investigation.Budgets{MaxRevisions: 1})
		if err != nil {
			t.Fatal(err)
		}
		if nested {
			component, err = investigation.Report("report", "client", "work", "study", investigation.Budgets{MaxRevisions: 1})
			if err != nil {
				t.Fatal(err)
			}
		}
		bindings := map[string]string{}
		for k := range investigation.ResultSchema() {
			bindings[k] = k
		}
		caller, err := author.Compile(author.Definition{Name: "external caller", Outputs: investigation.ResultSchema(), Ports: map[string]author.Port{"work": component.Port}, Steps: []author.Step{author.Send("work", map[string]author.Expr{"question": author.String("question")}), author.Receive("work", bindings), author.Success()}})
		if err != nil {
			t.Fatal(err)
		}
		component.Instances["client"] = author.Instance{Artifact: caller}
		if _, err = component.Instances.Provision(); err != nil {
			t.Fatal(err)
		}
	}
}

// An unresolved sibling/review remains the same identity on resume. Removing a
// send-before-receive or treating unknown as revision would fail this test.
func TestUnknownDoesNotRetryOrRevise(t *testing.T) {
	for _, tc := range []struct {
		mode            string
		models, lookups int
		revision        int64
	}{
		{"unknown-helper", 4, 1, 0}, {"unknown-review", 7, 2, 0}, {"unknown-review2", 13, 4, 1},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			c := investigation.DefaultConfig()
			c.Model = "controlled"
			asset, _ := effects.NewAsset("packet", "fixture", []byte("alpha release supports Linux.\nbeta rollback requires version 2.\ngamma approval is recorded.\ndelta window is seven minutes.\n"))
			p, err := investigation.Provision("question", c, []effects.Asset{asset})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "unknown.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			provider := &controlled{mode: tc.mode}
			identity := ""
			for attempt := 0; attempt < 2; attempt++ {
				if attempt == 1 {
					s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
					if err != nil {
						t.Fatal(err)
					}
				}
				_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 10000, MaxDispatch: 10000, MaxInFlight: 2, Adapters: map[string]effects.Adapter{evidence.ModelCapability: provider, evidence.LookupCapability: lookup.New()}})
				if err != nil {
					t.Fatal(err)
				}
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				v, err := investigation.Inspect(t.Context(), path)
				if err != nil {
					t.Fatal(err)
				}
				if v.Result != nil || len(v.Unknown) != 1 || v.ModelRequests != tc.models || v.LookupRequests != tc.lookups || v.Attempts != tc.models+tc.lookups {
					t.Fatalf("unknown state: result=%v unknown=%v model=%d lookup=%d attempts=%d", v.Result, v.Unknown, v.ModelRequests, v.LookupRequests, v.Attempts)
				}
				if identity != "" && identity != v.Unknown[0] {
					t.Fatal("unknown identity changed")
				}
				identity = v.Unknown[0]
				foundPending := false
				foundSibling := false
				for _, r := range v.Runs {
					if r.ID.Subject == "investigation" && r.Projection.Data["revision_count"].Number != tc.revision {
						t.Fatal("unknown initiated a revision")
					}
					if r.ID.Subject == "investigation/helper2" && r.Projection.Status == "succeeded" {
						foundSibling = true
					}
					for k, val := range r.Projection.Data {
						if strings.HasPrefix(k, "pending_") && val.Text == identity {
							foundPending = true
						}
					}
				}
				if !foundPending {
					t.Fatal("unknown lost pending identity")
				}
				if tc.mode == "unknown-helper" && !foundSibling {
					t.Fatal("sibling did not make independent progress")
				}
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			t.Logf("INVESTIGATION {\"case\":%q,\"unknown_id\":%q,\"model_requests\":%d,\"lookup_requests\":%d,\"revision_count\":%d}", tc.mode, identity, tc.models, tc.lookups, tc.revision)
		})
	}
}
