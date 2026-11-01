//go:build integration

package effectboundary_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/runtime"
)

// This process-local endpoint is an additional public-interface probe, not the
// independent persistent endpoint used for the SIGKILL campaign.
type world struct {
	mu       sync.Mutex
	requests []effectboundary.WireRequest
	receipts []effectboundary.Receipt
	total    int64
	drop     bool
}

func (w *world) serve(out http.ResponseWriter, r *http.Request) {
	var request effectboundary.WireRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(out, err.Error(), 400)
		return
	}
	w.mu.Lock()
	w.requests = append(w.requests, request)
	w.total += request.Delta
	receipt := effectboundary.Receipt{Key: request.Key, Profile: request.Profile, Delta: request.Delta, Mutation: int64(len(w.receipts) + 1), Total: w.total, Process: "public-probe", PID: os.Getpid()}
	w.receipts = append(w.receipts, receipt)
	w.mu.Unlock()
	if w.drop && request.Delta == 1 {
		connection, _, err := out.(http.Hijacker).Hijack()
		if err == nil {
			connection.Close()
		}
		return
	}
	if err := json.NewEncoder(out).Encode(receipt); err != nil {
		return
	}
}
func (w *world) copy() (requests []effectboundary.WireRequest, receipts []effectboundary.Receipt) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append(requests, w.requests...), append(receipts, w.receipts...)
}
func record(t *testing.T, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(raw))
	if dir := os.Getenv("ALAN_PHASE2A_PARENT_EVIDENCE"); dir != "" {
		if err = os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(t.Name()) + ".json"
		if err = os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func TestParentBoundary(t *testing.T) {
	cases := []struct {
		name, mode, knowledge string
		drop                  bool
		status                string
		results               int
	}{
		{name: "completed", mode: "success", knowledge: "historical_application_receipt", status: "succeeded", results: 2},
		{name: "selected_unsuccessful", mode: "selected-error", knowledge: "local_failure_world_unresolved", status: "succeeded", results: 2},
		{name: "real_postcommit_connection_loss", mode: "success", drop: true, knowledge: "local_failure_world_unresolved", status: "succeeded", results: 2},
		{name: "unrecorded_unknown", mode: "unknown", knowledge: "may_have_applied", status: "open", results: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			w := &world{drop: tc.drop}
			server := httptest.NewServer(http.HandlerFunc(w.serve))
			defer server.Close()
			d, err := effectboundary.New(effectboundary.Config{Parent: true, Peer: true, Delta: 1})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "store.db")
			s, err := runtime.Create(ctx, path, d.Provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: effectboundary.Registry(server.URL, tc.mode)})
			if err != nil {
				t.Fatal(err)
			}
			root, err := s.Snapshot(ctx, runtime.RunID{Subject: "root", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			peer, err := s.Snapshot(ctx, runtime.RunID{Subject: "peer", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			if root.Projection.Status != tc.status || root.Projection.Data["knowledge"].Text != tc.knowledge || peer.Projection.Status != "succeeded" {
				t.Fatalf("root=%+v peer=%+v report=%+v", root, peer, report)
			}
			if tc.name == "unrecorded_unknown" {
				if len(report.UnknownAdapters) != 1 || root.Projection.Data["child_status"].Text != "" {
					t.Fatalf("parent selected absent result: %+v", report)
				}
			} else {
				if len(report.UnknownAdapters) != 0 {
					t.Fatal(report)
				}
				expected := "failed"
				if tc.mode == "success" && !tc.drop {
					expected = "succeeded"
				}
				if root.Projection.Data["child_status"].Text != expected {
					t.Fatal(root)
				}
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			inspection, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if len(inspection.Tables["adapter_results"]) != tc.results {
				t.Fatal("result count", len(inspection.Tables["adapter_results"]))
			}
			requests, receipts := w.copy()
			if len(requests) != 2 || len(receipts) != 2 {
				t.Fatalf("world applications %+v", receipts)
			}
			if requests[0].Delta != 1 || requests[1].Delta != 2 {
				t.Fatalf("peer did not follow primary contact: %+v", requests)
			}
			// All retained preparations are checked against the exact sent wire, not a
			// newly reconstructed request or application-level surrogate key.
			for _, row := range inspection.Tables["adapter_attempts"] {
				var attempt runtime.AdapterAttempt
				if err = json.Unmarshal([]byte(row["descriptor"].(string)), &attempt); err != nil {
					t.Fatal(err)
				}
				matched := false
				for _, request := range requests {
					raw, e := json.Marshal(request)
					if e != nil {
						t.Fatal(e)
					}
					if string(raw) == attempt.Prepared.Wire && request.Key == attempt.Prepared.Request.Key {
						matched = true
					}
				}
				if !matched {
					t.Fatal("wire did not match retained preparation")
				}
			}
			replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			after, _ := w.copy()
			if len(after) != len(requests) {
				t.Fatal("replay contacted endpoint")
			}
			// Reopening unresolved adapters must not borrow endpoint idempotency as
			// permission to execute again. A changed registry mode cannot repair it.
			reopened, err := runtime.Open(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			recovery, err := reopened.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: effectboundary.Registry(server.URL, "success")})
			if err != nil {
				t.Fatal(err)
			}
			if err = reopened.Close(); err != nil {
				t.Fatal(err)
			}
			after, _ = w.copy()
			if len(after) != 2 {
				t.Fatal("recovery repeated adapter")
			}
			record(t, map[string]any{"boundary": tc.name, "physical_cap": 1, "report": report, "root": root, "peer": peer, "inspection": inspection, "oracle_requests": requests, "oracle_receipts": receipts, "replay": replay, "recovery": recovery})
		})
	}
}
func TestPrepareIsPureAndBindsRequest(t *testing.T) {
	w := &world{}
	server := httptest.NewServer(http.HandlerFunc(w.serve))
	defer server.Close()
	adapter := effectboundary.Counter{Endpoint: server.URL, Mode: "success"}
	request := effects.Request{ID: "intent", Key: "original-key", Payload: map[string]runtime.Value{"delta": {Kind: "int", Number: 1}}}
	prepared, err := adapter.Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	requests, _ := w.copy()
	if len(requests) != 0 {
		t.Fatal("prepare contacted world")
	}
	if prepared.Request.Key != request.Key || prepared.Wire != `{"Key":"original-key","Profile":"idempotent","Delta":1}` {
		t.Fatal(prepared)
	}
	candidate := adapter.Execute(context.Background(), prepared)
	if candidate.Disposition != "validated" {
		t.Fatal(candidate)
	}
}

// heldAdapter exposes only a causal worker-entry gate to the research host. It
// never supplies an observation or a result to the ordinary parent.
type heldAdapter struct {
	delegate effects.Adapter
	entered  chan struct{}
	release  chan struct{}
}

func (a heldAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	return a.delegate.Prepare(r)
}
func (a heldAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	if p.Request.Payload["delta"].Number == 1 {
		close(a.entered)
		select {
		case <-a.release:
		case <-ctx.Done():
			return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
		}
	}
	return a.delegate.Execute(ctx, p)
}
func TestParentSelectsPrivateFailureBeforeEffectResult(t *testing.T) {
	ctx := t.Context()
	w := &world{}
	server := httptest.NewServer(http.HandlerFunc(w.serve))
	defer server.Close()
	d, err := effectboundary.New(effectboundary.Config{Parent: true, FailAfterRequest: true, Delta: 1})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "store.db")
	s, err := runtime.Create(ctx, path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	gate := heldAdapter{delegate: effectboundary.Counter{Endpoint: server.URL, Mode: "success"}, entered: make(chan struct{}), release: make(chan struct{})}
	released := false
	defer func() {
		if !released {
			close(gate.release)
		}
	}()
	type result struct {
		report runtime.RunReport
		err    error
	}
	done := make(chan result, 1)
	go func() {
		report, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: map[string]effects.Adapter{effectboundary.Capability: gate}})
		done <- result{report, e}
	}()
	select {
	case <-gate.entered:
	case r := <-done:
		t.Fatalf("no reserved worker: %+v", r)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var root runtime.RunState
	// Snapshot establishes terminal selection, not elapsed time. The loop cannot
	// manufacture progress; a bounded watchdog diagnoses a missing transition.
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		root, err = s.Snapshot(ctx, runtime.RunID{Subject: "root", Generation: 1})
		if err != nil {
			t.Fatal(err)
		}
		if root.Projection.Status != "open" {
			break
		}
		select {
		case <-deadline.C:
			t.Fatal("parent did not select failed child while worker held")
		default:
			goruntime.Gosched()
		}
	}
	if root.Projection.Data["child_status"].Text != "failed" || root.Projection.Data["child_response_present"].Flag || root.Projection.Data["knowledge"].Text != "local_failure_world_unresolved" {
		t.Fatal(root)
	}
	requests, _ := w.copy()
	if len(requests) != 0 {
		t.Fatal("held adapter launched")
	}
	before, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Tables["adapter_attempts"]) != 1 || len(before.Tables["adapter_results"]) != 0 {
		t.Fatal("missing reservation or invented result")
	}
	close(gate.release)
	released = true
	var run result
	select {
	case run = <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if run.err != nil {
		t.Fatal(run.err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	requests, receipts := w.copy()
	if len(receipts) != 1 || len(after.Tables["adapter_results"]) != 1 {
		t.Fatal("late authentic result absent")
	}
	replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	record(t, map[string]any{"boundary": "private_failure_before_effect_result", "root_selected_while_reserved": root, "before": before, "after": after, "report": run.report, "oracle_requests": requests, "oracle_receipts": receipts, "replay": replay})
}
