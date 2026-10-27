package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func addProgram(t *testing.T, p *Provision, subject string, code []Instruction, data map[string]Value) {
	t.Helper()
	b, id, err := EncodeProgram(Program{Semantics: SemanticVersion, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: SemanticVersion, Bytes: b})
	p.Subjects = append(p.Subjects, Subject{ID: subject, Generation: 1, Owner: "owner", Revision: 1})
	p.Runs = append(p.Runs, InitialRun{ID: RunID{subject, 1}, ProgramID: id, Projection: Projection{Status: "open", Data: data}})
}
func inputProgram() []Instruction {
	return []Instruction{{Op: "receive", Match: &Match{Kind: "input", Provenance: "local-input", Source: Lit(Text("local")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Operation: Lit(Text(""))}, Bind: map[string]string{"number": "number"}}, {Op: "end", Outcome: "succeeded"}}
}
func messageStore(t *testing.T, p Provision) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "messages.sqlite")
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, path
}
func inputFixture(t *testing.T) Provision {
	t.Helper()
	p := Provision{}
	addProgram(t, &p, "receiver", inputProgram(), map[string]Value{})
	p.Permissions = []Permission{{Kind: "input", Source: "local", Target: "receiver", Operation: "input", Obligation: "work"}}
	return p
}
func current(t *testing.T, s *Store, subject string) RunState {
	t.Helper()
	r, err := s.Snapshot(t.Context(), RunID{subject, 1})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestInputAdmissionImmutableAndLogicalDeduplication(t *testing.T) {
	s, path := messageStore(t, inputFixture(t))
	request := Input{TransportID: "t1", LogicalID: "l1", Target: "receiver", Obligation: "work", Payload: map[string]Value{"number": Int(500)}}
	first, err := s.AdmitInput(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Accepted || first.Bound == nil || *first.Bound != (RunID{"receiver", 1}) {
		t.Fatalf("admission %+v", first)
	}
	duplicate, err := s.AdmitInput(t.Context(), request)
	if err != nil || !reflect.DeepEqual(first, duplicate) {
		t.Fatalf("duplicate %+v %v", duplicate, err)
	}
	conflicting := request
	conflicting.Obligation = "changed"
	if _, err = s.AdmitInput(t.Context(), conflicting); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed transport descriptor: %v", err)
	}
	request.TransportID = "t2"
	if _, err = s.AdmitInput(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if count(t, s, "observations") != 1 || count(t, s, "admissions") != 2 {
		t.Fatal("transport/logical identities conflated")
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || report.Steps != 2 || current(t, s, "receiver").Projection.Data["number"] != Int(500) {
		t.Fatalf("automatic receive %+v", report)
	}
	request.TransportID = "t1"
	again, err := s.AdmitInput(t.Context(), request)
	if err != nil || !reflect.DeepEqual(again, first) {
		t.Fatalf("closed-run retransmit %+v %v", again, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedAdmissionIsRetainedAndCannotForgeRuntimeOrigin(t *testing.T) {
	s, _ := messageStore(t, inputFixture(t))
	request := Input{TransportID: "denied", LogicalID: "logical", Target: "unknown", Payload: map[string]Value{"Kind": Text("terminal"), "Provenance": Text("runtime-terminal")}}
	rejection, err := s.AdmitInput(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if rejection.Accepted || rejection.Reason == "" || count(t, s, "observations") != 0 {
		t.Fatalf("missing denial %+v", rejection)
	}
	again, err := s.AdmitInput(t.Context(), request)
	if err != nil || !reflect.DeepEqual(again, rejection) {
		t.Fatalf("denial lost %+v %v", again, err)
	}
	request.TransportID = "ordinary"
	request.LogicalID = "ordinary"
	request.Target = "receiver"
	request.Obligation = "work"
	result, err := s.AdmitInput(t.Context(), request)
	if err != nil || !result.Accepted {
		t.Fatalf("ordinary input %+v %v", result, err)
	}
	var raw []byte
	if err = s.conn.QueryRowContext(t.Context(), "SELECT descriptor FROM observations WHERE id=?", result.ObservationID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var observation Observation
	if err = json.Unmarshal(raw, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Kind != "input" || observation.Provenance != "local-input" || observation.Source != "local" {
		t.Fatalf("forged input origin %+v", observation)
	}
}

func TestMessageCommitAdmissionAndLostAcknowledgment(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "sender", []Instruction{{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(0)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(17))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	receiver := inputProgram()
	receiver[0].Match.Kind = "message"
	receiver[0].Match.Provenance = "ordinary-message"
	receiver[0].Match.Source = Lit(Text("sender"))
	addProgram(t, &p, "receiver", receiver, map[string]Value{})
	p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "work"}}
	s, path := messageStore(t, p)
	if _, err := s.DeliverMessage(t.Context(), "uncommitted", "transport"); err == nil {
		t.Fatal("uncommitted effect dispatched")
	}
	emitted, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	intentID := `["sender",1,1,0]`
	first, err := s.DeliverMessage(t.Context(), intentID, "transport")
	if err != nil || !first.Accepted {
		t.Fatalf("message %+v %v", first, err)
	}
	if _, err = s.Commit(t.Context(), emitted.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	// Delivery acknowledgement is deliberately absent; retained authorization and
	// admission survive sender closure and physical reopening.
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	again, err := reopened.DeliverMessage(t.Context(), intentID, "transport")
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatalf("lost acknowledgment %+v %v", again, err)
	}
	if err = reopened.AcknowledgeMessage(t.Context(), intentID, "transport"); err != nil {
		t.Fatal(err)
	}
	if count(t, reopened, "observations") != 1 || count(t, reopened, "message_acknowledgments") != 1 {
		t.Fatal("duplicate message/ack")
	}
	report, err := reopened.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || current(t, reopened, "receiver").Projection.Data["number"] != Int(17) {
		t.Fatal(report)
	}
}

func TestTerminalDeliveryAfterClosureAndLostAcknowledgment(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "child", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, &p, "observer", []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text("child")), Generation: Lit(Int(1)), Obligation: Lit(Text("obligation")), Operation: Lit(Text(""))}, Bind: map[string]string{"outcome": "outcome"}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "observation", Source: "child", Target: "observer", Operation: "terminal", Revision: 1, Obligation: "obligation"}}
	s, path := messageStore(t, p)
	if _, err := s.DeliverTerminal(t.Context(), RunID{"child", 1}, TerminalDestination{Target: "observer", Obligation: "obligation"}, "early"); err == nil {
		t.Fatal("uncommitted terminal delivered")
	}
	if _, err := s.Commit(t.Context(), current(t, s, "child").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	destination := TerminalDestination{Target: "observer", Obligation: "obligation"}
	first, err := s.DeliverTerminal(t.Context(), RunID{"child", 1}, destination, "notice")
	if err != nil || !first.Accepted {
		t.Fatalf("notice %+v %v", first, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	again, err := s.DeliverTerminal(t.Context(), RunID{"child", 1}, destination, "notice")
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatalf("notice replay %+v %v", again, err)
	}
	if err = s.AcknowledgeTerminal(t.Context(), RunID{"child", 1}, destination, "notice"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10}); err != nil {
		t.Fatal(err)
	}
	if current(t, s, "observer").Projection.Data["outcome"] != Text("failed") || count(t, s, "observations") != 1 {
		t.Fatal("terminal not consumed exactly once")
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionFailureRollsBackInboxInsertion(t *testing.T) {
	s, _ := messageStore(t, inputFixture(t))
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_admission BEFORE INSERT ON admissions BEGIN SELECT RAISE(ABORT,'test admission rollback'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AdmitInput(t.Context(), Input{TransportID: "t", LogicalID: "l", Target: "receiver", Payload: map[string]Value{}}); err == nil {
		t.Fatal("expected admission failure")
	}
	if count(t, s, "admissions") != 0 || count(t, s, "observations") != 0 {
		t.Fatal("partial admission/inbox mutation")
	}
}
func TestClosedSourceDenialAndMissingRouteRemainRetained(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(fmt.Sprint(closed), func(t *testing.T) {
			p := Provision{}
			addProgram(t, &p, "sender", []Instruction{{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text(""))}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			addProgram(t, &p, "receiver", inputProgram(), map[string]Value{})
			s, path := messageStore(t, p)
			emitted, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			if closed {
				if _, err = s.Commit(t.Context(), emitted.Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.DeliverMessage(t.Context(), `["sender",1,1,0]`, "denied")
			if err != nil {
				t.Fatal(err)
			}
			expected := "route_denied"
			if closed {
				expected = "source_denied:source_closed"
			}
			if result.Accepted || result.Reason != expected || count(t, s, "attempts") != 1 || count(t, s, "observations") != 0 {
				t.Fatalf("denial %+v", result)
			}
			again, err := s.DeliverMessage(t.Context(), `["sender",1,1,0]`, "denied")
			if err != nil || !reflect.DeepEqual(result, again) {
				t.Fatalf("denial changed %+v %v", again, err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestReplayRejectsAdmissionAndAuthenticatedOriginCorruption(t *testing.T) {
	for _, query := range []string{
		`UPDATE admissions SET result=replace(CAST(result AS TEXT),'"Accepted":true','"Accepted":false')`,
		`UPDATE observations SET descriptor=replace(CAST(descriptor AS TEXT),'local-input','runtime-terminal')`,
		`DELETE FROM admissions`,
	} {
		t.Run(query, func(t *testing.T) {
			s, path := messageStore(t, inputFixture(t))
			if _, err := s.AdmitInput(t.Context(), Input{TransportID: "t", LogicalID: "l", Target: "receiver", Obligation: "work", Payload: map[string]Value{}}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("unexplained authenticated observation/admission accepted")
			}
		})
	}
}

func TestTerminalAcknowledgmentRequiresRetainedAdmission(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "child", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, &p, "observer", inputProgram(), map[string]Value{})
	destination := TerminalDestination{Target: "observer", Obligation: "public"}
	p.Permissions = []Permission{{Kind: "observation", Source: "child", Target: "observer", Operation: "terminal", Revision: 1, Obligation: "public"}}
	s, path := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "child").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.AcknowledgeTerminal(t.Context(), RunID{"child", 1}, destination, "missing"); err == nil {
		t.Fatal("acknowledgment invented admission")
	}
	raw, err := json.Marshal(map[string]bool{destination.key(): true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE terminal_obligations SET delivery_status=? WHERE subject='child'", raw); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("phantom delivery acknowledgment accepted")
	}
}

func TestAdmissionEnforcesExactPermissionObligation(t *testing.T) {
	t.Run("input obligation enforcement and multiple grants", func(t *testing.T) {
		p := Provision{}
		code := []Instruction{
			{Op: "receive", Match: &Match{Kind: "input", Provenance: "local-input", Source: Lit(Text("local")), Generation: Lit(Int(1)), Obligation: Lit(Text("public_ob")), Operation: Lit(Text(""))}, Bind: map[string]string{"value": "public_val"}},
			{Op: "receive", Match: &Match{Kind: "input", Provenance: "local-input", Source: Lit(Text("local")), Generation: Lit(Int(1)), Obligation: Lit(Text("audit_ob")), Operation: Lit(Text(""))}, Bind: map[string]string{"value": "audit_val"}},
			{Op: "end", Outcome: "succeeded"},
		}
		addProgram(t, &p, "receiver", code, map[string]Value{})
		p.Permissions = []Permission{
			{Kind: "input", Source: "local", Target: "receiver", Operation: "input", Obligation: "public_ob"},
			{Kind: "input", Source: "local", Target: "receiver", Operation: "input", Obligation: "audit_ob"},
		}
		s, path := messageStore(t, p)

		deniedReq := Input{TransportID: "t-admin", LogicalID: "l-admin", Target: "receiver", Obligation: "admin_ob", Payload: map[string]Value{"value": Text("forged-admin")}}
		denied, err := s.AdmitInput(t.Context(), deniedReq)
		if err != nil {
			t.Fatal(err)
		}
		if denied.Accepted || denied.Reason != "route_denied" || denied.ObservationID != "" || count(t, s, "observations") != 0 || count(t, s, "admissions") != 1 {
			t.Fatalf("cross-obligation input bypass: got %+v, observations=%d", denied, count(t, s, "observations"))
		}
		deniedAgain, err := s.AdmitInput(t.Context(), deniedReq)
		if err != nil || !reflect.DeepEqual(denied, deniedAgain) {
			t.Fatalf("denied input decision not retained: %+v %v", deniedAgain, err)
		}

		pubReq := Input{TransportID: "t-public", LogicalID: "l-public", Target: "receiver", Obligation: "public_ob", Payload: map[string]Value{"value": Text("allowed-public")}}
		pubRes, err := s.AdmitInput(t.Context(), pubReq)
		if err != nil || !pubRes.Accepted || pubRes.ObservationID == "" || count(t, s, "observations") != 1 {
			t.Fatalf("granted public_ob input rejected: %+v %v", pubRes, err)
		}

		auditReq := Input{TransportID: "t-audit", LogicalID: "l-audit", Target: "receiver", Obligation: "audit_ob", Payload: map[string]Value{"value": Text("allowed-audit")}}
		auditRes, err := s.AdmitInput(t.Context(), auditReq)
		if err != nil || !auditRes.Accepted || auditRes.ObservationID == "" || count(t, s, "observations") != 2 {
			t.Fatalf("granted audit_ob input rejected: %+v %v", auditRes, err)
		}

		report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
		if err != nil {
			t.Fatal(err)
		}
		state := current(t, s, "receiver")
		if !report.Quiescent || state.Projection.Status != "succeeded" || state.Projection.Data["public_val"] != Text("allowed-public") || state.Projection.Data["audit_val"] != Text("allowed-audit") {
			t.Fatalf("unexpected receiver state: report=%+v projection=%+v", report, state.Projection)
		}

		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := Open(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		reopenedDenied, err := reopened.AdmitInput(t.Context(), deniedReq)
		if err != nil || !reflect.DeepEqual(denied, reopenedDenied) {
			t.Fatalf("reopened denied input changed: %+v %v", reopenedDenied, err)
		}
		reopenedPub, err := reopened.AdmitInput(t.Context(), pubReq)
		if err != nil || !reflect.DeepEqual(pubRes, reopenedPub) {
			t.Fatalf("reopened accepted input changed: %+v %v", reopenedPub, err)
		}
		if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
			t.Fatalf("replay failed: %v", err)
		}
	})

	t.Run("message obligation enforcement and multiple grants", func(t *testing.T) {
		p := Provision{}
		senderCode := []Instruction{
			{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("admin_ob")), Payload: map[string]Operand{"value": Lit(Text("forged-admin"))}}},
			{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("public_ob")), Payload: map[string]Operand{"value": Lit(Text("allowed-public"))}}},
			{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("audit_ob")), Payload: map[string]Operand{"value": Lit(Text("allowed-audit"))}}},
			{Op: "end", Outcome: "succeeded"},
		}
		receiverCode := []Instruction{
			{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("sender")), Generation: Lit(Int(1)), Obligation: Lit(Text("public_ob")), Operation: Lit(Text(""))}, Bind: map[string]string{"value": "public_val"}},
			{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("sender")), Generation: Lit(Int(1)), Obligation: Lit(Text("audit_ob")), Operation: Lit(Text(""))}, Bind: map[string]string{"value": "audit_val"}},
			{Op: "end", Outcome: "succeeded"},
		}
		addProgram(t, &p, "sender", senderCode, map[string]Value{})
		addProgram(t, &p, "receiver", receiverCode, map[string]Value{})
		p.Permissions = []Permission{
			{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "public_ob"},
			{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "audit_ob"},
		}
		s, path := messageStore(t, p)
		for range 3 {
			if _, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), ""); err != nil {
				t.Fatal(err)
			}
		}

		adminIntent := `["sender",1,1,0]`
		publicIntent := `["sender",1,2,0]`
		auditIntent := `["sender",1,3,0]`

		denied, err := s.DeliverMessage(t.Context(), adminIntent, "tx-admin")
		if err != nil {
			t.Fatal(err)
		}
		if denied.Accepted || denied.Reason != "route_denied" || denied.ObservationID != "" || count(t, s, "observations") != 0 {
			t.Fatalf("cross-obligation message bypass: got %+v, observations=%d", denied, count(t, s, "observations"))
		}

		pubRes, err := s.DeliverMessage(t.Context(), publicIntent, "tx-public")
		if err != nil || !pubRes.Accepted || pubRes.ObservationID == "" || count(t, s, "observations") != 1 {
			t.Fatalf("granted public_ob message rejected: %+v %v", pubRes, err)
		}
		if err = s.AcknowledgeMessage(t.Context(), publicIntent, "tx-public"); err != nil {
			t.Fatal(err)
		}

		auditRes, err := s.DeliverMessage(t.Context(), auditIntent, "tx-audit")
		if err != nil || !auditRes.Accepted || auditRes.ObservationID == "" || count(t, s, "observations") != 2 {
			t.Fatalf("granted audit_ob message rejected: %+v %v", auditRes, err)
		}
		if err = s.AcknowledgeMessage(t.Context(), auditIntent, "tx-audit"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), ""); err != nil {
			t.Fatal(err)
		}

		for range 3 {
			obs := ""
			switch current(t, s, "receiver").Projection.PC {
			case 0:
				obs = pubRes.ObservationID
			case 1:
				obs = auditRes.ObservationID
			}
			if _, err = s.Commit(t.Context(), current(t, s, "receiver").Authority(), obs); err != nil {
				t.Fatal(err)
			}
		}
		state := current(t, s, "receiver")
		if state.Projection.Status != "succeeded" || state.Projection.Data["public_val"] != Text("allowed-public") || state.Projection.Data["audit_val"] != Text("allowed-audit") {
			t.Fatalf("unexpected receiver projection: %+v", state.Projection)
		}

		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := Open(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.Close()
		reopenedDenied, err := reopened.DeliverMessage(t.Context(), adminIntent, "tx-admin")
		if err != nil || !reflect.DeepEqual(denied, reopenedDenied) {
			t.Fatalf("reopened denied message changed: %+v %v", reopenedDenied, err)
		}
		reopenedPub, err := reopened.DeliverMessage(t.Context(), publicIntent, "tx-public")
		if err != nil || !reflect.DeepEqual(pubRes, reopenedPub) {
			t.Fatalf("reopened accepted message changed: %+v %v", reopenedPub, err)
		}
		if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
			t.Fatalf("replay failed: %v", err)
		}
	})
}
