package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvestigationSpecification(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha\nβeta\n"), 0600); err != nil {
		t.Fatal(err)
	}
	valid := `{"question":"What do the documents say?","queries":["alpha","βeta"],"documents":[{"id":"guide","path":"a.txt"}],"provider":{"model":"test-model","endpoint":"http://127.0.0.1:1234/v1/responses","max_output_tokens":64,"max_response_bytes":4096},"limits":{"max_effects":4,"max_in_flight":2,"max_total_output_tokens":128}}`
	cases := []struct {
		name, input string
		ok          bool
	}{
		{"valid", valid, true},
		{"no second query", strings.Replace(valid, `,"βeta"`, "", 1), false},
		{"extra query", strings.Replace(valid, `"βeta"]`, `"βeta","third"]`, 1), false},
		{"overspent allocation", strings.Replace(valid, `"max_total_output_tokens":128`, `"max_total_output_tokens":127`, 1), false},
		{"response too large for the retained investigator", strings.Replace(valid, "4096", "131073", 1), false},
		{"foreign endpoint", strings.Replace(valid, "http://127.0.0.1:1234", "https://example.org", 1), false},
		{"empty document list", strings.Replace(valid, `[{"id":"guide","path":"a.txt"}]`, `[]`, 1), false},
		{"unknown spending field", strings.Replace(valid, `"max_effects":4`, `"max_effects":4,"estimated_cost":100`, 1), false},
		{"trailing value", valid + " {}", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, "spec.json")
			if err := os.WriteFile(path, []byte(tt.input), 0600); err != nil {
				t.Fatal(err)
			}
			spec, assets, err := loadSpec(path)
			if (err == nil) != tt.ok {
				t.Fatalf("loadSpec error %v", err)
			}
			if tt.ok && (spec.Question != "What do the documents say?" || len(assets) != 1 || assets[0].Content != "alpha\nβeta\n" || assets[0].Source != "a.txt") {
				t.Fatalf("bad retained input %v", assets)
			}
		})
	}
}

// encoding/json accepts repeated keys and case-insensitive aliases by default.
// An explicit spending authorization must have exactly one interpretation.
func TestAmbiguousSpecificationCannotCreateAStore(t *testing.T) {
	base, err := os.ReadFile("testdata/case.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"repeated limit", `"max_effects": 4`, `"max_effects": 1, "max_effects": 4`},
		{"case alias", `"max_effects": 4`, `"max_effects": 1, "MAX_EFFECTS": 4`},
		{"alias only", `"limits":`, `"LiMiTs":`},
		{"escaped duplicate", `"max_effects": 4`, `"max_ef\u0066ects": 1, "max_effects": 4`},
		{"repeated container", `"provider": {`, `"provider": {}, "provider": {`},
		{"case folded Unicode", `"question":`, `"queſtion":`},
		{"document alias", `"id": "access"`, `"Id":"other","id": "access"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			spec := filepath.Join(dir, "case.json")
			data := strings.Replace(string(base), tc.old, tc.replacement, 1)
			if data == string(base) {
				t.Fatal("ineffective test substitution")
			}
			for _, f := range []string{"inspection.txt", "access.txt"} {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("local source"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(spec, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			db := filepath.Join(dir, "case.sqlite")
			if err := run(t.Context(), []string{"init", db, spec}, io.Discard); err == nil {
				t.Fatal("ambiguous configuration initialized")
			}
			if _, err := os.Stat(db); !os.IsNotExist(err) {
				t.Fatal("ambiguous authorization created a store", err)
			}
		})
	}
}
