package compose_test

import (
	"context"
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/compose"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// deterministicLeaf doubles integers through an ordinary Add step. Overflow is an
// explicit runtime arithmetic error, never wrapping or a fabricated item value.
func deterministicLeaf() (author.Template, error) {
	a, err := author.Compile(author.Definition{
		Name: "double integer",
		Ports: map[string]author.Port{"public": {
			Peer: "caller", Obligation: "work",
			Receive: schema.Record{"number": {Kind: "int"}},
			Send:    schema.Record{"doubled": {Kind: "int"}},
		}},
		Steps: []author.Step{
			author.Receive("public", map[string]string{"number": "number"}),
			author.Add("doubled", author.Var("number"), author.Var("number")),
			author.Send("public", map[string]author.Expr{"doubled": author.Var("doubled")}),
			author.Success(),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
}

// collectionProvision runs the direct collection/declarative leaf used by these tests.
func collectionProvision(leaf author.Template) (runtime.Provision, error) {
	collection, err := compose.New(leaf, compose.DefaultOptions())
	if err != nil {
		return runtime.Provision{}, err
	}
	input, output, err := collection.Schemas()
	if err != nil {
		return runtime.Provision{}, err
	}
	artifact, err := author.Compile(author.Definition{Name: "collection intake", Invocations: map[string]author.Template{"collection": collection},
		Collections: map[string]schema.Field{"results": output.Record["results"]}, Values: map[string]schema.Value{"all_succeeded": schema.Bool(false)},
		Ports: map[string]author.Port{"intake": {Local: true, Peer: "local", Obligation: "work", Receive: input.Record}},
		Steps: []author.Step{author.Receive("intake", map[string]string{"items": "submitted_items"}),
			author.Invoke("pending", "collection", map[string]author.Expr{"items": author.Var("submitted_items")}, nil),
			author.AwaitInvocation("collection", "pending", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"results": "received_results", "all_succeeded": "received_success"}}),
			author.IfEqual(author.Var("present"), author.Boolean(true), []author.Step{author.Set("results", author.Var("received_results")), author.Set("all_succeeded", author.Var("received_success"))}, nil),
			author.IfEqual(author.Var("status"), author.String("succeeded"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		return runtime.Provision{}, err
	}
	p, err := (author.Application{"root": {Artifact: artifact, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		return runtime.Provision{}, err
	}
	if p.Composition == nil {
		return runtime.Provision{}, fmt.Errorf("missing closed catalogue")
	}
	allowed := make([]string, 0, len(p.Composition.Templates))
	for _, t := range p.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	p.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}}, Templates: allowed, Limits: runtime.ResourceLimits{MaxInvocations: 32, MaxCreatedRuns: 64, MaxDepth: 3, MaxEffects: 96, MaxInFlight: 1}}}
	return p, nil
}

// submitCollection admits transport input only. Collection acceptance is its later ordinary
// Receive commit, retaining items in that invocation's own projection. Transport
// retransmission reuses identity; a root accepts one obligation in this profile.
func submitCollection(ctx context.Context, s *runtime.Store, identity string, items schema.Value) (runtime.AdmissionResult, error) {
	if identity == "" {
		return runtime.AdmissionResult{}, fmt.Errorf("work identity required")
	}
	return s.AdmitInput(ctx, runtime.Input{TransportID: identity, LogicalID: identity, Target: "root", Generation: 1, Obligation: "work", Payload: map[string]schema.Value{"items": items}})
}
