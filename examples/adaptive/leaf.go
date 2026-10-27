package adaptive

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

const ScalarCapability = "adaptive-observe"

// ScalarLeaf returns a controlled Request/Await leaf, or a pure identity leaf.
func ScalarLeaf(controlled bool) (author.Template, error) {
	input := schema.Record{"value": integer()}
	output := schema.Record{"observation": integer()}
	d := author.Definition{Name: "adaptive scalar", Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: input, Send: output}}}
	d.Steps = []author.Step{author.Receive("public", bindings("value"))}
	if controlled {
		d.Effects = map[string]author.Effect{"observe": {Capability: ScalarCapability, Input: input, Output: output}}
		d.Steps = append(d.Steps, author.Request("request", "observe", exprs("value"), nil), author.Await("observe", "request", bindings("observation", "valid")), author.IfEqual(author.Var("valid"), author.Boolean(true), nil, []author.Step{author.Failure()}))
	} else {
		d.Steps = append(d.Steps, author.Set("observation", author.Var("value")))
	}
	d.Steps = append(d.Steps, author.Send("public", exprs("observation")), author.Success())
	a, err := author.Compile(d)
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
}
