// Command consumer demonstrates the exported Phase 1J author/runtime boundary.
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
	join "github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

func execute(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected init, run, recover, inspect or replay")
	}
	command := args[0]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "retained database")
	candidate := flags.String("candidate", "a", "a, b or named probe")
	sMode := flags.String("s-mode", "answer", "answer, fail, absent or pending S")
	wrongRequest := flags.Bool("wrong-request", false, "send the other request id")
	premature := flags.Bool("premature-report", false, "report before consuming S")
	id := flags.Int64("id", 1, "application id")
	value := flags.Int64("value", 0, "Boolean integer")
	mode := flags.String("mode", "answer", "request mode")
	i := flags.Int64("i", 3, "invocation ceiling")
	r := flags.Int64("r", 3, "created-run ceiling")
	depth := flags.Int64("depth", 3, "depth ceiling")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *path == "" || flags.NArg() != 0 {
		return fmt.Errorf("-db is required; positional arguments are not accepted")
	}
	encode := func(v any) error { return json.NewEncoder(out).Encode(v) }
	switch command {
	case "init":
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
		d, err := join.New(c, substitution.Config{Shape: "parallel", Request: substitution.Request{ID: *id, Value: *value, Mode: *mode}, Limits: runtime.ResourceLimits{MaxInvocations: *i, MaxCreatedRuns: *r, MaxDepth: *depth, MaxInFlight: 1}}, join.Options{SMode: *sMode, WrongRequest: *wrongRequest, PrematureReport: *premature})
		if err != nil {
			return err
		}
		s, err := runtime.Create(ctx, *path, d.Provision, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if err = s.Close(); err != nil {
			return err
		}
		return encode(map[string]any{"candidate": c.ID(), "entry": d.Entry.ID(), "fixed_runs": d.FixedRuns})
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
		s, err := runtime.Inspect(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(s)
	case "replay":
		r, err := runtime.Replay(ctx, *path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(r)
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
