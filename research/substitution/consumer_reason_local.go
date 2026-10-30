package substitution

import "fmt"

// ConsumerReasonLocalConfig names the exact reused local environment. Closure
// belongs to the relation's adversary; open progress sets Closure=false separately.
func ConsumerReasonLocalConfig(c JoinConfig) Config {
	return Config{
		Shape: "local", Request: c.Request, I: 3, R: 3, Depth: 2,
		Local: true, Peer: true, Closure: true, EntryDepth: 1,
	}
}

// CheckConsumerReasonLocalMaximum supplements the old answer-only completion
// test with exact all-mode outcomes and helper/pending obligations. The caller
// must establish maximality by exhausting successors, not by truncating a trace.
func CheckConsumerReasonLocalMaximum(c Config, v Variant, s State) error {
	profile := c.Local && c.Shape == "local" && c.Peer && !c.Closure &&
		c.EntryDepth == 1 && c.Depth == 2 && c.I == 3 && c.R == 3
	if !profile || v != Local && v != Helper {
		return fmt.Errorf("outside all-mode local open-progress profile")
	}
	if err := ValidateJoinConfig(JoinConfig{Request: c.Request, I: c.I, R: c.R, Depth: c.Depth}); err != nil {
		return err
	}
	e, h := s.Ops[entry], s.Ops[helper]
	charge, choice := 2, 1
	if v == Helper {
		charge, choice = 3, 2
		correctHelper := h.Admission == 1 && h.RequestStage == 2 && h.Freeze == success &&
			h.Published && h.Delivered && h.Consumed && h.Status == "succeeded" &&
			h.Reason == "member_succeeded" && h.Present && h.ID == c.Request.ID && h.Value == 1-c.Request.Value
		if !correctHelper {
			return fmt.Errorf("helper progress lacks original correct consumed outcome")
		}
	} else if h.Ticket != excluded || h.Admission != 0 || h.Published || h.Consumed {
		return fmt.Errorf("local branch has unexpected helper work")
	}
	if s.Closed || s.PeerTicket != spent || s.I != charge || s.R != charge || s.E != 0 {
		return fmt.Errorf("local open progress violates service/one-charge resource assumptions")
	}
	if e.Admission != 1 || e.RequestStage != 2 || e.Choice != choice || e.FailureReady {
		return fmt.Errorf("candidate lacks original requested and entitled local/helper path")
	}
	if c.Request.Mode == "pending" {
		unmatched := e.PC == pending && e.Freeze == none && e.Reply == 0 && !e.Published &&
			!e.Delivered && !e.Consumed && e.Status == "" && e.Reason == "" && !e.Present
		if !unmatched || s.RootPC != await || s.RootStatus != "" {
			return fmt.Errorf("candidate must reach the designated unmatched receive without terminal outcome")
		}
		return nil
	}
	status, reason, freeze := "succeeded", "member_succeeded", success
	switch c.Request.Mode {
	case "fail":
		status, reason, freeze = "failed", "member_failed", failure
	case "absent":
		status, reason = "failed", "missing_reply"
	}
	completed := e.PC == terminal && e.Freeze == freeze && e.Published && e.Delivered && e.Consumed &&
		e.Status == status && e.Reason == reason && s.RootPC == terminal && s.RootStatus == status
	if !completed {
		return fmt.Errorf("candidate did not reach and deliver its exact mode envelope")
	}
	if c.Request.Mode == "answer" {
		if e.Reply != 2 || !e.Present || e.ID != c.Request.ID || e.Value != 1-c.Request.Value {
			return fmt.Errorf("answer progress lacks its original correct present reply")
		}
	} else if e.Reply != 0 || e.Present || e.ID != 0 || e.Value != 0 {
		return fmt.Errorf("fail/absent progress invented a reply or payload")
	}
	return nil
}

type ConsumerReasonLocalMaximum struct {
	State State
	Error string
}

// ConsumerReasonLocalRankCap bounds unchanged open local graphRank: two live
// operations <=27 each, unused tickets <=6, ledger <=6, root/peer PCs <=7 each,
// peer ticket <=3, private PCs <=15+6. Register[entry][7] stays zero in A/B.
const ConsumerReasonLocalRankCap = 100*(2*27+6+6+7+7+3) + 15 + 6

type ConsumerReasonLocalQualification struct {
	Config                            Config
	Entitled                          bool
	RankMinimum, RankMaximum, RankCap int
	Variant                           Variant
	Relation                          RelationResult
	Progress                          Census
	Maxima                            []ConsumerReasonLocalMaximum
	Failures                          []string
}

// QualifyConsumerReasonLocal is gated with the product census. It reuses the
// original exact relation and explorer, wrapping only successor observation to
// retain and assert every actual maximum. No old primitive or domain is changed.
func QualifyConsumerReasonLocal(c JoinConfig, v Variant) (ConsumerReasonLocalQualification, error) {
	result := ConsumerReasonLocalQualification{
		Config: ConsumerReasonLocalConfig(c), Variant: v, Entitled: ConsumerReasonEntitled(c),
		RankCap: ConsumerReasonLocalRankCap,
		Maxima:  []ConsumerReasonLocalMaximum{}, Failures: []string{},
	}
	if !ConsumerReasonEntitled(c) || v != Local && v != Helper {
		return result, fmt.Errorf("outside declared all-mode local qualification profile")
	}
	result.Relation = CheckLocalRelation(result.Config, CandidateGraph(v))
	open := result.Config
	open.Closure = false
	graph := CandidateGraph(v)
	result.RankMinimum, result.RankMaximum = graphRank(Initial(open)), graphRank(Initial(open))
	result.Progress = ExploreGraph(open, func(c Config, s State) []Edge {
		edges := graph(c, s)
		for _, edge := range edges {
			rank := graphRank(edge.Next)
			result.RankMinimum = min(result.RankMinimum, rank)
			result.RankMaximum = max(result.RankMaximum, rank)
			if rank > result.RankCap {
				result.Failures = append(result.Failures, "local rank exceeds finite cap")
			}
		}
		if len(edges) == 0 {
			maximum := ConsumerReasonLocalMaximum{State: s}
			if err := CheckConsumerReasonLocalMaximum(c, v, s); err != nil {
				maximum.Error = err.Error()
				result.Failures = append(result.Failures, err.Error())
			}
			result.Maxima = append(result.Maxima, maximum)
		}
		return edges
	})
	r, p := result.Relation, result.Progress
	if r.Initials != 1 || r.Failures != 0 || len(r.Counterexamples) != 0 {
		result.Failures = append(result.Failures, "local relation failed")
	}
	if p.RankFailures != 0 || p.Cycles != 0 || p.Cutoffs != 0 || p.Stuck != 0 || len(p.Counterexamples) != 0 {
		result.Failures = append(result.Failures, "local exhaustive progress/safety failed")
	}
	if len(result.Maxima) == 0 {
		result.Failures = append(result.Failures, "local exploration has no checked maximum")
	}
	return result, nil
}
