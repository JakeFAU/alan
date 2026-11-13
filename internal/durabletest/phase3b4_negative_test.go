//go:build integration

package durabletest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestPhase3B4CampaignNegative(t *testing.T) {
	for _, name := range []string{"N08", "N09a", "N09b", "N09c", "N10a", "N10g", "N10h", "N11b", "N11c-shaped", "N11d", "issuer-generation", "issuer-root", "issuer-revision", "recipient-generation", "recipient-root", "recipient-revision", "other-request", "request-generation", "service-generation"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			n := 0
			var branch *bool
			if name == "N08" || name == "N09b" {
				n = 1
			}
			if name == "N09a" || name == "N09c" {
				v := true
				branch = &v
			}
			p, m := bindingFixture(t, ua, ub, n, branch, false)
			want := ""
			maxSteps := 256
			limits := runtime.DefaultLimits()
			switch name {
			case "N08":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = `["a",1,4,0]` })
				want = "authority_service_mismatch"
			case "N09a":
				reciprocalEdit(t, &p, "a", func(program map[string]any) {
					in := reciprocalInstruction(program, 4)
					for k := range in {
						delete(in, k)
					}
					in["Op"] = "branch"
					in["A"] = reciprocalLiteral(schema.Bool(true))
					in["B"] = reciprocalLiteral(schema.Bool(true))
					in["Target"] = 8
					in["Otherwise"] = 8
				})
				m.Reciprocal.BAService = `["a",1,6,0]`
				want = "authority_service_mismatch"
			case "N09b":
				reciprocalEdit(t, &p, "a", func(program map[string]any) {
					in := reciprocalInstruction(program, 3)
					in["Dst"] = "live_service_target"
					in["A"] = reciprocalLiteral(schema.Text(ua))
					reciprocalInstruction(program, 4)["Effect"].(map[string]any)["Target"] = map[string]any{"Ref": "live_service_target"}
				})
				want = "authority_action_mismatch"
			case "N09c":
				reciprocalEdit(t, &p, "a", func(program map[string]any) {
					in := reciprocalInstruction(program, 3)
					in["Target"] = 14
					in["Otherwise"] = 14
				})
			case "N10a":
				p.Composition.Roots[0].Limits.MaxEffects = 0
				want = "effect_limit"
			case "N10g":
				maxSteps = 3
			case "N10h":
				limits.MaxProgramBytes = len(p.Programs[0].Bytes) - 1
			case "N11b":
				reciprocalEdit(t, &p, "a", func(program map[string]any) {
					delete(reciprocalInstruction(program, 3)["Effect"].(map[string]any), "Authority")
				})
				want = "authority_missing"
			case "N11c-shaped":
				reciprocalUse(t, &p, func(u *runtime.AuthorityUse) { u.GrantID = `["lifecycle-outcome","[\"b\",1,2,0]"]` })
				want = "authority_grant_missing"
			case "N11d":
				own := m.Reciprocal.BAServiceRoot.Permission
				own.Source = "a"
				p.Permissions = append(p.Permissions, own)
				reciprocalUse(t, &p, func(u *runtime.AuthorityUse) { u.GrantID = m.Reciprocal.BARequest })
				want = "authority_grant_missing"
			case "issuer-generation":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.Root.Run.Generation = 2 })
			case "issuer-root":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.Root.RootRun = m.Reciprocal.A })
			case "issuer-revision":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.Root.Permission.Revision = 2 })
			case "recipient-generation":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.Recipient.Generation = 2 })
			case "recipient-root":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.RecipientRoot = m.Reciprocal.B })
			case "recipient-revision":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.RecipientRevision = 2 })
			case "other-request":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.OriginalRequestID = m.Reciprocal.ABRequest })
			case "request-generation":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.OriginalRequestID = `["b",2,1,0]` })
			case "service-generation":
				reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = `["a",2,4,0]` })
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			baseline3b3Write(t, dir, "limits.json", limits)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, limits)
			baseline3b3Write(t, dir, "create.json", map[string]any{"Error": fmt.Sprint(err), "ProgramBytes": []int{len(p.Programs[0].Bytes), len(p.Programs[1].Bytes)}})
			if name == "N10h" {
				if !errors.Is(err, runtime.ErrLimit) {
					t.Fatal("wrong byte boundary", err)
				}
				bindingEndpoints(t, dir, pa, pb, ca, cb)
				bindingIndex(t, dir, bindingCase{Name: name, Schedule: "public-create", Classification: "program-byte-limit", Manifest: m}, p)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			report, runErr := bindingRun(t, s, dir, maxSteps)
			s.Close()
			if name == "N10g" {
				if !errors.Is(runErr, runtime.ErrLimit) || report.Steps != 3 {
					t.Fatal("wrong step boundary", report, runErr)
				}
			} else if runErr != nil {
				t.Fatal(runErr)
			}
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			snap := bindingCapture(t, db, filepath.Join(dir, "final"))
			if endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
				t.Fatal("negative exposed", endpoints)
			}
			e, _ := snap.Effect(m.Reciprocal.BAService)
			classification := want
			if want != "" {
				if e == nil || e.ExternalAttempt == nil || e.ExternalAttempt.Allowed || e.ExternalAttempt.Reason != want || e.ExternalReceipt != nil || len(e.AuthorityAdmissions) != 0 {
					t.Fatalf("wrong layer want %s: %+v", want, e)
				}
				if name == "N10a" && !e.ExternalAttempt.Authority.Allowed {
					t.Fatal("resource denial conflated with permission", e)
				}
			} else if name == "N09c" {
				if len(snap.Effects) != 0 {
					t.Fatal("skipped service appeared")
				}
				classification = "service-skipped-unpaid"
			} else if name == "N10g" {
				if len(snap.Effects) != 0 {
					t.Fatal("service beyond budget")
				}
				classification = "run-step-limit"
			} else {
				d, ok := snap.LifecycleDecision(m.Reciprocal.Grant)
				if !ok || d.Accepted || len(snap.Effects) != 0 {
					t.Fatal("false grant escaped", d)
				}
				classification = d.Reason
			}
			baseline3b3Write(t, dir, "causal-assertions.json", map[string]any{"Service": e, "Lifecycle": snap.LifecycleDecisions, "Classification": classification, "Pending": snap.Pending})
			bindingIndex(t, dir, bindingCase{Name: name, Schedule: "automatic-run-negative-public-input", Classification: classification, Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}}}, p)
		})
	}
}

// These controls use a genuine lifecycle emitter, not a changed subject row.
func TestPhase3B4CampaignLifecycle(t *testing.T) {
	for _, name := range []string{"N10c", "N10d", "N10e-holder", "N10e-recipient"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m := bindingFixture(t, ua, ub, 0, nil, false)
			operation, target := "CancelRun", "a"
			if name == "N10d" {
				operation = "RetireSubject"
			}
			if name == "N10e-holder" {
				target = "b"
			}
			emitter := "b"
			if target == "b" {
				emitter = "a"
			}
			// The emitter appends two explicit lifecycle instructions at a public
			// waiting PC after the historical selected request and grant exist.
			insert := 5
			if emitter == "a" {
				insert = 4
			}
			makeEmit := func(op string) any {
				return map[string]any{"Op": "emit", "Dst": "lifecycle_control", "Effect": map[string]any{"Kind": "lifecycle", "Target": reciprocalLiteral(schema.Text(target)), "Generation": reciprocalLiteral(schema.Int(1)), "Obligation": reciprocalLiteral(schema.Text("lifecycle-control")), "Payload": map[string]any{"operation": reciprocalLiteral(schema.Text(op)), "revision": reciprocalLiteral(schema.Int(1))}}}
			}
			reciprocalEdit(t, &p, emitter, func(program map[string]any) {
				code := program["Code"].([]any)
				program["Code"] = append(append([]any{}, code[:insert]...), makeEmit(operation), makeEmit("Retry"), reciprocalInput("hold-control"))
				if emitter == "b" {
					reciprocalInstruction(program, 3)["Otherwise"] = insert + 2
				}
			})
			for _, op := range []string{operation, "Retry"} {
				p.Permissions = append(p.Permissions, runtime.Permission{Kind: "lifecycle", Source: emitter, Target: target, Operation: op, Revision: 1})
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
			bindingGrant(t, s, m.Reciprocal)
			reciprocalCommit(t, s, m.Reciprocal.A, q)
			ready := bindingDeliver(t, s, m.Reciprocal.Ready)
			reciprocalCommit(t, s, m.Reciprocal.A, ready)
			reciprocalCommit(t, s, m.Reciprocal.A, "")
			before := bindingCaptureLive(t, db, filepath.Join(dir, "before-close"))
			qBefore := bindingSelected(t, before, m.Reciprocal.A, m.Reciprocal.BARequest)
			run := m.Reciprocal.B
			if emitter == "a" {
				run = m.Reciprocal.A
			}
			decide := func(sequence int) runtime.LifecycleResult {
				reciprocalCommit(t, s, run, "")
				raw, _ := json.Marshal([]any{emitter, 1, sequence, 0})
				id := string(raw)
				auth, err := s.AuthorizeIntent(t.Context(), id)
				if err != nil || !auth.Allowed {
					t.Fatal(auth, err)
				}
				r, err := s.LoadLifecycleRequest(t.Context(), id)
				if err != nil {
					t.Fatal(err)
				}
				d, err := s.DecideLifecycle(t.Context(), r)
				if err != nil {
					t.Fatal(err)
				}
				return d
			}
			closed := decide(insert + 1)
			if !closed.Accepted {
				t.Fatal(closed)
			}
			if name == "N10e-holder" || name == "N10e-recipient" {
				retry := decide(insert + 2)
				if retry.Accepted || retry.Reason != "invocation_retry_unsupported" {
					t.Fatal("unexpected generation writer", retry)
				}
				baseline3b3Write(t, dir, "construction-obstruction.json", map[string]any{"Claim": "changed generation before fresh gate", "ActualRetry": retry, "Status": "unsupported-public-transition"})
			}
			if name != "N10e-holder" && name != "N10e-recipient" {
				bindingFirstDecision(t, s, dir)
			}
			s.Close()
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			after := bindingCapture(t, db, filepath.Join(dir, "final"))
			qAfter := bindingSelected(t, after, m.Reciprocal.A, m.Reciprocal.BARequest)
			if !reflectObservation(qBefore, qAfter) || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
				t.Fatal("historical selection/exposure", qBefore, qAfter, endpoints)
			}
			e, _ := after.Effect(m.Reciprocal.BAService)
			classification := "generation-transition-obstructed"
			if name == "N10c" || name == "N10d" {
				if e == nil || e.ExternalAttempt == nil || e.ExternalAttempt.Allowed {
					t.Fatal(e)
				}
				classification = e.ExternalAttempt.Reason
			} else if e == nil || e.ExternalAttempt != nil {
				t.Fatal("obstruction was dispatched", e)
			}
			bindingIndex(t, dir, bindingCase{Name: name, Schedule: "controlled-public-lifecycle", Classification: classification, Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "before-close", Directory: "before-close"}, {Ordinal: 2, Predecessor: "before-close", Label: "final", Directory: "final"}}}, p)
		})
	}
}
func reflectObservation(a, b *runtime.ObservationView) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}
