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

func TestCompileRejectsStaticallyInvalidSchemasAndPreservesValidVariables(t *testing.T) {
	enumField := schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"fast", "slow"}}
	shortField := schema.Field{Kind: "string", MaxBytes: 4}
	childArt, err := author.Compile(author.Definition{
		Ports: map[string]author.Port{"entry": {Peer: "caller", Obligation: "task", Receive: schema.Record{"mode": enumField}, Send: schema.Record{"ok": shortField}}},
		Steps: []author.Step{author.Receive("entry", map[string]string{"mode": "m"}), author.Send("entry", map[string]author.Expr{"ok": author.String("done")}), author.Success()},
	})
	if err != nil {
		t.Fatal(err)
	}
	childTpl, err := author.NewTemplate(author.Application{"worker": {Artifact: childArt, Inputs: map[string]schema.Value{}}}, "worker", "entry")
	if err != nil {
		t.Fatal(err)
	}

	manyBranchSteps := make([]author.Step, 0, 25)
	for range 20 {
		manyBranchSteps = append(manyBranchSteps, author.IfEqual(author.Integer(0), author.Integer(1),
			[]author.Step{author.Set("tmp", author.String("TOO_LONG_STRING")), author.Goto("join")},
			nil,
		))
	}
	manyBranchSteps = append(manyBranchSteps,
		author.Set("tmp", author.String("TOO_LONG_STRING")),
		author.Label("join"),
		author.Set("short", author.Var("tmp")),
		author.Success(),
	)

	invalidCases := []struct {
		name string
		d    author.Definition
	}{
		{
			name: "invalid output enum literal",
			d: author.Definition{
				Outputs: schema.Record{"mode": enumField},
				Steps:   []author.Step{author.Set("mode", author.String("INVALID_ENUM")), author.Success()},
			},
		},
		{
			name: "invalid output bounded string literal",
			d: author.Definition{
				Outputs: schema.Record{"short": shortField},
				Steps:   []author.Step{author.Set("short", author.String("TOO_LONG_STRING")), author.Success()},
			},
		},
		{
			name: "invalid output propagated through variable",
			d: author.Definition{
				Outputs: schema.Record{"mode": enumField},
				Steps:   []author.Step{author.Set("tmp", author.String("INVALID_ENUM")), author.Set("mode", author.Var("tmp")), author.Success()},
			},
		},
		{
			name: "invalid initial value flowing to output",
			d: author.Definition{
				Outputs: schema.Record{"mode": enumField},
				Values:  map[string]schema.Value{"mode": schema.Text("INVALID_ENUM")},
				Steps:   []author.Step{author.Success()},
			},
		},
		{
			name: "invalid enum passing through Var into Request",
			d: author.Definition{
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": shortField}}},
				Steps: []author.Step{
					author.Set("tmp", author.String("NOT_IN_ENUM")),
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("tmp")}, nil),
					author.Success(),
				},
			},
		},
		{
			name: "invalid value passing through Var into Send",
			d: author.Definition{
				Ports: map[string]author.Port{"out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("tmp", author.String("NOT_IN_ENUM")),
					author.Send("out", map[string]author.Expr{"mode": author.Var("tmp")}),
					author.Success(),
				},
			},
		},
		{
			name: "invalid value passing through Var into Invoke",
			d: author.Definition{
				Invocations: map[string]author.Template{"sub": childTpl},
				Steps: []author.Step{
					author.Set("tmp", author.String("NOT_IN_ENUM")),
					author.Invoke("inv", "sub", map[string]author.Expr{"mode": author.Var("tmp")}, nil),
					author.Success(),
				},
			},
		},
		{
			name: "invalid value passing through Var into Append",
			d: author.Definition{
				Collections: map[string]schema.Field{"items": {Kind: "list", MaxItems: 4, MaxBytes: 512, Element: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("tmp", author.String("NOT_IN_ENUM")),
					author.Append("items", map[string]author.Expr{"mode": author.Var("tmp")}),
					author.Success(),
				},
			},
		},
		{
			name: "invalid value in Request Updates targeting Output",
			d: author.Definition{
				Outputs: schema.Record{"mode": enumField},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"q": shortField}}},
				Steps: []author.Step{
					author.Request("req", "tool", map[string]author.Expr{"q": author.String("ok")}, map[string]author.Expr{"mode": author.String("NOT_IN_ENUM")}),
					author.Success(),
				},
			},
		},
		{
			name: "must-constant invalid value preserved across more than 16 branch joins",
			d: author.Definition{
				Outputs: schema.Record{"short": shortField},
				Steps:   manyBranchSteps,
			},
		},
		{
			name: "contradicted IfEqual then branch preserves invalid static constant",
			d: author.Definition{
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("x", author.String("NOT_IN_ENUM")),
					author.IfEqual(author.Var("x"), author.String("fast"),
						[]author.Step{
							author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("x")}, nil),
							author.Success(),
						},
						[]author.Step{author.Success()},
					),
				},
			},
		},
		{
			name: "contradicted IfEqual else branch preserves invalid static constant",
			d: author.Definition{
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("x", author.String("NOT_IN_ENUM")),
					author.IfEqual(author.Var("x"), author.String("NOT_IN_ENUM"),
						[]author.Step{author.Success()},
						[]author.Step{
							author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("x")}, nil),
							author.Success(),
						},
					),
				},
			},
		},
		{
			name: "disjoint input enum flowing into Request enum",
			d: author.Definition{
				Inputs:  schema.Record{"mode": {Kind: "string", MaxBytes: 10, Enum: []string{"a", "b"}}},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("mode")}, nil),
					author.Success(),
				},
			},
		},
		{
			name: "disjoint input MaxBytes shorter than all target enum members",
			d: author.Definition{
				Inputs:  schema.Record{"mode": {Kind: "string", MaxBytes: 2}},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("mode")}, nil),
					author.Success(),
				},
			},
		},
		{
			name: "contradicted IfEqual then branch on dynamic input enum does not launder incompatible source enum",
			d: author.Definition{
				Inputs:  schema.Record{"x": {Kind: "string", MaxBytes: 16, Enum: []string{"pending"}}},
				Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 16, Enum: []string{"done"}}},
				Steps: []author.Step{
					author.IfEqual(author.Var("x"), author.String("done"),
						[]author.Step{author.Set("out", author.Var("x")), author.Success()},
						[]author.Step{author.Failure()},
					),
				},
			},
		},
		{
			name: "ForEach over Receive-bound list preserves element enum constraints",
			d: author.Definition{
				Ports: map[string]author.Port{
					"in": {
						Local:      true,
						Peer:       "local",
						Obligation: "in",
						Receive: schema.Record{
							"items": {
								Kind:     "list",
								MaxItems: 4,
								MaxBytes: 512,
								Element:  schema.Record{"status": {Kind: "string", MaxBytes: 16, Enum: []string{"pending"}}},
							},
						},
					},
				},
				Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 16, Enum: []string{"done"}}},
				Steps: []author.Step{
					author.Receive("in", map[string]string{"items": "items"}),
					author.ForEach(author.Var("items"), map[string]string{"status": "out"}, author.Success()),
					author.Failure(),
				},
			},
		},
		{
			name: "nested ForEach preserves inner list element enum constraints",
			d: author.Definition{
				Collections: map[string]schema.Field{
					"outer": {
						Kind:     "list",
						MaxItems: 4,
						MaxBytes: 512,
						Element: schema.Record{
							"children": {
								Kind:     "list",
								MaxItems: 4,
								MaxBytes: 256,
								Element:  schema.Record{"x": {Kind: "string", MaxBytes: 16, Enum: []string{"pending"}}},
							},
						},
					},
				},
				Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 16, Enum: []string{"done"}}},
				Steps: []author.Step{
					author.Append("outer", map[string]author.Expr{
						"children": author.Value(schema.List(map[string]schema.Value{"x": schema.Text("pending")})),
					}),
					author.ForEach(author.Var("outer"), map[string]string{"children": "child"},
						author.ForEach(author.Var("child"), map[string]string{"x": "out"},
							author.Success(),
						),
					),
					author.Failure(),
				},
			},
		},
		{
			name: "ambiguous SendUsed ports sharing Peer and Obligation with conflicting Send schemas",
			d: author.Definition{
				Inputs: schema.Record{"dyn": {Kind: "string", MaxBytes: 32}},
				Ports: map[string]author.Port{
					"strict_out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}},
					"wide_out":   {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": {Kind: "string", MaxBytes: 32}}},
				},
				Steps: []author.Step{
					author.IfEqual(author.Var("dyn"), author.String("NEVER"),
						[]author.Step{author.Send("wide_out", map[string]author.Expr{"mode": author.String("any")})},
						[]author.Step{author.Send("strict_out", map[string]author.Expr{"mode": author.Var("dyn")})},
					),
					author.Success(),
				},
			},
		},
		{
			name: "Append inside ForEach does not erase unmodified outer variable Must-Constant sharing element field name",
			d: author.Definition{
				Outputs: schema.Record{"out": enumField},
				Collections: map[string]schema.Field{
					"items": {Kind: "list", MaxItems: 4, MaxBytes: 256, Element: schema.Record{"v": shortField}},
					"log":   {Kind: "list", MaxItems: 4, MaxBytes: 256, Element: schema.Record{"mode": shortField}},
				},
				Steps: []author.Step{
					author.Set("mode", author.String("INVALID_ENUM")),
					author.ForEach(author.Var("items"), map[string]string{"v": "item_v"},
						author.Append("log", map[string]author.Expr{"mode": author.String("ok")}),
					),
					author.Set("out", author.Var("mode")),
					author.Success(),
				},
			},
		},
	}

	for _, tc := range invalidCases {
		t.Run("reject_"+tc.name, func(t *testing.T) {
			if _, err := author.Compile(tc.d); err == nil {
				t.Fatalf("expected compile rejection for %s, got nil", tc.name)
			}
		})
	}

	validCases := []struct {
		name string
		d    author.Definition
	}{
		{
			name: "valid enum and bounded string variables across Set Request Send Append Outputs",
			d: author.Definition{
				Outputs:     schema.Record{"mode": enumField, "short": shortField},
				Collections: map[string]schema.Field{"items": {Kind: "list", MaxItems: 4, MaxBytes: 512, Element: schema.Record{"mode": enumField}}},
				Effects:     map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": shortField}}},
				Ports:       map[string]author.Port{"out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("tmp_mode", author.String("fast")),
					author.Set("tmp_short", author.String("ok")),
					author.Append("items", map[string]author.Expr{"mode": author.Var("tmp_mode")}),
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("tmp_mode")}, nil),
					author.Send("out", map[string]author.Expr{"mode": author.Var("tmp_mode")}),
					author.Set("mode", author.Var("tmp_mode")),
					author.Set("short", author.Var("tmp_short")),
					author.Success(),
				},
			},
		},
		{
			name: "runtime-dependent input variable flowing to stricter Request Send and Outputs",
			d: author.Definition{
				Inputs:  schema.Record{"dynamic": {Kind: "string", MaxBytes: 20}},
				Outputs: schema.Record{"short": shortField},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Ports:   map[string]author.Port{"out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("dynamic")}, nil),
					author.Send("out", map[string]author.Expr{"mode": author.Var("dynamic")}),
					author.Set("short", author.Var("dynamic")),
					author.Success(),
				},
			},
		},
		{
			name: "FR1 contradicted IfEqual branch does not pollute valid constant across join",
			d: author.Definition{
				Outputs: schema.Record{"out": enumField},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.Set("mode", author.String("fast")),
					author.IfEqual(author.Var("mode"), author.String("NOT_IN_ENUM"),
						[]author.Step{author.Set("seen", author.Boolean(true))},
						[]author.Step{author.Set("seen", author.Boolean(false))},
					),
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("mode")}, nil),
					author.Set("out", author.Var("mode")),
					author.Success(),
				},
			},
		},
		{
			name: "FR2 correlated flag guard on fallback sentinel value",
			d: author.Definition{
				Inputs:  schema.Record{"go": {Kind: "bool"}},
				Outputs: schema.Record{"out": enumField},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.IfEqual(author.Var("go"), author.Boolean(true),
						[]author.Step{author.Set("mode", author.String("fast"))},
						[]author.Step{author.Set("mode", author.String("UNSET_SENTINEL"))},
					),
					author.IfEqual(author.Var("go"), author.Boolean(true),
						[]author.Step{
							author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("mode")}, nil),
							author.Set("out", author.Var("mode")),
							author.Success(),
						},
						[]author.Step{author.Failure()},
					),
				},
			},
		},
		{
			name: "FR3 collection row correlation guarded by active column in ForEach",
			d: author.Definition{
				Collections: map[string]schema.Field{
					"rows": {
						Kind: "list", MaxItems: 4, MaxBytes: 512,
						Element: schema.Record{"active": {Kind: "bool"}, "mode": {Kind: "string", MaxBytes: 20}},
					},
				},
				Values: map[string]schema.Value{
					"rows": schema.List(
						map[string]schema.Value{"active": schema.Bool(false), "mode": schema.Text("SKIP_ROW")},
						map[string]schema.Value{"active": schema.Bool(true), "mode": schema.Text("fast")},
					),
				},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
				Steps: []author.Step{
					author.ForEach(author.Var("rows"), map[string]string{"active": "act", "mode": "m"},
						author.IfEqual(author.Var("act"), author.Boolean(true),
							[]author.Step{author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("m")}, nil)},
							nil,
						),
					),
					author.Success(),
				},
			},
		},
		{
			name: "FR4 non-empty ForEach overwriting initial sentinel before Success",
			d: author.Definition{
				Outputs: schema.Record{"x": enumField},
				Collections: map[string]schema.Field{
					"items": {
						Kind: "list", MaxItems: 4, MaxBytes: 256,
						Element: schema.Record{"v": enumField},
					},
				},
				Values: map[string]schema.Value{
					"x":     schema.Text("UNSET_SENTINEL"),
					"items": schema.List(map[string]schema.Value{"v": schema.Text("fast")}),
				},
				Steps: []author.Step{
					author.ForEach(author.Var("items"), map[string]string{"v": "x"}),
					author.Success(),
				},
			},
		},
		{
			name: "FR5 intermediate scratch assignment to output variable overwritten before Success",
			d: author.Definition{
				Outputs: schema.Record{"mode": enumField},
				Steps: []author.Step{
					author.Set("mode", author.String("TEMP_WORKING_VALUE")),
					author.Set("mode", author.String("fast")),
					author.Success(),
				},
			},
		},
		{
			name: "multi-port SendUsed allowed when obligations differ or schemas are identical aliases",
			d: author.Definition{
				Ports: map[string]author.Port{
					"out_a":     {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}},
					"out_alias": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}},
					"out_audit": {Peer: "receiver", Obligation: "audit", Send: schema.Record{"note": {Kind: "string", MaxBytes: 32}}},
					"unused":    {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": {Kind: "string", MaxBytes: 32}}},
				},
				Steps: []author.Step{
					author.Send("out_a", map[string]author.Expr{"mode": author.String("fast")}),
					author.Send("out_alias", map[string]author.Expr{"mode": author.String("slow")}),
					author.Send("out_audit", map[string]author.Expr{"note": author.String("audit-trail")}),
					author.Success(),
				},
			},
		},
	}

	for _, tc := range validCases {
		t.Run("accept_"+tc.name, func(t *testing.T) {
			if _, err := author.Compile(tc.d); err != nil {
				t.Fatalf("unexpected compile failure for %s: %v", tc.name, err)
			}
		})
	}
}
