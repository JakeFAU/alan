package executor

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// A legacy name that was unknown before this feature remains inert retained
// data. Neither a literal spelling nor a dynamic spelling may mint authority.
func TestRequestAuthorityLegacyUnknownOperationCompatibility(t *testing.T) {
	for _, operation := range []string{"DelegateAuthority", "RevokeAuthority"} {
		for _, dynamic := range []bool{false, true} {
			name := operation + "/literal"
			if dynamic {
				name = operation + "/ref"
			}
			t.Run(name, func(t *testing.T) {
				operand := Lit(Text(operation))
				if dynamic {
					operand = Ref("operation")
				}
				p := Provision{}
				addProgram(t, &p, "A", []Instruction{{Op: "emit", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("legacy-unknown")), Payload: map[string]Operand{"operation": operand, "revision": Lit(Int(1))}}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{"operation": Text(operation)})
				path := filepath.Join(t.TempDir(), "legacy.sqlite")
				s, err := Create(t.Context(), path, p, DefaultLimits())
				if err != nil {
					t.Fatalf("legacy Create must retain formerly unknown operation: %v", err)
				}
				defer s.Close()
				if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
					t.Fatalf("legacy Commit must retain formerly unknown operation: %v", err)
				}
				id := identity("A", int64(1), int64(1), 0)
				if _, err = s.LoadLifecycleRequest(t.Context(), id); err == nil || !strings.Contains(err.Error(), "unknown lifecycle operation") {
					t.Fatalf("legacy loader must keep operation inert: %v", err)
				}
				if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
				if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				reopened, err := Open(t.Context(), path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if err = reopened.Close(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// Check each reader independently, including clean legacy controls. These
// malformed owned fixtures never call a transport.
func authorityLegacyReaders(t *testing.T, path string, malformed bool) {
	t.Helper()
	for _, reader := range []struct {
		name string
		read func() error
	}{
		{"Replay", func() error { _, err := Replay(t.Context(), path, DefaultLimits()); return err }},
		{"InspectExecution", func() error { _, err := InspectExecution(t.Context(), path, DefaultLimits()); return err }},
		{"Open", func() error {
			s, err := Open(t.Context(), path, DefaultLimits())
			if err == nil {
				err = s.Close()
			}
			return err
		}},
	} {
		t.Run(reader.name, func(t *testing.T) {
			err := reader.read()
			if malformed && err == nil {
				t.Error("legacy reader accepted forbidden typed authority witness")
			}
			if !malformed && err != nil {
				t.Fatalf("clean legacy reader: %v", err)
			}
		})
	}
}

func TestRequestAuthorityLegacyExternalWitness(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		name := "clean"
		if malformed {
			name = "forbidden_witness"
		}
		t.Run(name, func(t *testing.T) {
			s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1234/apply", "idempotent"))
			if _, err := s.Commit(t.Context(), current(t, s, "client").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), identity("client", int64(1), int64(1), 0))
			if err != nil {
				t.Fatal(err)
			}
			if malformed {
				a.Authority = &EffectAuthority{Version: RequestAuthorityVersion, Namespace: s.Namespace(), Allowed: true, Reason: "authorized"}
				raw, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.conn.ExecContext(t.Context(), "UPDATE external_attempts SET descriptor=? WHERE intent_id=?", raw, a.Intent.ID); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			authorityLegacyReaders(t, path, malformed)
		})
	}
}
