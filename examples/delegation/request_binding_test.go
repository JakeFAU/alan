package delegation_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
)

func TestRequestBindingProvision(t *testing.T) {
	for _, steps := range []int{0, 1, 3, 8, 32} {
		t.Run(fmt.Sprint(steps), func(t *testing.T) {
			o := delegation.RequestBindingOptions{EndpointA: "http://127.0.0.1:1234", EndpointB: "http://127.0.0.1:1235", LocalSteps: steps, TerminalHandshake: true}
			p, m, err := delegation.RequestBinding(o)
			if err != nil {
				t.Fatal(err)
			}
			p2, m2, err := delegation.RequestBinding(o)
			if err != nil || !reflect.DeepEqual(p, p2) || !reflect.DeepEqual(m, m2) {
				t.Fatal("determinism", err)
			}
			if m.ServiceSequence != int64(4+steps) || m.ReplySequence != int64(7+steps) || len(m.ProgramIDs) != 2 {
				t.Fatal(m)
			}
			path := filepath.Join(t.TempDir(), "binding.sqlite")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, choice := range []bool{false, true} {
		p, m, err := delegation.RequestBinding(delegation.RequestBindingOptions{EndpointA: "http://127.0.0.1:1234", EndpointB: "http://127.0.0.1:1235", ForwardBranch: &choice})
		if err != nil || m.ServiceSequence != 7 {
			t.Fatal(m, err)
		}
		s, err := runtime.Create(t.Context(), filepath.Join(t.TempDir(), "branch.sqlite"), p, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
	}
	for _, n := range []int{-1, 33} {
		if _, _, err := delegation.RequestBinding(delegation.RequestBindingOptions{LocalSteps: n}); err == nil {
			t.Fatal("invalid length accepted")
		}
	}
}

func ExampleRequestBinding() {
	_, m, err := delegation.RequestBinding(delegation.RequestBindingOptions{EndpointA: "http://127.0.0.1:1234/apply", EndpointB: "http://127.0.0.1:1235/apply", LocalSteps: 3})
	if err != nil {
		panic(err)
	}
	fmt.Println(m.ServiceSequence, m.ReplySequence)
	fmt.Println(m.Reciprocal.BAService)
	// Output:
	// 7 10
	// ["a",1,7,0]
}
