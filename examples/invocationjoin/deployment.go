// Package invocationjoin builds the single-parent Phase 1L invocation experiment.
// It uses original invocation outcomes as its two required completion facts.
package invocationjoin

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Options selects an existing, separately identified S outcome-policy control.
// Empty or answer preserves the original S template.
type Options struct{ SMode string }

// Deployment retains the original templates and the newly compiled fixed parent.
type Deployment struct {
	Provision            runtime.Provision
	Candidate, Successor author.Template
	Parent               author.Artifact
	FixedRuns            int
}

// New constructs one parent parameterized only by the candidate binding. Old
// builders supply original template values; their provisions are never executed.
// Recovery opens the retained database instead of calling New again.
func New(candidate author.Template, config substitution.Config, options Options) (Deployment, error) {
	d := Deployment{Candidate: candidate, FixedRuns: 1}
	originalConfig := config
	originalConfig.Shape = "sequential"
	original, err := substitution.New(candidate, originalConfig)
	if err != nil {
		return d, err
	}
	d.Successor = original.Peer
	switch options.SMode {
	case "", "answer":
	case "fail", "absent", "pending":
		controlConfig := config
		controlConfig.Shape = "parallel"
		control, err := join.New(candidate, controlConfig, join.Options{SMode: options.SMode})
		if err != nil {
			return d, err
		}
		d.Successor = control.Peer
	default:
		return d, fmt.Errorf("unknown S control %q", options.SMode)
	}
	d.Parent, err = parent(candidate, d.Successor)
	if err != nil {
		return d, err
	}
	d.Provision, err = (author.Application{"root": {Artifact: d.Parent, Inputs: map[string]schema.Value{
		"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value), "request_mode": schema.Text(config.Request.Mode),
	}}}).Provision()
	if err != nil {
		return d, err
	}
	allowed := []string{}
	for _, t := range d.Provision.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: config.Limits}}
	return d, nil
}

func requireEqual(a, b author.Expr, next []author.Step) author.Step {
	return author.IfEqual(a, b, next, []author.Step{author.Failure()})
}

func parent(candidate, successor author.Template) (author.Artifact, error) {
	// Build from the inside out so status and presence checks dominate every read
	// of a possibly absent response. These checks follow both required awaits.
	checks := []author.Step{author.Success()}
	for _, c := range []struct{ a, b author.Expr }{
		{author.Var("s_value"), author.Integer(0)},
		{author.Var("s_id"), author.Var("request_id")},
		{author.Var("s_present"), author.Boolean(true)},
		{author.Var("s_status"), author.String("succeeded")},
		{author.Var("candidate_value"), author.Var("expected_value")},
		{author.Var("candidate_id"), author.Var("request_id")},
		{author.Var("candidate_present"), author.Boolean(true)},
		{author.Var("candidate_status"), author.String("succeeded")},
	} {
		checks = []author.Step{requireEqual(c.a, c.b, checks)}
	}
	r := protocol.Request()
	return author.Compile(author.Definition{
		Name:        "phase1l-invocation-join-parent",
		Inputs:      schema.Record{"request_id": r["id"], "request_value": r["value"], "request_mode": r["mode"]},
		Invocations: map[string]author.Template{"candidate": candidate, "successor": successor},
		Steps: []author.Step{
			author.Invoke("candidate_operation", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
			author.Invoke("s_operation", "successor", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil),
			author.AwaitInvocation("candidate", "candidate_operation", author.InvocationOutcomeBindings{Status: "candidate_status", Reason: "candidate_reason", ResponsePresent: "candidate_present", Response: map[string]string{"id": "candidate_id", "value": "candidate_value"}}),
			author.AwaitInvocation("successor", "s_operation", author.InvocationOutcomeBindings{Status: "s_status", Reason: "s_reason", ResponsePresent: "s_present", Response: map[string]string{"id": "s_id", "value": "s_value"}}),
			author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
			checks[0],
		},
	})
}
