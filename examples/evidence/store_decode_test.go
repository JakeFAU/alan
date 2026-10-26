package evidence

import (
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestInspectionRunsRejectMalformedRows(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(runtime.Inspection)
	}{
		{"missing subject ID", func(s runtime.Inspection) { delete(s.Tables["subjects"][0], "id") }},
		{"missing revision", func(s runtime.Inspection) { delete(s.Tables["subjects"][0], "revision") }},
		{"null revision", func(s runtime.Inspection) { s.Tables["subjects"][0]["revision"] = nil }},
		{"wrong revision type", func(s runtime.Inspection) { s.Tables["subjects"][0]["revision"] = "one" }},
		{"missing sequence", func(s runtime.Inspection) { delete(s.Tables["runs"][0], "sequence") }},
		{"null program", func(s runtime.Inspection) { s.Tables["runs"][0]["program_id"] = nil }},
		{"negative generation", func(s runtime.Inspection) { s.Tables["runs"][0]["generation"] = int64(-1) }},
		{"wrong sequence type", func(s runtime.Inspection) { s.Tables["runs"][0]["sequence"] = "one" }},
		{"unlinked subject", func(s runtime.Inspection) { s.Tables["runs"][0]["subject"] = "missing" }},
		{"missing projection", func(s runtime.Inspection) { delete(s.Tables["runs"][0], "projection") }},
		{"invalid projection JSON", func(s runtime.Inspection) { s.Tables["runs"][0]["projection"] = "{" }},
		{"null projection JSON", func(s runtime.Inspection) { s.Tables["runs"][0]["projection"] = "null" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := runtime.Inspection{Tables: map[string][]map[string]any{
				"subjects": {{"id": "caller", "revision": int64(1)}},
				"runs":     {{"subject": "caller", "generation": int64(1), "program_id": "program", "epoch": int64(1), "sequence": int64(0), "projection": `{"PC":0,"Data":{},"Status":"open"}`}},
			}}
			tc.mutate(snapshot)
			if _, err := inspectionRuns(snapshot); err == nil {
				t.Fatal("malformed inspection row accepted")
			}
		})
	}
}
