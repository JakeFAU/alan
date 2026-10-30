//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/author"
	app "github.com/JakeFAU/alan/examples/callablejoin"
	policy "github.com/JakeFAU/alan/examples/consumingpolicy"
	original "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"path/filepath"
	"testing"
)

func callableConfig(mode string) model.CallableJoinConfig {
	return model.CallableJoinConfig{Request: model.Request{ID: 1, Value: 0, Mode: mode}, I: 4, R: 4, Depth: 3}
}

func callableFixture(t *testing.T, name, control string, c model.CallableJoinConfig) (app.Deployment, string) {
	t.Helper()
	component := candidate(t, name)
	if control == "absent-deliberate-failure" {
		var e error
		component, e = policy.DeliberateFailure()
		must(t, e)
	}
	d, e := app.New(component, original.Config{Request: original.Request{ID: int64(c.Request.ID), Value: int64(c.Request.Value), Mode: c.Request.Mode}, Limits: limits(c.I, c.R, c.Depth)})
	must(t, e)
	if control != "" && control != "absent-deliberate-failure" {
		d.JoinArtifact, d.Join, e = app.NewControl(control, component, d.Successor)
		must(t, e)
		d.Outer, e = app.NewOuter(d.Join)
		must(t, e)
		d.Provision, e = (author.Application{"root": {Artifact: d.Outer, Inputs: map[string]schema.Value{"request_id": schema.Int(int64(c.Request.ID)), "request_value": schema.Int(int64(c.Request.Value)), "request_mode": schema.Text(c.Request.Mode)}}}).Provision()
		must(t, e)
		allowed := []string{}
		for _, template := range d.Provision.Composition.Templates {
			allowed = append(allowed, template.ID)
		}
		root := runtime.RunID{Subject: "root", Generation: 1}
		d.Provision.Composition.Roots = []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root}, Templates: allowed, Limits: limits(c.I, c.R, c.Depth)}}
	}
	path := filepath.Join(t.TempDir(), "retained.db")
	store, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, e)
	must(t, store.Close())
	return d, path
}

func callableRun(t *testing.T, path string) runtime.RunReport {
	t.Helper()
	s, e := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, e)
	defer s.Close()
	report, e := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 512, MaxDispatch: 64, MaxInFlight: 1})
	must(t, e)
	return report
}

func callableChecked(t *testing.T, s runtime.Inspection, c model.CallableJoinConfig, d app.Deployment) callableTrace {
	t.Helper()
	tr, e := decodeCallableJoin(s, c, d)
	if e != nil {
		retain(t, "decode-error", map[string]any{"inspection": s, "error": e.Error()})
		t.Fatal(e)
	}
	if !tr.RawValid || len(tr.Public)+len(tr.Truth)+len(tr.Component)+len(tr.Contract)+len(tr.Outer) != 0 {
		retain(t, "verdict-error", map[string]any{"inspection": s, "trace": tr})
		t.Fatal("callable verdict", tr.Public, tr.Truth, tr.Component, tr.Contract, tr.Outer)
	}
	if c.Request.Mode == "pending" {
		if tr.Pending == nil || tr.Report != nil || tr.Classification != nil || tr.Decision != nil || tr.JoinEnd != nil || tr.OuterEnd != nil || s.Roots[0].Outcome != nil || projection(s, "root").PC != 1 {
			t.Fatal("pending boundary")
		}
		jrow := communicationRow(s, "composition_admission_decisions", "operation_id", tr.Handles["j"])
		j := decode[runtime.InvocationAdmission](jrow, "descriptor").Members[0]
		if projection(s, j.Subject).PC != 5 || tr.Consumptions["s"].Observation.ID == "" {
			t.Fatal("pending J original e await")
		}
	} else {
		tag := "answer"
		if c.Request.Mode == "fail" {
			tag = "candidate_failed"
		}
		if c.Request.Mode == "absent" {
			tag = "candidate_missing_reply"
		}
		if tr.Report == nil || tr.Classification == nil || tr.Decision == nil || tr.JoinEnd == nil || tr.OuterEnd == nil || tr.Report.Diagnostic.Classification != tag || tr.Decision.Tag != tag {
			t.Fatal("completed diagnostics/Ends", tr)
		}
		want := "succeeded"
		if tag != "answer" {
			want = "failed"
		}
		if s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != want || s.Roots[0].Outcome.Reason != "member_"+want {
			t.Fatal("root envelope")
		}
	}
	return tr
}

func callableRetain(t *testing.T, kind, name, control string, c model.CallableJoinConfig, d app.Deployment, s runtime.Inspection, tr callableTrace, report runtime.RunReport) {
	t.Helper()
	static := 0
	for _, p := range d.Provision.Programs {
		var program struct{ Code []any }
		must(t, json.Unmarshal(p.Bytes, &program))
		static += len(program.Code)
	}
	// Each fixed program is instantiated once; O's MaxItems1 next repeats once.
	if report.Steps > static+1 || report.Dispatches > 3*int(s.Roots[0].Charges.Invocations) {
		t.Fatal("artifact bound", report.Steps, static)
	}
	mapping, e := callableStaticMapping(d)
	must(t, e)
	var runnerReport any = report
	if kind == "late-reply" {
		runnerReport = nil
	}
	retainedTransitions := len(s.Tables["composition_admission_decisions"]) + len(s.Tables["message_acknowledgments"]) + len(s.Tables["composition_outcome_deliveries"])
	if len(s.Tables["history"]) > static+1 || retainedTransitions > 3*int(s.Roots[0].Charges.Invocations) {
		t.Fatal("actual retained transition bounds")
	}
	retain(t, "result", map[string]any{"kind": kind, "candidate": name, "control": control, "config": c, "inspection": s, "trace": tr, "report": runnerReport, "mapping": mapping, "actual_local_commits": len(s.Tables["history"]), "protocol_dispatch_transitions": retainedTransitions, "limits": map[string]int{"steps": 512, "dispatch": 64, "capacity": 1}, "bounds": map[string]int{"static_instructions": static, "commits": static + 1, "dispatch": 3 * int(s.Roots[0].Charges.Invocations)}, "identities": map[string]string{"outer": d.Outer.ID(), "join": d.Join.ID(), "join_program": d.JoinArtifact.ID(), "candidate": d.Candidate.ID(), "successor": d.Successor.ID()}})
}

func TestCallableJoinRuntime(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for id := 1; id <= 2; id++ {
			for value := 0; value <= 1; value++ {
				t.Run(fmt.Sprintf("%s/id%d-value%d", name, id, value), func(t *testing.T) {
					c := callableConfig("answer")
					c.Request.ID, c.Request.Value = id, value
					d, path := callableFixture(t, name, "", c)
					r := callableRun(t, path)
					s := inspect(t, path)
					tr := callableChecked(t, s, c, d)
					spend := int64(3)
					if name == "b" {
						spend = 4
					}
					if s.Roots[0].Charges.Invocations != spend || s.Roots[0].Charges.CreatedRuns != spend || s.Roots[0].Charges.Effects != 0 {
						t.Fatal("charges")
					}
					callableRetain(t, "automatic", name, "", c, d, s, tr, r)
					replayUnchanged(t, path)
				})
			}
		}
	}
}

func TestCallableJoinModes(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, mode := range []string{"fail", "absent", "pending"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				c := callableConfig(mode)
				d, path := callableFixture(t, name, "", c)
				r := callableRun(t, path)
				s := inspect(t, path)
				tr := callableChecked(t, s, c, d)
				spend := int64(3)
				if name == "b" {
					spend = 4
				}
				if s.Roots[0].Charges.Invocations != spend || s.Roots[0].Charges.CreatedRuns != spend || s.Roots[0].Charges.Effects != 0 {
					t.Fatal("charges")
				}
				if mode == "pending" && (!r.Quiescent || len(r.OpenRuns) != 3) {
					t.Fatal("actual quiescent pending runs", r)
				}
				callableRetain(t, "mode", name, "", c, d, s, tr, r)
				replayUnchanged(t, path)
			})
		}
	}
}

func TestCallableJoinControls(t *testing.T) {
	for _, control := range []string{"no-report-success", "no-report-failure", "false-candidate-failed", "absent-deliberate-failure"} {
		t.Run(control, func(t *testing.T) {
			c := callableConfig("answer")
			name := "a"
			if control == "false-candidate-failed" {
				c.Request.Mode = "absent"
				name = "b"
			}
			if control == "absent-deliberate-failure" {
				c.Request.Mode = "absent"
			}
			d, path := callableFixture(t, name, control, c)
			r := callableRun(t, path)
			s := inspect(t, path)
			tr, e := decodeCallableJoin(s, c, d)
			must(t, e)
			if !tr.RawValid || len(tr.Public) != 0 || len(tr.Contract) == 0 || len(tr.Outer) == 0 {
				t.Fatal("control layers", tr)
			}
			componentBad := control == "absent-deliberate-failure"
			if (len(tr.Component) > 0) != componentBad || (len(tr.Truth) > 0) == componentBad {
				t.Fatal("truth/profile separation", tr.Truth, tr.Component)
			}
			pub := decode[runtime.InvocationOutcome](communicationRow(s, "composition_publications", "operation_id", tr.Handles["j"]), "descriptor")
			if control == "no-report-success" || control == "no-report-failure" {
				reason := "member_failed"
				if control == "no-report-success" {
					reason = "missing_reply"
				}
				if tr.Report != nil || pub.ResponsePresent || pub.Status != "failed" || pub.Reason != reason || tr.Decision.Tag != "invalid_join_outcome" {
					t.Fatal("missing report control", pub, tr.Decision)
				}
				for _, role := range []string{"e", "s"} {
					o := tr.Consumptions[role].Observation.Payload
					want := int64(0)
					if role == "e" {
						want = 1
					}
					if o["status"].Text != "succeeded" || o["reason"].Text != "member_succeeded" || !o["response_present"].Flag || o["response.id"].Number != 1 || o["response.value"].Number != want {
						t.Fatal("control children not valid", role, o)
					}
				}
			} else {
				reason := "missing_reply"
				if componentBad {
					reason = "member_failed"
				}
				if tr.Consumptions["e"].Observation.Payload["reason"].Text != reason || tr.Consumptions["e"].Observation.Payload["response_present"].Flag || tr.Report.Diagnostic.Classification != "candidate_failed" || tr.Decision.Tag != "candidate_failed" || !pub.ResponsePresent || pub.Reason != "member_failed" {
					t.Fatal("actual false/mode report", tr)
				}
			}
			callableRetain(t, "control", name, control, c, d, s, tr, r)
			replayUnchanged(t, path)
		})
	}
}

func TestCallableJoinLateReply(t *testing.T) {
	c := callableConfig("absent")
	d, path := callableFixture(t, "b", "", c)
	driver := newCommunicationDriver(t, path)
	root := runtime.RunID{Subject: "root", Generation: 1}
	if !driver.step(root) {
		t.Fatal("outer invoke")
	}
	tr, e := decodeCallableJoin(inspect(t, path), c, d)
	must(t, e)
	jop := tr.Handles["j"]
	invocationJoinAdmit(t, driver, jop)
	ja := decode[runtime.InvocationAdmission](communicationRow(inspect(t, path), "composition_admission_decisions", "operation_id", jop), "descriptor")
	j := ja.Members[0]
	for n := 0; n < 4; n++ {
		if !driver.step(j) {
			t.Fatal("J request/invokes")
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
		if tr.Report != nil {
			break
		}
		if !driver.step(j) {
			t.Fatal("J classification/send blocked")
		}
	}
	send := inspect(t, path)
	tr, e = decodeCallableJoin(send, c, d)
	must(t, e)
	if tr.Report == nil || tr.Report.Diagnostic.Classification != "candidate_missing_reply" || tr.JoinEnd != nil || len(tr.Truth) != 0 || len(tr.Contract) == 0 {
		t.Fatal("Send prefix", tr)
	}
	reply := tr.Report.Intent.ID
	retain(t, "send-cut", map[string]any{"kind": "send-prefix", "config": c, "inspection": send, "trace": tr})
	if !driver.step(j) || !driver.step(j) {
		t.Fatal("separate branch/Failure")
	}
	ended := inspect(t, path)
	et, e := decodeCallableJoin(ended, c, d)
	must(t, e)
	if et.JoinEnd == nil || communicationRow(ended, "attempts", "intent_id", reply) != nil || communicationRow(ended, "composition_replies", "operation_id", reply) != nil {
		t.Fatal("reply was already authorized/admitted")
	}
	prep, e := driver.store.PrepareInvocation(t.Context(), jop)
	must(t, e)
	if prep.Ready || prep.Pending != "reply_admission" {
		t.Fatal("frozen reply wait", prep)
	}
	retain(t, "end-cut", map[string]any{"kind": "end-before-first-authorization", "config": c, "inspection": ended, "trace": et, "preparation": prep})
	auth := driver.authorize(reply)
	if !auth.Allowed || auth.SourceStatus != "failed" || auth.Sequence != et.JoinEnd.Sequence {
		t.Fatal("first post-End authorization", auth)
	}
	driver.deliver(reply)
	prep, e = driver.store.PrepareInvocation(t.Context(), jop)
	must(t, e)
	if !prep.Ready || !prep.Outcome.ResponsePresent || prep.Outcome.Status != "failed" || prep.Outcome.Reason != "member_failed" {
		t.Fatal("late report publication preparation", prep)
	}
	_, e = driver.store.SettleInvocation(t.Context(), jop)
	must(t, e)
	for n := 0; n < 128; n++ {
		if !driver.step(root) {
			break
		}
	}
	s := inspect(t, path)
	_, e = driver.store.SettleRoot(t.Context(), s.Roots[0].RootID)
	must(t, e)
	s = inspect(t, path)
	final := callableChecked(t, s, c, d)
	stablePrefix(t, send, s)
	stablePrefix(t, ended, s)
	orders := map[string]int64{}
	for i, event := range final.Events {
		if event.Fact.Operation == "j" {
			orders[event.Fact.Kind] = final.Orders[i]
		}
	}
	if !(orders["reply_commit"] < orders["terminal_freeze"] && orders["terminal_freeze"] < orders["reply_authorize"] && orders["reply_authorize"] < orders["reply_admit"] && orders["reply_admit"] < orders["publish"] && orders["publish"] < orders["outcome_consume"]) {
		t.Fatal("late causal order", orders)
	}
	callableRetain(t, "late-reply", "b", "", c, d, s, final, runtime.RunReport{})
	retain(t, "causality", map[string]any{"orders": orders, "first_authorization": auth, "binding_exception": "original entry-member message reply bound to caller/generation/work; source revision/generation unchanged; only bound replies bypass closed-source denial"})
	replayUnchanged(t, path)
}
