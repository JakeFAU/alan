package substitution

import "testing"

// This is a direct frozen-contract oracle, not a comparison of two interpreters
// sharing protocol transitions: helper data is never a candidate publication.
func TestModeResponseAbsenceIndependent(t *testing.T) {
	for _, mode := range []string{"fail", "absent", "pending"} {
		t.Run(mode, func(t *testing.T) {
			c := Config{Shape: "bare", Request: Request{1, 0, mode}, I: 3, R: 3, Depth: 3}
			for _, v := range []Variant{Reference, Helper} {
				graph := CandidateGraph(v)
				if v == Reference {
					graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
				}
				queue := []State{Initial(c)}
				seen := map[State]bool{}
				for i := 0; i < len(queue); i++ {
					s := queue[i]
					if seen[s] {
						continue
					}
					seen[s] = true
					o := s.Ops[entry]
					if o.Present || o.ID != 0 || o.Value != 0 {
						t.Fatalf("%s %s has response without permitted answer publication: %+v", v, mode, o)
					}
					edges := graph(c, s)
					if len(edges) == 0 {
						switch mode {
						case "fail":
							if !o.Published || o.Status != "failed" || o.Reason != "member_failed" {
								t.Fatalf("fail envelope %+v", o)
							}
						case "absent":
							if !o.Published || o.Status != "failed" || o.Reason != "missing_reply" {
								t.Fatalf("absent envelope %+v", o)
							}
						case "pending":
							if o.Published || o.Freeze != none || o.PC != pending {
								t.Fatalf("pending is not a failed/absent envelope: %+v", o)
							}
						}
					}
					for _, e := range edges {
						queue = append(queue, e.Next)
					}
				}
			}
		})
	}
}
