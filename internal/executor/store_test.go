package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(t *testing.T, receive bool) Provision {
	t.Helper()
	first := Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "mutation", Target: Lit(Text("endpoint")), Generation: Lit(Int(1)), Obligation: Lit(Text("ob")), Payload: map[string]Operand{"amount": Lit(Int(700))}}, Updates: map[string]Operand{"spent": Lit(Int(1))}}
	if receive {
		first = Instruction{Op: "receive", Match: &Match{Kind: "input", Provenance: "trusted-fixture", Source: Lit(Text("sender")), Generation: Lit(Int(1)), Obligation: Lit(Text("ob")), Operation: Lit(Text(""))}, Bind: map[string]string{"number": "answer"}}
	}
	p := Program{Semantics: SemanticVersion, Code: []Instruction{first, {Op: "end", Outcome: "succeeded"}}}
	b, id, err := EncodeProgram(p)
	if err != nil {
		t.Fatal(err)
	}
	return Provision{Programs: []Artifact{{ID: id, Semantics: SemanticVersion, Bytes: b}}, Subjects: []Subject{{ID: "worker", Generation: 42, Owner: "owner", Revision: 7}}, Runs: []InitialRun{{ID: RunID{"worker", 42}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{"spent": Int(0)}}}}}
}
func newStore(t *testing.T, receive bool) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "executor.sqlite")
	s, err := Create(t.Context(), path, fixture(t, receive), DefaultLimits())
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
func state(t *testing.T, s *Store) RunState {
	t.Helper()
	r, err := s.Snapshot(t.Context(), RunID{"worker", 42})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func count(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCommitReopenReplayRetainsIntentAndProjection(t *testing.T) {
	s, path := newStore(t, false)
	before := state(t, s)
	after, err := s.Commit(t.Context(), before.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if after.Sequence != 1 || after.Projection.Data["spent"] != Int(1) || after.Projection.Data["pending"].Text == "" || count(t, s, "intents") != 1 || count(t, s, "history") != 1 {
		t.Fatalf("commit: %+v", after)
	}
	report, err := Replay(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if report.HistoryCount != 1 || report.IntentCount != 1 || report.SQLiteVersion == "" {
		t.Fatalf("replay: %+v", report)
	}
	if got := state(t, s); !reflect.DeepEqual(got, after) {
		t.Fatal("read-only replay changed live projection")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	resumed := state(t, reopened)
	if resumed.Epoch != after.Epoch+1 || resumed.ID != after.ID || resumed.Sequence != 1 || !reflect.DeepEqual(resumed.Projection, after.Projection) {
		t.Fatalf("reopen reset state: %+v", resumed)
	}
	if _, err = reopened.Commit(t.Context(), after.Authority(), ""); !errors.Is(err, ErrStale) {
		t.Fatalf("old epoch: %v", err)
	}
	terminal, err := reopened.Commit(t.Context(), resumed.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Projection.Status != "succeeded" || count(t, reopened, "terminal_obligations") != 1 {
		t.Fatal("terminal outcome not retained")
	}
	if _, err = reopened.Commit(t.Context(), terminal.Authority(), ""); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed: %v", err)
	}
}

func TestObservationSelectionAtomicAndImmutable(t *testing.T) {
	s, path := newStore(t, true)
	o := Observation{ID: "o1", Run: RunID{"worker", 42}, Kind: "input", Provenance: "trusted-fixture", Source: "sender", SourceGeneration: 1, Obligation: "ob", Payload: map[string]Value{"number": Int(800)}}
	if err := s.recordObservation(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	if err := s.recordObservation(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	conflict := o
	conflict.Kind = "other"
	if err := s.recordObservation(t.Context(), conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("identity conflict: %v", err)
	}
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_history BEFORE INSERT ON history BEGIN SELECT RAISE(ABORT,'test rollback'); END`); err != nil {
		t.Fatal(err)
	}
	before := state(t, s)
	if _, err := s.Commit(t.Context(), before.Authority(), o.ID); err == nil {
		t.Fatal("expected transaction failure")
	}
	if !reflect.DeepEqual(state(t, s), before) || count(t, s, "history") != 0 {
		t.Fatal("partial commit")
	}
	var consumed sql.NullInt64
	if err := s.conn.QueryRowContext(t.Context(), `SELECT consumed_sequence FROM observations WHERE id='o1'`).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed.Valid {
		t.Fatal("observation consumed despite rollback")
	}
	if _, err := s.conn.ExecContext(t.Context(), "DROP TRIGGER fail_history"); err != nil {
		t.Fatal(err)
	}
	after, err := s.Commit(t.Context(), before.Authority(), o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Projection.Data["answer"] != Int(800) {
		t.Fatal(after)
	}
	if _, err := s.Commit(t.Context(), before.Authority(), o.ID); !errors.Is(err, ErrStale) {
		t.Fatalf("duplicate stale selection: %v", err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestIntentInsertFailureRollsBackAllExecutionWrites(t *testing.T) {
	s, _ := newStore(t, false)
	before := state(t, s)
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_intent BEFORE INSERT ON intents BEGIN SELECT RAISE(ABORT,'test rollback'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), before.Authority(), ""); err == nil {
		t.Fatal("expected intent failure")
	}
	if !reflect.DeepEqual(before, state(t, s)) || count(t, s, "history") != 0 || count(t, s, "intents") != 0 {
		t.Fatal("partial execution persisted")
	}
}

func TestStaleAuthorityRejected(t *testing.T) {
	s, _ := newStore(t, false)
	a := state(t, s).Authority()
	for _, change := range []func(*Authority){func(a *Authority) { a.Revision++ }, func(a *Authority) { a.Epoch++ }, func(a *Authority) { a.Sequence++ }, func(a *Authority) { a.Run.Generation-- }} {
		bad := a
		change(&bad)
		if _, err := s.Commit(t.Context(), bad, ""); !errors.Is(err, ErrStale) {
			t.Fatalf("stale %+v: %v", bad, err)
		}
	}
	if count(t, s, "history") != 0 {
		t.Fatal("stale authority wrote history")
	}
}

func TestReopenRejectsCorruptionAndVersions(t *testing.T) {
	for _, query := range []string{
		`UPDATE metadata SET schema_version=99`,
		`UPDATE metadata SET semantics='unknown'`,
		`UPDATE programs SET bytes='{}'`,
		`UPDATE programs SET semantics='unknown'`,
		`UPDATE runs SET projection='{}'`,
		`UPDATE history SET successor='{}'`,
		`UPDATE intents SET descriptor='{}'`,
		`DELETE FROM intents`,
	} {
		t.Run(query, func(t *testing.T) {
			s, path := newStore(t, false)
			if _, err := s.Commit(t.Context(), state(t, s).Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(t.Context(), path, DefaultLimits()); err == nil {
				reopened.Close()
				t.Fatal("corruption accepted")
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("replay accepted corruption")
			}
		})
	}
}

func TestExclusiveOwnerAndAliases(t *testing.T) {
	if path := os.Getenv("ALAN_LOCK_CHILD"); path != "" {
		s, err := Open(context.Background(), path, DefaultLimits())
		if s != nil {
			s.Close()
		}
		if !errors.Is(err, ErrLocked) {
			t.Fatalf("child acquired store: %v", err)
		}
		return
	}
	s, path := newStore(t, false)
	before := state(t, s)
	child := exec.Command(os.Args[0], "-test.run=^TestExclusiveOwnerAndAliases$")
	child.Env = append(os.Environ(), "ALAN_LOCK_CHILD="+path)
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("child: %s %v", out, err)
	}
	alias := filepath.Join(filepath.Dir(path), "alias.sqlite")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(t.Context(), alias, DefaultLimits()); !errors.Is(err, ErrLocked) {
		if other != nil {
			other.Close()
		}
		t.Fatalf("symlink alias: %v", err)
	}
	hard := filepath.Join(filepath.Dir(path), "hard.sqlite")
	if err := os.Link(path, hard); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(t.Context(), hard, DefaultLimits()); err == nil {
		other.Close()
		t.Fatal("hard alias accepted")
	}
	if !reflect.DeepEqual(before, state(t, s)) {
		t.Fatal("second opener changed epoch")
	}
}

func TestReadOnlyReplaySettingsAndNoProvisioning(t *testing.T) {
	s, path := newStore(t, false)
	version, source, err := verifyConnection(t.Context(), s.conn, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SQLite version=%s source=%s", version, source)
	inspector, err := connect(t.Context(), path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	if _, _, err = verifyConnection(t.Context(), inspector.conn, true); err != nil {
		t.Fatal(err)
	}
	if _, err = inspector.conn.ExecContext(t.Context(), "UPDATE runs SET epoch=99"); err == nil {
		t.Fatal("inspection connection permitted mutation")
	}
	missing := filepath.Join(t.TempDir(), "missing.sqlite")
	if _, err = Replay(t.Context(), missing, DefaultLimits()); err == nil {
		t.Fatal("replay created store")
	}
	if _, err = os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing store changed: %v", err)
	}
	if other, err := Create(t.Context(), path, fixture(t, false), DefaultLimits()); err == nil {
		other.Close()
		t.Fatal("provisioned existing store")
	}
}

func TestReplayRejectsChangedSelectedObservation(t *testing.T) {
	s, path := newStore(t, true)
	o := Observation{ID: "selected", Run: RunID{"worker", 42}, Kind: "input", Provenance: "trusted-fixture", Source: "sender", SourceGeneration: 1, Obligation: "ob", Payload: map[string]Value{"number": Int(8)}}
	if err := s.recordObservation(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	after, err := s.Commit(t.Context(), state(t, s).Authority(), o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), after.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), `UPDATE observations SET descriptor=replace(CAST(descriptor AS TEXT),'"Number":8','"Number":9')`); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("selected observation changed without detection")
	}
}

func TestArtifactsRetainedExactlyAndConflictsRejected(t *testing.T) {
	provision := fixture(t, false)
	provision.Programs = append(provision.Programs, provision.Programs[0])
	path := filepath.Join(t.TempDir(), "artifacts.sqlite")
	s, err := Create(t.Context(), path, provision, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var retained []byte
	if err = s.conn.QueryRowContext(t.Context(), "SELECT bytes FROM programs WHERE id=?", provision.Programs[0].ID).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if string(retained) != string(provision.Programs[0].Bytes) || count(t, s, "programs") != 1 {
		t.Fatal("artifact bytes or content identity changed")
	}
	provision.Programs[1].Bytes = []byte(`{"Semantics":"changed"}`)
	if other, err := Create(t.Context(), filepath.Join(t.TempDir(), "conflict.sqlite"), provision, DefaultLimits()); err == nil {
		other.Close()
		t.Fatal("conflicting artifact identity accepted")
	}
}

func TestReplayRejectsTerminalRecordCorruption(t *testing.T) {
	for _, query := range []string{`DELETE FROM terminal_obligations`, `UPDATE terminal_obligations SET outcome='{}'`, `UPDATE terminal_obligations SET destinations='["ungranted"]'`, `UPDATE terminal_obligations SET delivery_status='{"ungranted":true}'`} {
		t.Run(query, func(t *testing.T) {
			s, path := newStore(t, false)
			first, err := s.Commit(t.Context(), state(t, s).Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), first.Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if _, err = s.conn.ExecContext(t.Context(), query); err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("terminal corruption accepted")
			}
		})
	}
}

func TestCorruptionRejectedBeforeEpochAdvance(t *testing.T) {
	s, path := newStore(t, false)
	before := state(t, s)
	if _, err := s.conn.ExecContext(t.Context(), `UPDATE runs SET projection='{}'`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(t.Context(), path, DefaultLimits()); err == nil {
		other.Close()
		t.Fatal("corruption accepted")
	}
	inspector, err := connect(t.Context(), path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	var epoch int64
	if err = inspector.conn.QueryRowContext(t.Context(), "SELECT epoch FROM runs").Scan(&epoch); err != nil {
		t.Fatal(err)
	}
	if epoch != before.Epoch {
		t.Fatal("invalid ledger advanced epoch")
	}
}

func TestNonReceiveCannotConsumeSelectedObservation(t *testing.T) {
	s, _ := newStore(t, false)
	observation := Observation{ID: "unrelated", Run: RunID{"worker", 42}, Kind: "input", Provenance: "trusted-fixture", Source: "sender", SourceGeneration: 1, Payload: map[string]Value{}}
	if err := s.recordObservation(t.Context(), observation); err != nil {
		t.Fatal(err)
	}
	before := state(t, s)
	if _, err := s.Commit(t.Context(), before.Authority(), observation.ID); !errors.Is(err, ErrIneligible) {
		t.Fatalf("emit accepted selected observation: %v", err)
	}
	var consumed sql.NullInt64
	if err := s.conn.QueryRowContext(t.Context(), "SELECT consumed_sequence FROM observations WHERE id=?", observation.ID).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed.Valid || !reflect.DeepEqual(before, state(t, s)) || count(t, s, "history") != 0 || count(t, s, "intents") != 0 {
		t.Fatal("non-receive consumed observation or committed execution")
	}
}

func TestGeneratedOutputLimitPrecedesExecutionWrites(t *testing.T) {
	for _, chars := range []int{500000, 100000} {
		t.Run(fmt.Sprint(chars), func(t *testing.T) {
			program, initial := amplifiedOutputFixture(chars)
			encoded, id, err := EncodeProgram(program)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "output-limit.sqlite")
			run := RunID{"worker", 1}
			provision := Provision{Programs: []Artifact{{ID: id, Semantics: SemanticVersion, Bytes: encoded}}, Subjects: []Subject{{ID: run.Subject, Generation: run.Generation, Owner: "owner", Revision: 1}}, Runs: []InitialRun{{ID: run, ProgramID: id, Projection: initial}}}
			store, err := Create(t.Context(), path, provision, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			observation := Observation{ID: "waiting", Run: run, Kind: "input", Provenance: "trusted-fixture", Source: "sender", SourceGeneration: 1, Payload: map[string]Value{}}
			if err = store.recordObservation(t.Context(), observation); err != nil {
				t.Fatal(err)
			}
			before, err := store.Snapshot(t.Context(), run)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.Commit(t.Context(), before.Authority(), "")
			if chars == 500000 {
				if !errors.Is(err, ErrLimit) {
					t.Fatalf("amplified intent committed: %v", err)
				}
				after, snapshotErr := store.Snapshot(t.Context(), run)
				if snapshotErr != nil {
					t.Fatal(snapshotErr)
				}
				if !reflect.DeepEqual(before, after) || count(t, store, "history") != 0 || count(t, store, "intents") != 0 || count(t, store, "terminal_obligations") != 0 {
					t.Fatal("output rejection partially mutated ledger")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if count(t, store, "history") != 1 || count(t, store, "intents") != 1 {
					t.Fatal("below-limit output did not commit")
				}
				var descriptorBytes int
				if err = store.conn.QueryRowContext(t.Context(), "SELECT length(descriptor) FROM intents").Scan(&descriptorBytes); err != nil {
					t.Fatal(err)
				}
				if descriptorBytes != 800625 {
					t.Fatalf("unexpected descriptor bytes: %d", descriptorBytes)
				}
			}
			var consumed sql.NullInt64
			if err = store.conn.QueryRowContext(t.Context(), "SELECT consumed_sequence FROM observations WHERE id='waiting'").Scan(&consumed); err != nil {
				t.Fatal(err)
			}
			if consumed.Valid {
				t.Fatal("output step consumed unrelated waiting observation")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReplayUsesConfiguredGeneratedOutputLimit(t *testing.T) {
	program, initial := amplifiedOutputFixture(500000)
	encoded, id, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits()
	limits.MaxOutputBytes = 5 << 20
	run := RunID{"worker", 1}
	path := filepath.Join(t.TempDir(), "large-output.sqlite")
	store, err := Create(t.Context(), path, Provision{Programs: []Artifact{{ID: id, Semantics: SemanticVersion, Bytes: encoded}}, Subjects: []Subject{{ID: run.Subject, Generation: 1, Owner: "owner", Revision: 1}}, Runs: []InitialRun{{ID: run, ProgramID: id, Projection: initial}}}, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	before, err := store.Snapshot(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Commit(t.Context(), before.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, limits); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("replay ignored output limit")
	}
}

func TestGeneratedOutputLimitDoesNotConsumeReceive(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxOutputBytes = 1 // Even the empty intent array requires two bytes.
	store, err := Create(t.Context(), filepath.Join(t.TempDir(), "receive-output-limit.sqlite"), fixture(t, true), limits)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	observation := Observation{ID: "selected", Run: RunID{"worker", 42}, Kind: "input", Provenance: "trusted-fixture", Source: "sender", SourceGeneration: 1, Obligation: "ob", Payload: map[string]Value{"number": Int(800)}}
	if err = store.recordObservation(t.Context(), observation); err != nil {
		t.Fatal(err)
	}
	before := state(t, store)
	if _, err = store.Commit(t.Context(), before.Authority(), observation.ID); !errors.Is(err, ErrLimit) {
		t.Fatalf("receive output limit: %v", err)
	}
	var consumed sql.NullInt64
	if err = store.conn.QueryRowContext(t.Context(), "SELECT consumed_sequence FROM observations WHERE id=?", observation.ID).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed.Valid || !reflect.DeepEqual(before, state(t, store)) || count(t, store, "history") != 0 || count(t, store, "intents") != 0 {
		t.Fatal("output rejection consumed observation or mutated execution")
	}
}
