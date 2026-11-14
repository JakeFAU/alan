// Command weather-agent provisions, runs and inspects a durable Open-Meteo read.
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

const usage = `weather-agent start --db FILE --lat LAT --lon LON --units UNITS --endpoint URL --allowance N
weather-agent {run|resume|status|inspect|replay} --db FILE
weather-agent refresh --db FILE --read N
Add --json for structured output. Inspect and status --json include the retained history.`

var errUsage = errors.New("invalid weather-agent command or arguments (run weather-agent --help)")

func run(ctx context.Context, args []string, out io.Writer) error {
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
	path := flags.String("db", "", "persistent SQLite database")
	jsonOutput := flags.Bool("json", false, "write the structured weather report")
	var c configuration
	var n int
	switch action {
	case "start":
		flags.StringVar(&c.Latitude, "lat", "", "latitude")
		flags.StringVar(&c.Longitude, "lon", "", "longitude")
		flags.StringVar(&c.Units, "units", "", "celsius or fahrenheit")
		flags.StringVar(&c.Endpoint, "endpoint", "", "explicit Open-Meteo forecast endpoint")
		flags.IntVar(&c.Allowance, "allowance", -1, "maximum lifetime physical opportunities (0–8)")
	case "refresh":
		flags.IntVar(&n, "read", 0, "number of the new, separately approved read")
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
	if *path == "" || flags.NArg() != 0 {
		return errUsage
	}
	if action == "start" {
		p, err := c.provision(false)
		if err != nil {
			return err
		}
		s, err := runtime.Create(ctx, *path, p, runtime.DefaultLimits())
		switch {
		case err == nil:
			if err := s.Close(); err != nil {
				return err
			}
		case !errors.Is(err, os.ErrExist):
			return err
		}
	}
	initial, saved, err := inspect(ctx, *path)
	if err != nil {
		return err
	}
	if action == "start" && c != saved {
		return errors.New("existing weather configuration differs; use its original parameters or select a new path")
	}
	var earlier *runtime.ExecutionSnapshot
	var commits *int
	switch action {
	case "replay":
		replay, err := runtime.Replay(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		// Replay verifies its own read-only cut. A live writer can advance
		// between that cut and inspection; never label another answer replayed.
		after, err := runtime.InspectExecution(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if replay.Namespace != initial.Namespace || !reflect.DeepEqual(initial, after) || len(replay.Runs) != len(initial.Runs) {
			return errors.New("weather computation changed during replay; repeat replay")
		}
		for _, replayed := range replay.Runs {
			r, found := initial.Run(replayed.ID)
			if !found || r.Sequence != replayed.Sequence || r.ProgramID != replayed.ProgramID || !reflect.DeepEqual(r.Projection, replayed.Projection) {
				return errors.New("inspected weather answer differs from replay")
			}
		}
		commits = &replay.HistoryCount
	case "run", "resume", "refresh":
		final, before, err := operate(ctx, *path, initial.Namespace, saved, action, n)
		if err != nil {
			return err
		}
		initial, earlier = final, before
	}
	detailed := action == "inspect" || action == "status" && *jsonOutput
	result, err := summarize(initial, earlier, saved, detailed)
	if err != nil {
		return err
	}
	result.ReplayedCommits = commits
	if !*jsonOutput && action != "inspect" {
		return result.writeReport(out, initial, *path, action == "status")
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func inspect(ctx context.Context, path string) (*runtime.ExecutionSnapshot, configuration, error) {
	s, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, configuration{}, err
	}
	c, err := retainedConfiguration(s)
	if err != nil {
		return nil, c, err
	}
	return s, c, verifyWeatherEvidence(s, c)
}

func operate(ctx context.Context, path, namespace string, c configuration, action string, n int) (final, before *runtime.ExecutionSnapshot, err error) {
	limits := runtime.DefaultLimits()
	owner, err := runtime.Open(ctx, path, limits)
	if err != nil {
		return nil, nil, err
	}
	defer func() { err = errors.Join(err, owner.Close()) }()
	before, saved, err := inspect(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	if owner.Namespace() != namespace || before.Namespace != namespace || c != saved {
		return nil, nil, errors.New("weather store changed while acquiring ownership")
	}
	// Only old deployments still rely on the weather owner's ingress checks.
	if len(before.Composition.Continuations) == 0 {
		for read := 2; read <= c.Allowance; read++ {
			if _, err := legacyApproval(before, read); err != nil {
				return nil, nil, err
			}
		}
	}
	if action == "refresh" {
		if err := approve(ctx, owner, before, c.Allowance, n); err != nil {
			return nil, nil, err
		}
	} else {
		adapter, err := openmeteo.New(c.Endpoint)
		if err != nil {
			return nil, nil, err
		}
		_, err = owner.RunProgress(ctx, runtime.RunOptions{
			Adapters: map[string]effects.Adapter{capability: adapter},
			MaxSteps: 512, MaxDispatch: 128, MaxInFlight: 1, ExternalTimeout: 30 * time.Second,
		})
		if err != nil {
			return nil, nil, err
		}
	}
	final, _, err = inspect(ctx, path)
	return final, before, err
}

// Only an explicit invocation of this command supplies a new local approval.
// Its stable target and delivery identity make repeating the command safe.
func approve(ctx context.Context, s *runtime.Store, cut *runtime.ExecutionSnapshot, allowance, n int) error {
	if n < 2 {
		return errors.New("refresh needs the number of a new read, starting at 2")
	}
	if n > allowance {
		return fmt.Errorf("read %d exceeds the original allowance of %d", n, allowance)
	}
	var err error
	if len(cut.Composition.Continuations) == 0 {
		err = admitLegacyApproval(ctx, s, cut, n)
	} else {
		var basis runtime.ContinuationBasis
		var reason string
		basis, reason, err = weatherApprovalBasis(cut, n)
		if err == nil {
			_, err = s.ApproveContinuation(ctx, basis, reason)
		}
	}
	if err != nil {
		predecessor, _ := cut.Run(readID(n - 1))
		action := fmt.Sprintf("approve read %d (%s) after predecessor read %d (%s)", n, readID(n).Subject, n-1, predecessor.ID.Subject)
		if len(predecessor.EffectIDs) == 1 {
			op, _ := cut.Effect(predecessor.EffectIDs[0])
			action += fmt.Sprintf(", original operation %s", op.OperationID)
			if op.AdapterAttempt == nil || !op.AdapterAttempt.Allowed {
				action += " has no paid reservation"
			}
		} else {
			action += ", which has no original operation"
		}
		return fmt.Errorf("%s: %w", action, err)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
