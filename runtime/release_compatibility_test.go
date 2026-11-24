package runtime_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// ALAN_RELEASE_STORES points at historical database images v03.sqlite through
// v16.sqlite and any matching WALs. Every owner receives its own writable copy.
func TestReleaseHistoricalStores(t *testing.T) {
	root := os.Getenv("ALAN_RELEASE_STORES")
	if root == "" {
		t.Skip("release qualification supplies the historical database matrix")
	}
	for v := 3; v <= 16; v++ {
		t.Run(fmt.Sprintf("v%02d", v), func(t *testing.T) {
			name := fmt.Sprintf("v%02d.sqlite", v)
			copy := filepath.Join(t.TempDir(), name)
			for _, suffix := range []string{"", "-wal"} {
				data, err := os.ReadFile(filepath.Join(root, name+suffix))
				if suffix != "" && errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(copy+suffix, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx := t.Context()
			l := runtime.DefaultLimits()
			read := func() runtime.ReplayReport {
				t.Helper()
				cut, err := runtime.InspectExecution(ctx, copy, l)
				if err != nil {
					t.Fatal(err)
				}
				r, err := runtime.Replay(ctx, copy, l)
				if err != nil {
					t.Fatal(err)
				}
				if cut.StorageVersion != v || cut.Namespace != r.Namespace || len(cut.Runs) != len(r.Runs) {
					t.Fatal("format, namespace or run count differs")
				}
				for _, state := range r.Runs {
					got, ok := cut.Run(state.ID)
					if !ok || !reflect.DeepEqual(got.RunState, state) {
						t.Fatalf("projection differs for %+v", state.ID)
					}
				}
				return r
			}
			before := read()
			s, err := runtime.Open(ctx, copy, l)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			after := read()
			if before.Namespace != after.Namespace || before.HistoryCount != after.HistoryCount ||
				before.IntentCount != after.IntentCount || len(before.Runs) != len(after.Runs) {
				t.Fatal("recovery changed recorded history")
			}
			for i, state := range before.Runs {
				got := after.Runs[i]
				if got.Epoch != state.Epoch && got.Epoch != state.Epoch+1 {
					t.Fatal("unexpected recovery epoch")
				}
				got.Epoch = state.Epoch
				if !reflect.DeepEqual(got, state) {
					t.Fatalf("recovery changed run %+v", state.ID)
				}
			}
			t.Logf("format %d, %d runs, %d commits, %d intents", v, len(after.Runs), after.HistoryCount, after.IntentCount)
		})
	}
}
