package executor

// Admissions have their own insertion order. A prior binding proves a target
// existed and cannot later revert to an earlier generation. Close and retire
// are permanent for the generation and subject respectively.
type transportBounds struct {
	generation map[string]int64
	retired    map[string]bool
	closed     map[RunID]bool
}

func (c *causalCut) validTransport(d AdmissionDescriptor, r AdmissionResult, seen *transportBounds, routes map[Permission]bool, terminal map[RunID]terminalSource) bool {
	var w transportWitness
	w.source = c.auth[d.IntentID]
	if bound := r.Bound; bound != nil {
		s := c.runs[*bound].Subject
		w.retired = s.Retired && s.Generation == bound.Generation
		t, closed := terminal[*bound]
		w.closed = closed && c.lifecycle[t.closure].Request.Operation != "RetireSubject"
		revision := int64(0)
		if d.Kind == "message" {
			revision = c.intents[d.IntentID].revision
		}
		w.permitted = d.Kind == "terminal" || routes[Permission{Kind: d.Kind, Source: d.Source.Subject, Target: d.Target, Operation: d.Kind, Revision: revision, Obligation: d.Obligation}]
		w.schemaValid = authorAdmissionValid(c.resolved[*bound], d)
	}
	return seen.valid(d, r, w)
}

// Immutable route and program evidence determines the admission after source,
// generation, retirement and closure precedence. The latter two may also have
// happened later, so a refusal requires a compatible retained transition.
type transportWitness struct {
	source                                  Authorization
	retired, closed, permitted, schemaValid bool
}

func (seen *transportBounds) valid(d AdmissionDescriptor, r AdmissionResult, w transportWitness) bool {
	previous := seen.generation[d.Target]
	bound := r.Bound
	if bound == nil {
		if previous != 0 || d.Kind != "input" && d.Source.Subject == d.Target {
			return false
		}
	} else {
		if previous > bound.Generation {
			return false
		}
		seen.generation[d.Target] = bound.Generation
	}
	if d.Kind == "message" {
		a := w.source
		if !a.Allowed {
			return !r.Accepted && r.Reason == "source_denied:"+a.Reason
		}
	}
	if bound == nil {
		// A dynamic member may not yet have existed at this unordered input.
		return !r.Accepted && r.Reason == "target_missing"
	}
	if !r.Accepted && r.Reason == "target_retired" {
		if !w.retired {
			return false
		}
		seen.retired[d.Target] = true
		return true
	}
	if seen.retired[d.Target] {
		return false
	}
	if d.Generation != 0 && d.Generation != bound.Generation {
		return !r.Accepted && r.Reason == "target_generation_changed"
	}
	if !r.Accepted && r.Reason == "target_closed" {
		if !w.closed {
			return false
		}
		seen.closed[*bound] = true
		return true
	}
	if seen.closed[*bound] {
		return false
	}
	// No ordinary transport timestamp orders it against a later closure.
	// Immutable route and program witnesses still determine these decisions.
	if !w.permitted {
		return !r.Accepted && r.Reason == "route_denied"
	}
	if !w.schemaValid {
		return !r.Accepted && r.Reason == "schema_invalid"
	}
	return r.Accepted && r.Reason == "accepted"
}

// The ordinary history has a durable append order even in the oldest
// supported storage version. Verify its provable predecessors and, for
// scoped reductions, agreement with the independently retained decision order.
func (c *causalCut) checkHistoryOrder() error {
	seen := map[RunID]int64{}
	earlierOperation := func(id string) bool {
		if id == "" {
			return true
		}
		i, ok := c.intents[id]
		if !ok || seen[i.Run] < i.Sequence || seen[i.Run] < c.auth[id].Source.Sequence {
			return false
		}
		if w := c.lifecycle[id].Before; w != nil && seen[w.Run] < w.Sequence {
			return false
		}
		return true
	}
	var lastScoped int64
	for _, step := range c.snap.history {
		run := c.runs[step.Run]
		if run.RootID != "" {
			order := c.executionOrder(step.Run, step.Sequence)
			if order <= lastScoped {
				return corrupt("history append order contradicts scoped reduction facts")
			}
			lastScoped = order
		}
		if step.Sequence == 1 && !earlierOperation(run.OriginOperationID) {
			return corrupt("run reduced before original creation")
		}
		if o := c.observations[step.SelectedObservationID]; o != nil {
			if r := o.ProducerReduction; r != nil && seen[r.Run] < r.Sequence {
				return corrupt("observation selected before original producing reduction")
			}
			if !earlierOperation(o.SourceOperationID) {
				return corrupt("observation selected before its original operation or authority")
			}
		}
		seen[step.Run] = step.Sequence
	}
	return nil
}
