package executor

import (
	"encoding/json"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"testing"
)

// Independent review attack: keep the genuine intent, grant, selection,
// permission and original debit, but coherently replace the raw wire request.
func TestRequestAuthoritySubstitutedWireRequest(t *testing.T) {
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
			s, path, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
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

// Same sender, generation, target, obligation and payload do not make a second
// request the original named by the accepted grant.
func TestRequestAuthorityEqualPayloadRequestSubstitution(t *testing.T) {
	p := requestAuthorityFixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
		u.GrantID = identity("B", int64(1), int64(3), 0)
	})
	authorityReplaceProgram(t, &p, "B", func(p *Program) {
		first := p.Code[0]
		p.Code = append([]Instruction{first, first}, p.Code[1:]...)
	})
	s, path := messageStore(t, p)
	for n := 0; n < 2; n++ {
		if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	d := authorizedDecision(t, s, committedRequest(t, s, "B"))
	if !d.Accepted {
		t.Fatal(d)
	}
	var second AdmissionResult
	for n := int64(1); n <= 2; n++ {
		a, err := s.DeliverMessage(t.Context(), identity("B", int64(1), n, 0), identity("transport", n))
		if err != nil || !a.Accepted {
			t.Fatalf("%+v %v", a, err)
		}
		second = a
	}
	if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), second.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(2), 0))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("original=[B,1,1,0]; selected=[B,1,2,0]; effect allowed=%v reason=%s", a.Allowed, a.Reason)
	if a.Allowed || a.Reason != "authority_request_not_admitted" {
		t.Fatal("equal-payload request substituted")
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
