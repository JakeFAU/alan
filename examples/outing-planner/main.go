// Command outing-planner runs a pair of retained programs for a one-hour walk.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

const usage = `outing-planner start --db FILE --config FILE [--weather assess-first|confirm-first|delegated] [--helper-admission 0|1]
outing-planner {run|resume|status|inspect|replay} --db FILE
outing-planner check-replacement --config FILE --fixture FILE [--weather assess-first|confirm-first|delegated] [--helper-admission 0|1] [--output-dir DIR]
Add --json for a structured outing report. Inspect and check-replacement return JSON.`

var errUsage = errors.New("invalid outing-planner command or arguments (run outing-planner --help)")

func runCLI(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	action := args[0]
	if action == "help" || action == "--help" || action == "-h" {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	flags := flag.NewFlagSet(action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var path, configPath, variant, fixturePath, outputDir string
	var jsonOutput bool
	var allowance int64
	if action != "check-replacement" {
		flags.StringVar(&path, "db", "", "persistent SQLite database")
		flags.BoolVar(&jsonOutput, "json", false, "write the structured outing report")
	}
	switch action {
	case "start", "check-replacement":
		flags.StringVar(&configPath, "config", "", "outing configuration")
		flags.StringVar(&variant, "weather", assessFirst, "Weather variant")
		flags.Int64Var(&allowance, "helper-admission", 1, "delegated helper admission allowance")
		if action == "check-replacement" {
			flags.StringVar(&fixturePath, "fixture", "", "versioned native forecast fixture")
			flags.StringVar(&outputDir, "output-dir", "", "retain evidence files in this directory")
		}
	case "run", "resume", "status", "inspect", "replay":
	default:
		return errUsage
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(out)
			flags.PrintDefaults()
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errUsage
	}
	options := defaultDeploymentOptions(variant)
	var optionErr error
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "helper-admission" {
			if variant != delegatedWeather {
				optionErr = errors.New("helper admission requires delegated Weather")
			}
			options.HelperAdmission = allowance
		}
	})
	if optionErr != nil {
		return optionErr
	}
	if action == "start" || action == "check-replacement" {
		if err := options.validate(); err != nil {
			return err
		}
	}
	if action == "check-replacement" {
		if configPath == "" || fixturePath == "" {
			return errUsage
		}
		return checkReplacement(ctx, replacementOptions{
			ConfigPath: configPath, FixturePath: fixturePath, Variant: variant, OutputDir: outputDir,
			HelperAdmission: options.HelperAdmission,
		}, out)
	}
	if path == "" || action == "start" && configPath == "" {
		return errUsage
	}
	var cut *runtime.ExecutionSnapshot
	var c configuration
	var err error
	if action == "start" {
		cut, c, err = startPlannerDeployment(ctx, path, configPath, options, time.Now())
	} else {
		var info retainedDeploymentInfo
		cut, info, err = inspectPlannerDeployment(ctx, path)
		c, options = info.Configuration, info.Options
	}
	if err != nil {
		return err
	}
	var progress *runtime.RunProgress
	var replay *runtime.ReplayReport
	switch action {
	case "run", "resume":
		var advanced runtime.RunProgress
		cut, advanced, err = operateBoundedDeployment(ctx, path, cut.Namespace, c, options, standardBounds)
		progress = &advanced
	case "replay":
		replay, err = replayPlanner(ctx, path, cut)
	}
	if err != nil {
		return err
	}
	report, err := buildReport(cut, c, options.Weather)
	if err != nil {
		return err
	}
	report.Progress, report.Replay = progress, replay
	if jsonOutput || action == "inspect" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	return report.writeText(out)
}

func startPlanner(
	ctx context.Context,
	path, configPath, variant string,
	now time.Time,
) (*runtime.ExecutionSnapshot, configuration, error) {
	return startPlannerDeployment(ctx, path, configPath, defaultDeploymentOptions(variant), now)
}

func startPlannerDeployment(
	ctx context.Context,
	path, configPath string,
	options deploymentOptions,
	now time.Time,
) (*runtime.ExecutionSnapshot, configuration, error) {
	if err := options.validate(); err != nil {
		return nil, configuration{}, err
	}
	raw, err := readBoundedFile(configPath, maxConfiguration)
	if err != nil {
		return nil, configuration{}, err
	}
	cut, retained, err := inspectPlannerDeployment(ctx, path)
	if errors.Is(err, os.ErrNotExist) {
		c, configErr := configurationAt(raw, now.UTC().Format(time.RFC3339Nano))
		if configErr != nil {
			return nil, c, configErr
		}
		if err := createPlannerDeployment(ctx, path, c, options); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, c, err
		}
		cut, retained, err = inspectPlannerDeployment(ctx, path)
	}
	saved := retained.Configuration

	if err != nil {
		return nil, saved, err
	}
	c, err := configurationAt(raw, saved.ReferenceTime)
	if err != nil {
		return nil, saved, err
	}
	if !reflect.DeepEqual(c, saved) || options != retained.Options {
		if options.Weather == delegatedWeather || retained.Options.Weather == delegatedWeather {
			return nil, saved, errors.New("existing outing deployment differs; use its original configuration and deployment options, resume, or use a fresh Store")
		}
		return nil, saved, errors.New("existing outing deployment differs; use its original configuration and Weather variant or a fresh Store")
	}
	return cut, saved, nil
}

func replayPlanner(ctx context.Context, path string, cut *runtime.ExecutionSnapshot) (*runtime.ReplayReport, error) {
	replayed, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	after, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if replayed.Namespace != cut.Namespace || !reflect.DeepEqual(cut, after) || len(replayed.Runs) != len(cut.Runs) {
		return nil, errors.New("outing computation changed during replay; repeat replay")
	}
	for _, state := range replayed.Runs {
		r, ok := cut.Run(state.ID)
		if !ok || !reflect.DeepEqual(r.RunState, state) {
			return nil, errors.New("inspected outing state differs from replay")
		}
	}
	return &replayed, nil
}

func createPlanner(ctx context.Context, path string, c configuration, variant string) error {
	return createPlannerDeployment(ctx, path, c, defaultDeploymentOptions(variant))
}

func createPlannerDeployment(ctx context.Context, path string, c configuration, options deploymentOptions) error {
	p, err := c.provisionDeployment(options)
	if err != nil {
		return err
	}
	return createProvisionedPlanner(ctx, path, p)
}

func createProvisionedPlanner(ctx context.Context, path string, p runtime.Provision) error {
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	return s.Close()
}

func inspectPlanner(ctx context.Context, path string) (*runtime.ExecutionSnapshot, configuration, string, error) {
	s, info, err := inspectPlannerDeployment(ctx, path)
	return s, info.Configuration, info.Options.Weather, err
}

func inspectPlannerDeployment(ctx context.Context, path string) (*runtime.ExecutionSnapshot, retainedDeploymentInfo, error) {
	s, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, retainedDeploymentInfo{}, err
	}
	info, err := retainedDeployment(s)
	if err == nil {
		err = verifyPlannerDeploymentEvidence(s, info)
	}
	return s, info, err
}

type plannerBounds struct {
	MaxSteps    int           `json:"max_steps"`
	MaxDispatch int           `json:"max_dispatch"`
	MaxInFlight int           `json:"max_in_flight"`
	Timeout     time.Duration `json:"external_timeout_ns"`
}

var standardBounds = plannerBounds{MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1, Timeout: 30 * time.Second}

func operatePlanner(
	ctx context.Context,
	path, namespace string,
	c configuration,
	variant string,
) (*runtime.ExecutionSnapshot, runtime.RunProgress, error) {
	return operateBoundedPlanner(ctx, path, namespace, c, variant, standardBounds)
}

func operateBoundedPlanner(
	ctx context.Context,
	path, namespace string,
	c configuration,
	variant string,
	bounds plannerBounds,
) (*runtime.ExecutionSnapshot, runtime.RunProgress, error) {
	return operateBoundedDeployment(ctx, path, namespace, c, defaultDeploymentOptions(variant), bounds)
}

func operateBoundedDeployment(
	ctx context.Context,
	path, namespace string,
	c configuration,
	options deploymentOptions,
	bounds plannerBounds,
) (final *runtime.ExecutionSnapshot, progress runtime.RunProgress, err error) {
	owner, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, progress, err
	}
	defer func() { err = errors.Join(err, owner.Close()) }()
	before, saved, err := inspectPlannerDeployment(ctx, path)
	if err != nil {
		return nil, progress, err
	}
	if owner.Namespace() != namespace || before.Namespace != namespace || !reflect.DeepEqual(c, saved.Configuration) || options != saved.Options {
		return nil, progress, errors.New("outing store changed while acquiring ownership")
	}
	adapter, err := openmeteo.NewHourly(saved.Configuration.Endpoint)
	if err != nil {
		return nil, progress, err
	}
	progress, runErr := owner.RunProgress(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{capability: adapter},
		MaxSteps: bounds.MaxSteps, MaxDispatch: bounds.MaxDispatch,
		MaxInFlight: bounds.MaxInFlight, ExternalTimeout: bounds.Timeout,
	})
	if runErr != nil && !errors.Is(runErr, runtime.ErrLimit) {
		return nil, progress, runErr
	}
	final, finalDeployment, err := inspectPlannerDeployment(ctx, path)
	if err != nil {
		return nil, progress, errors.Join(runErr, err)
	}
	if final.Namespace != namespace || !reflect.DeepEqual(c, finalDeployment.Configuration) || finalDeployment.Options != options {
		return nil, progress, errors.Join(runErr, errors.New("outing store changed during execution"))
	}
	return final, progress, runErr
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runCLI(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
