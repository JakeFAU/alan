package executor

import (
	"reflect"
)

// Reactivation advances an open run's epoch without a reduction. A historical
// witness may therefore be anywhere between the adjacent recorded epochs.
func (c *causalCut) epochAt(id RunID, sequence, epoch int64) bool {
	run := c.runs[id]
	if run == nil || sequence < 0 || sequence > run.Sequence || epoch < 1 || epoch > run.Epoch {
		return false
	}
	if sequence > 0 && epoch < c.steps[ReductionRef{id, sequence}].Epoch {
		return false
	}
	return sequence == run.Sequence || epoch <= c.steps[ReductionRef{id, sequence + 1}].Epoch
}

// The ordinary history orders reductions across runs. At a lifecycle request's
// authorized source sequence, preceding scoped reductions are already durable.
func (c *causalCut) lifecycleSourcePrefixes(ordered []LifecycleResult) map[ReductionRef]int64 {
	wanted := make(map[ReductionRef]int64, len(ordered))
	for _, r := range ordered {
		a := c.auth[r.Request.Intent.ID]
		wanted[ReductionRef{a.Source.Run, a.Source.Sequence}] = 0
	}
	if len(wanted) == 0 {
		return wanted
	}
	var prefix int64
	for _, step := range c.snap.history {
		if c.membership[step.Run] != "" {
			prefix = max(prefix, c.executionOrder(step.Run, step.Sequence))
		}
		if _, ok := wanted[step.ReductionRef]; ok {
			wanted[step.ReductionRef] = prefix
		}
	}
	return wanted
}

func (c *causalCut) checkLifecycleAuthority(ordered []LifecycleResult) error {
	origins := map[string]rootOrigin{}
	states := map[string]Subject{}
	for _, row := range c.tables["run_origins"] {
		if row["kind"] == "root" {
			o, err := inspectionDecode[rootOrigin](row, "descriptor")
			if err != nil {
				return err
			}
			if o.Subject.Retired {
				return corrupt("invalid retired root origin")
			}
			origins[o.Subject.ID], states[o.Subject.ID] = o, o.Subject
		}
	}
	for _, run := range c.snap.Runs {
		if run.Origin == "invocation" {
			// The original member binding fixes these fields at creation.
			s := Subject{ID: run.ID.Subject, Generation: 1, Revision: 1, Owner: "invocation:" + run.OwningInvocationID}
			origins[s.ID] = rootOrigin{Subject: s, ProgramID: run.ProgramID, Initial: run.Initial}
			states[s.ID] = s
		}
	}
	for _, row := range c.tables["run_origins"] {
		if row["kind"] == "retry" && row["descriptor"] != compositionJSON(origins[inspectionRun(row).Subject]) {
			return corrupt("retry changed original subject, artifact or initial projection")
		}
	}
	grants := map[Permission]bool{}
	for _, p := range c.tables["permissions"] {
		if p["kind"] == "lifecycle" {
			grants[Permission{Source: inspectText(p, "source"), Target: inspectText(p, "target"), Operation: inspectText(p, "operation"), Revision: inspectInt(p, "revision")}] = true
		}
	}
	closed := map[RunID]retainedClosure{}
	last := map[RunID]Authority{}
	sourcePrefixes := c.lifecycleSourcePrefixes(ordered)
	var lowerBound int64
	for _, r := range ordered {
		req := r.Request
		w := r.Before
		s, exists := states[req.Intent.Target]
		auth := c.auth[req.Intent.ID]
		lowerBound = max(lowerBound, sourcePrefixes[ReductionRef{auth.Source.Run, auth.Source.Sequence}])
		srcOrder := c.factOrder(compositionLifecycleFact, req.Intent.ID)
		if srcOrder != 0 {
			if srcOrder <= lowerBound {
				return corrupt("lifecycle decision order contradicts ordered facts")
			}
			lowerBound = srcOrder
		}
		origin := origins[req.Intent.Target]
		if exists {
			first := c.runs[RunID{s.ID, origin.Subject.Generation}]
			if first.Origin == "invocation" {
				// Decisions with no scoped fact need the complete cross-journal
				// creation check after admissions and effects have been indexed.
				admission := c.factOrder(compositionAdmissionFact, first.OriginOperationID)
				if srcOrder != 0 {
					exists = admission < srcOrder
				} else if w == nil {
					exists = false
				}
			}
		}
		if (w != nil) != exists {
			return corrupt("lifecycle decision target existence")
		}
		if w != nil {
			if w.Run != (RunID{s.ID, s.Generation}) || w.Owner != s.Owner || w.Revision != s.Revision || w.Retired != s.Retired || w.RetryCancelled != s.RetryCancelled ||
				w.ProgramID != origin.ProgramID || !c.epochAt(w.Run, w.Sequence, w.Epoch) || w.Sequence < last[w.Run].Sequence || w.Epoch < last[w.Run].Epoch {
				return corrupt("lifecycle target authority witness")
			}
			proj := origin.Initial
			if w.Sequence > 0 {
				proj = c.after[ReductionRef{w.Run, w.Sequence}]
			}
			if closure, ok := closed[w.Run]; ok {
				if closure.sequence != w.Sequence {
					return corrupt("lifecycle witness advanced after closure")
				}
				proj = closure.projection
			}
			if !reflect.DeepEqual(proj, w.Projection) || proj.Status != "open" && w.Epoch != c.runs[w.Run].Epoch {
				return corrupt("lifecycle target projection witness")
			}
			if srcOrder != 0 && c.membership[w.Run] != "" {
				prev, next := c.executionOrder(w.Run, w.Sequence), c.executionOrder(w.Run, w.Sequence+1)
				if prev >= srcOrder || (next != 0 && srcOrder >= next) {
					return corrupt("lifecycle decision outside target reduction prefix")
				}
			}
			last[w.Run] = Authority{Run: w.Run, Sequence: w.Sequence, Epoch: w.Epoch, Revision: w.Revision}
		}
		granted := grants[Permission{Source: req.Intent.Run.Subject, Target: req.Intent.Target, Operation: req.Operation, Revision: req.ExpectedRevision}]
		// The runtime's pure policy checks the supplied witnesses. No Store
		// operation, reducer, or new decision is invoked during inspection.
		var want LifecycleResult
		var err error
		if authorityOperation(req.Operation) && c.authorityValidated {
			want = r
		} else {
			want, err = decideTarget(req, r.DecisionSequence, w, origin, granted)
		}
		if err != nil {
			return corrupt("invalid retained lifecycle decision: %v", err)
		}
		if w != nil && req.Operation == "Retry" && c.membership[w.Run] != "" {
			after := *w
			want.Accepted, want.Successor, want.Reason, want.After = false, 0, "invocation_retry_unsupported", &after
		}
		if !reflect.DeepEqual(want, r) {
			return corrupt("retained lifecycle decision disagrees with its authority")
		}
		if r.Accepted {
			s.Generation, s.Retired = r.After.Run.Generation, r.After.Retired
			states[s.ID] = s
			if r.Successor != 0 {
				next := c.runs[r.After.Run]
				if next == nil || next.Origin != "retry" || next.OriginOperationID != req.Intent.ID {
					return corrupt("lifecycle retry missing original successor")
				}
			} else if w.Projection.Status == "open" && r.After.Projection.Status != "open" {
				closed[w.Run] = retainedClosure{r.After.Projection, w.Sequence, req.Intent.ID, w.Revision}
			}
		}
	}
	for _, run := range c.snap.Runs {
		if run.Subject != states[run.ID.Subject] {
			return corrupt("current subject authority differs from original lifecycle")
		}
	}
	for _, row := range c.tables["terminal_obligations"] {
		id := inspectionRun(row)
		closure, ok := closed[id]
		if op := inspectText(row, "operation_id"); ok || op != "" {
			if !ok || op != closure.operation || row["sequence"] != closure.sequence || row["source_revision"] != closure.revision || c.runs[id] == nil || !reflect.DeepEqual(c.runs[id].Projection, closure.projection) {
				return corrupt("terminal obligation differs from original lifecycle closure")
			}
			delete(closed, id)
		}
	}
	if len(closed) != 0 {
		return corrupt("original lifecycle closure missing its terminal obligation")
	}
	return nil
}
