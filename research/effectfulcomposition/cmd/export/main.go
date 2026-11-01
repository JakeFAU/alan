// Command export emits one complete finite graph per JSONL line.
package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	ec "github.com/JakeFAU/alan/research/effectfulcomposition"
	"log"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: export EMPTY_OUTPUT_DIRECTORY")
	}
	dst := os.Args[1]
	f, e := os.OpenFile(filepath.Join(dst, "graphs.jsonl.gz"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		log.Fatal(e)
	}
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	type count struct {
		Name                 string
		States, Edges, Sinks int
		Frontiers            map[string]int
	}
	counts := []count{}
	attacks := []ec.Counterexample{}
	ns, ne, nt := 0, 0, 0
	for _, p := range ec.Profiles() {
		g := ec.Explore(p)
		if e := ec.Check(g); e != nil {
			log.Fatal(e)
		}
		c := count{Name: p.Name, States: len(g.Rows), Frontiers: map[string]int{}}
		for _, r := range g.Rows {
			c.Edges += len(r.Edges)
			if len(r.Edges) == 0 {
				c.Sinks++
				c.Frontiers[r.Class]++
			}
		}
		counts = append(counts, c)
		ns += c.States
		ne += c.Edges
		nt += c.Sinks
		attacks = append(attacks, ec.Attacks(g)...)
		if e := enc.Encode(g); e != nil {
			log.Fatal(e)
		}
	}
	if e := gz.Close(); e != nil {
		log.Fatal(e)
	}
	if e := f.Close(); e != nil {
		log.Fatal(e)
	}
	for k, v := range map[string]any{"summary.json": counts, "reds.json": attacks} {
		b, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			log.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dst, k), append(b, '\n'), 0644); e != nil {
			log.Fatal(e)
		}
	}
	fmt.Printf("profiles=%d states=%d edges=%d sinks=%d adverse witnesses=%d\n", len(counts), ns, ne, nt, len(attacks))
}
