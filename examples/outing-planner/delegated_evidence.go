package main

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type delegatedReport struct {
	HelperAdmission int64                     `json:"helper_admission"`
	HelperTemplate  string                    `json:"helper_template_id"`
	OperationID     string                    `json:"invocation_operation_id,omitempty"`
	InvocationID    string                    `json:"invocation_id,omitempty"`
	HelperRun       *runtime.RunID            `json:"helper_run,omitempty"`
	HelperStatus    string                    `json:"helper_status,omitempty"`
	Invocations     []*runtime.InvocationView `json:"invocations"`
	Roots           []runtime.RootCompletion  `json:"roots"`
}

func reportDelegation(s *runtime.ExecutionSnapshot, info retainedDeploymentInfo) *delegatedReport {
	result := &delegatedReport{
		HelperAdmission: info.Options.HelperAdmission, HelperTemplate: info.HelperTemplateID, HelperRun: info.HelperRun,
		Invocations: append([]*runtime.InvocationView{}, s.Invocations...), Roots: s.Roots,
	}
	if info.HelperRun != nil {
		run, _ := s.Run(*info.HelperRun)
		result.HelperStatus = run.Projection.Status
	}
	if len(result.Invocations) == 1 {
		call := result.Invocations[0]
		result.OperationID = call.OperationID
		if call.Admission != nil {
			result.InvocationID = call.Admission.Invocation.ID
		}
	}
	return result
}

// Public inspection authenticates the retained facts. Check that the complete
// detached view accounts for the fixed application protocol and its original
// operations, including at partial cuts.
type delegatedCut struct {
	*runtime.ExecutionSnapshot
	observations map[string]*runtime.ObservationView
	admissions   map[string]*runtime.AdmissionView
	operations   map[string]*runtime.Intent
	messages     map[string]*runtime.Intent
	noticed      map[string]bool
	admitted     map[string]bool
}

func verifyDelegatedEvidence(s *runtime.ExecutionSnapshot, info retainedDeploymentInfo) error {
	cut := delegatedCut{
		ExecutionSnapshot: s, observations: make(map[string]*runtime.ObservationView, len(s.Observations)),
		admissions: make(map[string]*runtime.AdmissionView, len(s.Admissions)),
		operations: make(map[string]*runtime.Intent, len(s.Operations)), messages: make(map[string]*runtime.Intent, 4),
		noticed: make(map[string]bool, len(s.Observations)), admitted: make(map[string]bool, len(s.Admissions)),
	}
	if err := cut.index(); err != nil {
		return err
	}
	weather, _ := s.Run(weatherRun)
	config := info.Configuration
	request, err := frozenHelperRequest(config, weather.Initial.Data)
	if err != nil {
		return err
	}
	proposal := map[string]schema.Value{proposalID: request[proposalID], candidates: request[candidates]}
	p := cut.messages[proposalPort]
	if p != nil && !equalPayload(p.Payload, proposal) {
		return fmt.Errorf("retained delegated proposal differs from frozen candidates")
	}
	proposalSelection, err := cut.messageSelection(p, weatherRun)
	if err != nil {
		return err
	}

	var call *runtime.InvocationView
	var checked []map[string]schema.Value
	if len(s.Invocations) == 1 {
		call = s.Invocations[0]
		if proposalSelection == nil || proposalSelection.Sequence >= call.Sequence || !equalPayload(call.Inputs, request) {
			return fmt.Errorf("delegated invocation lacks its original prior proposal selection")
		}
		if a := call.Authorization; a != nil && !cut.authorized(a, cut.operations[call.OperationID]) {
			return fmt.Errorf("delegated invocation lacks its original authorization")
		}
		if info.HelperRun == nil && (call.Freeze != nil || len(call.Replies) != 0) {
			return fmt.Errorf("delegated invocation without a member contains helper work")
		}
		if info.HelperRun != nil {
			checked, err = cut.helperResponse(config, request, call, *info.HelperRun)
			if err != nil {
				return err
			}
		}
		if err := cut.checkPublication(call); err != nil {
			return err
		}
	}
	assessment := cut.messages[assessmentPort]
	if assessment != nil {
		if call == nil || call.Outcome == nil || call.Outcome.Status != "succeeded" || !call.Outcome.ResponsePresent ||
			call.SelectedAt == nil || call.SelectedAt.Sequence >= assessment.Sequence ||
			!equalPayload(assessment.Payload, call.Outcome.Response) {
			return fmt.Errorf("delegated assessment precedes or differs from the original selected helper outcome")
		}
	}
	assessmentSelection, err := cut.messageSelection(assessment, outingRun)
	if err != nil {
		return err
	}
	decision := cut.messages[decisionPort]
	decisionSelection, err := cut.messageSelection(decision, weatherRun)
	if err != nil {
		return err
	}
	outing, _ := s.Run(outingRun)
	if decision != nil {
		if assessmentSelection == nil || assessmentSelection.Sequence >= decision.Sequence {
			return fmt.Errorf("delegated decision precedes Outing's original assessment selection")
		}
		if weather.Projection.Status != "open" && (decisionSelection == nil || decisionSelection.Sequence >= weather.Sequence) {
			return fmt.Errorf("terminal Weather lacks its prior original decision selection")
		}
		if err := checkDecision(s, decision, proposal, checked); err != nil {
			return err
		}
	} else if outing.Projection.Status != "open" || weather.Projection.Status != "open" && !selectedHelperFailure(call, weather) {
		return fmt.Errorf("delegated caller terminated without consuming its required evidence")
	}

	if len(cut.noticed) != len(s.Observations) || len(cut.admitted) != len(s.Admissions) {
		return fmt.Errorf("unexpected delegated observation or transport admission")
	}
	return cut.checkRootsAndPending(info, call)
}

func selectedHelperFailure(call *runtime.InvocationView, weather *runtime.ExecutionRun) bool {
	return weather.Projection.Status == "failed" && call != nil && call.SelectedAt != nil &&
		call.SelectedAt.Sequence < weather.Sequence && call.Outcome != nil && call.Outcome.Status != "succeeded"
}

func (c *delegatedCut) index() error {
	for i := range c.Operations {
		op := &c.Operations[i]
		if original, ok := c.Operation(op.ID); !ok || !reflect.DeepEqual(op, original) {
			return fmt.Errorf("delegated operation %q differs from its indexed original", op.ID)
		}
		if c.operations[op.ID] != nil {
			return fmt.Errorf("duplicate delegated operation %q", op.ID)
		}
		c.operations[op.ID] = op
		if op.Kind == "message" {
			c.messages[op.Obligation] = op
		}
	}
	for _, o := range c.Observations {
		if o == nil || o.ID == "" || c.observations[o.ID] != nil {
			return fmt.Errorf("missing or duplicate delegated observation")
		}
		if original, ok := c.Observation(o.ID); !ok || !reflect.DeepEqual(o, original) {
			return fmt.Errorf("delegated observation %q differs from its indexed original", o.ID)
		}
		c.observations[o.ID] = o
	}
	for _, a := range c.Admissions {
		if a == nil || a.ID == "" || c.admissions[a.ID] != nil {
			return fmt.Errorf("missing or duplicate delegated admission")
		}
		c.admissions[a.ID] = a
	}
	// History identifies the complete operation and selection set, independent
	// of the effect and invocation views and of the exported pending list.
	seenOps, seenSelections := make(map[string]bool), make(map[string]bool)
	for offset := 0; ; {
		page, err := c.History(offset, runtime.MaxHistoryPage)
		if err != nil {
			return err
		}
		for _, step := range page.Steps {
			if id := step.SelectedObservationID; id != "" {
				o := c.observations[id]
				if o == nil || seenSelections[id] || o.SelectedAt == nil || *o.SelectedAt != step.ReductionRef || o.Run != step.Run {
					return fmt.Errorf("delegated observation %q disagrees with its selection run or history", id)
				}
				seenSelections[id] = true
			}
			for _, id := range step.OperationIDs {
				op := c.operations[id]
				if op == nil || seenOps[id] || op.Run != step.Run || op.Sequence != step.Sequence {
					return fmt.Errorf("delegated operation %q disagrees with its original commit", id)
				}
				seenOps[id] = true
			}
		}
		if !page.More {
			break
		}
		offset = page.Next
	}
	if len(seenOps) != len(c.Operations) {
		return fmt.Errorf("delegated operation has no original commit")
	}
	for _, o := range c.Observations {
		if o.SelectedAt != nil && !seenSelections[o.ID] {
			return fmt.Errorf("delegated observation %q has no original selection", o.ID)
		}
	}
	return nil
}

func (c *delegatedCut) authorized(auth *runtime.Authorization, op *runtime.Intent) bool {
	if auth == nil || op == nil || !auth.Allowed || auth.Reason != "authorized" || auth.IntentID != op.ID ||
		auth.Source.Run != op.Run || auth.Source.Sequence < op.Sequence || auth.Source.Revision != 1 || auth.ClosureOperation != "" {
		return false
	}
	step, ok := c.Reduction(runtime.ReductionRef{Run: auth.Source.Run, Sequence: auth.Source.Sequence})
	boundReply := op.Kind == "message" && op.Obligation == helperObligation
	return ok && step.Epoch <= auth.Source.Epoch && (step.Status == "open" || boundReply)
}

func (c *delegatedCut) messageSelection(op *runtime.Intent, target runtime.RunID) (*runtime.ReductionRef, error) {
	if op == nil {
		return nil, nil
	}
	a, err := c.transport(op, "message", target)
	if err != nil || a == nil || !a.Accepted {
		return nil, err
	}
	o := c.observations[a.ObservationID]
	if o == nil || o.Kind != "message" || o.Origin != "message" || o.Run != target ||
		o.SourceOperationID != op.ID || o.Source != op.Run.Subject || o.SourceGeneration != op.Run.Generation ||
		o.Provenance != "ordinary-message" || o.Obligation != op.Obligation || o.Operation != "" ||
		o.ProducerRun == nil || *o.ProducerRun != op.Run ||
		o.ProducerReduction == nil || *o.ProducerReduction != (runtime.ReductionRef{Run: op.Run, Sequence: op.Sequence}) ||
		!equalPayload(o.Payload, op.Payload) || !slices.Equal(o.Admissions, []string{a.ID}) {
		return nil, fmt.Errorf("delegated message %q lacks its original delivery", op.ID)
	}
	c.noticed[o.ID] = true
	return o.SelectedAt, nil
}

func (c *delegatedCut) transport(op *runtime.Intent, domain string, target runtime.RunID) (*runtime.AdmissionView, error) {
	var result *runtime.AdmissionView
	for _, a := range c.Admissions {
		if a.SourceOperationID != op.ID {
			continue
		}
		if result != nil || a.Domain != domain || a.Obligation != op.Obligation || a.Target != target.Subject ||
			(a.Bound != nil && *a.Bound != target) || (a.Accepted && a.Bound == nil) {
			return nil, fmt.Errorf("unexpected transport for delegated operation %q", op.ID)
		}
		if domain == "message" && (a.Source != op.Run || a.RequestedGeneration != target.Generation || a.LogicalID != op.ID || a.Correlation != "") {
			return nil, fmt.Errorf("delegated message %q transport disagrees with its original route", op.ID)
		}
		result = a
		c.admitted[a.ID] = true
	}
	return result, nil
}

func (c *delegatedCut) helperResponse(
	config configuration, request map[string]schema.Value, call *runtime.InvocationView, helper runtime.RunID,
) ([]map[string]schema.Value, error) {
	o := c.observations[call.RequestObservationID]
	if o == nil || o.Run != helper || o.Kind != "message" || o.Origin != "invocation_request" ||
		o.Provenance != "runtime-invocation-request" || !equalPayload(o.Payload, request) ||
		o.ProducerRun == nil || *o.ProducerRun != weatherRun || o.ProducerReduction == nil || *o.ProducerReduction != (runtime.ReductionRef{Run: weatherRun, Sequence: call.Sequence}) {
		return nil, fmt.Errorf("delegated helper lacks its original request observation")
	}
	c.noticed[o.ID] = true
	previous := o.SelectedAt
	rows, err := request[candidates].Records()
	if err != nil {
		return nil, err
	}
	adapter, err := openmeteo.NewHourly(config.Endpoint)
	if err != nil {
		return nil, err
	}
	forecasts := slices.Clone(c.Effects)
	slices.SortFunc(forecasts, func(a, b *runtime.ExecutionEffect) int { return cmp.Compare(a.Sequence, b.Sequence) })
	checked := make([]map[string]schema.Value, 0, len(rows))
	for i, effect := range forecasts {
		if previous == nil || previous.Sequence >= effect.Sequence {
			return nil, fmt.Errorf("forecast %q precedes its original helper selection", effect.OperationID)
		}
		if err := c.checkForecastChain(effect, helper); err != nil {
			return nil, err
		}
		row, err := checkForecast(c.ExecutionSnapshot, config, adapter, effect, rows[i], helper)
		if err != nil {
			return nil, err
		}
		previous = nil
		if effect.Settlement != nil {
			previous = c.observations[effect.Settlement.ObservationID].SelectedAt
		}
		if row != nil {
			checked = append(checked, row)
		}
	}
	reply := c.messages[helperObligation]
	if reply != nil {
		expected := map[string]schema.Value{proposalID: request[proposalID], assessments: schema.List(checked...)}
		if previous == nil || previous.Sequence >= reply.Sequence || len(checked) != len(rows) || !equalPayload(reply.Payload, expected) {
			return nil, fmt.Errorf("delegated helper reply lacks its complete original selected forecast evidence")
		}
	}
	if err := c.checkReply(call, helper, reply, forecasts); err != nil {
		return nil, err
	}
	return checked, nil
}

func (c *delegatedCut) checkForecastChain(e *runtime.ExecutionEffect, helper runtime.RunID) error {
	op := c.operations[e.OperationID]
	if e.SourceRevision != 1 || e.ExternalAttempt != nil || e.ExternalReceipt != nil || len(e.AuthorityAdmissions) != 0 ||
		len(e.ReadOpportunities) != 0 || len(e.ReadsOfOriginal) != 0 {
		return fmt.Errorf("unsupported delegated forecast protocol %q", e.OperationID)
	}
	a, err := c.transport(op, "adapter", helper)
	if err != nil {
		return err
	}
	if auth := e.Authorization; auth != nil && !c.authorized(auth, op) {
		return fmt.Errorf("forecast %q authorization differs from the original operation", op.ID)
	}
	if attempt := e.AdapterAttempt; attempt != nil && (e.Authorization == nil || !reflect.DeepEqual(attempt.Intent, *op) ||
		attempt.AuthorizationID != e.Authorization.ID || attempt.Authority != nil) {
		return fmt.Errorf("forecast %q has a different original attempt", op.ID)
	}
	if dispatch := e.DispatchAdmission; dispatch != nil && (e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed ||
		dispatch.OperationID != op.ID || dispatch.PreparedDigest != e.AdapterAttempt.Prepared.Digest) {
		return fmt.Errorf("forecast %q has a different original dispatch", op.ID)
	}
	if result := e.AdapterResult; result != nil {
		attempt := e.AdapterAttempt
		if attempt == nil || result.Candidate.ID != op.ID || result.Candidate.Key != attempt.Prepared.Request.Key ||
			result.Read != nil || result.Reconciled != nil || result.RefusalWitness != "" ||
			(attempt.Allowed && e.DispatchAdmission == nil) {
			return fmt.Errorf("forecast %q has a different original result", op.ID)
		}
		values, disposition := effects.Result(attempt.Prepared.Request.Output.Record, result.Candidate)
		if result.Disposition != disposition || e.ResultDisposition != disposition || !equalPayload(result.Values, values) {
			return fmt.Errorf("forecast %q changed its original result values or disposition", op.ID)
		}
	} else if e.ResultDisposition != "" {
		return fmt.Errorf("forecast %q claims a disposition without its original result", op.ID)
	}
	if e.Settlement == nil {
		if a != nil || e.State == runtime.EffectSettled {
			return fmt.Errorf("forecast %q lacks original settlement", op.ID)
		}
		return nil
	}
	o := c.observations[e.Settlement.ObservationID]
	if o == nil || o.Kind != "adapter" || o.Origin != "adapter" || o.Provenance != "runtime-adapter" ||
		o.Source != capability || o.Obligation != effectName || e.State != runtime.EffectSettled || e.AdapterResult == nil ||
		a == nil || !a.Accepted || a.ObservationID != o.ID || !slices.Equal(o.Admissions, []string{a.ID}) ||
		!equalPayload(o.Payload, e.AdapterResult.Values) {
		return fmt.Errorf("forecast %q settlement does not carry its original result", op.ID)
	}
	c.noticed[o.ID] = true
	return nil
}

func (c *delegatedCut) checkReply(call *runtime.InvocationView, helper runtime.RunID, reply *runtime.Intent, forecasts []*runtime.ExecutionEffect) error {
	var replyIDs []string
	if reply != nil {
		replyIDs = append(replyIDs, reply.ID)
	}
	switch len(call.Replies) {
	case 0:
		if reply != nil {
			if a, err := c.transport(reply, "message", weatherRun); err != nil || a != nil {
				return fmt.Errorf("delegated helper transport lacks its original reply record")
			}
		}
	case 1:
		r := call.Replies[0]
		if reply == nil || r.InvocationID != call.Admission.Invocation.ID || r.RootID != call.Admission.Invocation.RootID ||
			!reflect.DeepEqual(r.Intent, *reply) || !c.authorized(&r.Authorization, reply) || !r.Accepted || r.Reason != "recorded_reply" {
			return fmt.Errorf("delegated helper reply differs from its original authorized operation")
		}
		a, err := c.transport(reply, "message", weatherRun)
		if err != nil || a == nil || !a.Accepted || a.ObservationID != "" || a.Reason != r.Reason {
			return fmt.Errorf("delegated helper reply lacks its original bound admission")
		}
	default:
		return fmt.Errorf("extra delegated helper reply")
	}
	run, _ := c.Run(helper)
	if f := call.Freeze; f != nil {
		effectIDs := make([]string, 0, len(forecasts))
		for _, e := range forecasts {
			effectIDs = append(effectIDs, e.OperationID)
		}
		slices.Sort(effectIDs)
		if f.InvocationID != call.Admission.Invocation.ID || f.Run != helper || f.Sequence != run.Sequence ||
			f.Status == "open" || f.Status != run.Projection.Status || len(f.InvocationIntents) != 0 ||
			!slices.Equal(f.ReplyIntents, replyIDs) || !slices.Equal(f.ExternalIntents, effectIDs) ||
			(reply != nil && reply.Sequence >= f.Sequence) {
			return fmt.Errorf("delegated helper freeze lacks its original terminal intent prefix")
		}
	} else if run.Projection.Status != "open" {
		return fmt.Errorf("terminal delegated helper lacks its original freeze")
	}
	return nil
}

func (c *delegatedCut) checkPublication(call *runtime.InvocationView) error {
	out := call.Outcome
	if call.Closure != nil {
		return fmt.Errorf("unexpected delegated invocation closure")
	}
	if out == nil {
		if call.SelectedAt != nil || call.OutcomeObservationID != "" || call.ResponseOperationID != "" {
			return fmt.Errorf("delegated published evidence without original outcome")
		}
		return nil
	}
	admission := call.Admission
	if admission == nil || out.OperationID != call.OperationID || out.InvocationID != admission.Invocation.ID ||
		out.TemplateID != call.TemplateID || out.Caller != weatherRun || out.DecisionOrder <= 0 || len(call.Dependencies) != 0 {
		return fmt.Errorf("delegated publication differs from its original invocation")
	}
	if admission.Accepted {
		f := call.Freeze
		if f == nil || f.DecisionOrder >= out.DecisionOrder || len(call.Replies) != len(f.ReplyIntents) {
			return fmt.Errorf("delegated publication precedes the original terminal freeze or reply admission")
		}
		if out.ResponsePresent {
			if len(call.Replies) != 1 || call.ResponseOperationID != call.Replies[0].Intent.ID ||
				!equalPayload(out.Response, call.Replies[0].Intent.Payload) ||
				out.Status != f.Status || out.Reason != "member_"+f.Status {
				return fmt.Errorf("delegated publication changes the original recorded helper reply")
			}
		} else {
			reason := "member_" + f.Status
			if f.Status == "succeeded" {
				reason = "missing_reply"
			}
			if len(call.Replies) != 0 || out.Status != "failed" || out.Reason != reason {
				return fmt.Errorf("delegated outcome lacks a valid original failure")
			}
		}
	} else if out.Status != "rejected" || out.Reason != admission.Reason || out.ResponsePresent ||
		call.Freeze != nil || len(call.Replies) != 0 {
		return fmt.Errorf("delegated publication differs from the original invocation refusal")
	}
	if !out.ResponsePresent && (len(out.Response) != 0 || call.ResponseOperationID != "") {
		return fmt.Errorf("delegated absent response carries original reply fields")
	}
	o := c.observations[call.OutcomeObservationID]
	payload := map[string]schema.Value{
		"status": schema.Text(out.Status), "reason": schema.Text(out.Reason), "response_present": schema.Bool(out.ResponsePresent),
	}
	for key, v := range out.Response {
		payload["response."+key] = v
	}
	if o == nil || o.Run != weatherRun || o.Kind != "invocation" || o.Origin != "invocation_outcome" || o.Provenance != "runtime-invocation" ||
		o.Source != call.TemplateID || o.SourceGeneration != 1 || o.Obligation != helperObligation ||
		o.SourceOperationID != call.OperationID || o.Operation != call.OperationID || !equalPayload(o.Payload, payload) ||
		!reflect.DeepEqual(o.SelectedAt, call.SelectedAt) || (o.SelectedAt != nil && o.SelectedAt.Sequence <= call.Sequence) {
		return fmt.Errorf("delegated outcome observation differs from the original published result")
	}
	c.noticed[o.ID] = true
	return nil
}
