package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// The evidence export is derived after the independent checks. Neither the
// scheduler nor the test verdict reads an exported trace.
func continuationStop(t *testing.T, witness string, pid, contacts int, s *runtime.ExecutionSnapshot) map[string]any {
	t.Helper()
	cut := weatherInterruption(t, witness, pid, contacts, s)
	d, approved := s.Continuation(runtime.RunID{Subject: "weather-2", Generation: 1})
	cut["approval_committed"] = approved
	if approved {
		cut["approval_order"] = d.DecisionOrder
		cut["approval_observation"] = d.ObservationID
		if d.SelectedAt != nil {
			cut["approval_selected"] = d.SelectedAt.Sequence
		}
	}
	return cut
}

func recordContinuationCase(t *testing.T, s *runtime.ExecutionSnapshot, p *weatherProvider, stops ...map[string]any) {
	t.Helper()
	d, ok := s.Continuation(runtime.RunID{Subject: "weather-2", Generation: 1})
	if !ok || d.SelectedAt == nil {
		t.Fatal("missing selected authority in final evidence")
	}
	httpLog := make([]map[string]any, 0)
	for _, c := range p.contacts() {
		entry := map[string]any{"original": c.operation, "client_id": c.clientID}
		if c.written {
			entry["http_status"] = c.status
			entry["response_sha256"] = fmt.Sprintf("%x", sha256.Sum256([]byte(c.body)))
		}
		httpLog = append(httpLog, entry)
	}
	var spent int64
	for _, r := range s.Roots {
		spent += r.Charges.Effects
	}
	row := map[string]any{"case": t.Name(), "namespace": s.Namespace, "stops": stops, "contacts": httpLog, "charged": spent,
		"approval": d.Approval, "decision_order": d.DecisionOrder, "selected": d.SelectedAt, "approval_observation": d.ObservationID,
		"successor": d.SuccessorOperationID, "effects": weatherFacts(t, s)}
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("M5_CONTROLLED %s", b)
}
