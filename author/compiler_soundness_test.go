package author

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

func assertCompileStringFlow(t *testing.T, err error, accept bool, refusal string) {
	t.Helper()
	if accept {
		if err != nil {
			t.Fatalf("expected valid flow to compile, got %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), refusal) {
		t.Fatalf("expected refusal containing %q, got %v", refusal, err)
	}
}

func TestCompileStringSchemaContainment(t *testing.T) {
	cases := []struct {
		name           string
		source, target schema.Field
		accept         bool
	}{
		{"enum overlap", schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"a", "b"}}, schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"a"}}, false},
		{"unconstrained to enum", schema.Field{Kind: "string", MaxBytes: 10}, schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"a"}}, false},
		{"larger bound", schema.Field{Kind: "string", MaxBytes: 10}, schema.Field{Kind: "string", MaxBytes: 5}, false},
		{"Unicode byte bound", schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a", "é"}}, schema.Field{Kind: "string", MaxBytes: 1}, false},
		{"subset", schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"a"}}, schema.Field{Kind: "string", MaxBytes: 5, Enum: []string{"a", "b"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Definition{Inputs: schema.Record{"in": tc.source}, Outputs: schema.Record{"out": tc.target}, Steps: []Step{Set("out", Var("in")), Success()}})
			assertCompileStringFlow(t, err, tc.accept, `output "out": source string schema exceeds destination bound or enum`)
		})
	}
}
func TestCompileStringSchemaBranchJoin(t *testing.T) {
	for _, tc := range []struct {
		name, a, b string
		accept     bool
	}{{"invalid", "x", "y", false}, {"valid", "a", "b", true}, {"mixed", "a", "x", false}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Definition{Inputs: schema.Record{"flag": {Kind: "bool"}}, Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 2, Enum: []string{"a", "b"}}}, Steps: []Step{IfEqual(Var("flag"), Boolean(true), []Step{Set("out", String(tc.a))}, []Step{Set("out", String(tc.b))}), Success()}})
			assertCompileStringFlow(t, err, tc.accept, `output "out": source string schema exceeds destination bound or enum`)
		})
	}
}

func TestCompileLoopInvalidatesGuardFacts(t *testing.T) {
	d := Definition{Inputs: schema.Record{"flag": {Kind: "bool"}}, Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 1, Enum: []string{"a"}}}, Collections: map[string]schema.Field{"items": {Kind: "list", MaxItems: 1, MaxBytes: 256, Element: schema.Record{"v": {Kind: "bool"}}}}, Values: map[string]schema.Value{"items": schema.List(map[string]schema.Value{"v": schema.Bool(false)})}, Steps: []Step{
		IfEqual(Var("flag"), Boolean(true), []Step{Set("out", String("x"))}, []Step{Set("out", String("a"))}),
		ForEach(Var("items"), map[string]string{"v": "flag"}),
		IfEqual(Var("flag"), Boolean(true), []Step{Failure()}, []Step{Success()}),
	}}
	_, err := Compile(d)
	assertCompileStringFlow(t, err, false, `output "out": source string schema exceeds destination bound or enum`)
}

func TestCompileAdapterDefaultAtStringSink(t *testing.T) {
	field := schema.Field{Kind: "string", MaxBytes: 2, Enum: []string{"ok"}}
	for _, guarded := range []bool{false, true} {
		d := Definition{
			Outputs: schema.Record{"out": field},
			Effects: map[string]Effect{"tool": {Capability: "tool", Output: schema.Record{"out": field}}},
			Steps: []Step{
				Request("op", "tool", nil, nil),
				Await("tool", "op", map[string]string{"out": "out", schema.ResultValid: "valid"}),
			},
		}
		if guarded {
			d.Steps = append(d.Steps, IfEqual(Var("valid"), Boolean(true), []Step{Success()}, []Step{Failure()}))
		} else {
			d.Steps = append(d.Steps, Success())
		}
		_, err := Compile(d)
		assertCompileStringFlow(t, err, guarded, `output "out": source string schema exceeds destination bound or enum`)
	}
}

func TestCompiledStringBoundsAgainstReducer(t *testing.T) {
	words := []string{"a", "bb", "é"}
	enum := func(mask int) []string {
		values := []string{}
		for i, word := range words {
			if mask&(1<<i) != 0 {
				values = append(values, word)
			}
		}
		return values
	}
	accepted, rejected := 0, 0
	for a := 1; a < 1<<len(words); a++ {
		for b := 1; b < 1<<len(words); b++ {
			input, output := enum(a), enum(b)
			artifact, err := Compile(Definition{
				Inputs: schema.Record{
					"choice": {Kind: "bool"},
					"source": {Kind: "string", MaxBytes: 3, Enum: input},
				},
				Outputs: schema.Record{"result": {Kind: "string", MaxBytes: 2, Enum: output}},
				Steps: []Step{
					IfEqual(Var("choice"), Boolean(true), []Step{Set("result", Var("source"))},
						[]Step{Set("result", String(output[0]))}),
					Success(),
				},
			})
			if err != nil {
				if a&^b == 0 {
					t.Fatalf("Compile rejects subset %q -> %q: %v", input, output, err)
				}
				assertCompileStringFlow(t, err, false, `output "result": source string schema exceeds destination bound or enum`)
				rejected++
				continue
			}
			accepted++
			var program executor.Program
			if err := json.Unmarshal(artifact.Bytes(), &program); err != nil {
				t.Fatal(err)
			}
			for _, flag := range []bool{false, true} {
				for _, value := range input {
					state := executor.Projection{Status: "open", Data: map[string]schema.Value{
						"choice": schema.Bool(flag), "source": schema.Text(value),
					}}
					for i := 0; state.Status == "open" && i < len(program.Code); i++ {
						next, err := executor.Reduce(program, state, nil, executor.RunID{Subject: "test", Generation: 1},
							int64(i+1), executor.DefaultLimits())
						if err != nil {
							t.Fatalf("%q -> %q, choice %t with %q: %v", input, output, flag, value, err)
						}
						state = next.Projection
					}
					if state.Status != "succeeded" {
						t.Fatalf("expected Success for %q -> %q: %+v", input, output, state)
					}
				}
			}
		}
	}
	if accepted == 0 || rejected == 0 {
		t.Fatalf("expected both accepted and rejected programs, got %d and %d", accepted, rejected)
	}
}

func TestCompileImmutableStringEqualityGuards(t *testing.T) {
	field := schema.Field{Kind: "string", MaxBytes: 8, Enum: []string{"fast", "slow"}}
	inputs := schema.Record{"in": {Kind: "string", MaxBytes: 16}}
	steps := []Step{IfEqual(Var("in"), String("fast"), []Step{Set("tmp", String("fast"))}, []Step{Set("tmp", String("SENTINEL"))}), IfEqual(Var("in"), String("fast"), []Step{Set("out", Var("tmp")), Success()}, []Step{Failure()})}
	artifact, err := Compile(Definition{Inputs: inputs, Outputs: schema.Record{"out": field}, Steps: steps})
	if err != nil {
		t.Fatalf("same immutable guard should compile: %v", err)
	}
	var p executor.Program
	if err := json.Unmarshal(artifact.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"fast", "slow", "other"} {
		state := executor.Projection{Status: "open", Data: map[string]schema.Value{"in": schema.Text(input)}}
		for seq := int64(1); state.Status == "open" && seq <= int64(len(p.Code)); seq++ {
			r, err := executor.Reduce(p, state, nil, executor.RunID{Subject: "worker", Generation: 1}, seq, executor.DefaultLimits())
			if err != nil {
				t.Fatalf("input %q: %v", input, err)
			}
			state = r.Projection
		}
		want := "failed"
		if input == "fast" {
			want = "succeeded"
		}
		if state.Status != want {
			t.Fatalf("input %q: status %s, want %s", input, state.Status, want)
		}
	}
}

func TestCompileStringEqualityGuardDoesNotConflatePredicates(t *testing.T) {
	field := schema.Field{Kind: "string", MaxBytes: 8, Enum: []string{"fast"}}
	inputs := schema.Record{"in": {Kind: "string", MaxBytes: 16}}
	first := IfEqual(Var("in"), String("fast"), []Step{Set("tmp", String("fast"))}, []Step{Set("tmp", String("BAD"))})
	cases := []struct {
		name  string
		steps []Step
	}{
		{"changed input", []Step{first, Set("in", String("fast")), IfEqual(Var("in"), String("fast"), []Step{Set("out", Var("tmp")), Success()}, []Step{Failure()})}},
		{"different literal", []Step{first, IfEqual(Var("in"), String("slow"), []Step{Set("out", Var("tmp")), Success()}, []Step{Failure()})}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Definition{Inputs: inputs, Outputs: schema.Record{"out": field}, Steps: tc.steps})
			assertCompileStringFlow(t, err, false, `output "out": source string schema exceeds destination bound or enum`)
		})
	}
}

func TestCompileCollectionRowStringContainment(t *testing.T) {
	tests := []struct {
		name   string
		source schema.Field
		target schema.Field
		accept bool
	}{
		{"partial enum overlap", schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a", "b"}}, schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a"}}, false},
		{"unconstrained to enum", schema.Field{Kind: "string", MaxBytes: 4}, schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a"}}, false},
		{"larger bound", schema.Field{Kind: "string", MaxBytes: 8}, schema.Field{Kind: "string", MaxBytes: 4}, false},
		{"enum subset", schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a"}}, schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a", "b"}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := schema.Field{Kind: "list", MaxBytes: 128, MaxItems: 1, Element: schema.Record{"v": tc.source}}
			_, err := Compile(Definition{Inputs: schema.Record{"items": list}, Ports: map[string]Port{"out": {Peer: "peer", Obligation: "work", Send: schema.Record{"v": tc.target}}}, Steps: []Step{ForEach(Var("items"), map[string]string{"v": "v"}, Send("out", map[string]Expr{"v": Var("v")})), Success()}})
			assertCompileStringFlow(t, err, tc.accept, `payload "v": source string schema exceeds destination bound or enum`)
		})
	}
}
func TestCompileCorrelatedLiteralRows(t *testing.T) {
	row := schema.Record{"active": {Kind: "bool"}, "mode": {Kind: "string", MaxBytes: 16}}
	list := schema.Field{Kind: "list", MaxBytes: 512, MaxItems: 4, Element: row}
	for _, tc := range []struct {
		name        string
		activeValue string
		rebind      bool
		accept      bool
	}{{"valid", "fast", false, true}, {"invalid active row", "bad", false, false}, {"rebound guard", "fast", true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			body := []Step{}
			if tc.rebind {
				body = append(body, Set("active", Boolean(true)))
			}
			body = append(body, IfEqual(Var("active"), Boolean(true), []Step{Send("out", map[string]Expr{"mode": Var("mode")})}, nil))
			d := Definition{Collections: map[string]schema.Field{"rows": list}, Values: map[string]schema.Value{"rows": schema.List(map[string]schema.Value{"active": schema.Bool(false), "mode": schema.Text("bad")}, map[string]schema.Value{"active": schema.Bool(true), "mode": schema.Text(tc.activeValue)})}, Ports: map[string]Port{"out": {Peer: "peer", Obligation: "work", Send: schema.Record{"mode": {Kind: "string", MaxBytes: 8, Enum: []string{"fast"}}}}}, Steps: []Step{ForEach(Var("rows"), map[string]string{"active": "active", "mode": "mode"}, body...), Success()}}
			_, err := Compile(d)
			assertCompileStringFlow(t, err, tc.accept, `payload "mode": source string schema exceeds destination bound or enum`)
		})
	}
}

func TestCompileLoopExitPreservesRowBounds(t *testing.T) {
	field := schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"a", "b"}}
	list := schema.Field{Kind: "list", MaxBytes: 128, MaxItems: 2, Element: schema.Record{"v": field}}
	_, err := Compile(Definition{Inputs: schema.Record{"items": list}, Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 4, Enum: []string{"a"}}}, Values: map[string]schema.Value{"out": schema.Text("a")}, Steps: []Step{ForEach(Var("items"), map[string]string{"v": "out"}), Success()}})
	assertCompileStringFlow(t, err, false, `output "out": source string schema exceeds destination bound or enum`)
}
func TestCompileLoopGuardCannotNarrowPriorIteration(t *testing.T) {
	list := schema.Field{Kind: "list", MaxBytes: 512, MaxItems: 2, Element: schema.Record{"active": {Kind: "bool"}, "mode": {Kind: "string", MaxBytes: 8}}}
	_, err := Compile(Definition{Collections: map[string]schema.Field{"rows": list}, Values: map[string]schema.Value{"saved": schema.Text("a"), "rows": schema.List(map[string]schema.Value{"active": schema.Bool(false), "mode": schema.Text("bad")}, map[string]schema.Value{"active": schema.Bool(true), "mode": schema.Text("a")})}, Ports: map[string]Port{"out": {Peer: "peer", Obligation: "work", Send: schema.Record{"v": {Kind: "string", MaxBytes: 4, Enum: []string{"a"}}}}}, Steps: []Step{ForEach(Var("rows"), map[string]string{"active": "active", "mode": "mode"}, IfEqual(Var("active"), Boolean(true), []Step{Send("out", map[string]Expr{"v": Var("saved")})}, []Step{Set("saved", Var("mode"))})), Success()}})
	assertCompileStringFlow(t, err, false, `payload "v": source string schema exceeds destination bound or enum`)
}

func TestCompileLoopExitZeroAndMultipleRows(t *testing.T) {
	list := schema.Field{Kind: "list", MaxBytes: 512, MaxItems: 3, Element: schema.Record{"v": {Kind: "string", MaxBytes: 8}}}
	for _, tc := range []struct {
		name    string
		initial string
		rows    []map[string]schema.Value
		accept  bool
	}{
		{"empty preserves valid initial", "a", nil, true},
		{"empty preserves invalid initial", "bad", nil, false},
		{"multiple all valid", "bad", []map[string]schema.Value{{"v": schema.Text("a")}, {"v": schema.Text("a")}}, true},
		{"multiple includes invalid", "a", []map[string]schema.Value{{"v": schema.Text("a")}, {"v": schema.Text("bad")}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(Definition{Collections: map[string]schema.Field{"items": list}, Values: map[string]schema.Value{"items": schema.List(tc.rows...), "out": schema.Text(tc.initial)}, Outputs: schema.Record{"out": {Kind: "string", MaxBytes: 4, Enum: []string{"a"}}}, Steps: []Step{ForEach(Var("items"), map[string]string{"v": "out"}), Success()}})
			assertCompileStringFlow(t, err, tc.accept, `output "out":`)
		})
	}
}

func TestCompileKnownEmptyLoopPreservesHistoricalArtifact(t *testing.T) {
	rows := schema.Field{Kind: "list", MaxBytes: 256, MaxItems: 2, Element: schema.Record{"n": {Kind: "int"}}}
	artifact, err := Compile(Definition{
		Collections: map[string]schema.Field{"outer": rows, "inner": rows},
		Steps: []Step{
			ForEach(Var("outer"), nil,
				Set("copied", Var("inner")),
				ForEach(Var("copied"), nil)),
			Success(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The nested loop and list copy retain their collection metadata even
	// though the outer list was initialized empty at fdac2f8.
	const want = "a1ca6f5c97121366fb2ebb1fdce7f7093af7e4d6b3ede6df2367a01d13a1b7d2"
	if artifact.ID() != want {
		t.Fatalf("artifact ID = %s, want %s", artifact.ID(), want)
	}
}

func TestCompileLoopAdapterInvariantRequiresFailureOnInvalid(t *testing.T) {
	f := schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"ok"}}
	list := schema.Field{Kind: "list", MaxBytes: 128, MaxItems: 1, Element: schema.Record{"b": {Kind: "bool"}}}
	for _, tc := range []struct {
		name    string
		invalid []Step
		accept  bool
	}{
		{"invalid terminates", []Step{Failure()}, true},
		{"invalid continues", nil, false},
		{"invalid succeeds", []Step{Success()}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := Definition{Collections: map[string]schema.Field{"items": list}, Values: map[string]schema.Value{"items": schema.List(map[string]schema.Value{"b": schema.Bool(true)}), "out": schema.Text("ok")}, Outputs: schema.Record{"out": f}, Effects: map[string]Effect{"e": {Capability: "e", Output: schema.Record{"v": f}}}, Steps: []Step{ForEach(Var("items"), nil, Request("req", "e", nil, nil), Await("e", "req", map[string]string{"v": "out", "valid": "valid"}), IfEqual(Var("valid"), Boolean(true), nil, tc.invalid)), Success()}}
			_, err := Compile(d)
			assertCompileStringFlow(t, err, tc.accept, `output "out": source string schema exceeds destination bound or enum`)
		})
	}
}
