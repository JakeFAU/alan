package substitution

import "testing"

func TestCandidateProducts(t *testing.T) {
	for _, v := range []Variant{Local, Helper} {
		t.Run(string(v), func(t *testing.T) {
			total := Census{Name: string(v)}
			var records []Census
			for _, c := range Configurations() {
				r := ExploreGraph(c, CandidateGraph(v))
				records = append(records, r)
				total.Configurations++
				total.Entitled += r.Entitled
				total.States += r.States
				total.Transitions += r.Transitions
				total.Revisits += r.Revisits
				total.Rejections += r.Rejections
				total.Terminal += r.Terminal
				total.Pending += r.Pending
				total.Stuck += r.Stuck
				total.RankFailures += r.RankFailures
				total.Cutoffs += r.Cutoffs
				if len(r.Counterexamples) > 0 {
					emit(t, "candidate-first-failure-"+string(v), r)
					t.Fatalf("%s: %+v", r.Name, r.Counterexamples)
				}
			}
			emit(t, "candidate-census-"+string(v), records)
			emit(t, "candidate-summary-"+string(v), total)
			if total.Entitled != 84 {
				t.Fatal(total)
			}
			t.Logf("%+v", total)
		})
	}
}
func TestCandidateLocalRelations(t *testing.T) {
	for _, v := range []Variant{Local, Helper} {
		t.Run(string(v), func(t *testing.T) {
			result := RelationResult{}
			progress := Census{Name: string(v) + "-local-no-closure"}
			for _, c := range LocalConfigurations() {
				r := CheckLocalRelation(c, CandidateGraph(v))
				result.Initials += r.Initials
				result.Pairs += r.Pairs
				result.ComponentChecks += r.ComponentChecks
				result.EnvironmentChecks += r.EnvironmentChecks
				result.Stutters += r.Stutters
				result.Failures += r.Failures
				if r.Failures > 0 {
					emit(t, "local-first-failure-"+string(v), r)
					t.Fatalf("%+v %+v", c, r)
				}
				c.Closure = false
				p := ExploreGraph(c, CandidateGraph(v))
				progress.Configurations++
				progress.Entitled += p.Entitled
				progress.States += p.States
				progress.Transitions += p.Transitions
				progress.Revisits += p.Revisits
				progress.Rejections += p.Rejections
				progress.Terminal += p.Terminal
				progress.Pending += p.Pending
				progress.Stuck += p.Stuck
				progress.RankFailures += p.RankFailures
				if len(p.Counterexamples) > 0 {
					emit(t, "local-progress-failure-"+string(v), p)
					t.Fatalf("%+v %+v", c, p.Counterexamples)
				}
			}
			emit(t, "local-relation-"+string(v), result)
			emit(t, "local-progress-"+string(v), progress)
			t.Logf("%+v", result)
		})
	}
}
