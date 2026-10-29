package substitution

import "testing"

func TestRootSuccessClosesParallelAdmissionCounterexample(t *testing.T) {
	c := Config{Shape: "parallel", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 2}
	if !Entitled(c) {
		t.Fatal("frozen initial entitlement unexpectedly removed")
	}
	for _, v := range []Variant{Reference, Local, Helper} {
		t.Run(string(v), func(t *testing.T) {
			graph := CandidateGraph(v)
			if v == Reference {
				graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
			}
			result := ExploreGraph(c, RootTerminalFence(graph))
			found := false
			for _, failure := range result.Counterexamples {
				if failure.Code == "completion" {
					found = true
					closed := false
					scope := false
					for _, e := range failure.Trace {
						if e.Kind == "close_root" {
							closed = true
						}
						if e.Kind == "admit" && e.Role == "s" && e.Reason == "scope_closed" {
							scope = true
						}
					}
					if closed || !scope {
						t.Fatalf("counterexample does not isolate ordinary root success: %+v", failure)
					}
				}
			}
			if !found {
				t.Fatal("expected Gc counterexample missing")
			}
			emit(t, "root-success-parallel-counterexample-"+string(v), result)
		})
	}
}

type extensionSummary struct {
	Variant                                                                       Variant
	Total                                                                         Census
	FailedConfigurations, FailedEntitled, UnaffectedEntitled, UnaffectedCompleted int
	FailedByShape                                                                 map[string]int
}

func TestRuntimeBoundaryFullCensus(t *testing.T) {
	for _, v := range []Variant{Reference, Local, Helper} {
		t.Run(string(v), func(t *testing.T) {
			graph := CandidateGraph(v)
			if v == Reference {
				graph = func(c Config, s State) []Edge { return Successors(c, Reference, s) }
			}
			extension := RuntimeBoundaryGraph(graph)
			summary := extensionSummary{Variant: v, FailedByShape: map[string]int{}}
			var records []Census
			for _, c := range Configurations() {
				r := ExploreGraph(c, extension)
				records = append(records, r)
				total := &summary.Total
				total.Configurations++
				total.Entitled += r.Entitled
				total.States += r.States
				total.Transitions += r.Transitions
				total.Revisits += r.Revisits
				total.Rejections += r.Rejections
				total.Terminal += r.Terminal
				total.Pending += r.Pending
				total.Stuck += r.Stuck
				total.RankFailures += r.RankFailures
				failed := len(r.Counterexamples) > 0
				if failed {
					summary.FailedConfigurations++
					summary.FailedByShape[c.Shape]++
					if Entitled(c) {
						summary.FailedEntitled++
					}
				}
				for _, f := range r.Counterexamples {
					if f.Code != "completion" {
						t.Fatalf("unexpected safety/structure failure %s %+v", r.Name, f)
					}
				}
				if c.Shape != "parallel" && Entitled(c) {
					summary.UnaffectedEntitled++
					if !failed {
						summary.UnaffectedCompleted++
					}
				}
			}
			if summary.Total.Entitled != 84 || summary.FailedEntitled != 8 || summary.UnaffectedCompleted != 76 || summary.Total.Stuck != 0 || summary.Total.RankFailures != 0 {
				t.Fatalf("unexpected bounded partition %+v", summary)
			}
			emit(t, "boundary-extension-census-"+string(v), records)
			emit(t, "boundary-extension-summary-"+string(v), summary)
			t.Logf("%+v", summary)
		})
	}
}
func TestRootCancellationDoesNotCancelStaticPeer(t *testing.T) {
	c := Config{Shape: "parallel", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 2, Closure: true}
	graph := RuntimeBoundaryGraph(func(c Config, s State) []Edge { return Successors(c, Reference, s) })
	var closed State
	found := false
	for _, e := range graph(c, Initial(c)) {
		if e.Event.Kind == "close_root" {
			closed = e.Next
			found = true
		}
	}
	if !found || closed.PeerStatus != "" || closed.PeerPC != uncreated {
		t.Fatalf("invented peer cancellation: %+v", closed)
	}
	var committed State
	found = false
	for _, e := range graph(c, closed) {
		if e.Event.Kind == "invoke_commit" && e.Event.Operation == "s" {
			committed = e.Next
			found = true
		}
	}
	if !found {
		t.Fatal("open static peer cannot execute its own Invoke")
	}
	denied := false
	for _, e := range graph(c, committed) {
		if e.Event.Kind == "admit" && e.Event.Operation == "s" {
			if e.Event.Status != "rejected" || e.Event.Reason != "scope_closed" || e.Event.I != 0 {
				t.Fatalf("bad closed scope denial %+v", e.Event)
			}
			denied = true
		}
	}
	if !denied {
		t.Fatal("missing scoped denial")
	}
	result := ExploreGraph(c, graph)
	if result.Stuck != 0 || result.RankFailures != 0 {
		t.Fatal(result)
	}
	emit(t, "boundary-cancellation-parallel", result)
}
