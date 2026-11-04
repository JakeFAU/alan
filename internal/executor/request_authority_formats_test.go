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
				// This is a freshly generated closed Store, not archival evidence. The old
				// reader receives a disposable complete copy after both owners checkpointed.
				copied := filepath.Join(t.TempDir(), "old-reader.sqlite")
				for _, suffix := range []string{"", "-wal", "-shm"} {
					b, e := os.ReadFile(path + suffix)
					if errors.Is(e, os.ErrNotExist) && suffix != "" {
						continue
					}
					if e != nil {
						t.Fatal(e)
					}
					if e = os.WriteFile(copied+suffix, b, 0600); e != nil {
						t.Fatal(e)
					}
				}
				for _, command := range []string{"inspect", "replay"} {
					out, e := exec.CommandContext(t.Context(), old, command, "--db", copied).CombinedOutput()
					if e == nil || !strings.Contains(string(out), "unsupported version") {
						t.Fatalf("old %s exit=%v output=%s", command, e, out)
					}
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
