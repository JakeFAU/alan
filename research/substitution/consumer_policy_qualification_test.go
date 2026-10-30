package substitution

import (
	"fmt"
	"testing"
)

// Qualification uses the independently approved C_se reference unchanged. Keep
// reference and replacement entry points separate so K1 can be checked first.
func qualifyConsumerPolicy(t *testing.T, variant Variant) {
	t.Helper()
	records := []ConsumerPolicyCensus{}
	for _, c := range ConsumerPolicyConfigurations() {
		t.Run(fmt.Sprintf("id%d/value%d", c.Request.ID, c.Request.Value), func(t *testing.T) {
			result, err := ExploreConsumerPolicy(c, variant)
			if err != nil {
				t.Error(err)
				return
			}
			records = append(records, result)
			t.Logf("context=%s variant=%s states=%d model_states=%d edges=%d maxima=%d rank=%d..%d/%d",
				result.Context, variant, result.States, result.ModelStates, result.Edges,
				len(result.Maxima), result.RankMinimum, result.RankMaximum, result.RankCap)
			badInvariant := result.RankFailures != 0 || result.StateFailures != 0 || result.PrefixFailures != 0
			badProperty := result.ComponentFailures != 0 || result.GuaranteeFailures != 0
			incomplete := result.Stuck != 0 || result.Pending != 0 || result.Cycles != 0 || result.Cutoffs != 0
			if badInvariant || badProperty || incomplete || len(result.Counterexamples) != 0 {
				t.Errorf("failure preserved: %+v", result)
			}
			charges := map[int]bool{}
			for _, maximum := range result.Maxima {
				o := maximum.Observation
				charges[o.I] = true
				badAccounting := o.I != o.R || o.E != 0 || o.FixedRuns != 1
				if maximum.Kind != "terminal" || len(maximum.Guarantee) != 0 || badAccounting {
					t.Errorf("maximal result/accounting: %+v", maximum)
				}
			}
			want := []int{2, 3}
			if variant == Local {
				want = []int{2}
			} else if variant == Helper {
				want = []int{3}
			}
			if len(charges) != len(want) {
				t.Errorf("permitted branch charges %v, want %v", charges, want)
			}
			for _, charge := range want {
				if !charges[charge] {
					t.Errorf("missing permitted branch charge%d", charge)
				}
			}
		})
	}
	// Emit every result, including failures; never turn a failing schedule into
	// an excluded configuration or a modified reference premise.
	emit(t, "consumer-policy-census-"+string(variant), records)
}

func TestConsumerPolicyReferenceQualification(t *testing.T) {
	qualifyConsumerPolicy(t, Reference)
}

func TestConsumerPolicyReplacementQualification(t *testing.T) {
	for _, variant := range []Variant{Local, Helper} {
		t.Run(string(variant), func(t *testing.T) { qualifyConsumerPolicy(t, variant) })
	}
}
