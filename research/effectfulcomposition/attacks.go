package effectfulcomposition

// A counterexample is a shortest BFS prefix within the named profile, whose
// initial state represents the explicit prefix described in the domain.
type Counterexample struct {
	Claim, Profile string
	End            int
	Actions        []string
}

func first(g Graph, pred func(State) bool) (int, []string, bool) {
	parent := make([]int, len(g.Rows))
	action := make([]string, len(g.Rows))
	for i := 1; i < len(parent); i++ {
		parent[i] = -1
	}
	for _, r := range g.Rows {
		if pred(r.State) {
			var path []string
			for n := r.ID; n > 0; n = parent[n] {
				path = append(path, action[n])
			}
			for l, h := 0, len(path)-1; l < h; l, h = l+1, h-1 {
				path[l], path[h] = path[h], path[l]
			}
			return r.ID, path, true
		}
		for _, e := range r.Edges {
			if e.To != 0 && parent[e.To] < 0 {
				parent[e.To] = r.ID
				action[e.To] = e.Action
			}
		}
	}
	return 0, nil, false
}
func Attacks(g Graph) []Counterexample {
	out := []Counterexample{}
	attack := func(claim string, pred func(State) bool) {
		if n, path, ok := first(g, pred); ok {
			out = append(out, Counterexample{Claim: claim, Profile: g.Profile.Name, End: n, Actions: path})
		}
	}
	switch g.Profile.Case {
	case "fork-AB-answer-pending-id1":
		attack("adoption guarantees a later successful parent", func(s State) bool {
			return s.Joined == 2 && s.S[0].Witness == 1 && s.S[0].Query[0].Adoption == "adopted" && s.Decision == ""
		})
	case "adopt-duplicate":
		attack("a successful parent requires its last query to adopt", func(s State) bool { return s.Decision == "applied" && s.S[0].Query[1].Adoption == "duplicate" })
	case "order":
		attack("both authentic receipts have mutation=total=1", func(s State) bool { return s.World[0].Role != 0 && s.World[1].Role != 0 && s.World[1].Total != 1 })
	case "read-only":
		attack("found alone resolves its original invocation", func(s State) bool { return s.Joined == 1 && s.S[1].Witness != 0 && !s.S[1].Published })
	case "late-application":
		attack("eventual application guarantees success with finite reads", func(s State) bool {
			return s.Joined == 1 && s.World[1].Role != 0 && s.S[1].Query[1].Status == "not_found"
		})
	case "cap-one", "cap-one-granted":
		attack("semantic allowance guarantees dispatch before issuer End", func(s State) bool { return s.S[1].Mut == "denied" && s.S[1].Closed && s.Cost[1].Post == 0 })
	case "manual-ack":
		attack("completed parent implies both resolvers complete", func(s State) bool { return s.ParentRoot && (s.S[0].Ack == "closed" || s.S[1].Ack == "closed") })
	case "lost-found":
		attack("same business result after recovery preserves exact cost", func(s State) bool {
			return s.Decision == "applied" && s.Cost[1].Read == 1 && s.Cost[1].Ord == 2 && s.Cost[1].Get == 2
		})
	case "competing":
		attack("duplicate selection plus original publication implies original settlement", func(s State) bool {
			return s.Extra.Phase == "selected" && s.Extra.Adoption == "duplicate" && s.S[0].Published && !s.S[0].Settled
		})
	}
	return out
}
