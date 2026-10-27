package executor

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCompositionMessageAuthorizationPrefix(t *testing.T) {
	for _, scoped := range []bool{true, false} {
		name := "scoped"
		if !scoped {
			name = "unrepresented"
		}
		t.Run(name, func(t *testing.T) {
			p := Provision{}
			send := Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("receiver")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Payload: map[string]Operand{"number": Lit(Int(17))}}}
			receive := Instruction{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("sender")), Generation: Lit(Int(1)), Obligation: Lit(Text("work")), Operation: Lit(Text(""))}, Bind: map[string]string{"number": "number"}}
			addProgram(t, &p, "sender", []Instruction{send, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			addProgram(t, &p, "receiver", []Instruction{receive, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			p.Permissions = []Permission{{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Revision: 1}}
			members := []RunID{{"receiver", 1}}
			if scoped {
				members = append(members, RunID{"sender", 1})
			}
			p.Composition = &compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"receiver", 1}, Members: members, Limits: compositionLimits{MaxInFlight: 1}}}}
			path := t.TempDir() + "/message.sqlite"
			s, err := Create(t.Context(), path, p, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if _, err = s.Commit(t.Context(), current(t, s, "sender").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			admitted, err := s.DeliverMessage(t.Context(), identity("sender", int64(1), int64(1), 0), "original-transport")
			if err != nil || !admitted.Accepted {
				t.Fatal(admitted, err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "receiver").Authority(), admitted.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal("valid baseline", err)
			}
			facts, err := readCompositionFacts(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			if !scoped {
				if len(facts) != 1 || facts[0].Run.Subject != "receiver" {
					t.Fatal("unexpected unrepresented fixture", facts)
				}
				return
			}
			if len(facts) != 3 || facts[1].Kind != compositionAuthorizationFact || facts[2].Run.Subject != "receiver" {
				t.Fatal("unexpected fixture", facts)
			}
			reorderCompositionConsumptionFacts(t, s, []int{0, 2, 1})
			_, err = Replay(t.Context(), path, DefaultLimits())
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "observation consumed before source authorization") {
				t.Fatalf("expected exact authorization predecessor rejection: %v", err)
			}
			t.Logf("causal rejection: %v", err)
		})
	}
}

// A denied lifecycle settlement has an authorization but no target decision.
// Its closed source cannot consume it; the existing source-sequence guard must
// also reject moving the denied authorization before the closing execution.
func TestCompositionSourceDeniedLifecycleAuthorizationPrefix(t *testing.T) {
	p := lifecycleFixture(t, []string{"CancelRun"}, false)
	p.Composition = &compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"requester", 1}, Members: []RunID{{"requester", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
	path := t.TempDir() + "/denied.sqlite"
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	request := committedRequest(t, s, "requester")
	if _, err = s.Commit(t.Context(), current(t, s, "requester").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	if err != nil || auth.Allowed || auth.Reason != "source_closed" {
		t.Fatal(auth, err)
	}
	admitted, err := s.SettleLifecycle(t.Context(), request.Intent.ID, "denied-result")
	if err != nil || !admitted.Accepted {
		t.Fatal(admitted, err)
	}
	outcome, err := lifecycleOutcome(t.Context(), s.conn, request.Intent.ID)
	if err != nil || outcome.Disposition != "source_denied" || count(t, s, "lifecycle_results") != 0 {
		t.Fatal(outcome, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal("valid source-denied baseline", err)
	}
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	// Execution1, execution2, its adjacent freeze, denied authorization.
	if len(facts) != 4 || facts[2].Kind != compositionFreezeFact || facts[3].Kind != compositionAuthorizationFact {
		t.Fatal("unexpected fixture", facts)
	}
	reorderCompositionConsumptionFacts(t, s, []int{0, 3, 1, 2})
	_, err = Replay(t.Context(), path, DefaultLimits())
	if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "authorization before source prefix") {
		t.Fatalf("expected existing authorization sequence rejection: %v", err)
	}
	t.Logf("existing causal rejection: %v", err)
}

func reorderCompositionConsumptionFacts(t *testing.T, s *Store, order []int) {
	t.Helper()
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != len(order) {
		t.Fatal("unexpected fact count", facts)
	}
	before := map[string][]map[string]any{}
	for _, table := range inspectionTables {
		rows, err := inspectTable(t.Context(), s.conn, table)
		if err != nil {
			t.Fatal(err)
		}
		before[table] = rows
	}
	if _, err = s.conn.ExecContext(t.Context(), "DELETE FROM composition_facts"); err != nil {
		t.Fatal(err)
	}
	for n, index := range order {
		f := facts[index]
		f.Order = int64(n + 1)
		if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_facts VALUES(?,?,?,?)", f.Order, f.ID, f.Kind, compositionJSON(f)); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range inspectionTables {
		rows, err := inspectTable(t.Context(), s.conn, table)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rows, before[table]) {
			t.Fatalf("ordinary table changed: %s", table)
		}
	}
	after, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	for n, index := range order {
		want := facts[index]
		want.Order = int64(n + 1)
		if !reflect.DeepEqual(after[n], want) {
			t.Fatal("fact identity/source digest changed")
		}
	}
	fk, err := s.conn.QueryContext(t.Context(), "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	bad := fk.Next()
	err = fk.Err()
	fk.Close()
	if err != nil || bad {
		t.Fatal("FK violation", err)
	}
}
