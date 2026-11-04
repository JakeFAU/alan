package delegation_test

import (
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
)

func TestBaseline(t *testing.T) {
	for _, scenario := range []delegation.BaselineScenario{delegation.BaselineLocal, delegation.BaselineMissing, delegation.BaselineClaim, delegation.BaselineCartesian, delegation.BaselineResultLaunder} {
		t.Run(string(scenario), func(t *testing.T) {
			p, err := delegation.Baseline(scenario)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Subjects) != 2 || len(p.Runs) != 2 || len(p.Programs) != 2 || p.Composition != nil {
				t.Fatal("expected exactly two ordinary static roots")
			}
			var grants []runtime.Permission
			for _, permission := range p.Permissions {
				if permission.Kind == "adapter" {
					grants = append(grants, permission)
				}
			}
			want := []runtime.Permission{
				{Kind: "adapter", Source: "a", Target: delegation.BaselineCounterARed, Operation: "increment", Revision: 1, Obligation: "increment"},
				{Kind: "adapter", Source: "b", Target: delegation.BaselineCounterBBlue, Operation: "increment", Revision: 1, Obligation: "increment"},
			}
			if !reflect.DeepEqual(grants, want) {
				t.Fatalf("principal-local complete tuples: got %+v want %+v", grants, want)
			}
			again, err := delegation.Baseline(scenario)
			if err != nil || !reflect.DeepEqual(p, again) {
				t.Fatal("construction not deterministic", err)
			}
		})
	}
	if _, err := delegation.Baseline("unknown"); err == nil {
		t.Fatal("unknown scenario accepted")
	}
}
