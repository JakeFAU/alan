//go:build integration

package executor

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/effects"
)

func gateStore(t *testing.T, tracked bool) (*Store, string, string) {
	t.Helper()
	p := Provision{}
	addAdapterProgram(t, &p, "worker", true)
	id := RunID{"worker", 1}
	p.Composition = &compositionProvision{RecordDispatchAdmissions: tracked, Roots: []compositionRoot{{
		Run: id, Members: []RunID{id}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1},
	}}}
	path := filepath.Join(t.TempDir(), "gate.sqlite")
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	r, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	return s, path, r.Projection.Data["pending"].Text
}

func TestDispatchAdmissionValidatesOriginalGate(t *testing.T) {
	for _, tc := range []struct {
		name, change string
		want         error
	}{
		{"original", "", nil},
		{"missing", "DELETE FROM adapter_dispatch_admissions", ErrCorrupt},
		{"mismatched preparation", "UPDATE adapter_dispatch_admissions SET prepared_digest=printf('%064d',1)", ErrCorrupt},
		{"before reservation", "UPDATE adapter_dispatch_admissions SET after_fact=1", ErrCorrupt},
		{"after terminal", "UPDATE adapter_dispatch_admissions SET after_fact=(SELECT max(decision_order) FROM composition_facts)", ErrCorrupt},
		{"masked format", "UPDATE metadata SET schema_version=4; PRAGMA user_version=4", ErrVersion},
		{"masked ordinary format", "UPDATE metadata SET schema_version=3; PRAGMA user_version=3", ErrVersion},
		{"missing journal", "DROP TABLE adapter_dispatch_admissions", ErrVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path, id := gateStore(t, true)
			ctx, l := t.Context(), DefaultLimits()
			_, err := s.Run(ctx, RunOptions{MaxSteps: 30, MaxDispatch: 30, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}})
			if err != nil {
				t.Fatal(err)
			}
			if tc.change != "" {
				if _, err := s.conn.ExecContext(ctx, tc.change); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			before := inspectionFingerprint(t, path)
			cut, inspect := InspectExecution(ctx, path, l)
			_, raw := Inspect(ctx, path, l)
			_, replay := Replay(ctx, path, l)
			if before != inspectionFingerprint(t, path) {
				t.Fatal("reader changed durable bytes")
			}
			owner, open := Open(ctx, path, l)
			if owner != nil {
				owner.Close()
			}
			if !errors.Is(inspect, tc.want) || !errors.Is(raw, tc.want) || !errors.Is(replay, tc.want) || !errors.Is(open, tc.want) {
				t.Fatal(inspect, raw, replay, open)
			}
			if tc.want != nil {
				if cut != nil || owner != nil || inspectionFingerprint(t, path) != before {
					t.Fatal("rejected store changed")
				}
				return
			}
			e, ok := cut.Effect(id)
			if !ok || cut.StorageVersion != 9 || !cut.Features.DispatchAdmissions || e.DispatchAdmission == nil || e.DispatchAdmission.PreparedDigest != e.AdapterAttempt.Prepared.Digest || e.Dispatch.Minimum != 1 || *e.Dispatch.Maximum != 1 || cut.Roots[0].Charges.Effects != 1 {
				t.Fatal(e)
			}
		})
	}
}

func TestDispatchAdmissionFailureNeverExposesOrRefunds(t *testing.T) {
	s, path, id := gateStore(t, true)
	ctx, l := t.Context(), DefaultLimits()
	if _, err := s.conn.ExecContext(ctx, `CREATE TRIGGER veto BEFORE INSERT ON adapter_dispatch_admissions BEGIN SELECT RAISE(IGNORE); END`); err != nil {
		t.Fatal(err)
	}
	opts := RunOptions{MaxSteps: 30, MaxDispatch: 30, Adapters: map[string]effects.Adapter{"model": callbackAdapter{execute: func(context.Context, effects.Prepared) effects.Candidate { panic("failed gate released a transport") }}}}
	if _, err := s.Run(ctx, opts); err == nil {
		t.Fatal("gate failed to verify admission write")
	}
	if _, err := s.conn.ExecContext(ctx, "DROP TRIGGER veto"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		owner, err := Open(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		_, err = owner.Run(ctx, opts)
		owner.Close()
		if err != nil {
			t.Fatal(err)
		}
		cut, err := InspectExecution(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		e, ok := cut.Effect(id)
		if !ok || e.AdapterResult != nil || e.DispatchAdmission != nil || e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 0 || cut.Roots[0].Charges.Effects != 1 {
			t.Fatal(e)
		}
	}
}

func TestDispatchAdmissionConflictPromotionAndLegacy(t *testing.T) {
	for _, tracked := range []bool{true, false} {
		s, path, id := gateStore(t, tracked)
		ctx, l := t.Context(), DefaultLimits()
		if _, err := s.Run(ctx, RunOptions{MaxSteps: 30, MaxDispatch: 30, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}}); err != nil {
			t.Fatal(err)
		}
		a, err := loadAdapterAttempt(ctx, s.conn, id)
		if err != nil {
			t.Fatal(err)
		}
		c := effects.Candidate{ID: id, Key: a.Prepared.Request.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text("other")}}
		if _, err = s.recordAdapterResult(ctx, a, c); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		cut, err := InspectExecution(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		want := 7
		if tracked {
			want = 10
		}
		e, ok := cut.Effect(id)
		if !ok || cut.StorageVersion != want || !cut.Features.ConflictJournal || len(e.AdditionalReceipts) != 1 || (e.DispatchAdmission != nil) != tracked {
			t.Fatal(cut.StorageVersion, e)
		}
		if _, err := Replay(ctx, path, l); err != nil {
			t.Fatal(err)
		}
		owner, err := Open(ctx, path, l)
		if err != nil {
			t.Fatal(err)
		}
		owner.Close()
	}
}
