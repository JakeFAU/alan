//go:build integration

package executor

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

// Exercise the original writer and real receipt-conflict promotion. Ordinary
// stores can retain never-created targets. Only composition stores can later
// create the target via an invocation.
func TestLifecycleRecoveryAcrossFormats(t *testing.T) {
	for version := 3; version <= 8; version++ {
		for _, forged := range []bool{false, true} {
			t.Run(fmt.Sprintf("v%d/forged=%t", version, forged), func(t *testing.T) {
				ctx := t.Context()
				path := filepath.Join(t.TempDir(), "cut.sqlite")
				composed := version != 3 && version != 6
				p := externalProvision(t, "http://127.0.0.1:1", "non-idempotent")
				if composed {
					inv, profile := invocationFixture(t, "succeeded", 1)
					p.Programs = append(p.Programs, inv.Programs...)
					p.Subjects = append(p.Subjects, inv.Subjects...)
					p.Runs = append(p.Runs, inv.Runs...)
					p.Permissions = append(p.Permissions, inv.Permissions...)
					p.Composition = &profile
				}
				if version == 5 || version == 8 {
					p.Reconciliation = &ReconciliationProfile{}
				}
				addRetirementController(t, &p, []string{"CancelRun"})
				s, err := Create(ctx, path, p, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				client, err := s.Commit(ctx, current(t, s, "client").Authority(), "")
				if err != nil {
					t.Fatal(err)
				}
				op := client.Projection.Data["pending"].Text
				attempt, _, err := s.prepareExternal(ctx, op)
				if err != nil || !attempt.Allowed {
					t.Fatal(attempt, err)
				}
				if version >= 6 {
					receipt := receiptFor(attempt.Request)
					if err := s.recordExternalReceipt(ctx, attempt, receipt); err != nil {
						t.Fatal(err)
					}
					receipt.Total++
					if err := s.recordExternalReceipt(ctx, attempt, receipt); !errors.Is(err, ErrConflict) {
						t.Fatal(err)
					}
				}
				target := "never-created"
				if composed {
					target = compositionMemberID(compositionInvocationID(s.Namespace(), identity("root", 1, 1, 0)), "entry").Subject
					if forged && !invokeOnce(t, s).Accepted {
						t.Fatal("initial member creation refused")
					}
				} else if forged {
					target = "client"
				}
				const controller = "retirement-controller"
				input, err := s.AdmitInput(ctx, Input{TransportID: "control", LogicalID: "lookup", Target: controller, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(target)}})
				if err != nil || !input.Accepted {
					t.Fatal(input, err)
				}
				if _, err = s.Commit(ctx, current(t, s, controller).Authority(), input.ObservationID); err != nil {
					t.Fatal(err)
				}
				d := authorizedDecision(t, s, committedRequest(t, s, controller))
				if !forged && d.Reason != "target_missing" || forged && (d.Reason != "grant_denied" || d.Before == nil) {
					t.Fatal("writer decisions", d)
				}
				if composed && !forged && !invokeOnce(t, s).Accepted {
					t.Fatal("later creation refused")
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				cut, err := InspectExecution(ctx, path, DefaultLimits())
				if err != nil || cut.StorageVersion != version {
					t.Fatal(cut, err)
				}
				conflicts := 0
				if version >= 6 {
					conflicts = 1
				}
				if effect, ok := cut.Effect(op); !ok || effect.ExternalAttempt == nil || effect.Settlement != nil || (effect.ExternalReceipt != nil) != (conflicts > 0) || len(effect.AdditionalReceipts) != conflicts {
					t.Fatal("unexpected original effect evidence", effect)
				}
				if original, ok := cut.LifecycleDecision(d.Request.Intent.ID); !ok || !reflect.DeepEqual(original, d) {
					t.Fatal(original)
				}
				report, err := Replay(ctx, path, DefaultLimits())
				if err != nil || report.Namespace != cut.Namespace || len(report.Runs) != len(cut.Runs) {
					t.Fatal(report, err)
				}
				if forged {
					editor, err := connect(ctx, path, false)
					if err != nil {
						t.Fatal(err)
					}
					d.Before, d.After, d.Reason = nil, nil, "target_missing"
					if _, err = editor.conn.ExecContext(ctx, "UPDATE lifecycle_results SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(d), d.Request.Intent.ID); err != nil {
						t.Fatal(err)
					}
					if composed {
						fact := compositionIdentity("alan/fact/lifecycle/v1", d.Request.Intent.ID)
						r, err := editor.conn.ExecContext(ctx, "DELETE FROM composition_facts WHERE id=?", fact)
						if err != nil {
							t.Fatal(err)
						}
						if n, err := r.RowsAffected(); err != nil || n != 1 {
							t.Fatal(n, err)
						}
					}
					if err = editor.Close(); err != nil {
						t.Fatal(err)
					}
					before := inspectionFingerprint(t, path)
					if result, err := InspectExecution(ctx, path, DefaultLimits()); result != nil || !errors.Is(err, ErrCorrupt) {
						t.Fatal("inspection accepted forged absence", err)
					}
					if _, err := Replay(ctx, path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
						t.Fatal("replay accepted forged absence", err)
					}
					owner, err := Open(ctx, path, DefaultLimits())
					if owner != nil {
						owner.Close()
						t.Fatal("granted invalid owner")
					}
					if !errors.Is(err, ErrCorrupt) || inspectionFingerprint(t, path) != before {
						t.Fatal("failed open changed durable state", err)
					}
					return
				}
				owner, err := Open(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if err = owner.Close(); err != nil {
					t.Fatal(err)
				}
				after, err := InspectExecution(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				for _, r := range cut.Runs {
					if r.Projection.Status == "open" {
						r.Epoch++
					}
				}
				if !reflect.DeepEqual(cut, after) {
					t.Fatal("successful reopen changed evidence beyond the open generations' activations")
				}
			})
		}
	}
}
