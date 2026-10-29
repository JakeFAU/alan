//go:build integration

package substitutiontest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	joinapp "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	communication "github.com/JakeFAU/alan/research/communication"
	model "github.com/JakeFAU/alan/research/substitution"
)

func communicationFixture(t *testing.T) string {
	t.Helper()
	d, e := joinapp.New(candidate(t, "b"), app.Config{Shape: "parallel", Request: app.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: limits(3, 3, 2)}, joinapp.Options{})
	must(t, e)
	path := filepath.Join(t.TempDir(), "communication.db")
	create(t, d, path)
	return path
}
func TestCommunicationJ1PublicPair(t *testing.T) {
	for _, order := range []string{"late", "early"} {
		t.Run(order, func(t *testing.T) {
			path := communicationFixture(t)
			s := driveCommunication(t, path, communication.J1, order)
			tr, e := decodeCommunication(s, model.Request{ID: 1, Value: 0, Mode: "answer"}, communication.J1)
			must(t, e)
			retain(t, "result", map[string]any{"profile": "public-driver", "order": order, "inspection": s, "trace": tr})
			m := tr.Messages[0]
			if os.Getenv("ALAN_K_RED") == "send-is-enough" && order == "late" {
				if !m.FirstAuthorization.Allowed {
					t.Fatal("falsified Send-is-enough hypothesis: first authorization", m.FirstAuthorization.Reason)
				}
			}
			if order == "late" {
				if m.FirstAuthorization.Allowed || m.Admission.Accepted || m.Observation != nil || m.Consumption != nil || projection(s, "root").Status != "open" || tr.Aggregate != "" {
					t.Fatal("late disposition/pending mismatch")
				}
			} else {
				if !m.FirstAuthorization.Allowed || !m.Admission.Accepted || m.Consumption == nil || projection(s, "root").Status != "succeeded" || tr.Aggregate != "succeeded" {
					t.Fatal("early original delivery/consumption absent")
				}
			}
			if s.Roots[0].Charges.Invocations != 3 || s.Roots[0].Charges.CreatedRuns != 3 || s.Roots[0].Charges.Effects != 0 || len(s.Tables["runs"]) != 5 {
				t.Fatal("unexpected actual charges/runs")
			}
			replayUnchanged(t, path)
		})
	}
}

func TestCommunicationAutomatic(t *testing.T) {
	path := communicationFixture(t)
	p := startCommunication(t, path, "report-no-authorization", "automatic", "")
	ack := p.acknowledgment(t)
	before := inspect(t, path)
	if !communicationCut(before, "report-no-authorization") {
		t.Fatal("fresh automatic post-Send predicate absent")
	}
	queue := communicationQueue(before)
	want := []string{communicationIdentity("message-delivery", communicationReport(before).ID)}
	if !reflect.DeepEqual(queue["pending_internal_keys"], want) || len(queue["unfinished_invocations"].([]string)) != 0 {
		t.Fatal("actual queue not sole report", queue)
	}
	retain(t, "automatic-cut", map[string]any{"inspection": before, "ack": ack, "queue": queue, "profile": "unchanged automatic B/J1"})
	p.resume(t)
	p.wait(t, false)
	after := inspect(t, path)
	tr, e := decodeCommunication(after, model.Request{ID: 1, Value: 0, Mode: "answer"}, communication.J1)
	must(t, e)
	if tr.Aggregate != "succeeded" || !tr.Messages[0].FirstAuthorization.Allowed || tr.Messages[0].Consumption == nil {
		t.Fatal("automatic actual completion absent")
	}
	retain(t, "automatic-final", map[string]any{"inspection": after, "trace": tr, "j2_runtime": "unavailable; frozen branch type mismatch construction obstruction"})
	replayUnchanged(t, path)
}
