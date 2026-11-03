package boundary

import "testing"

func TestSpecifiedGenerationEpochOperationAndAttemptFilterExactly(t *testing.T) {
	filter := Event{Boundary: "lifecycle.decided", Subject: "inner", Generation: 1, Epoch: 2, Sequence: 7, Intent: "opA", Attempt: "authA", Phase: "accepted"}
	if !matches(filter, filter) {
		t.Fatal("matching event skipped")
	}
	for _, change := range []func(*Event){func(e *Event) { e.Generation = 2 }, func(e *Event) { e.Epoch = 99 }, func(e *Event) { e.Intent = "opB" }, func(e *Event) { e.Attempt = "authB" }, func(e *Event) { e.Sequence = 6 }, func(e *Event) { e.Phase = "rejected" }} {
		wrong := filter
		change(&wrong)
		if matches(filter, wrong) {
			t.Fatal("wrong event matches", wrong)
		}
	}
	if !matches(Event{Boundary: "lifecycle.decided"}, filter) {
		t.Fatal("documented wildcard semantics changed")
	}
}
