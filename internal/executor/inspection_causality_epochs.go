package executor

import "math/big"

// A successful Open increments every currently open generation once. t(e) is
// the number of successful reopens before event e. An open generation at
// epoch k therefore fixes t(e) - t(birth) = k-1, even if it has since closed.
type reopenEquation struct {
	from, to int
	delta    int64
}

func (g *memberOrder) checkActivationCounts() error {
	// Both readers already authenticated each witness against its own run.
	// With no reactivations, every epoch equation is satisfied at time zero.
	reactivated := false
	for _, r := range g.runs {
		if r.epoch > 1 {
			reactivated = true
			break
		}
	}
	if !reactivated {
		return nil
	}
	// Index zero is the common provision event. The extra last index is the
	// inspected cut. Every retained transaction falls between them.
	cut := len(g.edges)
	before := make([][]int, cut+1)
	copy(before, g.edges)
	before[0] = append(before[0], cut)
	for i := 1; i < cut; i++ {
		before[0] = append(before[0], i)
		before[i] = append(before[i], cut)
	}
	equations := g.dispatchOwner
	valid := true
	witness := func(n int, r creationRun, epoch int64, open bool) {
		birth := g.creation[r.ID]
		if n == 0 || (birth == 0 && r.origin != "root") || epoch < 1 || epoch > r.epoch {
			valid = false
			return
		}
		if open {
			equations = append(equations, reopenEquation{birth, n, epoch - 1})
			return
		}
		// A closed generation keeps its final epoch through later reopens.
		// The record proves only that closure already happened.
		closure := g.closure[r.ID]
		if closure == 0 || r.epoch != epoch {
			valid = false
			return
		}
		before[closure] = append(before[closure], n)
	}
	for _, r := range g.runs {
		end := g.closure[r.ID]
		if r.open {
			end = cut
		}
		witness(end, r, r.epoch, true)
	}
	for _, s := range g.history {
		witness(g.reductions[s.ReductionRef], g.runs[s.Run], s.Epoch, true)
	}
	for id, a := range g.authorizations {
		n := g.records[protocolRecord{compositionAuthorizationFact, id}]
		ref := ReductionRef{a.Source.Run, a.Source.Sequence}
		open := a.ClosureOperation == "" && g.openAfter[ref]
		witness(n, g.runs[a.Source.Run], a.Source.Epoch, open)
	}
	for _, d := range g.decisions {
		if w := d.Before; w != nil {
			n := g.records[protocolRecord{compositionLifecycleFact, d.Request.Intent.ID}]
			witness(n, g.runs[w.Run], w.Epoch, w.Projection.Status == "open")
		}
	}
	for n, a := range g.readWitnesses {
		witness(n, g.runs[a.Run], a.Epoch, true)
	}
	if !valid || !feasibleReopens(before, equations) {
		return corrupt("retained authority contradicts shared activation epochs")
	}
	return nil
}

// feasibleReopens checks monotone integer times and exact differences. Their
// common origin is arbitrary, so a satisfying assignment may be translated
// to make provision zero. No step depends on the numeric size of an epoch.
func feasibleReopens(before [][]int, equations []reopenEquation) bool {
	type arc struct {
		to, equation int
		reverse      bool
	}
	equal := make([][]arc, len(before))
	for i, e := range equations {
		equal[e.from] = append(equal[e.from], arc{e.to, i, false})
		equal[e.to] = append(equal[e.to], arc{e.from, i, true})
	}

	// Within an equality component, every event's time is the component's
	// unknown base plus its uniquely determined offset. Traverse once in
	// both directions and reject conflicting offsets on any return path.
	component := make([]int, len(before))
	offset := make([]big.Int, len(before))
	var queue []int
	var delta, candidate big.Int
	components := 0
	for first := range before {
		if component[first] != 0 {
			continue
		}
		components++
		component[first] = components
		queue = append(queue[:0], first)
		for i := 0; i < len(queue); i++ {
			u := queue[i]
			for _, a := range equal[u] {
				delta.SetInt64(equations[a.equation].delta)
				if a.reverse {
					delta.Neg(&delta)
				}
				candidate.Add(&offset[u], &delta)
				if component[a.to] == 0 {
					component[a.to] = components
					offset[a.to].Set(&candidate)
					queue = append(queue, a.to)
				} else if candidate.Cmp(&offset[a.to]) != 0 {
					return false
				}
			}
		}
	}

	// t(v) >= t(u) becomes base(v) >= base(u) + offset(u) - offset(v).
	// Keep only the strongest bound for each ordered pair of components.
	type bound struct {
		to  int
		gap big.Int
	}
	bounds := make([][]bound, components+1)
	type pair struct{ from, to int }
	position := make(map[pair]int)
	for u, next := range before {
		for _, v := range next {
			key := pair{component[u], component[v]}
			if key.from == key.to {
				if offset[u].Cmp(&offset[v]) > 0 {
					return false
				}
				continue
			}
			candidate.Sub(&offset[u], &offset[v])
			i, present := position[key]
			if !present {
				i = len(bounds[key.from])
				position[key] = i
				bounds[key.from] = append(bounds[key.from], bound{to: key.to})
			}
			existing := &bounds[key.from][i].gap
			if !present || candidate.Cmp(existing) > 0 {
				existing.Set(&candidate)
			}
		}
	}

	// Frontier Bellman-Ford finds the least bases from a common zero lower
	// bound. A finite solution never needs a longest path with a repeated
	// component. An increase on round C therefore proves a positive cycle.
	base := make([]big.Int, components+1)
	scheduled := make([]int, components+1)
	queue = queue[:0]
	for i := 1; i <= components; i++ {
		queue = append(queue, i)
	}
	var changed []int
	for round := 1; round <= components && len(queue) > 0; round++ {
		changed = changed[:0]
		for _, from := range queue {
			for i := range bounds[from] {
				b := &bounds[from][i]
				candidate.Add(&base[from], &b.gap)
				if candidate.Cmp(&base[b.to]) > 0 {
					base[b.to].Set(&candidate)
					if scheduled[b.to] != round {
						scheduled[b.to] = round
						changed = append(changed, b.to)
					}
				}
			}
		}
		queue, changed = changed, queue
	}
	return len(queue) == 0
}
