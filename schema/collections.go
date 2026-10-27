package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

const MaxSchemaNesting = 8
const MaxCollectionItems = 10000
const MaxCollectionBytes = 1 << 20

// ListData owns its records. Its JSON representation is always an array, including [].
// Construction and access go through List, Records and At to prevent aliases.
type ListData struct{ records []map[string]Value }

func (l ListData) MarshalJSON() ([]byte, error) {
	if !(Value{Kind: "list", List: &l}).Valid() {
		return nil, fmt.Errorf("invalid or oversized list")
	}
	if l.records == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(l.records)
}
func (l *ListData) UnmarshalJSON(raw []byte) error {
	if len(raw) > MaxCollectionBytes+listValueOverhead {
		return fmt.Errorf("list aggregate byte bound")
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("list must be an array")
	}
	if err := strictJSON(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var rows []map[string]Value
	if err := d.Decode(&rows); err != nil {
		return err
	}
	candidate := Value{Kind: "list", List: &ListData{records: rows}}
	if !candidate.Valid() {
		return fmt.Errorf("invalid or oversized list")
	}
	l.records = rows
	return nil
}

// List copies a finite native list. Invalid elements remain invalid and are rejected by Validate.
func List(records ...map[string]Value) Value {
	candidate := Value{Kind: "list", List: &ListData{records: records}}
	if !candidate.Valid() {
		return Value{Kind: "list"}
	}
	return Value{Kind: "list", List: &ListData{records: cloneRows(records, 0)}}
}
func (v Value) Records() ([]map[string]Value, error) {
	if v.Kind != "list" || !v.Valid() {
		return nil, fmt.Errorf("expected list")
	}
	return cloneRows(v.List.records, 0), nil
}
func (v Value) Len() (int, error) {
	if v.Kind != "list" || !v.Valid() {
		return 0, fmt.Errorf("expected list")
	}
	return len(v.List.records), nil
}
func (v Value) At(index int) (map[string]Value, error) {
	n, err := v.Len()
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= n {
		return nil, fmt.Errorf("list index outside snapshot")
	}
	return CloneValues(v.List.records[index]), nil
}

// Clone returns independent list storage. Invalid values stay invalid.
func (v Value) Clone() Value {
	if v.List != nil && !v.Valid() {
		return Value{Kind: "list"}
	}
	return cloneValue(v, 0)
}
func CloneValues(values map[string]Value) map[string]Value {
	if values == nil {
		return nil
	}
	out := make(map[string]Value, len(values))
	for k, v := range values {
		out[k] = v.Clone()
	}
	return out
}
func cloneRows(rows []map[string]Value, depth int) []map[string]Value {
	out := make([]map[string]Value, len(rows))
	for i, row := range rows {
		if row == nil {
			continue
		}
		out[i] = map[string]Value{}
		for k, v := range row {
			out[i][k] = cloneValue(v, depth)
		}
	}
	return out
}
func cloneValue(v Value, depth int) Value {
	if v.List != nil {
		if depth > MaxSchemaNesting {
			return Value{Kind: "invalid"}
		}
		v.List = &ListData{records: cloneRows(v.List.records, depth+1)}
	}
	return v
}
func (v Value) Equal(other Value) bool {
	if !v.Valid() || !other.Valid() {
		return false
	}
	return equalValue(v, other)
}
func equalValue(a, b Value) bool {
	if a.Kind != b.Kind || a.Number != b.Number || a.Text != b.Text || a.Flag != b.Flag {
		return false
	}
	if a.Kind != "list" {
		return true
	}
	if len(a.List.records) != len(b.List.records) {
		return false
	}
	for i, row := range a.List.records {
		other := b.List.records[i]
		if len(row) != len(other) {
			return false
		}
		for k, v := range row {
			w, ok := other[k]
			if !ok || !equalValue(v, w) {
				return false
			}
		}
	}
	return true
}
func Default(f Field) Value {
	if f.Kind == "list" {
		return List()
	}
	return Value{Kind: f.Kind}
}
func validValue(v Value, depth int) bool {
	if depth > MaxSchemaNesting {
		return false
	}
	if v.Kind != "list" && v.List != nil {
		return false
	}
	switch v.Kind {
	case "int":
		return v.Text == "" && !v.Flag
	case "string":
		return v.Number == 0 && !v.Flag && utf8.ValidString(v.Text)
	case "bool":
		return v.Number == 0 && v.Text == ""
	case "list":
		if v.Number != 0 || v.Text != "" || v.Flag || v.List == nil || len(v.List.records) > MaxCollectionItems {
			return false
		}
		for _, row := range v.List.records {
			if row == nil || len(row) > MaxFields {
				return false
			}
			for k, w := range row {
				if CheckIdentifier(k) != nil || !validValue(w, depth+1) {
					return false
				}
			}
		}
		return true
	}
	return false
}
func checkRecord(r Record, depth int) error {
	if depth > MaxSchemaNesting || len(r) > MaxFields {
		return fmt.Errorf("schema nesting or field limit")
	}
	for name, f := range r {
		if err := CheckIdentifier(name); err != nil {
			return fmt.Errorf("field: %w", err)
		}
		if err := checkField(f, depth); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
	}
	return nil
}
func checkField(f Field, depth int) error {
	if f.Kind != "list" && (f.Element != nil || f.MaxItems != 0 || f.UniqueKey != "") {
		return fmt.Errorf("nonlist constraints")
	}
	switch f.Kind {
	case "string":
		if f.MaxBytes <= 0 || f.MaxBytes > MaxStringBytes {
			return fmt.Errorf("invalid string bound")
		}
		seen := map[string]bool{}
		for _, v := range f.Enum {
			if !utf8.ValidString(v) || len(v) > f.MaxBytes || seen[v] {
				return fmt.Errorf("invalid enum")
			}
			seen[v] = true
		}
	case "int", "bool":
		if f.MaxBytes != 0 || len(f.Enum) != 0 {
			return fmt.Errorf("nonstring constraints")
		}
	case "list":
		if f.Element == nil || f.MaxItems <= 0 || f.MaxItems > MaxCollectionItems || f.MaxBytes < 2 || f.MaxBytes > MaxCollectionBytes || len(f.Enum) != 0 {
			return fmt.Errorf("invalid list bounds or element")
		}
		if err := checkRecord(f.Element, depth+1); err != nil {
			return err
		}
		if f.UniqueKey != "" {
			key, ok := f.Element[f.UniqueKey]
			if !ok || key.Kind != "string" {
				return fmt.Errorf("unique key requires string element field")
			}
		}
	default:
		return fmt.Errorf("unsupported kind %q", f.Kind)
	}
	return nil
}
func validateField(f Field, v Value, depth int) error {
	if !validValue(v, depth) || f.Kind != v.Kind {
		return fmt.Errorf("expected %s value", f.Kind)
	}
	if v.Kind == "string" && (len(v.Text) > f.MaxBytes || len(f.Enum) > 0 && !slices.Contains(f.Enum, v.Text)) {
		return fmt.Errorf("string outside bound or enum")
	}
	if v.Kind != "list" {
		return nil
	}
	if len(v.List.records) > f.MaxItems {
		return fmt.Errorf("list item bound")
	}
	// Count canonical bytes incrementally; never marshal an amplified whole list.
	remaining := f.MaxBytes - 2
	seen := map[string]bool{}
	for i, row := range v.List.records {
		if len(row) != len(f.Element) {
			return fmt.Errorf("closed record fields differ")
		}
		for k, field := range f.Element {
			w, ok := row[k]
			if !ok {
				return fmt.Errorf("missing field %q", k)
			}
			if err := validateField(field, w, depth+1); err != nil {
				return fmt.Errorf("element %d field %q: %w", i, k, err)
			}
		}
		if f.UniqueKey != "" {
			key := row[f.UniqueKey].Text
			if seen[key] {
				return fmt.Errorf("duplicate list key")
			}
			seen[key] = true
		}
		raw, err := json.Marshal(row)
		if err != nil {
			return err
		}
		if i > 0 {
			remaining--
		}
		if len(raw) > remaining {
			return fmt.Errorf("list byte bound")
		}
		remaining -= len(raw)
	}
	return nil
}

// Append validates the entire new list before returning it. The original is untouched on failure.
func (f Field) Append(v Value, row map[string]Value) (Value, error) {
	if f.Kind != "list" || v.Kind != "list" || v.List == nil {
		return Value{}, fmt.Errorf("append requires list field and value")
	}
	if err := f.Validate(v); err != nil {
		return Value{}, err
	}
	if len(v.List.records) >= f.MaxItems {
		return Value{}, fmt.Errorf("list item bound")
	}
	rows := make([]map[string]Value, len(v.List.records)+1)
	copy(rows, v.List.records)
	rows[len(rows)-1] = row
	out := Value{Kind: "list", List: &ListData{records: rows}}
	if err := f.Validate(out); err != nil {
		return Value{}, err
	}
	out = out.Clone()
	return out, nil
}
func cloneRecord(r Record) Record {
	out := Record{}
	for k, f := range r {
		f.Enum = slices.Clone(f.Enum)
		if len(f.Enum) == 0 {
			f.Enum = nil
		}
		slices.Sort(f.Enum)
		if f.Element != nil {
			f.Element = cloneRecord(f.Element)
		}
		out[k] = f
	}
	return out
}
func (f Field) Retain() (Field, error) {
	r, err := (Record{"value": f}).Retain()
	if err != nil {
		return Field{}, err
	}
	return r.Record["value"], nil
}
func (r Record) HasCollections() bool {
	for _, f := range r {
		if f.Kind == "list" {
			return true
		}
	}
	return false
}

// UnmarshalJSON rejects lossy UTF-8 decoding and ambiguous object keys before
// accepting native values. It does not replace schema validation.
func (v *Value) UnmarshalJSON(raw []byte) error {
	if err := strictJSON(raw); err != nil {
		return err
	}
	type plain Value
	var out plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return err
	}
	*v = Value(out)
	return nil
}
func strictJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 JSON")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > MaxSchemaNesting*4+4 {
			return fmt.Errorf("JSON nesting bound")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				t, err = d.Token()
				if err != nil {
					return err
				}
				key, ok := t.(string)
				if !ok || seen[key] {
					return fmt.Errorf("duplicate JSON field")
				}
				seen[key] = true
				if err = walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err = walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

// The outer Value header is outside a Field's canonical list-byte bound. Every
// recursive node spends from one shared budget before any deep copy or marshal.
const listValueOverhead = len(`{"Kind":"list","Number":0,"Text":"","Flag":false,"List":}`)

func boundedValueSize(v Value, remaining *int, depth int) error {
	take := func(n int) error {
		if n < 0 || n > *remaining {
			return fmt.Errorf("list aggregate byte bound")
		}
		*remaining -= n
		return nil
	}
	if depth > MaxSchemaNesting {
		return fmt.Errorf("list nesting bound")
	}
	if v.Kind != "list" {
		if v.List != nil {
			return fmt.Errorf("nonlist payload")
		}
		if len(v.Kind) > *remaining || len(v.Text) > *remaining {
			return fmt.Errorf("list aggregate byte bound")
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return take(len(raw))
	}
	if v.List == nil || len(v.List.records) > MaxCollectionItems {
		return fmt.Errorf("invalid list")
	}
	if err := take(listValueOverhead + 2); err != nil {
		return err
	}
	for i, row := range v.List.records {
		if i > 0 {
			if err := take(1); err != nil {
				return err
			}
		}
		if err := take(2); err != nil {
			return err
		}
		if len(row) > MaxFields {
			return fmt.Errorf("list record field bound")
		}
		first := true
		for key, child := range row {
			if !first {
				if err := take(1); err != nil {
					return err
				}
			}
			first = false
			if len(key) > *remaining {
				return fmt.Errorf("list aggregate byte bound")
			}
			raw, err := json.Marshal(key)
			if err != nil {
				return err
			}
			if err = take(len(raw) + 1); err != nil {
				return err
			}
			if err = boundedValueSize(child, remaining, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// MarshalJSON retains an explicit empty element record on list fields while
// leaving every legacy scalar schema byte unchanged.
func (f Field) MarshalJSON() ([]byte, error) {
	type plain Field
	if f.Kind != "list" {
		return json.Marshal(plain(f))
	}
	return json.Marshal(struct {
		Kind      string
		MaxBytes  int
		Enum      []string
		Element   Record
		MaxItems  int    `json:",omitempty"`
		UniqueKey string `json:",omitempty"`
	}{f.Kind, f.MaxBytes, f.Enum, f.Element, f.MaxItems, f.UniqueKey})
}

const MaxSchemaFields = 4096

func boundedSchema(r Record, remaining *int, depth int) error {
	if depth > MaxSchemaNesting {
		return fmt.Errorf("schema nesting bound")
	}
	for _, f := range r {
		if *remaining <= 0 {
			return fmt.Errorf("schema total field bound")
		}
		*remaining -= 1
		if f.Element != nil {
			if err := boundedSchema(f.Element, remaining, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
