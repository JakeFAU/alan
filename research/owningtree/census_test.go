package owningtree

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const archive = "../../docs/foundational/phase1t/evidence/model-census.json.gz"

type row struct {
	scenario
	totals
	Rank, Maxima, Charges, Depth int
	Serviced                     boundary
	GraphSHA256                  string
}

type census struct {
	Coverage string
	Cutoff   bool
	Version  string
	Cases    []row
	ByShape  map[string]totals
}

// Read the maximal cut from the submitted graph. Recover the stopped PC by the
// first unconsumed handle in local Await order, verifying delivery at this cut.
func inspectMaximum(root *member, g graph, p poset) (boundary, int, int, error) {
	var b boundary
	charges, depth := 0, 0
	if e, ok := g[key(root.role, "decide")]; ok {
		b.Decision = e.Facts[0]
		b.Product = g[key(root.children[0].role, "select")].Facts[2]
	}
	var walk func(*member) error
	walk = func(n *member) error {
		if n != root {
			if _, ok := g[key(n.role, "admit")]; !ok {
				return fmt.Errorf("%s never charged", n.role)
			}
			charges++
			depth = max(depth, strings.Count(n.role, "/"))
		}
		if _, published := g[key(n.role, "publish")]; published {
			at, _ := slices.BinarySearch(p.names, key(n.role, "publish"))
			for j, id := range p.names {
				if strings.HasPrefix(id, n.role+"/") && p.before[at].intersect(singleton(j)) == (mask{}) {
					return fmt.Errorf("%s published before %s settled", n.role, id)
				}
			}
		} else if _, ok := g[key(n.role, "unmatched")]; ok {
			b.Waits = append(b.Waits, n.role+" -> second-entry")
		} else {
			order := slices.Clone(n.children)
			if n.kind == "leaf" {
				slices.Reverse(order)
			}
			blocked := false
			for _, child := range order {
				_, delivered := g[key(child.role, "publish")]
				_, consumed := g[key(child.role, "select")]
				switch {
				case consumed && (!delivered || blocked):
					return fmt.Errorf("out-of-order selection %s", child.role)
				case !consumed && !blocked:
					if delivered {
						return fmt.Errorf("unserved selection %s", child.role)
					}
					b.Waits = append(b.Waits, n.role+" -> "+child.role)
					blocked = true
				case !consumed && delivered:
					b.DeliveredUnselected = append(b.DeliveredUnselected, child.role)
				}
			}
			if !blocked {
				return fmt.Errorf("nonterminal maximum at %s", n.role)
			}
		}
		for _, child := range n.children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	err := walk(root)
	slices.Sort(b.Waits)
	slices.Sort(b.DeliveredUnselected)
	return b, charges, depth, err
}

var modes = []string{"answer", "fail", "absent", "pending"}

func campaign(f func(scenario)) {
	for _, shape := range []string{"fork", "left-nested", "right-nested", "relay-above", "relay-left"} {
		n := 2
		if strings.HasSuffix(shape, "-nested") {
			n = 3
		}
		base := 4
		if shape == "fork" {
			base = 16
		}
		requestCount := 1
		for range n {
			requestCount *= base
		}
		for placement := 0; placement < 1<<n; placement++ {
			var assignment strings.Builder
			for i := range n {
				assignment.WriteByte("AB"[(placement>>(n-i-1))&1])
			}
			for vector := 0; vector < requestCount; vector++ {
				requests := make([]request, n)
				digits := vector
				for i := n - 1; i >= 0; i-- {
					d := digits % base
					digits /= base
					id, v := 1, d/4%2
					if base == 16 {
						id += d / 8
					} else {
						if i == 2 {
							id = 2
						}
						v = i % 2
					}
					requests[i] = request{id, v, modes[d%4]}
				}
				f(scenario{shape, assignment.String(), requests})
			}
		}
	}
}

func TestFiniteTreeCensus(t *testing.T) {
	c := census{Coverage: "all reachable downsets and all productive covers of each declared case", Version: "1T-open-ideal-v1", ByShape: map[string]totals{}}
	countCache := map[string]totals{}
	graphPath := strings.Replace(archive, "census.json", "graphs.jsonl", 1)
	var refs *bufio.Scanner
	var export *gzip.Writer
	if os.Getenv("ALAN_WRITE_TREE_CENSUS") == "1" {
		f, err := os.Create(graphPath)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		export = gzip.NewWriter(f)
	} else {
		f, err := os.Open(graphPath)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		z, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		defer z.Close()
		refs = bufio.NewScanner(z)
	}
	campaign(func(s scenario) {
		root := s.root()
		want, expectedFrontier := expected(root) // Freeze obligations before reading the submission.
		g := submitted(root)
		p, err := audit(want, g)
		if err != nil {
			t.Fatalf("%+v: %v", s, err)
		}
		got, charges, depth, err := inspectMaximum(root, g, p)
		if err != nil || !reflect.DeepEqual(expectedFrontier, got) {
			t.Fatalf("maximum %+v: got %+v want %+v: %v", s, got, expectedFrontier, err)
		}

		// Reuse counts only on byte-identical strict-order matrices (including their length).
		matrix := make([]byte, 0, 32*len(p.before))
		for _, m := range p.before {
			for _, v := range m {
				for range 8 {
					matrix = append(matrix, byte(v))
					v >>= 8
				}
			}
		}
		mkey := string(matrix)
		count, ok := countCache[mkey]
		if !ok {
			count = p.count()
			countCache[mkey] = count
		}
		raw, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		if export != nil {
			if _, err := export.Write(append(raw, '\n')); err != nil {
				t.Fatal(err)
			}
		} else if !refs.Scan() || !bytes.Equal(refs.Bytes(), raw) {
			t.Fatalf("graph archive differs for %+v (%v)", s, refs.Err())
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(raw))
		c.Cases = append(c.Cases, row{s, count, len(g), 1, charges, depth, got, hash})
		total := c.ByShape[s.Shape]
		c.ByShape[s.Shape] = totals{plus(total.States, count.States), plus(total.Edges, count.Edges)}
	})
	if export != nil {
		if err := export.Close(); err != nil {
			t.Fatal(err)
		}
	} else if refs.Scan() || refs.Err() != nil {
		t.Fatalf("trailing graph or read error: %v", refs.Err())
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ALAN_WRITE_TREE_CENSUS") == "1" {
		var buf bytes.Buffer
		z := gzip.NewWriter(&buf)
		if _, err = z.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err = z.Close(); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(archive, buf.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	z, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	retained, err := io.ReadAll(z)
	if err != nil || !bytes.Equal(raw, retained) {
		t.Fatalf("census differs: %v", err)
	}
	t.Logf("%d cases %v", len(c.Cases), c.ByShape)
}
