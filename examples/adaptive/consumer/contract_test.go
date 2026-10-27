package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type observed struct{ paired bool }

func (a observed) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "controlled", MaxResponseBytes: 8192}, nil
}
func (a observed) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	v := p.Request.Payload["value"].Number
	if a.paired && v == 2 {
		v = 0
	}
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"observation": schema.Int(v)}}
}
func TestAdaptivePublicConsumerDirectNested(t *testing.T) {
	raw, e := os.ReadFile("../../../docs/foundational/phase1h/evidence/fixtures/expected-work.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Cases []struct {
			Case           string `json:"case"`
			OrderedResults []struct {
				ID          int64  `json:"work_id"`
				Value       int64  `json:"value"`
				Observation int64  `json:"observation"`
				Round       int64  `json:"round"`
				Parent      *int64 `json:"predecessor"`
			} `json:"ordered_results"`
			IDs          []int64   `json:"ordered_result_ids"`
			Inputs       []int64   `json:"ordered_result_inputs"`
			Observations [][]int64 `json:"round_observations"`
			Charges      struct {
				Direct, Nested struct {
					Invocations int64 `json:"accepted_invocations"`
					Runs        int64 `json:"created_runs_scalar"`
				}
			} `json:"charges"`
		}
	}
	if e = json.Unmarshal(raw, &fixture); e != nil {
		t.Fatal(e)
	}
	leaf, e := adaptive.ScalarLeaf(true)
	if e != nil {
		t.Fatal(e)
	}
	component, e := adaptive.NewScalar(leaf)
	if e != nil {
		t.Fatal(e)
	}
	for _, nested := range []bool{false, true} {
		for _, capacity := range []int{1, 2} {
			for _, paired := range []bool{false, true} {
				t.Run(fmt.Sprintf("nested=%v/capacity=%d/paired=%v", nested, capacity, paired), func(t *testing.T) {
					d, e := adaptive.New(component, nested, adaptive.DefaultLimits(int64(capacity)))
					if e != nil {
						t.Fatal(e)
					}
					if d.Adaptive.ID() != component.ID() {
						t.Fatal("adaptive identity changed")
					}
					path := filepath.Join(t.TempDir(), "consumer.sqlite")
					s, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
					if e != nil {
						t.Fatal(e)
					}
					defer s.Close()
					seed := schema.List(map[string]schema.Value{"work_id": schema.Int(1), "value": schema.Int(0)}, map[string]schema.Value{"work_id": schema.Int(2), "value": schema.Int(1)}, map[string]schema.Value{"work_id": schema.Int(3), "value": schema.Int(2)})
					a, e := adaptive.Submit(t.Context(), s, "work", map[string]schema.Value{"seeds": seed, "max_rounds": schema.Int(3), "max_work": schema.Int(8)})
					if e != nil || !a.Accepted {
						t.Fatal(a, e)
					}
					report, e := s.Run(t.Context(), runtime.RunOptions{Adapters: map[string]effects.Adapter{adaptive.ScalarCapability: observed{paired}}, MaxSteps: 20000, MaxDispatch: 2000, MaxInFlight: capacity})
					if e != nil {
						t.Fatal(e)
					}
					root, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
					if e != nil {
						t.Fatal(e)
					}
					results, e := root.Projection.Data["results"].Records()
					if e != nil {
						t.Fatal(e)
					}
					ci := 0
					if paired {
						ci = 1
					}
					f := fixture.Cases[ci]
					wantIDs := f.IDs
					wantInputs := f.Inputs
					observations := []int64{}
					for _, r := range f.Observations {
						observations = append(observations, r...)
					}
					if !paired {
						for _, r := range f.OrderedResults {
							wantIDs = append(wantIDs, r.ID)
							wantInputs = append(wantInputs, r.Value)
						}
					}
					if len(results) != len(wantIDs) {
						t.Fatal("frozen ledger result count")
					}
					operations := map[string]bool{}
					for i, r := range results {
						responses, e := r["responses"].Records()
						if e != nil || len(responses) != 1 {
							t.Fatal("response absent", e)
						}
						op := r["operation"].Text
						if r["work_id"].Number != wantIDs[i] || r["value"].Number != wantInputs[i] || responses[0]["observation"].Number != observations[i] || operations[op] || op == "" {
							t.Fatal("independent fixture mismatch", i, r)
						}
						operations[op] = true
					}
					charges := f.Charges.Direct
					if nested {
						charges = f.Charges.Nested
					}
					if report.Roots[0].Charges.Invocations != charges.Invocations || report.Roots[0].Charges.CreatedRuns != charges.Runs || report.Roots[0].Charges.Effects != int64(len(wantIDs)) {
						t.Fatal("frozen charge ledger mismatch")
					}
					if root.Projection.Data["stop_reason"].Text != "exhausted" || !root.Projection.Data["complete"].Flag {
						t.Fatal("frozen completion mismatch")
					}
					inspection, e := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
					if e != nil {
						t.Fatal(e)
					}
					if len(inspection.Tables["composition_results"]) != len(results)+1+map[bool]int{false: 0, true: 1}[nested] {
						t.Fatal("public outcome inspection incomplete")
					}
					if _, e = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); e != nil {
						t.Fatal(e)
					}
				})
			}
		}
	}
}
