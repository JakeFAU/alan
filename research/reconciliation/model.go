// Package reconciliation defines a finite protocol independent of the executor.
package reconciliation

import "fmt"

// Each profile fixes an original transport schedule, subsequent read service,
// grants and the finite number of owner losses that the environment can choose.
type Profile struct {
	Name, Origin, Policy, Service, Loss                string
	Queries                                            int
	Concurrent, ReadOnly, BadTarget, CloseReader, Peer bool
}

func Profiles() []Profile {
	return []Profile{
		{Name: "admission-race", Origin: "race", Queries: 1},
		{Name: "matched-original-loss", Origin: "paired", Queries: 1},
		{Name: "held-packet-refresh", Origin: "delayed", Queries: 2},
		{Name: "lost-negative-later-found", Origin: "delayed", Queries: 1, Loss: "snapshot"},
		{Name: "spent-before-send", Origin: "applied", Queries: 1, Loss: "exhaust"},
		{Name: "retention-cut", Origin: "applied", Queries: 1, Loss: "retained"},
		{Name: "pre-import-cut", Origin: "applied", Queries: 1, Loss: "returned"},
		{Name: "existing-interaction-failure", Origin: "failed", Queries: 1},
		{Name: "selected-error-policy", Origin: "applied", Policy: "selected-error", Queries: 1},
		{Name: "unknown-policy", Origin: "applied", Policy: "unknown", Queries: 1},
		{Name: "read-without-import", Origin: "applied", ReadOnly: true, Queries: 1},
		{Name: "raw-keyed", Origin: "applied", Policy: "raw-keyed", Queries: 1},
		{Name: "raw-non-idempotent", Origin: "applied", Policy: "raw-plain", Queries: 1},
		{Name: "ambiguous-raw", Origin: "twice", Policy: "raw-plain", Queries: 1},
		{Name: "unbound-target", Origin: "applied", BadTarget: true, Queries: 1},
		{Name: "endpoint-unavailable", Origin: "applied", Service: "unavailable", Queries: 1},
		{Name: "foreign-snapshot", Origin: "applied", Service: "foreign", Queries: 1},
		{Name: "malformed-response", Origin: "applied", Service: "malformed", Queries: 1},
		{Name: "two-resolvers", Origin: "applied", Concurrent: true, Queries: 2},
		{Name: "resolver-revocation-and-peer", Origin: "applied", CloseReader: true, Peer: true, Queries: 1},
	}
}

// Spent and retained fields survive Open; Active and Acquired describe a
// particular owner's opportunity. A snapshot number refers into the audit log.
type Query struct {
	Stage                       string
	Spent                       int
	BoundTo, Acquired, Retained int
	Outcome, Adoption           string
	Closed, Drained, Selected   bool
}

type Snapshot struct {
	Target, Query, Ordinal, Applications int
	Status                               string
}

type State struct {
	Original, Interaction                                                       string
	Revoked, Quiescent, Held, Dead                                              bool
	OldOwnerLost                                                                bool
	Losses, Applications, SnapshotCount                                         int
	Log                                                                         [4]Snapshot
	Reads                                                                       [2]Query
	Target, OriginalDebits, ReadDebits, ReconciliationSource                    int
	Settled, OriginalFailureSelected, Published, OutcomeSelected, Used, PeerRan bool
}

func Initial() State {
	return State{Original: "consider", Target: 1, OriginalDebits: 1}
}

type Step struct {
	Action string
	Next   State
}
type addStep func(string, State)

func Outgoing(p Profile, s State) []Step {
	out := []Step{}
	add := func(a string, n State) { out = append(out, Step{a, n}) }
	// The packet is external. Its application is independent even of a dead Store.
	if s.Held {
		n := s
		n.Held = false
		n.Applications++
		add("world.apply", n)
	}
	if s.Dead {
		n := s
		n.Dead = false
		add("owner.open", n)
		return out
	}
	originalSteps(p, s, add)
	if !s.Quiescent {
		return out
	}
	if p.Peer && !s.PeerRan {
		n := s
		n.PeerRan = true
		add("peer.commit", n)
	}
	for i := 0; i < p.Queries; i++ {
		readSteps(p, s, i, add)
	}
	if s.Interaction != "" && !s.Settled {
		n := s
		n.Settled = true
		add("original.settle", n)
	}
	// The captured aggregate needs an original result. The production contract
	// permits publication eligibility before the distinct settlement commit.
	if s.Interaction != "" && !s.Published {
		n := s
		n.Published = true
		add("aggregate.publish", n)
	}
	if s.Published && !s.OutcomeSelected {
		n := s
		n.OutcomeSelected = true
		add("parent.select-original", n)
	}
	if s.OutcomeSelected && !s.Used {
		for i := 0; i < p.Queries; i++ {
			q := s.Reads[i]
			if q.Selected && q.Outcome == "found" && (q.Adoption == "adopted" || s.Interaction == "transport_error" && q.Adoption == "existing") {
				n := s
				n.Used = true
				add("parent.use-correlated-evidence", n)
				break
			}
		}
	}
	return out
}

func originalSteps(p Profile, s State, add addStep) {
	// Profiles select controlled cut locations. The race profile permits R
	// on either side of A; paired loss permits death at either local cut.
	revoke := s.Original == "handled" || p.Origin == "race" ||
		p.Origin == "paired" && (s.Original == "admitted" || s.Original == "returned") ||
		p.Origin == "delayed" && s.Original == "packet"
	if !s.Revoked && revoke && (p.Origin != "failed" || s.OriginalFailureSelected) {
		n := s
		n.Revoked = true
		add("original.revoke", n)
	}
	switch s.Original {
	case "consider":
		n := s
		if s.Revoked {
			n.Original = "handled"
			add("original.refuse", n)
		} else {
			n.Original = "admitted"
			add("original.admit", n)
		}
	case "admitted":
		if p.Origin != "paired" || !s.Revoked {
			n := s
			n.Original = "entered"
			add("original.enter", n)
		}
	case "entered":
		n := s
		n.Original = "packet"
		n.Held = true
		add("world.accept", n)
	case "packet":
		if !s.Held && p.Origin != "delayed" {
			n := s
			if p.Origin == "twice" && s.Applications == 1 {
				// A second genuine sender is the raw multiplicity ablation.
				n.Held = true
				add("world.accept-second-original", n)
			} else {
				n.Original = "returned"
				add("original.return", n)
			}
		}
	case "returned":
		if p.Origin != "paired" {
			n := s
			n.Original = "handled"
			if p.Origin == "failed" {
				n.Interaction = "transport_error"
				n.Original = "failure-recorded"
			}
			add("original.handle", n)
		}
	case "failure-recorded":
		n := s
		n.Settled = true
		n.Original = "handled"
		add("original.settle-failure", n)
	case "handled":
		if p.Origin == "failed" && !s.OriginalFailureSelected {
			n := s
			n.OriginalFailureSelected = true
			add("issuer.select-failure", n)
		}
		if s.Revoked && !s.Quiescent {
			n := s
			n.Quiescent = true
			add("original.Q", n)
		}
	}
	if !s.OldOwnerLost && s.Revoked && (p.Origin == "paired" && (s.Original == "admitted" || s.Original == "returned") || p.Origin == "delayed" && s.Original == "packet") {
		n := s
		n.OldOwnerLost = true
		n.Dead = true
		n.Original = "handled"
		n.Quiescent = false
		add("owner.kill-original", n)
	}
}

func readSteps(p Profile, s State, i int, add addStep) {
	q := s.Reads[i]
	step := func(label string, n State) { add(fmt.Sprintf("q%d.%s", i+1, label), n) }
	next := func(label, stage string) { n := s; n.Reads[i].Stage = stage; step(label, n) }
	if p.CloseReader && !q.Closed && q.Stage != "" {
		n := s
		n.Reads[i].Closed = true
		step("revoke", n)
	}
	if q.Closed && !q.Drained && (q.Stage == "handled" || q.Stage == "ready") {
		n := s
		n.Reads[i].Drained = true
		step("Q", n)
	}
	switch q.Stage {
	case "":
		if i == 0 || p.Concurrent && s.Reads[i-1].Stage != "" || !p.Concurrent && s.Reads[i-1].Selected {
			n := s
			n.ReadDebits++
			n.Reads[i].Stage = "ready"
			step("request", n)
		}
	case "ready":
		if q.Closed || p.BadTarget || q.Spent == 2 {
			n := s
			r := &n.Reads[i]
			r.Stage = "retained"
			r.Outcome = "refused"
			if q.Closed {
				r.Adoption = "resolver_closed"
			} else if p.BadTarget {
				r.Adoption = "unbound"
			} else {
				r.Adoption = "exhausted"
			}
			step("refuse", n)
		} else {
			n := s
			n.Reads[i].BoundTo = s.Target
			n.Reads[i].Spent++
			n.Reads[i].Stage = "admitted"
			step("spend-and-admit", n)
		}
	case "admitted":
		next("enter", "entered")
	case "entered":
		if p.Service == "unavailable" {
			next("transport-fails", "returned")
			break
		}
		n := s
		id := s.SnapshotCount + 1
		status := "not_found"
		if s.Applications == 1 {
			status = "found"
		} else if s.Applications > 1 {
			status = "conflict"
		}
		target := q.BoundTo
		if p.Service == "foreign" {
			target = 2
		}
		n.Log[id-1] = Snapshot{Target: target, Query: i + 1, Ordinal: q.Spent, Applications: s.Applications, Status: status}
		n.SnapshotCount = id
		n.Reads[i].Acquired = id
		n.Reads[i].Stage = "snapshot"
		step("snapshot", n)
	case "snapshot":
		next("return", "returned")
	case "returned":
		n := s
		r := &n.Reads[i]
		r.Stage = "retained"
		r.Retained = r.Acquired
		r.Acquired = 0
		if q.Acquired == 0 || p.Service == "malformed" || s.Log[q.Acquired-1].Target != q.BoundTo {
			r.Retained = 0
			r.Outcome = "transport_error"
			if p.Service == "malformed" || p.Service == "foreign" {
				r.Outcome = "protocol_invalid"
			}
		} else {
			r.Outcome = s.Log[r.Retained-1].Status
			if r.Outcome == "found" {
				switch {
				case p.ReadOnly:
					r.Adoption = "denied"
				case p.Policy == "raw-keyed" || p.Policy == "raw-plain":
					r.Adoption = "evidence_only"
				case p.Policy != "":
					r.Adoption = "unsupported_policy"
				case s.Interaction == "transport_error":
					r.Adoption = "existing"
				case s.ReconciliationSource != 0:
					r.Adoption = "duplicate"
				default:
					r.Adoption = "adopted"
					n.ReconciliationSource = i + 1
					n.Interaction = "reconciled"
				}
			}
		}
		step("retain", n)
	case "retained":
		if q.Adoption != "adopted" && q.Adoption != "duplicate" || s.Settled {
			next("handle", "handled")
		}
	case "handled":
		if !q.Selected && !q.Closed {
			n := s
			n.Reads[i].Selected = true
			step("select-evidence", n)
		}
	}
	lose := p.Loss == "exhaust" && q.Stage == "admitted" && s.Losses < 2 ||
		s.Losses == 0 && (p.Loss == "snapshot" && q.Stage == "snapshot" || p.Loss == "returned" && q.Stage == "returned" || p.Loss == "retained" && q.Stage == "retained")
	if lose {
		n := s
		n.Dead = true
		n.Quiescent = false
		n.Losses++
		r := &n.Reads[i]
		r.Acquired = 0
		if q.Stage != "retained" {
			r.Stage = "ready"
		}
		step("kill-owner", n)
	}
}
