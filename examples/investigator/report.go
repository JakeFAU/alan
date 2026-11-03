package main

import (
	"context"
	"encoding/json"
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
	OperationID       string         `json:"operation_id,omitempty"`
	State             string         `json:"state"`
	ReservationReason string         `json:"reservation_reason,omitempty"`
	Disposition       string         `json:"disposition,omitempty"`
	Model             string         `json:"configured_model,omitempty"`
	ReturnedModel     string         `json:"returned_model,omitempty"`
	HTTPStatus        int            `json:"http_status,omitempty"`
	ClientRequestID   string         `json:"client_request_id,omitempty"`
	RequestID         string         `json:"provider_request_id,omitempty"`
	ResponseID        string         `json:"provider_response_id,omitempty"`
	Usage             *reportedUsage `json:"provider_reported_usage"`
	Diagnostic        string         `json:"diagnostic_response,omitempty"`
	attempt           *runtime.AdapterAttempt
	result            *runtime.AdapterResult
}

func describeEffect(id string, attempt *runtime.AdapterAttempt, result *runtime.AdapterResult) effectReport {
	e := effectReport{OperationID: id, State: "not_requested", attempt: attempt, result: result}
	if id == "" {
		return e
	}
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
	if u := c.Usage; u != nil {
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
	if e.attempt == nil {
		return
	}
	a.ReservationDecisions++
	if e.attempt.Allowed {
		executions.Maximum++
		if e.result != nil {
			executions.Minimum++
		}
	}
}

func (a *accounting) countModel(e effectReport) {
	a.countEffect(e, &a.ModelExecutions)
	if e.attempt != nil && e.attempt.Allowed {
		a.ProviderContacts.Maximum++
		a.ReservedOutputTokens += e.attempt.Prepared.MaxOutputTokens
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

// Inspect validates the ledger and supplies one read-only transaction snapshot.
// An answer must never mix a newer parent selection with an older worker result.
func readReport(ctx context.Context, path string, limits runtime.Limits) (investigationReport, error) {
	s, err := runtime.Inspect(ctx, path, limits)
	if err != nil {
		return investigationReport{}, err
	}
	if _, err := approvedDeployment(s); err != nil {
		return investigationReport{}, err
	}
	return reportSnapshot(s)
}

func reportSnapshot(s runtime.Inspection) (investigationReport, error) {
	id := runtime.RunID{Subject: "investigator", Generation: 1}
	if len(s.Roots) != 1 || s.Roots[0].Run != id {
		return investigationReport{}, errors.New("expected one investigator root")
	}
	root := s.Roots[0]
	parent, err := readProjection(s, id)
	if err != nil {
		return investigationReport{}, err
	}
	data := parent.Data
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

func (r *investigationReport) readAnswer(s runtime.Inspection, data map[string]schema.Value) error {
	switch status := data["completion"].Text; status {
	case "completed", "disagreement", "partial", "incomplete":
		r.Status = status
	default:
		return errors.New("parent has no recognized completion")
	}
	accepted := map[string]workerReport{}
	for _, w := range r.Workers {
		if w.SelectedAt == 0 {
			return fmt.Errorf("parent did not consume %s outcome", w.Role)
		}
		if data[w.Role+"_accepted"].Flag {
			accepted[w.Role] = w
		}
	}
	findings, err := data["findings"].Records()
	if err != nil {
		return err
	}
	for _, row := range findings {
		role := row["role"].Text
		w, ok := accepted[role]
		if !ok {
			return errors.New("finding lacks a distinct authored worker acceptance")
		}
		f, err := w.verifyFinding(s, row, r.Question)
		if err != nil {
			return fmt.Errorf("%s finding: %w", w.Role, err)
		}
		r.Findings = append(r.Findings, f)
		delete(accepted, role)
	}
	if len(accepted) != 0 {
		return errors.New("an accepted worker is missing from the authored findings")
	}
	gaps, err := data["gaps"].Records()
	if err != nil {
		return err
	}
	for _, row := range gaps {
		r.Unresolved = append(r.Unresolved, openQuestion{
			Role: row["role"].Text, Reason: row["reason"].Text, Question: row["question"].Text,
		})
	}
	return nil
}

// Inspection exposes typed JSON descriptors in named table cells. These helpers
// isolate its storage-facing row representation from the answer format.
func findRow(rows []map[string]any, key string, value any) map[string]any {
	for _, row := range rows {
		if row[key] == value {
			return row
		}
	}
	return nil
}

func cell[T any](row map[string]any, key string) (*T, error) {
	if row == nil {
		return nil, nil
	}
	raw, ok := row[key].(string)
	if !ok {
		return nil, fmt.Errorf("missing %s JSON in inspection", key)
	}
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func readProjection(s runtime.Inspection, id runtime.RunID) (*runtime.Projection, error) {
	for _, row := range s.Tables["runs"] {
		if row["subject"] == id.Subject && row["generation"] == id.Generation {
			return cell[runtime.Projection](row, "projection")
		}
	}
	return nil, fmt.Errorf("missing retained run %v", id)
}

func readEffect(s runtime.Inspection, run runtime.RunID, id, capability, effect, protocol string) (effectReport, error) {
	if id == "" {
		return describeEffect("", nil, nil), nil
	}
	attempt, err := cell[runtime.AdapterAttempt](findRow(s.Tables["adapter_attempts"], "intent_id", id), "descriptor")
	if err != nil {
		return effectReport{}, err
	}
	result, err := cell[runtime.AdapterResult](findRow(s.Tables["adapter_results"], "intent_id", id), "descriptor")
	if err != nil {
		return effectReport{}, err
	}
	if attempt != nil {
		req := attempt.Prepared.Request
		if attempt.Intent.Run != run || attempt.Intent.ID != id || req.ID != id ||
			req.Capability != capability || req.Effect != effect || attempt.Prepared.Protocol != protocol {
			return effectReport{}, errors.New("original effect is not the expected worker operation")
		}
	}
	if result != nil && (attempt == nil || result.Candidate.ID != id || result.Candidate.Key != attempt.Prepared.Request.Key) {
		return effectReport{}, errors.New("adapter result does not correspond to the original request")
	}
	return describeEffect(id, attempt, result), nil
}

func readWorker(s runtime.Inspection, root runtime.RootCompletion, data map[string]schema.Value, role string) (workerReport, error) {
	prefix := role + "_"
	w := workerReport{
		Role: role, Query: data[prefix+"query"].Text, OperationID: data[prefix+"call"].Text,
		Lookup: describeEffect("", nil, nil), Model: describeEffect("", nil, nil),
	}
	admission, err := cell[runtime.InvocationAdmission](
		findRow(s.Tables["composition_admission_decisions"], "operation_id", w.OperationID), "descriptor")
	if err != nil || admission == nil {
		return w, err
	}
	i := admission.Invocation
	w.InvocationID, w.Admission = i.ID, admission.Reason
	if i.OperationID != w.OperationID || i.Caller != root.Run || i.RootID != root.RootID ||
		i.Inputs["role"].Text != role || i.Inputs["query"].Text != w.Query || !i.Inputs["question"].Equal(data["question"]) {
		return w, errors.New("invocation differs from the original research assignment")
	}
	if admission.Accepted {
		if len(admission.Members) != 1 {
			return w, errors.New("research invocation must have one bound worker")
		}
		w.Run = &admission.Members[0]
		p, err := readProjection(s, *w.Run)
		if err != nil {
			return w, err
		}
		w.LocalStatus = p.Status
		w.Lookup, err = readEffect(s, *w.Run, p.Data["lookup_id"].Text, "corpus", "search", "lookup/tokens/v1")
		if err != nil {
			return w, err
		}
		w.Model, err = readEffect(s, *w.Run, p.Data["model_id"].Text, "model", "reason", "openai/responses/v1")
		if err != nil {
			return w, err
		}
	}
	outcome, err := cell[runtime.InvocationOutcome](findRow(s.Tables["composition_publications"], "operation_id", w.OperationID), "descriptor")
	if err != nil || outcome == nil {
		return w, err
	}
	if outcome.InvocationID != i.ID || outcome.OperationID != i.OperationID || outcome.Caller != i.Caller || outcome.TemplateID != i.TemplateID {
		return w, errors.New("publication belongs to another invocation")
	}
	w.outcome, w.OutcomeID = outcome, outcome.ID
	w.InvocationStatus, w.InvocationReason, w.ResponsePresent = outcome.Status, outcome.Reason, &outcome.ResponsePresent
	delivery := findRow(s.Tables["composition_outcome_deliveries"], "operation_id", w.OperationID)
	obs := findRow(s.Tables["observations"], "id", delivery["observation_id"])
	if obs == nil || obs["subject"] != root.Run.Subject || obs["generation"] != root.Run.Generation {
		return w, errors.New("publication missing its original parent delivery")
	}
	// Two observations can belong to the same parent and template. Only the
	// original operation distinguishes these two calls to the research worker.
	envelope, err := cell[struct{ Operation, Source, Kind, Provenance string }](obs, "descriptor")
	if err != nil {
		return w, err
	}
	if envelope.Operation != i.OperationID || envelope.Source != i.TemplateID ||
		envelope.Kind != "invocation" || envelope.Provenance != "runtime-invocation" {
		return w, errors.New("delivered observation belongs to another operation")
	}
	w.SelectedAt, _ = obs["consumed_sequence"].(int64)
	if w.SelectedAt != 0 {
		commit := findRow(s.Tables["history"], "selected_observation", obs["id"])
		if commit["subject"] != root.Run.Subject || commit["generation"] != root.Run.Generation || commit["sequence"] != w.SelectedAt {
			return w, errors.New("parent history lacks the matching selection")
		}
		if data[prefix+"status"].Text != outcome.Status || data[prefix+"reason"].Text != outcome.Reason ||
			data[prefix+"present"] != schema.Bool(outcome.ResponsePresent) {
			return w, errors.New("parent selected a different outcome envelope")
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

func (w workerReport) verifyFinding(s runtime.Inspection, row map[string]schema.Value, question string) (finding, error) {
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
	retrieved := lookup.result.Candidate.Payload
	prompt, proposal := model.attempt.Prepared.Request.Payload, model.result.Candidate.Payload
	if lookup.attempt.Prepared.Request.Payload["query"].Text != w.Query || prompt["role"].Text != w.Role ||
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
	asset, err := cell[effects.Asset](findRow(s.Tables["corpus_assets"], "id", c.Document), "descriptor")
	if err != nil {
		return finding{}, err
	}
	if asset == nil {
		return finding{}, fmt.Errorf("cannot read retained document %q", c.Document)
	}
	if err := verifySpan(*asset, c); err != nil {
		return finding{}, err
	}
	c.Source = asset.Source
	return finding{Role: w.Role, Verdict: row["verdict"].Text, Interpretation: row["interpretation"].Text, Citation: c}, nil
}
