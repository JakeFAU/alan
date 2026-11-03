// Command quickstart runs an ordinary program that looks up a line in a retained document.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func create(ctx context.Context, path string, limits runtime.Limits) (*runtime.Store, error) {
	program, err := author.Compile(author.Definition{
		Name:    "find-line",
		Outputs: schema.Record{"line": lookup.OutputSchema()["snippet"]},
		Ports: map[string]author.Port{
			"continue": {Peer: "local", Local: true, Obligation: "continue", Receive: schema.Record{}},
		},
		Effects: map[string]author.Effect{
			"search": {Capability: "corpus", Input: lookup.InputSchema(), Output: lookup.OutputSchema()},
		},
		Steps: []author.Step{
			author.Set("query", author.String("SQLite")),
			author.Receive("continue", nil),
			author.Request("request", "search", map[string]author.Expr{"query": author.Var("query")}, nil),
			author.Await("search", "request", map[string]string{"snippet": "line", "valid": "found"}),
			author.IfEqual(author.Var("found"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
	if err != nil {
		return nil, err
	}
	p, err := (author.Application{"reader": {Artifact: program}}).Provision()
	if err != nil {
		return nil, err
	}
	asset, err := effects.NewAsset("guide", "quickstart", []byte("A program can wait for input.\nSQLite retains its state."))
	if err != nil {
		return nil, err
	}
	p.Assets = []effects.Asset{asset}
	return runtime.Create(ctx, path, p, limits)
}

const usage = "usage: quickstart {start|inspect|resume|replay} STORE"

func run(ctx context.Context, action, path string) (err error) {
	limits := runtime.DefaultLimits()
	var s *runtime.Store
	switch action {
	case "start":
		s, err = create(ctx, path, limits)
	case "resume":
		s, err = runtime.Open(ctx, path, limits)
	case "inspect":
		snapshot, err := runtime.Inspect(ctx, path, limits)
		if err != nil {
			return err
		}
		for _, row := range snapshot.Tables["runs"] {
			fmt.Printf("%s generation=%v epoch=%v sequence=%v projection=%s\n",
				row["subject"], row["generation"], row["epoch"], row["sequence"], row["projection"])
		}
		fmt.Printf("programs=%d attempts=%d results=%d\n",
			len(snapshot.Tables["programs"]), len(snapshot.Tables["adapter_attempts"]), len(snapshot.Tables["adapter_results"]))
		return nil
	case "replay":
		report, err := runtime.Replay(ctx, path, limits)
		if err != nil {
			return err
		}
		for _, state := range report.Runs {
			fmt.Printf("%s %s line=%q\n", state.ID.Subject, state.Projection.Status, state.Projection.Data["line"].Text)
		}
		fmt.Printf("verified commits=%d intents=%d\n", report.HistoryCount, report.IntentCount)
		return nil
	default:
		return errors.New(usage)
	}
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Close()) }()

	options := runtime.RunOptions{MaxSteps: 32, MaxDispatch: 8}
	if action == "resume" {
		input, err := s.AdmitInput(ctx, runtime.Input{
			TransportID: "continue-delivery", LogicalID: "continue-1", Target: "reader", Generation: 1,
			Obligation: "continue", Payload: map[string]schema.Value{},
		})
		if err != nil {
			return err
		}
		if !input.Accepted {
			return fmt.Errorf("input rejected: %s", input.Reason)
		}
		options.Adapters = map[string]effects.Adapter{"corpus": lookup.New()}
	}
	report, err := s.Run(ctx, options)
	if err == nil {
		fmt.Printf("quiescent=%t open_runs=%d\n", report.Quiescent, len(report.OpenRuns))
	}
	return err
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
