package semantics

import "testing"

func TestPrepareInterruptAndAtomicCommit(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}})
	before := canonical(m.Runs)
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Slot: 0})
	if canonical(m.Runs) != before || len(m.Intents) != 0 || len(m.Attempts) != 0 || m.World.Mutations != 0 {
		t.Fatal("prepare escaped its speculative boundary")
	}
	mustStep(t, &m, event{Kind: interrupt, Slot: 0})
	if canonical(m.Runs) != before {
		t.Fatal("interruption changed journal")
	}
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Slot: 0})
	mustStep(t, &m, event{Kind: commit, Slot: 0})
	r := m.Runs[runID("a", 1)]
	if r.N != 1 || len(m.Intents) != 1 || len(m.Attempts) != 0 || m.World.Mutations != 0 {
		t.Fatal("commit must atomically record state and intent, without dispatch")
	}
}

func TestStaleEpochAndCompetingSequence(t *testing.T) {
	m := fixture([]instruction{{Op: set, Value: 1}, {Op: success}})
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Slot: 0})
	mustStep(t, &m, life(reactivate, "a", 1))
	mustReject(t, &m, event{Kind: commit, Slot: 0}, "epoch")
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Slot: 1})
	mustStep(t, &m, event{Kind: interrupt, Slot: 0})
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Slot: 0})
	mustStep(t, &m, event{Kind: commit, Slot: 1})
	mustReject(t, &m, event{Kind: commit, Slot: 0}, "sequence")
}

func TestAuthorizedApplicationAndLateReceiptRemainSeparate(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}})
	turn(t, &m, "a", 1, "")
	id := intentID("a", 1, 1)
	mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
	mustStep(t, &m, life(retire, "a", 1))
	mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: 1})
	if m.World.Mutations != 1 || m.Intents[id].Settled {
		t.Fatal("application implied receipt")
	}
	frozen := canonical(m.Runs[runID("a", 1)])
	mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
	if !m.Intents[id].Settled || canonical(m.Runs[runID("a", 1)]) != frozen {
		t.Fatal("late receipt must settle evidence without reviving or changing the closed journal")
	}
}

func TestOwnershipAndGenerationAreSeparate(t *testing.T) {
	for _, k := range []kind{reactivate, abandon, cancel, retire, retry} {
		t.Run(string(k), func(t *testing.T) {
			m := fixture([]instruction{{Op: fault}})
			stale := life(k, "a", 1)
			stale.W = 1 // matches g/e, but not ownership revision 7
			mustReject(t, &m, stale, "ownership")
			stale.W = 7
			stale.Owner = "stranger"
			mustReject(t, &m, stale, "ownership")
		})
	}
	m := fixture([]instruction{{Op: emit}, {Op: fault}})
	turn(t, &m, "a", 1, "")
	id := intentID("a", 1, 1)
	mustReject(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 1}, "ownership")
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1})
	mustStep(t, &m, life(abandon, "a", 1))
	mustStep(t, &m, life(retry, "a", 1))
	mustReject(t, &m, event{Kind: commit}, "generation")
	mustReject(t, &m, life(abandon, "a", 1), "generation")
	mustStep(t, &m, life(cancel, "a", 1))
	if m.Runs[runID("a", 2)].Projection.Status != "active" {
		t.Fatal("old cancellation redirected to new run")
	}
}

func TestEveryTerminalStatusFencesNewAttempts(t *testing.T) {
	for _, name := range []string{"succeeded", "failed", "abandoned", "cancelled", "retired"} {
		t.Run(name, func(t *testing.T) {
			last := success
			if name == "failed" {
				last = fault
			}
			m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: last}})
			turn(t, &m, "a", 1, "")
			id := intentID("a", 1, 1)
			mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
			switch name {
			case "succeeded", "failed":
				turn(t, &m, "a", 1, "")
			case "abandoned":
				mustStep(t, &m, life(abandon, "a", 1))
			case "cancelled":
				mustStep(t, &m, life(cancel, "a", 1))
			case "retired":
				mustStep(t, &m, life(retire, "a", 1))
			}
			mustReject(t, &m, event{Kind: authorize, ID: id, Attempt: 2, Owner: "owner", W: 7}, "closed")
			mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: 1})
			closed := canonical(m.Runs)
			mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
			mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
			if canonical(m.Runs) != closed || m.World.Mutations != 1 || len(m.Receipts) != 1 {
				t.Fatal("late or duplicate settlement changed closed computation")
			}
		})
	}
}

func TestCancellationPoliciesAndLateCancel(t *testing.T) {
	for _, retryCancelled := range []bool{false, true} {
		name := "retryCancelled=false"
		if retryCancelled {
			name = "retryCancelled=true"
		}
		t.Run(name, func(t *testing.T) {
			m := fixture([]instruction{{Op: fault}})
			m.Config.RetryCancelled = retryCancelled
			mustStep(t, &m, life(cancel, "a", 1))
			if retryCancelled {
				mustStep(t, &m, life(retry, "a", 1))
			} else {
				mustReject(t, &m, life(retry, "a", 1), "retry policy")
			}
			mustStep(t, &m, life(retire, "a", m.Subjects["a"].Current))
			mustReject(t, &m, life(retry, "a", m.Subjects["a"].Current), "retry policy")
		})
	}
	for _, ended := range []string{"failed", "abandoned"} {
		t.Run("late_cancel_"+ended, func(t *testing.T) {
			m := fixture([]instruction{{Op: fault}})
			if ended == "failed" {
				turn(t, &m, "a", 1, "")
			} else {
				mustStep(t, &m, life(abandon, "a", 1))
			}
			frozen := canonical(m.Runs)
			mustStep(t, &m, life(cancel, "a", 1))
			if canonical(m.Runs) != frozen {
				t.Fatal("late cancellation overwrote existing terminal outcome")
			}
			mustStep(t, &m, life(retry, "a", 1))
			if m.Config.RetryCancelled || m.Subjects["a"].Current != 2 {
				t.Fatal("late cancellation unexpectedly stopped eligible retry")
			}
		})
	}
}

func TestAdmissionDuplicatesReorderingAndReceiptCorrelation(t *testing.T) {
	m := fixture([]instruction{{Op: receive, Tag: "work"}, {Op: receive, Tag: "work"}, {Op: success}})
	m.Config.Inputs = map[string]message{"m1": {Source: "env", Target: "a", Tag: "work", Value: 1}, "m2": {Source: "env", Target: "a", Tag: "work", Value: 2}}
	mustStep(t, &m, event{Kind: admit, ID: "m2"})
	mustStep(t, &m, event{Kind: admit, ID: "m1"})
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Observation: messageObservationID("m1")})
	mustStep(t, &m, event{Kind: interrupt})
	if m.Runs[runID("a", 1)].Inbox[messageObservationID("m1")].Consumed {
		t.Fatal("interruption consumed selected message")
	}
	turn(t, &m, "a", 1, messageObservationID("m1"))
	mustStep(t, &m, event{Kind: admit, ID: "m1"})
	mustReject(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Observation: messageObservationID("m1")}, "waiting")
	turn(t, &m, "a", 1, messageObservationID("m2"))
	if m.Runs[runID("a", 1)].Projection.Data[0] != 3 || len(m.Runs[runID("a", 1)].Inbox) != 2 {
		t.Fatal("receive selection or dedup incorrect")
	}
	m.Config.Inputs["m1"] = message{Source: "env", Target: "a", Tag: "work", Value: 7}
	mustReject(t, &m, event{Kind: admit, ID: "m1"}, "descriptor conflict")
	mustReject(t, &m, event{Kind: recordReceipt, ID: "forged", Attempt: 1}, "receipt correlation")
}

func TestServiceBindingSurvivesLostAcknowledgmentAndRetry(t *testing.T) {
	m := fixture([]instruction{{Op: fault}})
	m.Config.Inputs["request"] = message{Source: "env", Target: "a", Tag: "work", Value: 1}
	mustStep(t, &m, event{Kind: admit, ID: "request"})
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, life(retry, "a", 1))
	mustStep(t, &m, event{Kind: admit, ID: "request"})
	if ad := m.Admissions[transportID("", "request")]; ad.Ack || ad.Bound != runID("a", 1) || !ad.Accepted {
		t.Fatal("duplicate rebound or fabricated acknowledgement")
	}
	if len(m.Runs[runID("a", 2)].Inbox) != 0 {
		t.Fatal("duplicate admitted into later generation")
	}
	mustStep(t, &m, event{Kind: acknowledge, ID: "request"})
	m.Config.Inputs["exact-old-run"] = message{Source: "env", Target: "a", G: 1, Tag: "work"}
	mustStep(t, &m, event{Kind: admit, ID: "exact-old-run"})
	if m.Admissions[transportID("", "exact-old-run")].Accepted {
		t.Fatal("closed run target accepted new message")
	}
}

func TestEndpointAmbiguityAndGenerationScopedIdentity(t *testing.T) {
	for _, idempotent := range []bool{true, false} {
		name := "nonidempotent"
		if idempotent {
			name = "idempotent"
		}
		t.Run(name, func(t *testing.T) {
			m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: fault}})
			m.Config.Idempotent = idempotent
			turn(t, &m, "a", 1, "")
			id := intentID("a", 1, 1)
			for _, n := range []int{1, 2} {
				mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: n, Owner: "owner", W: 7})
				mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: n})
			}
			want := 2
			if idempotent {
				want = 1
			}
			if m.World.Mutations != want || m.Intents[id].Settled {
				t.Fatal("lost receipt confused unknown completion with nonexecution")
			}
			mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 2})
			mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
			if len(m.Runs[runID("a", 1)].Inbox) != 1 || len(m.Receipts) != 2 {
				t.Fatal("reordered receipts created duplicate observations")
			}
			turn(t, &m, "a", 1, "")
			mustStep(t, &m, life(retry, "a", 1))
			turn(t, &m, "a", 1, "")
			newID := intentID("a", 2, 1)
			mustStep(t, &m, event{Kind: authorize, ID: newID, Attempt: 1, Owner: "owner", W: 7})
			mustStep(t, &m, event{Kind: applyExternal, ID: newID, Attempt: 1})
			if m.World.Mutations != want+1 {
				t.Fatal("generation identity silently deduplicated business operation")
			}
		})
	}
}

func TestWorldIsInvisibleToOrdinaryDecisions(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: success}})
	turn(t, &m, "a", 1, "")
	id := intentID("a", 1, 1)
	mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
	applied := copyOf(m)
	mustStep(t, &applied, event{Kind: applyExternal, ID: id, Attempt: 1})
	for _, e := range []event{{Kind: prepare, A: "a", G: 1, C: 1}, life(reactivate, "a", 1), life(abandon, "a", 1), life(cancel, "a", 1), life(retire, "a", 1), {Kind: authorize, ID: id, Attempt: 2, Owner: "owner", W: 7}} {
		left, lwhy := step(m, e)
		right, rwhy := step(applied, e)
		right.World = left.World
		if lwhy != rwhy || canonical(left) != canonical(right) {
			t.Fatalf("%s inspected unrecorded external world", e.Kind)
		}
	}
}

func TestReadyContinuationProgressAndFaultVersusInterruption(t *testing.T) {
	m := fixture([]instruction{{Op: fork, PC: 3}, {Op: receive, Tag: "never"}, {Op: done}, {Op: set, Value: 2}, {Op: done}})
	turn(t, &m, "a", 1, "")
	mustReject(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1}, "waiting")
	turn(t, &m, "a", 2, "")
	turn(t, &m, "a", 2, "")
	if r := m.Runs[runID("a", 1)]; r.N != 3 || r.Projection.Data[0] != 2 || len(r.Projection.Frontier) != 1 {
		t.Fatal("waiting continuation blocked ready sibling")
	}
	m = fixture([]instruction{{Op: fault}})
	mustStep(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1})
	mustStep(t, &m, event{Kind: interrupt})
	mustReject(t, &m, life(retry, "a", 1), "retry policy")
	turn(t, &m, "a", 1, "")
	if m.Runs[runID("a", 1)].Projection.Status != "failed" {
		t.Fatal("defined fault did not end run")
	}
	mustStep(t, &m, life(retry, "a", 1))
}

func TestFrozenNotificationDeliveryAndReplaySensitivity(t *testing.T) {
	m := fixture([]instruction{{Op: receive, Tag: "work"}, {Op: emit, Tag: "charge", Value: 2}, {Op: fault}})
	m.Config.Inputs["m"] = message{Source: "env", Target: "a", Tag: "work", Value: 3}
	mustStep(t, &m, event{Kind: admit, ID: "m"})
	turn(t, &m, "a", 1, messageObservationID("m"))
	mustStep(t, &m, life(reactivate, "a", 1))
	turn(t, &m, "a", 1, "")
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, life(retry, "a", 1))
	current := canonical(m.Runs[runID("a", 2)])
	mustStep(t, &m, event{Kind: deliver, A: "a", G: 1})
	mustStep(t, &m, event{Kind: deliver, A: "a", G: 1})
	if current != canonical(m.Runs[runID("a", 2)]) || m.Delivered[runID("a", 1)] != (outcome{Status: "failed", Value: 3, Route: "observer"}) {
		t.Fatal("old notification changed correlation or revived computation")
	}
	before := canonical(m)
	if err := replay(m, runID("a", 1), "v1"); err != nil {
		t.Fatal(err)
	}
	if canonical(m) != before {
		t.Fatal("replay mutated live state or dispatched historical work")
	}
	if err := replay(m, runID("a", 1), "v2"); err == nil {
		t.Fatal("wrong version replay accepted")
	}
	for _, field := range []string{"data", "frontier", "selection", "payload", "intent", "notification", "epoch", "sequence"} {
		t.Run(field, func(t *testing.T) {
			bad := copyOf(m)
			r := bad.Runs[runID("a", 1)]
			switch field {
			case "data":
				r.Projection.Data[0]++
			case "frontier":
				r.Projection.Frontier = []continuation{{ID: 1, PC: 0}}
			case "selection":
				r.History[0].Selection.Observation = "other"
			case "payload":
				o := r.Inbox[messageObservationID("m")]
				o.Value++
				r.Inbox[messageObservationID("m")] = o
			case "intent":
				bad.Intents[intentID("a", 1, 2)].Descriptor.Value++
			case "notification":
				r.Outcome.Route = "attacker"
			case "epoch":
				r.E++
			case "sequence":
				r.N++
			}
			if replay(bad, runID("a", 1), "v1") == nil {
				t.Fatal("tampered replay accepted")
			}
		})
	}
}

func TestReceiptReceiveRejectsSpoofedOrdinaryMessage(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: receive, Tag: "receipt:charge", Receipt: true}, {Op: success}})
	m.Config.Inputs["spoof"] = message{Source: "env", Target: "a", Tag: "receipt:charge", Value: 1}
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, event{Kind: admit, ID: "spoof"})
	mustReject(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Observation: messageObservationID("spoof")}, "correlation")
}

func TestSameTagReceiptMustMatchSelectedEffectReference(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1, Ref: 0}, {Op: emit, Tag: "charge", Value: 1, Ref: 1}, {Op: receive, Tag: "receipt:charge", Receipt: true, Ref: 0}, {Op: success}})
	turn(t, &m, "a", 1, "")
	turn(t, &m, "a", 1, "")
	second := intentID("a", 1, 2)
	first := intentID("a", 1, 1)
	for _, id := range []string{second, first} {
		mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
		mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: 1})
		mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
		if id == second {
			mustReject(t, &m, event{Kind: prepare, A: "a", G: 1, C: 1, Observation: "receipt:" + second}, "correlation")
		}
	}
	turn(t, &m, "a", 1, "receipt:"+first)
	if m.Runs[runID("a", 1)].Inbox["receipt:"+second].Consumed {
		t.Fatal("wrong same-tag effect consumed")
	}
	if err := replay(m, runID("a", 1), "v1"); err != nil {
		t.Fatal(err)
	}
}

func TestIndependentMonitorDetectsSpoofedReceiptConsumption(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: receive, Tag: "receipt:charge", Receipt: true}, {Op: success}})
	m.Config.Inputs["spoof"] = message{Source: "env", Target: "a", Tag: "receipt:charge", Value: 1}
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, event{Kind: admit, ID: "spoof"})
	r := m.Runs[runID("a", 1)]
	o := r.Inbox[messageObservationID("spoof")]
	// Simulate a faulty reducer that ignores correlation, without invoking its guard.
	p := &proposal{A: "a", G: 1, E: r.E, N: r.N, Version: r.Version, Selection: selection{C: 1, Observation: messageObservationID("spoof")}, Observed: &o, After: copyOf(r.Projection)}
	p.After.Data[0]++
	p.After.Frontier[0].PC++
	m.Proposals[0] = p
	next, why := step(m, event{Kind: commit})
	if why != "" {
		t.Fatal("redundant guard concealed checker control:", why)
	}
	if _, violation := inspect(m, next, event{Kind: commit}, freshMonitor()); violation != "receive-correlation" {
		t.Fatalf("monitor violation = %q", violation)
	}
}

func TestSendReceiptAcknowledgesAdmission(t *testing.T) {
	m, _ := controlWitness(t)
	for id, ad := range m.Admissions {
		if !ad.Ack {
			t.Fatalf("Send receipt for %s exposed acceptance without acknowledgment", id)
		}
	}
}

func TestOrdinaryMessageCannotAliasReceiptObservation(t *testing.T) {
	m := fixture([]instruction{{Op: emit, Tag: "charge", Value: 1}, {Op: receive, Tag: "ordinary"}, {Op: receive, Tag: "receipt:charge", Receipt: true}, {Op: success}})
	id := intentID("a", 1, 1)
	raw := "receipt:" + id
	m.Config.Inputs[raw] = message{Source: "env", Target: "a", Tag: "ordinary", Value: 3}
	mustStep(t, &m, event{Kind: admit, ID: raw})
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
	mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: 1})
	mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
	r := m.Runs[runID("a", 1)]
	if len(r.Inbox) != 2 || r.Inbox[messageObservationID(raw)].Kind != "message" || r.Inbox[messageObservationID(raw)].Value != 3 {
		t.Fatal("receipt overwrote an accepted ordinary message with a colliding raw identity")
	}
	turn(t, &m, "a", 1, messageObservationID(raw))
	turn(t, &m, "a", 1, "receipt:"+id)
	if m.Runs[runID("a", 1)].Projection.Data[0] != 4 {
		t.Fatal("both distinct observations were not receivable")
	}
	if err := replay(m, runID("a", 1), "v1"); err != nil {
		t.Fatal(err)
	}
}

func TestExternalRawIDCannotAliasGeneratedSendAdmission(t *testing.T) {
	c := fixture([]instruction{{Op: send, Tag: "job", Target: "b", Value: 1}, {Op: receive, Tag: "receipt:job", Receipt: true}, {Op: success}}).Config
	c.Programs["b"] = []instruction{{Op: success}}
	c.Routes["b"] = "observer-b"
	c.Edges["a>b"] = true
	c.Edges["env>b"] = true
	id := intentID("a", 1, 1)
	c.Inputs[id] = message{Source: "env", Target: "b", Tag: "unrelated", Value: 3}
	m := newMachine(c)
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, event{Kind: authorize, ID: id, Attempt: 1, Owner: "owner", W: 7})
	mustStep(t, &m, event{Kind: applyExternal, ID: id, Attempt: 1})
	mustStep(t, &m, event{Kind: admit, ID: id})
	mustReject(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1}, "no admission")
	mustStep(t, &m, event{Kind: admit, ID: id, Origin: "send"})
	mustStep(t, &m, event{Kind: recordReceipt, ID: id, Attempt: 1})
	if len(m.Runs[runID("b", 1)].Inbox) != 2 {
		t.Fatal("ingress and generated Send with equal raw IDs did not coexist")
	}
	turn(t, &m, "a", 1, "receipt:"+id)
	if m.Runs[runID("a", 1)].Projection.Data[0] != 1 {
		t.Fatal("matching Send admission was not received")
	}
}
