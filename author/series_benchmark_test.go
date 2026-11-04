package author_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

func BenchmarkBoundedSeries(b *testing.B) {
	for _, count := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			d, options := seriesDefinition(), seriesOptions(count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := author.BoundedSeries(d, options); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSeriesInspect(b *testing.B) {
	for _, count := range []int{1, 8, 32, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(count))
			if err != nil {
				b.Fatal(err)
			}
			p, err := s.Provision()
			if err != nil {
				b.Fatal(err)
			}
			path := filepath.Join(b.TempDir(), "series.db")
			owner, err := runtime.Create(b.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				b.Fatal(err)
			}
			if err = owner.Close(); err != nil {
				b.Fatal(err)
			}
			snapshot, err := runtime.InspectExecution(b.Context(), path, runtime.DefaultLimits())
			if err != nil {
				b.Fatal(err)
			}
			if _, err = s.Inspect(snapshot); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := s.Inspect(snapshot); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
