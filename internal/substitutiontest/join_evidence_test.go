//go:build integration

package substitutiontest

import (
	"fmt"
	"reflect"

	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/executor"
	joinmodel "github.com/JakeFAU/alan/research/join"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func joinReport(p map[string]schema.Value) joinmodel.Report {
	return joinmodel.Report{RequestID: int(p["request_id"].Number), SOperation: p["s_operation"].Text, Outcome: joinmodel.Outcome{Status: p["status"].Text, Reason: p["reason"].Text, Present: p["response_present"].Flag, ID: int(p["response_id"].Number), Value: int(p["response_value"].Number)}}
}
func joinOutcome(p map[string]schema.Value) joinmodel.Outcome {
	return joinmodel.Outcome{Status: p["status"].Text, Reason: p["reason"].Text, Present: p["response_present"].Flag, ID: int(p["response.id"].Number), Value: int(p["response.value"].Number)}
}

// decodeJoin extends the reviewed evidence validator only at ordinary reports.
// Behavioral truth is exclusively checked by the independently authored oracle.
func decodeJoin(s runtime.Inspection, q app.Request) (joinmodel.Trace, error) {
	t := joinmodel.Trace{Request: model.Request{ID: int(q.ID), Value: int(q.Value), Mode: q.Mode}, Root: "root", Peer: "peer", Generation: 1, Obligation: "phase1j_s_join"}
	fail := func(f string, a ...any) (joinmodel.Trace, error) { return t, fmt.Errorf(f, a...) }
	if _, e := validateRetained(s); e != nil {
		return t, e
	}
	intents := map[string]intent{}
	reports := map[string]intent{}
	for _, row := range s.Tables["intents"] {
		i, e := readEvidence[intent](row, "descriptor")
		if e != nil {
			return t, e
		}
		intents[i.ID] = i
		if i.Run.Subject == t.Peer && i.Kind == "message" {
			reports[i.ID] = i
		}
	}
	observations := map[string]observation{}
	consumed := map[string]int64{}
	for _, row := range s.Tables["observations"] {
		o, e := readEvidence[observation](row, "descriptor")
		if e != nil {
			return t, e
		}
		observations[o.ID] = o
		if row["consumed_sequence"] != nil {
			seq, e := evidenceNumber(row["consumed_sequence"])
			if e != nil {
				return t, e
			}
			consumed[o.ID] = seq
		}
	}
	histories := map[string]evidenceHistory{}
	for _, row := range s.Tables["history"] {
		id, e := evidenceID(row)
		if e != nil {
			return t, e
		}
		seq, e := evidenceNumber(row["sequence"])
		if e != nil {
			return t, e
		}
		h := evidenceHistory{Run: id, Sequence: seq}
		h.Successor, e = readEvidence[runtime.Projection](row, "successor")
		if e != nil {
			return t, e
		}
		h.Intents, e = readEvidence[[]intent](row, "intents")
		if e != nil {
			return t, e
		}
		h.Selected, _ = row["selected_observation"].(string)
		histories[evidenceKey(id, seq)] = h
	}
	runs := map[runtime.RunID]evidenceRun{}
	programs := map[string]evidenceProgram{}
	for _, row := range s.Tables["programs"] {
		p, e := readEvidence[evidenceProgram](row, "bytes")
		if e != nil {
			return t, e
		}
		programs[row["id"].(string)] = p
	}
	for _, row := range s.Tables["runs"] {
		id, e := evidenceID(row)
		if e != nil {
			return t, e
		}
		initial, e := readEvidence[runtime.Projection](row, "initial")
		if e != nil {
			return t, e
		}
		current, e := readEvidence[runtime.Projection](row, "projection")
		if e != nil {
			return t, e
		}
		seq, e := evidenceNumber(row["sequence"])
		if e != nil {
			return t, e
		}
		runs[id] = evidenceRun{ID: id, Program: row["program_id"].(string), Initial: initial, Current: current, Sequence: seq}
	}
	authorizations := map[string]executor.Authorization{}
	for _, row := range s.Tables["attempts"] {
		id, _ := row["intent_id"].(string)
		in, ok := reports[id]
		if !ok {
			continue
		}
		req, e := readEvidence[struct {
			Intent   intent
			Revision int64
		}](row, "descriptor")
		if e != nil {
			return t, e
		}
		a, e := readEvidence[executor.Authorization](row, "result")
		if e != nil {
			return t, e
		}
		if !reflect.DeepEqual(req.Intent, in) || a.IntentID != id || a.ID != row["id"] || a.Source.Run != in.Run || a.Source.Sequence < in.Sequence || a.Source.Sequence > runs[in.Run].Sequence || a.Source.Epoch <= 0 || a.Source.Revision <= 0 {
			return fail("report authorization original intent/source boundary mismatch")
		}
		p := runs[in.Run].Initial
		if a.Source.Sequence > 0 {
			h, ok := histories[evidenceKey(in.Run, a.Source.Sequence)]
			if !ok {
				return fail("report authorization missing source snapshot")
			}
			p = h.Successor
		}
		if a.Allowed && (a.Reason != "authorized" || p.Status != "open" || req.Revision != a.Source.Revision) {
			return fail("report authorization unsupported allowed source")
		}
		if !a.Allowed && a.Reason == "source_closed" && p.Status == "open" {
			return fail("report authorization unsupported source_closed")
		}
		if !a.Allowed && a.Reason != "source_closed" {
			return fail("report authorization outside static generation/revision profile: %s", a.Reason)
		}
		authorizations[id] = a
		status := "rejected"
		if a.Allowed {
			status = "accepted"
		}
		t.Events = append(t.Events, joinmodel.Event{Kind: "report_authorize", Actor: in.Run.Subject, Operation: id, Sequence: a.Source.Sequence, Status: status, Reason: a.Reason})
	}
	delivered := map[string]string{}
	admissions := map[string]string{}
	for _, row := range s.Tables["admissions"] {
		d, e := readEvidence[executor.AdmissionDescriptor](row, "descriptor")
		if e != nil {
			return t, e
		}
		in, ok := reports[d.IntentID]
		if !ok {
			continue
		}
		r, e := readEvidence[runtime.AdmissionResult](row, "result")
		if e != nil {
			return t, e
		}
		a, ok := authorizations[in.ID]
		if !ok || d.Kind != "message" || d.Source != in.Run || d.Target != in.Target || d.Generation != in.Generation || d.Obligation != in.Obligation || d.LogicalID != in.ID || d.Operation != "" || !sameValues(d.Payload, in.Payload) || d.TransportID != r.TransportID || row["transport_id"] != d.TransportID {
			return fail("report admission original message/authorization mismatch")
		}
		if !a.Allowed && (r.Accepted || r.Reason != "source_denied:"+a.Reason) {
			return fail("report admission bypasses source refusal")
		}
		if r.Accepted {
			o, ok := observations[r.ObservationID]
			if !ok || !a.Allowed || r.Reason != "accepted" || r.Bound == nil || *r.Bound != (runtime.RunID{Subject: in.Target, Generation: in.Generation}) || row["observation_id"] != o.ID || o.Run != *r.Bound || o.Kind != "message" || o.Provenance != "ordinary-message" || o.Source != in.Run.Subject || o.SourceGeneration != in.Run.Generation || o.Obligation != in.Obligation || o.Operation != "" || !sameValues(o.Payload, in.Payload) {
				return fail("report delivery observation provenance/payload mismatch")
			}
			if old, ok := delivered[o.ID]; ok && old != in.ID {
				return fail("report observation aliases distinct intents")
			}
			delivered[o.ID] = in.ID
		} else if r.ObservationID != "" || row["observation_id"] != nil {
			return fail("refused report has observation")
		}
		admissions[row["id"].(string)] = in.ID
		status := "rejected"
		if r.Accepted {
			status = "accepted"
		}
		t.Events = append(t.Events, joinmodel.Event{Kind: "report_deliver", Actor: in.Run.Subject, Operation: in.ID, Status: status, Reason: r.Reason, Report: joinReport(d.Payload)})
	}
	for _, row := range s.Tables["message_acknowledgments"] {
		id, _ := row["intent_id"].(string)
		if _, ok := reports[id]; ok && admissions[fmt.Sprint(row["admission_id"])] != id {
			return fail("report acknowledgment original admission mismatch")
		}
	}
	for id, o := range observations {
		if o.Kind == "message" && o.Source == t.Peer && o.Run.Subject == t.Root {
			if _, ok := delivered[id]; !ok {
				return fail("report observation lacks original admission")
			}
		}
	}
	ads := map[string]runtime.InvocationAdmission{}
	for _, row := range s.Tables["composition_admission_decisions"] {
		a, e := readEvidence[runtime.InvocationAdmission](row, "descriptor")
		if e != nil {
			return t, e
		}
		ads[a.Invocation.OperationID] = a
	}
	role := func(op string) string {
		i := intents[op]
		if i.Run.Subject == t.Root {
			return "candidate"
		}
		if i.Run.Subject == t.Peer {
			return "s"
		}
		return "helper"
	}
	for _, row := range s.Tables["composition_facts"] {
		f, e := readEvidence[fact](row, "descriptor")
		if e != nil {
			return t, e
		}
		if f.Kind == "admission" {
			a := ads[f.OperationID]
			ev := joinmodel.Event{Kind: "admit", Actor: f.Run.Subject, Operation: f.OperationID, Role: role(f.OperationID), Sequence: f.Sequence, Status: "rejected", Reason: a.Reason}
			if a.Accepted {
				ev.Status = "accepted"
				ev.I = 1
				ev.R = len(a.Members)
			}
			t.Events = append(t.Events, ev)
		}
		if f.Kind != "execution" {
			continue
		}
		h := histories[evidenceKey(f.Run, f.Sequence)]
		for _, in := range h.Intents {
			if _, ok := reports[in.ID]; ok {
				t.Events = append(t.Events, joinmodel.Event{Kind: "report_commit", Actor: in.Run.Subject, Operation: in.ID, Sequence: in.Sequence, Source: in.Run.Subject, Target: in.Target, Generation: in.Generation, Obligation: in.Obligation, Report: joinReport(in.Payload)})
			}
		}
		if h.Selected != "" {
			o := observations[h.Selected]
			if o.Kind == "invocation" {
				p := runs[h.Run].Initial
				if h.Sequence > 1 {
					p = histories[evidenceKey(h.Run, h.Sequence-1)].Successor
				}
				ins := programs[runs[h.Run].Program].Code[p.PC]
				for field, destination := range ins.Bind {
					if _, present := o.Payload[field]; !present {
						if _, bound := h.Successor.Data[destination]; bound {
							return fail("absent invocation response retains stale binding")
						}
					}
				}
				t.Events = append(t.Events, joinmodel.Event{Kind: "outcome_consume", Actor: f.Run.Subject, Sequence: f.Sequence, Operation: o.Operation, Role: role(o.Operation), Outcome: joinOutcome(o.Payload)})
			}
			if o.Kind == "message" && o.Source == t.Peer && o.Run.Subject == t.Root {
				id, ok := delivered[o.ID]
				if !ok || consumed[o.ID] != h.Sequence {
					return fail("selected report lacks original delivery/consumption")
				}
				p := runs[h.Run].Initial
				if h.Sequence > 1 {
					p = histories[evidenceKey(h.Run, h.Sequence-1)].Successor
				}
				code := programs[runs[h.Run].Program].Code
				if p.PC < 0 || p.PC >= len(code) {
					return fail("report receive PC invalid")
				}
				ins := code[p.PC]
				if ins.Op != "receive" || ins.Match == nil {
					return fail("report selection not receive")
				}
				m := ins.Match
				source, e := evidenceValue(m.Source, p.Data)
				if e != nil {
					return t, e
				}
				generation, e := evidenceValue(m.Generation, p.Data)
				if e != nil {
					return t, e
				}
				obligation, e := evidenceValue(m.Obligation, p.Data)
				if e != nil {
					return t, e
				}
				operation, e := evidenceValue(m.Operation, p.Data)
				if e != nil {
					return t, e
				}
				if m.Kind != o.Kind || m.Provenance != o.Provenance || source.Text != o.Source || generation.Number != o.SourceGeneration || obligation.Text != o.Obligation || operation.Text != o.Operation {
					return fail("report selected guard/provenance mismatch")
				}
				for field, dest := range ins.Bind {
					v, ok := o.Payload[field]
					actual, bound := h.Successor.Data[dest]
					if ok != bound || (ok && !reflect.DeepEqual(v, actual)) {
						return fail("report selected payload/bound successor mismatch")
					}
				}
				t.Events = append(t.Events, joinmodel.Event{Kind: "report_consume", Actor: h.Run.Subject, Operation: id, Sequence: h.Sequence, Source: o.Source, Target: o.Run.Subject, Generation: o.SourceGeneration, Obligation: o.Obligation, Report: joinReport(o.Payload)})
			}
		}
		if h.Successor.Status != "open" && (h.Run.Subject == t.Root || h.Run.Subject == t.Peer) {
			kind := "root_terminal"
			if h.Run.Subject == t.Peer {
				kind = "peer_terminal"
			}
			t.Events = append(t.Events, joinmodel.Event{Kind: kind, Actor: h.Run.Subject, Sequence: h.Sequence, Status: h.Successor.Status})
		}
	}
	var spentI, spentR int64
	for _, a := range ads {
		if a.Accepted {
			spentI++
			spentR += int64(len(a.Members))
		}
	}
	if s.Roots[0].Charges.Invocations != spentI || s.Roots[0].Charges.CreatedRuns != spentR || s.Roots[0].Charges.Effects != 0 {
		return fail("join actual charges differ from original admission debits")
	}
	for _, row := range s.Tables["lifecycle_results"] {
		l, e := readEvidence[evidenceLifecycle](row, "result")
		if e != nil {
			return t, e
		}
		if l.Accepted && l.After != nil && l.After.Run.Subject == t.Root && l.After.Projection.Status == "cancelled" {
			t.Events = append(t.Events, joinmodel.Event{Kind: "root_terminal", Actor: t.Root, Sequence: l.After.Sequence, Status: "cancelled", Reason: l.Reason})
		}
	}
	return t, nil
}

// joinCut observes committed retained state only. It neither chooses work nor
// supplies inputs. The first cut additionally establishes the decisive lifetime.
func joinCut(s runtime.Inspection, cut string) bool {
	candidateConsumed, reportCommitted, reportConsumed, sAdmitted := false, false, false, false
	for _, row := range s.Tables["observations"] {
		o := decode[observation](row, "descriptor")
		if row["consumed_sequence"] == nil {
			continue
		}
		if o.Run.Subject == "root" && o.Kind == "invocation" {
			candidateConsumed = true
		}
		if o.Run.Subject == "root" && o.Kind == "message" && o.Source == "peer" {
			reportConsumed = true
		}
	}
	for _, row := range s.Tables["intents"] {
		i := decode[intent](row, "descriptor")
		if i.Run.Subject == "peer" && i.Kind == "message" {
			reportCommitted = true
		}
	}
	for _, row := range s.Tables["composition_admission_decisions"] {
		a := decode[runtime.InvocationAdmission](row, "descriptor")
		if a.Invocation.Caller.Subject == "peer" {
			sAdmitted = true
		}
	}
	p := projection(s, "root")
	atJoin := false
	for _, row := range s.Tables["runs"] {
		if row["subject"] != "root" {
			continue
		}
		for _, program := range s.Tables["programs"] {
			if program["id"] == row["program_id"] {
				code := decode[evidenceProgram](program, "bytes").Code
				if p.PC >= 0 && p.PC < len(code) {
					i := code[p.PC]
					atJoin = i.Op == "receive" && i.Match != nil && i.Match.Kind == "message"
				}
			}
		}
	}
	switch cut {
	case "candidate-consumed":
		return candidateConsumed && !reportCommitted && !sAdmitted && p.Status == "open" && atJoin
	case "report-committed":
		return candidateConsumed && reportCommitted && !reportConsumed && p.Status == "open" && atJoin
	}
	return false
}
