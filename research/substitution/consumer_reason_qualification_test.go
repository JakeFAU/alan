package substitution

import (
	"fmt"
	"testing"
)

// Qualification starts only after the source/contract/bounds freeze APPROVE.
// Reference, replacements and local checks remain independently selectable.
func qualifyConsumerReason(t *testing.T, variant Variant) {
	t.Helper()
	type record struct {
		ConsumerReasonCensus
		GAnswerChecked  bool
		GAnswerFailures int
	}
	records := []record{}
	for _, c := range ConsumerReasonConfigurations() {
		t.Run(fmt.Sprintf("id%d/value%d/%s", c.Request.ID, c.Request.Value, c.Request.Mode), func(t *testing.T) {
			census, err := ExploreConsumerReason(c, variant)
			if err != nil {
				t.Error(err)
				return
			}
			r := record{ConsumerReasonCensus: census, GAnswerChecked: c.Request.Mode == "answer"}
			badInvariant := census.RankFailures != 0 || census.StateFailures != 0 || census.PrefixFailures != 0
			badProperty := census.ComponentFailures != 0 || census.GuaranteeFailures != 0
			incomplete := census.Stuck != 0 || census.Cycles != 0 || census.Cutoffs != 0 || len(census.Maxima) == 0
			if !census.Entitled || badInvariant || badProperty || incomplete || len(census.Counterexamples) != 0 {
				t.Errorf("qualification failure retained: entitled=%t invariant=%t property=%t incomplete=%t counterexamples=%d",
					census.Entitled, badInvariant, badProperty, incomplete, len(census.Counterexamples))
			}
			charges := map[int]bool{}
			wantKind := "terminal"
			if c.Request.Mode == "pending" {
				wantKind = "pending"
			}
			for _, maximum := range census.Maxima {
				o := maximum.Observation
				charges[o.I] = true
				if maximum.Kind != wantKind || len(maximum.Guarantee) != 0 || o.I != o.R || o.E != 0 || o.FixedRuns != 1 {
					t.Errorf("wrong maximum kind/property/accounting: kind=%s observation=%+v failures=%+v",
						maximum.Kind, o, maximum.Guarantee)
				}
				if r.GAnswerChecked {
					// G_answer forgets the additional classification event only;
					// original envelopes, selections, decisions and order are intact.
					projection := []JoinEvent{}
					for _, event := range maximum.Trace {
						if event.Fact.Kind != "parent_classify" {
							projection = append(projection, event.JoinEvent)
						}
					}
					if failures := CheckConsumerPolicyGuarantee(c, projection); len(failures) != 0 {
						r.GAnswerFailures++
						t.Errorf("answer maximum does not preserve G_answer: %+v", failures)
					}
				}
			}
			wantCharges := []int{2, 3}
			if variant == Local {
				wantCharges = []int{2}
			} else if variant == Helper {
				wantCharges = []int{3}
			}
			if len(charges) != len(wantCharges) {
				t.Errorf("permitted branch charges %v, want %v", charges, wantCharges)
			}
			for _, want := range wantCharges {
				if !charges[want] {
					t.Errorf("missing permitted branch charge%d", want)
				}
			}
			records = append(records, r)
			t.Logf("variant=%s states=%d model_states=%d edges=%d maxima=%d terminal=%d pending=%d rank=%d..%d/%d G_answer_checked=%t",
				variant, census.States, census.ModelStates, census.Edges, len(census.Maxima), census.Terminal,
				census.Pending, census.RankMinimum, census.RankMaximum, census.RankCap, r.GAnswerChecked)
		})
	}
	// All records, including failures, are preserved without filtering/retries.
	emit(t, "consumer-reason-census-"+string(variant), records)
}

func TestConsumerReasonReferenceQualification(t *testing.T) {
	qualifyConsumerReason(t, Reference)
}

func TestConsumerReasonReplacementQualification(t *testing.T) {
	for _, variant := range []Variant{Local, Helper} {
		t.Run(string(variant), func(t *testing.T) { qualifyConsumerReason(t, variant) })
	}
}

func TestConsumerReasonLocalQualification(t *testing.T) {
	records := []ConsumerReasonLocalQualification{}
	for _, variant := range []Variant{Local, Helper} {
		for _, c := range ConsumerReasonConfigurations() {
			name := fmt.Sprintf("%s/id%d/value%d/%s", variant, c.Request.ID, c.Request.Value, c.Request.Mode)
			t.Run(name, func(t *testing.T) {
				r, err := QualifyConsumerReasonLocal(c, variant)
				if err != nil {
					t.Error(err)
					return
				}
				records = append(records, r)
				if !r.Entitled || len(r.Failures) != 0 {
					t.Errorf("all-mode local relation/progress failed: %+v", r.Failures)
				}
				t.Logf("pairs=%d component_checks=%d environment_checks=%d stutters=%d states=%d edges=%d maxima=%d terminal=%d pending=%d rank=%d..%d/%d",
					r.Relation.Pairs, r.Relation.ComponentChecks, r.Relation.EnvironmentChecks, r.Relation.Stutters,
					r.Progress.States, r.Progress.Transitions, len(r.Maxima), r.Progress.Terminal, r.Progress.Pending,
					r.RankMinimum, r.RankMaximum, r.RankCap)
			})
		}
	}
	emit(t, "consumer-reason-local-all-modes", records)
}
