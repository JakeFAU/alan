package executor

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

// Each case retains a real profile through Create, rather than constructing a
// private profile literal whose fields could disagree with its retained bytes.
func invocationModeStore(t *testing.T, mode string) (*Store, string, *compositionProvision) {
	t.Helper()
	p, c := compositionFixture(t)
	path := filepath.Join(t.TempDir(), "mode.sqlite")
	var s *Store
	var err error
	switch mode {
	case "ordinary":
		s, err = Create(t.Context(), path, p, DefaultLimits())
	case "foundation":
		s, err = createCompositionStore(t.Context(), path, p, c, DefaultLimits())
	case "invocation":
		c.Version = compositionProtocolVersion
		s, err = createInvocationStore(t.Context(), path, p, c, DefaultLimits())
	case "reconciliation":
		c.Version = compositionProtocolVersion
		p.Composition = &c
		p.Reconciliation = &ReconciliationProfile{}
		s, err = Create(t.Context(), path, p, DefaultLimits())
	case "dispatch":
		c.Version = compositionProtocolVersion
		c.RecordDispatchAdmissions = true
		p.Composition = &c
		s, err = Create(t.Context(), path, p, DefaultLimits())
	default:
		t.Fatalf("unknown test mode %q", mode)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, path, &c
}

func TestInvocationModeRetainedProfiles(t *testing.T) {
	for _, mode := range []string{"ordinary", "foundation", "invocation", "reconciliation", "dispatch"} {
		t.Run(mode, func(t *testing.T) {
			s, path, caller := invocationModeStore(t, mode)
			want := mode != "ordinary" && mode != "foundation"
			if got := s.invocationEnabled(); got != want {
				t.Fatalf("mode=%t want %t", got, want)
			}
			var original string
			var retained compositionProvision
			if s.composition != nil {
				original = s.composition.raw
				var err error
				retained, err = s.composition.descriptor()
				if err != nil {
					t.Fatal(err)
				}
				caller.Version = "caller-mutated-version"
				caller.Roots[0].Members[0] = RunID{"caller-mutated", 99}
				caller.Roots[0].Limits.MaxEffects = 99999
				decoded, err := s.composition.descriptor()
				if err != nil {
					t.Fatal(err)
				}
				decoded.Version = "decoded-mutated-version"
				decoded.Roots[0].Members[0] = RunID{"decoded-mutated", 99}
				decoded.Roots[0].Limits.MaxEffects = 99999
				fresh, err := s.composition.descriptor()
				if err != nil {
					t.Fatal(err)
				}
				if s.composition.raw != original || !reflect.DeepEqual(fresh, retained) || s.invocationEnabled() != want {
					t.Fatal("caller or decoded aliases changed retained profile/mode")
				}
			}
			oldProfile := s.composition
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			var reopened *Store
			var err error
			if mode == "foundation" {
				reopened, err = openCompositionStore(t.Context(), path, DefaultLimits())
			} else {
				reopened, err = Open(t.Context(), path, DefaultLimits())
			}
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if reopened.invocationEnabled() != want {
				t.Fatal("fresh owner changed mode")
			}
			if mode == "ordinary" {
				if reopened.composition != nil {
					t.Fatal("ordinary owner acquired composition profile")
				}
			} else if reopened.composition == oldProfile || reopened.composition.raw != original {
				t.Fatal("fresh owner did not reconstruct an independent retained profile")
			}
		})
	}
}

func TestInvocationModeIndependentStores(t *testing.T) {
	ordinary, _, _ := invocationModeStore(t, "ordinary")
	foundation, _, _ := invocationModeStore(t, "foundation")
	invocation, _, _ := invocationModeStore(t, "invocation")
	// The fixture uses the same subject/generation names in these independent
	// stores, so a name-based or process-global projection would be insufficient.
	if ordinary.invocationEnabled() || foundation.invocationEnabled() || !invocation.invocationEnabled() {
		t.Fatal("independent stores shared protocol mode")
	}
	if foundation.composition == invocation.composition {
		t.Fatal("independent stores shared a retained profile")
	}
}

func TestInvocationModeQueryAllocations(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	enabled := false
	allocations := testing.AllocsPerRun(100, func() { enabled = s.invocationEnabled() })
	if !enabled {
		t.Fatal("retained invocation protocol disabled")
	}
	if allocations != 0 {
		t.Fatalf("retained mode query allocated %.0f objects per call; want zero", allocations)
	}
}

func invocationModeEpochs(t *testing.T, q queryer) map[RunID]int64 {
	t.Helper()
	rows, err := q.QueryContext(t.Context(), "SELECT subject,generation,epoch FROM runs ORDER BY subject,generation")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	epochs := map[RunID]int64{}
	for rows.Next() {
		var id RunID
		var epoch int64
		if err = rows.Scan(&id.Subject, &id.Generation, &epoch); err != nil {
			t.Fatal(err)
		}
		epochs[id] = epoch
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	return epochs
}

func TestInvocationModeCorruptionRejectsBeforeRenewal(t *testing.T) {
	for _, mutation := range []string{"truncated bytes", "column version", "descriptor version", "coherent foundation downgrade", "schema marker"} {
		t.Run(mutation, func(t *testing.T) {
			s, path, _ := invocationModeStore(t, "invocation")
			before := invocationModeEpochs(t, s.conn)
			c, err := s.composition.descriptor()
			if err != nil {
				t.Fatal(err)
			}
			var query string
			var args []any
			switch mutation {
			case "truncated bytes":
				query = "UPDATE composition_profile SET descriptor=?"
				args = []any{[]byte("{")}
			case "column version":
				query = "UPDATE composition_profile SET version=?"
				args = []any{"future"}
			case "descriptor version":
				c.Version = "future"
				raw, e := json.Marshal(c)
				if e != nil {
					t.Fatal(e)
				}
				query = "UPDATE composition_profile SET descriptor=?"
				args = []any{raw}
			case "coherent foundation downgrade":
				c.Version = ""
				raw, e := json.Marshal(c)
				if e != nil {
					t.Fatal(e)
				}
				query = "UPDATE composition_profile SET version=?,descriptor=?"
				args = []any{compositionProfileVersion, raw}
			case "schema marker":
				query = "UPDATE metadata SET schema_version=3; PRAGMA user_version=3"
			}
			// Deliberate corruption injection, never a source of runtime authority.
			if _, err = s.conn.ExecContext(t.Context(), query, args...); err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if owner, e := Open(t.Context(), path, DefaultLimits()); e == nil {
				owner.Close()
				t.Fatal("corrupt profile activated")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("corrupt profile replayed")
			}
			if _, err = Inspect(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("raw inspection accepted corrupt profile")
			}
			if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("typed inspection accepted corrupt profile")
			}
			reader, err := connect(t.Context(), path, true)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			if after := invocationModeEpochs(t, reader.conn); !reflect.DeepEqual(before, after) {
				t.Fatalf("failed activation renewed epochs: before=%v after=%v", before, after)
			}
		})
	}
}
