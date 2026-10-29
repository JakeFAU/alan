//go:build integration

package substitutiontest

import (
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

// The only low-level authoring is the declared granted environment operator.
// Candidate and context programs always execute under automatic Store.Run.
func addOperator(t *testing.T, p *runtime.Provision) {
	code := []executor.Instruction{{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: executor.Lit(schema.Text("local")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("cancel")), Operation: executor.Lit(schema.Text(""))}}, {Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "lifecycle", Target: executor.Lit(schema.Text("root")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("control")), Payload: map[string]executor.Operand{"operation": executor.Lit(schema.Text("CancelRun")), "revision": executor.Lit(schema.Int(1))}}}, {Op: "end", Outcome: "succeeded"}}
	raw, id, e := executor.EncodeProgram(executor.Program{Semantics: executor.SemanticVersion, Code: code})
	must(t, e)
	p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: executor.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: "operator", Generation: 1, Owner: "operator", Revision: 1})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: "operator", Generation: 1}, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: "operator", Operation: "input", Obligation: "cancel"}, runtime.Permission{Kind: "lifecycle", Source: "operator", Target: "root", Operation: "CancelRun", Revision: 1})
}
func admitCancel(t *testing.T, s *runtime.Store) runtime.AdmissionResult {
	t.Helper()
	a, e := s.AdmitInput(t.Context(), runtime.Input{TransportID: "cancel-operator", LogicalID: "cancel-operator", Target: "operator", Generation: 1, Obligation: "cancel", Payload: map[string]schema.Value{}})
	must(t, e)
	if !a.Accepted {
		t.Fatal(a)
	}
	return a
}
func cancelStopped(t *testing.T, path string) executor.LifecycleResult {
	t.Helper()
	s, e := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, e)
	defer s.Close()
	a := admitCancel(t, s)
	r, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "operator", Generation: 1})
	must(t, e)
	r, e = s.Commit(t.Context(), r.Authority(), a.ObservationID)
	must(t, e)
	r, e = s.Commit(t.Context(), r.Authority(), "")
	must(t, e)
	q, e := s.LoadLifecycleRequest(t.Context(), r.Projection.Data["pending"].Text)
	must(t, e)
	auth, e := s.AuthorizeIntent(t.Context(), q.Intent.ID)
	must(t, e)
	if !auth.Allowed {
		t.Fatal(auth)
	}
	result, e := s.DecideLifecycle(t.Context(), q)
	must(t, e)
	if !result.Accepted || result.Reason != "closed" || result.Before.Projection.Status != "open" || result.After.Projection.Status != "cancelled" {
		t.Fatal(result)
	}
	return result
}
func runClosure(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, cut := range []string{"entry-commit", "entry-admitted", "helper-commit", "reply-commit", "published"} {
			if name == "a" && cut == "helper-commit" {
				continue
			}
			t.Run(name+"/"+cut, func(t *testing.T) {
				q := app.Request{ID: 1, Value: 0, Mode: "answer"}
				d, path := provision(t, name, "bare", q, limits(3, 3, 3))
				addOperator(t, &d.Provision)
				create(t, d, path)
				p := start(t, path, d.Candidate.ID(), cut, "", false)
				ack := p.acknowledgment(t)
				before := inspect(t, path)
				retain(t, "before", before)
				p.wait(t, true)
				decision := cancelStopped(t, path)
				closed := inspect(t, path)
				retain(t, "closure", map[string]any{"decision": decision, "inspection": closed, "fixed_runs": d.FixedRuns + 1, "operator_outside_root": true, "ack": ack})
				run(t, path)
				after := inspect(t, path)
				stablePrefix(t, before, after)
				if after.Roots[0].LocalStatus != "cancelled" || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" {
					t.Fatal("closure lost", after.Roots)
				}
				n := normalize(after, d, q)
				requireEvidence(t, n)
				if !n.Context.Closed {
					t.Fatal("actual root cancellation must decode Closed=true")
				}
				if len(n.Violations) > 0 {
					t.Fatal(n.Violations)
				}
				retain(t, "after", map[string]any{"inspection": after, "normalized": n, "scope": "runtime prefix safety; model closure variants require separate model refinement mapping"})
				replayUnchanged(t, path)
			})
		}
	}
}
