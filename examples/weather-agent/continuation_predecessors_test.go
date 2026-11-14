package main

import (
	"fmt"
	"github.com/JakeFAU/alan/runtime"
	"io"
	"reflect"
	"testing"
)

// A pending local recovery never needs a new read. Every other eligible
// predecessor keeps its original result or uncertainty after the new approval.
func TestWeatherContinuationPredecessors(t *testing.T) {
	binary := publicWeatherBinary(t)
	for _, tc := range []struct {
		name, boundary, phase, purpose string
		state                          runtime.EffectStage
		contacts, spent                int
		eligible                       bool
	}{
		{"unspent", "intent.authorized", "authorized", "refresh_after_report", runtime.EffectAuthorized, 0, 0, false},
		{"retained_unsettled", "adapter.result-recorded", "pre-settlement", "refresh_after_report", runtime.EffectResultUnsettled, 1, 1, false},
		{"settled_unselected", "adapter.settled", "pre-consumption", "refresh_after_report", runtime.EffectSettled, 1, 1, false},
		{"spent_before_dispatch", "dispatch.consider", "pre-admission", "new_read_after_undispatched_reservation", runtime.EffectReservedUnknown, 0, 1, true},
		{"lost_after_acceptance", "", "", "new_read_after_unknown_response", runtime.EffectReservedUnknown, 1, 1, true},
		{"retained_refusal", "", "", "new_read_after_unavailable_result", runtime.EffectSettled, 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := weatherReply{status: 200, body: morningWeather}
			if tc.name == "lost_after_acceptance" {
				first.hold = true
			}
			if tc.name == "retained_refusal" {
				first = weatherReply{status: 429, body: `{"error":true,"reason":"rate limited"}`}
			}
			next := weatherReply{status: 200, body: laterWeather}
			replies := []weatherReply{first, next}
			if tc.name == "spent_before_dispatch" {
				replies = []weatherReply{next}
			}
			p := newWeatherProvider(t, replies...)
			db := startWeather(t, binary, p.endpoint(), 2)
			var pauses []map[string]any
			var owner *weatherOwner
			var witness string
			if tc.boundary != "" {
				f := weatherCut{Subject: "weather-1", Generation: 1, Boundary: tc.boundary, Phase: tc.phase}
				owner = launchWeather(t, binary, db, "run", &f)
				witness = owner.awaitCut(t).witness()
			} else if first.hold {
				owner = launchWeather(t, binary, db, "run", nil)
				accepted := p.waitAccepted(t)
				witness = fmt.Sprintf("endpoint accepted index=%d operation=%s, response withheld", accepted.index, accepted.operation)
			} else {
				weatherCommand(t, binary, "run", "--db", db)
			}
			if owner != nil {
				owner.kill(t, witness)
			}
			at, _ := inspectAndReplayWeather(t, binary, db, p, 2, tc.spent)
			p.requireContacts(t, tc.contacts)
			before := weatherEffect(t, at, "weather-1")
			if before.State != tc.state || secondApproval(t, at) != nil || tc.name == "settled_unselected" && selectedWeather(t, at, before) != nil {
				t.Fatalf("wrong predecessor cut %s: %+v", tc.name, before)
			}
			var present, absent []string
			switch tc.name {
			case "unspent":
				present = []string{"Original read can run", "Opportunity unspent", "Resume may contact the provider"}
			case "retained_unsettled", "settled_unselected":
				present = []string{"Response retained (HTTP 200)", "Observation unselected", "Retained result needs local work", "weather-agent resume"}
				absent = []string{"Resume may contact the provider"}
			case "spent_before_dispatch":
				present = []string{"before local dispatch", "Optional separate approval for read 2", "Opportunity spent"}
				absent = []string{"provider response is unknown", "weather-agent resume"}
			case "lost_after_acceptance":
				present = []string{"provider response is unknown", "Resume will not retry this original", "does not settle it", "Optional separate approval for read 2"}
				absent = []string{"weather-agent resume"}
			case "retained_refusal":
				present = []string{"Response retained (HTTP 429)", "Observation selected", "http_rejected", "Optional separate approval for read 2"}
			}
			if !tc.eligible {
				absent = append(absent, "weather-agent refresh", "Eligible for separate approval")
			}
			requireWeatherText(t, humanWeatherCommand(t, "status", "--db", db), append(present, before.OperationID), absent)
			p.requireContacts(t, tc.contacts)
			if owner != nil {
				pauses = append(pauses, continuationStop(t, witness, owner.cmd.Process.Pid, tc.contacts, at))
			}
			if !tc.eligible {
				if err := run(t.Context(), []string{"refresh", "--db", db, "--read", "2"}, io.Discard); err == nil {
					t.Fatal("approved a fresh opportunity while the original can still progress")
				}
				unchanged, _ := inspectAndReplayWeather(t, binary, db, p, 2, tc.spent)
				p.requireContacts(t, tc.contacts)
				if secondApproval(t, unchanged) != nil || len(unchanged.Effects) != 1 {
					t.Fatal("refused refresh retained usable authority or an effect")
				}
				weatherCommand(t, binary, "resume", "--db", db)
				var v weatherView
				at, v = inspectAndReplayWeather(t, binary, db, p, 2, 1)
				authenticWeather(t, at, v.read(t, 1), 1, p.contacts()[0])
			}
			original := weatherEffect(t, at, "weather-1")
			contacts := len(p.contacts())
			weatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
			p.requireContacts(t, contacts)
			weatherCommand(t, binary, "resume", "--db", db)
			s, final := inspectAndReplayWeather(t, binary, db, p, 2, 2)
			p.requireContacts(t, contacts+1)
			one, two := final.read(t, 1), final.read(t, 2)
			confirmDecisionBasis(t, s, original)
			secondApproval(t, s)
			retained := weatherEffect(t, s, "weather-1")
			if s.Namespace != at.Namespace || one.Operation != original.OperationID || two.Previous != one.Operation ||
				two.Operation == one.Operation || two.ApprovalReason != tc.purpose || !reflect.DeepEqual(original.AdapterResult, retained.AdapterResult) ||
				!reflect.DeepEqual(original.AdapterAttempt, retained.AdapterAttempt) || !reflect.DeepEqual(original.DispatchAdmission, retained.DispatchAdmission) {
				t.Fatal("fresh approval changed its scope or rewrote the predecessor")
			}
			authenticWeather(t, s, two, 2, p.contacts()[contacts])
			switch tc.name {
			case "spent_before_dispatch", "lost_after_acceptance":
				unresolvedWeather(t, s, one, original.OperationID)
			case "retained_refusal":
				if one.State != "unavailable" || original.AdapterResult == nil || original.AdapterResult.Candidate.HTTPStatus != 429 || selectedWeather(t, s, retained) == nil {
					t.Fatal("refresh replaced or hid the original selected HTTP refusal")
				}
			default:
				authenticWeather(t, s, one, 1, p.contacts()[0])
			}
			recordContinuationCase(t, s, p, pauses...)
		})
	}
}
