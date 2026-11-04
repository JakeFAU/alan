package executor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"context"
	"github.com/JakeFAU/alan/internal/boundary"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"sync"
)

func TestRequestAuthorityOrdinaryTwoAgentRun(t *testing.T) {
	dir := t.TempDir()
	resourcePath := filepath.Join(dir, "endpoint.sqlite")
	ledger, err := resourcefixture.Open(t.Context(), resourcePath, "request-authority")
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resourcefixture.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		receipt, err := ledger.Apply(r.Context(), req)
		if err != nil {
			t.Error(err)
			return
		}
		if err = json.NewEncoder(w).Encode(receipt); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	p := Provision{}
	bRoot := AuthorityRoot{Run: RunID{"B", 1}, RootRun: RunID{"B", 1}, Permission: Permission{Kind: "external", Source: "B", Target: server.URL, Operation: "apply", Revision: 1, Obligation: "b"}}
	aRoot := bRoot
	aRoot.Permission.Obligation = "a"
	claim := DelegationClaim{Version: RequestAuthorityVersion, Root: bRoot, Recipient: RunID{"A", 1}, RecipientRoot: RunID{"A", 1}, RecipientRevision: 1, OriginalRequestID: identity("B", int64(1), int64(1), 0), ServiceOperationID: identity("A", int64(1), int64(4), 0)}
	raw, _ := json.Marshal(claim)
	emit := func(target, ob string) Instruction {
		return Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text(ob)), Payload: map[string]Operand{}}}
	}
	receive := func(source, ob string) Instruction {
		return Instruction{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text(source)), Generation: Lit(Int(1)), Obligation: Lit(Text(ob)), Operation: Lit(Text(""))}}
	}
	external := func(ob string, use AuthorityUse) Instruction {
		return Instruction{Op: "emit", Dst: "effect", Effect: &Effect{Kind: "external", Target: Lit(Text(server.URL)), Generation: Lit(Int(1)), Obligation: Lit(Text(ob)), Authority: &use, Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(1))}}}
	}
	receipt := func(ob string) Instruction {
		return Instruction{Op: "receive", Match: &Match{Kind: "receipt", Provenance: "resource-receipt", Source: Lit(Text(server.URL)), Generation: Lit(Int(1)), Obligation: Lit(Text(ob)), Operation: Ref("effect")}}
	}
	addProgram(t, &p, "A", []Instruction{emit("B", "AB"), receive("B", "ready"), receive("B", "BA"), external("b", AuthorityUse{Version: RequestAuthorityVersion, Path: "delegated", Root: bRoot, GrantID: identity("B", int64(1), int64(2), 0)}), receipt("b"), emit("B", "BA-paid"), receive("B", "AB-paid"), {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "B", []Instruction{emit("A", "BA"), {Op: "emit", Dst: "grant", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("grant")), Payload: map[string]Operand{"operation": Lit(Text("DelegateAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(raw)))}}},
		{Op: "receive", Match: &Match{Kind: "lifecycle", Provenance: "runtime-lifecycle", Source: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("grant")), Operation: Ref("grant")}, Bind: map[string]string{"accepted": "accepted"}},
		{Op: "branch", A: Ref("accepted"), B: Lit(Value{Kind: "bool", Flag: true}), Target: 4, Otherwise: 11}, emit("A", "ready"), receive("A", "BA-paid"), receive("A", "AB"), external("a", AuthorityUse{Version: RequestAuthorityVersion, Path: "own", Root: aRoot}), receipt("a"), emit("A", "AB-paid"), {Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"}}, map[string]Value{})
	p.Permissions = []Permission{bRoot.Permission, aRoot.Permission}
	for _, v := range []struct{ source, target, ob string }{{"A", "B", "AB"}, {"B", "A", "BA"}, {"B", "A", "ready"}, {"A", "B", "BA-paid"}, {"B", "A", "AB-paid"}} {
		p.Permissions = append(p.Permissions, Permission{Kind: "message", Source: v.source, Target: v.target, Operation: "message", Revision: 1, Obligation: v.ob})
	}
	p.Composition = &CompositionProvision{RequestAuthority: RequestAuthorityVersion, Roots: []ResourceRoot{{Run: RunID{"A", 1}, Members: []RunID{{"A", 1}}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}}, {Run: RunID{"B", 1}, Members: []RunID{{"B", 1}}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}}}}
	s, path := messageStore(t, p)
	var mu sync.Mutex
	var gates []boundary.Event
	ctx := boundary.WithObserver(t.Context(), func(_ context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.admitted" {
			mu.Lock()
			gates = append(gates, e)
			mu.Unlock()
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	if current(t, s, "A").Projection.Status != "succeeded" || current(t, s, "B").Projection.Status != "succeeded" || len(report.UnresolvedIntents) != 0 {
		t.Fatalf("incomplete ordinary run: %+v", report)
	}
	mu.Lock()
	if len(gates) != 2 {
		t.Errorf("gates=%+v", gates)
	}
	for _, g := range gates {
		if g.AuthorityAdmissionOrdinal != 1 {
			t.Errorf("missing ordinal: %+v", g)
		}
	}
	mu.Unlock()
	snap, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Effects) != 2 || len(snap.LifecycleDecisions) != 1 || !snap.LifecycleDecisions[0].Accepted {
		t.Fatalf("evidence %+v", snap)
	}
	for _, e := range snap.Effects {
		if e.Settlement == nil || len(e.AuthorityAdmissions) != 1 {
			t.Fatalf("unpaid effect %+v", e)
		}
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	endpoint, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Contacts != 2 || len(endpoint.Applications) != 2 {
		t.Fatalf("endpoint %+v", endpoint)
	}
}
