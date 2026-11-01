package reconciliation

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func checkState(p Profile, s State) error {
	if s.Target != 1 || s.OriginalDebits != 1 || s.Applications < 0 || s.Applications > 2 || s.SnapshotCount > 4 {
		return fmt.Errorf("original charge or world bounds")
	}
	if s.Quiescent && (!s.Revoked || s.Original != "handled" || s.Dead) {
		return fmt.Errorf("original drain")
	}
	if s.Settled && s.Interaction == "" || s.Published && s.Interaction == "" || s.OutcomeSelected && !s.Published {
		return fmt.Errorf("original settlement/publication chain")
	}
	if s.Interaction == "transport_error" && p.Origin != "failed" || s.OriginalFailureSelected && s.Interaction != "transport_error" {
		return fmt.Errorf("immutable interaction")
	}
	debits := 0
	used := false
	for i, q := range s.Reads {
		if q.Stage != "" {
			debits++
		}
		if q.Spent < 0 || q.Spent > 2 || i >= p.Queries && q.Stage != "" {
			return fmt.Errorf("query bounds")
		}
		if q.Selected && q.Stage != "handled" || q.Drained && !q.Closed {
			return fmt.Errorf("improper selection or Q")
		}
		if q.Closed && q.Spent == 0 && q.Outcome != "" {
			return fmt.Errorf("closed resolver acquired result before admission")
		}
		if q.Retained != 0 {
			if q.Retained > s.SnapshotCount {
				return fmt.Errorf("invented snapshot")
			}
			record := s.Log[q.Retained-1]
			if record.Target != s.Target || record.Target != q.BoundTo || record.Query != i+1 || record.Ordinal != q.Spent || record.Status != q.Outcome {
				return fmt.Errorf("unauthenticated snapshot")
			}
		} else if q.Outcome == "found" || q.Outcome == "not_found" || q.Outcome == "conflict" {
			return fmt.Errorf("observation lacks original snapshot")
		}
		if q.Adoption == "adopted" && (q.Outcome != "found" || p.ReadOnly || p.Policy != "" || p.BadTarget || s.ReconciliationSource != i+1) {
			return fmt.Errorf("invalid original import")
		}
		used = used || (p.Concurrent || i == p.Queries-1) && q.Selected && q.Outcome == "found" && (q.Adoption == "adopted" || s.Interaction == "transport_error" && q.Adoption == "existing")
	}
	if debits != s.ReadDebits {
		return fmt.Errorf("logical debit")
	}
	if s.Interaction == "reconciled" {
		i := s.ReconciliationSource - 1
		if i < 0 || i >= p.Queries || s.Reads[i].Adoption != "adopted" {
			return fmt.Errorf("original result has no import proof")
		}
	} else if s.ReconciliationSource != 0 {
		return fmt.Errorf("spurious original result reference")
	}
	if s.Rejected && (s.Used || !s.OutcomeSelected || p.Concurrent || !s.Reads[1].Selected || s.Reads[1].Adoption != "duplicate") {
		return fmt.Errorf("unsupported rejection")
	}
	if s.Used && (!s.OutcomeSelected || !used) {
		return fmt.Errorf("consumer missing selections")
	}
	return nil
}

func checkEdge(p Profile, s State, a string, n State) error {
	if err := checkState(p, n); err != nil {
		return err
	}
	if n.Applications != s.Applications && (a != "world.apply" || !s.Held || n.Applications != s.Applications+1) {
		return fmt.Errorf("business application caused by %s", a)
	}
	if s.Interaction != "" && n.Interaction != s.Interaction {
		return fmt.Errorf("rewritten interaction")
	}
	if a == "original.Q" && s.Interaction != "" && !s.Settled {
		return fmt.Errorf("owner acknowledged Q before surviving original settlement")
	}
	if a == "original.admit" && s.Revoked {
		return fmt.Errorf("original admitted after R")
	}
	if a == "world.apply" && !s.Held {
		return fmt.Errorf("world lacks packet")
	}
	if strings.Contains(a, "kill") && n.Held != s.Held {
		return fmt.Errorf("lost independent packet")
	}
	for i, q := range s.Reads {
		r := n.Reads[i]
		if r.Spent != q.Spent && (r.Spent != q.Spent+1 || q.Closed || !s.Quiescent || a != fmt.Sprintf("q%d.spend-and-admit", i+1)) {
			return fmt.Errorf("ordinal reset or fresh admission without authority")
		}
		if q.Outcome != "" && (r.Outcome != q.Outcome || r.Retained != q.Retained || r.Adoption != q.Adoption || r.Spent != q.Spent) {
			return fmt.Errorf("replaced terminal query")
		}
		if q.Adoption != "adopted" && r.Adoption == "adopted" && (s.Interaction != "" || n.Interaction != "reconciled" || !s.Quiescent) {
			return fmt.Errorf("non-atomic or premature adoption")
		}
	}
	for i := 0; i < s.SnapshotCount; i++ {
		if s.Log[i] != n.Log[i] {
			return fmt.Errorf("replaced endpoint snapshot")
		}
	}
	if n.SnapshotCount != s.SnapshotCount {
		record := n.Log[s.SnapshotCount]
		status := "not_found"
		if s.Applications == 1 {
			status = "found"
		} else if s.Applications > 1 {
			status = "conflict"
		}
		if n.SnapshotCount != s.SnapshotCount+1 || record.Status != status || record.Applications != s.Applications {
			return fmt.Errorf("incorrect read linearization")
		}
	}
	return nil
}

func TestCompleteProfiles(t *testing.T) {
	for _, p := range Profiles() {
		t.Run(p.Name, func(t *testing.T) {
			g := Explore(p)
			sinks := map[string]bool{}
			for i, r := range g.Rows {
				if err := checkState(p, r.State); err != nil {
					t.Fatalf("row %d: %v", i, err)
				}
				seen := map[string]bool{}
				for _, e := range r.Edges {
					if seen[e.Action] {
						t.Fatalf("ambiguous edge in row %d", i)
					}
					seen[e.Action] = true
					if err := checkEdge(p, r.State, e.Action, g.Rows[e.To].State); err != nil {
						t.Fatalf("row %d %s: %v", i, e.Action, err)
					}
				}
				if len(r.Edges) == 0 {
					sinks[r.Class] = true
				}
			}
			if len(sinks) == 0 {
				t.Fatal("no terminal profiles")
			}
			switch p.Name {
			case "matched-original-loss", "held-packet-refresh", "lost-negative-later-found", "admission-race":
				expected := map[string]bool{"not_found": true, "ordinary-continuation": true}
				if p.Name == "held-packet-refresh" {
					expected["final-duplicate-guard-failed"] = true
				}
				if !reflect.DeepEqual(sinks, expected) {
					t.Fatal(sinks)
				}
			case "spent-before-send":
				if !sinks["exhausted"] || !sinks["ordinary-continuation"] {
					t.Fatal(sinks)
				}
			case "existing-interaction-failure":
				if len(sinks) != 1 || !sinks["applied-after-failed-interaction"] {
					t.Fatal(sinks)
				}
			}
		})
	}
}

func profileNamed(name string) Profile {
	for _, p := range Profiles() {
		if p.Name == name {
			return p
		}
	}
	panic(name)
}

func TestCounterexamples(t *testing.T) {
	cases := []struct {
		name, profile string
		witness       func(State) bool
	}{
		{"application-after-Q", "held-packet-refresh", func(s State) bool { return s.Quiescent && s.Held && s.Applications == 0 }},
		{"absence-then-found", "held-packet-refresh", func(s State) bool {
			return s.Reads[0].Outcome == "not_found" && s.Reads[1].Outcome == "found" && s.Used
		}},
		{"early-import-late-duplicate", "held-packet-refresh", func(s State) bool { return s.Rejected && !s.Used && s.Settled && s.Published }},
		{"lost-absence-new-snapshot", "lost-negative-later-found", func(s State) bool {
			return s.Losses == 1 && s.Log[0].Status == "not_found" && s.Log[1].Status == "found" && s.Reads[0].Retained == 2 && s.Used
		}},
		{"no-import-entitlement", "read-without-import", func(s State) bool { return s.Reads[0].Selected && s.Reads[0].Outcome == "found" && s.Interaction == "" }},
		{"found-unsupported-policy", "selected-error-policy", func(s State) bool { return s.Reads[0].Adoption == "unsupported_policy" && !s.Published }},
		{"durable-exhaustion", "spent-before-send", func(s State) bool { return s.Losses == 2 && s.Reads[0].Adoption == "exhausted" && s.SnapshotCount == 0 }},
		{"query-alone-insufficient", "raw-keyed", func(s State) bool { return s.Reads[0].Selected && s.Reads[0].Outcome == "found" && !s.Published }},
		{"peer-and-revocation-during-read", "resolver-revocation-and-peer", func(s State) bool {
			return s.PeerRan && s.Reads[0].Stage == "entered" && s.Reads[0].Closed && !s.Reads[0].Drained
		}},
		{"equivalent-import", "two-resolvers", func(s State) bool {
			return s.Reads[1].Adoption == "adopted" && s.Reads[0].Adoption == "duplicate" && s.Settled
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, r := range Explore(profileNamed(c.profile)).Rows {
				if c.witness(r.State) {
					return
				}
			}
			t.Fatal("missing adverse witness")
		})
	}
}
