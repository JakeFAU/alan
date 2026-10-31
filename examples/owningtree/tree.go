// Package owningtree constructs retained finite owning trees over the Phase 1N join.
package owningtree

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Leaf reuses J, its exact original candidate, and the original successor S.
func Leaf(candidate author.Template) (author.Template, error) {
	in, out, err := candidate.Schemas()
	if err != nil {
		return author.Template{}, err
	}
	wantIn, err := protocol.Request().Retain()
	if err != nil {
		return author.Template{}, err
	}
	wantOut, err := protocol.Response().Retain()
	if err != nil {
		return author.Template{}, err
	}
	if in.ID != wantIn.ID || out.ID != wantOut.ID {
		return author.Template{}, fmt.Errorf("incompatible candidate request or response schema")
	}

	// The historical builder needs a legal input to return S. Only its Peer is
	// used. No host input or operation from that provision enters the template.
	original, err := substitution.New(candidate, substitution.Config{
		Shape: "sequential", Request: substitution.Request{ID: 1, Mode: "answer"},
	})
	if err != nil {
		return author.Template{}, err
	}
	_, leaf, err := callablejoin.NewJoin(candidate, original.Peer)
	return leaf, err
}

type branch struct {
	name, prefix string
	template     author.Template
}

func Relay(child author.Template) (author.Template, error) {
	return member("relay", branch{"child", "", child})
}

func Fork(left, right author.Template) (author.Template, error) {
	return member("fork", branch{"left", "l_", left}, branch{"right", "r_", right})
}

func member(name string, children ...branch) (author.Template, error) {
	art, err := compile(name, children)
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: art, Inputs: map[string]schema.Value{}}}, "entry", callablejoin.PublicPort)
}

func Outer(child author.Template) (author.Artifact, error) {
	return compile("outer", []branch{{"tree", "", child}})
}

func Provision(child author.Template, request map[string]schema.Value, limits runtime.ResourceLimits) (runtime.Provision, error) {
	art, err := Outer(child)
	if err != nil {
		return runtime.Provision{}, err
	}
	if err := art.Metadata().Inputs.Record.Validate(request); err != nil {
		return runtime.Provision{}, err
	}
	for name, v := range request {
		if strings.HasSuffix(name, "id") && (v.Number < 1 || v.Number > 2) || strings.HasSuffix(name, "value") && (v.Number < 0 || v.Number > 1) {
			return runtime.Provision{}, fmt.Errorf("%s outside the finite leaf domain", name)
		}
	}
	p, err := (author.Application{"root": {Artifact: art, Inputs: request}}).Provision()
	if err != nil {
		return p, err
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	var allowed []string
	for _, t := range p.Composition.Templates {
		allowed = append(allowed, t.ID)
	}
	p.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: limits}}
	return p, nil
}

func addFields(dst schema.Record, prefix string, src schema.Record) {
	for k, v := range src {
		dst[prefix+k] = v
	}
}

// leafPaths recognizes full binary products directly from retained schemas.
// Relay deliberately has no wire-level marker. Claims about a child's behavior
// require an additional premise independent of this shape check.
func leafPaths(input, output schema.Retained) ([]string, error) {
	leaf, fork := map[string]bool{}, map[string]bool{}
	wantIn, wantOut := schema.Record{}, schema.Record{}
	for name := range input.Record {
		prefix, ok := strings.CutSuffix(name, "id")
		if !ok {
			continue
		}
		rest := prefix
		for rest != "" {
			if !strings.HasPrefix(rest, "l_") && !strings.HasPrefix(rest, "r_") {
				return nil, fmt.Errorf("bad role in %q", name)
			}
			fork[prefix[:len(prefix)-len(rest)]] = true
			rest = rest[2:]
		}
		leaf[prefix] = true
		addFields(wantIn, prefix, protocol.Request())
		addFields(wantOut, prefix, callablejoin.Report())
	}
	for p := range fork {
		if leaf[p] || !(leaf[p+"l_"] || fork[p+"l_"]) || !(leaf[p+"r_"] || fork[p+"r_"]) {
			return nil, fmt.Errorf("incomplete binary product at %q", p)
		}
	}
	i, err := wantIn.Retain()
	if err != nil {
		return nil, err
	}
	o, err := wantOut.Retain()
	if err != nil {
		return nil, err
	}
	if len(leaf) == 0 || input.ID != i.ID || output.ID != o.ID {
		return nil, fmt.Errorf("incompatible owning-tree product schema")
	}
	return slices.Sorted(maps.Keys(leaf)), nil
}

func compile(name string, children []branch) (author.Artifact, error) {
	d := author.Definition{Name: "phase1t-" + name, Invocations: map[string]author.Template{}}
	input, output := schema.Record{}, schema.Record{}
	receive := map[string]string{}
	reply := map[string]author.Expr{}
	var invokes, selects []author.Step
	for _, c := range children {
		in, out, err := c.template.Schemas()
		if err != nil {
			return author.Artifact{}, err
		}
		paths, err := leafPaths(in, out)
		if err != nil {
			return author.Artifact{}, err
		}
		addFields(input, c.prefix, in.Record)
		addFields(output, c.prefix, out.Record)
		d.Invocations[c.name] = c.template
		args := map[string]author.Expr{}
		for field := range in.Record {
			wire := c.prefix + field
			args[field] = author.Var(wire)
			receive[wire] = wire
		}
		bind := map[string]string{}
		for field := range out.Record {
			wire := c.prefix + field
			bind[field] = "report_" + wire
			reply[wire] = author.Var(bind[field])
		}
		op := c.name + "_operation"
		invokes = append(invokes, author.Invoke(op, c.name, args, nil))
		selects = append(selects, author.AwaitInvocation(c.name, op, author.InvocationOutcomeBindings{
			Status: c.name + "_status", Reason: c.name + "_reason", ResponsePresent: c.name + "_present", Response: bind,
		}))
		selects = append(selects, validate(c, paths)...)
	}
	outer := name == "outer"
	if outer {
		d.Inputs = input
	} else {
		d.Ports = map[string]author.Port{callablejoin.PublicPort: {Peer: "caller", Obligation: "work", Receive: input, Send: output}}
		d.Steps = append(d.Steps, author.Receive(callablejoin.PublicPort, receive))
	}
	d.Steps = append(d.Steps, invokes...)
	d.Steps = append(d.Steps, flag("tree_valid", true), flag("tree_answer", true))
	d.Steps = append(d.Steps, selects...)
	var forward []author.Step
	if !outer {
		forward = append(forward, author.Send(callablejoin.PublicPort, reply))
	}
	forward = append(forward, author.IfEqual(author.Var("tree_answer"), author.Boolean(true),
		[]author.Step{decision("answer"), author.Success()}, []author.Step{decision("diagnosed"), author.Failure()}))
	d.Steps = append(d.Steps, author.IfEqual(author.Var("tree_valid"), author.Boolean(true), forward,
		[]author.Step{decision("invalid_tree_outcome"), author.Failure()}))
	return author.Compile(d)
}
