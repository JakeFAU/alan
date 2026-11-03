// Package schema defines closed typed records shared by authoring and effects.
package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Value is a tagged scalar or bounded record list. Use Equal for structural equality.
// Scalar values remain comparable; list pointer identity has no semantic meaning.
type Value struct {
	Kind   string
	Number int64
	Text   string
	Flag   bool
	List   *ListData `json:",omitempty"`
}

func Int(n int64) Value   { return Value{Kind: "int", Number: n} }
func Text(s string) Value { return Value{Kind: "string", Text: s} }
func Bool(b bool) Value   { return Value{Kind: "bool", Flag: b} }
func (v Value) Valid() bool {
	if v.List != nil {
		remaining := MaxCollectionBytes + listValueOverhead
		if boundedValueSize(v, &remaining, 0) != nil {
			return false
		}
	}
	return validValue(v, 0)
}

// Field declares a required scalar or bounded list of closed records. Strings
// require a finite UTF-8 byte bound; Enum restricts strings to exact values.
// Lists require Element, MaxItems and a total canonical array-byte MaxBytes.
// UniqueKey optionally requires distinct string identities within each list.
type Field struct {
	Kind      string
	MaxBytes  int
	Enum      []string
	Element   Record `json:",omitempty"`
	MaxItems  int    `json:",omitempty"`
	UniqueKey string `json:",omitempty"`
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
	remaining := MaxSchemaFields
	if err := boundedSchema(r, &remaining, 0); err != nil {
		return err
	}
	return checkRecord(r, 0)
}
func (f Field) Validate(v Value) error {
	if err := (Record{"value": f}).Check(); err != nil {
		return err
	}
	if !v.Valid() {
		return fmt.Errorf("invalid or oversized value")
	}
	return validateField(f, v, 0)
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
	copy := cloneRecord(r)
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
		if !values[ResultValid].Flag && !v.Equal(Default(f)) {
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
