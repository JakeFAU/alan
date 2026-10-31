package substitution

import "fmt"

// An accounting map is on original raw state IDs, never on history-lifted IDs.
type weakDemandClaim struct {
	Raw    string
	Demand [3]int
}

func checkWeakDemandMap(claims []weakDemandClaim) []weakIssue {
	seen := map[string][3]int{}
	issues := []weakIssue{}
	for _, claim := range claims {
		if old, ok := seen[claim.Raw]; ok && old != claim.Demand {
			issues = append(issues, weakIssue{
				Clause: "raw_demand_function", Detail: fmt.Sprintf("%s: %v versus %v", claim.Raw, old, claim.Demand),
			})
		} else {
			seen[claim.Raw] = claim.Demand
		}
	}
	return issues
}

// Original prefixes retain all original successor fields and normalized facts.
// This is a transcription/anchor check against the historical model, not an
// independent proof of that model or a new candidate adapter.
func checkWeakOriginalPrefix(c Config, steps []Edge) []weakIssue {
	s := Initial(c)
	for i, e := range steps {
		legal := false
		for _, original := range Successors(c, Reference, s) {
			if original == e {
				legal = true
				break
			}
		}
		if !legal {
			return []weakIssue{{Clause: "original_transition", Detail: fmt.Sprintf("step %d", i), Edge: e.Event.Kind}}
		}
		s = e.Next
	}
	return []weakIssue{}
}
