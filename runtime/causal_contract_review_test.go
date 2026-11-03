package runtime_test

import (
	"database/sql"
	"errors"
	"testing"

	rt "github.com/JakeFAU/alan/runtime"
)

func TestCausalRecoveryContradictoryRetainedOrder(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		change  string
	}{
		{"present", "authorization after witnessed target epoch"},
		{"missing", "source reductions reversed in history"},
	} {
		t.Run(tc.change, func(t *testing.T) {
			ctx, limits := t.Context(), rt.DefaultLimits()
			path := recoveryCopy(t, tc.fixture)
			cut := recoveryCut(t, path)
			owner, err := rt.Open(ctx, path, limits)
			if err != nil {
				t.Fatal(err)
			}
			if err := owner.Close(); err != nil {
				t.Fatal(err)
			}
			next := recoveryCut(t, path)
			checkRecoveryRenewal(t, cut, next)
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var updated sql.Result
			var want int64
			if tc.fixture == "present" {
				// The controller and the younger member are both open at the
				// cut, at epochs 3 and 2. A controller authorization at epoch
				// 3 cannot precede the recorded member lookup at epoch 1.
				updated, err = db.ExecContext(ctx, `UPDATE attempts SET result=CAST(json_set(result,'$.Source.Epoch',3) AS BLOB)
					WHERE intent_id=? AND json_extract(result,'$.Source.Epoch')=2`, recoveryLife)
				want = 1
			} else {
				// Exchange this source's two journal positions without changing
				// its relation to any other source or any retained field.
				var positions int64
				if err := db.QueryRowContext(ctx, "SELECT sum(rowid) FROM history WHERE subject=?", recoveryController).Scan(&positions); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, "UPDATE history SET rowid=-rowid WHERE subject=?", recoveryController); err != nil {
					t.Fatal(err)
				}
				updated, err = db.ExecContext(ctx, "UPDATE history SET rowid=?+rowid WHERE subject=?", positions, recoveryController)
				want = 2
			}
			if err != nil {
				t.Fatal(err)
			}
			if n, err := updated.RowsAffected(); err != nil || n != want {
				t.Fatal(n, err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			before := recoveryImage(t, path)
			if cut, err := rt.InspectExecution(ctx, path, limits); cut != nil || !errors.Is(err, rt.ErrCorrupt) {
				t.Errorf("inspection accepted impossible order: %v", err)
			}
			if _, err := rt.Replay(ctx, path, limits); !errors.Is(err, rt.ErrCorrupt) {
				t.Errorf("replay accepted impossible order: %v", err)
			}
			owner, err = rt.Open(ctx, path, limits)
			if owner != nil {
				owner.Close()
			}
			if owner != nil || !errors.Is(err, rt.ErrCorrupt) || recoveryImage(t, path) != before {
				t.Fatalf("owner failed to reject impossible order without durable change: %v", err)
			}
		})
	}
}

func TestCausalRecoveryMissingWithContiguousFacts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		reverse  []string
		feasible bool
	}{
		{"both journals prove presence", nil, false},
		{"only observations prove presence", []string{"admissions"}, false},
		{"only admissions prove presence", []string{"observations"}, false},
		{"neither journal proves presence", []string{"admissions", "observations"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, limits := t.Context(), rt.DefaultLimits()
			path := recoveryCopy(t, "forged-contiguous")
			if len(tc.reverse) > 0 {
				db, err := sql.Open("sqlite3", path)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				tx, err := db.BeginTx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback()
				for _, table := range tc.reverse {
					// Exchange only the two append positions in this journal.
					for _, assignment := range []string{"-rowid WHERE rowid IN (1,2)", "3+rowid WHERE rowid IN (-1,-2)"} {
						result, err := tx.ExecContext(ctx, "UPDATE "+table+" SET rowid="+assignment)
						if err != nil {
							t.Fatal(err)
						}
						if n, err := result.RowsAffected(); err != nil || n != 2 {
							t.Fatalf("%s changed %d rows: %v", table, n, err)
						}
					}
				}
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			}
			before := recoveryImage(t, path)
			cut, inspect := rt.InspectExecution(ctx, path, limits)
			_, replay := rt.Replay(ctx, path, limits)
			if recoveryImage(t, path) != before {
				t.Fatal("reader changed the database or WAL")
			}
			owner, open := rt.Open(ctx, path, limits)
			if owner != nil {
				defer owner.Close()
			}
			t.Logf("InspectExecution=%v, Replay=%v, Open=%v", inspect, replay, open)
			if tc.feasible {
				if cut == nil || owner == nil || inspect != nil || replay != nil || open != nil {
					t.Fatal("rejected a compatible lookup")
				}
				checkRecoveryEvidence(t, cut, "target_missing", true, true)
			} else if cut != nil || owner != nil || !errors.Is(inspect, rt.ErrCorrupt) || !errors.Is(replay, rt.ErrCorrupt) || !errors.Is(open, rt.ErrCorrupt) || recoveryImage(t, path) != before {
				t.Fatal("failed to reject and preserve the contradictory cut")
			}
		})
	}
}
