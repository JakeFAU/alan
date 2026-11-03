package executor

import "testing"

func TestContinuationBasisPreservesApprovedCut(t *testing.T) {
	target := RunID{"new", 1}
	prior := ContinuationBasis{Namespace: "sample", Target: target, PredecessorID: "paid-original", PreparedDigest: "original-digest", Admitted: true}
	s := ExecutionSnapshot{Namespace: "sample", Continuations: []*ContinuationView{{Approval: ContinuationApproval{ContinuationBasis: prior}}}}
	basis, err := s.ContinuationBasis(target)
	if err != nil || basis != prior || basis.ResultDigest != "" {
		t.Fatalf("duplicate after a later result must retain the original approval cut: %+v, %v", basis, err)
	}
	if _, err := s.ContinuationBasis(RunID{"unprovisioned", 1}); err == nil {
		t.Fatal("unprovisioned target can request an approval")
	}
}
