package executor

import (
	"errors"
	"reflect"
	"testing"
)

func TestExecutionInspectionOwnsOneReadOnlyCut(t *testing.T) {
	s, path := newStore(t, false)
	initial := state(t, s)
	before, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	step, err := s.Commit(t.Context(), initial.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	after, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	old, ok := before.Run(initial.ID)
	if !ok || old.Sequence != 0 || old.Epoch != initial.Epoch || len(before.Runs) != 1 {
		t.Fatal("initial cut", old)
	}
	current, ok := after.Run(initial.ID)
	page, err := after.History(0, 1)
	if !ok || !reflect.DeepEqual(current.RunState, step) || err != nil || page.Total != 1 || page.More || len(page.Steps) != 1 || page.Steps[0].Run != initial.ID || len(page.Steps[0].OperationIDs) != 1 {
		t.Fatalf("second cut: %+v, %+v, %v", current, page, err)
	}
	if history, err := before.History(0, 1); err != nil || history.Total != 0 {
		t.Fatal("previous snapshot changed", history, err)
	}
	if _, err = after.History(0, 0); !errors.Is(err, ErrLimit) {
		t.Fatal("accepted unbounded diagnostics", err)
	}
	if got := state(t, s); !reflect.DeepEqual(got, step) {
		t.Fatal("inspection advanced owner", got)
	}
}

func TestExecutionInspectionRejectsForgedHistorySource(t *testing.T) {
	s, path := newStore(t, false)
	if _, err := s.Commit(t.Context(), state(t, s).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE history SET source_revision=source_revision+1"); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("forged revision accepted: %v %v", cut, err)
	}
}

func requireInspectionAcknowledgment(t *testing.T, path string, delivery AdmissionResult, kind string) {
	t.Helper()
	s, err := InspectExecution(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range s.Pending {
		if p.Kind != kind {
			continue
		}
		for _, id := range p.AdmissionIDs {
			if a, ok := s.Admission(id); ok && a.TransportID == delivery.TransportID && a.ObservationID == delivery.ObservationID {
				return
			}
		}
	}
	t.Fatalf("missing pending acknowledgment of %s: %+v", delivery.TransportID, s.Pending)
}
