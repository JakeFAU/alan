package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/reconciliation"
	"github.com/JakeFAU/alan/runtime"
)

// The host releases an empty service input only after Q for the exact source.
// IDs and application facts reach the resolver exclusively through its program.
func execute(ctx context.Context, args []string) (any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("expected init, issue, resolve, refresh, replay or static")
	}
	if len(args) == 1 {
		switch args[0] {
		case "init", "issue", "resolve", "refresh", "replay", "static":
			return nil, fmt.Errorf("%s requires -db", args[0])
		default:
			// Preserve the original single-path static construction control.
			return static(ctx, args[0])
		}
	}
	command := args[0]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	path := f.String("db", "", "retained Store")
	endpoint := f.String("endpoint", "", "ledger-specific original /apply URL")
	observe := f.Bool("observe-original", false, "source awaits and selects original interaction")
	adopt := f.Bool("adopt", true, "grant separate original evidence adoption")
	queries := f.Int("queries", 1, "one or two logical reads")
	if err := f.Parse(args[1:]); err != nil {
		return nil, err
	}
	if *path == "" || f.NArg() != 0 {
		return nil, fmt.Errorf("require -db and no positional arguments")
	}
	switch command {
	case "static":
		return static(ctx, *path)
	case "replay":
		return runtime.Replay(ctx, *path, runtime.DefaultLimits())
	case "init", "issue", "resolve", "refresh":
	default:
		return nil, fmt.Errorf("unknown operation %q", command)
	}
	lookup, err := receiptquery.LookupEndpoint(*endpoint)
	if err != nil {
		return nil, err
	}
	parent := runtime.RunID{Subject: "parent", Generation: 1}
	resolver := runtime.RunID{Subject: "resolver", Generation: 1}
	if command == "init" {
		p, err := reconciliation.New(reconciliation.Options{Queries: *queries, ObserveOriginal: *observe})
		if err != nil {
			return nil, err
		}
		p.Reconciliation = &runtime.ReconciliationProfile{Grants: []runtime.ReceiptGrant{{Resolver: resolver, Invocation: `["parent",1,1,0]`, Member: "source", Sequence: 2, Endpoint: *endpoint, Adopt: *adopt}}}
		s, err := runtime.Create(ctx, *path, p, runtime.DefaultLimits())
		if err != nil {
			return nil, err
		}
		return map[string]any{"namespace": s.Namespace(), "grant": p.Reconciliation.Grants[0]}, s.Close()
	}
	s, err := runtime.Open(ctx, *path, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	defer s.Close()
	opts := runtime.RunOptions{MaxSteps: 240, MaxDispatch: 240, MaxInFlight: 1, ExternalTimeout: 60 * time.Second, Adapters: map[string]effects.Adapter{
		receiptquery.CounterCapability: receiptquery.Counter{Endpoint: *endpoint, Mode: "success"},
		receiptquery.ReadCapability:    receiptquery.Reader{Endpoint: lookup},
	}}
	if command == "issue" {
		return s.Run(ctx, opts)
	}
	// Recovery can finish ordinary delivery of the genuine handoff before the
	// service gate. An already reserved mutation cannot be re-exposed here.
	if _, err = s.Run(ctx, opts); err != nil {
		return nil, err
	}
	p, err := s.Snapshot(ctx, parent)
	if err != nil {
		return nil, err
	}
	inv := p.Projection.Data["invocation"].Text
	// PrepareInvocation observes the real invocation, without creating an admission.
	before, err := s.PrepareInvocation(ctx, inv)
	if err != nil {
		return nil, err
	}
	// A published preparation omits its freeze. PrepareInvocation above already
	// established an admission, so this returns its retained original membership.
	admission, err := s.AdmitInvocation(ctx, inv)
	if err != nil {
		return nil, err
	}
	if !admission.Accepted || len(admission.Members) != 1 {
		return nil, fmt.Errorf("expected singleton issuing invocation")
	}
	src := admission.Members[0]
	if err = s.AwaitDispatchQuiescence(ctx, runtime.DispatchAuthority{Namespace: s.Namespace(), Run: src}); err != nil {
		return nil, err
	}
	ob := "start"
	if command == "refresh" {
		ob = "refresh"
	}
	a, err := s.AdmitInput(ctx, runtime.Input{TransportID: "public-service-" + ob, LogicalID: "public-service-" + ob, Target: resolver.Subject, Generation: 1, Obligation: ob, Payload: map[string]runtime.Value{}})
	if err != nil {
		return nil, err
	}
	if !a.Accepted {
		return nil, fmt.Errorf("service input rejected: %s", a.Reason)
	}
	run, err := s.Run(ctx, opts)
	if err != nil {
		return nil, err
	}
	after, err := s.PrepareInvocation(ctx, inv)
	if err != nil {
		return nil, err
	}
	states := map[string]runtime.RunState{}
	for name, id := range map[string]runtime.RunID{"source": src, "parent": parent, "resolver": resolver} {
		state, err := s.Snapshot(ctx, id)
		if err != nil {
			return nil, err
		}
		states[name] = state
	}
	return map[string]any{"Q": src, "namespace": s.Namespace(), "before": before, "after": after, "states": states, "run": run}, nil
}
