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

// Rebinding must preserve policy, guards, original templates, ownership and
// entitlement. C_se's frozen artifact IDs also detect changes to that parent.
func TestReasonOriginalBindings(t *testing.T) {
	var common []byte
	for _, tc := range []struct {
		name, sFirstID string
		build          func() (author.Template, error)
	}{
		{"a", "34e225b0678bcb069c5a9a75a95a8caaff443a559bfb00211d8070ce4d05631a", a.New},
		{"b", "30614a673022dbdb3badc2313d3817bd05316c5180cff7b358d51e65fbaff098", b.New},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			cfg := substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 2, MaxInFlight: 1}}
			old, err := consumingpolicy.NewSFirst(c, cfg, consumingpolicy.Options{})
			if err != nil || old.Parent.ID() != tc.sFirstID {
				t.Fatal("C_se artifact changed", old.Parent.ID(), err)
			}
			d, err := consumingpolicy.NewReason(c, cfg, consumingpolicy.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(d.Candidate.Bytes(), c.Bytes()) || !bytes.Equal(d.Successor.Bytes(), old.Successor.Bytes()) || d.Parent.ID() == old.Parent.ID() {
				t.Fatal("original components or distinct context identity")
			}
			if d.FixedRuns != 1 || len(d.Provision.Runs) != 1 || d.Provision.Runs[0].ID.Subject != "root" || len(d.Provision.Composition.Roots) != 1 || d.Provision.Composition.Roots[0].Limits != cfg.Limits {
				t.Fatal("owning parent or entitlement changed")
			}
			var artifact map[string]any
			if err := json.Unmarshal(d.Parent.Bytes(), &artifact); err != nil {
				t.Fatal(err)
			}
			code := artifact["Code"].([]any)
			for i, want := range []string{"emit", "emit", "receive", "receive"} {
				if code[i].(map[string]any)["Op"] != want {
					t.Fatalf("both commitments and joins must precede policy: pc%d", i)
				}
			}
			if code[0].(map[string]any)["Dst"] != "candidate_operation" || code[1].(map[string]any)["Dst"] != "s_operation" || code[2].(map[string]any)["Match"].(map[string]any)["Operation"].(map[string]any)["Ref"] != "s_operation" || code[3].(map[string]any)["Match"].(map[string]any)["Operation"].(map[string]any)["Ref"] != "candidate_operation" {
				t.Fatal("lost original handle or changed required order")
			}
			for _, raw := range code {
				in := raw.(map[string]any)
				if in["Op"] == "branch" && (in["A"].(map[string]any)["Ref"] == "request_mode" || in["B"].(map[string]any)["Ref"] == "request_mode") {
					t.Fatal("request mode used as actual classification")
				}
				if in["Op"] == "emit" && in["Effect"].(map[string]any)["Kind"] != "invocation" {
					t.Fatal("ordinary report/effect introduced")
				}
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
				t.Fatal("A/B policy/guards differ beyond candidate binding")
			}
		})
	}
}

func TestReasonRejectsInvalidConstruction(t *testing.T) {
	c, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		id   int64
		s    string
	}{
		{"id", 0, "answer"}, {"s-mode", 1, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := consumingpolicy.NewReason(c, substitution.Config{Request: substitution.Request{ID: tc.id, Mode: "answer"}}, consumingpolicy.Options{SMode: tc.s}); err == nil {
				t.Fatal("invalid construction accepted")
			}
		})
	}
}
