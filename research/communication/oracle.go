// Package communication specifies Phase 1K ordinary-message obligations without
// importing ALAN authoring, execution, storage, or runtime implementations.
package communication

import (
	"github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/research/substitution"
)

const (
	J1               = "J1"
	J2               = "J2"
	ReportObligation = "phase1k_s_report"
	AckObligation    = "phase1k_s_ack"
)

type Ack struct {
	RequestID  int
	SOperation string
}
type Payload struct {
	Report join.Report
	Ack    Ack
}

// Authorization is the first durable source disposition, including the original
// source snapshot. Reauthorizations must equal this record, even after reopening.
type Authorization struct {
	ID, IntentID, Source                  string
	Generation, Epoch, Revision, Sequence int64
	SourceStatus                          string
	Allowed                               bool
	Reason                                string
}

// Admission and Observation are distinct from actual Receive consumption.
// ID is the original transport/admission identity, not the message intent ID.
type Admission struct {
	ID, IntentID, AuthorizationID string
	Accepted                      bool
	Reason, ObservationID         string
}
type Observation struct {
	ID, AdmissionID, IntentID, Source, Target, Obligation string
	SourceGeneration, TargetGeneration                    int64
	Payload                                               Payload
}
type DispatchAcknowledgment struct{ IntentID, AdmissionID string }

// Consumption must be decoded from successful history.Selected and the matching
// consumed_sequence, never inferred from admission or dispatch bookkeeping.
type Consumption struct {
	IntentID, ObservationID, Receiver string
	Generation, Sequence              int64
}
type Message struct {
	Kind, IntentID, Source, Target, Obligation         string
	SourceGeneration, TargetGeneration, CommitSequence int64
	Payload                                            Payload
	FirstAuthorization                                 *Authorization
	Reauthorizations                                   []Authorization
	Admission                                          *Admission
	Readmissions                                       []Admission
	Observation                                        *Observation
	DispatchAcknowledgment                             *DispatchAcknowledgment
	Consumption                                        *Consumption
}

// Truth explicitly maps original S/candidate/report events into the unchanged
// J1 oracle. Array order is not a global causal order. Empty Aggregate is pending.
type Trace struct {
	Profile       string
	Truth         join.Trace
	AckObligation string
	Messages      []Message
	Aggregate     string
}

// CheckTrace checks prefix safety. It accepts an honestly pending trace. The raw
// adapter must establish authentic history, snapshots and selection before calling
// it; these plain data records are not authentication tokens.
func CheckTrace(t Trace) []substitution.Violation {
	failures := join.CheckTrace(t.Truth)
	fail := func(i int, code, detail string) {
		failures = append(failures, substitution.Violation{Index: i, Code: code, Detail: detail})
	}
	if t.Profile != J1 && t.Profile != J2 {
		fail(-1, "profile", "unknown communication profile")
	}
	if t.Truth.Root == "" || t.Truth.Peer == "" || t.Truth.Root == t.Truth.Peer || t.Truth.Generation < 1 || t.Truth.Obligation == "" {
		fail(-1, "trace_identity", "missing fixed context identities")
	}
	terminals := map[string]join.Event{}
	commits := map[string]join.Event{}
	deliveries := map[string]join.Event{}
	consumes := map[string]join.Event{}
	for _, e := range t.Truth.Events {
		switch e.Kind {
		case "root_terminal", "peer_terminal":
			terminals[e.Actor] = e
		case "report_commit":
			commits[e.Operation] = e
		case "report_deliver":
			if e.Status == "accepted" {
				deliveries[e.Operation] = e
			}
		case "report_consume":
			consumes[e.Operation] = e
		}
	}
	seen := map[string]bool{}
	kinds := map[string]int{}
	var report, ack *Message
	for i := range t.Messages {
		m := &t.Messages[i]
		if m.IntentID == "" || seen[m.IntentID] {
			fail(i, "message_identity", "missing or duplicate original intent")
		}
		seen[m.IntentID] = true
		kinds[m.Kind]++
		if kinds[m.Kind] > 1 {
			fail(i, "extra_message", m.Kind)
		}
		source, target, obligation := "", "", ""
		switch m.Kind {
		case "report":
			report = m
			source = t.Truth.Peer
			target = t.Truth.Root
			obligation = t.Truth.Obligation
		case "ack":
			ack = m
			source = t.Truth.Root
			target = t.Truth.Peer
			obligation = t.AckObligation
			if t.Profile != J2 || obligation == "" {
				fail(i, "ack_profile", "ACK outside J2 or missing correlation obligation")
			}
		default:
			fail(i, "message_kind", m.Kind)
		}
		if m.Source != source || m.Target != target || m.Obligation != obligation || m.SourceGeneration != t.Truth.Generation || m.TargetGeneration != t.Truth.Generation || m.CommitSequence <= 0 {
			fail(i, "message_provenance", m.IntentID)
		}
		term, terminal := terminals[m.Source]
		if terminal && m.CommitSequence >= term.Sequence {
			fail(i, "commit_after_terminal", m.IntentID)
		}
		a := m.FirstAuthorization
		if a != nil {
			if a.ID == "" || a.IntentID != m.IntentID || a.Source != m.Source || a.Generation != m.SourceGeneration || a.Epoch < 1 || a.Revision < 1 {
				fail(i, "authorization_identity", m.IntentID)
			}
			if a.Sequence < m.CommitSequence {
				fail(i, "authorization_sequence", m.IntentID)
			}
			if a.Allowed {
				if a.SourceStatus != "open" || a.Reason != "authorized" || (terminal && a.Sequence >= term.Sequence) {
					fail(i, "authorization_source", "allowed first authorization lacks an open source snapshot")
				}
			} else if a.Reason == "source_closed" {
				if a.SourceStatus == "open" || !terminal || a.Sequence < term.Sequence || a.SourceStatus != term.Status {
					fail(i, "authorization_source", "source_closed lacks corresponding terminal snapshot")
				}
			}
		}
		for _, r := range m.Reauthorizations {
			if a == nil || r != *a {
				fail(i, "authorization_reuse", m.IntentID)
			}
		}
		d := m.Admission
		if d != nil {
			if a == nil || d.ID == "" || d.IntentID != m.IntentID || (a != nil && d.AuthorizationID != a.ID) {
				fail(i, "admission_identity", m.IntentID)
			}
			if d.Accepted && d.Reason != "accepted" {
				fail(i, "admission_reason", m.IntentID)
			}
			if a != nil && !a.Allowed && (!d.Accepted && d.Reason != "source_denied:"+a.Reason) {
				fail(i, "admission_reason", m.IntentID)
			}
			if d.Accepted && (a == nil || !a.Allowed) {
				fail(i, "admission_authorization", m.IntentID)
			}
			if d.Accepted && d.ObservationID == "" {
				fail(i, "observation_admission", "accepted admission lacks observation identity")
			}
			if !d.Accepted && d.ObservationID != "" {
				fail(i, "observation_admission", "refused admission claims an observation")
			}
		}
		for _, r := range m.Readmissions {
			if d == nil || r != *d {
				fail(i, "admission_reuse", m.IntentID)
			}
		}
		o := m.Observation
		if o != nil {
			if d == nil || !d.Accepted {
				fail(i, "observation_admission", m.IntentID)
			}
			if d == nil || o.ID == "" || o.ID != d.ObservationID || o.AdmissionID != d.ID || o.IntentID != m.IntentID || o.Source != m.Source || o.Target != m.Target || o.Obligation != m.Obligation || o.SourceGeneration != m.SourceGeneration || o.TargetGeneration != m.TargetGeneration {
				fail(i, "observation_identity", m.IntentID)
			}
			if o.Payload != m.Payload {
				fail(i, "observation_payload", m.IntentID)
			}
		} else if d != nil && d.Accepted {
			fail(i, "observation_admission", "accepted admission lacks retained observation")
		}
		if k := m.DispatchAcknowledgment; k != nil {
			if d == nil || k.IntentID != m.IntentID || k.AdmissionID != d.ID {
				fail(i, "dispatch_identity", m.IntentID)
			}
		}
		if c := m.Consumption; c != nil {
			if o == nil || c.IntentID != m.IntentID || (o != nil && c.ObservationID != o.ID) || c.Receiver != m.Target || c.Generation != m.TargetGeneration || c.Sequence <= 0 {
				fail(i, "consumption_identity", m.IntentID)
			}
			if end, ok := terminals[m.Target]; ok && c.Sequence >= end.Sequence {
				fail(i, "consumption_after_terminal", m.IntentID)
			}
		}
		if m.Kind == "report" {
			e, ok := commits[m.IntentID]
			if !ok || e.Sequence != m.CommitSequence || e.Report != m.Payload.Report || e.Source != m.Source || e.Target != m.Target || e.Generation != m.SourceGeneration || e.Obligation != m.Obligation || m.Payload.Ack != (Ack{}) {
				fail(i, "report_mapping", m.IntentID)
			}
			delivered, ok := deliveries[m.IntentID]
			accepted := d != nil && d.Accepted
			if ok != accepted || (ok && delivered.Report != m.Payload.Report) {
				fail(i, "report_delivery", m.IntentID)
			}
			consumed, ok := consumes[m.IntentID]
			if ok != (m.Consumption != nil) || (ok && (consumed.Sequence != m.Consumption.Sequence || consumed.Report != m.Payload.Report)) {
				fail(i, "report_consumption", m.IntentID)
			}
		}
	}
	for id := range commits {
		if !seen[id] {
			fail(-1, "report_mapping", "truth event has no original message record")
		}
	}
	if ack != nil {
		consumed, ok := join.Event{}, false
		if report != nil {
			consumed, ok = consumes[report.IntentID]
		}
		if !ok || ack.CommitSequence <= consumed.Sequence {
			fail(-1, "ack_after_report", "ACK must follow actual report consumption")
		}
		if report == nil || ack.Payload.Ack.RequestID != t.Truth.Request.ID || ack.Payload.Ack.SOperation != report.Payload.Report.SOperation || ack.Payload.Report != (join.Report{}) {
			fail(-1, "ack_correlation", "ACK must echo original request and S handle")
		}
		// These are peer-local coordinates. The intervening report delivery and
		// root ACK commitment give causal links, not a cross-run total order.
		if report != nil && ack.Consumption != nil {
			if report.CommitSequence >= ack.Consumption.Sequence {
				fail(-1, "ack_after_report_commit", "peer ACK consumption must follow its original report Send")
			}
			if a := report.FirstAuthorization; a != nil && a.Sequence >= ack.Consumption.Sequence {
				fail(-1, "ack_after_report_authorization", "first report authorization must precede peer ACK consumption")
			}
		}
	}
	if t.Profile == J2 {
		if e, ok := terminals[t.Truth.Root]; ok && (ack == nil || ack.CommitSequence >= e.Sequence) {
			fail(-1, "ack_before_root_terminal", "root terminated before original ACK commitment")
		}
		if e, ok := terminals[t.Truth.Peer]; ok && (ack == nil || ack.Consumption == nil || ack.Consumption.Sequence >= e.Sequence) {
			fail(-1, "ack_before_peer_terminal", "peer terminated without earlier exact ACK consumption")
		}
	}
	if t.Aggregate != "" {
		r, rok := terminals[t.Truth.Root]
		p, pok := terminals[t.Truth.Peer]
		if !rok || !pok {
			fail(-1, "aggregate_members", "aggregate completion with an open static member")
		} else {
			want := "failed"
			if r.Status == "succeeded" && p.Status == "succeeded" {
				want = "succeeded"
			}
			if t.Aggregate != want {
				fail(-1, "aggregate_status", t.Aggregate)
			}
		}
	}
	return failures
}
