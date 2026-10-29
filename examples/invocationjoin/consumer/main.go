// Command consumer demonstrates Phase 1L using public imports in a separate module.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/invocationjoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

func execute(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected init, artifacts, run, recover, inspect or replay")
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "retained database")
	candidate := flags.String("candidate", "a", "a, b or named probe")
	sMode := flags.String("s-mode", "answer", "answer, fail, absent or pending S")
	id := flags.Int64("id", 1, "application id")
	value := flags.Int64("value", 0, "Boolean integer")
	mode := flags.String("mode", "answer", "request mode")
	i := flags.Int64("i", 3, "invocation ceiling")
	r := flags.Int64("r", 3, "created-run ceiling")
	depth := flags.Int64("depth", 2, "depth ceiling")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*path == "" && command != "artifacts") {
		return fmt.Errorf("-db is required; positional arguments are not accepted")
	}
	encode := func(v any) error { return json.NewEncoder(out).Encode(v) }
	switch command {
	case "init", "artifacts":
		var c author.Template
		var err error
		switch *candidate {
		case "a":
			c, err = a.New()
		case "b":
			c, err = b.New()
		default:
			c, err = substitution.Probe(*candidate)
		}
		if err != nil {
			return err
		}
		d, err := invocationjoin.New(c, substitution.Config{Request: substitution.Request{ID: *id, Value: *value, Mode: *mode}, Limits: runtime.ResourceLimits{MaxInvocations: *i, MaxCreatedRuns: *r, MaxDepth: *depth, MaxEffects: 0, MaxInFlight: 1}}, invocationjoin.Options{SMode: *sMode})
		if err != nil {
			return err
		}
		if command == "artifacts" {
			return encode(struct {
				Candidate, Successor, Parent json.RawMessage
				Provision                    runtime.Provision
			}{d.Candidate.Bytes(), d.Successor.Bytes(), d.Parent.Bytes(), d.Provision})
		}
		s, err := runtime.Create(ctx, *path, d.Provision, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if err := s.Close(); err != nil {
			return err
		}
		return encode(map[string]any{"candidate": d.Candidate.ID(), "successor": d.Successor.ID(), "parent": d.Parent.ID(), "fixed_runs": d.FixedRuns})
	case "run", "recover":
		s, err := runtime.Open(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		defer s.Close()
		report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
		if err != nil {
			return err
		}
		return encode(report)
	case "inspect":
		snapshot, err := runtime.Inspect(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(snapshot)
	case "replay":
		report, err := runtime.Replay(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(report)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := execute(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
