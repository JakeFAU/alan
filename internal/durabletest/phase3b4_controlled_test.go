//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

func bindingDeliver(t *testing.T, s *runtime.Store, id string) string {
	t.Helper()
	a, err := s.AuthorizeIntent(t.Context(), id)
	if err != nil || !a.Allowed {
		t.Fatal("source", a, err)
	}
	transport := "controlled / transport for " + id
	r, err := s.DeliverMessage(t.Context(), id, transport)
	if err != nil || !r.Accepted {
		t.Fatal("delivery", r, err)
	}
	if err = s.AcknowledgeMessage(t.Context(), id, transport); err != nil {
		t.Fatal(err)
	}
	return r.ObservationID
}
func bindingGrant(t *testing.T, s *runtime.Store, m delegation.Manifest) {
	t.Helper()
	reciprocalCommit(t, s, m.B, "")
	a, err := s.AuthorizeIntent(t.Context(), m.Grant)
	if err != nil || !a.Allowed {
		t.Fatal(a, err)
	}
	r, err := s.LoadLifecycleRequest(t.Context(), m.Grant)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.DecideLifecycle(t.Context(), r)
	if err != nil || !d.Accepted {
		t.Fatal(d, err)
	}
	a2, err := s.SettleLifecycle(t.Context(), m.Grant, "controlled grant result")
	if err != nil || !a2.Accepted {
		t.Fatal(a2, err)
	}
	reciprocalCommit(t, s, m.B, a2.ObservationID)
	reciprocalCommit(t, s, m.B, "")
	reciprocalCommit(t, s, m.B, "")
}
func bindingInitial(t *testing.T, s *runtime.Store, m delegation.Manifest) (string, string) {
	t.Helper()
	reciprocalCommit(t, s, m.A, "")
	ab := bindingDeliver(t, s, m.ABRequest)
	reciprocalCommit(t, s, m.B, "")
	q := bindingDeliver(t, s, m.BARequest)
	return ab, q
}
func bindingCaptureLive(t *testing.T, path, dir string) *runtime.ExecutionSnapshot {
	t.Helper()
	backup := filepath.Join(t.TempDir(), "capture.sqlite")
	reciprocalBackup(t, path, backup)
	return bindingCapture(t, backup, dir)
}
func bindingNarrow(t *testing.T, p *runtime.Provision, m *delegation.RequestBindingManifest) {
	t.Helper()
	reciprocalEdit(t, p, "a", func(program map[string]any) {
		code := program["Code"].([]any)
		program["Code"] = []any{code[0], code[3], code[1], code[2], map[string]any{"Op": "end", "Outcome": "failed"}}
	})
	m.ServiceSequence = 2
	m.Reciprocal.BAService = `["a",1,2,0]`
	m.Reciprocal.BAReply = ""
	m.ReplySequence = 0
	m.Reciprocal.Claim.ServiceOperationID = m.Reciprocal.BAService
	bindingClaimAll(t, p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = m.Reciprocal.BAService })
}

// Run's pending-dispatch phase precedes its next ordinary commit. Stop that
// next staged commit without fabricating a decision or selecting its observation.
func bindingFirstDecision(t *testing.T, s *runtime.Store, dir string) {
	t.Helper()
	stop := errors.New("controlled boundary before next ordinary commit")
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
		if e.Boundary == "execution.staged" {
			return stop
		}
		return nil
	})
	report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	prior, globErr := filepath.Glob(filepath.Join(dir, "decision-run-*.json"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	baseline3b3Write(t, dir, fmt.Sprintf("decision-run-%02d.json", len(prior)+1), map[string]any{"Report": report, "Error": fmt.Sprint(err), "ControlledStop": errors.Is(err, stop)})
	if err != nil && !errors.Is(err, stop) {
		t.Fatal(err)
	}
}

func TestPhase3B4CampaignControlled(t *testing.T) {
	for _, name := range []string{"P08", "P09", "N01", "N02", "N03", "N04"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m := bindingFixture(t, ua, ub, 0, nil, false)
			if name == "N01" {
				bindingQ2(t, &p, &m, false)
			}
			if name == "N02" || name == "N03" || name == "N04" {
				bindingNarrow(t, &p, &m)
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			_, q := bindingInitial(t, s, m.Reciprocal)
			c := bindingCase{Name: name, Schedule: "controlled-public-transitions-then-run", Manifest: m}
			capture := func(label string) *runtime.ExecutionSnapshot {
				snap := bindingCaptureLive(t, db, filepath.Join(dir, label))
				prev := ""
				if len(c.Captures) > 0 {
					prev = c.Captures[len(c.Captures)-1].Directory
				}
				c.Captures = append(c.Captures, bindingCaptureRef{Ordinal: len(c.Captures) + 1, Predecessor: prev, Label: label, Directory: label})
				return snap
			}
			capture("admitted-before-selection")
			switch name {
			case "P08":
				reciprocalCommit(t, s, m.Reciprocal.A, q)
				capture("selected-before-grant")
				bindingGrant(t, s, m.Reciprocal)
				ready := bindingDeliver(t, s, m.Reciprocal.Ready)
				reciprocalCommit(t, s, m.Reciprocal.A, ready)
			case "P09":
				bindingGrant(t, s, m.Reciprocal)
				capture("grant-before-selection")
				reciprocalCommit(t, s, m.Reciprocal.A, q)
				ready := bindingDeliver(t, s, m.Reciprocal.Ready)
				reciprocalCommit(t, s, m.Reciprocal.A, ready)
			case "N01":
				bindingGrant(t, s, m.Reciprocal)
				ready := bindingDeliver(t, s, m.Reciprocal.Ready)
				reciprocalCommit(t, s, m.Reciprocal.A, ready)
				reciprocalCommit(t, s, m.Reciprocal.B, "")
				q2 := bindingDeliver(t, s, `["b",1,6,0]`)
				reciprocalCommit(t, s, m.Reciprocal.A, q2)
				c.AuxiliaryRequests = []string{`["b",1,6,0]`}
			case "N02":
				bindingGrant(t, s, m.Reciprocal)
				reciprocalCommit(t, s, m.Reciprocal.A, "")
				capture("service-before-selection")
				reciprocalCommit(t, s, m.Reciprocal.A, q)
			case "N03":
				reciprocalCommit(t, s, m.Reciprocal.A, "")
			case "N04":
				bindingGrant(t, s, m.Reciprocal)
				reciprocalCommit(t, s, m.Reciprocal.A, "")
			}
			if name == "P08" || name == "P09" {
				report, err := bindingRun(t, s, dir, 256)
				if err != nil || !report.Quiescent || len(report.OpenRuns) != 0 {
					t.Fatal(report, err)
				}
				c.Classification = "safe-complete"
			} else {
				if name == "N01" {
					reciprocalCommit(t, s, m.Reciprocal.A, "")
				}
				bindingFirstDecision(t, s, dir)
				before := capture("first-refusal")
				e, _ := before.Effect(m.Reciprocal.BAService)
				want := "authority_request_not_selected"
				if name == "N03" {
					want = "authority_grant_missing"
				}
				if e == nil || e.ExternalAttempt == nil || e.ExternalAttempt.Allowed || e.ExternalAttempt.Authority.Reason != want || e.ExternalAttempt.Authority.RequestConsumedSequence != 0 || e.ExternalAttempt.Authority.RequestObservationID != "" {
					t.Fatal("wrong refusal layer", e)
				}
				if name == "N03" {
					bindingGrant(t, s, m.Reciprocal)
				}
				if name == "N03" || name == "N04" {
					reciprocalCommit(t, s, m.Reciprocal.A, q)
				}
				if name != "N01" {
					ready := bindingDeliver(t, s, m.Reciprocal.Ready)
					reciprocalCommit(t, s, m.Reciprocal.A, ready)
				}
				bindingFirstDecision(t, s, dir)
				after := capture("after-later-facts")
				later, _ := after.Effect(m.Reciprocal.BAService)
				if !reflect.DeepEqual(e.ExternalAttempt, later.ExternalAttempt) || len(later.AuthorityAdmissions) != 0 {
					t.Fatal("refusal repaired", later)
				}
				c.Classification = want
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			snap := bindingCapture(t, db, filepath.Join(dir, "final"))
			c.Captures = append(c.Captures, bindingCaptureRef{Ordinal: len(c.Captures) + 1, Predecessor: c.Captures[len(c.Captures)-1].Directory, Label: "final", Directory: "final"})
			if name == "P08" || name == "P09" {
				bindingComplete(t, snap, m.Reciprocal, endpoints)
			} else {
				if endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal(endpoints)
				}
				qv := bindingSelected(t, snap, m.Reciprocal.A, m.Reciprocal.BARequest)
				if name == "N01" {
					if qv.SelectedAt != nil {
						t.Fatal("q became selected", qv)
					}
					q2 := bindingSelected(t, snap, m.Reciprocal.A, c.AuxiliaryRequests[0])
					if q2.SelectedAt == nil || q2.SelectedAt.Sequence != 3 {
						t.Fatal("auxiliary selection missing", q2)
					}
				} else if qv.SelectedAt == nil || qv.SelectedAt.Sequence != 3 {
					t.Fatal("future q selection missing", qv)
				}
			}
			// Open must preserve refusals, selections and exact program identities.
			reopened, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			reopened.Close()
			bindingCapture(t, db, filepath.Join(dir, "post-open"))
			c.Captures = append(c.Captures, bindingCaptureRef{Ordinal: len(c.Captures) + 1, Predecessor: "final", Label: "post-open", Directory: "post-open"})
			bindingIndex(t, dir, c, p)
		})
	}
}

func TestPhase3B4CampaignRevocation(t *testing.T) {
	for _, name := range []string{"N05", "N06", "N07"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m := bindingFixture(t, ua, ub, 0, nil, false)
			reciprocalRevokeProgram(t, &p, &m.Reciprocal)
			bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			did := false
			if name == "N05" {
				_, q := bindingInitial(t, s, m.Reciprocal)
				bindingGrant(t, s, m.Reciprocal)
				reciprocalCommit(t, s, m.Reciprocal.A, q)
				ready := bindingDeliver(t, s, m.Reciprocal.Ready)
				reciprocalCommit(t, s, m.Reciprocal.A, ready)
				bindingCaptureLive(t, db, filepath.Join(dir, "before-revoke"))
				reciprocalRevoke(t, s, m.Reciprocal, dir)
				bindingCaptureLive(t, db, filepath.Join(dir, "after-revoke"))
				did = true
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
				target := name == "N05" && e.Boundary == "execution.committed" && e.Sequence == 4 || name == "N06" && e.Boundary == "dispatch.consider" && e.Phase == "pre-admission" || name == "N07" && e.Boundary == "authority.admission" && e.Phase == "allowed"
				if !did && target && e.Intent == m.Reciprocal.BAService {
					did = true
					baseline3b3Write(t, dir, "temporal-boundary.json", e)
					bindingCaptureLive(t, db, filepath.Join(dir, "before-revoke"))
					reciprocalRevoke(t, s, m.Reciprocal, dir)
					bindingCaptureLive(t, db, filepath.Join(dir, "after-revoke"))
				}
				return nil
			})
			report, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			s.Close()
			baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
			if runErr != nil || !did {
				t.Fatal(runErr, did)
			}
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			snap := bindingCapture(t, db, filepath.Join(dir, "final"))
			e, _ := snap.Effect(m.Reciprocal.BAService)
			if e == nil || e.ExternalAttempt == nil {
				t.Fatal(e)
			}
			if name == "N05" {
				if e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 0 || e.ExternalReceipt != nil {
					t.Fatal(e)
				}
			} else if !e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 1 {
				t.Fatal(e)
			}
			if name == "N07" {
				bindingComplete(t, snap, m.Reciprocal, endpoints)
			} else if endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 || name == "N06" && e.AuthorityAdmissions[0].Allowed {
				t.Fatal("fresh revoked exposure", e, endpoints)
			}
			raw, _ := json.Marshal(e.ExternalAttempt.Authority)
			baseline3b3Write(t, dir, "historical-authority.json", json.RawMessage(raw))
			bindingIndex(t, dir, bindingCase{Name: name, Schedule: "controlled-public-revocation-at-boundary", Classification: "historical-versus-current-authority", Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "before-revoke", Directory: "before-revoke"}, {Ordinal: 2, Predecessor: "before-revoke", Label: "after-revoke", Directory: "after-revoke"}, {Ordinal: 3, Predecessor: "after-revoke", Label: "final", Directory: "final"}}}, p)
		})
	}
}
