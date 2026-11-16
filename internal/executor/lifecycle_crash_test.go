//go:build integration

package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

func TestLifecycleDecisionSurvivesLostAcknowledgment(t *testing.T) {
	ctx := t.Context()
	if path := os.Getenv("ALAN_LIFECYCLE_CRASH_DB"); path != "" {
		s, err := Open(ctx, path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		req, err := s.LoadLifecycleRequest(ctx, os.Getenv("ALAN_LIFECYCLE_CRASH_INTENT"))
		if err != nil {
			t.Fatal(err)
		}

		// The callback runs after the SQLite commit and before DecideLifecycle
		// returns to its caller. The owner dies without delivering the result.
		killAfterCommit := boundary.WithObserver(ctx, func(_ context.Context, event boundary.Event) error {
			if event.Boundary == "lifecycle.decided" && event.Intent == req.Intent.ID {
				if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
					return err
				}
				select {} // Wait for delivery on hosts where Kill returns first.
			}
			return nil
		})
		result, err := s.DecideLifecycle(killAfterCommit, req)
		t.Fatalf("decision returned to caller before process death: %+v, %v", result, err)
	}

	for _, tc := range []struct {
		name      string
		accepted  bool
		reason    string
		successor int64
	}{
		{name: "accepted", accepted: true, reason: "retried", successor: 2},
		{name: "rejected", reason: "not_retryable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			s, path := messageStore(t, lifecycleFixture(t, []string{"Retry"}, false))
			if tc.accepted {
				if _, err := s.Commit(ctx, current(t, s, "worker").Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			req := committedRequest(t, s, "requester")
			auth, err := s.AuthorizeIntent(ctx, req.Intent.ID)
			if err != nil || !auth.Allowed {
				t.Fatalf("authorize lifecycle request: %+v, %v", auth, err)
			}
			if count(t, s, "lifecycle_results") != 0 {
				t.Fatal("setup already decided the lifecycle request")
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}

			childCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			child := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestLifecycleDecisionSurvivesLostAcknowledgment$")
			child.Env = append(os.Environ(), "ALAN_LIFECYCLE_CRASH_DB="+path, "ALAN_LIFECYCLE_CRASH_INTENT="+req.Intent.ID)
			out, err := child.CombinedOutput()
			var exit *exec.ExitError
			if childCtx.Err() != nil || !errors.As(err, &exit) {
				t.Fatalf("expected child SIGKILL at decision boundary: %v, %v\n%s", err, childCtx.Err(), out)
			}
			status, ok := exit.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatalf("child missed the decision boundary: %v\n%s", err, out)
			}
			t.Logf("SIGKILL pid=%d before lifecycle result delivery", exit.Pid())

			stopped, err := InspectExecution(ctx, path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			retained, found := stopped.LifecycleDecision(req.Intent.ID)
			if !found || len(stopped.LifecycleDecisions) != 1 || !reflect.DeepEqual(retained.Request, req) ||
				retained.DecisionSequence != 1 || retained.Accepted != tc.accepted ||
				retained.Reason != tc.reason || retained.Successor != tc.successor {
				t.Fatalf("wrong durable decision after SIGKILL: %+v", retained)
			}
			worker := RunID{Subject: "worker", Generation: 1}
			successor, created := stopped.Run(RunID{Subject: worker.Subject, Generation: 2})
			if created != tc.accepted || created && (successor.OriginOperationID != req.Intent.ID ||
				successor.Predecessor == nil || *successor.Predecessor != worker || successor.Sequence != 0) {
				t.Fatalf("wrong durable successor after SIGKILL: %+v", successor)
			}
			if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
				t.Fatal("replay after SIGKILL:", err)
			}

			reopened, err := Open(ctx, path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if !tc.accepted {
				// A fresh Retry could now succeed. This request must return its
				// earlier rejection without creating a new generation.
				if _, err := reopened.Commit(ctx, current(t, reopened, "worker").Authority(), ""); err != nil {
					t.Fatal(err)
				}
			}
			beforeRetry := inspectionFingerprint(t, path)
			got, err := reopened.DecideLifecycle(ctx, req)
			if err != nil || !reflect.DeepEqual(got, retained) {
				t.Fatalf("retry changed the retained decision: got %+v, want %+v, error %v", got, retained, err)
			}
			if inspectionFingerprint(t, path) != beforeRetry {
				t.Fatal("retry modified the database or WAL")
			}
		})
	}
}
