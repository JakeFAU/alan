package substitution

import "fmt"

// RootTerminalFence is an adversarial runtime-boundary interpretation kept
// separate from the frozen K1 contract. Code inspection of
// internal/executor/invocation.go:165-177 shows that ANY non-open root projection
// closes final admission, including successful root termination. This adapter
// changes neither initial entitlement nor the external-cancellation bit.
// Its counterexample falsifies correspondence of K1's parallel Gc premise to
// that boundary; it does not falsify an implication whose C[K1] premise fails.
func RootTerminalFence(graph Graph) Graph {
	return func(c Config, s State) []Edge {
		edges := graph(c, s)
		for index := range edges {
			e := &edges[index]
			if e.Event.Kind != "admit" || e.Event.Status != "accepted" || s.RootStatus == "" {
				continue
			}
			i := -1
			for j := range s.Ops {
				if name(j) == e.Event.Operation {
					i = j
				}
			}
			if i < 0 {
				continue
			}
			n := s
			n.Ops[i].Admission = 2
			n.Ops[i].AdmissionReason = "scope_closed"
			exclude(&n, i)
			e.Next = n
			e.Event.Status = "rejected"
			e.Event.Reason = "scope_closed"
			e.Event.I = 0
			e.Event.R = 0
		}
		return edges
	}
}

// RuntimeBoundaryGraph additionally preserves the static peer's own lifecycle
// when the root is cancelled. Root cancellation fences future admissions but
// does not invent a cancellation transition for another static member.
func RuntimeBoundaryGraph(graph Graph) Graph {
	return RootTerminalFence(func(c Config, s State) []Edge {
		edges := graph(c, s)
		for i := range edges {
			e := &edges[i]
			if s.Closed && e.Event.Kind == "admit" && e.Event.Operation == "s" && e.Event.Reason == "source_closed" && s.Ops[sibling].Source == -2 {
				e.Event.Reason = "scope_closed"
				e.Next.Ops[sibling].AdmissionReason = "scope_closed"
			}
			if e.Event.Kind == "close_root" && c.Shape == "parallel" {
				e.Next.PeerPC = s.PeerPC
				e.Next.PeerStatus = s.PeerStatus
			}
		}
		if s.Closed && c.Shape == "parallel" {
			open := s
			open.Closed = false
			for _, e := range graph(c, open) {
				if e.Event.Operation != "s" || (e.Event.Kind != "invoke_commit" && e.Event.Kind != "outcome_consume") {
					continue
				}
				e.Next.Closed = true
				e.Next.RootPC = s.RootPC
				e.Next.RootStatus = s.RootStatus
				edges = append(edges, e)
			}
		}
		return edges
	})
}

// CheckBoundaryContext returns the entire census alongside membership: a
// matching observed runtime outcome does not erase a failed Gc claim. In
// particular entitled parallel failures retain completion counterexamples.
func CheckBoundaryContext(c Config, v Variant, observed ContextObservation) (Census, error) {
	graph := CandidateGraph(v)
	if v == Reference {
		graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
	}
	result := ExploreGraph(c, RuntimeBoundaryGraph(graph))
	for _, o := range result.Outcomes {
		if o == observed {
			return result, nil
		}
	}
	return result, fmt.Errorf("observation outside runtime-boundary extension: %+v", observed)
}
