//go:build integration

package adaptivetest

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This exact low-level granted operator is environment policy injection, reused
// from compositiontest; it is not an application or new public authoring API.
func addOperator(t *testing.T, p *runtime.Provision) {
	t.Helper()
	code := []executor.Instruction{{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: executor.Lit(schema.Text("local")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("cancel")), Operation: executor.Lit(schema.Text(""))}}, {Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "lifecycle", Target: executor.Lit(schema.Text("root")), Generation: executor.Lit(schema.Int(1)), Obligation: executor.Lit(schema.Text("control")), Payload: map[string]executor.Operand{"operation": executor.Lit(schema.Text("CancelRun")), "revision": executor.Lit(schema.Int(1))}}}, {Op: "end", Outcome: "succeeded"}}
	raw, id, err := executor.EncodeProgram(executor.Program{Semantics: executor.SemanticVersion, Code: code})
	must(t, err)
	p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: executor.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: "operator", Generation: 1, Owner: "operator", Revision: 1})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: "operator", Generation: 1}, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: "operator", Operation: "input", Obligation: "cancel"}, runtime.Permission{Kind: "lifecycle", Source: "operator", Target: "root", Operation: "CancelRun", Revision: 1})
}
func admitCancel(t *testing.T, s *runtime.Store) runtime.AdmissionResult {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "cancel-operator", LogicalID: "cancel-operator", Target: "operator", Generation: 1, Obligation: "cancel", Payload: map[string]schema.Value{}})
	must(t, err)
	if !a.Accepted {
		t.Fatal(a)
	}
	return a
}
func cancelStoppedOwner(t *testing.T, s *runtime.Store) executor.LifecycleResult {
	t.Helper()
	a := admitCancel(t, s)
	r, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "operator", Generation: 1})
	must(t, err)
	r, err = s.Commit(t.Context(), r.Authority(), a.ObservationID)
	must(t, err)
	r, err = s.Commit(t.Context(), r.Authority(), "")
	must(t, err)
	request, err := s.LoadLifecycleRequest(t.Context(), r.Projection.Data["pending"].Text)
	must(t, err)
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	must(t, err)
	if !auth.Allowed {
		t.Fatal(auth)
	}
	result, err := s.DecideLifecycle(t.Context(), request)
	must(t, err)
	if !result.Accepted || result.Reason != "closed" || result.Before.Projection.Status != "open" || result.After.Projection.Status != "cancelled" {
		t.Fatal("actual open-to-cancelled required", result)
	}
	return result
}
func TestAdaptiveClosureAfterDecision(t *testing.T) {
	c := component(t, false)
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			d, path := provision(t, c, nested, 1, nil)
			addOperator(t, &d.Provision)
			create(t, d, path, true)
			p := start(t, path, c.ID(), "decision", "", false, "")
			ack := p.acknowledgment(t)
			before := inspect(t, path)
			must(t, durableAck(before, ack))
			must(t, cutMonitor(before, c.ID(), "decision", ack))
			retain(t, "before", before)
			p.wait(t, true)
			s, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
			must(t, err)
			decision := cancelStoppedOwner(t, s)
			must(t, s.Close())
			fresh := start(t, path, c.ID(), "", "", false, "")
			fresh.wait(t, false)
			after := inspect(t, path)
			retain(t, "after", after)
			must(t, preserve(before, after))
			if after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" || !reflect.DeepEqual(before.Roots[0].Charges, after.Roots[0].Charges) {
				t.Fatal("root closure/charges", after.Roots)
			}
			ops := itemOperations(after, c.ID())
			late := 0
			for _, id := range []int64{7, 10, 11} {
				if op := ops[id]; op != "" {
					o, ok := outcome(after, op)
					if !ok || o.Status != "rejected" || (o.Reason != "scope_closed" && o.Reason != "source_closed") || admitted(after, op) {
						t.Fatal("late admission not denied", id, o)
					}
					late++
				}
			}
			if late == 0 {
				t.Fatal("no actual attempted late admission")
			}
			// Public repeat of already-known outcomes is explicitly duplicate settlement,
			// not a claim of a newly arriving transport result.
			s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
			must(t, err)
			for _, id := range []int64{1, 2, 3} {
				_, err = s.SettleInvocation(t.Context(), ops[id])
				must(t, err)
			}
			must(t, s.Close())
			duplicate := inspect(t, path)
			if !reflect.DeepEqual(after, duplicate) {
				t.Fatal("duplicate known settlement changed closed store")
			}
			emit(t, map[string]any{"kind": "closure_after_decision", "ack": ack, "lifecycle_decision": decision, "actual_late_admissions_denied": late, "charges": after.Roots[0].Charges, "duplicate_known_settlement_unchanged": true, "new_late_transport_in_this_schedule": false, "fixed_static_runs": 2, "runtime_verdict": "cancelled", "application_verdict": "original decision and partial evidence retained; no reply promised"})
		})
	}
}

const peerCapability = "phase1h-closure-peer"

func addPeer(t *testing.T, p *runtime.Provision) {
	t.Helper()
	input := schema.Record{"value": {Kind: "int"}}
	output := schema.Record{"observation": {Kind: "int"}}
	a, err := author.Compile(author.Definition{Name: "closure companion static peer", Ports: map[string]author.Port{"start": {Local: true, Peer: "local", Obligation: "start", Receive: schema.Record{}}}, Effects: map[string]author.Effect{"controlled": {Capability: peerCapability, Input: input, Output: output}}, Steps: []author.Step{author.Receive("start", map[string]string{}), author.Request("first", "controlled", map[string]author.Expr{"value": author.Integer(91)}, nil), author.Await("controlled", "first", map[string]string{"observation": "first_observation", "valid": "first_valid"}), author.Request("second", "controlled", map[string]author.Expr{"value": author.Integer(92)}, nil), author.Await("controlled", "second", map[string]string{"observation": "second_observation", "valid": "second_valid"}), author.Success()}})
	must(t, err)
	q, err := (author.Application{"peer": {Artifact: a, Inputs: map[string]schema.Value{}}}).Provision()
	must(t, err)
	p.Programs = append(p.Programs, q.Programs...)
	p.Subjects = append(p.Subjects, q.Subjects...)
	p.Runs = append(p.Runs, q.Runs...)
	p.Permissions = append(p.Permissions, q.Permissions...)
	p.Composition.Roots[0].Members = append(p.Composition.Roots[0].Members, runtime.RunID{Subject: "peer", Generation: 1})
	p.Composition.Roots[0].Capabilities = append(p.Composition.Roots[0].Capabilities, peerCapability)
}
func TestAdaptiveClosureLateTransportCompanion(t *testing.T) {
	provider, server := newProvider(t, 91, false, false)
	c := component(t, false)
	d, path := provision(t, c, false, 2, nil)
	addPeer(t, &d.Provision)
	addOperator(t, &d.Provision)
	create(t, d, path, false)
	s, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, err)
	defer s.Close()
	a, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "peer-start", LogicalID: "peer-start", Target: "peer", Generation: 1, Obligation: "start", Payload: map[string]schema.Value{}})
	must(t, err)
	if !a.Accepted {
		t.Fatal(a)
	}
	closed := make(chan runtime.Inspection, 1)
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "lifecycle.decided" && e.Subject == "operator" {
			snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				return err
			}
			closed <- snap
		}
		return nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, runtime.RunOptions{Adapters: map[string]effects.Adapter{peerCapability: httpAdapter{server.URL}}, MaxSteps: 1000, MaxDispatch: 100, MaxInFlight: 2, ExternalTimeout: 10 * time.Minute})
		done <- err
	}()
	select {
	case <-provider.entered:
	case <-time.After(time.Minute):
		t.Fatal("watchdog: companion transport absent")
	}
	before := inspect(t, path)
	if projection(before, "peer").Status != "open" || len(before.Tables["adapter_results"]) != 0 || before.Roots[0].Charges.Effects != 1 {
		t.Fatal("no original held attempt", before.Roots)
	}
	admitCancel(t, s)
	var closure runtime.Inspection
	select {
	case closure = <-closed:
	case err := <-done:
		t.Fatal("Run before closure", err)
	case <-time.After(time.Minute):
		t.Fatal("watchdog: cancellation absent")
	}
	if len(closure.Tables["lifecycle_results"]) != 1 {
		t.Fatal("missing lifecycle result")
	}
	retain(t, "held", before)
	retain(t, "closure_before_release", closure)
	result := decode[executor.LifecycleResult](closure.Tables["lifecycle_results"][0], "result")
	if !result.Accepted || result.Reason != "closed" || result.Before.Projection.Status != "open" || result.After.Projection.Status != "cancelled" || projection(closure, "peer").Status != "open" {
		t.Fatal("companion cancellation/source", result)
	}
	ledger, _, _ := provider.snapshot()
	if contacts(ledger) != 1 || len(closure.Tables["adapter_results"]) != 0 {
		t.Fatal("response arrived before closure")
	}
	provider.unblock()
	select {
	case err := <-done:
		must(t, err)
	case <-time.After(time.Minute):
		t.Fatal("watchdog: late result drain")
	}
	after := inspect(t, path)
	retain(t, "after", after)
	must(t, preserve(before, after))
	q := projection(after, "peer")
	if !q.Data["first_valid"].Flag || q.Data["second_valid"].Flag || q.Data["first_observation"].Number != 91 || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" || after.Roots[0].Charges.Invocations != 0 || after.Roots[0].Charges.CreatedRuns != 0 || after.Roots[0].Charges.Effects != 1 {
		t.Fatal("late evidence/denial outcome", after.Roots, q.Data)
	}
	if !reflect.DeepEqual(closure.Tables["composition_closures"], after.Tables["composition_closures"]) {
		t.Fatal("closure prefix rewritten")
	}
	second := q.Data["second"].Text
	allowedSource := false
	for _, r := range after.Tables["attempts"] {
		if r["intent_id"] == second {
			var a struct{ Allowed bool }
			must(t, jsonUnmarshal(r["result"], &a))
			allowedSource = a.Allowed
		}
	}
	if !allowedSource {
		t.Fatal("second source authorization not allowed")
	}
	denied := false
	for _, r := range after.Tables["composition_reservations"] {
		var v struct {
			OperationID string
			Allowed     bool
			Reason      string
		}
		must(t, jsonUnmarshal(r["descriptor"], &v))
		if v.OperationID == second && !v.Allowed && v.Reason == "scope_closed" {
			denied = true
		}
	}
	if !denied {
		t.Fatal("missing exact scope_closed reservation")
	}
	last, _, peak := provider.snapshot()
	must(t, contactMonitor(after, last))
	if contacts(last) != 1 || peak != 1 {
		t.Fatal("second contact or excess capacity")
	}
	emit(t, map[string]any{"kind": "closure_late_transport_companion", "lifecycle_decision": result, "first_operation": q.Data["first"], "second_operation": q.Data["second"], "second_source_authorization_allowed": true, "second_reservation_reason": "scope_closed", "provider_ledger": last, "charges": after.Roots[0].Charges, "fixed_static_runs": 3, "adaptive_input_submitted": false, "runtime_verdict": "cancelled root immutable", "application_verdict": "separate ordinary scoped peer late-result and reservation witness"})
}
