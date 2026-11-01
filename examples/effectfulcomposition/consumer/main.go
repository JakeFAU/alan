// Command consumer drives the public two-effect composition over one retained Store.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	product "github.com/JakeFAU/alan/examples/effectfulcomposition"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

var parent = runtime.RunID{Subject: "parent", Generation: 1}

type roleState struct {
	Admission   runtime.InvocationAdmission
	Original    runtime.RunState
	Resolver    runtime.RunState
	Preparation runtime.InvocationPreparation
}

func role(ctx context.Context, s *runtime.Store, p runtime.RunState, name string) (roleState, error) {
	var r roleState
	inv := p.Projection.Data[name+"_invocation"].Text
	if inv == "" {
		return r, fmt.Errorf("missing %s invocation", name)
	}
	// Selecting this genuine target proves the child already exists. Asking for
	// its retained admission below cannot authorize a not-yet-admitted child.
	if p.Projection.Data[name+"_target"].Text == "" {
		return r, fmt.Errorf("missing %s authentic caller handoff", name)
	}
	var err error
	r.Admission, err = s.AdmitInvocation(ctx, inv)
	if err != nil {
		return r, err
	}
	if !r.Admission.Accepted || len(r.Admission.Members) != 1 {
		return r, fmt.Errorf("expected one admitted issuer for %s", name)
	}
	r.Original, err = s.Snapshot(ctx, r.Admission.Members[0])
	if err != nil {
		return r, err
	}
	r.Resolver, err = s.Snapshot(ctx, runtime.RunID{Subject: "resolver_" + name, Generation: 1})
	if err != nil {
		return r, err
	}
	r.Preparation, err = s.PrepareInvocation(ctx, inv)
	return r, err
}

func generation(s *runtime.Store, id runtime.RunID) runtime.DispatchAuthority {
	return runtime.DispatchAuthority{Namespace: s.Namespace(), Run: id}
}

// The host checks the cut and can obtain Q for each revoked source. None of
// these observations are passed to the ordinary programs.
func frontier(ctx context.Context, s *runtime.Store) (any, error) {
	p, err := s.Snapshot(ctx, parent)
	if err != nil {
		return nil, err
	}
	roles := map[string]roleState{}
	targets := map[string]bool{}
	for _, name := range []string{"l", "r"} {
		r, err := role(ctx, s, p, name)
		if err != nil {
			return nil, err
		}
		roles[name] = r
		target := r.Original.Projection.Data["original"].Text
		if targets[target] || target == "" || r.Original.Projection.Data["echo"].Text != target ||
			r.Resolver.Projection.Data["received"].Text != target || p.Projection.Data[name+"_target"].Text != target ||
			p.Projection.Data[name+"_resolver_ack"].Text != target {
			return nil, fmt.Errorf("missing or misbound %s handoff", name)
		}
		targets[target] = true
		f := r.Preparation.Freeze
		if r.Original.Projection.Status != "succeeded" || r.Resolver.Projection.Status != "open" ||
			r.Resolver.Projection.Data["query1"].Text != "" || r.Preparation.Ready || r.Preparation.Pending != "external_outcome" ||
			f.Status != "succeeded" || len(f.ExternalIntents) != 1 || f.ExternalIntents[0] != target ||
			len(f.ReplyIntents) != 1 {
			return nil, fmt.Errorf("unexpected %s original/query frontier", name)
		}
		if err := s.AwaitDispatchQuiescence(ctx, generation(s, r.Original.ID)); err != nil {
			return nil, err
		}
		if err := s.AwaitDispatchQuiescence(ctx, generation(s, r.Resolver.ID)); !errors.Is(err, runtime.ErrDispatchAuthorityOpen) {
			return nil, fmt.Errorf("%s resolver not eligible: %v", name, err)
		}
	}
	if p.Projection.Status != "open" || p.Projection.Data["tree_invocation"].Text == "" {
		return nil, errors.New("missing open caller or committed pure invocation")
	}
	if err := s.AwaitDispatchQuiescence(ctx, generation(s, p.ID)); !errors.Is(err, runtime.ErrDispatchAuthorityOpen) {
		return nil, fmt.Errorf("parent not eligible: %v", err)
	}
	return map[string]any{"namespace": s.Namespace(), "parent": p, "roles": roles}, nil
}

type leafOptions struct {
	candidate, mode  string
	id, value, delta int64
	queries          int
	observe, adopt   bool
}

func (l *leafOptions) bind(f *flag.FlagSet, prefix, implementation string, value, delta int64) {
	f.StringVar(&l.candidate, prefix, implementation, "pure leaf A or B")
	f.StringVar(&l.mode, prefix+"-mode", "answer", "pure request mode")
	f.Int64Var(&l.id, prefix+"-id", 1, "pure request ID")
	f.Int64Var(&l.value, prefix+"-value", value, "pure request value")
	f.Int64Var(&l.delta, prefix+"-delta", delta, "original counter delta")
	f.IntVar(&l.queries, prefix+"-queries", 0, "one or two logical reads, overriding -queries")
	f.BoolVar(&l.observe, prefix+"-observe", false, "issuer selects original interaction")
	f.BoolVar(&l.adopt, prefix+"-adopt", true, "grant import of the original receipt")
}

func (l leafOptions) role(commonQueries int) product.RoleOptions {
	n := l.queries
	if n == 0 {
		n = commonQueries
	}
	return product.RoleOptions{Delta: l.delta, Queries: n, ObserveOriginal: l.observe, ReadOnly: !l.adopt}
}

func execute(ctx context.Context, args []string) (result any, err error) {
	if len(args) == 0 {
		return nil, errors.New("require init, issue, frontier, service, resolve, refresh or replay")
	}
	command := args[0]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("db", "", "retained Store")
	endpoint := f.String("endpoint", "", "ledger-specific apply URL")
	selected := f.String("role", "", "l or r for empty resolver service input")
	var left, right leafOptions
	left.bind(f, "left", "A", 0, 1)
	right.bind(f, "right", "B", 1, 2)
	relay := f.String("relay", "", "pure Relay location: above, left, right or empty")
	serviceInput := f.String("input", "start", "empty service input: start or refresh")
	runAfterService := f.Bool("run", false, "run the scheduler after service enqueues its input")
	invocations := f.Int64("invocations", -1, "override parent-root invocation budget")
	created := f.Int64("runs", -1, "override parent-root created-run budget")
	depth := f.Int64("depth", -1, "override parent-root depth budget")
	effectBudget := f.Int64("effects", -1, "override parent-root effect budget")
	queries := f.Int("queries", 2, "logical reads per role")
	capacity := f.Int("capacity", 2, "physical dispatch capacity")
	if err := f.Parse(args[1:]); err != nil {
		return nil, err
	}
	if *path == "" || f.NArg() != 0 || *capacity < 1 || *capacity > 2 {
		return nil, errors.New("require -db, capacity one or two, and no positional arguments")
	}
	if command == "replay" {
		return runtime.Replay(ctx, *path, runtime.DefaultLimits())
	}
	switch command {
	case "frontier", "init", "issue", "service", "resolve", "refresh":
	default:
		return nil, fmt.Errorf("unknown operation %q", command)
	}
	var lookup string
	if command != "frontier" && (command != "service" || *runAfterService) {
		lookup, err = receiptquery.LookupEndpoint(*endpoint)
		if err != nil {
			return nil, err
		}
	}
	if command == "init" {
		tree, err := product.PureFork(left.candidate, right.candidate, *relay)
		if err != nil {
			return nil, err
		}
		d, err := product.New(product.Options{Tree: tree, Inputs: map[string]schema.Value{
			"l_id": schema.Int(left.id), "l_value": schema.Int(left.value), "l_mode": schema.Text(left.mode),
			"r_id": schema.Int(right.id), "r_value": schema.Int(right.value), "r_mode": schema.Text(right.mode),
		}, Left: left.role(*queries), Right: right.role(*queries), Endpoint: *endpoint, Capacity: int64(*capacity)})
		if err != nil {
			return nil, err
		}
		for i := range d.Provision.Composition.Roots {
			r := &d.Provision.Composition.Roots[i]
			if r.Run != parent {
				continue
			}
			if *invocations >= 0 {
				r.Limits.MaxInvocations = *invocations
			}
			if *created >= 0 {
				r.Limits.MaxCreatedRuns = *created
			}
			if *depth >= 0 {
				r.Limits.MaxDepth = *depth
			}
			if *effectBudget >= 0 {
				r.Limits.MaxEffects = *effectBudget
			}
		}
		s, err := runtime.Create(ctx, *path, d.Provision, runtime.DefaultLimits())
		if err != nil {
			return nil, err
		}
		return map[string]any{"namespace": s.Namespace(), "grants": d.Provision.Reconciliation.Grants, "roots": d.Provision.Composition.Roots}, s.Close()
	}
	s, err := runtime.Open(ctx, *path, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	if command == "frontier" {
		return frontier(ctx, s)
	}
	if command == "resolve" || command == "refresh" || command == "service" {
		if *selected != "l" && *selected != "r" {
			return nil, errors.New("select -role l or r")
		}
		p, err := s.Snapshot(ctx, parent)
		if err != nil {
			return nil, err
		}
		r, err := role(ctx, s, p, *selected)
		if err != nil {
			return nil, err
		}
		if target := r.Original.Projection.Data["original"].Text; target == "" || r.Resolver.Projection.Data["received"].Text != target || p.Projection.Data[*selected+"_target"].Text != target {
			return nil, errors.New("resolver has not received its authentic target")
		}
		if err = s.AwaitDispatchQuiescence(ctx, generation(s, r.Original.ID)); err != nil {
			return nil, err
		}
		ob := "start"
		if command == "service" {
			ob = *serviceInput
		}
		if command == "refresh" {
			ob = "refresh"
		}
		if ob != "start" && ob != "refresh" {
			return nil, errors.New("require empty start or refresh service input")
		}
		id := "consumer-" + r.Resolver.ID.Subject + "-" + ob
		a, err := s.AdmitInput(ctx, runtime.Input{TransportID: id, LogicalID: id, Target: r.Resolver.ID.Subject, Generation: r.Resolver.ID.Generation, Obligation: ob, Payload: map[string]schema.Value{}})
		if err != nil {
			return nil, err
		}
		if !a.Accepted {
			return nil, fmt.Errorf("service input rejected: %s", a.Reason)
		}
		if command == "service" && !*runAfterService {
			return a, nil
		}
	}
	options := runtime.RunOptions{MaxSteps: 2048, MaxDispatch: 2048, MaxInFlight: *capacity, ExternalTimeout: 90 * time.Second, Adapters: map[string]effects.Adapter{
		receiptquery.CounterCapability: receiptquery.Counter{Endpoint: *endpoint, Mode: "success"}, receiptquery.ReadCapability: receiptquery.Reader{Endpoint: lookup},
	}}
	return s.Run(ctx, options)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	r, err := execute(ctx, os.Args[1:])
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(r)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
