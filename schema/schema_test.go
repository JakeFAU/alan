package schema_test

import (
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func TestClosedBoundedRecord(t *testing.T) {
	s := schema.Record{"choice": {Kind: "string", MaxBytes: 4, Enum: []string{"yes", "no"}}, "n": {Kind: "int"}, "ok": {Kind: "bool"}}
	good := map[string]schema.Value{"choice": schema.Text("yes"), "n": schema.Int(2), "ok": schema.Bool(true)}
	if err := s.Validate(good); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		values map[string]schema.Value
	}{
		{"missing", map[string]schema.Value{}}, {"extra", map[string]schema.Value{"choice": schema.Text("yes"), "n": schema.Int(2), "ok": schema.Bool(true), "extra": schema.Int(0)}}, {"enum", map[string]schema.Value{"choice": schema.Text("bad"), "n": schema.Int(2), "ok": schema.Bool(true)}}, {"wrong type", map[string]schema.Value{"choice": schema.Int(1), "n": schema.Int(2), "ok": schema.Bool(true)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if s.Validate(tt.values) == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
}

func TestResultValidationSeparatesEnvelope(t *testing.T) {
	payload := schema.Record{"decision": {Kind: "string", MaxBytes: 20, Enum: []string{"lookup", "answer"}}}
	invalid := map[string]schema.Value{"decision": schema.Text(""), "valid": schema.Bool(false), "disposition": schema.Text("schema_invalid")}
	if err := schema.ValidateResult(payload, invalid); err != nil {
		t.Fatal(err)
	}
	invalid["valid"] = schema.Bool(true)
	if schema.ValidateResult(payload, invalid) == nil {
		t.Fatal("invalid successful payload accepted")
	}
	invalid["decision"] = schema.Text("lookup")
	if err := schema.ValidateResult(payload, invalid); err != nil {
		t.Fatal(err)
	}
	invalid["valid"] = schema.Bool(false)
	if schema.ValidateResult(payload, invalid) == nil {
		t.Fatal("invalid result carries nondefault payload")
	}
	if _, err := schema.ResultRecord(schema.Record{"valid": {Kind: "bool"}}); err == nil {
		t.Fatal("reserved validation field allowed")
	}
}

func TestRetainCanonicalizesEquivalentSchemas(t *testing.T) {
	a, err := (schema.Record{"a": {Kind: "string", MaxBytes: 10, Enum: []string{"yes", "no"}}, "b": {Kind: "bool"}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	b, err := (schema.Record{"b": {Kind: "bool", Enum: []string{}}, "a": {Kind: "string", MaxBytes: 10, Enum: []string{"no", "yes"}}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatal("equivalent schema insertion order/empty constraints changed identity")
	}
}
