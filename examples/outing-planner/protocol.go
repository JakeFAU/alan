package main

import (
	"maps"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

const (
	outingSubject  = "outing"
	weatherSubject = "weather"
	capability     = "open-meteo-hourly"
	effectName     = "forecast"

	assessFirst  = "assess-first"
	confirmFirst = "confirm-first"

	proposalPort   = "outing-proposal"
	assessmentPort = "outing-assessment"
	decisionPort   = "outing-decision"

	recommended = "recommended"
	noMatch     = "no_match"
	unavailable = "unavailable"
	suitable    = "suitable"
	unsuitable  = "unsuitable"
)

var (
	idField         = schema.Field{Kind: "string", MaxBytes: 64}
	operationField  = schema.Field{Kind: "string", MaxBytes: 128}
	hourlyInput     = openmeteo.HourlyInputSchema()
	candidateFields = schema.Record{
		"id": idField, "park_id": {Kind: "string", MaxBytes: 32}, "name": {Kind: "string", MaxBytes: maxParkName},
		openmeteo.Latitude: hourlyInput[openmeteo.Latitude], openmeteo.Longitude: hourlyInput[openmeteo.Longitude],
		openmeteo.StartHour: hourlyInput[openmeteo.StartHour], openmeteo.EndHour: hourlyInput[openmeteo.EndHour],
		"travel_minutes": {Kind: "int"},
	}
	rowFields = extendRecord(candidateFields, schema.Record{
		"eligible": {Kind: "bool"},
		"reason":   {Kind: "string", MaxBytes: 64},
	})
	assessmentFields = extendRecord(candidateFields, schema.Record{
		"forecast_operation": operationField,
		"weather_status":     {Kind: "string", MaxBytes: 11, Enum: []string{suitable, unsuitable, unavailable}},
	})
	candidateList  = listField(candidateFields, "id")
	allRowsList    = listField(rowFields, "id")
	assessmentList = listField(assessmentFields, "id")
	weatherCodes   = schema.Field{
		Kind: "list", MaxItems: 32, MaxBytes: 2048, Element: schema.Record{"code": {Kind: "int"}},
	}
	outcomeField     = schema.Field{Kind: "string", MaxBytes: 11, Enum: []string{recommended, noMatch, unavailable}}
	proposalSchema   = schema.Record{"proposal_id": idField, "candidates": candidateList}
	assessmentSchema = schema.Record{"proposal_id": idField, "assessments": assessmentList}
	decisionSchema   = extendRecord(proposalSchema, schema.Record{"outcome": outcomeField, "selection": assessmentList})
)

func listField(fields schema.Record, unique string) schema.Field {
	return schema.Field{Kind: "list", MaxItems: maxCandidates, MaxBytes: 32 << 10, Element: fields, UniqueKey: unique}
}

func extendRecord(base, extra schema.Record) schema.Record {
	combined := maps.Clone(base)
	maps.Copy(combined, extra)
	return combined
}

func candidateRecord(row candidate) map[string]schema.Value {
	return map[string]schema.Value{
		"id": schema.Text(row.ID), "park_id": schema.Text(row.ParkID), "name": schema.Text(row.Name),
		openmeteo.Latitude: schema.Text(row.Latitude), openmeteo.Longitude: schema.Text(row.Longitude),
		openmeteo.StartHour: schema.Text(row.Start), openmeteo.EndHour: schema.Text(row.End),
		"travel_minutes": schema.Int(int64(row.TravelMinutes)),
	}
}

func recordBindings(fields schema.Record, prefix string) map[string]string {
	bindings := make(map[string]string, len(fields))
	for field := range fields {
		bindings[field] = prefix + field
	}
	return bindings
}

func recordExpressions(fields schema.Record, prefix string) map[string]author.Expr {
	expressions := make(map[string]author.Expr, len(fields))
	for field := range fields {
		expressions[field] = author.Var(prefix + field)
	}
	return expressions
}

func peerPorts(weather bool) map[string]author.Port {
	if weather {
		return map[string]author.Port{
			proposalPort:   {Peer: outingSubject, Obligation: proposalPort, Receive: proposalSchema},
			assessmentPort: {Peer: outingSubject, Obligation: assessmentPort, Send: assessmentSchema},
			decisionPort:   {Peer: outingSubject, Obligation: decisionPort, Receive: decisionSchema},
		}
	}
	return map[string]author.Port{
		proposalPort:   {Peer: weatherSubject, Obligation: proposalPort, Send: proposalSchema},
		assessmentPort: {Peer: weatherSubject, Obligation: assessmentPort, Receive: assessmentSchema},
		decisionPort:   {Peer: weatherSubject, Obligation: decisionPort, Send: decisionSchema},
	}
}
