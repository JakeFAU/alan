package main

import (
	"bytes"
	"testing"
)

func TestExecuteRejectsMissingDatabase(t *testing.T) {
	var out bytes.Buffer
	if err := execute(t.Context(), []string{"run"}, &out); err == nil {
		t.Fatal("missing database accepted")
	}
}
