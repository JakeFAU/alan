//go:build integration

package durabletest

import (
	"context"
	"errors"
	"testing"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/internal/boundary"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func nestedCounter(t *testing.T) rt.Provision {
	t.Helper()
	entry := map[string]author.Port{"caller": {Peer: "caller", Obligation: "call"}}
	template := func(d author.Definition) author.Template {
		t.Helper()
		a, err := author.Compile(d)
		if err != nil {
			t.Fatal(err)
		}
		tmpl, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "caller")
		if err != nil {
			t.Fatal(err)
		}
		return tmpl
	}
	leaf := template(author.Definition{Ports: entry, Effects: map[string]author.Effect{
		"counter": {Capability: receiptquery.CounterCapability, Input: receiptquery.CounterInput(), Output: receiptquery.CounterOutput()},
	}, Steps: []author.Step{
		author.Receive("caller", nil),
		author.Request("original", "counter", map[string]author.Expr{"delta": author.Integer(1)}, nil),
		author.Await("counter", "original", map[string]string{"total": "total", "valid": "valid"}),
		author.Send("caller", nil), author.Success(),
	}})
	frame := template(author.Definition{Ports: entry, Invocations: map[string]author.Template{"leaf": leaf}, Steps: []author.Step{
		author.Receive("caller", nil), author.Invoke("inner", "leaf", nil, nil), author.Failure(),
	}})
	root, err := author.Compile(author.Definition{Invocations: map[string]author.Template{"frame": frame}, Steps: []author.Step{
		author.Invoke("outer", "frame", nil, nil),
		author.AwaitInvocation("frame", "outer", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present"}),
		author.Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: root, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	id := rt.RunID{Subject: "root", Generation: 1}
	p.Composition.Roots = []rt.ResourceRoot{{Run: id, Members: []rt.RunID{id}, Templates: []string{frame.ID(), leaf.ID()}, Capabilities: []string{receiptquery.CounterCapability}, Limits: rt.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 2, MaxEffects: 1, MaxInFlight: 1}}}
	return p
}

func TestOwningScopeQuiescenceWaitsForDescendant(t *testing.T) {
	h := newHistory2c(t)
	endpoint, ready := h.endpoint("", nil)
	s, err := rt.Create(t.Context(), h.db, nestedCounter(t), rt.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	root := rt.RunID{Subject: "root", Generation: 1}
	frame := invokeEntry(t, s, h.db, publicStep(t, s, root, "").Projection.Data["outer"].Text)
	leaf := invokeEntry(t, s, h.db, publicStep(t, s, frame, "").Projection.Data["inner"].Text)
	original := publicStep(t, s, leaf, "").Projection.Data["original"].Text
	held := make(chan boundary.Event, 1)
	release := make(chan struct{})
	ctx, cancelWork := context.WithCancel(t.Context())
	defer cancelWork()
	done := runReaders(t, s, ready, 1, func(runCtx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.admitted" && e.Intent == original {
			held <- e
			select {
			case <-release:
			case <-ctx.Done():
			case <-runCtx.Done():
			}
		}
		return runCtx.Err()
	})
	if cut := recvReadBoundary(t, held, done); cut.Subject != leaf.Subject {
		t.Fatal("wrong descendant held", cut)
	}
	if state := untilTerminal(t, s, frame); state.Projection.Status != "failed" {
		t.Fatal(state)
	}
	direct := rt.DispatchAuthority{Namespace: s.Namespace(), Run: frame}
	if err = s.AwaitDispatchQuiescence(t.Context(), direct); err != nil {
		t.Fatal("frame has no direct worker", err)
	}
	scope := direct
	scope.OwningScope = true
	wait, cancelWait := context.WithCancel(t.Context())
	waiting := false
	err = s.AwaitDispatchQuiescence(boundary.WithObserver(wait, func(_ context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.waiting" {
			waiting = true
			cancelWait()
		}
		return wait.Err()
	}), scope)
	cancelWait()
	if !waiting || !errors.Is(err, context.Canceled) {
		t.Fatal("owning scope skipped its admitted descendant", err)
	}
	h.expectWorld(0, 0, 0)
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = s.AwaitDispatchQuiescence(t.Context(), scope); err != nil {
		t.Fatal("descendant did not drain", err)
	}
	state, err := s.Snapshot(t.Context(), leaf)
	if err != nil || state.Projection.Status != "succeeded" || !state.Projection.Data["valid"].Flag ||
		state.Projection.Data["total"].Number != 1 || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", original) != 1 {
		t.Fatal("descendant's result did not settle", state, err)
	}
	h.expectWorld(1, 1, 0)
	h.replay("drained descendant")
	phaseStopEndpoint(t, endpoint)
}
