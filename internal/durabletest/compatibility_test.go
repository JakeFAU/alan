//go:build integration

package durabletest

import (
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/internal/storetest"
	rt "github.com/JakeFAU/alan/runtime"
)

func TestPhase2BExactPriorStores(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			path := storetest.Copy(t, "recorded-"+route)
			before := inspect(t, path)
			originalKey, originalAttempt := phaseRequest(t, before, route)
			_, err := rt.Replay(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, inspect(t, path)) {
				t.Fatal("replay changed pinned earlier state")
			}
			s, err := rt.Open(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var options = rt.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}
			// Neither this known raw receipt nor the recorded adapter result requires a live endpoint/registry.
			report, err := s.Run(t.Context(), options)
			if err != nil {
				t.Fatal(err)
			}
			root := rt.RunID{Subject: "client", Generation: 1}
			if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, root, false)); err != nil {
				t.Fatal(err)
			}
			after := inspect(t, path)
			key, attempt := phaseRequest(t, after, route)
			state, _, _ := projection(after, "client", 1)
			if state.Status != "succeeded" || key != originalKey || attempt != originalAttempt || phaseResultCount(before, route) != 1 || phaseResultCount(after, route) != 1 {
				t.Fatal(state, report)
			}
			_, err = rt.Replay(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, inspect(t, path)) {
				t.Fatal("replay changed recovered state")
			}

		})
	}
}
