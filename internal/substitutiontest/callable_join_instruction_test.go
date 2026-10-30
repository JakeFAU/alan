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
	"strings"
)

type callablePCMapping struct {
	Program, Role, Mapping, Position string
	PC                               int
	Instruction                      executor.Instruction
}

func callableStaticMapping(d callableapp.Deployment) ([]callablePCMapping, error) {
	result := []callablePCMapping{}
	for _, artifact := range d.Provision.Programs {
		var p executor.Program
		if e := json.Unmarshal(artifact.Bytes, &p); e != nil {
			return nil, e
		}
		role := "original-component"
		if artifact.ID == d.JoinArtifact.ID() {
			role = "J"
		}
		if artifact.ID == d.Outer.ID() {
			role = "O"
		}
		for pc, in := range p.Code {
			mapping := ""
			switch in.Op {
			case "set":
				mapping = "checked private stutter: exact register initialization/update"
				if in.Dst == "classification" {
					mapping = "J classification plus immutable-tag ghost decision, same commit; no extra checkpoint"
				}
				if in.Dst == "decision" {
					mapping = "O observed public decision; separate later End"
				}
			case "branch", "jump":
				mapping = "checked private control stutter: exact operand/edge; no End inferred"
			case "append":
				mapping = "checked private list-construction stutter: exact bounded append"
			case "foreach", "next", "loop":
				mapping = "checked private cursor stutter: snapshot/binding/index/removal retained atomically"
			case "receive":
				mapping = "original request/outcome selection, binding and consumption in one commit"
			case "emit":
				if in.Effect == nil {
					return nil, fmt.Errorf("missing mapped effect")
				}
				if in.Effect.Kind == "invocation" {
					mapping = "original invocation intent and handle write in one commit"
				} else if in.Effect.Kind == "message" {
					mapping = "original reply Send commitment; authorization/admission separate"
				} else {
					return nil, fmt.Errorf("unmapped effect")
				}
			case "end":
				mapping = "local End and scope freeze in one owner transaction; publication separate"
			default:
				return nil, fmt.Errorf("unmapped static instruction %s", in.Op)
			}
			if pc >= len(p.Author.Positions) {
				return nil, fmt.Errorf("missing original source position")
			}
			result = append(result, callablePCMapping{Program: artifact.ID, Role: role, Mapping: mapping, Position: p.Author.Positions[pc], PC: pc, Instruction: in})
		}
	}
	return result, nil
}

// callableStep checks retained instructions, never calls Reduce or computes a
// projection to commit. It covers the complete fixed artifact instruction set.
func callableStep(p executor.Program, pre, post runtime.Projection, in executor.Instruction, selected string, o executor.Observation, intents []intent, binding invocationJoinBinding, template executor.TemplateDescriptor) error {
	data := schema.CloneValues(pre.Data)
	cursors := map[int]executor.Cursor{}
	for k, v := range pre.Cursors {
		cursors[k] = executor.Cursor{Snapshot: v.Snapshot.Clone(), Index: v.Index}
	}
	pc, status := pre.PC+1, "open"
	eval := func(v executor.Operand) (schema.Value, error) { return invocationJoinOperand(v, pre) }
	endpoint := func(v executor.Operand) (schema.Value, error) {
		x, e := eval(v)
		if e != nil {
			return x, e
		}
		if x.Text == template.CallerPeer && binding.OperationID != "" {
			return schema.Text(binding.Caller.Subject), nil
		}
		return x, nil
	}
	if in.Op != "emit" && len(intents) != 0 || in.Op != "receive" && selected != "" {
		return fmt.Errorf("unexpected selection/effect")
	}
	switch in.Op {
	case "set":
		x, e := eval(in.A)
		if e != nil {
			return e
		}
		if in.Collection != nil {
			if e = in.Collection.Validate(x); e != nil {
				return e
			}
		}
		data[in.Dst] = x
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
	case "emit":
		if len(intents) != 1 || in.Effect == nil {
			return fmt.Errorf("emit cardinality")
		}
		emitted := intents[0]
		effect := in.Effect
		target, e := eval(effect.Target)
		if e != nil {
			return e
		}
		generation, e := eval(effect.Generation)
		if e != nil {
			return e
		}
		if effect.Kind == "message" {
			target, e = endpoint(effect.Target)
			if e != nil {
				return e
			}
			if binding.OperationID != "" {
				generation = schema.Int(binding.Caller.Generation)
			}
		}
		obligation, e := eval(effect.Obligation)
		if e != nil {
			return e
		}
		if emitted.Kind != effect.Kind || emitted.Target != target.Text || emitted.Generation != generation.Number || emitted.Obligation != obligation.Text || len(emitted.Payload) != len(effect.Payload) {
			return fmt.Errorf("emit endpoint/kind/payload shape")
		}
		for k, v := range effect.Payload {
			x, e := eval(v)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(x, emitted.Payload[k]) {
				return fmt.Errorf("emit payload %s", k)
			}
		}
		for k, v := range in.Updates {
			x, e := eval(v)
			if e != nil {
				return e
			}
			data[k] = x
		}
		if in.Dst != "" {
			data[in.Dst] = schema.Text(emitted.ID)
		}
	case "receive":
		if selected == "" || in.Match == nil || selected != o.ID {
			return fmt.Errorf("receive selection")
		}
		m := in.Match
		source, e := eval(m.Source)
		if e != nil {
			return e
		}
		gen, e := eval(m.Generation)
		if e != nil {
			return e
		}
		ob, e := eval(m.Obligation)
		if e != nil {
			return e
		}
		op, e := eval(m.Operation)
		if e != nil {
			return e
		}
		provenance := m.Provenance
		if m.Kind == "message" && binding.OperationID != "" {
			source, e = endpoint(m.Source)
			if e != nil {
				return e
			}
			gen = schema.Int(binding.Caller.Generation)
			op = schema.Text(binding.OperationID)
			provenance = "runtime-invocation-request"
		}
		if o.Kind != m.Kind || o.Provenance != provenance || o.Source != source.Text || o.SourceGeneration != gen.Number || o.Obligation != ob.Text || o.Operation != op.Text {
			return fmt.Errorf("original receive match")
		}
		if m.Kind == "invocation" {
			if e := callableOutcomeSchema(p.Author.Invocations[ob.Text].Output.Record, o.Payload); e != nil {
				return e
			}
		} else if e := p.Author.Receives[pre.PC].Record.Validate(o.Payload); e != nil {
			return e
		}
		for field, dst := range in.Bind {
			if m.Kind == "invocation" && strings.HasPrefix(field, "response.") && !o.Payload["response_present"].Flag {
				delete(data, dst)
				continue
			}
			v, ok := o.Payload[field]
			if !ok {
				return fmt.Errorf("missing bound field %s", field)
			}
			data[dst] = v
		}
	case "append":
		row := map[string]schema.Value{}
		for k, v := range in.Updates {
			x, e := eval(v)
			if e != nil {
				return e
			}
			row[k] = x
		}
		x, e := p.Author.Collections[in.Dst].Append(pre.Data[in.Dst], row)
		if e != nil {
			return e
		}
		data[in.Dst] = x
	case "foreach":
		x, e := eval(in.A)
		if e != nil {
			return e
		}
		if in.Collection == nil {
			return fmt.Errorf("missing collection contract")
		}
		if e = in.Collection.Validate(x); e != nil {
			return e
		}
		if _, ok := cursors[pre.PC]; ok {
			return fmt.Errorf("cursor reinitialized")
		}
		cursors[pre.PC] = executor.Cursor{Snapshot: x.Clone(), Index: 0}
	case "next":
		c, ok := cursors[in.Target]
		if !ok {
			return fmt.Errorf("next lacks cursor")
		}
		n, e := c.Snapshot.Len()
		if e != nil {
			return e
		}
		if c.Index < 0 || c.Index > n {
			return fmt.Errorf("cursor index")
		}
		if c.Index == n {
			delete(cursors, in.Target)
			pc = p.Code[in.Target].Otherwise
		} else {
			row, e := c.Snapshot.At(c.Index)
			if e != nil {
				return e
			}
			for field, dst := range in.Bind {
				v, ok := row[field]
				if !ok {
					return fmt.Errorf("cursor row binding")
				}
				data[dst] = v
			}
			c.Index++
			cursors[in.Target] = c
		}
	case "loop":
		if _, ok := cursors[in.Target]; !ok {
			return fmt.Errorf("loop lacks cursor")
		}
		pc = in.Target + 1
	case "end":
		pc = pre.PC
		status = in.Outcome
	default:
		return fmt.Errorf("unmapped instruction %s", in.Op)
	}
	if len(cursors) == 0 {
		cursors = nil
	}
	if post.PC != pc || post.Status != status || !reflect.DeepEqual(post.Data, data) || !reflect.DeepEqual(post.Cursors, cursors) {
		return fmt.Errorf("actual successor/data/cursor mismatch")
	}
	return nil
}

func callableOutcomeSchema(response schema.Record, p map[string]schema.Value) error {
	if p["status"].Kind != "string" || p["reason"].Kind != "string" || p["response_present"].Kind != "bool" {
		return fmt.Errorf("outcome envelope types")
	}
	if !p["response_present"].Flag {
		if len(p) != 3 || p["status"].Text == "succeeded" {
			return fmt.Errorf("absent outcome envelope")
		}
		return nil
	}
	r := map[string]schema.Value{}
	for k, v := range p {
		if strings.HasPrefix(k, "response.") {
			r[strings.TrimPrefix(k, "response.")] = v
		}
	}
	if len(p) != len(r)+3 {
		return fmt.Errorf("extra outcome fields")
	}
	return response.Validate(r)
}

func callableDiagnostic(p map[string]schema.Value) (model.JoinDiagnostic, error) {
	if e := callableapp.Report().Validate(p); e != nil {
		return model.JoinDiagnostic{}, e
	}
	n, e := p["answers"].Len()
	if e != nil {
		return model.JoinDiagnostic{}, e
	}
	d := model.JoinDiagnostic{ID: int(p["id"].Number), Classification: p["classification"].Text, Count: n}
	if n == 1 {
		r, e := p["answers"].At(0)
		if e != nil {
			return d, e
		}
		d.AnswerID, d.AnswerValue = int(r["id"].Number), int(r["value"].Number)
	}
	return d, nil
}

func callableEnvelope(v *model.Observation, d *model.JoinDiagnostic, p map[string]schema.Value) error {
	if v.Operation != "j" {
		return invocationJoinEnvelope(v, p)
	}
	v.Status, v.Reason, v.Present = p["status"].Text, p["reason"].Text, p["response_present"].Flag
	*d = model.JoinDiagnostic{}
	if v.Present {
		r := map[string]schema.Value{}
		for k, x := range p {
			if strings.HasPrefix(k, "response.") {
				r[strings.TrimPrefix(k, "response.")] = x
			}
		}
		var e error
		*d, e = callableDiagnostic(r)
		return e
	}
	if len(p) != 3 {
		return fmt.Errorf("absent J response has payload")
	}
	return nil
}

func callablePending(s runtime.Inspection, tr callableTrace, programs map[string]executor.Program, runs map[runtime.RunID]evidenceRun, bindings map[string]invocationJoinBinding, memberOp map[runtime.RunID]string) error {
	if tr.Pending == nil {
		return nil
	}
	member := tr.Pending.Run
	r := runs[member]
	p := programs[r.Program]
	pc := r.Current.PC
	if r.Current.Status != "open" || pc != tr.Pending.NextPC || p.Code[pc].Op != "receive" {
		return fmt.Errorf("pending marker not original unmatched final receive")
	}
	b := bindings[memberOp[member]]
	m := p.Code[pc].Match
	if m == nil || m.Source.Literal == nil || m.Source.Literal.Text != "caller" || m.Operation.Literal == nil || m.Operation.Literal.Text != "" {
		return fmt.Errorf("pending original match")
	}
	requested := false
	for _, step := range tr.Steps {
		if step.Step.Run == member && step.Step.PC == 0 && step.Selected != "" {
			requested = true
		}
	}
	if !requested {
		return fmt.Errorf("pending missing original request consume")
	}
	for _, row := range s.Tables["observations"] {
		o := decode[executor.Observation](row, "descriptor")
		if row["consumed_sequence"] == nil && o.Run == member && o.Kind == "message" && o.Provenance == "runtime-invocation-request" && o.Source == b.Caller.Subject && o.SourceGeneration == b.Caller.Generation && o.Operation == b.OperationID && o.Obligation == m.Obligation.Literal.Text {
			return fmt.Errorf("pending has eligible release")
		}
	}
	if r.Program == "ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67" {
		h := tr.Consumptions["h"]
		o := h.Observation
		if h.Step.Run != member || h.Step.PC != 2 || h.Step.Order >= tr.Pending.Order || o.Payload["status"].Text != "succeeded" || o.Payload["reason"].Text != "member_succeeded" || !o.Payload["response_present"].Flag {
			return fmt.Errorf("pending helper not actually consumed")
		}
	}
	if tr.Consumptions["e"].Observation.ID != "" || communicationRow(s, "composition_publications", "operation_id", tr.Handles["e"]) != nil {
		return fmt.Errorf("pending e published/consumed")
	}
	return nil
}

// Ordinary admission rows are checked separately from composition reply facts.
func callableAdmissions(s runtime.Inspection, intents map[string]executor.Intent, attempts map[string]executor.Authorization, replies map[string]invocationJoinReply) error {
	seen := map[string]bool{}
	for _, row := range s.Tables["admissions"] {
		d, e := readEvidence[executor.AdmissionDescriptor](row, "descriptor")
		if e != nil {
			return e
		}
		r, e := readEvidence[runtime.AdmissionResult](row, "result")
		if e != nil {
			return e
		}
		in, ok := intents[d.IntentID]
		reply, exists := replies[d.IntentID]
		if !ok || !exists || d.Kind != "message" || d.LogicalID != in.ID || d.Source != in.Run || d.Target != in.Target || d.Generation != in.Generation || d.Obligation != in.Obligation || !reflect.DeepEqual(d.Payload, in.Payload) || d.TransportID != r.TransportID || r.Accepted != reply.Accepted || r.Reason != reply.Reason || r.ObservationID != "" || r.Bound == nil || r.Bound.Subject != in.Target || r.Bound.Generation != in.Generation || !attempts[in.ID].Allowed {
			return fmt.Errorf("reply transport admission linkage")
		}
		seen[in.ID] = true
	}
	for id := range replies {
		if !seen[id] {
			return fmt.Errorf("reply has no transport admission")
		}
	}
	for _, row := range s.Tables["message_acknowledgments"] {
		id, _ := row["intent_id"].(string)
		if !seen[id] {
			return fmt.Errorf("ack has no admitted reply")
		}
	}
	return nil
}
