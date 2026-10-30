package consumingpolicy

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// NewReason constructs C_reason: consume both original outcomes, validate S,
// then classify the candidate's actual status, reason and guarded response.
func NewReason(candidate author.Template, config substitution.Config, options Options) (Deployment, error) {
	// Reuse original-template selection and validation only. The S-first
	// provision is replaced below; it is never executed.
	d, err := NewSFirst(candidate, config, options)
	if err != nil {
		return d, err
	}
	d.Parent, err = reasonParent(candidate, d.Successor)
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

func classified(tag string, terminal author.Step) []author.Step {
	return []author.Step{author.Set("classification", author.String(tag)), terminal}
}

// reasonGuard is an author-time combinator, not a runtime classifier. Every
// failed guard commits the defensive classification before deliberate Failure.
func reasonGuard(a, b author.Expr, next []author.Step) author.Step {
	return author.IfEqual(a, b, next, classified("invalid_outcome", author.Failure()))
}

func reasonParent(candidate, successor author.Template) (author.Artifact, error) {
	request := protocol.Request()
	candidateDecision := reasonGuard(author.Var("candidate_status"), author.String("failed"), []author.Step{
		reasonGuard(author.Var("candidate_present"), author.Boolean(false), []author.Step{
			author.IfEqual(author.Var("candidate_reason"), author.String("member_failed"),
				classified("candidate_failed", author.Failure()), []author.Step{
					reasonGuard(author.Var("candidate_reason"), author.String("missing_reply"),
						classified("candidate_missing_reply", author.Failure())),
				}),
		}),
	})
	candidateAnswer := reasonGuard(author.Var("candidate_reason"), author.String("member_succeeded"), []author.Step{
		reasonGuard(author.Var("candidate_present"), author.Boolean(true), []author.Step{
			reasonGuard(author.Var("candidate_id"), author.Var("request_id"), []author.Step{
				reasonGuard(author.Var("candidate_value"), author.Var("expected_value"), classified("answer", author.Success())),
			}),
		}),
	})
	// This fixed S guard dominates all candidate interpretation and every
	// potentially absent S payload read. Both joins already ran before it.
	sDecision := reasonGuard(author.Var("s_status"), author.String("succeeded"), []author.Step{
		reasonGuard(author.Var("s_reason"), author.String("member_succeeded"), []author.Step{
			reasonGuard(author.Var("s_present"), author.Boolean(true), []author.Step{
				reasonGuard(author.Var("s_id"), author.Var("request_id"), []author.Step{
					reasonGuard(author.Var("s_value"), author.Integer(0), []author.Step{
						author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
						author.IfEqual(author.Var("candidate_status"), author.String("succeeded"), []author.Step{candidateAnswer}, []author.Step{candidateDecision}),
					}),
				}),
			}),
		}),
	})
	return author.Compile(author.Definition{
		Name:        "phase1m-reason-parent",
		Inputs:      schema.Record{"request_id": request["id"], "request_value": request["value"], "request_mode": request["mode"]},
		Invocations: map[string]author.Template{"candidate": candidate, "successor": successor},
		Steps: []author.Step{
			author.Invoke("candidate_operation", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
			author.Invoke("s_operation", "successor", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil),
			author.AwaitInvocation("successor", "s_operation", author.InvocationOutcomeBindings{Status: "s_status", Reason: "s_reason", ResponsePresent: "s_present", Response: map[string]string{"id": "s_id", "value": "s_value"}}),
			author.AwaitInvocation("candidate", "candidate_operation", author.InvocationOutcomeBindings{Status: "candidate_status", Reason: "candidate_reason", ResponsePresent: "candidate_present", Response: map[string]string{"id": "candidate_id", "value": "candidate_value"}}),
			sDecision,
		},
	})
}

// DeliberateFailure is a separately named executable control, not a K1
// candidate: any single original request is consumed, then its member fails.
func DeliberateFailure() (author.Template, error) {
	artifact, err := author.Compile(author.Definition{
		Name: "phase1m-control-deliberate-failure",
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Request(), Send: protocol.Response(),
		}},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{"id": "id", "value": "value", "mode": "mode"}),
			author.Failure(),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
}
