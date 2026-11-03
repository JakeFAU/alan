package main

import (
	"fmt"
	"maps"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Frozen M4 handoff and M5 protected-continuation producers. Existing stores
// are recognized only against these exact declarations.
// Each new read selects its own protected approval before issuing a request.
func legacyReadProgram(n, count int, legacy bool) (author.Artifact, error) {
	request := openmeteo.InputSchema()
	output := openmeteo.OutputSchema()
	inputs := maps.Clone(request)
	inputs["endpoint"] = schema.Field{Kind: "string", MaxBytes: 2048}
	inputs["allowance"] = schema.Field{Kind: "int"}
	ports := map[string]author.Port{}
	var steps []author.Step
	if legacy {
		steps = legacyReadPrefix(n, ports)
	} else if n > 1 {
		ports["approval"] = author.ContinuationPort(approval, afterReservation, afterUnknown, afterUnavailable, afterReport)
		steps = append(steps, author.Receive("approval", map[string]string{
			"previous_operation_id": "previous_operation_id", "purpose": "approval_reason",
		}))
	}
	steps = append(steps, author.Request("operation_id", effectName, map[string]author.Expr{
		openmeteo.Latitude: author.Var(openmeteo.Latitude), openmeteo.Longitude: author.Var(openmeteo.Longitude),
		openmeteo.Units: author.Var(openmeteo.Units),
	}, nil))
	if legacy && n < count {
		steps = append(steps, legacySendHandle(n, ports))
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

func (c configuration) legacyProvision(legacy bool) (runtime.Provision, error) {
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
		program, err := legacyReadProgram(n, max(1, c.Allowance), legacy)
		if err != nil {
			return runtime.Provision{}, err
		}
		app[id.Subject] = author.Instance{Artifact: program, Inputs: input}
		if !legacy && n > 1 {
			profile.Continuations = append(profile.Continuations, runtime.ContinuationSlot{
				Predecessor: readID(n - 1), Target: id, Effect: effectName, Obligation: approval, ApprovalID: approvalID(n),
			})
		}
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
