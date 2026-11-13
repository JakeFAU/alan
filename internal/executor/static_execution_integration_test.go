//go:build integration

package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	recordschema "github.com/JakeFAU/alan/schema"
	"github.com/mattn/go-sqlite3"
)

func staticAdapterStore(t *testing.T, n int, await bool) (*Store, string) {
	t.Helper()
	p := Provision{}
	c := CompositionProvision{Version: compositionProtocolVersion, RecordDispatchAdmissions: true}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("r%02d", i)
		addAdapterProgram(t, &p, name, await)
		id := RunID{name, 1}
		c.Roots = append(c.Roots, ResourceRoot{Run: id, Members: []RunID{id}, Capabilities: []string{"model"}, Limits: ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
	}
	p.Composition = &c
	s, path := messageStore(t, p)
	if s.execution == nil {
		t.Fatal("static adapter fixture not armed")
	}
	return s, path
}
func staticPrepare(t *testing.T, s *Store, id string) *adapterWork {
	t.Helper()
	w, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || w == nil {
		t.Fatalf("prepare %+v %v", w, err)
	}
	op, reason, err := s.registerDispatch(t.Context(), id)
	if err != nil || reason != "" || op == nil {
		t.Fatalf("dispatch %s %v", reason, err)
	}
	s.finishDispatch(op, nil)
	return w
}
func staticResult(t *testing.T, s *Store, w *adapterWork) AdapterResult {
	t.Helper()
	c := callbackAdapter{}.Execute(t.Context(), w.attempt.Prepared)
	r, err := s.recordAdapterResult(t.Context(), w.attempt, c)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func staticAssertActive(t *testing.T, s *Store) {
	t.Helper()
	if s.execution == nil || s.executionFallbacks != 0 {
		t.Fatalf("ordinary static path fell back: %d", s.executionFallbacks)
	}
}

func TestStaticExecutionHotWritesAndAffectedRoot(t *testing.T) {
	s, _ := staticAdapterStore(t, 16, true)
	works := make([]*adapterWork, 16)
	for i := range works {
		name := fmt.Sprintf("r%02d", i)
		run, err := s.Commit(t.Context(), current(t, s, name).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		works[i] = staticPrepare(t, s, run.Projection.Data["pending"].Text)
	}
	staticCompare(t, s, nil)
	if _, err := s.settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	r := staticResult(t, s, works[7])
	baselineRoots := staticRootLoads(t, s, func() {
		if _, err := s.staticBaselineSettleRoots(t.Context()); err != nil {
			t.Fatal(err)
		}
	})
	if baselineRoots != 16 {
		t.Fatalf("baseline root calibration=%d", baselineRoots)
	}
	if loads := staticRootLoads(t, s, func() {
		if _, err := s.settleRoots(t.Context()); err != nil {
			t.Fatal(err)
		}
	}); loads != 1 {
		t.Fatalf("single changed root visited %d roots", loads)
	}
	if err := s.settleAdapter(t.Context(), works[7].attempt, r); err != nil {
		t.Fatal(err)
	}
	want, baselineRuns, we := staticReadyRunLoads(t, s, func() (*readyRun, error) { return s.staticBaselineSelectReady(t.Context(), nil) })
	got, loads, ge := staticReadyRunLoads(t, s, func() (*readyRun, error) { return s.selectReady(t.Context(), nil) })
	if baselineRuns != 8 {
		t.Fatalf("baseline run calibration=%d", baselineRuns)
	}
	t.Logf("one changed root among16: baseline roots=%d candidate roots=1; baseline run loads=%d candidate run loads=%d", baselineRoots, baselineRuns, loads)
	if !reflect.DeepEqual(got, want) || fmt.Sprint(ge) != fmt.Sprint(we) || loads != 1 {
		t.Fatalf("affected ready %+v/%v versus %+v/%v loads=%d", got, ge, want, we, loads)
	}
	if got == nil || got.authority.Run.Subject != "r07" {
		t.Fatal("wrong affected run")
	}
	old := got.authority
	if _, err := s.Commit(t.Context(), old, got.observation); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), old, got.observation); !errors.Is(err, ErrStale) {
		t.Fatalf("stale authority: %v", err)
	}
	staticCompare(t, s, nil)
	ready, err := s.selectReady(t.Context(), nil)
	if err != nil || ready == nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), ready.authority, ready.observation); err != nil {
		t.Fatal(err)
	}
	if _, err = s.settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Duplicates preserve the winning result and do not create new candidates.
	c := callbackAdapter{}.Execute(t.Context(), works[7].attempt.Prepared)
	if _, err = s.recordAdapterResult(t.Context(), works[7].attempt, c); err != nil {
		t.Fatal(err)
	}
	if err = s.settleAdapter(t.Context(), works[7].attempt, r); err != nil {
		t.Fatal(err)
	}
	staticCompare(t, s, nil)
	pending, err := s.pendingDeliveries(t.Context())
	if err != nil || len(pending) != 15 {
		t.Fatalf("live frontier=%d %v", len(pending), err)
	}
	staticAssertActive(t, s)
}

func TestStaticExecutionFrozenDifferentialSchedule(t *testing.T) {
	initial, path := staticAdapterStore(t, 4, true)
	if err := initial.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stores []*Store
	var paths []string
	for range 2 {
		fork := t.TempDir() + "/fork.sqlite"
		if err = os.WriteFile(fork, raw, 0600); err != nil {
			t.Fatal(err)
		}
		s, e := Open(t.Context(), fork, DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { s.Close() })
		stores = append(stores, s)
		paths = append(paths, fork)
	}
	stores[0].execution = nil
	var last *RunID
	for turn := 0; turn < 12; turn++ {
		want, we := stores[0].staticBaselineSelectReady(t.Context(), last)
		got, ge := stores[1].selectReady(t.Context(), last)
		if !reflect.DeepEqual(want, got) || fmt.Sprint(we) != fmt.Sprint(ge) || got == nil {
			t.Fatalf("turn=%d selection %+v/%v %+v/%v", turn, want, we, got, ge)
		}
		for _, s := range stores {
			run, e := s.Commit(t.Context(), got.authority, got.observation)
			if e != nil {
				t.Fatal(e)
			}
			if run.Sequence == 1 {
				w := staticPrepare(t, s, run.Projection.Data["pending"].Text)
				r := staticResult(t, s, w)
				if e = s.settleAdapter(t.Context(), w.attempt, r); e != nil {
					t.Fatal(e)
				}
			}
		}
		a, ae := stores[0].staticBaselineSettleRoots(t.Context())
		b, be := stores[1].settleRoots(t.Context())
		if a != b || fmt.Sprint(ae) != fmt.Sprint(be) {
			t.Fatalf("roots %v/%v %v/%v", a, ae, b, be)
		}
		id := got.authority.Run
		last = &id
		var reports []RunReport
		for _, s := range stores {
			r, e := s.finishRun(t.Context(), RunReport{}, false, nil)
			if e != nil {
				t.Fatal(e)
			}
			reports = append(reports, r)
		}
		if !reflect.DeepEqual(reports[0], reports[1]) {
			t.Fatalf("turn %d reports differ", turn)
		}
	}
	staticAssertActive(t, stores[1])
	before, err := Inspect(t.Context(), paths[0], DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	after, err := Inspect(t.Context(), paths[1], DefaultLimits())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("inspection differs %v", err)
	}
	a, err := Replay(t.Context(), paths[0], DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Replay(t.Context(), paths[1], DefaultLimits())
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("replay differs %v", err)
	}
}

func TestStaticExecutionCommitBoundaries(t *testing.T) {
	for _, kind := range []string{"rollback", "cancel", "commit_failure", "post_commit"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := staticAdapterStore(t, 2, true)
			marker := errors.New("injected boundary")
			ctx := t.Context()
			if kind == "commit_failure" {
				if err := s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterCommitHook(func() int { return 1 }); return nil }); err != nil {
					t.Fatal(err)
				}
			} else {
				ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
					if kind == "post_commit" && e.Boundary == "execution.committed" {
						return marker
					}
					if kind != "post_commit" && e.Boundary == "execution.staged" {
						return marker
					}
					return nil
				})
				if kind == "cancel" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
			}
			_, err := s.Commit(ctx, current(t, s, "r00").Authority(), "")
			if err == nil {
				t.Fatal("injected failure accepted")
			}
			if kind == "commit_failure" {
				if e := s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterCommitHook(nil); return nil }); e != nil {
					t.Fatal(e)
				}
			}
			staticCompare(t, s, nil)
			state := current(t, s, "r00")
			if kind == "post_commit" {
				if state.Sequence != 1 || count(t, s, "intents") != 1 {
					t.Fatal("lost committed delta")
				}
				staticAssertActive(t, s)
			} else if state.Sequence != 0 || count(t, s, "intents") != 0 {
				t.Fatal("tentative commit escaped")
			}
		})
	}
}

func TestStaticExecutionLateKnownAndConflict(t *testing.T) {
	s, path := staticAdapterStore(t, 1, false)
	run, err := s.Commit(t.Context(), current(t, s, "r00").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	w := staticPrepare(t, s, run.Projection.Data["pending"].Text)
	if _, err = s.Commit(t.Context(), run.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if progress, e := s.settleRoots(t.Context()); e != nil || progress {
		t.Fatalf("unknown effect published root: %v %v", progress, e)
	}
	r := staticResult(t, s, w)
	if progress, e := s.settleRoots(t.Context()); e != nil || !progress {
		t.Fatalf("known before settlement not observed: %v %v", progress, e)
	}
	staticAssertActive(t, s)
	loser := callbackAdapter{}.Execute(t.Context(), w.attempt.Prepared)
	loser.Payload["answer"] = Text("other")
	if _, err = s.recordAdapterResult(t.Context(), w.attempt, loser); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict %v", err)
	}
	if s.execution != nil {
		t.Fatal("durable conflict did not disable reuse")
	}
	winner, err := loadAdapterResult(t.Context(), s.conn, w.attempt.Intent.ID)
	if err != nil || !reflect.DeepEqual(winner, r) {
		t.Fatal("conflict changed winner")
	}
	staticCompare(t, s, nil)
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if reopened.execution != nil {
		t.Fatal("conflicting historical shape armed")
	}
}

func TestStaticExecutionPostCommitBookkeepingFailure(t *testing.T) {
	s, _ := staticAdapterStore(t, 1, true)
	committed := false
	err := s.conn.Raw(func(d any) error {
		c := d.(*sqlite3.SQLiteConn)
		c.RegisterCommitHook(func() int { committed = true; return 0 })
		c.RegisterAuthorizer(func(op int, a, b, db string) int {
			if committed && op == sqlite3.SQLITE_FUNCTION && b == "total_changes" {
				return sqlite3.SQLITE_DENY
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Commit(t.Context(), current(t, s, "r00").Authority(), "")
	if e := s.conn.Raw(func(d any) error {
		c := d.(*sqlite3.SQLiteConn)
		c.RegisterCommitHook(nil)
		c.RegisterAuthorizer(nil)
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if err != nil || state.Sequence != 1 || count(t, s, "intents") != 1 {
		t.Fatalf("bookkeeping changed committed result: %+v %v", state, err)
	}
	if s.execution != nil {
		t.Fatal("failed bookkeeping retained reuse")
	}
	staticCompare(t, s, nil)
}
func TestStaticExecutionConsumedObservationCorruption(t *testing.T) {
	s, _ := staticAdapterStore(t, 2, true)
	run, err := s.Commit(t.Context(), current(t, s, "r00").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	w := staticPrepare(t, s, run.Projection.Data["pending"].Text)
	result := staticResult(t, s, w)
	if err = s.settleAdapter(t.Context(), w.attempt, result); err != nil {
		t.Fatal(err)
	}
	ready, err := s.selectReady(t.Context(), &RunID{"r01", 1})
	if err != nil || ready == nil || ready.observation == "" {
		t.Fatalf("ready %+v %v", ready, err)
	}
	if _, err = s.Commit(t.Context(), ready.authority, ready.observation); err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE observations SET descriptor=? WHERE id=?", []byte("{}"), ready.observation); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizeIntent(t.Context(), w.attempt.Intent.ID); err != nil {
		t.Fatal(err)
	}
	if s.execution != nil {
		t.Fatal("pre-BEGIN guard swallowed consumed corruption")
	}
	staticCompare(t, s, nil)
	if _, err = s.selectReady(t.Context(), nil); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("consumed row lost validation: %v", err)
	}
}
func TestStaticExecutionDeleteAndReplaceIntent(t *testing.T) {
	for _, kind := range []string{"delete", "replace"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := staticAdapterStore(t, 2, true)
			for _, name := range []string{"r00", "r01"} {
				if _, err := s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			staticCompare(t, s, nil)
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			query := "DELETE FROM intents"
			if kind == "replace" {
				query = "INSERT OR REPLACE INTO intents SELECT 'replacement',subject,generation,sequence,ordinal,source_revision,descriptor FROM intents WHERE subject='r00'"
			}
			if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			staticCompare(t, s, nil)
			if s.execution != nil {
				t.Fatal("destructive direct write retained old IDs")
			}
		})
	}
}
func TestStaticExecutionLargeLiveFrontier(t *testing.T) {
	s, _ := staticAdapterStore(t, 64, true)
	for i := 0; i < 64; i++ {
		if _, err := s.Commit(t.Context(), current(t, s, fmt.Sprintf("r%02d", i)).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		staticCompare(t, s, nil)
		pending, err := s.pendingDeliveries(t.Context())
		if err != nil || len(pending) != 64 {
			t.Fatalf("large frontier %d %v", len(pending), err)
		}
	}
	staticAssertActive(t, s)
}

func staticContinuationStore(t *testing.T, n int) (*Store, string, []ContinuationSlot) {
	t.Helper()
	p := Provision{}
	c := CompositionProvision{Version: compositionProtocolVersion, RecordDispatchAdmissions: true}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("r%02d", i)
		addAdapterProgram(t, &p, name, true)
		if i > 0 {
			var program Program
			if err := decode(p.Programs[i].Bytes, &program); err != nil {
				t.Fatal(err)
			}
			empty, _ := recordschema.Record{}.Retain()
			field := recordschema.Field{Kind: "string", MaxBytes: 4096}
			input, err := (recordschema.Record{"previous_operation_id": field, "purpose": field, "prepared_digest": field, "result_digest": field}).Retain()
			if err != nil {
				t.Fatal(err)
			}
			program.Author.Ports = map[string]PortDeclaration{"approval": {Local: true, Peer: "local", Obligation: "approve", Receive: input, Send: empty, ReceiveUsed: true}}
			program.Author.Receives = map[int]recordschema.Retained{0: input, 2: program.Author.Receives[1]}
			program.Author.Positions = append([]string{"approval"}, program.Author.Positions...)
			program.Code = append([]Instruction{{Op: "receive", Match: &Match{Kind: "input", Provenance: "local-input", Source: Lit(Text("local")), Generation: Lit(Int(1)), Obligation: Lit(Text("approve")), Operation: Lit(Text(""))}}}, program.Code...)
			raw, id, err := EncodeProgram(program)
			if err != nil {
				t.Fatal(err)
			}
			p.Programs[i] = Artifact{ID: id, Semantics: program.Semantics, Bytes: raw}
			p.Runs[i].ProgramID = id
			p.Permissions = append(p.Permissions, Permission{Kind: "input", Source: "local", Target: name, Operation: "input", Obligation: "approve"})
			c.Continuations = append(c.Continuations, ContinuationSlot{Predecessor: p.Runs[i-1].ID, Target: p.Runs[i].ID, Effect: "call", Obligation: "approve", ApprovalID: name + "-approval"})
		}
		id := p.Runs[i].ID
		c.Roots = append(c.Roots, ResourceRoot{Run: id, Members: []RunID{id}, Capabilities: []string{"model"}, Limits: ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
	}
	p.Composition = &c
	s, path := messageStore(t, p)
	if s.execution == nil {
		t.Fatal("continuation shape unarmed")
	}
	return s, path, c.Continuations
}
func TestStaticExecutionContinuationDifferentialRuns(t *testing.T) {
	initial, path, slots := staticContinuationStore(t, 8)
	if err := initial.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stores := make([]*Store, 2)
	paths := make([]string, 2)
	for i := range stores {
		paths[i] = t.TempDir() + "/chain.sqlite"
		if err = os.WriteFile(paths[i], raw, 0600); err != nil {
			t.Fatal(err)
		}
		stores[i], err = Open(t.Context(), paths[i], DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		s := stores[i]
		t.Cleanup(func() { s.Close() })
	}
	stores[0].execution = nil
	options := RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}}
	for turn := 0; turn < 8; turn++ {
		var reports []RunReport
		for i, s := range stores {
			if turn > 0 {
				cut, e := InspectExecution(t.Context(), paths[i], DefaultLimits())
				if e != nil {
					t.Fatal(e)
				}
				basis, e := cut.ContinuationBasis(slots[turn-1].Target)
				if e != nil {
					t.Fatal(e)
				}
				// Forged predecessor evidence must still fail the original transaction.
				wrong := basis
				wrong.PreparedDigest = identity("forged")
				if _, e = s.ApproveContinuation(t.Context(), wrong, "fresh"); !errors.Is(e, ErrConflict) {
					t.Fatalf("forged basis accepted: %v", e)
				}
				approval, e := s.ApproveContinuation(t.Context(), basis, "fresh")
				if e != nil || !approval.Accepted {
					t.Fatalf("approval %+v %v", approval, e)
				}
				duplicate, e := s.ApproveContinuation(t.Context(), basis, "fresh")
				if e != nil || !reflect.DeepEqual(approval, duplicate) {
					t.Fatalf("duplicate changed %+v %v", duplicate, e)
				}
			}
			r, e := s.Run(t.Context(), options)
			if e != nil {
				t.Fatal(e)
			}
			reports = append(reports, r)
		}
		if !reflect.DeepEqual(reports[0], reports[1]) {
			t.Fatalf("turn %d reports differ", turn)
		}
		staticAssertActive(t, stores[1])
	}
	a, err := Inspect(t.Context(), paths[0], DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Inspect(t.Context(), paths[1], DefaultLimits())
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("continuation inspection differs: %v", err)
	}
	ar, err := Replay(t.Context(), paths[0], DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	br, err := Replay(t.Context(), paths[1], DefaultLimits())
	if err != nil || !reflect.DeepEqual(ar, br) {
		t.Fatalf("continuation replay differs: %v", err)
	}
	// Whole required reports are still produced, but an unchanged completed owner
	// performs no readiness loads or root preparations before that final report.
	if _, err = stores[1].settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	if loads := staticRootLoads(t, stores[1], func() {
		if _, err = stores[1].settleRoots(t.Context()); err != nil {
			t.Fatal(err)
		}
	}); loads != 0 {
		t.Fatalf("completed roots=%d", loads)
	}
}

func TestStaticExecutionUnknownThenLaterRun(t *testing.T) {
	s, _ := staticAdapterStore(t, 1, true)
	calls := 0
	adapter := callbackAdapter{execute: func(_ context.Context, p effects.Prepared) effects.Candidate {
		calls++
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	}}
	options := RunOptions{MaxSteps: 20, MaxDispatch: 20, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"model": adapter}}
	first, err := s.Run(t.Context(), options)
	if err != nil || calls != 1 || len(first.UnknownAdapters) != 1 || len(first.PendingDeliveries) != 1 {
		t.Fatalf("unknown report %+v calls=%d %v", first, calls, err)
	}
	second, err := s.Run(t.Context(), options)
	if err != nil || calls != 1 || second.Dispatches != 0 || len(second.UnknownAdapters) != 1 {
		t.Fatalf("uncertainty retried %+v calls=%d %v", second, calls, err)
	}
	attempt, err := loadAdapterAttempt(t.Context(), s.conn, first.UnknownAdapters[0])
	if err != nil {
		t.Fatal(err)
	}
	result := callbackAdapter{}.Execute(t.Context(), attempt.Prepared)
	if _, err = s.recordAdapterResult(t.Context(), attempt, result); err != nil {
		t.Fatal(err)
	}
	final, err := s.Run(t.Context(), options)
	if err != nil || calls != 1 || final.Dispatches != 1 || len(final.UnknownAdapters) != 0 || len(final.PendingDeliveries) != 0 || len(final.OpenRuns) != 0 {
		t.Fatalf("later known result %+v calls=%d %v", final, calls, err)
	}
	staticAssertActive(t, s)
}
func TestStaticExecutionConcurrentDiscoveryAndResult(t *testing.T) {
	s, _ := staticAdapterStore(t, 8, true)
	var w *adapterWork
	for i := 0; i < 8; i++ {
		run, err := s.Commit(t.Context(), current(t, s, fmt.Sprintf("r%02d", i)).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		if i == 3 {
			w = staticPrepare(t, s, run.Projection.Data["pending"].Text)
		}
	}
	staticCompare(t, s, nil)
	done := make(chan error, 1)
	go func() {
		for range 50 {
			if _, err := s.selectReady(t.Context(), nil); err != nil {
				done <- err
				return
			}
			if _, err := s.settleRoots(t.Context()); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	result := staticResult(t, s, w)
	if err := s.settleAdapter(t.Context(), w.attempt, result); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	staticCompare(t, s, nil)
	ready, err := s.selectReady(t.Context(), nil)
	if err != nil || ready == nil || ready.authority.Run.Subject != "r03" {
		t.Fatalf("concurrent invalidation lost %+v %v", ready, err)
	}
	staticAssertActive(t, s)
}

// Fail SQLite statement preparation before the first publication DML. This
// leaves total_changes unchanged, so an anomaly guard cannot supply the retry.
func staticDenyRootPublication(t *testing.T, s *Store, denied func()) func() {
	t.Helper()
	s.mu.Lock()
	err := s.conn.Raw(func(d any) error {
		d.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, col, db string) int {
			if op == sqlite3.SQLITE_INSERT && table == "composition_root_publications" {
				if denied != nil {
					denied()
				}
				return sqlite3.SQLITE_DENY
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	})
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		s.mu.Lock()
		err := s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil); return nil })
		s.mu.Unlock()
		if err != nil {
			t.Error(err)
		}
	}
}

func TestStaticExecutionRootPublicationRetry(t *testing.T) {
	for _, via := range []string{"root_discovery", "ordinary_Run"} {
		t.Run(via, func(t *testing.T) {
			s, _ := staticAdapterStore(t, 1, false)
			run, err := s.Commit(t.Context(), current(t, s, "r00").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			w := staticPrepare(t, s, run.Projection.Data["pending"].Text)
			contacts := 0
			adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
				contacts++
				return callbackAdapter{}.Execute(ctx, p)
			}}
			result, err := s.recordAdapterResult(t.Context(), w.attempt, adapter.Execute(t.Context(), w.attempt.Prepared))
			if err != nil {
				t.Fatal(err)
			}
			if err = s.settleAdapter(t.Context(), w.attempt, result); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), run.Authority(), ""); err != nil {
				t.Fatal(err)
			}
			var priorOrder int64
			if err = s.conn.QueryRowContext(t.Context(), "SELECT max(decision_order) FROM composition_facts").Scan(&priorOrder); err != nil {
				t.Fatal(err)
			}
			before, err := readExecutionStamp(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			removeFault := staticDenyRootPublication(t, s, nil)
			var progress bool
			var firstErr error
			if via == "ordinary_Run" {
				var failed RunReport
				failed, firstErr = s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10, Adapters: map[string]effects.Adapter{"model": adapter}})
				if failed.Quiescent || failed.Steps != 0 || failed.Dispatches != 0 {
					removeFault()
					t.Fatalf("failed public Run changed progress: %+v", failed)
				}
			} else {
				progress, firstErr = s.settleRoots(t.Context())
			}
			removeFault()
			var sqliteErr sqlite3.Error
			if progress || !errors.As(firstErr, &sqliteErr) || sqliteErr.Code != sqlite3.ErrAuth {
				t.Fatalf("publication error changed: progress=%v error=%v", progress, firstErr)
			}
			after, err := readExecutionStamp(t.Context(), s.conn)
			if err != nil || after != before {
				t.Fatalf("pre-DML failure changed stamp: before=%+v after=%+v error=%v", before, after, err)
			}
			if count(t, s, "composition_root_publications") != 0 {
				t.Fatal("failed publication became durable")
			}
			staticAssertActive(t, s)
			t.Logf("first publication: error=%v; unchanged stamp=%+v; publications=0", firstErr, after)
			if via == "root_discovery" {
				progress, err = s.settleRoots(t.Context())
				if err != nil || !progress {
					t.Fatalf("ordinary root discovery lost retry: progress=%v error=%v", progress, err)
				}
			} else {
				report, e := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10, Adapters: map[string]effects.Adapter{"model": adapter}})
				if e != nil || !report.Quiescent || report.Steps != 0 || report.Dispatches != 0 {
					t.Fatalf("ordinary Run retry: report=%+v error=%v", report, e)
				}
			}
			if count(t, s, "composition_root_publications") != 1 {
				t.Fatal("ordinary retry did not publish the ready root")
			}
			if contacts != 1 || count(t, s, "composition_charges") != 1 || count(t, s, "adapter_dispatch_admissions") != 1 || count(t, s, "settlements") != 1 || current(t, s, "r00").Sequence != 2 {
				t.Fatal("retry changed contacts, charge, admission, settlement or local execution")
			}
			rootID := s.execution.rootOf[run.ID]
			published, e := s.PrepareRoot(t.Context(), rootID)
			if e != nil || !published.Ready || published.Outcome.ID == "" || published.Outcome.Status != "succeeded" || published.Outcome.DecisionOrder != priorOrder+1 {
				t.Fatalf("wrong publication/order: %+v %v", published, e)
			}
			if progress, err = s.staticBaselineSettleRoots(t.Context()); err != nil || progress {
				t.Fatalf("baseline found omitted publication: progress=%v error=%v", progress, err)
			}
			staticAssertActive(t, s)
		})
	}
}

func TestStaticExecutionRootPublicationFailurePreservesNewerInvalidations(t *testing.T) {
	s, _ := staticAdapterStore(t, 2, true)
	works := make([]*adapterWork, 2)
	for i := range works {
		run, err := s.Commit(t.Context(), current(t, s, fmt.Sprintf("r%02d", i)).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		works[i] = staticPrepare(t, s, run.Projection.Data["pending"].Text)
	}
	// Consume the initial dirty roots while both are blocked. Complete only r00.
	if _, err := s.settleRoots(t.Context()); err != nil {
		t.Fatal(err)
	}
	result := staticResult(t, s, works[0])
	if err := s.settleAdapter(t.Context(), works[0].attempt, result); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		ready, err := s.selectReady(t.Context(), nil)
		if err != nil || ready == nil || ready.authority.Run.Subject != "r00" {
			t.Fatalf("r00 ready %+v %v", ready, err)
		}
		if _, err = s.Commit(t.Context(), ready.authority, ready.observation); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	fault := make(chan struct{}, 1)
	removeFault := staticDenyRootPublication(t, s, func() {
		select {
		case fault <- struct{}{}:
		default:
		}
	})
	done := make(chan error, 1)
	go func() {
		select {
		case <-fault:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		// The denied INSERT occurs after the r00 dirty bit was consumed. These
		// supported operations therefore invalidate newer work, on both roots.
		if _, err := s.AuthorizeIntent(ctx, works[0].attempt.Intent.ID); err != nil {
			done <- err
			return
		}
		candidate := callbackAdapter{}.Execute(ctx, works[1].attempt.Prepared)
		_, err := s.recordAdapterResult(ctx, works[1].attempt, candidate)
		done <- err
	}()
	progress, err := s.settleRoots(ctx)
	removeFault()
	var sqliteErr sqlite3.Error
	if progress || !errors.As(err, &sqliteErr) || sqliteErr.Code != sqlite3.ErrAuth {
		t.Fatalf("first publication %v %v", progress, err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	staticAssertActive(t, s)
	for _, w := range works {
		root := s.execution.rootOf[w.attempt.Intent.Run]
		if !s.execution.dirtyRoots[root] {
			t.Fatalf("newer invalidation lost for %s", w.attempt.Intent.Run.Subject)
		}
	}
	if count(t, s, "composition_root_publications") != 0 {
		t.Fatal("failed publication retained")
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
	if err != nil || !report.Quiescent || count(t, s, "composition_root_publications") != 2 || count(t, s, "composition_charges") != 2 || count(t, s, "adapter_dispatch_admissions") != 2 {
		t.Fatalf("concurrent invalidations not completed: %+v %v", report, err)
	}
	staticAssertActive(t, s)
}

func TestStaticExecutionRootPublicationCommitFailureDoesNotReviveIndex(t *testing.T) {
	s, _ := staticAdapterStore(t, 1, false)
	run, err := s.Commit(t.Context(), current(t, s, "r00").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	w := staticPrepare(t, s, run.Projection.Data["pending"].Text)
	result := staticResult(t, s, w)
	if err = s.settleAdapter(t.Context(), w.attempt, result); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), run.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if err = s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterCommitHook(func() int { return 1 }); return nil }); err != nil {
		t.Fatal(err)
	}
	progress, commitErr := s.settleRoots(t.Context())
	if err = s.conn.Raw(func(d any) error { d.(*sqlite3.SQLiteConn).RegisterCommitHook(nil); return nil }); err != nil {
		t.Fatal(err)
	}
	var sqliteErr sqlite3.Error
	if progress || !errors.As(commitErr, &sqliteErr) || sqliteErr.ExtendedCode != sqlite3.ErrConstraintCommitHook {
		t.Fatalf("commit failure %v %v", progress, commitErr)
	}
	if s.execution != nil || s.executionFallbacks != 1 || count(t, s, "composition_root_publications") != 0 {
		t.Fatal("requeue revived an invalidated index or retained a failed publication")
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 10, MaxDispatch: 10})
	if err != nil || !report.Quiescent || report.Steps != 0 || report.Dispatches != 0 || count(t, s, "composition_root_publications") != 1 {
		t.Fatalf("fallback retry %+v %v", report, err)
	}
	if s.execution != nil {
		t.Fatal("fallback retry revived index")
	}
}
