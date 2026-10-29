//go:build integration

package substitutiontest

import (
	app "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"path/filepath"
	"testing"
)

func TestSubstitutionRootSuccessFence(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		t.Run(name, func(t *testing.T) {
			q := app.Request{ID: 1, Value: 0, Mode: "answer"}
			l := limits(3, 3, 2)
			d, e := app.DelayedParallel(candidate(t, name), app.Config{Shape: "parallel", Request: q, Limits: l}, 32)
			must(t, e)
			path := filepath.Join(t.TempDir(), "retained.db")
			create(t, d, path)
			r := run(t, path)
			s := inspect(t, path)
			n := normalize(s, d, q)
			requireEvidence(t, n)
			if projection(s, "root").Status != "succeeded" || projection(s, "peer").Status != "failed" || n.Context.Closed {
				t.Fatal("concrete witness absent", n.Context)
			}
			denied := false
			for _, row := range s.Tables["composition_admission_decisions"] {
				a := decode[runtime.InvocationAdmission](row, "descriptor")
				if a.Invocation.Caller.Subject == "peer" && !a.Accepted && a.Reason == "scope_closed" {
					denied = true
				}
			}
			if !denied {
				t.Fatal("S scope refusal absent")
			}
			v := model.Local
			if name == "b" {
				v = model.Helper
			}
			c := model.Config{Shape: "parallel", Request: model.Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
			err := model.CheckCandidateContext(c, v, n.Context)
			if err == nil {
				t.Fatal("frozen optimistic K1 accepted counterexample")
			}
			boundaryCensus, boundaryErr := model.CheckBoundaryContext(c, v, n.Context)
			must(t, boundaryErr)
			if len(boundaryCensus.Counterexamples) == 0 {
				t.Fatal("boundary extension erased completion failure")
			}
			n.ModelMembership = err.Error()
			retain(t, "counterexample", map[string]any{"boundary_census": boundaryCensus, "boundary_membership": "accepted with preserved completion counterexamples", "report": r, "inspection": s, "normalized": n, "initial_entitled": model.Entitled(c), "peer_extra_pure_sets": 32, "runtime_source_fence": "compositionScopeClosed root local status != open", "result": "frozen K1 product correspondence fails; theorem implication C[K]Gc premise not established"})
			replayUnchanged(t, path)
		})
	}
}
