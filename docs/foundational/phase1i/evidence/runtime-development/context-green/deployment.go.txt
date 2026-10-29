// Package substitution builds the five finite Phase 1I experimental contexts.
// It is an ordinary public-API consumer, not a runtime orchestration facility.
package substitution

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type Request struct {
	ID, Value int64
	Mode      string
}

type Config struct {
	Shape   string
	Request Request
	Limits  runtime.ResourceLimits
}

type Deployment struct {
	Provision              runtime.Provision
	Candidate, Entry, Peer author.Template
	FixedRuns              int
}

// New provisions a fresh context; recovery only opens its retained store.
func New(candidate author.Template, config Config) (Deployment, error) {
	d := Deployment{Candidate: candidate, Entry: candidate, FixedRuns: 1}
	switch config.Shape {
	case "bare", "hidden", "sequential", "parallel", "wrapped":
	default:
		return d, fmt.Errorf("unknown Phase 1I context %q", config.Shape)
	}
	if config.Request.ID < 1 || config.Request.ID > 2 || config.Request.Value < 0 || config.Request.Value > 1 {
		return d, fmt.Errorf("request is outside the Phase 1I finite input profile")
	}
	var err error
	if config.Shape == "wrapped" {
		d.Entry, err = wrapper(candidate)
		if err != nil {
			return d, err
		}
	}
	if config.Shape == "sequential" || config.Shape == "parallel" {
		d.Peer, err = successor()
		if err != nil {
			return d, err
		}
	}
	root, err := caller(d.Entry, d.Peer, config.Shape)
	if err != nil {
		return d, err
	}
	input := map[string]schema.Value{
		"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value),
		"request_mode": schema.Text(config.Request.Mode),
	}
	app := author.Application{"root": {Artifact: root, Inputs: input}}
	members := []runtime.RunID{{Subject: "root", Generation: 1}}
	if config.Shape == "parallel" {
		peer, err := peerCaller(d.Peer)
		if err != nil {
			return d, err
		}
		app["peer"] = author.Instance{Artifact: peer, Inputs: map[string]schema.Value{
			"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value),
		}}
		members = append(members, runtime.RunID{Subject: "peer", Generation: 1})
		d.FixedRuns++
	}
	d.Provision, err = app.Provision()
	if err != nil {
		return d, err
	}
	allowed := []string{}
	for _, template := range d.Provision.Composition.Templates {
		allowed = append(allowed, template.ID)
	}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{
		Run: runtime.RunID{Subject: "root", Generation: 1}, Members: members,
		Templates: allowed, Limits: config.Limits,
	}}
	return d, nil
}

func requestExpressions() map[string]author.Expr {
	return map[string]author.Expr{
		"id": author.Var("request_id"), "value": author.Var("request_value"),
		"mode": author.Var("request_mode"),
	}
}

func requestInputs() schema.Record {
	r := protocol.Request()
	return schema.Record{
		"request_id": r["id"], "request_value": r["value"], "request_mode": r["mode"],
	}
}

func candidateAwait() author.Step {
	return author.AwaitInvocation("candidate", "pending", author.InvocationOutcomeBindings{
		Status: "child_status", Reason: "child_reason", ResponsePresent: "child_present",
		Response: map[string]string{"id": "answer_id", "value": "answer_value"},
	})
}

func onPresentSuccess(status, present string, next []author.Step) author.Step {
	return author.IfEqual(author.Var(status), author.String("succeeded"),
		[]author.Step{author.IfEqual(author.Var(present), author.Boolean(true),
			next, []author.Step{author.Failure()})},
		[]author.Step{author.Failure()})
}

func peerSteps(id, value string) []author.Step {
	return []author.Step{
		author.Invoke("peer_pending", "peer", map[string]author.Expr{
			"id": author.Var(id), "value": author.Var(value),
		}, nil),
		author.AwaitInvocation("peer", "peer_pending", author.InvocationOutcomeBindings{
			Status: "peer_status", Reason: "peer_reason", ResponsePresent: "peer_present",
			Response: map[string]string{"id": "peer_id", "value": "peer_value"},
		}),
		onPresentSuccess("peer_status", "peer_present", []author.Step{author.Success()}),
	}
}

func caller(candidate, peer author.Template, shape string) (author.Artifact, error) {
	invocations := map[string]author.Template{"candidate": candidate}
	next := []author.Step{author.Success()}
	if shape == "sequential" {
		invocations["peer"] = peer
		next = peerSteps("answer_id", "answer_value")
	}
	return author.Compile(author.Definition{
		Inputs: requestInputs(), Invocations: invocations,
		Steps: []author.Step{
			author.Invoke("pending", "candidate", requestExpressions(), nil),
			candidateAwait(),
			onPresentSuccess("child_status", "child_present", next),
		},
	})
}

func peerCaller(peer author.Template) (author.Artifact, error) {
	return author.Compile(author.Definition{
		Inputs:      schema.Record{"request_id": {Kind: "int"}, "request_value": {Kind: "int"}},
		Invocations: map[string]author.Template{"peer": peer},
		Steps:       peerSteps("request_id", "request_value"),
	})
}

func wrapper(candidate author.Template) (author.Template, error) {
	a, err := author.Compile(author.Definition{
		Invocations: map[string]author.Template{"candidate": candidate},
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Request(), Send: protocol.Response(),
		}},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{
				"id": "request_id", "value": "request_value", "mode": "request_mode",
			}),
			author.Invoke("pending", "candidate", requestExpressions(), nil),
			candidateAwait(),
			onPresentSuccess("child_status", "child_present", []author.Step{
				author.Send(protocol.Port, map[string]author.Expr{
					"id": author.Var("answer_id"), "value": author.Var("answer_value"),
				}),
				author.Success(),
			}),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(
		author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}},
		"entry", protocol.Port,
	)
}

func successor() (author.Template, error) {
	a, err := author.Compile(author.Definition{
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Response(), Send: protocol.Response(),
		}},
		Steps: []author.Step{
			author.Receive(protocol.Port, map[string]string{"id": "id"}),
			author.Send(protocol.Port, map[string]author.Expr{"id": author.Var("id"), "value": author.Integer(0)}),
			author.Success(),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(
		author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}},
		"entry", protocol.Port,
	)
}
