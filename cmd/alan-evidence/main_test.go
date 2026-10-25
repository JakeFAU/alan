package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCLIUsage(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing", nil}, {"unknown", []string{"other"}}, {"db", []string{"run"}}, {"extra", []string{"replay", "--db", "a", "extra"}}, {"flag", []string{"run", "--db", "a", "--model", "x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := runCLI(context.Background(), tc.args, &out, func(string) string { return "" })
			var usage usageError
			if !errors.As(err, &usage) {
				t.Fatalf("expected usage error, got %v", err)
			}
		})
	}
}
func TestCLIHelp(t *testing.T) {
	var out bytes.Buffer
	if err := runCLI(t.Context(), []string{"init", "--help"}, &out, func(string) string { return "" }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "model") || !strings.Contains(out.String(), "corpus") {
		t.Fatalf("help: %s", out.String())
	}
}
