package owningtree

import (
	"fmt"
	"math"
	"math/bits"
	"slices"
)

// Fixed-width only for the declared campaign. Every overflow is a checker error.
type mask [4]uint64

func singleton(i int) (m mask) { m[i/64] = 1 << uint(i%64); return m }
func (a mask) union(b mask) mask {
	for i := range a {
		a[i] |= b[i]
	}
	return a
}
func (a mask) minus(b mask) mask {
	for i := range a {
		a[i] &^= b[i]
	}
	return a
}
func (a mask) intersect(b mask) mask {
	for i := range a {
		a[i] &= b[i]
	}
	return a
}
func (a mask) size() int {
	return bits.OnesCount64(a[0]) + bits.OnesCount64(a[1]) + bits.OnesCount64(a[2]) + bits.OnesCount64(a[3])
}
func (a mask) first() int {
	for i, v := range a {
		if v != 0 {
			return 64*i + bits.TrailingZeros64(v)
		}
	}
	return -1
}
func (a mask) each(f func(int)) {
	for i, v := range a {
		for v != 0 {
			f(64*i + bits.TrailingZeros64(v))
			v &= v - 1
		}
	}
}

type poset struct {
	names         []string
	full          mask
	before, after []mask // strict transitive closure
}

func compile(g graph) (poset, error) {
	p := poset{}
	if len(g) > 256 {
		return p, fmt.Errorf("graph representation limit: %d", len(g))
	}
	for id := range g {
		p.names = append(p.names, id)
	}
	slices.Sort(p.names)
	p.before = make([]mask, len(g))
	p.after = make([]mask, len(g))
	index := map[string]int{}
	for i, id := range p.names {
		index[id] = i
		p.full = p.full.union(singleton(i))
	}
	visited := make([]byte, len(g))
	var fill func(int) error
	fill = func(i int) error {
		switch visited[i] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("cycle at %s", p.names[i])
		}
		e := g[p.names[i]]
		if e.ID != p.names[i] {
			return fmt.Errorf("wrong target bit for %s: %s", p.names[i], e.ID)
		}
		visited[i] = 1
		for _, id := range e.After {
			j, ok := index[id]
			if !ok {
				return fmt.Errorf("%s requires nonexistent %s", e.ID, id)
			}
			if err := fill(j); err != nil {
				return err
			}
			p.before[i] = p.before[i].union(p.before[j]).union(singleton(j))
		}
		visited[i] = 2
		return nil
	}
	for i := range p.names {
		if err := fill(i); err != nil {
			return p, err
		}
		p.before[i].each(func(j int) { p.after[j] = p.after[j].union(singleton(i)) })
	}
	return p, nil
}

// Equality of every enable cone is equality of outgoing sets at every reachable
// ideal, including labels and destinations. This checks all alternatives before any
// counting/memoization. For unequal cones the smaller/missing principal cut witnesses it.
func audit(oracle, supplied graph) (poset, error) {
	want, err := compile(oracle)
	if err != nil {
		return want, err
	}
	got, err := compile(supplied)
	if err != nil {
		return got, err
	}
	if !slices.Equal(want.names, got.names) {
		return got, fmt.Errorf("different event alphabets")
	}
	for i, id := range want.names {
		w, g := oracle[id], supplied[id]
		if w.Owner != g.Owner || w.SnapshotOf != g.SnapshotOf || !slices.Equal(w.Facts, g.Facts) {
			return got, fmt.Errorf("%s: wrong owner or ordered word", id)
		}
		if want.before[i] != got.before[i] {
			return got, fmt.Errorf("%s: wrong enable cone", id)
		}
	}
	return got, nil
}

type totals struct{ States, Edges uint64 }

func plus(a, b uint64) uint64 {
	if a > math.MaxUint64-b {
		panic("census addition overflow")
	}
	return a + b
}
func times(a, b uint64) uint64 {
	if a != 0 && b > math.MaxUint64/a {
		panic("census multiplication overflow")
	}
	return a * b
}

// Count downsets. Split disconnected comparability components by Cartesian product.
// Otherwise partition by membership of a balanced pivot v. Between the two halves
// precisely I(P || v) edges add v (P || v is the induced incomparable subposet).
func (p poset) count() totals {
	cache := map[mask]totals{{}: {1, 0}}
	var visit func(mask) totals
	visit = func(set mask) totals {
		if old, ok := cache[set]; ok {
			return old
		}
		front := singleton(set.first())
		seen := front
		for front != (mask{}) {
			var adj mask
			front.each(func(i int) { adj = adj.union(p.before[i]).union(p.after[i]) })
			front = adj.intersect(set).minus(seen)
			seen = seen.union(front)
		}
		var result totals
		if rest := set.minus(seen); rest != (mask{}) {
			a, b := visit(seen), visit(rest)
			result = totals{times(a.States, b.States), plus(times(a.Edges, b.States), times(b.Edges, a.States))}
		} else {
			pivot, score := -1, -1
			set.each(func(i int) {
				balance := min(set.intersect(p.before[i]).size(), set.intersect(p.after[i]).size())
				if balance > score {
					score, pivot = balance, i
				}
			})
			other := set.minus(singleton(pivot))
			absent := other.minus(p.after[pivot])
			present := other.minus(p.before[pivot])
			a, b := visit(absent), visit(present)
			cross := visit(absent.intersect(present))
			result = totals{plus(a.States, b.States), plus(plus(a.Edges, b.Edges), cross.States)}
		}
		cache[set] = result
		return result
	}
	return visit(p.full)
}
