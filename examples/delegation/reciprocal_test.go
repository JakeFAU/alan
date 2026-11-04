package delegation_test

import (
	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReciprocalProvision(t *testing.T) {
	for _, scenario := range []delegation.Scenario{delegation.S1Own, delegation.S2Handshake, delegation.S3Missing, delegation.S3Claim} {
		t.Run(string(scenario), func(t *testing.T) {
			options := delegation.Options{EndpointA: "http://127.0.0.1:1234/apply", EndpointB: "http://127.0.0.1:1235/apply", Scenario: scenario, TerminalHandshake: true}
			p, m, err := delegation.Reciprocal(options)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == delegation.S1Own && m.Grant != "" {
				t.Fatal("own path unnecessarily requires delegation")
			}
			p2, m2, err := delegation.Reciprocal(options)
			if err != nil || !reflect.DeepEqual(p, p2) || !reflect.DeepEqual(m, m2) {
				t.Fatal("nondeterministic", err)
			}
			if len(p.Subjects) != 2 || len(p.Runs) != 2 || len(p.Programs) != 2 || p.Composition.RequestAuthority != runtime.RequestAuthorityVersion {
				t.Fatal("wrong domain")
			}
			for _, grant := range p.Permissions {
				if grant.Kind == "external" && grant.Source == "a" && scenario != delegation.S1Own {
					t.Fatal("ambient A authority")
				}
			}
			s, err := runtime.Create(t.Context(), filepath.Join(t.TempDir(), "runtime.sqlite"), p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if m.ABRequest != `["a",1,1,0]` || m.BAService != `["a",1,4,0]` || m.ABService != `["b",1,9,0]` {
				t.Fatal("locator drift", m)
			}
		})
	}
	if _, _, err := delegation.Reciprocal(delegation.Options{}); err == nil {
		t.Fatal("invalid options accepted")
	}
}
