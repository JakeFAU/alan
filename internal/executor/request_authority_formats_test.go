package executor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
)

func TestRequestAuthorityFormatsConflictPromotionAndOldReader(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		name := "13"
		if conflict {
			name = "14"
		}
		t.Run(name, func(t *testing.T) {
			s, path, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
			op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
			if err != nil || reason != "" {
				t.Fatal(err)
			}
			s.finishDispatch(op, nil)
			if conflict {
				receipt := resourcefixture.Receipt{Key: a.Request.Key, Profile: a.Request.Profile, Delta: a.Request.Delta, Mutation: 1, Total: 1, Process: "controlled-test", PID: 1}
				if err = s.recordExternalReceipt(t.Context(), a, receipt); err != nil {
					t.Fatal(err)
				}
				receipt.Total++
				if err = s.recordExternalReceipt(t.Context(), a, receipt); !errors.Is(err, ErrConflict) {
					t.Fatalf("losing receipt %v", err)
				}
			}
			version, err := storedSchemaNumber(t.Context(), s.conn)
			want := 13
			if conflict {
				want = 14
			}
			if err != nil || version != want {
				t.Fatalf("version=%d %v", version, err)
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
			t.Run("frozen_old_reader", func(t *testing.T) {
				old := os.Getenv("ALAN_PHASE3B3_OLD_READER")
				if old == "" {
					t.Skip("set ALAN_PHASE3B3_OLD_READER for frozen-reader qualification")
				}
				for _, command := range []string{"inspect", "replay"} {
					t.Run(command, func(t *testing.T) {
						// Each reader receives its own complete copy of the closed Store.
						copied := filepath.Join(t.TempDir(), "old-reader.sqlite")
						for _, suffix := range []string{"", "-wal", "-shm"} {
							data, err := os.ReadFile(path + suffix)
							if errors.Is(err, os.ErrNotExist) && suffix != "" {
								continue
							}
							if err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(copied+suffix, data, 0600); err != nil {
								t.Fatal(err)
							}
						}
						out, err := exec.CommandContext(t.Context(), old, command, "--db", copied).CombinedOutput()
						var exitErr *exec.ExitError
						if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || strings.TrimSpace(string(out)) != "unsupported version" {
							t.Fatalf("old %s exit=%v output=%q, want exit 1 and unsupported version", command, err, out)
						}
						t.Logf("old %s exit=%d output=%q", command, exitErr.ExitCode(), out)
					})
				}
			})
		})
	}
}
func TestRequestAuthorityFormatsFeatureMasking(t *testing.T) {
	s, path, _, _ := authorityReady(t, requestAuthorityFixture(t, nil), true)
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE metadata SET schema_version=4"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "PRAGMA user_version=4"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("Replay accepted masked feature")
	}
	if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("inspection accepted masked feature")
	}
}
func TestRequestAuthoritySharedRootHolderIsDistinct(t *testing.T) {
	p := requestAuthorityFixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
		d.Root.RootRun = RunID{"A", 1}
		u.Root = d.Root
	})
	p.Composition.Roots = []ResourceRoot{{Run: RunID{"A", 1}, Members: []RunID{{"A", 1}, {"B", 1}}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}}}
	s, path, _, a := authorityReady(t, p, true)
	if !a.Allowed || a.Authority.Use.Root.Run == a.Authority.Use.Root.RootRun {
		t.Fatalf("holder collapsed to root owner: %+v", a)
	}
	op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || reason != "" {
		t.Fatal(err)
	}
	s.finishDispatch(op, nil)
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
