// Package boundary reports real execution boundaries and optionally pauses on an
// inherited pipe. It never selects work, changes state, or dispatches operations.
package boundary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
)

type Event struct {
	Scenario, Token, Store, Boundary, Phase, Subject string
	PID                                              int
	Generation, Epoch, Sequence                      int64
	Intent, Attempt                                  string
	InFlight                                         []string
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
	var filters []Event
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal([]byte(raw), &filters); err != nil {
			return ctx, err
		}
	} else {
		var filter Event
		if err := json.Unmarshal([]byte(raw), &filter); err != nil {
			return ctx, err
		}
		filters = []Event{filter}
	}
	if len(filters) == 0 {
		return ctx, fmt.Errorf("empty barrier filters")
	}
	for _, filter := range filters {
		if filter.Scenario == "" || filter.Token == "" || filter.Boundary == "" || filter.Store != store {
			return ctx, fmt.Errorf("invalid barrier identity")
		}
	}
	output := os.NewFile(3, "boundary-ack")
	if err := syscall.SetNonblock(4, true); err != nil {
		return ctx, err
	}
	input := os.NewFile(4, "boundary-release")
	var mu sync.Mutex
	var pause sync.Mutex
	used := make([]bool, len(filters))
	active := map[string]string{}
	return WithObserver(ctx, func(ctx context.Context, event Event) error {
		mu.Lock()
		if event.Boundary == "external.authorized" {
			active[event.Intent] = event.Attempt
		}
		if event.Boundary == "external.finished" {
			delete(active, event.Intent)
		}
		for _, attempt := range active {
			event.InFlight = append(event.InFlight, attempt)
		}
		index := -1
		for i, filter := range filters {
			if !used[i] && matches(filter, event) {
				index = i
				used[i] = true
				break
			}
		}
		mu.Unlock()
		if index < 0 {
			return nil
		}
		filter := filters[index]
		event.Scenario = filter.Scenario
		event.Token = filter.Token
		event.Store = store
		event.PID = os.Getpid()
		// Only selected pauses serialize. Unselected worker boundaries never wait here.
		pause.Lock()
		defer pause.Unlock()
		if err := json.NewEncoder(output).Encode(event); err != nil {
			return err
		}
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := input.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
				return err
			}
			var release [1]byte
			_, err := io.ReadFull(input, release[:])
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			if err != nil {
				return err
			}
			if release[0] == 'p' {
				if err := ctx.Err(); err != nil {
					return err
				}
				probe := event
				probe.Phase = "holding"
				if err := json.NewEncoder(output).Encode(probe); err != nil {
					return err
				}
				continue
			}
			if release[0] != 'c' {
				return fmt.Errorf("invalid barrier release")
			}
			return ctx.Err()
		}
	}), nil
}

// Zero numeric and empty string filter fields are wildcards; specified fields match exactly.
func matches(filter, event Event) bool {
	return !(event.Boundary != filter.Boundary || (filter.Subject != "" && event.Subject != filter.Subject) || (filter.Sequence != 0 && event.Sequence != filter.Sequence) || (filter.Generation != 0 && event.Generation != filter.Generation) || (filter.Epoch != 0 && event.Epoch != filter.Epoch) || (filter.Attempt != "" && event.Attempt != filter.Attempt) || (filter.Phase != "" && event.Phase != filter.Phase) || (filter.Intent != "" && event.Intent != filter.Intent))
}

// Inherit preserves a process observer while using the request cancellation lifetime.
func Inherit(ctx, source context.Context) context.Context {
	if observer, ok := source.Value(observerKey{}).(Observer); ok {
		return WithObserver(ctx, observer)
	}
	return ctx
}
