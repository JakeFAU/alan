// Command consumer provisions and runs a finite owning tree using the public API.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type term struct {
	template     author.Template
	paths        []string
	slots, depth int
}

type parser struct{ text string }

func (p *parser) take(token string) bool {
	next, ok := strings.CutPrefix(p.text, token)
	if ok {
		p.text = next
	}
	return ok
}
func (p *parser) parse() (term, error) {
	var node term
	switch {
	case p.take("relay("):
		child, err := p.parse()
		if err != nil {
			return node, err
		}
		if !p.take(")") {
			return node, fmt.Errorf("expected closing relay parenthesis")
		}
		node = child
		node.slots++
		node.depth++
		node.template, err = owningtree.Relay(child.template)
		return node, err
	case p.take("fork("):
		left, err := p.parse()
		if err != nil {
			return node, err
		}
		if !p.take(",") {
			return node, fmt.Errorf("expected fork separator")
		}
		right, err := p.parse()
		if err != nil {
			return node, err
		}
		if !p.take(")") {
			return node, fmt.Errorf("expected closing fork parenthesis")
		}
		node.template, err = owningtree.Fork(left.template, right.template)
		node.slots = 1 + left.slots + right.slots
		node.depth = 1 + max(left.depth, right.depth)
		for _, path := range left.paths {
			node.paths = append(node.paths, "l_"+path)
		}
		for _, path := range right.paths {
			node.paths = append(node.paths, "r_"+path)
		}
		return node, err
	default:
		var candidate author.Template
		var err error
		switch {
		case p.take("A"):
			candidate, err = a.New()
		case p.take("B"):
			candidate, err = b.New()
		default:
			return node, fmt.Errorf("expected A, B, relay or fork near %q", p.text)
		}
		if err != nil {
			return node, err
		}
		node.template, err = owningtree.Leaf(candidate)
		node.paths = []string{""}
		node.slots = 4
		node.depth = 3
		return node, err
	}
}

func inputs(paths []string, raw string) (map[string]schema.Value, error) {
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, err
	}
	if len(rows) != len(paths) {
		return nil, fmt.Errorf("got %d requests for %d leaves", len(rows), len(paths))
	}
	out := map[string]schema.Value{}
	for i, path := range paths {
		r := rows[i]
		var id, value *int64
		var mode *string
		if len(r) != 3 || json.Unmarshal(r["id"], &id) != nil || id == nil || json.Unmarshal(r["value"], &value) != nil || value == nil ||
			json.Unmarshal(r["mode"], &mode) != nil || mode == nil {
			return nil, fmt.Errorf("invalid leaf request %d", i)
		}
		out[path+"id"] = schema.Int(*id)
		out[path+"value"] = schema.Int(*value)
		out[path+"mode"] = schema.Text(*mode)
	}
	return out, nil
}

func execute(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("consumer", flag.ContinueOnError)
	tree := flags.String("term", "fork(A,B)", "A, B, relay(T) or fork(T,T)")
	requests := flags.String("requests", `[{"id":1,"value":0,"mode":"answer"},{"id":1,"value":1,"mode":"absent"}]`, "ordered JSON array of leaf requests")
	path := flags.String("db", "", "new SQLite database path")
	artifacts := flags.Bool("artifacts", false, "emit the complete retained provision and entry template")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || (!*artifacts && *path == "") {
		return fmt.Errorf("provide -db PATH or -artifacts, with no positional arguments")
	}
	p := parser{text: *tree}
	node, err := p.parse()
	if err != nil {
		return err
	}
	if p.text != "" {
		return fmt.Errorf("unexpected trailing term text %q", p.text)
	}
	request, err := inputs(node.paths, *requests)
	if err != nil {
		return err
	}
	prov, err := owningtree.Provision(node.template, request, runtime.ResourceLimits{MaxInvocations: int64(node.slots), MaxCreatedRuns: int64(node.slots), MaxDepth: int64(node.depth), MaxInFlight: 1})
	if err != nil {
		return err
	}
	if *artifacts {
		return json.NewEncoder(out).Encode(struct {
			Tree      json.RawMessage
			Provision runtime.Provision
		}{node.template.Bytes(), prov})
	}
	maxCost := 0
	for _, a := range prov.Programs {
		var program struct{ Code []struct{ Op string } }
		if err := json.Unmarshal(a.Bytes, &program); err != nil {
			return err
		}
		cost := len(program.Code)
		for _, in := range program.Code {
			if in.Op == "foreach" {
				cost++
			}
		}
		maxCost = max(cost, maxCost)
	}
	options := runtime.RunOptions{MaxSteps: 1 + (node.slots+1)*maxCost, MaxDispatch: 1 + 3*node.slots, MaxInFlight: 1}
	store, err := runtime.Create(ctx, *path, prov, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	report, err := store.Run(ctx, options)
	if err != nil {
		store.Close()
		return err
	}
	root, err := store.Snapshot(ctx, runtime.RunID{Subject: "root", Generation: 1})
	closeErr := store.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	inspect, err := runtime.Inspect(ctx, *path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	replay, err := runtime.Replay(ctx, *path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Term, Template        string
		MaxSteps, MaxDispatch int
		Root                  runtime.RunState
		Report                runtime.RunReport
		Inspection            runtime.Inspection
		Replay                runtime.ReplayReport
	}{*tree, node.template.ID(), options.MaxSteps, options.MaxDispatch, root, report, inspect, replay})
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := execute(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
