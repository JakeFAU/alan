package callablejoin

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

const PublicPort = "public"
const AnswerListBytes = 192

// Report is a closed diagnostic. MaxItems is an upper bound; O separately
// checks exact cardinality. Integer kinds retain the original schemas.
func Report() schema.Record {
	return schema.Record{
		"id":             {Kind: "int"},
		"classification": {Kind: "string", MaxBytes: 32, Enum: []string{"answer", "candidate_failed", "candidate_missing_reply", "invalid_outcome"}},
		"answers":        {Kind: "list", MaxItems: 1, MaxBytes: AnswerListBytes, Element: protocol.Response()},
	}
}

func classified(tag string) []author.Step {
	return []author.Step{author.Set("classification", author.String(tag))}
}

func joinGuard(a, b author.Expr, next []author.Step) author.Step {
	return author.IfEqual(a, b, next, classified("invalid_outcome"))
}

// NewJoin constructs J directly, with a public request and invocation-bound
// reply. Its classifier only inspects the two actual original consumptions.
func NewJoin(candidate, successor author.Template) (author.Artifact, author.Template, error) {
	failure := joinGuard(author.Var("candidate_status"), author.String("failed"), []author.Step{
		joinGuard(author.Var("candidate_present"), author.Boolean(false), []author.Step{
			author.IfEqual(author.Var("candidate_reason"), author.String("member_failed"), classified("candidate_failed"), []author.Step{
				joinGuard(author.Var("candidate_reason"), author.String("missing_reply"), classified("candidate_missing_reply")),
			}),
		}),
	})
	answer := joinGuard(author.Var("candidate_reason"), author.String("member_succeeded"), []author.Step{
		joinGuard(author.Var("candidate_present"), author.Boolean(true), []author.Step{
			joinGuard(author.Var("candidate_id"), author.Var("request_id"), []author.Step{
				joinGuard(author.Var("candidate_value"), author.Var("expected_value"), []author.Step{
					author.Append("answers", map[string]author.Expr{"id": author.Var("candidate_id"), "value": author.Var("candidate_value")}),
					author.Set("classification", author.String("answer")),
				}),
			}),
		}),
	})
	policy := joinGuard(author.Var("s_status"), author.String("succeeded"), []author.Step{
		joinGuard(author.Var("s_reason"), author.String("member_succeeded"), []author.Step{
			joinGuard(author.Var("s_present"), author.Boolean(true), []author.Step{
				joinGuard(author.Var("s_id"), author.Var("request_id"), []author.Step{
					joinGuard(author.Var("s_value"), author.Integer(0), []author.Step{
						author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
						author.IfEqual(author.Var("candidate_status"), author.String("succeeded"), []author.Step{answer}, []author.Step{failure}),
					}),
				}),
			}),
		}),
	})
	artifact, err := author.Compile(author.Definition{
		Name: "phase1n-callable-join", Invocations: map[string]author.Template{"candidate": candidate, "successor": successor},
		Collections: map[string]schema.Field{"answers": Report()["answers"]},
		Ports:       map[string]author.Port{PublicPort: {Peer: "caller", Obligation: "work", Receive: protocol.Request(), Send: Report()}},
		Steps: []author.Step{
			author.Receive(PublicPort, map[string]string{"id": "request_id", "value": "request_value", "mode": "request_mode"}),
			author.Invoke("candidate_operation", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
			author.Invoke("s_operation", "successor", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil),
			author.AwaitInvocation("successor", "s_operation", author.InvocationOutcomeBindings{Status: "s_status", Reason: "s_reason", ResponsePresent: "s_present", Response: map[string]string{"id": "s_id", "value": "s_value"}}),
			author.AwaitInvocation("candidate", "candidate_operation", author.InvocationOutcomeBindings{Status: "candidate_status", Reason: "candidate_reason", ResponsePresent: "candidate_present", Response: map[string]string{"id": "candidate_id", "value": "candidate_value"}}),
			policy,
			author.Send(PublicPort, map[string]author.Expr{"id": author.Var("request_id"), "classification": author.Var("classification"), "answers": author.Var("answers")}),
			author.IfEqual(author.Var("classification"), author.String("answer"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		return artifact, author.Template{}, err
	}
	template, err := author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", PublicPort)
	return artifact, template, err
}
