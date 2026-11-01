package effectfulcomposition

import (
	"fmt"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type RoleOptions struct {
	Delta                     int64
	Queries                   int
	ObserveOriginal, ReadOnly bool
}
type Options struct {
	Tree        Tree
	Inputs      map[string]schema.Value
	Left, Right RoleOptions
	Endpoint    string
	Capacity    int64
}
type Deployment struct {
	Provision         runtime.Provision
	Parent            author.Artifact
	Left, Right, Tree author.Template
	Resolvers         map[string]author.Artifact
}

func normalize(o RoleOptions, delta int64) (RoleOptions, error) {
	if o.Delta == 0 {
		o.Delta = delta
	}
	if o.Queries == 0 {
		o.Queries = 1
	}
	if (o.Delta != 1 && o.Delta != 2) || (o.Queries != 1 && o.Queries != 2) {
		return o, fmt.Errorf("require delta and logical query count in {1,2}")
	}
	return o, nil
}
func str(n int) schema.Field { return schema.Field{Kind: "string", MaxBytes: n} }
func boolean() schema.Field  { return schema.Field{Kind: "bool"} }
func integer() schema.Field  { return schema.Field{Kind: "int"} }
func issuance() schema.Record {
	return schema.Record{"issued": boolean(), "target": str(1024), "observed": boolean(), "interaction": str(128), "valid": boolean(), "mutation": integer(), "total": integer()}
}
func sendTarget(port, reg string) author.Step {
	return author.Send(port, map[string]author.Expr{"target": author.Var(reg)})
}
func receiveTarget(port, reg string) author.Step {
	return author.Receive(port, map[string]string{"target": reg})
}
func duplex(peer, ob string, r schema.Record) author.Port {
	return author.Port{Peer: peer, Obligation: ob, Send: r, Receive: r}
}

func issuer(role string, sequence int64, o RoleOptions) (author.Template, error) {
	d := author.Definition{Name: "phase2d-issuer-" + role, Ports: map[string]author.Port{
		"entry":   {Peer: "caller", Obligation: role + "_issue", Receive: schema.Record{}, Send: issuance()},
		"handoff": duplex("caller", role+"_identity", rq.ReadInput()),
	}, Effects: map[string]author.Effect{"increment": {Capability: rq.CounterCapability, Input: rq.CounterInput(), Output: rq.CounterOutput()}}, Steps: []author.Step{
		author.Receive("entry", nil), author.Request("original", "increment", map[string]author.Expr{"delta": author.Integer(o.Delta)}, nil),
	}}
	reply := map[string]author.Expr{"issued": author.Boolean(true), "target": author.Var("original"), "observed": author.Boolean(o.ObserveOriginal), "interaction": author.String("issued"), "valid": author.Boolean(false), "mutation": author.Integer(0), "total": author.Integer(0)}
	if !o.ObserveOriginal {
		d.Steps = append(d.Steps, author.Send("entry", reply))
	}
	d.Steps = append(d.Steps, sendTarget("handoff", "original"), receiveTarget("handoff", "echo"))
	if o.ObserveOriginal {
		obs := map[string]string{"disposition": "original_disposition", "valid": "original_valid", "mutation": "original_mutation", "total": "original_total"}
		for field, reg := range obs {
			if field == "disposition" {
				field = "interaction"
			}
			reply[field] = author.Var(reg)
		}
		d.Steps = append(d.Steps, author.Await("increment", "original", obs), author.Send("entry", reply), author.IfEqual(author.Var("original_valid"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()}))
	} else {
		d.Steps = append(d.Steps, author.Success())
	}
	a, err := author.Compile(d)
	if err != nil {
		return author.Template{}, err
	}
	return author.NewPairedHandoffTemplate(author.Application{"source": {Artifact: a, Inputs: map[string]schema.Value{}}}, "source", "entry", runtime.InvocationHandoff{
		Caller: runtime.RunID{Subject: "parent", Generation: 1}, Sequence: sequence, CallerPort: role + "_handoff", EntryPort: "handoff",
	})
}

func New(o Options) (Deployment, error) {
	var d Deployment
	var err error
	if o.Left, err = normalize(o.Left, 1); err != nil {
		return d, err
	}
	if o.Right, err = normalize(o.Right, 2); err != nil {
		return d, err
	}
	if o.Capacity == 0 {
		o.Capacity = 2
	}
	if o.Capacity < 1 || o.Capacity > 2 {
		return d, fmt.Errorf("require physical capacity one or two")
	}
	if o.Tree.template.ID() == "" || len(o.Tree.paths) == 0 {
		return d, fmt.Errorf("require a legal Series 1 tree")
	}
	in, _, err := o.Tree.template.Schemas()
	if err != nil {
		return d, err
	}
	if err = in.Record.Validate(o.Inputs); err != nil {
		return d, err
	}
	for _, p := range o.Tree.paths {
		id, v := o.Inputs[p+"id"].Number, o.Inputs[p+"value"].Number
		if (id != 1 && id != 2) || (v != 0 && v != 1) {
			return d, fmt.Errorf("pure request %q outside Series 1 finite domain", p)
		}
	}
	d.Tree = o.Tree.template
	d.Left, err = issuer("l", 1, o.Left)
	if err != nil {
		return d, err
	}
	d.Right, err = issuer("r", 2, o.Right)
	if err != nil {
		return d, err
	}
	d.Parent, err = parentProgram(o, d.Left, d.Right)
	if err != nil {
		return d, err
	}
	app := author.Application{"parent": {Artifact: d.Parent, Inputs: o.Inputs}}
	d.Resolvers = map[string]author.Artifact{}
	var roots []runtime.ResourceRoot
	var grants []runtime.ReceiptGrant
	for _, role := range []string{"l", "r"} {
		ro, seq := o.Left, 1
		if role == "r" {
			ro, seq = o.Right, 2
		}
		r, err := resolverProgram(role, ro.Queries)
		if err != nil {
			return d, err
		}
		id := runtime.RunID{Subject: "resolver_" + role, Generation: 1}
		app[id.Subject] = author.Instance{Artifact: r}
		d.Resolvers[role] = r
		roots = append(roots, runtime.ResourceRoot{Run: id, Members: []runtime.RunID{id}, Capabilities: []string{rq.ReadCapability}, Limits: runtime.ResourceLimits{MaxEffects: int64(ro.Queries), MaxInFlight: 1}})
		grants = append(grants, runtime.ReceiptGrant{Resolver: id, Invocation: fmt.Sprintf(`["parent",1,%d,0]`, seq), Member: "source", Sequence: 2, Endpoint: o.Endpoint, Adopt: !ro.ReadOnly})
	}
	d.Provision, err = app.Provision()
	if err != nil {
		return d, err
	}
	var allowed []string
	for _, t := range d.Provision.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	p := runtime.RunID{Subject: "parent", Generation: 1}
	roots = append(roots, runtime.ResourceRoot{Run: p, Members: []runtime.RunID{p}, Capabilities: []string{rq.CounterCapability}, Templates: allowed, Limits: runtime.ResourceLimits{MaxDepth: max(o.Tree.depth, 1), MaxInvocations: o.Tree.slots + 2, MaxCreatedRuns: o.Tree.slots + 2, MaxEffects: 2, MaxInFlight: o.Capacity}})
	d.Provision.Composition.Roots = roots
	if o.Endpoint != "" {
		if _, err = rq.EndpointID(o.Endpoint, "apply"); err != nil {
			return d, err
		}
		d.Provision.Reconciliation = &runtime.ReconciliationProfile{Grants: grants}
	}
	return d, nil
}
