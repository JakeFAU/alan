package communication

import (
	"slices"
	"testing"

	"github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/research/substitution"
)

func goodTrace() Trace {
	s := join.Outcome{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1, Value: 0}
	report := join.Report{RequestID: 1, SOperation: "s-original", Outcome: s}
	rep := join.Event{Actor: "peer", Operation: "report-original", Source: "peer", Target: "root", Generation: 1, Obligation: ReportObligation, Report: report}
	commit := rep
	commit.Kind = "report_commit"
	commit.Sequence = 34
	delivered := rep
	delivered.Kind = "report_deliver"
	delivered.Status = "accepted"
	consumed := rep
	consumed.Kind = "report_consume"
	consumed.Actor = "root"
	consumed.Sequence = 3
	tr := Trace{Profile: J2, AckObligation: AckObligation, Truth: join.Trace{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Root: "root", Peer: "peer", Generation: 1, Obligation: ReportObligation, Events: []join.Event{
		{Kind: "outcome_consume", Actor: "peer", Operation: "s-original", Role: "s", Sequence: 33, Outcome: s}, commit, delivered,
		{Kind: "outcome_consume", Actor: "root", Operation: "e-original", Role: "candidate", Sequence: 2, Outcome: join.Outcome{Status: "succeeded", Reason: "member_succeeded", Present: true, ID: 1, Value: 1}}, consumed,
		{Kind: "root_terminal", Actor: "root", Sequence: 5, Status: "succeeded"},
		{Kind: "peer_terminal", Actor: "peer", Sequence: 36, Status: "succeeded"},
	}}, Aggregate: "succeeded"}
	message := func(kind, id, source, target, obligation string, seq, receiveSeq int64, p Payload) Message {
		auth := Authorization{ID: "auth-" + id, IntentID: id, Source: source, Generation: 1, Epoch: 1, Revision: 1, Sequence: seq, SourceStatus: "open", Allowed: true, Reason: "authorized"}
		adm := Admission{ID: "delivery-" + id, IntentID: id, AuthorizationID: auth.ID, Accepted: true, Reason: "accepted", ObservationID: "observation-" + id}
		return Message{Kind: kind, IntentID: id, Source: source, Target: target, Obligation: obligation, SourceGeneration: 1, TargetGeneration: 1, CommitSequence: seq, Payload: p, FirstAuthorization: &auth, Reauthorizations: []Authorization{auth}, Admission: &adm, Readmissions: []Admission{adm}, Observation: &Observation{ID: adm.ObservationID, AdmissionID: adm.ID, IntentID: id, Source: source, Target: target, Obligation: obligation, SourceGeneration: 1, TargetGeneration: 1, Payload: p}, DispatchAcknowledgment: &DispatchAcknowledgment{IntentID: id, AdmissionID: adm.ID}, Consumption: &Consumption{IntentID: id, ObservationID: adm.ObservationID, Receiver: target, Generation: 1, Sequence: receiveSeq}}
	}
	tr.Messages = []Message{message("report", "report-original", "peer", "root", ReportObligation, 34, 3, Payload{Report: report}), message("ack", "ack-original", "root", "peer", AckObligation, 4, 35, Payload{Ack: Ack{RequestID: 1, SOperation: "s-original"}})}
	return tr
}
func hasCode(v []substitution.Violation, code string) bool {
	return slices.ContainsFunc(v, func(x substitution.Violation) bool { return x.Code == code })
}
func TestOracleAdversarialCausality(t *testing.T) {
	cases := []struct {
		name, code string
		mutate     func(*Trace)
	}{
		{"denial_reason", "admission_reason", func(t *Trace) { t.Messages[1].Admission.Reason = "source_denied:source_closed" }},
		{"denial_is_not_delivery", "admission_authorization", func(t *Trace) {
			t.Messages[1].FirstAuthorization.Allowed = false
			t.Messages[1].FirstAuthorization.Reason = "source_closed"
			t.Messages[1].Reauthorizations = nil
		}},
		{"dispatch_ack_is_not_delivery", "observation_admission", func(t *Trace) {
			t.Messages[1].Admission.Accepted = false
			t.Messages[1].Admission.ObservationID = ""
			t.Messages[1].Readmissions = nil
		}},
		{"admission_is_not_consumption", "ack_before_peer_terminal", func(t *Trace) { t.Messages[1].Consumption = nil }},
		{"foreign_selected_observation", "consumption_identity", func(t *Trace) { t.Messages[1].Consumption.ObservationID = "foreign" }},
		{"foreign_message", "admission_identity", func(t *Trace) { t.Messages[1].Admission.IntentID = "foreign" }},
		{"foreign_authorization", "admission_identity", func(t *Trace) { t.Messages[1].Admission.AuthorizationID = "foreign" }},
		{"changed_authorization", "authorization_reuse", func(t *Trace) { t.Messages[1].Reauthorizations[0].Sequence++ }},
		{"changed_admission", "admission_reuse", func(t *Trace) { t.Messages[1].Readmissions[0].Reason = "changed" }},
		{"authorization_predates_send", "authorization_sequence", func(t *Trace) { t.Messages[1].FirstAuthorization.Sequence = 3 }},
		{"allowed_after_terminal", "authorization_source", func(t *Trace) { t.Messages[1].FirstAuthorization.Sequence = 5 }},
		{"consumption_without_observation", "consumption_identity", func(t *Trace) { t.Messages[1].Observation = nil }},
		{"foreign_receiver", "consumption_identity", func(t *Trace) { t.Messages[1].Consumption.Receiver = "root" }},
		{"stale_generation", "observation_identity", func(t *Trace) { t.Messages[1].Observation.TargetGeneration = 2 }},
		{"changed_payload", "observation_payload", func(t *Trace) { t.Messages[1].Observation.Payload.Ack.RequestID = 2 }},
		{"wrong_ack_request", "ack_correlation", func(t *Trace) { t.Messages[1].Payload.Ack.RequestID = 2 }},
		{"wrong_ack_handle", "ack_correlation", func(t *Trace) { t.Messages[1].Payload.Ack.SOperation = "other-s" }},
		{"ack_before_report", "ack_after_report", func(t *Trace) { t.Messages[1].CommitSequence = 3 }},
		{"root_before_ack", "ack_before_root_terminal", func(t *Trace) { t.Truth.Events[5].Sequence = 4 }},
		{"peer_before_ack", "ack_before_peer_terminal", func(t *Trace) { t.Truth.Events[6].Sequence = 35 }},
		{"aggregate_with_open_peer", "aggregate_members", func(t *Trace) { t.Truth.Events = t.Truth.Events[:6] }},
		{"report_admitted_unconsumed", "report_consumption", func(t *Trace) { t.Messages[0].Consumption = nil }},
		{"report_truth_changed", "report_mapping", func(t *Trace) { t.Messages[0].Payload.Report.Outcome.Reason = "invented" }},
		{"wrong_dispatch_link", "dispatch_identity", func(t *Trace) { t.Messages[1].DispatchAcknowledgment.AdmissionID = "other" }},
	}
	if f := CheckTrace(goodTrace()); len(f) > 0 {
		t.Fatal(f)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := goodTrace()
			c.mutate(&tr)
			if f := CheckTrace(tr); !hasCode(f, c.code) {
				t.Fatalf("want %s, got %+v", c.code, f)
			}
		})
	}
}
func TestOraclePendingAndReordered(t *testing.T) {
	tr := goodTrace()
	slices.Reverse(tr.Truth.Events)
	slices.Reverse(tr.Messages)
	if f := CheckTrace(tr); len(f) > 0 {
		t.Fatal(f)
	}
	tr = goodTrace()
	tr.Aggregate = ""
	tr.Truth.Events = tr.Truth.Events[:6]
	a := &tr.Messages[1]
	a.FirstAuthorization.Allowed = false
	a.FirstAuthorization.Reason = "source_closed"
	a.FirstAuthorization.SourceStatus = "succeeded"
	a.FirstAuthorization.Sequence = 5
	a.Reauthorizations = []Authorization{*a.FirstAuthorization}
	a.Admission.Accepted = false
	a.Admission.Reason = "source_denied:source_closed"
	a.Admission.ObservationID = ""
	a.Readmissions = []Admission{*a.Admission}
	a.Observation = nil
	a.Consumption = nil
	if f := CheckTrace(tr); len(f) > 0 {
		t.Fatalf("honest pending refusal rejected: %+v", f)
	}
	a.Admission = nil
	a.Readmissions = nil
	a.DispatchAcknowledgment = nil
	if f := CheckTrace(tr); len(f) > 0 {
		t.Fatalf("authorization-only prefix rejected: %+v", f)
	}
}

func TestOracleJ1AndAuthorizationEpoch(t *testing.T) {
	tr := goodTrace()
	tr.Profile = J1
	tr.Truth.Obligation = join.Obligation
	tr.Messages = tr.Messages[:1]
	for i := range tr.Truth.Events {
		if tr.Truth.Events[i].Obligation == ReportObligation {
			tr.Truth.Events[i].Obligation = join.Obligation
		}
	}
	tr.Messages[0].Obligation = join.Obligation
	tr.Messages[0].Observation.Obligation = join.Obligation
	// A renewed activation epoch need not equal the epoch of the Send history.
	tr.Messages[0].FirstAuthorization.Epoch = 7
	tr.Messages[0].Reauthorizations = []Authorization{*tr.Messages[0].FirstAuthorization}
	if f := CheckTrace(tr); len(f) > 0 {
		t.Fatal(f)
	}
	tr.Truth.Events[0].Outcome.Reason = "altered-source-outcome"
	if f := CheckTrace(tr); !hasCode(f, "report_outcome") {
		t.Fatalf("unchanged truthful-report oracle not applied: %+v", f)
	}
}
func TestOracleAdmittedPendingPrefix(t *testing.T) {
	tr := goodTrace()
	tr.Aggregate = ""
	tr.Truth.Events = tr.Truth.Events[:6]
	tr.Messages[1].Consumption = nil
	if f := CheckTrace(tr); len(f) > 0 {
		t.Fatalf("admitted-but-unconsumed honest prefix rejected: %+v", f)
	}
	tr = goodTrace()
	tr.Messages[0].FirstAuthorization = nil
	tr.Messages[0].Reauthorizations = nil
	if f := CheckTrace(tr); !hasCode(f, "admission_identity") {
		t.Fatalf("accepted report without auth allowed: %+v", f)
	}
}

// The report/ACK round trip must not place ACK consumption before its own
// causal prerequisites in the same peer's retained sequence.
func TestOraclePeerRoundTripCausality(t *testing.T) {
	cases := []struct {
		name, code string
		mutate     func(*Trace)
	}{
		{"ack_consumed_before_report_send", "ack_after_report_commit", func(tr *Trace) { tr.Messages[1].Consumption.Sequence = 1 }},
		{"ack_consumed_at_report_send", "ack_after_report_commit", func(tr *Trace) { tr.Messages[1].Consumption.Sequence = 34 }},
		{"report_authorized_at_ack_consumption", "ack_after_report_authorization", func(tr *Trace) {
			tr.Messages[0].FirstAuthorization.Sequence = 35
			tr.Messages[0].Reauthorizations = []Authorization{*tr.Messages[0].FirstAuthorization}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := goodTrace()
			if failures := CheckTrace(tr); len(failures) != 0 {
				t.Fatalf("valid control rejected: %+v", failures)
			}
			tc.mutate(&tr)
			if failures := CheckTrace(tr); !hasCode(failures, tc.code) {
				t.Fatalf("want %s for impossible peer round trip, got %+v", tc.code, failures)
			}
			slices.Reverse(tr.Truth.Events)
			slices.Reverse(tr.Messages)
			if failures := CheckTrace(tr); !hasCode(failures, tc.code) {
				t.Fatalf("reordering removed %s: %+v", tc.code, failures)
			}
		})
	}
}
