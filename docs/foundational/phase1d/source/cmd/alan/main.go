package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/JakeFAU/alan/internal/executor"
)

type usageError struct{ message string }

func (e usageError) Error() string { return e.message }
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := runCLI(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		var usage usageError
		if errors.As(err, &usage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
func runCLI(ctx context.Context, args []string, out io.Writer) (err error) {
	if len(args) == 0 {
		return usageError{"usage: alan <init|run|input|inspect|replay> --db path"}
	}
	command := args[0]
	switch command {
	case "init", "run", "input", "inspect", "replay":
	default:
		return usageError{"unknown command: " + command}
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	db := flags.String("db", "", "SQLite store path")
	var provision, request, demo, successorInput, endpoint, profile *string
	var retryBudget *int64
	if command == "init" {
		provision = flags.String("provision", "", "JSON provision file")
		demo = flags.String("demo", "", "ordinary demo fixture: nested or external")
		endpoint = flags.String("endpoint", "", "external fixture literal loopback HTTP URL")
		profile = flags.String("profile", "idempotent", "external fixture: idempotent or non-idempotent")
		retryBudget = flags.Int64("retry-budget", 1, "ordinary nested fixture retry budget")
		successorInput = flags.String("successor-input", "success", "explicit nested successor input")
	}
	if command == "input" {
		request = flags.String("request", "", "JSON ordinary input file")
	}
	options := executor.RunOptions{MaxSteps: 10000, MaxDispatch: 10000}
	if command == "run" {
		flags.IntVar(&options.MaxSteps, "max-steps", options.MaxSteps, "maximum execution commits")
		flags.IntVar(&options.MaxDispatch, "max-dispatch", options.MaxDispatch, "maximum supported deliveries")
	}
	limits := executor.DefaultLimits()
	flags.IntVar(&limits.MaxProgramBytes, "max-program-bytes", limits.MaxProgramBytes, "program byte bound")
	flags.IntVar(&limits.MaxDataBytes, "max-data-bytes", limits.MaxDataBytes, "projection byte bound")
	flags.IntVar(&limits.MaxObservationBytes, "max-observation-bytes", limits.MaxObservationBytes, "admission/observation byte bound")
	flags.IntVar(&limits.MaxOutputBytes, "max-output-bytes", limits.MaxOutputBytes, "generated intent-array byte bound")
	if parseErr := flags.Parse(args[1:]); parseErr != nil {
		return usageError{parseErr.Error()}
	}
	if flags.NArg() != 0 || *db == "" {
		return usageError{"--db is required; positional arguments are not accepted"}
	}
	if limits.MaxProgramBytes <= 0 || limits.MaxDataBytes <= 0 || limits.MaxObservationBytes <= 0 || limits.MaxOutputBytes <= 0 {
		return usageError{"byte limits must be positive"}
	}
	encode := func(value any) error {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	switch command {
	case "inspect":
		snapshot, e := executor.Inspect(ctx, *db, limits)
		if e != nil {
			return e
		}
		return encode(snapshot)
	case "replay":
		report, e := executor.Replay(ctx, *db, limits)
		if e != nil {
			return e
		}
		return encode(report)
	case "init":
		if (*provision == "") == (*demo == "") {
			return usageError{"choose exactly one of --provision or --demo nested|external"}
		}
		var seed executor.Provision
		if *demo != "" {
			switch *demo {
			case "nested":
				seed, err = executor.NestedProvision(executor.NestedConfig{RetryBudget: *retryBudget, SuccessorInput: *successorInput})
			case "external":
				seed, err = executor.ExternalProvision(*endpoint, *profile)
			default:
				return usageError{"unknown demo"}
			}
			if err != nil {
				return err
			}
		} else if err = readJSON(*provision, &seed); err != nil {
			return err
		}
		store, e := executor.Create(ctx, *db, seed, limits)
		if e != nil {
			return e
		}
		if err = store.Close(); err != nil {
			return err
		}
		return encode(map[string]string{"status": "initialized", "store": *db})
	case "input":
		if *request == "" {
			return usageError{"--request is required"}
		}
		// Parse before Open so malformed/forged input cannot even advance an epoch.
		var input executor.Input
		if err = readJSON(*request, &input); err != nil {
			return err
		}
		store, e := executor.Open(ctx, *db, limits)
		if e != nil {
			return e
		}
		defer func() { err = errors.Join(err, store.Close()) }()
		result, e := store.AdmitInput(ctx, input)
		if e != nil {
			return e
		}
		return encode(result)
	case "run":
		ctx, err = boundary.FromEnvironment(ctx, *db)
		if err != nil {
			return err
		}
		if options.MaxSteps <= 0 || options.MaxDispatch <= 0 {
			return usageError{"runner budgets must be positive"}
		}
		store, e := executor.Open(ctx, *db, limits)
		if e != nil {
			return e
		}
		defer func() { err = errors.Join(err, store.Close()) }()
		report, e := store.Run(ctx, options)
		if writeErr := encode(report); writeErr != nil {
			return errors.Join(e, writeErr)
		}
		return e
	}
	return usageError{"unsupported command"}
}
func readJSON(path string, target any) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("%s must contain one JSON value", path)
	}
	return nil
}
