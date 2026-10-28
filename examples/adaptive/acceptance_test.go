package adaptive_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Inspect an authentic retained transition that emitted the first round-two
// item intent. The complete run is used to obtain immutable history; this test
// does not simulate a crash or manufacture a partial projection.
func TestAdaptiveAcceptedFrontierBeforeFirstIntent(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			x := execute(t, scalar(t, false), nested, 1, request(0, 1, 2), nil)
			assertStop(t, x, "exhausted", 7, 3, []int64{})
			found := false
			for _, h := range x.inspection.Tables["history"] {
				var p runtime.Projection
				decode(t, h["successor"], &p)
				if _, ok := p.Data["tokens"]; !ok {
					continue
				}
				if p.Data["rounds"].Number != 1 || p.Data["work_id"].Number != 7 || !reflect.DeepEqual(ids(t, p.Data["current"]), []int64{7, 10, 11}) || len(rows(t, p.Data["handles"])) != 0 {
					continue
				}
				var intents []struct{ ID, Kind string }
				decode(t, h["intents"], &intents)
				if len(intents) != 1 || intents[0].Kind != "invocation" {
					continue
				}
				found = true
				record := struct {
					Transition       bytePeak
					Operation        string
					Current, History []int64
					Accepted, Trial  int64
				}{retainedTransition(t, h, p.PC), intents[0].ID, ids(t, p.Data["current"]), ids(t, p.Data["history"]), p.Data["work_count"].Number, p.Data["trial"].Number}
				raw, _ := json.Marshal(record)
				t.Log("ADAPTIVE_ACCEPTANCE_BOUNDARY " + string(raw))
				if record.Accepted != 6 || record.Trial != 6 {
					t.Fatalf("whole current frontier was accepted before intent: count=%d trial=%d want6; current=%v", record.Accepted, record.Trial, record.Current)
				}
				// history records the admission-loop prefix. The full current frontier and
				// accepted count persist independently; history is complete before derivation.
				if !reflect.DeepEqual(record.History, []int64{1, 2, 3, 7}) {
					t.Fatal("admission-loop history prefix", record.History)
				}
			}
			if !found {
				t.Fatal("no real partial round-two item-intent transition retained")
			}
		})
	}
}

func TestAdaptiveAcceptedCountDenialAndEarlyStops(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			c := scalar(t, false)
			enclosing := int64(1)
			if nested {
				enclosing = 2
			}
			limit := runtime.ResourceLimits{MaxInvocations: enclosing + 2, MaxCreatedRuns: 64, MaxDepth: 3, MaxEffects: 96, MaxInFlight: 1}
			x := execute(t, c, nested, 1, request(0, 1, 2), nil, limit)
			assertStop(t, x, "root_budget_denied", 3, 1, []int64{7})
			if x.report.Roots[0].Charges.Invocations != enclosing+2 || x.report.Roots[0].Charges.CreatedRuns != enclosing+2 {
				t.Fatal("accepted work must not fabricate runtime charges", x.report.Roots)
			}
			for _, v := range []struct {
				rounds, work, count, consumed int64
				stop                          string
				remaining                     []int64
			}{{3, 2, 0, 0, "work_limit", []int64{1, 2, 3}}, {3, 5, 3, 1, "work_limit", []int64{7, 10, 11}}, {1, 8, 3, 1, "round_limit", []int64{7, 10, 11}}} {
				q := request(0, 1, 2)
				q["max_rounds"] = schema.Int(v.rounds)
				q["max_work"] = schema.Int(v.work)
				x := execute(t, c, nested, 1, q, nil)
				assertStop(t, x, v.stop, v.count, v.consumed, v.remaining)
				if x.report.Roots[0].Charges.Invocations != enclosing+v.count {
					t.Fatal("unperformed frontier was charged", v, x.report.Roots)
				}
			}
		})
	}
}
