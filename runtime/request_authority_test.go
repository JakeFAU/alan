package runtime_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"

	runtime "github.com/JakeFAU/alan/runtime"
)

// The low-level public Artifact route uses namespace-free literals. It needs no
// internal helper, mutable host grant API, or artifact rewrite after Create.
func TestRequestAuthorityPublicCommittedControl(t *testing.T) {
	root := runtime.AuthorityRoot{Run: runtime.RunID{Subject: "B", Generation: 1}, RootRun: runtime.RunID{Subject: "B", Generation: 1}, Permission: runtime.Permission{Kind: "external", Source: "B", Target: "http://127.0.0.1:1234", Operation: "apply", Revision: 1, Obligation: "b"}}
	claim := runtime.DelegationClaim{Version: runtime.RequestAuthorityVersion, Root: root, Recipient: runtime.RunID{Subject: "A", Generation: 1}, RecipientRoot: root.Run, RecipientRevision: 1, OriginalRequestID: `["B",1,1,0]`, ServiceOperationID: `["A",1,2,0]`}
	claimBytes, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	literal := func(v runtime.Value) any { return map[string]any{"Literal": v} }
	str := func(s string) any { return literal(runtime.Value{Kind: "string", Text: s}) }
	integer := func(n int64) any { return literal(runtime.Value{Kind: "int", Number: n}) }
	effect := func(kind, target, ob string, payload map[string]any) map[string]any {
		return map[string]any{"Kind": kind, "Target": str(target), "Generation": integer(1), "Obligation": str(ob), "Payload": payload}
	}
	p := runtime.Provision{Permissions: []runtime.Permission{root.Permission}}
	for _, subject := range []string{"A", "B"} {
		code := []any{map[string]any{"Op": "end", "Outcome": "succeeded"}}
		if subject == "B" {
			code = []any{map[string]any{"Op": "emit", "Effect": effect("message", "A", "request", map[string]any{})}, map[string]any{"Op": "emit", "Effect": effect("lifecycle", "A", "grant", map[string]any{"operation": str("DelegateAuthority"), "revision": integer(1), "claim": str(string(claimBytes))})}, code[0]}
		}
		raw, e := json.Marshal(map[string]any{"Semantics": "alan/1d/v1", "Code": code})
		if e != nil {
			t.Fatal(e)
		}
		digest := sha256.Sum256(raw)
		id := hex.EncodeToString(digest[:])
		p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: "alan/1d/v1", Bytes: raw})
		run := runtime.RunID{Subject: subject, Generation: 1}
		p.Subjects = append(p.Subjects, runtime.Subject{ID: subject, Generation: 1, Owner: "fixture", Revision: 1})
		p.Runs = append(p.Runs, runtime.InitialRun{ID: run, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]runtime.Value{}}})
	}
	p.Composition = &runtime.CompositionProvision{RequestAuthority: runtime.RequestAuthorityVersion, Roots: []runtime.ResourceRoot{{Run: root.Run, Members: []runtime.RunID{root.Run, {Subject: "A", Generation: 1}}, Limits: runtime.ResourceLimits{MaxEffects: 2, MaxInFlight: 1}}}}
	path := filepath.Join(t.TempDir(), "public.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for n := 0; n < 2; n++ {
		state, e := s.Snapshot(t.Context(), root.Run)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.Commit(t.Context(), state.Authority(), ""); e != nil {
			t.Fatal(e)
		}
	}
	req, err := s.LoadLifecycleRequest(t.Context(), `["B",1,2,0]`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizeIntent(t.Context(), req.Intent.ID); err != nil {
		t.Fatal(err)
	}
	decision, err := s.DecideLifecycle(t.Context(), req)
	if err != nil || !decision.Accepted || decision.Authority == nil || decision.Authority.Namespace != s.Namespace() {
		t.Fatalf("%+v %v", decision, err)
	}
	snap, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Features.RequestAuthority || len(snap.LifecycleDecisions) != 1 || snap.LifecycleDecisions[0].Authority.Root != root {
		t.Fatalf("missing typed authority evidence %+v", snap.LifecycleDecisions)
	}
}
