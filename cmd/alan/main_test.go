package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/JakeFAU/alan/internal/executor"
	"os"
	"path/filepath"
	"testing"
)

func cliProvision(t *testing.T) executor.Provision {
	t.Helper()
	p := executor.Program{Semantics: executor.SemanticVersion, Code: []executor.Instruction{{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: executor.Lit(executor.Text("local")), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("work")), Operation: executor.Lit(executor.Text(""))}, Bind: map[string]string{"value": "value"}}, {Op: "end", Outcome: "succeeded"}}}
	b, id, err := executor.EncodeProgram(p)
	if err != nil {
		t.Fatal(err)
	}
	return executor.Provision{Programs: []executor.Artifact{{ID: id, Semantics: executor.SemanticVersion, Bytes: b}}, Subjects: []executor.Subject{{ID: "worker", Generation: 1, Owner: "owner", Revision: 1}}, Runs: []executor.InitialRun{{ID: executor.RunID{Subject: "worker", Generation: 1}, ProgramID: id, Projection: executor.Projection{Status: "open", Data: map[string]executor.Value{}}}}, Permissions: []executor.Permission{{Kind: "input", Source: "local", Target: "worker", Operation: "input", Obligation: "work"}}}
}
func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestCLIInitRunInputInspectReplay(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "store.sqlite")
	provision := filepath.Join(dir, "provision.json")
	request := filepath.Join(dir, "input.json")
	writeJSON(t, provision, cliProvision(t))
	execute := func(args ...string) []byte {
		t.Helper()
		var out bytes.Buffer
		if err := runCLI(t.Context(), args, &out); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	execute("init", "--db", db, "--provision", provision)
	before := execute("run", "--db", db)
	var report executor.RunReport
	if err := json.Unmarshal(before, &report); err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || len(report.OpenRuns) != 1 || report.Steps != 0 {
		t.Fatalf("waiting mislabeled %+v", report)
	}
	writeJSON(t, request, executor.Input{TransportID: "input1", LogicalID: "logical1", Target: "worker", Obligation: "work", Payload: map[string]executor.Value{"value": executor.Int(73)}})
	execute("input", "--db", db, "--request", request)
	result := execute("run", "--db", db)
	if err := json.Unmarshal(result, &report); err != nil {
		t.Fatal(err)
	}
	if report.Steps != 2 || len(report.OpenRuns) != 0 {
		t.Fatalf("ordinary progress %+v", report)
	}
	inspection := execute("inspect", "--db", db)
	execute("replay", "--db", db)
	if after := execute("inspect", "--db", db); !bytes.Equal(inspection, after) {
		t.Fatal("CLI replay changed store")
	}
}
func TestCLIRejectsForgedOriginAndUsage(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "store.sqlite")
	provision := filepath.Join(dir, "p.json")
	request := filepath.Join(dir, "forged.json")
	writeJSON(t, provision, cliProvision(t))
	var out bytes.Buffer
	if err := runCLI(t.Context(), []string{"init", "--db", db, "--provision", provision}, &out); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(request, []byte(`{"TransportID":"x","LogicalID":"x","Target":"worker","Kind":"terminal","Provenance":"runtime-terminal","Payload":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCLI(t.Context(), []string{"input", "--db", db, "--request", request}, &out); err == nil {
		t.Fatal("forged runtime provenance accepted")
	}
	for _, args := range [][]string{nil, {"unknown"}, {"run"}, {"inspect", "--db", db, "extra"}} {
		if err := runCLI(context.Background(), args, &out); err == nil {
			t.Fatalf("bad usage accepted: %v", args)
		}
	}
}

func TestCLINestedDemoExecutesOrdinaryPrograms(t *testing.T) {
	db := filepath.Join(t.TempDir(), "nested.sqlite")
	var out bytes.Buffer
	if err := runCLI(t.Context(), []string{"init", "--db", db, "--demo", "nested"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runCLI(t.Context(), []string{"run", "--db", db}, &out); err != nil {
		t.Fatal(err)
	}
	var report executor.RunReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Quiescent || len(report.OpenRuns) != 0 || len(report.UnresolvedIntents) != 0 {
		t.Fatalf("demo incomplete %+v", report)
	}
}
