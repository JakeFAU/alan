package performance_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

var provisionSink aruntime.Provision
var cutSink *aruntime.ExecutionSnapshot
var replaySink aruntime.ReplayReport

func definition() author.Definition {
	return author.Definition{Name: "measurement", Inputs: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 64}}}}, Steps: []author.Step{
		author.Request("operation", "read", map[string]author.Expr{"station": author.Var("station")}, nil),
		author.Await("read", "operation", map[string]string{"sample": "sample"}), author.Success()}}
}

func manual(n int) (aruntime.Provision, error) {
	d := definition()
	first, err := author.Compile(d)
	if err != nil {
		return aruntime.Provision{}, err
	}
	d.Ports = map[string]author.Port{"consent": author.ContinuationPort("approve", "operator")}
	d.Steps = append([]author.Step{author.Receive("consent", nil)}, d.Steps...)
	later, err := author.Compile(d)
	if err != nil {
		return aruntime.Provision{}, err
	}
	app := author.Application{}
	composition := &aruntime.CompositionProvision{RecordDispatchAdmissions: true}
	for i := 1; i <= n; i++ {
		id := aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1}
		a := first
		if i > 1 {
			a = later
		}
		app[id.Subject] = author.Instance{Artifact: a, Inputs: map[string]schema.Value{"station": schema.Text("north")}}
		composition.Roots = append(composition.Roots, aruntime.ResourceRoot{Run: id, Members: []aruntime.RunID{id}, Capabilities: []string{"meter"}, Limits: aruntime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
		if i > 1 {
			composition.Continuations = append(composition.Continuations, aruntime.ContinuationSlot{Predecessor: aruntime.RunID{Subject: fmt.Sprintf("series-%d", i-1), Generation: 1}, Target: id, Effect: "read", Obligation: "approve", ApprovalID: fmt.Sprintf("approve-%d", i)})
		}
	}
	p, err := app.Provision()
	p.Composition = composition
	return p, err
}

type meter struct{ calls int }

func (m *meter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "measurement/v1", Endpoint: "deterministic-local-no-http", Wire: r.Payload["station"].Text, MaxResponseBytes: 256}, nil
}
func (m *meter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}}
}
func execute(b *testing.B, s *aruntime.Store, path string, n int) {
	b.Helper()
	m := &meter{}
	opts := aruntime.RunOptions{MaxSteps: 16*n + 32, MaxDispatch: n + 8, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"meter": m}}
	for i := 1; i <= n; i++ {
		if i > 1 {
			cut, e := aruntime.InspectExecution(b.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			basis, e := cut.ContinuationBasis(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
			if e != nil {
				b.Fatal(e)
			}
			if _, e = s.ApproveContinuation(b.Context(), basis, "operator"); e != nil {
				b.Fatal(e)
			}
		}
		if _, e := s.Run(b.Context(), opts); e != nil {
			b.Fatal(e)
		}
	}
	if m.calls != n {
		b.Fatalf("physical adapter entries=%d want=%d", m.calls, n)
	}
}
func BenchmarkManualConstruction(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				p, e := manual(n)
				if e != nil {
					b.Fatal(e)
				}
				provisionSink = p
			}
		})
	}
}
func BenchmarkCreate(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			p, e := manual(n)
			if e != nil {
				b.Fatal(e)
			}
			root := b.TempDir()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				path := filepath.Join(root, fmt.Sprintf("%d.sqlite", i))
				s, e := aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
				if e != nil {
					b.Fatal(e)
				}
				if e = s.Close(); e != nil {
					b.Fatal(e)
				}
			}
		})
	}
}
func BenchmarkExecute(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			p, e := manual(n)
			if e != nil {
				b.Fatal(e)
			}
			root := b.TempDir()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				path := filepath.Join(root, fmt.Sprintf("%d.sqlite", i))
				s, e := aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
				if e != nil {
					b.Fatal(e)
				}
				b.StartTimer()
				execute(b, s, path, n)
				b.StopTimer()
				if e = s.Close(); e != nil {
					b.Fatal(e)
				}
				b.StartTimer()
			}
		})
	}
}
func BenchmarkInspectReplay(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			p, e := manual(n)
			if e != nil {
				b.Fatal(e)
			}
			path := filepath.Join(b.TempDir(), "store.sqlite")
			s, e := aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			execute(b, s, path, n)
			if e = s.Close(); e != nil {
				b.Fatal(e)
			}
			b.Run("Inspect", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					cut, e := aruntime.InspectExecution(b.Context(), path, aruntime.DefaultLimits())
					if e != nil {
						b.Fatal(e)
					}
					if len(cut.Runs) != n {
						b.Fatalf("runs=%d", len(cut.Runs))
					}
					cutSink = cut
				}
			})
			b.Run("Replay", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					r, e := aruntime.Replay(b.Context(), path, aruntime.DefaultLimits())
					if e != nil {
						b.Fatal(e)
					}
					replaySink = r
				}
			})
		})
	}
}

// CompleteBound256 executes exactly once and reuses its completed store for read benchmarks.
func BenchmarkCompleteBound256(b *testing.B) {
	p, e := manual(256)
	if e != nil {
		b.Fatal(e)
	}
	path := filepath.Join(b.TempDir(), "store.sqlite")
	s, e := aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
	if e != nil {
		b.Fatal(e)
	}
	b.Run("Execute", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if i != 0 {
				b.Fatal("use -benchtime=1x for stateful complete execution")
			}
			execute(b, s, path, 256)
		}
	})
	if e = s.Close(); e != nil {
		b.Fatal(e)
	}
	b.Run("Inspect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cut, e := aruntime.InspectExecution(b.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			if len(cut.Runs) != 256 {
				b.Fatalf("runs=%d", len(cut.Runs))
			}
			cutSink = cut
		}
	})
	b.Run("Replay", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			r, e := aruntime.Replay(b.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			replaySink = r
		}
	})
}

// ManualLoopConstruction mirrors the historical weather author's compile-per-
// position pattern, using this controlled typed meter definition at each bound.
// ManualConstruction above is the stronger two-cached-artifact comparator.
func manualLoop(n int) (aruntime.Provision, error) {
	app := author.Application{}
	composition := &aruntime.CompositionProvision{RecordDispatchAdmissions: true}
	for i := 1; i <= n; i++ {
		d := definition()
		if i > 1 {
			d.Ports = map[string]author.Port{"consent": author.ContinuationPort("approve", "operator")}
			d.Steps = append([]author.Step{author.Receive("consent", nil)}, d.Steps...)
		}
		a, e := author.Compile(d)
		if e != nil {
			return aruntime.Provision{}, e
		}
		id := aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1}
		app[id.Subject] = author.Instance{Artifact: a, Inputs: map[string]schema.Value{"station": schema.Text("north")}}
		composition.Roots = append(composition.Roots, aruntime.ResourceRoot{Run: id, Members: []aruntime.RunID{id}, Capabilities: []string{"meter"}, Limits: aruntime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
		if i > 1 {
			composition.Continuations = append(composition.Continuations, aruntime.ContinuationSlot{Predecessor: aruntime.RunID{Subject: fmt.Sprintf("series-%d", i-1), Generation: 1}, Target: id, Effect: "read", Obligation: "approve", ApprovalID: fmt.Sprintf("approve-%d", i)})
		}
	}
	p, e := app.Provision()
	p.Composition = composition
	return p, e
}
func BenchmarkManualLoopConstruction(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				p, e := manualLoop(n)
				if e != nil {
					b.Fatal(e)
				}
				provisionSink = p
			}
		})
	}
}
