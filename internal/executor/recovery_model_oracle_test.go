package executor

// This checker works backwards from the retained fields. Its four variables
// are the number of successful reopens before creation, source authorization,
// target lookup and source closure. It never generates an execution schedule.
func recoveryCutFeasible(c recoveryModelCut) bool {
	if c.RootEpoch < 1 || c.RootEpoch > 3 || c.Creations < 0 || c.Creations > 1 {
		return false
	}
	reopens := c.RootEpoch - 1
	type interval struct{ low, high int }
	times := map[byte]interval{}
	var edges [][2]byte
	before := func(first, next byte) { edges = append(edges, [2]byte{first, next}) }
	fixed := func(event byte, when int) { times[event] = interval{when, when} }

	if c.Creations == 0 {
		if c.MemberEpoch != 0 {
			return false
		}
	} else {
		if c.MemberEpoch < 1 || c.MemberEpoch > c.RootEpoch {
			return false
		}
		// A member starts at epoch one and remains open to the cut.
		fixed('C', c.RootEpoch-c.MemberEpoch)
	}
	if c.CloseEpoch == 0 {
		if c.SourceEpoch != c.RootEpoch {
			return false
		}
	} else {
		if c.CloseEpoch < 1 || c.CloseEpoch > c.RootEpoch || c.SourceEpoch != c.CloseEpoch {
			return false
		}
		fixed('X', c.CloseEpoch-1)
	}

	switch c.AuthSequence {
	case 0:
		if c.AuthEpoch != 0 || c.AuthGeneration != 0 || c.AuthRevision != 0 || c.AuthReason != "" {
			return false
		}
	case 2, 3:
		if c.AuthEpoch < 1 || c.AuthEpoch > c.SourceEpoch || c.AuthGeneration != 1 || c.AuthRevision != 1 {
			return false
		}
		if c.AuthSequence == 2 {
			if c.AuthReason != "authorized" {
				return false
			}
			fixed('A', c.AuthEpoch-1)
			if c.CloseEpoch != 0 {
				before('A', 'X')
			}
		} else {
			if c.CloseEpoch == 0 || c.AuthEpoch != c.CloseEpoch || c.AuthReason != "source_closed" {
				return false
			}
			// A later owner can observe the frozen epoch at any point after closure.
			times['A'] = interval{c.CloseEpoch - 1, reopens}
			before('X', 'A')
		}
	default:
		return false
	}

	if c.Reason == "grant_denied" {
		birth, exists := times['C']
		if !exists || c.TargetEpoch < 1 || c.TargetEpoch > c.MemberEpoch || c.TargetGeneration != 1 || c.TargetRevision != 1 {
			return false
		}
		fixed('D', birth.low+c.TargetEpoch-1)
		before('C', 'D')
	} else {
		if c.TargetEpoch != 0 || c.TargetGeneration != 0 || c.TargetRevision != 0 {
			return false
		}
		switch c.Reason {
		case "target_missing":
			times['D'] = interval{0, reopens}
			if c.Creations == 1 {
				before('D', 'C')
			}
		case "":
		default:
			return false
		}
	}
	if _, decided := times['D']; decided {
		if c.AuthSequence != 2 {
			return false
		}
		before('A', 'D')
	}

	// Membership contributes a fact only for a participant at the event.
	// Validate the multiplicities before using the recorded relative order.
	visible := map[byte]bool{
		'C': c.Creations == 1, 'A': c.Scoped && c.AuthSequence != 0,
		'D': c.Reason != "" && (c.Scoped || c.Reason == "grant_denied"), 'X': c.Scoped && c.CloseEpoch != 0,
	}
	for i := 0; i < len(c.Facts); i++ {
		e := c.Facts[i]
		if !visible[e] {
			return false
		}
		delete(visible, e)
		if i > 0 {
			before(c.Facts[i-1], e)
		}
	}
	for _, missing := range visible {
		if missing {
			return false
		}
	}

	// Intersect the clock intervals along the partial order. Equal clocks
	// still require distinct ordered events, so a precedence cycle is invalid.
	incoming := map[byte]int{}
	for _, e := range edges {
		incoming[e[1]]++
	}
	var queue []byte
	for _, id := range []byte{'A', 'C', 'D', 'X'} {
		if _, ok := times[id]; ok && incoming[id] == 0 {
			queue = append(queue, id)
		}
	}
	for i := 0; i < len(queue); i++ {
		current := queue[i]
		when := times[current]
		if when.low > when.high {
			return false
		}
		for _, e := range edges {
			if e[0] != current {
				continue
			}
			v := times[e[1]]
			v.low = max(v.low, when.low)
			times[e[1]] = v
			incoming[e[1]]--
			if incoming[e[1]] == 0 {
				queue = append(queue, e[1])
			}
		}
	}
	return len(queue) == len(times)
}
