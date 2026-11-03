//go:build integration

package executor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

// In format 9 the accepted local input and the adapter result each commit
// durably in different journals. Opposite, genuinely executed transaction
// orders can have the same contents in every journal. This regression explains
// why a thin helper cannot reconstruct the approval boundary for older stores.
func TestContinuationLegacyLocalInputDoesNotOrderOriginalResult(t *testing.T) {
	ctx := t.Context()
	p := inputFixture(t)
	addAdapterProgram(t, &p, "worker", true)
	p.Composition = &compositionProvision{RecordDispatchAdmissions: true}
	for _, id := range []RunID{{"worker", 1}, {"receiver", 1}} {
		p.Composition.Roots = append(p.Composition.Roots, compositionRoot{Run: id, Members: []RunID{id},
			Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1}})
	}
	original := filepath.Join(t.TempDir(), "provisioned.sqlite")
	l := DefaultLimits()
	initial, err := Create(ctx, original, p, l)
	if err != nil {
		t.Fatal(err)
	}
	if err = initial.Close(); err != nil {
		t.Fatal(err)
	}
	// Close checkpointed the pristine provision. Each fork has the same
	// namespace, owner count, program identities, and resource limits.
	bytes, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	var history Inspection
	for _, first := range []bool{true, false} {
		path := filepath.Join(t.TempDir(), "fork.sqlite")
		if err = os.WriteFile(path, bytes, 0600); err != nil {
			t.Fatal(err)
		}
		s, err := Open(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		r, err := s.Commit(ctx, current(t, s, "worker").Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		op := r.Projection.Data["pending"].Text
		w, err := s.prepareAdapterWork(ctx, op, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		gate, reason, err := s.registerDispatch(ctx, op)
		if err != nil || gate == nil {
			t.Fatalf("gate: %s %v", reason, err)
		}
		candidate := callbackAdapter{}.Execute(ctx, w.attempt.Prepared)
		save := func() {
			if _, err := s.recordAdapterResult(ctx, w.attempt, candidate); err != nil {
				t.Fatal(err)
			}
		}
		if !first {
			save()
		}
		cut, err := InspectExecution(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		e, ok := cut.Effect(op)
		if !ok || (e.AdapterResult == nil) != first {
			t.Fatal("incorrect decision boundary")
		}
		d, err := s.AdmitInput(ctx, Input{TransportID: "operator-2", LogicalID: "operator-2", Target: "receiver", Generation: 1, Obligation: "work",
			Payload: map[string]Value{"number": Int(1), "predecessor": Text(op), "basis": Text("no retained response")}})
		if err != nil || !d.Accepted {
			t.Fatalf("input: %v %v", d, err)
		}
		if first {
			save()
		}
		s.finishDispatch(gate, nil)
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		stored, err := Inspect(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		if first {
			history = stored
		} else if !reflect.DeepEqual(history, stored) {
			t.Fatal("legacy ledger distinguishes ordinary input from unrelated result ordering")
		}
	}
}
