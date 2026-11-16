package main

import (
	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

const (
	rowsInput   = "normalized_candidates"
	configInput = "configuration"
	dateInput   = "normalized_date"
	proposalID  = "proposal_id"
	candidates  = "candidates"
	assessments = "assessments"
	outcome     = "outcome"
	selection   = "selection"
)

func outingDefinition() author.Definition {
	operations := listField(schema.Record{"forecast_operation": operationField}, "forecast_operation")
	steps := []author.Step{
		author.ForEach(author.Var(rowsInput), recordBindings(rowFields, "row_"),
			author.IfEqual(author.Var("row_eligible"), author.Boolean(true), []author.Step{
				author.Append(candidates, recordExpressions(candidateFields, "row_")),
			}, nil)),
		author.Send(proposalPort, recordExpressions(proposalSchema, "")),
		author.Receive(assessmentPort, map[string]string{proposalID: "assessment_id", assessments: assessments}),
		requireEqual(proposalID, "assessment_id"),
		author.ForEach(author.Var(assessments), recordBindings(assessmentFields, "assessment_"),
			author.IfEqual(author.Var("assessment_forecast_operation"), author.String(""),
				[]author.Step{author.Failure()}, nil),
			author.ForEach(author.Var("assessment_operations"), map[string]string{"forecast_operation": "prior_forecast"},
				author.IfEqual(author.Var("assessment_forecast_operation"), author.Var("prior_forecast"),
					[]author.Step{author.Failure()}, nil)),
			author.Append("assessment_operations", map[string]author.Expr{
				"forecast_operation": author.Var("assessment_forecast_operation"),
			}),
			author.Append("assessed_candidates", recordExpressions(candidateFields, "assessment_"))),
		// Ordered structural equality checks exact identities, correspondence and
		// coverage. No recommendation is committed from an invalid batch.
		requireEqual(candidates, "assessed_candidates"),
	}
	steps = append(steps, selectAssessment(), author.Send(decisionPort, recordExpressions(decisionSchema, "")), finishDecision())
	return author.Definition{
		Name: "weekend-outing",
		Inputs: schema.Record{
			configInput: {Kind: "string", MaxBytes: maxConfiguration},
			dateInput:   {Kind: "string", MaxBytes: 10},
			proposalID:  idField,
			rowsInput:   allRowsList,
		},
		Ports: peerPorts(false),
		Collections: map[string]schema.Field{
			candidates: candidateList, "assessed_candidates": candidateList,
			"assessment_operations": operations, selection: assessmentList,
		},
		Values:  map[string]schema.Value{outcome: schema.Text(noMatch)},
		Outputs: decisionSchema,
		Steps:   steps,
	}
}

func weatherDefinition(variant string) author.Definition {
	receiveDecision := []author.Step{
		author.Receive(decisionPort, recordBindings(decisionSchema, "decision_")),
		requireEqual(proposalID, "decision_"+proposalID),
		requireEqual(candidates, "decision_"+candidates),
	}
	steps := []author.Step{
		author.Receive(proposalPort, map[string]string{proposalID: "received_proposal_id", candidates: candidates}),
		requireEqual(proposalID, "received_proposal_id"),
	}
	if variant == confirmFirst {
		steps = append(steps, receiveDecision...)
	}
	steps = append(steps,
		assessWeather(),
		author.Send(assessmentPort, recordExpressions(assessmentSchema, "")),
	)
	if variant == assessFirst {
		steps = append(steps, receiveDecision...)
	}
	steps = append(steps,
		selectAssessment(),
		requireEqual(outcome, "decision_"+outcome),
		requireEqual(selection, "decision_"+selection),
		finishDecision(),
	)
	return author.Definition{
		// Both variants use the same receive and validation steps.
		Name: "weekend-weather",
		Inputs: schema.Record{
			proposalID: idField, openmeteo.Units: hourlyInput[openmeteo.Units], "allowed_weather_codes": weatherCodes,
		},
		Effects: map[string]author.Effect{effectName: {
			Capability: capability, Input: openmeteo.HourlyInputSchema(), Output: openmeteo.HourlyOutputSchema(),
		}},
		Ports:       peerPorts(true),
		Collections: map[string]schema.Field{assessments: assessmentList, selection: assessmentList},
		Values:      map[string]schema.Value{outcome: schema.Text(noMatch)},
		Outputs:     decisionSchema,
		Steps:       steps,
	}
}

func requireEqual(left, right string) author.Step {
	return author.IfEqual(author.Var(left), author.Var(right), nil, []author.Step{author.Failure()})
}

func assessWeather() author.Step {
	request := map[string]author.Expr{
		openmeteo.Latitude:  author.Var("candidate_" + openmeteo.Latitude),
		openmeteo.Longitude: author.Var("candidate_" + openmeteo.Longitude),
		openmeteo.StartHour: author.Var("candidate_" + openmeteo.StartHour),
		openmeteo.EndHour:   author.Var("candidate_" + openmeteo.EndHour),
		openmeteo.Units:     author.Var(openmeteo.Units),
	}
	assessment := recordExpressions(candidateFields, "candidate_")
	assessment["forecast_operation"] = author.Var("original_forecast")
	assessment["weather_status"] = author.Var("weather_status")
	return author.ForEach(author.Var(candidates), recordBindings(candidateFields, "candidate_"),
		author.Request("original_forecast", effectName, request, nil),
		author.Await(effectName, "original_forecast", map[string]string{
			schema.ResultValid: "valid_forecast", openmeteo.StartWeatherCode: "start_code", openmeteo.EndWeatherCode: "end_code",
		}),
		author.Set("weather_status", author.String(unavailable)),
		author.IfEqual(author.Var("valid_forecast"), author.Boolean(true), evaluateWeatherCodes(), nil),
		author.Append(assessments, assessment),
	)
}

func evaluateWeatherCodes() []author.Step {
	return []author.Step{
		author.Set("weather_status", author.String(unsuitable)),
		author.Set("start_allowed", author.Boolean(false)),
		author.Set("end_allowed", author.Boolean(false)),
		author.ForEach(author.Var("allowed_weather_codes"), map[string]string{"code": "allowed_code"},
			author.IfEqual(author.Var("start_code"), author.Var("allowed_code"), []author.Step{
				author.Set("start_allowed", author.Boolean(true)),
			}, nil),
			author.IfEqual(author.Var("end_code"), author.Var("allowed_code"), []author.Step{
				author.Set("end_allowed", author.Boolean(true)),
			}, nil),
		),
		author.IfEqual(author.Var("start_allowed"), author.Boolean(true), []author.Step{
			author.IfEqual(author.Var("end_allowed"), author.Boolean(true), []author.Step{
				author.Set("weather_status", author.String(suitable)),
			}, nil),
		}, nil),
	}
}

func selectAssessment() author.Step {
	// The list is already in rank order. Keep the first usable match and
	// require an exhaustive unsuitable batch before reporting no-match.
	return author.ForEach(author.Var(assessments), recordBindings(assessmentFields, "choice_"),
		author.IfEqual(author.Var(outcome), author.String(recommended), nil, []author.Step{
			author.IfEqual(author.Var("choice_weather_status"), author.String(suitable), []author.Step{
				author.Set(outcome, author.String(recommended)),
				author.Append(selection, recordExpressions(assessmentFields, "choice_")),
			}, []author.Step{
				author.IfEqual(author.Var("choice_weather_status"), author.String(unavailable), []author.Step{
					author.Set(outcome, author.String(unavailable)),
				}, nil),
			}),
		}),
	)
}

func finishDecision() author.Step {
	return author.IfEqual(author.Var(outcome), author.String(unavailable),
		[]author.Step{author.Failure()}, []author.Step{author.Success()})
}
