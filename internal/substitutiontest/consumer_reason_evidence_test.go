//go:build integration

package substitutiontest

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/JakeFAU/alan/internal/executor"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type reasonStep struct {
	Run             runtime.RunID
	Program         string
	Sequence, Order int64
	PC, NextPC      int
}

type reasonConsumption struct {
	Step        reasonStep
	Observation executor.Observation
	Bind        map[string]string
}

type reasonClassificationEvidence struct {
	Step     reasonStep
	Tag      string
	Decision string
	// These are the actual selected envelopes, not later publication lookups.
	Consumed map[string]reasonConsumption
}

type reasonPendingEvidence struct {
	Step                                             reasonStep
	Operation, RequestObservation, HelperObservation string
	ResolvedMatch                                    executor.Match
}

type consumerReasonTrace struct {
	RawValid                     bool
	Events                       []model.ConsumerReasonEvent
	Handles                      map[string]string
	Prefix, Component, Guarantee []model.Violation
	Consumptions                 map[string]reasonConsumption
	Classification               *reasonClassificationEvidence
	Terminal                     *reasonStep
	Pending                      *reasonPendingEvidence
}

type reasonOrderedEvent struct {
	order int64
	extra bool
	event model.ConsumerReasonEvent
}

// Authentication runs first, preserving old wrappers. The additional evidence
// observes actual history instructions and never chooses or commits a result.
func decodeConsumerReason(s runtime.Inspection, c model.JoinConfig) (consumerReasonTrace, error) {
	var orders []int64
	raw, err := decodeInvocationJoinRawOrdered(s, c, &orders)
	if err != nil {
		return consumerReasonTrace{}, err
	}
	out := consumerReasonTrace{Handles: raw.Handles, Consumptions: map[string]reasonConsumption{}}
	fail := func(message string) (consumerReasonTrace, error) {
		return consumerReasonTrace{}, fmt.Errorf("reason evidence: %s", message)
	}
	programs := map[string]executor.Program{}
	runs := map[runtime.RunID]evidenceRun{}
	histories := map[string]evidenceHistory{}
	observations := map[string]executor.Observation{}
	facts := []invocationJoinFact{}
	for _, row := range s.Tables["programs"] {
		p, e := readEvidence[executor.Program](row, "bytes")
		if e != nil {
			return out, e
		}
		programs[row["id"].(string)] = p
	}
	for _, row := range s.Tables["runs"] {
		id, e := evidenceID(row)
		if e != nil {
			return out, e
		}
		seq, e := evidenceNumber(row["sequence"])
		if e != nil {
			return out, e
		}
		initial, e := readEvidence[runtime.Projection](row, "initial")
		if e != nil {
			return out, e
		}
		current, e := readEvidence[runtime.Projection](row, "projection")
		if e != nil {
			return out, e
		}
		runs[id] = evidenceRun{id, row["program_id"].(string), initial, current, seq}
	}
	for _, row := range s.Tables["history"] {
		id, e := evidenceID(row)
		if e != nil {
			return out, e
		}
		seq, e := evidenceNumber(row["sequence"])
		if e != nil {
			return out, e
		}
		post, e := readEvidence[runtime.Projection](row, "successor")
		if e != nil {
			return out, e
		}
		intents, e := readEvidence[[]intent](row, "intents")
		if e != nil {
			return out, e
		}
		selected, _ := row["selected_observation"].(string)
		histories[evidenceKey(id, seq)] = evidenceHistory{id, seq, selected, post, intents}
	}
	for _, row := range s.Tables["observations"] {
		o, e := readEvidence[executor.Observation](row, "descriptor")
		if e != nil {
			return out, e
		}
		observations[o.ID] = o
	}
	for _, row := range s.Tables["composition_facts"] {
		f, e := readEvidence[invocationJoinFact](row, "descriptor")
		if e != nil {
			return out, e
		}
		facts = append(facts, f)
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Order < facts[j].Order })
	root := runtime.RunID{Subject: "root", Generation: 1}
	states := map[runtime.RunID]runtime.Projection{}
	for id, r := range runs {
		states[id] = r.Initial
	}
	ordered := []reasonOrderedEvent{}
	for i, event := range raw.Events {
		ordered = append(ordered, reasonOrderedEvent{order: orders[i], event: model.ConsumerReasonEvent{JoinEvent: event}})
	}
	add := func(order int64, f model.Observation, tag string) {
		ordered = append(ordered, reasonOrderedEvent{order: order, extra: true, event: model.ConsumerReasonEvent{JoinEvent: model.JoinEvent{Fact: f}, Classification: tag}})
	}
	selectedByOperation := map[string]reasonConsumption{}
	for _, f := range facts {
		if f.Kind != "execution" {
			continue
		}
		h := histories[evidenceKey(f.Run, f.Sequence)]
		pre := states[f.Run]
		p := programs[runs[f.Run].Program]
		in := p.Code[pre.PC]
		step := reasonStep{Run: f.Run, Program: runs[f.Run].Program, Sequence: f.Sequence, Order: f.Order, PC: pre.PC, NextPC: h.Successor.PC}
		if h.Selected != "" {
			o := observations[h.Selected]
			if o.Kind == "invocation" {
				consumed := reasonConsumption{Step: step, Observation: o, Bind: in.Bind}
				selectedByOperation[o.Operation] = consumed
				if f.Run == root {
					for _, role := range []string{"e", "s"} {
						if o.Operation == out.Handles[role] {
							out.Consumptions[role] = consumed
						}
					}
				}
			}
		}
		if f.Run == root {
			// Both selected envelope registers must remain the original observed values.
			for _, selected := range out.Consumptions {
				for field, dst := range selected.Bind {
					v, present := selected.Observation.Payload[field]
					if present && !reflect.DeepEqual(h.Successor.Data[dst], v) {
						return fail("selected outcome register overwritten")
					}
					if !present {
						if _, exists := h.Successor.Data[dst]; exists {
							return fail("absent outcome payload register invented")
						}
					}
				}
			}
			if in.Op == "set" && in.Dst == "classification" {
				if out.Classification != nil || len(out.Consumptions) != 2 || out.Consumptions["s"].Step.Sequence >= out.Consumptions["e"].Step.Sequence || out.Consumptions["e"].Step.Sequence >= f.Sequence {
					return fail("classification lacks both earlier original consumptions")
				}
				tag := h.Successor.Data["classification"]
				if tag.Kind != "string" || in.A.Literal == nil || !reflect.DeepEqual(*in.A.Literal, tag) || h.Successor.PC != pre.PC+1 || p.Code[h.Successor.PC].Op != "end" {
					return fail("classification not an actual literal Set reaching End")
				}
				consumed := map[string]reasonConsumption{"e": out.Consumptions["e"], "s": out.Consumptions["s"]}
				out.Classification = &reasonClassificationEvidence{Step: step, Tag: tag.Text, Decision: p.Code[h.Successor.PC].Outcome, Consumed: consumed}
				add(f.Order, model.Observation{Kind: "parent_classify", Operation: "root", Role: "parent", Request: c.Request}, tag.Text)
			}
			if h.Successor.Status == "open" && p.Code[h.Successor.PC].Op == "end" {
				if out.Classification == nil || out.Classification.Step != step {
					return fail("decision lacks its same-commit classification")
				}
			}
			if in.Op == "end" {
				if out.Classification == nil || out.Terminal != nil || out.Classification.Step.Sequence+1 != f.Sequence || pre.PC != out.Classification.Step.NextPC || h.Successor.Data["classification"].Text != out.Classification.Tag {
					return fail("End differs from classified terminal path")
				}
				saved := step
				out.Terminal = &saved
			}
		}
		states[f.Run] = h.Successor
	}
	// The sidecar recognizes only the two byte-identified original pending paths.
	// It does not project the request mode to a fake outcome or a fake receive.
	for _, mapping := range []struct {
		program      string
		branch, wait int
		helper       bool
	}{
		{"74152796cc6bb845eee44cc19813210a85ee26173629dd76e446fd042a85d5b7", 9, 11, false},
		{"ca5ab7efdb73587e2664e02f9cea2928fde178190768aa844a2c88e14c90aa67", 10, 12, true},
	} {
		op := out.Handles["e"]
		arow := communicationRow(s, "composition_admission_decisions", "operation_id", op)
		if arow == nil {
			continue
		}
		admission, e := readEvidence[runtime.InvocationAdmission](arow, "descriptor")
		if e != nil {
			return out, e
		}
		if !admission.Accepted || len(admission.Members) != 1 {
			continue
		}
		member := admission.Members[0]
		r := runs[member]
		if r.Program != mapping.program || r.Current.Status != "open" || r.Current.PC != mapping.wait {
			continue
		}
		p := programs[r.Program]
		var brow map[string]any
		for _, row := range s.Tables["composition_bindings"] {
			binding, err := readEvidence[invocationJoinBinding](row, "descriptor")
			if err != nil {
				return out, err
			}
			if binding.OperationID == op {
				brow = row
			}
		}
		if brow == nil {
			return fail("pending binding absent")
		}
		binding, e := readEvidence[invocationJoinBinding](brow, "descriptor")
		if e != nil {
			return out, e
		}
		requestRow := communicationRow(s, "composition_requests", "invocation_id", admission.Invocation.ID)
		if requestRow == nil {
			return fail("pending original request absent")
		}
		requestID := requestRow["observation_id"].(string)
		request := observations[requestID]
		first := histories[evidenceKey(member, 1)]
		if first.Selected != requestID || request.Operation != op || request.Run != member || request.Source != binding.Caller.Subject || request.SourceGeneration != binding.Caller.Generation {
			return fail("pending original request not selected")
		}
		pre := r.Initial
		var arrival *reasonStep
		for seq := int64(1); seq <= r.Sequence; seq++ {
			h := histories[evidenceKey(member, seq)]
			in := p.Code[pre.PC]
			if seq == 1 {
				want := schema.CloneValues(pre.Data)
				for field, dst := range in.Bind {
					if v, ok := request.Payload[field]; ok {
						want[dst] = v
					}
				}
				if in.Op != "receive" || h.Successor.PC != 1 || h.Successor.Status != "open" || len(h.Intents) != 0 || !sameValues(want, h.Successor.Data) {
					return fail("pending first request transition")
				}
			} else if e := invocationJoinParentStep(pre, h.Successor, in, h.Selected, observations[h.Selected], h.Intents); e != nil {
				return out, fmt.Errorf("pending candidate path: %w", e)
			}
			if pre.PC == mapping.branch && h.Successor.PC == mapping.wait {
				if arrival != nil || seq != r.Sequence || in.Op != "branch" || in.Otherwise != mapping.wait || h.Selected != "" {
					return fail("pending branch arrival differs")
				}
				for _, f := range facts {
					if f.Kind == "execution" && f.Run == member && f.Sequence == seq {
						v := reasonStep{member, r.Program, seq, f.Order, pre.PC, h.Successor.PC}
						arrival = &v
					}
				}
			}
			pre = h.Successor
		}
		if arrival == nil {
			return fail("pending lacks authenticated branch arrival")
		}
		original := p.Code[mapping.wait].Match
		if original == nil || !reflect.DeepEqual(original, p.Code[0].Match) || original.Kind != "message" || original.Provenance != "ordinary-message" || original.Source.Literal == nil || original.Source.Literal.Text != "caller" || original.Generation.Literal == nil || original.Generation.Literal.Number != 1 || original.Operation.Literal == nil || original.Operation.Literal.Text != "" {
			return fail("pending original receive match")
		}
		resolved := *original
		resolved.Source = executor.Lit(schema.Text(binding.Caller.Subject))
		resolved.Generation = executor.Lit(schema.Int(binding.Caller.Generation))
		resolved.Operation = executor.Lit(schema.Text(op))
		resolved.Provenance = "runtime-invocation-request"
		for _, row := range s.Tables["observations"] {
			o := observations[row["id"].(string)]
			if row["consumed_sequence"] == nil && o.Run == member && o.Kind == resolved.Kind && o.Provenance == resolved.Provenance && o.Source == resolved.Source.Literal.Text && o.SourceGeneration == resolved.Generation.Literal.Number && o.Operation == op && o.Obligation == resolved.Obligation.Literal.Text {
				return fail("pending has a matching release observation")
			}
		}
		helperID := ""
		if mapping.helper {
			helper, ok := selectedByOperation[out.Handles["h"]]
			if !ok || helper.Step.Run != member || helper.Step.PC != 2 || helper.Step.Order >= arrival.Order || helper.Observation.Payload["status"].Text != "succeeded" || helper.Observation.Payload["reason"].Text != "member_succeeded" || !helper.Observation.Payload["response_present"].Flag || helper.Observation.Payload["response.id"].Number != int64(c.Request.ID) || helper.Observation.Payload["response.value"].Number != int64(1-c.Request.Value) {
				return fail("pending missing original successful helper await")
			}
			helperID = helper.Observation.ID
		}
		if communicationRow(s, "composition_publications", "operation_id", op) != nil || out.Consumptions["e"].Observation.ID != "" {
			return fail("pending candidate already published/consumed")
		}
		out.Pending = &reasonPendingEvidence{*arrival, op, requestID, helperID, resolved}
		for _, kind := range []string{"private_step", "pending_receive"} {
			add(arrival.Order, model.Observation{Kind: kind, Operation: "e", Parent: "root", Role: "candidate", Request: c.Request}, "")
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].order != ordered[j].order {
			return ordered[i].order < ordered[j].order
		}
		return ordered[i].extra && !ordered[j].extra
	})
	for _, item := range ordered {
		out.Events = append(out.Events, item.event)
	}
	out.RawValid = true
	out.Prefix = model.CheckConsumerReasonPrefix(c, out.Events)
	out.Component = model.CheckConsumerReasonComponent(c, out.Events)
	out.Guarantee = model.CheckConsumerReasonGuarantee(c, out.Events)
	return out, nil
}
