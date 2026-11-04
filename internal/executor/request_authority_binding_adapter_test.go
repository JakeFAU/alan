//go:build integration

package executor

import (
	"context"
	"encoding/json"
	"github.com/JakeFAU/alan/effects"
	recordschema "github.com/JakeFAU/alan/schema"
	"testing"
	"time"
)

// Attack the derivable AdapterRequest, leaving opaque Wire/Endpoint untouched.
func TestRequestAuthoritySubstitutedAdapterRequest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*effects.Request)
	}{
		{"effect", func(r *effects.Request) { r.Effect = "double" }},
		{"id", func(r *effects.Request) { r.ID = "substituted" }},
		{"key", func(r *effects.Request) { r.Key = "substituted" }},
		{"capability", func(r *effects.Request) { r.Capability = "resource2" }},
		{"payload", func(r *effects.Request) { r.Payload = map[string]recordschema.Value{"extra": Text("value")} }},
		{"input", func(r *effects.Request) {
			r.Input, _ = (recordschema.Record{"extra": {Kind: "string", MaxBytes: 10}}).Retain()
		}},
		{"output", func(r *effects.Request) {
			r.Output, _ = (recordschema.Record{"extra": {Kind: "string", MaxBytes: 10}}).Retain()
		}},
		{"assets", func(r *effects.Request) {
			r.Assets = []effects.Asset{{ID: "extra", Source: "test", Version: "v1", Content: "forged"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := requestAuthorityAdapterFixture(t, "resource1", "increment")
			s, path := messageStore(t, p)
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			d := authorizedDecision(t, s, committedRequest(t, s, "B"))
			if !d.Accepted {
				t.Fatal(d)
			}
			ar, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "request")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ar.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			id := identity("A", int64(1), int64(2), 0)
			registry := map[string]effects.Adapter{"resource1": callbackAdapter{prepare: func(r effects.Request) (effects.Prepared, error) {
				return effects.Prepared{Request: r, Protocol: "binding-review/v1", Endpoint: "fixture://resource1", MaxResponseBytes: 4096}, nil
			}, execute: func(context.Context, effects.Prepared) effects.Candidate {
				t.Error("unexpected contact")
				return effects.Candidate{}
			}}}
			work, err := s.prepareAdapterWork(t.Context(), id, registry, time.Second)
			if err != nil || work == nil {
				t.Fatalf("prepare %v", err)
			}
			a, err := loadAdapterAttempt(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&a.Prepared.Request)
			a.Prepared = effects.Freeze(a.Prepared)
			raw, _ := json.Marshal(a)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE adapter_attempts SET descriptor=? WHERE intent_id=?", raw, id); err != nil {
				t.Fatal(err)
			}
			var r compositionReservation
			var rr []byte
			if err = s.conn.QueryRowContext(t.Context(), "SELECT descriptor FROM composition_reservations WHERE operation_id=?", id).Scan(&rr); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(rr, &r); err != nil {
				t.Fatal(err)
			}
			r.AttemptDigest = compositionIdentity("alan/attempt-row/v1", a)
			rr, _ = json.Marshal(r)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_reservations SET descriptor=? WHERE operation_id=?", rr, id); err != nil {
				t.Fatal(err)
			}
			f, err := compositionProtocolEvidence(t.Context(), s.conn, compositionReservationFact, id)
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
			op, reason, gateErr := s.registerDispatch(t.Context(), id)
			t.Logf("intent operation=%s; prepared request operation=%s; gate opportunity=%v reason=%q error=%v", a.Intent.Obligation, a.Prepared.Request.Effect, op != nil, reason, gateErr)
			if op != nil {
				s.finishDispatch(op, nil)
			}
			_, re := Replay(t.Context(), path, DefaultLimits())
			_, ie := InspectExecution(t.Context(), path, DefaultLimits())
			t.Logf("Replay=%v; InspectExecution=%v", re, ie)
			if re == nil || ie == nil {
				t.Error("offline validator accepted rewritten derived request")
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if op != nil || gateErr == nil {
				t.Error("fresh gate released opportunity for adapter request differing from original effect")
			}
		})
	}

}

func TestRequestAuthorityLegacyAdapterWitness(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		name := "clean"
		if malformed {
			name = "forbidden_witness"
		}
		t.Run(name, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			if _, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second); err != nil {
				t.Fatal(err)
			}
			a, err := loadAdapterAttempt(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			if malformed {
				a.Authority = &EffectAuthority{Version: RequestAuthorityVersion, Namespace: s.Namespace(), Allowed: true, Reason: "authorized"}
				raw, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = s.conn.ExecContext(t.Context(), "UPDATE adapter_attempts SET descriptor=? WHERE intent_id=?", raw, id); err != nil {
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
