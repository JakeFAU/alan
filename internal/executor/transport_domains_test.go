package executor

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

func transportMessageFixture(t *testing.T) Provision {
	t.Helper()
	p := Provision{}
	addProgram(t, &p, "sender", []Instruction{{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(42))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	receiver := inputProgram()
	receiver[0].Match.Kind = "message"
	receiver[0].Match.Provenance = "ordinary-message"
	receiver[0].Match.Source = Lit(Text("sender"))
	addProgram(t, &p, "receiver", receiver, map[string]Value{})
	p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1, Obligation: "work"}, {Kind: "input", Source: "local", Target: "receiver", Operation: "input"}}
	return p
}
func addTerminalParticipants(t *testing.T, p *Provision) {
	t.Helper()
	addProgram(t, p, "child", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	addProgram(t, p, "observer", []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text("child")), Generation: Lit(Int(1)), Obligation: Lit(Text("public")), Operation: Lit(Text(""))}, Bind: map[string]string{"outcome": "public-outcome"}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "child", Target: "observer", Operation: "terminal", Revision: 1, Obligation: "public"})
}

func TestAcceptedInputCannotShadowAutomaticMessageTransport(t *testing.T) {
	s, path := messageStore(t, transportMessageFixture(t))
	// Preserve the exact raw accepted-input ID from independent review R1.
	const raw = `["message-delivery","[\"sender\",1,1,0]"]`
	input := Input{TransportID: raw, LogicalID: "ordinary-input", Target: "receiver", Payload: map[string]Value{"number": Int(7)}}
	admitted, err := s.AdmitInput(t.Context(), input)
	if err != nil || !admitted.Accepted {
		t.Fatalf("valid input %+v %v", admitted, err)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil {
		t.Fatalf("accepted ordinary input shadowed automatic message: raw=%s report=%+v err=%v", raw, report, err)
	}
	if report.Steps != 4 || report.Dispatches != 1 || !report.Quiescent || current(t, s, "receiver").Projection.Data["number"] != Int(42) {
		t.Fatalf("intended message did not progress %+v", report)
	}
	var consumed sql.NullInt64
	if err = s.conn.QueryRowContext(t.Context(), "SELECT consumed_sequence FROM observations WHERE id=?", admitted.ObservationID).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed.Valid {
		t.Fatal("receiver consumed the colliding ordinary input")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	again, err := reopened.AdmitInput(t.Context(), input)
	if err != nil || !reflect.DeepEqual(admitted, again) {
		t.Fatalf("input disposition changed %+v %v", again, err)
	}
	delivered, err := reopened.DeliverMessage(t.Context(), `["sender",1,1,0]`, raw)
	if err != nil || !delivered.Accepted || delivered.TransportID != raw || delivered.ObservationID == admitted.ObservationID {
		t.Fatalf("typed message retransmission %+v %v", delivered, err)
	}
	if err = reopened.AcknowledgeMessage(t.Context(), `["sender",1,1,0]`, raw); err != nil {
		t.Fatal(err)
	}
	assertTransportDomains(t, path, raw, "input", "message")
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestRejectedInputCannotShadowAutomaticTerminalTransport(t *testing.T) {
	p := Provision{}
	addTerminalParticipants(t, &p)
	s, path := messageStore(t, p)
	// Preserve the exact raw rejected-input ID from independent review R1.
	const raw = `["terminal-delivery","child",1,1,"observer","public"]`
	input := Input{TransportID: raw, LogicalID: "rejected-input", Target: "missing", Payload: map[string]Value{}}
	rejected, err := s.AdmitInput(t.Context(), input)
	if err != nil || rejected.Accepted || rejected.Reason != "target_missing" {
		t.Fatalf("input rejection %+v %v", rejected, err)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil {
		t.Fatalf("rejected ordinary input shadowed terminal: raw=%s report=%+v err=%v", raw, report, err)
	}
	if report.Steps != 3 || report.Dispatches != 1 || !report.Quiescent || current(t, s, "observer").Projection.Data["public-outcome"] != Text("failed") {
		t.Fatalf("terminal did not progress %+v", report)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	again, err := reopened.AdmitInput(t.Context(), input)
	if err != nil || !reflect.DeepEqual(rejected, again) {
		t.Fatalf("rejection changed %+v %v", again, err)
	}
	destination := TerminalDestination{Target: "observer", Obligation: "public"}
	delivered, err := reopened.DeliverTerminal(t.Context(), RunID{"child", 1}, destination, raw)
	if err != nil || !delivered.Accepted || delivered.TransportID != raw {
		t.Fatalf("terminal retransmission %+v %v", delivered, err)
	}
	if err = reopened.AcknowledgeTerminal(t.Context(), RunID{"child", 1}, destination, raw); err != nil {
		t.Fatal(err)
	}
	if count(t, reopened, "observations") != 1 {
		t.Fatal("terminal duplicated")
	}
	assertTransportDomains(t, path, raw, "input", "terminal")
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestMessageTerminalAndInputShareRawTransportWithoutSharingIdentity(t *testing.T) {
	p := transportMessageFixture(t)
	addTerminalParticipants(t, &p)
	// A second terminal source makes changed same-domain descriptors observable.
	addProgram(t, &p, "other-child", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "other-child", Target: "observer", Operation: "terminal", Revision: 1, Obligation: "public"})
	// Emit a second distinct intent so the same raw message ID must still conflict
	// within the message domain even though input/terminal domains remain isolated.
	sender := Program{Semantics: SemanticVersion, Code: []Instruction{
		{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(42))}}},
		{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(43))}}},
		{Op: "end", Outcome: "succeeded"},
	}}
	encoded, id, err := EncodeProgram(sender)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: SemanticVersion, Bytes: encoded})
	p.Runs[0].ProgramID = id
	s, path := messageStore(t, p)
	const raw = "same caller-chosen transport"
	input := Input{TransportID: raw, LogicalID: "ordinary", Target: "missing", Payload: map[string]Value{}}
	inputResult, err := s.AdmitInput(t.Context(), input)
	if err != nil || inputResult.Accepted {
		t.Fatalf("input %+v %v", inputResult, err)
	}
	emitted, err := s.Commit(t.Context(), current(t, s, "sender").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	message, err := s.DeliverMessage(t.Context(), `["sender",1,1,0]`, raw)
	if err != nil || !message.Accepted {
		t.Fatalf("input shadowed chosen message ID %+v %v", message, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "child").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	destination := TerminalDestination{Target: "observer", Obligation: "public"}
	terminal, err := s.DeliverTerminal(t.Context(), RunID{"child", 1}, destination, raw)
	if err != nil || !terminal.Accepted {
		t.Fatalf("message shadowed chosen terminal ID %+v %v", terminal, err)
	}
	if err = s.AcknowledgeMessage(t.Context(), `["sender",1,1,0]`, raw); err != nil {
		t.Fatal(err)
	}
	if err = s.AcknowledgeTerminal(t.Context(), RunID{"child", 1}, destination, raw); err != nil {
		t.Fatal(err)
	}
	changed := input
	changed.Target = "receiver"
	if _, err = s.AdmitInput(t.Context(), changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("input conflict guard lost: %v", err)
	}
	if _, err = s.Commit(t.Context(), emitted.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DeliverMessage(t.Context(), `["sender",1,2,0]`, raw); !errors.Is(err, ErrConflict) {
		t.Fatalf("message conflict guard lost: %v", err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "other-child").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DeliverTerminal(t.Context(), RunID{"other-child", 1}, destination, raw); !errors.Is(err, ErrConflict) {
		t.Fatalf("terminal conflict guard lost: %v", err)
	}
	assertTransportDomains(t, path, raw, "input", "message", "terminal")
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	again, err := reopened.DeliverMessage(t.Context(), `["sender",1,1,0]`, raw)
	if err != nil || !reflect.DeepEqual(message, again) {
		t.Fatalf("typed message binding lost %+v %v", again, err)
	}
	notice, err := reopened.DeliverTerminal(t.Context(), RunID{"child", 1}, destination, raw)
	if err != nil || !reflect.DeepEqual(terminal, notice) {
		t.Fatalf("typed terminal binding lost %+v %v", notice, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func assertTransportDomains(t *testing.T, path, raw string, want ...string) {
	t.Helper()
	snapshot, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	keys := map[string]string{}
	for _, row := range snapshot.Tables["admissions"] {
		if row["transport_id"] == raw {
			domain, ok := row["domain"].(string)
			if !ok {
				t.Fatal("missing trusted admission domain")
			}
			key, ok := row["id"].(string)
			if !ok || keys[key] != "" {
				t.Fatal("typed admission keys collided")
			}
			keys[key] = domain
			got[domain] = true
		}
	}
	if len(got) != len(want) {
		t.Fatalf("domains %+v want %v", got, want)
	}
	for _, domain := range want {
		if !got[domain] {
			t.Fatalf("domain %s missing", domain)
		}
	}
	for _, row := range snapshot.Tables["message_acknowledgments"] {
		key, ok := row["admission_id"].(string)
		if !ok || keys[key] != "message" {
			t.Fatal("message acknowledgment does not reference typed admission")
		}
	}
}

func TestReplayRejectsChangedTransportDomainAndAcknowledgmentBinding(t *testing.T) {
	for _, query := range []string{
		`UPDATE admissions SET domain='terminal' WHERE domain='input'`,
		`UPDATE admissions SET transport_id='changed' WHERE domain='input'`,
		`UPDATE admissions SET id='untyped-raw-id' WHERE domain='input'`,
		`UPDATE message_acknowledgments SET admission_id=(SELECT id FROM admissions WHERE domain='input')`,
	} {
		t.Run(query, func(t *testing.T) {
			s, path := messageStore(t, transportMessageFixture(t))
			input := Input{TransportID: `["message-delivery","[\"sender\",1,1,0]"]`, LogicalID: "ordinary-input", Target: "receiver", Payload: map[string]Value{"number": Int(7)}}
			if _, err := s.AdmitInput(t.Context(), input); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("transport domain/key/ACK corruption accepted")
			}
		})
	}
}
