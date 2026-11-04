package performance_test

import (
	"bytes"
	"compress/gzip"
	aruntime "github.com/JakeFAU/alan/runtime"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFreshStorageFormats(t *testing.T) {
	for _, n := range []int{1, 8, 32, 256} {
		t.Run(stringName(n), func(t *testing.T) {
			p, e := manual(n)
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "store.sqlite")
			s, e := aruntime.Create(t.Context(), path, p, aruntime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			if e = s.Close(); e != nil {
				t.Fatal(e)
			}
			cut, e := aruntime.InspectExecution(t.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			want := 11
			if n == 1 {
				want = 9
			}
			if cut.StorageVersion != want {
				t.Fatalf("format=%d want=%d", cut.StorageVersion, want)
			}
			if _, e = aruntime.Replay(t.Context(), path, aruntime.DefaultLimits()); e != nil {
				t.Fatal(e)
			}
			reopened, e := aruntime.Open(t.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			if e = reopened.Close(); e != nil {
				t.Fatal(e)
			}
			t.Logf("count=%d storage=%d; create/inspect/replay/reopen valid; zero adapter contacts (registry never supplied)", n, cut.StorageVersion)
		})
	}
}
func stringName(n int) string {
	if n == 1 {
		return "1"
	}
	if n == 8 {
		return "8"
	}
	if n == 32 {
		return "32"
	}
	return "256"
}
func TestHistoricalArchivesRemainReadOnly(t *testing.T) {
	root := os.Getenv("ALAN_SOURCE")
	if root == "" {
		t.Fatal("ALAN_SOURCE must identify frozen source")
	}
	for _, tc := range []struct {
		name, file string
		version    int
	}{
		{"format4-provisioned", "runtime/testdata/lifecycle-recovery/provisioned.sqlite.gz", 4},
		{"format4-missing", "runtime/testdata/lifecycle-recovery/missing.sqlite.gz", 4},
		{"format4-present", "runtime/testdata/lifecycle-recovery/present.sqlite.gz", 4},
		{"archived-weather", "examples/weather-agent/testdata/live.sqlite.gz", 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, e := os.Open(filepath.Join(root, tc.file))
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			r, e := gzip.NewReader(f)
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			raw, e := io.ReadAll(r)
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "archive.sqlite")
			if e = os.WriteFile(path, raw, 0600); e != nil {
				t.Fatal(e)
			}
			cut, e := aruntime.InspectExecution(t.Context(), path, aruntime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			if cut.StorageVersion != tc.version {
				t.Fatalf("storage=%d want=%d", cut.StorageVersion, tc.version)
			}
			if _, e = aruntime.Replay(t.Context(), path, aruntime.DefaultLimits()); e != nil {
				t.Fatal(e)
			}
			after, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(raw, after) {
				t.Fatal("read APIs changed main database")
			}
			wal, e := os.ReadFile(path + "-wal")
			if e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
			if len(wal) != 0 {
				t.Fatal("read APIs wrote WAL")
			}
			if len(cut.Continuations) != 0 {
				t.Fatal("historical format acquired continuation")
			}
			t.Logf("format=%d retained_runs=%d; InspectExecution/Replay byte-preserving without provider registry", cut.StorageVersion, len(cut.Runs))
		})
	}
}
