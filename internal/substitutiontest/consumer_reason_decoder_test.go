//go:build integration

package substitutiontest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Detached row corruptions are not executable component mutations. Rehashing
// changed history forces the actual payload/path validator to face the attack.
func TestConsumerReasonDecoder(t *testing.T) {
	for _, mutation := range []string{"classification-tag", "classification-branch", "consumed-reason", "pending-branch", "final-tag-only"} {
		t.Run(mutation, func(t *testing.T) {
			c := consumerPolicyConfig()
			c.Request.Mode = "absent"
			if mutation == "pending-branch" {
				c.Request.Mode = "pending"
			}
			_, path := consumerReasonFixture(t, "b", "answer", c)
			run(t, path)
			original := inspect(t, path)
			trace, err := decodeConsumerReason(original, c)
			must(t, err)
			if !trace.RawValid || len(trace.Prefix) != 0 || len(trace.Guarantee) != 0 {
				t.Fatal("invalid source", trace)
			}
			bytes, err := json.Marshal(original)
			must(t, err)
			var changed runtime.Inspection
			must(t, json.Unmarshal(bytes, &changed))
			var row map[string]any
			for _, r := range changed.Tables["history"] {
				seq, err := evidenceNumber(r["sequence"])
				must(t, err)
				switch mutation {
				case "classification-tag":
					if r["subject"] == "root" && seq == trace.Classification.Step.Sequence {
						row = r
					}
				case "classification-branch":
					if r["subject"] == "root" && seq == trace.Classification.Step.Sequence-2 {
						row = r
					}
				case "consumed-reason":
					if r["subject"] == "root" && seq == trace.Consumptions["e"].Step.Sequence {
						row = r
					}
				case "pending-branch":
					if r["subject"] == trace.Pending.Step.Run.Subject && seq == trace.Pending.Step.Sequence {
						row = r
					}
				}
			}
			if mutation == "final-tag-only" {
				row = communicationRow(changed, "runs", "subject", "root")
				p := decode[runtime.Projection](row, "projection")
				p.Data["classification"] = schema.Text("candidate_failed")
				invocationJoinWriteJSON(t, row, "projection", p)
			} else {
				if row == nil {
					t.Fatal("mutation row absent")
				}
				p := decode[runtime.Projection](row, "successor")
				switch mutation {
				case "classification-tag":
					p.Data["classification"] = schema.Text("candidate_failed")
				case "classification-branch":
					p.PC = 31
				case "consumed-reason":
					p.Data["candidate_reason"] = schema.Text("member_failed")
				case "pending-branch":
					p.Data["requested_mode"] = schema.Text("absent")
				}
				invocationJoinWriteJSON(t, row, "successor", p)
				invocationJoinRehashHistory(t, &changed, row)
				if mutation == "pending-branch" {
					// Keep the apparent second-receive PC, but forge the predecessor mode
					// to absent: its recorded branch would have selected Success instead.
					for _, previous := range changed.Tables["history"] {
						seq, e := evidenceNumber(previous["sequence"])
						must(t, e)
						if previous["subject"] == trace.Pending.Step.Run.Subject && seq == trace.Pending.Step.Sequence-1 {
							pre := decode[runtime.Projection](previous, "successor")
							pre.Data["requested_mode"] = schema.Text("absent")
							invocationJoinWriteJSON(t, previous, "successor", pre)
							invocationJoinRehashHistory(t, &changed, previous)
						}
					}
					r := communicationRow(changed, "runs", "subject", trace.Pending.Step.Run.Subject)
					invocationJoinWriteJSON(t, r, "projection", p)
				}
			}
			rejected, err := decodeConsumerReason(changed, c)
			if err == nil || rejected.RawValid {
				t.Fatal("accepted detached corruption", mutation, rejected)
			}
			if mutation == "pending-branch" && !strings.Contains(err.Error(), "pending candidate path") {
				t.Fatal("pending mutation did not reach branch validation", err)
			}
			retain(t, "rejection", map[string]any{"kind": "decoder-negative", "mutation": mutation, "error": err.Error(), "config": c, "source_inspection": original, "mutated_inspection": changed})
		})
	}
}

func TestConsumerReasonClassificationCut(t *testing.T) {
	c := consumerPolicyConfig()
	c.Request.Mode = "absent"
	_, path := consumerReasonFixture(t, "b", "answer", c)
	driver := newCommunicationDriver(t, path)
	root := runtime.RunID{Subject: "root", Generation: 1}
	if !driver.step(root) || !driver.step(root) {
		t.Fatal("original invoke commits")
	}
	raw, err := decodeInvocationJoinRaw(inspect(t, path), c)
	must(t, err)
	invocationJoinAdmit(t, driver, raw.Handles["e"])
	invocationJoinAdmit(t, driver, raw.Handles["s"])
	invocationJoinComplete(t, driver, raw.Handles["e"])
	invocationJoinComplete(t, driver, raw.Handles["s"])
	// Stop at an actual committed classification Set, before its separate End.
	for i := 0; i < 52; i++ {
		s := inspect(t, path)
		p := projection(s, "root")
		if _, ok := p.Data["classification"]; ok {
			trace, err := decodeConsumerReason(s, c)
			must(t, err)
			if !trace.RawValid || len(trace.Prefix) != 0 || len(trace.Component) != 0 || len(trace.Guarantee) == 0 || trace.Classification == nil || trace.Terminal != nil || p.Status != "open" || s.Roots[0].Outcome != nil {
				t.Fatal("classification cut", trace, p)
			}
			classify, decide := 0, 0
			for _, event := range trace.Events {
				if event.Fact.Kind == "parent_classify" {
					classify++
				}
				if event.Fact.Kind == "parent_decide" {
					decide++
				}
			}
			if classify != 1 || decide != 1 || trace.Classification.Tag != "candidate_missing_reply" || trace.Classification.Decision != "failed" {
				t.Fatal("same-commit expansion", trace)
			}
			retain(t, "cut", map[string]any{"kind": "classification-cut", "config": c, "inspection": s, "trace": trace, "predicate": "classification and decision share one committed Set; End remains unexecuted"})
			report, err := driver.store.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
			must(t, err)
			final := inspect(t, path)
			completed := consumerReasonCheck(t, final, c, "candidate_missing_reply", "missing_reply", "failed", true)
			stablePrefix(t, s, final)
			retain(t, "result", map[string]any{"kind": "classification-cut-completed", "config": c, "inspection": final, "trace": completed, "report": report})
			return
		}
		if !driver.step(root) {
			t.Fatal("cannot reach actual setter")
		}
	}
	t.Fatal("no classification within compiled instruction bound")
}
