//go:build integration

package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

func fixtureAssetScopes(t *testing.T, p *Provision, capability string) {
	t.Helper()
	a, err := effects.NewAsset("doc", "fixture", []byte("qualified content"))
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = []effects.Asset{a}
	p.AssetScopes = map[string][]string{capability: {a.ID}}
}

func checkScopedHistory(t *testing.T, source string, want int) {
	t.Helper()
	l := DefaultLimits()
	ctx := t.Context()
	// The fixture owner is idle between synchronous operations. Copy the exact
	// database and WAL for an independent reopen without changing its epochs.
	path := filepath.Join(t.TempDir(), fmt.Sprintf("format-%d.sqlite", want))
	for _, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(source + suffix)
		if suffix != "" && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path+suffix, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	owner, err := Open(ctx, path, l)
	if err != nil {
		t.Fatalf("format %d reopen: %v", want, err)
	}
	if err = owner.Close(); err != nil {
		t.Fatal(err)
	}
	snap, err := InspectExecution(ctx, path, l)
	if err != nil || snap.StorageVersion != want || snap.AssetScopes == nil {
		t.Fatalf("format %d inspection: %v", want, err)
	}
	if _, err := Replay(ctx, path, l); err != nil {
		t.Fatalf("format %d replay: %v", want, err)
	}
	if old := os.Getenv("ALAN_V010_OLD_READER"); old != "" {
		for _, cmd := range []string{"inspect", "replay"} {
			out, err := exec.CommandContext(ctx, old, cmd, "--db", path).CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || strings.TrimSpace(string(out)) != "unsupported version" {
				t.Fatalf("frozen %s format %d: %v %s", cmd, want, err, out)
			}
		}
	}
	t.Logf("format %d inspected, reopened and replayed", want)
}

func TestScopedAssetHistoryFormats(t *testing.T) {
	// The protected-continuation pair is exercised by TestScopedContinuationConflictPromotion.
	for _, base := range []int{3, 4, 5, 9, 13, 15} {
		t.Run(fmt.Sprint(base+assetScopeFormatOffset), func(t *testing.T) {
			var s *Store
			var path string
			var attempt ExternalAttempt
			ctx := t.Context()
			if base == 13 || base == 15 {
				p := requestAuthorityFixture(t, nil)
				if base == 15 {
					p = bindingV2Fixture(t, nil)
				}
				fixtureAssetScopes(t, &p, "unused")
				s, path, _, attempt = authorityReady(t, p, true)
				op, reason, err := s.registerDispatch(ctx, attempt.Intent.ID)
				if err != nil || reason != "" {
					t.Fatal(reason, err)
				}
				s.finishDispatch(op, nil)
			} else {
				p := externalProvision(t, "http://127.0.0.1:1", "non-idempotent")
				if base != 3 {
					inv, profile := invocationFixture(t, "succeeded", 1)
					p.Programs = append(p.Programs, inv.Programs...)
					p.Subjects = append(p.Subjects, inv.Subjects...)
					p.Runs = append(p.Runs, inv.Runs...)
					p.Permissions = append(p.Permissions, inv.Permissions...)
					profile.RecordDispatchAdmissions = base == 9
					p.Composition = &profile
				}
				if base == 5 {
					p.Reconciliation = &ReconciliationProfile{}
				}
				fixtureAssetScopes(t, &p, "unused")
				s, path = messageStore(t, p)
				r, err := s.Commit(ctx, current(t, s, "client").Authority(), "")
				if err != nil {
					t.Fatal(err)
				}
				attempt, _, err = s.prepareExternal(ctx, r.Projection.Data["pending"].Text)
				if err != nil || !attempt.Allowed {
					t.Fatal(attempt, err)
				}
			}
			first := receiptFor(attempt.Request)
			if err := s.recordExternalReceipt(ctx, attempt, first); err != nil {
				t.Fatal(err)
			}
			checkScopedHistory(t, path, base+assetScopeFormatOffset)
			first.Total++
			if err := s.recordExternalReceipt(ctx, attempt, first); !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			promoted := base + 1
			if base <= 5 {
				promoted = base + 3
			}
			checkScopedHistory(t, path, promoted+assetScopeFormatOffset)
		})
	}
}

func TestScopedAssetClosedCompositionWithholdsPreparation(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "a", false)
	addAdapterProgram(t, &p, "b", true)
	fixtureAssetScopes(t, &p, "model")
	p.Composition = &CompositionProvision{Roots: []ResourceRoot{{Run: RunID{"a", 1}, Members: []RunID{{"a", 1}, {"b", 1}}, Capabilities: []string{"model"}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}}}}
	s, path := messageStore(t, p)
	ctx := t.Context()
	if _, err := s.Commit(ctx, current(t, s, "a").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	r, err := s.Commit(ctx, current(t, s, "b").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := r.Projection.Data["pending"].Text
	if a, err := s.AuthorizeIntent(ctx, id); err != nil || !a.Allowed {
		t.Fatal(a, err)
	}
	if _, err := s.Commit(ctx, current(t, s, "a").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	denied := callbackAdapter{prepare: func(effects.Request) (effects.Prepared, error) {
		t.Fatal("asset disclosure after root closure")
		return effects.Prepared{}, nil
	}}
	if w, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": denied}, time.Second); err != nil || w != nil {
		t.Fatal(w, err)
	}
	a, err := loadAdapterAttempt(ctx, s.conn, id)
	if err != nil || a.Allowed || a.Reason != "scope_closed" {
		t.Fatal(a, err)
	}
	if count(t, s, "composition_charges") != 0 {
		t.Fatal("refusal spent effect budget")
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestScopedAssetTamper(t *testing.T) {
	for _, change := range []string{"manifest", "version", "mask format", "drop and mask", "historical selection"} {
		t.Run(change, func(t *testing.T) {
			p := Provision{}
			addAdapterProgram(t, &p, "worker", true)
			fixtureAssetScopes(t, &p, "model")
			s, path := messageStore(t, p)
			ctx := t.Context()
			if _, err := s.Run(ctx, RunOptions{MaxSteps: 8, MaxDispatch: 8, Adapters: map[string]effects.Adapter{"model": callbackAdapter{}}}); err != nil {
				t.Fatal(err)
			}
			want := ErrCorrupt
			switch change {
			case "manifest":
				if _, err := s.conn.ExecContext(ctx, "UPDATE capability_asset_scopes SET descriptor=CAST('{}' AS BLOB)"); err != nil {
					t.Fatal(err)
				}
			case "version":
				a, err := effects.NewAsset("doc", "fixture", []byte("different bytes"))
				if err != nil {
					t.Fatal(err)
				}
				raw, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.conn.ExecContext(ctx, "UPDATE corpus_assets SET descriptor=?", raw); err != nil {
					t.Fatal(err)
				}
			case "mask format", "drop and mask":
				if change == "drop and mask" {
					if _, err := s.conn.ExecContext(ctx, "DROP TABLE capability_asset_scopes"); err != nil {
						t.Fatal(err)
					}
				}
				want = ErrVersion
				if _, err := s.conn.ExecContext(ctx, "UPDATE metadata SET schema_version=3; PRAGMA user_version=3"); err != nil {
					t.Fatal(err)
				}
			case "historical selection":
				if _, err := s.conn.ExecContext(ctx, "UPDATE metadata SET asset_scope_digest=?", compositionIdentity("alan/asset-scope/v1", assetManifest{})); err != nil {
					t.Fatal(err)
				}
				raw := []byte("{}")
				if _, err := s.conn.ExecContext(ctx, "UPDATE capability_asset_scopes SET descriptor=?", raw); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			_, a := Inspect(ctx, path, DefaultLimits())
			_, b := InspectExecution(ctx, path, DefaultLimits())
			_, c := Replay(ctx, path, DefaultLimits())
			o, d := Open(ctx, path, DefaultLimits())
			if o != nil {
				o.Close()
			}
			for _, err := range []error{a, b, c, d} {
				if !errors.Is(err, want) {
					t.Fatalf("%s: %v expected %v", change, err, want)
				}
			}
		})
	}
}

func TestScopedAssetRevokedDelegation(t *testing.T) {
	p := requestAuthorityAdapterFixture(t, "resource1", "increment")
	fixtureAssetScopes(t, &p, "resource1")
	var root AuthorityRoot
	authorityReplaceProgram(t, &p, "A", func(program *Program) { root = program.Code[1].Effect.Authority.Root })
	claim, err := json.Marshal(RevocationClaim{Version: RequestAuthorityVersion, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	authorityReplaceProgram(t, &p, "B", func(program *Program) {
		program.Code = append(program.Code[:2], Instruction{Op: "emit", Dst: "pending", Effect: &Effect{
			Kind: "lifecycle", Target: Lit(Text("B")), Generation: Lit(Int(1)), Obligation: Lit(Text("revoke")),
			Payload: map[string]Operand{"operation": Lit(Text("RevokeAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(claim)))},
		}}, Instruction{Op: "end", Outcome: "succeeded"})
	})
	ctx := t.Context()
	s, path := messageStore(t, p)
	if _, err = s.Commit(ctx, current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if grant := authorizedDecision(t, s, committedRequest(t, s, "B")); !grant.Accepted {
		t.Fatal(grant)
	}
	msg, err := s.DeliverMessage(ctx, identity("B", int64(1), int64(1), 0), "delivery")
	if err != nil || !msg.Accepted {
		t.Fatal(err, msg)
	}
	if _, err = s.Commit(ctx, current(t, s, "A").Authority(), msg.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(ctx, current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if revoked := authorizedDecision(t, s, committedRequest(t, s, "B")); !revoked.Accepted {
		t.Fatal(revoked)
	}
	id := identity("A", int64(1), int64(2), 0)
	callback := callbackAdapter{prepare: func(effects.Request) (effects.Prepared, error) {
		t.Fatal("revoked delegated capability received selected bytes")
		return effects.Prepared{}, nil
	}}
	work, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"resource1": callback}, time.Second)
	if err != nil || work != nil {
		t.Fatal(work, err)
	}
	attempt, err := loadAdapterAttempt(ctx, s.conn, id)
	if err != nil || attempt.Allowed || attempt.Authority.Reason != "authority_revoked" {
		t.Fatal(attempt, err)
	}
	if _, err = Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExecution(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestScopedAssetRestartAfterPaidReservation(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "worker", true)
	fixtureAssetScopes(t, &p, "model")
	unrelated, err := effects.NewAsset("other", "fixture", []byte("excluded"))
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = append(p.Assets, unrelated)
	ctx := t.Context()
	s, path := messageStore(t, p)
	step, err := s.Commit(ctx, current(t, s, "worker").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := step.Projection.Data["pending"].Text
	work, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || work == nil || !work.attempt.Allowed {
		t.Fatal(work, err)
	}
	pending := work.attempt.Prepared
	if !slices.Equal(pending.Request.Assets, p.Assets[:1]) {
		t.Fatal(pending.Request.Assets)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	// The reservation is durable and its external result is unknown. The
	// recovered owner must neither reconstruct it via an adapter nor redispatch.
	if next, err := owner.prepareAdapterWork(ctx, id, nil, time.Second); err != nil || next != nil {
		t.Fatal(next, err)
	}
	cut, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	e, ok := cut.Effect(id)
	if !ok || e.AdapterAttempt == nil || e.AdapterResult != nil || e.AdapterAttempt.Prepared.Digest != pending.Digest || !slices.Equal(e.AdapterAttempt.Prepared.Request.Assets, p.Assets[:1]) {
		t.Fatal("paid selection changed during recovery", e)
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
