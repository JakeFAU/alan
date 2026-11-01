// Package effectfulcomposition models the bounded two-original protocol. Its
// inputs are hypotheses, never executor records, constructor code or SQL rows.
package effectfulcomposition

import "fmt"

type Profile struct {
	Name, Slice, Case, Assignment, Shape string
	Modes                                [2]string
	IDs, Deltas, Queries                 [2]int
	Capacity                             int
}

func base(name, slice string) Profile {
	return Profile{Name: name, Slice: slice, Case: name, Assignment: "AB", Shape: "fork", Modes: [2]string{"answer", "answer"}, IDs: [2]int{1, 2}, Deltas: [2]int{1, 2}, Queries: [2]int{1, 1}, Capacity: 2}
}
func Profiles() []Profile {
	ps := []Profile{}
	for _, name := range []string{"order", "equal-order", "cap-one", "cap-one-granted"} {
		p := base(name, "launch")
		switch name {
		case "equal-order":
			p.Deltas[1] = 1
		case "cap-one", "cap-one-granted":
			p.Capacity = 1
		}
		ps = append(ps, p)
	}
	for _, name := range []string{"adopt-duplicate", "later-error", "direct-existing", "retained-failure", "read-only", "absent-refresh", "late-application", "loss-refresh", "exhaust", "import-kill", "selection-kill", "lost-found", "intact-found", "reverse-read", "competing", "invalid-role", "incompatible", "manual-ack", "fresh-ack", "slow-read"} {
		p := base(name, "read")
		switch name {
		case "adopt-duplicate", "later-error", "incompatible":
			p.Queries[0] = 2
		case "absent-refresh", "late-application":
			p.Queries[1] = 2
		case "slow-read":
			p.Capacity = 1
		}
		ps = append(ps, p)
	}
	modes := []string{"answer", "fail", "absent", "pending"}
	for _, a := range []string{"AA", "AB", "BA", "BB"} {
		for _, l := range modes {
			for _, r := range modes {
				for _, id := range []int{1, 2} {
					name := fmt.Sprintf("fork-%s-%s-%s-id%d", a, l, r, id)
					p := base(name, "tree")
					p.Assignment = a
					p.Modes = [2]string{l, r}
					p.IDs[1] = id
					ps = append(ps, p)
				}
			}
		}
	}
	for _, shape := range []string{"relay-fork", "fork-relay"} {
		for _, r := range []string{"answer", "fail", "pending"} {
			p := base(shape+"-"+r, "tree")
			p.Shape = shape
			p.Modes[1] = r
			ps = append(ps, p)
		}
	}
	return ps
}

type Leaf struct {
	ID, Value int
	Tag       string
	Answers   []int
}
type Pure struct {
	Leaves                          [2]Leaf
	Outcome, Frontier               string
	ActualSlots, CommonSlots, Depth int
}

func Tree(p Profile) Pure {
	t := Pure{ActualSlots: 9, CommonSlots: 11, Depth: 4, Outcome: "answer", Frontier: "complete"}
	if p.Shape != "fork" {
		t.ActualSlots++
		t.CommonSlots++
		t.Depth++
	}
	for i, mode := range p.Modes {
		if p.Assignment[i] == 'B' {
			t.ActualSlots++
		}
		l := Leaf{ID: p.IDs[i], Value: i, Answers: []int{}}
		switch mode {
		case "answer":
			l.Tag = "answer"
			l.Answers = []int{1 - i}
		case "fail":
			l.Tag = "candidate_failed"
		case "absent":
			l.Tag = "candidate_missing_reply"
		case "pending":
			l.Tag = "pending"
		}
		t.Leaves[i] = l
		if l.Tag != "answer" {
			t.Outcome = "diagnosed"
		}
	}
	switch {
	case p.Modes[0] == "pending":
		t.Outcome = "pending"
		t.Frontier = "await-left"
	case p.Modes[1] == "pending":
		t.Outcome = "pending"
		t.Frontier = "left-selected/await-right"
	}
	return t
}

// Spend is maintained separately for L, R, the pure tree and the competing
// reader X. Active roots are P (L/R mutations plus T), resolver L/R and X.
type Spend struct{ I, R, Mut, Read, Ord, Post, Get, Apply int }
type Receipt struct{ Role, Mutation, Total int } // Role is 1=L or 2=R; zero means no receipt.
type Snapshot struct {
	Target, Query, Ordinal, At int
	Receipt                    Receipt
}
type Query struct {
	Phase, Status, Adoption string
	Ord, Serial, At         int
	Receipt                 Receipt
}
type Side struct {
	ReportTarget                                                  int
	ReportProof                                                   Receipt
	Mut, Original, Report, Ack                                    string
	Closed, Q, Settled, Published, AckSelected, ResolverEnd, Root bool
	Query                                                         [2]Query
	Witness                                                       int // logical index (one-based); Role is fixed by owning resolver
	Proof                                                         Receipt
	Conflict                                                      bool
}
type State struct {
	Owner, Epoch, Losses  int // owner 1 live, 0 dead
	S                     [2]Side
	Extra                 Query
	World                 [2]Receipt
	Serial, Total         int
	Log                   [5]Snapshot
	Cost                  [4]Spend
	TreeStep              bool
	Joined                int
	Decision              string
	ParentEnd, ParentRoot bool
}

func apply(s *State, p Profile, i int) {
	s.Total += p.Deltas[i]
	n := s.Cost[0].Apply + s.Cost[1].Apply + 1
	s.World[i] = Receipt{Role: i + 1, Mutation: n, Total: s.Total}
	s.Cost[i].Apply++
}
func delayed(p Profile) bool {
	return p.Case == "absent-refresh" || p.Case == "late-application" || p.Case == "loss-refresh"
}
func Initial(p Profile) State {
	var s State
	s.Owner = 1
	s.Epoch = 1
	t := Tree(p)
	s.Cost[2] = Spend{I: t.ActualSlots - 2, R: t.ActualSlots - 2}
	s.TreeStep = p.Case != "slow-read"
	for i := range s.S {
		s.Cost[i] = Spend{I: 1, R: 1}
		s.S[i].Mut = "requested"
	}
	if p.Slice == "launch" {
		if p.Capacity == 1 {
			s.S[0].Mut = "admitted"
			s.Cost[0].Mut = 1
			if p.Case == "cap-one-granted" {
				s.S[1].Mut = "granted"
			}
		}
		return s
	}
	for i := range s.S {
		s.S[i].Mut = "handled"
		s.S[i].Closed = true
		s.S[i].Q = true
		s.Cost[i].Mut = 1
		s.Cost[i].Post = 1
	}
	order := [2]int{0, 1}
	if p.Case == "reverse-read" {
		order = [2]int{1, 0}
	}
	for _, i := range order {
		if !(i == 1 && delayed(p)) {
			apply(&s, p, i)
		}
	}
	if p.Case == "direct-existing" || p.Case == "retained-failure" {
		v := &s.S[0]
		v.Original = "direct"
		if p.Case == "retained-failure" {
			v.Original = "failure"
		}
		v.Settled = true
		v.Published = true
	}
	if p.Slice == "tree" {
		for i := range s.S {
			v := &s.S[i]
			s.Serial++
			v.Original = "imported"
			v.Settled = true
			v.Published = true
			v.Query[0] = Query{Phase: "selected", Ord: 1, Status: "found", Adoption: "adopted", Serial: s.Serial, At: 2, Receipt: s.World[i]}
			v.Witness = 1
			v.Proof = s.World[i]
			v.Report = "booked"
			v.ReportTarget = i + 1
			v.ReportProof = v.Proof
			s.Log[i] = Snapshot{Target: i + 1, Query: 1, Ordinal: 1, At: 2, Receipt: s.World[i]}
			s.Cost[i].Read = 1
			s.Cost[i].Ord = 1
			s.Cost[i].Get = 1
		}
	}
	return s
}
