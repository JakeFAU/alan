package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/reconciliation"
	"github.com/JakeFAU/alan/runtime"
	"os"
	"path/filepath"
	"testing"
)

// unknown is the cold construction control. Execute makes no network call and
// supplies no receipt. A genuine physical world is tested in the process suite.
type unknown struct{}

func (unknown) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "construction/no-evidence/v1", Wire: "{}", MaxResponseBytes: 100}, nil
}
func (unknown) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
}
func TestOrdinaryConstruction(t *testing.T) {
	dir := t.TempDir()
	if dst := os.Getenv("ALAN_PHASE2C_CONSTRUCTION"); dst != "" {
		dir = dst
	}
	path := filepath.Join(dir, "invoked.sqlite")
	p, err := reconciliation.New(reconciliation.Options{})
	if err != nil {
		t.Fatal("build", err)
	}
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal("create", err)
	}
	defer s.Close()
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: map[string]effects.Adapter{reconciliation.Counter: unknown{}}})
	if err != nil {
		t.Fatal("run", err)
	}
	parent, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "parent", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "resolver", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	inv := parent.Projection.Data["invocation"].Text
	a, err := s.AdmitInvocation(t.Context(), inv)
	if err != nil || !a.Accepted || len(a.Members) != 1 {
		t.Fatal(a, err)
	}
	src, err := s.Snapshot(t.Context(), a.Members[0])
	if err != nil {
		t.Fatal(err)
	}
	i, err := s.PrepareInvocation(t.Context(), inv)
	if err != nil {
		t.Fatal(err)
	}
	id := src.Projection.Data["original"].Text
	if id == "" || src.Projection.Data["echo"].Text != id || parent.Projection.Data["original"].Text != id || resolver.Projection.Data["received"].Text != id || parent.Projection.Data["resolver_ack"].Text != id {
		t.Fatal("handoff", src, parent, resolver)
	}
	if i.Ready || i.Pending != "external_outcome" || i.Freeze.Status != "succeeded" || len(i.Freeze.ExternalIntents) != 1 || i.Freeze.ExternalIntents[0] != id || len(i.Freeze.ReplyIntents) != 1 || len(i.Freeze.InvocationIntents) != 0 {
		t.Fatal("wrong pending state", i)
	}
	if parent.Projection.Status != "open" || resolver.Projection.Status != "open" {
		t.Fatal(parent, resolver)
	}
	if err = s.AwaitDispatchQuiescence(t.Context(), runtime.DispatchAuthority{Namespace: s.Namespace(), Run: src.ID}); err != nil {
		t.Fatal(err)
	}
	for _, r := range []runtime.RunID{parent.ID, resolver.ID} {
		if err = s.AwaitDispatchQuiescence(t.Context(), runtime.DispatchAuthority{Namespace: s.Namespace(), Run: r}); !errors.Is(err, runtime.ErrDispatchAuthorityOpen) {
			t.Fatal(r, err)
		}
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal("replay", err)
	}
	s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal("reopen", err)
	}
	defer s.Close()
	if err = s.AwaitDispatchQuiescence(t.Context(), runtime.DispatchAuthority{Namespace: s.Namespace(), Run: src.ID}); err != nil {
		t.Fatal("reopened Q", err)
	}
	t.Logf("original=%s source=%s parent=%d resolver=%d pending=%s", id, src.Projection.Status, parent.Sequence, resolver.Sequence, i.Pending)
	if dst := os.Getenv("ALAN_PHASE2C_CONSTRUCTION"); dst != "" {
		raw, err := json.MarshalIndent(map[string]any{"namespace": s.Namespace(), "source": src, "parent": parent, "resolver": resolver, "aggregate": i, "run": report, "replay": r}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dst, "invoked.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
