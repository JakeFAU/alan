package substitution

import "fmt"

func CallableJoinLocalConfig(c CallableJoinConfig) Config {
	return Config{Shape: "local", Request: c.Request, I: c.I - 1, R: c.R - 1, Depth: c.Depth,
		Local: true, Peer: true, Closure: true, EntryDepth: 2}
}

// CallableJoinLocalProjection starts after the prior J charge. Only accepted S
// admission spends q. The scratch local caller retains e's selected envelope;
// its PC is not O's PC or J's S-first await phase.
func CallableJoinLocalProjection(c CallableJoinConfig, s CallableJoinState) (State, error) {
	if !CallableJoinEntitled(c) || !s.InvocationAdmitted {
		return State{}, fmt.Errorf("local projection requires entitled post-J-admission state")
	}
	p := Initial(CallableJoinLocalConfig(c))
	for _, i := range []int{entry, helper} {
		p.Ops[i] = s.Inner.Component.Ops[i]
		p.PrivatePC[i] = s.Inner.Component.PrivatePC[i]
		p.PrivateRegisters[i] = s.Inner.Component.PrivateRegisters[i]
	}
	p.I, p.R = s.Inner.Component.I, s.Inner.Component.R
	other := s.Inner.Component.Ops[sibling]
	if other.Admission == 2 {
		return State{}, fmt.Errorf("refused original S cannot be erased as an accepted q debit")
	}
	if other.Admission == 1 {
		p.PeerTicket = spent
	}
	e := p.Ops[entry]
	if e.Ticket != possible {
		p.RootPC = await
	}
	if e.Consumed {
		p.RootPC, p.RootStatus = terminal, e.Status
		p.RootID, p.RootValue, p.RootPresent = e.ID, e.Value, e.Present
	}
	return p, nil
}

// CheckCallableJoinProjectionEdge checks the additive depth/debit/event mapping.
// The full constructor's common request and S-first selection gates restrict
// local caller actions; they do not invent a peer debit or component action.
func CheckCallableJoinProjectionEdge(
	c CallableJoinConfig,
	v Variant,
	s CallableJoinState,
	edge CallableJoinEdge,
) error {
	if !s.InvocationAdmitted {
		return nil // The separate public entry prefix establishes the initial pair.
	}
	before, err := CallableJoinLocalProjection(c, s)
	if err != nil {
		return err
	}
	after, err := CallableJoinLocalProjection(c, edge.Next)
	if err != nil {
		return err
	}
	f := edge.Event.Fact
	if f.Operation == "s" && f.Kind == "admit" {
		if f.Status != "accepted" || f.I != 1 || f.R != 1 {
			return fmt.Errorf("S admission is not the declared sole accepted debit")
		}
		f = Observation{Kind: "peer_debit", Operation: "q", I: 1, R: 1}
	} else if f.Operation == "e" || f.Operation == "h" {
		if f.Kind == "invoke_authorize" || f.Kind == "reply_authorize" {
			if before != after {
				return fmt.Errorf("authorization changed coarse local state")
			}
			return nil
		}
		if f.Parent == "j" {
			f.Parent = "root"
		}
		if f.Kind == "terminal_freeze" {
			f.Status = "" // Coarse K1 records Freeze; full event additionally names it.
		}
		if f.Kind == "private_step" {
			f = Observation{Kind: f.Kind, Operation: f.Operation, Role: f.Role}
		}
	} else {
		if before != after {
			return fmt.Errorf("nondebit S/J/caller step is not a local stutter")
		}
		return nil
	}
	local := CallableJoinLocalConfig(c)
	local.Closure = false
	graph := CandidateGraph(v)
	if v == Reference {
		graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
	}
	for _, candidate := range graph(local, before) {
		if candidate.Event == f && candidate.Next == after && candidate.Environment == edge.Environment {
			return nil
		}
	}
	return fmt.Errorf("unaligned local event %s/%s", f.Operation, f.Kind)
}

// CheckCallableJoinLocalMaximum is all-mode, physical EntryDepth2/Depth3.
// Its caller must independently establish zero enabled successors.
func CheckCallableJoinLocalMaximum(c Config, v Variant, s State) error {
	profile := c.Shape == "local" && c.Local && c.Peer && !c.Closure &&
		c.EntryDepth == 2 && c.Depth == 3 && c.I == 3 && c.R == 3
	if !profile || v != Local && v != Helper {
		return fmt.Errorf("outside aligned all-mode local profile")
	}
	if err := ValidateCallableJoinConfig(CallableJoinConfig{Request: c.Request, I: 4, R: 4, Depth: 3}); err != nil {
		return err
	}
	e, h := s.Ops[entry], s.Ops[helper]
	charge, choice := 2, 1
	if v == Helper {
		charge, choice = 3, 2
		valid := h.Depth == 3 && h.Admission == 1 && h.RequestStage == 2 && h.Freeze == success &&
			h.Published && h.Delivered && h.Consumed && h.Status == "succeeded" &&
			h.Reason == "member_succeeded" && h.Present && h.ID == c.Request.ID && h.Value == 1-c.Request.Value
		if !valid {
			return fmt.Errorf("aligned helper lacks original correct selected outcome")
		}
	} else if h.Ticket != excluded || h.Admission != 0 || h.Published || h.Consumed {
		return fmt.Errorf("local path has helper work")
	}
	if e.Depth != 2 || s.Closed || s.PeerTicket != spent || s.I != charge || s.R != charge || s.E != 0 {
		return fmt.Errorf("aligned local depth/one-q debit/service condition failed")
	}
	if e.Input != c.Request || e.Admission != 1 || e.RequestStage != 2 || e.Choice != choice || e.FailureReady {
		return fmt.Errorf("candidate lacks original entitled request/path")
	}
	if c.Request.Mode == "pending" {
		valid := e.PC == pending && e.Freeze == none && e.Reply == 0 && !e.Published && !e.Delivered &&
			!e.Consumed && e.Status == "" && e.Reason == "" && !e.Present && e.ID == 0 && e.Value == 0 &&
			s.RootPC == await && s.RootStatus == ""
		if !valid {
			return fmt.Errorf("candidate has not reached original unmatched receive")
		}
		return nil
	}
	status, reason, freeze := "succeeded", "member_succeeded", success
	switch c.Request.Mode {
	case "fail":
		status, reason, freeze = "failed", "member_failed", failure
	case "absent":
		status, reason = "failed", "missing_reply"
	case "answer":
	default:
		return fmt.Errorf("unknown local mode")
	}
	valid := e.PC == terminal && e.Freeze == freeze && e.Published && e.Delivered && e.Consumed &&
		e.Status == status && e.Reason == reason && s.RootPC == terminal && s.RootStatus == status
	if !valid {
		return fmt.Errorf("candidate lacks exact completed mode envelope")
	}
	if c.Request.Mode == "answer" {
		if e.Reply != 2 || !e.Present || e.ID != c.Request.ID || e.Value != 1-c.Request.Value {
			return fmt.Errorf("answer lacks original correct present reply")
		}
	} else if e.Reply != 0 || e.Present || e.ID != 0 || e.Value != 0 {
		return fmt.Errorf("fail/absent invented a response")
	}
	return nil
}

type CallableJoinLocalQualification struct {
	Config                            Config
	Variant                           Variant
	Relation                          RelationResult
	Progress                          Census
	RankMinimum, RankMaximum, RankCap int
	Maxima                            []ConsumerReasonLocalMaximum
	Failures                          []string
}

// QualifyCallableJoinLocal is a post-review Task3b operation, not a construction
// check. It runs the old relation on the new physical local environment and a
// separate open all-mode progress search. No historical result is inherited.
func QualifyCallableJoinLocal(c CallableJoinConfig, v Variant) (CallableJoinLocalQualification, error) {
	r := CallableJoinLocalQualification{Config: CallableJoinLocalConfig(c), Variant: v,
		RankCap: ConsumerReasonLocalRankCap, Maxima: []ConsumerReasonLocalMaximum{}, Failures: []string{}}
	if !CallableJoinEntitled(c) || v != Local && v != Helper {
		return r, fmt.Errorf("outside aligned replacement qualification profile")
	}
	r.Relation = CheckLocalRelation(r.Config, CandidateGraph(v))
	open := r.Config
	open.Closure = false
	graph := CandidateGraph(v)
	r.RankMinimum, r.RankMaximum = graphRank(Initial(open)), graphRank(Initial(open))
	r.Progress = ExploreGraph(open, func(c Config, s State) []Edge {
		edges := graph(c, s)
		for _, edge := range edges {
			rank := graphRank(edge.Next)
			r.RankMinimum, r.RankMaximum = min(r.RankMinimum, rank), max(r.RankMaximum, rank)
			if rank > r.RankCap {
				r.Failures = append(r.Failures, "aligned local rank exceeds cap")
			}
		}
		if len(edges) == 0 {
			m := ConsumerReasonLocalMaximum{State: s}
			if err := CheckCallableJoinLocalMaximum(c, v, s); err != nil {
				m.Error = err.Error()
				r.Failures = append(r.Failures, err.Error())
			}
			r.Maxima = append(r.Maxima, m)
		}
		return edges
	})
	if r.Relation.Failures != 0 || r.Relation.Initials != 1 {
		r.Failures = append(r.Failures, "aligned local relation failed")
	}
	p := r.Progress
	if p.RankFailures != 0 || p.Cycles != 0 || p.Cutoffs != 0 || p.Stuck != 0 ||
		len(p.Counterexamples) != 0 || len(r.Maxima) == 0 {
		r.Failures = append(r.Failures, "aligned open progress/rank/safety failed")
	}
	return r, nil
}
