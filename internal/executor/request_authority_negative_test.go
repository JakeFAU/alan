package executor

import (
	"encoding/json"
	"errors"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"path/filepath"
	"reflect"
	"testing"
)

func authorityReplaceProgram(t *testing.T, p *Provision, subject string, edit func(*Program)) {
	t.Helper()
	for n, r := range p.Runs {
		if r.ID.Subject == subject {
			for j, a := range p.Programs {
				if a.ID == r.ProgramID {
					var program Program
					if err := decode(a.Bytes, &program); err != nil {
						t.Fatal(err)
					}
					edit(&program)
					raw, id, err := EncodeProgram(program)
					if err != nil {
						t.Fatal(err)
					}
					p.Programs[j] = Artifact{ID: id, Semantics: program.Semantics, Bytes: raw}
					p.Runs[n].ProgramID = id
					return
				}
			}
		}
	}
	t.Fatal("program missing")
}
func authorityReady(t *testing.T, p Provision, grantFirst bool) (*Store, string, LifecycleRequest, ExternalAttempt) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "authority.sqlite")
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	req := committedRequest(t, s, "B")
	if grantFirst {
		authorizedDecision(t, s, req)
	}
	a, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "request")
	if err != nil || !a.Accepted {
		t.Fatalf("%+v %v", a, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), a.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	attempt, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(2), 0))
	if err != nil {
		t.Fatal(err)
	}
	return s, path, req, attempt
}
func TestRequestAuthorityStickyRefusal(t *testing.T) {
	s, path, req, before := authorityReady(t, requestAuthorityFixture(t, nil), false)
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
func TestRequestAuthorityRevocationOrderingAndUnaffectedTuple(t *testing.T) {
	p := requestAuthorityFixture(t, nil)
	var root AuthorityRoot
	authorityReplaceProgram(t, &p, "A", func(p *Program) { root = p.Code[1].Effect.Authority.Root })
	raw, _ := json.Marshal(RevocationClaim{Version: RequestAuthorityVersion, Root: root})
	other := root
	other.Permission.Obligation = "r"
	p.Permissions = append(p.Permissions, other.Permission)
	authorityReplaceProgram(t, &p, "B", func(p *Program) {
		p.Code = append(p.Code[:2], Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("B")), Generation: Lit(Int(1)), Obligation: Lit(Text("revoke")), Payload: map[string]Operand{"operation": Lit(Text("RevokeAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(raw)))}}}, Instruction{Op: "emit", Effect: &Effect{Kind: "external", Target: Lit(Text(other.Permission.Target)), Generation: Lit(Int(1)), Obligation: Lit(Text("r")), Authority: &AuthorityUse{Version: RequestAuthorityVersion, Path: "own", Root: other}, Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(1))}}}, Instruction{Op: "end", Outcome: "succeeded"})
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
func TestRequestAuthorityForgedLifecycleRequest(t *testing.T) {
	p := requestAuthorityFixture(t, nil)
	s, _ := messageStore(t, p)
	if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	req := committedRequest(t, s, "B")
	if _, err := s.AuthorizeIntent(t.Context(), req.Intent.ID); err != nil {
		t.Fatal(err)
	}
	forged := req
	forged.SourceRevision++
	if _, err := s.DecideLifecycle(t.Context(), forged); !errors.Is(err, ErrConflict) {
		t.Fatalf("forged request: %v", err)
	}
	if count(t, s, "lifecycle_results") != 0 {
		t.Fatal("forged request retained")
	}
	d, err := s.DecideLifecycle(t.Context(), req)
	if err != nil || !d.Accepted {
		t.Fatalf("%+v %v", d, err)
	}
}
func TestRequestAuthorityCorruptPrimaryGate(t *testing.T) {
	for _, mutation := range []string{"digest", "epoch", "ordinal", "allowed", "root", "namespace", "order"} {
		t.Run(mutation, func(t *testing.T) {
			s, path, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
			op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
			if err != nil || reason != "" {
				t.Fatal(err)
			}
			s.finishDispatch(op, nil)
			facts, err := readCompositionFacts(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			f := facts[len(facts)-1]
			switch mutation {
			case "digest":
				f.AuthorityAdmission.AttemptDigest = "forged"
			case "epoch":
				f.AuthorityAdmission.Source.Epoch++
			case "ordinal":
				f.AuthorityAdmission.Ordinal++
			case "allowed":
				f.AuthorityAdmission.Allowed = false
			case "root":
				f.RootIDs = f.RootIDs[:1]
			case "namespace":
				f.AuthorityAdmission.Namespace = "other"
			case "order":
				f.Order++
			}
			// Recompute the digest: encoding integrity alone must not prove policy.
			f.RecordDigest = compositionIdentity("alan/authority-admission/v1", f.AuthorityAdmission)
			raw, _ := json.Marshal(f)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_facts SET descriptor=? WHERE id=?", raw, f.ID); err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("Replay accepted forged gate")
			}
			if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("inspection accepted forged gate")
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(t.Context(), path, DefaultLimits())
			if opened != nil {
				opened.Close()
			}
			if err == nil {
				t.Fatal("Open accepted forged gate")
			}
		})
	}
}

func TestRequestAuthorityMissingSelectorAndNoFallback(t *testing.T) {
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
			p := requestAuthorityFixture(t, nil)
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
func TestRequestAuthorityDelegatorCompletionRetainsConsent(t *testing.T) {
	s, path, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
	if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || op == nil || reason != "" {
		t.Fatalf("completion revoked consent: %s %v", reason, err)
	}
	s.finishDispatch(op, nil)
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestRequestAuthorityFormatsRejectUnsupportedAndLegacy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Provision)
	}{
		{"unknown", func(p *Provision) { p.Composition.RequestAuthority = "future" }},
		{"legacy", func(p *Provision) { p.Composition.RequestAuthority = "" }},
		{"dispatch_journal", func(p *Provision) { p.Composition.RecordDispatchAdmissions = true }},
		{"extra_principal", func(p *Provision) {
			addProgram(t, p, "C", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := requestAuthorityFixture(t, nil)
			tc.mutate(&p)
			s, err := Create(t.Context(), filepath.Join(t.TempDir(), "invalid.sqlite"), p, DefaultLimits())
			if s != nil {
				s.Close()
			}
			if err == nil {
				t.Fatal("unsupported profile accepted")
			}
		})
	}
}
func TestRequestAuthorityCanonicalClaims(t *testing.T) {
	for _, mutate := range []func(string) string{
		func(raw string) string { return " " + raw },
		func(raw string) string { return raw[:len(raw)-1] + `,"Version":"alan/request-authority/v1"}` },
		func(raw string) string { return raw[:len(raw)-1] + `,"Unknown":true}` },
	} {
		p := requestAuthorityFixture(t, nil)
		authorityReplaceProgram(t, &p, "B", func(p *Program) {
			v := p.Code[1].Effect.Payload["claim"]
			v.Literal = &Value{Kind: "string", Text: mutate(v.Literal.Text)}
			p.Code[1].Effect.Payload["claim"] = v
		})
		s, _ := messageStore(t, p)
		if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err == nil {
			t.Fatal("malformed claim committed")
		}
	}
}

func TestRequestAuthorityImmediateSelection(t *testing.T) {
	for _, gap := range []bool{false, true} {
		name := "unselected"
		if gap {
			name = "intervening_reduction"
		}
		t.Run(name, func(t *testing.T) {
			p := requestAuthorityFixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
				if gap {
					d.ServiceOperationID = identity("A", int64(1), int64(3), 0)
				}
			})
			authorityReplaceProgram(t, &p, "A", func(p *Program) {
				step := Instruction{Op: "set", Dst: "unrelated", A: Lit(Int(1))}
				if gap {
					p.Code = append(p.Code[:1], append([]Instruction{step}, p.Code[1:]...)...)
				} else {
					p.Code[0] = step
				}
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
			selected := ""
			if gap {
				selected = admission.ObservationID
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), selected); err != nil {
				t.Fatal(err)
			}
			if gap {
				if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			state, err := s.Commit(t.Context(), current(t, s, "A").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), state.Sequence, 0))
			if err != nil || a.Allowed || a.Authority.Reason != "authority_request_not_selected" {
				t.Fatalf("%+v %v", a, err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestAuthorityResultRequiresRetainedGate(t *testing.T) {
	s, path, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
	receipt := resourcefixture.Receipt{Key: a.Request.Key, Profile: a.Request.Profile, Delta: a.Request.Delta, Mutation: 1, Total: 1, Process: "controlled-test", PID: 1}
	if err := s.recordExternalReceipt(t.Context(), a, receipt); err == nil {
		t.Fatal("result without gate retained")
	}
	op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
	if err != nil || reason != "" {
		t.Fatal(err)
	}
	s.finishDispatch(op, nil)
	if err = s.recordExternalReceipt(t.Context(), a, receipt); err != nil {
		t.Fatal(err)
	}
	// Remove the earlier gate and close the resulting global-order gap, including
	// the retained result's fact ordinal. A mere count/digest check is insufficient.
	facts, err := readCompositionFacts(t.Context(), s.conn)
	if err != nil {
		t.Fatal(err)
	}
	var deleted int64
	for _, f := range facts {
		if f.Kind == compositionAuthorityAdmissionFact {
			deleted = f.Order
			if _, err = s.conn.ExecContext(t.Context(), "DELETE FROM composition_facts WHERE id=?", f.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, f := range facts {
		if f.Order > deleted {
			f.Order--
			raw, _ := json.Marshal(f)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_facts SET decision_order=?,descriptor=? WHERE id=?", f.Order, raw, f.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("Replay accepted result with omitted gate")
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err == nil {
		t.Fatal("inspection accepted result with omitted gate")
	}
}

func TestRequestAuthorityGateRequiresOriginalDebit(t *testing.T) {
	s, _, _, a := authorityReady(t, requestAuthorityFixture(t, nil), true)
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

func TestRequestAuthorityControlRequiresCurrentSource(t *testing.T) {
	s, path := messageStore(t, requestAuthorityFixture(t, nil))
	if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	req := committedRequest(t, s, "B")
	if _, err := s.AuthorizeIntent(t.Context(), req.Intent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	d, err := s.DecideLifecycle(t.Context(), req)
	if err != nil || d.Accepted || d.Reason != "authority_source_closed" {
		t.Fatalf("%+v %v", d, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestRequestAuthorityGrantCannotBeReused(t *testing.T) {
	p := requestAuthorityFixture(t, nil)
	authorityReplaceProgram(t, &p, "A", func(p *Program) { p.Code = append(p.Code[:2], p.Code[1], Instruction{Op: "end", Outcome: "succeeded"}) })
	s, path, _, first := authorityReady(t, p, true)
	if !first.Allowed {
		t.Fatal(first)
	}
	if _, err := s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	second, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(3), 0))
	if err != nil || second.Allowed || second.Authority.Reason != "authority_service_mismatch" {
		t.Fatalf("%+v %v", second, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestRequestAuthorityInvalidSelectionDoesNotFallback(t *testing.T) {
	p := requestAuthorityFixture(t, func(d *DelegationClaim, u *AuthorityUse, p *Provision) {
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
func TestRequestAuthorityNoRedelegation(t *testing.T) {
	p := requestAuthorityFixture(t, nil)
	var root AuthorityRoot
	authorityReplaceProgram(t, &p, "A", func(p *Program) { root = p.Code[1].Effect.Authority.Root })
	claim := DelegationClaim{Version: RequestAuthorityVersion, Root: root, Recipient: RunID{"B", 1}, RecipientRoot: RunID{"B", 1}, RecipientRevision: 1, OriginalRequestID: identity("B", int64(1), int64(1), 0), ServiceOperationID: identity("B", int64(1), int64(4), 0)}
	raw, _ := json.Marshal(claim)
	authorityReplaceProgram(t, &p, "A", func(p *Program) {
		p.Code = append(p.Code[:2], Instruction{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("B")), Generation: Lit(Int(1)), Obligation: Lit(Text("redelegate")), Payload: map[string]Operand{"operation": Lit(Text("DelegateAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(raw)))}}}, Instruction{Op: "end", Outcome: "succeeded"})
	})
	s, path, _, a := authorityReady(t, p, true)
	if !a.Allowed {
		t.Fatal(a)
	}
	d := authorizedDecision(t, s, committedRequest(t, s, "A"))
	if d.Accepted || d.Reason != "authority_origin_mismatch" {
		t.Fatalf("redelegation %+v", d)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
