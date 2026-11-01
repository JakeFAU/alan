package reconciliation

// Each edge is recorded before a previously reached target is deduplicated.
type Edge struct {
	Action string
	To     int
}
type Row struct {
	State State
	Edges []Edge
	Class string
}
type Graph struct {
	Profile Profile
	Rows    []Row
}

func Class(s State) string {
	switch {
	case s.Rejected:
		return "final-duplicate-guard-failed"
	case s.Dead:
		return "owner-lost"
	case s.Used && s.Interaction == "transport_error":
		return "applied-after-failed-interaction"
	case s.Used:
		return "ordinary-continuation"
	}
	for i := len(s.Reads) - 1; i >= 0; i-- {
		q := s.Reads[i]
		if q.Stage == "" {
			continue
		}
		if q.Closed && q.Stage == "ready" {
			return "revoked-before-admission"
		}
		if q.Stage == "retained" {
			return "accounting-owed"
		}
		if q.Stage != "handled" {
			return "read-service"
		}
		if q.Adoption != "" {
			return q.Adoption
		}
		return q.Outcome
	}
	if s.Quiescent && s.Held {
		return "drained-with-live-packet"
	}
	if s.Quiescent {
		return "original-drained"
	}
	return "original-service"
}

func Explore(p Profile) Graph {
	start := Initial()
	g := Graph{Profile: p, Rows: []Row{{State: start}}}
	ids := map[State]int{start: 0}
	for i := 0; i < len(g.Rows); i++ {
		s := g.Rows[i].State
		edges := []Edge{}
		for _, step := range Outgoing(p, s) {
			id, ok := ids[step.Next]
			if !ok {
				id = len(g.Rows)
				ids[step.Next] = id
				g.Rows = append(g.Rows, Row{State: step.Next})
			}
			edges = append(edges, Edge{step.Action, id})
		}
		g.Rows[i].Edges = edges
		g.Rows[i].Class = Class(s)
	}
	return g
}
