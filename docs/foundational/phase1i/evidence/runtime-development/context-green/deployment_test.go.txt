package substitution_test

import (
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This local fixture tests context wiring only. It is neither independent A/B
// nor the semantic contract oracle; the input used below always has value zero.
func replyFixture(t *testing.T, reply bool) author.Template {
	t.Helper()
	steps := []author.Step{author.Receive(protocol.Port, map[string]string{"id": "id"})}
	if reply {
		steps = append(steps, author.Send(protocol.Port, map[string]author.Expr{
			"id": author.Var("id"), "value": author.Integer(1),
		}))
	}
	steps = append(steps, author.Success())
	a, err := author.Compile(author.Definition{
		Ports: map[string]author.Port{protocol.Port: {
			Peer: protocol.Peer, Obligation: protocol.Obligation,
			Receive: protocol.Request(), Send: protocol.Response(),
		}},
		Steps: steps,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := author.NewTemplate(
		author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}},
		"entry", protocol.Port,
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDeploymentContexts(t *testing.T) {
	for _, shape := range []string{"bare", "sequential", "parallel", "hidden", "wrapped"} {
		for _, reply := range []bool{true, false} {
			name := shape + "/present"
			if !reply {
				name = shape + "/absent"
			}
			t.Run(name, func(t *testing.T) {
				config := substitution.Config{
					Shape: shape, Request: substitution.Request{ID: 2, Value: 0, Mode: "answer"},
					Limits: runtime.ResourceLimits{
						MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 3, MaxInFlight: 1,
					},
				}
				d, err := substitution.New(replyFixture(t, reply), config)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "context.sqlite")
				s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = s.Close() })
				report, err := s.Run(t.Context(), runtime.RunOptions{
					MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1,
				})
				if err != nil {
					t.Fatal(err)
				}
				root, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
				if err != nil {
					t.Fatal(err)
				}
				wantStatus := "failed"
				if reply {
					wantStatus = "succeeded"
					if root.Projection.Data["answer_id"].Number != 2 || root.Projection.Data["answer_value"].Number != 1 {
						t.Fatal("context changed the candidate reply", root.Projection.Data)
					}
				}
				if root.Projection.Status != wantStatus {
					t.Fatalf("root status %q, want %q", root.Projection.Status, wantStatus)
				}
				wantInvocations := int64(1)
				if shape == "parallel" || shape == "wrapped" || (shape == "sequential" && reply) {
					wantInvocations++
				}
				if len(report.Roots) != 1 || report.Roots[0].Charges.Invocations != wantInvocations ||
					report.Roots[0].Charges.CreatedRuns != wantInvocations || report.Roots[0].Charges.Effects != 0 {
					t.Fatalf("context charges: %+v, want %d", report.Roots, wantInvocations)
				}
				wantFixed := 1
				if shape == "parallel" {
					wantFixed = 2
					peer, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "peer", Generation: 1})
					if err != nil || peer.Projection.Status != "succeeded" {
						t.Fatal("independent peer did not complete", peer, err)
					}
				}
				if d.FixedRuns != wantFixed || len(d.Provision.Runs) != wantFixed {
					t.Fatal("fixed provisioning cost", d.FixedRuns, len(d.Provision.Runs))
				}
				if _, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
