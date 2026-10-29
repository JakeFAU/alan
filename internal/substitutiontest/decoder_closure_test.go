//go:build integration

package substitutiontest

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/runtime"
	"os"
	"path/filepath"
	"testing"
)

func originalRecord(t *testing.T, profile, test, name string) runtime.Inspection {
	t.Helper()
	path := filepath.Join("../../docs/foundational/phase1i/evidence/runtime", profile, hash(test)[:12], name+".json.gz")
	f, e := os.Open(path)
	must(t, e)
	defer f.Close()
	z, e := gzip.NewReader(f)
	must(t, e)
	defer z.Close()
	var d struct {
		Inspection runtime.Inspection `json:"inspection"`
	}
	must(t, json.NewDecoder(z).Decode(&d))
	return d.Inspection
}
func TestSubstitutionDecoderClosureFlags(t *testing.T) {
	for _, profile := range []string{"normal-v1", "race-v1"} {
		for _, name := range []string{"a", "b"} {
			for _, cut := range []string{"entry-commit", "entry-admitted", "helper-commit", "reply-commit", "published"} {
				if name == "a" && cut == "helper-commit" {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", profile, name, cut), func(t *testing.T) {
					test := "TestSubstitutionClosure/" + name + "/" + cut
					s := originalRecord(t, profile, test, "after")
					q := app.Request{ID: 1, Value: 0, Mode: "answer"}
					d, _ := provision(t, name, "bare", q, limits(3, 3, 3))
					n := normalize(s, d, q)
					retain(t, "original-closure", map[string]any{"inspection": s, "normalized": n})
					if len(n.Violations) > 0 {
						t.Fatal(n.Violations)
					}
					if !n.Context.Closed {
						t.Fatal("real accepted CancelRun decoded Closed=false")
					}
				})
			}
		}
	}
}
