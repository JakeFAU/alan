package author

import (
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func TestInvocationHandleRoundTripThroughList(t *testing.T) {
	leaf := invocationTemplate(t)
	handles := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 8192, Element: schema.Record{"operation": {Kind: "string", MaxBytes: 4096}}}
	_, err := Compile(Definition{Invocations: map[string]Template{"child": leaf}, Collections: map[string]schema.Field{"handles": handles}, Steps: []Step{
		Invoke("pending", "child", nil, nil), Append("handles", map[string]Expr{"operation": Var("pending")}),
		ForEach(Var("handles"), map[string]string{"operation": "saved"}, AwaitInvocation("child", "saved", InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"})), Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
}
func TestInvocationAwaitRejectsKnownWrongDeclarationAndNonString(t *testing.T) {
	leaf := invocationTemplate(t)
	for _, steps := range [][]Step{
		{Invoke("pending", "other", nil, nil), AwaitInvocation("child", "pending", InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"}), Success()},
		{Set("pending", Integer(7)), AwaitInvocation("child", "pending", InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"}), Success()},
	} {
		if _, err := Compile(Definition{Invocations: map[string]Template{"child": leaf, "other": leaf}, Steps: steps}); err == nil {
			t.Fatal("invalid handle accepted")
		}
	}
}
func TestAdapterAwaitStillRejectsPlainString(t *testing.T) {
	_, err := Compile(Definition{Effects: map[string]Effect{"effect": {Capability: "test", Input: schema.Record{}, Output: schema.Record{}}}, Values: map[string]schema.Value{"saved": schema.Text("opaque")}, Steps: []Step{Await("effect", "saved", nil), Success()}})
	if err == nil {
		t.Fatal("adapter Await lost request declaration proof")
	}
}
