package substitution

import (
	"fmt"
	"testing"
)

// Qualification consumes the independently reviewed reference. It does not
// change its transitions, failure predicates, rank or frozen input partition.
func qualifyInvocationJoin(t *testing.T, variant Variant) {
	t.Helper()
	records := []JoinCensus{}
	for _, config := range JoinConfigurations() {
		t.Run(fmt.Sprintf("id%d/value%d", config.Request.ID, config.Request.Value), func(t *testing.T) {
			result, err := ExploreJoin(config, variant)
			if err != nil {
				t.Error(err)
				return
			}
			records = append(records, result)
			t.Logf("variant=%s states=%d model_states=%d edges=%d revisits=%d maxima=%d ranks=%d..%d/%d", variant, result.States, result.ModelStates, result.Edges, result.Revisits, len(result.Maxima), result.RankMinimum, result.RankMaximum, result.RankCap)
			if result.RankFailures != 0 || result.StateFailures != 0 || result.PrefixFailures != 0 || result.ComponentFailures != 0 || result.GuaranteeFailures != 0 || result.Stuck != 0 || result.Pending != 0 || result.Cycles != 0 || result.Cutoffs != 0 || len(result.Counterexamples) != 0 {
				t.Errorf("qualification failure retained: rank=%d state=%d prefix=%d component=%d G=%d stuck=%d pending=%d cycles=%d cutoffs=%d counterexamples=%+v", result.RankFailures, result.StateFailures, result.PrefixFailures, result.ComponentFailures, result.GuaranteeFailures, result.Stuck, result.Pending, result.Cycles, result.Cutoffs, result.Counterexamples)
			}
			charges := map[int]bool{}
			for _, maximum := range result.Maxima {
				o := maximum.Observation
				charges[o.I] = true
				if maximum.Kind != "terminal" || len(maximum.Guarantee) != 0 || o.I != o.R || o.E != 0 || o.FixedRuns != 1 {
					t.Errorf("maximal result/accounting: %+v", maximum)
				}
			}
			want := []int{2, 3}
			if variant == Local {
				want = []int{2}
			}
			if variant == Helper {
				want = []int{3}
			}
			if len(charges) != len(want) {
				t.Errorf("charges %v, expected %v", charges, want)
			}
			for _, charge := range want {
				if !charges[charge] {
					t.Errorf("missing permitted charge %d", charge)
				}
			}
		})
	}
	// Retain every completed exploration even when a configuration failed. No
	// failing state/configuration is omitted or retried under altered semantics.
	emit(t, "invocation-join-census-"+string(variant), records)
}

func TestInvocationJoinReferenceQualification(t *testing.T) {
	qualifyInvocationJoin(t, Reference)
}

func TestInvocationJoinReplacementQualification(t *testing.T) {
	for _, variant := range []Variant{Local, Helper} {
		t.Run(string(variant), func(t *testing.T) { qualifyInvocationJoin(t, variant) })
	}
}

func TestInvocationJoinLocalRefinementSubset(t *testing.T) {
	type record struct {
		Variant      Variant
		Config       Config
		Relation     RelationResult
		OpenProgress Census
	}
	records := []record{}
	for _, variant := range []Variant{Local, Helper} {
		for _, join := range JoinConfigurations() {
			name := fmt.Sprintf("%s/id%d/value%d", variant, join.Request.ID, join.Request.Value)
			t.Run(name, func(t *testing.T) {
				config := Config{Shape: "local", Request: join.Request, I: 3, R: 3, Depth: 2, Local: true, Peer: true, Closure: true, EntryDepth: 1}
				relation := CheckLocalRelation(config, CandidateGraph(variant))
				open := config
				open.Closure = false
				progress := ExploreGraph(open, CandidateGraph(variant))
				records = append(records, record{Variant: variant, Config: config, Relation: relation, OpenProgress: progress})
				if relation.Initials != 1 || relation.Failures != 0 || len(relation.Counterexamples) != 0 || progress.RankFailures != 0 || progress.Cutoffs != 0 || progress.Stuck != 0 || len(progress.Counterexamples) != 0 {
					t.Errorf("local relation/progress failed: %+v %+v", relation, progress)
				}
				t.Logf("pairs=%d component_checks=%d environment_checks=%d stutters=%d open_states=%d open_edges=%d", relation.Pairs, relation.ComponentChecks, relation.EnvironmentChecks, relation.Stutters, progress.States, progress.Transitions)
			})
		}
	}
	emit(t, "invocation-join-local-subset", records)
}
