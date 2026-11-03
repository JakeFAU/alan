package executor

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/JakeFAU/alan/internal/storetest"
)

// Compare all durable bytes. SQLite's read transaction may create an empty WAL
// alongside a checkpointed database. The shared-memory lock index is transient.
func inspectionFingerprint(t testing.TB, path string) [2][32]byte {
	t.Helper()
	db, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wal, err := os.ReadFile(path + "-wal")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return [2][32]byte{sha256.Sum256(db), sha256.Sum256(wal)}
}

func TestExecutionInspectionHistoricalSnapshots(t *testing.T) {
	for _, name := range []string{"legacy-effects", "recorded-adapter", "recorded-idempotent", "adopted-handoff", "unrecorded-read"} {
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			path := storetest.Copy(t, name)
			limits := DefaultLimits()
			bytes := inspectionFingerprint(t, path)
			cut, err := InspectExecution(ctx, path, limits)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := Replay(ctx, path, limits)
			if err != nil {
				t.Fatal(err)
			}
			page, err := cut.History(0, 1)
			if err != nil || page.Total != replay.HistoryCount || len(cut.Operations) != replay.IntentCount || cut.Namespace != replay.Namespace || len(cut.Runs) != len(replay.Runs) {
				t.Fatal("mismatched snapshot totals")
			}
			for _, r := range replay.Runs {
				v, ok := cut.Run(r.ID)
				if !ok || !reflect.DeepEqual(r, v.RunState) {
					t.Fatal("mismatched original run", r.ID)
				}
			}
			if inspectionFingerprint(t, path) != bytes {
				t.Fatal("read-only inspection or replay changed durable bytes")
			}
			t.Logf("format %d: %d runs, %d steps, %d effects", cut.StorageVersion, len(cut.Runs), page.Total, len(cut.Effects))
		})
	}
}

func diagnosticStore(t testing.TB, path string, n int) *Store {
	t.Helper()
	program := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "add", Dst: "n", A: Ref("n"), B: Lit(Int(1))}, {Op: "jump", Target: 0}}}
	bytes, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	p := Provision{Programs: []Artifact{{ID: id, Bytes: bytes, Semantics: SemanticVersion}}, Subjects: []Subject{{ID: "counter", Owner: "owner", Generation: 1, Revision: 1}}, Runs: []InitialRun{{ID: RunID{"counter", 1}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{"n": Int(0)}}}}}
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Snapshot(t.Context(), p.Runs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for range n {
		state, err = s.Commit(t.Context(), state.Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// Keep the owner live and race committed writes with many complete read cuts.
// Every returned projection and every page must name the same durable prefix.
func TestExecutionInspectionDuringOwnerWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "counter.sqlite")
	s := diagnosticStore(t, path, 4)
	defer s.Close()
	id := RunID{"counter", 1}
	ctx := t.Context()
	r, err := s.Snapshot(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	begin, done := make(chan struct{}), make(chan error, 1)
	go func() {
		<-begin
		var e error
		for range 120 {
			r, e = s.Commit(ctx, r.Authority(), "")
			if e != nil {
				break
			}
		}
		done <- e
	}()
	close(begin)
	for range 25 {
		cut, e := InspectExecution(ctx, path, DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		state, ok := cut.Run(id)
		page, e := cut.History(0, 3)
		if !ok || e != nil || state.Sequence != int64(page.Total) || state.Projection.Data["n"].Number != (state.Sequence+1)/2 || state.Epoch != 1 || !page.More || page.Next != 3 {
			t.Fatal("mixed cuts", state, page, e)
		}
		if page, e = cut.History(page.Next, MaxHistoryPage); e != nil || page.More || page.Next != page.Total || page.Steps[len(page.Steps)-1].Sequence != state.Sequence {
			t.Fatal("mixed page", page, e)
		}
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	r, err = s.Snapshot(ctx, id)
	if err != nil || r.Epoch != 1 || r.Sequence != 124 {
		t.Fatal("owner changed", r, err)
	}
}

func BenchmarkExecutionInspection(b *testing.B) {
	for _, n := range []int{250, 1000, 2000} {
		path := filepath.Join(b.TempDir(), fmt.Sprintf("inspect-%d.sqlite", n))
		s := diagnosticStore(b, path, n)
		if err := s.Close(); err != nil {
			b.Fatal(err)
		}
		for _, mode := range []string{"raw", "typed"} {
			b.Run(strconv.Itoa(n)+"/"+mode, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if mode == "raw" {
						if _, e := Inspect(b.Context(), path, DefaultLimits()); e != nil {
							b.Fatal(e)
						}
					} else if _, e := InspectExecution(b.Context(), path, DefaultLimits()); e != nil {
						b.Fatal(e)
					}
				}
			})
		}
	}
}
