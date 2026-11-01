package effectfulcomposition

import (
	"fmt"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

func queryFields() schema.Record {
	r := rq.ReadOutput()
	r["valid"] = boolean()
	r["disposition"] = str(128)
	return r
}
func reportSchema() schema.Record {
	r := schema.Record{"target": str(1024), "queries": integer(), "witness": boolean(), "witness_target": str(1024), "witness_query": str(1024), "witness_adoption": str(128), "witness_mutation": integer(), "witness_total": integer(), "conflict": boolean()}
	for i := 1; i <= 2; i++ {
		q := fmt.Sprintf("query%d_", i)
		r[q+"id"] = str(1024)
		for k, f := range queryFields() {
			r[q+k] = f
		}
	}
	return r
}
func eq(reg string, want author.Expr, yes, no []author.Step) author.Step {
	return author.IfEqual(author.Var(reg), want, yes, no)
}
func setBool(reg string, v bool) author.Step { return author.Set(reg, author.Boolean(v)) }
func setText(reg, s string) author.Step      { return author.Set(reg, author.String(s)) }
func require(reg string, want author.Expr, otherwise []author.Step) author.Step {
	return eq(reg, want, nil, otherwise)
}

// A read that failed has its actual normalized envelope. It cannot supply any
// new receipt fields. A later disagreement retains the earlier witness too.
func remember(q string) author.Step {
	conflict := []author.Step{setBool("conflict", true)}
	first := []author.Step{setBool("witness", true), author.Append("witness_ids", map[string]author.Expr{"id": author.Var(q)}), author.ForEach(author.Var("witness_ids"), map[string]string{"id": "saved_query_id"}, author.Set("witness_query", author.Var("saved_query_id")))}
	var compare []author.Step
	for _, k := range []string{"target", "mutation", "total"} {
		first = append(first, author.Set("witness_"+k, author.Var(q+"_"+k)))
		compare = append(compare, require("witness_"+k, author.Var(q+"_"+k), conflict))
	}
	first = append(first, author.Set("witness_adoption", author.Var(q+"_adoption")))
	found := eq(q+"_target", author.Var("received"), []author.Step{eq("witness", author.Boolean(true), compare, first)}, conflict)
	return eq(q+"_valid", author.Boolean(true), []author.Step{
		eq(q+"_adoption", author.String("conflict"), conflict, []author.Step{
			eq(q+"_status", author.String("conflict"), conflict, []author.Step{eq(q+"_status", author.String("found"), []author.Step{found}, nil)}),
		}),
	}, nil)
}

func resolverProgram(role string, queries int) (author.Artifact, error) {
	d := author.Definition{Name: "phase2d-resolver-" + role,
		Collections: map[string]schema.Field{"witness_ids": {Kind: "list", MaxItems: 1, MaxBytes: 2048, Element: schema.Record{"id": str(1024)}}}, Ports: map[string]author.Port{
			"relay":  duplex("parent", role+"_relay", rq.ReadInput()),
			"report": {Peer: "parent", Obligation: role + "_report", Send: reportSchema(), Receive: rq.ReadInput()},
		}, Effects: map[string]author.Effect{"lookup": {Capability: rq.ReadCapability, Input: rq.ReadInput(), Output: rq.ReadOutput()}}}
	d.Steps = []author.Step{receiveTarget("relay", "received"), sendTarget("relay", "received"), setBool("witness", false), setBool("conflict", false), setText("witness_query", ""), setText("witness_target", ""), setText("witness_adoption", ""), author.Set("witness_mutation", author.Integer(0)), author.Set("witness_total", author.Integer(0))}
	report := map[string]author.Expr{"target": author.Var("received"), "queries": author.Integer(int64(queries))}
	for _, field := range []string{"witness", "witness_query", "witness_target", "witness_adoption", "witness_mutation", "witness_total", "conflict"} {
		report[field] = author.Var(field)
	}
	for i := 1; i <= 2; i++ {
		q := fmt.Sprintf("query%d", i)
		if i > queries {
			report[q+"_id"] = author.String("")
			for k, f := range queryFields() {
				report[q+"_"+k] = author.Value(schema.Default(f))
			}
			continue
		}
		port := "start"
		if i == 2 {
			port = "refresh"
		}
		d.Ports[port] = author.Port{Peer: "local", Obligation: port, Local: true, Receive: schema.Record{}}
		bindings := map[string]string{}
		for k := range queryFields() {
			bindings[k] = q + "_" + k
			report[q+"_"+k] = author.Var(q + "_" + k)
		}
		report[q+"_id"] = author.Var(q)
		d.Steps = append(d.Steps, author.Receive(port, nil), author.Request(q, "lookup", map[string]author.Expr{"target": author.Var("received")}, nil), author.Await("lookup", q, bindings), remember(q))
	}
	d.Steps = append(d.Steps, author.Send("report", report), receiveTarget("report", "report_ack"), eq("report_ack", author.Var("received"), []author.Step{author.Success()}, []author.Step{author.Failure()}))
	return author.Compile(d)
}
