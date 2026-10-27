package adaptive

import (
	"context"
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type Deployment struct {
	Provision       runtime.Provision
	Adaptive, Entry author.Template
}

func DefaultLimits(capacity int64) runtime.ResourceLimits {
	return runtime.ResourceLimits{MaxInvocations: 32, MaxCreatedRuns: 64, MaxDepth: 3, MaxEffects: 96, MaxInFlight: capacity}
}

// New authors a fixed intake and optional wrapper. Recovery only opens the store.
func New(component author.Template, nested bool, limits runtime.ResourceLimits) (Deployment, error) {
	d := Deployment{Adaptive: component, Entry: component}
	var err error
	if nested {
		d.Entry, err = forwarder(component, false)
		if err != nil {
			return d, err
		}
	}
	intake, err := forwardArtifact(d.Entry, true)
	if err != nil {
		return d, err
	}
	d.Provision, err = (author.Application{"root": {Artifact: intake, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		return d, err
	}
	if d.Provision.Composition == nil {
		return d, fmt.Errorf("missing closed catalogue")
	}
	allowed := []string{}
	for _, v := range d.Provision.Composition.Templates {
		allowed = append(allowed, v.ID)
	}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}}, Templates: allowed, Capabilities: []string{ScalarCapability, evidence.ModelCapability, evidence.LookupCapability}, Limits: limits}}
	return d, nil
}
func forwarder(component author.Template, local bool) (author.Template, error) {
	a, err := forwardArtifact(component, local)
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
}
func forwardArtifact(component author.Template, local bool) (author.Artifact, error) {
	input, output, err := component.Schemas()
	if err != nil {
		return author.Artifact{}, err
	}
	port := "public"
	peer := "caller"
	if local {
		port = "intake"
		peer = "local"
	}
	receive := map[string]string{}
	payload := map[string]author.Expr{}
	response := map[string]string{}
	reply := map[string]author.Expr{}
	for k := range input.Record {
		receive[k] = "request_" + k
		payload[k] = author.Var("request_" + k)
	}
	for k := range output.Record {
		response[k] = k
		reply[k] = author.Var(k)
	}
	steps := []author.Step{
		author.Receive(port, receive), author.Invoke("pending", "adaptive", payload, nil),
		author.AwaitInvocation("adaptive", "pending", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: response}),
	}
	if !local {
		steps = append(steps, eq("present", author.Boolean(true), author.Send(port, reply)))
	}
	steps = append(steps, author.IfEqual(author.Var("status"), author.String("succeeded"), []author.Step{author.Success()}, []author.Step{author.Failure()}))
	return author.Compile(author.Definition{Name: "adaptive forwarding", Invocations: map[string]author.Template{"adaptive": component}, Ports: map[string]author.Port{port: {Local: local, Peer: peer, Obligation: "work", Receive: input.Record, Send: func() schema.Record {
		if local {
			return nil
		}
		return output.Record
	}()}}, Steps: steps})
}
func Submit(ctx context.Context, s *runtime.Store, identity string, request map[string]schema.Value) (runtime.AdmissionResult, error) {
	if identity == "" {
		return runtime.AdmissionResult{}, fmt.Errorf("work identity required")
	}
	return s.AdmitInput(ctx, runtime.Input{TransportID: identity, LogicalID: identity, Target: "root", Generation: 1, Obligation: "work", Payload: request})
}
