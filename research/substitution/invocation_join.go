package substitution

import "fmt"

// JoinConfig names the bounded C_inv environment. Effects are zero, there is one
// fixed parent, generations/grants are stable, and cancellation is excluded.
type JoinConfig struct {
	Request     Request
	I, R, Depth int
}

// JoinEvent is a normalized original fact, in a causally consistent order.
// Authorization booleans describe its retained source snapshot. ReplyBound means
// protocol/message/entry-member/caller-subject/caller-generation/obligation all
// match the original invocation binding. Authentication of those raw fields and
// the single e/h/s identity renaming happens outside this pure oracle.
type JoinEvent struct {
	Fact                               Observation
	SourceOpen, SourceRetired          bool
	RevisionMatches, GenerationMatches bool
	ReplyBound                         bool
}

// JoinState retains the old component representation but replaces its caller.
// ParentStage: 0 commit e, 1 commit s, 2 first await, 3 second await, 4 decide,
// 5 End, 6 terminal. JoinSuccessors retains the historical e-then-s policy;
// ConsumerPolicySuccessors selects s then e. Root publication follows End.
// Authorization: 0 unresolved, 1 allowed, 2 refused; first dispositions persist.
type JoinState struct {
	Component                                   State
	ParentStage                                 int
	Decision, ParentStatus                      string
	RootPublished                               bool
	InvocationAuthorization, ReplyAuthorization [slots]int
}

type JoinEdge struct {
	Event       JoinEvent
	Next        JoinState
	Environment bool
}

// JoinRankCap follows from three operations with stage sum <=27 each, ledger
// <=6, parent stage <=6, six authorization dispositions <=2, publication <=1,
// and candidate/helper private PCs <=15+6. Unused wrapper/root fields are fixed.
const JoinRankCap = 100*(3*27+6+6+6*2+1) + 15 + 6

func ValidateJoinConfig(c JoinConfig) error {
	if c.Request.ID < 1 || c.Request.ID > 2 || c.Request.Value < 0 || c.Request.Value > 1 {
		return fmt.Errorf("request outside id1/2,value0/1 domain")
	}
	switch c.Request.Mode {
	case "answer", "fail", "absent", "pending":
	default:
		return fmt.Errorf("unknown request mode %q", c.Request.Mode)
	}
	if c.I < 0 || c.I > 3 || c.R < 0 || c.R > 3 || c.Depth < 1 || c.Depth > 2 {
		return fmt.Errorf("limits outside I/R0..3,depth1..2 domain")
	}
	return nil
}

func JoinConfigurations() []JoinConfig {
	out := []JoinConfig{}
	for id := 1; id <= 2; id++ {
		for value := 0; value <= 1; value++ {
			out = append(out, JoinConfig{Request: Request{ID: id, Value: value, Mode: "answer"}, I: 3, R: 3, Depth: 2})
		}
	}
	return out
}

func JoinEntitled(c JoinConfig) bool {
	return ValidateJoinConfig(c) == nil && c.Request.Mode == "answer" && c.I == 3 && c.R == 3 && c.Depth == 2
}

func joinCoreConfig(c JoinConfig) Config {
	return Config{Shape: "local", Request: c.Request, I: c.I, R: c.R, Depth: c.Depth, Local: true, EntryDepth: 1}
}

func JoinInitial(c JoinConfig) JoinState {
	s := Initial(joinCoreConfig(c))
	s.Ops[sibling] = Operation{Role: "s", Source: -1, Depth: 1, Input: Request{ID: c.Request.ID, Value: c.Request.Value}}
	return JoinState{Component: s}
}

func joinMeta(s State, i int, kind string) Observation {
	o := s.Ops[i]
	parent := "root"
	if i == helper {
		parent = "e"
	}
	return Observation{Kind: kind, Operation: name(i), Parent: parent, Role: o.Role, Request: o.Input}
}

func joinSourceOpen(s JoinState, i int) bool {
	if i == helper {
		return s.Component.Ops[entry].Freeze == none
	}
	return s.ParentStatus == ""
}

func joinCorrect(c JoinConfig, e, s Operation) bool {
	return e.Consumed && s.Consumed && e.Status == "succeeded" && e.Present &&
		e.ID == c.Request.ID && e.Value == 1-c.Request.Value &&
		s.Status == "succeeded" && s.Present && s.ID == c.Request.ID && s.Value == 0
}

// JoinSuccessors uses inherited K1/A/B component mechanics, replacing only old
// caller transitions and splitting authorization from admission. It neither
// explores the product nor suppresses a failure/refusal transition.
func JoinSuccessors(c JoinConfig, v Variant, s JoinState) []JoinEdge {
	return joinSuccessors(c, v, s, consumeCandidateFirst)
}

// joinSuccessors shares protocol mechanics across the two fixed parents.
// Only original-outcome receive gates depend on their consumption order.
func joinSuccessors(c JoinConfig, v Variant, s JoinState, order joinConsumption) []JoinEdge {
	first, second := entry, sibling
	if order == consumeSFirst {
		first, second = sibling, entry
	}
	out := []JoinEdge{}
	for _, i := range []int{entry, helper, sibling} {
		o := s.Component.Ops[i]
		if o.Ticket == committed && o.Admission == 0 && s.InvocationAuthorization[i] == 0 {
			n := s
			n.InvocationAuthorization[i] = 1
			f := joinMeta(s.Component, i, "invoke_authorize")
			f.Status, f.Reason = "allowed", "authorized"
			open := joinSourceOpen(s, i)
			if !open {
				n.InvocationAuthorization[i] = 2
				f.Status, f.Reason = "refused", "source_closed"
			}
			out = append(out, JoinEdge{Event: JoinEvent{Fact: f, SourceOpen: open, RevisionMatches: true, GenerationMatches: true}, Next: n, Environment: true})
		}
		if o.Reply == 1 && s.ReplyAuthorization[i] == 0 {
			n := s
			n.ReplyAuthorization[i] = 1
			f := joinMeta(s.Component, i, "reply_authorize")
			f.Status, f.Reason = "allowed", "authorized"
			out = append(out, JoinEdge{Event: JoinEvent{Fact: f, SourceOpen: o.Freeze == none, RevisionMatches: true, GenerationMatches: true, ReplyBound: true}, Next: n, Environment: true})
		}
	}

	view := s.Component
	if s.ParentStage == 2 || s.ParentStage == 3 {
		view.RootPC = await
	}
	graph := CandidateGraph(v)
	if v == Reference {
		graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
	}
	for _, edge := range graph(joinCoreConfig(c), view) {
		i := -1
		for _, j := range []int{entry, helper, sibling} {
			if edge.Event.Operation == name(j) {
				i = j
			}
		}
		if i < 0 {
			continue // No old peer/closure/wrapper action belongs to this context.
		}
		n := s
		n.Component = edge.Next
		// Old caller fields are scratch used solely to ask the reused graph for
		// its original-handle receive. No old caller decision survives adaptation.
		n.Component.RootPC, n.Component.RootStatus = 0, ""
		n.Component.RootID, n.Component.RootValue, n.Component.RootPresent = 0, 0, false
		f := edge.Event
		f.Parent, f.Role, f.Request = joinMeta(s.Component, i, f.Kind).Parent, s.Component.Ops[i].Role, s.Component.Ops[i].Input
		switch f.Kind {
		case "invoke_commit":
			if i == entry {
				if s.ParentStage != 0 {
					continue
				}
				n.ParentStage = 1
				n.Component.Ops[sibling].Enabled = true
			}
			if i == sibling {
				if s.ParentStage != 1 {
					continue
				}
				n.ParentStage = 2
			}
		case "admit":
			if s.InvocationAuthorization[i] == 0 {
				continue
			}
			if s.InvocationAuthorization[i] == 2 || s.ParentStatus != "" {
				n.Component = s.Component
				n.Component.Ops[i].Admission = 2
				reason := "scope_closed"
				if s.InvocationAuthorization[i] == 2 {
					reason = "source_denied:source_closed"
				}
				n.Component.Ops[i].AdmissionReason = reason
				exclude(&n.Component, i)
				f.Status, f.Reason, f.I, f.R = "rejected", reason, 0, 0
			}
		case "reply_admit":
			if s.ReplyAuthorization[i] == 0 {
				continue
			}
		case "terminal_freeze":
			f.Status = "succeeded"
			if n.Component.Ops[i].Freeze == failure {
				f.Status = "failed"
			}
		case "outcome_consume":
			if i == first {
				if s.ParentStage != 2 {
					continue
				}
				n.ParentStage = 3
			}
			if i == second {
				if s.ParentStage != 3 {
					continue
				}
				n.ParentStage = 4
			}
		}
		out = append(out, JoinEdge{Event: JoinEvent{Fact: f}, Next: n, Environment: edge.Environment})
	}
	if s.ParentStage == 4 {
		n := s
		n.ParentStage, n.Decision = 5, "failed"
		if joinCorrect(c, s.Component.Ops[entry], s.Component.Ops[sibling]) {
			n.Decision = "succeeded"
		}
		out = append(out, JoinEdge{Event: JoinEvent{Fact: Observation{Kind: "parent_decide", Operation: "root", Role: "parent", Request: c.Request, Status: n.Decision}}, Next: n})
	}
	if s.ParentStage == 5 {
		n := s
		n.ParentStage, n.ParentStatus = 6, s.Decision
		out = append(out, JoinEdge{Event: JoinEvent{Fact: Observation{Kind: "parent_terminal", Operation: "root", Role: "parent", Request: c.Request, Status: n.ParentStatus}}, Next: n})
	}
	if s.ParentStage == 6 && !s.RootPublished {
		n := s
		n.RootPublished = true
		out = append(out, JoinEdge{Event: JoinEvent{Fact: Observation{Kind: "root_publish", Operation: "root", Role: "parent", Request: c.Request, Status: s.ParentStatus, Reason: "member_" + s.ParentStatus}}, Next: n, Environment: true})
	}
	return out
}

// JoinRank is intrinsic. Every semantic stage is nondecreasing. The inherited
// A13/B16/helper7 private branch/Jump targets also strictly advance when a private
// instruction changes; they are bounded by 15+6. Payloads/history are not rank.
func JoinRank(s JoinState) int {
	stage := s.Component.I + s.Component.R + s.Component.E + s.ParentStage
	if s.RootPublished {
		stage++
	}
	for _, i := range []int{entry, helper, sibling} {
		o := s.Component.Ops[i]
		stage += o.Ticket + o.Admission + o.RequestStage + o.PC + o.Choice + o.Reply + o.Freeze
		for _, flag := range []bool{o.Enabled, o.FailureReady, o.Published, o.Delivered, o.Consumed} {
			if flag {
				stage++
			}
		}
		stage += s.InvocationAuthorization[i] + s.ReplyAuthorization[i]
	}
	return 100*stage + s.Component.PrivatePC[entry] + s.Component.PrivatePC[helper]
}

// CheckJoinState checks representation/resource invariants, not K1 payload
// conformance or G. Keeping these apart permits honest faulty-component traces.
func CheckJoinState(c JoinConfig, s JoinState) error {
	return checkJoinState(c, s, consumeCandidateFirst)
}

func checkJoinState(c JoinConfig, s JoinState, order joinConsumption) error {
	if err := ValidateJoinConfig(c); err != nil {
		return err
	}
	base := JoinInitial(c).Component
	core := s.Component
	if core.Closed || core.RootPC != 0 || core.RootStatus != "" || core.RootID != 0 || core.RootValue != 0 || core.RootPresent || core.PeerPC != terminal || core.PeerStatus != "" || core.PeerTicket != excluded || core.E != 0 {
		return fmt.Errorf("unused caller/closure/effect state changed")
	}
	if core.Ops[wrapper] != base.Ops[wrapper] || core.PrivatePC[wrapper] != 0 || core.PrivatePC[sibling] != 0 || core.PrivateRegisters[wrapper] != [8]int{} || core.PrivateRegisters[sibling] != [8]int{} || s.InvocationAuthorization[wrapper] != 0 || s.ReplyAuthorization[wrapper] != 0 {
		return fmt.Errorf("unused wrapper or S private state changed")
	}
	if s.ParentStage < 0 || s.ParentStage > 6 || core.PrivatePC[entry] < 0 || core.PrivatePC[entry] > 15 || core.PrivatePC[helper] < 0 || core.PrivatePC[helper] > 6 {
		return fmt.Errorf("parent/private control outside bounds")
	}
	accepted, demand := 0, 0
	for _, i := range []int{entry, helper, sibling} {
		o, original := core.Ops[i], base.Ops[i]
		if o.Role != original.Role || o.Source != original.Source || o.Depth != original.Depth || o.Input != original.Input {
			return fmt.Errorf("operation %s identity/request binding changed", name(i))
		}
		for _, b := range [][2]int{{o.Ticket, 3}, {o.Admission, 2}, {o.RequestStage, 2}, {o.PC, 8}, {o.Choice, 2}, {o.Reply, 3}, {o.Freeze, 2}, {s.InvocationAuthorization[i], 2}, {s.ReplyAuthorization[i], 2}} {
			if b[0] < 0 || b[0] > b[1] {
				return fmt.Errorf("operation %s stage outside finite domain", name(i))
			}
		}
		for _, payload := range [][2]int{{o.ReturnedID, 2}, {o.ReturnedValue, 1}, {o.ReplyID, 2}, {o.ReplyValue, 1}, {o.ID, 2}, {o.Value, 1}} {
			if payload[0] < 0 || payload[0] > payload[1] {
				return fmt.Errorf("operation payload outside finite domain")
			}
		}
		if !joinFiniteString(o.Status, "", "succeeded", "failed", "rejected") ||
			!joinFiniteString(o.Reason, "", "member_succeeded", "member_failed", "missing_reply", "depth_limit", "invocation_limit", "created_run_limit", "source_closed", "source_denied:source_closed", "scope_closed") ||
			!joinFiniteString(o.AdmissionReason, "", "depth_limit", "invocation_limit", "created_run_limit", "source_closed", "source_denied:source_closed", "scope_closed") || o.ReplyReason != "" {
			return fmt.Errorf("operation status/reason outside finite domain")
		}
		for _, value := range core.PrivateRegisters[i] {
			if value < 0 || value > 2 {
				return fmt.Errorf("private register outside 0..2")
			}
		}
		if o.Admission == 1 {
			accepted++
			if o.Depth > c.Depth || o.Ticket != spent {
				return fmt.Errorf("accepted operation depth/ticket mismatch")
			}
		}
		if o.Ticket == possible || o.Ticket == committed {
			demand++
		}
		if o.Consumed && !o.Delivered || o.Delivered && !o.Published || o.Admission != 0 && s.InvocationAuthorization[i] == 0 || o.Reply >= 2 && s.ReplyAuthorization[i] == 0 {
			return fmt.Errorf("operation %s missing protocol predecessor", name(i))
		}
		if o.Present && !o.Published || !o.Present && (o.ID != 0 || o.Value != 0) {
			return fmt.Errorf("publication/presence mismatch")
		}
	}
	if core.I != accepted || core.R != accepted || core.I > c.I || core.R > c.R {
		return fmt.Errorf("ledger differs from accepted singleton admissions")
	}
	if JoinEntitled(c) && (core.I+demand > c.I || core.R+demand > c.R) {
		return fmt.Errorf("spent plus remaining demand exceeds entitlement")
	}
	first, second := entry, sibling
	if order == consumeSFirst {
		first, second = sibling, entry
	}
	if s.ParentStage >= 2 && core.Ops[sibling].Ticket == possible || s.ParentStage >= 3 && !core.Ops[first].Consumed || s.ParentStage >= 4 && !core.Ops[second].Consumed {
		return fmt.Errorf("parent skipped commitment/consumption")
	}
	if s.ParentStage < 5 && s.Decision != "" || s.ParentStage >= 5 && s.Decision != "succeeded" && s.Decision != "failed" || s.ParentStage < 6 && s.ParentStatus != "" || s.ParentStage == 6 && s.ParentStatus != s.Decision || s.RootPublished && s.ParentStage != 6 {
		return fmt.Errorf("parent decision/terminal/publication mismatch")
	}
	if JoinRank(s) > JoinRankCap {
		return fmt.Errorf("rank exceeds declared cap")
	}
	return nil
}

func joinFiniteString(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
