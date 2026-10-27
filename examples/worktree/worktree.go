// Package worktree demonstrates post-initialization collection input using only
// public authoring and runtime APIs. Application policy runs in retained programs.
package worktree

import (
	"context"
	"fmt"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/compose"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Deployment exposes immutable authoring identities and the fixed initial graph.
// New is an authoring/deployment operation. Recovery uses runtime.Open and Run.
type Deployment struct {
	Provision               runtime.Provision
	Leaf, Collection, Entry author.Template
}

func DefaultLimits(capacity int64) runtime.ResourceLimits {
	return runtime.ResourceLimits{MaxInvocations: 32, MaxCreatedRuns: 64, MaxDepth: 3, MaxEffects: 96, MaxInFlight: capacity}
}

// Deterministic doubles integers through an ordinary Add step. Overflow is an
// explicit runtime arithmetic error, never wrapping or a fabricated item value.
func Deterministic() (author.Template, error) {
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

// Evidence retains the existing helper's unchanged two-program policy as a leaf.
// The caller supplies controlled adapters to Run; no provider settings are read.
func Evidence(b evidence.Budgets) (author.Template, error) {
	c, err := evidence.New("entry", "caller", "work", b)
	if err != nil {
		return author.Template{}, err
	}
	for name, instance := range c.Instances {
		instance.Inputs = map[string]schema.Value{}
		c.Instances[name] = instance
	}
	return author.NewTemplate(c.Instances, "entry", "caller")
}

// New creates a single static intake/root run and a closed retained catalogue.
// The direct path is root -> collection -> leaf; nested inserts one wrapper.
// Both profiles use the exact same collection template and public contract.
func New(leaf author.Template, nested bool, limits runtime.ResourceLimits) (Deployment, error) {
	d := Deployment{Leaf: leaf}
	var err error
	d.Collection, err = compose.New(leaf, compose.DefaultOptions())
	if err != nil {
		return d, err
	}
	d.Entry = d.Collection
	if nested {
		d.Entry, err = wrapper(d.Collection)
		if err != nil {
			return d, err
		}
	}
	input, output, err := d.Entry.Schemas()
	if err != nil {
		return d, err
	}
	artifact, err := author.Compile(author.Definition{Name: "collection intake", Invocations: map[string]author.Template{"collection": d.Entry},
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
		return d, err
	}
	d.Provision, err = (author.Application{"root": {Artifact: artifact, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		return d, err
	}
	if d.Provision.Composition == nil {
		return d, fmt.Errorf("missing closed catalogue")
	}
	allowed := make([]string, 0, len(d.Provision.Composition.Templates))
	for _, t := range d.Provision.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: runtime.RunID{Subject: "root", Generation: 1}, Members: []runtime.RunID{{Subject: "root", Generation: 1}}, Templates: allowed, Capabilities: []string{evidence.ModelCapability, evidence.LookupCapability}, Limits: limits}}
	return d, nil
}

func wrapper(collection author.Template) (author.Template, error) {
	input, output, err := collection.Schemas()
	if err != nil {
		return author.Template{}, err
	}
	a, err := author.Compile(author.Definition{Name: "collection wrapper", Invocations: map[string]author.Template{"collection": collection},
		Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: input.Record, Send: output.Record}},
		Steps: []author.Step{author.Receive("public", map[string]string{"items": "items"}), author.Invoke("pending", "collection", map[string]author.Expr{"items": author.Var("items")}, nil),
			author.AwaitInvocation("collection", "pending", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"results": "results", "all_succeeded": "all_succeeded"}}),
			author.IfEqual(author.Var("present"), author.Boolean(true), []author.Step{author.Send("public", map[string]author.Expr{"results": author.Var("results"), "all_succeeded": author.Var("all_succeeded")})}, nil),
			author.IfEqual(author.Var("status"), author.String("succeeded"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
}

// Submit admits transport input only. Collection acceptance is its later ordinary
// Receive commit, retaining items in that invocation's own projection. Transport
// retransmission reuses identity; a root accepts one obligation in this profile.
func Submit(ctx context.Context, s *runtime.Store, identity string, items schema.Value) (runtime.AdmissionResult, error) {
	if identity == "" {
		return runtime.AdmissionResult{}, fmt.Errorf("work identity required")
	}
	return s.AdmitInput(ctx, runtime.Input{TransportID: identity, LogicalID: identity, Target: "root", Generation: 1, Obligation: "work", Payload: map[string]schema.Value{"items": items}})
}
