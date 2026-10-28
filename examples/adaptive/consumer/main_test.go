package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestCLIAdaptiveRequest(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "nested"}[nested], func(t *testing.T) {
			db := filepath.Join(t.TempDir(), "cli.sqlite")
			input := filepath.Join(t.TempDir(), "request.json")
			if e := os.WriteFile(input, []byte(`{"seeds":[{"work_id":1,"value":0},{"work_id":2,"value":1},{"work_id":3,"value":2}],"max_rounds":3,"max_work":8}`), 0600); e != nil {
				t.Fatal(e)
			}
			var out bytes.Buffer
			args := []string{"init", "--db", db, "--capacity", "1"}
			if nested {
				args = append(args, "--nested")
			}
			if e := runCLI(t.Context(), args, &out); e != nil {
				t.Fatal(e)
			}
			for _, a := range [][]string{{"submit", "--db", db, "--id", "work", "--input", input}, {"run", "--db", db}, {"inspect", "--db", db}, {"replay", "--db", db}} {
				out.Reset()
				if e := runCLI(t.Context(), a, &out); e != nil {
					t.Fatal(a, e)
				}
			}
			before, e := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			out.Reset()
			if e = runCLI(t.Context(), []string{"recover", "--db", db}, &out); e != nil {
				t.Fatal(e)
			}
			after, e := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			a, _ := json.Marshal(before.Tables)
			b, _ := json.Marshal(after.Tables)
			if !bytes.Equal(a, b) {
				t.Fatal("recover changed completed retained state")
			}
		})
	}
	for _, raw := range []string{`{"seeds":[],"max_rounds":3}`, `{"seeds":[],"max_rounds":3,"max_work":8,"max_work":1}`, `{"Seeds":[],"max_rounds":3,"max_work":8}`, `{"seeds":[{"work_id":1}],"max_rounds":3,"max_work":8}`, `{"seeds":null,"max_rounds":3,"max_work":8}`} {
		input := filepath.Join(t.TempDir(), "bad.json")
		os.WriteFile(input, []byte(raw), 0600)
		if _, e := readRequest(input); e == nil {
			t.Fatal("invalid complete request accepted", raw)
		}
	}
}
