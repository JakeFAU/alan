// Package consumingpolicy authors parents that retain both original invocation
// handles and apply a result policy after consuming their required outcomes.
package consumingpolicy

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Options selects a separately identified S control. Empty preserves original S.
type Options struct{ SMode string }

// Deployment retains original components and the newly authored fixed parent.
type Deployment struct {
	Provision            runtime.Provision
	Candidate, Successor author.Template
	Parent               author.Artifact
	FixedRuns            int
}

// NewSFirst constructs C_se from the behavioral contract. Public historical
// builders provide template values only; their provisions are never executed.
func NewSFirst(candidate author.Template, config substitution.Config, options Options) (Deployment, error) {
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
	d.Parent, err = sFirstParent(candidate, d.Successor)
	if err != nil {
		return d, err
	}
	d.Provision, err = (author.Application{"root": {Artifact: d.Parent, Inputs: map[string]schema.Value{
		"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value), "request_mode": schema.Text(config.Request.Mode),
	}}}).Provision()
	if err != nil {
		return d, err
	}
	allowed := make([]string, 0, len(d.Provision.Composition.Templates))
	for _, template := range d.Provision.Composition.Templates {
		allowed = append(allowed, template.ID)
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: config.Limits}}
	return d, nil
}

// validResponse nests presence and status outside payload reads. On any failed
// guard the program reaches Failure, only after both required joins have run.
func validResponse(prefix string, value author.Expr, next []author.Step) author.Step {
	failure := []author.Step{author.Failure()}
	return author.IfEqual(author.Var(prefix+"_status"), author.String("succeeded"), []author.Step{
		author.IfEqual(author.Var(prefix+"_present"), author.Boolean(true), []author.Step{
			author.IfEqual(author.Var(prefix+"_id"), author.Var("request_id"), []author.Step{
				author.IfEqual(author.Var(prefix+"_value"), value, next, failure),
			}, failure),
		}, failure),
	}, failure)
}

func sFirstParent(candidate, successor author.Template) (author.Artifact, error) {
	request := protocol.Request()
	return author.Compile(author.Definition{
		Name:        "phase1m-s-first-parent",
		Inputs:      schema.Record{"request_id": request["id"], "request_value": request["value"], "request_mode": request["mode"]},
		Invocations: map[string]author.Template{"candidate": candidate, "successor": successor},
		Steps: []author.Step{
			author.Invoke("candidate_operation", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
			author.Invoke("s_operation", "successor", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil),
			author.AwaitInvocation("successor", "s_operation", author.InvocationOutcomeBindings{Status: "s_status", Reason: "s_reason", ResponsePresent: "s_present", Response: map[string]string{"id": "s_id", "value": "s_value"}}),
			author.AwaitInvocation("candidate", "candidate_operation", author.InvocationOutcomeBindings{Status: "candidate_status", Reason: "candidate_reason", ResponsePresent: "candidate_present", Response: map[string]string{"id": "candidate_id", "value": "candidate_value"}}),
			author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
			validResponse("candidate", author.Var("expected_value"), []author.Step{
				validResponse("s", author.Integer(0), []author.Step{author.Success()}),
			}),
		},
	})
}
