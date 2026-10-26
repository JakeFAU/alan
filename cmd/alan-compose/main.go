// alan-compose executes a retained public collection with post-init work data.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"unicode/utf8"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
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
func runCLI(ctx context.Context, args []string, out io.Writer) error {
	const usage = "usage: alan-compose <init|submit|admit|run|recover|inspect|replay> --db PATH; init [--nested] [--leaf deterministic|evidence] [--capacity 1|2]; submit --id ID --input FILE; run [--controlled-endpoint http://127.0.0.1:PORT]"
	if len(args) == 0 {
		return usageError{usage}
	}
	command := args[0]
	if command == "help" || command == "--help" {
		_, err := fmt.Fprintln(out, usage)
		return err
	}
	switch command {
	case "init", "submit", "admit", "run", "recover", "inspect", "replay":
	default:
		return usageError{usage}
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	db := f.String("db", "", "store path")
	var nested bool
	leaf := "deterministic"
	capacity := 1
	var input, identity, endpoint string
	switch command {
	case "init":
		f.BoolVar(&nested, "nested", false, "insert one ordinary wrapper")
		f.StringVar(&leaf, "leaf", leaf, "deterministic or evidence")
		f.IntVar(&capacity, "capacity", 1, "persisted physical capacity: 1 or 2")
	case "submit", "admit":
		f.StringVar(&input, "input", "", "JSON list of item_id plus number or question")
		f.StringVar(&identity, "id", "", "stable work transport identity")
	case "run", "recover":
		f.StringVar(&endpoint, "controlled-endpoint", "", "explicit literal loopback Responses endpoint; no live providers")
	}
	f.Usage = func() { fmt.Fprintln(out, usage); f.SetOutput(out); f.PrintDefaults(); f.SetOutput(io.Discard) }
	if err := f.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err.Error()}
	}
	if *db == "" || f.NArg() != 0 {
		return usageError{"--db is required; positional arguments are not accepted"}
	}
	encode := func(v any) error { e := json.NewEncoder(out); e.SetIndent("", "  "); return e.Encode(v) }
	switch command {
	case "init":
		if capacity != 1 && capacity != 2 {
			return usageError{"capacity must be 1 or 2"}
		}
		template, err := worktree.Deterministic()
		switch leaf {
		case "deterministic":
		case "evidence":
			template, err = worktree.Evidence(evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
		default:
			return usageError{"leaf must be deterministic or evidence"}
		}
		if err != nil {
			return err
		}
		d, err := worktree.New(template, nested, worktree.DefaultLimits(int64(capacity)))
		if err != nil {
			return err
		}
		if leaf == "evidence" {
			asset, err := effects.NewAsset("durability", "controlled:phase1g", []byte("WAL records changes before updating the main database.\n"))
			if err != nil {
				return err
			}
			d.Provision.Assets = []effects.Asset{asset}
		}
		s, err := runtime.Create(ctx, *db, d.Provision, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		namespace := s.Namespace()
		if err = s.Close(); err != nil {
			return err
		}
		return encode(map[string]any{"namespace": namespace, "status": "initialized; awaiting work", "collection_template": d.Collection.ID(), "entry_template": d.Entry.ID(), "leaf_template": d.Leaf.ID(), "static_runs": len(d.Provision.Runs), "templates": len(d.Provision.Composition.Templates)})
	case "submit", "admit":
		if input == "" || identity == "" {
			return usageError{"submit requires --input and --id"}
		}
		items, err := readItems(input)
		if err != nil {
			return usageError{err.Error()}
		}
		s, err := runtime.Open(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		defer s.Close()
		result, err := worktree.Submit(ctx, s, identity, items)
		if err != nil {
			return err
		}
		if err = encode(result); err != nil {
			return err
		}
		if !result.Accepted {
			return fmt.Errorf("transport input rejected: %s", result.Reason)
		}
		return nil
	case "run", "recover":
		adapters := map[string]effects.Adapter{evidence.LookupCapability: lookup.New()}
		if endpoint != "" {
			u, err := url.Parse(endpoint)
			if err != nil || u.Scheme != "http" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() || u.User != nil {
				return usageError{"controlled endpoint must be literal loopback HTTP"}
			}
			model, err := openai.New(openai.Config{APIKey: "phase1g-controlled", Model: "controlled", Endpoint: endpoint, MaxOutputTokens: 1024, MaxResponseBytes: 65536})
			if err != nil {
				return err
			}
			adapters[evidence.ModelCapability] = model
		}
		ctx, err := boundary.FromEnvironment(ctx, *db)
		if err != nil {
			return err
		}
		s, err := runtime.Open(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		defer s.Close()
		report, err := s.Run(ctx, runtime.RunOptions{Adapters: adapters, MaxSteps: 10000, MaxDispatch: 1000, MaxInFlight: 2})
		if err != nil {
			return err
		}
		return encode(report)
	case "inspect":
		snapshot, err := runtime.Inspect(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		var result map[string]schema.Value
		for _, row := range snapshot.Tables["runs"] {
			if row["subject"] == "root" {
				raw, ok := row["projection"].(string)
				if !ok {
					return fmt.Errorf("invalid root projection")
				}
				var p runtime.Projection
				if err = json.Unmarshal([]byte(raw), &p); err != nil {
					return err
				}
				result = p.Data
			}
		}
		return encode(struct {
			Result   map[string]schema.Value
			Snapshot runtime.Inspection
		}{result, snapshot})
	case "replay":
		report, err := runtime.Replay(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		return encode(report)
	}
	return nil
}

func readItems(path string) (schema.Value, error) {
	f, err := os.Open(path)
	if err != nil {
		return schema.Value{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return schema.Value{}, err
	}
	if len(raw) > 1<<20 || !utf8.Valid(raw) {
		return schema.Value{}, fmt.Errorf("work JSON exceeds byte bound or contains invalid UTF-8")
	}
	if err = checkItemKeys(raw); err != nil {
		return schema.Value{}, err
	}
	var rows []struct {
		ItemID   string  `json:"item_id"`
		Number   *int64  `json:"number"`
		Question *string `json:"question"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&rows); err != nil {
		return schema.Value{}, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return schema.Value{}, fmt.Errorf("work JSON must contain exactly one list")
	}
	if rows == nil {
		return schema.Value{}, fmt.Errorf("work JSON must be a list, including [] for empty work")
	}
	records := make([]map[string]schema.Value, len(rows))
	for i, row := range rows {
		if row.ItemID == "" || (row.Number == nil) == (row.Question == nil) {
			return schema.Value{}, fmt.Errorf("item %d requires item_id and exactly one number or question", i)
		}
		records[i] = map[string]schema.Value{"item_id": schema.Text(row.ItemID)}
		if row.Number != nil {
			records[i]["number"] = schema.Int(*row.Number)
		} else {
			records[i]["question"] = schema.Text(*row.Question)
		}
	}
	return schema.List(records...), nil
}

// checkItemKeys requires canonical field names and rejects duplicates before
// encoding/json can case-fold aliases or replace an earlier identity/payload.
func checkItemKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('[') {
		return fmt.Errorf("work JSON must be a list")
	}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return err
		}
		if token != json.Delim('{') {
			return fmt.Errorf("work items must be records")
		}
		seen := map[string]bool{}
		for decoder.More() {
			token, err = decoder.Token()
			if err != nil {
				return err
			}
			name, ok := token.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid item field")
			}
			if name != "item_id" && name != "number" && name != "question" {
				return fmt.Errorf("unknown item field %q", name)
			}
			seen[name] = true
			var value json.RawMessage
			if err = decoder.Decode(&value); err != nil {
				return err
			}
		}
		if _, err = decoder.Token(); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
