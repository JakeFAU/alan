package executor

import (
	"errors"
	"testing"
)

func TestInspectContradictoryHistoryOrder(t *testing.T) {
	t.Run("unscoped selection before emission", func(t *testing.T) {
		s, path := retainedMessageSelection(t, false)
		if _, err := s.conn.ExecContext(t.Context(), "UPDATE history SET rowid=0 WHERE subject='receiver'"); err != nil {
			t.Fatal(err)
		}
		if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
			t.Fatalf("accepted selection before the original source: %v", err)
		}
	})
	t.Run("scoped fact order opposes append order", func(t *testing.T) {
		p := Provision{}
		for _, name := range []string{"left", "right"} {
			addProgram(t, &p, name, []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
		}
		p.Composition = &compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"left", 1}, Members: []RunID{{"left", 1}, {"right", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
		s, path := messageStore(t, p)
		for _, name := range []string{"left", "right"} {
			if _, err := s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
			t.Fatal(err)
		}
		if _, err := s.conn.ExecContext(t.Context(), "UPDATE history SET rowid=0 WHERE subject='right'"); err != nil {
			t.Fatal(err)
		}
		if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
			t.Fatalf("accepted incompatible original append and scoped fact orders: %v", err)
		}
	})
}

func TestInspectRefusalRequiresItsTransportAndTarget(t *testing.T) {
	for _, tc := range []struct {
		name, target, obligation, reason string
	}{
		{"operator cannot have message source denial", "missing", "work", "source_denied:source_retired"},
		{"absent target has no closed generation", "missing", "work", "target_closed"},
		{"original static target cannot be missing", "receiver", "ungranted", "target_missing"},
		{"open target is not retired", "receiver", "ungranted", "target_retired"},
		{"wildcard cannot have wrong generation", "receiver", "ungranted", "target_generation_changed"},
		{"permitted schema reason requires a grant", "receiver", "ungranted", "schema_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path := messageStore(t, inputFixture(t))
			a, err := s.AdmitInput(t.Context(), Input{TransportID: "t", LogicalID: "id", Target: tc.target, Obligation: tc.obligation, Payload: map[string]Value{"number": Int(42)}})
			if err != nil || a.Accepted {
				t.Fatal(a, err)
			}
			if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal("genuine refusal", err)
			}
			a.Reason = tc.reason
			if tc.reason == "target_missing" {
				a.Bound = nil
			}
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE admissions SET result=CAST(? AS BLOB)", compositionJSON(a)); err != nil {
				t.Fatal(err)
			}
			if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
				t.Fatalf("accepted impossible transport refusal %s: %v", a.Reason, err)
			}
		})
	}
}
