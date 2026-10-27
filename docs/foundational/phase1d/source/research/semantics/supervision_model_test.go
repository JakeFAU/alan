package semantics

import (
	"encoding/json"
	"fmt"
	"slices"
)

const (
	authorizeLifecycle kind = "AuthorizeLifecycle"
	decideLifecycle    kind = "DecideLifecycle"
	admitLifecycle     kind = "AdmitLifecycleResult"
	suspect            kind = "Suspect"
)

// Slots are ordinary finite values. The interpreter does not branch on their
// policy meanings; only the immutable fixture program does that.
const (
	childGeneration = iota
	classification
	spent
	limit
	acceptedResult
	successor
	revision
	scratch
)

type localData struct {
	Values  [8]int
	Names   [2]string
	Pending string
	Last    string
	Seen    []string
}
type lifecycleRequest struct {
	ID, Requester      string
	RequesterG         int
	Obligation, Target string
	ExpectedG, W       int
	Operation          kind
}
type lifecycleAuthorization struct {
	Request string
	Owner   string
	W       int
}
type lifecycleResult struct {
	Request   string
	Accepted  bool
	Reason    string
	Successor int
}
type runtimeNotice struct {
	Kind, Subject           string
	G, E                    int
	Logical, Classification string
	Result                  lifecycleResult
}
type noticeAdmission struct {
	Notice   runtimeNotice
	Bound    string
	Accepted bool
}
type lifecycleLedger struct {
	Authorized map[string]lifecycleAuthorization
	Results    map[string]lifecycleResult
	Notices    map[string]noticeAdmission
}
type lifecycleGrant struct {
	Requester, Target string
	Operation         kind
	W                 int
}
type policyContract struct {
	Child          string
	Limit          int
	AllowAbandoned bool
}
type supervisionContract struct {
	Grants   []lifecycleGrant
	Policies map[string]policyContract
}

func requestValue(raw string) lifecycleRequest {
	var v lifecycleRequest
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		panic(err)
	}
	return v
}
func noticeValue(raw string) runtimeNotice {
	var v runtimeNotice
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		panic(err)
	}
	return v
}
func ensureLifecycle(m *machine) {
	if m.Lifecycle == nil {
		m.Lifecycle = &lifecycleLedger{Authorized: map[string]lifecycleAuthorization{}, Results: map[string]lifecycleResult{}, Notices: map[string]noticeAdmission{}}
	}
}
func hasGrant(c config, q lifecycleRequest) bool {
	if c.Supervision == nil {
		return false
	}
	for _, g := range c.Supervision.Grants {
		if g.Requester == q.Requester && g.Target == q.Target && g.Operation == q.Operation && g.W == q.W {
			return true
		}
	}
	return false
}
func terminalAdmission(m *machine, e event) string {
	r := m.Runs[runID(e.A, e.G)]
	target := r.Outcome.Route
	if m.Subjects[target] == nil {
		return ""
	} // Preserve historical external notification routes.
	if !m.Config.Edges[e.A+">"+target] {
		return "observation permission"
	}
	n := runtimeNotice{Kind: "terminal", Subject: e.A, G: e.G, Logical: runID(e.A, e.G) + "/terminal", Classification: r.Outcome.Status}
	return admitNotice(m, "terminal:"+runID(e.A, e.G)+":"+e.ID, target, n)
}
func admitNotice(m *machine, id, target string, n runtimeNotice) string {
	ensureLifecycle(m)
	if old, ok := m.Lifecycle.Notices[id]; ok {
		if old.Notice != n {
			return "notice conflict"
		}
		return ""
	}
	s := m.Subjects[target]
	if s == nil {
		return "unknown observer"
	}
	bound := runID(target, s.Current)
	r := m.Runs[bound]
	active := !s.Retired && r.Projection.Status == "active"
	if active && len(r.Inbox) >= m.Config.Bounds.Inbox {
		return "capacity:inbox"
	}
	m.Lifecycle.Notices[id] = noticeAdmission{Notice: n, Bound: bound, Accepted: active}
	if active {
		r.Inbox[id] = observation{ID: id, Tag: noticeTag(n.Kind), Kind: n.Kind, Origin: "runtime", Sender: n.Subject, SenderG: n.G, Notice: canonical(n)}
	}
	return ""
}
func supervisionStep(m *machine, e event) (bool, string) {
	switch e.Kind {
	case suspect:
		r := m.Runs[runID(e.A, e.G)]
		if r == nil || r.Projection.Status != "active" {
			return true, "closed"
		}
		target := m.Config.Routes[e.A]
		if !m.Config.Edges[e.A+">"+target] {
			return true, "observation permission"
		}
		n := runtimeNotice{Kind: "suspicion", Subject: e.A, G: e.G, E: r.E, Logical: fmt.Sprintf("%s/e%d/suspicion", runID(e.A, e.G), r.E)}
		return true, admitNotice(m, "suspicion:"+n.Logical+":"+e.ID, target, n)
	case authorizeLifecycle:
		in := m.Intents[e.ID]
		if in == nil || in.Descriptor.Lifecycle == "" {
			return true, "uncommitted lifecycle intent"
		}
		d := in.Descriptor
		if !m.owned(e, d.A) {
			return true, "source ownership"
		}
		ensureLifecycle(m)
		if old, ok := m.Lifecycle.Authorized[e.ID]; ok {
			if old.Request != d.Lifecycle {
				return true, "descriptor conflict"
			}
			return true, ""
		}
		r, s := m.Runs[runID(d.A, d.G)], m.Subjects[d.A]
		if s.Retired || s.Current != d.G || r.Projection.Status != "active" || in.Suppressed {
			return true, "closed"
		}
		m.Lifecycle.Authorized[e.ID] = lifecycleAuthorization{Request: d.Lifecycle, Owner: e.Owner, W: e.W}
		return true, ""
	case decideLifecycle:
		if m.Lifecycle == nil {
			return true, "unauthorized lifecycle intent"
		}
		au, ok := m.Lifecycle.Authorized[e.ID]
		if !ok {
			return true, "unauthorized lifecycle intent"
		}
		q := requestValue(au.Request)
		if old, ok := m.Lifecycle.Results[e.ID]; ok && m.Config.Mutant != "operation-dedup" {
			if old.Request != au.Request {
				return true, "descriptor conflict"
			}
			return true, ""
		}
		result := lifecycleResult{Request: au.Request}
		s := m.Subjects[q.Target]
		switch {
		case m.Config.Mutant != "lifecycle-authority" && !hasGrant(m.Config, q):
			result.Reason = "authority"
		case s == nil:
			result.Reason = "unknown target"
		case s.W != q.W:
			result.Reason = "target ownership revision"
		case s.Current != q.ExpectedG:
			result.Reason = "expected generation"
		default:
			// This uses precisely the baseline authority transition. It makes no
			// recovery choice, and does not advance the requesting ordinary program.
			next, why := step(*m, event{Kind: q.Operation, A: q.Target, G: q.ExpectedG, Owner: s.Owner, W: q.W})
			if why != "" {
				result.Reason = why
			} else {
				*m = next
				result.Accepted = true
				if q.Operation == retry {
					result.Successor = m.Subjects[q.Target].Current
				}
			}
		}
		m.Lifecycle.Results[e.ID] = result
		return true, ""
	case admitLifecycle:
		if m.Lifecycle == nil {
			return true, "no lifecycle result"
		}
		result, ok := m.Lifecycle.Results[e.ID]
		if !ok {
			return true, "no lifecycle result"
		}
		q := requestValue(result.Request)
		n := runtimeNotice{Kind: "lifecycle", Subject: q.Target, G: q.ExpectedG, Logical: q.ID, Result: result}
		return true, admitNotice(m, "lifecycle:"+e.ID+":"+e.Origin, q.Requester, n)
	}
	return false, ""
}

// Generic local operators only read the selected run. The closed metadata
// projections below make protocol evidence representable in ordinary state.
func localInstruction(c config, r *run, p *proposal, in instruction, index int) (bool, string) {
	local := in.Mode == "local" || in.Mode == "copy" || in.Mode == "add" || in.Mode == "name" || in.Mode == "identity"
	typed := in.Mode == "terminal" || in.Mode == "observation" || in.Mode == "lifecycle"
	if !local && !typed {
		return false, ""
	}
	if p.After.Local == nil {
		p.After.Local = &localData{}
	}
	d := p.After.Local
	if in.Slot < 0 || in.Slot >= len(d.Values) || in.Source < 0 || in.Source >= len(d.Values) {
		return true, "local slot"
	}
	switch in.Op {
	case set:
		switch in.Mode {
		case "local":
			d.Values[in.Slot] = in.Value
		case "copy":
			d.Values[in.Slot] = d.Values[in.Source]
		case "add":
			d.Values[in.Slot] += in.Value
		case "name":
			if in.Slot >= len(d.Names) {
				return true, "name slot"
			}
			d.Names[in.Slot] = in.Tag
		case "identity":
			if in.Slot >= len(d.Names) {
				return true, "name slot"
			}
			d.Names[in.Slot] = fmt.Sprintf("%s/%s/init%d", runID(r.A, r.G), in.Key, r.N+1)
		default:
			return true, "local mode"
		}
	case branch:
		value := in.Value
		if in.FromReg {
			value = d.Values[in.Source]
		}
		if d.Values[in.Slot] == value {
			p.After.Frontier[index].PC = in.PC
		}
	case emit:
		if in.Mode != "lifecycle" || d.Pending != "" {
			return true, "pending lifecycle operation"
		}
		if d.Names[0] == "" || d.Names[1] == "" {
			return true, "uninitialized context"
		}
		if in.Tag != string(retry) && in.Tag != string(reactivate) && in.Tag != string(abandon) && in.Tag != string(cancel) && in.Tag != string(retire) {
			return true, "lifecycle operation"
		}
		id := intentID(r.A, r.G, r.N+1)
		q := lifecycleRequest{ID: id, Requester: r.A, RequesterG: r.G, Obligation: d.Names[1], Target: d.Names[0], ExpectedG: d.Values[childGeneration], W: d.Values[revision], Operation: kind(in.Tag)}
		p.Intent = &descriptor{Kind: emit, A: r.A, G: r.G, N: r.N + 1, Tag: in.Tag, Target: q.Target, TargetG: q.ExpectedG, Lifecycle: canonical(q)}
		d.Pending = id
		d.Values[in.Slot] += in.Delta // Generic local delta in the same atomic projection.
	case receive:
		o, ok := r.Inbox[p.Selection.Observation]
		if !ok || o.Consumed {
			return true, "waiting"
		}
		if o.Origin != "runtime" || o.Notice == "" {
			return true, "observation provenance"
		}
		n := noticeValue(o.Notice)
		if o.Tag != in.Tag || (n.Kind != in.Mode && !(in.Mode == "observation" && (n.Kind == "terminal" || n.Kind == "suspicion"))) {
			return true, "observation kind"
		}
		if in.Mode == "terminal" || in.Mode == "observation" {
			if c.Mutant != "failure-correlation" && (n.Subject != d.Names[0] || n.G != d.Values[childGeneration] || slices.Contains(d.Seen, n.Logical)) {
				return true, "terminal correlation"
			}
			if len(d.Seen) >= 8 {
				return true, "capacity:logical notifications"
			}
			d.Seen = append(d.Seen, n.Logical)
			if n.Kind == "suspicion" {
				if n.E < 1 {
					return true, "suspicion provenance"
				}
				d.Values[classification] = 5
			} else {
				switch n.Classification {
				case "succeeded":
					d.Values[classification] = 1
				case "failed":
					d.Values[classification] = 2
				case "abandoned":
					d.Values[classification] = 3
				case "cancelled":
					d.Values[classification] = 4
				default:
					return true, "terminal classification"
				}
			}
		} else {
			q := requestValue(n.Result.Request)
			correlated := q.Requester == r.A && q.RequesterG == r.G && q.Obligation == d.Names[1] && q.ID == d.Pending && q.Target == d.Names[0] && q.ExpectedG == d.Values[childGeneration] && q.W == d.Values[revision]
			if !correlated && c.Mutant != "old-lifecycle-result" {
				return true, "lifecycle correlation"
			}
			d.Values[acceptedResult] = 0
			if n.Result.Accepted {
				d.Values[acceptedResult] = 1
			}
			d.Values[successor] = n.Result.Successor
			d.Pending = ""
			if c.Mutant == "budget-persistence" {
				d.Values[spent] = 0
			}
		}
		d.Last = o.Notice
		p.Observed = &o
	default:
		return true, "local opcode"
	}
	for _, v := range d.Values {
		if v < 0 || v > c.Bounds.Value {
			return true, "capacity:local data"
		}
	}
	return true, ""
}

func noticeTag(kind string) string {
	if kind == "terminal" || kind == "suspicion" {
		return "child"
	}
	return kind
}
