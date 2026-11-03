package executor

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectInvocationCausalCuts(t *testing.T) {
	p, profile := invocationFixture(t, "failed", 1)
	awaitingInvocationRoot(t, &p)
	path := filepath.Join(t.TempDir(), "causal.sqlite")
	s, err := createInvocationStore(t.Context(), path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := first.Projection.Data["pending"].Text
	view := func() *InvocationView {
		t.Helper()
		snapshot, err := InspectExecution(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		v, ok := snapshot.Invocation(id)
		if !ok {
			t.Fatal("original invocation not found")
		}
		return v
	}
	if v := view(); v.Admission != nil || v.State != "committed" || v.Outcome != nil {
		t.Fatalf("before admission: %+v", v)
	}
	if _, err := s.AuthorizeIntent(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if v := view(); v.Authorization == nil || !v.Authorization.Allowed || v.SourceRevision != 1 || v.Admission != nil || v.State != "authorized" {
		t.Fatalf("authorized but unadmitted: %+v", v)
	}
	a, err := s.AdmitInvocation(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	v := view()
	if v.Admission == nil || !v.Admission.Accepted || len(v.Admission.Members) != 2 || v.Outcome != nil || v.RequestObservationID == "" {
		t.Fatalf("bound workers: %+v", v)
	}
	if len(v.Dependencies) != 2 || v.Dependencies[0].DependsOnRun == nil || v.Dependencies[1].DependsOnRun == nil {
		t.Fatal("unfrozen scope lost its two member dependencies", v.Dependencies)
	}
	entry := compositionMemberID(a.Invocation.ID, "entry")
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), v.RequestObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	reply := identity(entry.Subject, 1, 2, 0)
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if v := view(); v.Freeze == nil || v.Freeze.Status != "failed" || v.Outcome != nil {
		t.Fatalf("frozen only: %+v", v)
	} else if len(v.Dependencies) != 1 || v.Dependencies[0].Kind != "invocation_reply_admission" || v.Dependencies[0].DependsOn != reply {
		t.Fatal("unpublished failure must still await its exact frozen reply", v.Dependencies)
	}
	if _, err = s.DeliverMessage(t.Context(), reply, "test-reply"); err != nil {
		t.Fatal(err)
	}
	if v := view(); len(v.Dependencies) != 1 || v.Dependencies[0].Kind != "invocation_publication" {
		t.Fatal("all needed evidence retained, but not yet published", v.Dependencies)
	}
	settled, err := s.SettleInvocation(t.Context(), id)
	if err != nil || !settled.Accepted {
		t.Fatal(settled, err)
	}
	v = view()
	if v.Outcome == nil || v.Outcome.Status != "failed" || !v.Outcome.ResponsePresent || v.Outcome.Response["result"].Text != "invalid_result" || v.State != "published" || v.OutcomeObservationID != settled.ObservationID || v.SelectedAt != nil || len(v.Replies) != 1 || !v.Replies[0].Accepted || v.ResponseOperationID != reply {
		t.Fatalf("published failed outcome awaiting selection: %+v", v)
	}
	selected, err := s.Commit(t.Context(), current(t, s, "root").Authority(), settled.ObservationID)
	if err != nil {
		t.Fatal(err)
	}
	if v := view(); v.SelectedAt == nil || *v.SelectedAt != (ReductionRef{selected.ID, selected.Sequence}) || v.State != "consumed" {
		t.Fatalf("parent's actual selection: %+v", v)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ table, mutation string }{
		{"composition_outcome_deliveries", "UPDATE composition_outcome_deliveries SET observation_id=(SELECT observation_id FROM composition_requests)"},
		{"composition_requests", "UPDATE composition_requests SET descriptor=CAST(json_set(descriptor,'$.Run.Generation',2) AS BLOB)"},
		{"composition_publications", "UPDATE composition_publications SET descriptor=CAST(json_set(descriptor,'$.OperationID','forged') AS BLOB)"},
		{"composition_results", "DELETE FROM composition_results"},
		{"composition_bindings", "DELETE FROM composition_bindings"},
		{"missing_observation", "PRAGMA foreign_keys=OFF; DELETE FROM observations WHERE id IN (SELECT observation_id FROM composition_outcome_deliveries)"},
		{"forged_observation_correlation", "UPDATE observations SET descriptor=CAST(json_set(descriptor,'$.Operation','foreign') AS BLOB) WHERE id IN (SELECT observation_id FROM composition_outcome_deliveries)"},
		{"incorrect_consuming_reduction", "UPDATE observations SET consumed_sequence=1 WHERE id IN (SELECT observation_id FROM composition_outcome_deliveries)"},
	} {
		t.Run(tc.table, func(t *testing.T) {
			disposable := filepath.Join(t.TempDir(), "damaged.sqlite")
			if err := os.WriteFile(disposable, original, 0600); err != nil {
				t.Fatal(err)
			}
			editor, err := connect(t.Context(), disposable, false)
			if err != nil {
				t.Fatal(err)
			}
			defer editor.Close()
			if _, err = editor.conn.ExecContext(t.Context(), tc.mutation); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectExecution(t.Context(), disposable, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("impossible link accepted: %v", err)
			}
		})
	}
}

// Both refusals use the same code, caller, generation, status, input and
// reason. Only their original operation IDs distinguish the two publications.
func TestInspectRefusedSiblingInvocationIdentity(t *testing.T) {
	p, config := invocationFixture(t, "succeeded", 1)
	config.Roots[0].Limits.MaxInvocations = 0
	path := filepath.Join(t.TempDir(), "siblings.sqlite")
	s, err := createInvocationStore(t.Context(), path, p, config, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, second := invokeOnce(t, s), invokeOnce(t, s)
	for _, a := range []compositionAdmission{first, second} {
		if a.Accepted || a.Reason != "invocation_limit" {
			t.Fatal(a)
		}
		if _, err = s.SettleInvocation(t.Context(), a.Invocation.OperationID); err != nil {
			t.Fatal(err)
		}
	}
	snap, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []compositionAdmission{first, second} {
		v, ok := snap.Invocation(a.Invocation.OperationID)
		if !ok || v.Admission.Accepted || v.Outcome == nil || v.Outcome.Status != "rejected" || v.Outcome.InvocationID != a.Invocation.ID || v.RequestObservationID != "" || v.SelectedAt != nil {
			t.Fatal("wrong refusal or publication", v)
		}
	}
	a := snap.Invocations[0]
	b := snap.Invocations[1]
	// Deliberately swap two authentic IDs (including their UNIQUE-key
	// transition), leaving both foreign keys pointing at real observations.
	if _, err = s.conn.ExecContext(t.Context(), "BEGIN; PRAGMA defer_foreign_keys=ON; UPDATE composition_outcome_deliveries SET observation_id='intermediate' WHERE operation_id=?; UPDATE composition_outcome_deliveries SET observation_id=? WHERE operation_id=?; UPDATE composition_outcome_deliveries SET observation_id=? WHERE operation_id=?; COMMIT",
		a.OperationID, a.OutcomeObservationID, b.OperationID, b.OutcomeObservationID, a.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("swapped authentic sibling publications accepted: %v", err)
	}
}

func TestInspectInputTransportDistinctFromSelection(t *testing.T) {
	s, path := messageStore(t, inputFixture(t))
	req := Input{TransportID: "first", LogicalID: "same-logical-input", Target: "receiver", Obligation: "work", Payload: map[string]Value{"number": Int(17)}}
	first, err := s.AdmitInput(t.Context(), req)
	if err != nil || !first.Accepted {
		t.Fatal(first, err)
	}
	req.TransportID = "second"
	if _, err := s.AdmitInput(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	req.Target, req.TransportID = "missing", "refused"
	if _, err := s.AdmitInput(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	before, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	o, ok := before.Observation(first.ObservationID)
	if !ok || o.SelectedAt != nil || len(o.Admissions) != 2 || o.ProducerRun != nil || o.Source != "local" || len(before.Admissions) != 3 {
		t.Fatal("logical input, transport, or external producer", o)
	}
	for _, a := range before.Admissions {
		if a.TransportID == "refused" && (a.Accepted || a.Reason != "target_missing" || a.Bound != nil || a.ObservationID != "") {
			t.Fatal("refusal was given a recipient", a)
		}
	}
	after, err := s.Commit(t.Context(), current(t, s, "receiver").Authority(), first.ObservationID)
	if err != nil {
		t.Fatal(err)
	}
	cut, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	o, _ = cut.Observation(first.ObservationID)
	if o.SelectedAt == nil || *o.SelectedAt != (ReductionRef{after.ID, after.Sequence}) {
		t.Fatal("input was not linked to the actual reduction", o)
	}
}

func retainedMessageSelection(t *testing.T, scoped bool) (*Store, string) {
	t.Helper()
	p := Provision{}
	send := Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(17))}}}
	addProgram(t, &p, "sender", []Instruction{send, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	code := inputProgram()
	code[0].Match.Kind, code[0].Match.Provenance, code[0].Match.Source = "message", "ordinary-message", Lit(Text("sender"))
	addProgram(t, &p, "receiver", code, map[string]Value{})
	p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "work"}}
	if scoped {
		p.Composition = &compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"receiver", 1}, Members: []RunID{{"receiver", 1}, {"sender", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
	}
	s, path := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	admission, err := s.DeliverMessage(t.Context(), identity("sender", 1, 1, 0), "transport")
	if err != nil || !admission.Accepted {
		t.Fatal(admission, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "receiver").Authority(), admission.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal("valid pre-mutation graph", err)
	}
	return s, path
}

func TestInspectSelectionRequiresEarlierSourceAuthorization(t *testing.T) {
	s, path := retainedMessageSelection(t, true)
	reorderCompositionConsumptionFacts(t, s, []int{0, 2, 1})
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE composition_facts SET descriptor=CAST(descriptor AS BLOB)"); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectExecution(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Fatal("consumption preceding sender authorization was accepted", err)
	}
}

func TestInspectTerminalAcknowledgmentRequiresTransport(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "child", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, &p, "observer", inputProgram(), map[string]Value{})
	d := TerminalDestination{Target: "observer", Obligation: "public"}
	p.Permissions = []Permission{{Kind: "observation", Source: "child", Target: d.Target, Operation: "terminal", Revision: 1, Obligation: d.Obligation}}
	s, path := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "child").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal("valid pre-mutation pending terminal delivery", err)
	}
	value, err := json.Marshal(map[string]bool{d.key(): true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE terminal_obligations SET delivery_status=? WHERE subject='child'", value); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Fatal("unbacked terminal acknowledgment was accepted", err)
	}
}
