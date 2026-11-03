//go:build integration

package executor

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

func rejectDispatchHistory(t *testing.T, s *Store, path string) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, limits := t.Context(), DefaultLimits()
	before := inspectionFingerprint(t, path)
	cut, inspected := InspectExecution(ctx, path, limits)
	_, replayed := Replay(ctx, path, limits)
	owner, opened := Open(ctx, path, limits)
	if owner != nil {
		owner.Close()
	}
	if cut != nil || owner != nil || !errors.Is(inspected, ErrCorrupt) || !errors.Is(replayed, ErrCorrupt) || !errors.Is(opened, ErrCorrupt) {
		t.Fatalf("impossible dispatch history: inspection %v, replay %v, open %v", inspected, replayed, opened)
	}
	if before != inspectionFingerprint(t, path) {
		t.Fatal("rejected history changed durable bytes")
	}
}

func TestDispatchAdmissionRejectsDuplicateRow(t *testing.T) {
	ctx := t.Context()
	s, path, _ := gateStore(t, true)
	if _, err := s.Run(ctx, RunOptions{MaxSteps: 30, MaxDispatch: 30, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}}); err != nil {
		t.Fatal(err)
	}
	// Damage the unique constraint and duplicate a genuine row. Each gate is
	// locally plausible. Readers must still agree that only one was possible.
	if _, err := s.conn.ExecContext(ctx, `ALTER TABLE adapter_dispatch_admissions RENAME TO held_gates;
		CREATE TABLE adapter_dispatch_admissions (
			intent_id TEXT NOT NULL REFERENCES adapter_attempts(intent_id),
			prepared_digest TEXT NOT NULL,
			after_fact INTEGER NOT NULL REFERENCES composition_facts(decision_order));
		INSERT INTO adapter_dispatch_admissions SELECT * FROM held_gates;
		INSERT INTO adapter_dispatch_admissions SELECT * FROM held_gates;
		DROP TABLE held_gates`); err != nil {
		t.Fatal(err)
	}
	rejectDispatchHistory(t, s, path)
}

// An independent peer witnesses the order of the worker's reservation and
// owner activations without producing or settling the worker's adapter result.
func dispatchPeerStore(t *testing.T) (*Store, string, *adapterWork) {
	t.Helper()
	ctx := t.Context()
	p := Provision{}
	addAdapterProgram(t, &p, "worker", true)
	addProgram(t, &p, "peer", []Instruction{
		{Op: "set", Dst: "n", A: Lit(Int(1))},
		{Op: "set", Dst: "n", A: Lit(Int(2))},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]Value{})
	p.Composition = &compositionProvision{RecordDispatchAdmissions: true}
	for _, id := range []RunID{{"worker", 1}, {"peer", 1}} {
		p.Composition.Roots = append(p.Composition.Roots, compositionRoot{
			Run: id, Members: []RunID{id}, Capabilities: []string{"model"},
			Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1},
		})
	}
	path := filepath.Join(t.TempDir(), "gate-peer.sqlite")
	s, err := Create(ctx, path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	r, err := s.Commit(ctx, current(t, s, "worker").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := r.Projection.Data["pending"].Text
	work, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || work == nil {
		t.Fatalf("missing first reservation: %v", err)
	}
	return s, path, work
}

func commitDispatchPeer(t *testing.T, s *Store, epoch int64) int64 {
	t.Helper()
	ctx := t.Context()
	peer, err := s.Commit(ctx, current(t, s, "peer").Authority(), "")
	if err != nil || peer.Epoch != epoch {
		t.Fatalf("peer witness at epoch %d: state %+v, err %v", epoch, peer, err)
	}
	fact := compositionIdentity("alan/fact/execution/v1", []any{peer.ID, peer.Sequence})
	var order int64
	if err := s.conn.QueryRowContext(ctx, "SELECT decision_order FROM composition_facts WHERE id=?", fact).Scan(&order); err != nil {
		t.Fatal(err)
	}
	return order
}

func insertPhantomGate(t *testing.T, s *Store, work *adapterWork, after int64) {
	t.Helper()
	if _, err := s.conn.ExecContext(t.Context(), "INSERT INTO adapter_dispatch_admissions VALUES(?,?,?)",
		work.attempt.Intent.ID, work.attempt.Prepared.Digest, after); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchAdmissionRequiresOriginalOwner(t *testing.T) {
	s, path, work := dispatchPeerStore(t)
	commitDispatchPeer(t, s, 1) // The worker already spent its reservation.
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Close() })
	after := commitDispatchPeer(t, owner, 2)
	insertPhantomGate(t, owner, work, after)
	rejectDispatchHistory(t, owner, path)
}

func TestDispatchAdmissionResultCannotSurviveAsUnwrittenResponse(t *testing.T) {
	ctx := t.Context()
	s, path, work := dispatchPeerStore(t)
	id := work.attempt.Intent.ID
	gate, reason, err := s.registerDispatch(ctx, id)
	if err != nil || gate == nil || reason != "" {
		t.Fatalf("original gate: %q, %v", reason, err)
	}
	s.finishDispatch(gate, nil)
	commitDispatchPeer(t, s, 1)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Close() })
	commitDispatchPeer(t, owner, 2)
	// An ordinary adapter has no cross-owner receipt ingress. A correctly
	// correlated first result still had to be written before its worker died.
	c := effects.Candidate{
		ID: id, Key: work.attempt.Prepared.Request.Key, Disposition: "validated",
		Payload: map[string]Value{"answer": Text("impossible")},
	}
	values, disposition := effects.Result(work.attempt.Prepared.Request.Output.Record, c)
	r := AdapterResult{Candidate: c, Values: values, Disposition: disposition}
	r.Digest = adapterResultDigest(r)
	tx, err := beginOwnedTx(ctx, owner.conn)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO adapter_results VALUES(?,CAST(? AS BLOB))", id, compositionJSON(r)); err != nil {
		t.Fatal(err)
	}
	if err = owner.recordCompositionSource(ctx, tx, compositionExternalResultFact, id); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	rejectDispatchHistory(t, owner, path)
}

func TestDispatchAdmissionRequiresCommittedPrefix(t *testing.T) {
	s, path, work := dispatchPeerStore(t)
	commitDispatchPeer(t, s, 1)
	commitDispatchPeer(t, s, 1)
	terminal := commitDispatchPeer(t, s, 1)
	// The independent peer's terminal reduction and root freeze share one
	// transaction. The worker is still eligible on both sides of it.
	var kind string
	if err := s.conn.QueryRowContext(t.Context(), "SELECT kind FROM composition_facts WHERE decision_order=?", terminal+1).Scan(&kind); err != nil || kind != string(compositionFreezeFact) {
		t.Fatalf("terminal fact %d missing its atomic freeze: %q, %v", terminal, kind, err)
	}
	insertPhantomGate(t, s, work, terminal)
	rejectDispatchHistory(t, s, path)
}

func concurrentDispatchStore(t *testing.T, capacity int64) (*Store, string, []string) {
	t.Helper()
	ctx := t.Context()
	p := Provision{}
	members := []RunID{{"a", 1}, {"b", 1}}
	for _, m := range members {
		addAdapterProgram(t, &p, m.Subject, true)
	}
	p.Composition = &compositionProvision{RecordDispatchAdmissions: true, Roots: []compositionRoot{{
		Run: members[0], Members: members, Capabilities: []string{"model"},
		Limits: compositionLimits{MaxEffects: 2, MaxInFlight: capacity},
	}}}
	path := filepath.Join(t.TempDir(), "concurrent.sqlite")
	s, err := Create(ctx, path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	var ids []string
	for _, m := range members {
		state, err := s.Commit(ctx, current(t, s, m.Subject).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, state.Projection.Data["pending"].Text)
	}
	return s, path, ids
}

func TestDispatchAdmissionsAllowConcurrentCallsWithinRootCapacity(t *testing.T) {
	s, path, ids := concurrentDispatchStore(t, 2)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	entered, release := make(chan string, 2), make(chan struct{})
	adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		entered <- p.Request.ID
		select {
		case <-release:
		case <-ctx.Done():
		}
		return callbackAdapter{}.Execute(ctx, p)
	}}
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 2, Adapters: map[string]effects.Adapter{"model": adapter}})
		done <- err
	}()
	called := map[string]bool{}
	for range ids {
		select {
		case id := <-entered:
			called[id] = true
		case err := <-done:
			t.Fatalf("runner exited before both calls: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	cut, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		e, ok := cut.Effect(id)
		if !called[id] || !ok || e.DispatchAdmission == nil || e.AdapterResult != nil {
			t.Fatalf("missing in-flight admission for %s", id)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	cut, err = InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		e, ok := cut.Effect(id)
		if !ok || e.AdapterResult == nil || e.AdapterResult.Disposition != "validated" {
			t.Fatalf("missing retained result for %s", id)
		}
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	owner.Close()
}

func TestDispatchAdmissionsRejectProvenRootCapacityOverlap(t *testing.T) {
	ctx := t.Context()
	s, path, ids := concurrentDispatchStore(t, 1)
	// Bypass the runner's capacity check to construct a contradiction. Both
	// calls pass their gates before either original result is retained.
	var work []*adapterWork
	var gates []*dispatchOpportunity
	for _, id := range ids {
		w, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
		if err != nil || w == nil {
			t.Fatalf("missing reservation for %s: %v", id, err)
		}
		gate, reason, err := s.registerDispatch(ctx, id)
		if err != nil || reason != "" || gate == nil {
			t.Fatalf("missing gate for %s: %q, %v", id, reason, err)
		}
		work = append(work, w)
		gates = append(gates, gate)
	}
	for i, w := range work {
		c := callbackAdapter{}.Execute(ctx, w.attempt.Prepared)
		if _, err := s.recordAdapterResult(ctx, w.attempt, c); err != nil {
			t.Fatal(err)
		}
		s.finishDispatch(gates[i], nil)
	}
	rejectDispatchHistory(t, s, path)
}
