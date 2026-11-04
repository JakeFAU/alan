package main

import (
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// A new weather agent waits directly for runtime approval. Losing the first
// provider response must never strand it waiting on an extra peer handshake.
func TestWeatherSuccessorWaitsOnlyForProtectedApproval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "weather.sqlite")
	appCommand(t, provisioning(path, "http://127.0.0.1:1/v1/forecast", 2)...)
	s, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	next, _ := s.Run(readID(2))
	if next.Wait == nil || next.Wait.Kind != "input" || next.Wait.Obligation != declaredWeatherSlot(t, s, 2).Obligation {
		t.Fatalf("successor first needs %+v", next.Wait)
	}
}
