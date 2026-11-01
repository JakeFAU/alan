package effectboundary

import (
	"fmt"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type Config struct {
	Parent, Peer, FailAfterRequest bool
	Delta                          int64
}
type Deployment struct {
	Provision      runtime.Provision
	Child          author.Template
	Parent, Source author.Artifact
}

func input() schema.Record  { return schema.Record{"delta": {Kind: "int"}} }
func output() schema.Record { return schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}} }
func effect() map[string]author.Effect {
	return map[string]author.Effect{"increment": {Capability: Capability, Input: input(), Output: output()}}
}
func requestSteps(fail bool, reply bool) []author.Step {
	steps := []author.Step{author.Request("operation", "increment", map[string]author.Expr{"delta": author.Var("delta")}, nil)}
	if fail {
		return append(steps, author.Failure())
	}
	steps = append(steps, author.Await("increment", "operation", map[string]string{"valid": "valid", "disposition": "disposition", "mutation": "mutation", "total": "total"}))
	yes := []author.Step{}
	if reply {
		yes = append(yes, author.Send("caller", map[string]author.Expr{"mutation": author.Var("mutation"), "total": author.Var("total")}))
	}
	yes = append(yes, author.Success())
	return append(steps, author.IfEqual(author.Var("valid"), author.Boolean(true), yes, []author.Step{author.Failure()}))
}

// Source is an ordinary effect source whose Request is its first commit.
func Source() (author.Artifact, error) {
	return author.Compile(author.Definition{Name: "phase2a-source", Inputs: input(), Effects: effect(), Steps: requestSteps(false, false)})
}
func child(fail bool) (author.Template, error) {
	steps := append([]author.Step{author.Receive("caller", map[string]string{"delta": "delta"})}, requestSteps(fail, true)...)
	a, err := author.Compile(author.Definition{Name: fmt.Sprintf("phase2a-child-private-failure-%t", fail), Effects: effect(), Ports: map[string]author.Port{"caller": {Peer: "caller", Obligation: "counter-request", Receive: input(), Send: output()}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "caller")
}

// New builds the source and optionally an ordinary owning invocation parent and
// an independently enabled effectful peer. The peer has its own original key.
func New(c Config) (Deployment, error) {
	var d Deployment
	var err error
	if c.Delta == 0 {
		c.Delta = 1
	}
	if c.Delta != 1 && c.Delta != 2 {
		return d, fmt.Errorf("delta must be one or two")
	}
	d.Source, err = Source()
	if err != nil {
		return d, err
	}
	app := author.Application{}
	if !c.Parent {
		app["client"] = author.Instance{Artifact: d.Source, Inputs: map[string]schema.Value{"delta": schema.Int(c.Delta)}}
	} else {
		d.Child, err = child(c.FailAfterRequest)
		if err != nil {
			return d, err
		}
		d.Parent, err = author.Compile(author.Definition{Name: "phase2a-ordinary-parent", Inputs: input(), Invocations: map[string]author.Template{"child": d.Child}, Steps: []author.Step{
			author.Set("knowledge", author.String("may_have_applied")),
			author.Invoke("child_operation", "child", map[string]author.Expr{"delta": author.Var("delta")}, nil),
			author.AwaitInvocation("child", "child_operation", author.InvocationOutcomeBindings{Status: "child_status", Reason: "child_reason", ResponsePresent: "child_response_present", Response: map[string]string{"mutation": "child_mutation", "total": "child_total"}}),
			author.IfEqual(author.Var("child_status"), author.String("succeeded"), []author.Step{author.IfEqual(author.Var("child_response_present"), author.Boolean(true), []author.Step{author.Set("knowledge", author.String("historical_application_receipt")), author.Success()}, []author.Step{author.Failure()})}, []author.Step{author.Set("knowledge", author.String("local_failure_world_unresolved")), author.Success()}),
		}})
		if err != nil {
			return d, err
		}
		app["root"] = author.Instance{Artifact: d.Parent, Inputs: map[string]schema.Value{"delta": schema.Int(c.Delta)}}
	}
	if c.Peer {
		// Independent bounded private work makes the primary child's request
		// eligible first. No child observation, endpoint fact or host signal
		// enables this peer: it is runnable from the initial provision.
		peerSteps := append([]author.Step{author.Repeat(12, author.Set("private_work", author.Integer(1)))}, requestSteps(false, false)...)
		peer, compileErr := author.Compile(author.Definition{Name: "phase2a-independent-peer", Inputs: input(), Effects: effect(), Steps: peerSteps})
		if compileErr != nil {
			return d, compileErr
		}
		app["peer"] = author.Instance{Artifact: peer, Inputs: map[string]schema.Value{"delta": schema.Int(2)}}
	}
	d.Provision, err = app.Provision()
	if err != nil {
		return d, err
	}
	if c.Parent {
		root := runtime.RunID{Subject: "root", Generation: 1}
		templates := []string{}
		for _, t := range d.Provision.Composition.Templates {
			templates = append(templates, t.ID)
		}
		d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: templates, Capabilities: []string{Capability}, Limits: runtime.ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 1, MaxInFlight: 1}}}
	}
	return d, nil
}
