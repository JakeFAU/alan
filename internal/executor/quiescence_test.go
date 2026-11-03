package executor

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

type dispatchPause struct{ held, release chan struct{} }

func newDispatchPause() dispatchPause { return dispatchPause{make(chan struct{}), make(chan struct{})} }
func (p dispatchPause) observe(ctx context.Context, event boundary.Event) error {
	close(p.held)
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func awaitSignal(t *testing.T, ctx context.Context, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestPhase2BMixedDuplicatesAndQuiescence(t *testing.T) {
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	var contacts atomic.Int64
	s, _, source := dispatchEndFixture(t, dispatchReceiptServer(t, &contacts).URL)
	target := DispatchAuthority{Namespace: s.Namespace(), Run: source.ID}
	if err := s.AwaitDispatchQuiescence(ctx, target); !errors.Is(err, ErrDispatchAuthorityOpen) {
		t.Fatal(err)
	}
	a, b, record := newDispatchPause(), newDispatchPause(), newDispatchPause()
	ended := make(chan struct{})
	op := source.Projection.Data["op"].Text
	direct := make(chan error, 1)
	go func() {
		c := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			if e.Boundary == "dispatch.admitted" {
				return a.observe(ctx, e)
			}
			return nil
		})
		_, err := s.DispatchExternal(c, op)
		direct <- err
	}()
	awaitSignal(t, ctx, a.held)
	run := make(chan error, 1)
	go func() {
		c := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			switch e.Boundary {
			case "execution.committed":
				if e.Phase == "terminal" {
					close(ended)
				}
			case "dispatch.admitted":
				return b.observe(ctx, e)
			case "external.result-recorded":
				return record.observe(ctx, e)
			}
			return nil
		})
		_, err := s.Run(c, RunOptions{MaxSteps: 8, MaxDispatch: 8})
		run <- err
	}()
	awaitSignal(t, ctx, b.held)
	awaitSignal(t, ctx, ended)
	waitThenCancel := func() {
		t.Helper()
		c, cancel := context.WithCancel(ctx)
		defer cancel()
		waiting := false
		c = boundary.WithObserver(c, func(ctx context.Context, e boundary.Event) error {
			if e.Boundary == "dispatch.waiting" {
				waiting = true
				cancel()
			}
			return nil
		})
		if err := s.AwaitDispatchQuiescence(c, target); !errors.Is(err, context.Canceled) || !waiting {
			t.Fatalf("expected pending Q, waiting=%t err=%v", waiting, err)
		}
	}
	waitThenCancel()
	close(b.release)
	awaitSignal(t, ctx, record.held)
	waitThenCancel()
	close(record.release)
	if err := <-run; err != nil {
		t.Fatal(err)
	}
	if count(t, s, "settlements") != 1 {
		t.Fatal("automatic opportunity did not settle")
	}
	waitThenCancel() // One operation, already settled. Its other exposure can still call.
	close(a.release)
	if err := <-direct; err != nil {
		t.Fatal(err)
	}
	if err := s.AwaitDispatchQuiescence(ctx, target); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != 2 || count(t, s, "external_attempts") != 1 || count(t, s, "settlements") != 1 {
		t.Fatal("duplicate physical lifetimes or retained identity lost", contacts.Load())
	}
	t.Log("both A < End; Q pending before L, after durable recording, and after first H. Two POSTs, one reservation, one settlement, final Q")
}

func TestPhase2BStickyAccountingAndRecoveredSettlement(t *testing.T) {
	var contacts atomic.Int64
	s, path, source := dispatchEndFixture(t, dispatchReceiptServer(t, &contacts).URL)
	target := DispatchAuthority{Namespace: s.Namespace(), Run: source.ID}
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TEMP TRIGGER reject_phase2b BEFORE INSERT ON settlements BEGIN SELECT RAISE(ABORT,'phase2b settlement failure'); END`); err != nil {
		t.Fatal(err)
	}
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.admitted" {
			_, err := s.Commit(t.Context(), source.Authority(), "")
			return err
		}
		return nil
	})
	_, err := s.DispatchExternal(ctx, source.Projection.Data["op"].Text)
	if err == nil || !strings.Contains(err.Error(), "phase2b settlement failure") {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "DROP TRIGGER reject_phase2b"); err != nil {
		t.Fatal(err)
	}
	if err = s.AwaitDispatchQuiescence(t.Context(), target); err == nil || !strings.Contains(err.Error(), "phase2b settlement failure") {
		t.Fatal("lost handling error", err)
	}
	if count(t, s, "external_receipts") != 1 || count(t, s, "settlements") != 0 {
		t.Fatal("failed H misrepresented")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err = reopened.AwaitDispatchQuiescence(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	if count(t, reopened, "settlements") != 1 || current(t, reopened, "source").Sequence != source.Sequence+1 || contacts.Load() != 1 {
		t.Fatal("Q recovery failed or advanced control/world")
	}
	t.Log("actual SQLite settlement rejection reached caller and Q; fresh owner Q alone settled retained receipt on original generation")
}

func TestPhase2BQTargetValidation(t *testing.T) {
	s, _, state := dispatchEndFixture(t, "http://127.0.0.1:1")
	target := DispatchAuthority{Namespace: s.Namespace(), Run: state.ID}
	for _, tc := range []struct {
		target DispatchAuthority
		want   error
	}{
		{DispatchAuthority{}, ErrStale}, {DispatchAuthority{Namespace: "foreign", Run: state.ID}, ErrStale},
		{DispatchAuthority{Namespace: s.Namespace(), Run: RunID{"source", 2}}, sql.ErrNoRows},
		{DispatchAuthority{Namespace: s.Namespace(), Run: state.ID, OwningScope: true}, ErrStale}, {target, ErrDispatchAuthorityOpen},
	} {
		if err := s.AwaitDispatchQuiescence(t.Context(), tc.target); !errors.Is(err, tc.want) {
			t.Fatalf("%+v got %v want %v", tc.target, err, tc.want)
		}
	}
	if _, err := s.Commit(t.Context(), state.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.AwaitDispatchQuiescence(ctx, target); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.AwaitDispatchQuiescence(t.Context(), target); err != nil {
		t.Fatal(err)
	}
}

func TestPhase2BClosePinsDirectAndReopenKeepsAuthority(t *testing.T) {
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	s, path, source := dispatchEndFixture(t, server.URL)
	target := DispatchAuthority{Namespace: s.Namespace(), Run: source.ID}
	pause := newDispatchPause()
	oldCtx := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.admitted" {
			return pause.observe(ctx, e)
		}
		return nil
	})
	id := source.Projection.Data["op"].Text
	direct := make(chan error, 1)
	go func() { _, err := s.DispatchExternal(oldCtx, id); direct <- err }()
	awaitSignal(t, ctx, pause.held)
	s.ownerMu.Lock()
	closing := s.ownerClosing
	s.ownerMu.Unlock()
	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()
	awaitSignal(t, ctx, closing)
	if other, err := Open(ctx, path, DefaultLimits()); !errors.Is(err, ErrLocked) {
		if other != nil {
			other.Close()
		}
		t.Fatalf("new owner admitted while old direct call still holds before L: %v", err)
	}
	if _, err := s.DispatchExternal(ctx, id); !errors.Is(err, sql.ErrConnDone) {
		t.Fatal("closing owner accepted new work", err)
	}
	if err := s.AwaitDispatchQuiescence(ctx, target); !errors.Is(err, sql.ErrConnDone) {
		t.Fatal("Close substituted for revocation", err)
	}
	if current(t, s, "source").Epoch != source.Epoch {
		t.Fatal("refused Open changed epoch")
	}
	close(pause.release)
	if err := <-direct; err != nil {
		t.Fatal(err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err = reopened.AwaitDispatchQuiescence(ctx, target); !errors.Is(err, ErrDispatchAuthorityOpen) {
		t.Fatal("owner shutdown revoked logical authority", err)
	}
	if settled, err := reopened.DispatchExternal(ctx, id); err != nil || settled {
		t.Fatal(settled, err)
	}
	if contacts.Load() != 2 || current(t, reopened, "source").Epoch != source.Epoch+1 {
		t.Fatal("expected distinct owner admissions", contacts.Load())
	}
	t.Log("direct held after A, Close began, Open refused with unchanged epoch; released one old call, fresh Open advanced epoch and admitted same eligible operation")
}
