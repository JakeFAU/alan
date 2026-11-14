package main

import (
	"fmt"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

type seriesSummary struct {
	ID             string `json:"id"`
	Bound          int    `json:"bound"`
	Selected       int    `json:"selected"`
	Completed      int    `json:"completed"`
	Failed         int    `json:"failed"`
	Pending        int    `json:"pending"`
	Unresolved     int    `json:"unresolved"`
	BoundExhausted bool   `json:"bound_exhausted"`
}

type summary struct {
	Series             *seriesSummary              `json:"series,omitempty"`
	Namespace          string                      `json:"namespace"`
	Latitude           string                      `json:"latitude"`
	Longitude          string                      `json:"longitude"`
	Units              string                      `json:"units"`
	Endpoint           string                      `json:"endpoint"`
	Allowance          int                         `json:"allowance"`
	OpportunitiesSpent int64                       `json:"opportunities_spent"`
	ProviderCostUSD    *string                     `json:"provider_cost_usd"`
	Reads              []readSummary               `json:"reads"`
	Attribution        string                      `json:"attribution"`
	Pending            []runtime.PendingObligation `json:"pending,omitempty"`
	History            []runtime.ExecutionStep     `json:"history,omitempty"`
	RefusedInputs      []*runtime.AdmissionView    `json:"refused_local_inputs,omitempty"`
	ReplayedCommits    *int                        `json:"replayed_commits,omitempty"`
}

type contactBounds struct {
	Minimum int `json:"minimum"`
	Maximum int `json:"maximum"`
}

type readSummary struct {
	SeriesState            author.SeriesState         `json:"series_state,omitempty"`
	Read                   int                        `json:"read"`
	Agent                  runtime.RunID              `json:"agent"`
	State                  string                     `json:"state"`
	LocalStatus            string                     `json:"local_status"`
	PreviousOperationID    string                     `json:"previous_operation_id,omitempty"`
	Continuation           *runtime.ContinuationView  `json:"continuation,omitempty"`
	ApprovalObservationID  string                     `json:"approval_observation_id,omitempty"`
	ApprovalReason         string                     `json:"approval_reason,omitempty"`
	ApprovalPreparedDigest string                     `json:"approval_prepared_digest,omitempty"`
	ApprovalResultDigest   string                     `json:"approval_result_digest,omitempty"`
	ApprovalSelectedAt     *runtime.ReductionRef      `json:"approval_selected_at,omitempty"`
	RefreshInputIssue      string                     `json:"refresh_input_issue,omitempty"`
	OperationID            string                     `json:"operation_id,omitempty"`
	Capability             string                     `json:"capability,omitempty"`
	EffectStage            runtime.EffectStage        `json:"effect_stage,omitempty"`
	Authorization          *runtime.Authorization     `json:"authorization,omitempty"`
	RequestKey             string                     `json:"request_key,omitempty"`
	RequestURL             string                     `json:"request_url,omitempty"`
	ReservationReason      string                     `json:"reservation_reason,omitempty"`
	OpportunitySpent       bool                       `json:"opportunity_spent"`
	DispatchAdmission      *runtime.DispatchAdmission `json:"dispatch_admission,omitempty"`
	Dispatch               *runtime.EffectDispatch    `json:"adapter_executions,omitempty"`
	ProviderContacts       contactBounds              `json:"provider_contacts"`
	Disposition            string                     `json:"disposition,omitempty"`
	ResultDigest           string                     `json:"result_digest,omitempty"`
	HTTPStatus             int                        `json:"http_status,omitempty"`
	Response               string                     `json:"retained_response,omitempty"`
	ObservationID          string                     `json:"settlement_observation_id,omitempty"`
	SelectedObservationID  string                     `json:"selected_observation_id,omitempty"`
	SelectedAt             *runtime.ReductionRef      `json:"selected_at,omitempty"`
	ResultOrigin           string                     `json:"result_origin,omitempty"`
	Source                 string                     `json:"source,omitempty"`
	ObservedAt             string                     `json:"observed_at,omitempty"`
	GridLatitude           string                     `json:"grid_latitude,omitempty"`
	GridLongitude          string                     `json:"grid_longitude,omitempty"`
	Temperature            string                     `json:"temperature,omitempty"`
	TemperatureUnit        string                     `json:"temperature_unit,omitempty"`
	Humidity               string                     `json:"humidity,omitempty"`
	HumidityUnit           string                     `json:"humidity_unit,omitempty"`
	Report                 string                     `json:"report,omitempty"`
}

func summarize(s, before *runtime.ExecutionSnapshot, c configuration, detailed bool) (summary, error) {
	out := summary{Namespace: s.Namespace, Latitude: c.Latitude, Longitude: c.Longitude, Units: c.Units,
		Endpoint: c.Endpoint, Allowance: c.Allowance, Attribution: openmeteo.Attribution}
	if c.Endpoint != openmeteo.DefaultEndpoint {
		out.Attribution = "Synthetic data from the configured loopback fixture using the Open-Meteo API format (https://open-meteo.com/en/docs)."
	}
	for _, root := range s.Roots {
		out.OpportunitiesSpent += root.Charges.Effects
	}
	var positions []author.SeriesPosition
	if c.Allowance > 0 {
		if series, err := c.series(); err == nil {
			if view, err := series.Inspect(s); err == nil {
				positions = view.Positions
				out.Series = &seriesSummary{ID: view.ID, Bound: view.Bound, Selected: view.Selected, Completed: view.Completed, Failed: view.Failed, Pending: view.Pending, Unresolved: view.Unresolved, BoundExhausted: view.BoundExhausted}
			}
		}
	}
	for n := 1; n <= max(1, c.Allowance); n++ {
		run, _ := s.Run(readID(n)) // retainedConfiguration verified all original runs.
		if positions != nil {
			run = positions[n-1].Run
		}
		read := readSummary{Read: n, Agent: run.ID, LocalStatus: run.Projection.Status,
			PreviousOperationID: run.Projection.Data["previous_operation_id"].Text, State: "awaiting_request"}
		if positions != nil {
			read.SeriesState = positions[n-1].State
		}
		if n > 1 {
			read.State = "awaiting_refresh_approval"
			read.Continuation, _ = s.Continuation(run.ID)
			var o *runtime.ObservationView
			var err error
			if read.Continuation != nil {
				o, _ = s.Observation(read.Continuation.ObservationID)
				read.PreviousOperationID = read.Continuation.Approval.PredecessorID
			} else if len(s.Composition.Continuations) == 0 {
				o, err = legacyApproval(s, n)
			}
			if err != nil {
				if len(run.EffectIDs) != 0 {
					return out, err
				}
				read.State, read.RefreshInputIssue = "refresh_input_rejected", err.Error()
				out.Reads = append(out.Reads, read)
				continue
			}
			if o != nil {
				read.ApprovalObservationID = o.ID
				read.ApprovalReason = o.Payload["purpose"].Text
				read.ApprovalPreparedDigest = o.Payload["prepared_digest"].Text
				read.ApprovalResultDigest = o.Payload["result_digest"].Text
				read.ApprovalSelectedAt = o.SelectedAt
				read.State = "refresh_approved"
			}
		}
		if len(run.EffectIDs) == 1 {
			e, _ := s.Effect(run.EffectIDs[0])
			if err := read.describe(s, before, run, e, c, detailed); err != nil {
				return out, err
			}
		} else if len(run.EffectIDs) > 1 || run.Projection.Status == "succeeded" {
			return out, fmt.Errorf("%s lacks a unique original weather effect", run.ID.Subject)
		} else if run.Projection.Status != "open" {
			read.State = "unavailable"
		}
		out.Reads = append(out.Reads, read)
	}
	if detailed {
		out.Pending = s.Pending
		for _, a := range s.Admissions {
			if a.Domain == "input" && !a.Accepted {
				out.RefusedInputs = append(out.RefusedInputs, a)
			}
		}
		for offset := 0; ; {
			page, err := s.History(offset, runtime.MaxHistoryPage)
			if err != nil {
				return out, err
			}
			out.History = append(out.History, page.Steps...)
			if !page.More {
				break
			}
			offset = page.Next
		}
	}
	return out, nil
}

func (r *readSummary) describe(s, before *runtime.ExecutionSnapshot, run *runtime.ExecutionRun, e *runtime.ExecutionEffect, c configuration, detailed bool) error {
	r.OperationID, r.Capability, r.EffectStage, r.Disposition = e.OperationID, e.Capability, e.State, e.ResultDisposition
	r.Authorization, r.Dispatch, r.DispatchAdmission = e.Authorization, &e.Dispatch, e.DispatchAdmission
	r.State = string(e.State)
	if a := e.AdapterAttempt; a != nil {
		r.RequestURL, r.RequestKey, r.ReservationReason = a.Prepared.Wire, a.Prepared.Request.Key, a.Reason
		r.OpportunitySpent = a.Allowed
	}
	if e.Dispatch.Maximum != nil {
		r.ProviderContacts.Maximum = *e.Dispatch.Maximum
	}
	if e.State == runtime.EffectReservedUnknown {
		r.State = "unresolved"
		if e.DispatchAdmission == nil {
			r.State = "spent_before_dispatch"
		}
	}
	if a := e.AdapterResult; a != nil {
		r.ResultDigest = a.Digest
		r.HTTPStatus = a.Candidate.HTTPStatus
		if r.HTTPStatus != 0 {
			r.ProviderContacts.Minimum = 1
		}
		if detailed {
			r.Response = a.Candidate.Response
		}
		if a.Disposition == "validated" {
			r.ResultOrigin = "previously_retained"
			if before != nil {
				if prior, found := before.Effect(e.OperationID); !found || prior.AdapterResult == nil {
					r.ResultOrigin = "read_during_this_run"
				}
			}
		}
	}
	if e.Settlement != nil {
		r.ObservationID = e.Settlement.ObservationID
		o, _ := s.Observation(r.ObservationID)
		if o.SelectedAt == nil {
			r.State = "awaiting_selection"
		} else {
			r.SelectedAt, r.SelectedObservationID = o.SelectedAt, o.ID
			r.State = "observation_selected"
		}
	}
	if run.Projection.Status == "open" {
		return nil
	}
	if run.Projection.Status != "succeeded" {
		r.State = "unavailable"
		return nil
	}
	data := run.Projection.Data
	if r.SelectedAt == nil || e.AdapterResult == nil || r.Disposition != "validated" || !data["valid"].Flag ||
		data["disposition"].Text != "validated" || data["operation_id"].Text != e.OperationID {
		return fmt.Errorf("%s lacks its original selected, validated observation", run.ID.Subject)
	}
	for field, value := range e.AdapterResult.Candidate.Payload {
		if !data[field].Equal(value) {
			return fmt.Errorf("%s report field %s differs from the original response", run.ID.Subject, field)
		}
	}
	r.State = "reported"
	r.Source, r.ObservedAt = data[openmeteo.Source].Text, data[openmeteo.ObservedAt].Text
	r.GridLatitude, r.GridLongitude = data[openmeteo.GridLatitude].Text, data[openmeteo.GridLongitude].Text
	r.Temperature, r.TemperatureUnit = data[openmeteo.Temperature].Text, data[openmeteo.TemperatureUnit].Text
	r.Humidity, r.HumidityUnit = data[openmeteo.Humidity].Text, data[openmeteo.HumidityUnit].Text
	r.Report = fmt.Sprintf("At %s, %s reports %s %s with %s%s relative humidity for (%s, %s), using grid (%s, %s).",
		r.ObservedAt, r.Source, r.Temperature, r.TemperatureUnit, r.Humidity, r.HumidityUnit, c.Latitude, c.Longitude, r.GridLatitude, r.GridLongitude)
	return nil
}
