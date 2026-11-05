package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

// The authorizer observes preparation of the actual program-row SELECT on the
// Store's pinned connection. database/sql does not cache these QueryRow calls.
// The callback always permits the query; no production hook or driver is added.
func p3CountLoads(t *testing.T, s *Store, selectFn func() (*readyRun, error)) (*readyRun, error, int) {
	t.Helper()
	loads := 0
	err := s.conn.Raw(func(driver any) error {
		driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, column, db string) int {
			if op == sqlite3.SQLITE_READ && table == "programs" && column == "bytes" {
				loads++
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.conn.Raw(func(driver any) error { driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil }); err != nil {
			t.Error(err)
		}
	}()
	r, err := selectFn()
	return r, err, loads
}

// All artifacts are genuinely encoded and retained by Create. Different suffix
// instructions create distinct IDs; each run starts at a waiting receive.
func p3WaitingProvision(t *testing.T, runs, programs int) Provision {
	t.Helper()
	p := Provision{}
	for i := 0; i < runs; i++ {
		code := inputProgram()
		code = append(code[:1], Instruction{Op: "set", Dst: "tag", A: Lit(Int(int64(i % programs)))}, code[1])
		name := fmt.Sprintf("r%02d", i)
		addProgram(t, &p, name, code, map[string]Value{"input": Int(int64(i))})
		p.Permissions = append(p.Permissions, Permission{Kind: "input", Source: "local", Target: name, Operation: "input", Obligation: "work"})
	}
	return p
}

func p3Compare(t *testing.T, ctx context.Context, s *Store, last *RunID) (*readyRun, error) {
	t.Helper()
	want, wantErr := s.p3BaselineSelectReady(ctx, last)
	got, gotErr := s.selectReady(ctx, last)
	if !reflect.DeepEqual(got, want) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
		t.Fatalf("readiness differs: got=%+v error=%v; baseline=%+v error=%v", got, gotErr, want, wantErr)
	}
	for _, target := range []error{context.Canceled, context.DeadlineExceeded, sql.ErrConnDone, sql.ErrNoRows, ErrCorrupt, ErrVersion, ErrLimit, ErrIneligible} {
		if errors.Is(gotErr, target) != errors.Is(wantErr, target) {
			t.Fatalf("error identity differs for %v", target)
		}
	}
	return got, gotErr
}

func TestP3ReadinessLoadReuse(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		runs, programs, want int
	}{
		{"shared_four", 32, 4, 4}, {"distinct", 32, 32, 32}, {"no_visit", 0, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := messageStore(t, p3WaitingProvision(t, tc.runs, tc.programs))
			baseline, err, baselineLoads := p3CountLoads(t, s, func() (*readyRun, error) { return s.p3BaselineSelectReady(t.Context(), nil) })
			if err != nil || baseline != nil || baselineLoads != tc.runs {
				t.Fatalf("baseline calibration: ready=%+v error=%v loads=%d want=%d", baseline, err, baselineLoads, tc.runs)
			}
			got, err, loads := p3CountLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
			if err != nil || got != nil {
				t.Fatalf("quiescence changed: %+v %v", got, err)
			}
			t.Logf("eligible=%d distinct=%d baseline_loads=%d candidate_loads=%d", tc.runs, tc.programs, baselineLoads, loads)
			if loads != tc.want {
				t.Fatalf("redundant successful readiness loads: got %d want %d", loads, tc.want)
			}
		})
	}
}

func p3Admit(t *testing.T, s *Store, target, key string, n int64) string {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), Input{TransportID: key, LogicalID: key, Target: target, Obligation: "work", Payload: map[string]Value{"number": Int(n)}})
	if err != nil || !a.Accepted {
		t.Fatalf("input %+v: %v", a, err)
	}
	observations, err := readObservations(t.Context(), s.conn, s.limits)
	if err != nil {
		t.Fatal(err)
	}
	for id, o := range observations {
		if o.Run.Subject == target && o.Payload["number"] == Int(n) {
			return id
		}
	}
	t.Fatal("accepted observation missing")
	return ""
}

func TestP3ReadinessOrderingAndLaterCalls(t *testing.T) {
	s, _ := messageStore(t, p3WaitingProvision(t, 3, 1))
	if r, err := p3Compare(t, t.Context(), s, nil); r != nil || err != nil {
		t.Fatalf("waiting: %+v %v", r, err)
	}
	observations := []string{p3Admit(t, s, "r01", "one", 11), p3Admit(t, s, "r01", "two", 12)}
	sort.Strings(observations)
	later := p3Admit(t, s, "r02", "three", 13)
	r, err := p3Compare(t, t.Context(), s, nil)
	if err != nil || r == nil || r.authority != current(t, s, "r01").Authority() || r.observation != observations[0] {
		t.Fatalf("waiting-before-ready/sorted observations: %+v %v", r, err)
	}
	last := RunID{"r01", 1}
	r, err = p3Compare(t, t.Context(), s, &last)
	if err != nil || r == nil || r.authority != current(t, s, "r02").Authority() || r.observation != later {
		t.Fatalf("rotation: %+v %v", r, err)
	}
	if _, err := s.Commit(t.Context(), r.authority, r.observation); err != nil {
		t.Fatal(err)
	}
	r, err = p3Compare(t, t.Context(), s, &last)
	if err != nil || r == nil || r.authority.Sequence != 1 || r.observation != "" {
		t.Fatalf("later projection/sequence: %+v %v", r, err)
	}
}

func TestP3ReadinessEarlyReturnAndDisposableCorruption(t *testing.T) {
	for _, kind := range []string{"missing", "bytes", "semantics", "limit"} {
		t.Run(kind, func(t *testing.T) {
			p := Provision{}
			addProgram(t, &p, "a", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			addProgram(t, &p, "z", inputProgram(), map[string]Value{})
			s, _ := messageStore(t, p)
			// Deliberately impossible retained history in a disposable store, before
			// readiness starts. This is not a supported concurrent mutation model.
			var err error
			switch kind {
			case "missing":
				_, err = s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF")
				if err == nil {
					_, err = s.conn.ExecContext(t.Context(), "DELETE FROM programs WHERE id=?", p.Runs[1].ProgramID)
				}
			case "bytes":
				_, err = s.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=? WHERE id=?", []byte("{}"), p.Runs[1].ProgramID)
			case "semantics":
				_, err = s.conn.ExecContext(t.Context(), "UPDATE programs SET semantics='future' WHERE id=?", p.Runs[1].ProgramID)
			case "limit":
				s.limits.MaxProgramBytes = len(p.Programs[0].Bytes)
			}
			if err != nil {
				t.Fatal(err)
			}
			r, err := p3Compare(t, t.Context(), s, nil)
			if err != nil || r == nil || r.authority.Run.Subject != "a" {
				t.Fatalf("unvisited corruption surfaced: %+v %v", r, err)
			}
			last := RunID{"a", 1}
			r, err = p3Compare(t, t.Context(), s, &last)
			want := map[string]error{"missing": sql.ErrNoRows, "bytes": ErrCorrupt, "semantics": ErrVersion, "limit": ErrLimit}[kind]
			if r != nil || !errors.Is(err, want) {
				t.Fatalf("first-use %s: %+v %v", kind, r, err)
			}
			// A previous successful call must not retain a later failed validation.
			if kind == "semantics" {
				if _, err = s.conn.ExecContext(t.Context(), "UPDATE programs SET semantics=? WHERE id=?", SemanticVersion, p.Runs[1].ProgramID); err != nil {
					t.Fatal(err)
				}
				if _, err = p3Compare(t, t.Context(), s, &last); err != nil {
					t.Fatalf("failed-load state leaked: %v", err)
				}
			}
		})
	}
}

func TestP3ReadinessGenerationAndReopen(t *testing.T) {
	s, path := messageStore(t, lifecycleFixture(t, []string{"Retry"}, false))
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	result := authorizedDecision(t, s, committedRequest(t, s, "requester"))
	if !result.Accepted || result.Successor != 2 {
		t.Fatal(result)
	}
	last := RunID{"requester", 1}
	r, err := p3Compare(t, t.Context(), s, &last)
	if err != nil || r == nil || r.authority.Run != (RunID{"worker", 2}) {
		t.Fatalf("generation: %+v %v", r, err)
	}
	before := r.authority
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	r, err = p3Compare(t, t.Context(), reopened, &last)
	if err != nil || r == nil || r.authority.Epoch != before.Epoch+1 || r.authority.Run != before.Run || r.authority.Sequence != before.Sequence {
		t.Fatalf("reopen: %+v %v", r, err)
	}
	// Disposable resurrection of the old projection isolates the generation
	// predicate from the terminal-status predicate. No supported API does this.
	old := current(t, reopened, "worker")
	old.Projection.Status = "open"
	old.Projection.PC = 0
	raw, err := json.Marshal(old.Projection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.conn.ExecContext(t.Context(), "UPDATE runs SET projection=? WHERE subject=? AND generation=?", raw, "worker", 1); err != nil {
		t.Fatal(err)
	}
	r, err = p3Compare(t, t.Context(), reopened, &last)
	if err != nil || r == nil || r.authority.Run != (RunID{"worker", 2}) {
		t.Fatalf("stale open generation selected: %+v %v", r, err)
	}
}

func TestP3ReadinessIndependentStoresAndCancellation(t *testing.T) {
	p := p3WaitingProvision(t, 2, 1)
	a, _ := messageStore(t, p)
	b, _ := messageStore(t, p)
	p3Admit(t, a, "r01", "only-a", 1)
	if r, err := p3Compare(t, t.Context(), a, nil); err != nil || r == nil {
		t.Fatalf("first: %+v %v", r, err)
	}
	if r, err := p3Compare(t, t.Context(), b, nil); err != nil || r != nil {
		t.Fatalf("cross Store: %+v %v", r, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if r, err := p3Compare(t, ctx, a, nil); r != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %+v %v", r, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if r, err := p3Compare(t, t.Context(), a, nil); r != nil || !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("close: %+v %v", r, err)
	}
}

func TestP3ReadinessBindingCorruption(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	admission := invokeOnce(t, s)
	if !admission.Accepted {
		t.Fatal(admission)
	}
	for _, member := range admission.Members {
		last := RunID{member.Subject, member.Generation}
		if _, err := p3Compare(t, t.Context(), s, &last); err != nil {
			t.Fatal(err)
		}
	}
	// Disposable pre-call corruption: no real API permits malformed binding bytes.
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE composition_bindings SET descriptor=?", []byte(`{"Version":"invalid"}`)); err != nil {
		t.Fatal(err)
	}
	// Start immediately before the first member in sorted RunID order.
	ids, err := readRunIDs(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		if strings.HasPrefix(id.Subject, "root") {
			continue
		}
		var last *RunID
		if i > 0 {
			last = &ids[i-1]
		}
		if r, err := p3Compare(t, t.Context(), s, last); err == nil || r != nil {
			t.Fatalf("invalid per-run binding escaped: %+v %v", r, err)
		}
		return
	}
	t.Fatal("no admitted member")
}

func TestP3ReadinessRequestAuthorityContext(t *testing.T) {
	for _, version := range []int{13, 14} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			p := requestAuthorityFixture(t, nil)
			var s *Store
			expected := RunID{"B", 1}
			if version == 13 {
				s, _ = messageStore(t, p)
			} else {
				var attempt ExternalAttempt
				s, _, _, attempt = authorityReady(t, p, true)
				if !attempt.Allowed {
					t.Fatal("authority fixture refused exact grant")
				}
				op, reason, err := s.registerDispatch(t.Context(), attempt.Intent.ID)
				if err != nil || reason != "" {
					t.Fatalf("authority admission: %s %v", reason, err)
				}
				s.finishDispatch(op, nil)
				first := receiptFor(attempt.Request)
				if err := s.recordExternalReceipt(t.Context(), attempt, first); err != nil {
					t.Fatal(err)
				}
				losing := first
				losing.Total++
				if err := s.recordExternalReceipt(t.Context(), attempt, losing); !errors.Is(err, ErrConflict) {
					t.Fatalf("real conflict promotion: %v", err)
				}
				expected = RunID{"A", 1}
			}
			if n, err := storedSchemaNumber(t.Context(), s.conn); err != nil || n != version {
				t.Fatalf("format=%d: %v", n, err)
			}
			if r, err := p3Compare(t, t.Context(), s, nil); err != nil || r == nil || r.authority.Run != expected {
				t.Fatalf("authority first selection: %+v %v", r, err)
			}
			// The authority profile is tested separately from bounded continuation.
			// Disposable pre-call feature-context corruption must not be masked by a
			// successful earlier program validation.
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE metadata SET schema_version=4 WHERE singleton=1"); err != nil {
				t.Fatal(err)
			}
			if r, err := p3Compare(t, t.Context(), s, nil); err == nil || r != nil {
				t.Fatalf("feature context escaped: %+v %v", r, err)
			}

		})
	}
}

func TestP3ReadinessFreshValidationEveryCallAndStore(t *testing.T) {
	p := p3WaitingProvision(t, 2, 1)
	a, _ := messageStore(t, p)
	b, _ := messageStore(t, p)
	for _, s := range []*Store{a, b, a} {
		if r, err := p3Compare(t, t.Context(), s, nil); err != nil || r != nil {
			t.Fatalf("waiting: %+v %v", r, err)
		}
	}
	// Disposable changes after a successful visit: the next call must load
	// again, and the other Store must not inherit either context or cached data.
	a.limits.MaxProgramBytes = 1
	if _, err := p3Compare(t, t.Context(), a, nil); !errors.Is(err, ErrLimit) {
		t.Fatalf("changed limits: %v", err)
	}
	if _, err := p3Compare(t, t.Context(), b, nil); err != nil {
		t.Fatalf("Store context leaked: %v", err)
	}
	a.limits = DefaultLimits()
	if _, err := a.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=?", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if _, err := p3Compare(t, t.Context(), a, nil); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("later bytes: %v", err)
	}
	if _, err := p3Compare(t, t.Context(), b, nil); err != nil {
		t.Fatalf("corruption leaked to another Store: %v", err)
	}
}

func TestP3ReadinessClosedRunsNeverLoaded(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "closed", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, _ := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "closed").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	// Disposable artifact corruption is unreachable after the genuine completion.
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE programs SET bytes=?", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	r, err, loads := p3CountLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
	if err != nil || r != nil || loads != 0 {
		t.Fatalf("ineligible load: %+v %v loads=%d", r, err, loads)
	}
	if _, err = p3Compare(t, t.Context(), s, nil); err != nil {
		t.Fatal(err)
	}
}

func TestP3ReadinessDisposableRetiredOpenRun(t *testing.T) {
	s, _ := messageStore(t, p3WaitingProvision(t, 1, 1))
	// Disposable impossible history separates retired eligibility from status:
	// ordinary RetireSubject also cancels an open projection.
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE subjects SET retired=1"); err != nil {
		t.Fatal(err)
	}
	r, err, loads := p3CountLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
	if err != nil || r != nil || loads != 0 {
		t.Fatalf("retired-open visit: %+v %v loads=%d", r, err, loads)
	}
	if _, err = p3Compare(t, t.Context(), s, nil); err != nil {
		t.Fatal(err)
	}
}
