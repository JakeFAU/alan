package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	_ "github.com/mattn/go-sqlite3"
)

// The writer is always the ordinary application. SQL is confined to disposable
// adversarial copies; no original positive fixture or historical archive changes.
func TestWeatherSeriesCorruption(t *testing.T) {
	binary := publicWeatherBinary(t)
	p := newWeatherProvider(t, seriesReplies()...)
	original := startWeather(t, binary, p.endpoint(), 8)
	completeSeriesThrough(t, binary, original, 8)
	positive, _ := inspectAndReplayWeather(t, binary, original, p, 8, 8)
	originalBytes := weatherDBBytes(t, original)
	cases := []struct{ name, mutation string }{
		{"position_identity", `UPDATE subjects SET id='weather-03' WHERE id='weather-3'`},
		{"generation", `UPDATE runs SET generation=2 WHERE subject='weather-3'`},
		{"slot_predecessor", `UPDATE composition_profile SET descriptor=CAST(json_set(descriptor,'$.Continuations[2].Predecessor.Subject','weather-1') AS BLOB)`},
		{"approval_target", `UPDATE composition_profile SET descriptor=CAST(json_set(descriptor,'$.Continuations[2].Target.Subject','weather-8') AS BLOB)`},
		{"scoped_grant", `UPDATE permissions SET target='foreign-provider' WHERE kind='adapter' AND source='weather-3'`},
		{"root_allowance", `UPDATE composition_profile SET descriptor=CAST(json_set(descriptor,'$.Roots[0].Limits.MaxEffects',2) AS BLOB)`},
		{"selected_approval", `UPDATE observations SET consumed_sequence=2 WHERE subject='weather-4' AND json_extract(descriptor,'$.Provenance')='local-input'`},
		{"selected_observation", `UPDATE observations SET consumed_sequence=4 WHERE subject='weather-3' AND json_extract(descriptor,'$.Kind')='adapter'`},
		{"observation_relation", `UPDATE observations SET descriptor=CAST(json_set(descriptor,'$.Operation','foreign-operation') AS BLOB) WHERE subject='weather-3' AND json_extract(descriptor,'$.Kind')='adapter'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copy := copyWeatherApprovalCut(t, original)
			editor, err := sql.Open("sqlite3", copy)
			if err != nil {
				t.Fatal(err)
			}
			result, err := editor.Exec(tc.mutation)
			if err != nil {
				editor.Close()
				t.Fatal(err)
			}
			rows, err := result.RowsAffected()
			if err != nil || rows < 1 {
				t.Fatalf("mutation did not affect a real row: %d %v", rows, err)
			}
			if err := editor.Close(); err != nil {
				t.Fatal(err)
			}
			before := weatherDBBytes(t, copy)
			if _, err := runtime.Inspect(t.Context(), copy, runtime.DefaultLimits()); err == nil {
				t.Error("Inspect accepted impossible history")
			}
			if _, err := runtime.InspectExecution(t.Context(), copy, runtime.DefaultLimits()); err == nil {
				t.Error("InspectExecution accepted impossible history")
			}
			if _, err := runtime.Replay(t.Context(), copy, runtime.DefaultLimits()); err == nil {
				t.Error("Replay accepted impossible history")
			}
			var output bytes.Buffer
			if err := run(t.Context(), []string{"status", "--db", copy}, &output); err == nil || output.Len() != 0 {
				t.Errorf("status did not reject impossible history (err=%v, bytes=%d)", err, output.Len())
			}
			if before != weatherDBBytes(t, copy) {
				t.Fatal("readers modified corrupt DB/WAL")
			}
			owner, err := runtime.Open(t.Context(), copy, runtime.DefaultLimits())
			if err == nil {
				owner.Close()
				t.Error("Open accepted impossible history")
			}
			if before != weatherDBBytes(t, copy) {
				t.Fatal("failed Open modified corrupt DB/WAL")
			}
			p.requireContacts(t, 8)
		})
	}
	// Independently valid alternative deployments demonstrate source matching,
	// without depending on the runtime corruption checks above.
	t.Run("valid_foreign_declaration", func(t *testing.T) {
		series := seriesDeclaration(t, p)
		provision, err := series.Provision()
		if err != nil {
			t.Fatal(err)
		}
		provision.Composition.Continuations[2].ApprovalID = "foreign-approval"
		db := filepath.Join(t.TempDir(), "foreign.sqlite")
		owner, err := runtime.Create(t.Context(), db, provision, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if err := owner.Close(); err != nil {
			t.Fatal(err)
		}
		cut := weatherSnapshot(t, db)
		if _, err := series.Inspect(cut); err == nil {
			t.Fatal("exact declaration accepted changed approval identity")
		}
		if _, err := retainedConfiguration(cut); err == nil {
			t.Fatal("damaged series escaped through legacy fallback")
		}
	})
	if originalBytes != weatherDBBytes(t, original) {
		t.Fatal("positive fixture modified")
	}
	if _, err := seriesDeclaration(t, p).Inspect(positive); err != nil {
		t.Fatal(err)
	}
	p.requireContacts(t, 8)
}
