package author_test

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// The local control has no external world. It deliberately leaves both original
// requests unresolved, so an invocation reply cannot supply the tested identity.
type unresolvedHandoffEffect struct{}

func (unresolvedHandoffEffect) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "test/unknown", MaxResponseBytes: 256}, nil
}
func (unresolvedHandoffEffect) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
}

type handoffBuilder func(author.Application, string, string, runtime.InvocationHandoff) (author.Template, error)
type handoffEdge struct {
	name, port, ob string
	at             int64
	build          handoffBuilder
}

func pairedHandoffProvision(t *testing.T, edges []handoffEdge) (runtime.Provision, error) {
	t.Helper()
	target := schema.Record{"target": {Kind: "string", MaxBytes: 1024}}
	parent := runtime.RunID{Subject: "owner", Generation: 1}
	ports := map[string]author.Port{}
	invocations := map[string]author.Template{}
	var calls, dialogue []author.Step
	var templates []string
	for _, e := range edges {
		port := author.Port{Peer: "caller", Obligation: e.ob, Send: target, Receive: target}
		child, err := author.Compile(author.Definition{Ports: map[string]author.Port{"entry": {Peer: "caller", Obligation: "issue"}, "handoff": port}, Effects: map[string]author.Effect{"original": {Capability: "unknown", Input: schema.Record{}, Output: schema.Record{}}}, Steps: []author.Step{
			author.Receive("entry", nil), author.Request("original_id", "original", nil, nil),
			author.Send("handoff", map[string]author.Expr{"target": author.Var("original_id")}),
			author.Receive("handoff", map[string]string{"target": "selected_ack"}), author.Send("entry", nil), author.Success(),
		}})
		if err != nil {
			return runtime.Provision{}, err
		}
		h := runtime.InvocationHandoff{Caller: parent, Sequence: e.at, CallerPort: e.port, EntryPort: "handoff"}
		template, err := e.build(author.Application{"source": {Artifact: child, Inputs: map[string]schema.Value{}}}, "source", "entry", h)
		if err != nil {
			return runtime.Provision{}, err
		}
		port.Peer = runtime.HandoffPeer
		ports[e.port] = port
		invocations[e.name] = template
		templates = append(templates, template.ID())
		calls = append(calls, author.Invoke("call_"+e.name, e.name, nil, nil))
		dialogue = append(dialogue, author.Receive(e.port, map[string]string{"target": "id_" + e.name}), author.Send(e.port, map[string]author.Expr{"target": author.Var("id_" + e.name)}))
	}
	// Waiting for the first actual outcome intentionally keeps the parent open.
	steps := append(calls, dialogue...)
	steps = append(steps, author.AwaitInvocation(edges[0].name, "call_"+edges[0].name, author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"}), author.Success())
	p, err := author.Compile(author.Definition{Ports: ports, Invocations: invocations, Steps: steps})
	if err != nil {
		return runtime.Provision{}, err
	}
	provision, err := (author.Application{"owner": {Artifact: p}}).Provision()
	if err != nil {
		return provision, err
	}
	provision.Composition.Roots = []runtime.ResourceRoot{{Run: parent, Members: []runtime.RunID{parent}, Templates: templates, Capabilities: []string{"unknown"}, Limits: runtime.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxEffects: 2, MaxInFlight: 1, MaxDepth: 1}}}
	return provision, nil
}

func pairedEdges() []handoffEdge {
	return []handoffEdge{{"left", "left_id", "left_identity", 1, author.NewPairedHandoffTemplate}, {"right", "right_id", "right_identity", 2, author.NewPairedHandoffTemplate}}
}
func TestPhase2DPublicPairedHandoff(t *testing.T) {
	edges := pairedEdges()
	p, err := pairedHandoffProvision(t, edges)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pair.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal("public two-edge Create:", err)
	}
	defer s.Close()
	_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"unknown": unresolvedHandoffEffect{}}})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "owner", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, e := range edges {
		call := parent.Projection.Data["call_"+e.name].Text
		a, err := s.AdmitInvocation(t.Context(), call)
		if err != nil || !a.Accepted || len(a.Members) != 1 {
			t.Fatal(a, err)
		}
		child, err := s.Snapshot(t.Context(), a.Members[0])
		if err != nil {
			t.Fatal(err)
		}
		id := child.Projection.Data["original_id"].Text
		if id == "" || ids[id] || parent.Projection.Data["id_"+e.name].Text != id || child.Projection.Data["selected_ack"].Text != id || child.Projection.Status != "succeeded" {
			t.Fatalf("invalid %s handoff: %s / %s", e.name, id, child.Projection.Status)
		}
		ids[id] = true
		join, err := s.PrepareInvocation(t.Context(), call)
		if err != nil || join.Ready || join.Pending != "external_outcome" || len(join.Freeze.ExternalIntents) != 1 || join.Freeze.ExternalIntents[0] != id {
			t.Fatal(join, err)
		}
		t.Log(e.name, call, id, child.Projection.Data["selected_ack"].Text, join.Pending)
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestPhase2DHandoffDeclarations(t *testing.T) {
	for _, name := range []string{"one-v2", "three-v2", "two-v1", "mixed", "same-sequence", "same-port", "same-obligation", "wrong-caller", "different-entry-obligation", "stale-revision", "ambient-placeholder-grant", "unknown-version", "other-root"} {
		t.Run(name, func(t *testing.T) {
			edges := pairedEdges()
			switch name {
			case "one-v2":
				edges = edges[:1]
			case "three-v2":
				edges = append(edges, handoffEdge{"third", "third_id", "third_identity", 3, author.NewPairedHandoffTemplate})
			case "two-v1":
				edges[0].build = author.NewHandoffTemplate
				edges[1].build = author.NewHandoffTemplate
			case "mixed":
				edges[1].build = author.NewHandoffTemplate
			case "same-sequence":
				edges[1].at = 1
			case "same-port":
				edges[1].port = edges[0].port
			case "same-obligation":
				edges[1].ob = edges[0].ob
			case "wrong-caller":
				edges[1].build = func(a author.Application, m, p string, h runtime.InvocationHandoff) (author.Template, error) {
					h.Caller.Subject = "other"
					return author.NewPairedHandoffTemplate(a, m, p, h)
				}
			case "different-entry-obligation":
				edges[1].build = func(a author.Application, m, p string, h runtime.InvocationHandoff) (author.Template, error) {
					h.EntryPort = p
					return author.NewPairedHandoffTemplate(a, m, p, h)
				}
			}
			p, err := pairedHandoffProvision(t, edges)
			if err == nil {
				switch name {
				case "stale-revision":
					p.Subjects[0].Revision = 2
				case "ambient-placeholder-grant":
					p.Permissions = append(p.Permissions, runtime.Permission{Kind: "message", Source: "owner", Target: runtime.HandoffPeer, Revision: 1, Obligation: "left_identity", Operation: "message"})
				case "unknown-version":
					p.Composition.Templates[0].Version = "alan/identity-handoff-template/v3"
				case "other-root":
					p.Composition.Roots[0].Templates = p.Composition.Roots[0].Templates[:1]
				}
				s, e := runtime.Create(t.Context(), filepath.Join(t.TempDir(), "pair.sqlite"), p, runtime.DefaultLimits())
				if e == nil {
					s.Close()
					t.Fatal("unsupported handoff configuration accepted")
				}
				err = e
			}
			if name == "unknown-version" && !errors.Is(err, runtime.ErrVersion) {
				t.Fatalf("incompatible version: %v", err)
			}
			t.Log(err)
		})
	}
}

func handoffGrants(t *testing.T, path string) map[[3]string]bool {
	t.Helper()
	got, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	out := map[[3]string]bool{}
	for _, g := range got.Tables["permissions"] {
		if g["kind"] != "message" {
			continue
		}
		if g["revision"] != int64(1) || g["operation"] != "message" {
			t.Fatalf("invalid grant %+v", g)
		}
		k := [3]string{g["source"].(string), g["target"].(string), g["obligation"].(string)}
		out[k] = true
	}
	return out
}
func TestPhase2DHandoffAdmissionGrants(t *testing.T) {
	for _, swapped := range []bool{false, true} {
		name := "matched"
		if swapped {
			name = "same-member-wrong-template"
		}
		t.Run(name, func(t *testing.T) {
			edges := pairedEdges()
			if swapped {
				edges[0].at, edges[1].at = edges[1].at, edges[0].at
			}
			p, err := pairedHandoffProvision(t, edges)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "pair.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			parent, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "owner", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			for range edges {
				parent, err = s.Commit(t.Context(), parent.Authority(), "")
				if err != nil {
					t.Fatal(err)
				}
			}
			want := map[[3]string]bool{}
			for i, e := range edges {
				if got := handoffGrants(t, path); !reflect.DeepEqual(got, want) {
					t.Fatalf("before admission %d: %v != %v", i, got, want)
				}
				call := parent.Projection.Data["call_"+e.name].Text
				a, err := s.AdmitInvocation(t.Context(), call)
				if err != nil || !a.Accepted || len(a.Members) != 1 {
					t.Fatal(a, err)
				}
				if !swapped {
					child := a.Members[0].Subject
					want[[3]string{"owner", child, e.ob}] = true
					want[[3]string{child, "owner", e.ob}] = true
				}
				if got := handoffGrants(t, path); !reflect.DeepEqual(got, want) {
					t.Fatalf("after admission %d: %v != %v", i, got, want)
				}
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if !swapped {
				// Relabel genuine grants on a disposable store. The corresponding
				// original template/operation/admission facts remain unchanged.
				db, err := sql.Open("sqlite3", path)
				if err != nil {
					t.Fatal(err)
				}
				change, err := db.Exec("UPDATE permissions SET obligation='right_identity' WHERE source='owner' AND obligation='left_identity'")
				if err != nil {
					t.Fatal(err)
				}
				n, err := change.RowsAffected()
				if err != nil || n != 1 {
					t.Fatal(n, err)
				}
				if err = db.Close(); err != nil {
					t.Fatal(err)
				}
				if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err == nil || !strings.Contains(err.Error(), "permissions") {
					t.Fatalf("accepted altered grant or wrong failure: %v", err)
				}
			}
		})
	}
}

// This is a real schema-5 original-operation Store produced by the v1 handoff
// implementation. Offline replay of its old source, observations and grants
// must not require adding a second template or changing its persisted bytes.
func TestPhase2DHistoricalSingleHandoff(t *testing.T) {
	fixture := filepath.Join("..", "docs", "foundational", "phase2c", "evidence", "process-normal", "atomic-import", "final-local.sqlite.gz")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v1.sqlite")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(raw) != sha256.Sum256(after) {
		t.Fatal("historical bytes changed", err)
	}
	s, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "parent", Generation: 1})
	if err != nil || got.Projection.Status != "succeeded" || got.Projection.Data["decision"].Text != "applied" {
		t.Fatal(got, err)
	}
}
