package invocationjoin_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/invocationjoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

func TestNewOriginalTemplatesAndInvokeBeforeAwait(t *testing.T) {
	var contextBody []byte
	for _, tc := range []struct {
		name  string
		build func() (author.Template, error)
	}{{"a", a.New}, {"b", b.New}} {
		t.Run(tc.name, func(t *testing.T) {
			candidate, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			cfg := substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 2, MaxInFlight: 1}}
			originalCfg := cfg
			originalCfg.Shape = "sequential"
			original, err := substitution.New(candidate, originalCfg)
			if err != nil {
				t.Fatal(err)
			}
			d, err := invocationjoin.New(candidate, cfg, invocationjoin.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(d.Candidate.Bytes(), candidate.Bytes()) || !bytes.Equal(d.Successor.Bytes(), original.Peer.Bytes()) {
				t.Fatal("original template changed")
			}
			if d.FixedRuns != 1 || len(d.Provision.Runs) != 1 || d.Provision.Runs[0].ID.Subject != "root" {
				t.Fatal("expected one fixed parent")
			}
			if len(d.Provision.Composition.Roots) != 1 || d.Provision.Composition.Roots[0].Limits != cfg.Limits {
				t.Fatal("allowance changed")
			}
			var p struct {
				Code []struct {
					Op     string
					Effect *struct{ Kind string }
					Dst    string
					A      struct{ Ref string }
				}
			}
			if err := json.Unmarshal(d.Parent.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if len(p.Code) < 4 || p.Code[0].Op != "emit" || p.Code[1].Op != "emit" || p.Code[0].Effect == nil || p.Code[1].Effect == nil || p.Code[0].Effect.Kind != "invocation" || p.Code[1].Effect.Kind != "invocation" || p.Code[2].Op != "receive" || p.Code[3].Op != "receive" {
				t.Fatalf("invoke/invoke/await/await prefix: %+v", p.Code)
			}
			var normalized map[string]any
			if err := json.Unmarshal(d.Parent.Bytes(), &normalized); err != nil {
				t.Fatal(err)
			}
			normalized["Code"].([]any)[0].(map[string]any)["Effect"].(map[string]any)["Target"] = "candidate-binding"
			normalized["Code"].([]any)[2].(map[string]any)["Match"].(map[string]any)["Source"] = "candidate-binding"
			normalized["Author"].(map[string]any)["Invocations"].(map[string]any)["candidate"].(map[string]any)["TemplateID"] = "candidate-binding"
			raw, err := json.Marshal(normalized)
			if err != nil {
				t.Fatal(err)
			}
			if contextBody == nil {
				contextBody = raw
			} else if !bytes.Equal(contextBody, raw) {
				t.Fatal("parent differs beyond the candidate binding")
			}
		})
	}
}
