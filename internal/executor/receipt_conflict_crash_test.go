//go:build integration

package executor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/mattn/go-sqlite3"
)

// The child dies either inside SQLite's commit hook or after the recorder
// acknowledges the conflict. Neither path closes the connection or the WAL.
func TestReceiptConflictCrash(t *testing.T) {
	ctx := t.Context()
	id := identity("client", 1, 1, 0)
	if path := os.Getenv("ALAN_CONFLICT_CRASH_DB"); path != "" {
		s, err := Open(ctx, path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		a, err := loadExternalAttempt(ctx, s.conn, id)
		if err != nil {
			t.Fatal(err)
		}
		loser := receiptFor(a.Request)
		loser.Total = 10
		kill := func() {
			if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
				panic(err)
			}
			// Darwin can return from Kill before delivering the signal.
			select {}
		}
		if os.Getenv("ALAN_CONFLICT_CRASH_CUT") == "precommit" {
			if err := s.conn.Raw(func(c any) error {
				c.(*sqlite3.SQLiteConn).RegisterCommitHook(func() int { kill(); return 1 })
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			t.Fatalf("survived the commit hook: %v", s.recordExternalReceipt(ctx, a, loser))
		}
		if err := s.recordExternalReceipt(ctx, a, loser); !errors.Is(err, ErrConflict) {
			t.Fatalf("recorder returned %v before crash", err)
		}
		kill()
		return
	}

	for _, version := range []int{3, 4, 5} {
		for _, cut := range []string{"precommit", "acknowledged"} {
			t.Run(fmt.Sprintf("v%d/%s", version, cut), func(t *testing.T) {
				p := externalProvision(t, "http://127.0.0.1:1", "non-idempotent")
				if version >= 4 {
					client := RunID{"client", 1}
					p.Composition = &compositionProvision{Roots: []compositionRoot{{
						Run: client, Members: []RunID{client}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1},
					}}}
				}
				if version == 5 {
					p.Reconciliation = &ReconciliationProfile{}
				}
				path := filepath.Join(t.TempDir(), "effect.sqlite")
				s, err := Create(ctx, path, p, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				if _, err := s.Commit(ctx, current(t, s, "client").Authority(), ""); err != nil {
					t.Fatal(err)
				}
				a, _, err := s.prepareExternal(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.recordExternalReceipt(ctx, a, receiptFor(a.Request)); err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReceiptConflictCrash$")
				cmd.Env = append(os.Environ(), "ALAN_CONFLICT_CRASH_DB="+path, "ALAN_CONFLICT_CRASH_CUT="+cut)
				out, err := cmd.CombinedOutput()
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
					t.Fatalf("child was not killed at %s: %v\n%s", cut, err, out)
				}
				replayed, err := Replay(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				seen, err := Inspect(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				wantVersion := version
				if cut == "acknowledged" {
					wantVersion = version + 3
					if len(replayed.ReceiptEvidence) != 1 || replayed.ReceiptEvidence[0].IntentID != id ||
						replayed.ReceiptEvidence[0].State != "conflict" || len(seen.Tables["receipt_conflicts"]) != 1 {
						t.Fatalf("lost acknowledged conflict: %+v", replayed.ReceiptEvidence)
					}
				} else if _, exists := seen.Tables["receipt_conflicts"]; exists || len(replayed.ReceiptEvidence) != 0 {
					t.Fatal("uncommitted conflict survived the crash")
				}
				if got := seen.Tables["metadata"][0]["schema_version"]; got != int64(wantVersion) {
					t.Fatalf("physical format %v, want %d", got, wantVersion)
				}
				typed, err := InspectExecution(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				e, ok := typed.Effect(id)
				if !ok || typed.StorageVersion != wantVersion || e.Source != a.Intent.Run || e.Sequence != a.Intent.Sequence ||
					e.State != "result_unsettled" || e.Settlement != nil || !reflect.DeepEqual(e.ExternalReceipt, new(receiptFor(a.Request))) ||
					len(e.AdditionalReceipts) != len(replayed.ReceiptEvidence) || (e.ReceiptEvidenceState == "conflict") != (cut == "acknowledged") {
					t.Fatalf("typed crash cut does not preserve original receipt and later evidence: %+v", e)
				}
				s, err = Open(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				// No server is listening. Recovery must settle and select the first receipt.
				if _, err := s.Run(ctx, RunOptions{MaxSteps: 8, MaxDispatch: 8}); err != nil {
					t.Fatal(err)
				}
				run := current(t, s, "client")
				if run.Projection.Status != "succeeded" || run.Projection.Data["total"] != Int(5) ||
					count(t, s, "settlements") != 1 {
					t.Fatalf("recovered continuation: %+v", run.Projection)
				}
				again, err := Replay(ctx, path, DefaultLimits())
				if err != nil || !reflect.DeepEqual(again.ReceiptEvidence, replayed.ReceiptEvidence) {
					t.Fatalf("settlement altered receipt evidence: %+v, %v", again.ReceiptEvidence, err)
				}
				recovered, err := InspectExecution(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				settled, ok := recovered.Effect(id)
				if !ok || settled.Settlement == nil || settled.State != "settled" || !reflect.DeepEqual(settled.ExternalReceipt, e.ExternalReceipt) {
					t.Fatalf("typed reopened settlement lost its original receipt: %+v", settled)
				}
				obs, ok := recovered.Observation(settled.Settlement.ObservationID)
				if !ok || obs.Operation != id || obs.SelectedAt == nil || obs.SelectedAt.Run != a.Intent.Run {
					t.Fatalf("typed reopened settlement lost its original consumer: %+v", obs)
				}
			})
		}
	}
}
