//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	callableapp "github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/internal/executor"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"reflect"
	"sort"
)

type callableInstruction struct {
	Step        reasonStep
	Instruction executor.Instruction
	Pre, Post   runtime.Projection
	Selected    string
}
type callableReportEvidence struct {
	Step       reasonStep
	Intent     executor.Intent
	Diagnostic model.JoinDiagnostic
}
type callableTrace struct {
	RawValid                                  bool
	Events                                    []model.CallableJoinEvent
	Orders                                    []int64
	Handles                                   map[string]string
	Steps                                     []callableInstruction
	Consumptions                              map[string]reasonConsumption
	Classification, Decision                  *reasonClassificationEvidence
	Report                                    *callableReportEvidence
	JoinEnd, OuterEnd, Pending                *reasonStep
	Public, Truth, Component, Contract, Outer []model.Violation
}

// Record-link authentication derives from the historical L/M decoder. Topology,
// all instruction checks, public report/outer mapping and depth accounting are
// explicit new N logic; the old root-PC decoder is never called.
func decodeCallableJoin(s runtime.Inspection, c model.CallableJoinConfig, d callableapp.Deployment) (out callableTrace, err error) {
	out.Handles = map[string]string{}
	out.Consumptions = map[string]reasonConsumption{}
	orders := &out.Orders
	fail := func(f string, a ...any) (callableTrace, error) {
		return callableTrace{}, fmt.Errorf(f, a...)
	}
	if e := invocationJoinUniqueRows(s); e != nil {
		return out, e
	}
	closed, e := validateRetained(s)
	if e != nil {
		return out, e
	}
	if closed || len(s.Tables["lifecycle_results"]) != 0 {
		return fail("outside stable no-cancellation profile")
	}
	root, e := readEvidence[runtime.ResourceRoot](s.Tables["composition_roots"][0], "descriptor")
	if e != nil {
		return out, e
	}
	if root.Run != (runtime.RunID{Subject: "root", Generation: 1}) || len(root.Members) != 1 || root.Members[0] != root.Run || root.Limits.MaxInvocations != int64(c.I) || root.Limits.MaxCreatedRuns != int64(c.R) || root.Limits.MaxDepth != int64(c.Depth) || root.Limits.MaxEffects != 0 || root.Limits.MaxInFlight != 1 {
		return fail("root/profile differs from declared C_inv")
	}
	if s.Roots[0].Limits != root.Limits {
		return fail("root completion limits mismatch")
	}
	programs := map[string]executor.Program{}
	runs := map[runtime.RunID]evidenceRun{}

	for _, r := range s.Tables["programs"] {
		p, e := readEvidence[executor.Program](r, "bytes")
		if e != nil {
			return out, e
		}
		id, _ := r["id"].(string)
		raw, _ := r["bytes"].(string)
		if fmt.Sprintf("%x", shaBytes([]byte(raw))) != id {
			return fail("program bytes/identity mismatch")
		}
		programs[id] = p
		expected := false
		for _, original := range d.Provision.Programs {
			if original.ID == id && string(original.Bytes) == raw {
				expected = true
			}
		}
		if !expected {
			return fail("program is not an original supplied deployment artifact")
		}
	}
	sourceSubjects := map[string]runtime.Subject{}
	for _, r := range s.Tables["subjects"] {
		g, e := evidenceNumber(r["generation"])
		if e != nil {
			return out, e
		}
		rev, e := evidenceNumber(r["revision"])
		if e != nil {
			return out, e
		}
		ret, e := evidenceNumber(r["retired"])
		if e != nil {
			return out, e
		}
		if g != 1 || rev != 1 || ret != 0 {
			return fail("unstable generation/revision/retirement")
		}
		id, ok := r["id"].(string)
		if !ok || id == "" {
			return fail("source subject identity missing")
		}
		sourceSubjects[id] = runtime.Subject{ID: id, Generation: g, Revision: rev, Retired: ret != 0}
	}
	for _, r := range s.Tables["runs"] {
		id, e := evidenceID(r)
		if e != nil {
			return out, e
		}
		initial, e := readEvidence[runtime.Projection](r, "initial")
		if e != nil {
			return out, e
		}
		current, e := readEvidence[runtime.Projection](r, "projection")
		if e != nil {
			return out, e
		}
		seq, e := evidenceNumber(r["sequence"])
		if e != nil {
			return out, e
		}
		pid, _ := r["program_id"].(string)
		runs[id] = evidenceRun{id, pid, initial, current, seq}

	}
	if len(s.Tables["subjects"]) != len(runs) {
		return fail("subject/run coverage")
	}
	initial := runs[root.Run].Initial
	if len(initial.Data) != 3 {
		return fail("parent initial fields")
	}
	q, e := invocationJoinRequest(map[string]schema.Value{"id": initial.Data["request_id"], "value": initial.Data["request_value"], "mode": initial.Data["request_mode"]}, true)
	if e != nil || q != c.Request {
		return fail("original parent inputs differ from checker config")
	}
	histories := map[string]evidenceHistory{}
	historyRows := map[string]map[string]any{}
	for _, r := range s.Tables["history"] {
		id, e := evidenceID(r)
		if e != nil {
			return out, e
		}
		seq, e := evidenceNumber(r["sequence"])
		if e != nil {
			return out, e
		}
		p, e := readEvidence[runtime.Projection](r, "successor")
		if e != nil {
			return out, e
		}
		ins, e := readEvidence[[]intent](r, "intents")
		if e != nil {
			return out, e
		}
		sel, _ := r["selected_observation"].(string)
		key := evidenceKey(id, seq)
		histories[key] = evidenceHistory{id, seq, sel, p, ins}
		historyRows[key] = r
	}
	// Establish the inverse link before any normalization: every retained
	// intent must be exactly one full original emitting-history intent. Neither
	// a future admission nor mutually agreeing altered authorization rows can
	// stand in for the commitment's actual source authority.
	commitments := map[string]executor.Intent{}
	commitmentRevisions := map[string]int64{}
	for key, h := range histories {
		row := historyRows[key]
		revision, e := evidenceNumber(row["source_revision"])
		if e != nil {
			return out, e
		}
		source, ok := sourceSubjects[h.Run.Subject]
		if !ok || source.Generation != h.Run.Generation || source.Revision != revision {
			return fail("history revision differs from original source subject")
		}
		full, e := readEvidence[[]executor.Intent](row, "intents")
		if e != nil {
			return out, e
		}
		for _, in := range full {
			if _, exists := commitments[in.ID]; exists {
				return fail("intent has multiple original commitments")
			}
			if in.ID == "" || in.Run != h.Run || in.Sequence != h.Sequence {
				return fail("emitted intent original source/sequence mismatch")
			}
			commitments[in.ID] = in
			commitmentRevisions[in.ID] = revision
		}
	}
	intentRevisions := map[string]int64{}
	intents := map[string]executor.Intent{}
	intentRows := map[string]map[string]any{}
	for _, r := range s.Tables["intents"] {
		in, e := readEvidence[executor.Intent](r, "descriptor")
		if e != nil {
			return out, e
		}
		original, ok := commitments[in.ID]
		generation, e := evidenceNumber(r["generation"])
		if e != nil {
			return out, e
		}
		sequence, e := evidenceNumber(r["sequence"])
		if e != nil {
			return out, e
		}
		ordinal, e := evidenceNumber(r["ordinal"])
		if e != nil {
			return out, e
		}
		revision, e := evidenceNumber(r["source_revision"])
		if e != nil {
			return out, e
		}
		if !ok || !reflect.DeepEqual(original, in) || r["id"] != in.ID || r["subject"] != in.Run.Subject || generation != in.Run.Generation || sequence != in.Sequence || ordinal != int64(in.Ordinal) || revision != commitmentRevisions[in.ID] {
			return fail("intent lacks exact original commitment/source metadata")
		}
		intentRevisions[in.ID] = revision
		intents[in.ID] = in
		intentRows[in.ID] = r
	}
	if len(commitments) != len(intents) {
		return fail("intent/commitment bijection incomplete")
	}
	observations := map[string]executor.Observation{}
	observationRows := map[string]map[string]any{}
	for _, r := range s.Tables["observations"] {
		o, e := readEvidence[executor.Observation](r, "descriptor")
		if e != nil {
			return out, e
		}
		observations[o.ID] = o
		observationRows[o.ID] = r
	}
	admissions := map[string]runtime.InvocationAdmission{}
	bindings := map[string]invocationJoinBinding{}
	memberOp := map[runtime.RunID]string{}
	for _, r := range s.Tables["composition_admission_decisions"] {
		a, e := readEvidence[runtime.InvocationAdmission](r, "descriptor")
		if e != nil {
			return out, e
		}
		revision, ok := intentRevisions[a.Invocation.OperationID]
		if !ok || a.Invocation.SourceRevision != revision {
			return fail("admission differs from original commitment source revision")
		}
		admissions[a.Invocation.OperationID] = a
		for _, m := range a.Members {
			memberOp[m] = a.Invocation.OperationID
		}
	}
	for _, r := range s.Tables["composition_bindings"] {
		b, e := readEvidence[invocationJoinBinding](r, "descriptor")
		if e != nil {
			return out, e
		}
		bindings[b.OperationID] = b
	}
	templates := map[string]executor.TemplateDescriptor{}
	for _, r := range s.Tables["composition_templates"] {
		v, e := readEvidence[executor.TemplateDescriptor](r, "descriptor")
		if e != nil {
			return out, e
		}
		templates[v.ID] = v
		found := false
		for _, original := range d.Provision.Composition.Templates {
			if reflect.DeepEqual(v, original) {
				found = true
			}
		}
		if !found {
			return fail("template differs from original deployment")
		}
	}
	for id, run := range runs {
		if id == root.Run {
			if len(d.Provision.Runs) != 1 || !reflect.DeepEqual(run.Initial, d.Provision.Runs[0].Projection) || run.Program != d.Outer.ID() {
				return fail("outer initial/program differs")
			}
			continue
		}
		b := bindings[memberOp[id]]
		template := templates[b.TemplateID]
		member, ok := template.Members[template.EntryMember]
		if !ok || b.Members[template.EntryMember] != id || member.ProgramID != run.Program || !reflect.DeepEqual(member.Initial, run.Initial) {
			return fail("dynamic initial/program differs from original template")
		}
	}
	requests := map[string]executor.Observation{}
	for _, r := range s.Tables["composition_requests"] {
		o, e := readEvidence[executor.Observation](r, "descriptor")
		if e != nil {
			return out, e
		}
		if r["observation_id"] != o.ID || r["invocation_id"] != admissions[o.Operation].Invocation.ID || !reflect.DeepEqual(o, observations[o.ID]) {
			return fail("request original row/observation linkage")
		}
		if _, ok := requests[o.Operation]; ok {
			return fail("duplicate original request")
		}
		requests[o.Operation] = o
	}
	attempts := map[string]executor.Authorization{}
	attemptRows := map[string]map[string]any{}
	for _, r := range s.Tables["attempts"] {
		a, e := readEvidence[executor.Authorization](r, "result")
		if e != nil {
			return out, e
		}
		var req struct {
			Intent   executor.Intent
			Revision int64
		}
		req, e = readEvidence[struct {
			Intent   executor.Intent
			Revision int64
		}](r, "descriptor")
		if e != nil {
			return out, e
		}
		in, ok := intents[a.IntentID]
		rev := intentRevisions[a.IntentID]
		source := sourceSubjects[in.Run.Subject]
		if !ok || r["id"] != a.ID || r["intent_id"] != in.ID || !reflect.DeepEqual(req.Intent, in) || req.Revision != rev || a.Source.Run != in.Run || a.Source.Revision != rev || a.Source.Revision != source.Revision || a.Source.Run.Generation != source.Generation || a.Source.Epoch <= 0 {
			return fail("authorization original intent/source/revision linkage")
		}
		attempts[in.ID] = a
		attemptRows[in.ID] = r
	}
	replies := map[string]invocationJoinReply{}
	for _, r := range s.Tables["composition_replies"] {
		v, e := readEvidence[invocationJoinReply](r, "descriptor")
		if e != nil {
			return out, e
		}
		if !reflect.DeepEqual(v.Intent, intents[v.Intent.ID]) || !reflect.DeepEqual(v.Authorization, attempts[v.Intent.ID]) || r["operation_id"] != v.Intent.ID {
			return fail("reply original authorization/intent linkage")
		}
		replies[v.Intent.ID] = v
	}
	publications := map[string]runtime.InvocationOutcome{}
	publicationRows := map[string]map[string]any{}
	for _, r := range s.Tables["composition_publications"] {
		p, e := readEvidence[runtime.InvocationOutcome](r, "descriptor")
		if e != nil {
			return out, e
		}
		publications[p.OperationID] = p
		publicationRows[p.OperationID] = r
	}
	deliveries := map[string]string{}
	for _, r := range s.Tables["composition_outcome_deliveries"] {
		op, _ := r["operation_id"].(string)
		id, _ := r["observation_id"].(string)
		if deliveries[op] != "" {
			return fail("duplicate outcome delivery")
		}
		deliveries[op] = id
	}
	facts := []invocationJoinFact{}
	for _, r := range s.Tables["composition_facts"] {
		f, e := readEvidence[invocationJoinFact](r, "descriptor")
		if e != nil {
			return out, e
		}
		order, e := evidenceNumber(r["decision_order"])
		if e != nil {
			return out, e
		}
		if f.Version != "alan/composition-fact/v1" || f.Order != order || f.ID != r["id"] || f.Kind != r["kind"] || !reflect.DeepEqual(f.RootIDs, []string{root.ID}) {
			return fail("fact original identity/order/root")
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Order < facts[j].Order })
	rename := map[string]string{}
	metadata := map[string]model.Observation{}
	sequences := map[runtime.RunID]int64{}
	states := map[runtime.RunID]runtime.Projection{}
	for id, r := range runs {
		states[id] = r.Initial
	}
	for id, o := range observations {
		if o.Kind == "message" {
			if o.Provenance != "runtime-invocation-request" || !reflect.DeepEqual(o, requests[o.Operation]) {
				return fail("orphan/unsupported request observation %s", id)
			}
		} else if o.Kind != "invocation" {
			return fail("observation outside C_inv profile")
		}
	}
	authorized := map[string]bool{}
	seenAdmissions := map[string]bool{}
	seenReplies := map[string]bool{}
	seenPublications := map[string]bool{}
	seenHistories := map[string]bool{}
	executedIntents := map[string]bool{}
	var currentDiagnostic model.JoinDiagnostic
	emit := func(op, kind string) model.Observation { v := metadata[op]; v.Kind = kind; return v }
	currentOrder := int64(0)
	appendEvent := func(v model.Observation) {
		event := model.CallableJoinEvent{JoinEvent: model.JoinEvent{Fact: v}}
		if v.Operation == "j" && (v.Kind == "reply_commit" || v.Kind == "reply_admit" || v.Kind == "publish" || v.Kind == "outcome_deliver" || v.Kind == "outcome_consume") {
			event.Diagnostic = currentDiagnostic
		}
		out.Events = append(out.Events, event)
		if orders != nil {
			*orders = append(*orders, currentOrder)
		}
	}
	parentMeta := func(kind, status string) model.Observation {
		return model.Observation{Kind: kind, Operation: "root", Role: "caller", Request: c.Request, Status: status}
	}
	boundReply := func(in executor.Intent) bool {
		op := memberOp[in.Run]
		b, ok := bindings[op]
		if !ok {
			return false
		}
		t := templates[b.TemplateID]
		p := programs[b.Programs[t.EntryMember]]
		if p.Author == nil {
			return false
		}
		port, ok := p.Author.Ports[t.EntryPort]
		return ok && port.Send.Record.Validate(in.Payload) == nil && b.Protocol == "alan/composition/v1" && in.Kind == "message" && b.Members[t.EntryMember] == in.Run && in.Target == b.Caller.Subject && in.Generation == b.Caller.Generation && in.Obligation == port.Obligation
	}
	for index, f := range facts {
		currentOrder = f.Order
		if f.Order != int64(index+1) {
			return fail("noncontiguous/duplicate fact order")
		}
		switch f.Kind {
		case "execution":
			key := evidenceKey(f.Run, f.Sequence)
			h, ok := histories[key]
			row := historyRows[key]
			if !ok || seenHistories[key] || f.Sequence != sequences[f.Run]+1 {
				return fail("execution original history order")
			}
			seenHistories[key] = true
			epoch, e := evidenceNumber(row["epoch"])
			if e != nil {
				return out, e
			}
			rev, e := evidenceNumber(row["source_revision"])
			if e != nil {
				return out, e
			}
			source := struct {
				Run                       runtime.RunID
				Sequence, Epoch, Revision int64
				Selected                  string
				Successor, Intents        json.RawMessage
			}{f.Run, f.Sequence, epoch, rev, h.Selected, json.RawMessage(row["successor"].(string)), json.RawMessage(row["intents"].(string))}
			if f.RecordDigest != invocationJoinDigest("alan/history-row/v1", source) {
				return fail("execution source digest")
			}
			pre := states[f.Run]
			program := programs[runs[f.Run].Program]
			if pre.Status != "open" || pre.PC < 0 || pre.PC >= len(program.Code) {
				return fail("execution predecessor not open original instruction")
			}
			in := program.Code[pre.PC]
			binding := bindings[memberOp[f.Run]]
			template := templates[binding.TemplateID]
			if e := callableStep(program, pre, h.Successor, in, h.Selected, observations[h.Selected], h.Intents, binding, template); e != nil {
				return fail("instruction %s sequence%d PC%d: %v", f.Run.Subject, f.Sequence, pre.PC, e)
			}
			step := reasonStep{Run: f.Run, Program: runs[f.Run].Program, Sequence: f.Sequence, Order: f.Order, PC: pre.PC, NextPC: h.Successor.PC}
			out.Steps = append(out.Steps, callableInstruction{Step: step, Instruction: in, Pre: pre, Post: h.Successor, Selected: h.Selected})

			for _, original := range h.Intents {
				if executedIntents[original.ID] {
					return fail("intent repeated across original execution facts")
				}
				executedIntents[original.ID] = true
				actual := intents[original.ID]
				if actual.Kind == "invocation" {
					role, parent, label, handle := "", "", "", ""
					switch {
					case f.Run == root.Run && in.Dst == "join_operation":
						role, parent, label, handle = "join", "root", "j", "join_operation"
					case rename[memberOp[f.Run]] == "j" && in.Dst == "candidate_operation":
						role, parent, label, handle = "candidate", "j", "e", "candidate_operation"
					case rename[memberOp[f.Run]] == "j" && in.Dst == "s_operation":
						role, parent, label, handle = "s", "j", "s", "s_operation"
					case rename[memberOp[f.Run]] == "e" && in.Dst == "helper_operation":
						role, parent, label, handle = "helper", "e", "h", "helper_operation"
					default:
						return fail("unmapped invocation owner/register")
					}

					if out.Handles[label] != "" || h.Successor.Data[handle].Kind != "string" || h.Successor.Data[handle].Text != actual.ID {
						return fail("original handle not its emitted intent")
					}
					request, e := invocationJoinRequest(actual.Payload, label != "s")
					if e != nil {
						return out, e
					}
					want := c.Request
					if label == "h" {
						want.Mode = "answer"
					}
					if label == "s" {
						want.Mode = ""
					}
					if request != want {
						return fail("original request changed for %s", label)
					}
					out.Handles[label] = actual.ID
					rename[actual.ID] = label
					metadata[actual.ID] = model.Observation{Operation: label, Parent: parent, Role: role, Request: request}
					appendEvent(emit(actual.ID, "invoke_commit"))
				} else if actual.Kind == "message" {
					op := memberOp[actual.Run]
					if rename[op] == "" || !boundReply(actual) {
						return fail("message lacks original reply binding")
					}
					if in.Effect == nil || in.Effect.Kind != "message" {
						return fail("reply is not original Send")
					}
					v := emit(op, "reply_commit")
					v.Status = "committed"
					v.ID = int(actual.Payload["id"].Number)
					v.Value = int(actual.Payload["value"].Number)
					if rename[op] == "j" {
						currentDiagnostic, e = callableDiagnostic(actual.Payload)
						if e != nil {
							return out, e
						}
						v.ID, v.Value = 0, 0
						out.Report = &callableReportEvidence{Step: step, Intent: actual, Diagnostic: currentDiagnostic}
					} else if len(actual.Payload) != 2 || actual.Payload["id"].Kind != "int" || actual.Payload["value"].Kind != "int" {
						return fail("reply fields")
					}
					appendEvent(v)
				} else {
					return fail("unexpected effect %s", actual.Kind)
				}
			}
			if h.Selected != "" {
				o := observations[h.Selected]
				if o.Provenance == "runtime-invocation-request" {
					a := admissions[o.Operation]
					b := bindings[o.Operation]
					t := templates[b.TemplateID]
					port := program.Author.Ports[t.EntryPort]
					if !a.Accepted || !reflect.DeepEqual(requests[o.Operation], o) || b.Members[t.EntryMember] != f.Run || o.Source != b.Caller.Subject || o.SourceGeneration != b.Caller.Generation || o.Obligation != port.Obligation || !sameValues(o.Payload, a.Invocation.Inputs) || in.Op != "receive" || in.Match == nil || in.Match.Kind != "message" || in.Match.Provenance != "ordinary-message" {
						return fail("selected original request provenance/binding")
					}
					src, e := invocationJoinOperand(in.Match.Source, pre)
					if e != nil {
						return out, e
					}
					ob, e := invocationJoinOperand(in.Match.Obligation, pre)
					if e != nil {
						return out, e
					}
					g, e := invocationJoinOperand(in.Match.Generation, pre)
					if e != nil {
						return out, e
					}
					operation, e := invocationJoinOperand(in.Match.Operation, pre)
					if e != nil {
						return out, e
					}
					if src.Text != t.CallerPeer || ob.Text != o.Obligation || g.Number != 1 || operation.Text != "" {
						return fail("original request receive guard")
					}
					for field, dst := range in.Bind {
						if value, present := o.Payload[field]; present && !reflect.DeepEqual(h.Successor.Data[dst], value) {
							return fail("request selected binding payload")
						}
					}
					appendEvent(emit(o.Operation, "request_consume"))
				} else if o.Kind == "invocation" {
					v := emit(o.Operation, "outcome_consume")
					if rename[o.Operation] == "" {
						return fail("selection of unknown original handle")
					}
					if e := callableEnvelope(&v, &currentDiagnostic, o.Payload); e != nil {
						return out, e
					}
					out.Consumptions[rename[o.Operation]] = reasonConsumption{Step: step, Observation: o, Bind: in.Bind}
					appendEvent(v)
				} else {
					return fail("unexpected selected observation")
				}
			}
			role := rename[memberOp[f.Run]]
			if role == "j" && in.Op == "set" && in.Dst == "classification" {
				tag := h.Successor.Data["classification"].Text
				if out.Classification != nil || out.Consumptions["s"].Observation.ID == "" || out.Consumptions["e"].Observation.ID == "" {
					return fail("classification lacks both original consumes")
				}
				out.Classification = &reasonClassificationEvidence{Step: step, Tag: tag, Consumed: map[string]reasonConsumption{"e": out.Consumptions["e"], "s": out.Consumptions["s"]}}
				status := "failed"
				if tag == "answer" {
					status = "succeeded"
				}
				out.Classification.Decision = status
				v := emit(out.Handles["j"], "join_classify")
				out.Events = append(out.Events, model.CallableJoinEvent{JoinEvent: model.JoinEvent{Fact: v}, Classification: tag})
				out.Orders = append(out.Orders, f.Order)
				v.Kind, v.Status = "join_decide", status
				appendEvent(v)
			}
			if f.Run == root.Run && in.Op == "set" && in.Dst == "decision" {
				if out.Decision != nil || out.Consumptions["j"].Observation.ID == "" {
					return fail("outer decision lacks original J consume")
				}
				tag := h.Successor.Data["decision"].Text
				status := "failed"
				if tag == "answer" {
					status = "succeeded"
				}
				out.Decision = &reasonClassificationEvidence{Step: step, Tag: tag, Decision: status, Consumed: map[string]reasonConsumption{"j": out.Consumptions["j"]}}
				out.Events = append(out.Events, model.CallableJoinEvent{JoinEvent: model.JoinEvent{Fact: parentMeta("outer_decide", status)}, Decision: tag})
				out.Orders = append(out.Orders, f.Order)
			}
			if in.Op == "end" {
				if role == "j" {
					out.JoinEnd = &step
				}
				if f.Run == root.Run {
					out.OuterEnd = &step
					appendEvent(parentMeta("outer_terminal", h.Successor.Status))
				}
			}
			// Only byte-identified original pending paths can emit this marker.
			pendingPC := -1
			if runs[f.Run].Program == "74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7" && pre.PC == 9 {
				pendingPC = 11
			}
			if runs[f.Run].Program == "ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67" && pre.PC == 10 {
				pendingPC = 12
			}
			if role == "e" && h.Successor.PC == pendingPC {
				out.Pending = &step
				for _, kind := range []string{"private_step", "pending_receive"} {
					appendEvent(emit(out.Handles["e"], kind))
				}
			}

			states[f.Run] = h.Successor
			sequences[f.Run] = f.Sequence
		case "authorization":
			a, ok := attempts[f.OperationID]
			original := intents[f.OperationID]
			if !ok || authorized[f.OperationID] || a.Source.Sequence != sequences[original.Run] || a.Source.Run != f.Run || original.Sequence != f.Sequence || f.RecordDigest != invocationJoinDigest("alan/authorization-row/v1", json.RawMessage(attemptRows[f.OperationID]["result"].(string))) {
				return fail("first authorization ordered source snapshot")
			}
			authorized[f.OperationID] = true
			op, kind := original.ID, "invoke_authorize"
			if original.Kind == "message" {
				op, kind = memberOp[original.Run], "reply_authorize"
			}
			if rename[op] == "" {
				return fail("authorization lacks original operation")
			}
			v := emit(op, kind)
			v.Status = "refused"
			if a.Allowed {
				v.Status = "allowed"
			}
			v.Reason = a.Reason
			source := sourceSubjects[original.Run.Subject]
			revisionMatches := a.Source.Revision == intentRevisions[original.ID] && a.Source.Revision == source.Revision
			generationMatches := a.Source.Run.Generation == source.Generation
			if !revisionMatches || !generationMatches || !a.Allowed || a.Reason != "authorized" || source.Retired || states[original.Run].Status != "open" && !boundReply(original) {
				return fail("authorization violates stable original source/bound-reply rule")
			}
			out.Events = append(out.Events, model.CallableJoinEvent{JoinEvent: model.JoinEvent{Fact: v, SourceOpen: states[original.Run].Status == "open", SourceRetired: source.Retired, RevisionMatches: revisionMatches, GenerationMatches: generationMatches, ReplyBound: kind == "reply_authorize" && boundReply(original)}})
			if orders != nil {
				*orders = append(*orders, currentOrder)
			}
		case "admission":
			a, ok := admissions[f.OperationID]
			if !ok || rename[f.OperationID] == "" || seenAdmissions[f.OperationID] {
				return fail("admission unknown or repeated original")
			}
			seenAdmissions[f.OperationID] = true
			row := communicationRow(s, "composition_admission_decisions", "operation_id", f.OperationID)
			if f.RecordDigest != invocationJoinDigest("alan/admission-row/v1", json.RawMessage(row["descriptor"].(string))) {
				return fail("admission source digest")
			}
			expectedDepth := int64(2)
			if rename[f.OperationID] == "j" {
				expectedDepth = 1
			}
			if rename[f.OperationID] == "h" {
				expectedDepth = 3
			}
			if a.Invocation.Depth != expectedDepth {
				return fail("original invocation depth mismatch")
			}
			v := emit(f.OperationID, "admit")
			v.Status = "rejected"
			v.Reason = a.Reason
			if a.Accepted {
				b := bindings[f.OperationID]
				t := templates[b.TemplateID]
				req := requests[f.OperationID]
				entryProgram := programs[b.Programs[t.EntryMember]]
				if entryProgram.Author == nil {
					return fail("entry request lacks original metadata")
				}
				port, exists := entryProgram.Author.Ports[t.EntryPort]
				if !exists || req.Obligation != port.Obligation || port.Receive.Record.Validate(req.Payload) != nil {
					return fail("unconsumed original request schema/obligation")
				}
				if len(t.Members) != 1 || b.Protocol != "alan/composition/v1" || b.Depth != a.Invocation.Depth || b.Members[t.EntryMember] != req.Run || req.Kind != "message" || req.Provenance != "runtime-invocation-request" || req.Source != b.Caller.Subject || req.SourceGeneration != b.Caller.Generation || req.Operation != f.OperationID || !sameValues(req.Payload, intents[f.OperationID].Payload) {
					return fail("admission original request/binding")
				}
				charge := communicationRow(s, "composition_charges", "operation_id", f.OperationID)
				var ch struct {
					OperationID, RootID, Kind                        string
					Invocations, CreatedRuns, Effects, DecisionOrder int64
				}
				ch, e = readEvidence[struct {
					OperationID, RootID, Kind                        string
					Invocations, CreatedRuns, Effects, DecisionOrder int64
				}](charge, "descriptor")
				if e != nil {
					return out, e
				}
				if ch.OperationID != f.OperationID || ch.RootID != root.ID || ch.Kind != "invocation" || ch.DecisionOrder != f.Order || ch.Invocations != 1 || ch.CreatedRuns != 1 || ch.Effects != 0 {
					return fail("admission actual charge")
				}
				v.Status = "accepted"
				v.Reason = ""
				v.I = 1
				v.R = 1
			} else if communicationRow(s, "composition_charges", "operation_id", f.OperationID) != nil {
				return fail("refused admission charged")
			}
			appendEvent(v)
		case "reply":
			r, ok := replies[f.OperationID]
			op := memberOp[f.Run]
			if !ok || rename[op] == "" || seenReplies[f.OperationID] {
				return fail("reply fact unknown/repeated")
			}
			seenReplies[f.OperationID] = true
			row := communicationRow(s, "composition_replies", "operation_id", f.OperationID)
			if f.RecordDigest != invocationJoinDigest("alan/reply-row/v1", json.RawMessage(row["descriptor"].(string))) || r.InvocationID != admissions[op].Invocation.ID || !boundReply(r.Intent) {
				return fail("reply fact original binding/digest")
			}
			v := emit(op, "reply_admit")
			v.Status = "rejected"
			v.Reason = r.Reason
			v.Present = r.Accepted
			if r.Accepted {
				v.Status = "accepted"
				v.Reason = ""
			}
			v.ID = int(r.Intent.Payload["id"].Number)
			v.Value = int(r.Intent.Payload["value"].Number)
			if rename[op] == "j" {
				currentDiagnostic, e = callableDiagnostic(r.Intent.Payload)
				if e != nil {
					return out, e
				}
				v.ID, v.Value = 0, 0
			}
			appendEvent(v)
		case "freeze":
			row := communicationRow(s, "composition_freezes", "scope_id", f.OperationID)
			if row == nil || f.RecordDigest != invocationJoinDigest("alan/freeze-row/v1", json.RawMessage(row["descriptor"].(string))) {
				return fail("freeze source digest")
			}
			if f.Run != root.Run {
				op := memberOp[f.Run]
				if rename[op] == "" {
					return fail("freeze original operation")
				}
				v := emit(op, "terminal_freeze")
				v.Status = states[f.Run].Status
				appendEvent(v)
			}
		case "outcome":
			if f.OperationID == "root:"+root.ID {
				row := communicationRow(s, "composition_root_publications", "scope_id", f.OperationID)
				if row == nil || f.RecordDigest != invocationJoinDigest("alan/outcome-row/v1", json.RawMessage(row["descriptor"].(string))) {
					return fail("root outcome digest")
				}
				p := s.Roots[0].Outcome
				if p == nil {
					return fail("root outcome absent")
				}
				v := parentMeta("root_publish", p.Status)
				v.Reason = p.Reason
				appendEvent(v)
				continue
			}
			p, ok := publications[f.OperationID]
			if !ok || rename[f.OperationID] == "" || seenPublications[f.OperationID] || f.RecordDigest != invocationJoinDigest("alan/outcome-row/v1", json.RawMessage(publicationRows[f.OperationID]["descriptor"].(string))) {
				return fail("publication original fact/digest")
			}
			seenPublications[f.OperationID] = true
			o, ok := observations[deliveries[f.OperationID]]
			if !ok || o.Operation != f.OperationID || o.Kind != "invocation" || o.Run != p.Caller {
				return fail("atomic publication lacks original outcome delivery")
			}
			v := emit(f.OperationID, "publish")
			if e := callableEnvelope(&v, &currentDiagnostic, o.Payload); e != nil {
				return out, e
			}
			appendEvent(v)
			v.Kind = "outcome_deliver"
			appendEvent(v)
		default:
			return fail("fact outside selected profile: %s", f.Kind)
		}
	}
	accepted := 0
	for _, a := range admissions {
		if a.Accepted {
			accepted++
		}
	}
	charges := s.Roots[0].Charges
	if len(s.Tables["composition_charges"]) != accepted || charges.Invocations != int64(accepted) || charges.CreatedRuns != int64(accepted) || charges.Effects != 0 || charges.RootID != root.ID {
		return fail("admission/charge cardinality or root completion total mismatch")
	}
	for _, row := range s.Tables["composition_charges"] {
		op, _ := row["operation_id"].(string)
		if !admissions[op].Accepted || row["kind"] != "invocation" || row["root_id"] != root.ID {
			return fail("unlinked or refused admission charge")
		}
	}
	if len(executedIntents) != len(intents) || len(seenHistories) != len(histories) || len(authorized) != len(attempts) || len(seenAdmissions) != len(admissions) || len(seenReplies) != len(replies) || len(seenPublications) != len(publications) {
		return fail("unmapped raw evidence rows")
	}
	for id, r := range runs {
		if sequences[id] != r.Sequence || !reflect.DeepEqual(states[id], r.Current) {
			return fail("final projection differs from ordered original history")
		}
		if id != root.Run && rename[memberOp[id]] == "" {
			return fail("unknown dynamic run")
		}
		registerRoles := map[string]string{}
		if id == root.Run {
			registerRoles["j"] = "join_operation"
		}
		if rename[memberOp[id]] == "j" {
			registerRoles["e"] = "candidate_operation"
			registerRoles["s"] = "s_operation"
		}
		if rename[memberOp[id]] == "e" {
			registerRoles["h"] = "helper_operation"
		}
		for label, register := range registerRoles {
			if op := out.Handles[label]; op != "" && r.Current.Data[register].Text != op {
				return fail("original handle changed")
			}
		}

	}
	if e := callablePending(s, out, programs, runs, bindings, memberOp); e != nil {
		return fail("pending evidence: %v", e)
	}
	if e := callableAdmissions(s, intents, attempts, replies); e != nil {
		return out, e
	}
	out.RawValid = true
	out.Public = model.CheckCallableJoinPublic(c, out.Events)
	out.Truth = model.CheckCallableJoinPrefix(c, out.Events)
	out.Component = model.CheckCallableJoinComponent(c, out.Events)
	out.Contract = model.CheckCallableJoinContract(c, out.Events)
	out.Outer = model.CheckCallableJoinOuterGuarantee(c, out.Events)
	return out, nil
}
