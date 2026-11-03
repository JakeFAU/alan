package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/internal/fixture"
)

// A missing demo grant, wrong receipt correlation, or blind non-idempotent
// repeat must fail this end-to-end test of the ordinary CLI fixture.
func TestCLIExternalDemo(t *testing.T) {
	for _, profile := range []string{"idempotent", "non-idempotent"} {
		for _, missing := range []bool{false, true} {
			t.Run(profile+"/missing="+map[bool]string{true: "true", false: "false"}[missing], func(t *testing.T) {
				dir := t.TempDir()
				resourcePath := filepath.Join(dir, "resource.sqlite")
				resource, err := fixture.Open(t.Context(), resourcePath, "cli-demo")
				if err != nil {
					t.Fatal(err)
				}
				defer resource.Close()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req fixture.Request
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					receipt, err := resource.Apply(r.Context(), req)
					if err != nil {
						t.Error(err)
						return
					}
					if missing {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					if err := json.NewEncoder(w).Encode(receipt); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				db := filepath.Join(dir, "executor.sqlite")
				call := func(args ...string) []byte {
					t.Helper()
					var out bytes.Buffer
					if err := runCLI(t.Context(), args, &out); err != nil {
						t.Fatal(err)
					}
					return out.Bytes()
				}
				call("init", "--db", db, "--demo", "external", "--endpoint", server.URL, "--profile", profile)
				for range 2 {
					var report executor.RunReport
					if err := json.Unmarshal(call("run", "--db", db), &report); err != nil {
						t.Fatal(err)
					}
					if !report.Quiescent {
						t.Fatal(report)
					}
					if missing && (len(report.UnknownExternal) != 1 || len(report.OpenRuns) != 1 || report.OpenRuns[0].Subject != "client") {
						t.Fatalf("uncertainty/peer progress %+v", report)
					}
					if !missing && (len(report.OpenRuns) != 0 || len(report.UnresolvedIntents) != 0) {
						t.Fatalf("receipt not consumed %+v", report)
					}
				}
				before := call("inspect", "--db", db)
				endpointBefore, err := fixture.Inspect(t.Context(), resourcePath)
				if err != nil {
					t.Fatal(err)
				}
				if len(endpointBefore.Applications) != 1 || endpointBefore.Total != 5 {
					t.Fatalf("effect %+v", endpointBefore)
				}
				expectedContacts := int64(1)
				if missing && profile == "idempotent" {
					expectedContacts = 2
				}
				if endpointBefore.Contacts != expectedContacts {
					t.Fatalf("contacts %+v", endpointBefore)
				}
				call("replay", "--db", db)
				if !bytes.Equal(before, call("inspect", "--db", db)) {
					t.Fatal("replay mutated executor")
				}
				endpointAfter, err := fixture.Inspect(t.Context(), resourcePath)
				if err != nil {
					t.Fatal(err)
				}
				if endpointAfter.Contacts != endpointBefore.Contacts {
					t.Fatal("replay contacted endpoint")
				}
			})
		}
	}
}
func TestCLIExternalDemoRejectsInvalidConfiguration(t *testing.T) {
	for _, args := range [][]string{
		{"--profile", "idempotent"},
		{"--endpoint", "http://example.com/apply", "--profile", "idempotent"},
		{"--endpoint", "http://127.0.0.1:9999/apply", "--profile", "unknown"},
	} {
		db := filepath.Join(t.TempDir(), "invalid.sqlite")
		command := append([]string{"init", "--db", db, "--demo", "external"}, args...)
		var out bytes.Buffer
		if err := runCLI(t.Context(), command, &out); err == nil {
			t.Fatalf("invalid config accepted %v", args)
		}
	}
}
