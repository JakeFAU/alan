package substitution

import "testing"

func TestNamedClosureReferenceBoundaries(t *testing.T) {
	type record struct {
		Name       string
		Applicable bool
		Result     ProbeResult
	}
	var records []record
	boundaries := []string{"entry-commit", "entry-admitted", "helper-commit", "reply-commit", "published"}
	for _, v := range []Variant{Local, Helper} {
		for _, candidate := range []bool{false, true} {
			for _, boundary := range boundaries {
				label := "reference/"
				if candidate {
					label = "artifact/"
				}
				label += string(v) + "/" + boundary
				if boundary == "helper-commit" && v == Local {
					records = append(records, record{Name: label, Applicable: false})
					continue
				}
				c := Config{Shape: "bare", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 3}
				base := Graph(func(c Config, s State) []Edge { return Successors(c, v, s) })
				if candidate {
					base = CandidateGraph(v)
				}
				graph := func(c Config, s State) []Edge {
					reached := false
					switch boundary {
					case "entry-commit":
						reached = s.Ops[entry].Ticket == committed
					case "entry-admitted":
						reached = s.Ops[entry].Admission == 1 && s.Ops[entry].RequestStage == 1
					case "helper-commit":
						reached = s.Ops[helper].Ticket == committed
					case "reply-commit":
						reached = s.Ops[entry].Reply == 1
					case "published":
						reached = s.Ops[entry].Published
					}
					if reached && !s.Closed {
						closed := c
						closed.Closure = true
						for _, e := range Successors(closed, v, s) {
							if e.Event.Kind == "close_root" {
								return []Edge{e}
							}
						}
					}
					return base(c, s)
				}
				r := Probe(c, graph, false)
				if r.Stuck != 0 || r.RankFailures != 0 {
					t.Fatalf("%s %+v", label, r)
				}
				for _, o := range r.Outcomes {
					if !o.Closed || o.RootStatus != "cancelled" {
						t.Fatalf("closure missed %s %+v", label, o)
					}
				}
				records = append(records, record{label, true, r})
			}
		}
	}
	emit(t, "closure-reference-boundaries", records)
}
