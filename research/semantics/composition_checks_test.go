package semantics

import "slices"

type compositionFacts struct {
	Contributions     map[string][]string
	RejectedOld       map[string]bool
	RejectedDuplicate map[string]bool
	LostAckRetry      map[string]bool
	UnresolvedClosure map[string]bool
}

func freshCompositionFacts() *compositionFacts {
	return &compositionFacts{Contributions: map[string][]string{}, RejectedOld: map[string]bool{}, RejectedDuplicate: map[string]bool{}, LostAckRetry: map[string]bool{}, UnresolvedClosure: map[string]bool{}}
}

// Contract inspection is deliberately separate from the reducer's guards. In
// particular origin is reconstructed from admission and immutable Send history.
func inspectComposition(before, after machine, e event, mon *monitor) string {
	if after.Config.Composition == nil {
		return ""
	}
	if mon.Composition == nil {
		mon.Composition = freshCompositionFacts()
	}
	facts := mon.Composition
	for id, r := range after.Runs {
		if len(r.Projection.Saved) > 2 || len(r.Projection.Pending) > 2 || len(r.Projection.Accepted) > 4 || len(after.Config.Programs[r.A]) > 12 {
			return "composition-capacity"
		}
		old := before.Runs[id]
		if old == nil || old.N == r.N {
			continue
		}
		h := r.History[len(r.History)-1]
		var in instruction
		for _, ct := range old.Projection.Frontier {
			if ct.ID == h.Selection.C {
				in = before.Config.Programs[r.A][ct.PC]
			}
		}
		if in.Mode == "result" && h.Observed != nil {
			o := *h.Observed
			var ad *admission
			for aid, a := range before.Admissions {
				if admissionObservationID(aid) == o.ID {
					ad = a
				}
			}
			if ad == nil || ad.Origin != "send" || ad.Message.Source != o.Sender || ad.Message.SenderG != o.SenderG {
				return "composition-origin"
			}
			d := before.Intents[ad.RawID]
			if d == nil || d.Descriptor.A != o.Sender || d.Descriptor.G != o.SenderG || d.Descriptor.Envelope != o.Envelope {
				return "composition-origin"
			}
			env := decodeEnvelope(o.Envelope)
			matched := false
			for _, req := range old.Projection.Pending {
				if req.Envelope == env && req.Peer == o.Sender {
					matched = true
				}
			}
			if !matched {
				return "composition-generation-correlation"
			}
			sender := before.Runs[runID(o.Sender, o.SenderG)]
			if sender == nil || o.Value != before.Config.Composition.UnitValue || !supportedRequest(before, sender, env, map[string]bool{}) {
				return "composition-unsupported-contribution"
			}
			key := canonical(env)
			if slices.Contains(facts.Contributions[id], key) {
				return "composition-logical-duplicate"
			}
			facts.Contributions[id] = append(facts.Contributions[id], key)
			if canonical(facts.Contributions[id]) != canonical(r.Projection.Accepted) {
				return "composition-accepted-state"
			}
		}
		if h.Intent != nil && in.Mode == "reply" {
			if !supportedRequest(before, old, decodeEnvelope(h.Intent.Envelope), map[string]bool{}) {
				return "composition-unsupported-reply"
			}
		}
		if r.Projection.Status == "succeeded" && r.A != after.Config.Composition.Outer {
			if len(r.Projection.Saved) == 0 {
				return "composition-unsupported-success"
			}
			for _, env := range r.Projection.Saved {
				if !supportedRequest(after, r, env, map[string]bool{}) {
					return "composition-unsupported-success"
				}
			}
		}
		if r.Projection.Status == "succeeded" && r.A == after.Config.Composition.Outer {
			branches := map[string]bool{}
			for _, key := range facts.Contributions[id] {
				env := decodeEnvelope(key)
				if _, ok := after.Config.Composition.BranchLeaves[env.Key]; !ok || branches[env.Key] {
					return "composition-unsupported-success"
				}
				branches[env.Key] = true
			}
			if len(branches) != len(after.Config.Composition.BranchLeaves) || len(facts.Contributions[id]) != 2 || r.Projection.Data[0] != 2*after.Config.Composition.UnitValue {
				return "composition-unsupported-success"
			}
		}
	}
	for id, result := range after.World.Applied {
		if _, ok := before.World.Applied[id]; ok {
			continue
		}
		d := after.Intents[result.Intent].Descriptor
		if d.Kind == emit {
			c := after.Config.Composition
			if d.Tag != c.UnitTag || d.Value != c.UnitValue || d.Envelope == "" {
				return "composition-resource-footprint"
			}
			root, ok := requestRoot(after, decodeEnvelope(d.Envelope), map[string]bool{})
			if !ok || c.BranchLeaves[root.Key] != d.A {
				return "composition-resource-path"
			}
			for otherID, other := range after.Intents {
				if otherID == result.Intent || other.Descriptor.Kind != emit {
					continue
				}
				if _, applied := after.World.Keys[otherID]; !applied {
					continue
				}
				otherRoot, valid := requestRoot(after, decodeEnvelope(other.Descriptor.Envelope), map[string]bool{})
				if !valid {
					return "composition-resource-path"
				}
				if otherRoot == root {
					return "composition-resource-cardinality"
				}
			}
		}
	}
	return ""
}

// requestRoot resolves the full public request identity, not only its branch
// label or the immediate downstream request. Distinct root generations remain
// independent. Ambiguous captured roots cannot establish a declared unit path.
func requestRoot(m machine, env envelope, seen map[string]bool) (envelope, bool) {
	key := canonical(env)
	if seen[key] {
		return envelope{}, false
	}
	seen[key] = true
	defer delete(seen, key)
	req := m.Intents[intentID(env.Caller, env.Generation, env.Request)]
	if req == nil || req.Descriptor.Kind != send || req.Descriptor.Envelope != key {
		return envelope{}, false
	}
	if env.Caller == m.Config.Composition.Outer {
		return env, true
	}
	r := m.Runs[runID(env.Caller, env.Generation)]
	if r == nil {
		return envelope{}, false
	}
	for _, h := range r.History {
		if h.Kind != "commit" || h.N+1 != env.Request {
			continue
		}
		var root envelope
		found := false
		for _, up := range h.After.Saved {
			candidate, ok := requestRoot(m, up, seen)
			if !ok || found && candidate != root {
				return envelope{}, false
			}
			root = candidate
			found = true
		}
		return root, found
	}
	return envelope{}, false
}

// Follow only evidence already consumed by this run, never the scalar aggregate
// or the mere existence of an unrelated external mutation.
func supportedRequest(m machine, r *run, env envelope, seen map[string]bool) bool {
	key := runID(r.A, r.G) + canonical(env)
	if seen[key] {
		return false
	}
	seen[key] = true
	for _, h := range r.History {
		if h.Observed == nil {
			continue
		}
		o := *h.Observed
		if o.Kind == "receipt" {
			d := m.Intents[o.Correlation]
			if d != nil && d.Descriptor.Kind == emit && d.Descriptor.Envelope == canonical(env) && d.Descriptor.Tag == m.Config.Composition.UnitTag && o.Value == m.Config.Composition.UnitValue {
				return true
			}
		}
		if o.Kind == "message" && o.Origin == "send" {
			down := decodeEnvelope(o.Envelope)
			if down.Caller != r.A || down.Generation != r.G {
				continue
			}
			req := m.Intents[intentID(down.Caller, down.Generation, down.Request)]
			if req == nil || req.Descriptor.Target != o.Sender {
				continue
			}
			for _, rh := range r.History {
				if rh.N+1 == down.Request && slices.Contains(rh.After.Saved, env) {
					child := m.Runs[runID(o.Sender, o.SenderG)]
					if child != nil && supportedRequest(m, child, down, seen) {
						return true
					}
				}
			}
		}
	}
	return false
}
