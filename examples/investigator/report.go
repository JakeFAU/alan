package main

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type finding struct {
	Role           string   `json:"role"`
	Verdict        string   `json:"model_verdict"`
	Interpretation string   `json:"model_interpretation"`
	Citation       citation `json:"verified_source"`
}

type openQuestion struct {
	Role     string `json:"role"`
	Reason   string `json:"reason"`
	Question string `json:"question"`
}

type workerReport struct {
	Role             string         `json:"role"`
	Query            string         `json:"query"`
	OperationID      string         `json:"invocation_operation_id,omitempty"`
	InvocationID     string         `json:"invocation_id,omitempty"`
	Admission        string         `json:"invocation_admission,omitempty"`
	Run              *runtime.RunID `json:"run,omitempty"`
	LocalStatus      string         `json:"local_status,omitempty"`
	OutcomeID        string         `json:"outcome_id,omitempty"`
	InvocationStatus string         `json:"invocation_status,omitempty"`
	InvocationReason string         `json:"invocation_reason,omitempty"`
	ResponsePresent  *bool          `json:"response_present,omitempty"`
	SelectedAt       int64          `json:"parent_selected_sequence,omitempty"`
	Assessment       string         `json:"parent_assessment,omitempty"`
	Lookup           effectReport   `json:"lookup"`
	Model            effectReport   `json:"model"`
	outcome          *runtime.InvocationOutcome
}

type investigationReport struct {
	Question           string         `json:"question"`
	Status             string         `json:"status"`
	ParentStatus       string         `json:"parent_local_status"`
	RootStatus         string         `json:"root_outcome,omitempty"`
	PendingReason      string         `json:"root_pending_reason,omitempty"`
	PendingInvocations []string       `json:"pending_invocations"`
	UnresolvedEffects  []string       `json:"unresolved_effects"`
	Workers            []workerReport `json:"workers"`
	Findings           []finding      `json:"findings"`
	Unresolved         []openQuestion `json:"unresolved_questions"`
	Accounting         accounting     `json:"accounting"`
}

// Start and End are byte offsets, with End excluded.
type citation struct {
	Document string `json:"document"`
	Source   string `json:"source,omitempty"`
	Version  string `json:"sha256"`
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
	Quote    string `json:"quote"`
}

type reportedUsage struct {
	InputTokens  *int64 `json:"input_tokens"`
	OutputTokens *int64 `json:"output_tokens"`
	TotalTokens  *int64 `json:"total_tokens"`
}

type effectReport struct {
	OperationID          string         `json:"operation_id,omitempty"`
	State                string         `json:"state"`
	ReservationReason    string         `json:"reservation_reason,omitempty"`
	Disposition          string         `json:"disposition,omitempty"`
	ResponseBodyEvidence string         `json:"response_body_evidence,omitempty"`
	Model                string         `json:"configured_model,omitempty"`
	ReturnedModel        string         `json:"returned_model,omitempty"`
	HTTPStatus           int            `json:"http_status,omitempty"`
	ClientRequestID      string         `json:"client_request_id,omitempty"`
	RequestID            string         `json:"provider_request_id,omitempty"`
	ResponseID           string         `json:"provider_response_id,omitempty"`
	Usage                *reportedUsage `json:"provider_reported_usage"`
	Diagnostic           string         `json:"diagnostic_response,omitempty"`
	effect               *runtime.ExecutionEffect
}

func describeEffect(effect *runtime.ExecutionEffect) effectReport {
	if effect == nil {
		return effectReport{State: "not_requested"}
	}
	e := effectReport{OperationID: effect.OperationID, effect: effect}
	attempt, result := effect.AdapterAttempt, effect.AdapterResult
	e.State = "awaiting_reservation"
	if attempt == nil {
		return e
	}
	e.Model, e.ReservationReason = attempt.Prepared.Model, attempt.Reason
	if attempt.Allowed && attempt.Prepared.Protocol == "openai/responses/v1" {
		e.ClientRequestID = openai.ClientRequestID(attempt.Prepared.Request.Key)
	}
	e.State = "reserved_outcome_unknown"
	if !attempt.Allowed {
		e.State = "dispatch_refused"
	}
	if result == nil {
		return e
	}
	e.Disposition = result.Disposition
	if !attempt.Allowed {
		return e
	}
	c := result.Candidate
	switch result.Disposition {
	case "validated":
		e.State = "retained_validated"
	case invalidInterpretation, invalidQuestion:
		e.State = "local_validation_failed"
	default:
		e.State = "retained_unsuccessful"
	}
	if result.Disposition != "validated" {
		e.Diagnostic = c.Response
	}
	e.ReturnedModel, e.HTTPStatus = c.ReturnedModel, c.HTTPStatus
	e.RequestID, e.ResponseID = c.RequestID, c.ResponseID
	if u := effect.Usage; u != nil {
		e.Usage = &reportedUsage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens}
	}
	return e
}

type bounds struct {
	Minimum int `json:"minimum"`
	Maximum int `json:"maximum"`
}

type accounting struct {
	InvocationRequests   int     `json:"invocation_requests"`
	InvocationsCharged   int64   `json:"invocations_charged"`
	CreatedRuns          int64   `json:"worker_runs_created"`
	EffectRequests       int     `json:"effect_requests"`
	ReservationDecisions int     `json:"effect_reservation_decisions"`
	EffectsCharged       int64   `json:"effect_reservations_charged"`
	LookupExecutions     bounds  `json:"lookup_executions"`
	ModelExecutions      bounds  `json:"model_adapter_executions"`
	ProviderContacts     bounds  `json:"provider_http_contacts"`
	ReservedOutputTokens int     `json:"output_token_allowance_reserved"`
	CostEstimateUSD      *string `json:"cost_estimate_usd"`
}

// One ordinary attempt permits at most one Execute. Its reservation survives
// a crash on either side of the actual call. Only a retained HTTP response
// witnesses provider contact. Individual provider counters can be absent.
func (a *accounting) countEffect(e effectReport, executions *bounds) {
	if e.OperationID == "" {
		return
	}
	a.EffectRequests++
	if e.effect.AdapterAttempt != nil {
		a.ReservationDecisions++
	}
	executions.Minimum += e.effect.Dispatch.Minimum
	// The approved investigator deploys ordinary lookup and Responses
	// adapters, both with a finite execution bound per original operation.
	if max := e.effect.Dispatch.Maximum; max != nil {
		executions.Maximum += *max
	}
}

func (a *accounting) countModel(e effectReport) {
	a.countEffect(e, &a.ModelExecutions)
	if e.effect != nil && e.effect.AdapterAttempt != nil && e.effect.AdapterAttempt.Allowed {
		a.ProviderContacts.Maximum++
		a.ReservedOutputTokens += e.effect.AdapterAttempt.Prepared.MaxOutputTokens
		if e.HTTPStatus > 0 {
			a.ProviderContacts.Minimum++
		}
	}
}

func verifySpan(asset effects.Asset, c citation) error {
	if err := asset.Check(); err != nil {
		return err
	}
	if asset.ID != c.Document || asset.Version != c.Version {
		return errors.New("citation does not identify the retained document version")
	}
	if c.Start < 0 || c.Start >= c.End || c.End > int64(len(asset.Content)) {
		return errors.New("citation outside retained document bytes")
	}
	if !utf8.ValidString(c.Quote) || asset.Content[c.Start:c.End] != c.Quote {
		return errors.New("citation differs from retained UTF-8 bytes")
	}
	return nil
}

// Run, original effect and selection relationships come from one typed cut.
// An answer must never mix a newer parent selection with an older worker result.
func readReport(ctx context.Context, path string, limits runtime.Limits) (investigationReport, error) {
	s, err := runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		return investigationReport{}, err
	}
	approval, err := approvedDeployment(s)
	if err != nil {
		return investigationReport{}, err
	}
	r, err := reportSnapshot(s)
	if err != nil {
		return investigationReport{}, err
	}
	for i := range r.Workers {
		m := &r.Workers[i].Model
		if m.effect == nil || m.effect.AdapterResult == nil || !m.effect.AdapterAttempt.Allowed {
			continue
		}
		m.ResponseBodyEvidence = "consistent"
		if approval.incompleteBodies[m.OperationID] {
			// The retained negative is compatible with the surviving diagnostic.
			// The raw inspector still exposes the adapter's original claims.
			m.ResponseBodyEvidence = "incomplete"
			m.ReturnedModel, m.ResponseID, m.Usage = "", "", nil
		}
	}
	return r, nil
}

func reportSnapshot(s *runtime.ExecutionSnapshot) (investigationReport, error) {
	id := runtime.RunID{Subject: "investigator", Generation: 1}
	if len(s.Roots) != 1 || s.Roots[0].Run != id {
		return investigationReport{}, errors.New("expected one investigator root")
	}
	root := s.Roots[0]
	parent, ok := s.Run(id)
	if !ok {
		return investigationReport{}, errors.New("investigator parent missing from typed snapshot")
	}
	data := parent.Projection.Data
	for _, input := range []string{"question", "primary_query", "counter_query", "settings"} {
		if !data[input].Equal(parent.Initial.Data[input]) {
			return investigationReport{}, fmt.Errorf("investigator input %s differs from its original assignment", input)
		}
	}
	r := investigationReport{
		Question: data["question"].Text, Status: "pending", ParentStatus: root.LocalStatus,
		PendingReason: root.Pending, PendingInvocations: root.PendingInvocations, UnresolvedEffects: root.UnresolvedExternal,
		Findings: []finding{}, Unresolved: []openQuestion{},
		Accounting: accounting{InvocationsCharged: root.Charges.Invocations, CreatedRuns: root.Charges.CreatedRuns, EffectsCharged: root.Charges.Effects},
	}
	for _, role := range []string{"primary", "counter"} {
		worker, err := readWorker(s, root, data, role)
		if err != nil {
			return investigationReport{}, fmt.Errorf("%s worker: %w", role, err)
		}
		r.Workers = append(r.Workers, worker)
		if worker.OperationID != "" {
			r.Accounting.InvocationRequests++
		}
		r.Accounting.countEffect(worker.Lookup, &r.Accounting.LookupExecutions)
		r.Accounting.countModel(worker.Model)
	}
	if root.Outcome == nil {
		return r, nil
	}
	r.RootStatus = root.Outcome.Status
	if r.RootStatus != "succeeded" {
		r.Status = "incomplete"
		r.Unresolved = append(r.Unresolved, openQuestion{Role: "both", Reason: root.Outcome.Reason, Question: r.Question})
		return r, nil
	}
	if err := r.readAnswer(s, data); err != nil {
		return investigationReport{}, fmt.Errorf("verify authored answer: %w", err)
	}
	return r, nil
}

func (r *investigationReport) readAnswer(s *runtime.ExecutionSnapshot, data map[string]schema.Value) error {
	rows, err := data["findings"].Records()
	if err != nil {
		return err
	}
	for _, w := range r.Workers {
		if w.SelectedAt == 0 || w.outcome == nil {
			return fmt.Errorf("parent did not consume %s outcome", w.Role)
		}
		if err := w.verifyReply(s); err != nil {
			return fmt.Errorf("%s reply: %w", w.Role, err)
		}
		reason, question := w.assessment(r.Question)
		accepted := reason == "validated"
		if w.Assessment != reason || !data[w.Role+"_accepted"].Equal(schema.Bool(accepted)) ||
			!data[w.Role+"_question"].Equal(schema.Text(question)) {
			return fmt.Errorf("%s assessment differs from the selected outcome", w.Role)
		}
		if !accepted {
			r.Unresolved = append(r.Unresolved, openQuestion{Role: w.Role, Reason: reason, Question: question})
			continue
		}
		i := len(r.Findings)
		if i >= len(rows) || rows[i]["role"].Text != w.Role {
			return fmt.Errorf("missing or out-of-order finding for %s", w.Role)
		}
		f, err := w.verifyFinding(s, rows[i], r.Question)
		if err != nil {
			return fmt.Errorf("%s finding: %w", w.Role, err)
		}
		r.Findings = append(r.Findings, f)
		if w.outcome.Response["unresolved"].Text != "" {
			r.Unresolved = append(r.Unresolved, openQuestion{Role: w.Role, Reason: "model_question", Question: question})
		}
	}
	if len(rows) != len(r.Findings) {
		return errors.New("authored answer contains a finding without an accepted worker")
	}
	switch len(r.Findings) {
	case 0:
		r.Status = "incomplete"
	case 1:
		r.Status = "partial"
	case 2:
		r.Status = "completed"
		if r.Findings[0].Verdict != r.Findings[1].Verdict {
			r.Status = "disagreement"
			r.Unresolved = append(r.Unresolved, openQuestion{Role: "both", Reason: "disagreement", Question: r.Question})
		}
	}
	if !data["completion"].Equal(schema.Text(r.Status)) {
		return fmt.Errorf("authored completion %q differs from %q established by the selected findings", data["completion"].Text, r.Status)
	}
	gaps := make([]map[string]schema.Value, 0, len(r.Unresolved))
	for _, gap := range r.Unresolved {
		gaps = append(gaps, map[string]schema.Value{
			"role": schema.Text(gap.Role), "reason": schema.Text(gap.Reason), "question": schema.Text(gap.Question),
		})
	}
	if !data["gaps"].Equal(schema.List(gaps...)) {
		return errors.New("authored gaps differ from the selected outcomes")
	}
	return nil
}

// Apply the investigator's published acceptance rules to the original selected
// reply. A worker that reported a failure can still have supplied a response.
func (w workerReport) assessment(question string) (string, string) {
	reply := w.outcome.Response
	switch {
	case !w.outcome.ResponsePresent:
		return "worker_failed", question
	case !reply["lookup_valid"].Flag:
		return "lookup_failed", question
	case reply["model_disposition"].Text == invalidInterpretation:
		return "invalid_interpretation", question
	case reply["model_disposition"].Text == invalidQuestion:
		return "invalid_question", question
	case !reply["model_valid"].Flag:
		return "model_failed", question
	}
	if followup := reply["unresolved"].Text; followup != "" {
		question = followup
	}
	switch {
	case w.InvocationStatus != "succeeded":
		return "worker_failed", question
	case reply["verdict"].Text == "insufficient":
		return "insufficient", question
	case reply["quote"].Text == "" || !reply["quote"].Equal(reply["snippet"]):
		return "citation_mismatch", question
	default:
		return "validated", question
	}
}

// The worker copies the selected lookup and model fields into its reply.
// This also authenticates reported gaps on branches which produce no finding.
func (w workerReport) verifyReply(s *runtime.ExecutionSnapshot) error {
	if !w.outcome.ResponsePresent {
		return nil
	}
	reply := w.outcome.Response
	check := func(name string, e effectReport) error {
		if e.effect == nil || e.effect.AdapterResult == nil || e.effect.Settlement == nil ||
			!reply[name+"_id"].Equal(schema.Text(e.OperationID)) {
			return fmt.Errorf("missing original %s result", name)
		}
		obs, ok := s.Observation(e.effect.Settlement.ObservationID)
		if !ok || obs.SelectedAt == nil || w.Run == nil || obs.SelectedAt.Run != *w.Run {
			return fmt.Errorf("worker did not select its %s result", name)
		}
		for field, value := range e.effect.AdapterResult.Values {
			target := field
			if field == "valid" || field == "disposition" {
				target = name + "_" + field
			}
			if !reply[target].Equal(value) {
				return fmt.Errorf("field %s differs from the original %s result", target, name)
			}
		}
		return nil
	}
	if err := check("lookup", w.Lookup); err != nil {
		return err
	}
	if reply["lookup_valid"].Flag {
		return check("model", w.Model)
	}
	if w.Model.effect != nil || !reply["model_id"].Equal(schema.Text("")) ||
		!reply["model_valid"].Equal(schema.Bool(false)) || !reply["model_disposition"].Equal(schema.Text("not_requested")) {
		return errors.New("unsuccessful lookup carries unsupported model evidence")
	}
	for field, typ := range modelProposal() {
		if !reply[field].Equal(schema.Default(typ)) {
			return fmt.Errorf("unsuccessful lookup supplied model field %s", field)
		}
	}
	return nil
}

func readEffect(s *runtime.ExecutionSnapshot, run runtime.RunID, id, capability, operation, protocol string) (effectReport, error) {
	if id == "" {
		return describeEffect(nil), nil
	}
	effect, ok := s.Effect(id)
	if !ok || effect.Source != run || effect.Kind != "adapter" || effect.Capability != capability || effect.Name != operation {
		return effectReport{}, errors.New("worker's authored handle does not identify its expected effect")
	}
	if a := effect.AdapterAttempt; a != nil && a.Prepared.Protocol != protocol {
		return effectReport{}, errors.New("worker effect used a different adapter protocol")
	}
	return describeEffect(effect), nil
}

func readWorker(s *runtime.ExecutionSnapshot, root runtime.RootCompletion, data map[string]schema.Value, role string) (workerReport, error) {
	prefix := role + "_"
	w := workerReport{
		Role: role, Query: data[prefix+"query"].Text, OperationID: data[prefix+"call"].Text,
		Lookup: describeEffect(nil), Model: describeEffect(nil),
	}
	if w.OperationID == "" {
		return w, nil
	}
	call, ok := s.Invocation(w.OperationID)
	if !ok || call.Caller != root.Run || call.Inputs["role"].Text != role ||
		call.Inputs["query"].Text != w.Query || !call.Inputs["question"].Equal(data["question"]) {
		return w, errors.New("invocation differs from the original research assignment")
	}
	if admission := call.Admission; admission != nil {
		w.InvocationID, w.Admission = admission.Invocation.ID, admission.Reason
		if admission.Accepted {
			if len(admission.Members) != 1 {
				return w, errors.New("research invocation must have one bound worker")
			}
			member, ok := s.Run(admission.Members[0])
			if !ok {
				return w, errors.New("invoked research worker missing from snapshot")
			}
			w.Run = &member.ID
			w.LocalStatus = member.Projection.Status
			var err error
			w.Lookup, err = readEffect(s, member.ID, member.Projection.Data["lookup_id"].Text, "corpus", "search", "lookup/tokens/v1")
			if err != nil {
				return w, err
			}
			w.Model, err = readEffect(s, member.ID, member.Projection.Data["model_id"].Text, "model", "reason", "openai/responses/v1")
			if err != nil {
				return w, err
			}
		}
	}
	outcome := call.Outcome
	if outcome == nil {
		return w, nil
	}
	w.outcome, w.OutcomeID = outcome, outcome.ID
	w.InvocationStatus, w.InvocationReason, w.ResponsePresent = outcome.Status, outcome.Reason, &outcome.ResponsePresent
	if call.SelectedAt != nil {
		w.SelectedAt = call.SelectedAt.Sequence
		// Selection is an ALAN fact. These role-prefixed destinations are
		// the investigator's own authored response bindings.
		if data[prefix+"status"].Text != outcome.Status || data[prefix+"reason"].Text != outcome.Reason ||
			data[prefix+"present"] != schema.Bool(outcome.ResponsePresent) {
			return w, errors.New("parent bound different worker outcome fields")
		}
		for name, value := range outcome.Response {
			if !data[prefix+name].Equal(value) {
				return w, fmt.Errorf("parent binding %s differs from selected response", name)
			}
		}
		w.Assessment = data[prefix+"assessment"].Text
	}
	return w, nil
}

func (w workerReport) verifyFinding(s *runtime.ExecutionSnapshot, row map[string]schema.Value, question string) (finding, error) {
	if w.Run == nil || w.outcome == nil || w.SelectedAt == 0 || w.InvocationStatus != "succeeded" ||
		!w.outcome.ResponsePresent || w.Assessment != "validated" || row["invocation_operation_id"].Text != w.OperationID {
		return finding{}, errors.New("finding lacks its original successful worker outcome")
	}
	reply := w.outcome.Response
	lookup, model := w.Lookup, w.Model
	if lookup.State != "retained_validated" || model.State != "retained_validated" ||
		!reply["lookup_valid"].Flag || !reply["model_valid"].Flag ||
		row["lookup_id"].Text != lookup.OperationID || row["model_id"].Text != model.OperationID ||
		!reply["lookup_id"].Equal(row["lookup_id"]) || !reply["model_id"].Equal(row["model_id"]) {
		return finding{}, errors.New("finding does not name both original validated effect requests")
	}
	retrieved := lookup.effect.AdapterResult.Candidate.Payload
	prompt, proposal := model.effect.AdapterAttempt.Prepared.Request.Payload, model.effect.AdapterResult.Candidate.Payload
	if lookup.effect.AdapterAttempt.Prepared.Request.Payload["query"].Text != w.Query || prompt["role"].Text != w.Role ||
		prompt["query"].Text != w.Query || prompt["question"].Text != question {
		return finding{}, errors.New("provider request differs from assigned investigation")
	}
	for _, key := range []string{"document", "version", "start", "end", "snippet"} {
		field := key
		if key == "snippet" {
			field = "quote"
		}
		if !row[field].Equal(retrieved[key]) || !prompt[key].Equal(retrieved[key]) || !reply[key].Equal(retrieved[key]) {
			return finding{}, fmt.Errorf("field %s differs from the original lookup", field)
		}
	}
	for _, key := range []string{"quote", "verdict", "interpretation"} {
		if !row[key].Equal(proposal[key]) || !reply[key].Equal(proposal[key]) {
			return finding{}, fmt.Errorf("field %s differs from the original model proposal", key)
		}
	}
	c := citation{Document: row["document"].Text, Version: row["version"].Text,
		Start: row["start"].Number, End: row["end"].Number, Quote: row["quote"].Text}
	asset, ok := s.Asset(c.Document)
	if !ok {
		return finding{}, fmt.Errorf("cannot read retained document %q", c.Document)
	}
	if err := verifySpan(*asset, c); err != nil {
		return finding{}, err
	}
	c.Source = asset.Source
	return finding{Role: w.Role, Verdict: row["verdict"].Text, Interpretation: row["interpretation"].Text, Citation: c}, nil
}
