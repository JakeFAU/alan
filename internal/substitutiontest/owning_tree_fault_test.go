//go:build integration

package substitutiontest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/callablejoin"
	tree "github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// An authentic absent child response deletes a previously defined register.
// The unchecked read belongs to a hostile component inside an ordinary Fork.
func TestOwningTreeFaultIsNotPending(t *testing.T) {
	absent := owningBad(t, "absent-success")
	a, err := author.Compile(author.Definition{
		Name:        "phase1t-unguarded-diagnostic",
		Invocations: map[string]author.Template{"bad": absent},
		Ports:       map[string]author.Port{join.PublicPort: {Peer: "caller", Obligation: "work", Receive: protocol.Request(), Send: join.Report()}},
		Steps: []author.Step{
			author.Receive(join.PublicPort, map[string]string{"id": "id", "value": "value", "mode": "mode"}),
			author.Set("selected_id", author.Integer(73)),
			author.Invoke("saved", "bad", map[string]author.Expr{"id": author.Var("id"), "value": author.Var("value"), "mode": author.Var("mode")}, nil),
			author.AwaitInvocation("bad", "saved", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"id": "selected_id"}}),
			author.Set("unguarded", author.Var("selected_id")), author.Failure(),
		},
	})
	must(t, err)
	unsafe, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", join.PublicPort)
	must(t, err)
	leaf, err := tree.Leaf(candidate(t, "a"))
	must(t, err)
	f, err := tree.Fork(unsafe, leaf)
	must(t, err)
	q := map[string]schema.Value{"l_id": schema.Int(1), "l_value": schema.Int(0), "l_mode": schema.Text("answer"), "r_id": schema.Int(1), "r_value": schema.Int(1), "r_mode": schema.Text("answer")}
	p, err := tree.Provision(f, q, limits(7, 7, 4))
	must(t, err)
	file := filepath.Join(t.TempDir(), "fault.db")
	s, err := runtime.Create(t.Context(), file, p, runtime.DefaultLimits())
	must(t, err)
	steps, dispatch := owningBounds(t, p, 7)
	r, runErr := s.Run(t.Context(), runtime.RunOptions{MaxInFlight: 1, MaxSteps: steps + 1, MaxDispatch: dispatch + 1})
	must(t, s.Close())
	if runErr == nil || !strings.Contains(runErr.Error(), `missing or invalid data "selected_id"`) || r.Quiescent {
		t.Fatal("expected runner fault", r, runErr)
	}
	raw := inspect(t, file)
	calls := owningCalls(t, raw)
	root := runtime.RunID{Subject: "root", Generation: 1}
	fc := calls[root]["tree"]
	uc := calls[fc.a.Members[0]]["left"]
	member := uc.a.Members[0]
	bad := calls[member]["bad"]
	state := projection(raw, member.Subject)
	if bad.out == nil || bad.out.ResponsePresent || bad.out.Reason != "missing_reply" || !bad.selected || uc.out != nil || fc.out != nil || raw.Roots[0].Outcome != nil || state.PC != 4 || state.Data["saved"] != schema.Text(bad.a.Invocation.OperationID) {
		t.Fatal("incorrect real fault frontier", state, bad)
	}
	if _, exists := state.Data["selected_id"]; exists {
		t.Fatal("absent field survived actual selection")
	}
	retain(t, "manual", map[string]any{"kind": "optional-fault", "inspection": raw, "error": runErr.Error(), "report": r, "bad_program": a.ID()})
}
