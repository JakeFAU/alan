package delegation_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

func TestRequestBindingAutomatic(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name   string
		steps  int
		branch *bool
		ready  bool
	}{{name: "nonadjacent"}, {name: "adjacent", ready: true}, {name: "one", steps: 1}, {name: "three", steps: 3}, {name: "branch_true", branch: &yes}, {name: "branch_false", branch: &no}} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := func(name string) (string, string) {
				path := filepath.Join(t.TempDir(), "endpoint.sqlite")
				ledger, err := resource.Open(t.Context(), path, name)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { ledger.Close() })
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req resource.Request
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					receipt, err := ledger.Apply(r.Context(), req)
					if err != nil {
						t.Error(err)
						return
					}
					if err = json.NewEncoder(w).Encode(receipt); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				return server.URL, path
			}
			ua, pa := endpoint("a")
			ub, pb := endpoint("b")
			p, m, err := delegation.RequestBinding(delegation.RequestBindingOptions{EndpointA: ua, EndpointB: ub, LocalSteps: tc.steps, ForwardBranch: tc.branch, ReadyFirst: tc.ready, TerminalHandshake: true})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 128, MaxDispatch: 128})
			if err != nil {
				t.Fatal(err)
			}
			if len(report.UnresolvedIntents) != 0 {
				t.Fatal(report)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			snap, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			for _, run := range []runtime.RunID{m.Reciprocal.A, m.Reciprocal.B} {
				r, ok := snap.Run(run)
				if !ok || r.Projection.Status != "succeeded" {
					t.Fatal("not complete", r)
				}
			}
			for _, id := range []string{m.Reciprocal.BAService, m.Reciprocal.ABService} {
				e, ok := snap.Effect(id)
				if !ok || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || e.Settlement == nil {
					t.Fatal("service", e)
				}
				o, ok := snap.Observation(e.Settlement.ObservationID)
				if !ok || o.SelectedAt == nil {
					t.Fatal("result unconsumed", o)
				}
			}
			for _, path := range []string{pa, pb} {
				endpoint, err := resource.Inspect(t.Context(), path)
				if err != nil || endpoint.Contacts != 1 || len(endpoint.Applications) != 1 {
					t.Fatal(endpoint, err)
				}
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			reopened, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			reopened.Close()
		})
	}
}
