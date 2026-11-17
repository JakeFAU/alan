package main

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

var (
	outingRun  = runtime.RunID{Subject: outingSubject, Generation: 1}
	weatherRun = runtime.RunID{Subject: weatherSubject, Generation: 1}
)

func (c configuration) provision(variant string) (runtime.Provision, error) {
	return c.provisionDeployment(defaultDeploymentOptions(variant))
}

func (c configuration) provisionDeployment(options deploymentOptions) (runtime.Provision, error) {
	if err := options.validate(); err != nil {
		return runtime.Provision{}, err
	}
	plan, err := c.normalize()
	if err != nil {
		return runtime.Provision{}, err
	}
	raw, err := c.serialized()
	if err != nil {
		return runtime.Provision{}, err
	}
	rows := make([]map[string]schema.Value, 0, len(plan.Candidates))
	var opportunities int64
	for _, row := range plan.Candidates {
		record := candidateRecord(row)
		record["eligible"] = schema.Bool(row.Eligible)
		record["reason"] = schema.Text(row.Reason)
		rows = append(rows, record)
		if row.Eligible {
			opportunities++
		}
	}
	if options.Weather == delegatedWeather && opportunities > maxDelegatedCandidates {
		return runtime.Provision{}, errors.New("delegated Weather supports at most two eligible candidates")
	}
	outing, err := author.Compile(outingDefinition())
	if err != nil {
		return runtime.Provision{}, fmt.Errorf("compile Outing: %w", err)
	}
	var helper author.Template
	var definition author.Definition
	if options.Weather == delegatedWeather {
		helper, err = weatherHelperTemplate()
		if err != nil {
			return runtime.Provision{}, fmt.Errorf("retain Weather helper: %w", err)
		}
		definition = delegatedWeatherDefinition(helper)
	} else {
		definition = weatherDefinition(options.Weather)
	}
	weather, err := author.Compile(definition)
	if err != nil {
		return runtime.Provision{}, fmt.Errorf("compile Weather: %w", err)
	}
	codes := make([]map[string]schema.Value, 0, len(c.allowedCodes()))
	for _, code := range c.allowedCodes() {
		codes = append(codes, map[string]schema.Value{"code": schema.Int(code)})
	}
	id := schema.Text(digestText(string(raw)))
	app := author.Application{
		outingSubject: {
			Artifact: outing, Inputs: map[string]schema.Value{
				configInput: schema.Text(string(raw)), dateInput: schema.Text(plan.Date),
				proposalID: id, rowsInput: schema.List(rows...),
			},
		},
		weatherSubject: {
			Artifact: weather, Inputs: map[string]schema.Value{
				proposalID: id, openmeteo.Units: schema.Text(c.Units),
				"allowed_weather_codes": schema.List(codes...),
			},
		},
	}
	p, err := app.Provision()
	if err != nil {
		return runtime.Provision{}, err
	}
	if p.Composition == nil {
		p.Composition = &runtime.CompositionProvision{}
	}
	p.Composition.RecordDispatchAdmissions = true
	root := runtime.ResourceRoot{
		Run: weatherRun, Members: []runtime.RunID{weatherRun}, Capabilities: []string{capability},
		Limits: runtime.ResourceLimits{MaxInFlight: 1, MaxEffects: opportunities},
	}
	if options.Weather == delegatedWeather {
		root.Templates = []string{helper.ID()}
		root.Limits.MaxInvocations = options.HelperAdmission
		root.Limits.MaxCreatedRuns = 1
		root.Limits.MaxDepth = 1
	}
	p.Composition.Roots = []runtime.ResourceRoot{
		{Run: outingRun, Members: []runtime.RunID{outingRun}, Limits: runtime.ResourceLimits{MaxInFlight: 1}},
		root,
	}
	return p, nil
}

func retainedConfiguration(s *runtime.ExecutionSnapshot) (configuration, string, error) {
	info, err := retainedDeployment(s)
	return info.Configuration, info.Options.Weather, err
}

func verifyDeployment(s *runtime.ExecutionSnapshot, want runtime.Provision) error {
	if len(s.Runs) != len(want.Runs) || !samePermissions(s.Permissions, want.Permissions) {
		return errors.New("retained outing programs, inputs or grants differ")
	}
	return verifyInitialDeployment(s, want)
}

func verifyInitialDeployment(s *runtime.ExecutionSnapshot, want runtime.Provision) error {
	d := s.Deployment
	if !slices.Equal(d.Subjects, want.Subjects) || !reflect.DeepEqual(d.Runs, want.Runs) ||
		!slices.Equal(d.Permissions, want.Permissions) || len(s.Composition.Roots) != len(want.Composition.Roots) {
		return errors.New("retained outing programs, inputs or grants differ")
	}
	for i, expected := range want.Composition.Roots {
		position := slices.IndexFunc(s.Composition.Roots, func(root runtime.ResourceRoot) bool {
			return root.Run == expected.Run
		})
		if position < 0 {
			return errors.New("missing retained outing resource scope")
		}
		root := s.Composition.Roots[position]
		r, ok := s.Run(expected.Run)
		if !ok || r.Origin != "root" || root.ID == "" || r.RootID != root.ID ||
			r.Subject != want.Subjects[i] || r.ProgramID != want.Runs[i].ProgramID ||
			!reflect.DeepEqual(r.Initial, want.Runs[i].Projection) ||
			!slices.Equal(root.Members, expected.Members) || !slices.Equal(root.Capabilities, expected.Capabilities) ||
			!slices.Equal(root.Templates, expected.Templates) || root.Limits != expected.Limits {
			return errors.New("retained outing resource scope differs")
		}
	}
	return nil
}

func samePermissions(got, want []runtime.Permission) bool {
	if len(got) != len(want) {
		return false
	}
	remaining := make(map[runtime.Permission]bool, len(want))
	for _, g := range want {
		remaining[g] = true
	}
	for _, g := range got {
		if !remaining[g] {
			return false
		}
		delete(remaining, g)
	}
	return len(remaining) == 0
}
