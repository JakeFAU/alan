package executor

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestInspectDynamicTargetAcrossEventStreams(t *testing.T) {
	const controller = "retirement-controller"
	for _, witness := range []string{"observation", "bound transport", "authorization", "later redelivery"} {
		t.Run(witness, func(t *testing.T) {
			ctx := t.Context()
			p, profile := invocationFixture(t, "succeeded", 1)
			addRetirementController(t, &p, []string{"CancelRun", "CancelRun"})
			path := filepath.Join(t.TempDir(), "events.sqlite")
			s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			op := identity("root", 1, 1, 0)
			inv := compositionInvocationID(s.Namespace(), op)
			entry := compositionMemberID(inv, "entry")
			create := func() {
				a := invokeOnce(t, s)
				if !a.Accepted || a.Invocation.ID != inv {
					t.Fatal(a)
				}
			}
			if witness == "observation" || witness == "bound transport" {
				create()
			}
			if witness == "bound transport" {
				a, err := s.AdmitInput(ctx, Input{TransportID: "exists", LogicalID: "exists", Target: entry.Subject, Payload: map[string]Value{}})
				if err != nil || a.Bound == nil || *a.Bound != entry || a.Accepted {
					t.Fatal(a, err)
				}
			}
			in := Input{TransportID: "first", LogicalID: "control", Target: controller, Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(entry.Subject)}}
			a, err := s.AdmitInput(ctx, in)
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			if _, err = s.Commit(ctx, current(t, s, controller).Authority(), a.ObservationID); err != nil {
				t.Fatal(err)
			}
			request := committedRequest(t, s, controller)
			if witness == "authorization" {
				create()
				if _, err = s.Commit(ctx, mustSnapshot(t, s, entry).Authority(), compositionIdentity("alan/invocation-request/v1", inv)); err != nil {
					t.Fatal(err)
				}
				if _, err = s.Commit(ctx, mustSnapshot(t, s, entry).Authority(), ""); err != nil {
					t.Fatal(err)
				}
				if _, err = s.AuthorizeIntent(ctx, identity(entry.Subject, 1, 2, 0)); err != nil {
					t.Fatal(err)
				}
			}
			decision := authorizedDecision(t, s, request)
			if witness == "later redelivery" {
				if decision.Before != nil || decision.Reason != "target_missing" {
					t.Fatal(decision)
				}
				create()
				in.TransportID = "redelivery"
				again, err := s.AdmitInput(ctx, in)
				if err != nil || !again.Accepted || again.ObservationID != a.ObservationID {
					t.Fatal(again, err)
				}
				snapshot, err := InspectExecution(ctx, path, DefaultLimits())
				if err != nil {
					t.Fatal("a later redelivery cannot precede the original selection", err)
				}
				retained, ok := snapshot.LifecycleDecision(request.Intent.ID)
				if !ok || retained.Before != nil || retained.Reason != "target_missing" {
					t.Fatal(retained)
				}
				return
			}
			if decision.Before == nil || decision.Before.Run != entry || decision.Accepted {
				t.Fatal(decision)
			}
			if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
				t.Fatal("genuine target witness", err)
			}
			decision.Before, decision.After, decision.Reason = nil, nil, "target_missing"
			if _, err := s.conn.ExecContext(ctx, "UPDATE lifecycle_results SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(decision), request.Intent.ID); err != nil {
				t.Fatal(err)
			}
			fact := compositionIdentity("alan/fact/lifecycle/v1", request.Intent.ID)
			if _, err := s.conn.ExecContext(ctx, "DELETE FROM composition_facts WHERE id=?", fact); err != nil {
				t.Fatal(err)
			}
			if snapshot, err := InspectExecution(ctx, path, DefaultLimits()); snapshot != nil || !errors.Is(err, ErrCorrupt) {
				t.Fatalf("missing target after earlier %s evidence: %v", witness, err)
			}
		})
	}
}

func TestInspectMissingTransportBindingAcrossCreation(t *testing.T) {
	for _, tc := range []struct {
		name           string
		beforeCreation bool
	}{{"genuine absence survives retransmission", true}, {"reject absence after ordered observation", false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			p, profile := invocationFixture(t, "succeeded", 1)
			addRetirementController(t, &p, []string{"CancelRun"})
			path := filepath.Join(t.TempDir(), "transport.sqlite")
			s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			inv := compositionInvocationID(s.Namespace(), identity("root", 1, 1, 0))
			entry := compositionMemberID(inv, "entry")
			in := Input{TransportID: "visit", LogicalID: "visit", Target: entry.Subject, Generation: 1, Payload: map[string]Value{}}
			if tc.beforeCreation {
				a, err := s.AdmitInput(ctx, in)
				if err != nil || a.Bound != nil || a.Reason != "target_missing" {
					t.Fatal(a, err)
				}
			}
			if call := invokeOnce(t, s); !call.Accepted || call.Invocation.ID != inv {
				t.Fatal(call)
			}
			// This accepted input follows the new member request in observations
			// and precedes the subsequent transport in admissions.
			a, err := s.AdmitInput(ctx, Input{TransportID: "bridge", LogicalID: "bridge", Target: "retirement-controller", Generation: 1, Obligation: "control", Payload: map[string]Value{"target": Text(entry.Subject)}})
			if err != nil || !a.Accepted {
				t.Fatal(a, err)
			}
			visit, err := s.AdmitInput(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if tc.beforeCreation {
				if visit.Bound != nil || visit.Reason != "target_missing" {
					t.Fatal(visit)
				}
				if _, err := InspectExecution(ctx, path, DefaultLimits()); err != nil {
					t.Fatal("retransmission changed original target absence", err)
				}
				return
			}
			if visit.Bound == nil || *visit.Bound != entry || visit.Reason != "route_denied" {
				t.Fatal(visit)
			}
			visit.Bound, visit.Reason = nil, "target_missing"
			if _, err := s.conn.ExecContext(ctx, "UPDATE admissions SET result=CAST(? AS BLOB) WHERE id=?", compositionJSON(visit), admissionKey("input", in.TransportID)); err != nil {
				t.Fatal(err)
			}
			if snapshot, err := InspectExecution(ctx, path, DefaultLimits()); snapshot != nil || !errors.Is(err, ErrCorrupt) {
				t.Fatalf("accepted absent target after observed creation: %v", err)
			}
		})
	}
}
