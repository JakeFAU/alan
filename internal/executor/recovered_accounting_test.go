//go:build integration

package executor

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
)

// The first owner obtains a real transport result and stops immediately after
// recording it. Each recovery service gets a fresh Store with no admission.
func recoveredFixture(t *testing.T, route string, calls *atomic.Int64) (string, DispatchAuthority, recordedDispatch) {
	t.Helper()
	stop := errors.New("test stopped after durable result")
	var s *Store
	var path, id, subject string
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == route+".result-recorded" {
			return stop
		}
		return nil
	})
	var err error
	if route == "external" {
		var state RunState
		s, path, state = dispatchEndFixture(t, dispatchReceiptServer(t, calls).URL)
		id = state.Projection.Data["op"].Text
		subject = state.ID.Subject
		// Direct dispatch does not automatically execute the source's End.
		_, err = s.DispatchExternal(ctx, id)
		if _, commitErr := s.Commit(t.Context(), state.Authority(), ""); commitErr != nil {
			t.Fatal(commitErr)
		}
	} else {
		provision := Provision{}
		subject = "agent"
		addAdapterProgram(t, &provision, subject, false)
		s, path = messageStore(t, provision)
		state, commitErr := s.Commit(t.Context(), current(t, s, subject).Authority(), "")
		if commitErr != nil {
			t.Fatal(commitErr)
		}
		id = state.Projection.Data["pending"].Text
		a := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
			calls.Add(1)
			return callbackAdapter{}.Execute(ctx, p)
		}}
		_, err = s.Run(ctx, RunOptions{Adapters: map[string]effects.Adapter{"model": a}, MaxSteps: 8, MaxDispatch: 8})
	}
	if !errors.Is(err, stop) || calls.Load() != 1 || count(t, s, "settlements") != 0 {
		t.Fatal("failed recording cut", err, calls.Load())
	}
	source := current(t, s, subject)
	if source.Projection.Status != "succeeded" {
		t.Fatal("expected original End", source)
	}
	target := DispatchAuthority{Namespace: s.Namespace(), Run: source.ID}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	return path, target, recordedDispatch{id: id, kind: route, source: source.ID}
}

func TestPhase2BRecoveredAccounting(t *testing.T) {
	for _, test := range []struct{ route, service string }{{"external", "Q"}, {"external", "Run"}, {"external", "direct"}, {"adapter", "Q"}, {"adapter", "Run"}} {
		t.Run(test.route+"-"+test.service, func(t *testing.T) {
			var calls atomic.Int64
			path, target, record := recoveredFixture(t, test.route, &calls)
			s, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			// A canceled waiter cannot poison this authentic result.
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if err = s.AwaitDispatchQuiescence(ctx, target); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if _, err = s.conn.ExecContext(t.Context(), `CREATE TEMP TRIGGER recovered BEFORE INSERT ON settlements BEGIN SELECT RAISE(ABORT,'recovered settlement abort'); END`); err != nil {
				t.Fatal(err)
			}
			switch test.service {
			case "Q":
				err = s.AwaitDispatchQuiescence(t.Context(), target)
			case "Run":
				_, err = s.Run(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8})
			case "direct":
				_, err = s.DispatchExternal(t.Context(), record.id)
			}
			if err == nil || !strings.Contains(err.Error(), "recovered settlement abort") {
				t.Fatal("service did not return SQLite rejection", err)
			}
			if _, err = s.conn.ExecContext(t.Context(), "DROP TRIGGER recovered"); err != nil {
				t.Fatal(err)
			}
			// Ordinary accounting can independently finish. Its success cannot
			// erase the previously reported original failure on this owner.
			if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 8, MaxDispatch: 8}); err != nil {
				t.Fatal(err)
			}
			if count(t, s, "settlements") != 1 {
				t.Fatal("expected repaired settlement")
			}
			if err = s.AwaitDispatchQuiescence(t.Context(), target); err == nil || !strings.Contains(err.Error(), "recovered settlement abort") {
				t.Fatal("same-owner Q silently forgot the rejection", err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			next, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer next.Close()
			if err = next.AwaitDispatchQuiescence(t.Context(), target); err != nil || calls.Load() != 1 || count(t, next, "settlements") != 1 {
				t.Fatal(err, calls.Load())
			}
			t.Log("original H cut, reopen, genuine recovery settlement error, repair, sticky same-owner Q error, third owner Q, single call")
		})
	}
}

func TestPhase2BRecoveredSourceMismatch(t *testing.T) {
	for _, route := range []string{"external", "adapter"} {
		t.Run(route, func(t *testing.T) {
			var calls atomic.Int64
			path, target, record := recoveredFixture(t, route, &calls)
			s, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			stale := record
			stale.source.Generation++
			if _, err = s.settleRecordedDispatch(t.Context(), stale); !errors.Is(err, ErrConflict) {
				t.Errorf("accepted mismatched recovered origin: %v", err)
			}
			if count(t, s, "settlements") != 0 {
				t.Fatal("stale binding settled another generation's result")
			}

			if err = s.AwaitDispatchQuiescence(t.Context(), target); err != nil || calls.Load() != 1 || count(t, s, "settlements") != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}
