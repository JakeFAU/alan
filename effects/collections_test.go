package effects

import (
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func TestResultCollectionDefaultsAndCopy(t *testing.T) {
	output := schema.Record{"items": {Kind: "list", MaxItems: 2, MaxBytes: 512, Element: schema.Record{"x": {Kind: "int"}}}}
	invalid, _ := Result(output, Candidate{Disposition: "schema_invalid"})
	if err := schema.ValidateResult(output, invalid); err != nil {
		t.Fatalf("invalid result lacks canonical list default: %v", err)
	}
	candidate := Candidate{Disposition: "validated", Payload: map[string]schema.Value{"items": schema.List(map[string]schema.Value{"x": schema.Int(1)})}}
	result, _ := Result(output, candidate)
	if err := candidate.Payload["items"].List.UnmarshalJSON([]byte(`[{"x":{"Kind":"int","Number":99}}]`)); err != nil {
		t.Fatal(err)
	}
	row, err := result["items"].At(0)
	if err != nil || row["x"] != schema.Int(1) {
		t.Fatal("normalized result aliases candidate")
	}
}
