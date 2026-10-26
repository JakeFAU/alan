package schema

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBoundedLists(t *testing.T) {
	var record Record
	if err := json.Unmarshal([]byte(`{"items":{"Kind":"list","MaxItems":8,"MaxBytes":4096,"UniqueKey":"id","Element":{"id":{"Kind":"string","MaxBytes":20},"payload":{"Kind":"string","MaxBytes":20}}}}`), &record); err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, 3, 7, 8, 9} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			rows := make([]map[string]Value, count)
			for i := range rows {
				rows[i] = map[string]Value{"id": Text(fmt.Sprint(i)), "payload": Text("same")}
			}
			raw, _ := json.Marshal(rows)
			var value Value
			if err := json.Unmarshal(append(append([]byte(`{"Kind":"list","List":`), raw...), '}'), &value); err != nil {
				t.Fatal(err)
			}
			err := record.Validate(map[string]Value{"items": value})
			if count <= 8 && err != nil {
				t.Fatalf("bounded work set rejected: %v", err)
			}
			if count == 9 && err == nil {
				t.Fatal("over-limit accepted")
			}
		})
	}
}

func TestListBoundsCopiesAndCanonicalization(t *testing.T) {
	f := Field{Kind: "list", MaxItems: 8, MaxBytes: 4096, UniqueKey: "id", Element: Record{"id": {Kind: "string", MaxBytes: 20}, "payload": {Kind: "string", MaxBytes: 20}}}
	rows := []map[string]Value{{"id": Text("a"), "payload": Text("same")}, {"id": Text("b"), "payload": Text("same")}}
	value := List(rows...)
	rows[0]["id"] = Text("changed")
	copied, _ := value.Records()
	copied[1]["id"] = Text("changed")
	at, _ := value.At(0)
	if at["id"] != Text("a") {
		t.Fatal("constructor alias")
	}
	at, _ = value.At(1)
	if at["id"] != Text("b") {
		t.Fatal("accessor alias")
	}
	if !value.Equal(List(map[string]Value{"payload": Text("same"), "id": Text("a")}, map[string]Value{"payload": Text("same"), "id": Text("b")})) {
		t.Fatal("equality used identity or map order")
	}
	if value.Equal(List(map[string]Value{"id": Text("b"), "payload": Text("same")}, map[string]Value{"id": Text("a"), "payload": Text("same")})) {
		t.Fatal("order ignored")
	}
	if _, err := value.At(-1); err == nil {
		t.Fatal("negative index")
	}
	if _, err := value.At(2); err == nil {
		t.Fatal("past end")
	}
	a, _ := json.Marshal(List())
	b, _ := json.Marshal(List([]map[string]Value{}...))
	if string(a) != string(b) || string(a) != `{"Kind":"list","Number":0,"Text":"","Flag":false,"List":[]}` {
		t.Fatalf("empty encoding: %s %s", a, b)
	}
	for _, row := range []map[string]Value{{"id": Text("a"), "payload": Text("different")}, {"id": Text("c"), "payload": Text(string([]byte{0xff}))}, {"id": Text("c"), "payload": Int(1)}, {"id": Text("c"), "payload": Text("same"), "extra": Bool(true)}} {
		if _, err := f.Append(value, row); err == nil {
			t.Fatalf("bad append accepted: %+v", row)
		}
	}
	n, _ := value.Len()
	if n != 2 {
		t.Fatal("failed append partially mutated source")
	}
	raw, _ := json.Marshal(value.List)
	small := f
	small.MaxBytes = len(raw) - 1
	if small.Validate(value) == nil {
		t.Fatal("canonical byte limit ignored")
	}
	small.MaxBytes = len(raw)
	if err := small.Validate(value); err != nil {
		t.Fatal("exact byte bound", err)
	}
	nested := Field{Kind: "list", MaxItems: 1, MaxBytes: 4096, Element: Record{"children": f}}
	if err := nested.Validate(List(map[string]Value{"children": value})); err != nil {
		t.Fatal(err)
	}
	nested.MaxBytes = len(raw)
	if nested.Validate(List(map[string]Value{"children": value})) == nil {
		t.Fatal("outer total bytes ignored")
	}
	retained, err := (Record{"outer": nested}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	nested.Element["children"] = Field{Kind: "int"}
	if retained.Record["outer"].Element["children"].Kind != "list" {
		t.Fatal("nested schema alias")
	}
	cycle := Record{}
	cycle["loop"] = Field{Kind: "list", MaxItems: 1, MaxBytes: 10, Element: cycle}
	if cycle.Check() == nil {
		t.Fatal("cyclic schema accepted")
	}
	for _, bad := range []Field{{Kind: "list", MaxItems: 0, MaxBytes: 4096, Element: Record{}}, {Kind: "list", MaxItems: 1, MaxBytes: 1, Element: Record{}}, {Kind: "list", MaxItems: 1, MaxBytes: 4096, Element: Record{}, UniqueKey: "missing"}, {Kind: "int", Element: Record{}}, {Kind: "list", MaxItems: 1, MaxBytes: 4096, Element: Record{"key": {Kind: "int"}}, UniqueKey: "key"}} {
		if _, err := bad.Retain(); err == nil {
			t.Fatalf("malformed schema accepted: %+v", bad)
		}
	}
	scalar, _ := json.Marshal(Text("old"))
	if string(scalar) != `{"Kind":"string","Number":0,"Text":"old","Flag":false}` {
		t.Fatal("legacy scalar bytes changed")
	}
}
func TestListRejectsMalformedJSON(t *testing.T) {
	for _, raw := range []string{
		`{"Kind":"list","List":[{"x":{"Kind":"int","Number":1,"extra":2}}]}`,
		`{"Kind":"list","List":[{"x":{"Kind":"int","Number":1},"x":{"Kind":"int","Number":2}}]}`,
		"{\"Kind\":\"list\",\"List\":[{\"x\":{\"Kind\":\"string\",\"Text\":\"" + string([]byte{0xff}) + "\"}}]}",
	} {
		var v Value
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			t.Fatalf("malformed list JSON accepted: %q", raw)
		}
	}
}

func TestListRejectsAliasedAggregateBeforeCopy(t *testing.T) {
	leaf := List(map[string]Value{"text": Text(string(make([]byte, 4096)))})
	shared := make([]map[string]Value, 300)
	for i := range shared {
		shared[i] = map[string]Value{"child": leaf}
	}
	oversized := Value{Kind: "list", List: &ListData{records: shared}}
	if oversized.Valid() {
		t.Fatal("small aliased graph exceeds aggregate canonical byte bound")
	}
	if oversized.Clone().Valid() {
		t.Fatal("clone accepted amplified graph")
	}
	if List(shared...).Valid() {
		t.Fatal("constructor accepted amplified graph")
	}
}

func TestListNestingAndEmptyRecordSchemaRoundTrip(t *testing.T) {
	field := Field{Kind: "list", MaxItems: 1, MaxBytes: MaxCollectionBytes, Element: Record{"leaf": {Kind: "string", MaxBytes: 4}}}
	value := List(map[string]Value{"leaf": Text("ok")})
	for depth := 1; depth < MaxSchemaNesting; depth++ {
		field = Field{Kind: "list", MaxItems: 1, MaxBytes: MaxCollectionBytes, Element: Record{"child": field}}
		value = List(map[string]Value{"child": value})
	}
	if err := field.Validate(value); err != nil {
		t.Fatalf("supported nesting boundary: %v", err)
	}
	extra := Field{Kind: "list", MaxItems: 1, MaxBytes: MaxCollectionBytes, Element: Record{"child": field}}
	if _, err := extra.Retain(); err == nil {
		t.Fatal("excess nesting schema accepted")
	}
	if List(map[string]Value{"child": value}).Valid() {
		t.Fatal("excess nesting value accepted")
	}
	empty := Record{"items": {Kind: "list", MaxItems: 1, MaxBytes: 4, Element: Record{}}}
	retained, err := empty.Retain()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(retained)
	if err != nil {
		t.Fatal(err)
	}
	var reopened Retained
	if err = json.Unmarshal(raw, &reopened); err != nil {
		t.Fatal(err)
	}
	if err = reopened.Check(); err != nil {
		t.Fatal("empty element record lost", err)
	}
	if err = reopened.Record.Validate(map[string]Value{"items": List(map[string]Value{})}); err != nil {
		t.Fatal(err)
	}
	if List(nil).Valid() {
		t.Fatal("null record accepted")
	}
	alias := Record{"leaf": {Kind: "int"}}
	for i := 0; i < 4; i++ {
		next := Record{}
		for j := 0; j < 16; j++ {
			next[fmt.Sprint(j)] = Field{Kind: "list", MaxItems: 1, MaxBytes: 1024, Element: alias}
		}
		alias = next
	}
	if alias.Check() == nil {
		t.Fatal("aliased schema exceeds total field bound")
	}
}

func TestAppendRejectsWrongKindsAndMalformedLists(t *testing.T) {
	list := Field{Kind: "list", MaxItems: 2, MaxBytes: 512, Element: Record{"x": {Kind: "int"}}}
	tests := []struct {
		name  string
		field Field
		value Value
	}{
		{"int receiver", Field{Kind: "int"}, Int(1)},
		{"bool receiver", Field{Kind: "bool"}, Bool(true)},
		{"string receiver", Field{Kind: "string", MaxBytes: 8}, Text("one")},
		{"scalar schema list value", Field{Kind: "int"}, List()},
		{"list schema scalar value", list, Int(1)},
		{"nil list storage", list, Value{Kind: "list"}},
		{"mixed scalar list payload", list, Value{Kind: "list", Number: 1, List: &ListData{}}},
		{"wrong tagged list storage", list, Value{Kind: "int", List: &ListData{}}},
		{"malformed list schema", Field{Kind: "list"}, List()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if panicValue := recover(); panicValue != nil {
					t.Errorf("Append panicked instead of returning an error: %v", panicValue)
				}
			}()
			if _, err := test.field.Append(test.value, map[string]Value{"x": Int(1)}); err == nil {
				t.Fatal("invalid append accepted")
			}
		})
	}
}

func TestAppendValidListStillCopies(t *testing.T) {
	field := Field{Kind: "list", MaxItems: 2, MaxBytes: 512, Element: Record{"x": {Kind: "int"}}}
	source := List()
	row := map[string]Value{"x": Int(1)}
	result, err := field.Append(source, row)
	if err != nil {
		t.Fatal(err)
	}
	row["x"] = Int(99)
	got, err := result.At(0)
	if err != nil || got["x"] != Int(1) {
		t.Fatalf("valid append or copy failed: %+v %v", got, err)
	}
	count, err := source.Len()
	if err != nil || count != 0 {
		t.Fatalf("append mutated source: count=%d err=%v", count, err)
	}
}
