// Command export writes the complete bounded transition relations. One JSON
// object per line: profile header, every numbered row, then the next header.
package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"

	m "github.com/JakeFAU/alan/research/effectprotocol"
)

type summary struct {
	Profile                        string
	States, Edges, MaximumOutgoing int
	Frontiers                      map[string]int
}

func run(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	out := gzip.NewWriter(f)
	enc := json.NewEncoder(out)
	var summaries []summary
	for _, p := range m.Profiles() {
		g := m.Explore(p)
		if err = m.Check(g); err != nil {
			return err
		}
		if err = enc.Encode(struct{ Profile m.Profile }{p}); err != nil {
			return err
		}
		s := summary{Profile: p.Name, States: len(g.Rows), Frontiers: map[string]int{}}
		for _, r := range g.Rows {
			if err = enc.Encode(r); err != nil {
				return err
			}
			s.Edges += len(r.Outgoing)
			s.MaximumOutgoing = max(s.MaximumOutgoing, len(r.Outgoing))
			if len(r.Outgoing) == 0 {
				s.Frontiers[m.Frontier(p, r.State)]++
			}
		}
		summaries = append(summaries, s)
	}
	if err = enc.Encode(map[string]string{"PairProfile": "unknown-plus-responsive-single-slot"}); err != nil {
		return err
	}
	rows := m.PeerGraph()
	s := summary{Profile: "unknown-plus-responsive-single-slot", States: len(rows), Frontiers: map[string]int{}}
	for _, r := range rows {
		if err = enc.Encode(r); err != nil {
			return err
		}
		s.Edges += len(r.Outgoing)
		s.MaximumOutgoing = max(s.MaximumOutgoing, len(r.Outgoing))
		if len(r.Outgoing) == 0 {
			s.Frontiers["unresolved-and-completed"]++
		}
	}
	summaries = append(summaries, s)
	if err = out.Close(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	return e.Encode(summaries)
}
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: export GRAPH.jsonl.gz")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
