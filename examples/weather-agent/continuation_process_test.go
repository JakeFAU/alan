package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// Recover the tested sequence numbers from an independent successful run. Test
// verdicts below depend on the stopped and reopened ledgers and HTTP log.
type continuationPath struct {
	operation                   string
	approval, request, terminal int64
}

func secondApproval(t *testing.T, s *runtime.ExecutionSnapshot) *runtime.ObservationView {
	t.Helper()
	var approval *runtime.ObservationView
	for _, o := range s.Observations {
		if o.Run.Subject == "weather-2" && o.Obligation == declaredWeatherSlot(t, s, 2).Obligation {
			if o.Run.Generation != 1 || o.Provenance != "local-input" || approval != nil {
				t.Fatalf("spurious or repeated approval for read two: %+v", o)
			}
			approval = o
		}
	}
	// Independently join the native continuation view to the ordinary selected
	// observation, the origin ledger and the successor operation.
	run, ok := s.Run(runtime.RunID{Subject: "weather-2", Generation: 1})
	if !ok {
		t.Fatal("missing successor generation")
	}
	v, ok := s.Continuation(run.ID)
	if ok != (approval != nil) {
		t.Fatalf("continuation view disagrees with the durable approval for %v", run.ID)
	}
	if !ok {
		return nil
	}
	verifyContinuationProof(t, s, run, approval, v)
	return approval
}

func verifyContinuationProof(t *testing.T, s *runtime.ExecutionSnapshot, target *runtime.ExecutionRun, o *runtime.ObservationView, v *runtime.ContinuationView) {
	t.Helper()
	first := weatherEffect(t, s, "weather-1")
	decision := v.Approval
	priorGate := first.DispatchAdmission != nil && first.DispatchAdmission.AfterFact < v.DecisionOrder
	if v.DecisionOrder < 1 || v.ObservationID != o.ID || !reflect.DeepEqual(v.SelectedAt, o.SelectedAt) ||
		decision.Namespace != s.Namespace || decision.Target != target.ID ||
		decision.PredecessorID != first.OperationID || decision.PreparedDigest != first.AdapterAttempt.Prepared.Digest ||
		decision.Admitted != priorGate || decision.Reason != o.Payload["purpose"].Text ||
		o.Payload["prepared_digest"].Text != decision.PreparedDigest || o.Payload["result_digest"].Text != decision.ResultDigest {
		t.Fatalf("continuation view contradicts the original reservation, result or selected observation: %+v", v)
	}
	prepared := first.AdapterAttempt.Prepared
	prepared.Request.ID, prepared.Request.Key, prepared.Digest = "", "", ""
	scope, err := json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Admissions) != 1 || fmt.Sprintf("%x", sha256.Sum256(scope)) != decision.ScopeDigest {
		t.Fatal("approval did not bind the complete prepared action and its original admission")
	}
	admission, ok := s.Admission(o.Admissions[0])
	if !ok || !admission.Accepted || admission.ObservationID != o.ID || admission.Target != target.ID.Subject ||
		admission.RequestedGeneration != target.ID.Generation || admission.Obligation != declaredWeatherSlot(t, s, 2).Obligation || admission.TransportID != declaredWeatherSlot(t, s, 2).ApprovalID ||
		admission.LogicalID != admission.TransportID {
		t.Fatalf("approval lost its scoped admission: %+v", admission)
	}
	if len(target.EffectIDs) == 0 {
		if v.SuccessorOperationID != "" {
			t.Fatal("continuation view fabricated a successor")
		}
		return
	}
	e := weatherEffect(t, s, "weather-2")
	if len(target.EffectIDs) != 1 || v.SuccessorOperationID != e.OperationID || v.SelectedAt == nil || v.SelectedAt.Run != e.Source ||
		v.SelectedAt.Sequence >= e.Sequence || decision.TargetRevision != e.SourceRevision || (e.AdapterAttempt != nil && e.AdapterAttempt.Prepared.Wire != first.AdapterAttempt.Prepared.Wire) {
		t.Fatal("successor escaped the selected approval, original action or one-effect scope")
	}
}

// Read the absence or presence of the response from the separately captured
// pre-approval database cut. A later response need not have been known then.
func confirmDecisionBasis(t *testing.T, after *runtime.ExecutionSnapshot, before *runtime.ExecutionEffect) {
	t.Helper()
	v, ok := after.Continuation(runtime.RunID{Subject: "weather-2", Generation: 1})
	wantDigest := ""
	if before.AdapterResult != nil {
		wantDigest = before.AdapterResult.Digest
	}
	if !ok || v.Approval.PredecessorID != before.OperationID || v.Approval.Admitted != (before.DispatchAdmission != nil) || v.Approval.ResultDigest != wantDigest {
		t.Fatal("retained approval differs from the independently observed predecessor at the decision")
	}
}

func secondReadReference(t *testing.T, binary string) continuationPath {
	t.Helper()
	p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather}, weatherReply{status: 200, body: laterWeather})
	db := startWeather(t, binary, p.endpoint(), 2)
	weatherCommand(t, binary, "run", "--db", db)
	weatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
	weatherCommand(t, binary, "resume", "--db", db)
	s, _ := inspectAndReplayWeather(t, binary, db, p, 2, 2)
	p.requireContacts(t, 2)
	e := weatherEffect(t, s, "weather-2")
	a := secondApproval(t, s)
	r, _ := s.Run(e.Source)
	first := weatherEffect(t, s, "weather-1")
	if a == nil || a.SelectedAt == nil || a.SelectedAt.Run != e.Source || a.SelectedAt.Sequence >= e.Sequence ||
		r.Projection.Status != "succeeded" || first.OperationID == e.OperationID || e.OperationID != p.contacts()[1].operation {
		t.Fatal("reference successor lacks its distinct operation, prior selected approval or terminal observation")
	}
	return continuationPath{e.OperationID, a.SelectedAt.Sequence, e.Sequence, r.Sequence}
}

type continuationCut struct {
	name, boundary, phase string
	sequence              int64
	approval              string
	effect                runtime.EffectStage
	gate, terminal        bool
	contacts, spent       int // additional to the first read
	lost                  bool
}

func (c continuationCut) check(t *testing.T, at continuationPath, s *runtime.ExecutionSnapshot, report weatherView, first *runtime.ExecutionEffect) {
	t.Helper()
	prev := weatherEffect(t, s, "weather-1")
	if prev.OperationID != first.OperationID || !reflect.DeepEqual(prev.AdapterAttempt, first.AdapterAttempt) ||
		!reflect.DeepEqual(prev.AdapterResult, first.AdapterResult) || !reflect.DeepEqual(prev.Settlement, first.Settlement) ||
		!reflect.DeepEqual(prev.DispatchAdmission, first.DispatchAdmission) || !reflect.DeepEqual(prev.Authorization, first.Authorization) {
		t.Fatal("the successor changed its original predecessor evidence")
	}
	a := secondApproval(t, s)
	if (a == nil) != (c.approval == "absent") {
		t.Fatalf("wrong approval commitment at cut %s: %+v", c.name, a)
	}
	if a != nil {
		confirmDecisionBasis(t, s, first)
		if (a.SelectedAt != nil) != (c.approval == "selected") || a.Payload["previous_operation_id"].Text != first.OperationID {
			t.Fatalf("wrong approval selection or predecessor at cut %s: %+v", c.name, a)
		}
		if a.SelectedAt != nil {
			step, ok := s.Reduction(*a.SelectedAt)
			if !ok || step.SelectedObservationID != a.ID || step.Sequence != at.approval {
				t.Fatal("reported approval selection differs from the original reduction")
			}
		}
	}
	r, ok := s.Run(runtime.RunID{Subject: "weather-2", Generation: 1})
	if !ok || (r.Projection.Status == "succeeded") != c.terminal || (report.read(t, 2).Report != "") != c.terminal {
		t.Fatalf("wrong local terminal evidence at %s: %+v", c.name, r)
	}
	if c.effect == "" {
		if len(r.EffectIDs) != 0 {
			t.Fatalf("uncommitted successor acquired an operation: %v", r.EffectIDs)
		}
		return
	}
	e := weatherEffect(t, s, "weather-2")
	if e.OperationID != at.operation || e.OperationID == prev.OperationID || e.Sequence != at.request || e.State != c.effect || a == nil || a.SelectedAt == nil ||
		(e.DispatchAdmission != nil) != c.gate || (e.AdapterAttempt != nil) != (c.spent == 1) || (e.Authorization != nil) != (c.effect != runtime.EffectCommitted) {
		t.Fatalf("incorrect successor commitment, authority, charge or gate at %s: %+v", c.name, e)
	}
	if (e.AdapterResult != nil) != (c.effect == runtime.EffectResultUnsettled || c.effect == runtime.EffectSettled) ||
		(e.Settlement != nil) != (c.effect == runtime.EffectSettled) || (selectedWeather(t, s, e) != nil) != c.terminal {
		t.Fatalf("wrong result, settlement or consumption at %s", c.name)
	}
}

func TestWeatherContinuationProcess(t *testing.T) {
	binary := publicWeatherBinary(t)
	at := secondReadReference(t, binary)
	cases := []continuationCut{
		{name: "before_approval_commit", boundary: "continuation.approval", phase: "pre-commit", approval: "absent"},
		{name: "after_approval_commit", boundary: "continuation.approval", phase: "post-commit", approval: "pending"},
		{name: "before_approval_selection", boundary: "execution.staged", phase: "pre-commit", sequence: at.approval, approval: "pending"},
		{name: "after_approval_selection", boundary: "execution.committed", phase: "post-commit", sequence: at.approval, approval: "selected"},
		{name: "before_successor_commit", boundary: "execution.staged", phase: "pre-commit", sequence: at.request, approval: "selected"},
		{name: "after_successor_commit", boundary: "execution.committed", phase: "post-commit", sequence: at.request, approval: "selected", effect: runtime.EffectCommitted},
		{name: "after_paid_reservation", boundary: "dispatch.consider", phase: "pre-admission", approval: "selected", effect: runtime.EffectReservedUnknown, spent: 1, lost: true},
		{name: "after_final_admission", boundary: "dispatch.admitted", phase: "pre-call", approval: "selected", effect: runtime.EffectReservedUnknown, spent: 1, gate: true, lost: true},
		{name: "accepted_lost_response", approval: "selected", effect: runtime.EffectReservedUnknown, gate: true, contacts: 1, spent: 1, lost: true},
		{name: "retained_before_settlement", boundary: "adapter.result-recorded", phase: "pre-settlement", approval: "selected", effect: runtime.EffectResultUnsettled, gate: true, contacts: 1, spent: 1},
		{name: "settled_before_consumption", boundary: "adapter.settled", phase: "pre-consumption", approval: "selected", effect: runtime.EffectSettled, gate: true, contacts: 1, spent: 1},
		{name: "after_terminal", boundary: "execution.committed", phase: "terminal", sequence: at.terminal, approval: "selected", effect: runtime.EffectSettled, gate: true, contacts: 1, spent: 1, terminal: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather}, weatherReply{status: 200, body: laterWeather, hold: c.boundary == ""})
			db := startWeather(t, binary, p.endpoint(), 2)
			weatherCommand(t, binary, "run", "--db", db)
			firstContact := p.waitAccepted(t)
			initial := weatherSnapshot(t, db)
			first := weatherEffect(t, initial, "weather-1")
			if firstContact.operation != first.OperationID || secondApproval(t, initial) != nil {
				t.Fatal("incorrect pre-approval reference state")
			}
			action := "resume"
			if c.boundary == "continuation.approval" {
				action = "approve-second"
			} else {
				weatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
			}
			var owner *weatherOwner
			var witness string
			if c.boundary == "" {
				owner = launchWeather(t, binary, db, action, nil)
				contact := p.waitAccepted(t)
				if contact.index != 2 || contact.operation != at.operation {
					t.Fatalf("wrong operation at independent endpoint: %+v", contact)
				}
				witness = fmt.Sprintf("endpoint accepted index=%d operation=%s, response withheld", contact.index, contact.operation)
			} else {
				filter := weatherCut{Subject: "weather-2", Generation: 1, Boundary: c.boundary, Phase: c.phase, Sequence: c.sequence}
				switch {
				case c.boundary == "continuation.approval":
					filter.Intent, filter.Attempt = first.OperationID, declaredWeatherSlot(t, initial, 2).ApprovalID
				case c.sequence == at.request || c.effect != "" && c.sequence != at.terminal:
					filter.Intent = at.operation
				}
				owner = launchWeather(t, binary, db, action, &filter)
				witness = owner.awaitCut(t).witness()
			}
			owner.kill(t, witness)
			stopped, v := inspectAndReplayWeather(t, binary, db, p, 2, 1+c.spent)
			p.requireContacts(t, 1+c.contacts)
			c.check(t, at, stopped, v, first)
			approval := secondApproval(t, stopped)
			if approval == nil {
				for range 2 {
					weatherCommand(t, binary, "resume", "--db", db)
				}
				s, _ := inspectAndReplayWeather(t, binary, db, p, 2, 1)
				p.requireContacts(t, 1)
				if len(s.Effects) != 1 || secondApproval(t, s) != nil {
					t.Fatal("reopening invented an approval or successor")
				}
			}
			for range 2 {
				weatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
			}
			p.requireContacts(t, 1+c.contacts)
			for range 2 {
				weatherCommand(t, binary, "resume", "--db", db)
			}
			final, v := inspectAndReplayWeather(t, binary, db, p, 2, 2)
			a := secondApproval(t, final)
			finalDecision, decided := final.Continuation(runtime.RunID{Subject: "weather-2", Generation: 1})
			if !decided {
				t.Fatal("lost approval after fresh-owner recovery")
			}
			if stopDecision, ok := stopped.Continuation(finalDecision.Approval.Target); ok &&
				(stopDecision.Approval != finalDecision.Approval || stopDecision.DecisionOrder != finalDecision.DecisionOrder || stopDecision.ObservationID != finalDecision.ObservationID) {
				t.Fatal("fresh owners enlarged or replaced an existing approval")
			}
			if final.Namespace != initial.Namespace || a == nil || a.SelectedAt == nil || approval != nil && (a.ID != approval.ID || approval.SelectedAt != nil && *a.SelectedAt != *approval.SelectedAt) {
				t.Fatal("reopen lost or changed the recorded approval")
			}
			terminal := continuationCut{name: c.name, approval: "selected", effect: runtime.EffectSettled, gate: true, terminal: true, spent: 1}
			if c.lost {
				terminal.effect, terminal.gate, terminal.terminal = runtime.EffectReservedUnknown, c.gate, false
				unresolvedWeather(t, final, v.read(t, 2), at.operation)
				p.requireContacts(t, 1+c.contacts)
			} else {
				p.requireContacts(t, 2)
				authenticWeather(t, final, v.read(t, 2), 2, p.contacts()[1])
			}
			terminal.check(t, at, final, v, first)
			deniedWeatherCommand(t, binary, "refresh", "--db", db, "--read", "3")
			recordContinuationCase(t, final, p, continuationStop(t, witness, owner.cmd.Process.Pid, 1+c.contacts, stopped))
		})
	}
}

// Check current slots against their compiled source declaration, independently
// of the history's approval observation and admission joins above.
func declaredWeatherSlot(t *testing.T, s *runtime.ExecutionSnapshot, n int) runtime.ContinuationSlot {
	t.Helper()
	c, err := retainedConfiguration(s)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.provision(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, slot := range p.Composition.Continuations {
		if slot.Target == readID(n) {
			for _, actual := range s.Composition.Continuations {
				if actual.Target == slot.Target {
					if actual != slot {
						t.Fatal("retained slot differs from current source")
					}
					return slot
				}
			}
		}
	}
	t.Fatal("missing declared current slot")
	return runtime.ContinuationSlot{}
}
