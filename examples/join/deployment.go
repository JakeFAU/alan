// Package join builds the ordinary retained Phase 1J join experiment.
// It uses public authoring and runtime APIs; the peer's report is an assertion,
// and Send commitment does not guarantee source authorization or delivery.
package join

import (
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Options select separately identified adverse controls. Empty options preserve S.
type Options struct {
	SMode           string // empty/answer: original S; fail, absent, pending: control artifacts
	PrematureReport bool   // report after S Invoke, before consuming its outcome
	WrongRequest    bool   // report the other request id in the finite profile
}

// New provisions two static runs and one request per generation. Recovery opens
// the original store rather than calling New again.
func New(candidate author.Template, config substitution.Config, options Options) (substitution.Deployment, error) {
	if config.Shape == "" {
		config.Shape = "parallel"
	}
	if config.Shape != "parallel" {
		return substitution.Deployment{}, fmt.Errorf("join requires parallel context")
	}
	// This is the public source of the original S artifact, never a reconstruction.
	d, err := substitution.New(candidate, config)
	if err != nil {
		return d, err
	}
	switch options.SMode {
	case "", "answer":
	case "fail", "absent", "pending":
		d.Peer, err = controlS(options.SMode)
		if err != nil {
			return d, err
		}
	default:
		return d, fmt.Errorf("unknown S control %q", options.SMode)
	}
	root, err := rootCaller(candidate)
	if err != nil {
		return d, err
	}
	peer, err := peerCaller(d.Peer, options)
	if err != nil {
		return d, err
	}
	d.Provision, err = (author.Application{
		"root": {Artifact: root, Inputs: map[string]schema.Value{"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value), "request_mode": schema.Text(config.Request.Mode)}},
		"peer": {Artifact: peer, Inputs: map[string]schema.Value{"request_id": schema.Int(config.Request.ID), "request_value": schema.Int(config.Request.Value)}},
	}).Provision()
	if err != nil {
		return d, err
	}
	allowed := []string{}
	for _, t := range d.Provision.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}, {Subject: "peer", Generation: 1}}, Templates: allowed, Limits: config.Limits}}
	return d, nil
}

func reportSchema() schema.Record {
	return schema.Record{
		"request_id": {Kind: "int"}, "s_operation": {Kind: "string", MaxBytes: 256},
		"status": {Kind: "string", MaxBytes: 20, Enum: []string{"succeeded", "failed", "cancelled", "abandoned", "rejected"}},
		"reason": {Kind: "string", MaxBytes: 4096}, "response_present": {Kind: "bool"},
		"response_id": {Kind: "int"}, "response_value": {Kind: "int"},
	}
}
func joinReceive() author.Step {
	return author.Receive("join", map[string]string{"request_id": "join_request_id", "s_operation": "join_s_operation", "status": "join_status", "reason": "join_reason", "response_present": "join_present", "response_id": "join_id", "response_value": "join_value"})
}
func requireEqual(a, b author.Expr, next []author.Step) author.Step {
	return author.IfEqual(a, b, next, []author.Step{author.Failure()})
}
func rootCaller(candidate author.Template) (author.Artifact, error) {
	checks := []author.Step{author.Success()}
	for _, c := range []struct{ a, b author.Expr }{
		{author.Var("join_value"), author.Integer(0)},
		{author.Var("join_id"), author.Var("request_id")},
		{author.Var("join_present"), author.Boolean(true)},
		{author.Var("join_status"), author.String("succeeded")},
		{author.Var("answer_value"), author.Var("expected_value")},
		{author.Var("answer_id"), author.Var("request_id")},
		{author.Var("child_present"), author.Boolean(true)},
		{author.Var("child_status"), author.String("succeeded")},
	} {
		checks = []author.Step{requireEqual(c.a, c.b, checks)}
	}
	r := protocol.Request()
	return author.Compile(author.Definition{Name: "phase1j-join-root", Inputs: schema.Record{"request_id": r["id"], "request_value": r["value"], "request_mode": r["mode"]}, Invocations: map[string]author.Template{"candidate": candidate}, Ports: map[string]author.Port{"join": {Peer: "peer", Obligation: "phase1j_s_join", Receive: reportSchema()}}, Steps: []author.Step{
		author.Invoke("pending", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
		author.AwaitInvocation("candidate", "pending", author.InvocationOutcomeBindings{Status: "child_status", Reason: "child_reason", ResponsePresent: "child_present", Response: map[string]string{"id": "answer_id", "value": "answer_value"}}),
		joinReceive(),
		// Wrong correlation consumes an assertion but cannot discharge this obligation.
		// The frozen one-report control then remains at this additional receive.
		author.IfEqual(author.Var("join_request_id"), author.Var("request_id"), nil, []author.Step{joinReceive(), author.Failure()}),
		author.IfEqual(author.Var("request_value"), author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
		checks[0],
	}})
}
func report(present bool) author.Step {
	id, value := author.Integer(0), author.Integer(0)
	if present {
		id, value = author.Var("peer_id"), author.Var("peer_value")
	}
	return author.Send("join", map[string]author.Expr{
		"request_id": author.Var("report_request_id"), "s_operation": author.Var("peer_pending"),
		"status": author.Var("peer_status"), "reason": author.Var("peer_reason"), "response_present": author.Var("peer_present"), "response_id": id, "response_value": value,
	})
}
func peerCaller(peer author.Template, options Options) (author.Artifact, error) {
	steps := []author.Step{}
	for i := 0; i < 32; i++ {
		steps = append(steps, author.Set("schedule_marker", author.Integer(int64(i))))
	}
	steps = append(steps, author.Invoke("peer_pending", "peer", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil))
	await := author.AwaitInvocation("peer", "peer_pending", author.InvocationOutcomeBindings{Status: "peer_status", Reason: "peer_reason", ResponsePresent: "peer_present", Response: map[string]string{"id": "peer_id", "value": "peer_value"}})
	if !options.PrematureReport {
		steps = append(steps, await)
	} else {
		steps = append(steps, author.Set("peer_status", author.String("succeeded")), author.Set("peer_reason", author.String("")), author.Set("peer_present", author.Boolean(true)), author.Set("peer_id", author.Var("request_id")), author.Set("peer_value", author.Integer(0)))
	}
	steps = append(steps, author.Set("report_request_id", author.Var("request_id")))
	if options.WrongRequest {
		steps = append(steps, author.IfEqual(author.Var("request_id"), author.Integer(1), []author.Step{author.Set("report_request_id", author.Integer(2))}, []author.Step{author.Set("report_request_id", author.Integer(1))}))
	}
	steps = append(steps, author.IfEqual(author.Var("peer_present"), author.Boolean(true), []author.Step{report(true)}, []author.Step{report(false)}))
	if options.PrematureReport {
		steps = append(steps, await)
	}
	// Local Success means this reporting program finished, including on S failure.
	steps = append(steps, author.Success())
	return author.Compile(author.Definition{Name: "phase1j-join-static-peer", Inputs: schema.Record{"request_id": {Kind: "int"}, "request_value": {Kind: "int"}}, Invocations: map[string]author.Template{"peer": peer}, Ports: map[string]author.Port{"join": {Peer: "root", Obligation: "phase1j_s_join", Send: reportSchema()}}, Steps: steps})
}
func controlS(mode string) (author.Template, error) {
	steps := []author.Step{author.Receive(protocol.Port, map[string]string{"id": "id"})}
	switch mode {
	case "fail":
		steps = append(steps, author.Failure())
	case "absent":
		steps = append(steps, author.Success())
	case "pending":
		steps = append(steps, author.Receive(protocol.Port, map[string]string{"id": "id"}), author.Success())
	}
	artifact, err := author.Compile(author.Definition{Name: "phase1j-control-s-" + mode, Ports: map[string]author.Port{protocol.Port: {Peer: protocol.Peer, Obligation: protocol.Obligation, Receive: protocol.Response(), Send: protocol.Response()}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
}
