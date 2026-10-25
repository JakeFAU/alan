package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestCLIRejectsExplicitNonpositiveDispatchOptionsBeforeOpen(t *testing.T) {
	for _, args := range [][]string{{"--max-in-flight", "0"}, {"--max-in-flight", "-1"}, {"--external-timeout", "0s"}, {"--external-timeout", "-1s"}} {
		all := append([]string{"run", "--db", "nonexistent-no-open.sqlite"}, args...)
		var out bytes.Buffer
		var usage usageError
		if err := runCLI(t.Context(), all, &out); !errors.As(err, &usage) {
			t.Fatalf("invalid options reached store open %v: %v", args, err)
		}
	}
}
