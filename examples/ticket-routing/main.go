// Command ticket-routing is a standalone, credential-free ALAN application.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

const capability = "routing-desk"

var ticket = schema.Record{"priority": {Kind: "int"}}
var decision = schema.Record{"queue": {Kind: "string", MaxBytes: 24}}
var triager = runtime.RunID{Subject: "triager", Generation: 1}

func program() (author.Artifact, error) {
	return author.Compile(author.Definition{
		Name: "route-ticket", Outputs: decision,
		Ports: map[string]author.Port{
			"inbox": {Peer: "local", Local: true, Obligation: "ticket", Receive: ticket},
		},
		Effects: map[string]author.Effect{
			"route": {Capability: capability, Input: ticket, Output: decision},
		},
		Steps: []author.Step{
			author.Receive("inbox", map[string]string{"priority": "priority"}),
			author.Request("call", "route", map[string]author.Expr{"priority": author.Var("priority")}, nil),
			author.Await("route", "call", map[string]string{"queue": "queue", schema.ResultValid: "ok"}),
			author.IfEqual(author.Var("ok"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		},
	})
}

type routingDesk struct{ calls atomic.Int32 }

func (*routingDesk) Prepare(r effects.Request) (effects.Prepared, error) {
	if r.Capability != capability || r.Effect != "route" {
		return effects.Prepared{}, errors.New("unknown routing request")
	}
	if err := ticket.Validate(r.Payload); err != nil {
		return effects.Prepared{}, err
	}
	return effects.Prepared{Request: r, Protocol: "routing-table/v1", MaxResponseBytes: 128}, nil
}

func (d *routingDesk) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	d.calls.Add(1)
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	if ctx.Err() != nil {
		return c
	}
	queue := "standard"
	if p.Request.Payload["priority"].Number >= 3 {
		queue = "expedited"
	}
	c.Disposition = "validated"
	c.Payload = map[string]schema.Value{"queue": schema.Text(queue)}
	return c
}

func start(ctx context.Context, path string) (err error) {
	a, err := program()
	if err != nil {
		return err
	}
	p, err := (author.Application{"triager": {Artifact: a}}).Provision()
	if err != nil {
		return err
	}
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	input, err := s.AdmitInput(ctx, runtime.Input{
		TransportID: "webhook-783", LogicalID: "ticket-783", Target: triager.Subject,
		Generation: triager.Generation, Obligation: "ticket", Payload: map[string]schema.Value{"priority": schema.Int(3)},
	})
	if err != nil {
		return err
	}
	if !input.Accepted {
		return fmt.Errorf("ticket rejected: %s", input.Reason)
	}
	_, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 8})
	if !errors.Is(err, runtime.ErrCapabilityUnavailable) {
		return fmt.Errorf("expected unavailable routing desk, got %v", err)
	}
	cut, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	run, found := cut.Run(triager)
	if !found || run.Projection.Status != "open" || len(cut.Effects) != 1 ||
		cut.Effects[0].State != runtime.EffectCommitted || cut.Effects[0].AdapterAttempt != nil {
		return errors.New("expected an open ticket with an unspent routing request")
	}
	fmt.Println("ticket accepted; routing request retained; attempts=0")
	return nil
}

func resume(ctx context.Context, path string) (err error) {
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	desk := new(routingDesk)
	r, err := s.Run(ctx, runtime.RunOptions{
		MaxSteps: 20, MaxDispatch: 8, Adapters: map[string]effects.Adapter{capability: desk},
	})
	if err != nil {
		return err
	}
	if !r.Quiescent || len(r.OpenRuns) != 0 || desk.calls.Load() != 1 {
		return errors.New("expected one routing call and a completed ticket")
	}
	fmt.Println("routing calls=1; open runs=0")
	return nil
}

func run(ctx context.Context, action, path string) error {
	switch action {
	case "start":
		return start(ctx, path)
	case "resume":
		return resume(ctx, path)
	case "inspect":
		cut, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		r, found := cut.Run(triager)
		if !found || r.Projection.Status != "succeeded" || r.Projection.Data["queue"].Text != "expedited" ||
			len(cut.Effects) != 1 || cut.Effects[0].State != runtime.EffectSettled || cut.Effects[0].AdapterResult == nil {
			return errors.New("unexpected ticket or effect state")
		}
		fmt.Println("triager succeeded; queue=expedited")
	case "replay":
		r, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if r.IntentCount != 1 || r.HistoryCount != 5 || len(r.Runs) != 1 || r.Runs[0].ID != triager ||
			r.Runs[0].Projection.Status != "succeeded" || r.Runs[0].Projection.Data["queue"].Text != "expedited" {
			return errors.New("unexpected replayed decision")
		}
		fmt.Println("replay verified 5 commits, 1 intent; queue=expedited")
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: triage {start|resume|inspect|replay} STORE")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
