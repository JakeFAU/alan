package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestCLIInitializesBeforeWorkAndRecoversRetainedArtifacts(t *testing.T) {
	db := filepath.Join(t.TempDir(), "work.sqlite")
	input := filepath.Join(t.TempDir(), "later.json")
	invoke := func(args ...string) []byte {
		t.Helper()
		var out bytes.Buffer
		if err := runCLI(t.Context(), args, &out); err != nil {
			t.Fatal(args, err)
		}
		return out.Bytes()
	}
	invoke("init", "--db", db, "--nested")
	before, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Tables["runs"]) != 1 || len(before.Tables["composition_invocations"]) != 0 {
		t.Fatal("work during init")
	}
	if err = os.WriteFile(input, []byte(`[{"item_id":"item-a","number":2},{"item_id":"item-b","number":-3},{"item_id":"item-c","number":2}]`), 0600); err != nil {
		t.Fatal(err)
	}
	invoke("submit", "--db", db, "--input", input, "--id", "one-work")
	invoke("recover", "--db", db)
	after, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Roots) != 1 || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "succeeded" || after.Roots[0].Charges.Invocations != 5 {
		t.Fatal(after.Roots)
	}
	if len(after.Tables["programs"]) != len(before.Tables["programs"]) {
		t.Fatal("recompiled topology")
	}
	var view struct{ Result map[string]schema.Value }
	if err = json.Unmarshal(invoke("inspect", "--db", db), &view); err != nil {
		t.Fatal(err)
	}
	rows, err := view.Result["results"].Records()
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	invoke("replay", "--db", db)
}

func TestCLIRejectsMalformedInputAndUnsafeTransport(t *testing.T) {
	cases := [][]string{nil, {"init"}, {"unknown"}, {"init", "--db", "x", "--input", "x"}, {"run", "--db", "x", "--controlled-endpoint", "https://api.openai.com/v1/responses"}}
	for _, args := range cases {
		var out bytes.Buffer
		if err := runCLI(t.Context(), args, &out); err == nil {
			t.Fatal("accepted", args)
		}
	}
	db := filepath.Join(t.TempDir(), "work.sqlite")
	var out bytes.Buffer
	if err := runCLI(t.Context(), []string{"init", "--db", db}, &out); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`[{"item_id":"a","number":2,"extra":true}]`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCLI(t.Context(), []string{"submit", "--db", db, "--input", path, "--id", "bad"}, &out); err == nil {
		t.Fatal("unknown work field")
	}
}

func TestCLINullIsNotAnEmptyTypedWorkList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "null.json")
	if err := os.WriteFile(path, []byte("null"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readItems(path); err == nil {
		t.Fatal("null fabricated an empty accepted list")
	}
}

func TestCLIRejectsDuplicateItemFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.json")
	if err := os.WriteFile(path, []byte(`[{"item_id":"first","item_id":"second","number":2}]`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readItems(path); err == nil {
		t.Fatal("duplicate JSON identity silently replaced")
	}
}

func TestCLIWorkFieldsRequireCanonicalNames(t *testing.T) {
	cases := []struct {
		name, input string
		want        map[string]schema.Value
	}{
		{name: "canonical-number", input: `[{"item_id":"item-a","number":2}]`, want: map[string]schema.Value{"item_id": schema.Text("item-a"), "number": schema.Int(2)}},
		{name: "canonical-question", input: `[{"question":"WAL?","item_id":"question-a"}]`, want: map[string]schema.Value{"item_id": schema.Text("question-a"), "question": schema.Text("WAL?")}},
		{name: "case-only-identity", input: `[{"ITEM_ID":"case-only","number":2}]`},
		{name: "case-only-number", input: `[{"item_id":"item-a","NUMBER":2}]`},
		{name: "case-only-question", input: `[{"item_id":"question-a","QUESTION":"WAL?"}]`},
		{name: "case-only-all", input: `[{"ITEM_ID":"case-only","NUMBER":2}]`},
		{name: "identity-alias", input: `[{"item_id":"original","ITEM_ID":"replacement","number":2}]`},
		{name: "identity-alias-first", input: `[{"ITEM_ID":"replacement","item_id":"original","number":2}]`},
		{name: "number-alias", input: `[{"item_id":"item-a","number":2,"NUMBER":100}]`},
		{name: "question-alias", input: `[{"item_id":"question-a","question":"WAL?","QUESTION":"replacement"}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "work.json")
			if err := os.WriteFile(path, []byte(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := readItems(path)
			if tc.want == nil {
				if err == nil {
					t.Fatal("noncanonical identity or payload accepted", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatal("canonical work rejected", err)
			}
			rows, err := got.Records()
			if err != nil || len(rows) != 1 || len(rows[0]) != len(tc.want) {
				t.Fatal("canonical work changed", rows, err)
			}
			for field, want := range tc.want {
				if !rows[0][field].Equal(want) {
					t.Fatal("canonical field changed", field, rows[0][field], want)
				}
			}
		})
	}
}
