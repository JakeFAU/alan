package substitution

import (
	"fmt"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

// Probe constructs separate experimental artifacts. They are not K1 candidates.
// weak-a/weak-b promise constant zero; p/q differ by exactly one pure Set.
// wrong-id and extra-helper are valid public programs that intentionally violate K1.
func Probe(name string) (author.Template, error) {
	switch name {
	case "weak-a", "weak-b", "p", "q", "wrong-id", "extra-helper":
	default:
		return author.Template{}, fmt.Errorf("unknown probe %q", name)
	}
	helperNeeded := name == "weak-b" || name == "p" || name == "q" || name == "extra-helper"
	zero := name == "weak-a" || name == "weak-b"
	var helper author.Template
	var err error
	if helperNeeded {
		helper, err = probeHelper(zero)
		if err != nil {
			return author.Template{}, err
		}
	}
	steps := []author.Step{author.Receive(protocol.Port, map[string]string{"id": "id", "value": "value", "mode": "mode"})}
	invocations := map[string]author.Template{}
	if name == "q" {
		steps = append(steps, author.Set("scratch", author.Integer(0)))
	}
	if helperNeeded {
		invocations["helper"] = helper
		count := 1
		if name == "extra-helper" {
			count = 2
		}
		for i := 0; i < count; i++ {
			steps = append(steps,
				author.Invoke("helper_handle", "helper", map[string]author.Expr{"id": author.Var("id"), "value": author.Var("value"), "mode": author.String("answer")}, nil),
				author.AwaitInvocation("helper", "helper_handle", author.InvocationOutcomeBindings{Status: "helper_status", Reason: "helper_reason", ResponsePresent: "helper_present", Response: map[string]string{"id": "result_id", "value": "result_value"}}),
				author.IfEqual(author.Var("helper_status"), author.String("succeeded"), nil, []author.Step{author.Failure()}))
		}
	} else {
		steps = append(steps, author.Set("result_id", author.Var("id")))
		if zero {
			steps = append(steps, author.Set("result_value", author.Integer(0)))
		} else {
			steps = append(steps, author.IfEqual(author.Var("value"), author.Integer(0), []author.Step{author.Set("result_value", author.Integer(1))}, []author.Step{author.Set("result_value", author.Integer(0))}))
		}
	}
	if name == "wrong-id" {
		steps = append(steps, author.IfEqual(author.Var("id"), author.Integer(1), []author.Step{author.Set("result_id", author.Integer(2))}, []author.Step{author.Set("result_id", author.Integer(1))}))
	}
	steps = append(steps, author.Send(protocol.Port, map[string]author.Expr{"id": author.Var("result_id"), "value": author.Var("result_value")}), author.Success())
	artifact, err := author.Compile(author.Definition{Name: "phase1i-probe-" + name, Invocations: invocations, Ports: map[string]author.Port{protocol.Port: {Peer: protocol.Peer, Obligation: protocol.Obligation, Receive: protocol.Request(), Send: protocol.Response()}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
}
func probeHelper(zero bool) (author.Template, error) {
	steps := []author.Step{author.Receive(protocol.Port, map[string]string{"id": "id", "value": "value"})}
	if zero {
		steps = append(steps, author.Set("result", author.Integer(0)))
	} else {
		steps = append(steps, author.IfEqual(author.Var("value"), author.Integer(0), []author.Step{author.Set("result", author.Integer(1))}, []author.Step{author.Set("result", author.Integer(0))}))
	}
	steps = append(steps, author.Send(protocol.Port, map[string]author.Expr{"id": author.Var("id"), "value": author.Var("result")}), author.Success())
	a, err := author.Compile(author.Definition{Name: fmt.Sprintf("phase1i-probe-helper-zero-%v", zero), Ports: map[string]author.Port{protocol.Port: {Peer: protocol.Peer, Obligation: protocol.Obligation, Receive: protocol.Request(), Send: protocol.Response()}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
}

// DelayedParallel is a separately identified concrete scheduling witness. Its
// peer executes delay bounded acyclic pure Sets before the original S Invoke.
// The original five context constructors are unchanged.
func DelayedParallel(candidate author.Template, config Config, delay int) (Deployment, error) {
	config.Shape = "parallel"
	d, err := New(candidate, config)
	if err != nil {
		return d, err
	}
	if delay < 1 || delay > 64 {
		return d, fmt.Errorf("delay outside finite fixture profile")
	}
	steps := []author.Step{}
	for i := 0; i < delay; i++ {
		steps = append(steps, author.Set("schedule_marker", author.Integer(int64(i))))
	}
	steps = append(steps, peerSteps("request_id", "request_value")...)
	delayed, err := author.Compile(author.Definition{Name: "phase1i-delayed-static-peer", Inputs: schema.Record{"request_id": {Kind: "int"}, "request_value": {Kind: "int"}}, Invocations: map[string]author.Template{"peer": d.Peer}, Steps: steps})
	if err != nil {
		return d, err
	}
	p, err := (author.Application{"peer": {Artifact: delayed, Inputs: map[string]schema.Value{"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value)}}}).Provision()
	if err != nil {
		return d, err
	}
	for _, program := range p.Programs {
		found := false
		for _, old := range d.Provision.Programs {
			if old.ID == program.ID {
				found = true
			}
		}
		if !found {
			d.Provision.Programs = append(d.Provision.Programs, program)
		}
	}
	for i, run := range d.Provision.Runs {
		if run.ID.Subject == "peer" {
			d.Provision.Runs[i] = p.Runs[0]
		}
	}
	return d, nil
}
