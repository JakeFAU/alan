//go:build integration

package compositiontest

import (
	"fmt"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"path/filepath"
	"reflect"
	"testing"
)

// A deployment-granted static operator waits for a test-injected command.
// These schedules exercise lifecycle serialization, not supervisory policy.
func addOperator(t *testing.T, p *runtime.Provision) {
	t.Helper()
	code := []executor.Instruction{
		{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: executor.Lit(schema.Text("local")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("cancel")), Operation: executor.Lit(schema.Text(""))}},
		{Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "lifecycle", Target: executor.Lit(schema.Text("root")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("control")), Payload: map[string]executor.Operand{"operation": executor.Lit(schema.Text("CancelRun")), "revision": executor.Lit(schema.Int(1))}}},
		{Op: "end", Outcome: "succeeded"},
	}
	raw, id, err := executor.EncodeProgram(executor.Program{Semantics: executor.SemanticVersion, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: executor.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: "operator", Generation: 1, Owner: "operator", Revision: 1})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: "operator", Generation: 1}, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: "operator", Operation: "input", Obligation: "cancel"})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "lifecycle", Source: "operator", Target: "root", Operation: "CancelRun", Revision: 1})
}
func cancelAsOperator(t *testing.T, s *runtime.Store) executor.LifecycleResult {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "cancel-operator", LogicalID: "cancel-operator", Target: "operator", Generation: 1, Obligation: "cancel", Payload: map[string]schema.Value{}})
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
	r, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "operator", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.Commit(t.Context(), r.Authority(), a.ObservationID)
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.Commit(t.Context(), r.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	request, err := s.LoadLifecycleRequest(t.Context(), r.Projection.Data["pending"].Text)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	if err != nil || !auth.Allowed {
		t.Fatal(auth, err)
	}
	result, err := s.DecideLifecycle(t.Context(), request)
	if err != nil || !result.Accepted {
		t.Fatal(result, err)
	}
	return result
}
func TestOperatorClosureAtCompositionCuts(t *testing.T) {
	for _, cut := range []string{"pre-admission", "accepted", "recorded", "partial"} {
		t.Run(cut, func(t *testing.T) {
			leaf, err := worktree.Deterministic()
			if err != nil {
				t.Fatal(err)
			}
			d, err := worktree.New(leaf, false, worktree.DefaultLimits(1))
			if err != nil {
				t.Fatal(err)
			}
			addOperator(t, &d.Provision)
			path := filepath.Join(t.TempDir(), "closed.sqlite")
			s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			a, err := worktree.Submit(t.Context(), s, "three", input())
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			s.Close()
			child := start(t, path, d.Collection.ID(), cut, false)
			ack := child.acknowledgment(t)
			before := inspect(t, path)
			if err = durableAck(before, ack); err != nil {
				t.Fatal(err)
			}
			child.wait(t, true)
			s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			decision := cancelAsOperator(t, s)
			s.Close()
			recovered := start(t, path, d.Collection.ID(), "", false)
			recovered.wait(t, false)
			after := inspect(t, path)
			if len(after.Roots) != 1 || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" {
				t.Fatal("closed root reopened", after.Roots)
			}
			if !reflect.DeepEqual(before.Roots[0].Charges, after.Roots[0].Charges) {
				t.Fatal("closure changed original charges", before.Roots, after.Roots)
			}
			for _, original := range before.Tables["composition_results"] {
				found := false
				for _, r := range after.Tables["composition_results"] {
					if reflect.DeepEqual(original, r) {
						found = true
					}
				}
				if !found {
					t.Fatal("closure erased known result")
				}
			}
			if cut == "pre-admission" && len(after.Tables["composition_invocations"]) != 0 {
				t.Fatal("creation after root closure")
			}
			replay := start(t, path, d.Collection.ID(), "", true)
			replay.wait(t, false)
			if !reflect.DeepEqual(after, inspect(t, path)) {
				t.Fatal("Replay mutated closure")
			}
			emit(t, map[string]any{"kind": "operator_closure", "cut": cut, "ack": ack, "decision": decision, "before": before, "after": after, "schedule": fmt.Sprintf("acknowledged %s, SIGKILL owner, open operator, cancel root, close operator, fresh Run", cut)})
		})
	}
}
