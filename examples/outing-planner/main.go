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

const usage = `outing-planner start --db FILE --config FILE [--weather assess-first|confirm-first]
outing-planner {run|resume|status} --db FILE
Commands return the validated execution snapshot as JSON.`

var errUsage = errors.New("invalid outing-planner command or arguments (run outing-planner --help)")

func runCLI(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	action := args[0]
	flags := flag.NewFlagSet(action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "persistent SQLite database")
	flags.Bool("json", true, "write execution snapshot JSON")
	var configPath, variant string
	switch action {
	case "help", "--help", "-h":
		_, err := fmt.Fprintln(out, usage)
		return err
	case "start":
		flags.StringVar(&configPath, "config", "", "outing configuration")
		flags.StringVar(&variant, "weather", assessFirst, "Weather variant")
	case "run", "resume", "status":
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
	if *path == "" || flags.NArg() != 0 {
		return errUsage
	}
	if action == "start" {
		if configPath == "" {
			return errUsage
		}
		c, err := readConfiguration(configPath, time.Now())
		if err != nil {
			return err
		}
		if err := createPlanner(ctx, *path, c, variant); err != nil {
			return err
		}
	}
	cut, c, variant, err := inspectPlanner(ctx, *path)
	if err != nil {
		return err
	}
	if action == "run" || action == "resume" {
		cut, _, err = operatePlanner(ctx, *path, cut.Namespace, c, variant)
		if err != nil {
			return err
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(cut)
}

func createPlanner(ctx context.Context, path string, c configuration, variant string) error {
	p, err := c.provision(variant)
	if err != nil {
		return err
	}
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	return s.Close()
}

func inspectPlanner(ctx context.Context, path string) (*runtime.ExecutionSnapshot, configuration, string, error) {
	s, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, configuration{}, "", err
	}
	c, variant, err := retainedConfiguration(s)
	if err == nil {
		err = verifyPlannerEvidence(s, c, variant)
	}
	return s, c, variant, err
}

func operatePlanner(
	ctx context.Context,
	path, namespace string,
	c configuration,
	variant string,
) (final *runtime.ExecutionSnapshot, progress runtime.RunProgress, err error) {
	owner, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, progress, err
	}
	defer func() { err = errors.Join(err, owner.Close()) }()
	before, saved, savedVariant, err := inspectPlanner(ctx, path)
	if err != nil {
		return nil, progress, err
	}
	if owner.Namespace() != namespace || before.Namespace != namespace || !reflect.DeepEqual(c, saved) || variant != savedVariant {
		return nil, progress, errors.New("outing store changed while acquiring ownership")
	}
	adapter, err := openmeteo.NewHourly(saved.Endpoint)
	if err != nil {
		return nil, progress, err
	}
	progress, err = owner.RunProgress(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{capability: adapter},
		MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1, ExternalTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, progress, err
	}
	final, _, _, err = inspectPlanner(ctx, path)
	if err == nil && final.Namespace != namespace {
		return nil, progress, errors.New("outing store changed during execution")
	}
	return final, progress, err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runCLI(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
