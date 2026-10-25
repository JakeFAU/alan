package author_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

func textRecord(name string) schema.Record {
	return schema.Record{name: {Kind: "string", MaxBytes: 200}}
}
func TestCompileDeterministicImmutable(t *testing.T) {
	d := author.Definition{Name: "answer", Inputs: textRecord("question"), Outputs: textRecord("answer"), Templates: map[string]string{"plan": "verbatim\nplanning", "reply": "answer the question"}, Steps: []author.Step{author.Set("answer", author.Var("question")), author.Success()}}
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	d.Templates = map[string]string{"reply": "answer the question", "plan": "verbatim\nplanning"}
	b, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID() != b.ID() || !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("map order changed artifact")
	}
	raw := a.Bytes()
	raw[0] = '!'
	d.Templates["plan"] = "changed"
	d.Inputs["question"] = schema.Field{Kind: "bool"}
	meta := a.Metadata()
	meta.Templates["plan"] = "changed"
	meta.Inputs.Record["question"] = schema.Field{Kind: "bool"}
	if !bytes.Equal(a.Bytes(), b.Bytes()) || a.Metadata().Templates["plan"] != "verbatim\nplanning" {
		t.Fatal("artifact aliases caller memory")
	}
	if len(a.Metadata().Positions) == 0 || a.Metadata().Inputs.ID == "" || a.Metadata().Compiler == "" {
		t.Fatal("missing recovery metadata")
	}
	for _, field := range []string{"Compiler", "Inputs", "Outputs", "Positions", "Templates"} {
		if !bytes.Contains(a.Bytes(), []byte(field)) {
			t.Fatalf("missing retained %s", field)
		}
	}
}

func TestCompileRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name string
		def  author.Definition
		want string
	}{
		{"missing reference", author.Definition{Steps: []author.Step{author.Set("x", author.Var("missing")), author.Success()}}, "missing"},
		{"duplicate label", author.Definition{Steps: []author.Step{author.Label("same"), author.Label("same"), author.Success()}}, "same"},
		{"missing label", author.Definition{Steps: []author.Step{author.Goto("gone"), author.Success()}}, "gone"},
		{"backward label", author.Definition{Steps: []author.Step{author.Label("again"), author.Set("x", author.String("x")), author.Goto("again"), author.Success()}}, "unbounded"},
		{"negative repeat", author.Definition{Steps: []author.Step{author.Repeat(-1, author.Success()), author.Success()}}, "repeat"},
		{"repeat limit", author.Definition{Steps: []author.Step{author.Repeat(10001, author.Set("x", author.String("x"))), author.Success()}}, "limit"},
		{"wrong add", author.Definition{Steps: []author.Step{author.Add("x", author.String("x"), author.Integer(1)), author.Success()}}, "int"},
		{"branch types", author.Definition{Steps: []author.Step{author.IfEqual(author.String("x"), author.Integer(1), []author.Step{author.Success()}, []author.Step{author.Failure()})}}, "type"},
		{"partial definition", author.Definition{Steps: []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(false), []author.Step{author.Set("x", author.Integer(1))}, nil), author.Set("y", author.Var("x")), author.Success()}}, "x"},
		{"missing output", author.Definition{Outputs: textRecord("answer"), Steps: []author.Step{author.Success()}}, "answer"},
		{"unknown effect", author.Definition{Steps: []author.Step{author.Request("p", "missing", nil, nil), author.Success()}}, "missing"},
		{"fallthrough", author.Definition{Steps: []author.Step{author.Set("x", author.Integer(1))}}, "terminal"},
		{"invalid schema", author.Definition{Inputs: schema.Record{"bad": {Kind: "object"}}, Steps: []author.Step{author.Success()}}, "bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := author.Compile(tt.def)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestCompileEffectsAndBoundedControl(t *testing.T) {
	d := author.Definition{Name: "bounded", Values: map[string]schema.Value{"spent": schema.Int(0)}, Effects: map[string]author.Effect{"plan": {Capability: "model", Input: textRecord("question"), Output: schema.Record{"answer": {Kind: "string", MaxBytes: 200}}}}, Templates: map[string]string{"prompt": "retained"}, Steps: []author.Step{
		author.Repeat(0, author.Set("unused", author.Var("absent"))),
		author.Repeat(2, author.Add("spent", author.Var("spent"), author.Integer(1))),
		author.Request("pending", "plan", map[string]author.Expr{"question": author.String("why?")}, map[string]author.Expr{"spent": author.Offset("spent", 1)}),
		author.Await("plan", "pending", map[string]string{"valid": "valid", "disposition": "disposition", "answer": "answer"}),
		author.IfEqual(author.Var("valid"), author.Boolean(true), []author.Step{author.Goto("done")}, []author.Step{author.Failure()}), author.Label("done"), author.Success(),
	}}
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Code []struct {
			Op      string
			Updates map[string]any
			Effect  *struct{ Kind string }
		}
	}
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, in := range p.Code {
		if in.Op == "add" {
			n++
		}
		if in.Effect != nil && (in.Effect.Kind != "adapter" || len(in.Updates) != 1) {
			t.Fatal("effect/budget not generic and atomic")
		}
	}
	if n != 2 {
		t.Fatalf("repeat expanded %d times", n)
	}
	if a.Metadata().Effects["plan"].Output.ID == "" {
		t.Fatal("effect schema not retained")
	}
}

func TestCompileTerminalBranchesAndScopedRepeatExit(t *testing.T) {
	for _, steps := range [][]author.Step{
		{author.IfEqual(author.Boolean(true), author.Boolean(false), []author.Step{author.Success()}, []author.Step{author.Failure()})},
		{author.Repeat(2, author.Label("start"), author.Set("x", author.Integer(1)), author.Goto("done")), author.Label("done"), author.Success()},
	} {
		if _, err := author.Compile(author.Definition{Steps: steps}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCompileBindingAndRequestErrors(t *testing.T) {
	base := func() author.Definition {
		return author.Definition{Effects: map[string]author.Effect{"plan": {Capability: "model", Input: textRecord("q"), Output: textRecord("answer")}}, Ports: map[string]author.Port{"peer": {Peer: "peer", Obligation: "q", Receive: schema.Record{"x": {Kind: "int"}, "y": {Kind: "int"}}}}}
	}
	for _, tt := range []struct {
		name  string
		steps []author.Step
	}{
		{"duplicate binding", []author.Step{author.Receive("peer", map[string]string{"x": "v", "y": "v"}), author.Success()}},
		{"unknown binding", []author.Step{author.Receive("peer", map[string]string{"no": "v"}), author.Success()}},
		{"wrong payload", []author.Step{author.Request("pending", "plan", map[string]author.Expr{"q": author.Integer(2)}, nil), author.Success()}},
		{"fake request", []author.Step{author.Set("pending", author.String("not-an-intent")), author.Await("plan", "pending", nil), author.Success()}},
		{"update collision", []author.Step{author.Request("pending", "plan", map[string]author.Expr{"q": author.String("q")}, map[string]author.Expr{"pending": author.String("fake")}), author.Success()}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := base()
			d.Steps = tt.steps
			if _, err := author.Compile(d); err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
	d := base()
	d.Effects["plan"] = author.Effect{Capability: "model", Output: schema.Record{"valid": {Kind: "bool"}}}
	d.Steps = []author.Step{author.Success()}
	if _, err := author.Compile(d); err == nil {
		t.Fatal("provider can declare runtime validation field")
	}
}

func TestCompileRejectsAmbiguousPortsAndInvalidUTF8(t *testing.T) {
	_, err := author.Compile(author.Definition{Ports: map[string]author.Port{"a": {Peer: "peer", Obligation: "q", Receive: textRecord("q")}, "b": {Peer: "peer", Obligation: "q", Receive: schema.Record{"q": {Kind: "int"}}}}, Steps: []author.Step{author.Receive("a", nil), author.Receive("b", nil), author.Success()}})
	if err == nil {
		t.Fatal("ambiguous receive contract accepted")
	}
	_, err = author.Compile(author.Definition{Templates: map[string]string{"bad": string([]byte{0xff})}, Steps: []author.Step{author.Success()}})
	if err == nil {
		t.Fatal("template bytes silently changed by encoding")
	}
}

func TestCompileRejectsInvalidUTF8Identifiers(t *testing.T) {
	bad := string([]byte{0xfe})
	other := string([]byte{0xff})
	tests := []struct {
		name string
		d    author.Definition
	}{
		{"exact aliasing probe", author.Definition{Steps: []author.Step{author.Set(bad, author.Integer(1)), author.Set(other, author.Integer(2)), author.IfEqual(author.Var(bad), author.Integer(1), []author.Step{author.Success()}, []author.Step{author.Failure()})}}},
		{"reference", author.Definition{Steps: []author.Step{author.Set("x", author.Var(bad)), author.Success()}}},
		{"label", author.Definition{Steps: []author.Step{author.Label(bad), author.Success()}}},
		{"goto", author.Definition{Steps: []author.Step{author.Goto(bad), author.Success()}}},
		{"name", author.Definition{Name: bad, Steps: []author.Step{author.Success()}}},
		{"value", author.Definition{Values: map[string]schema.Value{bad: schema.Int(1)}, Steps: []author.Step{author.Success()}}},
		{"effect", author.Definition{Effects: map[string]author.Effect{bad: {Capability: "model"}}, Steps: []author.Step{author.Success()}}},
		{"capability", author.Definition{Effects: map[string]author.Effect{"effect": {Capability: bad}}, Steps: []author.Step{author.Success()}}},
		{"port", author.Definition{Ports: map[string]author.Port{bad: {Peer: "peer", Obligation: "q"}}, Steps: []author.Step{author.Success()}}},
		{"peer", author.Definition{Ports: map[string]author.Port{"port": {Peer: bad, Obligation: "q"}}, Steps: []author.Step{author.Success()}}},
		{"obligation", author.Definition{Ports: map[string]author.Port{"port": {Peer: "peer", Obligation: bad}}, Steps: []author.Step{author.Success()}}},
		{"template key", author.Definition{Templates: map[string]string{bad: "prompt"}, Steps: []author.Step{author.Success()}}},
		{"input field", author.Definition{Inputs: schema.Record{bad: {Kind: "int"}}, Steps: []author.Step{author.Success()}}},
		{"binding destination", author.Definition{Ports: map[string]author.Port{"port": {Peer: "peer", Obligation: "q", Receive: textRecord("q")}}, Steps: []author.Step{author.Receive("port", map[string]string{"q": bad}), author.Success()}}},
		{"binding field", author.Definition{Ports: map[string]author.Port{"port": {Peer: "peer", Obligation: "q", Receive: textRecord("q")}}, Steps: []author.Step{author.Receive("port", map[string]string{bad: "q"}), author.Success()}}},
		{"request destination", author.Definition{Effects: map[string]author.Effect{"plan": {Capability: "model"}}, Steps: []author.Step{author.Request(bad, "plan", nil, nil), author.Success()}}},
		{"request effect", author.Definition{Steps: []author.Step{author.Request("pending", bad, nil, nil), author.Success()}}},
		{"await identity", author.Definition{Effects: map[string]author.Effect{"plan": {Capability: "model"}}, Steps: []author.Step{author.Await("plan", bad, nil), author.Success()}}},
		{"update destination", author.Definition{Effects: map[string]author.Effect{"plan": {Capability: "model"}}, Steps: []author.Step{author.Request("pending", "plan", nil, map[string]author.Expr{bad: author.Integer(1)}), author.Success()}}},
		{"payload reference", author.Definition{Effects: map[string]author.Effect{"plan": {Capability: "model", Input: textRecord("q")}}, Steps: []author.Step{author.Request("pending", "plan", map[string]author.Expr{"q": author.Var(bad)}, nil), author.Success()}}},
		{"unreachable destination", author.Definition{Steps: []author.Step{author.Success(), author.Set(bad, author.Integer(1)), author.Success()}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := author.Compile(tt.d)
			if err == nil || !strings.Contains(err.Error(), "UTF-8") {
				t.Fatalf("want useful UTF-8 rejection, got %v", err)
			}
		})
	}
}

func TestCompilePreservesUnicodeEffectRequestIdentity(t *testing.T) {
	a, err := author.Compile(author.Definition{Effects: map[string]author.Effect{"推論": {Capability: "模型", Input: textRecord("質問"), Output: textRecord("回答")}}, Values: map[string]schema.Value{"費用": schema.Int(0)}, Steps: []author.Step{author.Request("保留", "推論", map[string]author.Expr{"質問": author.String("なぜ")}, map[string]author.Expr{"費用": author.Offset("費用", 1)}), author.Await("推論", "保留", map[string]string{"valid": "有効", "回答": "結果"}), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	if a.Metadata().Effects["推論"].Capability != "模型" || !bytes.Contains(a.Bytes(), []byte("保留")) {
		t.Fatal("Unicode effect identifiers changed")
	}
}
