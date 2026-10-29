//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/JakeFAU/alan/internal/executor"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type invocationJoinTrace struct {
	RawValid                     bool
	Events                       []model.JoinEvent
	Handles                      map[string]string
	Prefix, Component, Guarantee []model.Violation
}
type invocationJoinFact struct {
	Version, ID, Kind         string
	Order                     int64
	RootIDs                   []string
	Run                       runtime.RunID
	Sequence                  int64
	OperationID, RecordDigest string
}
type invocationJoinBinding struct {
	Protocol string
	evidenceBinding
	Depth int64
}
type invocationJoinReply struct {
	InvocationID, RootID string
	Intent               executor.Intent
	Authorization        executor.Authorization
	Accepted             bool
	Reason               string
}

func invocationJoinDigest(domain string, v any) string { return hash([]any{domain, v}) }
func invocationJoinOperand(o executor.Operand, p runtime.Projection) (schema.Value, error) {
	if o.Offset != nil {
		return schema.Value{}, fmt.Errorf("unexpected offset")
	}
	if o.Ref != "" {
		v, ok := p.Data[o.Ref]
		if !ok {
			return v, fmt.Errorf("absent original register %s", o.Ref)
		}
		return v, nil
	}
	if o.Literal == nil {
		return schema.Value{}, fmt.Errorf("missing original operand")
	}
	return *o.Literal, nil
}
func invocationJoinRequest(p map[string]schema.Value, mode bool) (model.Request, error) {
	want := 2
	if mode {
		want = 3
	}
	if len(p) != want || p["id"].Kind != "int" || p["value"].Kind != "int" || mode && p["mode"].Kind != "string" {
		return model.Request{}, fmt.Errorf("original request fields/kinds differ")
	}
	r := model.Request{ID: int(p["id"].Number), Value: int(p["value"].Number)}
	if mode {
		r.Mode = p["mode"].Text
	}
	return r, nil
}
func invocationJoinEnvelope(o *model.Observation, p map[string]schema.Value) error {
	if p["status"].Kind != "string" || p["reason"].Kind != "string" || p["response_present"].Kind != "bool" {
		return fmt.Errorf("outcome fields missing")
	}
	o.Status = p["status"].Text
	o.Reason = p["reason"].Text
	o.Present = p["response_present"].Flag
	if o.Present {
		if p["response.id"].Kind != "int" || p["response.value"].Kind != "int" {
			return fmt.Errorf("present outcome lacks payload")
		}
		o.ID = int(p["response.id"].Number)
		o.Value = int(p["response.value"].Number)
	}
	return nil
}

// Raw validation precedes every return that supplies normalized events. The pure
// checkers remain independent verdicts; a truthful wrong-id history is not corrupt.
func decodeInvocationJoin(s runtime.Inspection, c model.JoinConfig) (out invocationJoinTrace, err error) {
	out.Handles = map[string]string{}
	fail := func(f string, a ...any) (invocationJoinTrace, error) {
		return invocationJoinTrace{}, fmt.Errorf(f, a...)
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
	decided := false
	emit := func(op, kind string) model.Observation { v := metadata[op]; v.Kind = kind; return v }
	appendEvent := func(v model.Observation) { out.Events = append(out.Events, model.JoinEvent{Fact: v}) }
	parentMeta := func(kind, status string) model.Observation {
		return model.Observation{Kind: kind, Operation: "root", Role: "parent", Request: c.Request, Status: status}
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
			if f.Run == root.Run {
				if e := invocationJoinParentStep(pre, h.Successor, in, h.Selected, observations[h.Selected], h.Intents); e != nil {
					return out, e
				}
			}
			for _, original := range h.Intents {
				if executedIntents[original.ID] {
					return fail("intent repeated across original execution facts")
				}
				executedIntents[original.ID] = true
				actual := intents[original.ID]
				if actual.Kind == "invocation" {
					role, parent, label, handle := "candidate", "root", "e", "candidate_operation"
					if f.Run == root.Run {
						if pre.PC == 1 {
							role, label, handle = "s", "s", "s_operation"
						} else if pre.PC != 0 {
							return fail("unexpected root invocation")
						}
					} else {
						role, parent, label, handle = "helper", "e", "h", "helper_operation"
						if rename[memberOp[f.Run]] != "e" {
							return fail("helper outside candidate")
						}
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
					if len(actual.Payload) != 2 || actual.Payload["id"].Kind != "int" || actual.Payload["value"].Kind != "int" {
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
					if e := invocationJoinEnvelope(&v, o.Payload); e != nil {
						return out, e
					}
					appendEvent(v)
				} else {
					return fail("unexpected selected observation")
				}
			}
			if f.Run == root.Run && !decided && h.Successor.Status == "open" && program.Code[h.Successor.PC].Op == "end" {
				appendEvent(parentMeta("parent_decide", program.Code[h.Successor.PC].Outcome))
				decided = true
			}
			if f.Run == root.Run && in.Op == "end" {
				if !decided {
					return fail("parent End without actual resolved pure path")
				}
				appendEvent(parentMeta("parent_terminal", h.Successor.Status))
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
			out.Events = append(out.Events, model.JoinEvent{Fact: v, SourceOpen: states[original.Run].Status == "open", SourceRetired: source.Retired, RevisionMatches: revisionMatches, GenerationMatches: generationMatches, ReplyBound: kind == "reply_authorize" && boundReply(original)})
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
			expectedDepth := int64(1)
			if rename[f.OperationID] == "h" {
				expectedDepth = 2
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
			if e := invocationJoinEnvelope(&v, o.Payload); e != nil {
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
		if id == root.Run {
			for label, register := range map[string]string{"e": "candidate_operation", "s": "s_operation"} {
				if op := out.Handles[label]; op != "" && r.Current.Data[register].Text != op {
					return fail("original root handle changed")
				}
			}
		}
	}
	out.RawValid = true
	out.Prefix = model.CheckJoinPrefix(c, out.Events)
	out.Component = model.CheckJoinComponent(c, out.Events)
	out.Guarantee = model.CheckJoinGuarantee(c, out.Events)
	return out, nil
}

// Check the actual finite parent path independently of Reduce. This observes
// executed control, never chooses/commits a replacement projection or decision.
func invocationJoinParentStep(pre, post runtime.Projection, in executor.Instruction, selected string, o executor.Observation, intents []intent) error {
	want := schema.CloneValues(pre.Data)
	pc := pre.PC + 1
	status := "open"
	eval := func(v executor.Operand) (schema.Value, error) { return invocationJoinOperand(v, pre) }
	switch in.Op {
	case "emit":
		if len(intents) != 1 || in.Effect == nil || in.Effect.Kind != "invocation" {
			return fmt.Errorf("parent original invocation emit")
		}
		if in.Dst == "" {
			return fmt.Errorf("parent original handle absent")
		}
		want[in.Dst] = schema.Text(intents[0].ID)
		for k, operand := range in.Effect.Payload {
			value, e := eval(operand)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(intents[0].Payload[k], value) {
				return fmt.Errorf("parent original emitted payload differs")
			}
		}
		if len(intents[0].Payload) != len(in.Effect.Payload) {
			return fmt.Errorf("parent emitted extra payload")
		}
		target, e := eval(in.Effect.Target)
		if e != nil {
			return e
		}
		obligation, e := eval(in.Effect.Obligation)
		if e != nil {
			return e
		}
		generation, e := eval(in.Effect.Generation)
		if e != nil {
			return e
		}
		if intents[0].Target != target.Text || intents[0].Obligation != obligation.Text || intents[0].Generation != generation.Number || len(in.Updates) != 0 {
			return fmt.Errorf("parent original invocation endpoints")
		}
	case "receive":
		if selected == "" || o.Kind != "invocation" {
			return fmt.Errorf("parent receive lacks original outcome")
		}
		for field, dst := range in.Bind {
			if value, ok := o.Payload[field]; ok {
				want[dst] = value
			}
		}
	case "set":
		v, e := eval(in.A)
		if e != nil {
			return e
		}
		want[in.Dst] = v
	case "branch":
		a, e := eval(in.A)
		if e != nil {
			return e
		}
		b, e := eval(in.B)
		if e != nil {
			return e
		}
		pc = in.Otherwise
		if reflect.DeepEqual(a, b) {
			pc = in.Target
		}
	case "jump":
		pc = in.Target
	case "end":
		pc = pre.PC
		status = in.Outcome
	default:
		return fmt.Errorf("unmapped parent instruction %s", in.Op)
	}
	if in.Op != "emit" && len(intents) != 0 || in.Op != "receive" && selected != "" || post.PC != pc || post.Status != status || !sameValues(post.Data, want) {
		return fmt.Errorf("actual parent pure/guard path or payload mismatch at PC%d", pre.PC)
	}
	return nil
}

// Uniqueness is checked before maps can erase malformed duplicate source rows.
func invocationJoinUniqueRows(s runtime.Inspection) error {
	keys := map[string][]string{
		"programs": {"id"}, "subjects": {"id"}, "runs": {"subject", "generation"}, "history": {"subject", "generation", "sequence"}, "intents": {"id"}, "attempts": {"intent_id"}, "observations": {"id"}, "composition_templates": {"id"}, "composition_bindings": {"id"}, "composition_invocations": {"operation_id"}, "composition_members": {"subject", "generation"}, "composition_admission_decisions": {"operation_id"}, "composition_charges": {"operation_id", "kind"}, "composition_requests": {"invocation_id"}, "composition_replies": {"operation_id"}, "composition_publications": {"operation_id"}, "composition_outcome_deliveries": {"operation_id"}, "composition_freezes": {"scope_id"}, "composition_root_publications": {"scope_id"}, "composition_facts": {"id"},
	}
	for table, columns := range keys {
		seen := map[string]bool{}
		for _, row := range s.Tables[table] {
			parts := []any{}
			for _, key := range columns {
				value, ok := row[key]
				if !ok || value == nil {
					return fmt.Errorf("missing %s.%s", table, key)
				}
				parts = append(parts, value)
			}
			identity := hash(parts)
			if seen[identity] {
				return fmt.Errorf("duplicate %s original key", table)
			}
			seen[identity] = true
		}
	}
	for _, table := range []string{"external_attempts", "external_receipts", "adapter_attempts", "adapter_results", "composition_reservations", "composition_closures"} {
		if len(s.Tables[table]) != 0 {
			return fmt.Errorf("outside E0/stable profile: %s", table)
		}
	}
	return nil
}
