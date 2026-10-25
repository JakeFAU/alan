package semantics

import (
	"fmt"
	"strings"
)

type searchSpec struct {
	Name            string
	Initial         machine
	Events          []event
	Depth, StateCap int
	Policy          string
	WantViolation   string
	WantWitnesses   []string
	WantRejects     []string
}
type statistics struct {
	Name                                                        string
	Bounds                                                      bounds
	DepthLimit, StateCap                                        int
	Alphabet                                                    []string
	States, Transitions, Rejected, Revisits, Boundary, MaxDepth int
	Rejections                                                  map[string]int
	Witnesses                                                   map[string]bool
	Violation                                                   string
	Trace                                                       []string
}
type node struct {
	Machine machine
	Monitor monitor
	Trace   []string
}

func witness(before, after machine, e event, found map[string]bool) {
	found["event:"+string(e.Kind)] = true
	for a, s := range after.Subjects {
		if s.Current > 1 {
			found["new-generation"] = true
		}
		if s.Retired {
			found["retired"] = true
		}
		if r := after.Runs[runID(a, s.Current)]; r.E > 1 {
			found["reactivated"] = true
		}
	}
	for id, ad := range after.Admissions {
		if !ad.Ack && ad.Bound != runID(ad.Message.Target, after.Subjects[ad.Message.Target].Current) && e.Kind == admit && transportID(e.Origin, e.ID) == id {
			found["duplicate-after-retry-without-ack"] = true
		}
	}
	for id, in := range after.Intents {
		applications := 0
		for _, result := range after.World.Applied {
			if result.Intent == id {
				applications++
			}
		}
		if applications > 1 {
			found["multiple-attempts-applied"] = true
		}
		if after.hasAttempt(id) && !in.Settled && applications > 0 {
			found["applied-unresolved"] = true
		}
	}
	for id, rc := range after.Receipts {
		if _, already := before.Receipts[id]; already {
			continue
		}
		in := before.Intents[rc.Intent]
		if before.Runs[runID(in.Descriptor.A, in.Descriptor.G)].Projection.Status != "active" {
			found["late-settlement"] = true
		}
	}
	for id := range after.Delivered {
		r := after.Runs[id]
		if e.Kind == deliver && e.A == r.A && e.G == r.G && before.Subjects[r.A].Current > r.G {
			found["old-notification-after-retry"] = true
		}
	}
	if after.World.Mutations > 1 {
		found["repeated-mutation"] = true
	}
	if e.Kind == interrupt && canonical(before.Runs) == canonical(after.Runs) {
		found["interruption-preserves-journal"] = true
	}
}

// BFS checks every accepted successor before deduplication. Machine, frozen
// program, full histories, external world, proposals and monitor facts all enter
// identity. The first violation is therefore a shortest trace in this alphabet.
func explore(spec searchSpec) (statistics, error) {
	st := statistics{Name: spec.Name, Bounds: spec.Initial.Config.Bounds, DepthLimit: spec.Depth, StateCap: spec.StateCap, Rejections: map[string]int{}, Witnesses: map[string]bool{}}
	for _, e := range spec.Events {
		st.Alphabet = append(st.Alphabet, e.String())
	}
	mon := freshMonitor()
	mon.Policy = spec.Policy
	initial := node{Machine: copyOf(spec.Initial), Monitor: mon}
	queue := []node{initial}
	key := func(n node) string {
		return canonical(struct {
			Machine machine
			Monitor monitor
		}{n.Machine, n.Monitor})
	}
	seen := map[string]bool{key(initial): true}
	st.States = 1
	for head := 0; head < len(queue); head++ {
		n := queue[head]
		depth := len(n.Trace)
		if depth > st.MaxDepth {
			st.MaxDepth = depth
		}
		if depth == spec.Depth {
			st.Boundary++
			continue
		}
		for _, e := range spec.Events {
			next, why := step(n.Machine, e)
			if why != "" {
				st.Rejected++
				st.Rejections[why]++
				continue
			}
			st.Transitions++
			monitor, violation := inspect(n.Machine, next, e, n.Monitor)
			trace := append(append([]string{}, n.Trace...), e.String())
			if violation != "" {
				st.Violation = violation
				st.Trace = trace
				if len(trace) > st.MaxDepth {
					st.MaxDepth = len(trace)
				}
				return st, nil
			}
			witness(n.Machine, next, e, st.Witnesses)
			candidate := node{Machine: next, Monitor: monitor, Trace: trace}
			identity := key(candidate)
			if seen[identity] {
				st.Revisits++
				continue
			}
			if len(seen) >= spec.StateCap {
				return st, fmt.Errorf("hard state cap %d exhausted in %s", spec.StateCap, spec.Name)
			}
			seen[identity] = true
			st.States++
			if len(trace) > st.MaxDepth {
				st.MaxDepth = len(trace)
			}
			queue = append(queue, candidate)
		}
	}
	return st, nil
}
func expectation(st statistics, spec searchSpec) error {
	if st.Violation != spec.WantViolation {
		return fmt.Errorf("violation=%q want=%q", st.Violation, spec.WantViolation)
	}
	if st.Transitions == 0 || st.Rejected == 0 {
		return fmt.Errorf("vacuous exploration")
	}
	for _, w := range spec.WantWitnesses {
		if !st.Witnesses[w] {
			return fmt.Errorf("missing witness %q", w)
		}
	}
	for _, want := range spec.WantRejects {
		found := false
		for why, count := range st.Rejections {
			if strings.Contains(why, want) && count > 0 {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("missing rejection %q", want)
		}
	}
	return nil
}
