// alan-investigate provisions and executes an ordinary bounded investigation.
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
	"path/filepath"
	"strings"
	"syscall"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/investigation"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
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
	const usage = "usage: alan-investigate <init|run|recover|inspect|replay> --db PATH; init requires --question, --model and --corpus-manifest; run/recover require OPENAI_API_KEY"
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
	c := investigation.DefaultConfig()
	var question, corpus string
	var baseline bool
	if command == "init" {
		flags.StringVar(&question, "question", "", "one question, at most 4096 UTF-8 bytes")
		flags.StringVar(&corpus, "corpus-manifest", "", "required phase1f-corpus/v1 manifest; paths relative to the directory containing corpus/")
		flags.BoolVar(&baseline, "baseline", false, "provision the unchanged one-question evidence helper")
		flags.BoolVar(&c.Nested, "nested", false, "insert an ordinary report caller around the investigation")
		flags.IntVar(&c.MaxRevisions, "max-revisions", c.MaxRevisions, "revision ceiling: zero or one")
		flags.StringVar(&c.Model, "model", "", "required explicit OpenAI model ID; no inferred default")
		flags.StringVar(&c.Endpoint, "endpoint", c.Endpoint, "official Responses endpoint or literal loopback controlled test URL")
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
		if c.Model == "" || strings.TrimSpace(question) == "" || corpus == "" {
			return usageError{"init requires --model, --question and --corpus-manifest"}
		}
		if baseline && c.Nested {
			return usageError{"--baseline and --nested are mutually exclusive"}
		}
		if c.MaxRevisions < 0 || c.MaxRevisions > 1 {
			return usageError{"max revisions must be zero or one"}
		}
		assets, err := loadManifest(corpus)
		if err != nil {
			return err
		}
		var p runtime.Provision
		if baseline {
			ec := evidence.Config{Model: c.Model, Endpoint: c.Endpoint, MaxModelRequests: 2, MaxLookupRequests: 1, MaxOutputTokens: c.MaxOutputTokens, MaxResponseBytes: c.MaxResponseBytes, RequestTimeout: c.RequestTimeout}
			p, err = evidence.Provision(question, ec, assets)
		} else {
			p, err = investigation.Provision(question, c, assets)
		}
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
		return encode(map[string]any{"namespace": namespace, "baseline": baseline, "status": "initialized; no model contact"})
	case "run", "recover":
		var err error
		ctx, err = boundary.FromEnvironment(ctx, *db)
		if err != nil {
			return err
		}
		_, probeErr := investigation.Inspect(ctx, *db)
		var report runtime.RunReport
		if errors.Is(probeErr, investigation.ErrNotInvestigation) {
			report, err = evidence.Run(ctx, *db, getenv("OPENAI_API_KEY"))
		} else if probeErr != nil {
			return probeErr
		} else {
			report, err = investigation.Run(ctx, *db, getenv("OPENAI_API_KEY"))
		}
		if err != nil {
			return err
		}
		return encode(report)
	case "inspect":
		v, err := investigation.Inspect(ctx, *db)
		if errors.Is(err, investigation.ErrNotInvestigation) {
			baselineView, baselineErr := evidence.Inspect(ctx, *db)
			if baselineErr != nil {
				return baselineErr
			}
			return encode(baselineView)
		}
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

// loadManifest retains exact bytes and validates the frozen content identity.
// The fixture's editorial version is not the runtime's content-hash version.
func loadManifest(path string) ([]effects.Asset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 {
		return nil, fmt.Errorf("corpus manifest too large")
	}
	var manifest struct {
		Schema    string `json:"schema"`
		Documents []struct {
			ID, Path, SHA256 string
			Bytes            int
		} `json:"documents"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != "phase1f-corpus/v1" || len(manifest.Documents) == 0 || len(manifest.Documents) > effects.MaxAssets {
		return nil, fmt.Errorf("invalid corpus manifest schema or document count")
	}
	root := filepath.Dir(filepath.Dir(path))
	assets := make([]effects.Asset, 0, len(manifest.Documents))
	for _, doc := range manifest.Documents {
		clean := filepath.Clean(doc.Path)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("manifest path must stay beneath the corpus root")
		}
		asset, err := effects.Snapshot(doc.ID, filepath.Join(root, clean))
		if err != nil {
			return nil, err
		}
		if asset.Version != doc.SHA256 || len(asset.Content) != doc.Bytes {
			return nil, fmt.Errorf("corpus document %q does not match its manifest", doc.ID)
		}
		assets = append(assets, asset)
	}
	if err = effects.CheckAssets(assets); err != nil {
		return nil, err
	}
	return assets, nil
}
