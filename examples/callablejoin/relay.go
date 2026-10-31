package callablejoin

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

// NewRelay constructs a single-child diagnostic caller. It checks the envelope
// delivered to its own Await and sends the selected report on its own reply port.
func NewRelay(child author.Template) (author.Template, error) {
	invalid := []author.Step{
		author.Send(PublicPort, map[string]author.Expr{
			"id": author.Var("request_id"), "classification": author.String("invalid_outcome"),
			"answers": author.Var("empty_answers"),
		}),
		author.Failure(),
	}
	forward := []author.Step{
		author.Send(PublicPort, map[string]author.Expr{
			"id": author.Var("report_id"), "classification": author.Var("report_classification"),
			"answers": author.Var("report_answers"),
		}),
		author.IfEqual(author.Var("decision"), author.String("answer"),
			[]author.Step{author.Success()}, []author.Step{author.Failure()}),
	}
	artifact, err := author.Compile(author.Definition{
		Name:        "phase1r-diagnostic-relay",
		Invocations: map[string]author.Template{"child": child},
		Collections: map[string]schema.Field{"empty_answers": Report()["answers"]},
		Ports: map[string]author.Port{PublicPort: {
			Peer: "caller", Obligation: "work", Receive: protocol.Request(), Send: Report(),
		}},
		Steps: []author.Step{
			author.Receive(PublicPort, map[string]string{
				"id": "request_id", "value": "request_value", "mode": "request_mode",
			}),
			author.Invoke("child_operation", "child", map[string]author.Expr{
				"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode"),
			}, nil),
			author.AwaitInvocation("child", "child_operation", author.InvocationOutcomeBindings{
				Status: "join_status", Reason: "join_reason", ResponsePresent: "join_present",
				Response: map[string]string{"id": "report_id", "classification": "report_classification", "answers": "report_answers"},
			}),
			diagnosticDecision(),
			author.IfEqual(author.Var("decision"), author.String("invalid_join_outcome"), invalid, forward),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", PublicPort)
}
