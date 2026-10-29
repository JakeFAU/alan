package substitution

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

const EffectCapability = "phase1i-controlled-effect"

// EffectCompanion is an ordinary public Request/Await program, separate from K1.
// It issues a second request after the first authentic response to expose the
// final reservation fence when a static companion survives root closure.
func EffectCompanion(twice bool) (author.Template, error) {
	steps := []author.Step{author.Receive(protocol.Port, map[string]string{"id": "id", "value": "value"}), author.Request("effect_operation", "controlled", map[string]author.Expr{"value": author.Var("value")}, nil), author.Await("controlled", "effect_operation", map[string]string{"observation": "effect_value", "valid": "effect_valid"})}
	if twice {
		steps = append(steps, author.Request("second_operation", "controlled", map[string]author.Expr{"value": author.Integer(99)}, nil), author.Await("controlled", "second_operation", map[string]string{"valid": "second_valid"}))
	}
	steps = append(steps, author.IfEqual(author.Var("effect_valid"), author.Boolean(true), []author.Step{author.Send(protocol.Port, map[string]author.Expr{"id": author.Var("id"), "value": author.Var("effect_value")}), author.Success()}, []author.Step{author.Failure()}))
	a, err := author.Compile(author.Definition{Name: "phase1i-k1e", Ports: map[string]author.Port{protocol.Port: {Peer: protocol.Peer, Obligation: protocol.Obligation, Receive: protocol.Request(), Send: protocol.Response()}}, Effects: map[string]author.Effect{"controlled": {Capability: EffectCapability, Input: schema.Record{"value": {Kind: "int"}}, Output: schema.Record{"observation": {Kind: "int"}}}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
}
