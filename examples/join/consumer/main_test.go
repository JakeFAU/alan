package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestConsumerInitRunInspectReplay(t *testing.T) {
	for _, candidate := range []string{"a", "b"} {
		t.Run(candidate, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "retained.db")
			for _, command := range []string{"init", "run", "inspect", "replay", "recover"} {
				var out bytes.Buffer
				if err := execute(context.Background(), []string{command, "-db", path, "-candidate", candidate, "-id", "2", "-value", "1", "-i", "3", "-r", "3", "-depth", "3"}, &out); err != nil {
					t.Fatal(command, err)
				}
				if !json.Valid(out.Bytes()) {
					t.Fatalf("%s returned invalid JSON: %s", command, out.String())
				}
			}
			s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Roots) != 1 || s.Roots[0].LocalStatus != "succeeded" {
				t.Fatal(s.Roots)
			}
		})
	}
}

func TestConsumerWrongRequestRemainsOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retained.db")
	for _, command := range []string{"init", "run", "recover", "replay"} {
		var out bytes.Buffer
		if err := execute(t.Context(), []string{command, "-db", path, "-wrong-request"}, &out); err != nil {
			t.Fatal(command, err)
		}
	}
	s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Roots) != 1 || s.Roots[0].LocalStatus != "open" || s.Roots[0].Outcome != nil {
		t.Fatal("wrong request falsely completed", s.Roots)
	}
}
