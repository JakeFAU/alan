//go:build integration

package substitutiontest

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	joinapp "github.com/JakeFAU/alan/examples/invocationjoin"
	app "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

func invocationJoinFixture(t *testing.T, name, mode, sMode string, budget int) (joinapp.Deployment, string, model.JoinConfig) {
	t.Helper()
	cfg := model.JoinConfig{Request: model.Request{ID: 1, Value: 0, Mode: mode}, I: budget, R: budget, Depth: 2}
	d, e := joinapp.New(candidate(t, name), app.Config{Request: app.Request{ID: 1, Value: 0, Mode: mode}, Limits: limits(budget, budget, 2)}, joinapp.Options{SMode: sMode})
	must(t, e)
	path := filepath.Join(t.TempDir(), "retained.db")
	s, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, e)
	must(t, s.Close())
	return d, path, cfg
}

// Catches changed result/correlation, skipped original joins and excess spending.
func TestInvocationJoinRuntime(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for id := 1; id <= 2; id++ {
			for value := 0; value <= 1; value++ {
				t.Run(fmt.Sprintf("%s/id%d-value%d", name, id, value), func(t *testing.T) {
					cfg := model.JoinConfig{Request: model.Request{ID: id, Value: value, Mode: "answer"}, I: 3, R: 3, Depth: 2}
					d, e := joinapp.New(candidate(t, name), app.Config{Request: app.Request{ID: int64(id), Value: int64(value), Mode: "answer"}, Limits: limits(3, 3, 2)}, joinapp.Options{})
					must(t, e)
					path := filepath.Join(t.TempDir(), "retained.db")
					s, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
					must(t, e)
					must(t, s.Close())
					report := run(t, path)
					raw := inspect(t, path)
					tr, e := decodeInvocationJoin(raw, cfg)
					must(t, e)
					if len(tr.Prefix) != 0 || len(tr.Component) != 0 || len(tr.Guarantee) != 0 {
						t.Fatal(tr)
					}
					want := int64(2)
					if name == "b" {
						want = 3
					}
					if d.FixedRuns != 1 || raw.Roots[0].Charges.Invocations != want || raw.Roots[0].Charges.CreatedRuns != want || raw.Roots[0].Charges.Effects != 0 || len(raw.Tables["runs"]) != int(want)+1 {
						t.Fatal("accounting", raw.Roots)
					}
					retain(t, "result", map[string]any{"inspection": raw, "trace": tr, "report": report, "fixed_runs": d.FixedRuns, "candidate": d.Candidate.ID(), "successor": d.Successor.ID(), "parent": d.Parent.ID()})
					replayUnchanged(t, path)
				})
			}
		}
	}
}

// Catches promotion of absence, skipped second joins, invented pending outcomes,
// loss of the real mutant reply, and treating insufficient entitlement as success.
func TestInvocationJoinControls(t *testing.T) {
	cases := []struct {
		name, candidate, mode, sMode, status string
		budget                               int
	}{
		{"wrong-id", "wrong-id", "answer", "answer", "failed", 3},
		{"budget2", "b", "answer", "answer", "failed", 2},
		{"candidate-fail", "b", "fail", "answer", "failed", 3},
		{"candidate-absent", "b", "absent", "answer", "failed", 3},
		{"candidate-pending", "b", "pending", "answer", "open", 3},
		{"s-fail", "b", "answer", "fail", "failed", 3},
		{"s-absent", "b", "answer", "absent", "failed", 3},
		{"s-pending", "b", "answer", "pending", "open", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, path, c := invocationJoinFixture(t, tc.candidate, tc.mode, tc.sMode, tc.budget)
			report := run(t, path)
			raw := inspect(t, path)
			tr, e := decodeInvocationJoin(raw, c)
			must(t, e)
			if !tr.RawValid || len(tr.Prefix) != 0 {
				t.Fatal("honest control history rejected", tr.Prefix)
			}
			if projection(raw, "root").Status != tc.status || len(tr.Guarantee) == 0 {
				t.Fatal("control promoted to G", tr)
			}
			if tc.name == "wrong-id" {
				if len(tr.Component) == 0 {
					t.Fatal("genuine mutant conformance violation missed")
				}
				found := false
				for _, v := range tr.Events {
					if v.Fact.Operation == "e" && v.Fact.Kind == "outcome_consume" {
						found = v.Fact.Present && v.Fact.ID == 2 && v.Fact.Value == 1 && v.Fact.Status == "succeeded"
					}
				}
				if !found {
					t.Fatal("actual wrong reply not preserved/consumed")
				}
			} else if len(tr.Component) != 0 {
				t.Fatal("component control rejected", tr.Component)
			}
			consumed := map[string]model.Observation{}
			published := map[string]model.Observation{}
			refused := []model.Observation{}
			for _, v := range tr.Events {
				switch v.Fact.Kind {
				case "outcome_consume":
					consumed[v.Fact.Operation] = v.Fact
				case "publish":
					published[v.Fact.Operation] = v.Fact
				case "admit":
					if v.Fact.Status == "rejected" {
						refused = append(refused, v.Fact)
					}
				}
			}
			if tc.status == "failed" {
				if consumed["e"].Kind == "" || consumed["s"].Kind == "" || raw.Roots[0].Outcome == nil || raw.Roots[0].Outcome.Status != "failed" {
					t.Fatal("failed control skipped join/publication")
				}
			}
			if tc.mode == "absent" {
				if consumed["e"].Present || consumed["e"].Reason != "missing_reply" {
					t.Fatal("candidate absence invented payload")
				}
				if _, ok := projection(raw, "root").Data["candidate_id"]; ok {
					t.Fatal("absent candidate response binding invented")
				}
			}
			if tc.sMode == "absent" {
				if consumed["s"].Present || consumed["s"].Reason != "missing_reply" {
					t.Fatal("S absence invented payload")
				}
				if _, ok := projection(raw, "root").Data["s_id"]; ok {
					t.Fatal("absent S response binding invented")
				}
			}
			if tc.mode == "pending" {
				if published["e"].Kind != "" || consumed["e"].Kind != "" || consumed["s"].Kind != "" || published["s"].Kind == "" {
					t.Fatal("pending candidate original wait not retained")
				}
			}
			if tc.sMode == "pending" {
				if consumed["e"].Kind == "" || consumed["s"].Kind != "" || published["s"].Kind != "" {
					t.Fatal("pending S original wait not retained")
				}
			}
			if tc.mode == "pending" || tc.sMode == "pending" {
				label := "e"
				if tc.sMode == "pending" {
					label = "s"
				}
				a := decode[runtime.InvocationAdmission](communicationRow(raw, "composition_admission_decisions", "operation_id", tr.Handles[label]), "descriptor")
				if len(a.Members) != 1 || !pendingReceive(raw, a.Members[0].Subject) {
					t.Fatal("pending required child is not at original unmatched receive")
				}
			}
			if tc.status == "open" && raw.Roots[0].Outcome != nil {
				t.Fatal("pending root published invented terminal")
			}
			if tc.budget == 2 {
				if len(refused) != 1 || refused[0].Reason != "invocation_limit" || raw.Roots[0].Charges.Invocations != 2 || raw.Roots[0].Charges.CreatedRuns != 2 {
					t.Fatal("actual scarcity/refusal evidence", refused, raw.Roots)
				}
			}
			retain(t, "result", map[string]any{"control": tc.name, "inspection": raw, "trace": tr, "report": report, "fixed_runs": d.FixedRuns, "actual_refused": refused})
			replayUnchanged(t, path)
		})
	}
}

func invocationJoinOriginal(t *testing.T, s runtime.Inspection, c model.JoinConfig) invocationJoinTrace {
	t.Helper()
	tr, e := decodeInvocationJoin(s, c)
	must(t, e)
	if !tr.RawValid || len(tr.Prefix) != 0 {
		t.Fatal("invalid original public prefix", tr.Prefix)
	}
	return tr
}
func invocationJoinAdmit(t *testing.T, d *communicationDriver, op string) {
	t.Helper()
	a, e := d.store.AdmitInvocation(t.Context(), op)
	must(t, e)
	d.record(map[string]any{"kind": "invocation_join_admit", "operation": op, "admission": a})
}
func invocationJoinComplete(t *testing.T, d *communicationDriver, op string) {
	t.Helper()
	for n := 0; n < 80; n++ {
		d.service()
		s := inspect(t, d.path)
		if communicationRow(s, "composition_outcome_deliveries", "operation_id", op) != nil {
			return
		}
		descendants := map[runtime.RunID]bool{}
		for changed := true; changed; {
			changed = false
			for _, row := range s.Tables["composition_admission_decisions"] {
				a := decode[runtime.InvocationAdmission](row, "descriptor")
				if a.Invocation.OperationID == op || descendants[a.Invocation.Caller] {
					for _, m := range a.Members {
						if !descendants[m] {
							descendants[m] = true
							changed = true
						}
					}
				}
			}
		}
		progress := false
		for _, row := range s.Tables["runs"] {
			id, e := evidenceID(row)
			must(t, e)
			if descendants[id] {
				progress = d.step(id) || progress
			}
		}
		if !progress { // A service-only transition may have admitted a new child.
			d.service()
			if reflect.DeepEqual(s, inspect(t, d.path)) {
				t.Fatal("public descendant schedule blocked", op)
			}
		}
	}
	t.Fatal("public descendant watchdog", op)
}
func invocationJoinCut(s runtime.Inspection, c model.JoinConfig) bool {
	tr, e := decodeInvocationJoin(s, c)
	if e != nil || len(tr.Prefix) != 0 || projection(s, "root").Status != "open" || projection(s, "root").PC != 3 || s.Roots[0].Outcome != nil || s.Roots[0].Charges.Invocations != 3 || s.Roots[0].Charges.CreatedRuns != 3 || s.Roots[0].Charges.Effects != 0 || tr.Handles["e"] == "" || tr.Handles["s"] == "" {
		return false
	}
	consumed := map[string]bool{}
	published := map[string]bool{}
	for _, v := range tr.Events {
		if v.Fact.Kind == "outcome_consume" {
			consumed[v.Fact.Operation] = true
		}
		if v.Fact.Kind == "publish" {
			published[v.Fact.Operation] = true
		}
	}
	if !consumed["e"] || consumed["s"] || published["s"] {
		return false
	}
	aRow := communicationRow(s, "composition_admission_decisions", "operation_id", tr.Handles["s"])
	if aRow == nil {
		return false
	}
	a := decode[runtime.InvocationAdmission](aRow, "descriptor")
	if !a.Accepted || len(a.Members) != 1 {
		return false
	}
	for _, row := range s.Tables["runs"] {
		id, e := evidenceID(row)
		if e == nil && id == a.Members[0] {
			seq, e := evidenceNumber(row["sequence"])
			p := decode[runtime.Projection](row, "projection")
			return e == nil && seq == 0 && p.Status == "open" && p.PC == 0
		}
	}
	return false
}
func invocationJoinPrefix(t *testing.T, d *communicationDriver, c model.JoinConfig, order string) runtime.Inspection {
	t.Helper()
	root := runtime.RunID{Subject: "root", Generation: 1}
	if !d.step(root) || !d.step(root) {
		t.Fatal("both original invokes not committed")
	}
	tr := invocationJoinOriginal(t, inspect(t, d.path), c)
	invocationJoinAdmit(t, d, tr.Handles["e"])
	invocationJoinAdmit(t, d, tr.Handles["s"])
	if order == "s-first" {
		invocationJoinComplete(t, d, tr.Handles["s"])
		before := inspect(t, d.path)
		prefix := invocationJoinOriginal(t, before, c)
		for _, v := range prefix.Events {
			if v.Fact.Kind == "publish" && v.Fact.Operation == "e" {
				t.Fatal("candidate completed before selected S-first cut")
			}
		}
		retain(t, "s-first-intermediate", map[string]any{"inspection": before, "trace": prefix, "outstanding": "S retained-unconsumed; candidate unfinished"})
	}
	invocationJoinComplete(t, d, tr.Handles["e"])
	if !d.step(root) {
		t.Fatal("original candidate outcome not selected")
	}
	s := inspect(t, d.path)
	invocationJoinOriginal(t, s, c)
	if order == "candidate-first" && !invocationJoinCut(s, c) {
		t.Fatal("exact unfinished-S cut absent")
	}
	return s
}

// Catches any reliance on sibling completion order while preserving fixed await order.
func TestInvocationJoinOrders(t *testing.T) {
	for _, order := range []string{"candidate-first", "s-first"} {
		t.Run(order, func(t *testing.T) {
			_, path, c := invocationJoinFixture(t, "b", "answer", "answer", 3)
			d := newCommunicationDriver(t, path)
			before := invocationJoinPrefix(t, d, c, order)
			retain(t, "selected-prefix", map[string]any{"inspection": before, "trace": invocationJoinOriginal(t, before, c), "order": order, "s_unproduced": invocationJoinCut(before, c)})
			// Run and the manual driver are sequential. Run executes the actual remaining
			// parent/S instructions and root settlement on this same Store.
			report, e := d.store.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
			must(t, e)
			if !report.Quiescent {
				t.Fatal(report)
			}
			after := inspect(t, path)
			tr := invocationJoinOriginal(t, after, c)
			if len(tr.Component) != 0 || len(tr.Guarantee) != 0 {
				t.Fatal(tr)
			}
			stablePrefix(t, before, after)
			retain(t, "result", map[string]any{"inspection": after, "trace": tr, "report": report, "order": order})
			replayUnchanged(t, path)
		})
	}
}
