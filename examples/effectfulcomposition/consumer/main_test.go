package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestPublicProvisionAndReplay(t *testing.T) {
	db := filepath.Join(t.TempDir(), "store.sqlite")
	ep := "http://127.0.0.1:1/phase2c/" + strings.Repeat("a", 64) + "/apply"
	if _, err := execute(t.Context(), []string{"init", "-db", db, "-endpoint", ep}); err != nil {
		t.Fatal(err)
	}
	report, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots) != 3 || len(report.Tables["receipt_read_ordinals"]) != 0 || len(report.Tables["adapter_attempts"]) != 0 {
		t.Fatal("initial root or debit population", report.Roots)
	}
	if _, err := execute(t.Context(), []string{"resolve", "-db", db, "-endpoint", ep, "-role", "l"}); err == nil || !strings.Contains(err.Error(), "missing l invocation") {
		t.Fatal("service before authentic target handoff", err)
	}
	if _, err := execute(t.Context(), []string{"replay", "-db", db}); err != nil {
		t.Fatal(err)
	}
}
