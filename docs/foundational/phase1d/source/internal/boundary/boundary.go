// Package boundary reports real execution boundaries and optionally pauses on an
// inherited pipe. It never selects work, changes state, or dispatches operations.
package boundary

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type Event struct {
	Scenario, Token, Store, Boundary, Phase, Subject string
	PID                                              int
	Generation, Epoch, Sequence                      int64
	Intent, Attempt                                  string
}
type observerKey struct{}
type Observer func(context.Context, Event) error

func WithObserver(ctx context.Context, observer Observer) context.Context {
	return context.WithValue(ctx, observerKey{}, observer)
}
func Reached(ctx context.Context, event Event) error {
	if observe, ok := ctx.Value(observerKey{}).(Observer); ok {
		return observe(ctx, event)
	}
	return nil
}

// FromEnvironment uses inherited fd3 for acknowledgments and fd4 for release.
// Absence of ALAN_BARRIER leaves execution unchanged. A selected boundary pauses
// once; EOF is an error, never permission to proceed.
func FromEnvironment(ctx context.Context, store string) (context.Context, error) {
	raw := os.Getenv("ALAN_BARRIER")
	if raw == "" {
		return ctx, nil
	}
	var filter Event
	if err := json.Unmarshal([]byte(raw), &filter); err != nil {
		return ctx, err
	}
	if filter.Scenario == "" || filter.Token == "" || filter.Boundary == "" || filter.Store != store {
		return ctx, fmt.Errorf("invalid barrier identity")
	}
	output := os.NewFile(3, "boundary-ack")
	input := os.NewFile(4, "boundary-release")
	var mu sync.Mutex
	done := false
	return WithObserver(ctx, func(ctx context.Context, event Event) error {
		mu.Lock()
		defer mu.Unlock()
		if done || !matches(filter, event) {
			return nil
		}
		event.Scenario = filter.Scenario
		event.Token = filter.Token
		event.Store = store
		event.PID = os.Getpid()
		if err := json.NewEncoder(output).Encode(event); err != nil {
			return err
		}
		var release [1]byte
		if _, err := io.ReadFull(input, release[:]); err != nil {
			return err
		}
		if release[0] != 'c' {
			return fmt.Errorf("invalid barrier release")
		}
		done = true
		return ctx.Err()
	}), nil
}

// Zero numeric and empty string filter fields are wildcards; specified fields match exactly.
func matches(filter, event Event) bool {
	return !(event.Boundary != filter.Boundary || (filter.Subject != "" && event.Subject != filter.Subject) || (filter.Sequence != 0 && event.Sequence != filter.Sequence) || (filter.Generation != 0 && event.Generation != filter.Generation) || (filter.Epoch != 0 && event.Epoch != filter.Epoch) || (filter.Attempt != "" && event.Attempt != filter.Attempt) || (filter.Phase != "" && event.Phase != filter.Phase) || (filter.Intent != "" && event.Intent != filter.Intent))
}
