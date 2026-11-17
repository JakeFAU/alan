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

type deploymentOptions struct {
	Weather         string
	HelperAdmission int64
}

func defaultDeploymentOptions(variant string) deploymentOptions {
	options := deploymentOptions{Weather: variant}
	if variant == delegatedWeather {
		options.HelperAdmission = 1
	}
	return options
}

func (o deploymentOptions) validate() error {
	switch o.Weather {
	case assessFirst, confirmFirst:
		if o.HelperAdmission != 0 {
			return errors.New("helper admission requires delegated Weather")
		}
	case delegatedWeather:
		if o.HelperAdmission != 0 && o.HelperAdmission != 1 {
			return errors.New("delegated Weather helper admission must be 0 or 1")
		}
	default:
		return fmt.Errorf("unsupported Weather variant %q", o.Weather)
	}
	return nil
}

type retainedDeploymentInfo struct {
	Configuration    configuration
	Options          deploymentOptions
	HelperTemplateID string
	HelperRun        *runtime.RunID
}

// retainedDeployment accepts views produced by InspectExecution, which
// authenticates the artifact bytes, namespace-bound roots and member bindings.
func retainedDeployment(s *runtime.ExecutionSnapshot) (retainedDeploymentInfo, error) {
	var info retainedDeploymentInfo
	if s == nil || s.Deployment == nil || s.Composition == nil || !s.Composition.RecordDispatchAdmissions ||
		s.Composition.RequestAuthority != "" || len(s.Composition.Continuations) != 0 ||
		len(s.Continuations) != 0 || s.Reconciliation != nil || len(s.Assets) != 0 || len(s.LifecycleDecisions) != 0 {
		return info, errors.New("expected the original outing deployment")
	}
	outing, ok := s.Run(outingRun)
	if !ok {
		return info, errors.New("missing original Outing generation")
	}
	c, err := decodeConfiguration([]byte(outing.Initial.Data[configInput].Text))
	info.Configuration = c
	if err != nil {
		return info, err
	}
	weather, ok := s.Run(weatherRun)
	if !ok {
		return info, errors.New("missing original Weather generation")
	}
	for _, variant := range []string{assessFirst, confirmFirst} {
		options := defaultDeploymentOptions(variant)
		want, err := c.provisionDeployment(options)
		if err != nil {
			return info, fmt.Errorf("invalid retained outing configuration: %w", err)
		}
		if weather.ProgramID == want.Runs[1].ProgramID {
			info.Options = options
			if len(s.Composition.Templates) != 0 || len(s.Invocations) != 0 {
				return info, errors.New("expected the original static outing deployment")
			}
			return info, verifyDeployment(s, want)
		}
	}

	position := slices.IndexFunc(s.Composition.Roots, func(root runtime.ResourceRoot) bool {
		return root.Run == weatherRun
	})
	if position < 0 {
		return info, errors.New("missing retained Weather resource scope")
	}
	info.Options = deploymentOptions{
		Weather: delegatedWeather, HelperAdmission: s.Composition.Roots[position].Limits.MaxInvocations,
	}
	want, err := c.provisionDeployment(info.Options)
	if err != nil {
		return info, fmt.Errorf("invalid retained delegated deployment: %w", err)
	}
	if weather.ProgramID != want.Runs[1].ProgramID {
		return info, errors.New("unsupported retained Weather artifact")
	}
	if err := verifyInitialDeployment(s, want); err != nil {
		return info, err
	}
	if !reflect.DeepEqual(s.Composition.Templates, want.Composition.Templates) {
		return info, errors.New("retained delegated helper catalogue differs")
	}
	template := want.Composition.Templates[0]
	info.HelperTemplateID = template.ID

	if len(s.Invocations) > 1 {
		return info, errors.New("retained Weather has extra invocations")
	}
	var call *runtime.InvocationView
	if len(s.Invocations) == 1 {
		call = s.Invocations[0]
		if call == nil {
			return info, errors.New("missing retained Weather invocation")
		}
		info.HelperRun, err = retainedHelperRun(s, want, c, weather, call, info.Options.HelperAdmission)
		if err != nil {
			return info, err
		}
	}
	allowedRuns := map[runtime.RunID]bool{outingRun: true, weatherRun: true}
	grants := want.Permissions
	if helper := info.HelperRun; helper != nil {
		allowedRuns[*helper] = true
		// Bound entry replies use the invocation protocol. The hourly adapter
		// permission is the worker's sole additional grant.
		grants = append(slices.Clone(grants), runtime.Permission{
			Kind: "adapter", Source: helper.Subject, Target: capability, Operation: effectName,
			Revision: 1, Obligation: effectName,
		})
	}
	for _, run := range s.Runs {
		if run == nil || !allowedRuns[run.ID] {
			return info, errors.New("unexpected run in retained delegated deployment")
		}
		delete(allowedRuns, run.ID)
	}
	if len(allowedRuns) != 0 || !samePermissions(s.Permissions, grants) {
		return info, errors.New("retained delegated runs or grants differ")
	}
	return info, verifyDelegatedOperations(s, info.HelperRun, call, want.Composition.Roots[1].Limits.MaxEffects)
}

func retainedHelperRun(
	s *runtime.ExecutionSnapshot,
	want runtime.Provision,
	c configuration,
	weather *runtime.ExecutionRun,
	call *runtime.InvocationView,
	allowance int64,
) (*runtime.RunID, error) {
	template := want.Composition.Templates[0]
	request, err := frozenHelperRequest(c, weather.Initial.Data)
	if err != nil {
		return nil, err
	}
	if call.OperationID == "" || call.TemplateID != template.ID || call.Caller != weatherRun ||
		call.SourceRevision != 1 || !equalPayload(call.Inputs, request) {
		return nil, errors.New("unsupported retained Weather invocation")
	}
	if auth := call.Authorization; auth != nil && (!auth.Allowed || auth.IntentID != call.OperationID ||
		auth.Source.Run != weatherRun || auth.Source.Revision != 1) {
		return nil, errors.New("retained Weather invocation authorization differs")
	}
	a := call.Admission
	if a == nil {
		if call.RequestObservationID != "" {
			return nil, errors.New("unadmitted Weather invocation has a request observation")
		}
		return nil, nil
	}
	i := a.Invocation
	if call.Authorization == nil || i.ID == "" || i.OperationID != call.OperationID || i.TemplateID != template.ID ||
		i.RootID != weather.RootID || i.Caller != weatherRun || i.ParentInvocationID != "" ||
		i.SourceSequence != call.Sequence || i.SourceRevision != 1 || i.Depth != 1 || !equalPayload(i.Inputs, request) {
		return nil, errors.New("retained Weather invocation admission differs")
	}
	if !a.Accepted {
		if allowance != 0 || a.Reason != "invocation_limit" || a.BindingID != "" ||
			len(a.Members) != 0 || call.RequestObservationID != "" {
			return nil, errors.New("unexpected retained Weather invocation refusal")
		}
		return nil, nil
	}
	if allowance != 1 || a.Reason != "accepted" || a.BindingID == "" || len(a.Members) != 1 {
		return nil, errors.New("unexpected retained Weather helper admission")
	}

	helper := a.Members[0]
	r, ok := s.Run(helper)
	expected := template.Members[helperMember]
	if !ok || helper.Generation != 1 || r.Origin != "invocation" || r.OwningInvocationID != i.ID ||
		r.OriginOperationID != call.OperationID || r.RootID != i.RootID || r.Predecessor != nil ||
		r.ProgramID != expected.ProgramID || !reflect.DeepEqual(r.Initial, expected.Initial) ||
		r.Subject != (runtime.Subject{ID: helper.Subject, Generation: 1, Owner: "invocation:" + i.ID, Revision: 1}) {
		return nil, errors.New("retained Weather helper member differs from admitted template binding")
	}
	observation, ok := s.Observation(call.RequestObservationID)
	if !ok || observation.Run != helper || observation.Kind != "message" || observation.Origin != "invocation_request" ||
		observation.SourceOperationID != call.OperationID || observation.Operation != call.OperationID ||
		observation.Source != weatherSubject || observation.SourceGeneration != 1 || observation.Obligation != helperObligation ||
		!equalPayload(observation.Payload, request) {
		return nil, errors.New("retained Weather helper request differs from admitted template binding")
	}
	return &helper, nil
}

func frozenHelperRequest(c configuration, inputs map[string]schema.Value) (map[string]schema.Value, error) {
	plan, err := c.normalize()
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]schema.Value, 0, maxDelegatedCandidates)
	for _, row := range plan.Candidates {
		if row.Eligible {
			rows = append(rows, candidateRecord(row))
		}
	}
	return map[string]schema.Value{
		proposalID: inputs[proposalID], candidates: schema.List(rows...),
		openmeteo.Units: inputs[openmeteo.Units], "allowed_weather_codes": inputs["allowed_weather_codes"],
	}, nil
}

func verifyDelegatedOperations(
	s *runtime.ExecutionSnapshot,
	helper *runtime.RunID,
	call *runtime.InvocationView,
	maxEffects int64,
) error {
	messages := make(map[string]bool, 4)
	forecasts := make(map[string]runtime.Intent, maxEffects)
	seenCall := false
	for _, op := range s.Operations {
		if op.ID == "" || op.Generation != 1 || op.Authority != nil {
			return fmt.Errorf("unexpected delegated operation %q", op.ID)
		}
		switch op.Kind {
		case "invocation":
			if seenCall || call == nil || op.ID != call.OperationID || op.Run != weatherRun ||
				op.Target != call.TemplateID || op.Obligation != helperObligation ||
				op.Sequence != call.Sequence || !equalPayload(op.Payload, call.Inputs) {
				return fmt.Errorf("unexpected delegated invocation %q", op.ID)
			}
			seenCall = true
		case "adapter":
			if helper == nil || op.Run != *helper || op.Target != capability || op.Obligation != effectName ||
				forecasts[op.ID].ID != "" {
				return fmt.Errorf("unexpected delegated forecast %q", op.ID)
			}
			forecasts[op.ID] = op
		case "message":
			reply := helper != nil && op.Run == *helper && op.Target == weatherSubject && op.Obligation == helperObligation
			if messages[op.Obligation] || (!reply && !validPeerOperation(op)) {
				return fmt.Errorf("unexpected delegated message %q", op.ID)
			}
			messages[op.Obligation] = true
		default:
			return fmt.Errorf("unexpected delegated operation %q", op.ID)
		}
	}
	if seenCall != (call != nil) || int64(len(forecasts)) > maxEffects || len(forecasts) != len(s.Effects) {
		return errors.New("retained delegated operations and effect views differ")
	}
	for _, effect := range s.Effects {
		if effect == nil {
			return errors.New("missing retained delegated effect")
		}
		op, ok := forecasts[effect.OperationID]
		if !ok || effect.Kind != op.Kind || effect.Source != op.Run || effect.Sequence != op.Sequence ||
			effect.Capability != op.Target || effect.Name != op.Obligation {
			return fmt.Errorf("unexpected retained delegated effect %q", effect.OperationID)
		}
		delete(forecasts, effect.OperationID)
	}
	return nil
}
