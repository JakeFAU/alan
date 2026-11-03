package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// The ledger retains both the raw reply and the usage details. HTML-escaped
// strings can expand sixfold in each copy when the result is marshaled.
func TestMaximumResponseFitsRetainedResult(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request struct{ Input []struct{ Content string } }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Input) != 1 {
			t.Error("invalid request", err)
			return
		}
		var in struct{ Role, Snippet string }
		if err := json.Unmarshal([]byte(request.Input[0].Content), &in); err != nil {
			t.Error(err)
			return
		}
		_, raw := fixtureResponse(in.Role, "supports", in.Snippet)
		prefix := `"output_tokens_details":{`
		expanded := strings.Replace(raw, prefix, prefix+`"padding":"","tail":1,`, 1)
		if expanded == raw {
			t.Error("missing usage details in fixture")
			return
		}
		fill := strings.Repeat("<", maxProviderResponseBytes-len(expanded))
		raw = strings.Replace(expanded, `"padding":""`, `"padding":"`+fill+`"`, 1)
		if len(raw) != maxProviderResponseBytes {
			t.Error("fixture did not reach the configured limit")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, raw)
	}))
	defer server.Close()
	dir := t.TempDir()
	input, err := os.ReadFile("testdata/case.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec investigationSpec
	if err := json.Unmarshal(input, &spec); err != nil {
		t.Fatal(err)
	}
	spec.Provider.Endpoint = server.URL + "/v1/responses"
	spec.Provider.MaxResponseBytes = maxProviderResponseBytes
	for _, doc := range spec.Documents {
		b, err := os.ReadFile(filepath.Join("testdata", doc.Path))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, doc.Path), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	file, db := filepath.Join(dir, "case.json"), filepath.Join(dir, "case.sqlite")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", db, file}, {"run", db}, {"resume", db}} {
		if err := run(t.Context(), args, io.Discard); err != nil {
			t.Fatalf("%s: %v", args[0], err)
		}
	}
	report, err := readReport(t.Context(), db, runtime.DefaultLimits())
	if err != nil || report.Status != "completed" || len(report.Findings) != 2 || calls.Load() != 2 {
		t.Fatalf("status %s, findings %d, contacts %d, error %v", report.Status, len(report.Findings), calls.Load(), err)
	}
	for _, worker := range report.Workers {
		u := worker.Model.Usage
		if u == nil || u.InputTokens == nil || u.OutputTokens == nil || u.TotalTokens == nil || *u.InputTokens != 17 || *u.OutputTokens != 3 || *u.TotalTokens != 20 {
			t.Fatal("lost provider-reported usage at the maximum response size")
		}
	}
}
