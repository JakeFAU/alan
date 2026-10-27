package semantics

import (
	"encoding/json"
	"slices"
)

// These finite operands are ordinary program data, not a privileged coordinator.
type envelope struct {
	Caller              string
	Generation, Request int
	Key, ReplyTo        string
}
type pendingRequest struct {
	Envelope envelope
	Peer     string
}
type compositionContract struct {
	Outer        string
	BranchLeaves map[string]string
	UnitTag      string
	UnitValue    int
}

func decodeEnvelope(raw string) envelope {
	var e envelope
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			panic(err)
		}
	}
	return e
}
func admittedObservation(id, origin string, msg message) observation {
	o := observation{ID: admissionObservationID(id), Tag: msg.Tag, Value: msg.Value, Kind: "message"}
	if msg.Envelope != "" {
		o.Envelope = msg.Envelope
		o.Sender = msg.Source
		o.SenderG = msg.SenderG
		o.Origin = origin
	}
	return o
}
func compositionReceive(c config, r *run, p *proposal, in instruction, o observation) string {
	if in.Mode == "" {
		return ""
	}
	if in.Mode != "capture" && in.Mode != "result" {
		return "invalid receive mode"
	}
	if c.Mutant != "origin" && (o.Origin != "send" || o.Sender == "" || o.SenderG < 1) {
		return "origin"
	}
	env := decodeEnvelope(o.Envelope)
	if env.Caller == "" || env.Generation < 1 || env.Request < 1 || env.Key == "" || env.ReplyTo == "" {
		return "envelope"
	}
	if in.Mode == "capture" {
		if env.Caller != o.Sender || env.Generation != o.SenderG {
			return "request provenance"
		}
		if in.Slot < 0 || in.Slot >= 2 || in.Slot > len(p.After.Saved) {
			return "capacity:saved"
		}
		if in.Slot == len(p.After.Saved) {
			p.After.Saved = append(p.After.Saved, env)
		} else {
			p.After.Saved[in.Slot] = env
		}
		return ""
	}
	matched := false
	for _, pending := range r.Projection.Pending {
		want, got := pending.Envelope, env
		if c.Mutant == "generation-correlation" {
			got.Generation = want.Generation
		}
		if got == want && (c.Mutant == "origin" || pending.Peer == o.Sender) {
			matched = true
		}
	}
	if !matched {
		return "request correlation"
	}
	key := canonical(env)
	if c.Mutant != "logical-duplicate" && slices.Contains(r.Projection.Accepted, key) {
		return "logical duplicate"
	}
	if len(p.After.Accepted) >= 4 {
		return "capacity:accepted"
	}
	p.After.Accepted = append(p.After.Accepted, key)
	return ""
}
func compositionSend(r *run, p *proposal, in instruction) string {
	if in.FromReg {
		p.Intent.Value = r.Projection.Data[in.Reg]
	}
	switch in.Mode {
	case "":
		return ""
	case "request":
		if in.Slot < 0 || in.Slot >= 2 || in.Slot > len(p.After.Pending) {
			return "capacity:pending"
		}
		e := envelope{Caller: r.A, Generation: r.G, Request: r.N + 1, Key: in.Key, ReplyTo: r.A}
		p.Intent.Envelope = canonical(e)
		pending := pendingRequest{Envelope: e, Peer: p.Intent.Target}
		if in.Slot == len(p.After.Pending) {
			p.After.Pending = append(p.After.Pending, pending)
		} else {
			p.After.Pending[in.Slot] = pending
		}
	case "reply", "effect":
		if in.Slot < 0 || in.Slot >= len(r.Projection.Saved) {
			return "missing upstream"
		}
		p.Intent.Envelope = canonical(r.Projection.Saved[in.Slot])
	default:
		return "invalid send mode"
	}
	return ""
}
