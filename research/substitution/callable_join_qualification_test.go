package substitution

import (
	"fmt"
	"testing"
)

func qualifyCallableJoinProduct(t *testing.T, variant Variant) {
	t.Helper()
	records := []CallableJoinCensus{}
	for _, c := range CallableJoinConfigurations() {
		t.Run(fmt.Sprintf("id%d/value%d/%s", c.Request.ID, c.Request.Value, c.Request.Mode), func(t *testing.T) {
			r, err := ExploreCallableJoin(c, variant)
			if err != nil {
				t.Error(err)
				return
			}
			records = append(records, r)
			badInvariant := r.StateFailures != 0 || r.RankFailures != 0 || r.PrefixFailures != 0
			badProperty := r.ComponentFailures != 0 || r.ProjectionFailures != 0 ||
				r.ContractFailures != 0 || r.OuterFailures != 0
			incomplete := r.Stuck != 0 || r.Cutoffs != 0 || r.Cycles != 0 || len(r.Maxima) == 0
			if !CallableJoinEntitled(c) || badInvariant || badProperty || incomplete || len(r.Counterexamples) != 0 {
				t.Errorf("retained product failure: invariant=%t property=%t incomplete=%t counterexamples=%d",
					badInvariant, badProperty, incomplete, len(r.Counterexamples))
			}
			if r.Rejections != 0 {
				t.Errorf("entitled product retained %d refusals", r.Rejections)
			}
			charges := map[int]bool{}
			for _, m := range r.Maxima {
				kind := "terminal"
				if c.Request.Mode == "pending" {
					kind = "pending"
				}
				if m.Kind != kind || len(m.Contract) != 0 || len(m.Outer) != 0 {
					t.Errorf("wrong maximum: kind=%s contract=%v outer=%v", m.Kind, m.Contract, m.Outer)
				}
				s := m.State
				charges[1+s.Inner.Component.I] = true
				if !s.InvocationAdmitted || s.Inner.Component.I != s.Inner.Component.R || s.Inner.Component.E != 0 {
					t.Error("maximum lost prior J charge or root accounting")
				}
			}
			want := []int{3, 4}
			if variant == Local {
				want = []int{3}
			} else if variant == Helper {
				want = []int{4}
			}
			if len(charges) != len(want) {
				t.Errorf("wrong independent branch charges: %v", charges)
			}
			for _, n := range want {
				if !charges[n] {
					t.Errorf("missing permitted charge branch %d", n)
				}
			}
			t.Logf("variant=%s states=%d model_states=%d edges=%d terminal=%d pending=%d rank=%d..%d/%d",
				variant, r.States, r.ModelStates, r.Edges, r.Terminal, r.Pending, r.RankMinimum, r.RankMaximum, r.RankCap)
		})
	}
	emit(t, "callable-join-product-"+string(variant), records)
}

func TestCallableJoinReferenceQualification(t *testing.T) {
	qualifyCallableJoinProduct(t, Reference)
}

func TestCallableJoinReplacementQualification(t *testing.T) {
	for _, variant := range []Variant{Local, Helper} {
		t.Run(string(variant), func(t *testing.T) { qualifyCallableJoinProduct(t, variant) })
	}
}

func TestCallableJoinAlignedLocalQualification(t *testing.T) {
	records := []CallableJoinLocalQualification{}
	for _, variant := range []Variant{Local, Helper} {
		for _, c := range CallableJoinConfigurations() {
			name := fmt.Sprintf("%s/id%d/value%d/%s", variant, c.Request.ID, c.Request.Value, c.Request.Mode)
			t.Run(name, func(t *testing.T) {
				r, err := QualifyCallableJoinLocal(c, variant)
				if err != nil {
					t.Error(err)
					return
				}
				records = append(records, r)
				if len(r.Failures) != 0 || r.Progress.Rejections != 0 {
					t.Errorf("retained aligned relation/progress failures: %v", r.Failures)
				}
				t.Logf("pairs=%d component=%d environment=%d stutters=%d states=%d edges=%d maxima=%d rank=%d..%d/%d",
					r.Relation.Pairs, r.Relation.ComponentChecks, r.Relation.EnvironmentChecks, r.Relation.Stutters,
					r.Progress.States, r.Progress.Transitions, len(r.Maxima), r.RankMinimum, r.RankMaximum, r.RankCap)
			})
		}
	}
	emit(t, "callable-join-aligned-local", records)
}

func TestCallableJoinFiniteLiftingQualification(t *testing.T) {
	type record struct {
		Config   CallableJoinConfig
		Variant  Variant
		Relation RelationResult
	}
	records := []record{}
	for _, variant := range []Variant{Local, Helper} {
		for _, c := range CallableJoinConfigurations() {
			name := fmt.Sprintf("%s/id%d/value%d/%s", variant, c.Request.ID, c.Request.Value, c.Request.Mode)
			t.Run(name, func(t *testing.T) {
				r, err := CheckCallableJoinLifting(c, variant)
				if err != nil {
					t.Error(err)
					return
				}
				records = append(records, record{Config: c, Variant: variant, Relation: r})
				if r.Initials != 1 || r.Failures != 0 || len(r.Counterexamples) != 0 {
					t.Errorf("retained finite lifting failure: %+v", r)
				}
				t.Logf("pairs=%d component=%d environment=%d stutters=%d", r.Pairs,
					r.ComponentChecks, r.EnvironmentChecks, r.Stutters)
			})
		}
	}
	emit(t, "callable-join-finite-lifting", records)
}
