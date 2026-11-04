package main

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Accept the original weather application and its exact inputs, grants and
// one-opportunity scopes before giving an owner any transport capability.
func retainedConfiguration(s *runtime.ExecutionSnapshot) (configuration, error) {
	var c configuration
	first, ok := s.Run(readID(1))
	if !ok || s.Deployment == nil || s.Composition == nil || !s.Composition.RecordDispatchAdmissions || s.Reconciliation != nil || len(s.Composition.Templates) != 0 || len(s.Assets) != 0 {
		return c, errors.New("expected an original weather deployment")
	}
	data := first.Initial.Data
	c = configuration{
		Latitude: data[openmeteo.Latitude].Text, Longitude: data[openmeteo.Longitude].Text, Units: data[openmeteo.Units].Text,
		Endpoint: data["endpoint"].Text, Allowance: int(data["allowance"].Number),
	}
	// Only the exact new declaration or an exact historical producer may own
	// this store. A damaged series never falls back to a heuristic recognizer.
	if c.Allowance > 0 {
		series, err := c.series()
		if err == nil {
			if _, err = series.Inspect(s); err == nil {
				return c, nil
			}
		}
	}
	want, err := c.legacyProvision(len(s.Composition.Continuations) == 0)
	if err != nil {
		return c, fmt.Errorf("retained weather configuration: %w", err)
	}
	if len(s.Composition.Continuations) != 0 && !slices.Equal(s.Composition.Continuations, want.Composition.Continuations) {
		return c, errors.New("original weather continuation slots differ")
	}
	d := s.Deployment
	if !slices.Equal(d.Subjects, want.Subjects) || !reflect.DeepEqual(d.Runs, want.Runs) || !slices.Equal(d.Permissions, want.Permissions) ||
		len(s.Runs) != len(want.Runs) || len(s.Composition.Roots) != len(want.Runs) {
		return c, errors.New("original weather programs, inputs or grants differ")
	}
	for _, root := range s.Composition.Roots {
		r, ok := s.Run(root.Run)
		if !ok || r.Origin != "root" || r.RootID != root.ID || len(root.Members) != 1 || root.Members[0] != root.Run ||
			!slices.Equal(root.Capabilities, []string{capability}) || len(root.Templates) != 0 || root.Limits != want.Composition.Roots[0].Limits {
			return c, errors.New("original weather resource scope differs")
		}
	}
	return c, nil
}

// The typed runtime ties a result to its original operation. This application
// additionally binds the native GET and successful payload to this deployment.
func verifyWeatherEvidence(s *runtime.ExecutionSnapshot, c configuration) error {
	adapter, err := openmeteo.New(c.Endpoint)
	if err != nil {
		return err
	}
	for _, e := range s.Effects {
		if e.Kind != "adapter" || e.Capability != capability || e.Name != effectName {
			return errors.New("unexpected operation in weather deployment")
		}
		op, ok := s.Operation(e.OperationID)
		if !ok || len(op.Payload) != 3 || op.Payload[openmeteo.Latitude] != schema.Text(c.Latitude) ||
			op.Payload[openmeteo.Longitude] != schema.Text(c.Longitude) || op.Payload[openmeteo.Units] != schema.Text(c.Units) {
			return fmt.Errorf("original weather request %s differs from its provisioned parameters", e.OperationID)
		}
		a := e.AdapterAttempt
		if a == nil || !a.Allowed {
			continue
		}
		want, err := adapter.Prepare(a.Prepared.Request)
		if err != nil || want.Digest != a.Prepared.Digest {
			return fmt.Errorf("weather reservation %s differs from the configured request", e.OperationID)
		}
		if r := e.AdapterResult; r != nil {
			if err := openmeteo.VerifyResponseEvidence(a.Prepared, r.Candidate); err != nil {
				return fmt.Errorf("weather response %s: %w", e.OperationID, err)
			}
		}
	}
	return nil
}
