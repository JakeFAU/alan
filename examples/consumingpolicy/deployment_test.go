package consumingpolicy_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/consumingpolicy"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

// Reversing awaits, replacing an original template, adding a fixed peer, or
// changing entitlement must fail this construction-boundary test.
func TestSFirstOriginalBindings(t *testing.T) {
	var common []byte
	for _, tc := range []struct {
		name  string
		build func() (author.Template, error)
	}{{"a", a.New}, {"b", b.New}} {
		t.Run(tc.name, func(t *testing.T) {
			candidate, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			cfg := substitution.Config{Shape: "sequential", Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 2, MaxInFlight: 1}}
			original, err := substitution.New(candidate, cfg)
			if err != nil {
				t.Fatal(err)
			}
			d, err := consumingpolicy.NewSFirst(candidate, cfg, consumingpolicy.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(candidate.Bytes(), d.Candidate.Bytes()) || !bytes.Equal(original.Peer.Bytes(), d.Successor.Bytes()) {
				t.Fatal("original template changed")
			}
			if d.FixedRuns != 1 || len(d.Provision.Runs) != 1 || d.Provision.Runs[0].ID.Subject != "root" {
				t.Fatal("one fixed owning parent required")
			}
			if len(d.Provision.Composition.Roots) != 1 || d.Provision.Composition.Roots[0].Limits != cfg.Limits {
				t.Fatal("allowance changed")
			}
			var artifact map[string]any
			if err := json.Unmarshal(d.Parent.Bytes(), &artifact); err != nil {
				t.Fatal(err)
			}
			code := artifact["Code"].([]any)
			for i, want := range []string{"emit", "emit", "receive", "receive"} {
				if code[i].(map[string]any)["Op"] != want {
					t.Fatalf("prefix instruction %d", i)
				}
			}
			if code[0].(map[string]any)["Dst"] != "candidate_operation" || code[1].(map[string]any)["Dst"] != "s_operation" {
				t.Fatal("original handles not retained")
			}
			// The match carries the actual handle register; no fabricated operation ID.
			if code[2].(map[string]any)["Match"].(map[string]any)["Operation"].(map[string]any)["Ref"] != "s_operation" || code[3].(map[string]any)["Match"].(map[string]any)["Operation"].(map[string]any)["Ref"] != "candidate_operation" {
				t.Fatal("consumption order is not S then candidate")
			}
			code[0].(map[string]any)["Effect"].(map[string]any)["Target"] = "candidate-binding"
			code[3].(map[string]any)["Match"].(map[string]any)["Source"] = "candidate-binding"
			artifact["Author"].(map[string]any)["Invocations"].(map[string]any)["candidate"].(map[string]any)["TemplateID"] = "candidate-binding"
			raw, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			if common == nil {
				common = raw
			} else if !bytes.Equal(common, raw) {
				t.Fatal("A/B differ beyond candidate binding")
			}
		})
	}
}

func TestSFirstRejectsInvalidConstruction(t *testing.T) {
	candidate, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		id   int64
		s    string
	}{{"id", 0, "answer"}, {"s-mode", 1, "unknown"}} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := consumingpolicy.NewSFirst(candidate, substitution.Config{Request: substitution.Request{ID: tc.id, Mode: "answer"}}, consumingpolicy.Options{SMode: tc.s}); err == nil {
				t.Fatal("invalid construction accepted")
			}
		})
	}
}
