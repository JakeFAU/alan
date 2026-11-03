package executor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

const recoveryController = "retirement-controller"

type recoveryFixture struct {
	path, invocation, lifecycle string
	target                      RunID
}

// Stage the two intents at epoch one. Each subsequent character invokes one
// real owner operation. X ends the controller ordinarily. R drops the owner
// and reopens the committed store; hard process loss is tested separately.
func recoveryLedger(t *testing.T, scoped bool, schedule string) recoveryFixture {
	t.Helper()
	ctx := t.Context()
	p, profile := invocationFixture(t, "succeeded", 1)
	addRetirementController(t, &p, []string{"CancelRun"})
	if scoped {
		profile.Roots[0].Members = append(profile.Roots[0].Members, RunID{recoveryController, 1})
	}
	f := recoveryFixture{path: filepath.Join(t.TempDir(), "recovery.sqlite")}
	s, err := createInvocationStore(ctx, f.path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	emitted, err := s.Commit(ctx, current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	f.invocation = emitted.Projection.Data["pending"].Text
	if a, err := s.AuthorizeIntent(ctx, f.invocation); err != nil || !a.Allowed {
		t.Fatal(a, err)
	}
	f.target = compositionMemberID(compositionInvocationID(s.Namespace(), f.invocation), "entry")
	input, err := s.AdmitInput(ctx, Input{TransportID: "control", LogicalID: "control", Target: recoveryController, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(f.target.Subject)}})
	if err != nil || !input.Accepted {
		t.Fatal(input, err)
	}
	if _, err := s.Commit(ctx, current(t, s, recoveryController).Authority(), input.ObservationID); err != nil {
		t.Fatal(err)
	}
	req := committedRequest(t, s, recoveryController)
	f.lifecycle = req.Intent.ID
	for _, action := range schedule {
		switch action {
		case 'C':
			a, err := s.AdmitInvocation(ctx, f.invocation)
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
		case 'A':
			_, err = s.AuthorizeIntent(ctx, f.lifecycle)
		case 'D':
			_, err = s.DecideLifecycle(ctx, req)
		case 'X':
			_, err = s.Commit(ctx, current(t, s, recoveryController).Authority(), "")
		case 'R':
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = Open(ctx, f.path, DefaultLimits())
		default:
			t.Fatalf("invalid action %c", action)
		}
		if err != nil {
			t.Fatalf("%c in %q: %v", action, schedule, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = nil
	return f
}

// Read independent durable sources without calling any validator or using the
// model's action interpreter. Creations counts the focal origin named by the
// admission; MemberEpoch comes from the separate physical run.
func (f recoveryFixture) project(t *testing.T, db *sql.DB) recoveryModelCut {
	t.Helper()
	ctx := t.Context()
	var c recoveryModelCut
	require := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	row := func(q string, args ...any) *sql.Row { return db.QueryRowContext(ctx, q, args...) }
	require(row("SELECT count(*) FROM composition_static_members WHERE subject=?", recoveryController).Scan(&c.Scoped))
	require(row("SELECT epoch FROM runs WHERE subject='root'").Scan(&c.RootEpoch))
	require(row("SELECT epoch FROM runs WHERE subject=?", recoveryController).Scan(&c.SourceEpoch))
	require(row("SELECT coalesce(max(epoch),0) FROM runs WHERE subject=?", f.target.Subject).Scan(&c.MemberEpoch))
	require(row("SELECT coalesce(max(epoch),0) FROM history WHERE subject=? AND sequence=3", recoveryController).Scan(&c.CloseEpoch))
	var admission compositionAdmission
	if recoveryRecord(t, row("SELECT descriptor FROM composition_admission_decisions WHERE operation_id=?", f.invocation), &admission) && admission.Accepted {
		for _, id := range admission.Members {
			if id == f.target {
				c.Creations++
			}
		}
	}
	var a Authorization
	if recoveryRecord(t, row("SELECT result FROM attempts WHERE intent_id=?", f.lifecycle), &a) {
		c.AuthEpoch, c.AuthSequence = int(a.Source.Epoch), int(a.Source.Sequence)
		c.AuthGeneration, c.AuthRevision = int(a.Source.Run.Generation), int(a.Source.Revision)
		c.AuthReason = a.Reason
	}
	var d LifecycleResult
	if recoveryRecord(t, row("SELECT result FROM lifecycle_results WHERE id=?", f.lifecycle), &d) {
		c.Reason = d.Reason
		if d.Before != nil {
			c.TargetGeneration, c.TargetRevision, c.TargetEpoch = int(d.Before.Run.Generation), int(d.Before.Revision), int(d.Before.Epoch)
		}
	}
	rows, err := db.QueryContext(ctx, "SELECT descriptor FROM composition_facts ORDER BY decision_order")
	require(err)
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		require(rows.Scan(&raw))
		var fact compositionFact
		require(json.Unmarshal(raw, &fact))
		switch {
		case fact.Kind == compositionAdmissionFact && fact.OperationID == f.invocation:
			c.Facts += "C"
		case fact.Kind == compositionAuthorizationFact && fact.OperationID == f.lifecycle:
			c.Facts += "A"
		case fact.Kind == compositionLifecycleFact && fact.OperationID == f.lifecycle:
			c.Facts += "D"
		case fact.Kind == compositionExecutionFact && fact.Run.Subject == recoveryController && fact.Sequence == 3:
			c.Facts += "X"
		}
	}
	require(rows.Err())
	return c
}

func recoveryRecord(t *testing.T, row *sql.Row, dst any) bool {
	t.Helper()
	var raw []byte
	err := row.Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatal(err)
	}
	return true
}

// Mutations are made only on a fresh copy of a genuine closed database.
func (f recoveryFixture) copy(t *testing.T) recoveryFixture {
	t.Helper()
	raw, err := os.ReadFile(f.path)
	if err != nil {
		t.Fatal(err)
	}
	f.path = filepath.Join(t.TempDir(), "copy.sqlite")
	if err := os.WriteFile(f.path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f recoveryFixture) mutate(t *testing.T, db *sql.DB, mutation string) {
	t.Helper()
	ctx := t.Context()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	facts, err := readCompositionFacts(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	replaceFact := func(fact compositionFact) {
		exec("UPDATE composition_facts SET decision_order=?,descriptor=? WHERE id=?", fact.Order, []byte(compositionJSON(fact)), fact.ID)
	}
	switch mutation {
	case "swap_order":
		// Rewrite both order columns and retained descriptors consistently.
		var a, d compositionFact
		for _, fact := range facts {
			if fact.Kind == compositionAuthorizationFact && fact.OperationID == f.lifecycle {
				a = fact
			}
			if fact.Kind == compositionLifecycleFact && fact.OperationID == f.lifecycle {
				d = fact
			}
		}
		if a.ID == "" || d.ID == "" {
			t.Fatal("missing facts")
		}
		exec("UPDATE composition_facts SET decision_order=? WHERE id=?", d.Order+1000, d.ID)
		a.Order, d.Order = d.Order, a.Order
		replaceFact(a)
		replaceFact(d)
	case "duplicate_creation", "missing_creation":
		var a compositionAdmission
		if !recoveryRecord(t, db.QueryRowContext(ctx, "SELECT descriptor FROM composition_admission_decisions WHERE operation_id=?", f.invocation), &a) {
			t.Fatal("missing admission")
		}
		if mutation == "duplicate_creation" {
			a.Members = append(a.Members, f.target)
		} else {
			a.Members = slices.DeleteFunc(a.Members, func(id RunID) bool { return id == f.target })
		}
		exec("UPDATE composition_admission_decisions SET descriptor=? WHERE operation_id=?", []byte(compositionJSON(a)), f.invocation)
	case "source_generation", "source_revision", "source_epoch", "source_reason":
		a, err := loadAuthorization(ctx, db, f.lifecycle)
		if err != nil {
			t.Fatal(err)
		}
		switch mutation {
		case "source_generation":
			a.Source.Run.Generation++
		case "source_revision":
			a.Source.Revision++
		case "source_epoch":
			a.Source.Epoch += 3
		case "source_reason":
			a.Allowed, a.Reason = false, "source_closed"
		}
		exec("UPDATE attempts SET result=? WHERE intent_id=?", []byte(compositionJSON(a)), f.lifecycle)
		for _, old := range facts {
			if old.OperationID == f.lifecycle && old.Kind == compositionAuthorizationFact {
				fact, err := compositionSourceEvidence(ctx, db, old.Kind, f.lifecycle)
				if err != nil {
					t.Fatal(err)
				}
				fact.Order = old.Order
				replaceFact(fact)
			}
		}
	case "missing", "target_generation", "target_revision", "target_epoch", "target_reason":
		d, err := loadLifecycleResult(ctx, db, f.lifecycle)
		if err != nil || d.Before == nil {
			t.Fatal(d, err)
		}
		switch mutation {
		case "missing":
			d.Before, d.After, d.Reason = nil, nil, "target_missing"
		case "target_generation":
			d.Before.Run.Generation++
			d.After.Run.Generation++
		case "target_revision":
			d.Before.Revision++
			d.After.Revision++
		case "target_epoch":
			d.Before.Epoch += 3
			d.After.Epoch += 3
		case "target_reason":
			d.Reason = "generation_changed"
		}
		exec("UPDATE lifecycle_results SET result=? WHERE id=?", []byte(compositionJSON(d)), f.lifecycle)
		for _, old := range facts {
			if old.OperationID != f.lifecycle || old.Kind != compositionLifecycleFact {
				continue
			}
			if mutation == "missing" {
				var scoped bool
				if err := db.QueryRowContext(ctx, "SELECT count(*) FROM composition_static_members WHERE subject=?", recoveryController).Scan(&scoped); err != nil {
					t.Fatal(err)
				}
				if !scoped {
					exec("DELETE FROM composition_facts WHERE id=?", old.ID)
					continue
				}
			}
			fact, err := compositionLifecycleEvidence(ctx, db, d)
			if err != nil {
				t.Fatal(err)
			}
			fact.Order = old.Order
			replaceFact(fact)
		}
	default:
		t.Fatalf("unknown mutation %q", mutation)
	}
}

func (f recoveryFixture) qualify(t *testing.T, reachable map[recoveryModelCut]string, mutation string, want bool, expected *recoveryModelCut) {
	t.Helper()
	db, err := sql.Open("sqlite3", f.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if mutation != "" {
		f.mutate(t, db, mutation)
	}
	c := f.project(t, db)
	var retained LifecycleResult
	decided := recoveryRecord(t, db.QueryRowContext(t.Context(), "SELECT result FROM lifecycle_results WHERE id=?", f.lifecycle), &retained)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if expected != nil && *expected != c {
		t.Fatalf("writer projection %+v, model %+v", c, *expected)
	}
	_, forward := reachable[c]
	if forward != want || recoveryCutFeasible(c) != want {
		t.Fatalf("unexpected oracle result want %v cut %+v", want, c)
	}
	before := inspectionFingerprint(t, f.path)
	ctx := t.Context()
	cut, inspect := InspectExecution(ctx, f.path, DefaultLimits())
	report, replay := Replay(ctx, f.path, DefaultLimits())
	if inspectionFingerprint(t, f.path) != before {
		t.Fatal("reader changed durable bytes")
	}
	s, open := Open(ctx, f.path, DefaultLimits())
	if s != nil {
		defer s.Close()
	}
	if want {
		if cut == nil || s == nil || inspect != nil || replay != nil || open != nil {
			t.Fatalf("acceptance inspect=%v replay=%v open=%v cut=%+v", inspect, replay, open, c)
		}
		decision, found := cut.LifecycleDecision(f.lifecycle)
		if found != decided || !reflect.DeepEqual(decision, retained) {
			t.Fatal("inspection changed the original decision or its witnesses", decision, retained)
		}
		intent, found := cut.Operation(f.lifecycle)
		if !found || intent.ID != f.lifecycle || intent.Run != (RunID{recoveryController, 1}) || intent.Target != f.target.Subject || (decided && !reflect.DeepEqual(*intent, retained.Request.Intent)) {
			t.Fatal("inspection changed the original request", intent)
		}
		for _, run := range report.Runs {
			inspected, exists := cut.Run(run.ID)
			if !exists || !reflect.DeepEqual(inspected.RunState, run) {
				t.Fatal("readers disagree on run", run.ID)
			}
			if run.Projection.Status == "open" {
				run.Epoch++
			}
			renewed, err := s.Snapshot(ctx, run.ID)
			if err != nil || !reflect.DeepEqual(renewed, run) {
				t.Fatal("reopen changed logical state or renewed the wrong epoch", run.ID, err)
			}
		}

	} else if cut != nil || s != nil || !errors.Is(inspect, ErrCorrupt) || !errors.Is(replay, ErrCorrupt) || !errors.Is(open, ErrCorrupt) || inspectionFingerprint(t, f.path) != before {
		t.Fatalf("rejection inspect=%v replay=%v open=%v cut=%+v", inspect, replay, open, c)
	}
}

func TestRecoveryModelStoreComparison(t *testing.T) {
	reachable, _ := recoveryReachable()
	for _, scoped := range []bool{false, true} {
		name := "unscoped"
		if scoped {
			name = "scoped"
		}
		t.Run(name, func(t *testing.T) {
			for _, schedule := range []string{"", "A", "C", "AD", "ADC", "RAX", "XA", "XRA", "RCARD", "CADXR", "ADCRR", "CXRRA", "AXRRD", "ARCRD"} {
				t.Run("cut_"+schedule, func(t *testing.T) {
					f := recoveryLedger(t, scoped, schedule)
					want := recoveryInitial(scoped)
					for _, action := range schedule {
						var ok bool
						want, ok = want.step(action)
						if !ok {
							t.Fatal("bad schedule")
						}
					}
					f.copy(t).qualify(t, reachable, "", true, &want)
				})
			}
			// In CAD the unscoped source retains no order between A/D and C
			// once the target-only fact is removed. The corresponding reworded
			// history is observationally compatible with ADC. Scoped ADC and
			// CAD retain a distinguishing order even at equal final epochs.
			f := recoveryLedger(t, scoped, "CAD")
			for _, mutation := range []string{"missing", "duplicate_creation", "missing_creation", "source_generation", "source_revision", "source_reason", "source_epoch", "target_generation", "target_revision", "target_reason", "target_epoch", "swap_order"} {
				if mutation == "swap_order" && !scoped {
					continue
				}
				t.Run(mutation, func(t *testing.T) { f.copy(t).qualify(t, reachable, mutation, mutation == "missing" && !scoped, nil) })
			}
			f = recoveryLedger(t, scoped, "CADRR")
			t.Run("missing_equal_final_epochs", func(t *testing.T) { f.copy(t).qualify(t, reachable, "missing", !scoped, nil) })
			// Even without a scoped source its later authorization epoch can
			// establish that creation preceded the supposed missing lookup.
			f = recoveryLedger(t, scoped, "CRAD")
			t.Run("missing_after_reopen", func(t *testing.T) { f.copy(t).qualify(t, reachable, "missing", false, nil) })
		})
	}
}
