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

// Each read is an ordinary authored computation. It sends its original handle
// before waiting, so an operator can later authorize a distinct observation
// even if this read never gets a response.
func readProgram(n, count int) (author.Artifact, error) {
	request := openmeteo.InputSchema()
	output := openmeteo.OutputSchema()
	inputs := maps.Clone(request)
	inputs["endpoint"] = schema.Field{Kind: "string", MaxBytes: 2048}
	inputs["allowance"] = schema.Field{Kind: "int"}
	handle := schema.Record{"previous_operation_id": {Kind: "string", MaxBytes: 4096}}
	ports := map[string]author.Port{}
	var steps []author.Step
	if n > 1 {
		consent := maps.Clone(handle)
		consent["purpose"] = schema.Field{Kind: "string", MaxBytes: 64, Enum: []string{afterReservation, afterUnknown, afterUnavailable, afterReport}}
		consent["prepared_digest"] = schema.Field{Kind: "string", MaxBytes: 64}
		consent["result_digest"] = schema.Field{Kind: "string", MaxBytes: 64}
		ports["previous"] = author.Port{Peer: readID(n - 1).Subject, Obligation: "previous-read", Receive: handle}
		ports["approval"] = author.Port{Peer: "local", Local: true, Obligation: approval, Receive: consent}
		steps = append(steps,
			author.Receive("previous", map[string]string{"previous_operation_id": "previous_operation_id"}),
			author.Receive("approval", map[string]string{"previous_operation_id": "approved_operation_id", "purpose": "approval_reason"}),
			author.IfEqual(author.Var("previous_operation_id"), author.Var("approved_operation_id"), nil, []author.Step{author.Failure()}),
		)
	}
	steps = append(steps, author.Request("operation_id", effectName, map[string]author.Expr{
		openmeteo.Latitude: author.Var(openmeteo.Latitude), openmeteo.Longitude: author.Var(openmeteo.Longitude),
		openmeteo.Units: author.Var(openmeteo.Units),
	}, nil))
	if n < count {
		ports["next"] = author.Port{Peer: readID(n + 1).Subject, Obligation: "previous-read", Send: handle}
		steps = append(steps, author.Send("next", map[string]author.Expr{"previous_operation_id": author.Var("operation_id")}))
	}
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
	return author.Compile(author.Definition{
		Name: "weather-read", Inputs: inputs, Outputs: report, Ports: ports,
		Effects: map[string]author.Effect{effectName: {Capability: capability, Input: request, Output: output}},
		Steps:   append(steps, author.Success()),
	})
}

func (c configuration) provision() (runtime.Provision, error) {
	if c.Allowance < 0 || c.Allowance > maxReads {
		return runtime.Provision{}, fmt.Errorf("allowance must be between 0 and %d", maxReads)
	}
	if err := openmeteo.ValidateLocation(c.Latitude, c.Longitude, c.Units); err != nil {
		return runtime.Provision{}, err
	}
	if _, err := openmeteo.New(c.Endpoint); err != nil {
		return runtime.Provision{}, err
	}
	input := map[string]schema.Value{
		openmeteo.Latitude: schema.Text(c.Latitude), openmeteo.Longitude: schema.Text(c.Longitude), openmeteo.Units: schema.Text(c.Units),
		"endpoint": schema.Text(c.Endpoint), "allowance": schema.Int(int64(c.Allowance)),
	}
	app := author.Application{}
	profile := &runtime.CompositionProvision{RecordDispatchAdmissions: true}
	for n := 1; n <= max(1, c.Allowance); n++ {
		id := readID(n)
		program, err := readProgram(n, max(1, c.Allowance))
		if err != nil {
			return runtime.Provision{}, err
		}
		app[id.Subject] = author.Instance{Artifact: program, Inputs: input}
		// One independent resource scope per read keeps an earlier closed root
		// from closing admission to a separately approved later observation.
		profile.Roots = append(profile.Roots, runtime.ResourceRoot{
			Run: id, Members: []runtime.RunID{id}, Capabilities: []string{capability},
			Limits: runtime.ResourceLimits{MaxEffects: int64(min(c.Allowance, 1)), MaxInFlight: 1},
		})
	}
	p, err := app.Provision()
	p.Composition = profile
	return p, err
}
