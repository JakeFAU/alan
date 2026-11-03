//go:build integration

package durabletest

import (
	"fmt"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func target() schema.Record { return receiptquery.ReadInput() }
func issuance() schema.Record {
	return schema.Record{"target": {Kind: "string", MaxBytes: 1024}, "issued": {Kind: "bool"}}
}
func counter() author.Effect {
	return author.Effect{Capability: receiptquery.CounterCapability, Input: receiptquery.CounterInput(), Output: receiptquery.CounterOutput()}
}

func issuingSource() (author.Artifact, error) {
	d := author.Definition{Name: "phase2c-issuing-source", Effects: map[string]author.Effect{"increment": counter()},
		Ports: map[string]author.Port{
			"handoff": {Peer: "caller", Obligation: "identity", Send: target(), Receive: target()},
			"entry":   {Peer: "caller", Obligation: "issue", Send: issuance(), Receive: schema.Record{}},
		},
		Steps: []author.Step{
			author.Receive("entry", nil),
			author.Request("original", "increment", map[string]author.Expr{"delta": author.Integer(1)}, nil),
			author.Send("entry", map[string]author.Expr{"target": author.Var("original"), "issued": author.Boolean(true)}),
			author.Send("handoff", map[string]author.Expr{"target": author.Var("original")}),
			author.Receive("handoff", map[string]string{"target": "echo"}),
			author.Success(),
		},
	}
	return author.Compile(d)
}

func receipt() schema.Record { return receiptquery.ReadOutput() }

func resolverProgram(queries int) (author.Artifact, error) {
	ports := map[string]author.Port{
		"relay":  {Peer: "parent", Obligation: "resolver-target", Receive: target(), Send: target()},
		"report": {Peer: "parent", Obligation: "receipt", Send: receipt(), Receive: target()},
		"start":  {Peer: "local", Local: true, Obligation: "start", Receive: schema.Record{}},
	}
	steps := []author.Step{
		author.Receive("relay", map[string]string{"target": "received"}),
		author.Send("relay", map[string]author.Expr{"target": author.Var("received")}),
	}
	var report map[string]author.Expr
	for i := 1; i <= queries; i++ {
		port := "start"
		if i == 2 {
			port = "refresh"
			ports[port] = author.Port{Peer: "local", Local: true, Obligation: port, Receive: schema.Record{}}
		}
		name := fmt.Sprintf("query%d", i)
		bindings := map[string]string{"valid": name + "_valid", "disposition": name + "_disposition"}
		report = map[string]author.Expr{}
		for field := range receipt() {
			bindings[field] = name + "_" + field
			report[field] = author.Var(name + "_" + field)
		}
		steps = append(steps, author.Receive(port, nil), author.Request(name, "lookup", map[string]author.Expr{"target": author.Var("received")}, nil), author.Await("lookup", name, bindings))
	}
	steps = append(steps, author.Send("report", report), author.Receive("report", map[string]string{"target": "report_ack"}), author.Success())
	return author.Compile(author.Definition{Name: "phase2c-resolver", Effects: map[string]author.Effect{"lookup": {Capability: receiptquery.ReadCapability, Input: target(), Output: receipt()}}, Ports: ports, Steps: steps})
}

func parentProgram(child author.Template) (author.Artifact, error) {
	ports := map[string]author.Port{
		"handoff": {Peer: runtime.HandoffPeer, Obligation: "identity", Send: target(), Receive: target()},
		"relay":   {Peer: "resolver", Obligation: "resolver-target", Send: target(), Receive: target()},
		"report":  {Peer: "resolver", Obligation: "receipt", Receive: receipt(), Send: target()},
	}
	report := map[string]string{}
	for field := range receipt() {
		report[field] = "receipt_" + field
	}
	steps := []author.Step{
		author.Invoke("invocation", "issue", nil, nil),
		author.Receive("handoff", map[string]string{"target": "original"}),
		author.Send("relay", map[string]author.Expr{"target": author.Var("original")}),
		author.Receive("relay", map[string]string{"target": "resolver_ack"}),
		author.IfEqual(author.Var("resolver_ack"), author.Var("original"), nil, []author.Step{author.Failure()}),
		author.Send("handoff", map[string]author.Expr{"target": author.Var("resolver_ack")}),
		author.AwaitInvocation("issue", "invocation", author.InvocationOutcomeBindings{Status: "source_status", Reason: "source_reason", ResponsePresent: "issued_present", Response: map[string]string{"issued": "issued", "target": "issued_target"}}),
		author.Receive("report", report),
		author.Send("report", map[string]author.Expr{"target": author.Var("receipt_target")}),
	}
	success := []author.Step{author.Set("decision", author.String("applied")), author.Success()}
	failure := []author.Step{author.Set("decision", author.String("applied_after_original_failure")), author.Success()}
	checks := []author.Step{author.IfEqual(author.Var("source_status"), author.String("succeeded"), []author.Step{
		author.IfEqual(author.Var("issued"), author.Boolean(true), []author.Step{
			author.IfEqual(author.Var("receipt_adopted"), author.Boolean(true), success, []author.Step{author.Failure()}),
		}, []author.Step{author.Failure()}),
	}, []author.Step{author.IfEqual(author.Var("source_status"), author.String("failed"), failure, []author.Step{author.Failure()})})}
	for _, check := range []struct{ a, b author.Expr }{
		{author.Var("issued_target"), author.Var("original")}, {author.Var("issued_present"), author.Boolean(true)},
		{author.Var("receipt_mutation"), author.Integer(1)}, {author.Var("receipt_total"), author.Integer(1)},
		{author.Var("receipt_status"), author.String("found")}, {author.Var("receipt_target"), author.Var("original")},
	} {
		checks = []author.Step{author.IfEqual(check.a, check.b, checks, []author.Step{author.Failure()})}
	}
	steps = append(steps, checks...)
	return author.Compile(author.Definition{Name: "phase2c-parent", Ports: ports, Invocations: map[string]author.Template{"issue": child}, Steps: steps})
}

// reconciliationProvision builds the retained source, parent and resolver used by the process tests.
func reconciliationProvision(queries int) (runtime.Provision, error) {
	a, err := issuingSource()
	if err != nil {
		return runtime.Provision{}, err
	}
	parent := runtime.RunID{Subject: "parent", Generation: 1}
	resolver := runtime.RunID{Subject: "resolver", Generation: 1}
	t, err := author.NewHandoffTemplate(author.Application{"source": {Artifact: a, Inputs: map[string]schema.Value{}}}, "source", "entry", runtime.InvocationHandoff{Caller: parent, Sequence: 1, CallerPort: "handoff", EntryPort: "handoff"})
	if err != nil {
		return runtime.Provision{}, err
	}
	par, err := parentProgram(t)
	if err != nil {
		return runtime.Provision{}, err
	}
	read, err := resolverProgram(queries)
	if err != nil {
		return runtime.Provision{}, err
	}
	p, err := (author.Application{"parent": {Artifact: par}, "resolver": {Artifact: read}}).Provision()
	if err != nil {
		return p, err
	}
	p.Composition.Roots = []runtime.ResourceRoot{
		{Run: parent, Members: []runtime.RunID{parent}, Templates: []string{t.ID()}, Capabilities: []string{receiptquery.CounterCapability}, Limits: runtime.ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 1, MaxInFlight: 1}},
		{Run: resolver, Members: []runtime.RunID{resolver}, Capabilities: []string{receiptquery.ReadCapability}, Limits: runtime.ResourceLimits{MaxEffects: int64(queries), MaxInFlight: 1}},
	}
	return p, nil
}
