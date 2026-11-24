//go:build integration

package runtime_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

type malformedAdapter struct {
	testAdapter
	change func(*effects.Candidate)
}

func (a *malformedAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := a.testAdapter.Execute(ctx, p)
	a.change(&c)
	return c
}

func TestMalformedAdapterResponsePreservesHistory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*effects.Candidate)
	}{
		{"usage", func(c *effects.Candidate) { c.Usage = &effects.Usage{Details: json.RawMessage{}} }},
		{"negative usage", func(c *effects.Candidate) { c.Usage = &effects.Usage{TotalTokens: new(int64(-1))} }},
		{"invalid UTF-8", func(c *effects.Candidate) { c.ResponseID = string([]byte{0xff}) }},
		{"non-UTF-8 disposition", func(c *effects.Candidate) { c.Disposition = string([]byte{0xff}) }},
		{"negative HTTP status", func(c *effects.Candidate) { c.HTTPStatus = -1 }},
		{"low HTTP status", func(c *effects.Candidate) { c.HTTPStatus = 99 }},
		{"high HTTP status", func(c *effects.Candidate) { c.HTTPStatus = 600 }},
		{"bound", func(c *effects.Candidate) { c.Response = strings.Repeat("x", 4097) }},
		{"key", func(c *effects.Candidate) { c.Key += "wrong" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), "agent.sqlite")
			l := runtime.DefaultLimits()
			s, err := runtime.Create(ctx, path, adapterProvision(t), l)
			if err != nil {
				t.Fatal(err)
			}
			a := &malformedAdapter{change: tc.change}
			opts := runtime.RunOptions{Adapters: map[string]effects.Adapter{"model": a}, MaxSteps: 20, MaxDispatch: 20}
			_, runErr := s.Run(ctx, opts)
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if !errors.Is(runErr, runtime.ErrAdapterOutput) || errors.Is(runErr, runtime.ErrCorrupt) {
				t.Fatalf("invalid adapter output classified as %v", runErr)
			}
			replay, err := runtime.Replay(ctx, path, l)
			if err != nil || len(replay.Runs) != 1 || replay.Runs[0].Projection.Status != "open" {
				t.Fatalf("replay %+v: %v", replay.Runs, err)
			}
			s, err = runtime.Open(ctx, path, l)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			report, err := s.Run(ctx, opts)
			if err != nil || len(report.UnknownAdapters) != 1 || a.calls.Load() != 1 {
				t.Fatalf("reopen: %+v err=%v calls=%d", report, err, a.calls.Load())
			}
		})
	}
}

func TestReadBudgetPreservesAdapterHistory(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "worker.sqlite")
	limits := runtime.DefaultLimits()
	s, err := runtime.Create(ctx, path, adapterProvision(t), limits)
	if err != nil {
		t.Fatal(err)
	}
	opts := runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": &testAdapter{}}}
	if _, err = s.Run(ctx, opts); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	smaller := limits
	smaller.MaxObservationBytes = 4096*8 - 1 // One byte short of this adapter's preparation allowance.
	_, inspectErr := runtime.Inspect(ctx, path, smaller)
	_, cutErr := runtime.InspectExecution(ctx, path, smaller)
	_, replayErr := runtime.Replay(ctx, path, smaller)
	owner, openErr := runtime.Open(ctx, path, smaller)
	if owner != nil {
		owner.Close()
	}
	for name, err := range map[string]error{"Inspect": inspectErr, "InspectExecution": cutErr, "Replay": replayErr, "Open": openErr} {
		if !errors.Is(err, runtime.ErrLimit) || errors.Is(err, runtime.ErrCorrupt) {
			t.Errorf("%s classified a smaller read limit as %v", name, err)
		}
	}
	if _, err := runtime.Replay(ctx, path, limits); err != nil {
		t.Fatal(err)
	}
}
