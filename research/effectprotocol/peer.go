package effectprotocol

// PairState composes two independent adapter operations against one counter.
// Subject 0 adds one and deliberately returns unknown. Subject 1 adds two
// and returns its authentic receipt. Both use the original operation kernel.
// Ordinal and Total preserve both possible orders of endpoint commits.
type PairState struct {
	Actors         [2]State
	Ordinal, Total [2]int
}

type PairEdge struct {
	Actor  int
	Action Action
	ID     int
}
type PairRow struct {
	ID       int
	State    PairState
	Outgoing []PairEdge
}

func peerProfile(i int) Profile {
	if i == 0 {
		return Profile{Route: "adapter", Idempotent: true, Reply: "unknown"}
	}
	return Profile{Route: "adapter", Idempotent: true}
}

// The runner returns to dispatch selection after accounting a received
// completion. A real worker can be done while its result awaits accounting.
func occupies(s State) bool {
	switch s.Volatile.Work {
	case "", "unanswered":
		return false
	case "accounted":
		return !s.Local.Settled
	default:
		return true
	}
}

func PeerGraph() []PairRow {
	start := PairState{Actors: [2]State{Initial(), Initial()}}
	ids := map[PairState]int{start: 0}
	states := []PairState{start}
	var rows []PairRow
	for n := 0; n < len(states); n++ {
		s := states[n]
		row := PairRow{n, s, []PairEdge{}}
		for actor, a := range s.Actors {
			for _, e := range Outgoing(peerProfile(actor), a) {
				if e.Action.Name == "store.reserve" && occupies(s.Actors[1-actor]) {
					continue
				}
				to := s
				to.Actors[actor] = e.To
				if e.Action.Name == "endpoint.apply" {
					to.Ordinal[actor] = s.Actors[1-actor].World.Applications + 1
					to.Total[actor] = s.Actors[1-actor].World.Applications*(2-actor) + actor + 1
				}
				id, ok := ids[to]
				if !ok {
					id = len(states)
					ids[to] = id
					states = append(states, to)
				}
				row.Outgoing = append(row.Outgoing, PairEdge{actor, e.Action, id})
			}
		}
		rows = append(rows, row)
	}
	return rows
}
