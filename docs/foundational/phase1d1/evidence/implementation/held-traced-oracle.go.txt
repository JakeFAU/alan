package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

// This oracle is deliberately unchanged between the synchronous RED and repaired GREEN.
// The controller only releases real boundaries; only Run commits or delivers work.
func TestAsyncHeldRequestPeerProgress(t *testing.T) {
	token := t.TempDir()
	trace := func(kind string, value any) {
		raw, _ := json.Marshal(map[string]any{"type": "ASYNC", "scenario": t.Name(), "token": token, "pid": os.Getpid(), "event": kind, "value": value})
		t.Log(string(raw))
	}
	held := make(chan resource.Request, 1)
	release := make(chan struct{})
	ended := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		trace("endpoint-holding", req)
		held <- req
		select {
		case <-release:
			json.NewEncoder(w).Encode(resource.Receipt{Key: req.Key, Profile: req.Profile, Delta: req.Delta, Mutation: 1, Total: req.Delta, Process: "held-resource", PID: 1})
		case <-r.Context().Done():
		}
		close(ended)
	}))
	defer server.Close()
	p := externalProvision(t, server.URL, "idempotent")
	addProgram(t, &p, "peer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "caller", []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text("peer")), Generation: Lit(Int(1)), Obligation: Lit(Text("done")), Operation: Lit(Text(""))}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "peer", Target: "caller", Operation: "terminal", Revision: 1, Obligation: "done"})
	s, path := messageStore(t, p)
	trace("fixture", map[string]any{"provision": p, "store": path})
	var req resource.Request
	peerCommitted := false
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		trace("boundary", e)
		if e.Subject == "peer" && e.Boundary == "execution.staged" {
			select {
			case req = <-held:
			case <-ctx.Done():
				return fmt.Errorf("held acknowledgment missing: %w", ctx.Err())
			}
			if req.Key != `["client",1,1,0]` || req.Authorization == "" {
				return fmt.Errorf("wrong held request %+v", req)
			}
		}
		if e.Subject == "peer" && e.Boundary == "execution.committed" {
			select {
			case <-ended:
				return fmt.Errorf("held request ended before peer commit")
			default:
			}
			peerCommitted = true
		}
		if e.Subject == "caller" && e.Boundary == "execution.committed" && e.Sequence == 1 {
			if !peerCommitted {
				return fmt.Errorf("caller before peer")
			}
			select {
			case <-ended:
				return fmt.Errorf("held request ended before peer delivery")
			default:
			}
			t.Logf("ASYNC held=%+v peer-commit=true caller-consumed=true release-after=true", req)
			trace("controller-response-release", req)
			once.Do(func() { close(release) })
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 100, MaxDispatch: 100})
	once.Do(func() { close(release) })
	snapshot, inspectErr := Inspect(t.Context(), path, DefaultLimits())
	trace("final-records", map[string]any{"snapshot": snapshot, "inspect_error": fmt.Sprint(inspectErr), "run_error": fmt.Sprint(err), "report": report})
	if err != nil {
		t.Fatal(err)
	}
	if !peerCommitted || len(report.UnresolvedIntents) != 0 || current(t, s, "client").Projection.Status != "succeeded" || current(t, s, "caller").Projection.Status != "succeeded" {
		t.Fatalf("not settled %+v", report)
	}
}
