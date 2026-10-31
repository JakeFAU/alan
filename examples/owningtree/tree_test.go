package owningtree_test

import (
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecursivePublicTypes(t *testing.T) {
	c, err := a.New()
	check(t, err)
	leaf, err := owningtree.Leaf(c)
	check(t, err)
	historical, err := callablejoin.New(c, substitution.Config{Request: substitution.Request{ID: 1, Mode: "answer"}})
	check(t, err)
	if leaf.ID() != historical.Join.ID() {
		t.Fatal("leaf identity")
	}
	pair, err := owningtree.Fork(leaf, leaf)
	check(t, err)
	relay, err := owningtree.Relay(pair)
	check(t, err)
	nested, err := owningtree.Fork(relay, leaf)
	check(t, err)
	in, out, err := nested.Schemas()
	check(t, err)
	req := map[string]schema.Value{}
	for _, p := range []string{"l_l_", "l_r_", "r_"} {
		req[p+"id"] = schema.Int(1)
		req[p+"value"] = schema.Int(1)
		req[p+"mode"] = schema.Text("absent")
		if out.Record[p+"answers"].MaxItems != 1 || out.Record[p+"answers"].MaxBytes != 192 {
			t.Fatal("answer list", p, out.Record)
		}
	}
	check(t, in.Record.Validate(req))
	if len(out.Record) != 9 {
		t.Fatal(out.Record)
	}
	prov, err := owningtree.Provision(nested, req, runtime.ResourceLimits{MaxInvocations: 15, MaxCreatedRuns: 15, MaxDepth: 6, MaxInFlight: 1})
	check(t, err)
	if len(prov.Runs) != 1 || prov.Runs[0].ID != (runtime.RunID{Subject: "root", Generation: 1}) {
		t.Fatal("outer root", prov.Runs)
	}
	req["l_l_value"] = schema.Int(2)
	if _, err = owningtree.Provision(nested, req, runtime.ResourceLimits{}); err == nil {
		t.Fatal("accepted non-Boolean input")
	}
	if _, err = owningtree.Relay(c); err == nil {
		t.Fatal("accepted scalar candidate as a reporting tree")
	}
	if _, err = owningtree.Outer(author.Template{}); err == nil {
		t.Fatal("accepted empty template")
	}
}

// A narrower mode enum would fail lawful requests after admission. Kind
// compatibility at Invoke does not establish the full frozen leaf boundary.
func TestLeafRejectsNarrowedCandidate(t *testing.T) {
	req := protocol.Request()
	req["mode"] = schema.Field{Kind: "string", MaxBytes: 8, Enum: []string{"answer"}}
	art, err := author.Compile(author.Definition{Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "substitution", Receive: req, Send: protocol.Response()}},
		Steps: []author.Step{author.Receive("public", nil), author.Send("public", map[string]author.Expr{"id": author.Integer(1), "value": author.Integer(0)}), author.Success()}})
	check(t, err)
	c, err := author.NewTemplate(author.Application{"entry": {Artifact: art, Inputs: map[string]schema.Value{}}}, "entry", "public")
	check(t, err)
	if _, err := owningtree.Leaf(c); err == nil {
		t.Fatal("accepted narrower mode enum")
	}
}
