package runtime

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestUninitializedStoreRefusesWithoutFaulting(t *testing.T) {
	var s Store
	ctx := context.Background()
	checks := []struct {
		name string
		call func() error
	}{
		{"Run", func() error { _, e := s.Run(ctx, RunOptions{MaxSteps: 1, MaxDispatch: 1}); return e }},
		{"RunProgress", func() error { _, e := s.RunProgress(ctx, RunOptions{MaxSteps: 1, MaxDispatch: 1}); return e }},
		{"AdmitInput", func() error { _, e := s.AdmitInput(ctx, Input{}); return e }},
		{"Snapshot", func() error { _, e := s.Snapshot(ctx, RunID{Subject: "a", Generation: 1}); return e }},
		{"Commit", func() error { _, e := s.Commit(ctx, Authority{}, ""); return e }},
		{"AuthorizeIntent", func() error { _, e := s.AuthorizeIntent(ctx, "i"); return e }},
		{"ContinuationBasis", func() error { _, e := s.ContinuationBasis(ctx, RunID{Subject: "a", Generation: 1}); return e }},
		{"ApproveContinuation", func() error { _, e := s.ApproveContinuation(ctx, ContinuationBasis{}, "r"); return e }},
		{"AwaitDispatchQuiescence", func() error { return s.AwaitDispatchQuiescence(ctx, DispatchAuthority{}) }},
		{"DeliverMessage", func() error { _, e := s.DeliverMessage(ctx, "i", "t"); return e }},
		{"AcknowledgeMessage", func() error { return s.AcknowledgeMessage(ctx, "i", "t") }},
		{"DeliverTerminal", func() error { _, e := s.DeliverTerminal(ctx, RunID{}, TerminalDestination{}, "t"); return e }},
		{"AcknowledgeTerminal", func() error { return s.AcknowledgeTerminal(ctx, RunID{}, TerminalDestination{}, "t") }},
		{"DispatchExternal", func() error { _, e := s.DispatchExternal(ctx, "i"); return e }},
		{"LoadLifecycleRequest", func() error { _, e := s.LoadLifecycleRequest(ctx, "i"); return e }},
		{"DecideLifecycle", func() error { _, e := s.DecideLifecycle(ctx, LifecycleRequest{}); return e }},
		{"SettleLifecycle", func() error { _, e := s.SettleLifecycle(ctx, "i", "t"); return e }},
		{"AdmitInvocation", func() error { _, e := s.AdmitInvocation(ctx, "o"); return e }},
		{"PrepareInvocation", func() error { _, e := s.PrepareInvocation(ctx, "o"); return e }},
		{"SettleInvocation", func() error { _, e := s.SettleInvocation(ctx, "o"); return e }},
		{"PrepareRoot", func() error { _, e := s.PrepareRoot(ctx, "r"); return e }},
		{"SettleRoot", func() error { _, e := s.SettleRoot(ctx, "r"); return e }},
	}
	for _, c := range checks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s PANICKED: %v", c.name, r)
				}
			}()
			err := c.call()
			if err == nil {
				t.Errorf("%s returned nil error on a zero Store", c.name)
				return
			}
			if !errors.Is(err, sql.ErrConnDone) {
				t.Errorf("%s error does not match sql.ErrConnDone: %v", c.name, err)
			}
		}()
	}
	if got := s.Namespace(); got != "" {
		t.Errorf("Namespace on zero Store = %q", got)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close on zero Store = %v", err)
	}
	// A nil receiver must also refuse rather than fault.
	var nilStore *Store
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("nil receiver Close PANICKED: %v", r)
			}
		}()
		if err := nilStore.Close(); err != nil {
			t.Errorf("nil Close = %v", err)
		}
	}()
}
