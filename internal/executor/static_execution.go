package executor

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sort"
)

// staticExecution is disposable discovery state, never authority. All access is
// under Store.mu. Only the closed, independently validated static shape can arm
// it. A failed premise permanently drops it for this owner; Open validates anew.
// Sorted frontier insertion/removal is O(F), not a general logarithmic bound.
type staticExecution struct {
	owner               *Store
	conn                *sql.Conn
	stamp               executionStamp
	limits              Limits
	namespace, profile  string
	profileOwner        *compositionProfile
	dispatch            bool
	ids                 []RunID
	rank                map[RunID]int
	candidates          []int
	observations        map[RunID][]string
	changedObservations map[string]bool
	pending             map[string]bool
	rootRank            map[string]int
	rootOf              map[RunID]string
	dirtyRoots          map[string]bool
}
type executionStamp struct{ changes, mainSchema, tempSchema, dataVersion int64 }
type executionDelta struct {
	run          RunID
	observations []string
	intents      []Intent
	settled      string
	root         string
	unsupported  bool
}

func readExecutionStamp(ctx context.Context, q queryer) (v executionStamp, err error) {
	err = q.QueryRowContext(ctx, "SELECT total_changes()").Scan(&v.changes)
	if err == nil {
		err = q.QueryRowContext(ctx, "PRAGMA main.schema_version").Scan(&v.mainSchema)
	}
	if err == nil {
		err = q.QueryRowContext(ctx, "PRAGMA temp.schema_version").Scan(&v.tempSchema)
	}
	if err == nil {
		err = q.QueryRowContext(ctx, "PRAGMA main.data_version").Scan(&v.dataVersion)
	}
	return
}
func (s *Store) disableExecution() {
	if s.execution != nil {
		s.execution = nil
		s.executionFallbacks++
	}
}
func (s *Store) guardExecution(ctx context.Context) bool {
	x := s.execution
	if x == nil {
		return false
	}
	if s.closed || x.owner != s || x.conn != s.conn || s.limits != x.limits || s.namespace != x.namespace || s.composition != x.profileOwner || s.composition.raw != x.profile || !s.invocationEnabled() || s.reconciliation || s.dispatchAdmissions != x.dispatch {
		s.disableExecution()
		return false
	}
	stamp, err := readExecutionStamp(ctx, s.conn)
	if err != nil || stamp != x.stamp {
		s.disableExecution()
		return false
	}
	return true
}

// The pre-BEGIN guard cannot be replaced by a post-COMMIT watermark: a direct
// write followed by a legitimate operation must not be silently acknowledged.
func (s *Store) beginOwnedTx(ctx context.Context) (*sql.Tx, error) {
	s.guardExecution(ctx)
	return beginOwnedTx(ctx, s.conn)
}

// commitExecutionTx does not acquire ownerMu: dispatch admission already holds
// it. Notification stays with commitOwnedTx. Bookkeeping cannot undo a commit.
func (s *Store) commitExecutionTx(ctx context.Context, tx *sql.Tx, d *executionDelta) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		s.disableExecution()
		return err
	}
	x := s.execution
	if x == nil {
		return nil
	}
	stamp, err := readExecutionStamp(ctx, s.conn)
	if err != nil || stamp.mainSchema != x.stamp.mainSchema || stamp.tempSchema != x.stamp.tempSchema || stamp.dataVersion != x.stamp.dataVersion {
		s.disableExecution()
		return nil
	}
	if d == nil {
		if stamp.changes != x.stamp.changes {
			s.disableExecution()
		}
		return nil
	}
	if d.unsupported {
		s.disableExecution()
		return nil
	}
	if d.run.Subject != "" {
		rank, ok := x.rank[d.run]
		if !ok {
			s.disableExecution()
			return nil
		}
		at, found := slices.BinarySearch(x.candidates, rank)
		if !found {
			x.candidates = slices.Insert(x.candidates, at, rank)
		}
		x.dirtyRoot(x.rootOf[d.run])
	}
	if d.root != "" {
		if _, ok := x.rootRank[d.root]; !ok {
			s.disableExecution()
			return nil
		}
		x.dirtyRoot(d.root)
	}
	for _, id := range d.observations {
		if id != "" {
			x.changedObservations[id] = true
		}
	}
	for _, i := range d.intents {
		if i.Kind != "adapter" || i.Authority != nil || i.Target == receiptReadCapability || i.Run != d.run {
			s.disableExecution()
			return nil
		}
		x.pending[i.ID] = true
	}
	if d.settled != "" {
		delete(x.pending, d.settled)
	}
	x.stamp = stamp
	return nil
}
func (x *staticExecution) dirtyRoot(id string) { x.dirtyRoots[id] = true }

// Initialization is opportunistic after activation. Errors here only select the
// original path; Create/Open have already established their public result.
func (s *Store) initializeExecution(ctx context.Context) {
	if s.composition == nil || !s.invocationEnabled() || s.reconciliation {
		return
	}
	c, err := s.composition.descriptor()
	if err != nil || c.RequestAuthority != "" || len(c.Templates) != 0 || len(c.Roots) == 0 {
		return
	}
	x := &staticExecution{owner: s, conn: s.conn, limits: s.limits, namespace: s.namespace, profile: s.composition.raw, profileOwner: s.composition, dispatch: s.dispatchAdmissions,
		rank: map[RunID]int{}, observations: map[RunID][]string{}, changedObservations: map[string]bool{}, pending: map[string]bool{}, rootRank: map[string]int{}, rootOf: map[RunID]string{}, dirtyRoots: map[string]bool{}}
	// Validation has checked the full profile; these exclusions define the smaller
	// dependency proof. Runtime unknown writers disable it regardless of shape.
	for _, table := range []string{"composition_templates", "composition_members", "composition_invocations", "composition_bindings", "composition_closures", "lifecycle_results", "external_attempts", "external_receipts"} {
		var present bool
		if err = s.conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+table+")").Scan(&present); err != nil || present {
			return
		}
	}
	var conflicts bool
	if err = s.conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name='receipt_conflicts')").Scan(&conflicts); err != nil {
		return
	}
	if conflicts {
		if err = s.conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM receipt_conflicts)").Scan(&conflicts); err != nil || conflicts {
			return
		}
	}
	var forbidden bool
	if err = s.conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='trigger') OR EXISTS(SELECT 1 FROM sqlite_temp_master WHERE type='trigger') OR EXISTS(SELECT 1 FROM permissions WHERE kind='observation' AND operation='terminal') OR EXISTS(SELECT 1 FROM subjects WHERE generation!=1 OR retired!=0 OR retry_cancelled!=0)`).Scan(&forbidden); err != nil || forbidden {
		return
	}
	ids, err := readRunIDs(ctx, s.conn)
	if err != nil || len(ids) != len(c.Roots) {
		return
	}
	x.ids = ids
	for rank, root := range c.Roots {
		if root.Run.Generation != 1 || len(root.Members) != 1 || root.Members[0] != root.Run || len(root.Templates) != 0 || x.rootOf[root.Run] != "" {
			return
		}
		x.rootRank[root.ID] = rank
		x.rootOf[root.Run] = root.ID
		x.dirtyRoot(root.ID)
	}
	for rank, id := range ids {
		if x.rootOf[id] == "" {
			return
		}
		run, e := loadRun(ctx, s.conn, id)
		if e != nil {
			return
		}
		p, e := loadProgram(ctx, s.conn, run.ProgramID, s.limits)
		if e != nil || hasHandoffPeerPort(p) {
			return
		}
		for _, ins := range p.Code {
			if ins.Effect != nil && (ins.Effect.Kind != "adapter" || ins.Effect.Authority != nil) {
				return
			}
		}
		x.rank[id] = rank
		x.candidates = append(x.candidates, rank)
	}
	// Retain identity membership, including validation of consumed observations.
	obs, err := readObservations(ctx, s.conn, s.limits)
	if err != nil {
		return
	}
	for id, o := range obs {
		if _, ok := x.rank[o.Run]; !ok {
			return
		}
		if !o.consumed.Valid {
			x.observations[o.Run] = append(x.observations[o.Run], id)
		}
	}
	for run := range x.observations {
		sort.Strings(x.observations[run])
	}
	intents, err := readIntents(ctx, s.conn)
	if err != nil {
		return
	}
	for id, i := range intents {
		if i.Kind != "adapter" || i.Authority != nil || i.Target == receiptReadCapability || x.rootOf[i.Run] == "" {
			return
		}
		var settled bool
		if err = s.conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM settlements WHERE id=?)", identity("adapter-outcome", id)).Scan(&settled); err != nil {
			return
		}
		if !settled {
			x.pending[id] = true
		}
	}
	// No reconciliation results or terminal routes are admitted by this shape.
	rows, err := s.conn.QueryContext(ctx, "SELECT descriptor FROM adapter_results")
	if err != nil {
		return
	}
	for rows.Next() {
		var raw []byte
		var r AdapterResult
		if rows.Scan(&raw) != nil || decode(raw, &r) != nil || r.Read != nil || r.Reconciled != nil || r.RefusalWitness != "" {
			rows.Close()
			return
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return
	}
	rows, err = s.conn.QueryContext(ctx, "SELECT destinations FROM terminal_obligations")
	if err != nil {
		return
	}
	for rows.Next() {
		var raw []byte
		var d []TerminalDestination
		if rows.Scan(&raw) != nil || decode(raw, &d) != nil || len(d) != 0 {
			rows.Close()
			return
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return
	}
	x.stamp, err = readExecutionStamp(ctx, s.conn)
	if err != nil {
		return
	}
	s.execution = x
}

// A primary-key query, never an unindexed subject predicate, uses the complete
// retained observation validator including consumed rows.
func (s *Store) executionObservation(ctx context.Context, id string) (retainedObservation, error) {
	rows, err := s.conn.QueryContext(ctx, "SELECT id,subject,generation,descriptor,consumed_sequence FROM observations WHERE id=?", id)
	if err != nil {
		return retainedObservation{}, err
	}
	observations, err := readObservationRows(rows, s.limits)
	if err != nil {
		return retainedObservation{}, err
	}
	o, ok := observations[id]
	if !ok {
		return o, sql.ErrNoRows
	}
	return o, nil
}
func (s *Store) selectExecutionReady(ctx context.Context, last *RunID) (*readyRun, error) {
	x := s.execution
	// Changes are validated before any lazy run traversal, matching readObservations.
	changed := make([]string, 0, len(x.changedObservations))
	for id := range x.changedObservations {
		changed = append(changed, id)
	}
	sort.Strings(changed)
	for _, id := range changed {
		o, err := s.executionObservation(ctx, id)
		if err != nil {
			return nil, err
		}
		if _, ok := x.rank[o.Run]; !ok {
			s.disableExecution()
			return s.selectReadyBaselineLocked(ctx, last)
		}
		list := x.observations[o.Run]
		at, found := slices.BinarySearch(list, id)
		if o.consumed.Valid && found {
			list = slices.Delete(list, at, at+1)
		} else if !o.consumed.Valid && !found {
			list = slices.Insert(list, at, id)
		}
		x.observations[o.Run] = list
		delete(x.changedObservations, id)
	}
	start := 0
	if last != nil {
		if rank, ok := x.rank[*last]; ok {
			start = rank + 1
		}
	}
	at, _ := slices.BinarySearch(x.candidates, start)
	ordered := append(slices.Clone(x.candidates[at:]), x.candidates[:at]...)
	// Compact negative certificates once. Repeated slices.Delete would shift
	// the initial frontier quadratically when all B runs are blocked.
	blocked := map[int]bool{}
	defer func() {
		if len(blocked) == 0 {
			return
		}
		kept := x.candidates[:0]
		for _, rank := range x.candidates {
			if !blocked[rank] {
				kept = append(kept, rank)
			}
		}
		x.candidates = kept
	}()
	programs := map[string]Program{}
	for _, rank := range ordered {
		id := x.ids[rank]
		run, err := loadRun(ctx, s.conn, id)
		if err != nil {
			return nil, err
		}
		var generation int64
		var retired bool
		if err = s.conn.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", id.Subject).Scan(&generation, &retired); err != nil {
			return nil, err
		}
		if run.Projection.Status == "open" && !retired && generation == id.Generation {
			p, ok := programs[run.ProgramID]
			if !ok {
				p, err = loadProgram(ctx, s.conn, run.ProgramID, s.limits)
				if err != nil {
					return nil, err
				}
				programs[run.ProgramID] = p
			}
			p, err = boundCompositionProgram(ctx, s.conn, p, id, s.limits)
			if err != nil {
				return nil, err
			}
			if p.Code[run.Projection.PC].Op != "receive" {
				if _, err = Reduce(p, run.Projection, nil, id, run.Sequence+1, s.limits); err != nil {
					return nil, err
				}
				return &readyRun{authority: run.Authority()}, nil
			}
			for _, oid := range x.observations[id] {
				o, err := s.executionObservation(ctx, oid)
				if err != nil {
					return nil, err
				}
				_, err = Reduce(p, run.Projection, &o.Observation, id, run.Sequence+1, s.limits)
				if errors.Is(err, ErrIneligible) {
					continue
				}
				if err != nil {
					return nil, err
				}
				return &readyRun{authority: run.Authority(), observation: oid}, nil
			}
		}
		// This negative certificate depends only on this run and its observations in
		// the guarded static shape. No Reduce call is moved ahead of original order.
		blocked[rank] = true
	}
	return nil, nil
}
func (s *Store) pendingExecutionDeliveries() []pendingDelivery {
	out := make([]pendingDelivery, 0, len(s.execution.pending))
	for id := range s.execution.pending {
		out = append(out, pendingDelivery{key: identity("adapter-delivery", id), intent: id, kind: "adapter"})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

func (s *Store) settleExecutionRoots(ctx context.Context) (bool, error) {
	s.mu.Lock()
	if !s.guardExecution(ctx) {
		s.mu.Unlock()
		return s.settleRootsBaseline(ctx)
	}
	x := s.execution
	roots := make([]string, 0, len(x.dirtyRoots))
	for id := range x.dirtyRoots {
		roots = append(roots, id)
	}
	sort.Slice(roots, func(i, j int) bool { return x.rootRank[roots[i]] < x.rootRank[roots[j]] })
	s.mu.Unlock()
	progressed := false
	for _, id := range roots {
		s.mu.Lock()
		if !s.guardExecution(ctx) {
			s.mu.Unlock()
			more, err := s.settleRootsBaseline(ctx)
			return progressed || more, err
		}
		before, err := prepareRoot(ctx, s.conn, id)
		if err == nil {
			delete(x.dirtyRoots, id)
		}
		s.mu.Unlock()
		if err != nil {
			return false, err
		}
		if before.Ready && before.Outcome.ID == "" {
			if _, err = s.SettleRoot(ctx, id); err != nil {
				// A pre-write failure has no committed delta or counter change
				// to restore this obligation. Requeue without replacing newer
				// dirty work or reviving an index disabled by another failure.
				s.mu.Lock()
				if s.execution == x {
					x.dirtyRoot(id)
				}
				s.mu.Unlock()
				return false, err
			}
			progressed = true
		}
		// A commit after preparation (including our own publication) dirties again.
		// Deletion under mu preceded the unlock, so newer invalidations survive.
	}
	return progressed, nil
}
