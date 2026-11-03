package executor

import (
	"context"
	"encoding/json"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The original RED makes two real loopback contacts while both handlers are
// held. Two synchronous calls bypass the Run owner's persisted ceiling of one.
func TestCompositionSynchronousDispatchCannotBypassOwner(t *testing.T) {
	for _, scoped := range []bool{true, false} {
		t.Run(map[bool]string{true: "scoped", false: "unscoped"}[scoped], func(t *testing.T) {
			contacts := make(chan struct{}, 4)
			release := make(chan struct{})
			var active, peak atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request resourcefixture.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				n := active.Add(1)
				for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
				}
				defer active.Add(-1)
				contacts <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				json.NewEncoder(w).Encode(receiptFor(request))
			}))
			defer server.Close()
			p := Provision{}
			for _, name := range []string{"a", "b"} {
				addExternalClient(t, &p, name, server.URL, "non-idempotent", 5)
			}
			root := compositionRoot{Run: RunID{"a", 1}, Members: []RunID{{"a", 1}, {"b", 1}}, Limits: compositionLimits{MaxEffects: 4, MaxInFlight: 1}}
			if !scoped {
				addProgram(t, &p, "owner", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
				root.Run = RunID{"owner", 1}
				root.Members = []RunID{root.Run}
			}
			c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{root}}
			p.Composition = &c
			path := t.TempDir() + "/dispatch.sqlite"
			s, err := Create(t.Context(), path, p, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			for _, name := range []string{"a", "b"} {
				if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			done := make(chan error, 2)
			for _, name := range []string{"a", "b"} {
				go func(name string) { _, err := s.DispatchExternal(ctx, identity(name, 1, 1, 0)); done <- err }(name)
			}
			contacted, completed := 0, 0
			released := false
			validErrors := true
			for completed < 2 {
				select {
				case <-contacts:
					contacted++
					if contacted == 2 {
						close(release)
						released = true
					}
				case err := <-done:
					completed++
					if err == nil || !strings.Contains(err.Error(), "requires Store.Run") {
						validErrors = false
					}
				}
			}
			if !released {
				close(release)
			}
			if contacted != 0 || !validErrors || count(t, s, "attempts") != 0 || count(t, s, "external_attempts") != 0 || count(t, s, "composition_reservations") != 0 {
				t.Fatalf("mode bypass: contacts=%d peak=%d source_authorizations=%d attempts=%d reservations=%d", contacted, peak.Load(), count(t, s, "attempts"), count(t, s, "external_attempts"), count(t, s, "composition_reservations"))
			}
			report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
			if err != nil {
				t.Fatal(err)
			}
			if !report.Quiescent || len(report.UnresolvedIntents) != 0 || peak.Load() != 1 {
				t.Fatal("normal owner path did not drain under capacity", report, peak.Load())
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
