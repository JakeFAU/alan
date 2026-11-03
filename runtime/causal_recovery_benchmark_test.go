package runtime_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	rt "github.com/JakeFAU/alan/runtime"
)

// ALAN_RECOVERY_BENCH_STORE selects an additional valid, checkpointed SQLite
// history. Each owner receives a new copy so every sample starts at the same epoch.
func BenchmarkCausalRecovery(b *testing.B) {
	for _, name := range []string{"missing", "present", "forged"} {
		b.Run(name, func(b *testing.B) {
			benchmarkRecoveryStore(b, recoveryCopy(b, name), name == "forged")
		})
	}
	if path := os.Getenv("ALAN_RECOVERY_BENCH_STORE"); path != "" {
		b.Run("supplied", func(b *testing.B) {
			if wal, err := os.Stat(path + "-wal"); err == nil && wal.Size() != 0 {
				b.Fatal("supplied store must be checkpointed")
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				b.Fatal(err)
			}
			benchmarkRecoveryStore(b, path, false)
		})
	}
}

func benchmarkRecoveryStore(b *testing.B, path string, corrupt bool) {
	limits := rt.DefaultLimits()
	image, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	for _, op := range []string{"InspectExecution", "Replay", "Open"} {
		b.Run(op, func(b *testing.B) {
			dir := b.TempDir()
			// Readers also use a private copy to leave the input's SQLite sidecars alone.
			p := filepath.Join(dir, "ledger.sqlite")
			if err := os.WriteFile(p, image, 0600); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				var err error
				switch op {
				case "InspectExecution":
					_, err = rt.InspectExecution(b.Context(), p, limits)
				case "Replay":
					_, err = rt.Replay(b.Context(), p, limits)
				case "Open":
					b.StopTimer()
					sampleDir, e := os.MkdirTemp(dir, "owner-")
					if e != nil {
						b.Fatal(e)
					}
					sample := filepath.Join(sampleDir, "ledger.sqlite")
					if e = os.WriteFile(sample, image, 0600); e != nil {
						b.Fatal(e)
					}
					b.StartTimer()
					var owner *rt.Store
					owner, err = rt.Open(b.Context(), sample, limits)
					b.StopTimer()
					if owner != nil {
						if e = owner.Close(); e != nil {
							b.Fatal(e)
						}
					}
					if e = os.RemoveAll(sampleDir); e != nil {
						b.Fatal(e)
					}
					b.StartTimer()
				}
				if corrupt && !errors.Is(err, rt.ErrCorrupt) || !corrupt && err != nil {
					b.Fatalf("%s: %v (expected corruption: %t)", op, err, corrupt)
				}
			}
		})
	}
}
