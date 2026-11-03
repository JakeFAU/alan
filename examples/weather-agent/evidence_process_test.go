package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// The trace is derived from the same detached snapshots and independent
// endpoint records checked by the campaign. It does not drive a test verdict.
func weatherFacts(t *testing.T, s *runtime.ExecutionSnapshot) []map[string]any {
	t.Helper()
	facts := make([]map[string]any, 0, len(s.Effects))
	for _, e := range s.Effects {
		run, _ := s.Run(e.Source)
		f := map[string]any{"operation": e.OperationID, "stage": e.State, "run": run.Projection.Status}
		f["execution_bounds"] = []int{e.Dispatch.Minimum, *e.Dispatch.Maximum}
		if previous := run.Projection.Data["previous_operation_id"].Text; previous != "" {
			f["previous_operation"] = previous
			for _, o := range s.Observations {
				if o.Run == e.Source && o.Provenance == "local-input" && o.Obligation == "approve-refresh" && o.SelectedAt != nil {
					f["selected_approval"] = o.ID
					f["approval_reason"] = o.Payload["purpose"].Text
					f["approved_prepared_digest"] = o.Payload["prepared_digest"].Text
					if digest := o.Payload["result_digest"].Text; digest != "" {
						f["approved_result_digest"] = digest
					}
				}
			}
		}
		if e.Authorization != nil {
			f["authorization"] = e.Authorization.ID
		}
		if a := e.AdapterAttempt; a != nil {
			f["charged"] = a.Allowed
			f["prepared_sha256"] = a.Prepared.Digest
		}
		if a := e.DispatchAdmission; a != nil {
			f["admitted_after_fact"] = a.AfterFact
		}
		if r := e.AdapterResult; r != nil {
			f["disposition"] = r.Disposition
			if r.Candidate.HTTPStatus != 0 {
				f["response_sha256"] = fmt.Sprintf("%x", sha256.Sum256([]byte(r.Candidate.Response)))
			}
			if v := r.Candidate.Payload["observed_at"].Text; v != "" {
				f["provider_time"] = v
			}
		}
		if at := selectedWeather(t, s, e); at != nil {
			f["selected_sequence"] = at.Sequence
		}
		facts = append(facts, f)
	}
	return facts
}

func weatherInterruption(t *testing.T, barrier string, pid, contacts int, s *runtime.ExecutionSnapshot) map[string]any {
	t.Helper()
	return map[string]any{"ack": barrier, "sigkill_pid": pid, "contacts_when_stopped": contacts, "effects": weatherFacts(t, s)}
}

func recordWeatherCase(t *testing.T, s *runtime.ExecutionSnapshot, p *weatherProvider, pauses ...map[string]any) {
	t.Helper()
	contacts := make([]map[string]any, 0)
	for _, c := range p.contacts() {
		item := map[string]any{"index": c.index, "operation": c.operation, "client_id": c.clientID, "get": c.uri}
		if c.written {
			item["status"] = c.status
			item["written_sha256"] = fmt.Sprintf("%x", sha256.Sum256([]byte(c.body)))
		}
		contacts = append(contacts, item)
	}
	var spent, allowance int64
	for _, root := range s.Roots {
		spent += root.Charges.Effects
		allowance += root.Limits.MaxEffects
	}
	row := map[string]any{
		"case": strings.TrimPrefix(t.Name(), "TestWeatherProcessRecovery/"), "namespace": s.Namespace,
		"allowance": allowance, "charged": spent, "contacts": contacts, "final": weatherFacts(t, s),
	}
	if len(pauses) != 0 {
		row["killed"] = pauses
	}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("M4_CONTROLLED %s", raw)
}
