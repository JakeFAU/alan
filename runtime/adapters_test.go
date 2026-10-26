//go:build integration

package runtime_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	runtime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type testAdapter struct {
	calls       atomic.Int64
	disposition string
}

func (a *testAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "controlled/v1", MaxResponseBytes: 4096}, nil
}
func (a *testAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	a.calls.Add(1)
	d := a.disposition
	if d == "" {
		d = "validated"
	}
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: d, Payload: map[string]schema.Value{"answer": schema.Text("retained")}}
}
func adapterProvision(t *testing.T) runtime.Provision {
	t.Helper()
	a, e := author.Compile(author.Definition{Name: "effect-test", Effects: map[string]author.Effect{"call": {Capability: "model", Input: schema.Record{"input": {Kind: "string", MaxBytes: 100}}, Output: schema.Record{"answer": {Kind: "string", MaxBytes: 100}}}}, Steps: []author.Step{author.Request("request", "call", map[string]author.Expr{"input": author.String("question")}, nil), author.Await("call", "request", map[string]string{"answer": "answer", "valid": "valid"}), author.IfEqual(author.Var("valid"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()})}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := (author.Application{"agent": {Artifact: a}}).Provision()
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestAdapterRunReopenReplay(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	a := &testAdapter{}
	report, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if e != nil {
		t.Fatal(e)
	}
	if len(report.OpenRuns) != 0 || a.calls.Load() != 1 {
		t.Fatalf("%+v calls=%d", report, a.calls.Load())
	}
	s.Close()
	s, e = runtime.Open(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20}); e != nil {
		t.Fatal(e)
	}
	s.Close()
	rr, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if rr.Runs[0].Projection.Status != "succeeded" {
		t.Fatal(rr)
	}
	inspection, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if len(inspection.Tables["adapter_attempts"]) != 1 || len(inspection.Tables["adapter_results"]) != 1 {
		t.Fatal("missing retained evidence")
	}
}
func TestMissingAdapterDoesNotSpendAttempt(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	_, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if e == nil {
		t.Fatal("missing registry not reported")
	}
	i, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	if len(i.Tables["adapter_attempts"]) != 0 {
		t.Fatal("missing config spent attempt")
	}
	a := &testAdapter{}
	_, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if e != nil || a.calls.Load() != 1 {
		t.Fatalf("recovery %v calls=%d", e, a.calls.Load())
	}
}
func TestUnknownAdapterNeverRepeats(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	a := &testAdapter{disposition: "unknown"}
	r, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.UnknownAdapters) != 1 {
		t.Fatalf("missing unknown %+v", r)
	}
	s.Close()
	s, e = runtime.Open(ctx, path, runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if e != nil || len(r.UnknownAdapters) != 1 || a.calls.Load() != 1 {
		t.Fatalf("repeat %+v %v calls %d", r, e, a.calls.Load())
	}
}
func TestKnownAdapterResultResumesWithoutRegistry(t *testing.T) {
	for _, stop := range []string{"adapter.result-recorded", "adapter.settled"} {
		t.Run(stop, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "store.db")
			s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			a := &testAdapter{}
			stopped := errors.New("observed boundary")
			ctx = boundary.WithObserver(ctx, func(_ context.Context, event boundary.Event) error {
				if event.Boundary == stop {
					if event.Intent == "" || event.Attempt == "" {
						t.Error("missing correlation")
					}
					return stopped
				}
				return nil
			})
			report, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
			if !errors.Is(e, stopped) || len(report.UnknownAdapters) != 0 {
				t.Fatalf("known result wrongly unknown %+v %v", report, e)
			}
			s.Close()
			s, e = runtime.Open(context.Background(), path, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			r, e := s.Run(context.Background(), runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20})
			if e != nil || len(r.OpenRuns) != 0 || a.calls.Load() != 1 {
				t.Fatalf("resume %+v %v calls=%d", r, e, a.calls.Load())
			}
			s.Close()
			if _, e = runtime.Replay(context.Background(), path, runtime.DefaultLimits()); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestInspectRejectsAdapterSettlementCorruption(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.db")
	s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": &testAdapter{}}})
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("UPDATE settlements SET descriptor='{}'"); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if _, e = runtime.Inspect(ctx, path, runtime.DefaultLimits()); e == nil {
		t.Fatal("corrupt adapter settlement inspected as valid")
	}
}
