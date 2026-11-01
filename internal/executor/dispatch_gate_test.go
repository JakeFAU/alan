package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	resource "github.com/JakeFAU/alan/internal/fixture"
)

func dispatchEndFixture(t *testing.T, target string) (*Store, string, RunState) {
	t.Helper()
	p := Provision{}
	addProgram(t, &p, "source", []Instruction{
		{Op: "emit", Dst: "op", Effect: &Effect{Kind: "external", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(1))}}},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "external", Source: "source", Target: target, Operation: "apply", Revision: 1}}
	s, path := messageStore(t, p)
	state, err := s.Commit(t.Context(), current(t, s, "source").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	return s, path, state
}

func dispatchReceiptServer(t *testing.T, contacts *atomic.Int64) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		contacts.Add(1)
		if err := json.NewEncoder(w).Encode(receiptFor(req)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestPhase2BFreshExposureAfterEnd(t *testing.T) {
	var contacts atomic.Int64
	s, _, state := dispatchEndFixture(t, dispatchReceiptServer(t, &contacts).URL)
	id := state.Projection.Data["op"].Text
	attempt, fresh, err := s.prepareExternal(t.Context(), id)
	if err != nil || !fresh || !attempt.Allowed {
		t.Fatal(attempt, fresh, err)
	}
	ended, err := s.Commit(t.Context(), state.Authority(), "")
	if err != nil || ended.Projection.Status != "succeeded" {
		t.Fatal(ended, err)
	}
	settled, err := s.DispatchExternal(context.Background(), id)
	t.Logf("reservation=%s, End acknowledged at sequence=%d, direct returned settled=%v err=%v, POSTs=%d", attempt.AuthorizationID, ended.Sequence, settled, err, contacts.Load())
	if err != nil || settled || contacts.Load() != 0 || count(t, s, "external_receipts") != 0 || count(t, s, "settlements") != 0 {
		t.Fatal("fresh post-End exposure entered transport or fabricated an outcome")
	}
}
