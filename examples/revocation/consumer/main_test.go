package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/runtime"
)

// The host observes Q and the program selects only its original invocation.
func TestOrdinaryParentAndHostWait(t *testing.T) {
	for _, tc := range []struct {
		mode, knowledge, child string
		open                   bool
	}{
		{"success", "historical_application_receipt", "succeeded", false},
		{"selected-error", "local_failure_world_unresolved", "failed", false},
		{"unknown", "may_have_applied", "", true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) {
				var q effectboundary.WireRequest
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					http.Error(out, err.Error(), 400)
					return
				}
				_ = json.NewEncoder(out).Encode(effectboundary.Receipt{Key: q.Key, Profile: q.Profile, Delta: q.Delta, Mutation: 1, Total: q.Delta, Process: "public-consumer", PID: 1})
			}))
			defer server.Close()
			deployment, err := effectboundary.New(effectboundary.Config{Parent: true, Delta: 1})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "retained.db")
			s, err := runtime.Create(t.Context(), path, deployment.Provision, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			namespace := s.Namespace()
			root := runtime.RunID{Subject: "root", Generation: 1}
			scope := runtime.DispatchAuthority{Namespace: namespace, Run: root, OwningScope: true}
			if err := s.AwaitDispatchQuiescence(t.Context(), scope); !errors.Is(err, runtime.ErrDispatchAuthorityOpen) {
				t.Fatal(err)
			}
			report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: effectboundary.Registry(server.URL, tc.mode)})
			if err != nil {
				t.Fatal(err)
			}
			state, err := s.Snapshot(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			if state.Projection.Data["knowledge"].Text != tc.knowledge || state.Projection.Data["child_status"].Text != tc.child || (state.Projection.Status == "open") != tc.open {
				t.Fatal(state)
			}
			if tc.open != (len(report.UnknownAdapters) == 1) {
				t.Fatal(report)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err = observe(t.Context(), []string{"-db", path, "-namespace", namespace, "-subject", "root", "-generation", "1", "-scope"}, &out)
			if tc.open {
				if !errors.Is(err, runtime.ErrDispatchAuthorityOpen) || out.Len() != 0 {
					t.Fatalf("eligible target: %v %s", err, &out)
				}
			} else if err != nil || !json.Valid(out.Bytes()) {
				t.Fatalf("revoked target: %v %s", err, &out)
			}
			inspection, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(map[string]any{"mode": tc.mode, "target": scope, "parent": state, "run": report, "host_output": out.String(), "inspection": inspection})
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(data))
		})
	}
}
