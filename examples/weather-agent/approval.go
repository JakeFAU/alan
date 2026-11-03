package main

import (
	"fmt"

	"github.com/JakeFAU/alan/runtime"
)

func approvalID(n int) string { return fmt.Sprintf("weather-refresh-%d", n) }

// The application decides when to spend another weather opportunity. ALAN
// separately binds the operator's decision to the exact evidence reviewed.
func weatherReason(cut *runtime.ExecutionSnapshot, basis runtime.ContinuationBasis, n int) (string, error) {
	if granted, ok := cut.Continuation(basis.Target); ok {
		return granted.Approval.Reason, nil
	}
	if basis.ResultDigest == "" {
		if basis.Admitted {
			return afterUnknown, nil
		}
		return afterReservation, nil
	}
	prior, _ := cut.Run(readID(n - 1))
	switch prior.Projection.Status {
	case "succeeded":
		return afterReport, nil
	case "failed":
		return afterUnavailable, nil
	default:
		return "", fmt.Errorf("read %d has local work to resume before approving another read", n-1)
	}
}
