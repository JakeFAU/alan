// Package schema defines the closed, flat typed records shared by authoring and effects.
package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

// Value is a tagged scalar. Exactly the field belonging to Kind may be nonzero.
type Value struct {
	Kind   string
	Number int64
	Text   string
	Flag   bool
}

func Int(n int64) Value   { return Value{Kind: "int", Number: n} }
func Text(s string) Value { return Value{Kind: "string", Text: s} }
func Bool(b bool) Value   { return Value{Kind: "bool", Flag: b} }
func (v Value) Valid() bool {
	switch v.Kind {
	case "int":
		return v.Text == "" && !v.Flag
	case "string":
		return v.Number == 0 && !v.Flag && utf8.ValidString(v.Text)
	case "bool":
		return v.Number == 0 && v.Text == ""
	}
	return false
}

// Field declares a required scalar. Strings require a finite UTF-8 byte bound.
// Enum, when nonempty, restricts strings to those exact values.
type Field struct {
	Kind     string
	MaxBytes int
	Enum     []string
}
type Record map[string]Field

// Retained contains both the canonical schema and its SHA-256 identity.
type Retained struct {
	ID     string
	Record Record
}

const MaxFields = 128
const MaxStringBytes = 1 << 20

func (r Record) Check() error {
	if len(r) > MaxFields {
		return fmt.Errorf("schema field limit")
	}
	for name, f := range r {
		if err := CheckIdentifier(name); err != nil {
			return fmt.Errorf("field: %w", err)
		}
		switch f.Kind {
		case "string":
			if f.MaxBytes <= 0 || f.MaxBytes > MaxStringBytes {
				return fmt.Errorf("field %q: invalid string bound", name)
			}
			seen := map[string]bool{}
			for _, v := range f.Enum {
				if !utf8.ValidString(v) || len(v) > f.MaxBytes || seen[v] {
					return fmt.Errorf("field %q: invalid enum", name)
				}
				seen[v] = true
			}
		case "int", "bool":
			if f.MaxBytes != 0 || len(f.Enum) != 0 {
				return fmt.Errorf("field %q: nonstring constraints", name)
			}
		default:
			return fmt.Errorf("field %q: unsupported kind %q", name, f.Kind)
		}
	}
	return nil
}
func (f Field) Validate(v Value) error {
	if !v.Valid() || f.Kind != v.Kind {
		return fmt.Errorf("expected %s value", f.Kind)
	}
	if v.Kind == "string" && (len(v.Text) > f.MaxBytes || len(f.Enum) > 0 && !slices.Contains(f.Enum, v.Text)) {
		return fmt.Errorf("string outside bound or enum")
	}
	return nil
}
func (r Record) Validate(values map[string]Value) error {
	if err := r.Check(); err != nil {
		return err
	}
	if len(values) != len(r) {
		return fmt.Errorf("closed record fields differ")
	}
	for name, f := range r {
		v, ok := values[name]
		if !ok {
			return fmt.Errorf("missing field %q", name)
		}
		if err := f.Validate(v); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
	}
	return nil
}

// Retain copies and canonicalizes a schema; map and enum insertion order is immaterial.
func (r Record) Retain() (Retained, error) {
	if err := r.Check(); err != nil {
		return Retained{}, err
	}
	copy := Record{}
	for k, f := range r {
		f.Enum = slices.Clone(f.Enum)
		if len(f.Enum) == 0 {
			f.Enum = nil
		}
		slices.Sort(f.Enum)
		copy[k] = f
	}
	raw, err := json.Marshal(copy)
	if err != nil {
		return Retained{}, err
	}
	h := sha256.Sum256(raw)
	return Retained{ID: hex.EncodeToString(h[:]), Record: copy}, nil
}
func (r Retained) Check() error {
	c, err := r.Record.Retain()
	if err != nil {
		return err
	}
	if c.ID != r.ID {
		return fmt.Errorf("schema identity mismatch")
	}
	return nil
}

const ResultValid = "valid"
const ResultDisposition = "disposition"
const MaxDispositionBytes = 128

// ResultRecord adds runtime-owned validation fields to a payload schema. Providers
// produce only the payload. Invalid results carry typed defaults for payload fields;
// consumers must branch on valid before reading them as successful output.
func ResultRecord(payload Record) (Record, error) {
	retained, err := payload.Retain()
	if err != nil {
		return nil, err
	}
	for _, name := range []string{ResultValid, ResultDisposition} {
		if _, ok := payload[name]; ok {
			return nil, fmt.Errorf("reserved result field %q", name)
		}
	}
	result := retained.Record
	result[ResultValid] = Field{Kind: "bool"}
	result[ResultDisposition] = Field{Kind: "string", MaxBytes: MaxDispositionBytes}
	return result, result.Check()
}

// ValidateResult checks the runtime envelope and either a valid payload or exact
// typed zero defaults. A failed result never gains valid payload semantics.
func ValidateResult(payload Record, values map[string]Value) error {
	result, err := ResultRecord(payload)
	if err != nil {
		return err
	}
	if len(result) != len(values) {
		return fmt.Errorf("result fields differ")
	}
	for _, name := range []string{ResultValid, ResultDisposition} {
		if err = result[name].Validate(values[name]); err != nil {
			return fmt.Errorf("result %s: %w", name, err)
		}
	}
	fields := map[string]Value{}
	for name, f := range payload {
		v, ok := values[name]
		if !ok {
			return fmt.Errorf("missing result %q", name)
		}
		if !values[ResultValid].Flag && v != (Value{Kind: f.Kind}) {
			return fmt.Errorf("invalid result requires typed default %q", name)
		}
		fields[name] = v
	}
	if values[ResultValid].Flag {
		return payload.Validate(fields)
	}
	return nil
}

// CheckIdentifier accepts nonempty UTF-8 names without normalizing them. This
// prevents encoding/json's invalid-byte replacement from merging distinct names.
func CheckIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("empty identifier")
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("invalid UTF-8 identifier %q", name)
	}
	return nil
}
