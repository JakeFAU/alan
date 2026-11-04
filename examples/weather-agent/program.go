package main

import (
	"fmt"
	"maps"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

const (
	capability = "open-meteo"
	effectName = "forecast"
	approval   = "approve-refresh"
	maxReads   = 8
)

const (
	afterReservation = "new_read_after_undispatched_reservation"
	afterUnknown     = "new_read_after_unknown_response"
	afterUnavailable = "new_read_after_unavailable_result"
	afterReport      = "refresh_after_report"
)

type configuration struct {
	Latitude, Longitude, Units, Endpoint string
	Allowance                            int
}

func readID(n int) runtime.RunID {
	return runtime.RunID{Subject: fmt.Sprintf("weather-%d", n), Generation: 1}
}

// weatherDefinition describes one original observation.
func weatherDefinition() author.Definition {
	request := openmeteo.InputSchema()
	output := openmeteo.OutputSchema()
	inputs := maps.Clone(request)
	inputs["endpoint"] = schema.Field{Kind: "string", MaxBytes: 2048}
	inputs["allowance"] = schema.Field{Kind: "int"}
	var steps []author.Step
	steps = append(steps, author.Request("operation_id", effectName, map[string]author.Expr{
		openmeteo.Latitude: author.Var(openmeteo.Latitude), openmeteo.Longitude: author.Var(openmeteo.Longitude),
		openmeteo.Units: author.Var(openmeteo.Units),
	}, nil))
	bindings := map[string]string{schema.ResultValid: "valid", schema.ResultDisposition: "disposition"}
	for field := range output {
		bindings[field] = field
	}
	steps = append(steps, author.Await(effectName, "operation_id", bindings),
		author.IfEqual(author.Var("valid"), author.Boolean(true), nil, []author.Step{author.Failure()}),
		author.IfEqual(author.Var("disposition"), author.String("validated"), nil, []author.Step{author.Failure()}),
	)
	// An HTTP success is insufficient. Only the typed, validated observation
	// can supply every field of this program's retained report.
	for _, field := range []string{openmeteo.Source, openmeteo.ObservedAt, openmeteo.GridLatitude, openmeteo.GridLongitude,
		openmeteo.Temperature, openmeteo.TemperatureUnit, openmeteo.Humidity, openmeteo.HumidityUnit} {
		steps = append(steps, author.IfEqual(author.Var(field), author.String(""), []author.Step{author.Failure()}, nil))
	}
	report := maps.Clone(output)
	maps.Copy(report, request)
	return author.Definition{
		Name: "weather-read", Inputs: inputs, Outputs: report,
		Effects: map[string]author.Effect{effectName: {Capability: capability, Input: request, Output: output}},
		Steps:   append(steps, author.Success()),
	}
}

func (c configuration) series() (author.Series, error) {
	if err := openmeteo.ValidateLocation(c.Latitude, c.Longitude, c.Units); err != nil {
		return author.Series{}, err
	}
	if _, err := openmeteo.New(c.Endpoint); err != nil {
		return author.Series{}, err
	}
	return author.BoundedSeries(weatherDefinition(), author.SeriesOptions{
		ID: "weather", Count: c.Allowance, First: author.ApproveAtProvision,
		Inputs:          map[string]schema.Value{openmeteo.Latitude: schema.Text(c.Latitude), openmeteo.Longitude: schema.Text(c.Longitude), openmeteo.Units: schema.Text(c.Units), "endpoint": schema.Text(c.Endpoint), "allowance": schema.Int(int64(c.Allowance))},
		ApprovalReasons: []string{afterReservation, afterUnknown, afterUnavailable, afterReport},
	})
}

func (c configuration) provision(legacy bool) (runtime.Provision, error) {
	if c.Allowance < 0 || c.Allowance > maxReads {
		return runtime.Provision{}, fmt.Errorf("allowance must be between 0 and %d", maxReads)
	}
	if legacy || c.Allowance == 0 {
		return c.legacyProvision(legacy)
	}
	series, err := c.series()
	if err != nil {
		return runtime.Provision{}, err
	}
	return series.Provision()
}
