//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	tree "github.com/JakeFAU/alan/examples/owningtree"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

var owningModes = []string{"answer", "fail", "absent", "pending"}

type owningNode struct {
	template author.Template
	children []*owningNode
	role     string
	leaf     string
	request  app.Request
}

// Programs are composed by the public author. Expected results and slot counts
// below are derived independently from the requested leaf assignment and shape.
func owningFixture(t *testing.T, shape, assignment string, requests []app.Request) *owningNode {
	t.Helper()
	leaves := make([]*owningNode, len(requests))
	for i, q := range requests {
		name := string(assignment[i])
		c := candidate(t, map[string]string{"A": "a", "B": "b"}[name])
		v, err := tree.Leaf(c)
		must(t, err)
		leaves[i] = &owningNode{template: v, leaf: name, request: q}
	}
	fork := func(l, r *owningNode) *owningNode {
		v, err := tree.Fork(l.template, r.template)
		must(t, err)
		l.role = "left"
		r.role = "right"
		return &owningNode{template: v, children: []*owningNode{l, r}}
	}
	relay := func(ch *owningNode) *owningNode {
		v, err := tree.Relay(ch.template)
		must(t, err)
		ch.role = "child"
		return &owningNode{template: v, children: []*owningNode{ch}}
	}
	switch shape {
	case "fork":
		return fork(leaves[0], leaves[1])
	case "left-nested":
		return fork(fork(leaves[0], leaves[1]), leaves[2])
	case "right-nested":
		return fork(leaves[0], fork(leaves[1], leaves[2]))
	case "relay-above":
		return relay(fork(leaves[0], leaves[1]))
	case "relay-left":
		return fork(relay(leaves[0]), leaves[1])
	default:
		t.Fatal("unknown shape", shape)
		return nil
	}
}

type owningSpec struct {
	request, report map[string]schema.Value
	decision        string
	b, h, spend     int
}

func (n *owningNode) spec() owningSpec {
	s := owningSpec{request: map[string]schema.Value{}, report: map[string]schema.Value{}, decision: "answer", b: 1, h: 1, spend: 1}
	if n.leaf != "" {
		q := n.request
		id := schema.Int(q.ID)
		s.request = map[string]schema.Value{"id": id, "value": schema.Int(q.Value), "mode": schema.Text(q.Mode)}
		s.b, s.h, s.spend = 4, 3, 3
		if n.leaf == "B" {
			s.spend++
		}
		if q.Mode == "pending" {
			s.decision = "pending"
			return s
		}
		tag := map[string]string{"answer": "answer", "fail": "candidate_failed", "absent": "candidate_missing_reply"}[q.Mode]
		answers := schema.List()
		if tag == "answer" {
			answers = schema.List(map[string]schema.Value{"id": id, "value": schema.Int(1 - q.Value)})
		} else {
			s.decision = "diagnosed"
		}
		s.report = map[string]schema.Value{"id": id, "classification": schema.Text(tag), "answers": answers}
		return s
	}
	for i, ch := range n.children {
		cs := ch.spec()
		s.b += cs.b
		s.spend += cs.spend
		s.h = max(s.h, cs.h+1)
		prefix := ""
		if len(n.children) == 2 {
			prefix = []string{"l_", "r_"}[i]
		}
		for k, v := range cs.request {
			s.request[prefix+k] = v
		}
		for k, v := range cs.report {
			s.report[prefix+k] = v
		}
		if cs.decision == "pending" || (s.decision == "answer" && cs.decision == "diagnosed") {
			s.decision = cs.decision
		}
	}
	return s
}

// One linear pass plus the final next instruction of each MaxItems=1 loop.
// Use the largest program cost for each possible singleton occurrence; repeated
// instances of identical bytes still count separately.
func owningBounds(t *testing.T, p runtime.Provision, b int) (int, int) {
	t.Helper()
	largest := 0
	for _, a := range p.Programs {
		var program struct {
			Code []struct {
				Op         string
				Collection *schema.Field
			}
		}
		must(t, json.Unmarshal(a.Bytes, &program))
		work := len(program.Code)
		for _, in := range program.Code {
			if in.Op == "foreach" && (in.Collection == nil || in.Collection.MaxItems != 1) {
				t.Fatal("unbounded fixture loop")
			}
			if in.Op == "next" {
				work++
			}
		}
		largest = max(largest, work)
	}
	return (b + 1) * largest, 3 * b
}

func owningRun(t *testing.T, p runtime.Provision, b int) (runtime.RunReport, runtime.Inspection, string) {
	t.Helper()
	step, dispatch := owningBounds(t, p, b)
	path := filepath.Join(t.TempDir(), "owning.db")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	must(t, err)
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: step + 1, MaxDispatch: dispatch + 1, MaxInFlight: 1})
	must(t, s.Close())
	must(t, err)
	raw := inspect(t, path)
	if !report.Quiescent || report.Steps > step || len(raw.Tables["history"]) > step || report.Dispatches > dispatch {
		t.Fatal("execution outside frozen bounds", report)
	}
	return report, raw, path
}

type owningCall struct {
	a        runtime.InvocationAdmission
	out      *runtime.InvocationOutcome
	selected bool
}

func owningCalls(t *testing.T, s runtime.Inspection) map[runtime.RunID]map[string]owningCall {
	t.Helper()
	byParent := map[runtime.RunID]map[string]owningCall{}
	for _, row := range s.Tables["composition_admission_decisions"] {
		a := decode[runtime.InvocationAdmission](row, "descriptor")
		op := a.Invocation.OperationID
		it := decode[intent](communicationRow(s, "intents", "id", op), "descriptor")
		children := byParent[it.Run]
		if children == nil {
			children = map[string]owningCall{}
			byParent[it.Run] = children
		}
		if _, dup := children[it.Obligation]; dup {
			t.Fatal("duplicate per-member role", it)
		}
		c := owningCall{a: a}
		if pr := communicationRow(s, "composition_publications", "operation_id", op); pr != nil {
			out := decode[runtime.InvocationOutcome](pr, "descriptor")
			c.out = &out
			delivery := communicationRow(s, "composition_outcome_deliveries", "operation_id", op)
			if delivery == nil {
				t.Fatal("missing atomic delivery", op)
			}
			ob := communicationRow(s, "observations", "id", delivery["observation_id"])
			if ob == nil {
				t.Fatal("missing actual observation", op)
			}
			c.selected = ob["consumed_sequence"] != nil
		}
		children[it.Obligation] = c
	}
	return byParent
}

func owningCheck(t *testing.T, s runtime.Inspection, n *owningNode, call owningCall, selected bool, calls map[runtime.RunID]map[string]owningCall) {
	t.Helper()
	sp := n.spec()
	if !call.a.Accepted || len(call.a.Members) != 1 || call.a.Invocation.TemplateID != n.template.ID() || !sameValues(call.a.Invocation.Inputs, sp.request) {
		t.Fatal("wrong original member, type or request", call.a)
	}
	member := call.a.Members[0]
	children := calls[member]
	if call.selected != selected || (call.out == nil) != (sp.decision == "pending") {
		t.Fatal("incorrect selection/frontier", member, selected, sp.decision, call)
	}
	if call.out != nil {
		status := "succeeded"
		if sp.decision == "diagnosed" {
			status = "failed"
		}
		if !call.out.ResponsePresent || call.out.Status != status || call.out.Reason != "member_"+status || !sameValues(call.out.Response, sp.report) {
			t.Fatal("incorrect product", member, call.out, sp.report)
		}
	}
	if n.leaf != "" {
		e, side := children["candidate"], children["successor"]
		if len(children) != 2 || !side.selected || side.out == nil || !side.out.ResponsePresent || side.out.Response["id"] != schema.Int(n.request.ID) || side.out.Response["value"] != schema.Int(0) || e.selected != (n.request.Mode != "pending") {
			t.Fatal("original leaf selections", children)
		}
		if n.leaf == "B" {
			h := calls[e.a.Members[0]]["complement"]
			if !h.selected || h.out == nil || !h.out.ResponsePresent || h.out.Response["value"] != schema.Int(1-n.request.Value) {
				t.Fatal("helper did not supply selected complement", h)
			}
		}
		return
	}
	if len(children) != len(n.children) {
		t.Fatal("missing child commitment", children)
	}
	blocked := false
	for _, ch := range n.children {
		cc := children[ch.role]
		if !blocked && ch.spec().decision == "pending" {
			blocked = true
			p := projection(s, member.Subject)
			if p.Data[ch.role+"_operation"] != schema.Text(cc.a.Invocation.OperationID) || p.Status != "open" {
				t.Fatal("wrong pending child", p)
			}
		}
		owningCheck(t, s, ch, cc, !blocked, calls)
	}
}

func owningCase(t *testing.T, shape, assignment string, requests []app.Request) {
	t.Helper()
	n := owningFixture(t, shape, assignment, requests)
	sp := n.spec()
	p, err := tree.Provision(n.template, sp.request, limits(sp.b, sp.b, sp.h))
	must(t, err)
	r, s, path := owningRun(t, p, sp.b)
	closed, err := validateRetained(s)
	must(t, err)
	if closed || r.RejectedAdmissions != 0 || len(s.Roots) != 1 || s.Roots[0].Charges.Invocations != int64(sp.spend) || s.Roots[0].Charges.CreatedRuns != int64(sp.spend) || s.Roots[0].Charges.Effects != 0 {
		t.Fatal("admission/accounting", r)
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	calls := owningCalls(t, s)
	owningCheck(t, s, n, calls[root]["tree"], sp.decision != "pending", calls)
	o := projection(s, "root")
	if sp.decision == "pending" {
		if _, ok := o.Data["decision"]; ok || o.Status != "open" || s.Roots[0].Outcome != nil {
			t.Fatal("invented outer boundary", o)
		}
	} else if o.Data["decision"] != schema.Text(sp.decision) || s.Roots[0].Outcome == nil {
		t.Fatal("incorrect root result", o)
	}
	replayUnchanged(t, path)
	step, dispatch := owningBounds(t, p, sp.b)
	retain(t, "case", map[string]any{"shape": shape, "assignment": assignment, "requests": requests, "decision": sp.decision, "steps_bound": step, "dispatch_bound": dispatch, "expected_slots": sp.b, "depth": sp.h, "report": r, "inspection": s})
}

func TestOwningTreePairs(t *testing.T) {
	for _, assignment := range []string{"AA", "AB", "BA", "BB"} {
		for i, left := range owningModes {
			for j, right := range owningModes {
				t.Run(assignment+"/"+left+"-"+right, func(t *testing.T) {
					t.Parallel()
					owningCase(t, "fork", assignment, []app.Request{{ID: 1, Value: 0, Mode: left}, {ID: int64(1 + (i+j)%2), Value: 1, Mode: right}})
				})
			}
		}
	}
}

func TestOwningTreeNested(t *testing.T) {
	for _, shape := range []string{"left-nested", "right-nested", "relay-above", "relay-left"} {
		for i, modes := range [][]string{{"answer", "fail", "absent"}, {"pending", "answer", "fail"}, {"fail", "pending", "answer"}, {"answer", "absent", "pending"}} {
			t.Run(fmt.Sprintf("%s/%d", shape, i), func(t *testing.T) {
				t.Parallel()
				q := []app.Request{{ID: 1, Value: 0, Mode: modes[0]}, {ID: 1, Value: 1, Mode: modes[1]}, {ID: 2, Value: 0, Mode: modes[2]}}
				assignment := "BAB"
				if shape == "relay-above" || shape == "relay-left" {
					q = q[:2]
					assignment = assignment[:2]
				}
				owningCase(t, shape, assignment, q)
			})
		}
	}
}
