package durabletest

import (
	"encoding/json"
	"os"
	"testing"
)

func TestBarrierRejectsCrossExecutionAndUnorderedEvidence(t *testing.T) {
	want := barrier{Scenario: "original-probe", Token: "start-A", PID: 123, Store: "store.sqlite", Boundary: "lifecycle.decided", Phase: "accepted", Subject: "inner", Generation: 1, Epoch: 2, Sequence: 7, Intent: "opA", Attempt: "authA"}
	if !matchingBarrier(want, want, 123) {
		t.Fatal("matching acknowledgment rejected")
	}
	for _, change := range []struct {
		name  string
		apply func(*barrier)
	}{
		{"generation", func(b *barrier) { b.Generation = 2 }}, {"epoch", func(b *barrier) { b.Epoch = 99 }}, {"operation", func(b *barrier) { b.Intent = "opB" }}, {"attempt", func(b *barrier) { b.Attempt = "authB" }},
		{"older sequence", func(b *barrier) { b.Sequence = 6 }}, {"newer sequence", func(b *barrier) { b.Sequence = 8 }}, {"wrong phase", func(b *barrier) { b.Phase = "pre-commit" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			wrong := want
			change.apply(&wrong)
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer read.Close()
			defer write.Close()
			if err = json.NewEncoder(write).Encode(wrong); err != nil {
				t.Fatal(err)
			}
			if _, err = readAcknowledgment(read, want, 123); err == nil {
				t.Fatalf("accepted cross-execution/order evidence: %+v expected %+v", wrong, want)
			}
		})
	}
}
