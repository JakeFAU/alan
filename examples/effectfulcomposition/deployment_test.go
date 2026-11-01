package effectfulcomposition_test

import (
	"bytes"
	"testing"

	frame "github.com/JakeFAU/alan/examples/effectfulcomposition"
	"github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/schema"
)

func TestFrameArtifacts(t *testing.T) {
	tree, err := frame.PureFork("A", "B", "above")
	if err != nil {
		t.Fatal(err)
	}
	input := map[string]schema.Value{"l_id": schema.Int(1), "r_id": schema.Int(1), "l_value": schema.Int(0), "r_value": schema.Int(1), "l_mode": schema.Text("answer"), "r_mode": schema.Text("absent")}
	o := frame.Options{Tree: tree, Inputs: input, Left: frame.RoleOptions{Queries: 2}, Right: frame.RoleOptions{ObserveOriginal: true}, Endpoint: "http://127.0.0.1:1234/phase2c/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/apply"}
	d, err := frame.New(o)
	if err != nil {
		t.Fatal(err)
	}
	ac, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	bc, err := b.New()
	if err != nil {
		t.Fatal(err)
	}
	l, err := owningtree.Leaf(ac)
	if err != nil {
		t.Fatal(err)
	}
	r, err := owningtree.Leaf(bc)
	if err != nil {
		t.Fatal(err)
	}
	f, err := owningtree.Fork(l, r)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := owningtree.Relay(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(d.Tree.Bytes(), exact.Bytes()) || tree.Slots() != 10 || tree.Depth() != 5 || len(d.Provision.Runs) != 3 {
		t.Fatal("pure bytes or frame bounds")
	}
	decl := d.Parent.Metadata().Invocations
	if decl["tree"].TemplateID != exact.ID() || decl["l_issue"].TemplateID != d.Left.ID() || decl["r_issue"].TemplateID != d.Right.ID() || d.Left.ID() == d.Right.ID() {
		t.Fatal("missing distinct caller bindings")
	}
	for i, g := range d.Provision.Reconciliation.Grants {
		if g.Invocation != []string{`["parent",1,1,0]`, `["parent",1,2,0]`}[i] || g.Member != "source" || g.Sequence != 2 || g.Endpoint != o.Endpoint {
			t.Fatal(g)
		}
	}
}
