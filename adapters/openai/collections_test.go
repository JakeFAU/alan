package openai_test

import (
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func TestPrepareRejectsUnsupportedCollections(t *testing.T) {
	a, err := openai.New(openai.Config{APIKey: "controlled-secret", Model: "controlled-model", Endpoint: "http://127.0.0.1:1", MaxOutputTokens: 10, MaxResponseBytes: 1000})
	if err != nil {
		t.Fatal(err)
	}
	list := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 512, Element: schema.Record{"x": {Kind: "int"}}}
	for _, side := range []string{"input", "output"} {
		t.Run(side, func(t *testing.T) {
			in := schema.Record{"instructions": {Kind: "string", MaxBytes: 100}, "input": {Kind: "string", MaxBytes: 100}}
			out := schema.Record{"answer": {Kind: "string", MaxBytes: 100}}
			payload := map[string]schema.Value{"instructions": schema.Text("controlled"), "input": schema.Text("work")}
			if side == "input" {
				in["items"] = list
				payload["items"] = schema.List()
			} else {
				out["items"] = list
			}
			input, err := in.Retain()
			if err != nil {
				t.Fatal(err)
			}
			output, err := out.Retain()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = a.Prepare(effects.Request{Input: input, Output: output, Payload: payload}); err == nil {
				t.Fatal("flat provider silently accepted list schema")
			}
		})
	}
}
