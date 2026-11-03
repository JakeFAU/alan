package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

func TestPhase2BTwoDirectOpportunities(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 7*time.Second)
	defer cancel()
	var posts atomic.Int64
	s, path, source := dispatchEndFixture(t, dispatchReceiptServer(t, &posts).URL)
	target := DispatchAuthority{Namespace: s.Namespace(), Run: source.ID}
	id := source.Projection.Data["op"].Text
	hold := func() (dispatchPause, <-chan error) {
		pause := newDispatchPause()
		done := make(chan error, 1)
		work := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			if e.Boundary == "dispatch.admitted" {
				return pause.observe(ctx, e)
			}
			return nil
		})
		go func() { _, err := s.DispatchExternal(work, id); done <- err }()
		awaitSignal(t, ctx, pause.held)
		return pause, done
	}
	first, a := hold()
	second, b := hold()
	_, err := s.Commit(ctx, source.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	close(first.release)
	if err = <-a; err != nil {
		t.Fatal(err)
	}
	half, err := Inspect(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 1 || len(half.Tables["external_receipts"]) != 1 || len(half.Tables["settlements"]) != 1 {
		t.Fatal("first opportunity did not finish accounting")
	}
	wait, stop := context.WithCancel(ctx)
	pending := false
	wait = boundary.WithObserver(wait, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.waiting" {
			pending = true
			stop()
		}
		return ctx.Err()
	})
	err = s.AwaitDispatchQuiescence(wait, target)
	stop()
	if !pending || !errors.Is(err, context.Canceled) {
		t.Fatal(pending, err)
	}
	close(second.release)
	if err = <-b; err != nil {
		t.Fatal(err)
	}
	if err = s.AwaitDispatchQuiescence(ctx, target); err != nil {
		t.Fatal(err)
	}
	final, err := Inspect(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 || len(final.Tables["external_attempts"]) != 1 || len(final.Tables["settlements"]) != 1 {
		t.Fatal("wrong duplicate accounting")
	}
	raw, err := json.Marshal(map[string]any{"target": target, "operation": id, "first_H": half, "q_pending_with_one_direct_left": pending, "second_H": final, "total_posts": posts.Load()})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(raw))
}

func TestPhase2BCloseBeforeDirectAdmission(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 7*time.Second)
	defer cancel()
	var posts atomic.Int64
	s, path, source := dispatchEndFixture(t, dispatchReceiptServer(t, &posts).URL)
	op := source.Projection.Data["op"].Text
	pause := newDispatchPause()
	work := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.consider" {
			return pause.observe(ctx, e)
		}
		return nil
	})
	direct := make(chan error, 1)
	go func() { _, err := s.DispatchExternal(work, op); direct <- err }()
	awaitSignal(t, ctx, pause.held)
	s.ownerMu.Lock()
	closing := s.ownerClosing
	s.ownerMu.Unlock()
	closeDone := make(chan error, 1)
	go func() { closeDone <- s.Close() }()
	awaitSignal(t, ctx, closing)
	if other, err := Open(ctx, path, DefaultLimits()); !errors.Is(err, ErrLocked) {
		if other != nil {
			other.Close()
		}
		t.Fatal(err)
	}
	close(pause.release)
	if err := <-direct; !errors.Is(err, sql.ErrConnDone) {
		t.Fatal("pre-A direct passed closing owner", err)
	}
	if err := <-closeDone; err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 0 {
		t.Fatal("old owner called", posts.Load())
	}
	fresh, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if err = fresh.AwaitDispatchQuiescence(ctx, DispatchAuthority{Namespace: fresh.Namespace(), Run: source.ID}); !errors.Is(err, ErrDispatchAuthorityOpen) {
		t.Fatal(err)
	}
	ok, err := fresh.DispatchExternal(ctx, op)
	if !ok || err != nil || posts.Load() != 1 {
		t.Fatal(ok, err, posts.Load())
	}
	t.Log("direct held before A; owner closing and live flock refused Open; old dispatch stopped with no POST; fresh owner reopened original source and submitted the saved request")
}
