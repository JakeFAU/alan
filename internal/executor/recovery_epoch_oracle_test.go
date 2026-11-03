package executor

import (
	"math"
	"math/rand"
	"testing"
)

type oracleConstraint struct {
	from, to int
	delta    int64
	exact    bool
}

// Max-plus transitive closure, independent of the sparse equality-component
// implementation. These cases use small weights so int64 is exact.
func oracleFeasible(n int, constraints []oracleConstraint) bool {
	const unreachable = -1 << 48
	d := make([][]int64, n)
	for i := range d {
		d[i] = make([]int64, n)
		for j := range d[i] {
			d[i][j] = unreachable
		}
		d[i][i] = 0
	}
	for _, e := range constraints {
		d[e.from][e.to] = max(d[e.from][e.to], e.delta)
		if e.exact {
			d[e.to][e.from] = max(d[e.to][e.from], -e.delta)
		}
	}
	for k := range n {
		for i := range n {
			for j := range n {
				if d[i][k] != unreachable && d[k][j] != unreachable {
					d[i][j] = max(d[i][j], d[i][k]+d[k][j])
				}
			}
		}
	}
	for i := range n {
		if d[i][i] > 0 {
			return false
		}
	}
	return true
}

func TestIndependentReopenFeasibilityOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(502))
	accepted, rejected := 0, 0
	for trial := range 6000 {
		n := 1 + rng.Intn(8)
		before := make([][]int, n)
		var eq []reopenEquation
		var constraints []oracleConstraint
		known := make([]int64, n)
		for i := range n {
			known[i] = int64(rng.Intn(12))
		}
		for range rng.Intn(n*n + 3) {
			a, b := rng.Intn(n), rng.Intn(n)
			if rng.Intn(3) != 0 {
				if trial%3 == 0 && known[a] > known[b] {
					a, b = b, a
				}
				before[a] = append(before[a], b)
				constraints = append(constraints, oracleConstraint{a, b, 0, false})
			} else {
				d := int64(rng.Intn(9) - 4)
				if trial%3 == 0 {
					d = known[b] - known[a]
				}
				eq = append(eq, reopenEquation{a, b, d})
				constraints = append(constraints, oracleConstraint{a, b, d, true})
			}
		}
		want := oracleFeasible(n, constraints)
		if want {
			accepted++
		} else {
			rejected++
		}
		if got := feasibleReopens(before, eq); got != want {
			t.Fatalf("trial %d: got %t; want %t; before=%v equations=%v", trial, got, want, before, eq)
		}
	}
	if accepted < 100 || rejected < 100 {
		t.Fatal(accepted, rejected)
	}
	huge := []reopenEquation{{0, 1, math.MaxInt64}, {1, 2, math.MaxInt64}, {2, 3, math.MaxInt64}}
	if !feasibleReopens(make([][]int, 4), huge) || feasibleReopens([][]int{nil, nil, nil, {0}}, huge) {
		t.Fatal("global sums greater than MaxUint64")
	}
	t.Logf("6000 comparisons: %d feasible, %d infeasible; oversized global sum tested", accepted, rejected)
}
