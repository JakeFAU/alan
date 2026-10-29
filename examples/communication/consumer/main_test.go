package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestConsumerTerminalProbe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terminal.db")
	var out bytes.Buffer
	if err := execute(t.Context(), []string{"-db", path}, &out); err != nil {
		t.Fatal(err)
	}
	var result struct{ SourceStatus, ReceiverStatus, ObservationID string }
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SourceStatus != "succeeded" || result.ReceiverStatus != "succeeded" || result.ObservationID == "" {
		t.Fatal("not a terminal-access result", result)
	}
	s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Tables["observations"]) != 1 || s.Tables["observations"][0]["consumed_sequence"] == nil {
		t.Fatal("executable did not actually consume terminal observation")
	}
}
func TestConsumerRejectsInvalidArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "missing_db"}, {name: "unknown_flag", args: []string{"-unknown"}},
		{name: "positional", args: []string{"extra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := execute(t.Context(), tc.args, &bytes.Buffer{}); err == nil {
				t.Fatal("accepted invalid probe arguments")
			}
		})
	}
}
