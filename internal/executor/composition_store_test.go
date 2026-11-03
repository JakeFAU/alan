package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func compositionFixture(t *testing.T) (Provision, compositionProvision) {
	t.Helper()
	p := Provision{}
	send := func(n int64) Instruction {
		return Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("caller")), Generation: Lit(Int(1)), Obligation: Lit(Text("reply")), Payload: map[string]Operand{"value": Lit(Int(n))}}}
	}
	addProgram(t, &p, "A", []Instruction{send(1), send(2), {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "B", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	return p, compositionProvision{Roots: []compositionRoot{{Run: RunID{"A", 1}, Members: []RunID{{"A", 1}, {"B", 1}}, Limits: compositionLimits{MaxInvocations: 4, MaxCreatedRuns: 8, MaxDepth: 3, MaxEffects: 8, MaxInFlight: 1}}}}
}
func compositionTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	p, c := compositionFixture(t)
	path := filepath.Join(t.TempDir(), "composition.sqlite")
	s, err := createCompositionStore(t.Context(), path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, path
}
func TestCompositionOrderedFailurePrefixes(t *testing.T) {
	type evidence struct {
		Schedule        []string
		History         any
		Intents         any
		Runs, Terminals any
		Facts           []compositionFact
		Frozen          []string
	}
	var all []evidence
	for _, schedule := range [][]string{{"A", "B", "A"}, {"A", "A", "B"}} {
		s, _ := compositionTestStore(t)
		for _, subject := range schedule {
			if _, err := s.Commit(t.Context(), current(t, s, subject).Authority(), ""); err != nil {
				t.Fatal(err)
			}
		}
		facts, err := readCompositionFacts(t.Context(), s.conn)
		if err != nil {
			t.Fatal(err)
		}
		var prefix, frozen []string
		for _, f := range facts {
			if f.Kind != compositionExecutionFact {
				t.Fatalf("unexpected fact %+v", f)
			}
			if f.Run.Subject == "A" {
				prefix = append(prefix, identity("A", int64(1), f.Sequence, 0))
			}
			if f.Run.Subject == "B" {
				frozen = append([]string(nil), prefix...)
			}
		}
		history, err := inspectTable(t.Context(), s.conn, "history")
		if err != nil {
			t.Fatal(err)
		}
		// Per-run comparison deliberately discards cross-run SQLite insertion order.
		byRun := map[string][]map[string]any{}
		for _, row := range history {
			key := row["subject"].(string)
			byRun[key] = append(byRun[key], row)
		}
		intents, err := readIntents(t.Context(), s.conn)
		if err != nil {
			t.Fatal(err)
		}
		visibleIntents := map[string]Intent{}
		for id, v := range intents {
			visibleIntents[id] = v.Intent
		}
		runs, err := inspectTable(t.Context(), s.conn, "runs")
		if err != nil {
			t.Fatal(err)
		}
		terminals, err := inspectTable(t.Context(), s.conn, "terminal_obligations")
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, evidence{schedule, byRun, visibleIntents, runs, terminals, facts, frozen})
		if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(all[0].History, all[1].History) || !reflect.DeepEqual(all[0].Intents, all[1].Intents) || !reflect.DeepEqual(all[0].Runs, all[1].Runs) || !reflect.DeepEqual(all[0].Terminals, all[1].Terminals) {
		t.Fatal("counterexample requires identical per-run histories and intents")
	}
	for i, want := range [][]string{{"A", "B", "A"}, {"A", "A", "B"}} {
		got := []string{}
		for _, f := range all[i].Facts {
			got = append(got, f.Run.Subject)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("logical fact ordering: %v != %v", got, want)
		}
	}
	if len(all[0].Frozen) != 1 || len(all[1].Frozen) != 2 {
		t.Fatalf("failure-prefix sets: %+v", all)
	}
	raw, _ := json.Marshal(all)
	t.Logf("executed counterexample: %s", raw)
}
func TestCompositionExecutionFactAtomicityAndGate(t *testing.T) {
	s, path := compositionTestStore(t)
	before := current(t, s, "A")
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TRIGGER reject_fact BEFORE INSERT ON composition_facts BEGIN SELECT RAISE(ABORT,'fact rollback'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), before.Authority(), ""); err == nil {
		t.Fatal("expected fact insertion failure")
	}
	if !reflect.DeepEqual(before, current(t, s, "A")) || count(t, s, "history") != 0 || count(t, s, "intents") != 0 || count(t, s, "composition_facts") != 0 {
		t.Fatal("fact failure left partial execution")
	}
	if _, err := s.conn.ExecContext(t.Context(), "DROP TRIGGER reject_fact"); err != nil {
		t.Fatal(err)
	}
	after, err := s.Commit(t.Context(), before.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), before.Authority(), ""); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	if count(t, s, "composition_facts") != 1 {
		t.Fatal("stale operation duplicated fact")
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatalf("public Replay gate: %v", err)
	}
	if _, err := Inspect(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatalf("public Inspect gate: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		if opened != nil {
			opened.Close()
		}
		t.Fatalf("public Open gate: %v", err)
	}
	reopened, err := openCompositionStore(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got := current(t, reopened, "A")
	if got.Epoch != after.Epoch+1 || got.Sequence != after.Sequence || !reflect.DeepEqual(got.Projection, after.Projection) {
		t.Fatalf("internal reopen lost state: %+v", got)
	}
	if count(t, reopened, "composition_facts") != 1 {
		t.Fatal("activation invented a logical fact")
	}
}
func TestCompositionFactsRejectCorruption(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"missing fact", "DELETE FROM composition_facts"},
		{"changed digest", "UPDATE composition_facts SET descriptor=replace(CAST(descriptor AS TEXT),'execution','lifecycle')"},
		{"missing member", "DELETE FROM composition_static_members WHERE subject='B'"},
		{"missing root", "DELETE FROM composition_roots"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := compositionTestStore(t)
			if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), tc.sql); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=ON"); err != nil {
				t.Fatal(err)
			}
			if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
				t.Fatal("corrupt foundation accepted")
			} else {
				t.Logf("foundation rejection: %v", err)
			}
		})
	}
}

func TestCompositionProfileIsImmutableAndLegacySchemaUntouched(t *testing.T) {
	p, c := compositionFixture(t)
	path := filepath.Join(t.TempDir(), "profile.sqlite")
	s, err := createCompositionStore(t.Context(), path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	original := s.composition.raw
	c.Roots[0].Members[0] = RunID{"mutated", 9}
	c.Roots[0].Limits.MaxEffects = 9999
	decoded, err := s.composition.descriptor()
	if err != nil {
		t.Fatal(err)
	}
	decoded.Roots[0].Members[0] = RunID{"mutated-again", 9}
	if original != s.composition.raw {
		t.Fatal("caller mutated retained profile")
	}
	if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	legacy, err := Create(t.Context(), filepath.Join(t.TempDir(), "legacy.sqlite"), p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer legacy.Close()
	var n int
	if err := legacy.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM sqlite_master WHERE type='table' AND name LIKE 'composition_%'").Scan(&n); err != nil || n != 0 {
		t.Fatal("legacy schema acquired composition tables", n, err)
	}
	if _, err := legacy.Commit(t.Context(), current(t, legacy, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := validateLedger(t.Context(), legacy.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestCompositionProfileRejectsInconsistentMembership(t *testing.T) {
	for _, name := range []string{"missing root", "duplicate member", "wrong generation", "missing member", "negative ceiling", "zero physical capacity", "future root version", "forged root identity"} {
		t.Run(name, func(t *testing.T) {
			p, c := compositionFixture(t)
			switch name {
			case "missing root":
				c.Roots[0].Members = c.Roots[0].Members[1:]
			case "duplicate member":
				c.Roots[0].Members = append(c.Roots[0].Members, c.Roots[0].Members[0])
			case "wrong generation":
				c.Roots[0].Members[0].Generation = 2
			case "missing member":
				c.Roots[0].Members = append(c.Roots[0].Members, RunID{"not-provisioned", 1})
			case "negative ceiling":
				c.Roots[0].Limits.MaxEffects = -1
			case "zero physical capacity":
				c.Roots[0].Limits.MaxInFlight = 0
			case "future root version":
				c.Roots[0].Version = "future"
			case "forged root identity":
				c.Roots[0].ID = "forged"
			}
			s, err := createCompositionStore(t.Context(), filepath.Join(t.TempDir(), "invalid.sqlite"), p, c, DefaultLimits())
			if s != nil {
				s.Close()
			}
			if err == nil {
				t.Fatal("invalid root membership admitted")
			}
		})
	}
}

func TestCompositionReadOnlyFoundationValidation(t *testing.T) {
	s, path := compositionTestStore(t)
	if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	before := current(t, s, "A")
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := connect(t.Context(), path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	tx, err := reader.conn.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err := verifyConnection(t.Context(), tx, true); err != nil {
		t.Fatal(err)
	}
	report, err := validateCompositionStore(t.Context(), tx, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if report.HistoryCount != 1 || report.IntentCount != 1 {
		t.Fatalf("wrong foundation report: %+v", report)
	}
	if _, err := tx.ExecContext(t.Context(), "UPDATE runs SET epoch=epoch+1"); err == nil {
		t.Fatal("read-only validator connection permits writes")
	}
	afterFacts, err := readCompositionFacts(t.Context(), tx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, current(t, s, "A")) || !reflect.DeepEqual(facts, afterFacts) {
		t.Fatal("foundation read changed durable state")
	}
}

func TestCompositionProfileRejectsInvalidEncodingBeforeJSONCopy(t *testing.T) {
	p, c := compositionFixture(t)
	// The malformed spelling would otherwise normalize to an existing valid name.
	p.Subjects[0].ID = "\ufffd"
	p.Runs[0].ID.Subject = "\ufffd"
	c.Roots[0].Run.Subject = string([]byte{0xff})
	c.Roots[0].Members[0].Subject = string([]byte{0xff})
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/invalid-encoding.sqlite", p, c, DefaultLimits())
	if s != nil {
		s.Close()
	}
	if err == nil {
		t.Fatal("JSON replacement changed root authority identity")
	}
}
