package executor

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestInspectRejectsFabricatedAuthority(t *testing.T) {
	reject := func(t *testing.T, path string) {
		t.Helper()
		if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
			t.Errorf("replay accepted impossible authority: %v", err)
		}
		snapshot, err := InspectExecution(t.Context(), path, DefaultLimits())
		if snapshot != nil || !errors.Is(err, ErrCorrupt) {
			t.Fatalf("inspection accepted impossible authority: %v", err)
		}
	}
	t.Run("foreign terminal route", func(t *testing.T) {
		s, path := newStore(t, false)
		for range 2 {
			if _, err := s.Commit(t.Context(), state(t, s).Authority(), ""); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.conn.ExecContext(t.Context(), "UPDATE terminal_obligations SET destinations=CAST(? AS BLOB)", compositionJSON([]TerminalDestination{{"invented", "wrong"}})); err != nil {
			t.Fatal(err)
		}
		reject(t, path)
	})
	t.Run("source epoch before emission", func(t *testing.T) {
		first, path := newStore(t, false)
		if err := first.Close(); err != nil {
			t.Fatal(err)
		}
		s, err := Open(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		r, err := s.Commit(t.Context(), state(t, s).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		a, err := s.AuthorizeIntent(t.Context(), r.Projection.Data["pending"].Text)
		if err != nil || a.Source.Epoch != 2 {
			t.Fatal(a, err)
		}
		a.Source.Epoch = 1
		if _, err = s.conn.ExecContext(t.Context(), "UPDATE attempts SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(a), a.ID); err != nil {
			t.Fatal(err)
		}
		reject(t, path)
	})
	t.Run("lifecycle decision without target grant", func(t *testing.T) {
		s, path := messageStore(t, lifecycleFixture(t, []string{"CancelRun"}, false))
		r := authorizedDecision(t, s, committedRequest(t, s, "requester"))
		if !r.Accepted {
			t.Fatal(r)
		}
		if _, err := s.conn.ExecContext(t.Context(), "DELETE FROM permissions WHERE kind='lifecycle'"); err != nil {
			t.Fatal(err)
		}
		reject(t, path)
	})
	for _, mutate := range []string{
		"UPDATE subjects SET revision=revision+1",
		"UPDATE subjects SET retired=1",
		"UPDATE subjects SET retry_cancelled=1-retry_cancelled",
	} {
		t.Run(mutate, func(t *testing.T) {
			s, path := newStore(t, false)
			if _, err := s.conn.ExecContext(t.Context(), mutate); err != nil {
				t.Fatal(err)
			}
			reject(t, path)
		})
	}
}

func TestInspectMissingTargetBeforeInvocationCreatesIt(t *testing.T) {
	ctx := t.Context()
	p, profile := invocationFixture(t, "succeeded", 1)
	addRetirementController(t, &p, []string{"CancelRun"})
	path := filepath.Join(t.TempDir(), "future.sqlite")
	s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	op := identity("root", 1, 1, 0)
	future := compositionMemberID(compositionInvocationID(s.Namespace(), op), "entry")
	const controller = "retirement-controller"
	a, err := s.AdmitInput(ctx, Input{TransportID: "t", LogicalID: "t", Target: controller, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(future.Subject)}})
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
	if _, err = s.Commit(ctx, current(t, s, controller).Authority(), a.ObservationID); err != nil {
		t.Fatal(err)
	}
	request := committedRequest(t, s, controller)
	decision := authorizedDecision(t, s, request)
	if decision.Reason != "target_missing" || decision.Before != nil {
		t.Fatal(decision)
	}
	if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if call := invokeOnce(t, s); !call.Accepted || call.Invocation.OperationID != op {
		t.Fatal(call)
	}
	after, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal("later creation invalidated earlier decision", err)
	}
	if _, err := Replay(ctx, path, DefaultLimits()); err != nil {
		t.Fatal("replay of a genuine absence followed by member creation", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal("reopen of a genuine absence followed by member creation", err)
	}
	defer reopened.Close()
	retained, found := after.LifecycleDecision(decision.Request.Intent.ID)
	if !found || retained.Before != nil || retained.Reason != "target_missing" {
		t.Fatal("original absence lost", retained)
	}
	if _, ok := after.Run(future); !ok {
		t.Fatal("created run missing")
	}
	// Each intent was inserted atomically with its reduction. Reversing only
	// this journal while retaining both descriptors is independently detectable.
	if _, err = reopened.conn.ExecContext(ctx, "UPDATE intents SET rowid=(SELECT max(rowid)+1 FROM intents) WHERE id=?", request.Intent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(ctx, path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Errorf("replay accepted reversed intent history: %v", err)
	}
	if _, err = InspectExecution(ctx, path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Errorf("inspection accepted reversed intent history: %v", err)
	}

}

func TestInspectRejectsMissingTargetAfterPriorWitness(t *testing.T) {
	for _, tc := range []struct {
		name       string
		decisions  int
		memberStep bool
	}{
		{"earlier_lifecycle_witness", 2, false}, {"earlier_scoped_reduction", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			p, profile := invocationFixture(t, "succeeded", 1)
			addRetirementController(t, &p, []string{"CancelRun", "CancelRun"})
			path := filepath.Join(t.TempDir(), "exist.sqlite")
			s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			initial := invokeOnce(t, s)
			entry := compositionMemberID(initial.Invocation.ID, "entry")
			if tc.memberStep {
				request := compositionIdentity("alan/invocation-request/v1", initial.Invocation.ID)
				if _, err := s.Commit(ctx, mustSnapshot(t, s, entry).Authority(), request); err != nil {
					t.Fatal(err)
				}
			}
			const controller = "retirement-controller"
			a, err := s.AdmitInput(ctx, Input{TransportID: "t", LogicalID: "t", Target: controller, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(entry.Subject)}})
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			if _, err = s.Commit(ctx, current(t, s, controller).Authority(), a.ObservationID); err != nil {
				t.Fatal(err)
			}
			var r LifecycleResult
			for range tc.decisions {
				req := committedRequest(t, s, controller)
				r = authorizedDecision(t, s, req)
				if r.Before == nil || r.Reason != "grant_denied" {
					t.Fatal(r)
				}
			}
			if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			id := r.Request.Intent.ID
			r.Before, r.After = nil, nil
			r.Reason = "target_missing"
			raw := []byte(compositionJSON(r))
			if _, err = s.conn.ExecContext(ctx, "UPDATE lifecycle_results SET result=? WHERE id=?", raw, id); err != nil {
				t.Fatal(err)
			}
			f := compositionIdentity("alan/fact/lifecycle/v1", id)
			if _, err = s.conn.ExecContext(ctx, "DELETE FROM composition_facts WHERE id=?", f); err != nil {
				t.Fatal(err)
			}
			cut, err := InspectExecution(ctx, path, DefaultLimits())
			if cut != nil || !errors.Is(err, ErrCorrupt) {
				t.Error("accepted absent target after its existence was witnessed")
			}
		})
	}
}
