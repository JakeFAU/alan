package callablejoin

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

func decided(tag string) []author.Step {
	return []author.Step{author.Set("decision", author.String(tag))}
}
func outerGuard(a, b author.Expr, next []author.Step) author.Step {
	return author.IfEqual(a, b, next, decided("invalid_join_outcome"))
}

// diagnosticDecision checks the selected envelope and sets the decision register.
// It has no access to the requested mode or the child's inner handles.
func diagnosticDecision() author.Step {
	answer := outerGuard(author.Var("join_status"), author.String("succeeded"), []author.Step{
		outerGuard(author.Var("join_reason"), author.String("member_succeeded"), []author.Step{
			outerGuard(author.Var("answer_seen"), author.Boolean(true), []author.Step{
				outerGuard(author.Var("answers_valid"), author.Boolean(true), decided("answer")),
			}),
		}),
	})
	failureTags := author.IfEqual(author.Var("report_classification"), author.String("candidate_failed"), decided("candidate_failed"), []author.Step{
		author.IfEqual(author.Var("report_classification"), author.String("candidate_missing_reply"), decided("candidate_missing_reply"), []author.Step{
			outerGuard(author.Var("report_classification"), author.String("invalid_outcome"), decided("invalid_outcome")),
		}),
	})
	failure := outerGuard(author.Var("join_status"), author.String("failed"), []author.Step{
		outerGuard(author.Var("join_reason"), author.String("member_failed"), []author.Step{
			outerGuard(author.Var("answer_seen"), author.Boolean(false), []author.Step{failureTags}),
		}),
	})
	invalidAnswer := []author.Step{author.Set("answers_valid", author.Boolean(false))}
	return outerGuard(author.Var("join_present"), author.Boolean(true), []author.Step{
		outerGuard(author.Var("report_id"), author.Var("request_id"), []author.Step{
			author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
			author.Set("answer_seen", author.Boolean(false)),
			author.Set("answers_valid", author.Boolean(true)),
			author.ForEach(author.Var("report_answers"), map[string]string{"id": "answer_id", "value": "answer_value"},
				author.IfEqual(author.Var("answer_seen"), author.Boolean(false), nil, invalidAnswer),
				author.Set("answer_seen", author.Boolean(true)),
				author.IfEqual(author.Var("answer_id"), author.Var("request_id"), []author.Step{
					author.IfEqual(author.Var("answer_value"), author.Var("expected_value"), nil, invalidAnswer),
				}, invalidAnswer)),
			author.IfEqual(author.Var("report_classification"), author.String("answer"), []author.Step{answer}, []author.Step{failure}),
		}),
	})
}

// NewOuter consumes its original J handle. All diagnostic reads are dominated
// by presence; answer-record reads occur only within the bounded list body.
func NewOuter(join author.Template) (author.Artifact, error) {
	request := protocol.Request()
	return author.Compile(author.Definition{
		Name: "phase1n-outer-caller", Invocations: map[string]author.Template{"join": join},
		Inputs: schema.Record{"request_id": request["id"], "request_value": request["value"], "request_mode": request["mode"]},
		Steps: []author.Step{
			author.Invoke("join_operation", "join", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
			author.AwaitInvocation("join", "join_operation", author.InvocationOutcomeBindings{Status: "join_status", Reason: "join_reason", ResponsePresent: "join_present", Response: map[string]string{"id": "report_id", "classification": "report_classification", "answers": "report_answers"}}),
			diagnosticDecision(),
			author.IfEqual(author.Var("decision"), author.String("answer"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
}
