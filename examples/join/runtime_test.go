//go:build integration

package join_test

import (
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

func TestJoinRuntimeControls(t *testing.T) {
	for _, tc := range []struct {
		name, candidateMode, wantRoot string
		options                       join.Options
	}{
		{name: "answer", wantRoot: "succeeded"},
		{name: "candidate_fail", candidateMode: "fail", wantRoot: "failed"},
		{name: "candidate_absent", candidateMode: "absent", wantRoot: "failed"},
		{name: "s_fail", options: join.Options{SMode: "fail"}, wantRoot: "failed"},
		{name: "s_absent", options: join.Options{SMode: "absent"}, wantRoot: "failed"},
		{name: "candidate_pending", candidateMode: "pending", wantRoot: "open"},
		{name: "candidate_failed_s_pending", candidateMode: "fail", options: join.Options{SMode: "pending"}, wantRoot: "open"},
		{name: "s_pending", options: join.Options{SMode: "pending"}, wantRoot: "open"},
		{name: "wrong_request", options: join.Options{WrongRequest: true}, wantRoot: "open"},
		{name: "premature_report", options: join.Options{PrematureReport: true}, wantRoot: "succeeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config()
			if tc.candidateMode != "" {
				cfg.Request.Mode = tc.candidateMode
			}
			d, e := join.New(candidate(t), cfg, tc.options)
			if e != nil {
				t.Fatal(e)
			}
			s, path := createStore(t, d)
			r, e := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
			if e != nil {
				t.Fatal(e)
			}
			root, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
			if e != nil {
				t.Fatal(e)
			}
			peer, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "peer", Generation: 1})
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("root=%s peer=%s steps=%d dispatches=%d candidate=%s/%s S=%s/%s report_request=%d", root.Projection.Status, peer.Projection.Status, r.Steps, r.Dispatches, root.Projection.Data["child_status"].Text, root.Projection.Data["child_reason"].Text, root.Projection.Data["join_status"].Text, root.Projection.Data["join_reason"].Text, root.Projection.Data["join_request_id"].Number)
			if root.Projection.Status != tc.wantRoot {
				t.Fatalf("root %s, want %s", root.Projection.Status, tc.wantRoot)
			}
			if tc.options.SMode == "pending" {
				if peer.Projection.Status != "open" {
					t.Fatal("pending S fabricated peer completion")
				}
			} else if peer.Projection.Status != "succeeded" {
				t.Fatal("peer protocol did not complete", peer.Projection)
			}
			if tc.name == "s_absent" || tc.name == "s_fail" {
				if _, ok := peer.Projection.Data["peer_id"]; ok {
					t.Fatal("absent response binding survived")
				}
				if root.Projection.Data["join_present"].Flag || root.Projection.Data["join_id"].Number != 0 || root.Projection.Data["join_value"].Number != 0 {
					t.Fatal("absent report invented response", root.Projection.Data)
				}
			}
			if _, e := runtime.Replay(t.Context(), path, runtime.DefaultLimits()); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func createStore(t *testing.T, d substitution.Deployment) (*runtime.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "retained.db")
	s, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}
func TestJoinOriginalNegativeAndAccounting(t *testing.T) {
	for _, tc := range []struct {
		name         string
		newCandidate func() (author.Template, error)
		charge       int64
	}{{"a", a.New, 2}, {"b", b.New, 3}} {
		for _, old := range []bool{true, false} {
			name := tc.name + "/join"
			if old {
				name = tc.name + "/original"
			}
			t.Run(name, func(t *testing.T) {
				c, e := tc.newCandidate()
				if e != nil {
					t.Fatal(e)
				}
				var d substitution.Deployment
				if old {
					d, e = substitution.DelayedParallel(c, config(), 32)
				} else {
					d, e = join.New(c, config(), join.Options{})
				}
				if e != nil {
					t.Fatal(e)
				}
				s, _ := createStore(t, d)
				r, e := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
				if e != nil {
					t.Fatal(e)
				}
				root, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
				if e != nil {
					t.Fatal(e)
				}
				peer, e := s.Snapshot(t.Context(), runtime.RunID{Subject: "peer", Generation: 1})
				if e != nil {
					t.Fatal(e)
				}
				if root.Projection.Status != "succeeded" {
					t.Fatal("root", root.Projection)
				}
				if old {
					if peer.Projection.Status != "failed" || len(r.Roots) != 1 || r.Roots[0].Outcome.Status != "failed" {
						t.Fatal("original lifetime negative vanished", peer.Projection, r.Roots)
					}
				} else {
					if peer.Projection.Status != "succeeded" || len(r.Roots) != 1 || r.Roots[0].Outcome.Status != "succeeded" {
						t.Fatal("sample did not complete", peer.Projection, r.Roots)
					}
					if r.Roots[0].Charges.Invocations != tc.charge || r.Roots[0].Charges.CreatedRuns != tc.charge || r.Roots[0].Charges.Effects != 0 {
						t.Fatal("charges", r.Roots)
					}
				}
			})
		}
	}
}
