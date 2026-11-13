package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"reflect"
	"testing"
)

func staticWaitingStore(t *testing.T, n int) (*Store, string) {
	t.Helper()
	p := p3WaitingProvision(t, n, n)
	c := CompositionProvision{Version: compositionProtocolVersion}
	for _, run := range p.Runs {
		id := run.ID
		c.Roots = append(c.Roots, ResourceRoot{Run: id, Members: []RunID{id}, Limits: ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 2, MaxInFlight: 1}})
	}
	path := t.TempDir() + "/static.sqlite"
	s, err := createInvocationStore(t.Context(), path, p, c, DefaultLimits())
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

func staticCompare(t *testing.T, s *Store, last *RunID) {
	t.Helper()
	want, we := s.staticBaselineSelectReady(t.Context(), last)
	got, ge := s.selectReady(t.Context(), last)
	if !reflect.DeepEqual(want, got) || fmt.Sprint(we) != fmt.Sprint(ge) {
		t.Fatalf("ready got=%+v/%v want=%+v/%v", got, ge, want, we)
	}
	wantD, we := staticBaselinePendingDeliveries(t.Context(), s.conn)
	gotD, ge := s.pendingDeliveries(t.Context())
	if !reflect.DeepEqual(wantD, gotD) || fmt.Sprint(we) != fmt.Sprint(ge) {
		t.Fatalf("delivery got=%+v/%v want=%+v/%v", gotD, ge, wantD, we)
	}
}

func staticRootLoads(t *testing.T, s *Store, fn func()) int {
	t.Helper()
	count := 0
	err := s.conn.Raw(func(d any) error {
		d.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, col, db string) int {
			if op == sqlite3.SQLITE_READ && table == "composition_roots" && col == "descriptor" {
				count++
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil })
	fn()
	return count
}
func TestStaticExecutionUnchangedReadiness(t *testing.T) {
	s, _ := staticWaitingStore(t, 16)
	staticCompare(t, s, nil)
	got, loads, err := p3CountLoads(t, s, func() (*readyRun, error) { return s.selectReady(context.Background(), nil) })
	if err != nil || got != nil || loads != 0 {
		t.Fatalf("unchanged discovery ready=%+v err=%v program loads=%d want 0", got, err, loads)
	}
	ready, visits, err := staticReadyRunLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
	if err != nil || ready != nil || visits != 0 {
		t.Fatalf("unchanged run visits=%d ready=%+v error=%v", visits, ready, err)
	}
}
func TestStaticExecutionUnchangedRoots(t *testing.T) {
	s, _ := staticWaitingStore(t, 16)
	if progress, err := s.staticBaselineSettleRoots(t.Context()); err != nil || progress {
		t.Fatalf("waiting baseline roots: %v %v", progress, err)
	}
	if _, err := s.settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	loads := staticRootLoads(t, s, func() {
		if _, err := s.settleRoots(t.Context()); err != nil {
			t.Fatal(err)
		}
	})
	if loads != 0 {
		t.Fatalf("unrelated root preparations=%d want 0", loads)
	}
}

func TestStaticExecutionFallbackPremises(t *testing.T) {
	for _, kind := range []string{"limits", "namespace", "profile", "program", "replace", "delete", "schema", "temp_trigger", "foreign"} {
		t.Run(kind, func(t *testing.T) {
			s, path := staticWaitingStore(t, 3)
			staticCompare(t, s, nil)
			if s.execution == nil {
				t.Fatal("eligible fixture not armed")
			}
			var err error
			switch kind {
			case "limits":
				s.limits.MaxProgramBytes = 1
			case "namespace":
				s.namespace += "changed"
			case "profile":
				copy := *s.composition
				s.composition = &copy
			case "program":
				_, err = s.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=?", []byte("{}"))
			case "replace":
				_, err = s.conn.ExecContext(t.Context(), "INSERT OR REPLACE INTO subjects SELECT id,generation,owner,revision,retired,retry_cancelled FROM subjects")
			case "delete":
				_, err = s.conn.ExecContext(t.Context(), "DELETE FROM permissions")
			case "schema":
				_, err = s.conn.ExecContext(t.Context(), "CREATE TABLE disposable(id INTEGER)")
			case "temp_trigger":
				_, err = s.conn.ExecContext(t.Context(), "CREATE TEMP TRIGGER unexpected AFTER INSERT ON main.history BEGIN UPDATE subjects SET revision=revision+1; END")
			case "foreign":
				other, e := connect(t.Context(), path, false)
				if e != nil {
					t.Fatal(e)
				}
				_, err = other.conn.ExecContext(t.Context(), "UPDATE subjects SET revision=revision+1")
				if e = other.Close(); e != nil {
					t.Fatal(e)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			staticCompare(t, s, nil)
			if s.execution != nil || s.executionFallbacks != 1 {
				t.Fatalf("premise did not permanently disable: index=%v fallbacks=%d", s.execution != nil, s.executionFallbacks)
			}
		})
	}
}
func TestStaticExecutionUntrustedWriteBeforeTransaction(t *testing.T) {
	s, _ := staticWaitingStore(t, 3)
	staticCompare(t, s, nil)
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=? WHERE id=(SELECT program_id FROM runs WHERE subject=?)", []byte("{}"), "r02"); err != nil {
		t.Fatal(err)
	}
	// Even a later ordinary admission cannot move the anomaly watermark forward.
	p3Admit(t, s, "r00", "legitimate", 1)
	if s.execution != nil {
		t.Fatal("legitimate write swallowed earlier corruption")
	}
	staticCompare(t, s, nil)
}
func TestStaticExecutionUnsupportedAdmissionAndFreshOwner(t *testing.T) {
	s, path := staticWaitingStore(t, 3)
	staticCompare(t, s, nil)
	p3Admit(t, s, "r01", "ordinary-input", 1)
	if s.execution != nil {
		t.Fatal("unsupported mutation retained index")
	}
	staticCompare(t, s, nil)
	before := current(t, s, "r01").Authority()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.execution == nil {
		t.Fatal("fresh validated owner did not arm")
	}
	staticCompare(t, reopened, nil)
	got, err := reopened.selectReady(t.Context(), nil)
	if err != nil || got == nil || got.authority.Epoch != before.Epoch+1 {
		t.Fatalf("fresh authority %+v %v", got, err)
	}
}
func TestStaticExecutionIndependentStoreAndCancellation(t *testing.T) {
	a, _ := staticWaitingStore(t, 2)
	b, _ := staticWaitingStore(t, 2)
	staticCompare(t, a, nil)
	staticCompare(t, b, nil)
	a.limits.MaxProgramBytes = 1
	staticCompare(t, a, nil)
	if b.execution == nil {
		t.Fatal("cross-owner invalidation")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := b.selectReady(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	staticCompare(t, b, nil)
}
func TestStaticExecutionNoOpAndRollback(t *testing.T) {
	s, _ := staticWaitingStore(t, 2)
	original := s.execution
	s.mu.Lock()
	tx, err := s.beginOwnedTx(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = s.commitOwnedTx(t.Context(), tx)
	s.mu.Unlock()
	if err != nil || s.execution != original {
		t.Fatalf("no-op: %v", err)
	}
	s.mu.Lock()
	tx, err = s.beginOwnedTx(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(t.Context(), "UPDATE subjects SET revision=revision+1")
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	s.mu.Unlock()
	staticCompare(t, s, nil)
	if s.execution != nil {
		t.Fatal("rollback change counter wasn't conservatively invalidated")
	}
	if current(t, s, "r00").Revision != 1 {
		t.Fatal("rollback published tentative revision")
	}
}

func TestStaticExecutionCopiedIndexAndClose(t *testing.T) {
	a, _ := staticWaitingStore(t, 2)
	b, _ := staticWaitingStore(t, 2)
	original := a.execution
	b.execution = a.execution
	staticCompare(t, b, nil)
	if b.execution != nil || a.execution != original {
		t.Fatal("foreign disposable state accepted or source changed")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if a.execution != nil {
		t.Fatal("Close retained index")
	}
}

func TestStaticExecutionLargeScopeFallsBack(t *testing.T) {
	p := p3WaitingProvision(t, 32, 1)
	root := ResourceRoot{Run: p.Runs[0].ID, Limits: ResourceLimits{MaxEffects: 64, MaxInFlight: 1}}
	for _, run := range p.Runs {
		root.Members = append(root.Members, run.ID)
	}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/scope.sqlite", p, CompositionProvision{Version: compositionProtocolVersion, Roots: []ResourceRoot{root}}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.execution != nil {
		t.Fatal("large multi-member scope entered narrow shape")
	}
	staticCompare(t, s, nil)
}

func TestStaticExecutionUnchangedDiscoveryHistory(t *testing.T) {
	s, _ := staticWaitingStore(t, 32)
	if _, err := s.selectReady(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	reads := map[string]int{}
	if err := s.conn.Raw(func(d any) error {
		d.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, col, db string) int {
			if op == sqlite3.SQLITE_READ {
				reads[table]++
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	defer s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil })
	for range 3 {
		if _, err := s.selectReady(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
		if _, err := s.pendingDeliveries(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := s.settleRoots(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"runs", "observations", "intents", "settlements", "terminal_obligations", "composition_roots"} {
		if reads[table] != 0 {
			t.Fatalf("unchanged discovery still reads %s: %d", table, reads[table])
		}
	}
}
func TestStaticExecutionPreexistingTriggerDisablesOpen(t *testing.T) {
	s, path := staticWaitingStore(t, 2)
	if _, err := s.conn.ExecContext(t.Context(), "CREATE TRIGGER benign AFTER INSERT ON history BEGIN SELECT 1; END"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.execution != nil {
		t.Fatal("preexisting trigger allowed reuse")
	}
	staticCompare(t, reopened, nil)
}

func staticReadyRunLoads(t *testing.T, s *Store, fn func() (*readyRun, error)) (*readyRun, int, error) {
	t.Helper()
	loads := 0
	if err := s.conn.Raw(func(d any) error {
		d.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, col, db string) int {
			if op == sqlite3.SQLITE_READ && table == "runs" && col == "program_id" {
				loads++
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	defer s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil })
	ready, err := fn()
	return ready, loads, err
}
