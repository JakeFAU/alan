//go:build integration

package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

const rootReportMembershipSQL = "SELECT root_id FROM composition_static_members WHERE subject=? AND generation=? UNION ALL SELECT root_id FROM composition_members WHERE subject=? AND generation=?"

// This wrapper counts the original SQL predicate, without adding a production
// hook. Overrides below are explicitly synthetic helper tests, never execution
// histories or evidence for supported writes during a report.
type rootReportQueryer struct {
	queryer
	lookups     []RunID
	chargeReads []string
	row         func(context.Context, string, ...any) *sql.Row
	rows        func(context.Context, string, ...any) (*sql.Rows, error)
}

func (q *rootReportQueryer) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	q.countChargeRead(query)
	if query == rootReportMembershipSQL {
		q.lookups = append(q.lookups, RunID{args[0].(string), args[1].(int64)})
	}
	if q.row != nil {
		return q.row(ctx, query, args...)
	}
	return q.queryer.QueryRowContext(ctx, query, args...)
}

func (q *rootReportQueryer) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	q.countChargeRead(query)
	if q.rows != nil {
		return q.rows(ctx, query, args...)
	}
	return q.queryer.QueryContext(ctx, query, args...)
}

func (q *rootReportQueryer) countChargeRead(query string) {
	if strings.Contains(query, "FROM composition_charges") && strings.Contains(strings.ToLower(query), "sum(") {
		q.chargeReads = append(q.chargeReads, query)
	}
}

// Both roots and the unscoped source are genuinely provisioned. Only ordinary
// Commit/prepare/receipt/settlement methods create their retained histories.
// There is no transport here: an allowed, paid attempt has unknown application.
func rootReportStore(t *testing.T) (*Store, string, map[string]ResourceRoot) {
	t.Helper()
	p := Provision{}
	const target = "http://127.0.0.1:1"
	for _, name := range []string{"a", "b", "unscoped"} {
		emit := Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(5))}}}
		addProgram(t, &p, name, []Instruction{emit, emit, {Op: "receive", Match: &Match{Kind: "receipt", Provenance: "resource-receipt", Source: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Operation: Ref("pending")}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
		p.Permissions = append(p.Permissions, Permission{Kind: "external", Source: name, Target: target, Operation: "apply", Revision: 1})
	}
	c := compositionProvision{Version: compositionProtocolVersion}
	for _, name := range []string{"a", "b"} {
		c.Roots = append(c.Roots, compositionRoot{Run: RunID{name, 1}, Members: []RunID{{name, 1}}, Limits: compositionLimits{MaxInvocations: 4, MaxCreatedRuns: 8, MaxDepth: 3, MaxEffects: 8, MaxInFlight: 1}})
	}
	path := t.TempDir() + "/root-report.sqlite"
	s, err := createInvocationStore(t.Context(), path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	roots := map[string]ResourceRoot{}
	for _, name := range []string{"a", "b"} {
		r, err := compositionRootFor(t.Context(), s.conn, RunID{name, 1})
		if err != nil {
			t.Fatal(err)
		}
		roots[name] = r
	}
	return s, path, roots
}

func rootReportCommit(t *testing.T, s *Store, subject string) string {
	t.Helper()
	r, err := s.Commit(t.Context(), current(t, s, subject).Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	return r.Projection.Data["pending"].Text
}

func rootReportWant(roots map[string]ResourceRoot, unresolved, unknown map[string][]string, charges map[string]int64) []RootCompletion {
	want := []RootCompletion{}
	for _, name := range []string{"a", "b"} {
		r := roots[name]
		want = append(want, RootCompletion{RootID: r.ID, Run: r.Run, LocalStatus: "open", Limits: r.Limits,
			Charges: compositionCharge{RootID: r.ID, Effects: charges[name]}, Pending: "members", PendingInvocations: []string{},
			UnresolvedExternal: append([]string{}, unresolved[name]...), UnknownExternal: append([]string{}, unknown[name]...)})
	}
	slices.SortFunc(want, func(a, b RootCompletion) int { return strings.Compare(a.RootID, b.RootID) })
	return want
}

func rootReportTables(t *testing.T, s *Store) map[string][]map[string]any {
	t.Helper()
	tables, err := stringList(t.Context(), s.conn, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]map[string]any{}
	for _, table := range tables {
		out[table], err = inspectTable(t.Context(), s.conn, table)
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func rootReportRead(t *testing.T, s *Store, q queryer) []RootCompletion {
	t.Helper()
	// Match finishRun's owner exclusion; no fixture writes occur in this cut.
	s.mu.Lock()
	defer s.mu.Unlock()
	reports, err := readRootCompletions(t.Context(), q)
	if err != nil {
		t.Fatal(err)
	}
	original := q
	if wrapped, ok := q.(*rootReportQueryer); ok {
		original = &rootReportQueryer{queryer: wrapped.queryer, row: wrapped.row, rows: wrapped.rows}
	}
	assertRootReportOriginal(t, original, reports, nil)
	return reports
}

func assertRootReportOriginal(t *testing.T, q queryer, got []RootCompletion, gotErr error) {
	t.Helper()
	want, wantErr := readRootCompletionsOriginal(t.Context(), q)
	if !reflect.DeepEqual(got, want) || (gotErr == nil) != (wantErr == nil) ||
		(gotErr != nil && gotErr.Error() != wantErr.Error()) {
		t.Fatalf("candidate: %#v, %v\n original: %#v, %v", got, gotErr, want, wantErr)
	}
}

func TestRootCompletionsChargeAggregationOncePerReport(t *testing.T) {
	s, _, roots := rootReportStore(t)
	for _, name := range []string{"a", "b", "a", "unscoped"} {
		op := rootReportCommit(t, s, name)
		if name == "unscoped" {
			continue
		}
		attempt, fresh, err := s.prepareExternal(t.Context(), op)
		if err != nil || !fresh || !attempt.Allowed {
			t.Fatal(attempt, fresh, err)
		}
	}
	q := &rootReportQueryer{queryer: s.conn}
	got := rootReportRead(t, s, q)
	want := rootReportWant(roots,
		map[string][]string{"a": {identity("a", 1, 1, 0), identity("a", 1, 2, 0)}, "b": {identity("b", 1, 1, 0)}},
		map[string][]string{"a": {identity("a", 1, 1, 0), identity("a", 1, 2, 0)}, "b": {identity("b", 1, 1, 0)}},
		map[string]int64{"a": 2, "b": 1},
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got, want)
	}
	t.Logf("charge aggregation statements=%d, roots=%d, retained intents=4", len(q.chargeReads), len(roots))
	if len(q.chargeReads) != 1 {
		t.Fatalf("charge aggregation executed %d times for %d roots, want 1", len(q.chargeReads), len(roots))
	}
	plan, err := s.conn.QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+q.chargeReads[0])
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	for plan.Next() {
		var id, parent, unused int
		var detail string
		if err := plan.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		t.Log(detail)
	}
	if err := plan.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRootCompletionsScopeTransitions(t *testing.T) {
	for _, status := range []string{"succeeded", "failed"} {
		t.Run(status, func(t *testing.T) {
			s := invocationStore(t, status, 1, nil)
			initial := rootReportRead(t, s, s.conn)
			if len(initial) != 1 || initial[0].Pending != "members" || initial[0].Outcome != nil {
				t.Fatal(initial)
			}
			child := invokeOnce(t, s)
			for range 2 {
				retirementStep(t, s, RunID{"root", 1}, "")
			}
			pending := rootReportRead(t, s, s.conn)
			if pending[0].Pending != "descendant_outcome" || pending[0].Outcome != nil ||
				!slices.Contains(pending[0].PendingInvocations, child.Invocation.OperationID) {
				t.Fatal(pending)
			}
			if status == "failed" {
				entry := compositionMemberID(child.Invocation.ID, "entry")
				retirementStep(t, s, entry, compositionIdentity("alan/invocation-request/v1", child.Invocation.ID))
				retirementStep(t, s, entry, "")
				retirementStep(t, s, entry, "")
				early := rootReportRead(t, s, s.conn)
				p, err := s.PrepareInvocation(t.Context(), child.Invocation.OperationID)
				if err != nil || p.Freeze.Status != "failed" || early[0].Outcome != nil {
					t.Fatal(early, p, err)
				}
			}
			run, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			complete := rootReportRead(t, s, s.conn)
			if !reflect.DeepEqual(run.Roots, complete) || complete[0].Outcome == nil ||
				complete[0].Outcome.Status != "succeeded" || len(complete[0].PendingInvocations) != 0 {
				t.Fatal(run.Roots, complete)
			}
		})
	}
}

func TestRootCompletionsRetiredScope(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, func(p *Provision, _ *compositionProvision) {
		addRetirementController(t, p, []string{"RetireSubject"})
		p.Permissions = append(p.Permissions, Permission{
			Kind: "lifecycle", Source: "retirement-controller", Target: "root", Operation: "RetireSubject", Revision: 1,
		})
	})
	child := invokeOnce(t, s)
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
	before := rootReportRead(t, s, s.conn)
	if before[0].Outcome != nil || !reflect.DeepEqual(before[0].PendingInvocations, []string{child.Invocation.OperationID}) {
		t.Fatal(before)
	}
	publication, err := s.SettleRoot(t.Context(), before[0].RootID)
	if err != nil || publication.Outcome.ID == "" || publication.Outcome.Status != "cancelled" ||
		publication.Outcome.Reason != "subject_retired" {
		t.Fatal(publication, err)
	}
	after := rootReportRead(t, s, s.conn)
	if !reflect.DeepEqual(after[0].Outcome, &publication.Outcome) ||
		!reflect.DeepEqual(after[0].PendingInvocations, before[0].PendingInvocations) ||
		after[0].Charges != before[0].Charges {
		t.Fatal(before, after)
	}
}

func TestRootCompletionsChargeSQLiteSemantics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		first    string
		second   string
		want     int64
		errorHas string
	}{
		{name: "integer", first: `{"Effects":9223372036854775806}`, second: `{"Effects":1}`, want: 9223372036854775807},
		{name: "overflow", first: `{"Effects":9223372036854775807}`, second: `{"Effects":1}`, errorHas: "integer overflow"},
		{name: "malformed", first: `{`, second: `{}`, errorHas: "malformed JSON"},
		{name: "fractional", first: `{"Effects":1.5}`, second: `{}`, errorHas: "invalid syntax"},
		{name: "integer_string_and_boolean", first: `{"Effects":"5"}`, second: `{"Effects":true}`, want: 6},
		{name: "null_and_missing", first: `{"Effects":null}`, second: `{}`},
		{name: "negative", first: `{"Effects":-4}`, second: `{"Effects":2}`, want: -2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, roots := rootReportStore(t)
			for _, raw := range []string{tc.first, tc.second} {
				op := rootReportCommit(t, s, "a")
				if _, _, err := s.prepareExternal(t.Context(), op); err != nil {
					t.Fatal(err)
				}
				overwriteRootReportCharge(t, s, op, raw)
			}
			got, err := readRootCompletions(t.Context(), s.conn)
			assertRootReportOriginal(t, s.conn, got, err)
			if tc.errorHas != "" {
				if err == nil || got != nil || !strings.Contains(err.Error(), tc.errorHas) {
					t.Fatalf("charge error: %#v, %v", got, err)
				}
				return
			}
			for _, r := range got {
				if r.RootID == roots["a"].ID && r.Charges.Effects != tc.want {
					t.Fatalf("charge sum: got %d, want %d", r.Charges.Effects, tc.want)
				}
			}
		})
	}
}

func overwriteRootReportCharge(t *testing.T, s *Store, op, raw string) {
	t.Helper()
	result, err := s.conn.ExecContext(t.Context(), "UPDATE composition_charges SET descriptor=? WHERE operation_id=?", []byte(raw), op)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		t.Fatal("charge mutation did not affect exactly one retained row", n, err)
	}
}

func TestRootCompletionsLaterChargeCannotMaskEarlierResultError(t *testing.T) {
	s, _, roots := rootReportStore(t)
	first, later := roots["a"], roots["b"]
	if first.ID > later.ID {
		first, later = later, first
	}
	firstOp := rootReportCommit(t, s, first.Run.Subject)
	laterOp := rootReportCommit(t, s, later.Run.Subject)
	firstAttempt, _, err := s.prepareExternal(t.Context(), firstOp)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.prepareExternal(t.Context(), laterOp); err != nil {
		t.Fatal(err)
	}
	if err := s.recordExternalReceipt(t.Context(), firstAttempt, receiptFor(firstAttempt.Request)); err != nil {
		t.Fatal(err)
	}
	overwriteRootReportCharge(t, s, laterOp, "{")
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE external_receipts SET descriptor='{}' WHERE intent_id=?", firstOp); err != nil {
		t.Fatal(err)
	}
	_, resultErr := externalOutcome(t.Context(), s.conn, firstOp)
	if resultErr == nil {
		t.Fatal("fixture requires an earlier invalid result")
	}
	got, err := readRootCompletions(t.Context(), s.conn)
	assertRootReportOriginal(t, s.conn, got, err)
	if err == nil || got != nil || err.Error() != resultErr.Error() {
		t.Fatalf("later malformed charge masked earlier result error: %#v, %v, want %v", got, err, resultErr)
	}
}

func TestRootCompletionsChargeAuthorizationFailure(t *testing.T) {
	s, _, _ := rootReportStore(t)
	if err := s.conn.Raw(func(driver any) error {
		driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(func(op int, table, column, _ string) int {
			if op == sqlite3.SQLITE_READ && table == "composition_charges" && column == "descriptor" {
				return sqlite3.SQLITE_DENY
			}
			return sqlite3.SQLITE_OK
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	defer s.conn.Raw(func(driver any) error {
		driver.(*sqlite3.SQLiteConn).RegisterAuthorizer(nil)
		return nil
	})
	q := &rootReportQueryer{queryer: s.conn}
	got, err := readRootCompletions(t.Context(), q)
	assertRootReportOriginal(t, s.conn, got, err)
	if err == nil || got != nil || len(q.chargeReads) != 1 {
		t.Fatalf("charge authorization failure: %#v, %v, charge reads %d", got, err, len(q.chargeReads))
	}
}

func TestRootCompletionsLaterMembershipCannotMaskEarlierResultError(t *testing.T) {
	s, _, roots := rootReportStore(t)
	first := roots["a"]
	if first.ID > roots["b"].ID {
		first = roots["b"]
	}
	op := rootReportCommit(t, s, first.Run.Subject)
	attempt, _, err := s.prepareExternal(t.Context(), op)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.recordExternalReceipt(t.Context(), attempt, receiptFor(attempt.Request)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE external_receipts SET descriptor='{}' WHERE intent_id=?", op); err != nil {
		t.Fatal(err)
	}
	rootReportCommit(t, s, "unscoped")
	q := &rootReportQueryer{queryer: s.conn}
	q.row = func(ctx context.Context, query string, args ...any) *sql.Row {
		if query == rootReportMembershipSQL && args[0] == "unscoped" {
			t.Error("membership reached after an earlier invalid result")
			return s.conn.QueryRowContext(ctx, "SELECT * FROM alan_missing_root_report_test_table")
		}
		return s.conn.QueryRowContext(ctx, query, args...)
	}
	got, err := readRootCompletions(t.Context(), q)
	assertRootReportOriginal(t, q, got, err)
	if err == nil || got != nil || !errors.Is(err, ErrCorrupt) {
		t.Fatalf("later membership masked earlier result error: %#v, %v", got, err)
	}
}

func TestRootCompletionsOrdinaryMembershipSemantics(t *testing.T) {
	s, _, roots := rootReportStore(t)
	a1 := rootReportCommit(t, s, "a")
	attempt, fresh, err := s.prepareExternal(t.Context(), a1)
	if err != nil || !fresh || !attempt.Allowed {
		t.Fatal(attempt, fresh, err)
	}
	a2 := rootReportCommit(t, s, "a")
	b1 := rootReportCommit(t, s, "b")
	if b, fresh, err := s.prepareExternal(t.Context(), b1); err != nil || !fresh || !b.Allowed {
		t.Fatal(b, fresh, err)
	}
	u1 := rootReportCommit(t, s, "unscoped")
	if a1 != identity("a", 1, 1, 0) || a2 != identity("a", 1, 2, 0) || b1 != identity("b", 1, 1, 0) || u1 != identity("unscoped", 1, 1, 0) {
		t.Fatal("original operation identities changed", a1, a2, b1, u1)
	}
	before := rootReportTables(t, s)
	q := &rootReportQueryer{queryer: s.conn}
	got := rootReportRead(t, s, q)
	want := rootReportWant(roots, map[string][]string{"a": {a1, a2}, "b": {b1}}, map[string][]string{"a": {a1}, "b": {b1}}, map[string]int64{"a": 1, "b": 1})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("complete report:\n got %#v\nwant %#v", got, want)
	}
	if after := rootReportTables(t, s); !reflect.DeepEqual(before, after) {
		t.Fatal("report changed durable rows")
	}
	if len(q.lookups) == 0 {
		t.Fatal("did not observe the original membership SQL")
	}
}

// This is the intended performance RED: semantic expectations are independent
// and already hold on baseline. Four genuine intents have three unique sources,
// one absent from membership; two roots previously repeat all four lookups.
func TestRootCompletionsMembershipQueriesOncePerRun(t *testing.T) {
	s, _, roots := rootReportStore(t)
	unresolved := map[string][]string{}
	unique := map[RunID]bool{}
	for step, subject := range []string{"a", "a", "b", "unscoped"} {
		operation := rootReportCommit(t, s, subject)
		if subject != "unscoped" {
			unresolved[subject] = append(unresolved[subject], operation)
		}
		unique[RunID{subject, 1}] = true
		before := rootReportTables(t, s)
		q := &rootReportQueryer{queryer: s.conn}
		got := rootReportRead(t, s, q)
		want := rootReportWant(roots, unresolved, nil, nil)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("prefix %d full report: got %#v, want %#v", step+1, got, want)
		}
		if after := rootReportTables(t, s); !reflect.DeepEqual(before, after) {
			t.Fatal("report changed durable rows")
		}
		seen := map[RunID]int{}
		for _, run := range q.lookups {
			seen[run]++
		}
		if len(q.lookups) != len(unique) {
			t.Errorf("prefix %d: membership queries=%d (%v), want one per full RunID=%d", step+1, len(q.lookups), seen, len(unique))
		}
		for run := range unique {
			if seen[run] != 1 {
				t.Errorf("prefix %d: source %+v queried %d times, want 1", step+1, run, seen[run])
			}
		}
	}
}

func TestRootCompletionsIndependentCallsAndStores(t *testing.T) {
	first, _, roots1 := rootReportStore(t)
	second, _, roots2 := rootReportStore(t)
	operation1 := rootReportCommit(t, first, "a")
	operation2 := rootReportCommit(t, second, "a")
	if first.Namespace() == second.Namespace() || roots1["a"].ID == roots2["a"].ID || operation1 != operation2 {
		t.Fatal("fixture requires coincident RunIDs/operations but distinct stores/root IDs")
	}
	for _, cut := range []struct {
		name  string
		store *Store
		roots map[string]ResourceRoot
	}{{"first", first, roots1}, {"second", second, roots2}, {"first_again", first, roots1}} {
		t.Run(cut.name, func(t *testing.T) {
			q := &rootReportQueryer{queryer: cut.store.conn}
			got := rootReportRead(t, cut.store, q)
			want := rootReportWant(cut.roots, map[string][]string{"a": {operation1}}, nil, nil)
			if !reflect.DeepEqual(got, want) || !slices.Contains(q.lookups, (RunID{"a", 1})) {
				t.Fatalf("fresh store/call reconstruction: got %#v, want %#v, lookups %v", got, want, q.lookups)
			}
		})
	}
}

// Synthetic helper-only descriptors permit generation two without claiming that
// a public generation-one scoped deployment can be reparented or retried.
func rootReportSyntheticIntents(t *testing.T, s *Store, intents []Intent) func(context.Context, string, ...any) (*sql.Rows, error) {
	t.Helper()
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		if !strings.HasPrefix(query, "SELECT id,subject,generation,sequence,ordinal,source_revision,descriptor FROM intents") {
			return s.conn.QueryContext(ctx, query, args...)
		}
		parts := make([]string, len(intents))
		values := []any{}
		for n, intent := range intents {
			raw, err := json.Marshal(intent)
			if err != nil {
				t.Fatal(err)
			}
			parts[n] = "SELECT ?,?,?,?,?,?,?"
			values = append(values, intent.ID, intent.Run.Subject, intent.Run.Generation, intent.Sequence, intent.Ordinal, int64(1), raw)
		}
		return s.conn.QueryContext(ctx, strings.Join(parts, " UNION ALL "), values...)
	}
}

func TestRootCompletionsSyntheticFullGenerationKey(t *testing.T) {
	s, _, roots := rootReportStore(t)
	intents := []Intent{{ID: "generation-one", Run: RunID{"same-subject", 1}, Kind: "invocation"}, {ID: "generation-two", Run: RunID{"same-subject", 2}, Kind: "invocation"}}
	q := &rootReportQueryer{queryer: s.conn, rows: rootReportSyntheticIntents(t, s, intents)}
	q.row = func(ctx context.Context, query string, args ...any) *sql.Row {
		if query == rootReportMembershipSQL {
			root := roots["a"].ID
			if args[1].(int64) == 2 {
				root = roots["b"].ID
			}
			return s.conn.QueryRowContext(ctx, "SELECT ?", root)
		}
		return s.conn.QueryRowContext(ctx, query, args...)
	}
	got := rootReportRead(t, s, q)
	want := rootReportWant(roots, nil, nil, nil)
	for n := range want {
		if want[n].RootID == roots["a"].ID {
			want[n].PendingInvocations = []string{"generation-one"}
		} else {
			want[n].PendingInvocations = []string{"generation-two"}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("full-generation membership: got %#v, want %#v", got, want)
	}
}

func TestRootCompletionsSyntheticAbsentThenPresentNextCall(t *testing.T) {
	s, _, roots := rootReportStore(t)
	intent := Intent{ID: "synthetic-unscoped", Run: RunID{"same-key", 1}, Kind: "invocation"}
	q := &rootReportQueryer{queryer: s.conn, rows: rootReportSyntheticIntents(t, s, []Intent{intent})}
	present := false
	q.row = func(ctx context.Context, query string, args ...any) *sql.Row {
		if query == rootReportMembershipSQL {
			if !present {
				return s.conn.QueryRowContext(ctx, "SELECT '' WHERE 0")
			}
			return s.conn.QueryRowContext(ctx, "SELECT ?", roots["a"].ID)
		}
		return s.conn.QueryRowContext(ctx, query, args...)
	}
	for _, next := range []bool{false, true} {
		present = next // Only between calls; this is not lawful unscoped reparenting.
		q.lookups = nil
		got := rootReportRead(t, s, q)
		want := rootReportWant(roots, nil, nil, nil)
		if present {
			for n := range want {
				if want[n].RootID == roots["a"].ID {
					want[n].PendingInvocations = []string{intent.ID}
				}
			}
		}
		if !reflect.DeepEqual(got, want) || len(q.lookups) == 0 {
			t.Fatalf("fresh lifetime present=%v: got %#v, want %#v, lookups %v", present, got, want, q.lookups)
		}
	}
}

func TestRootCompletionsErrorsAtRetainedBoundaries(t *testing.T) {
	for _, name := range []string{
		"entry_cancellation", "first_membership_cancellation", "first_membership_sql",
		"next_root_sql", "first_descriptor_decode", "first_descriptor_batch_decode", "next_descriptor_batch_decode",
	} {
		t.Run(name, func(t *testing.T) {
			s, _, roots := rootReportStore(t)
			rootReportCommit(t, s, "a")
			before := rootReportTables(t, s)
			q := &rootReportQueryer{queryer: s.conn}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if name == "entry_cancellation" {
				cancel()
			}
			ids := []string{roots["a"].ID, roots["b"].ID}
			slices.Sort(ids)
			injected := false
			q.rows = func(callCtx context.Context, query string, args ...any) (*sql.Rows, error) {
				if callCtx != ctx {
					t.Error("report replaced caller context")
				}
				if query == reportRootsSQL {
					switch name {
					case "next_root_sql", "first_descriptor_decode":
						// A failed bulk query must leave the retained scalar fault
						// at its original point in the traversal.
						return s.conn.QueryContext(callCtx, "SELECT * FROM alan_missing_root_report_test_table")
					case "first_descriptor_batch_decode", "next_descriptor_batch_decode":
						injected = true
						id := ids[0]
						if name == "next_descriptor_batch_decode" {
							id = ids[1]
						}
						return s.conn.QueryContext(callCtx, `SELECT id,
							CASE WHEN id=? THEN '{' ELSE descriptor END,
							NULL,NULL,0,0 FROM composition_roots`, id)
					}
				}
				return s.conn.QueryContext(callCtx, query, args...)
			}
			q.row = func(callCtx context.Context, query string, args ...any) *sql.Row {
				if callCtx != ctx {
					t.Error("report replaced caller context")
				}
				if query == rootReportMembershipSQL && (name == "first_membership_cancellation" || name == "first_membership_sql") {
					injected = true
					if name == "first_membership_cancellation" {
						cancel()
					} else {
						return s.conn.QueryRowContext(callCtx, "SELECT * FROM alan_missing_root_report_test_table")
					}
				}
				if query == "SELECT descriptor FROM composition_roots WHERE id=?" && name == "next_root_sql" && args[0] == ids[1] {
					injected = true
					return s.conn.QueryRowContext(callCtx, "SELECT * FROM alan_missing_root_report_test_table")
				}
				if query == "SELECT descriptor FROM composition_roots WHERE id=?" && name == "first_descriptor_decode" {
					injected = true
					return s.conn.QueryRowContext(callCtx, "SELECT '{'")
				}
				return s.conn.QueryRowContext(callCtx, query, args...)
			}
			s.mu.Lock()
			got, err := readRootCompletions(ctx, q)
			s.mu.Unlock()
			if err == nil || got != nil {
				t.Fatalf("error became successful/partial report: %#v, %v", got, err)
			}
			if strings.Contains(name, "cancellation") {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("context error changed: %v", err)
				}
			} else if strings.Contains(name, "descriptor") {
				var ignored ResourceRoot
				original := decode([]byte("{"), &ignored)
				if err.Error() != original.Error() {
					t.Fatalf("descriptor error changed: got %v, original %v", err, original)
				}
				if strings.HasPrefix(name, "first_") && len(q.lookups) != 0 {
					t.Fatal("membership prefetched before descriptor decode")
				}
			} else {
				var ignored any
				original := s.conn.QueryRowContext(t.Context(), "SELECT * FROM alan_missing_root_report_test_table").Scan(&ignored)
				if original == nil || err.Error() != original.Error() {
					t.Fatalf("SQL error changed: got %v, original %v", err, original)
				}
			}
			if name == "entry_cancellation" {
				if len(q.lookups) != 0 {
					t.Fatal("membership accessed before failed protocol gate")
				}
			} else if !injected {
				t.Fatal("did not reach intended retained boundary")
			}
			if (name == "first_membership_sql" || name == "first_membership_cancellation") && len(q.lookups) != 1 {
				t.Fatalf("first membership failure continued traversal: %v", q.lookups)
			}
			if (name == "next_root_sql" || name == "next_descriptor_batch_decode") && len(q.lookups) == 0 {
				t.Fatal("later retained SQL did not follow a membership lookup")
			}
			if after := rootReportTables(t, s); !reflect.DeepEqual(before, after) {
				t.Fatal("failed report changed durable rows")
			}
		})
	}
}

func TestRootCompletionsNonCompositionGate(t *testing.T) {
	s, _ := newStore(t, false)
	q := &rootReportQueryer{queryer: s.conn}
	got, err := readRootCompletions(t.Context(), q)
	if err != nil || got != nil || len(q.lookups) != 0 {
		t.Fatal(got, err, q.lookups)
	}
}

func TestRootCompletionsFreshDynamicMembership(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	if got := rootReportRead(t, s, s.conn); len(got) != 1 || got[0].Charges.Invocations != 0 || got[0].Pending != "members" {
		t.Fatal(got)
	}
	admission := invokeOnce(t, s)
	if !admission.Accepted || len(admission.Members) != 2 {
		t.Fatal(admission)
	}
	beforeRun := rootReportRead(t, s, s.conn)
	if len(beforeRun) != 1 || beforeRun[0].Charges.Invocations != 1 || beforeRun[0].Charges.CreatedRuns != 2 || !reflect.DeepEqual(beforeRun[0].PendingInvocations, []string{admission.Invocation.OperationID}) {
		t.Fatal(beforeRun)
	}
	for _, member := range admission.Members {
		root, err := compositionMembershipRoot(t.Context(), s.conn, member)
		if err != nil || root != beforeRun[0].RootID {
			t.Fatal("genuinely admitted child membership", member, root, err)
		}
	}
	run, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	q := &rootReportQueryer{queryer: s.conn}
	after := rootReportRead(t, s, q)
	if !reflect.DeepEqual(after, run.Roots) || len(after) != 1 || after[0].Outcome == nil || after[0].Outcome.Status != "succeeded" || after[0].LocalStatus != "succeeded" || after[0].Charges.Invocations != 2 || after[0].Charges.CreatedRuns != 4 || len(after[0].PendingInvocations) != 0 || len(after[0].UnresolvedExternal) != 0 || len(after[0].UnknownExternal) != 0 {
		t.Fatal(after, run.Roots)
	}
	intents, err := readIntents(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	childSource := false
	for _, intent := range intents {
		if slices.Contains(admission.Members, intent.Run) {
			childSource = true
			if !slices.Contains(q.lookups, intent.Run) {
				t.Fatal("later report missed newly admitted child's source", intent.Run)
			}
		}
	}
	if !childSource {
		t.Fatal("ordinary child execution emitted no retained intent")
	}
}

func TestRootCompletionsFreshLateReceiptSettlementAndConflict(t *testing.T) {
	s, path, roots := rootReportStore(t)
	id := rootReportCommit(t, s, "a")
	attempt, fresh, err := s.prepareExternal(t.Context(), id)
	if err != nil || !fresh || !attempt.Allowed {
		t.Fatal(attempt, fresh, err)
	}
	charges := map[string]int64{"a": 1}
	unknown := rootReportWant(roots, map[string][]string{"a": {id}}, map[string][]string{"a": {id}}, charges)
	if got := rootReportRead(t, s, s.conn); !reflect.DeepEqual(got, unknown) {
		t.Fatal(got, unknown)
	}
	view := func() *ExecutionEffect {
		t.Helper()
		snapshot, err := InspectExecution(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		effect, ok := snapshot.Effect(id)
		if !ok {
			t.Fatal("missing original effect")
		}
		return effect
	}
	if e := view(); e.State != "reserved_unknown" || e.Consequence != "unresolved" {
		t.Fatal(e)
	}
	receipt := receiptFor(attempt.Request)
	if err := s.recordExternalReceipt(t.Context(), attempt, receipt); err != nil {
		t.Fatal(err)
	}
	known := rootReportWant(roots, nil, nil, charges)
	if got := rootReportRead(t, s, s.conn); !reflect.DeepEqual(got, known) {
		t.Fatal("late result invisible", got, known)
	}
	if e := view(); e.State != "result_unsettled" || e.Settlement != nil {
		t.Fatal(e)
	}
	outcome, err := s.externalOutcome(t.Context(), id)
	if err != nil || outcome.Receipt == nil || !reflect.DeepEqual(*outcome.Receipt, receipt) {
		t.Fatal(outcome, err)
	}
	if err := s.settleExternal(t.Context(), outcome); err != nil {
		t.Fatal(err)
	}
	if got := rootReportRead(t, s, s.conn); !reflect.DeepEqual(got, known) {
		t.Fatal("settlement changed membership or debit", got, known)
	}
	if e := view(); e.State != "settled" || e.Settlement == nil {
		t.Fatal(e)
	}
	loser := receipt
	loser.Total++
	if err := s.recordExternalReceipt(t.Context(), attempt, loser); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if got := rootReportRead(t, s, s.conn); !reflect.DeepEqual(got, known) {
		t.Fatal("root known-result meaning changed after conflict", got, known)
	}
	if e := view(); e.Consequence != "conflicting_receipts" || e.ReceiptEvidenceState != "conflict" || len(e.AdditionalReceipts) != 1 || e.Settlement == nil {
		t.Fatal(e)
	}
	retained, err := s.externalOutcome(t.Context(), id)
	if err != nil || !reflect.DeepEqual(retained, outcome) || count(t, s, "settlements") != 1 || count(t, s, "receipt_conflicts") != 1 {
		t.Fatal("first result/debit/settlement replaced", retained, outcome, err)
	}
}
