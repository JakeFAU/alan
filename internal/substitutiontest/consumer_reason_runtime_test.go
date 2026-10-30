//go:build integration

package substitutiontest

import (
	"fmt"
	policyapp "github.com/JakeFAU/alan/examples/consumingpolicy"
	app "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"path/filepath"
	"testing"
)

func consumerReasonFixture(t *testing.T, name, sMode string, c model.JoinConfig) (policyapp.Deployment, string) {
	t.Helper()
	template := candidate(t, "a")
	if name == "deliberate-failure" {
		var err error
		template, err = policyapp.DeliberateFailure()
		must(t, err)
	} else {
		template = candidate(t, name)
	}
	d, err := policyapp.NewReason(template, app.Config{Request: app.Request{ID: int64(c.Request.ID), Value: int64(c.Request.Value), Mode: c.Request.Mode}, Limits: limits(c.I, c.R, c.Depth)}, policyapp.Options{SMode: sMode})
	must(t, err)
	path := filepath.Join(t.TempDir(), "retained.db")
	store, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, err)
	must(t, store.Close())
	return d, path
}

func consumerReasonCheck(t *testing.T, s runtime.Inspection, c model.JoinConfig, tag, reason, status string, conforming bool) consumerReasonTrace {
	t.Helper()
	trace, err := decodeConsumerReason(s, c)
	must(t, err)
	if !trace.RawValid || len(trace.Prefix) != 0 || conforming && (len(trace.Component) != 0 || len(trace.Guarantee) != 0) {
		retain(t, "rejected", map[string]any{"inspection": s, "config": c, "trace": trace})
		t.Fatal("reason policy rejected", trace)
	}
	if tag == "answer" && conforming {
		original := []model.JoinEvent{}
		for _, event := range trace.Events {
			if event.Fact.Kind != "parent_classify" {
				original = append(original, event.JoinEvent)
			}
		}
		if failures := model.CheckConsumerPolicyGuarantee(c, original); len(failures) != 0 {
			t.Fatal("separate G_answer", failures)
		}
	}
	root := projection(s, "root")
	if root.Status != status {
		t.Fatal("root status", root.Status, status)
	}
	actual := root.Data["classification"].Text
	if actual != tag {
		t.Fatal("actual committed tag", actual, tag)
	}
	consumed := map[string]model.Observation{}
	classes, decisions, terminals := 0, 0, 0
	for _, event := range trace.Events {
		f := event.Fact
		switch f.Kind {
		case "outcome_consume":
			consumed[f.Operation] = f
		case "parent_classify":
			classes++
		case "parent_decide":
			decisions++
		case "parent_terminal":
			terminals++
		}
	}
	if status == "open" {
		if _, ok := root.Data["classification"]; ok {
			t.Fatal("pending acquired classification")
		}
		if classes+decisions+terminals != 0 || root.PC != 3 || s.Roots[0].Outcome != nil || consumed["s"].Status != "succeeded" || consumed["e"].Kind != "" {
			t.Fatal("pending original wait", trace, root)
		}
	} else {
		if classes != 1 || decisions != 1 || terminals != 1 || consumed["s"].Kind == "" || consumed["e"].Kind == "" {
			t.Fatal("both joins/classification/decision/End required", trace)
		}
		if consumed["e"].Reason != reason {
			t.Fatal("candidate reason lost", consumed["e"], reason)
		}
		if s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != status || s.Roots[0].Outcome.Reason != "member_"+status {
			t.Fatal("root reason differs", s.Roots[0].Outcome)
		}
	}
	return trace
}

func TestConsumerReasonRuntime(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, c := range model.ConsumerPolicyConfigurations() {
			t.Run(fmt.Sprintf("%s/id%d-value%d", name, c.Request.ID, c.Request.Value), func(t *testing.T) {
				d, path := consumerReasonFixture(t, name, "answer", c)
				report := run(t, path)
				s := inspect(t, path)
				trace := consumerReasonCheck(t, s, c, "answer", "member_succeeded", "succeeded", true)
				spending := int64(2)
				if name == "b" {
					spending = 3
				}
				if s.Roots[0].Charges.Invocations != spending || s.Roots[0].Charges.CreatedRuns != spending || s.Roots[0].Charges.Effects != 0 || d.FixedRuns != 1 {
					t.Fatal("spending", s.Roots)
				}
				retain(t, "result", map[string]any{"kind": "automatic", "candidate_name": name, "config": c, "inspection": s, "trace": trace, "report": report, "parent": d.Parent.ID(), "candidate": d.Candidate.ID(), "successor": d.Successor.ID()})
				replayUnchanged(t, path)
			})
		}
	}
}

func TestConsumerReasonModes(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, mode := range []string{"fail", "absent", "pending"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				c := consumerPolicyConfig()
				c.Request.Mode = mode
				_, path := consumerReasonFixture(t, name, "answer", c)
				report := run(t, path)
				s := inspect(t, path)
				tag, reason, status := "candidate_failed", "member_failed", "failed"
				if mode == "absent" {
					tag, reason = "candidate_missing_reply", "missing_reply"
				}
				if mode == "pending" {
					tag, reason, status = "", "", "open"
				}
				trace := consumerReasonCheck(t, s, c, tag, reason, status, true)
				spending := int64(2)
				if name == "b" {
					spending = 3
				}
				if s.Roots[0].Charges.Invocations != spending || s.Roots[0].Charges.CreatedRuns != spending || s.Roots[0].Charges.Effects != 0 {
					t.Fatal("spending", s.Roots)
				}
				retain(t, "result", map[string]any{"kind": "mode", "candidate_name": name, "mode": mode, "config": c, "inspection": s, "trace": trace, "report": report})
				replayUnchanged(t, path)
			})
		}
	}
}

func TestConsumerReasonControls(t *testing.T) {
	for _, control := range []string{"wrong-id", "absent-deliberate-failure", "s-fail", "s-absent", "budget2"} {
		t.Run(control, func(t *testing.T) {
			c := consumerPolicyConfig()
			name, sMode, reason := "b", "answer", "member_succeeded"
			tag := "invalid_outcome"
			switch control {
			case "wrong-id":
				name = "wrong-id"
			case "absent-deliberate-failure":
				name = "deliberate-failure"
				c.Request.Mode = "absent"
				tag, reason = "candidate_failed", "member_failed"
			case "s-fail":
				sMode = "fail"
			case "s-absent":
				sMode = "absent"
			case "budget2":
				c.I, c.R = 2, 2
				reason = "member_failed"
				tag = "candidate_failed"
			}
			_, path := consumerReasonFixture(t, name, sMode, c)
			report := run(t, path)
			s := inspect(t, path)
			trace := consumerReasonCheck(t, s, c, tag, reason, "failed", false)
			if len(trace.Guarantee) == 0 {
				t.Fatal("nonconforming/profile control accepted")
			}
			componentBad := control == "wrong-id" || control == "absent-deliberate-failure"
			if (len(trace.Component) != 0) != componentBad {
				t.Fatal("component verdict", trace.Component)
			}
			if control == "wrong-id" {
				found := false
				for _, e := range trace.Events {
					if e.Fact.Kind == "outcome_consume" && e.Fact.Operation == "e" {
						found = e.Fact.Present && e.Fact.ID == 2 && e.Fact.Value == 1
					}
				}
				if !found {
					t.Fatal("honest wrong-id reply lost")
				}
			}
			if control == "budget2" {
				refused := 0
				for _, e := range trace.Events {
					f := e.Fact
					if f.Kind == "admit" && f.Status == "rejected" {
						if f.Reason != "invocation_limit" || f.I != 0 || f.R != 0 {
							t.Fatal(f)
						}
						refused++
					}
				}
				if refused != 1 || s.Roots[0].Charges.Invocations != 2 || s.Roots[0].Charges.CreatedRuns != 2 {
					t.Fatal("scarcity accounting")
				}
			}
			retain(t, "result", map[string]any{"kind": "control", "control": control, "config": c, "inspection": s, "trace": trace, "report": report})
			replayUnchanged(t, path)
		})
	}
}
