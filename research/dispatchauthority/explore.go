package dispatchauthority

type Target struct {
	Action string
	To     int
}
type Row struct {
	ID       int
	State    State
	Wait     string
	Outgoing []Target
}
type Graph struct {
	Profile Profile
	Rows    []Row
}

// Explore enumerates each labelled alternative before deduplicating successors.
func Explore(p Profile) Graph {
	initial := Initial(p)
	states := []State{initial}
	ids := map[State]int{initial: 0}
	g := Graph{Profile: p}
	for id := 0; id < len(states); id++ {
		s := states[id]
		row := Row{ID: id, State: s, Wait: Wait(p, s), Outgoing: []Target{}}
		for _, step := range Outgoing(p, s) {
			to, exists := ids[step.State]
			if !exists {
				to = len(states)
				ids[step.State] = to
				states = append(states, step.State)
			}
			row.Outgoing = append(row.Outgoing, Target{Action: step.Action, To: to})
		}
		g.Rows = append(g.Rows, row)
	}
	return g
}
