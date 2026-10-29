package substitution

import "testing"

func TestSeparateK1EProfiles(t *testing.T) {
	var results []EffectCensus
	for _, scenario := range []string{"known", "unknown-reopen", "closure-before-reservation", "reserved-then-closure", "late-live"} {
		t.Run(scenario, func(t *testing.T) {
			r := ExploreEffect(scenario)
			results = append(results, r)
			if len(r.Violations) > 0 || r.RankFailures != 0 {
				t.Fatalf("%+v", r)
			}
			if len(r.Outcomes) != 1 {
				t.Fatal(r)
			}
			last := r.Outcomes[0]
			switch scenario {
			case "unknown-reopen":
				if !last.Unknown || !last.Reopened || last.E != 1 || last.Published {
					t.Fatal(last)
				}
			case "closure-before-reservation":
				if !last.Authorized || last.Reserved || last.Launched || last.E != 0 {
					t.Fatal(last)
				}
			case "reserved-then-closure", "late-live":
				if !last.Closed || !last.Reserved || !last.Launched || !last.Known {
					t.Fatal(last)
				}
			}
		})
	}
	emit(t, "separate-effect-profiles", results)
}
