package adaptive

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/schema"
)

type profile struct {
	payload, observation        string
	payloadField, responseField schema.Field
}

func NewScalar(leaf author.Template) (author.Template, error) {
	return build(leaf, profile{"value", "observation", schema.Field{Kind: "int"}, schema.Field{Kind: "int"}})
}

func NewEvidence(leaf author.Template) (author.Template, error) {
	in, out, err := leaf.Schemas()
	if err != nil {
		return author.Template{}, err
	}
	expectedIn, err := evidence.RequestSchema().Retain()
	if err != nil {
		return author.Template{}, err
	}
	expectedOut, err := evidence.ResultSchema().Retain()
	if err != nil {
		return author.Template{}, err
	}
	if in.ID != expectedIn.ID || out.ID != expectedOut.ID {
		return author.Template{}, fmt.Errorf("evidence leaf must retain the exact helper request/result contract")
	}
	return build(leaf, profile{"question", "disposition", schema.Field{Kind: "string", MaxBytes: 512}, evidence.ResultSchema()["disposition"]})
}

func list(n int, fields schema.Record) schema.Field {
	return schema.Field{Kind: "list", MaxItems: n, MaxBytes: 262144, Element: fields}
}
func integer() schema.Field   { return schema.Field{Kind: "int"} }
func text(n int) schema.Field { return schema.Field{Kind: "string", MaxBytes: n} }
func exprs(names ...string) map[string]author.Expr {
	out := map[string]author.Expr{}
	for _, n := range names {
		out[n] = author.Var(n)
	}
	return out
}
func bindings(names ...string) map[string]string {
	out := map[string]string{}
	for _, n := range names {
		out[n] = n
	}
	return out
}
func setBool(n string, b bool) author.Step { return author.Set(n, author.Boolean(b)) }
func setInt(n string, i int64) author.Step { return author.Set(n, author.Integer(i)) }
func empty(n string) author.Step           { return author.Set(n, author.Value(schema.List())) }
func eq(n string, v author.Expr, yes ...author.Step) author.Step {
	return author.IfEqual(author.Var(n), v, yes, nil)
}
func increment(n string) author.Step { return author.Add(n, author.Var(n), author.Integer(1)) }

func (p profile) classify() []author.Step {
	steps := []author.Step{setInt("observation", -1)}
	if p.payload == "value" {
		for i := int64(0); i <= 2; i++ {
			steps = append(steps, eq("leaf_observation", author.Integer(i), setInt("observation", i)))
		}
		return steps
	}
	for _, v := range []struct {
		s string
		n int64
	}{{"answer", 0}, {"no_match", 0}, {"clarification", 1}, {"insufficient_evidence", 2}} {
		steps = append(steps, eq("leaf_observation", author.String(v.s), setInt("observation", v.n)))
	}
	return steps
}
