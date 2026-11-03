package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// Use the same benchmark file on the previous weather revision. The timed
// readers need no adapter. All setup contacts go to the loopback fixture.
func BenchmarkWeatherContinuation(b *testing.B) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, morningWeather)
	}))
	defer server.Close()
	command := func(args ...string) {
		b.Helper()
		if err := run(b.Context(), append(args, "--json"), io.Discard); err != nil {
			b.Fatal(err)
		}
	}
	limits := runtime.DefaultLimits()
	for _, size := range []string{"archived", "two", "eight"} {
		db := filepath.Join(b.TempDir(), "weather.sqlite")
		var beforeApproval []byte
		allowance := 2
		if size == "archived" {
			f, err := os.Open("testdata/live.sqlite.gz")
			if err != nil {
				b.Fatal(err)
			}
			r, err := gzip.NewReader(f)
			if err != nil {
				b.Fatal(err)
			}
			raw, err := io.ReadAll(r)
			if err != nil {
				b.Fatal(err)
			}
			if err = r.Close(); err != nil {
				b.Fatal(err)
			}
			if err = f.Close(); err != nil {
				b.Fatal(err)
			}
			if err = os.WriteFile(db, raw, 0600); err != nil {
				b.Fatal(err)
			}
		} else {
			if size == "eight" {
				allowance = 8
			}
			previous := contacts.Load()
			command(provisioning(db, server.URL+"/v1/forecast", allowance)...)
			for n := 1; n <= allowance; n++ {
				if n > 1 {
					command("refresh", "--read", strconv.Itoa(n), "--db", db)
				}
				command("resume", "--db", db)
				if n == 1 && size == "two" {
					var err error
					beforeApproval, err = os.ReadFile(db)
					if err != nil {
						b.Fatal(err)
					}
					if info, err := os.Stat(db + "-wal"); err == nil && info.Size() != 0 {
						b.Fatal("live WAL in benchmark seed")
					}
				}
			}
			if contacts.Load()-previous != int64(allowance) {
				b.Fatal("setup made an unexpected number of requests")
			}
		}
		seed, err := runtime.InspectExecution(b.Context(), db, limits)
		if err != nil {
			b.Fatal(err)
		}
		history, err := seed.History(0, 1)
		if err != nil {
			b.Fatal(err)
		}
		stat, err := os.Stat(db)
		if err != nil {
			b.Fatal(err)
		}
		workload := func(b *testing.B) {
			b.ReportMetric(float64(history.Total), "reductions/DB")
			b.ReportMetric(float64(stat.Size()), "bytes/DB")
		}
		b.Run("Inspect/"+size, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runtime.InspectExecution(b.Context(), db, limits); err != nil {
					b.Fatal(err)
				}
			}
			workload(b)
		})
		b.Run("Replay/"+size, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runtime.Replay(b.Context(), db, limits); err != nil {
					b.Fatal(err)
				}
			}
			workload(b)
		})
		if size != "two" {
			continue
		}
		b.Run("ApproveFirst/two", func(b *testing.B) {
			work := b.TempDir()
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				b.StopTimer()
				copyPath := filepath.Join(work, strconv.Itoa(i)+".sqlite")
				if err := os.WriteFile(copyPath, beforeApproval, 0600); err != nil {
					b.Fatal(err)
				}
				s, err := runtime.Open(b.Context(), copyPath, limits)
				if err != nil {
					b.Fatal(err)
				}
				cut, err := runtime.InspectExecution(b.Context(), copyPath, limits)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				err = approve(b.Context(), s, cut, 2, 2)
				b.StopTimer()
				if err != nil {
					b.Fatal(err)
				}
				if err = s.Close(); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
			}
		})
		b.Run("ApproveDuplicate/two", func(b *testing.B) {
			s, err := runtime.Open(b.Context(), db, limits)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() {
				if err := s.Close(); err != nil {
					b.Error(err)
				}
			})
			cut, err := runtime.InspectExecution(b.Context(), db, limits)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := approve(b.Context(), s, cut, 2, 2); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
