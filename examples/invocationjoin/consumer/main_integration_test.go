//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestConsumerOriginalOutcomeJoins(t *testing.T) {
	for _, tc := range []struct {
		name    string
		charges int64
	}{{"a", 2}, {"b", 3}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "retained.db")
			for _, command := range []string{"init", "run", "inspect", "replay", "recover"} {
				var out bytes.Buffer
				if err := execute(t.Context(), []string{command, "-db", path, "-candidate", tc.name}, &out); err != nil {
					t.Fatal(command, err)
				}
				if !json.Valid(out.Bytes()) {
					t.Fatal("invalid output", out.String())
				}
			}
			inspection, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if len(inspection.Roots) != 1 {
				t.Fatal(inspection.Roots)
			}
			root := inspection.Roots[0]
			if root.LocalStatus != "succeeded" || root.Outcome == nil || root.Outcome.Status != "succeeded" {
				t.Fatal(root)
			}
			if root.Charges.Invocations != tc.charges || root.Charges.CreatedRuns != tc.charges || root.Charges.Effects != 0 {
				t.Fatal(root.Charges)
			}
			checkOriginalJoins(t, inspection)
		})
	}
}

func decode[T any](t *testing.T, row map[string]any, column string) T {
	t.Helper()
	var result T
	raw, ok := row[column].(string)
	if !ok {
		t.Fatalf("missing %s", column)
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func checkOriginalJoins(t *testing.T, s runtime.Inspection) {
	t.Helper()
	var parent runtime.Projection
	for _, row := range s.Tables["runs"] {
		if row["subject"] == "root" {
			parent = decode[runtime.Projection](t, row, "projection")
		}
	}
	if parent.Status != "succeeded" || parent.Data["candidate_id"].Number != 1 || parent.Data["candidate_value"].Number != 1 || parent.Data["s_id"].Number != 1 || parent.Data["s_value"].Number != 0 {
		t.Fatal("actual parent result", parent)
	}
	candidate := parent.Data["candidate_operation"].Text
	successor := parent.Data["s_operation"].Text
	if candidate == "" || successor == "" || candidate == successor {
		t.Fatal("missing distinct original handles", parent.Data)
	}
	var latestCommit, firstConsumption int64
	for _, operation := range []string{candidate, successor} {
		var commit int64
		for _, row := range s.Tables["intents"] {
			if row["id"] == operation {
				intent := decode[struct {
					Kind     string
					Run      runtime.RunID
					Sequence int64
				}](t, row, "descriptor")
				if intent.Kind != "invocation" || intent.Run.Subject != "root" {
					t.Fatal("not parent's original invocation", intent)
				}
				commit = intent.Sequence
			}
		}
		if commit == 0 {
			t.Fatal("missing original intent", operation)
		}
		if commit > latestCommit {
			latestCommit = commit
		}
		admitted := false
		for _, row := range s.Tables["composition_invocations"] {
			if row["operation_id"] == operation {
				admitted = decode[runtime.InvocationAdmission](t, row, "admission").Accepted
			}
		}
		if !admitted {
			t.Fatal("missing original admission", operation)
		}
		var observationID string
		for _, row := range s.Tables["composition_outcome_deliveries"] {
			if row["operation_id"] == operation {
				observationID, _ = row["observation_id"].(string)
			}
		}
		var consumed int64
		for _, row := range s.Tables["observations"] {
			if row["id"] == observationID {
				observation := decode[struct {
					Operation, Kind, Provenance string
					Run                         runtime.RunID
					Payload                     map[string]runtime.Value
				}](t, row, "descriptor")
				if observation.Operation != operation || observation.Kind != "invocation" || observation.Provenance != "runtime-invocation" || observation.Run.Subject != "root" || observation.Payload["status"].Text != "succeeded" || !observation.Payload["response_present"].Flag {
					t.Fatal("wrong original selected outcome", observation)
				}
				consumed, _ = row["consumed_sequence"].(int64)
			}
		}
		if consumed == 0 {
			t.Fatal("original outcome not consumed", operation)
		}
		if firstConsumption == 0 || consumed < firstConsumption {
			firstConsumption = consumed
		}
		selected := false
		for _, row := range s.Tables["history"] {
			if row["subject"] == "root" && row["sequence"] == consumed && row["selected_observation"] == observationID {
				selected = true
			}
		}
		if !selected {
			t.Fatal("missing authentic selected history", operation, observationID, consumed)
		}
	}
	if latestCommit >= firstConsumption {
		t.Fatal("both invokes must commit before first await", latestCommit, firstConsumption)
	}
}

func TestConsumerNegativeStillConsumesSuccessor(t *testing.T) {
	for _, mode := range []string{"fail", "absent"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "retained.db")
			for _, command := range []string{"init", "run"} {
				var out bytes.Buffer
				if err := execute(t.Context(), []string{command, "-db", path, "-mode", mode}, &out); err != nil {
					t.Fatal(command, err)
				}
			}
			s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Roots) != 1 || s.Roots[0].LocalStatus != "failed" || s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "failed" {
				t.Fatal(s.Roots)
			}
			var parent runtime.Projection
			for _, row := range s.Tables["runs"] {
				if row["subject"] == "root" {
					parent = decode[runtime.Projection](t, row, "projection")
				}
			}
			if parent.Data["candidate_present"].Flag || parent.Data["s_status"].Text != "succeeded" || !parent.Data["s_present"].Flag {
				t.Fatal("negative skipped required successor", parent)
			}
			for _, field := range []string{"candidate_id", "candidate_value"} {
				if _, ok := parent.Data[field]; ok {
					t.Fatal("invented absent response binding", field)
				}
			}
			for _, field := range []string{"candidate_operation", "s_operation"} {
				operation := parent.Data[field].Text
				count := 0
				for _, row := range s.Tables["observations"] {
					observation := decode[struct {
						Operation string
						Run       runtime.RunID
					}](t, row, "descriptor")
					if observation.Run.Subject == "root" && observation.Operation == operation && row["consumed_sequence"] != nil {
						count++
					}
				}
				if operation == "" || count != 1 {
					t.Fatal("required original outcome consumption", field, operation, count)
				}
			}
		})
	}
}
