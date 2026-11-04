package executor

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// This is a pre-implementation falsifier of the frozen contract's assertion
// that literal programs can name exact existing resource-root IDs. The first
// store supplies a real canonical ID, without guessing the namespace. Reusing
// that exact provision in a new ordinary Create cannot preserve the root ID.
// Keep the failed run as evidence before revising the representation contract.
func TestRequestAuthorityLiteralRootLocator(t *testing.T) {
	p, c := compositionFixture(t)
	p.Composition = &c
	first, err := Create(t.Context(), filepath.Join(t.TempDir(), "first.sqlite"), p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := first.composition.descriptor()
	if err != nil {
		t.Fatal(err)
	}
	rootID := profile.Roots[0].ID
	t.Logf("first namespace=%s canonical root=%s", first.Namespace(), rootID)
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	// A precompiled AuthorityUse.Root.RootID and claim.RecipientRootID
	// would need this ID before Create; providing it as the existing public
	// ResourceRoot.ID is the only candidate ordinary provisioning route.
	c.Roots[0].ID = rootID
	p.Composition = &c
	second, err := Create(t.Context(), filepath.Join(t.TempDir(), "second.sqlite"), p, DefaultLimits())
	if second != nil {
		second.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "root identity mismatch") {
		t.Fatalf("expected historical literal-root rejection: %v", err)
	}
}

func TestRequestAuthorityFormatsAndLogicalLocators(t *testing.T) {
	p, c := compositionFixture(t)
	c.RequestAuthority = RequestAuthorityVersion
	p.Composition = &c
	var previous string
	for n := 0; n < 2; n++ {
		s, err := Create(t.Context(), filepath.Join(t.TempDir(), "authority.sqlite"), p, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		v, err := storedSchemaNumber(t.Context(), s.conn)
		if err != nil || v != 13 {
			t.Fatalf("format %d: %v", v, err)
		}
		actual, err := s.composition.descriptor()
		if err != nil {
			t.Fatal(err)
		}
		id := actual.Roots[0].ID
		if id == previous {
			t.Fatal("namespace-bound roots reused")
		}
		previous = id
		if actual.Roots[0].Run != c.Roots[0].Run {
			t.Fatal("logical root changed")
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []int{13, 14} {
		if protocolSchemaNumber(v) != 4 || hasDispatchAdmissionSchema(v) || hasContinuationSchema(v) || !hasRequestAuthoritySchema(v) {
			t.Fatalf("feature mask %d", v)
		}
	}
}

func requestAuthorityFixture(t *testing.T, mutate func(*DelegationClaim, *AuthorityUse, *Provision)) Provision {
	t.Helper()
	p := Provision{}
	root := AuthorityRoot{Run: RunID{"B", 1}, RootRun: RunID{"B", 1}, Permission: Permission{Kind: "external", Source: "B", Target: "http://127.0.0.1:1234/apply", Operation: "apply", Revision: 1, Obligation: "b"}}
	claim := DelegationClaim{Version: RequestAuthorityVersion, Root: root, Recipient: RunID{"A", 1}, RecipientRoot: RunID{"A", 1}, RecipientRevision: 1, OriginalRequestID: identity("B", int64(1), int64(1), 0), ServiceOperationID: identity("A", int64(1), int64(2), 0)}
	use := AuthorityUse{Version: RequestAuthorityVersion, Path: "delegated", Root: root, GrantID: identity("B", int64(1), int64(2), 0)}
	p.Permissions = []Permission{root.Permission, {Kind: "message", Source: "B", Target: "A", Operation: "message", Revision: 1, Obligation: "request"}}
	if mutate != nil {
		mutate(&claim, &use, &p)
	}
	raw, _ := json.Marshal(claim)
	addProgram(t, &p, "B", []Instruction{
		{Op: "emit", Dst: "request", Effect: &Effect{Kind: "message", Target: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("request")), Payload: map[string]Operand{}}},
		{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "lifecycle", Target: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("grant")), Payload: map[string]Operand{"operation": Lit(Text("DelegateAuthority")), "revision": Lit(Int(1)), "claim": Lit(Text(string(raw)))}}},
		{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	addProgram(t, &p, "A", []Instruction{
		{Op: "receive", Match: &Match{Kind: "message", Provenance: "ordinary-message", Source: Lit(Text("B")), Generation: Lit(Int(1)), Obligation: Lit(Text("request")), Operation: Lit(Text(""))}},
		{Op: "emit", Effect: &Effect{Kind: "external", Target: Lit(Text(root.Permission.Target)), Generation: Lit(Int(1)), Obligation: Lit(Text("b")), Authority: &use, Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(1))}}},
		{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Composition = &CompositionProvision{RequestAuthority: RequestAuthorityVersion, Roots: []ResourceRoot{
		{Run: RunID{"A", 1}, Members: []RunID{{"A", 1}}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}},
		{Run: RunID{"B", 1}, Members: []RunID{{"B", 1}}, Limits: ResourceLimits{MaxEffects: 4, MaxInFlight: 1}}}}
	return p
}

func TestRequestAuthorityDelegation(t *testing.T) {
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
			p := requestAuthorityFixture(t, tc.mutate)
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

func TestRequestAuthorityAdmissionAndReopen(t *testing.T) {
	p := requestAuthorityFixture(t, nil)
	path := filepath.Join(t.TempDir(), "gate.sqlite")
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	s.Commit(t.Context(), current(t, s, "B").Authority(), "")
	d := authorizedDecision(t, s, committedRequest(t, s, "B"))
	if !d.Accepted {
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
	a, _, err := s.prepareExternal(t.Context(), id)
	if err != nil || !a.Allowed {
		t.Fatalf("%+v %v", a, err)
	}
	for n := int64(1); n <= 2; n++ {
		retained, err := loadExternalAttempt(t.Context(), s.conn, id)
		if err != nil || retained.Request != a.Request {
			t.Fatalf("original wire identity changed across reopening: %+v %v", retained.Request, err)
		}
		if n == 2 && current(t, s, "A").Epoch == a.Request.Epoch {
			t.Fatal("reopened activation must differ from original request epoch")
		}
		op, reason, err := s.registerDispatch(t.Context(), id)
		if err != nil || reason != "" || op == nil {
			t.Fatalf("gate %s %v", reason, err)
		}
		facts, err := readCompositionFacts(t.Context(), s.conn)
		if err != nil {
			t.Fatal(err)
		}
		last := facts[len(facts)-1]
		if last.AuthorityAdmission == nil || last.AuthorityAdmission.Ordinal != n || last.AuthorityAdmission.Source.Epoch != current(t, s, "A").Epoch {
			t.Fatalf("gate %+v", last)
		}
		s.finishDispatch(op, nil)
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		s, err = Open(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
