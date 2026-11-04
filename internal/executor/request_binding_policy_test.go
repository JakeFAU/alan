package executor

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	resource "github.com/JakeFAU/alan/internal/fixture"
)

func TestRequestBindingStickyRefusal(t *testing.T) {
	s, path, req, before := authorityReady(t, bindingV2Fixture(t, nil), false)
	if before.Allowed || before.Authority.Reason != "authority_grant_missing" {
		t.Fatalf("%+v", before)
	}
	d := authorizedDecision(t, s, req)
	if !d.Accepted {
		t.Fatal(d)
	}
	after, _, err := s.prepareExternal(t.Context(), before.Intent.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("refusal changed %v", err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, _, err = reopened.prepareExternal(t.Context(), before.Intent.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("reopen repaired refusal %v", err)
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestRequestBindingRevocationOrderingAndUnaffectedTuple(t *testing.T) {
	p := bindingV2Fixture(t, nil)
	var root AuthorityRoot
	authorityReplaceProgram(t, &p, "A", func(p *Program) { root = p.Code[1].Effect.Authority.Root })
	raw, _ := json.Marshal(RevocationClaim{Version: RequestAuthorityBindingVersion, Root: root})
	other := root
	other.Permission.Obligation = "r"
	p.Permissions = append(p.Permissions, other.Permission)
	authorityReplaceProgram(t, &p, "B", func(p *Program) {
		p.Code = append(p.Code[:2], Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("B")), Generation: Lit(Int(1)), Obligation: Lit(Text("revoke")), Payload: map[string]Operand{"operation": Lit(Text("RevokeAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(raw)))}}}, Instruction{Op: "emit", Effect: &Effect{Kind: "external", Target: Lit(Text(other.Permission.Target)), Generation: Lit(Int(1)), Obligation: Lit(Text("r")), Authority: &AuthorityUse{Version: RequestAuthorityBindingVersion, Path: "own", Root: other}, Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(1))}}}, Instruction{Op: "end", Outcome: "succeeded"})
	})
	s, path, _, a := authorityReady(t, p, true)
	if !a.Allowed {
		t.Fatal(a)
	}
	op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || reason != "" {
		t.Fatalf("%s %v", reason, err)
	}
	s.finishDispatch(op, nil)
	revoked := authorizedDecision(t, s, committedRequest(t, s, "B"))
	if !revoked.Accepted {
		t.Fatal(revoked)
	}
	op, reason, err = s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || op != nil || reason != "authority_revoked" {
		t.Fatalf("revoke gate %s %v", reason, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	r, _, err := s.prepareExternal(t.Context(), identity("B", int64(1), int64(4), 0))
	if err != nil || !r.Allowed {
		t.Fatalf("unrelated tuple %+v %v", r, err)
	}
	op, reason, err = s.registerDispatch(t.Context(), r.Intent.ID)
	if err != nil || op == nil || reason != "" {
		t.Fatalf("unrelated gate %s %v", reason, err)
	}
	s.finishDispatch(op, nil)
	old, err := loadExternalAttempt(t.Context(), s.conn, a.Intent.ID)
	if err != nil || !reflect.DeepEqual(old, a) {
		t.Fatal("revocation changed first allow")
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
	defer reopened.Close()
	op, reason, err = reopened.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || op != nil || reason != "authority_revoked" {
		t.Fatalf("reopen gate %s %v", reason, err)
	}
}
func TestRequestBindingMissingSelectorAndNoFallback(t *testing.T) {
	for _, tc := range []struct {
		name   string
		edit   func(*Program)
		reason string
	}{
		{"missing", func(p *Program) { p.Code[1].Effect.Authority = nil }, "authority_missing"},
		{"own_cannot_borrow", func(p *Program) { p.Code[1].Effect.Authority.Path = "own"; p.Code[1].Effect.Authority.GrantID = "" }, "authority_owner_mismatch"},
		{"tuple_mix", func(p *Program) { p.Code[1].Effect.Obligation = Lit(Text("other")) }, "authority_action_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := bindingV2Fixture(t, nil)
			authorityReplaceProgram(t, &p, "A", tc.edit)
			s, path, _, a := authorityReady(t, p, true)
			if a.Allowed || a.Authority.Reason != tc.reason {
				t.Fatalf("%+v", a)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if count(t, s, "composition_charges") != 0 {
				t.Fatal("refusal charged")
			}
		})
	}
}
func TestRequestBindingGateRequiresOriginalDebit(t *testing.T) {
	s, _, _, a := authorityReady(t, bindingV2Fixture(t, nil), true)
	if _, err := s.conn.ExecContext(t.Context(), "DELETE FROM composition_charges WHERE operation_id=?", a.Intent.ID); err != nil {
		t.Fatal(err)
	}
	op, _, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if op != nil {
		s.finishDispatch(op, nil)
	}
	if err == nil {
		t.Fatal("gate accepted reservation without original debit")
	}
}

func TestRequestBindingInvalidSelectionDoesNotFallback(t *testing.T) {
	p := bindingV2Fixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
		own := d.Root.Permission
		own.Source = "A"
		p.Permissions = append(p.Permissions, own)
		u.GrantID = "not-a-grant"
	})
	_, _, _, a := authorityReady(t, p, true)
	if a.Allowed || a.Authority.Reason != "authority_grant_missing" {
		t.Fatalf("ambient fallback: %+v", a)
	}
}

func TestRequestBindingDelegation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mutate        func(*DelegationClaim, *AuthorityUse, *Provision)
		grant, effect bool
	}{
		{name: "exact", grant: true, effect: true},
		{name: "missing_original_permission", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { p.Permissions = p.Permissions[1:] }},
		{name: "wrong_root_generation", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.Root.RootRun.Generation = 2
			u.Root = c.Root
		}},
		{name: "wrong_holder_generation", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { c.Root.Run.Generation = 2; u.Root = c.Root }},
		{name: "wrong_revision", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.Root.Permission.Revision = 2
			u.Root = c.Root
		}},
		{name: "wrong_recipient_generation", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { c.Recipient.Generation = 2 }},
		{name: "wrong_recipient", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { c.Recipient.Subject = "B" }},
		{name: "wrong_issuer", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { c.Root.Run.Subject = "A"; u.Root = c.Root }},
		{name: "wrong_holder_root", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.Root.RootRun = RunID{"A", 1}
			u.Root = c.Root
		}},
		{name: "nonexistent_root", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.Root.RootRun = RunID{"missing", 1}
			u.Root = c.Root
		}},
		{name: "wrong_recipient_root", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) { c.RecipientRoot = RunID{"B", 1} }},
		{name: "wrong_request", mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.OriginalRequestID = identity("B", int64(1), int64(10), 0)
		}},
		{name: "wrong_service", grant: true, mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			c.ServiceOperationID = identity("A", int64(1), int64(3), 0)
		}},
		{name: "result_as_grant", grant: true, mutate: func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
			u.GrantID = identity("lifecycle-outcome", u.GrantID)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := bindingV2Fixture(t, tc.mutate)
			path := filepath.Join(t.TempDir(), "authority.sqlite")
			s, err := Create(t.Context(), path, p, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			req := committedRequest(t, s, "B")
			d := authorizedDecision(t, s, req)
			if d.Accepted != tc.grant {
				t.Fatalf("grant=%+v", d)
			}
			ar, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "request")
			if err != nil {
				t.Fatal(err)
			}
			// A's actual receive always selects the genuine request, independent of a false claim.
			if tc.name == "wrong_request" {
				return
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ar.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(2), 0))
			if err != nil {
				t.Fatal(err)
			}
			if a.Allowed != tc.effect || a.Authority == nil {
				t.Fatalf("effect=%+v", a)
			}
			if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestBindingSubstitutedWireRequest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*resource.Request)
	}{
		{"delta", func(r *resource.Request) { r.Delta = 999 }},
		{"profile", func(r *resource.Request) { r.Profile = "non-idempotent" }},
		{"key", func(r *resource.Request) { r.Key = "substituted" }},
		{"epoch", func(r *resource.Request) { r.Epoch++ }},
		{"authorization", func(r *resource.Request) { r.Authorization = "substituted" }},
		{"subject", func(r *resource.Request) { r.Subject = "B" }},
		{"generation", func(r *resource.Request) { r.Generation++ }},
		{"sequence", func(r *resource.Request) { r.Sequence++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path, _, a := authorityReady(t, bindingV2Fixture(t, nil), true)
			tc.mutate(&a.Request)
			raw, _ := json.Marshal(a)
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE external_attempts SET descriptor=? WHERE intent_id=?", raw, a.Intent.ID); err != nil {
				t.Fatal(err)
			}
			var r compositionReservation
			var rr []byte
			if err := s.conn.QueryRowContext(t.Context(), "SELECT descriptor FROM composition_reservations WHERE operation_id=?", a.Intent.ID).Scan(&rr); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(rr, &r); err != nil {
				t.Fatal(err)
			}
			r.AttemptDigest = compositionIdentity("alan/attempt-row/v1", a)
			rr, _ = json.Marshal(r)
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE composition_reservations SET descriptor=? WHERE operation_id=?", rr, a.Intent.ID); err != nil {
				t.Fatal(err)
			}
			f, err := compositionProtocolEvidence(t.Context(), s.conn, compositionReservationFact, a.Intent.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.conn.QueryRowContext(t.Context(), "SELECT decision_order FROM composition_facts WHERE id=?", f.ID).Scan(&f.Order); err != nil {
				t.Fatal(err)
			}
			fr, _ := json.Marshal(f)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_facts SET descriptor=? WHERE id=?", fr, f.ID); err != nil {
				t.Fatal(err)
			}
			work, err := s.prepareExternalWork(t.Context(), a.Intent.ID, DefaultExternalTimeout)
			if err != nil {
				t.Fatalf("prepare: %v", err)
			}
			if tc.name == "profile" {
				// Existing non-idempotent recovery suppresses preparation. Probe
				// the gate directly too; no physical work is performed here.
				if work != nil {
					t.Fatal("non-idempotent unknown unexpectedly prepared")
				}
			} else if work == nil {
				t.Fatal("expected retained idempotent work")
			}
			t.Logf("immutable request delta=%d; retained request delta=%d", a.Intent.Payload["delta"].Number, a.Request.Delta)
			op, reason, gateErr := s.registerDispatch(t.Context(), a.Intent.ID)
			t.Logf("gate opportunity=%v reason=%q error=%v", op != nil, reason, gateErr)
			if op != nil {
				s.finishDispatch(op, nil)
			}
			_, replayErr := Replay(t.Context(), path, DefaultLimits())
			_, inspectErr := InspectExecution(t.Context(), path, DefaultLimits())
			t.Logf("Replay=%v; InspectExecution=%v", replayErr, inspectErr)
			if replayErr == nil || inspectErr == nil {
				t.Error("offline validator accepted rewritten request")
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if op != nil || gateErr == nil {
				t.Error("fresh gate released opportunity for raw request differing from the original intent")
			}
		})
	}

}

func TestRequestBindingCompletedDelegatorBeforeService(t *testing.T) {
	s, path := messageStore(t, bindingV2Fixture(t, nil))
	if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
		t.Fatal(d)
	}
	admission, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "before-completion")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if current(t, s, "B").Projection.Status != "succeeded" {
		t.Fatal("holder not complete")
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(2), 0))
	if err != nil || !a.Allowed {
		t.Fatal(a.Authority, err)
	}
	op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || op == nil || reason != "" {
		t.Fatal(reason, err)
	}
	s.finishDispatch(op, nil)
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
