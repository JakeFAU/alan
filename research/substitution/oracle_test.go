package substitution

import "testing"

func TestIndependentFaultControls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		events []Observation
		code   string
	}{
		{"wrong-id", []Observation{{Kind: "reply_admit", Operation: "e", Role: "candidate", ID: 2, Value: 1, Present: true}}, "application_id"},
		{"extra-helper", []Observation{{Kind: "invoke_commit", Operation: "h1", Parent: "e", Role: "helper"}, {Kind: "invoke_commit", Operation: "h2", Parent: "e", Role: "helper"}}, "extra_helper"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckTrace(Request{ID: 1, Value: 0, Mode: "answer"}, tc.events)
			if len(got) == 0 || got[0].Code != tc.code {
				t.Fatalf("want %s, got %+v", tc.code, got)
			}
		})
	}
}
