//go:build integration

package executor

import (
	"cmp"
	"slices"
	"testing"
)

// Materialize one deterministic shortest writer schedule for every reachable
// projection, then qualify both readers and a real owner on that cut.
func TestRecoveryAllReachableStores(t *testing.T) {
	reachable, _ := recoveryReachable()
	type example struct {
		cut      recoveryModelCut
		schedule string
	}
	examples := make([]example, 0, len(reachable))
	for c, schedule := range reachable {
		examples = append(examples, example{c, schedule})
	}
	slices.SortFunc(examples, func(a, b example) int {
		if a.cut.Scoped != b.cut.Scoped {
			if a.cut.Scoped {
				return 1
			}
			return -1
		}
		return cmp.Compare(a.schedule, b.schedule)
	})
	for _, ex := range examples {
		scope := "unscoped"
		if ex.cut.Scoped {
			scope = "scoped"
		}
		t.Run(scope+"/"+ex.schedule, func(t *testing.T) {
			f := recoveryLedger(t, ex.cut.Scoped, ex.schedule)
			f.qualify(t, reachable, "", true, &ex.cut)
		})
	}
	t.Logf("%d distinct reachable projections compared with three APIs", len(examples))
}
