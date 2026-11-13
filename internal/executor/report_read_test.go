//go:build integration

package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

func reportScaleStore(t *testing.T, roots int) *Store {
	t.Helper()
	var p Provision
	const target = "http://127.0.0.1:1"
	c := compositionProvision{Version: compositionProtocolVersion}
	for i := range roots {
		name := fmt.Sprintf("root-%02d", i)
		emit := Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(5))}}}
		addProgram(t, &p, name, []Instruction{emit, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
		p.Permissions = append(p.Permissions, Permission{Kind: "external", Source: name, Target: target, Operation: "apply", Revision: 1})
		id := RunID{name, 1}
		c.Roots = append(c.Roots, compositionRoot{Run: id, Members: []RunID{id}, Limits: compositionLimits{MaxInvocations: 4, MaxCreatedRuns: 8, MaxDepth: 3, MaxEffects: 8, MaxInFlight: 1}})
	}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/report.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	for i := range roots {
		rootReportCommit(t, s, fmt.Sprintf("root-%02d", i))
	}
	return s
}

func reportAuthorizer(t *testing.T, s *Store, authorize func(int, string, string, string) int) {
	t.Helper()
	if err := s.conn.Raw(func(driver any) error {
		driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(authorize)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertFinishRunFrozen(t *testing.T, ctx context.Context, s *Store, initial RunReport, quiescent bool, runErr error) (RunReport, error) {
	t.Helper()
	got, err := s.finishRun(ctx, initial, quiescent, runErr)
	want, wantErr := s.finishRunFrozenReport(ctx, initial, quiescent, runErr)
	assertFrozenReportResult(t, got, err, want, wantErr, runErr)
	return got, err
}

func assertFrozenReportResult(t *testing.T, got RunReport, err error, want RunReport, wantErr, runErr error) {
	t.Helper()
	if !reflect.DeepEqual(got, want) || (err == nil) != (wantErr == nil) ||
		(err != nil && err.Error() != wantErr.Error()) {
		t.Fatalf("candidate: %#v, %v\n frozen: %#v, %v", got, err, want, wantErr)
	}
	for _, cause := range []error{runErr, sql.ErrNoRows, sql.ErrConnDone, context.Canceled, ErrCorrupt} {
		if cause != nil && errors.Is(err, cause) != errors.Is(wantErr, cause) {
			t.Fatalf("different joined error cause %v: %v vs %v", cause, err, wantErr)
		}
	}
}

// SQLite's authorizer sees each statement's projected columns at prepare time.
// Each target query references its column once, including the final open-runs pass.
func TestFinishRunReportDescriptorQueryCount(t *testing.T) {
	for _, size := range []int{1, 4, 8} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			s := reportScaleStore(t, size)
			counts := [2]int{}
			reportAuthorizer(t, s, func(op int, table, column, _ string) int {
				if op == sqlite3.SQLITE_READ {
					switch {
					case table == "composition_roots" && column == "descriptor":
						counts[0]++
					case table == "runs" && column == "projection":
						counts[1]++
					}
				}
				return sqlite3.SQLITE_OK
			})
			got, err := s.finishRun(t.Context(), RunReport{}, false, nil)
			candidateCounts := counts
			counts = [2]int{}
			want, wantErr := s.finishRunFrozenReport(t.Context(), RunReport{}, false, nil)
			reportAuthorizer(t, s, nil)
			assertFrozenReportResult(t, got, err, want, wantErr, nil)
			if err != nil || len(got.Roots) != size || len(got.OpenRuns) != size || len(got.UnresolvedIntents) != size {
				t.Fatalf("fixture did not produce %d roots, runs and intents: %#v, %v", size, got, err)
			}
			if counts != [2]int{2 * size, 2 * size} || candidateCounts != [2]int{1, 1} {
				t.Fatalf("root descriptor and run projection statements: candidate %v, frozen %v", candidateCounts, counts)
			}
		})
	}
}

func execReportFixture(t *testing.T, s *Store, query string, args ...any) {
	t.Helper()
	if _, err := s.conn.ExecContext(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func reportFixtureRoot(t *testing.T, s *Store) string {
	t.Helper()
	var id string
	if err := s.conn.QueryRowContext(t.Context(), "SELECT id FROM composition_roots").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// Insert deliberately incomplete evidence in the disposable fixture. A wrong
// indexed root_id ensures that the report follows the point lookup's scope_id.
func insertReportPublication(t *testing.T, s *Store, root string) {
	t.Helper()
	execReportFixture(t, s, "PRAGMA foreign_keys=OFF")
	execReportFixture(t, s, "INSERT INTO composition_root_publications VALUES(?,?,1,?)",
		"root:"+root, "wrong-indexed-root", `{ "ID": "publication", "Status": "succeeded" }`)
}

func TestFinishRunReportRootPreparationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func(*testing.T, *Store, string)
		wantErr  bool
		wantRoot bool
	}{
		{"absent gates", func(*testing.T, *Store, string) {}, false, false},
		{"published noncanonical by scope", insertReportPublication, false, true},
		{"published corrupt freeze and closure", func(t *testing.T, s *Store, root string) {
			insertReportPublication(t, s, root)
			execReportFixture(t, s, "INSERT INTO composition_freezes VALUES(?,?, '{')", "root:"+root, root)
			execReportFixture(t, s, "INSERT INTO composition_closures VALUES(?,?,1, '{')", "root:"+root, root)
		}, false, true},
		{"unpublished corrupt freeze", func(t *testing.T, s *Store, root string) {
			execReportFixture(t, s, "INSERT INTO composition_freezes VALUES(?,?, '{')", "root:"+root, root)
		}, true, false},
		{"unpublished corrupt closure", func(t *testing.T, s *Store, root string) {
			execReportFixture(t, s, "PRAGMA foreign_keys=OFF")
			execReportFixture(t, s, "INSERT INTO composition_closures VALUES(?,?,1, '{')", "root:"+root, root)
		}, true, false},
		{"malformed publication", func(t *testing.T, s *Store, root string) {
			insertReportPublication(t, s, root)
			execReportFixture(t, s, "UPDATE composition_root_publications SET descriptor='{'")
		}, true, false},
		{"odd storage class publication", func(t *testing.T, s *Store, root string) {
			insertReportPublication(t, s, root)
			execReportFixture(t, s, "UPDATE composition_root_publications SET descriptor=123")
		}, true, false},
		{"present NULL publication", func(t *testing.T, s *Store, root string) {
			insertReportPublication(t, s, root)
			// SQLITE_IGNORE exposes SQL NULL to Scan without changing the
			// actual row's presence or the declared NOT NULL schema.
			reportAuthorizer(t, s, func(op int, table, column, _ string) int {
				if op == sqlite3.SQLITE_READ && table == "composition_root_publications" && column == "descriptor" {
					return sqlite3.SQLITE_IGNORE
				}
				return sqlite3.SQLITE_OK
			})
		}, true, false},
		{"root descriptor malformed", func(t *testing.T, s *Store, root string) {
			execReportFixture(t, s, "UPDATE composition_roots SET descriptor='{' WHERE id=?", root)
		}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := reportScaleStore(t, 1)
			tc.mutate(t, s, reportFixtureRoot(t, s))
			got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{Steps: 11, Dispatches: 7}, true, nil)
			if (err != nil) != tc.wantErr || got.Steps != 11 || got.Dispatches != 7 || !got.Quiescent {
				t.Fatal(got, err)
			}
			if tc.wantErr {
				if got.Roots != nil || len(got.OpenRuns) != 0 {
					t.Fatal(got)
				}
				return
			}
			if len(got.Roots) != 1 {
				t.Fatal(got)
			}
			root := got.Roots[0]
			if tc.wantRoot {
				if root.Outcome == nil || root.Outcome.ID != "publication" || root.Pending != "" {
					t.Fatal(root)
				}
			} else if root.Outcome != nil || root.Pending != "members" {
				t.Fatal(root)
			}
		})
	}
}

func TestFinishRunReportPreparationAuthorization(t *testing.T) {
	for _, tc := range []struct{ table, column string }{
		{"composition_freezes", "descriptor"},
		{"composition_closures", "root_id"},
		{"composition_closures", "decision_order"},
		{"composition_closures", "descriptor"},
	} {
		for _, published := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s.%s/published=%t", tc.table, tc.column, published), func(t *testing.T) {
				s := reportScaleStore(t, 1)
				if published {
					insertReportPublication(t, s, reportFixtureRoot(t, s))
				}
				denials := 0
				reportAuthorizer(t, s, func(op int, table, column, _ string) int {
					if op == sqlite3.SQLITE_READ && table == tc.table && column == tc.column {
						denials++
						return sqlite3.SQLITE_DENY
					}
					return sqlite3.SQLITE_OK
				})
				got, err := s.finishRun(t.Context(), RunReport{}, false, nil)
				candidateDenials := denials
				denials = 0
				want, wantErr := s.finishRunFrozenReport(t.Context(), RunReport{}, false, nil)
				reportAuthorizer(t, s, nil)
				assertFrozenReportResult(t, got, err, want, wantErr, nil)
				if published {
					if err != nil || len(got.Roots) != 1 || got.Roots[0].Outcome == nil ||
						candidateDenials != 1 || denials != 0 {
						t.Fatal(got, err, candidateDenials, denials)
					}
				} else if err == nil || got.Roots != nil || candidateDenials != 2 || denials != 1 {
					t.Fatal(got, err, candidateDenials, denials)
				}
			})
		}
	}
}

func TestFinishRunReportRunBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		query      string
		wantErr    error
		errText    string
		partialRun bool
	}{
		{"missing run", "PRAGMA foreign_keys=OFF; DELETE FROM runs WHERE subject='a'", sql.ErrNoRows, "", false},
		{"missing subject", "PRAGMA foreign_keys=OFF; DELETE FROM subjects WHERE id='a'", sql.ErrNoRows, "", false},
		{"missing unrelated subject", "PRAGMA foreign_keys=OFF; DELETE FROM subjects WHERE id='unscoped'", sql.ErrNoRows, "", true},
		{"bad root run", "UPDATE runs SET projection='{' WHERE subject='a'", nil, "unexpected EOF", false},
		{"bad unrelated terminal run", "UPDATE runs SET projection='{\"Status\":\"succeeded\",\"Unknown\":true}' WHERE subject='unscoped'", nil, `unknown field "Unknown"`, true},
		{"nonintegral epoch", "UPDATE runs SET epoch=1.5 WHERE subject='unscoped'", nil, `column index 1, name "epoch"`, true},
		{"convertible epoch BLOB", "UPDATE runs SET epoch=x'31' WHERE subject='unscoped'", nil, "", false},
		{"convertible program BLOB", "PRAGMA foreign_keys=OFF; UPDATE runs SET program_id=CAST(program_id AS BLOB) WHERE subject='unscoped'", nil, "", false},
		{"noninteger subject revision", "UPDATE subjects SET revision='invalid' WHERE id='unscoped'", nil, `column index 3, name "revision"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := rootReportStore(t)
			execReportFixture(t, s, tc.query)
			// The generic table inspector treats all BLOBs as JSON. Check
			// byte-for-byte dump equality even for an invalid BLOB fixture.
			before := reportFixtureDump(t, s)
			runErr := errors.New("earlier run failure")
			initial := RunReport{Steps: 3, Dispatches: 5, RejectedAdmissions: 7}
			got, err := assertFinishRunFrozen(t, t.Context(), s, initial, true, runErr)
			if !errors.Is(err, runErr) || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) ||
				(tc.errText != "" && !strings.Contains(err.Error(), tc.errText)) {
				t.Fatal(err)
			}
			if tc.partialRun {
				if len(got.Roots) != 2 || !reflect.DeepEqual(got.OpenRuns, []RunID{{"a", 1}, {"b", 1}}) {
					t.Fatalf("later run error lost partial output: %#v", got)
				}
			} else if tc.wantErr != nil || tc.errText != "" {
				if got.Roots != nil || len(got.OpenRuns) != 0 {
					t.Fatalf("root error exposed partial root output: %#v", got)
				}
			} else if err.Error() != runErr.Error() || len(got.Roots) != 2 || len(got.OpenRuns) != 3 {
				t.Fatal(got, err)
			}
			if before != reportFixtureDump(t, s) {
				t.Fatal("report modified runs or subjects")
			}
		})
	}
}

func reportFixtureDump(t *testing.T, s *Store) string {
	t.Helper()
	var dump string
	if err := s.conn.QueryRowContext(t.Context(), `SELECT
		coalesce((SELECT group_concat(record) FROM (
			SELECT quote(subject)||quote(generation)||quote(program_id)||quote(initial)
				||quote(epoch)||quote(sequence)||quote(projection) AS record
			FROM runs ORDER BY subject,generation
		)), '') ||
		coalesce((SELECT group_concat(record) FROM (
			SELECT quote(id)||quote(generation)||quote(owner)||quote(revision)
				||quote(retired)||quote(retry_cancelled) AS record
			FROM subjects ORDER BY id
		)), '')`).Scan(&dump); err != nil {
		t.Fatal(err)
	}
	return dump
}

func TestFinishRunReportFreshEffectEvidence(t *testing.T) {
	s, _, roots := rootReportStore(t)
	first, later := roots["a"], roots["b"]
	if first.ID > later.ID {
		first, later = later, first
	}
	id := rootReportCommit(t, s, first.Run.Subject)
	attempt, fresh, err := s.prepareExternal(t.Context(), id)
	if err != nil || !fresh || !attempt.Allowed {
		t.Fatal(attempt, fresh, err)
	}
	check := func(unknown, pending bool) {
		t.Helper()
		got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
		if err != nil || len(got.Roots) != 2 || slices.Contains(got.Roots[0].UnknownExternal, id) != unknown ||
			slices.Contains(got.UnknownExternal, id) != unknown || (len(got.PendingDeliveries) == 1) != pending ||
			(len(got.UnresolvedIntents) == 1) != pending {
			t.Fatal(got, err)
		}
	}
	check(true, true)
	receipt := receiptFor(attempt.Request)
	if err := s.recordExternalReceipt(t.Context(), attempt, receipt); err != nil {
		t.Fatal(err)
	}
	check(false, true)
	outcome, err := s.externalOutcome(t.Context(), id)
	if err != nil || outcome.Receipt == nil {
		t.Fatal(outcome, err)
	}
	if err := s.settleExternal(t.Context(), outcome); err != nil {
		t.Fatal(err)
	}
	check(false, false)
	loser := receipt
	loser.Total++
	if err := s.recordExternalReceipt(t.Context(), attempt, loser); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	check(false, false)

	// Later malformed rows are visible to the batch but their validation
	// must remain behind the first root's original receipt correlation.
	execReportFixture(t, s, "UPDATE composition_roots SET descriptor='{' WHERE id=?", later.ID)
	execReportFixture(t, s, "UPDATE runs SET projection='{' WHERE subject=?", later.Run.Subject)
	badReceipt := receipt
	badReceipt.Delta++
	execReportFixture(t, s, "UPDATE external_receipts SET descriptor=json_set(descriptor, '$.Delta', ?) WHERE intent_id=?", badReceipt.Delta, id)
	_, receiptErr := s.externalOutcome(t.Context(), id)
	if receiptErr == nil {
		t.Fatal("fixture requires a receipt correlation failure")
	}
	got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, true, nil)
	if err == nil || err.Error() != receiptErr.Error() || got.Roots != nil {
		t.Fatal(got, err, receiptErr)
	}
}

func TestFinishRunReportPublishedAndRetiredHistory(t *testing.T) {
	for _, retire := range []bool{false, true} {
		t.Run(fmt.Sprintf("retire=%t", retire), func(t *testing.T) {
			s := invocationStore(t, "succeeded", 1, func(p *Provision, _ *compositionProvision) {
				if retire {
					addRetirementController(t, p, []string{"RetireSubject"})
					p.Permissions = append(p.Permissions, Permission{
						Kind: "lifecycle", Source: "retirement-controller", Target: "root", Operation: "RetireSubject", Revision: 1,
					})
				}
			})
			child := invokeOnce(t, s)
			if retire {
				input, err := s.AdmitInput(t.Context(), Input{
					TransportID: "retire-root", LogicalID: "retire-root", Target: "retirement-controller", Generation: 1,
					Obligation: "control", Payload: map[string]Value{"target": Text("root")},
				})
				if err != nil || !input.Accepted {
					t.Fatal(input, err)
				}
				retirementStep(t, s, RunID{"retirement-controller", 1}, input.ObservationID)
				decision := authorizedDecision(t, s, committedRequest(t, s, "retirement-controller"))
				if !decision.Accepted || !decision.After.Retired {
					t.Fatal(decision)
				}
			}
			before, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
			if err != nil || len(before.Roots) != 1 || before.Roots[0].Outcome != nil ||
				!slices.Contains(before.Roots[0].PendingInvocations, child.Invocation.OperationID) {
				t.Fatal(before, err)
			}
			if retire {
				publication, err := s.SettleRoot(t.Context(), before.Roots[0].RootID)
				if err != nil || !publication.Ready || publication.Outcome.Status != "cancelled" {
					t.Fatal(publication, err)
				}
			} else {
				run, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
				if err != nil || !run.Quiescent {
					t.Fatal(run, err)
				}
			}
			after, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, true, nil)
			want := "succeeded"
			if retire {
				want = "cancelled"
			}
			if err != nil || len(after.Roots) != 1 || after.Roots[0].Outcome == nil ||
				after.Roots[0].Outcome.Status != want || after.Roots[0].LocalStatus != want {
				t.Fatal(after, err)
			}
		})
	}
}

func TestRootCompletionsReportBatchDuplicateFallback(t *testing.T) {
	for _, tc := range []struct {
		name, query, point string
	}{
		{"root", reportRootsSQL,
			"SELECT descriptor FROM composition_roots WHERE id=?"},
		{"run", reportRunsSQL,
			"SELECT r.program_id,r.epoch,r.sequence,s.revision,r.projection FROM runs r JOIN subjects s ON s.id=r.subject WHERE r.subject=? AND r.generation=?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := reportScaleStore(t, 1)
			batches, points := 0, 0
			q := &rootReportQueryer{
				queryer: s.conn,
				rows: func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
					if query == tc.query {
						batches++
						return s.conn.QueryContext(ctx, tc.query+" UNION ALL "+tc.query)
					}
					return s.conn.QueryContext(ctx, query, args...)
				},
				row: func(ctx context.Context, query string, args ...any) *sql.Row {
					if query == tc.point {
						points++
					}
					return s.conn.QueryRowContext(ctx, query, args...)
				},
			}
			s.mu.Lock()
			got, err := readRootCompletions(t.Context(), q)
			s.mu.Unlock()
			assertRootReportOriginal(t, s.conn, got, err)
			wantPoints := 1
			if tc.name == "root" {
				wantPoints = 2
			}
			if err != nil || batches != 1 || points != wantPoints {
				t.Fatal(got, err, batches, points)
			}
		})
	}
}

func TestFinishRunReportNoCompositionAndCancellation(t *testing.T) {
	s, _ := newStore(t, true)
	runErr := errors.New("run stopped")
	initial := RunReport{Steps: 2, Dispatches: 1, RejectedAdmissions: 3}
	got, err := assertFinishRunFrozen(t, t.Context(), s, initial, false, runErr)
	if err.Error() != runErr.Error() || got.Roots != nil || !reflect.DeepEqual(got.OpenRuns, []RunID{{"worker", 42}}) {
		t.Fatal(got, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err = assertFinishRunFrozen(t, ctx, s, initial, false, runErr)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, runErr) || got.Roots != nil || len(got.OpenRuns) != 0 {
		t.Fatal(got, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	got, err = assertFinishRunFrozen(t, t.Context(), s, initial, false, runErr)
	if !errors.Is(err, sql.ErrConnDone) || !errors.Is(err, runErr) {
		t.Fatal(got, err)
	}
}

func TestFinishRunReportReaderCallLifetimes(t *testing.T) {
	first := reportScaleStore(t, 1)
	second := reportScaleStore(t, 1)
	firstID := reportFixtureRoot(t, first)
	// Force exactly the same root and run keys across independent Stores. Only
	// the root table's lookup key is shared, without rewriting other evidence.
	execReportFixture(t, second, "PRAGMA foreign_keys=OFF")
	execReportFixture(t, second, "UPDATE composition_roots SET id=?", firstID)
	assertOpen := func(s *Store, want bool) {
		t.Helper()
		got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
		if err != nil || len(got.Roots) != 1 || got.Roots[0].RootID != firstID ||
			(len(got.OpenRuns) == 1) != want || (got.Roots[0].LocalStatus == "open") != want {
			t.Fatal(got, err)
		}
	}
	assertOpen(first, true)
	assertOpen(second, true)
	execReportFixture(t, second, "UPDATE runs SET projection=json_set(projection,'$.Status','succeeded')")
	assertOpen(second, false)
	assertOpen(first, true)
}

func TestReportReaderCachedCancellationAndGenerations(t *testing.T) {
	s, _, _ := rootReportStore(t)
	execReportFixture(t, s, `INSERT INTO runs
		SELECT subject,2,program_id,initial,epoch,sequence,json_set(projection,'$.Status','succeeded')
		FROM runs WHERE subject='a' AND generation=1`)
	s.mu.Lock()
	defer s.mu.Unlock()
	reader := reportReader{q: s.conn}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	rootID := ""
	if err := s.conn.QueryRowContext(ctx, "SELECT id FROM composition_roots LIMIT 1").Scan(&rootID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []RunID{{"a", 1}, {"a", 2}} {
		got, err := reader.run(ctx, id)
		want, wantErr := loadRun(ctx, s.conn, id)
		if err != nil || wantErr != nil || !reflect.DeepEqual(got, want) {
			t.Fatal(got, err, want, wantErr)
		}
	}
	if _, err := reader.root(ctx, rootID); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := reader.run(ctx, RunID{"a", 1}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := reader.root(ctx, rootID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := reader.prepareRoot(ctx, rootID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestFinishRunReportEmptyRootTraversal(t *testing.T) {
	s := reportScaleStore(t, 1)
	// Provision requires a root. A disposable empty retained root table tests
	// the traversal's non-nil empty result and its unused preparation tables.
	execReportFixture(t, s, "PRAGMA foreign_keys=OFF; DELETE FROM composition_roots; DROP TABLE composition_freezes")
	got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
	if err != nil || got.Roots == nil || len(got.Roots) != 0 || len(got.OpenRuns) != 1 {
		t.Fatal(got, err)
	}
}

func TestFinishRunReportMissingPreparationTable(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(fmt.Sprintf("published=%t", published), func(t *testing.T) {
			s := reportScaleStore(t, 1)
			if published {
				insertReportPublication(t, s, reportFixtureRoot(t, s))
			}
			execReportFixture(t, s, "DROP TABLE composition_freezes")
			got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
			if published {
				if err != nil || len(got.Roots) != 1 || got.Roots[0].Outcome == nil {
					t.Fatal(got, err)
				}
			} else if err == nil || got.Roots != nil || !strings.Contains(err.Error(), "no such table: composition_freezes") {
				t.Fatal(got, err)
			}
		})
	}
}

func TestFinishRunReportFreezePresenceWithFunctionAuthorization(t *testing.T) {
	s := reportScaleStore(t, 1)
	root := reportFixtureRoot(t, s)
	execReportFixture(t, s, "INSERT INTO composition_freezes VALUES(?,?, '{')", "root:"+root, root)
	reportAuthorizer(t, s, func(op int, _, function, _ string) int {
		if op == sqlite3.SQLITE_FUNCTION && function == "typeof" {
			return sqlite3.SQLITE_IGNORE
		}
		return sqlite3.SQLITE_OK
	})
	got, err := assertFinishRunFrozen(t, t.Context(), s, RunReport{}, false, nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") || got.Roots != nil {
		t.Fatal(got, err)
	}
}
