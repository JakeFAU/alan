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
	if variant != assessFirst && variant != confirmFirst {
		return runtime.Provision{}, fmt.Errorf("unsupported Weather variant %q", variant)
	}
	plan, err := c.normalize()
	if err != nil {
		return runtime.Provision{}, err
	}
	raw, err := c.serialized()
	if err != nil {
		return runtime.Provision{}, err
	}
	outing, err := author.Compile(outingDefinition())
	if err != nil {
		return runtime.Provision{}, fmt.Errorf("compile Outing: %w", err)
	}
	weather, err := author.Compile(weatherDefinition(variant))
	if err != nil {
		return runtime.Provision{}, fmt.Errorf("compile Weather: %w", err)
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
	p.Composition = &runtime.CompositionProvision{
		RecordDispatchAdmissions: true,
		Roots: []runtime.ResourceRoot{
			{Run: outingRun, Members: []runtime.RunID{outingRun}, Limits: runtime.ResourceLimits{MaxInFlight: 1}},
			{
				Run: weatherRun, Members: []runtime.RunID{weatherRun}, Capabilities: []string{capability},
				Limits: runtime.ResourceLimits{MaxInFlight: 1, MaxEffects: opportunities},
			},
		},
	}
	return p, nil
}

// retainedConfiguration reauthors the exact supported deployment before a
// process can accept evidence or supply transport. InspectExecution has already
// verified the content-addressed artifacts and the namespace-bound root IDs.
func retainedConfiguration(s *runtime.ExecutionSnapshot) (configuration, string, error) {
	if s == nil || s.Deployment == nil || s.Composition == nil || !s.Composition.RecordDispatchAdmissions ||
		s.Composition.RequestAuthority != "" || len(s.Composition.Templates) != 0 ||
		len(s.Composition.Continuations) != 0 || s.Reconciliation != nil || len(s.Assets) != 0 ||
		len(s.Invocations) != 0 || len(s.LifecycleDecisions) != 0 {
		return configuration{}, "", errors.New("expected the original static outing deployment")
	}
	outing, ok := s.Run(outingRun)
	if !ok {
		return configuration{}, "", errors.New("missing original Outing generation")
	}
	c, err := decodeConfiguration([]byte(outing.Initial.Data[configInput].Text))
	if err != nil {
		return c, "", err
	}
	weather, ok := s.Run(weatherRun)
	if !ok {
		return c, "", errors.New("missing original Weather generation")
	}
	for _, variant := range []string{assessFirst, confirmFirst} {
		want, err := c.provision(variant)
		if err != nil {
			return c, "", fmt.Errorf("invalid retained outing configuration: %w", err)
		}
		if weather.ProgramID == want.Runs[1].ProgramID {
			return c, variant, verifyDeployment(s, want)
		}
	}
	return c, "", errors.New("unsupported retained Weather artifact")
}

func verifyDeployment(s *runtime.ExecutionSnapshot, want runtime.Provision) error {
	d := s.Deployment
	if !slices.Equal(d.Subjects, want.Subjects) || !reflect.DeepEqual(d.Runs, want.Runs) ||
		!slices.Equal(d.Permissions, want.Permissions) || !samePermissions(s.Permissions, want.Permissions) ||
		len(s.Runs) != len(want.Runs) || len(s.Composition.Roots) != len(want.Composition.Roots) {
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
			len(root.Templates) != 0 || root.Limits != expected.Limits {
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
