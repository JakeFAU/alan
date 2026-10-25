package semantics

import (
	"fmt"
	"strings"
	"testing"
)

// Monitor facts are never read by step or reduce. They are included in the BFS key.
type monitor struct {
	Bindings       map[string]string
	Frozen         map[string]outcome
	Retired        map[string]bool
	CancelRequests map[string]int
	Policy         string
}

func freshMonitor() monitor {
	return monitor{Bindings: map[string]string{}, Frozen: map[string]outcome{}, Retired: map[string]bool{}, CancelRequests: map[string]int{}}
}
func inspect(before, after machine, e event, previous monitor) (monitor, string) {
	mon := copyOf(previous)
	bad := func(s string) (monitor, string) { return mon, s }
	if canonical(before.Config) != canonical(after.Config) {
		return bad("immutable-program")
	}
	for a, s := range after.Subjects {
		if mon.Retired[a] && !s.Retired {
			return bad("retirement-monotonicity")
		}
		if before.Subjects[a].Retired && s.Current != before.Subjects[a].Current {
			return bad("retired-generation")
		}
		mon.Retired[a] = s.Retired
		if after.Runs[runID(a, s.Current)] == nil {
			return bad("current-run-reference")
		}
	}
	for id, r := range after.Runs {
		if r.A == "" || id != runID(r.A, r.G) || after.Subjects[r.A] == nil {
			return bad("run-reference")
		}
		seen := map[int]bool{}
		for _, c := range r.Projection.Frontier {
			if seen[c.ID] || c.ID <= 0 || c.PC < 0 || c.PC >= len(after.Config.Programs[r.A]) {
				return bad("frontier-validity")
			}
			seen[c.ID] = true
		}
		if len(r.Projection.Frontier) > after.Config.Bounds.Continuations || len(r.Inbox) > after.Config.Bounds.Inbox {
			return bad("finite-capacity")
		}
		if old := before.Runs[id]; old != nil {
			if r.Version != old.Version || r.N < old.N || r.N > old.N+1 {
				return bad("journal-monotonicity")
			}
			if len(r.History) < len(old.History) || len(old.History) > 0 && canonical(r.History[:len(old.History)]) != canonical(old.History) {
				return bad("history-immutability")
			}
			if r.N != old.N {
				p := before.Proposals[e.Slot]
				if e.Kind != commit || p == nil {
					return bad("unproposed-commit")
				}
				if p.A != r.A || p.G != r.G || before.Subjects[r.A].Current != p.G {
					return bad("commit-generation")
				}
				if p.E != old.E {
					return bad("commit-epoch")
				}
				if p.N != old.N {
					return bad("commit-sequence")
				}
				if old.Projection.Status != "active" || before.Subjects[r.A].Retired {
					return bad("commit-after-closure")
				}
				if p.Version != old.Version {
					return bad("commit-version")
				}
				if canonical(r.Projection) != canonical(p.After) {
					return bad("commit-proposal-mismatch")
				}
				if p.Observed != nil {
					o, ok := old.Inbox[p.Observed.ID]
					if !ok || o.Consumed || canonical(o) != canonical(*p.Observed) || !r.Inbox[o.ID].Consumed {
						return bad("observation-consumption")
					}
					pc := -1
					for _, c := range old.Projection.Frontier {
						if c.ID == p.Selection.C {
							pc = c.PC
						}
					}
					if pc < 0 {
						return bad("receive-selection")
					}
					in := before.Config.Programs[r.A][pc]
					if in.Op != receive || in.Tag != o.Tag {
						return bad("receive-selection")
					}
					if in.Receipt {
						if o.Kind != "receipt" || old.Projection.Refs[in.Ref] == "" || o.Correlation != old.Projection.Refs[in.Ref] {
							return bad("receive-correlation")
						}
					} else if o.Kind != "message" {
						return bad("receive-correlation")
					}
				}
			}
			if old.Projection.Status != "active" && canonical(r) != canonical(old) {
				return bad("closed-journal-rewrite")
			}
			if e.Kind == recordReceipt && (r.N != old.N || canonical(r.Projection) != canonical(old.Projection)) {
				return bad("receipt-advanced-program")
			}
		}
		if r.Projection.Status != "active" && r.Outcome == nil {
			return bad("missing-terminal-notification")
		}
		if frozen, ok := mon.Frozen[id]; ok && (r.Outcome == nil || *r.Outcome != frozen) {
			return bad("terminal-rewrite")
		}
		if r.Outcome != nil {
			if r.Outcome.Status != r.Projection.Status || r.Outcome.Value != r.Projection.Data[0] || r.Outcome.Route != after.Config.Routes[r.A] {
				return bad("notification-correlation")
			}
			mon.Frozen[id] = *r.Outcome
		}
	}
	for id, in := range after.Intents {
		d := in.Descriptor
		r := after.Runs[runID(d.A, d.G)]
		if r == nil || id != intentID(d.A, d.G, d.N) {
			return bad("intent-reference")
		}
		found := false
		for _, h := range r.History {
			if h.Kind == "commit" && h.N+1 == d.N && h.Intent != nil && *h.Intent == d {
				found = true
			}
		}
		if !found {
			return bad("effect-before-commit")
		}
		if old := before.Intents[id]; old != nil && old.Descriptor != d {
			return bad("intent-immutability")
		}
		if in.Settled {
			foundReceipt := false
			for _, receipt := range after.Receipts {
				if receipt.Intent == id {
					foundReceipt = true
				}
			}
			if !foundReceipt {
				return bad("settlement-without-evidence")
			}
		}
	}
	// Check actual newly durable authorizations, independently of authorization guards.
	for id, a := range after.Attempts {
		in := before.Intents[a.Intent]
		if old, ok := before.Attempts[id]; ok {
			if old != a {
				return bad("attempt-immutability")
			}
			continue
		}
		if in == nil {
			return bad("authorization-before-intent")
		}
		r := before.Runs[runID(in.Descriptor.A, in.Descriptor.G)]
		s := before.Subjects[in.Descriptor.A]
		if r.Projection.Status != "active" || s.Retired || s.Current != r.G {
			return bad("authorization-after-closure")
		}
		if a.Owner != s.Owner || a.W != s.W {
			return bad("authorization-ownership")
		}
		if e.Kind != authorize {
			return bad("unrecorded-authorization")
		}
	}
	for id, in := range after.Intents {
		if in.Suppressed && after.hasAttempt(id) {
			return bad("suppression-of-authorized-effect")
		}
	}
	for id, ad := range after.Admissions {
		identity := canonical(struct {
			Message  message
			Bound    string
			Accepted bool
		}{ad.Message, ad.Bound, ad.Accepted})
		if old, ok := mon.Bindings[id]; ok && old != identity {
			return bad("stable-service-binding")
		}
		mon.Bindings[id] = identity
		if id != transportID(ad.Origin, ad.RawID) {
			return bad("admission-origin-identity")
		}
		if ad.Origin == "send" {
			in := after.Intents[ad.RawID]
			if in == nil || in.Descriptor.Kind != send || ad.Message != sentMessage(in.Descriptor) || after.World.Outbox[ad.RawID] != ad.Message {
				return bad("send-admission-correlation")
			}
		} else if ad.Origin != "" {
			return bad("admission-origin")
		}
		count := 0
		for rid, r := range after.Runs {
			if o, ok := r.Inbox[admissionObservationID(id)]; ok {
				count++
				if rid != ad.Bound {
					return bad("stable-service-binding")
				}
				if o.ID != admissionObservationID(id) || o.Kind != "message" || o.Tag != ad.Message.Tag || o.Value != ad.Message.Value || o.Correlation != "" {
					return bad("admission-payload-immutability")
				}
			}
		}
		if ad.Accepted && count != 1 || !ad.Accepted && count != 0 {
			return bad("single-admission")
		}
	}
	for id, result := range after.World.Applied {
		a, ok := after.Attempts[id]
		if !ok || a.Intent != result.Intent {
			return bad("external-authorization")
		}
		if old, ok := before.World.Applied[id]; ok && old != result {
			return bad("external-result-immutability")
		}
	}
	if e.Kind != applyExternal && canonical(before.World) != canonical(after.World) {
		return bad("external-atomic-boundary")
	}
	for id, rc := range after.Receipts {
		a, ok := after.Attempts[id]
		if !ok || rc.Attempt != id || a.Intent != rc.Intent {
			return bad("receipt-correlation")
		}
		result, ok := after.World.Applied[id]
		if !ok || result.Intent != rc.Intent {
			return bad("receipt-without-result")
		}
		in := after.Intents[rc.Intent]
		if in.Descriptor.Kind == send {
			ad := after.Admissions[transportID("send", rc.Intent)]
			if ad == nil || !ad.Ack {
				return bad("send-receipt-without-acknowledgment")
			}
			if ad.Origin != "send" || ad.RawID != rc.Intent || ad.Message != sentMessage(in.Descriptor) {
				return bad("send-receipt-correlation")
			}
			want := 0
			if ad.Accepted {
				want = 1
			}
			if rc.Value != want {
				return bad("send-receipt-value")
			}
		} else if rc.Value != result.Value {
			return bad("receipt-value")
		}
		if old, ok := before.Receipts[id]; ok && old != rc {
			return bad("receipt-immutability")
		}
	}
	for id, value := range after.Delivered {
		r := after.Runs[id]
		if r == nil || r.Outcome == nil || *r.Outcome != value {
			return bad("delivery-correlation")
		}
	}
	if e.Kind == cancel {
		mon.CancelRequests[e.A] = e.G
	}
	if mon.Policy == "cancel-prevents-retry" {
		for a, g := range mon.CancelRequests {
			if after.Subjects[a].Current > g {
				return bad("policy:cancel-prevents-retry")
			}
		}
	}
	return mon, ""
}

// Replay reconstructs the specified run projection. Pending observations and the
// dispatch/settlement/admission ledgers are validated separately by inspect.
func replay(m machine, id, version string) error {
	original := m.Runs[id]
	if original == nil || version != original.Version || version != m.Config.Version {
		return fmt.Errorf("wrong program version")
	}
	shadow := freshRun(original.A, original.G, version)
	intents := map[string]descriptor{}
	for _, h := range original.History {
		switch h.Kind {
		case "reactivate":
			if h.E != shadow.E+1 || h.N != shadow.N || shadow.Projection.Status != "active" {
				return fmt.Errorf("invalid reactivation record")
			}
			shadow.E = h.E
		case "close":
			if shadow.Projection.Status != "active" || h.E != shadow.E || h.N != shadow.N {
				return fmt.Errorf("invalid closure record")
			}
			shadow.Projection.Status = h.Status
			shadow.Projection.Frontier = nil
		case "commit":
			if h.Version != shadow.Version || h.N != shadow.N || h.E != shadow.E || shadow.Projection.Status != "active" {
				return fmt.Errorf("invalid commit stamp")
			}
			if h.Observed != nil {
				if old, ok := shadow.Inbox[h.Observed.ID]; ok && old.Consumed {
					return fmt.Errorf("observation consumed twice")
				}
				shadow.Inbox[h.Observed.ID] = *h.Observed
			}
			p, why := reduce(m.Config, shadow, h.Selection)
			if why != "" {
				return fmt.Errorf("reduction: %s", why)
			}
			if canonical(p.After) != canonical(h.After) || canonical(p.Intent) != canonical(h.Intent) || canonical(p.Observed) != canonical(h.Observed) {
				return fmt.Errorf("reduction differs")
			}
			shadow.Projection = p.After
			shadow.N++
			if p.Observed != nil {
				o := *p.Observed
				o.Consumed = true
				shadow.Inbox[o.ID] = o
			}
			if p.Intent != nil {
				intents[intentID(shadow.A, shadow.G, shadow.N)] = *p.Intent
			}
		default:
			return fmt.Errorf("unknown history record")
		}
		if shadow.Projection.Status != "active" && shadow.Outcome == nil {
			shadow.Outcome = &outcome{Status: shadow.Projection.Status, Value: shadow.Projection.Data[0], Route: m.Config.Routes[shadow.A]}
		}
	}
	if shadow.N != original.N || shadow.E != original.E || canonical(shadow.Projection) != canonical(original.Projection) || canonical(shadow.Outcome) != canonical(original.Outcome) {
		return fmt.Errorf("replayed journal projection differs")
	}
	for id, o := range original.Inbox {
		if o.Consumed && canonical(shadow.Inbox[id]) != canonical(o) {
			return fmt.Errorf("consumed observation differs")
		}
	}
	for id, o := range shadow.Inbox {
		if !o.Consumed || canonical(original.Inbox[id]) != canonical(o) {
			return fmt.Errorf("replayed observation differs")
		}
	}
	for id, in := range m.Intents {
		if in.Descriptor.A == original.A && in.Descriptor.G == original.G {
			if intents[id] != in.Descriptor {
				return fmt.Errorf("intent differs")
			}
			delete(intents, id)
		}
	}
	if len(intents) != 0 {
		return fmt.Errorf("replayed intent missing in ledger")
	}
	return nil
}

func mustStep(t *testing.T, m *machine, e event) {
	t.Helper()
	next, why := step(*m, e)
	if why != "" {
		t.Fatalf("%s rejected: %s", e.Kind, why)
	}
	mon := freshMonitor()
	for id, ad := range m.Admissions {
		mon.Bindings[id] = canonical(struct {
			Message  message
			Bound    string
			Accepted bool
		}{ad.Message, ad.Bound, ad.Accepted})
	}
	for id, r := range m.Runs {
		if r.Outcome != nil {
			mon.Frozen[id] = *r.Outcome
		}
	}
	if _, violation := inspect(*m, next, e, mon); violation != "" {
		t.Fatalf("%s violated %s", e.Kind, violation)
	}
	*m = next
}
func mustReject(t *testing.T, m *machine, e event, want string) {
	t.Helper()
	next, why := step(*m, e)
	if !strings.Contains(why, want) {
		t.Fatalf("%s rejection = %q, want %q", e.Kind, why, want)
	}
	if canonical(next) != canonical(*m) {
		t.Fatal("rejected event mutated state")
	}
}
func turn(t *testing.T, m *machine, a string, c int, observation string) {
	t.Helper()
	mustStep(t, m, event{Kind: prepare, A: a, G: m.Subjects[a].Current, C: c, Observation: observation})
	mustStep(t, m, event{Kind: commit})
}
