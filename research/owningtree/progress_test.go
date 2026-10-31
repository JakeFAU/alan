package owningtree

import (
	"slices"
	"strings"
	"testing"
)

// A materialized worklist with direct predecessor guards, independent of count().
// This cross-check includes every reachable state of these two declared samples.
func TestTreeProgressEnumeration(t *testing.T) {
	for _, s := range []scenario{
		{"fork", "BB", []request{{1, 0, "answer"}, {1, 0, "answer"}}},
		{"fork", "AB", []request{{1, 1, "pending"}, {1, 1, "absent"}}},
	} {
		want, _ := expected(s.root())
		g := submitted(s.root())
		p, err := audit(want, g)
		if err != nil {
			t.Fatal(err)
		}
		guard := make([]mask, len(g))
		bit := make([]mask, len(g))
		for i, id := range p.names {
			bit[i] = singleton(i)
			for _, prev := range g[id].After {
				j, found := slices.BinarySearch(p.names, prev)
				if !found {
					t.Fatal(prev)
				}
				guard[i] = guard[i].union(singleton(j))
			}
		}
		var counts totals
		maxima := 0
		pre, post := map[string]bool{}, map[string]bool{}
		seen := map[mask]bool{{}: true}
		queue := []mask{{}}
		for len(queue) > 0 {
			ideal := queue[0]
			queue = queue[1:]
			counts.States++
			enabled := 0
			for i, id := range p.names {
				if ideal.intersect(bit[i]) != (mask{}) || guard[i].minus(ideal) != (mask{}) {
					continue
				}
				enabled++
				counts.Edges++
				child := ideal.union(bit[i])
				if child.size() != ideal.size()+1 {
					t.Fatal("productive stutter")
				}
				if strings.HasSuffix(id, ":reply_allow") {
					end, ok := slices.BinarySearch(p.names, key(g[id].SnapshotOf, "end"))
					if !ok {
						t.Fatal("no End")
					}
					if ideal.intersect(bit[end]) != (mask{}) {
						post[id] = true
					} else {
						pre[id] = true
					}
				}
				if !seen[child] {
					seen[child] = true
					queue = append(queue, child)
				}
			}
			if enabled == 0 {
				if ideal != p.full {
					t.Fatal("maximum before all enabled work settled")
				}
				maxima++
			}
		}
		if counted := p.count(); counted != counts || maxima != 1 {
			t.Fatalf("factored %+v worklist %+v maxima %d", counted, counts, maxima)
		}
		for id := range g {
			if strings.HasSuffix(id, ":reply_allow") && (!pre[id] || !post[id]) {
				t.Fatalf("%s lost a first-decision cut", id)
			}
		}
		t.Logf("%s %v: states=%d edges=%d maxima=%d", s.Assignment, s.Requests, counts.States, counts.Edges, maxima)
	}
}
