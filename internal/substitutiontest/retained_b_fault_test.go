//go:build integration

package substitutiontest

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// A distinct, freshly certified program actually selects an absent outcome and
// then reads a deleted field. The historical B candidate guards this read.
func TestRetainedBOptionalResponseFaultControl(t *testing.T) {
	port := map[string]author.Port{protocol.Port: {Peer: protocol.Peer, Obligation: protocol.Obligation,
		Receive: protocol.Request(), Send: protocol.Response()}}
	makeMember := func(def author.Definition) (author.Artifact, author.Template) {
		t.Helper()
		a, err := author.Compile(def)
		must(t, err)
		component, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", protocol.Port)
		must(t, err)
		return a, component
	}
	bind := map[string]string{"id": "id", "value": "value", "mode": "mode"}
	_, helper := makeMember(author.Definition{Name: "phase1s-no-helper-reply", Ports: port, Steps: []author.Step{
		author.Receive(protocol.Port, bind), author.Success(),
	}})
	member, component := makeMember(author.Definition{Name: "phase1s-unguarded-response", Ports: port,
		Invocations: map[string]author.Template{"helper": helper}, Steps: []author.Step{
			author.Receive(protocol.Port, bind), author.Set("result_value", author.Integer(73)),
			author.Invoke("operation", "helper", map[string]author.Expr{"id": author.Var("id"), "value": author.Var("value"), "mode": author.String("answer")}, nil),
			author.AwaitInvocation("helper", "operation", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"value": "result_value"}}),
			author.Send(protocol.Port, map[string]author.Expr{"id": author.Var("id"), "value": author.Var("result_value")}), author.Success(),
		}})
	cert, err := nominal.CheckTemplate(component)
	must(t, err)
	d, err := join.New(component, substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: limits(4, 4, 3)})
	must(t, err)
	bound := 1
	for _, a := range d.Provision.Programs {
		var code evidenceProgram
		must(t, json.Unmarshal(a.Bytes, &code))
		bound += len(code.Code)
	}
	file := filepath.Join(t.TempDir(), "missing-response.db")
	store, err := runtime.Create(t.Context(), file, d.Provision, runtime.DefaultLimits())
	must(t, err)
	report, runErr := store.Run(t.Context(), runtime.RunOptions{MaxSteps: bound + 1, MaxDispatch: 13, MaxInFlight: 1})
	must(t, store.Close())
	if runErr == nil || !strings.Contains(runErr.Error(), `missing or invalid data "result_value"`) || report.Quiescent {
		t.Fatalf("expected a propagated read fault: report=%+v err=%v", report, runErr)
	}
	s := inspect(t, file)
	var e, h runtime.InvocationAdmission
	for _, row := range s.Tables["composition_admission_decisions"] {
		a := decode[runtime.InvocationAdmission](row, "descriptor")
		switch a.Invocation.TemplateID {
		case component.ID():
			e = a
		case helper.ID():
			h = a
		}
	}
	if !e.Accepted || !h.Accepted || len(e.Members) != 1 || len(h.Members) != 1 || h.Invocation.Caller != e.Members[0] ||
		e.Invocation.Depth != 2 || h.Invocation.Depth != 3 || h.Invocation.ParentInvocationID != e.Invocation.ID {
		t.Fatal("wrong original call chain", e, h)
	}
	pub := communicationRow(s, "composition_publications", "operation_id", h.Invocation.OperationID)
	delivery := communicationRow(s, "composition_outcome_deliveries", "operation_id", h.Invocation.OperationID)
	if pub == nil || delivery == nil {
		t.Fatal("missing original helper publication or delivery")
	}
	out := decode[runtime.InvocationOutcome](pub, "descriptor")
	if out.OperationID != h.Invocation.OperationID || out.TemplateID != helper.ID() || out.Caller != e.Members[0] ||
		out.ResponsePresent || out.Status != "failed" || out.Reason != "missing_reply" || projection(s, h.Members[0].Subject).Status != "succeeded" {
		t.Fatal("wrong original absent outcome", out)
	}
	obsRow := communicationRow(s, "observations", "id", delivery["observation_id"])
	if obsRow == nil {
		t.Fatal("missing delivered observation")
	}
	obs := decode[observation](obsRow, "descriptor")
	if obs.Operation != h.Invocation.OperationID || obs.Run != e.Members[0] || obs.Source != helper.ID() || obs.SourceGeneration != 1 || obs.Obligation != "helper" ||
		obs.Kind != "invocation" || obs.Provenance != "runtime-invocation" || obs.Payload["response_present"] != schema.Bool(false) {
		t.Fatal("wrong original selected observation", obs)
	}
	var previous runtime.Projection
	selected := false
	for _, row := range s.Tables["history"] {
		if row["subject"] != e.Members[0].Subject || row["generation"] != e.Members[0].Generation {
			continue
		}
		if selected {
			t.Fatal("faulting Send committed")
		}
		post := decode[runtime.Projection](row, "successor")
		if row["selected_observation"] == obs.ID {
			if previous.PC != 3 || previous.Data["operation"] != schema.Text(h.Invocation.OperationID) || previous.Data["result_value"] != schema.Int(73) ||
				post.PC != 4 || post.Status != "open" || post.Data["present"] != schema.Bool(false) || post.Data["reason"] != schema.Text("missing_reply") ||
				row["sequence"] != obsRow["consumed_sequence"] {
				t.Fatal("wrong actual Await or successor", previous, post)
			}
			if _, present := post.Data["result_value"]; present {
				t.Fatal("absent field survived selection")
			}
			selected = true
		}
		previous = post
	}
	var prog struct {
		Code []struct {
			Op     string
			Effect *struct {
				Kind    string
				Payload map[string]struct{ Ref string }
			}
		}
	}
	must(t, json.Unmarshal(member.Bytes(), &prog))
	stopped := projection(s, e.Members[0].Subject)
	if !selected || stopped.PC != 4 || prog.Code[4].Op != "emit" || prog.Code[4].Effect == nil || prog.Code[4].Effect.Kind != "message" ||
		prog.Code[4].Effect.Payload["value"].Ref != "result_value" || communicationRow(s, "composition_publications", "operation_id", e.Invocation.OperationID) != nil ||
		s.Roots[0].Outcome != nil || len(s.Tables["history"]) > bound || report.Dispatches > 12 {
		t.Fatal("incorrect fault boundary", stopped)
	}
	t.Logf("candidate=%s helper=%s program=%s cert=%s commit_bound=%d dispatch_bound=12 actual_local=%d err=%v", component.ID(), helper.ID(), member.ID(), cert.Digest, bound, len(s.Tables["history"]), runErr)
	retain(t, "fault", map[string]any{"inspection": s, "template": json.RawMessage(component.Bytes()), "certificate": cert, "report": report, "fault": runErr.Error(), "commit_bound": bound, "dispatch_bound": 12})
}
