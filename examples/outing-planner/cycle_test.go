package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

func TestPlannerReplacementStepExhaustion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limited.db")
	bounds := standardBounds
	bounds.MaxSteps = 1
	r, err := runReplacementScenario(t.Context(), path, fixtureConfiguration(), confirmFirst, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if r.Outcome != boundExhausted || r.Cycle != nil || r.Report.Progress.Quiescent {
		t.Fatalf("incorrect classification at the step bound: %s, %+v", r.Outcome, r.Report.Progress)
	}
}

func TestPlannerReplacementPendingDeliveryControl(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "bad.db")
	c := fixtureConfiguration()
	if err := createPlanner(ctx, path, c, confirmFirst); err != nil {
		t.Fatal(err)
	}
	initial, _, _, err := inspectPlanner(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("stop after proposal commitment")
	observed := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary != "execution.committed" || e.Subject != outingSubject {
			return nil
		}
		cut, _, _, err := inspectPlanner(ctx, path)
		if err != nil {
			return err
		}
		for _, op := range cut.Operations {
			if op.Obligation == proposalPort {
				return stop
			}
		}
		return nil
	})
	if _, _, err := operatePlanner(observed, path, initial.Namespace, c, confirmFirst); !errors.Is(err, stop) {
		t.Fatalf("missing committed proposal cut: %v", err)
	}
	cut, _, _, err := inspectPlanner(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var delivery bool
	for _, p := range cut.Pending {
		if p.Kind == "message_awaiting_delivery" {
			delivery = true
			break
		}
	}
	if !delivery {
		t.Fatal("control has no eligible pending message")
	}
	// Quiescence alone cannot make a cut with pending delivery a cycle.
	if cycleAt(cut, runtime.RunProgress{Quiescent: true}) != nil {
		t.Fatal("pending delivery was misdiagnosed as a cycle")
	}
	bad, progress, err := operatePlanner(ctx, path, initial.Namespace, c, confirmFirst)
	if err != nil {
		t.Fatal(err)
	}
	if cycleAt(bad, progress) == nil {
		t.Fatal("actual reciprocal wait missing after proposal delivery")
	}
}
