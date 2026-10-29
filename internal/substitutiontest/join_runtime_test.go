//go:build integration

package substitutiontest

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	joinapp "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/boundary"
	joinmodel "github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/runtime"
)

// These are concrete samples of the frozen profile, not all-schedule claims.
func TestJoinRuntime(t *testing.T) {
	cases := []struct {
		name, candidate, mode, sMode, status string
		i, r                                 int
		premature, wrong                     bool
	}{
		{name: "a", candidate: "a", status: "succeeded", i: 3, r: 3},
		{name: "b", candidate: "b", status: "succeeded", i: 3, r: 3},
		{name: "invocation-limit", candidate: "b", status: "failed", i: 2, r: 3},
		{name: "run-limit", candidate: "b", status: "failed", i: 3, r: 2},
		{name: "premature", candidate: "a", status: "succeeded", i: 3, r: 3, premature: true},
		{name: "wrong-request", candidate: "a", status: "open", i: 3, r: 3, wrong: true},
		{name: "candidate-fail", candidate: "a", mode: "fail", status: "failed", i: 3, r: 3},
		{name: "candidate-absent", candidate: "a", mode: "absent", status: "failed", i: 3, r: 3},
		{name: "s-fail", candidate: "a", sMode: "fail", status: "failed", i: 3, r: 3},
		{name: "s-absent", candidate: "a", sMode: "absent", status: "failed", i: 3, r: 3},
		{name: "candidate-pending", candidate: "a", mode: "pending", status: "open", i: 3, r: 3},
		{name: "candidate-pending-s-failed", candidate: "a", mode: "pending", sMode: "fail", status: "open", i: 3, r: 3},
		{name: "s-pending-candidate-failed", candidate: "a", mode: "fail", sMode: "pending", status: "open", i: 3, r: 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.mode == "" {
				c.mode = "answer"
			}
			q := app.Request{ID: 1, Value: 0, Mode: c.mode}
			d, e := joinapp.New(candidate(t, c.candidate), app.Config{Shape: "parallel", Request: q, Limits: limits(c.i, c.r, 2)}, joinapp.Options{SMode: c.sMode, PrematureReport: c.premature, WrongRequest: c.wrong})
			must(t, e)
			path := filepath.Join(t.TempDir(), "retained.db")
			create(t, d, path)
			var decisive *runtime.Inspection
			ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, event boundary.Event) error {
				if decisive != nil || event.Boundary != "execution.committed" {
					return nil
				}
				s, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
				if e != nil {
					return e
				}
				if joinCut(s, "candidate-consumed") {
					decisive = &s
				}
				return nil
			})
			store, e := runtime.Open(ctx, path, runtime.DefaultLimits())
			must(t, e)
			report, e := store.Run(ctx, runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
			must(t, e)
			must(t, store.Close())
			if !report.Quiescent {
				t.Fatal("operational bound exhausted", report)
			}
			s := inspect(t, path)
			trace, e := decodeJoin(s, q)
			must(t, e)
			violations := joinmodel.CheckTrace(trace)
			retain(t, "result", map[string]any{"case": c.name, "request": q, "limits": limits(c.i, c.r, 2), "candidate": d.Candidate.ID(), "s": d.Peer.ID(), "fixed_runs": d.FixedRuns, "report": report, "inspection": s, "trace": trace, "violations": violations, "decisive": decisive})
			if projection(s, "root").Status != c.status {
				t.Fatalf("root=%s want=%s", projection(s, "root").Status, c.status)
			}
			if c.premature || c.wrong {
				if len(violations) == 0 {
					t.Fatal("invalid peer accepted")
				}
			} else if len(violations) > 0 {
				t.Fatal(violations)
			}

			if c.name == "invocation-limit" || c.name == "run-limit" {
				reason := "invocation_limit"
				if c.name == "run-limit" {
					reason = "created_run_limit"
				}
				found := false
				for _, event := range trace.Events {
					if event.Kind == "admit" && event.Role == "s" && event.Status == "rejected" && event.Reason == reason {
						found = true
					}
				}
				if !found {
					t.Fatal("distinct S capacity refusal absent", reason)
				}
			}
			if c.premature || c.wrong {
				code := "premature_report"
				if c.wrong {
					code = "request_correlation"
				}
				found := false
				for _, violation := range violations {
					if violation.Code == code {
						found = true
					}
				}
				if !found {
					t.Fatal("required independent causal/correlation rejection absent", code)
				}
			}
			if c.mode == "pending" || c.sMode == "pending" {
				caller := "root"
				if c.sMode == "pending" {
					caller = "peer"
				}
				found := false
				for _, row := range s.Tables["composition_admission_decisions"] {
					admission := decode[runtime.InvocationAdmission](row, "descriptor")
					if admission.Accepted && admission.Invocation.Caller.Subject == caller {
						for _, member := range admission.Members {
							if pendingReceive(s, member.Subject) {
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatal("pending branch lacks consumed original request and retained open receive")
				}
			}
			if c.name == "a" || c.name == "b" {
				if decisive == nil {
					t.Fatal("candidate-consumed/S-unadmitted mandatory receive witness absent")
				}
				if s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "succeeded" || projection(s, "peer").Status != "succeeded" {
					t.Fatal("aggregate/peer success absent")
				}
				want := int64(2)
				if c.candidate == "b" {
					want = 3
				}
				if s.Roots[0].Charges.Invocations != want || s.Roots[0].Charges.CreatedRuns != want || s.Roots[0].Charges.Effects != 0 {
					t.Fatal("unexpected spending", s.Roots[0].Charges)
				}
			}
			replayUnchanged(t, path)
		})
	}
}

func TestJoinOriginalContext(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, n := range []int{3, 4} {
			t.Run(fmt.Sprintf("%s-I%d-R%d", name, n, n), func(t *testing.T) {
				q := app.Request{ID: 1, Value: 0, Mode: "answer"}
				d, e := app.DelayedParallel(candidate(t, name), app.Config{Shape: "parallel", Request: q, Limits: limits(n, n, 2)}, 32)
				must(t, e)
				path := filepath.Join(t.TempDir(), "retained.db")
				create(t, d, path)
				r := run(t, path)
				s := inspect(t, path)
				normal := normalize(s, d, q)
				requireEvidence(t, normal)
				denied := false
				for _, row := range s.Tables["composition_admission_decisions"] {
					a := decode[runtime.InvocationAdmission](row, "descriptor")
					if a.Invocation.Caller.Subject == "peer" && !a.Accepted && a.Reason == "scope_closed" {
						denied = true
					}
				}
				if projection(s, "root").Status != "succeeded" || projection(s, "peer").Status != "failed" || s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "failed" || !denied {
					t.Fatal("historical lifetime counterexample absent")
				}
				retain(t, "result", map[string]any{"contract": "K1/C0", "candidate": name, "capacity": n, "report": r, "inspection": s, "normalized": normal})
				replayUnchanged(t, path)
			})
		}
	}
}
