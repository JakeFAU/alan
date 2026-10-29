// Package b provides the helper-based Phase 1I substitution candidate.
package b

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

// New returns one candidate member that awaits one ordinary singleton helper in
// every mode. Integer membership in K1's finite domain is an environment premise.
func New() (author.Template, error) {
	helper, err := complementHelper()
	if err != nil {
		return author.Template{}, err
	}
	modeAction := author.IfEqual(author.Var("requested_mode"), author.String("answer"), []author.Step{
		author.Send(protocol.Port, map[string]author.Expr{
			"id": author.Var("helper_id"), "value": author.Var("helper_value"),
		}),
		author.Success(),
	}, []author.Step{
		author.IfEqual(author.Var("requested_mode"), author.String("fail"), []author.Step{
			author.Failure(),
		}, []author.Step{
			author.IfEqual(author.Var("requested_mode"), author.String("absent"), []author.Step{
				author.Success(),
			}, []author.Step{
				// K1 supplies no second public request. This ordinary receive
				// therefore leaves pending open without a terminal publication.
				author.Receive(protocol.Port, map[string]string{
					"id": "unexpected_id", "value": "unexpected_value", "mode": "unexpected_mode",
				}),
				author.Failure(),
			}),
		}),
	})
	artifact, err := author.Compile(author.Definition{
		Name:        "phase1i-b-helper-candidate",
		Invocations: map[string]author.Template{"complement": helper},
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Request(), Send: protocol.Response(),
		}},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{
				"id": "requested_id", "value": "requested_value", "mode": "requested_mode",
			}),
			author.Invoke("helper_operation", "complement", map[string]author.Expr{
				"id": author.Var("requested_id"), "value": author.Var("requested_value"),
				"mode": author.String("answer"),
			}, nil),
			author.AwaitInvocation("complement", "helper_operation", author.InvocationOutcomeBindings{
				Status: "helper_status", Reason: "helper_reason", ResponsePresent: "helper_present",
				Response: map[string]string{"id": "helper_id", "value": "helper_value"},
			}),
			author.IfEqual(author.Var("helper_status"), author.String("succeeded"), []author.Step{
				author.IfEqual(author.Var("helper_present"), author.Boolean(true),
					[]author.Step{modeAction}, []author.Step{author.Failure()}),
			}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{
		"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}},
	}, "entry", protocol.Port)
}

func complementHelper() (author.Template, error) {
	artifact, err := author.Compile(author.Definition{
		Name: "phase1i-b-ordinary-complement",
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Request(), Send: protocol.Response(),
		}},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{
				"id": "input_id", "value": "input_value", "mode": "input_mode",
			}),
			author.IfEqual(author.Var("input_value"), author.Integer(1), []author.Step{
				author.Set("complement_value", author.Integer(0)),
			}, []author.Step{
				author.Set("complement_value", author.Integer(1)),
			}),
			author.Send(protocol.Port, map[string]author.Expr{
				"id": author.Var("input_id"), "value": author.Var("complement_value"),
			}),
			author.Success(),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{
		"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}},
	}, "entry", protocol.Port)
}
