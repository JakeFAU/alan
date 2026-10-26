// alan-evidence provisions and executes the ordinary evidence example.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"io"
	"os"
	"os/signal"
	"syscall"
)

type usageError struct{ message string }

func (e usageError) Error() string { return e.message }
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := runCLI(ctx, os.Args[1:], os.Stdout, os.Getenv)
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
func runCLI(ctx context.Context, args []string, out io.Writer, getenv func(string) string) error {
	const usage = "usage: alan-evidence <init|run|recover|inspect|replay> --db PATH; init requires --question and --model; run/recover require OPENAI_API_KEY"
	if len(args) == 0 {
		return usageError{usage}
	}
	command := args[0]
	if command == "--help" || command == "help" {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	switch command {
	case "init", "run", "recover", "inspect", "replay":
	default:
		return usageError{usage}
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	db := flags.String("db", "", "existing store path (new path for init)")
	c := evidence.DefaultConfig()
	var question, corpus string
	if command == "init" {
		flags.StringVar(&question, "question", "", "one question, at most 4096 UTF-8 bytes")
		flags.StringVar(&corpus, "corpus", "", "explicit UTF-8 file to snapshot; omitted uses bundled nonsensitive corpus")
		flags.StringVar(&c.Model, "model", "", "required explicit OpenAI model ID; no inferred default")
		flags.StringVar(&c.Endpoint, "endpoint", c.Endpoint, "official Responses endpoint or literal loopback controlled test URL")
		flags.IntVar(&c.MaxModelRequests, "max-model-requests", c.MaxModelRequests, "logical model ceiling [0,10]")
		flags.IntVar(&c.MaxLookupRequests, "max-lookup-requests", c.MaxLookupRequests, "logical lookup ceiling [0,10]")
		flags.IntVar(&c.MaxOutputTokens, "max-output-tokens", c.MaxOutputTokens, "provider output ceiling [1,32768]")
		flags.IntVar(&c.MaxResponseBytes, "max-response-bytes", c.MaxResponseBytes, "response body ceiling [1,65536]")
		flags.DurationVar(&c.RequestTimeout, "request-timeout", c.RequestTimeout, "positive per-effect transport timeout, at most 10m")
	}
	flags.Usage = func() {
		fmt.Fprintln(out, usage)
		flags.SetOutput(out)
		flags.PrintDefaults()
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err.Error()}
	}
	if flags.NArg() != 0 || *db == "" {
		return usageError{"--db is required; positional arguments are not accepted"}
	}
	encode := func(v any) error { e := json.NewEncoder(out); e.SetIndent("", "  "); return e.Encode(v) }
	switch command {
	case "init":
		if c.Model == "" || question == "" {
			return usageError{"init requires --model and --question"}
		}
		var assets []effects.Asset
		if corpus != "" {
			a, err := effects.Snapshot("selected-corpus", corpus)
			if err != nil {
				return err
			}
			assets = []effects.Asset{a}
		}
		p, err := evidence.Provision(question, c, assets)
		if err != nil {
			return usageError{err.Error()}
		}
		s, err := runtime.Create(ctx, *db, p, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		namespace := s.Namespace()
		if err := s.Close(); err != nil {
			return err
		}
		return encode(map[string]any{"namespace": namespace, "config": c, "status": "initialized; no model contact"})
	case "run", "recover":
		var err error
		ctx, err = boundary.FromEnvironment(ctx, *db)
		if err != nil {
			return err
		}
		report, err := evidence.Run(ctx, *db, getenv("OPENAI_API_KEY"))
		if err != nil {
			return err
		}
		return encode(report)
	case "inspect":
		v, err := evidence.Inspect(ctx, *db)
		if err != nil {
			return err
		}
		return encode(v)
	case "replay":
		v, err := runtime.Replay(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(v)
	}
	return nil
}
