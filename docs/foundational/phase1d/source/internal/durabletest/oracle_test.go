package durabletest

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"reflect"
)

// checkRecords is intentionally independent of Reduce, Replay, authority policy,
// and all fault injection paths. It compares retained evidence and simple links.
func checkRecords(before, after executor.Inspection, endpoint resource.Snapshot) []string {
	failures := []string{}
	for _, table := range []string{"history", "intents", "attempts", "lifecycle_results", "external_attempts", "external_receipts", "settlements", "run_origins"} {
		for _, old := range before.Tables[table] {
			found := false
			for _, row := range after.Tables[table] {
				if reflect.DeepEqual(old, row) {
					found = true
					break
				}
			}
			if !found {
				failures = append(failures, "retained "+table+" row changed or lost")
			}
		}
	}
	for _, run := range after.Tables["runs"] {
		var last map[string]any
		for _, h := range after.Tables["history"] {
			if h["subject"] == run["subject"] && h["generation"] == run["generation"] {
				if last == nil || h["sequence"].(int64) > last["sequence"].(int64) {
					last = h
				}
			}
		}
		expected := run["initial"]
		if last != nil {
			expected = last["successor"]
		}
		closedByAuthority := false
		for _, terminal := range after.Tables["terminal_obligations"] {
			if terminal["subject"] == run["subject"] && terminal["generation"] == run["generation"] && terminal["operation_id"] != nil {
				closedByAuthority = true
			}
		}
		if !closedByAuthority && expected != run["projection"] {
			failures = append(failures, "projection disagrees with committed ordinary history")
		}
	}
	applications := map[string]int{}
	for _, application := range endpoint.Applications {
		applications[application.Key]++
		committed, authorized := false, false
		for _, row := range after.Tables["intents"] {
			var intent executor.Intent
			json.Unmarshal([]byte(row["descriptor"].(string)), &intent)
			if intent.ID == application.Key && intent.Kind == "external" && intent.Payload["delta"] == executor.Int(application.Delta) {
				committed = true
			}
		}
		for _, row := range after.Tables["external_attempts"] {
			var attempt executor.ExternalAttempt
			json.Unmarshal([]byte(row["descriptor"].(string)), &attempt)
			if attempt.Intent.ID == application.Key && attempt.Allowed && attempt.Request.Key == application.Key {
				authorized = true
			}
		}
		if !committed || !authorized {
			failures = append(failures, "endpoint application lacks committed authorized intent")
		}
		if application.Profile == "idempotent" && applications[application.Key] > 1 {
			failures = append(failures, "duplicate idempotent mutation")
		}
	}
	return failures
}
func projection(snapshot executor.Inspection, subject string, generation int64) (executor.Projection, int64, int64) {
	for _, row := range snapshot.Tables["runs"] {
		if row["subject"] == subject && row["generation"] == generation {
			var p executor.Projection
			json.Unmarshal([]byte(row["projection"].(string)), &p)
			return p, row["epoch"].(int64), row["sequence"].(int64)
		}
	}
	return executor.Projection{}, 0, 0
}
func checkNested(snapshot executor.Inspection, outer, inner, worker string) error {
	p, _, _ := projection(snapshot, inner, 1)
	o, _, _ := projection(snapshot, outer, 1)
	w, _, seq := projection(snapshot, worker, 2)
	if p.Status != "succeeded" || p.Data["spent"] != executor.Int(1) || p.Data["expected"] != executor.Int(2) || o.Status != "succeeded" || w.Status != "succeeded" || seq < 3 {
		return fmt.Errorf("nested execution did not retain one budget charge and execute successful successor")
	}
	lifecycle := 0
	var operation string
	for _, row := range snapshot.Tables["intents"] {
		var intent executor.Intent
		json.Unmarshal([]byte(row["descriptor"].(string)), &intent)
		if intent.Kind == "lifecycle" {
			lifecycle++
			operation = intent.ID
			if intent.Run != (executor.RunID{Subject: inner, Generation: 1}) {
				return fmt.Errorf("supervisor generation changed")
			}
		}
	}
	if lifecycle != 1 || len(snapshot.Tables["lifecycle_results"]) != 1 {
		return fmt.Errorf("duplicate lifecycle operation")
	}
	consumed := false
	for _, row := range snapshot.Tables["observations"] {
		var ob executor.Observation
		json.Unmarshal([]byte(row["descriptor"].(string)), &ob)
		if ob.Kind == "lifecycle" && ob.Operation == operation && ob.Run == (executor.RunID{Subject: inner, Generation: 1}) && row["consumed_sequence"] != nil {
			consumed = true
		}
	}
	if !consumed {
		return fmt.Errorf("original lifecycle result unconsumed")
	}
	var program1, program2 any
	for _, row := range snapshot.Tables["runs"] {
		if row["subject"] == worker {
			if row["generation"] == int64(1) {
				program1 = row["program_id"]
			}
			if row["generation"] == int64(2) {
				program2 = row["program_id"]
			}
		}
	}
	if program1 != program2 {
		return fmt.Errorf("successor artifact changed")
	}
	return nil
}
