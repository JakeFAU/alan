//go:build integration

package substitutiontest

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/callablejoin"
	tree "github.com/JakeFAU/alan/examples/owningtree"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type owningSchedule struct {
	driver *communicationDriver
	term   *owningNode
	fork   runtime.RunID
	j, e   [2]runtime.RunID
	ops    [2]string
}

// Leave B at its original entry Receive on both sides. Each J already committed
// e then S. The shared root has paid seven once-only admissions at this cut.
func owningScheduleAtSeven(t *testing.T, l runtime.ResourceLimits, left, right string, operator bool) owningSchedule {
	t.Helper()
	n := owningFixture(t, "fork", "BB", []app.Request{{ID: 1, Value: 0, Mode: left}, {ID: 1, Value: 1, Mode: right}})
	p, err := tree.Provision(n.template, n.spec().request, l)
	must(t, err)
	if operator {
		addOperator(t, &p)
	}
	path := filepath.Join(t.TempDir(), "scheduled.db")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	must(t, err)
	must(t, s.Close())
	v := owningSchedule{driver: newCommunicationDriver(t, path), term: n}
	root := runtime.RunID{Subject: "root", Generation: 1}
	op := v.untilHandle(t, root, "tree_operation")
	v.fork = v.admitted(t, op)
	v.untilHandle(t, v.fork, "right_operation")
	f := projection(inspect(t, path), v.fork.Subject)
	v.ops = [2]string{f.Data["left_operation"].Text, f.Data["right_operation"].Text}
	for i, op := range v.ops {
		v.j[i] = v.admitted(t, op)
	}
	for i, j := range v.j {
		v.untilHandle(t, j, "s_operation")
		jp := projection(inspect(t, path), j.Subject)
		v.e[i] = v.admitted(t, jp.Data["candidate_operation"].Text)
		v.admitted(t, jp.Data["s_operation"].Text)
	}
	r := inspect(t, path).Roots[0]
	if r.Charges.Invocations != 7 || r.Charges.CreatedRuns != 7 || r.Charges.Effects != 0 {
		t.Fatal("seven original slots", r)
	}
	return v
}

func (s owningSchedule) untilHandle(t *testing.T, id runtime.RunID, register string) string {
	t.Helper()
	// Each prefix is straight-line, through at most init, Receive and two Invokes.
	for i := 0; i <= 4; i++ {
		p := projection(inspect(t, s.driver.path), id.Subject)
		if op := p.Data[register].Text; op != "" {
			return op
		}
		if i == 4 || !s.driver.step(id) {
			t.Fatalf("did not reach %s at %v PC %d", register, id, p.PC)
		}
	}
	panic("unreachable")
}
func (s owningSchedule) admitted(t *testing.T, op string) runtime.RunID {
	t.Helper()
	a, err := s.driver.store.AdmitInvocation(t.Context(), op)
	must(t, err)
	if !a.Accepted || a.Invocation.OperationID != op || len(a.Members) != 1 {
		t.Fatal("expected original accepted singleton", a)
	}
	return a.Members[0]
}

func TestOwningTreeTightLimits(t *testing.T) {
	for _, c := range []struct {
		name   string
		l      runtime.ResourceLimits
		reason string
	}{{"I", limits(8, 9, 4), "invocation_limit"}, {"R", limits(9, 8, 4), "created_run_limit"}, {"depth", limits(9, 9, 3), "depth_limit"}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := owningScheduleAtSeven(t, c.l, "answer", "answer", false)
			left := s.untilHandle(t, s.e[0], "helper_operation")
			right := s.untilHandle(t, s.e[1], "helper_operation")
			denied, want := left, int64(7)
			if c.name != "depth" {
				s.admitted(t, left)
				denied = right
				want = 8
			}
			a, err := s.driver.store.AdmitInvocation(t.Context(), denied)
			must(t, err)
			snap := inspect(t, s.driver.path)
			if a.Accepted || a.Invocation.OperationID != denied || a.Reason != c.reason || len(a.Members) != 0 || a.Invocation.Depth != 4 ||
				snap.Roots[0].Charges.Invocations != want || snap.Roots[0].Charges.CreatedRuns != want {
				t.Fatal("wrong refusal or charge", a, snap.Roots)
			}
			retain(t, "manual", map[string]any{"kind": "tight-" + c.name, "denied": denied, "reason": c.reason, "inspection": snap})
		})
	}
}

func TestOwningTreeDelayedFrontier(t *testing.T) {
	for _, first := range []string{"right", "left"} {
		t.Run(first, func(t *testing.T) {
			t.Parallel()
			l, r, winning := "pending", "answer", 1
			if first == "left" {
				l, r, winning = "fail", "pending", 0
			}
			s := owningScheduleAtSeven(t, limits(9, 9, 4), l, r, false)
			blocked := 1 - winning
			if !s.driver.step(s.e[blocked]) {
				t.Fatal("opposite B did not consume request")
			}
			if p := projection(inspect(t, s.driver.path), s.e[blocked].Subject); p.PC != 1 {
				t.Fatal("expected B before h Invoke", p)
			}
			invocationJoinComplete(t, s.driver, s.ops[winning])
			prefix := inspect(t, s.driver.path)
			if projection(prefix, s.e[blocked].Subject).PC != 1 || prefix.Roots[0].Charges.Invocations != 8 {
				t.Fatal("sibling debit did not cross fixed B1 cut")
			}
			// Advance past both private flag initializers to the first genuine Await.
			for range 2 {
				if !s.driver.step(s.fork) {
					t.Fatal("flag setup")
				}
			}
			var rejected map[string]any
			if first == "right" {
				delivery := communicationRow(prefix, "composition_outcome_deliveries", "operation_id", s.ops[1])
				if delivery == nil {
					t.Fatal("missing right delivery")
				}
				pre, err := s.driver.store.Snapshot(t.Context(), s.fork)
				must(t, err)
				_, err = s.driver.store.Commit(t.Context(), pre.Authority(), delivery["observation_id"].(string))
				if !errors.Is(err, executor.ErrIneligible) {
					t.Fatal("wrong original right observation at left Await", err)
				}
				after, e := s.driver.store.Snapshot(t.Context(), s.fork)
				must(t, e)
				rejected = map[string]any{"before": pre, "after": after, "observation_id": delivery["observation_id"], "error": err.Error()}
				if !reflect.DeepEqual(pre, after) {
					t.Fatal("unauthorized consumption advanced parent")
				}
			}
			p, err := tree.Provision(s.term.template, s.term.spec().request, limits(9, 9, 4))
			must(t, err)
			steps, dispatch := owningBounds(t, p, 9)
			drain, err := s.driver.store.Run(t.Context(), runtime.RunOptions{MaxSteps: steps + 1, MaxDispatch: dispatch + 1, MaxInFlight: 1})
			must(t, err)
			snap := inspect(t, s.driver.path)
			if !drain.Quiescent || snap.Roots[0].Outcome != nil || len(snap.Tables["history"]) > steps {
				t.Fatal("schedule did not reach pending frontier", drain)
			}
			calls := owningCalls(t, snap)
			root := runtime.RunID{Subject: "root", Generation: 1}
			owningCheck(t, snap, s.term, calls[root]["tree"], false, calls)
			retain(t, "manual", map[string]any{"kind": "delayed-" + first, "winning": s.ops[winning], "blocked_member": s.e[blocked], "rejected": rejected, "prefix": prefix, "report": drain, "inspection": snap})
		})
	}
}

func TestOwningTreeLateReply(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		name := "open"
		if cancel {
			name = "cancel"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := owningScheduleAtSeven(t, limits(9, 9, 4), "answer", "answer", cancel)
			hop := s.untilHandle(t, s.e[0], "helper_operation")
			h := s.admitted(t, hop)
			for range 4 {
				if !s.driver.step(h) {
					t.Fatal("h prefix")
				}
			}
			pre := inspect(t, s.driver.path)
			var replyID string
			for _, row := range pre.Tables["intents"] {
				in := decode[intent](row, "descriptor")
				if in.Kind == "message" && in.Run == h && in.Target == s.e[0].Subject && in.Payload["value"] == schema.Int(1) {
					replyID = in.ID
				}
			}
			if replyID == "" || communicationRow(pre, "attempts", "intent_id", replyID) != nil || projection(pre, h.Subject).PC != 6 {
				t.Fatal("missing original undecided Send")
			}
			var cancelID string
			if cancel {
				admitCancel(t, s.driver.store)
				op := runtime.RunID{Subject: "operator", Generation: 1}
				if !s.driver.step(op) || !s.driver.step(op) {
					t.Fatal("operator did not issue cancellation")
				}
				q, err := s.driver.store.LoadLifecycleRequest(t.Context(), projection(inspect(t, s.driver.path), op.Subject).Data["pending"].Text)
				must(t, err)
				c, err := s.driver.store.AuthorizeIntent(t.Context(), q.Intent.ID)
				must(t, err)
				if !c.Allowed {
					t.Fatal(c)
				}
				decision, err := s.driver.store.DecideLifecycle(t.Context(), q)
				must(t, err)
				if !decision.Accepted || decision.Reason != "closed" || decision.After.Run.Subject != "root" || decision.After.Projection.Status != "cancelled" {
					t.Fatal("wrong granted cancellation", decision)
				}
				cancelID = q.Intent.ID
			}
			if !s.driver.step(h) {
				t.Fatal("h did not End")
			}
			a := s.driver.authorize(replyID)
			if !a.Allowed || a.SourceStatus != "succeeded" || a.Source != h.Subject {
				t.Fatal("bound late first permission", a)
			}
			s.driver.deliver(replyID)
			snap := inspect(t, s.driver.path)
			bound := decode[reply](communicationRow(snap, "composition_replies", "operation_id", replyID), "descriptor")
			if !bound.Accepted || bound.Reason != "recorded_reply" || bound.Intent.Run != h || bound.Intent.Target != s.e[0].Subject {
				t.Fatal("wrong bound disposition", bound)
			}
			retain(t, "manual", map[string]any{"kind": "late-" + name, "helper": hop, "reply": replyID, "cancel": cancelID, "inspection": snap})
		})
	}
}

// This component has the authentic diagnostic port. Its behavior is outside
// the positive leaf premise. It exercises real selected invalid evidence.
func owningBad(t *testing.T, kind string) author.Template {
	t.Helper()
	port := map[string]author.Port{join.PublicPort: {Peer: "caller", Obligation: "work", Receive: protocol.Request(), Send: join.Report()}}
	steps := []author.Step{author.Receive(join.PublicPort, map[string]string{"id": "id", "value": "value", "mode": "mode"})}
	switch kind {
	case "absent-failure":
		steps = append(steps, author.Failure())
	case "absent-success":
		steps = append(steps, author.Success())
	case "wrong-answer":
		steps = append(steps, author.Append("answers", map[string]author.Expr{"id": author.Var("id"), "value": author.Var("value")}),
			author.Send(join.PublicPort, map[string]author.Expr{"id": author.Var("id"), "classification": author.String("answer"), "answers": author.Var("answers")}), author.Success())
	default:
		t.Fatal("unknown negative", kind)
	}
	a, err := author.Compile(author.Definition{Name: "phase1t-" + kind, Ports: port, Collections: map[string]schema.Field{"answers": join.Report()["answers"]}, Steps: steps})
	must(t, err)
	tmpl, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", join.PublicPort)
	must(t, err)
	return tmpl
}

func TestOwningTreeInvalid(t *testing.T) {
	for _, kind := range []string{"wrong-answer", "absent-failure", "absent-success"} {
		for _, mode := range []string{"answer", "pending"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				t.Parallel()
				bad := owningBad(t, kind)
				leaf, err := tree.Leaf(candidate(t, "b"))
				must(t, err)
				f, err := tree.Fork(bad, leaf)
				must(t, err)
				top, err := tree.Relay(f)
				must(t, err)
				q := map[string]schema.Value{"l_id": schema.Int(1), "l_value": schema.Int(0), "l_mode": schema.Text("answer"), "r_id": schema.Int(1), "r_value": schema.Int(1), "r_mode": schema.Text(mode)}
				// Relay + Fork + one hostile singleton + one four-slot J[B].
				p, err := tree.Provision(top, q, limits(7, 7, 5))
				must(t, err)
				_, s, _ := owningRun(t, p, 7)
				root := runtime.RunID{Subject: "root", Generation: 1}
				calls := owningCalls(t, s)
				rc := calls[root]["tree"]
				fc := calls[rc.a.Members[0]]["child"]
				kids := calls[fc.a.Members[0]]
				if !kids["left"].selected || kids["left"].out == nil {
					t.Fatal("bad original outcome not selected")
				}
				lo := kids["left"].out
				switch kind {
				case "wrong-answer":
					if !lo.ResponsePresent || lo.Status != "succeeded" {
						t.Fatal(lo)
					}
				case "absent-failure":
					if lo.ResponsePresent || lo.Reason != "member_failed" {
						t.Fatal(lo)
					}
				case "absent-success":
					if lo.ResponsePresent || lo.Reason != "missing_reply" {
						t.Fatal(lo)
					}
				}
				if mode == "pending" {
					if kids["right"].selected || kids["right"].out != nil || fc.out != nil || rc.out != nil || s.Roots[0].Outcome != nil {
						t.Fatal("invalid child caused early completion")
					}
				} else {
					for _, c := range []owningCall{fc, rc} {
						if c.out == nil || c.out.ResponsePresent || c.out.Reason != "member_failed" || !c.selected {
							t.Fatal("invalid propagation", c)
						}
					}
					if !kids["right"].selected || projection(s, "root").Data["decision"] != schema.Text("invalid_tree_outcome") {
						t.Fatal("missing right selection/invalid decision")
					}
				}
				retain(t, "manual", map[string]any{"kind": kind + "-" + mode, "inspection": s})
			})
		}
	}
}
