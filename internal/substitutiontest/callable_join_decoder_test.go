//go:build integration

package substitutiontest

import (
	"encoding/json"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

// These are detached corruptions of retained evidence, not live component lies.
func TestCallableJoinDecoder(t *testing.T) {
	for _, attack := range []string{"original-j-handle", "selected-j-observation", "classification-set", "report-successor", "join-branch", "outer-branch", "cursor-snapshot"} {
		t.Run(attack, func(t *testing.T) {
			mode := "absent"
			if attack == "cursor-snapshot" {
				mode = "answer"
			}
			c := callableConfig(mode)
			d, path := callableFixture(t, "b", "", c)
			callableRun(t, path)
			source := inspect(t, path)
			tr := callableChecked(t, source, c, d)
			raw, e := json.Marshal(source)
			must(t, e)
			var changed runtime.Inspection
			must(t, json.Unmarshal(raw, &changed))
			target := reasonStep{}
			switch attack {
			case "original-j-handle":
				for _, s := range tr.Steps {
					if s.Step.Run.Subject == "root" && s.Step.PC == 0 {
						target = s.Step
					}
				}
			case "selected-j-observation":
				target = tr.Consumptions["j"].Step
			case "classification-set":
				target = tr.Classification.Step
			case "report-successor":
				target = tr.Report.Step
			case "join-branch":
				for _, s := range tr.Steps {
					if s.Step.Run == tr.Report.Step.Run && s.Step.PC == 36 {
						target = s.Step
					}
				}
			case "outer-branch":
				for _, s := range tr.Steps {
					if s.Step.Run.Subject == "root" && s.Step.PC == 44 {
						target = s.Step
					}
				}
			case "cursor-snapshot":
				for _, s := range tr.Steps {
					if s.Step.Run.Subject == "root" && s.Step.PC == 10 {
						target = s.Step
					}
				}
			}
			if target.Sequence == 0 {
				t.Fatal("attack target missing")
			}
			var row map[string]any
			for _, r := range changed.Tables["history"] {
				id, e := evidenceID(r)
				must(t, e)
				seq, e := evidenceNumber(r["sequence"])
				must(t, e)
				if id == target.Run && seq == target.Sequence {
					row = r
				}
			}
			post := decode[runtime.Projection](row, "successor")
			switch attack {
			case "original-j-handle":
				post.Data["join_operation"] = schema.Text(tr.Handles["e"])
			case "selected-j-observation":
				row["selected_observation"] = tr.Consumptions["e"].Observation.ID
			case "classification-set", "report-successor":
				post.Data["classification"] = schema.Text("candidate_failed")
			case "join-branch":
				post.PC = 34
			case "outer-branch":
				post.PC = 47
			case "cursor-snapshot":
				cursor := post.Cursors[10]
				cursor.Snapshot = schema.List(map[string]schema.Value{"id": schema.Int(2), "value": schema.Int(0)})
				post.Cursors[10] = cursor
			}
			invocationJoinWriteJSON(t, row, "successor", post)
			invocationJoinRehashHistory(t, &changed, row)
			bad, e := decodeCallableJoin(changed, c, d)
			if e == nil || bad.RawValid {
				t.Fatal("detached corruption accepted", attack)
			}
			retain(t, "rejection", map[string]any{"kind": "decoder-negative", "attack": attack, "error": e.Error(), "config": c, "source_inspection": source, "mutated_inspection": changed, "target": target})
		})
	}
}

func TestCallableJoinPrefixes(t *testing.T) {
	for _, cut := range []string{"classification", "send"} {
		t.Run(cut, func(t *testing.T) {
			c := callableConfig("absent")
			d, path := callableFixture(t, "b", "", c)
			driver := newCommunicationDriver(t, path)
			root := runtime.RunID{Subject: "root", Generation: 1}
			if !driver.step(root) {
				t.Fatal("Invoke")
			}
			tr, e := decodeCallableJoin(inspect(t, path), c, d)
			must(t, e)
			invocationJoinAdmit(t, driver, tr.Handles["j"])
			admission := decode[runtime.InvocationAdmission](communicationRow(inspect(t, path), "composition_admission_decisions", "operation_id", tr.Handles["j"]), "descriptor")
			j := admission.Members[0]
			for n := 0; n < 4; n++ {
				if !driver.step(j) {
					t.Fatal("prefix setup")
				}
			}
			tr, e = decodeCallableJoin(inspect(t, path), c, d)
			must(t, e)
			invocationJoinComplete(t, driver, tr.Handles["e"])
			invocationJoinComplete(t, driver, tr.Handles["s"])
			for n := 0; n < 64; n++ {
				s := inspect(t, path)
				tr, e = decodeCallableJoin(s, c, d)
				must(t, e)
				reached := cut == "classification" && tr.Classification != nil || cut == "send" && tr.Report != nil
				if reached {
					if !tr.RawValid || len(tr.Public) != 0 || len(tr.Truth) != 0 || tr.JoinEnd != nil || tr.OuterEnd != nil || tr.Decision != nil || s.Roots[0].Outcome != nil || len(tr.Contract) == 0 || len(tr.Outer) == 0 {
						t.Fatal("prefix promoted to completion", tr)
					}
					jpub := communicationRow(s, "composition_publications", "operation_id", tr.Handles["j"])
					if jpub != nil {
						t.Fatal("prefix has J publication")
					}
					// The false claim is detached metadata; the authentic prefix stays intact.
					retain(t, "rejection", map[string]any{"kind": "completion-claim-negative", "cut": cut, "claim": map[string]any{"join_ended": true, "join_published": true, "outer_completed": true}, "config": c, "inspection": s, "trace": tr, "rejected_because": "original history has no End, J publication, O selected observation/decision or root publication"})
					return
				}
				if !driver.step(j) {
					t.Fatal("prefix path blocked")
				}
			}
			t.Fatal("prefix not reached within fixed artifact")
		})
	}
}
