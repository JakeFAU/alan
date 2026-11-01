package effectfulcomposition

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Each probe passes a genuinely selected failed invocation with no response to
// the exact guard used by F(T). Other guard inputs are inert typed defaults.
// The child has no effects and emits no candidate application information.
func TestAbsentEnvelopeGuards(t *testing.T) {
	for _, kind := range []string{"issuer", "tree"} {
		t.Run(kind, func(t *testing.T) {
			out := issuance()
			prefix := "l_source_"
			class := "l_class"
			guard := roleChecks("l")
			inputs := schema.Record{"l_target": str(1024)}
			for k, f := range reportSchema() {
				inputs["l_report_"+k] = f
			}
			fields := map[string]string{}
			for k := range out {
				fields[k] = "l_" + k
			}
			fields["target"] = "l_issued_target"
			if kind == "tree" {
				out = callablejoin.Report()
				prefix = "tree_"
				class = "tree_class"
				guard = treeChecks([]string{""})
				inputs = schema.Record{"id": integer(), "value": integer()}
				fields = map[string]string{}
				for k := range out {
					fields[k] = "tree_" + k
				}
			}
			child, err := author.Compile(author.Definition{Ports: map[string]author.Port{"entry": {Peer: "caller", Obligation: "test", Send: out}}, Steps: []author.Step{author.Receive("entry", nil), author.Failure()}})
			if err != nil {
				t.Fatal(err)
			}
			template, err := author.NewTemplate(author.Application{"source": {Artifact: child, Inputs: map[string]schema.Value{}}}, "source", "entry")
			if err != nil {
				t.Fatal(err)
			}
			steps := []author.Step{author.Invoke("pending", "child", nil, nil), author.AwaitInvocation("child", "pending", author.InvocationOutcomeBindings{Status: prefix + "status", Reason: prefix + "reason", ResponsePresent: prefix + "present", Response: fields}), setBool("accepted", true)}
			steps = append(steps, guard...)
			steps = append(steps, eq("accepted", author.Boolean(false), []author.Step{author.Success()}, []author.Step{author.Failure()}))
			parent, err := author.Compile(author.Definition{Name: "absence-" + kind, Inputs: inputs, Invocations: map[string]author.Template{"child": template}, Steps: steps})
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]schema.Value{}
			for k, f := range inputs {
				values[k] = schema.Default(f)
			}
			p, err := (author.Application{"parent": {Artifact: parent, Inputs: values}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			id := runtime.RunID{Subject: "parent", Generation: 1}
			p.Composition.Roots = []runtime.ResourceRoot{{Run: id, Members: []runtime.RunID{id}, Templates: []string{template.ID()}, Limits: runtime.ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxInFlight: 1}}}
			path := filepath.Join(t.TempDir(), "control.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if _, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
				t.Fatal(err)
			}
			r, err := s.Snapshot(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			v := r.Projection.Data
			if r.Projection.Status != "succeeded" || v[prefix+"status"].Text != "failed" || v[prefix+"present"].Flag || v["accepted"].Flag || v[class].Text != "invalid" {
				t.Fatal(r)
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnserializedWitnessHandleJoinRejected(t *testing.T) {
	_, err := author.Compile(author.Definition{
		Inputs:  schema.Record{"keep": boolean()},
		Effects: map[string]author.Effect{"lookup": {Capability: "probe", Input: schema.Record{}, Output: schema.Record{}}},
		Ports:   map[string]author.Port{"report": {Peer: "parent", Obligation: "test", Send: schema.Record{"id": str(1024)}}},
		Steps: []author.Step{
			setText("witness_query", ""), author.Request("query1", "lookup", nil, nil),
			eq("keep", author.Boolean(true), []author.Step{author.Set("witness_query", author.Var("query1"))}, nil),
			author.Send("report", map[string]author.Expr{"id": author.Var("witness_query")}), author.Success(),
		},
	})
	if err == nil || !strings.Contains(err.Error(), `"witness_query" is not definitely available`) {
		t.Fatal(err)
	}
}
