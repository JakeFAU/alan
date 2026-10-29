//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	app "github.com/JakeFAU/alan/examples/substitution"
	communication "github.com/JakeFAU/alan/research/communication"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

type communicationAuth struct {
	ID, IntentID     string
	Allowed          bool
	Reason           string
	Source           runtime.Authority
	ClosureOperation string
}

func communicationSourceProjection(s runtime.Inspection, id runtime.RunID, seq int64) runtime.Projection {
	if seq == 0 {
		for _, r := range s.Tables["runs"] {
			rid, _ := evidenceID(r)
			if rid == id {
				return decode[runtime.Projection](r, "initial")
			}
		}
	}
	for _, h := range s.Tables["history"] {
		rid, _ := evidenceID(h)
		n, _ := evidenceNumber(h["sequence"])
		if rid == id && n == seq {
			return decode[runtime.Projection](h, "successor")
		}
	}
	return runtime.Projection{}
}

// decodeCommunication authenticates retained original records before abstraction.
// J2 has no constructed runtime fixture; this adapter deliberately rejects it.
func decodeCommunication(s runtime.Inspection, q model.Request, protocol string) (communication.Trace, error) {
	out := communication.Trace{Profile: protocol}
	fail := func(msg string) (communication.Trace, error) {
		return out, fmt.Errorf("communication evidence: %s", msg)
	}
	if protocol != communication.J1 {
		return fail("J2 construction obstructed; no runtime correspondence")
	}
	truth, e := decodeJoin(s, app.Request{ID: int64(q.ID), Value: int64(q.Value), Mode: q.Mode})
	if e != nil {
		return out, e
	}
	out.Truth = truth
	for _, p := range s.Tables["programs"] {
		raw, ok := p["bytes"].(string)
		if !ok || fmt.Sprintf("%x", shaBytes([]byte(raw))) != p["id"] {
			return fail("program hash differs from original")
		}
	}
	in := communicationReport(s)
	if in.ID == "" {
		if v := communication.CheckTrace(out); len(v) > 0 {
			return fail(fmt.Sprint(v))
		}
		return out, nil
	}
	m := communication.Message{Kind: "report", IntentID: in.ID, Source: in.Run.Subject, Target: in.Target, Obligation: in.Obligation, SourceGeneration: in.Run.Generation, TargetGeneration: in.Generation, CommitSequence: in.Sequence, Payload: communication.Payload{Report: joinReport(in.Payload)}}
	var attempt map[string]any
	for _, r := range s.Tables["attempts"] {
		if r["intent_id"] == in.ID {
			if attempt != nil {
				return fail("duplicate authorization")
			}
			attempt = r
		}
	}
	if attempt != nil {
		a, e := readEvidence[communicationAuth](attempt, "result")
		if e != nil {
			return out, e
		}
		var req struct {
			Intent   json.RawMessage
			Revision int64
		}
		req, e = readEvidence[struct {
			Intent   json.RawMessage
			Revision int64
		}](attempt, "descriptor")
		if e != nil {
			return out, e
		}
		original := communicationRow(s, "intents", "id", in.ID)
		rev, e := evidenceNumber(original["source_revision"])
		if e != nil {
			return out, e
		}
		if string(req.Intent) != original["descriptor"] || req.Revision != rev || a.Source.Revision != rev || a.ID != communicationIdentity("authorization", in.ID) || attempt["id"] != a.ID {
			return fail("first authorization descriptor/identity/revision")
		}
		source := communicationSourceProjection(s, a.Source.Run, a.Source.Sequence)
		if source.Status == "" {
			return fail("source snapshot absent")
		}
		found := false
		expected := hash([]any{"alan/authorization-row/v1", json.RawMessage(attempt["result"].(string))})
		for _, r := range s.Tables["composition_facts"] {
			var f struct{ Kind, OperationID, RecordDigest string }
			f, e = readEvidence[struct{ Kind, OperationID, RecordDigest string }](r, "descriptor")
			if e != nil {
				return out, e
			}
			if f.Kind == "authorization" && f.OperationID == in.ID {
				if found || f.RecordDigest != expected {
					return fail("authorization source evidence digest")
				}
				found = true
			}
		}
		if !found {
			return fail("authorization causal fact absent")
		}
		m.FirstAuthorization = &communication.Authorization{ID: a.ID, IntentID: a.IntentID, Source: a.Source.Run.Subject, Generation: a.Source.Run.Generation, Epoch: a.Source.Epoch, Revision: a.Source.Revision, Sequence: a.Source.Sequence, SourceStatus: source.Status, Allowed: a.Allowed, Reason: a.Reason}
	}
	var admission map[string]any
	for _, r := range s.Tables["admissions"] {
		var d struct{ IntentID, TransportID, LogicalID string }
		d, e = readEvidence[struct{ IntentID, TransportID, LogicalID string }](r, "descriptor")
		if e != nil {
			return out, e
		}
		if d.IntentID != in.ID {
			continue
		}
		if admission != nil {
			return fail("multiple original report admissions")
		}
		admission = r
		if d.TransportID != communicationIdentity("message-delivery", in.ID) || d.LogicalID != in.ID || r["id"] != communicationIdentity("admission", "message", d.TransportID) || r["domain"] != "message" {
			return fail("original runner transport/admission identity")
		}
		rvalue, e := readEvidence[runtime.AdmissionResult](r, "result")
		if e != nil {
			return out, e
		}
		if m.FirstAuthorization == nil {
			return fail("admission lacks first authorization")
		}
		m.Admission = &communication.Admission{ID: r["id"].(string), IntentID: in.ID, AuthorizationID: m.FirstAuthorization.ID, Accepted: rvalue.Accepted, Reason: rvalue.Reason, ObservationID: rvalue.ObservationID}
		if rvalue.Accepted {
			row := communicationRow(s, "observations", "id", rvalue.ObservationID)
			if row == nil {
				return fail("admission observation absent")
			}
			o, e := readEvidence[observation](row, "descriptor")
			if e != nil {
				return out, e
			}
			if o.ID != communicationIdentity("observation", "message", in.ID) {
				return fail("original observation identity")
			}
			m.Observation = &communication.Observation{ID: o.ID, AdmissionID: m.Admission.ID, IntentID: in.ID, Source: o.Source, Target: o.Run.Subject, Obligation: o.Obligation, SourceGeneration: o.SourceGeneration, TargetGeneration: o.Run.Generation, Payload: communication.Payload{Report: joinReport(o.Payload)}}
			if row["consumed_sequence"] != nil {
				seq, e := evidenceNumber(row["consumed_sequence"])
				if e != nil {
					return out, e
				}
				selected := false
				for _, h := range s.Tables["history"] {
					id, _ := evidenceID(h)
					n, _ := evidenceNumber(h["sequence"])
					if id == o.Run && n == seq && h["selected_observation"] == o.ID {
						selected = true
					}
				}
				if !selected {
					return fail("admission is not actual selected consumption")
				}
				m.Consumption = &communication.Consumption{IntentID: in.ID, ObservationID: o.ID, Receiver: o.Run.Subject, Generation: o.Run.Generation, Sequence: seq}
			}
		}
	}
	if row := communicationRow(s, "message_acknowledgments", "intent_id", in.ID); row != nil {
		if m.Admission == nil || row["admission_id"] != m.Admission.ID {
			return fail("dispatcher ack original admission linkage")
		}
		m.DispatchAcknowledgment = &communication.DispatchAcknowledgment{IntentID: in.ID, AdmissionID: row["admission_id"].(string)}
	}
	if len(s.Roots) != 1 {
		return fail("root aggregate absent")
	}
	if s.Roots[0].Outcome != nil {
		out.Aggregate = s.Roots[0].Outcome.Status
	}
	out.Messages = []communication.Message{m}
	if v := communication.CheckTrace(out); len(v) > 0 {
		return fail(fmt.Sprint(v))
	}
	return out, nil
}
func communicationCut(s runtime.Inspection, cut string) bool {
	in := communicationReport(s)
	if in.ID == "" || projection(s, "root").Status != "open" || len(s.Roots) != 1 || s.Roots[0].Outcome != nil {
		return false
	}
	a := communicationRow(s, "attempts", "intent_id", in.ID)
	transport := communicationIdentity("message-delivery", in.ID)
	ad := communicationRow(s, "admissions", "transport_id", transport)
	ack := communicationRow(s, "message_acknowledgments", "intent_id", in.ID)
	peer := projection(s, "peer")
	switch cut {
	case "report-no-authorization":
		seq := int64(-1)
		for _, r := range s.Tables["runs"] {
			if r["subject"] == "peer" {
				seq, _ = evidenceNumber(r["sequence"])
			}
		}
		return peer.Status == "open" && seq == in.Sequence && a == nil && ad == nil && ack == nil
	case "report-authorized-peer-terminal":
		return peer.Status == "succeeded" && a != nil && decode[communicationAuth](a, "result").Allowed && ad == nil && ack == nil
	case "report-refused-peer-terminal":
		if peer.Status != "succeeded" || a == nil || ad == nil || ack == nil {
			return false
		}
		auth := decode[communicationAuth](a, "result")
		r := decode[runtime.AdmissionResult](ad, "result")
		return !auth.Allowed && auth.Reason == "source_closed" && !r.Accepted && r.Reason == "source_denied:source_closed" && r.ObservationID == "" && ack["admission_id"] == ad["id"]
	}
	return false
}
func communicationImmutable(tables []string, before, after runtime.Inspection) bool {
	for _, name := range tables {
		if !reflect.DeepEqual(before.Tables[name], after.Tables[name]) {
			return false
		}
	}
	return true
}

// The primary cut has all invocation outcomes delivered and empty terminal
// routes. Outside that predicate this is a queue diagnostic, not a general
// reimplementation of prepareInvocation/selectReady.
func communicationQueue(s runtime.Inspection) map[string]any {
	pending := []string{}
	unfinished := []string{}
	ready := []string{}
	waiting := map[string][]string{}
	for _, r := range s.Tables["intents"] {
		in := decode[intent](r, "descriptor")
		switch in.Kind {
		case "message":
			if communicationRow(s, "message_acknowledgments", "intent_id", in.ID) == nil {
				pending = append(pending, communicationIdentity("message-delivery", in.ID))
			}
		case "invocation":
			if communicationRow(s, "composition_outcome_deliveries", "operation_id", in.ID) == nil {
				unfinished = append(unfinished, in.ID)
			}
		}
	}
	for _, r := range s.Tables["terminal_obligations"] {
		var routes []struct{ Target, Obligation string }
		_ = json.Unmarshal([]byte(r["destinations"].(string)), &routes)
		var delivered map[string]bool
		_ = json.Unmarshal([]byte(r["delivery_status"].(string)), &delivered)
		for _, route := range routes {
			if !delivered[communicationIdentity("route", route.Target, route.Obligation)] {
				gen, _ := evidenceNumber(r["generation"])
				seq, _ := evidenceNumber(r["sequence"])
				pending = append(pending, communicationIdentity("terminal-delivery", r["subject"], gen, seq, route.Target, route.Obligation))
			}
		}
	}
	for _, r := range s.Tables["runs"] {
		id, _ := evidenceID(r)
		p := decode[runtime.Projection](r, "projection")
		if p.Status != "open" {
			continue
		}
		program := communicationRow(s, "programs", "id", r["program_id"])
		code := decode[evidenceProgram](program, "bytes").Code
		if code[p.PC].Op != "receive" {
			ready = append(ready, id.Subject)
		} else {
			waiting[id.Subject] = []string{}
			for _, oRow := range s.Tables["observations"] {
				o := decode[observation](oRow, "descriptor")
				if o.Run == id && oRow["consumed_sequence"] == nil {
					waiting[id.Subject] = append(waiting[id.Subject], o.ID)
				}
			}
		}
	}
	sort.Strings(pending)
	sort.Strings(unfinished)
	sort.Strings(ready)
	return map[string]any{"pending_internal_keys": pending, "unfinished_invocations": unfinished, "pure_ready_runs": ready, "receivers_unconsumed_ids": waiting, "scope": "exact primary cut; receive eligibility remains with public Commit"}
}
