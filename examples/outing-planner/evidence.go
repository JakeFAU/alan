package main

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// verifyPlannerEvidence uses the retained candidate order to bind each forecast
// to its original operation. It checks committed protocol outputs offline.
func verifyPlannerEvidence(s *runtime.ExecutionSnapshot, c configuration, variant string) error {
	if variant == delegatedWeather {
		info, err := retainedDeployment(s)
		if err != nil {
			return err
		}
		if info.Options.Weather != variant || !reflect.DeepEqual(info.Configuration, c) {
			return fmt.Errorf("delegated evidence differs from the retained deployment")
		}
		return verifyDelegatedEvidence(s, info)
	}
	return verifyDirectEvidence(s, c, variant)
}

func verifyPlannerDeploymentEvidence(s *runtime.ExecutionSnapshot, info retainedDeploymentInfo) error {
	if info.Options.Weather == delegatedWeather {
		return verifyDelegatedEvidence(s, info)
	}
	return verifyDirectEvidence(s, info.Configuration, info.Options.Weather)
}

func verifyDirectEvidence(s *runtime.ExecutionSnapshot, c configuration, variant string) error {
	plan, err := c.normalize()
	if err != nil {
		return err
	}
	raw, err := c.serialized()
	if err != nil {
		return err
	}
	id := schema.Text(digestText(string(raw)))
	adapter, err := openmeteo.NewHourly(c.Endpoint)
	if err != nil {
		return err
	}
	eligible := make([]map[string]schema.Value, 0, len(plan.Candidates))
	for _, row := range plan.Candidates {
		if row.Eligible {
			eligible = append(eligible, candidateRecord(row))
		}
	}
	proposalPayload := map[string]schema.Value{proposalID: id, candidates: schema.List(eligible...)}
	messages := make(map[string]*runtime.Intent, 3)
	for _, op := range s.Operations {
		if op.Kind == "adapter" && op.Run == weatherRun && op.Target == capability &&
			op.Generation == 1 && op.Obligation == effectName {
			continue
		}
		if op.Kind != "message" || messages[op.Obligation] != nil || !validPeerOperation(op) {
			return fmt.Errorf("unexpected outing operation %q", op.ID)
		}
		messages[op.Obligation] = &op
	}
	proposal := messages[proposalPort]
	if proposal != nil && !equalPayload(proposal.Payload, proposalPayload) {
		return fmt.Errorf("retained proposal does not match eligible candidates")
	}
	assessmentBatch := messages[assessmentPort]
	decision := messages[decisionPort]

	forecasts := slices.Clone(s.Effects)
	slices.SortFunc(forecasts, func(a, b *runtime.ExecutionEffect) int {
		if a.Sequence < b.Sequence {
			return -1
		}
		if a.Sequence > b.Sequence {
			return 1
		}
		return 0
	})
	if len(forecasts) > len(eligible) || variant == confirmFirst && len(forecasts) > 0 && decision == nil {
		return fmt.Errorf("retained forecasts exceed the available original opportunities")
	}
	weather, _ := s.Run(weatherRun)
	selectedProposal := selectedMessage(s, proposal, weatherRun)
	previousSelection := selectedProposal
	if variant == confirmFirst && len(forecasts) > 0 {
		previousSelection = selectedMessage(s, decision, weatherRun)
		if selectedProposal == nil || previousSelection == nil || previousSelection.Sequence <= selectedProposal.Sequence {
			return fmt.Errorf("confirmation-first forecasts lack prior proposal and decision selections")
		}
	}
	checked := make([]map[string]schema.Value, 0, len(forecasts))
	for i, forecast := range forecasts {
		if previousSelection == nil || previousSelection.Sequence >= forecast.Sequence {
			return fmt.Errorf("forecast %s precedes its required prior selection", forecast.OperationID)
		}
		row, err := checkForecast(s, c, adapter, forecast, eligible[i], weatherRun)
		if err != nil {
			return err
		}
		if row == nil {
			if i < len(forecasts)-1 {
				return fmt.Errorf("forecast %s advanced before receiving its original result", forecast.OperationID)
			}
			continue
		}
		checked = append(checked, row)
		o, _ := s.Observation(forecast.Settlement.ObservationID)
		previousSelection = o.SelectedAt
	}
	if assessmentBatch != nil {
		if selectedProposal == nil || len(checked) != len(eligible) {
			return fmt.Errorf("assessment batch lacks exhaustive original forecast evidence")
		}
		assessmentPayload := map[string]schema.Value{proposalID: id, assessments: schema.List(checked...)}
		if !equalPayload(assessmentBatch.Payload, assessmentPayload) {
			return fmt.Errorf("assessment batch disagrees with its retained forecast observations")
		}
		if previousSelection.Sequence >= assessmentBatch.Sequence {
			return fmt.Errorf("assessment batch precedes its required prior selection")
		}
	}

	outing, _ := s.Run(outingRun)
	if decision == nil {
		if outing.Projection.Status != "open" || weather.Projection.Status != "open" {
			return fmt.Errorf("outing completed without a retained decision")
		}
		return nil
	}
	assessmentSelection := selectedMessage(s, assessmentBatch, outingRun)
	if assessmentSelection == nil || assessmentSelection.Sequence >= decision.Sequence {
		return fmt.Errorf("decision precedes Outing's assessment selection")
	}
	return checkDecision(s, decision, proposalPayload, checked)
}

func validPeerOperation(op runtime.Intent) bool {
	if op.Generation != 1 {
		return false
	}
	switch op.Obligation {
	case proposalPort, decisionPort:
		return op.Run == outingRun && op.Target == weatherSubject
	case assessmentPort:
		return op.Run == weatherRun && op.Target == outingSubject
	default:
		return false
	}
}

func equalPayload(got, want map[string]schema.Value) bool {
	if len(got) != len(want) {
		return false
	}
	for key, expected := range want {
		if !got[key].Equal(expected) {
			return false
		}
	}
	return true
}

func checkForecast(
	s *runtime.ExecutionSnapshot,
	c configuration,
	adapter *openmeteo.HourlyAdapter,
	e *runtime.ExecutionEffect,
	row map[string]schema.Value,
	owner runtime.RunID,
) (map[string]schema.Value, error) {
	op, exists := s.Operation(e.OperationID)
	request := map[string]schema.Value{
		openmeteo.Latitude: row[openmeteo.Latitude], openmeteo.Longitude: row[openmeteo.Longitude],
		openmeteo.StartHour: row[openmeteo.StartHour], openmeteo.EndHour: row[openmeteo.EndHour],
		openmeteo.Units: schema.Text(c.Units),
	}
	if !exists || e.Kind != "adapter" || e.Source != owner || e.Capability != capability || e.Name != effectName ||
		!equalPayload(op.Payload, request) || len(e.AdditionalReceipts) != 0 {
		return nil, fmt.Errorf("forecast %s is not an original request for candidate %s", e.OperationID, row["id"].Text)
	}
	attempt, result := e.AdapterAttempt, e.AdapterResult
	if attempt != nil && attempt.Allowed {
		prepared := attempt.Prepared
		want, err := adapter.Prepare(prepared.Request)
		if err != nil || !reflect.DeepEqual(want, prepared) || prepared.Request.ID != e.OperationID ||
			prepared.Request.Capability != capability || prepared.Request.Effect != effectName ||
			!equalPayload(prepared.Request.Payload, request) {
			return nil, fmt.Errorf("forecast reservation %s disagrees with original request", e.OperationID)
		}
		if result != nil {
			if err := openmeteo.VerifyHourlyResponseEvidence(prepared, result.Candidate); err != nil {
				return nil, fmt.Errorf("forecast response %s: %w", e.OperationID, err)
			}
		}
	}
	valid := result != nil && result.Values[schema.ResultValid].Flag
	if valid && (attempt == nil || !attempt.Allowed || e.DispatchAdmission == nil ||
		result.Candidate.Disposition != "validated") {
		return nil, fmt.Errorf("forecast %s claims success without admitted native evidence", e.OperationID)
	}
	if e.Settlement == nil {
		return nil, nil
	}
	o, exists := s.Observation(e.Settlement.ObservationID)
	if !exists || o.SourceOperationID != e.OperationID || o.Run != owner || o.Operation != e.OperationID {
		return nil, fmt.Errorf("forecast %s lacks its original observation", e.OperationID)
	}
	if o.SelectedAt == nil {
		return nil, nil
	}
	if o.SelectedAt.Run != owner || o.SelectedAt.Sequence <= e.Sequence {
		return nil, fmt.Errorf("forecast %s has an invalid selection", e.OperationID)
	}
	status := unavailable
	if valid {
		status = unsuitable
		payload := result.Candidate.Payload
		codes := c.allowedCodes()
		if slices.Contains(codes, payload[openmeteo.StartWeatherCode].Number) &&
			slices.Contains(codes, payload[openmeteo.EndWeatherCode].Number) {
			status = suitable
		}
	}
	assessment := make(map[string]schema.Value, len(row)+2)
	for key, value := range row {
		assessment[key] = value
	}
	assessment["forecast_operation"] = schema.Text(e.OperationID)
	assessment["weather_status"] = schema.Text(status)
	return assessment, nil
}

func selectedMessage(s *runtime.ExecutionSnapshot, op *runtime.Intent, run runtime.RunID) *runtime.ReductionRef {
	if op != nil {
		for _, o := range s.Observations {
			if o.Kind == "message" && o.SourceOperationID == op.ID && o.Run == run && o.SelectedAt != nil {
				return o.SelectedAt
			}
		}
	}
	return nil
}

func checkDecision(
	s *runtime.ExecutionSnapshot,
	op *runtime.Intent,
	proposal map[string]schema.Value,
	checked []map[string]schema.Value,
) error {
	status := noMatch
	var chosen []map[string]schema.Value
	for _, row := range checked {
		weatherStatus := row["weather_status"].Text
		if weatherStatus == suitable {
			status, chosen = recommended, []map[string]schema.Value{row}
			break
		}
		if weatherStatus == unavailable {
			status = unavailable
		}
	}
	expected := map[string]schema.Value{
		proposalID: proposal[proposalID], candidates: proposal[candidates],
		outcome: schema.Text(status), selection: schema.List(chosen...),
	}
	if !equalPayload(op.Payload, expected) {
		return fmt.Errorf("retained decision differs from ranked forecast evidence")
	}
	outing, _ := s.Run(outingRun)
	weather, _ := s.Run(weatherRun)
	var delivered bool
	for _, o := range s.Observations {
		if o.Kind == "message" && o.SourceOperationID == op.ID && o.Run == weatherRun {
			delivered = true
			break
		}
	}
	wantTerminal := "succeeded"
	if status == unavailable {
		wantTerminal = "failed"
	}
	if outing.Projection.Status != "open" && (!delivered || outing.Projection.Status != wantTerminal) {
		return fmt.Errorf("outing program terminated without delivering its final decision")
	}
	weatherSelection := selectedMessage(s, op, weatherRun)
	if weather.Projection.Status != "open" && (weatherSelection == nil || weather.Projection.Status != wantTerminal) {
		return fmt.Errorf("weather program terminated without consuming the exact decision")
	}
	return nil
}
