// alan-adaptive executes a retained public adaptive component with post-init work data.
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
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/worktree"
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
	const usage = "usage: alan-adaptive <init|submit|admit|run|recover|inspect|replay> --db PATH; init [--nested] [--leaf scalar|controlled|evidence] [--capacity 1|2]; submit --id ID --input FILE; run [--controlled-endpoint http://127.0.0.1:PORT]"
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
	leaf := "scalar"
	capacity := 1
	var input, identity, endpoint string
	switch command {
	case "init":
		f.BoolVar(&nested, "nested", false, "insert one ordinary wrapper")
		f.StringVar(&leaf, "leaf", leaf, "scalar, controlled, or evidence")
		f.IntVar(&capacity, "capacity", 1, "persisted physical capacity: 1 or 2")
	case "submit", "admit":
		f.StringVar(&input, "input", "", "complete JSON application request including seeds and limits")
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
		var component author.Template
		var err error
		var template author.Template
		switch leaf {
		case "scalar", "controlled":
			template, err = adaptive.ScalarLeaf(leaf == "controlled")
			if err == nil {
				component, err = adaptive.NewScalar(template)
			}
		case "evidence":
			template, err = worktree.Evidence(evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
			if err == nil {
				component, err = adaptive.NewEvidence(template)
			}
		default:
			return usageError{"leaf must be scalar, controlled, or evidence"}
		}
		if err != nil {
			return err
		}
		d, err := adaptive.New(component, nested, adaptive.DefaultLimits(int64(capacity)))
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
		return encode(map[string]any{"namespace": namespace, "status": "initialized; awaiting work", "adaptive_template": d.Adaptive.ID(), "entry_template": d.Entry.ID(), "leaf_template": template.ID(), "static_runs": len(d.Provision.Runs), "templates": len(d.Provision.Composition.Templates)})
	case "submit", "admit":
		if input == "" || identity == "" {
			return usageError{"submit requires --input and --id"}
		}
		items, err := readRequest(input)
		if err != nil {
			return usageError{err.Error()}
		}
		s, err := runtime.Open(ctx, *db, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		defer s.Close()
		result, err := adaptive.Submit(ctx, s, identity, items)
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
			adapters[adaptive.ScalarCapability] = scalarHTTP{endpoint: endpoint}
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

// readRequest accepts exact complete application data. It never creates work or
// defaults persisted limits. Duplicate keys are rejected before JSON decoding.
func readRequest(path string) (map[string]schema.Value, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 || !utf8.Valid(raw) {
		return nil, fmt.Errorf("request exceeds byte bound or has invalid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	if err = uniqueJSON(d); err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, fmt.Errorf("request must contain exactly one object")
	}
	var q struct {
		Seeds *[]struct {
			WorkID   *int64  `json:"work_id"`
			Value    *int64  `json:"value"`
			Question *string `json:"question"`
		} `json:"seeds"`
		MaxRounds *int64 `json:"max_rounds"`
		MaxWork   *int64 `json:"max_work"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&q); err != nil {
		return nil, err
	}
	if q.Seeds == nil || q.MaxRounds == nil || q.MaxWork == nil {
		return nil, fmt.Errorf("request requires seeds, max_rounds and max_work")
	}
	records := []map[string]schema.Value{}
	for i, row := range *q.Seeds {
		if row.WorkID == nil || (row.Value == nil) == (row.Question == nil) {
			return nil, fmt.Errorf("seed %d requires work_id and exactly one value or question", i)
		}
		r := map[string]schema.Value{"work_id": schema.Int(*row.WorkID)}
		if row.Value != nil {
			r["value"] = schema.Int(*row.Value)
		} else {
			r["question"] = schema.Text(*row.Question)
		}
		records = append(records, r)
	}
	return map[string]schema.Value{"seeds": schema.List(records...), "max_rounds": schema.Int(*q.MaxRounds), "max_work": schema.Int(*q.MaxWork)}, nil
}
func uniqueJSON(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate JSON field")
			}
			seen[name] = true
			// Field spelling is strict; encoding/json otherwise accepts case aliases.
			switch name {
			case "seeds", "max_rounds", "max_work", "work_id", "value", "question":
			default:
				return fmt.Errorf("unknown request field %q", name)
			}
			if err = uniqueJSON(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = uniqueJSON(d); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid JSON container")
	}
	_, err = d.Token()
	return err
}
