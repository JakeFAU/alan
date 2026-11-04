//go:build integration

package durabletest

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

// This reuses the actual author.Compile K01 construction and changes only
// authority input versions and consequent artifact hashes before Create.
func TestPhase3B4AuthoredAdapterCompatibility(t *testing.T) {
	dir := t.TempDir()
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m := reciprocalAdapterProvision(t, ua, ub, reciprocalR1, "add-one")
	p.Composition.RequestAuthority = runtime.RequestAuthorityBindingVersion
	m.Claim.Version = runtime.RequestAuthorityBindingVersion
	reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { c.Version = runtime.RequestAuthorityBindingVersion })
	reciprocalEdit(t, &p, "a", func(program map[string]any) {
		effect := reciprocalInstruction(program, 2)["Effect"].(map[string]any)
		effect["Authority"] = runtime.AuthorityUse{Version: runtime.RequestAuthorityBindingVersion, Path: "delegated", Root: m.BAServiceRoot, GrantID: m.Grant}
	})
	if p.Programs[0].Semantics != author.SemanticVersion {
		t.Fatal("authored semantics lost")
	}
	path := filepath.Join(t.TempDir(), "authored.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Run(t.Context(), runtime.RunOptions{Adapters: reciprocalAdapters([]reciprocalAdapter{{ua, reciprocalR1}, {ub, reciprocalR2}}), MaxSteps: 128, MaxDispatch: 128, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	snap, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	e, ok := snap.Effect(m.BAService)
	if !ok || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.Settlement == nil || e.AdapterAttempt.Authority.Version != runtime.RequestAuthorityBindingVersion {
		t.Fatal(e)
	}
	result, ok := snap.Observation(e.Settlement.ObservationID)
	if !ok || result.SelectedAt == nil || result.Operation != m.BAService {
		t.Fatal(result)
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	reopened, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	reopened.Close()
	a, b := phaseSnapshot(t, pa), phaseSnapshot(t, pb)
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	if a.Contacts != 1 || len(a.Applications) != 1 || a.Total != 1 || b.Contacts != 0 {
		t.Fatal(a, b)
	}
}
