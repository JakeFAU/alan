package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/runtime"
)

const (
	pendingOutcome  = "pending"
	unknownForecast = "unknown_forecast"
)

type forecastChoice struct {
	CandidateID string `json:"candidate_id"`
	ParkID      string `json:"park_id"`
	Name        string `json:"name"`
	Start       string `json:"local_start"`
	End         string `json:"local_end"`
	Travel      int    `json:"one_way_travel_minutes"`
	Historical  bool   `json:"historical"`
	OperationID string `json:"forecast_operation"`
}

type plannerReport struct {
	Outcome            string                        `json:"outcome"`
	Explanation        string                        `json:"explanation"`
	Date               string                        `json:"date"`
	Timezone           string                        `json:"timezone"`
	Namespace          string                        `json:"namespace"`
	Variant            string                        `json:"weather_variant"`
	Configuration      configuration                 `json:"configuration"`
	Candidates         []candidate                   `json:"candidates"`
	Recommendation     *forecastChoice               `json:"recommendation,omitempty"`
	EvidenceGaps       []string                      `json:"evidence_gaps"`
	Proposal           *runtime.Intent               `json:"proposal,omitempty"`
	Assessment         *runtime.Intent               `json:"assessment,omitempty"`
	Decision           *runtime.Intent               `json:"decision,omitempty"`
	DecisionSelectedAt *runtime.ReductionRef         `json:"decision_selected_at,omitempty"`
	Runs               []*runtime.ExecutionRun       `json:"runs"`
	Effects            []*runtime.ExecutionEffect    `json:"effects"`
	Observations       []*runtime.ObservationView    `json:"observations"`
	Admissions         []*runtime.AdmissionView      `json:"admissions"`
	Permissions        []runtime.Permission          `json:"permissions"`
	Composition        *runtime.CompositionProvision `json:"composition"`
	Pending            []runtime.PendingObligation   `json:"pending"`
	History            []runtime.ExecutionStep       `json:"history"`
	Progress           *runtime.RunProgress          `json:"progress,omitempty"`
	Replay             *runtime.ReplayReport         `json:"replay,omitempty"`
}

// buildReport presents one validated inspection cut. Runtime selections are
// kept alongside the proposed and committed protocol data.
func buildReport(s *runtime.ExecutionSnapshot, c configuration, variant string) (plannerReport, error) {
	if s == nil {
		return plannerReport{}, fmt.Errorf("missing retained execution snapshot")
	}
	plan, err := c.normalize()
	if err != nil {
		return plannerReport{}, err
	}
	outing, haveOuting := s.Run(outingRun)
	weather, haveWeather := s.Run(weatherRun)
	if !haveOuting || !haveWeather {
		return plannerReport{}, fmt.Errorf("missing original outing or weather run")
	}
	r := plannerReport{
		Outcome: pendingOutcome, Explanation: "The retained plan still has work or a message to receive.",
		Date: plan.Date, Timezone: c.Timezone, Namespace: s.Namespace, Variant: variant,
		Configuration: c, Candidates: plan.Candidates, EvidenceGaps: []string{},
		Runs: s.Runs, Effects: append([]*runtime.ExecutionEffect{}, s.Effects...),
		Observations: append([]*runtime.ObservationView{}, s.Observations...),
		Admissions:   append([]*runtime.AdmissionView{}, s.Admissions...),
		Permissions:  s.Permissions, Composition: s.Composition,
		Pending: append([]runtime.PendingObligation{}, s.Pending...),
		History: []runtime.ExecutionStep{},
	}
	for offset := 0; ; {
		page, err := s.History(offset, runtime.MaxHistoryPage)
		if err != nil {
			return plannerReport{}, err
		}
		r.History = append(r.History, page.Steps...)
		if !page.More {
			break
		}
		offset = page.Next
	}

	for i := range s.Operations {
		op := &s.Operations[i]
		switch op.Obligation {
		case proposalPort:
			r.Proposal = op
		case assessmentPort:
			r.Assessment = op
		case decisionPort:
			r.Decision = op
		}
	}
	if r.Assessment != nil {
		rows, err := r.Assessment.Payload[assessments].Records()
		if err != nil {
			return plannerReport{}, err
		}
		for _, a := range rows {
			if a["weather_status"].Text == unavailable {
				r.EvidenceGaps = append(r.EvidenceGaps, fmt.Sprintf("%s at %s", a["name"].Text, a[openmeteo.StartHour].Text))
			}
		}
	}
	for _, e := range s.Effects {
		if e.DispatchAdmission != nil && e.AdapterResult == nil {
			r.Outcome = unknownForecast
			r.Explanation = fmt.Sprintf("Forecast dispatch %s was admitted and its response is unknown. The original operation remains outstanding.", e.OperationID)
			return r, nil
		}
	}

	r.DecisionSelectedAt = selectedMessage(s, r.Decision, weatherRun)
	if r.Decision == nil || r.DecisionSelectedAt == nil ||
		outing.Projection.Status == "open" || weather.Projection.Status == "open" {
		if r.Decision != nil {
			r.Explanation = "Outing committed a decision. Delivery, consumption or final completion is still pending."
		}
		return r, nil
	}

	decisionOutcome := r.Decision.Payload[outcome].Text
	if decisionOutcome == unavailable && outing.Projection.Status == "failed" && weather.Projection.Status == "failed" {
		r.Outcome = unavailable
		r.Explanation = "The planning attempt failed. Unavailable forecast evidence prevents a conclusion for these options."
		return r, nil
	}
	if outing.Projection.Status != "succeeded" || weather.Projection.Status != "succeeded" {
		return plannerReport{}, fmt.Errorf("retained decision and terminal states disagree")
	}
	switch decisionOutcome {
	case noMatch:
		r.Outcome = noMatch
		r.Explanation = "Every locally eligible option had a validated unsuitable forecast."
		if r.Proposal == nil {
			return plannerReport{}, fmt.Errorf("completed decision has no retained proposal")
		}
		rows, err := r.Proposal.Payload[candidates].Records()
		if err != nil {
			return plannerReport{}, err
		}
		if len(rows) == 0 {
			r.Explanation = "None of the supplied park and time options met the local planning constraints."
		}
	case recommended:
		chosen, err := r.Decision.Payload[selection].Records()
		if err != nil {
			return plannerReport{}, err
		}
		if len(chosen) != 1 {
			return plannerReport{}, fmt.Errorf("expected one retained recommendation")
		}
		id := chosen[0]["id"].Text
		pos := slices.IndexFunc(plan.Candidates, func(row candidate) bool { return row.ID == id })
		if pos < 0 {
			return plannerReport{}, fmt.Errorf("retained recommendation does not name an original candidate")
		}
		row := plan.Candidates[pos]
		zone, err := time.LoadLocation(c.Timezone)
		if err != nil {
			return plannerReport{}, err
		}
		start, err := parseExplicitTime(row.Start)
		if err != nil {
			return plannerReport{}, err
		}
		end, err := parseExplicitTime(row.End)
		if err != nil {
			return plannerReport{}, err
		}
		r.Outcome = recommended
		r.Explanation = "First eligible option with qualifying endpoint-hour samples, ranked by travel time, start time and park ID."
		r.Recommendation = &forecastChoice{
			CandidateID: id, ParkID: row.ParkID, Name: row.Name,
			Start: start.In(zone).Format(time.RFC3339), End: end.In(zone).Format(time.RFC3339),
			Travel: row.TravelMinutes, Historical: !time.Now().Before(start),
			OperationID: chosen[0]["forecast_operation"].Text,
		}
	default:
		return plannerReport{}, fmt.Errorf("unexpected retained outcome %q", decisionOutcome)
	}
	return r, nil
}

func (r plannerReport) writeText(out io.Writer) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Outing for %s (%s)\n", r.Date, r.Timezone)
	if chosen := r.Recommendation; chosen != nil {
		label := "Recommended outing"
		if chosen.Historical {
			label = "Recommended outing (historical)"
		}
		fmt.Fprintf(&text, "%s — %s, %s to %s, %d minutes one way.\n",
			label, chosen.Name, chosen.Start, chosen.End, chosen.Travel)
		for _, e := range r.Effects {
			if e.OperationID == chosen.OperationID && e.AdapterResult != nil {
				p := e.AdapterResult.Candidate.Payload
				fmt.Fprintf(&text, "Open-Meteo forecast at the start/end hours — WMO codes %d/%d, %s/%s %s (grid %s, %s).\n",
					p[openmeteo.StartWeatherCode].Number, p[openmeteo.EndWeatherCode].Number,
					p[openmeteo.StartTemperature].Text, p[openmeteo.EndTemperature].Text, p[openmeteo.TemperatureUnit].Text,
					p[openmeteo.GridLatitude].Text, p[openmeteo.GridLongitude].Text)
				break
			}
		}
	} else {
		fmt.Fprintln(&text, r.Outcome)
	}
	fmt.Fprintln(&text, r.Explanation)
	for _, gap := range r.EvidenceGaps {
		fmt.Fprintf(&text, "Forecast unavailable for %s.\n", gap)
	}
	for _, run := range r.Runs {
		fmt.Fprintf(&text, "%s generation %d is %s at PC %d", run.ID.Subject, run.ID.Generation, run.Projection.Status, run.Projection.PC)
		if run.Wait != nil {
			fmt.Fprintf(&text, ", awaiting %s from %s", run.Wait.Obligation, run.Wait.Source)
		}
		fmt.Fprintln(&text, ".")
	}
	if r.Replay != nil {
		fmt.Fprintf(&text, "Replayed %d retained reductions offline.\n", r.Replay.HistoryCount)
	}
	_, err := io.WriteString(out, text.String())
	return err
}
