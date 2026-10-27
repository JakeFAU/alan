package author

import (
	"encoding/json"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func invocationTemplate(t *testing.T) Template {
	t.Helper()
	record := schema.Record{"result": {Kind: "string", Enum: []string{"ok"}, MaxBytes: 20}}
	a, err := Compile(Definition{Ports: map[string]Port{"public": {Peer: "caller", Obligation: "work", Receive: schema.Record{}, Send: record}}, Steps: []Step{Receive("public", nil), Send("public", map[string]Expr{"result": String("ok")}), Success()}})
	if err != nil {
		t.Fatal(err)
	}
	app := Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}
	template, err := NewTemplate(app, "entry", "public")
	if err != nil {
		t.Fatal(err)
	}
	app["entry"] = Instance{}
	return template
}
func TestInvocationAuthoringRetainsTemplateAndAbsentOutcome(t *testing.T) {
	template := invocationTemplate(t)
	a, err := Compile(Definition{Invocations: map[string]Template{"child": template}, Values: map[string]schema.Value{"n": schema.Int(0)}, Steps: []Step{Invoke("pending", "child", nil, map[string]Expr{"n": Integer(1)}), AwaitInvocation("child", "pending", InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"result": "result"}}), Success()}})
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Semantics != CollectionSemanticVersion {
		t.Fatal(p.Semantics)
	}
	state := executor.Projection{Status: "open", Data: map[string]schema.Value{}}
	r, err := executor.Reduce(p, state, nil, executor.RunID{"root", 1}, 1, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	r, err = executor.Reduce(p, r.Projection, nil, executor.RunID{"root", 1}, 2, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Intents) != 1 || r.Intents[0].Kind != "invocation" || r.Projection.Data["n"].Number != 1 {
		t.Fatal(r)
	}
	o := executor.Observation{Run: executor.RunID{"root", 1}, Kind: "invocation", Provenance: "runtime-invocation", Source: template.ID(), SourceGeneration: 1, Obligation: "child", Operation: r.Intents[0].ID, Payload: map[string]schema.Value{"status": schema.Text("rejected"), "reason": schema.Text("depth_limit"), "response_present": schema.Bool(false)}}
	r, err = executor.Reduce(p, r.Projection, &o, executor.RunID{"root", 1}, 3, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if r.Projection.Data["present"].Flag || r.Projection.Data["status"].Text != "rejected" {
		t.Fatal(r)
	}
	if _, exists := r.Projection.Data["result"]; exists {
		t.Fatal("absent response fabricated")
	}
	provision, err := (Application{"root": {Artifact: a, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	if provision.Composition == nil || len(provision.Composition.Templates) != 1 || len(provision.Programs) != 2 {
		t.Fatal("catalogue not retained")
	}
}
func TestTemplateRejectsOpenInternalWiring(t *testing.T) {
	a, err := Compile(Definition{Ports: map[string]Port{"public": {Peer: "caller", Obligation: "work"}, "escape": {Peer: "missing", Obligation: "other"}}, Steps: []Step{Send("escape", nil), Success()}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewTemplate(Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public"); err == nil {
		t.Fatal("open template accepted")
	}
}
