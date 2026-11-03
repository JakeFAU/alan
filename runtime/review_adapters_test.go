//go:build integration

package runtime_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	runtime "github.com/JakeFAU/alan/runtime"
	"path/filepath"
	"testing"
)

func TestReviewR3InspectRequiresReverseAdapterObservationLinks(t *testing.T) {
	for _, settled := range []bool{false, true} {
		t.Run(map[bool]string{false: "known_unsettled", true: "orphan_observation"}[settled], func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "store.db")
			s, e := runtime.Create(ctx, path, adapterProvision(t), runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			runctx := ctx
			stop := errors.New("post-result")
			if !settled {
				runctx = boundary.WithObserver(ctx, func(_ context.Context, event boundary.Event) error {
					if event.Boundary == "adapter.result-recorded" {
						return stop
					}
					return nil
				})
			}
			_, e = s.Run(runctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": &testAdapter{}}})
			if settled && e != nil || !settled && !errors.Is(e, stop) {
				t.Fatal(e)
			}
			s.Close()
			if settled {
				db, e := sql.Open("sqlite3", path)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = db.Exec("DELETE FROM settlements; DELETE FROM admissions WHERE domain='adapter'"); e != nil {
					t.Fatal(e)
				}
				db.Close()
			}
			snapshot, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if settled {
				if e == nil {
					t.Fatal("orphan authenticated observation accepted by Inspect")
				}
			} else {
				if e != nil || len(snapshot.Tables["adapter_results"]) != 1 || len(snapshot.Tables["observations"]) != 0 {
					t.Fatal("known-unsettled evidence invalid", e)
				}
				if _, e = runtime.Replay(ctx, path, runtime.DefaultLimits()); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}
