package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"github.com/JakeFAU/alan/runtime"
)

const (
	circularDependency = "circular_dependency"
	boundExhausted     = "bound_exhausted"
	cycleMessage       = "The planner needs a forecast; the forecast service requires the planner’s final choice."
)

type replacementOptions struct {
	ConfigPath, FixturePath, Variant, OutputDir string
	HelperAdmission                             int64
}

type replacementReport struct {
	Outcome          string               `json:"outcome"`
	Explanation      string               `json:"explanation"`
	Scenario         string               `json:"scenario"`
	ReferenceTime    string               `json:"reference_time"`
	SourceConfigSHA  string               `json:"source_config_sha256"`
	DerivedConfigSHA string               `json:"derived_config_sha256"`
	FixtureSHA       string               `json:"fixture_sha256"`
	Bounds           plannerBounds        `json:"bounds"`
	Baseline         replacementRun       `json:"baseline"`
	Candidate        replacementRun       `json:"candidate"`
	Delegated        *delegatedComparison `json:"delegated_comparison,omitempty"`
}

type replacementRun struct {
	Store           string           `json:"store,omitempty"`
	Outcome         string           `json:"outcome"`
	Report          plannerReport    `json:"report"`
	FixtureRequests int              `json:"fixture_requests"`
	Cycle           *cycleWitness    `json:"cycle,omitempty"`
	Reopened        *reopenedWitness `json:"reopened,omitempty"`
	BoundError      string           `json:"bound_error,omitempty"`
	roots           []runtime.RootCompletion
}

type cycleWitness struct {
	Explanation string               `json:"explanation"`
	FirstAction string               `json:"first_missing_action"`
	Proposal    runtime.ReductionRef `json:"weather_proposal_selection"`
	Waits       []cycleWait          `json:"waits"`
}

type cycleWait struct {
	Run       runtime.RunID         `json:"run"`
	ProgramID string                `json:"program_id"`
	PC        int                   `json:"pc"`
	Wait      runtime.ReceiveWait   `json:"obligation"`
	History   runtime.ExecutionStep `json:"history"`
}

type reopenedWitness struct {
	Outcome  string              `json:"outcome"`
	Progress runtime.RunProgress `json:"progress"`
	Runs     []runtime.RunState  `json:"runs"`
	Cycle    *cycleWitness       `json:"cycle,omitempty"`
}

func checkReplacement(ctx context.Context, opts replacementOptions, out io.Writer) error {
	options := deploymentOptions{Weather: opts.Variant, HelperAdmission: opts.HelperAdmission}
	if err := options.validate(); err != nil {
		return err
	}
	fixture, fixtureRaw, err := readFixture(opts.FixturePath)
	if err != nil {
		return err
	}
	sourceRaw, err := readBoundedFile(opts.ConfigPath, maxConfiguration)
	if err != nil {
		return err
	}
	c, err := configurationAt(sourceRaw, fixture.ReferenceText)
	if err != nil {
		return err
	}
	reference, err := parseExplicitTime(c.ReferenceTime)
	if err != nil || !reference.Equal(fixture.ReferenceTime) {
		return errors.New("configuration reference_time differs from the fixture reference_time")
	}
	plan, err := c.normalize()
	if err != nil {
		return err
	}
	if options.Weather == delegatedWeather && countEligible(plan) > maxDelegatedCandidates {
		return errors.New("delegated Weather supports at most two eligible candidates")
	}
	if err := fixture.requireCandidates(plan); err != nil {
		return err
	}

	dir, err := replacementDirectory(opts.OutputDir, options.Weather == delegatedWeather)
	if err != nil {
		return err
	}
	if opts.OutputDir == "" {
		defer os.RemoveAll(dir)
	}
	handler := newFixtureHandler(fixture, c.Units)
	listener, err := listenForFixture()
	if err != nil {
		return fmt.Errorf("listen for loopback fixture: %w", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	defer func() {
		_ = server.Close()
		<-done
	}()

	c.Endpoint = "http://" + listener.Addr().String() + fixturePath
	derivedRaw, err := c.serialized()
	if err != nil {
		return err
	}
	baselineOptions := defaultDeploymentOptions(assessFirst)
	baselineProvision, err := c.provisionDeployment(baselineOptions)
	if err != nil {
		return err
	}
	candidateProvision, err := c.provisionDeployment(options)
	if err != nil {
		return err
	}
	if options.Weather == delegatedWeather {
		manifest := freezeDeploymentManifest(derivedRaw, plan, baselineProvision, candidateProvision, options)
		if err := writeDeploymentManifest(dir, manifest); err != nil {
			return err
		}
	}
	for _, input := range []struct {
		name string
		raw  []byte
	}{
		{"source-configuration.json", sourceRaw},
		{"derived-configuration.json", derivedRaw},
		{"fixture.json", fixtureRaw},
	} {
		if err := writeFresh(filepath.Join(dir, input.name), input.raw); err != nil {
			return err
		}
	}

	r := replacementReport{
		Scenario:      "Single-Store, two-program outing using the same frozen inputs and loopback forecasts in both deployments.",
		ReferenceTime: c.ReferenceTime, SourceConfigSHA: digestText(string(sourceRaw)),
		DerivedConfigSHA: digestText(string(derivedRaw)), FixtureSHA: digestText(string(fixtureRaw)),
		Bounds: standardBounds,
	}
	if options.Weather == delegatedWeather {
		r.Scenario = "Fresh direct and delegated Stores using the same frozen configuration and loopback forecasts."
	}
	r.Baseline, err = runProvisionedReplacement(ctx, filepath.Join(dir, "baseline.db"), c, baselineOptions, baselineProvision, r.Bounds)
	baselineRequests, fixtureErr := handler.results()
	if err := errors.Join(err, fixtureErr); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	r.Baseline.FixtureRequests = baselineRequests
	r.Candidate, err = runProvisionedReplacement(ctx, filepath.Join(dir, "candidate.db"), c, options, candidateProvision, r.Bounds)
	total, fixtureErr := handler.results()
	if err := errors.Join(err, fixtureErr); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	r.Candidate.FixtureRequests = total - r.Baseline.FixtureRequests
	if opts.OutputDir == "" {
		r.Baseline.Store, r.Candidate.Store = "", ""
	}
	r.Outcome, r.Explanation = r.Candidate.Outcome, r.Candidate.Report.Explanation
	if options.Weather == delegatedWeather {
		r.Delegated, err = compareDelegatedReplacement(r.Baseline, r.Candidate, plan)
		if err != nil {
			return err
		}
		if r.Delegated.PositiveReplacement {
			r.Explanation = "Both deployments completed this fixture with the same ranked answer. " + r.Explanation
		}
		if r.Candidate.BoundError != "" {
			r.Explanation = r.Candidate.BoundError
		}
	} else {
		if err := checkReplacementPair(r.Baseline, r.Candidate); err != nil {
			return err
		}
		switch {
		case !completedPlan(r.Baseline.Outcome):
			r.Outcome = r.Baseline.Outcome
			r.Explanation = "Baseline did not complete the question. " + r.Baseline.Report.Explanation
		case r.Candidate.Cycle != nil:
			r.Explanation = r.Candidate.Cycle.Explanation
		case completedPlan(r.Candidate.Outcome):
			r.Explanation = "Both deployments completed this fixture case. " + r.Explanation
		}
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := writeFresh(filepath.Join(dir, "replacement.json"), raw); err != nil {
		return err
	}
	_, err = out.Write(raw)
	return err
}

func completedPlan(outcome string) bool { return outcome == recommended || outcome == noMatch }

func replacementDirectory(path string, manifest bool) (string, error) {
	if path == "" {
		return os.MkdirTemp("", "outing-replacement-")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0700); err != nil {
		return "", err
	}
	files := []string{
		"baseline.db", "candidate.db", "baseline.db.lock", "candidate.db.lock",
		"source-configuration.json", "derived-configuration.json", "fixture.json", "replacement.json",
	}
	if manifest {
		files = append(files, "deployment-manifest.json")
	}
	for _, file := range files {
		if _, err := os.Lstat(filepath.Join(abs, file)); !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("replacement evidence file %s already exists in %s", file, abs)
		}
	}
	return abs, nil
}

func writeFresh(path string, raw []byte) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	_, err = f.Write(raw)
	return err
}

func checkReplacementPair(baseline, candidate replacementRun) error {
	b, c := baseline.Report, candidate.Report
	if b.Namespace == c.Namespace || !reflect.DeepEqual(b.Configuration, c.Configuration) ||
		len(b.Runs) != len(c.Runs) || b.Proposal != nil && c.Proposal != nil && !equalPayload(b.Proposal.Payload, c.Proposal.Payload) {
		return errors.New("replacement did not preserve the original planning question in fresh Stores")
	}
	for i, run := range b.Runs {
		replacement := c.Runs[i]
		if run.ID != replacement.ID || !reflect.DeepEqual(run.Initial, replacement.Initial) ||
			(run.ProgramID == replacement.ProgramID) != (run.ID == outingRun || c.Variant == assessFirst) {
			return fmt.Errorf("unexpected replacement artifact or inputs for %s", run.ID.Subject)
		}
	}
	return nil
}

func runReplacementScenario(
	ctx context.Context,
	path string,
	c configuration,
	variant string,
	bounds plannerBounds,
) (replacementRun, error) {
	options := defaultDeploymentOptions(variant)
	p, err := c.provisionDeployment(options)
	if err != nil {
		return replacementRun{}, err
	}
	return runProvisionedReplacement(ctx, path, c, options, p, bounds)
}

func runProvisionedReplacement(
	ctx context.Context,
	path string,
	c configuration,
	options deploymentOptions,
	p runtime.Provision,
	bounds plannerBounds,
) (replacementRun, error) {
	if err := createProvisionedPlanner(ctx, path, p); err != nil {
		return replacementRun{}, err
	}
	first, _, err := inspectPlannerDeployment(ctx, path)
	if err != nil {
		return replacementRun{}, err
	}
	final, progress, runErr := operateBoundedDeployment(ctx, path, first.Namespace, c, options, bounds)
	exhausted := final != nil && hitRunnerBound(bounds, progress, runErr)
	if runErr != nil && !exhausted {
		return replacementRun{}, runErr
	}
	report, err := buildReport(final, c, options.Weather)
	if err != nil {
		return replacementRun{}, err
	}
	report.Progress = &progress
	r := replacementRun{Store: path, Outcome: report.Outcome, Report: report, roots: final.Roots}
	if exhausted {
		r.Outcome = boundExhausted
		if options.Weather == delegatedWeather {
			r.BoundError = runErr.Error()
		} else {
			r.Report.Explanation = runErr.Error()
		}
		return r, nil
	}
	r.Cycle = cycleAt(final, progress)
	if r.Cycle != nil {
		r.Outcome = circularDependency
	}
	reopened, next, err := operateBoundedDeployment(ctx, path, final.Namespace, c, options, bounds)
	if err != nil {
		return replacementRun{}, err
	}
	evidence, err := buildReport(reopened, c, options.Weather)
	if err != nil {
		return replacementRun{}, err
	}
	proof := &reopenedWitness{
		Outcome: evidence.Outcome, Progress: next, Cycle: cycleAt(reopened, next),
		Runs: make([]runtime.RunState, 0, len(reopened.Runs)),
	}
	if proof.Cycle != nil {
		proof.Outcome = circularDependency
	}
	if err := validateReplacementReopen(r, evidence, proof); err != nil {
		return replacementRun{}, err
	}
	for _, run := range reopened.Runs {
		proof.Runs = append(proof.Runs, run.RunState)
	}
	r.Reopened = proof
	return r, nil
}

func hitRunnerBound(bounds plannerBounds, progress runtime.RunProgress, err error) bool {
	// Only the runner's own two explicit bound errors belong to this outcome.
	// An inspection, adapter, or cleanup error must still fail the command.
	if err == nil || !errors.Is(err, runtime.ErrLimit) {
		return false
	}
	switch err.Error() {
	case fmt.Sprintf("execution budget exhausted: %s", runtime.ErrLimit):
		return progress.Steps == bounds.MaxSteps
	case fmt.Sprintf("dispatch budget exhausted: %s", runtime.ErrLimit):
		return progress.Dispatches == bounds.MaxDispatch
	default:
		return false
	}
}

func validateReplacementReopen(before replacementRun, after plannerReport, proof *reopenedWitness) error {
	prior := before.Report
	if len(prior.Runs) != len(after.Runs) || !reflect.DeepEqual(prior.Delegated, after.Delegated) || !proof.Progress.Quiescent || proof.Progress.Steps != 0 || proof.Progress.Dispatches != 0 ||
		proof.Outcome != before.Outcome || !reflect.DeepEqual(before.Cycle, proof.Cycle) ||
		!reflect.DeepEqual(prior.History, after.History) || !reflect.DeepEqual(prior.Effects, after.Effects) ||
		!reflect.DeepEqual(prior.Observations, after.Observations) || !reflect.DeepEqual(prior.Admissions, after.Admissions) ||
		!reflect.DeepEqual(prior.Pending, after.Pending) || !reflect.DeepEqual(prior.Operations, after.Operations) {
		return errors.New("fresh owner did not preserve the settled or blocked replacement cut")
	}
	for i, reopened := range after.Runs {
		want := *prior.Runs[i]
		if reopened.Projection.Status == "open" {
			want.Epoch++
		}
		if !reflect.DeepEqual(&want, reopened) {
			return fmt.Errorf("fresh owner changed logical state of %s", reopened.ID.Subject)
		}
	}
	return nil
}

// cycleAt establishes only the closed cut of the recognized replacement pair.
// The reviewed artifact definitions send each missing message downstream of the
// peer receive. History ties each claimed receive to its actual committed cut.
func cycleAt(s *runtime.ExecutionSnapshot, progress runtime.RunProgress) *cycleWitness {
	if !progress.Quiescent || len(s.Effects) != 0 || len(s.Operations) != 1 || len(s.Observations) != 1 {
		return nil
	}
	_, variant, err := retainedConfiguration(s)
	if err != nil || variant != confirmFirst {
		return nil
	}
	proposal := &s.Operations[0]
	if proposal.Kind != "message" || proposal.Obligation != proposalPort || !validPeerOperation(*proposal) {
		return nil
	}
	proposalSelection := selectedMessage(s, proposal, weatherRun)
	if proposalSelection == nil {
		return nil
	}
	for _, pending := range s.Pending {
		switch pending.Kind {
		case "waiting_message", "root_member", "root_awaiting_publication":
		default:
			return nil
		}
	}

	waits := make([]cycleWait, 0, 2)
	for _, required := range []struct {
		run, peer  runtime.RunID
		obligation string
	}{
		{outingRun, weatherRun, assessmentPort},
		{weatherRun, outingRun, decisionPort},
	} {
		run, ok := s.Run(required.run)
		if !ok || run.Subject.Retired || run.Projection.Status != "open" || run.Instruction != "receive" || run.Wait == nil {
			return nil
		}
		wait := run.Wait
		if !wait.Resolved || len(wait.MatchingObservationIDs) != 0 || wait.Kind != "message" ||
			wait.Provenance != "ordinary-message" || wait.Source != required.peer.Subject ||
			wait.SourceGeneration != required.peer.Generation || wait.Obligation != required.obligation || wait.OperationID != "" {
			return nil
		}
		ref := runtime.ReductionRef{Run: required.run, Sequence: run.Sequence}
		step, ok := s.Reduction(ref)
		if !ok || step.AfterPC != run.Projection.PC {
			return nil
		}
		waits = append(waits, cycleWait{
			Run: run.ID, ProgramID: run.ProgramID, PC: run.Projection.PC, Wait: *wait, History: *step,
		})
	}
	return &cycleWitness{
		Explanation: cycleMessage,
		FirstAction: "Outing can send its decision only after receiving Weather's assessment. " +
			"Confirmation-first Weather can send its assessment only after receiving Outing's decision. " +
			"Either first missing message would require the other to have been received already.",
		Proposal: *proposalSelection,
		Waits:    waits,
	}
}

type replacementCost struct {
	Invocations int64 `json:"invocations"`
	CreatedRuns int64 `json:"created_runs"`
	Effects     int64 `json:"effects"`
}

type forecastOperationPair struct {
	CandidateID string `json:"candidate_id"`
	Baseline    string `json:"baseline_operation_id,omitempty"`
	Delegated   string `json:"candidate_operation_id,omitempty"`
}

type delegatedComparison struct {
	PositiveReplacement bool                    `json:"positive_replacement"`
	BaselineCost        replacementCost         `json:"baseline_cost"`
	CandidateCost       replacementCost         `json:"candidate_cost"`
	ForecastOperations  []forecastOperationPair `json:"forecast_operations"`
}

func countEligible(plan normalizedPlan) int {
	n := 0
	for _, row := range plan.Candidates {
		if row.Eligible {
			n++
		}
	}
	return n
}

func compareDelegatedReplacement(baseline, delegated replacementRun, plan normalizedPlan) (*delegatedComparison, error) {
	b, d := baseline.Report, delegated.Report
	if d.Delegated == nil || b.Variant != assessFirst || b.Namespace == d.Namespace ||
		!reflect.DeepEqual(b.Configuration, d.Configuration) || !slices.Equal(b.Candidates, d.Candidates) ||
		(b.Proposal != nil && d.Proposal != nil && !equalPayload(b.Proposal.Payload, d.Proposal.Payload)) {
		return nil, errors.New("delegated replacement changed the frozen planning question")
	}
	for _, id := range []runtime.RunID{outingRun, weatherRun} {
		bi, di := findReportRun(b, id), findReportRun(d, id)
		if bi == nil || di == nil || !reflect.DeepEqual(bi.Initial, di.Initial) ||
			(bi.ProgramID == di.ProgramID) != (id == outingRun) {
			return nil, fmt.Errorf("unexpected delegated replacement artifact or inputs for %s", id.Subject)
		}
	}
	comparison := &delegatedComparison{
		ForecastOperations: make([]forecastOperationPair, 0, countEligible(plan)),
		BaselineCost:       totalCost(baseline.roots),
		CandidateCost:      totalCost(delegated.roots),
	}
	directEffects, helperEffects := orderedForecasts(b.Effects), orderedForecasts(d.Effects)
	for _, row := range plan.Candidates {
		if !row.Eligible {
			continue
		}
		i := len(comparison.ForecastOperations)
		pair := forecastOperationPair{CandidateID: row.ID}
		if i < len(directEffects) {
			pair.Baseline = directEffects[i].OperationID
		}
		if i < len(helperEffects) {
			pair.Delegated = helperEffects[i].OperationID
		}
		if pair.Baseline != "" && pair.Delegated != "" &&
			(pair.Baseline == pair.Delegated || !equalFixtureEvidence(directEffects[i], helperEffects[i])) {
			return nil, fmt.Errorf("delegated replacement changed fixture evidence for candidate %s", row.ID)
		}
		comparison.ForecastOperations = append(comparison.ForecastOperations, pair)
	}

	if !completedPlan(baseline.Outcome) || !completedPlan(delegated.Outcome) ||
		!rootsPublished(baseline.roots) || !rootsPublished(delegated.roots) ||
		len(b.Pending) != 0 || len(d.Pending) != 0 {
		return comparison, nil
	}
	n := int64(len(comparison.ForecastOperations))
	if comparison.BaselineCost != (replacementCost{Effects: n}) ||
		comparison.CandidateCost != (replacementCost{Invocations: 1, CreatedRuns: 1, Effects: n}) ||
		baseline.FixtureRequests != int(n) || delegated.FixtureRequests != int(n) {
		return nil, errors.New("completed replacement has different original work or fixture contact costs")
	}
	if b.Decision == nil || d.Decision == nil || b.Decision.Payload[outcome].Text != d.Decision.Payload[outcome].Text ||
		b.Outcome != d.Outcome ||
		(b.Recommendation == nil) != (d.Recommendation == nil) ||
		(b.Recommendation != nil && b.Recommendation.CandidateID != d.Recommendation.CandidateID) {
		return nil, errors.New("delegated replacement changed the ranked answer")
	}
	comparison.PositiveReplacement = true
	return comparison, nil
}

func findReportRun(report plannerReport, id runtime.RunID) *runtime.ExecutionRun {
	i := slices.IndexFunc(report.Runs, func(r *runtime.ExecutionRun) bool { return r.ID == id })
	if i < 0 {
		return nil
	}
	return report.Runs[i]
}

func totalCost(roots []runtime.RootCompletion) replacementCost {
	var cost replacementCost
	for _, root := range roots {
		cost.Invocations += root.Charges.Invocations
		cost.CreatedRuns += root.Charges.CreatedRuns
		cost.Effects += root.Charges.Effects
	}
	return cost
}

func orderedForecasts(effects []*runtime.ExecutionEffect) []*runtime.ExecutionEffect {
	out := slices.Clone(effects)
	slices.SortFunc(out, func(a, b *runtime.ExecutionEffect) int {
		switch {
		case a.Sequence < b.Sequence:
			return -1
		case a.Sequence > b.Sequence:
			return 1
		default:
			return 0
		}
	})
	return out
}

func equalFixtureEvidence(a, b *runtime.ExecutionEffect) bool {
	if a.AdapterAttempt != nil && b.AdapterAttempt != nil &&
		(a.AdapterAttempt.Allowed != b.AdapterAttempt.Allowed ||
			!equalPayload(a.AdapterAttempt.Prepared.Request.Payload, b.AdapterAttempt.Prepared.Request.Payload)) {
		return false
	}
	if a.AdapterResult == nil || b.AdapterResult == nil {
		return true
	}
	left, right := a.AdapterResult.Candidate, b.AdapterResult.Candidate
	return left.Disposition == right.Disposition && left.HTTPStatus == right.HTTPStatus &&
		left.Response == right.Response && equalPayload(left.Payload, right.Payload)
}
