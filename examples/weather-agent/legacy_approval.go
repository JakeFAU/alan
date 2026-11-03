package main

import (
	"context"
	"fmt"
	"github.com/JakeFAU/alan/author"
	"maps"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Multi-read weather deployments created before continuation slots have only
// ordinary local inputs. Keep their original CLI trust boundary on reopen.

// A local operation handle repeats across independent stores. The frozen
// request digest also covers its namespace-bound key. A known outcome has
// its own digest covering the original body, disposition and typed values.
func legacyApprovalPayload(cut *runtime.ExecutionSnapshot, n int) (map[string]schema.Value, error) {
	prev, _ := cut.Run(readID(n - 1))
	if len(prev.EffectIDs) != 1 {
		return nil, fmt.Errorf("read %d has no original operation to refresh", n-1)
	}
	e, _ := cut.Effect(prev.EffectIDs[0])
	var purpose, result string
	switch {
	case e.State == runtime.EffectReservedUnknown && e.DispatchAdmission == nil:
		purpose = afterReservation
	case e.State == runtime.EffectReservedUnknown:
		purpose = afterUnknown
	case e.AdapterResult != nil && prev.Projection.Status == "succeeded":
		purpose, result = afterReport, e.AdapterResult.Digest
	case e.AdapterResult != nil && prev.Projection.Status == "failed":
		purpose, result = afterUnavailable, e.AdapterResult.Digest
	default:
		return nil, fmt.Errorf("read %d has local work to resume before approving another read", n-1)
	}
	return map[string]schema.Value{
		"previous_operation_id": schema.Text(e.OperationID), "purpose": schema.Text(purpose),
		"prepared_digest": schema.Text(e.AdapterAttempt.Prepared.Digest), "result_digest": schema.Text(result),
	}, nil
}

// An accepted local delivery is an operator assertion. Validate its target,
// original reservation, outcome and reason before presenting it as authority.
// The CLI holds the sole owner from this check through dispatch. An unknown
// original cannot later acquire a response through this ordinary adapter.
func legacyApproval(cut *runtime.ExecutionSnapshot, n int) (*runtime.ObservationView, error) {
	run, _ := cut.Run(readID(n))
	var consent *runtime.ObservationView
	for _, a := range cut.Admissions {
		if a.Domain != "input" || !a.Accepted || a.Target != run.ID.Subject || a.Obligation != approval {
			continue
		}
		o, ok := cut.Observation(a.ObservationID)
		if !ok || consent != nil || a.TransportID != approvalID(n) || a.LogicalID != approvalID(n) || a.RequestedGeneration != run.ID.Generation || a.Correlation != "" {
			return nil, fmt.Errorf("read %d has an unexpected local refresh input %q", n, a.TransportID)
		}
		want, err := legacyApprovalPayload(cut, n)
		if err != nil {
			return nil, fmt.Errorf("inapplicable refresh input for read %d: %w", n, err)
		}
		if !maps.Equal(o.Payload, want) {
			return nil, fmt.Errorf("read %d refresh input disagrees with the original reservation, result or recovery reason", n)
		}
		consent = o
	}
	if len(run.EffectIDs) != 0 && (consent == nil || consent.SelectedAt == nil) {
		return nil, fmt.Errorf("read %d has an operation without a selected refresh approval", n)
	}
	return consent, nil
}

func admitLegacyApproval(ctx context.Context, s *runtime.Store, cut *runtime.ExecutionSnapshot, n int) error {
	payload, err := legacyApprovalPayload(cut, n)
	if err != nil {
		return err
	}
	id := approvalID(n)
	decision, err := s.AdmitInput(ctx, runtime.Input{
		TransportID: id, LogicalID: id, Target: readID(n).Subject, Generation: 1, Obligation: approval,
		Payload: payload,
	})
	if err != nil {
		return err
	}
	if !decision.Accepted {
		return fmt.Errorf("refresh approval refused: %s", decision.Reason)
	}
	return nil
}

// These steps reproduce the immutable M4 program identities on existing
// multi-read stores. Newly provisioned agents never send this handoff.
func legacyReadPrefix(n int, ports map[string]author.Port) []author.Step {
	if n == 1 {
		return nil
	}
	handle := schema.Record{"previous_operation_id": {Kind: "string", MaxBytes: 4096}}
	ports["previous"] = author.Port{Peer: readID(n - 1).Subject, Obligation: "previous-read", Receive: handle}
	ports["approval"] = author.ContinuationPort(approval, afterReservation, afterUnknown, afterUnavailable, afterReport)
	return []author.Step{
		author.Receive("previous", map[string]string{"previous_operation_id": "previous_operation_id"}),
		author.Receive("approval", map[string]string{"previous_operation_id": "approved_operation_id", "purpose": "approval_reason"}),
		author.IfEqual(author.Var("previous_operation_id"), author.Var("approved_operation_id"), nil, []author.Step{author.Failure()}),
	}
}

func legacySendHandle(n int, ports map[string]author.Port) author.Step {
	ports["next"] = author.Port{Peer: readID(n + 1).Subject, Obligation: "previous-read", Send: schema.Record{"previous_operation_id": {Kind: "string", MaxBytes: 4096}}}
	return author.Send("next", map[string]author.Expr{"previous_operation_id": author.Var("operation_id")})
}
