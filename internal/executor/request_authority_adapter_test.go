//go:build integration

package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	recordschema "github.com/JakeFAU/alan/schema"
)

func requestAuthorityAdapterFixture(t *testing.T, capability, operation string) Provision {
	t.Helper()
	p := requestAuthorityFixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
		d.Root.Permission.Kind = "adapter"
		d.Root.Permission.Target = "resource1"
		d.Root.Permission.Operation = "increment"
		d.Root.Permission.Obligation = "increment"
		u.Root = d.Root
		p.Permissions[0] = d.Root.Permission
		p.Permissions = append(p.Permissions, Permission{Kind: "adapter", Source: "B", Target: "resource2", Operation: "double", Revision: 1, Obligation: "double"})
	})
	authorityReplaceProgram(t, &p, "A", func(p *Program) {
		empty, _ := recordschema.Record{}.Retain()
		p.Semantics = AuthorSemanticVersion
		p.Code[1].Effect.Kind = "adapter"
		p.Code[1].Effect.Target = Lit(Text(capability))
		p.Code[1].Effect.Obligation = Lit(Text(operation))
		p.Code[1].Effect.Payload = map[string]Operand{}
		p.Author = &ProgramMetadata{Compiler: AuthorCompilerVersion, Inputs: empty, Outputs: empty, Positions: []string{"request", "effect", "end"}, Effects: map[string]EffectDeclaration{operation: {Capability: capability, Input: empty, Output: empty}}, Ports: map[string]PortDeclaration{"request": {Peer: "B", Obligation: "request", Send: empty, Receive: empty, ReceiveUsed: true}}, Receives: map[int]recordschema.Retained{0: empty}}
	})
	p.Composition.Roots[0].Capabilities = []string{"resource1", "resource2"}
	return p
}

func TestRequestAuthorityAdapterCartesianAndUnknown(t *testing.T) {
	for _, tc := range []struct {
		name, resource, op string
		allowed, unknown   bool
	}{
		{"exact", "resource1", "increment", true, false},
		{"cross_resource1_double", "resource1", "double", false, false},
		{"cross_resource2_increment", "resource2", "increment", false, false},
		{"generic_unknown_no_retry", "resource1", "increment", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := requestAuthorityAdapterFixture(t, tc.resource, tc.op)
			path := filepath.Join(t.TempDir(), "adapter.sqlite")
			s, err := Create(t.Context(), path, p, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if s != nil {
					s.Close()
				}
			}()
			if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			d := authorizedDecision(t, s, committedRequest(t, s, "B"))
			if !d.Accepted {
				t.Fatal(d)
			}
			admission, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "request")
			if err != nil || !admission.Accepted {
				t.Fatalf("%+v %v", admission, err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			// Each registry binding fixes one actual mutable resource. Operations perform
			// different state transitions; payload strings cannot redirect either binding.
			values := map[string]int{"resource1": 3, "resource2": 7}
			contacts := 0
			registry := map[string]effects.Adapter{}
			for _, capability := range []string{"resource1", "resource2"} {
				registry[capability] = callbackAdapter{prepare: func(r effects.Request) (effects.Prepared, error) {
					return effects.Prepared{Request: r, Protocol: "authority-fixture/v1", Endpoint: "fixture://" + capability, MaxResponseBytes: 4096}, nil
				}, execute: func(_ context.Context, p effects.Prepared) effects.Candidate {
					contacts++
					switch p.Request.Effect {
					case "increment":
						values[capability]++
					case "double":
						values[capability] *= 2
					default:
						t.Error("unknown operation")
					}
					return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]Value{}}
				}}
			}
			id := identity("A", int64(1), int64(2), 0)
			work, err := s.prepareAdapterWork(t.Context(), id, registry, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			a, err := loadAdapterAttempt(t.Context(), s.conn, id)
			if err != nil || a.Allowed != tc.allowed {
				t.Fatalf("%+v %v", a, err)
			}
			if tc.unknown {
				if work == nil {
					t.Fatal("no initial opportunity")
				}
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				s, err = Open(t.Context(), path, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				work, err = s.prepareAdapterWork(t.Context(), id, registry, time.Second)
				if err != nil || work != nil || contacts != 0 {
					t.Fatalf("unknown adapter retried: %v", err)
				}
			} else if tc.allowed {
				op, reason, err := s.registerDispatch(t.Context(), id)
				if err != nil || reason != "" {
					t.Fatal(err)
				}
				result := transportAdapter(t.Context(), *work)
				if err = s.accountAdapter(t.Context(), result); err != nil {
					t.Fatal(err)
				}
				s.finishDispatch(op, nil)
				if contacts != 1 || values["resource1"] != 4 || values["resource2"] != 7 {
					t.Fatalf("actual resources %v contacts %d", values, contacts)
				}
			} else if work != nil || contacts != 0 {
				t.Fatal("cross-pair contacted resource")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			snap, err := InspectExecution(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if tc.unknown && (len(snap.Effects) != 1 || snap.Effects[0].State != EffectReservedUnknown) {
				raw, _ := json.Marshal(snap.Effects)
				t.Fatalf("unknown lost %s", raw)
			}
		})
	}
}
