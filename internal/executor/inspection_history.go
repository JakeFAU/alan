package executor

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

func readWait(m *Match, p Projection) *ReceiveWait {
	w := &ReceiveWait{Kind: m.Kind, Provenance: m.Provenance}
	var err error
	if w.Source, err = stringOperand(m.Source, p.Data); err != nil {
		return w
	}
	if w.SourceGeneration, err = intOperand(m.Generation, p.Data); err != nil {
		return w
	}
	if w.Obligation, err = stringOperand(m.Obligation, p.Data); err != nil {
		return w
	}
	if w.OperationID, err = stringOperand(m.Operation, p.Data); err != nil {
		return w
	}
	w.Resolved = true
	return w
}

func (w *ReceiveWait) matches(o Observation) bool {
	return w.Resolved && w.Kind == o.Kind && w.Provenance == o.Provenance && w.Source == o.Source && w.SourceGeneration == o.SourceGeneration && w.Obligation == o.Obligation && w.OperationID == o.Operation
}

func (b *inspectionBuilder) buildHistory() error {
	for _, row := range b.tables["intents"] {
		v, err := inspectionDecode[Intent](row, "descriptor")
		if err != nil {
			return err
		}
		if v.ID != row["id"] || v.Run != inspectionRun(row) || v.Sequence != row["sequence"] || int64(v.Ordinal) != row["ordinal"] || b.runs[v.Run] == nil || v.ID != identity(v.Run.Subject, v.Run.Generation, v.Sequence, v.Ordinal) {
			return corrupt("original operation identity")
		}
		b.intents[v.ID] = retainedIntent{Intent: v, revision: inspectInt(row, "source_revision"), raw: []byte(inspectText(row, "descriptor"))}
	}
	b.snap.history = make([]ExecutionStep, 0, len(b.tables["history"]))
	last := map[RunID]ExecutionStep{}
	seen := 0
	for _, row := range b.tables["history"] {
		id := inspectionRun(row)
		run := b.runs[id]
		seq := inspectInt(row, "sequence")
		ref := ReductionRef{id, seq}
		prior := last[id]
		if run == nil || seq != prior.Sequence+1 || seq > run.Sequence {
			return corrupt("history sequence %+v", ref)
		}
		epoch, revision := inspectInt(row, "epoch"), inspectInt(row, "source_revision")
		if epoch < 1 || epoch < prior.Epoch || epoch > run.Epoch || revision < 1 || revision < prior.SourceRevision || revision > run.Revision {
			return corrupt("history activation or authority %+v", ref)
		}
		pre := run.Initial
		if seq > 1 {
			pre = b.after[prior.ReductionRef]
		}
		program := b.resolved[id]
		if pre.Status != "open" || pre.PC < 0 || pre.PC >= len(program.Code) {
			return corrupt("commit from closed or missing instruction %+v", ref)
		}
		inst := program.Code[pre.PC]
		next, err := inspectionDecode[Projection](row, "successor")
		if err != nil {
			return err
		}
		if err = validateProjection(next, program, b.limits); err != nil {
			return corrupt("invalid retained successor %+v: %v", ref, err)
		}
		selected := inspectText(row, "selected_observation")
		if (inst.Op == "receive") != (selected != "") {
			return corrupt("receive selection %+v", ref)
		}
		if len(inspectText(row, "intents")) > b.limits.MaxOutputBytes {
			return ErrLimit
		}
		ops, err := inspectionDecode[[]Intent](row, "intents")
		if err != nil {
			return err
		}
		if ops == nil || (inst.Op == "emit" && len(ops) != 1) || (inst.Op != "emit" && len(ops) != 0) {
			return corrupt("instruction operations %+v", ref)
		}
		step := ExecutionStep{ReductionRef: ref, Epoch: epoch, SourceRevision: revision, BeforePC: pre.PC, AfterPC: next.PC, Instruction: inst.Op, Status: next.Status, SelectedObservationID: selected}
		if (inst.Op == "end" && (next.PC != pre.PC || next.Status != inst.Outcome)) || (inst.Op != "end" && next.Status != "open") {
			return corrupt("terminal reduction %+v", ref)
		}
		for n, v := range ops {
			stored, ok := b.intents[v.ID]
			if !ok || v.Run != id || v.Sequence != seq || v.Ordinal != n || stored.revision != revision || !reflect.DeepEqual(stored.Intent, v) {
				return corrupt("unbound operation %s", v.ID)
			}
			e := inst.Effect
			target, e1 := stringOperand(e.Target, pre.Data)
			gen, e2 := intOperand(e.Generation, pre.Data)
			ob, e3 := stringOperand(e.Obligation, pre.Data)
			if e1 != nil || e2 != nil || e3 != nil || v.Kind != e.Kind || v.Target != target || v.Generation != gen || v.Obligation != ob || v.Payload == nil || len(v.Payload) != len(e.Payload) {
				return corrupt("operation source %s", v.ID)
			}
			for key, operand := range e.Payload {
				want, e := eval(operand, pre.Data)
				val, ok := v.Payload[key]
				if e != nil || !ok || !val.Equal(want) {
					return corrupt("operation payload %s", v.ID)
				}
			}
			if inst.Dst != "" && next.Data[inst.Dst] != Text(v.ID) {
				return corrupt("original operation handle %s", v.ID)
			}
			seen++
			step.OperationIDs = append(step.OperationIDs, v.ID)
			run.OperationIDs = append(run.OperationIDs, v.ID)
			b.snap.Operations = append(b.snap.Operations, v)
		}
		if root := b.membership[id]; root != "" {
			f := b.facts[compositionIdentity("alan/fact/execution/v1", []any{id, seq})]
			e := compositionHistoryEvidence{Run: id, Sequence: seq, Epoch: epoch, Revision: revision, Selected: selected, Successor: json.RawMessage(inspectText(row, "successor")), Intents: json.RawMessage(inspectText(row, "intents"))}
			if f.Kind != compositionExecutionFact || f.Run != id || f.Sequence != seq || !slices.Equal(f.RootIDs, []string{root}) || f.RecordDigest != compositionIdentity("alan/history-row/v1", e) {
				return corrupt("composition execution fact %+v", ref)
			}
		}
		b.before[ref] = pre
		b.after[ref] = next
		b.snap.history = append(b.snap.history, step)
		b.steps[ref] = &b.snap.history[len(b.snap.history)-1]
		last[id] = step
	}
	if seen != len(b.intents) {
		return corrupt("orphan operation")
	}
	terminals := map[RunID]map[string]any{}
	for _, row := range b.tables["terminal_obligations"] {
		terminals[inspectionRun(row)] = row
	}
	lifecycle := map[string]LifecycleResult{}
	for _, row := range b.tables["lifecycle_results"] {
		d, e := inspectionDecode[LifecycleResult](row, "result")
		if e != nil {
			return e
		}
		lifecycle[d.Request.Intent.ID] = d
	}
	for _, run := range b.snap.Runs {
		step := last[run.ID]
		p := run.Initial
		if run.Sequence != step.Sequence {
			return corrupt("missing run history %+v", run.ID)
		}
		if run.Sequence > 0 {
			p = b.after[step.ReductionRef]
		}
		terminal := terminals[run.ID]
		if (terminal == nil) != (run.Projection.Status == "open") {
			return corrupt("terminal obligation %+v", run.ID)
		}
		if terminal != nil {
			d, e := inspectionDecode[Projection](terminal, "outcome")
			if e != nil || terminal["sequence"] != run.Sequence || !reflect.DeepEqual(d, run.Projection) {
				return corrupt("terminal outcome %+v", run.ID)
			}
			if op := inspectText(terminal, "operation_id"); op != "" {
				l := lifecycle[op]
				if !l.Accepted || l.Before == nil || l.After == nil || l.Before.Run != run.ID || l.After.Run != run.ID || l.Successor != 0 || l.Before.Sequence != run.Sequence || p.Status != "open" || !reflect.DeepEqual(l.Before.Projection, p) || !reflect.DeepEqual(l.After.Projection, d) || (d.Status != "abandoned" && d.Status != "cancelled") {
					return corrupt("invalid authority closure %+v", run.ID)
				}
				p = d
			}
		}
		if !reflect.DeepEqual(p, run.Projection) {
			return corrupt("unexplained current projection %+v", run.ID)
		}
		inst := b.resolved[run.ID].Code[p.PC]
		if p.Status == "open" {
			run.Instruction = inst.Op
			if inst.Match != nil && inst.Op == "receive" {
				run.Wait = readWait(inst.Match, p)
			}
		}
	}
	return nil
}

func (b *inspectionBuilder) checkSelections() error {
	for _, step := range b.snap.history {
		if step.SelectedObservationID == "" {
			continue
		}
		o := b.observations[step.SelectedObservationID]
		pre := b.before[step.ReductionRef]
		next := b.after[step.ReductionRef]
		in := b.resolved[step.Run].Code[pre.PC]
		if o == nil || o.Run != step.Run || o.SelectedAt == nil || *o.SelectedAt != step.ReductionRef || !readWait(in.Match, pre).matches(o.Observation) {
			return corrupt("wrong original selected at %+v", step.ReductionRef)
		}
		for field, dst := range in.Bind {
			if in.Match.Kind == "invocation" && strings.HasPrefix(field, "response.") && !o.Payload["response_present"].Flag {
				if _, exists := next.Data[dst]; exists {
					return corrupt("nonexistent response was bound at %+v", step.ReductionRef)
				}
				continue
			}
			v, ok := o.Payload[field]
			bound, present := next.Data[dst]
			if !ok || !present || !bound.Equal(v) {
				return corrupt("selected observation binding %+v", step.ReductionRef)
			}
		}
	}
	return nil
}
