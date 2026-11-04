//go:build integration

package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	recordschema "github.com/JakeFAU/alan/schema"
)

func TestRequestBindingAuthoredConflictFormats(t *testing.T) {
	for _, version := range []string{RequestAuthorityVersion, RequestAuthorityBindingVersion} {
		t.Run(version, func(t *testing.T) {
			p := requestAuthorityAdapterFixture(t, "resource1", "increment")
			p.Composition.RequestAuthority = version
			authorityReplaceProgram(t, &p, "B", func(p *Program) {
				c, _, err := parseAuthorityClaim(LifecycleRequest{Intent: Intent{Payload: map[string]Value{"operation": Text("DelegateAuthority"), "revision": Int(1), "claim": *p.Code[1].Effect.Payload["claim"].Literal}}, Operation: "DelegateAuthority"})
				if err != nil {
					t.Fatal(err)
				}
				c.Version = version
				raw, _ := json.Marshal(c)
				p.Code[1].Effect.Payload["claim"] = Lit(Text(string(raw)))
			})
			authorityReplaceProgram(t, &p, "A", func(p *Program) {
				p.Code[1].Effect.Authority.Version = version
				p.Code[1].Dst = "pending"
				result, _ := recordschema.ResultRecord(recordschema.Record{})
				retained, _ := result.Retain()
				p.Code = append(p.Code[:2], Instruction{Op: "receive", Match: &Match{Kind: "adapter", Provenance: "runtime-adapter", Source: Lit(Text("resource1")), Generation: Lit(Int(1)), Obligation: Lit(Text("increment")), Operation: Ref("pending")}, Bind: map[string]string{"valid": "valid"}}, p.Code[2])
				p.Author.Positions = []string{"request", "effect", "result", "end"}
				p.Author.Receives[2] = retained
			})
			s, path := messageStore(t, p)
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
				t.Fatal(d)
			}
			admission, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "request")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			id := identity("A", int64(1), int64(2), 0)
			contacts := 0
			registry := map[string]effects.Adapter{"resource1": callbackAdapter{execute: func(_ context.Context, p effects.Prepared) effects.Candidate {
				contacts++
				return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]Value{}}
			}}}
			work, err := s.prepareAdapterWork(t.Context(), id, registry, time.Second)
			if err != nil || work == nil {
				t.Fatal(err)
			}
			op, reason, err := s.registerDispatch(t.Context(), id)
			if err != nil || reason != "" {
				t.Fatal(reason, err)
			}
			completion := transportAdapter(t.Context(), *work)
			if err = s.accountAdapter(t.Context(), completion); err != nil {
				t.Fatal(err)
			}
			s.finishDispatch(op, nil)
			attempt, err := loadAdapterAttempt(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			first, err := loadAdapterResult(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			for _, conflict := range []bool{false, true} {
				if conflict {
					other := first.Candidate
					other.Response = "distinct valid retained evidence"
					if _, err = s.recordAdapterResult(t.Context(), attempt, other); !errors.Is(err, ErrConflict) {
						t.Fatal(err)
					}
				}
				want := 13
				if version == RequestAuthorityBindingVersion {
					want = 15
				}
				if conflict {
					want++
				}
				got, err := storedSchemaNumber(t.Context(), s.conn)
				if err != nil || got != want {
					t.Fatal(got, want, err)
				}
				if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
				if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
				s.Close()
				s, err = Open(t.Context(), path, DefaultLimits())
				if err != nil {
					t.Fatal(fmt.Sprint(want), err)
				}
			}
			defer s.Close()
			observation := observationFor(adapterDescriptor(attempt, first), attempt.Intent.Run)
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), observation.ID); err != nil {
				t.Fatal(err)
			}
			if !current(t, s, "A").Projection.Data["valid"].Flag || contacts != 1 {
				t.Fatal("retained original result lost", contacts)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
