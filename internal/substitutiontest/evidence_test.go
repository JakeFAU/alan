//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// These are retained-row shapes, not a second reducer. The checks below link
// observations/receive guards and terminal freezes before any lossy abstraction.
type evidenceOperand struct {
	Ref     string
	Literal *schema.Value
	Offset  json.RawMessage
}
type evidenceMatch struct {
	Kind, Provenance                          string
	Source, Generation, Obligation, Operation evidenceOperand
}
type evidenceInstruction struct {
	Op, Outcome string
	Match       *evidenceMatch
	Bind        map[string]string
}
type evidenceProgram struct{ Code []evidenceInstruction }
type evidenceRun struct {
	ID               runtime.RunID
	Program          string
	Initial, Current runtime.Projection
	Sequence         int64
}
type evidenceBinding struct {
	ID, Namespace, InvocationID, OperationID, TemplateID, RootID string
	Caller                                                       runtime.RunID
	Members                                                      map[string]runtime.RunID
	Programs                                                     map[string]string
}
type evidenceHistory struct {
	Run       runtime.RunID
	Sequence  int64
	Selected  string
	Successor runtime.Projection
	Intents   []intent
}
type evidenceLifecycle struct {
	Request struct {
		Intent    intent
		Operation string
	}
	Accepted      bool
	Reason        string
	Before, After *struct {
		Run        runtime.RunID
		ProgramID  string
		Sequence   int64
		Projection runtime.Projection
	}
}

func readEvidence[T any](r map[string]any, k string) (T, error) {
	var v T
	raw, ok := r[k].(string)
	if !ok {
		return v, fmt.Errorf("missing JSON %s", k)
	}
	if e := json.Unmarshal([]byte(raw), &v); e != nil {
		return v, fmt.Errorf("invalid JSON %s: %w", k, e)
	}
	return v, nil
}
func evidenceNumber(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case float64:
		if n == float64(int64(n)) {
			return int64(n), nil
		}
	case int:
		return int64(n), nil
	}
	return 0, fmt.Errorf("missing/inexact integer %v", v)
}
func evidenceID(r map[string]any) (runtime.RunID, error) {
	subject, ok := r["subject"].(string)
	if !ok || subject == "" {
		return runtime.RunID{}, fmt.Errorf("missing subject")
	}
	generation, e := evidenceNumber(r["generation"])
	return runtime.RunID{Subject: subject, Generation: generation}, e
}
func evidenceKey(id runtime.RunID, seq int64) string {
	return fmt.Sprintf("%s/%d/%d", id.Subject, id.Generation, seq)
}
func evidenceValue(o evidenceOperand, data map[string]schema.Value) (schema.Value, error) {
	if len(o.Offset) > 0 && string(o.Offset) != "null" {
		return schema.Value{}, fmt.Errorf("unsupported offset in retained receive guard")
	}
	if o.Ref != "" {
		v, ok := data[o.Ref]
		if !ok {
			return v, fmt.Errorf("missing receive handle/register %s", o.Ref)
		}
		return v, nil
	}
	if o.Literal == nil {
		return schema.Value{}, fmt.Errorf("missing receive operand")
	}
	return *o.Literal, nil
}
func sameValues(a, b map[string]schema.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		w, ok := b[k]
		if !ok || !reflect.DeepEqual(v, w) {
			return false
		}
	}
	return true
}
func sameIDs(a, b []string) bool {
	aa, bb := append([]string{}, a...), append([]string{}, b...)
	sort.Strings(aa)
	sort.Strings(bb)
	return reflect.DeepEqual(aa, bb)
}

// validateRetained returns only explicit logical cancellation, never success's
// admission fence. Missing/mismatched evidence is an error, not an empty trace.
func validateRetained(s runtime.Inspection) (closed bool, err error) {
	fail := func(format string, args ...any) (bool, error) { return false, fmt.Errorf(format, args...) }
	if len(s.Tables["composition_roots"]) != 1 || len(s.Roots) != 1 {
		return fail("expected one retained root and completion view")
	}
	root, e := readEvidence[runtime.ResourceRoot](s.Tables["composition_roots"][0], "descriptor")
	if e != nil {
		return false, e
	}
	if root.ID == "" || s.Roots[0].RootID != root.ID || s.Roots[0].Run != root.Run {
		return fail("root identity/completion mismatch")
	}
	runs := map[runtime.RunID]evidenceRun{}
	programs := map[string]evidenceProgram{}
	for _, row := range s.Tables["programs"] {
		p, e := readEvidence[evidenceProgram](row, "bytes")
		if e != nil {
			return false, e
		}
		id, ok := row["id"].(string)
		if !ok || id == "" {
			return fail("missing program identity")
		}
		programs[id] = p
	}
	for _, row := range s.Tables["runs"] {
		id, e := evidenceID(row)
		if e != nil {
			return false, e
		}
		initial, e := readEvidence[runtime.Projection](row, "initial")
		if e != nil {
			return false, e
		}
		current, e := readEvidence[runtime.Projection](row, "projection")
		if e != nil {
			return false, e
		}
		sequence, e := evidenceNumber(row["sequence"])
		if e != nil {
			return false, e
		}
		program, ok := row["program_id"].(string)
		if !ok || len(programs[program].Code) == 0 {
			return fail("run %v lacks original program", id)
		}
		if _, ok := runs[id]; ok {
			return fail("duplicate run %v", id)
		}
		runs[id] = evidenceRun{id, program, initial, current, sequence}
	}
	if r, ok := runs[root.Run]; !ok || r.Current.Status != s.Roots[0].LocalStatus {
		return fail("root local projection/completion mismatch")
	}
	static := map[runtime.RunID]bool{}
	for _, id := range root.Members {
		if _, ok := runs[id]; !ok {
			return fail("missing static member %v", id)
		}
		static[id] = true
	}
	for _, row := range s.Tables["composition_static_members"] {
		v, e := readEvidence[struct {
			RootID string
			Run    runtime.RunID
		}](row, "descriptor")
		if e != nil {
			return false, e
		}
		if v.RootID != root.ID || !static[v.Run] {
			return fail("static parent/root/member mismatch")
		}
		delete(static, v.Run)
	}
	if len(static) != 0 {
		return fail("missing static membership evidence")
	}
	for _, id := range root.Members {
		static[id] = true
	}
	intents := map[string]intent{}
	for _, row := range s.Tables["intents"] {
		in, e := readEvidence[intent](row, "descriptor")
		if e != nil {
			return false, e
		}
		if in.ID == "" || row["id"] != in.ID {
			return fail("intent identity mismatch")
		}
		intents[in.ID] = in
	}
	ads := map[string]runtime.InvocationAdmission{}
	byInv := map[string]runtime.InvocationAdmission{}
	memberParent := map[runtime.RunID]string{}
	for _, row := range s.Tables["composition_admission_decisions"] {
		a, e := readEvidence[runtime.InvocationAdmission](row, "descriptor")
		if e != nil {
			return false, e
		}
		i := a.Invocation
		in, ok := intents[i.OperationID]
		if !ok || row["operation_id"] != i.OperationID || i.ID == "" || i.RootID != root.ID || in.Kind != "invocation" || in.Run != i.Caller || in.Sequence != i.SourceSequence || in.Target != i.TemplateID || !sameValues(in.Payload, i.Inputs) {
			return fail("admission original intent/root/caller mismatch %s", i.OperationID)
		}
		if _, ok := ads[i.OperationID]; ok {
			return fail("duplicate admission")
		}
		ads[i.OperationID] = a
		byInv[i.ID] = a
		if a.Accepted {
			if len(a.Members) != 1 {
				return fail("fixture expects singleton admitted member")
			}
			for _, m := range a.Members {
				if _, ok := runs[m]; !ok {
					return fail("admitted member missing")
				}
				if memberParent[m] != "" {
					return fail("member has multiple parents")
				}
				memberParent[m] = i.ID
			}
		}
	}
	bindings := map[string]evidenceBinding{}
	for _, row := range s.Tables["composition_bindings"] {
		b, e := readEvidence[evidenceBinding](row, "descriptor")
		if e != nil {
			return false, e
		}
		a, ok := ads[b.OperationID]
		if !ok || !a.Accepted || b.ID != a.BindingID || b.Namespace != s.Namespace || b.InvocationID != a.Invocation.ID || b.TemplateID != a.Invocation.TemplateID || b.RootID != root.ID || b.Caller != a.Invocation.Caller || len(b.Members) != len(a.Members) {
			return fail("binding operation/parent/root/caller mismatch")
		}
		for name, m := range b.Members {
			if memberParent[m] != b.InvocationID || runs[m].Program != b.Programs[name] {
				return fail("binding member/program mismatch")
			}
		}
		bindings[b.InvocationID] = b
	}
	members := map[runtime.RunID]bool{}
	for _, row := range s.Tables["composition_members"] {
		m, e := readEvidence[struct {
			Run                                     runtime.RunID
			InvocationID, RootID, ProgramID, Member string
		}](row, "descriptor")
		if e != nil {
			return false, e
		}
		b, ok := bindings[m.InvocationID]
		if !ok || m.RootID != root.ID || memberParent[m.Run] != m.InvocationID || b.Members[m.Member] != m.Run || runs[m.Run].Program != m.ProgramID {
			return fail("member original parent/root/program mismatch")
		}
		members[m.Run] = true
	}
	for id, parent := range memberParent {
		if !members[id] || bindings[parent].InvocationID == "" {
			return fail("missing member/binding evidence")
		}
	}
	for _, a := range ads {
		expected := ""
		if parent := memberParent[a.Invocation.Caller]; parent != "" {
			expected = parent
		} else if !static[a.Invocation.Caller] {
			return fail("invocation caller outside retained root")
		}
		if a.Invocation.ParentInvocationID != expected {
			return fail("invocation parent/root/caller mismatch")
		}
	}
	histories := map[string]evidenceHistory{}
	for _, row := range s.Tables["history"] {
		id, e := evidenceID(row)
		if e != nil {
			return false, e
		}
		seq, e := evidenceNumber(row["sequence"])
		if e != nil {
			return false, e
		}
		succ, e := readEvidence[runtime.Projection](row, "successor")
		if e != nil {
			return false, e
		}
		ins, e := readEvidence[[]intent](row, "intents")
		if e != nil {
			return false, e
		}
		selected := ""
		if row["selected_observation"] != nil {
			var ok bool
			selected, ok = row["selected_observation"].(string)
			if !ok || selected == "" {
				return fail("invalid selected observation identity")
			}
		}
		key := evidenceKey(id, seq)
		if _, ok := histories[key]; ok {
			return fail("duplicate history")
		}
		histories[key] = evidenceHistory{id, seq, selected, succ, ins}
		for _, in := range ins {
			if actual, ok := intents[in.ID]; !ok || !reflect.DeepEqual(actual, in) || in.Run != id || in.Sequence != seq {
				return fail("history intent linkage missing/mismatch")
			}
		}
	}
	facts := map[string]fact{}
	executions := map[string]fact{}
	admitFacts := map[string]bool{}
	pubFacts := map[string]bool{}
	freezeFacts := map[string]fact{}
	for _, row := range s.Tables["composition_facts"] {
		f, e := readEvidence[fact](row, "descriptor")
		if e != nil {
			return false, e
		}
		order, e := evidenceNumber(row["decision_order"])
		if e != nil {
			return false, e
		}
		if f.ID == "" || row["id"] != f.ID || row["kind"] != f.Kind || order != f.Order || len(f.RootIDs) != 1 || f.RootIDs[0] != root.ID {
			return fail("missing fact identity")
		}
		facts[f.ID] = f
		switch f.Kind {
		case "execution":
			key := evidenceKey(f.Run, f.Sequence)
			if _, ok := histories[key]; !ok {
				return fail("execution fact without original history")
			}
			executions[key] = f
		case "admission":
			if _, ok := ads[f.OperationID]; !ok {
				return fail("admission fact missing original record")
			}
			a := ads[f.OperationID]
			if f.Run != a.Invocation.Caller || f.Sequence != a.Invocation.SourceSequence {
				return fail("admission fact source/caller mismatch")
			}
			admitFacts[f.OperationID] = true
		case "outcome":
			pubFacts[f.OperationID] = true
		case "freeze":
			freezeFacts[f.OperationID] = f
		}
	}
	for _, h := range histories {
		if (static[h.Run] || members[h.Run]) && executions[evidenceKey(h.Run, h.Sequence)].ID == "" {
			return fail("missing execution fact for history")
		}
	}
	for op := range ads {
		if !admitFacts[op] {
			return fail("missing admission fact")
		}
	}
	// Accepted root lifecycle transitions are independent cancellation evidence.
	cancellations := map[runtime.RunID]evidenceLifecycle{}
	for _, row := range s.Tables["lifecycle_results"] {
		l, e := readEvidence[evidenceLifecycle](row, "result")
		if e != nil {
			return false, e
		}
		if !l.Accepted || l.Reason != "closed" {
			continue
		}
		if l.Request.Operation != "CancelRun" && l.Request.Operation != "AbandonRun" {
			continue
		}
		if l.Before == nil || l.After == nil || l.Before.Run != l.After.Run || l.Request.Intent.Target != l.After.Run.Subject || l.Before.Projection.Status != "open" || l.After.Projection.Status != "cancelled" {
			return fail("accepted cancellation lifecycle linkage mismatch")
		}
		r, ok := runs[l.After.Run]
		if !ok || r.Current.Status != "cancelled" || r.Program != l.After.ProgramID || r.Sequence != l.After.Sequence {
			return fail("accepted cancellation/current projection mismatch")
		}
		if actual, ok := intents[l.Request.Intent.ID]; !ok || !reflect.DeepEqual(actual, l.Request.Intent) || actual.Kind != "lifecycle" || actual.Payload["operation"].Text != l.Request.Operation || !reflect.DeepEqual(r.Current, l.After.Projection) {
			return fail("cancellation original intent/projection linkage mismatch")
		}
		cancellations[l.After.Run] = l
		if l.After.Run == root.Run {
			closed = true
		}
	}
	if runs[root.Run].Current.Status == "cancelled" && !closed {
		return fail("cancelled root lacks actual accepted lifecycle transition")
	}
	if len(s.Tables["composition_closures"]) != 0 {
		return fail("logical closure outside declared CancelRun profile")
	}
	observations := map[string]observation{}
	consumed := map[string]int64{}
	for _, row := range s.Tables["observations"] {
		o, e := readEvidence[observation](row, "descriptor")
		if e != nil {
			return false, e
		}
		id, e := evidenceID(row)
		if e != nil {
			return false, e
		}
		if o.ID == "" || row["id"] != o.ID || id != o.Run {
			return fail("observation identity/caller mismatch")
		}
		if _, ok := observations[o.ID]; ok {
			return fail("duplicate observation")
		}
		observations[o.ID] = o
		if row["consumed_sequence"] != nil {
			seq, e := evidenceNumber(row["consumed_sequence"])
			if e != nil {
				return false, e
			}
			consumed[o.ID] = seq
			h, ok := histories[evidenceKey(o.Run, seq)]
			if !ok || h.Selected != o.ID {
				return fail("consumed observation lacks selected history")
			}
		}
	}
	publications := map[string]runtime.InvocationOutcome{}
	for _, row := range s.Tables["composition_publications"] {
		p, e := readEvidence[runtime.InvocationOutcome](row, "descriptor")
		if e != nil {
			return false, e
		}
		a, ok := ads[p.OperationID]
		if !ok || row["operation_id"] != p.OperationID || p.InvocationID != a.Invocation.ID || p.TemplateID != a.Invocation.TemplateID || p.Caller != a.Invocation.Caller || !pubFacts[p.OperationID] {
			return fail("publication operation/parent/root/caller mismatch")
		}
		publications[p.OperationID] = p
	}
	rootPublished := false
	for _, row := range s.Tables["composition_root_publications"] {
		rootPublished = true
		p, e := readEvidence[runtime.InvocationOutcome](row, "descriptor")
		if e != nil {
			return false, e
		}
		if p.OperationID != "root:"+root.ID || p.Caller != root.Run || s.Roots[0].Outcome == nil || !reflect.DeepEqual(p, *s.Roots[0].Outcome) || !pubFacts[p.OperationID] {
			return fail("root publication/completion linkage mismatch")
		}
	}
	if (s.Roots[0].Outcome != nil) != rootPublished {
		return fail("missing root publication/completion evidence")
	}
	for op := range pubFacts {
		if strings.HasPrefix(op, "root:") {
			if s.Roots[0].Outcome == nil {
				return fail("outcome fact missing root publication")
			}
		} else if _, ok := publications[op]; !ok {
			return fail("outcome fact missing publication")
		}
	}
	deliveries := map[string]string{}
	for _, row := range s.Tables["composition_outcome_deliveries"] {
		op, ok := row["operation_id"].(string)
		if !ok {
			return fail("missing delivered operation")
		}
		id, ok := row["observation_id"].(string)
		if !ok || id == "" {
			return fail("missing delivered observation")
		}
		o, ok := observations[id]
		p, present := publications[op]
		if !ok || !present || o.Kind != "invocation" || o.Operation != op || o.Run != p.Caller {
			return fail("delivery/publication/observation linkage mismatch")
		}
		deliveries[op] = id
	}
	for id, o := range observations {
		if o.Kind != "invocation" {
			continue
		}
		p, ok := publications[o.Operation]
		a, admitted := ads[o.Operation]
		if !ok || !admitted || deliveries[o.Operation] != id || o.Provenance != "runtime-invocation" || o.Source != a.Invocation.TemplateID || o.SourceGeneration != 1 || o.Run != a.Invocation.Caller || o.Obligation != intents[o.Operation].Obligation {
			return fail("invocation observation provenance/sourcegeneration/operation/obligation/caller mismatch")
		}
		expected := map[string]schema.Value{"status": schema.Text(p.Status), "reason": schema.Text(p.Reason), "response_present": schema.Bool(p.ResponsePresent)}
		if p.ResponsePresent {
			for k, v := range p.Response {
				expected["response."+k] = v
			}
		}
		if !sameValues(o.Payload, expected) {
			return fail("actual invocation observation payload differs from publication")
		}
	}
	// Check the selected original receive instruction and its handle from the
	// pre-transition retained projection; also check its actual binding outputs.
	for _, h := range histories {
		r, ok := runs[h.Run]
		if !ok {
			return fail("history run missing")
		}
		previous := r.Initial
		if h.Sequence > 1 {
			prev, ok := histories[evidenceKey(h.Run, h.Sequence-1)]
			if !ok {
				return fail("missing predecessor projection")
			}
			previous = prev.Successor
		}
		code := programs[r.Program].Code
		if previous.PC < 0 || previous.PC >= len(code) {
			return fail("invalid predecessor PC")
		}
		instruction := code[previous.PC]
		if h.Selected == "" {
			if instruction.Op == "receive" {
				return fail("receive history missing selected observation")
			}
			continue
		}
		o, ok := observations[h.Selected]
		if !ok || o.Run != h.Run || consumed[o.ID] != h.Sequence {
			return fail("selected observation missing/mismatched caller/consumption")
		}
		if instruction.Op != "receive" || instruction.Match == nil {
			return fail("selected observation not at retained receive")
		}
		m := instruction.Match
		if o.Kind == "invocation" {
			source, e := evidenceValue(m.Source, previous.Data)
			if e != nil {
				return false, e
			}
			generation, e := evidenceValue(m.Generation, previous.Data)
			if e != nil {
				return false, e
			}
			obligation, e := evidenceValue(m.Obligation, previous.Data)
			if e != nil {
				return false, e
			}
			operation, e := evidenceValue(m.Operation, previous.Data)
			if e != nil {
				return false, e
			}
			if m.Kind != o.Kind || m.Provenance != o.Provenance || source.Kind != "string" || source.Text != o.Source || generation.Kind != "int" || generation.Number != o.SourceGeneration || obligation.Text != o.Obligation || operation.Text != o.Operation || m.Operation.Ref == "" {
				return fail("selected receive guard/original handle mismatch")
			}
			for field, destination := range instruction.Bind {
				if value, present := o.Payload[field]; present {
					actual, ok := h.Successor.Data[destination]
					if !ok || !reflect.DeepEqual(actual, value) {
						return fail("selected observation/bound successor payload mismatch")
					}
				}
			}
		}
	}
	freezes := map[string]runtime.InvocationFreeze{}
	for _, row := range s.Tables["composition_freezes"] {
		f, e := readEvidence[runtime.InvocationFreeze](row, "descriptor")
		if e != nil {
			return false, e
		}
		ff, ok := freezeFacts[f.InvocationID]
		if !ok || ff.Run != f.Run || ff.Sequence != f.Sequence || ff.Order != f.DecisionOrder || row["scope_id"] != f.InvocationID || row["root_id"] != root.ID {
			return fail("terminal freeze fact/scope/root mismatch")
		}
		if _, ok := freezes[f.InvocationID]; ok {
			return fail("duplicate scope freeze")
		}
		r, ok := runs[f.Run]
		if !ok || r.Current.Status != f.Status || r.Sequence != f.Sequence || f.Status == "open" {
			return fail("local terminal projection/freeze mismatch")
		}
		scopeMembers := []runtime.RunID{}
		if f.InvocationID == "root:"+root.ID {
			scopeMembers = root.Members
		} else {
			a, ok := byInv[f.InvocationID]
			if !ok || !a.Accepted {
				return fail("freeze has no original admitted invocation")
			}
			scopeMembers = a.Members
		}
		inScope := map[runtime.RunID]bool{}
		for _, m := range scopeMembers {
			inScope[m] = true
		}
		if !inScope[f.Run] {
			return fail("freeze run not in original scope")
		}
		trigger, ok := facts[f.Trigger]
		if !ok || trigger.Run != f.Run || trigger.Sequence != f.Sequence {
			return fail("freeze trigger missing/mismatched terminal event")
		}
		if trigger.Kind == "execution" {
			h := histories[evidenceKey(f.Run, f.Sequence)]
			if h.Successor.Status != f.Status {
				return fail("terminal history/freeze mismatch")
			}
		} else if trigger.Kind == "lifecycle" {
			if cancellation, ok := cancellations[f.Run]; !ok || cancellation.Request.Intent.ID != trigger.OperationID {
				return fail("freeze lacks accepted cancellation event")
			}
		} else {
			return fail("unsupported freeze trigger")
		}
		inv, ext, replies := []string{}, []string{}, []string{}
		caller := runtime.RunID{}
		if a, ok := byInv[f.InvocationID]; ok {
			caller = a.Invocation.Caller
		}
		for id, in := range intents {
			if !inScope[in.Run] {
				continue
			}
			event, ok := executions[evidenceKey(in.Run, in.Sequence)]
			if !ok || event.Order >= f.DecisionOrder {
				continue
			}
			switch in.Kind {
			case "invocation":
				inv = append(inv, id)
			case "adapter", "external":
				ext = append(ext, id)
			case "message":
				if in.Target == caller.Subject {
					replies = append(replies, id)
				}
			}
		}
		if !sameIDs(inv, f.InvocationIntents) || !sameIDs(ext, f.ExternalIntents) || !sameIDs(replies, f.ReplyIntents) {
			return fail("terminal freeze captured intent obligations mismatch")
		}
		freezes[f.InvocationID] = f
	}
	// Every terminal singleton must have its own freeze. The multi-static root
	// scope may still be open while only one successful branch has completed.
	for id, parent := range memberParent {
		r := runs[id]
		if r.Current.Status != "open" {
			if _, ok := freezes[parent]; !ok {
				return fail("terminal member missing original freeze")
			}
		}
	}
	rootNeedsFreeze := false
	allSuccess := true
	for _, id := range root.Members {
		status := runs[id].Current.Status
		if status != "succeeded" {
			allSuccess = false
		}
		if status != "open" && status != "succeeded" {
			rootNeedsFreeze = true
		}
	}
	if allSuccess {
		rootNeedsFreeze = true
	}
	if rootNeedsFreeze {
		if _, ok := freezes["root:"+root.ID]; !ok {
			return fail("terminal root scope missing original freeze")
		}
	}
	for id, r := range runs {
		if !static[id] && !members[id] {
			continue
		}
		if r.Current.Status == "open" {
			continue
		}
		h, ok := histories[evidenceKey(id, r.Sequence)]
		if !ok {
			return fail("terminal member history missing")
		}
		if _, cancelled := cancellations[id]; cancelled {
			continue
		}
		if !reflect.DeepEqual(h.Successor, r.Current) {
			return fail("terminal projection differs from original terminal history")
		}
		previous := r.Initial
		if r.Sequence > 1 {
			previous = histories[evidenceKey(id, r.Sequence-1)].Successor
		}
		code := programs[r.Program].Code
		if previous.PC < 0 || previous.PC >= len(code) || code[previous.PC].Op != "end" || code[previous.PC].Outcome != r.Current.Status {
			return fail("local terminal not linked to original end instruction")
		}
	}
	replyRows := map[string]reply{}
	for _, row := range s.Tables["composition_replies"] {
		r, e := readEvidence[reply](row, "descriptor")
		if e != nil {
			return false, e
		}
		if actual, ok := intents[r.Intent.ID]; !ok || !reflect.DeepEqual(actual, r.Intent) {
			return fail("reply original intent/payload mismatch")
		}
		replyRows[r.Intent.ID] = r
	}
	for _, f := range facts {
		if f.Kind == "reply" {
			r, ok := replyRows[f.OperationID]
			if !ok || f.Run != r.Intent.Run || f.Sequence != r.Intent.Sequence {
				return fail("reply fact missing/mismatched original reply")
			}
		}
	}
	for op, p := range publications {
		a := ads[op]
		if !a.Accepted {
			if p.Status != "rejected" || p.Reason != a.Reason || p.ResponsePresent {
				return fail("refused admission/publication mismatch")
			}
			continue
		}
		f, ok := freezes[a.Invocation.ID]
		if !ok {
			return fail("published member missing original freeze")
		}
		response := map[string]schema.Value{}
		present := false
		for _, id := range f.ReplyIntents {
			r, ok := replyRows[id]
			if !ok || r.InvocationID != a.Invocation.ID {
				return fail("publication freeze reply evidence missing/mismatched")
			}
			if r.Accepted {
				response = r.Intent.Payload
				present = true
			}
		}
		if p.ResponsePresent != present || !sameValues(p.Response, response) {
			return fail("publication does not preserve actual accepted frozen reply")
		}
		if f.Status == "succeeded" && !present {
			if p.Status != "failed" || p.Reason != "missing_reply" {
				return fail("success-without-reply local/freeze/envelope mismatch")
			}
		} else if p.Status != f.Status || p.Reason != "member_"+f.Status {
			return fail("local terminal/freeze/publication mismatch")
		}
	}
	if outcome := s.Roots[0].Outcome; outcome != nil {
		freeze, ok := freezes["root:"+root.ID]
		if !ok || outcome.Status != freeze.Status || outcome.Reason != "member_"+freeze.Status || outcome.ResponsePresent {
			return fail("root terminal/freeze/publication mismatch")
		}
	}
	return closed, nil
}
