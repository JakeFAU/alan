package reconciliation

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// An extra Set moves the actual Invoke to sequence two. Even the correct
// caller and template cannot use the ordinary identity channel for that call.
func TestHandoffIsolation(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		shift, alias, malformed bool
		want                    string
	}{
		{name: "matching invocation"}, {name: "other invocation", shift: true},
		{name: "reserved alias registered", alias: true, want: "aliases a component"},
		{name: "wrong caller schema", malformed: true, want: "invalid caller handoff port"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			port := author.Port{Peer: "caller", Obligation: "identity", Send: target(), Receive: target()}
			child, err := author.Compile(author.Definition{Ports: map[string]author.Port{"entry": {Peer: "caller", Obligation: "call"}, "id": port}, Steps: []author.Step{
				author.Receive("entry", nil), author.Send("id", map[string]author.Expr{"target": author.String("ordinary-probe")}), author.Receive("id", map[string]string{"target": "echo"}), author.Send("entry", nil), author.Success(),
			}})
			if err != nil {
				t.Fatal(err)
			}
			caller := runtime.RunID{Subject: "parent", Generation: 1}
			template, err := author.NewHandoffTemplate(author.Application{"source": {Artifact: child, Inputs: map[string]schema.Value{}}}, "source", "entry", runtime.InvocationHandoff{Caller: caller, Sequence: 1, CallerPort: "id", EntryPort: "id"})
			if err != nil {
				t.Fatal(err)
			}
			port.Peer = runtime.HandoffPeer
			if tc.malformed {
				port.Receive = schema.Record{"target": {Kind: "string", MaxBytes: 128}}
			}
			var steps []author.Step
			if tc.shift {
				steps = append(steps, author.Set("offset", author.Integer(1)))
			}
			steps = append(steps, author.Invoke("call", "child", nil, nil), author.Receive("id", map[string]string{"target": "read"}), author.Send("id", map[string]author.Expr{"target": author.Var("read")}), author.AwaitInvocation("child", "call", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"}), author.Success())
			par, err := author.Compile(author.Definition{Ports: map[string]author.Port{"id": port}, Invocations: map[string]author.Template{"child": template}, Steps: steps})
			if err != nil {
				t.Fatal(err)
			}
			app := author.Application{"parent": {Artifact: par}}
			if tc.alias {
				alias, err := author.Compile(author.Definition{Steps: []author.Step{author.Success()}})
				if err != nil {
					t.Fatal(err)
				}
				app[runtime.HandoffPeer] = author.Instance{Artifact: alias}
			}
			p, err := app.Provision()
			if err == nil {
				p.Composition.Roots = []runtime.ResourceRoot{{Run: caller, Members: []runtime.RunID{caller}, Templates: []string{template.ID()}, Limits: runtime.ResourceLimits{MaxInvocations: 1, MaxCreatedRuns: 1, MaxDepth: 1, MaxInFlight: 1}}}
			}
			path := filepath.Join(t.TempDir(), "handoff.sqlite")
			var store *runtime.Store
			if err == nil {
				store, err = runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			}
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("expected %q, got %v", tc.want, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			r, err := store.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			state, err := store.Snapshot(t.Context(), caller)
			if err != nil {
				t.Fatal(err)
			}
			a, err := store.AdmitInvocation(t.Context(), state.Projection.Data["call"].Text)
			if err != nil || !a.Accepted || len(a.Members) != 1 {
				t.Fatal(a, err)
			}
			source, err := store.Snapshot(t.Context(), a.Members[0])
			if err != nil {
				t.Fatal(err)
			}
			if tc.shift {
				if r.RejectedAdmissions != 1 || state.Projection.Status != "open" || source.Projection.Status != "open" || state.Projection.Data["read"].Text != "" || source.Projection.Data["echo"].Text != "" {
					t.Fatal(r, source, state)
				}
			} else if state.Projection.Status != "succeeded" || source.Projection.Data["echo"].Text != "ordinary-probe" {
				t.Fatal(state, source)
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal("replay", err)
			}
		})
	}
}
