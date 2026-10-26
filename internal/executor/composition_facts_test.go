package executor

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCompositionLifecycleFactAtomicityAndDeduplication(t *testing.T) {
	p := lifecycleFixture(t, []string{"CancelRun", "CancelRun"}, false)
	profile := compositionProvision{Roots: []compositionRoot{{Run: RunID{"worker", 1}, Members: []RunID{{"worker", 1}}, Limits: compositionLimits{MaxInFlight: 1}}, {Run: RunID{"requester", 1}, Members: []RunID{{"requester", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
	path := filepath.Join(t.TempDir(), "lifecycle.sqlite")
	s, err := createCompositionStore(t.Context(), path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	request := committedRequest(t, s, "requester")
	auth, err := s.AuthorizeIntent(t.Context(), request.Intent.ID)
	if err != nil || !auth.Allowed {
		t.Fatal(auth, err)
	}
	before := current(t, s, "worker")
	if _, err = s.conn.ExecContext(t.Context(), `CREATE TRIGGER fail_lifecycle_fact BEFORE INSERT ON composition_facts WHEN NEW.kind='lifecycle' BEGIN SELECT RAISE(ABORT,'lifecycle fact rollback'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DecideLifecycle(t.Context(), request); err == nil {
		t.Fatal("missing lifecycle fact failure")
	}
	if !reflect.DeepEqual(before, current(t, s, "worker")) || count(t, s, "terminal_obligations") != 0 || count(t, s, "lifecycle_results") != 0 || count(t, s, "composition_facts") != 1 {
		t.Fatal("partial lifecycle transaction")
	}
	if _, err = s.conn.ExecContext(t.Context(), "DROP TRIGGER fail_lifecycle_fact"); err != nil {
		t.Fatal(err)
	}
	result, err := s.DecideLifecycle(t.Context(), request)
	if err != nil || !result.Accepted {
		t.Fatal(result, err)
	}
	again, err := s.DecideLifecycle(t.Context(), request)
	if err != nil || !reflect.DeepEqual(again, result) || count(t, s, "composition_facts") != 2 {
		t.Fatal("duplicate lifecycle added a fact", err)
	}
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts[1].RootIDs) != 2 || facts[1].Kind != compositionLifecycleFact {
		t.Fatalf("cross-root evidence: %+v", facts)
	}
	noChange := committedRequest(t, s, "requester")
	unchanged := authorizedDecision(t, s, noChange)
	if !unchanged.Accepted || unchanged.Reason != "already_terminal" {
		t.Fatal(unchanged)
	}
	n := count(t, s, "composition_facts")
	if _, err = s.DecideLifecycle(t.Context(), noChange); err != nil || count(t, s, "composition_facts") != n {
		t.Fatal("retransmitted no-op duplicated fact", err)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestCompositionFactAppendRejectsUnlinkedAndConflictingFacts(t *testing.T) {
	s, _ := compositionTestStore(t)
	if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOwnedTx(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	f, err := compositionExecutionEvidence(t.Context(), tx, RunID{"A", 1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	order, err := appendCompositionFact(t.Context(), tx, f)
	if err != nil || order != 1 {
		t.Fatal(order, err)
	}
	bad := f
	bad.RecordDigest = compositionIdentity("wrong", 1)
	if _, err := appendCompositionFact(t.Context(), tx, bad); err == nil {
		t.Fatal("invented source digest accepted")
	}
	bad = f
	bad.Kind = compositionOutcomeFact
	if _, err := appendCompositionFact(t.Context(), tx, bad); !errors.Is(err, ErrVersion) {
		t.Fatal("unfinished protocol kind enabled", err)
	}
	if _, err := appendCompositionFact(t.Context(), tx, compositionFact{}); err == nil {
		t.Fatal("empty arbitrary fact accepted")
	}
}
func TestCompositionFactReverseLinksAndOrder(t *testing.T) {
	for _, name := range []string{"extra fact", "missing origin", "changed root member", "reversed same-run prefix", "future profile", "future binding table"} {
		t.Run(name, func(t *testing.T) {
			s, _ := compositionTestStore(t)
			for range 2 {
				if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "extra fact":
				f, err := readCompositionFacts(t.Context(), s.conn)
				if err != nil {
					t.Fatal(err)
				}
				v := f[0]
				v.Order = 3
				v.ID = compositionIdentity("invented", 1)
				raw, _ := json.Marshal(v)
				if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_facts VALUES(?,?,?,?)", v.Order, v.ID, v.Kind, raw); err != nil {
					t.Fatal(err)
				}
			case "missing origin":
				if _, err := s.conn.ExecContext(t.Context(), "DELETE FROM run_origins WHERE subject='B'"); err != nil {
					t.Fatal(err)
				}
			case "changed root member":
				if _, err := s.conn.ExecContext(t.Context(), "UPDATE composition_static_members SET descriptor='{}' WHERE subject='B'"); err != nil {
					t.Fatal(err)
				}
			case "reversed same-run prefix":
				f, err := readCompositionFacts(t.Context(), s.conn)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.conn.ExecContext(t.Context(), "DELETE FROM composition_facts"); err != nil {
					t.Fatal(err)
				}
				for i := range f {
					v := f[1-i]
					v.Order = int64(i + 1)
					raw, _ := json.Marshal(v)
					if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_facts VALUES(?,?,?,?)", v.Order, v.ID, v.Kind, raw); err != nil {
						t.Fatal(err)
					}
				}
			case "future profile":
				if _, err := s.conn.ExecContext(t.Context(), "UPDATE composition_profile SET version='future'"); err != nil {
					t.Fatal(err)
				}
			case "future binding table":
				if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
					t.Fatal(err)
				}
				if _, err := s.conn.ExecContext(t.Context(), "INSERT INTO composition_bindings VALUES('invented','missing','{}')"); err != nil {
					t.Fatal(err)
				}
				if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=ON"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
				t.Fatal("invalid reverse link or ordering accepted")
			} else {
				t.Logf("foundation rejection: %v", err)
			}
		})
	}
}

func TestCompositionScopedRetryCannotEscapeFactCoverage(t *testing.T) {
	p := lifecycleFixture(t, []string{"Retry"}, false)
	c := compositionProvision{Roots: []compositionRoot{{Run: RunID{"worker", 1}, Members: []RunID{{"worker", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/retry.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Commit(t.Context(), current(t, s, "worker").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	request := committedRequest(t, s, "requester")
	if _, err := s.AuthorizeIntent(t.Context(), request.Intent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DecideLifecycle(t.Context(), request); !errors.Is(err, ErrVersion) {
		t.Fatalf("unfinished scoped Retry enabled: %v", err)
	}
	if count(t, s, "runs") != 2 || count(t, s, "lifecycle_results") != 0 || count(t, s, "composition_facts") != 1 {
		t.Fatal("scoped Retry escaped generation1 fact coverage")
	}
	if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestCompositionStaleLifecycleStillRecordsScopedBeforeState(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "requester", []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("worker")), Generation: Lit(Int(2)), Obligation: Lit(Text("obligation")), Payload: map[string]Operand{"operation": Lit(Text("CancelRun")), "revision": Lit(Int(1))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "worker", []Instruction{{Op: "end", Outcome: "failed"}}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "lifecycle", Source: "requester", Target: "worker", Operation: "CancelRun", Revision: 1}}
	c := compositionProvision{Roots: []compositionRoot{{Run: RunID{"worker", 1}, Members: []RunID{{"worker", 1}}, Limits: compositionLimits{MaxInFlight: 1}}}}
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/stale.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	request := committedRequest(t, s, "requester")
	result := authorizedDecision(t, s, request)
	if result.Accepted || result.Reason != "generation_changed" {
		t.Fatal(result)
	}
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].Run != (RunID{"worker", 1}) || facts[0].OperationID != request.Intent.ID {
		t.Fatalf("scoped before-state lacks ordered rejection: %+v", facts)
	}
	if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
