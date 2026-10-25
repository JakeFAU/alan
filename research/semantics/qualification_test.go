package semantics

import "testing"

func qualificationScenarios() []searchSpec {
	prep := event{Kind: prepare, A: "a", G: 1, C: 1}
	commit0 := event{Kind: commit}
	id := intentID("a", 1, 1)
	auth := event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7}
	apply := event{Kind: applyExternal, ID: id, Attempt: 1}
	receipt := event{Kind: recordReceipt, ID: id, Attempt: 1}
	epochCase := searchSpec{Name: "epoch_and_sequence", Initial: fixture([]instruction{{Op: set, Value: 1}, {Op: success}}), Events: []event{prep, commit0, {Kind: interrupt}, life(reactivate, "a", 1), {Kind: prepare, A: "a", G: 1, C: 1, Slot: 1}, {Kind: commit, Slot: 1}}, Depth: 5, StateCap: 10000, WantWitnesses: []string{"event:Commit", "reactivated", "interruption-preserves-journal"}, WantRejects: []string{"epoch", "sequence"}}
	closureCase := searchSpec{Name: "retirement_authorization_application_receipt", Initial: fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}}), Events: []event{prep, commit0, auth, apply, receipt, life(retire, "a", 1), {Kind: deliver, A: "a", G: 1}}, Depth: 7, StateCap: 10000, WantWitnesses: []string{"retired", "applied-unresolved", "late-settlement", "event:DeliverTerminal"}, WantRejects: []string{"closed"}}
	binding := fixture([]instruction{{Op: fault}})
	binding.Config.Inputs["m"] = message{Source: "env", Target: "a", Tag: "request", Value: 1}
	bindingCase := searchSpec{Name: "service_binding_lost_ack_retry", Initial: binding, Events: []event{{Kind: admit, ID: "m"}, {Kind: acknowledge, ID: "m"}, prep, commit0, life(retry, "a", 1), {Kind: deliver, A: "a", G: 1}}, Depth: 6, StateCap: 10000, WantWitnesses: []string{"duplicate-after-retry-without-ack", "new-generation", "old-notification-after-retry"}}
	late := searchSpec{Name: "abandon_retry_late_settlement", Initial: fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}}), Events: []event{prep, commit0, auth, apply, receipt, life(abandon, "a", 1), life(retry, "a", 1), {Kind: deliver, A: "a", G: 1}}, Depth: 8, StateCap: 20000, WantWitnesses: []string{"applied-unresolved", "late-settlement", "old-notification-after-retry", "new-generation"}}
	cancelFalse := searchSpec{Name: "cancel_retry_policy_false", Initial: fixture([]instruction{{Op: fault}}), Events: []event{life(cancel, "a", 1), life(retry, "a", 1), life(retire, "a", 1), {Kind: deliver, A: "a", G: 1}}, Depth: 4, StateCap: 1000, WantRejects: []string{"retry policy"}, WantWitnesses: []string{"retired", "event:DeliverTerminal"}}
	cancelTrue := copyOf(cancelFalse)
	cancelTrue.Name = "cancel_retry_policy_true"
	cancelTrue.Initial.Config.RetryCancelled = true
	cancelTrue.WantWitnesses = append(cancelTrue.WantWitnesses, "new-generation")
	messages := fixture([]instruction{{Op: receive, Tag: "work"}, {Op: receive, Tag: "work"}, {Op: success}})
	messages.Config.Inputs = map[string]message{"m1": {Source: "env", Target: "a", Tag: "work", Value: 1}, "m2": {Source: "env", Target: "a", Tag: "work", Value: 2}}
	messageCase := searchSpec{Name: "receive_choice_duplicate_interrupt", Initial: messages, Events: []event{{Kind: admit, ID: "m1"}, {Kind: admit, ID: "m2"}, {Kind: prepare, A: "a", G: 1, C: 1, Observation: messageObservationID("m1")}, {Kind: prepare, A: "a", G: 1, C: 1, Observation: messageObservationID("m2")}, commit0, {Kind: interrupt}}, Depth: 7, StateCap: 10000, WantWitnesses: []string{"event:Commit", "interruption-preserves-journal"}, WantRejects: []string{"waiting"}}
	endpoint := searchSpec{Name: "idempotent_ambiguous_attempts", Initial: fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}}), Events: []event{prep, commit0, auth, {Kind: authorize, ID: id, Attempt: 2, Owner: "owner", W: 7}, apply, {Kind: applyExternal, ID: id, Attempt: 2}, receipt, {Kind: recordReceipt, ID: id, Attempt: 2}}, Depth: 8, StateCap: 20000, WantWitnesses: []string{"multiple-attempts-applied", "applied-unresolved", "event:RecordReceipt"}}
	nonidempotent := copyOf(endpoint)
	nonidempotent.Name = "nonidempotent_ambiguous_attempts"
	nonidempotent.Initial.Config.Idempotent = false
	nonidempotent.WantWitnesses = append(nonidempotent.WantWitnesses, "repeated-mutation")
	cases := []searchSpec{epochCase, closureCase, bindingCase, late, cancelFalse, cancelTrue, messageCase, endpoint, nonidempotent}
	for _, entry := range []struct {
		Original          searchSpec
		Mutant, Violation string
	}{{epochCase, "epoch", "commit-epoch"}, {closureCase, "closure", "authorization-after-closure"}, {bindingCase, "binding", "stable-service-binding"}} {
		s := copyOf(entry.Original)
		s.Name = "broken_guard_" + entry.Mutant
		s.Initial.Config.Mutant = entry.Mutant
		s.WantViolation = entry.Violation
		s.WantWitnesses = nil
		s.WantRejects = nil
		cases = append(cases, s)
	}
	wrong := copyOf(cancelTrue)
	wrong.Name = "wrong_policy_cancel_means_retire"
	wrong.Policy = "cancel-prevents-retry"
	wrong.WantViolation = "policy:cancel-prevents-retry"
	wrong.WantWitnesses = nil
	wrong.WantRejects = nil
	cases = append(cases, wrong)
	lateCancel := searchSpec{Name: "wrong_policy_late_cancel_stops_retry", Initial: fixture([]instruction{{Op: fault}}), Events: []event{prep, commit0, life(cancel, "a", 1), life(retry, "a", 1)}, Depth: 4, StateCap: 1000, Policy: "cancel-prevents-retry", WantViolation: "policy:cancel-prevents-retry"}
	cases = append(cases, lateCancel)
	return cases
}

func TestQualification(t *testing.T) {
	for _, spec := range qualificationScenarios() {
		t.Run(spec.Name, func(t *testing.T) {
			stats, err := explore(spec)
			t.Log("QUALIFICATION " + canonical(stats))
			if err != nil {
				t.Fatal(err)
			}
			if err := expectation(stats, spec); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("static_control_witness", func(t *testing.T) {
		m, trace := controlWitness(t)
		t.Log("QUALIFICATION " + canonical(struct {
			Name                                                                                            string
			Bounds                                                                                          bounds
			Subjects, Events, Commits, Intents, Attempts, Applications, Admissions, Receipts, Notifications int
			Trace                                                                                           []string
		}{
			Name: "static_control_witness", Bounds: m.Config.Bounds, Subjects: 3, Events: len(trace), Commits: 20, Intents: len(m.Intents), Attempts: len(m.Attempts), Applications: len(m.World.Applied), Admissions: len(m.Admissions), Receipts: len(m.Receipts), Notifications: len(m.Delivered), Trace: trace,
		}))
	})
}

func TestExplorerResourceCapFails(t *testing.T) {
	spec := qualificationScenarios()[0]
	spec.StateCap = 1
	if _, err := explore(spec); err == nil {
		t.Fatal("hard resource cap was reported as a successful bounded search")
	}
}

func controlWitness(t *testing.T) (machine, []string) {
	t.Helper()
	c := fixture([]instruction{{Op: success}}).Config
	c.Programs = map[string][]instruction{
		"control": {{Op: send, Tag: "job-b", Target: "b", Value: 1}, {Op: send, Tag: "job-c", Target: "c", Value: 1, Ref: 1}, {Op: receive, Tag: "receipt:job-b", Reg: 1, Receipt: true}, {Op: receive, Tag: "receipt:job-c", Reg: 1, Receipt: true, Ref: 1}, {Op: receive, Tag: "result-b"}, {Op: receive, Tag: "result-c"}, {Op: branch, Value: 2, PC: 8}, {Op: fault}, {Op: success}},
		"b":       {{Op: receive, Tag: "job-b"}, {Op: emit, Tag: "task", Value: 1}, {Op: receive, Tag: "receipt:task", Reg: 1, Receipt: true}, {Op: send, Tag: "result-b", Target: "control", Value: 1}, {Op: receive, Tag: "receipt:result-b", Reg: 1, Receipt: true}, {Op: success}},
		"c":       {{Op: receive, Tag: "job-c"}, {Op: emit, Tag: "task", Value: 1}, {Op: receive, Tag: "receipt:task", Reg: 1, Receipt: true}, {Op: send, Tag: "result-c", Target: "control", Value: 1}, {Op: receive, Tag: "receipt:result-c", Reg: 1, Receipt: true}, {Op: success}},
	}
	c.Routes = map[string]string{"control": "caller", "b": "audit-b", "c": "audit-c"}
	c.Edges = map[string]bool{"control>b": true, "control>c": true, "b>control": true, "c>control": true}
	m := newMachine(c)
	var trace []string
	advance := func(e event) { mustStep(t, &m, e); trace = append(trace, e.String()) }
	local := func(a, obs string) {
		advance(event{Kind: prepare, A: a, G: 1, C: 1, Observation: obs})
		advance(event{Kind: commit})
	}
	dispatch := func(id string, isSend bool) {
		advance(event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
		advance(event{Kind: applyExternal, ID: id, Attempt: 1})
		if isSend {
			advance(event{Kind: admit, ID: id, Origin: "send"})
		}
		advance(event{Kind: recordReceipt, ID: id, Attempt: 1})
	}
	local("control", "")
	local("control", "")
	// Child c is delivered/completes first. Control still chooses its own receive order.
	dispatch(intentID("control", 1, 2), true)
	dispatch(intentID("control", 1, 1), true)
	for _, child := range []string{"c", "b"} {
		n := 1
		if child == "c" {
			n = 2
		}
		local(child, sendObservationID(intentID("control", 1, n)))
		local(child, "")
		dispatch(intentID(child, 1, 2), false)
		local(child, "receipt:"+intentID(child, 1, 2))
		local(child, "")
		dispatch(intentID(child, 1, 4), true)
		local(child, "receipt:"+intentID(child, 1, 4))
		local(child, "")
	}
	local("control", "receipt:"+intentID("control", 1, 1))
	local("control", "receipt:"+intentID("control", 1, 2))
	local("control", sendObservationID(intentID("b", 1, 4)))
	local("control", sendObservationID(intentID("c", 1, 4)))
	local("control", "")
	local("control", "")
	commits := 0
	for _, a := range []string{"control", "b", "c"} {
		r := m.Runs[runID(a, 1)]
		if r.Projection.Status != "succeeded" {
			t.Fatalf("%s did not finish", a)
		}
		commits += r.N
		advance(event{Kind: deliver, A: a, G: 1})
		before := canonical(m)
		if err := replay(m, runID(a, 1), "v1"); err != nil {
			t.Fatal(err)
		}
		if canonical(m) != before {
			t.Fatal("Control replay dispatched work")
		}
	}
	if commits != 20 || m.Runs[runID("control", 1)].Projection.Data[0] != 2 || m.World.Mutations != 2 {
		t.Fatal("Control result aggregation did not match two children")
	}
	return m, trace
}
