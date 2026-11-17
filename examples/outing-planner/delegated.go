package main

import (
	"fmt"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

const (
	delegatedWeather       = "delegated"
	helperMember           = "worker"
	helperPort             = "work"
	helperPeer             = "caller"
	helperObligation       = "assess"
	helperCall             = "helper_call"
	maxDelegatedCandidates = 2
)

func helperRequestSchema() schema.Record {
	return extendRecord(proposalSchema, schema.Record{
		openmeteo.Units: hourlyInput[openmeteo.Units], "allowed_weather_codes": weatherCodes,
	})
}

func weatherHelperTemplate() (author.Template, error) {
	request := helperRequestSchema()
	worker, err := author.Compile(author.Definition{
		Name: "weekend-weather-helper",
		Ports: map[string]author.Port{
			helperPort: {Peer: helperPeer, Obligation: helperObligation, Receive: request, Send: assessmentSchema},
		},
		Effects: map[string]author.Effect{
			effectName: {Capability: capability, Input: openmeteo.HourlyInputSchema(), Output: openmeteo.HourlyOutputSchema()},
		},
		Collections: map[string]schema.Field{assessments: assessmentList},
		Steps: []author.Step{
			author.Receive(helperPort, recordBindings(request, "")),
			assessWeather(),
			author.Send(helperPort, recordExpressions(assessmentSchema, "")),
			author.Success(),
		},
	})
	if err != nil {
		return author.Template{}, fmt.Errorf("compile Weather helper: %w", err)
	}
	return author.NewTemplate(author.Application{helperMember: {Artifact: worker}}, helperMember, helperPort)
}

func delegatedWeatherDefinition(helper author.Template) author.Definition {
	const response = "helper_response_"
	failure := []author.Step{author.Failure()}
	return author.Definition{
		Name: "weekend-weather-delegated",
		Inputs: schema.Record{
			proposalID: idField, openmeteo.Units: hourlyInput[openmeteo.Units], "allowed_weather_codes": weatherCodes,
		},
		Ports:       peerPorts(true),
		Invocations: map[string]author.Template{helperObligation: helper},
		Collections: map[string]schema.Field{
			"assessed_candidates":   candidateList,
			"assessment_operations": listField(schema.Record{"forecast_operation": operationField}, "forecast_operation"),
			selection:               assessmentList,
		},
		Values:  map[string]schema.Value{outcome: schema.Text(noMatch)},
		Outputs: decisionSchema,
		Steps: []author.Step{
			author.Receive(proposalPort, map[string]string{proposalID: "received_proposal_id", candidates: candidates}),
			requireEqual(proposalID, "received_proposal_id"),
			author.Invoke(helperCall, helperObligation, recordExpressions(helperRequestSchema(), ""), nil),
			author.AwaitInvocation(helperObligation, helperCall, author.InvocationOutcomeBindings{
				Status: "helper_status", Reason: "helper_reason", ResponsePresent: "helper_response_present",
				Response: recordBindings(assessmentSchema, response),
			}),
			author.IfEqual(author.Var("helper_status"), author.String("succeeded"), nil, failure),
			author.IfEqual(author.Var("helper_response_present"), author.Boolean(true), nil, failure),
			requireEqual(proposalID, response+proposalID),
			author.ForEach(author.Var(response+assessments), recordBindings(assessmentFields, "assessment_"),
				author.IfEqual(author.Var("assessment_forecast_operation"), author.String(""), failure, nil),
				author.ForEach(author.Var("assessment_operations"), map[string]string{"forecast_operation": "prior_forecast"},
					author.IfEqual(author.Var("assessment_forecast_operation"), author.Var("prior_forecast"), failure, nil)),
				author.Append("assessment_operations", map[string]author.Expr{
					"forecast_operation": author.Var("assessment_forecast_operation"),
				}),
				author.Append("assessed_candidates", recordExpressions(candidateFields, "assessment_")),
			),
			requireEqual(candidates, "assessed_candidates"),
			author.Set(assessments, author.Var(response+assessments)),
			author.Send(assessmentPort, recordExpressions(assessmentSchema, "")),
			author.Receive(decisionPort, recordBindings(decisionSchema, "decision_")),
			requireEqual(proposalID, "decision_"+proposalID),
			requireEqual(candidates, "decision_"+candidates),
			selectAssessment(),
			requireEqual(outcome, "decision_"+outcome),
			requireEqual(selection, "decision_"+selection),
			finishDecision(),
		},
	}
}
