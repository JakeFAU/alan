//go:build integration

package worktree_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// These are real controlled Responses transports through the existing adapter.
// Channels acknowledge transport entry/release and durable result recording;
// timeouts detect hangs but never establish causal order.
type provider struct {
	mu                     sync.Mutex
	active, peak, contacts int
	held                   chan struct{}
	release                chan struct{}
	entered                sync.Once
	releaseOnce            sync.Once
	mode                   string
}

func (p *provider) unblock() { p.releaseOnce.Do(func() { close(p.release) }) }
func (p *provider) counts() (int, int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active, p.peak, p.contacts
}
func (p *provider) serve(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.active++
	p.contacts++
	if p.active > p.peak {
		p.peak = p.active
	}
	p.mu.Unlock()
	defer func() { p.mu.Lock(); p.active--; p.mu.Unlock() }()
	var request struct{ Input []struct{ Content string } }
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if len(request.Input) != 1 {
		http.Error(w, "missing request", 400)
		return
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(request.Input[0].Content), &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	question, _ := input["question"].(string)
	if question == "hold" {
		p.entered.Do(func() { close(p.held) })
		select {
		case <-p.release:
		case <-r.Context().Done():
			return
		}
	}
	if p.mode == "unknown" && question == "unknown" {
		connection, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			connection.Close()
		}
		return
	}
	result := map[string]any{"decision": "clarification", "tool": "", "query": "", "message": "clarify:" + question}
	if p.mode == "answer" || p.mode == "invalid" {
		result = map[string]any{"decision": "lookup", "tool": "lookup", "query": "WAL records changes", "message": ""}
		if _, synthesis := input["snippet"]; synthesis {
			result = map[string]any{"answer": "WAL records changes before updating the main database.", "document": input["document"], "version": input["version"], "snippet": input["snippet"], "start": input["start"], "end": input["end"]}
			if p.mode == "invalid" {
				result["snippet"] = "invented evidence"
			}
		}
	}
	raw, _ := json.Marshal(result)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "controlled", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(raw)}}}}})
}
func controlled(t *testing.T, p *provider) (map[string]effects.Adapter, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(p.serve))
	adapter, err := openai.New(openai.Config{APIKey: "phase1g-controlled-only-key", Model: "controlled", Endpoint: server.URL, MaxOutputTokens: 1024, MaxResponseBytes: 65536})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]effects.Adapter{evidence.ModelCapability: adapter, evidence.LookupCapability: lookup.New()}, server.Close
}
func questions(values ...string) schema.Value {
	rows := make([]map[string]schema.Value, len(values))
	for i, value := range values {
		rows[i] = map[string]schema.Value{"item_id": schema.Text(fmt.Sprintf("question-%d", i)), "question": schema.Text(value)}
	}
	return schema.List(rows...)
}
func createEvidence(t *testing.T, nested bool, capacity int64, limits *runtime.ResourceLimits) (*runtime.Store, string, worktree.Deployment) {
	t.Helper()
	leaf, err := worktree.Evidence(evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
	if err != nil {
		t.Fatal(err)
	}
	l := worktree.DefaultLimits(capacity)
	if limits != nil {
		l = *limits
	}
	d, err := worktree.New(leaf, nested, l)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := effects.NewAsset("durability", "controlled", []byte("WAL records changes before updating the main database.\n"))
	if err != nil {
		t.Fatal(err)
	}
	d.Provision.Assets = []effects.Asset{asset}
	path := filepath.Join(t.TempDir(), "evidence.sqlite")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path, d
}
func submit(t *testing.T, s *runtime.Store, items schema.Value) {
	t.Helper()
	a, err := worktree.Submit(t.Context(), s, "questions", items)
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
}
func rootRows(t *testing.T, s *runtime.Store) []map[string]schema.Value {
	t.Helper()
	state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := state.Projection.Data["results"].Records()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
func leafOutcomes(snapshot runtime.Inspection, leafID string) (out []runtime.InvocationOutcome, err error) {
	for _, row := range snapshot.Tables["composition_results"] {
		var result runtime.InvocationOutcome
		raw, ok := row["descriptor"].(string)
		if !ok {
			return nil, fmt.Errorf("result descriptor absent")
		}
		if err = json.Unmarshal([]byte(raw), &result); err != nil {
			return nil, err
		}
		if result.TemplateID == leafID {
			out = append(out, result)
		}
	}
	return out, nil
}

func TestEvidenceCapacityAndReverseCompletion(t *testing.T) {
	for _, capacity := range []int64{1, 2} {
		for _, nested := range []bool{false, true} {
			t.Run(fmt.Sprintf("capacity=%d/nested=%v", capacity, nested), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
				defer cancel()
				p := &provider{held: make(chan struct{}), release: make(chan struct{})}
				registry, closeServer := controlled(t, p)
				defer closeServer()
				defer p.unblock()
				s, path, d := createEvidence(t, nested, capacity, nil)
				submit(t, s, questions("hold", "later", "third"))
				laterRecorded := make(chan struct{})
				allAdmitted := make(chan struct{})
				var admittedOnce sync.Once
				var once sync.Once
				observer := boundary.WithObserver(ctx, func(ctx context.Context, event boundary.Event) error {
					if event.Boundary != "invocation.result-recorded" && event.Boundary != "invocation.admitted" {
						return nil
					}
					snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
					if err != nil {
						return err
					}
					expected := 4
					if nested {
						expected = 5
					}
					if len(snapshot.Tables["composition_invocations"]) == expected {
						admittedOnce.Do(func() { close(allAdmitted) })
					}
					outcomes, err := leafOutcomes(snapshot, d.Leaf.ID())
					if err != nil {
						return err
					}
					for _, outcome := range outcomes {
						if outcome.Response["answer"].Text == "clarify:later" {
							once.Do(func() { close(laterRecorded) })
						}
					}
					return nil
				})
				done := make(chan error, 1)
				go func() {
					_, err := s.Run(observer, runtime.RunOptions{Adapters: registry, MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 2, ExternalTimeout: 150 * time.Second})
					done <- err
				}()
				joined := false
				defer func() {
					p.unblock()
					cancel()
					if !joined {
						<-done
					}
				}()
				select {
				case <-p.held:
				case err := <-done:
					joined = true
					t.Fatal("no held transport", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				select {
				case <-allAdmitted:
				case err := <-done:
					joined = true
					t.Fatal("no complete scatter while first transport held", err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				if capacity == 2 {
					select {
					case <-laterRecorded:
					case err := <-done:
						joined = true
						t.Fatal("no independent durable result", err)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
					active, peak, _ := p.counts()
					if active != 1 || peak > 2 {
						t.Fatal("first transport no longer held or exceeded capacity", active, peak)
					}
					snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
					if err != nil {
						t.Fatal(err)
					}
					outcomes, err := leafOutcomes(snapshot, d.Leaf.ID())
					if err != nil {
						t.Fatal(err)
					}
					for _, outcome := range outcomes {
						if outcome.Response["answer"].Text == "clarify:hold" {
							t.Fatal("earlier result already complete")
						}
					}
				}
				// All ordinary logical admissions and collection acceptance run while a
				// physical slot is held, including the nested wrapper at capacity one.
				snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				expectedAdmitted := 4
				if nested {
					expectedAdmitted = 5
				}
				if len(snapshot.Tables["composition_invocations"]) != expectedAdmitted {
					t.Fatal("no logical progress under held transport")
				}
				acceptance, err := collectionAcceptance(snapshot, d.Collection.ID(), questions("hold", "later", "third"))
				if err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(acceptance)
				t.Logf("collector_acceptance_before_release=%s", encoded)
				t.Logf("held_before_release=true admitted=%d peak=%d capacity=%d nested=%v", len(snapshot.Tables["composition_invocations"]), func() int { _, peak, _ := p.counts(); return peak }(), capacity, nested)
				p.unblock()
				select {
				case err := <-done:
					joined = true
					if err != nil {
						t.Fatal(err)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				_, peak, contacts := p.counts()
				if peak != int(capacity) || contacts != 3 {
					t.Fatal("physical accounting", peak, contacts)
				}
				rows := rootRows(t, s)
				if len(rows) != 3 {
					t.Fatal(rows)
				}
				for i, row := range rows {
					response, err := row["responses"].Records()
					if err != nil || len(response) != 1 || row["item_id"].Text != fmt.Sprintf("question-%d", i) || response[0]["answer"].Text != []string{"clarify:hold", "clarify:later", "clarify:third"}[i] {
						t.Fatal("completion order corrupted input mapping", row, err)
					}
				}
				final, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				enclosing := int64(1)
				if nested {
					enclosing = 2
				}
				if final.Roots[0].Charges.Invocations != 3+enclosing || final.Roots[0].Charges.CreatedRuns != 6+enclosing || final.Roots[0].Charges.Effects != 3 {
					t.Fatal(final.Roots)
				}
				if _, err = runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestEvidenceUnknownRetainsOriginalOperationAcrossRecovery(t *testing.T) {
	p := &provider{held: make(chan struct{}), release: make(chan struct{}), mode: "unknown"}
	registry, closeServer := controlled(t, p)
	defer closeServer()
	defer p.unblock()
	s, path, d := createEvidence(t, true, 2, nil)
	submit(t, s, questions("unknown", "independent"))
	report, err := s.Run(t.Context(), runtime.RunOptions{Adapters: registry, MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots[0].UnknownExternal) != 1 || report.Roots[0].Outcome != nil || report.Roots[0].Charges.Effects != 2 {
		t.Fatal("unknown permitted success", report)
	}
	before, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	outcomes, err := leafOutcomes(before, d.Leaf.ID())
	if err != nil || len(outcomes) != 1 || outcomes[0].Response["answer"].Text != "clarify:independent" {
		t.Fatal("independent work did not finish", outcomes, err)
	}
	_, _, contacts := p.counts()
	if contacts != 2 {
		t.Fatal("unexpected contacts", contacts)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	recovered, err := s.Run(t.Context(), runtime.RunOptions{Adapters: registry, MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 2})
	if err != nil {
		t.Fatal(err)
	}
	_, _, after := p.counts()
	if after != contacts || len(recovered.Roots[0].UnknownExternal) != 1 || recovered.Roots[0].UnknownExternal[0] != report.Roots[0].UnknownExternal[0] || recovered.Roots[0].Charges.Effects != 2 || recovered.Roots[0].Outcome != nil {
		t.Fatal("unknown identity or charge reset", recovered)
	}
	t.Logf("unknown_original=%s independent_outcome=%s independent_operation=%s contacts_before=%d contacts_after=%d effects=%d", report.Roots[0].UnknownExternal[0], outcomes[0].ID, outcomes[0].OperationID, contacts, after, recovered.Roots[0].Charges.Effects)
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceAnswerInvalidDiagnosticAndEffectBudget(t *testing.T) {
	for _, mode := range []string{"answer", "invalid", "effect-limit"} {
		t.Run(mode, func(t *testing.T) {
			providerMode := mode
			if mode == "effect-limit" {
				providerMode = "answer"
			}
			p := &provider{held: make(chan struct{}), release: make(chan struct{}), mode: providerMode}
			registry, closeServer := controlled(t, p)
			defer closeServer()
			defer p.unblock()
			limits := worktree.DefaultLimits(1)
			if mode == "effect-limit" {
				limits.MaxEffects = 1
			}
			s, path, _ := createEvidence(t, false, 1, &limits)
			submit(t, s, questions("WAL?"))
			report, err := s.Run(t.Context(), runtime.RunOptions{Adapters: registry, MaxSteps: 2000, MaxDispatch: 500, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			rows := rootRows(t, s)
			if len(rows) != 1 {
				t.Fatal(rows)
			}
			responses, err := rows[0]["responses"].Records()
			if err != nil || len(responses) != 1 {
				t.Fatal(rows, err)
			}
			wantStatus, wantDisposition := "succeeded", "answer"
			if mode != "answer" {
				wantStatus, wantDisposition = "failed", "invalid_result"
			}
			if rows[0]["status"].Text != wantStatus || responses[0]["disposition"].Text != wantDisposition || report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != wantStatus {
				t.Fatal("helper public policy changed", rows, report)
			}
			if mode == "answer" {
				if responses[0]["document"].Text != "durability" || responses[0]["snippet"].Text != "WAL records changes before updating the main database." || report.Roots[0].Charges.Effects != 3 {
					t.Fatal("citation or stage accounting", rows, report)
				}
			}
			if mode == "effect-limit" && report.Roots[0].Charges.Effects != 1 {
				t.Fatal("effect budget reset", report)
			}
			snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range snapshot.Tables["adapter_attempts"] {
				var attempt runtime.AdapterAttempt
				if err = json.Unmarshal([]byte(row["descriptor"].(string)), &attempt); err != nil {
					t.Fatal(err)
				}
				t.Logf("effect operation=%s stage=%s source=%s key=%s allowed=%v reason=%s", attempt.Intent.ID, attempt.Intent.Obligation, attempt.Intent.Run.Subject, attempt.Prepared.Request.Key, attempt.Allowed, attempt.Reason)
			}
			record, _ := json.Marshal(map[string]any{"rows": rows, "roots": report.Roots})
			t.Logf("public_evidence=%s", record)
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// This monitor joins public retained inspection relationships without invoking
// a compiler/reducer. Intake acceptance alone cannot satisfy this check.
func collectionAcceptance(snapshot runtime.Inspection, collectionID string, items schema.Value) (map[string]any, error) {
	for _, row := range snapshot.Tables["composition_invocations"] {
		var invocation runtime.InvocationDescriptor
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &invocation); err != nil {
			return nil, err
		}
		if invocation.TemplateID != collectionID {
			continue
		}
		for _, request := range snapshot.Tables["composition_requests"] {
			if request["invocation_id"] != invocation.ID {
				continue
			}
			observationID := request["observation_id"]
			for _, observation := range snapshot.Tables["observations"] {
				if observation["id"] != observationID || observation["consumed_sequence"] == nil {
					continue
				}
				for _, commit := range snapshot.Tables["history"] {
					if commit["subject"] != observation["subject"] || commit["generation"] != observation["generation"] || commit["sequence"] != observation["consumed_sequence"] || commit["selected_observation"] != observationID {
						continue
					}
					var projection runtime.Projection
					if err := json.Unmarshal([]byte(commit["successor"].(string)), &projection); err != nil {
						return nil, err
					}
					if !projection.Data["items"].Equal(items) {
						return nil, fmt.Errorf("collector Receive retained different work")
					}
					return map[string]any{"invocation": invocation.ID, "operation": invocation.OperationID, "template": collectionID, "subject": commit["subject"], "generation": commit["generation"], "receive_sequence": commit["sequence"], "observation": observationID, "items": items}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("collection Receive acceptance commit absent")
}
