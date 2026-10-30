package main

import (
	"bytes"
	"testing"
)

// CLI validation regression reused from the Phase 1L consumer.
func TestExecuteRejectsMissingDatabase(t *testing.T) {
	var out bytes.Buffer
	if err := execute(t.Context(), []string{"run"}, &out); err == nil {
		t.Fatal("missing database accepted")
	}
}

func TestExecuteRejectsUnknownPolicy(t *testing.T) {
	var out bytes.Buffer
	if err := execute(t.Context(), []string{"artifacts", "-policy", "unknown"}, &out); err == nil {
		t.Fatal("unknown policy accepted")
	}
}
