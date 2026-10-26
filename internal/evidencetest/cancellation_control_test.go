//go:build integration

package evidencetest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

// Use the actual stopped CLI input and its wildcard filter from review CR1.
// A positive epoch/sequence is insufficient unless it matches durable authority.
func TestCancellationWildcardBoundaryRejectsDurableSubstitutions(t *testing.T) {
	raw, err := os.ReadFile("testdata/cancellation-dequeue-boundary.json")
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		Filters []boundary.Event                      `json:"filters"`
		Ack     boundary.Event                        `json:"acknowledgment"`
		Before  struct{ Snapshot runtime.Inspection } `json:"before"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if len(input.Filters) != 1 || input.Filters[0].Epoch != 0 || input.Filters[0].Sequence != 0 || input.Filters[0].Boundary != "runner.received" {
		t.Fatal("control must retain the actual wildcard dequeue filter")
	}
	for _, tc := range []struct {
		name            string
		epoch, sequence int64
		reject          bool
	}{
		{"original", 0, 0, false}, {"epoch", 7, 0, true}, {"sequence", 0, 9, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := input.Ack
			got.Epoch += tc.epoch
			got.Sequence += tc.sequence
			err := validateAcknowledgment(got, input.Filters, input.Ack.PID, map[string]bool{})
			if err != nil {
				t.Fatalf("actual wildcard filter unexpectedly rejected input: %v", err)
			}
			err = validateDurableBoundary(input.Before.Snapshot, got)
			if (err != nil) != tc.reject {
				t.Fatalf("durable rejection=%t, want %t: %v", err != nil, tc.reject, err)
			}
			emit(t, map[string]any{"type": "cancellation-durable-identity-control", "case": tc.name, "filters": input.Filters, "acknowledgment": got, "durable_tables": input.Before.Snapshot.Tables, "rejected": err != nil, "result": fmt.Sprint(err)})
		})
	}
}
