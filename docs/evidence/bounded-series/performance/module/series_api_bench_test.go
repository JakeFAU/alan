//go:build series

package performance_test

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"os"
	"path/filepath"
	"testing"
)

var seriesSink author.Series
var viewSink author.SeriesView

func series(n int) (author.Series, error) {
	return author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: n, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
}
func BenchmarkSeriesConstruction(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				s, e := series(n)
				if e != nil {
					b.Fatal(e)
				}
				p, e := s.Provision()
				if e != nil {
					b.Fatal(e)
				}
				seriesSink = s
				provisionSink = p
			}
		})
	}
}
func BenchmarkSeriesProvisionCopy(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			s, e := series(n)
			if e != nil {
				b.Fatal(e)
			}
			b.ReportAllocs()
			for b.Loop() {
				p, e := s.Provision()
				if e != nil {
					b.Fatal(e)
				}
				provisionSink = p
			}
		})
	}
}

// Stateful Create and Execute subbenchmarks must use -benchtime=1x.
func BenchmarkSeriesRuntime(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			decl, e := series(n)
			if e != nil {
				b.Fatal(e)
			}
			p, e := decl.Provision()
			if e != nil {
				b.Fatal(e)
			}
			path := filepath.Join(b.TempDir(), "store.sqlite")
			var s *aruntime.Store
			b.Run("Create", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if i != 0 {
						b.Fatal("use -benchtime=1x")
					}
					s, e = aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
					if e != nil {
						b.Fatal(e)
					}
				}
			})
			b.Run("Execute", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if i != 0 {
						b.Fatal("use -benchtime=1x")
					}
					execute(b, s, path, n)
				}
			})
			if e = s.Close(); e != nil {
				b.Fatal(e)
			}
			var cut *aruntime.ExecutionSnapshot
			b.Run("Inspect", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					cut, e = aruntime.InspectExecution(b.Context(), path, aruntime.DefaultLimits())
					if e != nil {
						b.Fatal(e)
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
			b.Run("SeriesView", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					v, e := decl.Inspect(cut)
					if e != nil {
						b.Fatal(e)
					}
					if v.Completed != n || v.Spent != n || v.Selected != n || !v.BoundExhausted {
						b.Fatalf("bad aggregate %+v", v)
					}
					viewSink = v
				}
			})
		})
	}
}
func TestNewSeriesFixture(t *testing.T) {
	dir := os.Getenv("SERIES_FIXTURE_DIR")
	if dir == "" {
		t.Skip("set SERIES_FIXTURE_DIR to retain a genuine new series store")
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		t.Fatal(e)
	}
	const n = 8
	decl, e := series(n)
	if e != nil {
		t.Fatal(e)
	}
	p, e := decl.Provision()
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.MarshalIndent(p, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "provision.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "series11.sqlite")
	s, e := aruntime.Create(t.Context(), path, p, aruntime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	m := &meter{}
	opts := aruntime.RunOptions{MaxSteps: 160, MaxDispatch: 32, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"meter": m}}
	for i := 1; i <= n; i++ {
		if i > 1 {
			cut, e := aruntime.InspectExecution(t.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			basis, e := cut.ContinuationBasis(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.ApproveContinuation(t.Context(), basis, "operator"); e != nil {
				t.Fatal(e)
			}
		}
		if _, e = s.Run(t.Context(), opts); e != nil {
			t.Fatal(e)
		}
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	cut, e := aruntime.InspectExecution(t.Context(), path, aruntime.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	v, e := decl.Inspect(cut)
	if e != nil {
		t.Fatal(e)
	}
	if v.Completed != n || v.Selected != n || v.Spent != n || m.calls != n || cut.StorageVersion != 11 {
		t.Fatalf("aggregate %+v calls=%d format=%d", v, m.calls, cut.StorageVersion)
	}
	t.Logf("new helper produces genuine format11; %d distinct effects/selected/completed/spent and %d controlled adapter entries; namespace=%s", n, m.calls, cut.Namespace)
}

func TestCanonicalLoweringEquality(t *testing.T) {
	for _, n := range []int{1, 8, 32, 256} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			d := definition()
			first, e := author.Compile(d)
			if e != nil {
				t.Fatal(e)
			}
			later := first
			if n > 1 {
				d.Ports = map[string]author.Port{"series-approval": author.ContinuationPort("series-approval", "operator")}
				d.Steps = append([]author.Step{author.Receive("series-approval", nil)}, d.Steps...)
				later, e = author.Compile(d)
				if e != nil {
					t.Fatal(e)
				}
			}
			app := author.Application{}
			profile := &aruntime.CompositionProvision{Version: "alan/composition/v1", RecordDispatchAdmissions: true}
			for i := 1; i <= n; i++ {
				id := aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1}
				a := first
				if i > 1 {
					a = later
				}
				app[id.Subject] = author.Instance{Artifact: a, Inputs: map[string]schema.Value{"station": schema.Text("north")}}
				profile.Roots = append(profile.Roots, aruntime.ResourceRoot{Run: id, Members: []aruntime.RunID{id}, Capabilities: []string{"meter"}, Limits: aruntime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
				if i > 1 {
					profile.Continuations = append(profile.Continuations, aruntime.ContinuationSlot{Predecessor: aruntime.RunID{Subject: fmt.Sprintf("series-%d", i-1), Generation: 1}, Target: id, Effect: "read", Obligation: "series-approval", ApprovalID: id.Subject + "-approval"})
				}
			}
			manual, e := app.Provision()
			if e != nil {
				t.Fatal(e)
			}
			manual.Composition = profile
			decl, e := series(n)
			if e != nil {
				t.Fatal(e)
			}
			actual, e := decl.Provision()
			if e != nil {
				t.Fatal(e)
			}
			a, e := json.Marshal(manual)
			if e != nil {
				t.Fatal(e)
			}
			b, e := json.Marshal(actual)
			if e != nil {
				t.Fatal(e)
			}
			if string(a) != string(b) {
				t.Fatalf("canonical manual and helper provision differ\nmanual=%s\nhelper=%s", a, b)
			}
			t.Logf("bound%d complete public Provision JSON byte-equal to independently authored canonical manual expansion", n)
		})
	}
}
func TestRetainedSeriesFormats(t *testing.T) {
	dir := os.Getenv("SERIES_FIXTURE_DIR")
	if dir == "" {
		t.Skip("fixture directory required")
	}
	decl, e := series(8)
	if e != nil {
		t.Fatal(e)
	}
	for _, version := range []int{11, 12} {
		path := filepath.Join(dir, fmt.Sprintf("series%d.sqlite", version))
		cut, e := aruntime.InspectExecution(t.Context(), path, aruntime.DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		view, e := decl.Inspect(cut)
		if e != nil {
			t.Fatal(e)
		}
		if view.Completed != 8 || view.Selected != 8 || view.Spent != 8 || !view.BoundExhausted || cut.StorageVersion != version {
			t.Fatalf("format=%d aggregate=%+v", version, view)
		}
		if version == 12 && view.Unresolved != 1 {
			t.Fatalf("conflict not reported independently: %+v", view)
		}
		t.Logf("format%d selected=%d completed=%d spent=%d pending=%d unresolved=%d", version, view.Selected, view.Completed, view.Spent, view.Pending, view.Unresolved)
	}
}

func BenchmarkSeriesViewProvisioned(b *testing.B) {
	for _, n := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			decl, e := series(n)
			if e != nil {
				b.Fatal(e)
			}
			p, e := decl.Provision()
			if e != nil {
				b.Fatal(e)
			}
			path := filepath.Join(b.TempDir(), "store.sqlite")
			s, e := aruntime.Create(b.Context(), path, p, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			if e = s.Close(); e != nil {
				b.Fatal(e)
			}
			cut, e := aruntime.InspectExecution(b.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				b.Fatal(e)
			}
			b.ReportAllocs()
			for b.Loop() {
				v, e := decl.Inspect(cut)
				if e != nil {
					b.Fatal(e)
				}
				viewSink = v
			}
		})
	}
}
func BenchmarkSeriesViewCompleted8(b *testing.B) {
	dir := os.Getenv("SERIES_FIXTURE_DIR")
	if dir == "" {
		b.Fatal("fixture directory required")
	}
	decl, e := series(8)
	if e != nil {
		b.Fatal(e)
	}
	cut, e := aruntime.InspectExecution(b.Context(), filepath.Join(dir, "series11.sqlite"), aruntime.DefaultLimits())
	if e != nil {
		b.Fatal(e)
	}
	b.ReportAllocs()
	for b.Loop() {
		v, e := decl.Inspect(cut)
		if e != nil {
			b.Fatal(e)
		}
		viewSink = v
	}
}
