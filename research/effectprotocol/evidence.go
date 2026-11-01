package effectprotocol

// Request is the fixture contract quotient. Identity covers the immutable
// operation and fixed source binding. Payload is the complete selected body.
type Request struct{ Key, Identity, Payload string }
type Attestation struct {
	Key, Payload  string
	Number, Total int
}
type Entry struct {
	Request    Request
	First      Attestation
	Idempotent bool
}

// Ledger has capacity for the two distinct keys of the comparator. Its caller
// supplies the endpoint mode; every probe, including rejection, is ingress.
type Ledger struct {
	Entries                                [2]Entry
	Ingress, Contacts, Applications, Total int
}

func (w Ledger) Submit(q Request, idempotent bool) (Ledger, Attestation, string) {
	w.Ingress++
	var slot *Entry
	for i := range w.Entries {
		e := &w.Entries[i]
		if e.Request.Key == q.Key {
			if e.Request != q || e.Idempotent != idempotent {
				return w, Attestation{}, "key-conflict"
			}
			slot = e
			break
		}
		if e.Request.Key == "" && slot == nil {
			slot = e
		}
	}
	if slot == nil || q.Key == "" || q.Identity == "" || q.Payload != "one" && q.Payload != "two" {
		return w, Attestation{}, "refused"
	}
	w.Contacts++
	if idempotent && slot.First.Number != 0 {
		return w, slot.First, "duplicate"
	}
	w.Applications++
	w.Total++
	if q.Payload == "two" {
		w.Total++
	}
	r := Attestation{q.Key, q.Payload, w.Applications, w.Total}
	if slot.First.Number == 0 {
		*slot = Entry{Request: q, First: r, Idempotent: idempotent}
	}
	return w, r, "application"
}

// Retain compares a candidate offered for the original operation. The caller
// keeps the offer in the research trace even where the local row is unchanged.
func Retain(q Request, offeredFor string, prior, offer Attestation) (Attestation, string) {
	if offeredFor != q.Identity || offer.Key != q.Key || offer.Payload != q.Payload || offer.Number < 1 {
		return prior, "wrong-binding"
	}
	if prior.Number != 0 {
		if prior != offer {
			return prior, "conflicting-result"
		}
		return prior, "equal-result"
	}
	return offer, "recorded"
}
