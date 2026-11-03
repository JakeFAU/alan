package executor

import (
	"math"
	"testing"
)

func TestReviewFiniteClockOracle(t *testing.T) {
	// Enumerate integer clocks independently of the implementation. There
	// are four vertices and every nonzero offset is at most 2, so any
	// feasible instance has a translated solution of span at most 6.
	const vertices = 4
	var state uint64 = 19327611
	draw := func(n int) int {
		state = state*6364136223846793005 + 1442695040888963407
		return int((state >> 33) % uint64(n))
	}
	possible := func(before [][]int, eq []reopenEquation) bool {
		times := [vertices + 1]int64{}
		check := func() bool {
			for i, nodes := range before {
				for _, j := range nodes {
					if times[i] > times[j] {
						return false
					}
				}
			}
			for _, e := range eq {
				if times[e.to]-times[e.from] != e.delta {
					return false
				}
			}
			return true
		}
		for a := int64(0); a <= 6; a++ {
			for b := int64(0); b <= 6; b++ {
				for c := int64(0); c <= 6; c++ {
					for d := int64(0); d <= 6; d++ {
						if min(a, b, c, d) != 0 {
							continue
						}
						times = [vertices + 1]int64{0, a, b, c, d}
						if check() {
							return true
						}
					}
				}
			}
		}
		return false
	}
	yes, no := 0, 0
	for sample := 0; sample < 4000; sample++ {
		perm := []int{1, 2, 3, 4}
		for i := 3; i > 0; i-- {
			j := draw(i + 1)
			perm[i], perm[j] = perm[j], perm[i]
		}
		before := make([][]int, vertices+1)
		for i := 0; i < vertices; i++ {
			for j := i + 1; j < vertices; j++ {
				if draw(3) == 0 {
					before[perm[i]] = append(before[perm[i]], perm[j])
				}
			}
		}
		eq := make([]reopenEquation, draw(6))
		for i := range eq {
			eq[i] = reopenEquation{from: 1 + draw(vertices), to: 1 + draw(vertices), delta: int64(draw(5) - 2)}
		}
		want := possible(before, eq)
		if got := feasibleReopens(before, eq); got != want {
			t.Fatalf("sample %d got %v want %v order=%v equalities=%v", sample, got, want, before, eq)
		}
		if want {
			yes++
		} else {
			no++
		}
	}
	if yes < 100 || no < 100 {
		t.Fatal(yes, no)
	}
	t.Logf("%d feasible, %d infeasible", yes, no)
}
func TestReviewFiniteClockWideDistances(t *testing.T) {
	const edge int64 = math.MaxInt64 - 1
	const end = 7
	before := make([][]int, 2*end+2)
	var eq []reopenEquation
	for i := 1; i < end; i++ {
		eq = append(eq, reopenEquation{from: i, to: i + 1, delta: edge})
	}
	// These components share no equation. The first clock can stay over
	// twice the uint64 range behind the second while retaining a valid cut.
	for i := end + 1; i < 2*end; i++ {
		eq = append(eq, reopenEquation{from: i, to: i + 1, delta: edge - 1})
	}
	before[end] = []int{end + 1}
	if !feasibleReopens(before, eq) {
		t.Fatal("rejected feasible large relative offset")
	}
	// Closing the chain in the inequality graph is an impossible clock.
	// The raw before graph is acyclic; this contradiction uses the equalities.
	before[2*end] = []int{1}
	if feasibleReopens(before, eq) {
		t.Fatal("overflow concealed positive cycle")
	}
}
