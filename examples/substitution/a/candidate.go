// Package a authors the local Phase 1I candidate.
package a

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

// New retains a single member with no helper invocation or external effect.
// The experiment supplies id in {1,2} and value in {0,1}; integer schemas
// themselves do not enforce those finite contract domains.
func New() (author.Template, error) {
	entry, err := author.Compile(author.Definition{
		Name: "phase1i-local-a",
		Ports: map[string]author.Port{
			protocol.Port: {
				Peer: protocol.Peer, Obligation: protocol.Obligation,
				Receive: protocol.Request(), Send: protocol.Response(),
			},
		},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{
				"id": "request_id", "value": "request_value", "mode": "request_mode",
			}),
			author.IfEqual(
				author.Var("request_mode"),
				author.String("answer"),
				[]author.Step{
					author.IfEqual(
						author.Var("request_value"),
						author.Integer(0),
						[]author.Step{author.Send(protocol.Port, map[string]author.Expr{
							"id": author.Var("request_id"), "value": author.Integer(1),
						})},
						[]author.Step{author.Send(protocol.Port, map[string]author.Expr{
							"id": author.Var("request_id"), "value": author.Integer(0),
						})},
					),
					author.Success(),
				},
				[]author.Step{author.IfEqual(
					author.Var("request_mode"),
					author.String("fail"),
					[]author.Step{author.Failure()},
					[]author.Step{author.IfEqual(
						author.Var("request_mode"),
						author.String("absent"),
						[]author.Step{author.Success()},
						[]author.Step{
							// The declared environment supplies no second request.
							author.Receive(protocol.Port, map[string]string{}),
							author.Success(),
						},
					)},
				)},
			),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{
		"entry": {Artifact: entry, Inputs: map[string]schema.Value{}},
	}, "entry", protocol.Port)
}
