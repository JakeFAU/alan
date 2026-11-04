//go:build integration

package durabletest

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
)

func TestPhase3B4CampaignCompatibility(t *testing.T) {
	for _, name := range []string{"K01", "K02", "N11a-r1-add-ten", "N11a-r2-add-one"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			capability, operation := reciprocalR1, "add-one"
			if name == "N11a-r1-add-ten" {
				operation = "add-ten"
			}
			if name == "N11a-r2-add-one" {
				capability = reciprocalR2
			}
			p, m := reciprocalAdapterProvision(t, ua, ub, capability, operation)
			if name != "K01" {
				bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
				m.Claim.Version = runtime.RequestAuthorityBindingVersion
			}
			if p.Programs[0].Semantics != author.SemanticVersion {
				t.Fatal("not exact authored K program")
			}
			config := []reciprocalAdapter{{ua, reciprocalR1}, {ub, reciprocalR2}}
			manifest := delegation.RequestBindingManifest{Version: "alan/phase3b4-authored-compatibility/v1", Reciprocal: m, PrincipalRequests: [2]string{"", m.BARequest}}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", manifest)
			baseline3b3Write(t, dir, "deployment-adapters.json", config)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, err := s.Run(t.Context(), runtime.RunOptions{Adapters: reciprocalAdapters(config), MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			s.Close()
			if err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "run.json", report)
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			snap := bindingCapture(t, db, filepath.Join(dir, "final"))
			d, ok := snap.LifecycleDecision(m.Grant)
			classification := "exact-authored-service-consumed"
			if name == "K01" || name == "K02" {
				if !ok || !d.Accepted {
					t.Fatal(d)
				}
				e, ok := snap.Effect(m.BAService)
				if !ok || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.AdapterResult == nil || e.ResultDisposition != "validated" || e.Settlement == nil {
					t.Fatal(e)
				}
				o, ok := snap.Observation(e.Settlement.ObservationID)
				if !ok || o.SelectedAt == nil || o.Operation != m.BAService || o.Payload["mutation"].Number != 1 || o.Payload["total"].Number != 1 {
					t.Fatal(o)
				}
				reciprocalSelectedHistory(t, snap, o)
				a, _ := snap.Run(m.A)
				if a.Projection.Status != "succeeded" || endpoints["a"].Contacts != 1 || len(endpoints["a"].Applications) != 1 || endpoints["a"].Total != 1 || endpoints["b"].Contacts != 0 {
					t.Fatal(a, endpoints)
				}
			} else {
				classification = "cartesian-issuance-refused"
				if !ok || d.Accepted || len(snap.Effects) != 0 || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal(d, endpoints)
				}
			}
			s, err = runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			opened := bindingCapture(t, db, filepath.Join(dir, "post-open"))
			bindingRenewal(t, snap, opened)
			bindingIndex(t, dir, bindingCase{Name: name, Schedule: "automatic-run-authored-adapter", Classification: classification, Manifest: manifest, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}, {Ordinal: 2, Predecessor: "final", Label: "post-open", Directory: "post-open"}}}, p)
		})
	}
}
