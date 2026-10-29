package join

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/JakeFAU/alan/research/substitution"
	"os"
	"path/filepath"
	"testing"
)

func emit(t *testing.T, name string, v any) {
	t.Helper()
	if dir := os.Getenv("ALAN_JOIN_MODEL_EVIDENCE"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err = writer.Write(append(b, '\n')); err != nil {
			t.Fatal(err)
		}
		if err = writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name+".json.gz"), compressed.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func TestReferenceCensus(t *testing.T) { census(t, substitution.Reference) }
func TestCandidateCensus(t *testing.T) {
	for _, v := range []substitution.Variant{substitution.Local, substitution.Helper} {
		t.Run(string(v), func(t *testing.T) { census(t, v) })
	}
}
func census(t *testing.T, v substitution.Variant) {
	t.Helper()
	var results []Census
	for _, c := range Configurations() {
		r := Explore(c, v)
		results = append(results, r)
		if r.ResourceFailures != 0 || r.LifetimeFailures != 0 || r.ReportFailures != 0 || r.RankFailures != 0 || r.Cutoffs != 0 || r.Cycles != 0 || r.Stuck != 0 {
			t.Fatalf("invariant failure: %+v", r)
		}
		if r.Successful == 0 || r.Pending == 0 || r.Failed != 0 || r.DecisiveStates == 0 || r.EarlyReportStates == 0 || r.LateReportStates == 0 {
			t.Fatalf("missing decisive positive/negative schedule: %+v", r)
		}
		for _, w := range r.Witnesses {
			if failures := CheckTrace(Trace{Request: c.Request, Root: "root", Peer: "peer", Generation: 1, Obligation: Obligation, Events: w.Events}); len(failures) > 0 {
				t.Fatalf("%s: oracle rejected conforming witness: %+v", w.Code, failures)
			}
		}
		t.Logf("%s states=%d edges=%d maximal=%d success=%d pending=%d paths=%s", r.Name, r.States, r.Transitions, r.MaximalStates, r.Successful, r.Pending, r.MaximalExecutions)
	}
	emit(t, "census-"+string(v), results)
}
func TestOriginalFencePreserved(t *testing.T) {
	var results []Census
	for _, v := range []substitution.Variant{substitution.Reference, substitution.Local, substitution.Helper} {
		for _, c := range Configurations() {
			c.Context = C0
			r := Explore(c, v)
			results = append(results, r)
			if r.Failed == 0 || r.LifetimeFailures == 0 || r.ResourceFailures != 0 || r.RankFailures != 0 {
				t.Fatalf("C0 failure lost: %+v", r)
			}
		}
	}
	emit(t, "c0-comparison", results)
}
func TestConsumptionDoesNotTerminate(t *testing.T) {
	c := Configurations()[0]
	s := Initial(c)
	// Follow a root/candidate-only path. S remains an outstanding predeclared ticket.
	for !s.Base.Ops[entry].Consumed {
		found := false
		for _, e := range Successors(c, substitution.Reference, s) {
			if e.Event.Operation == "s" || e.Event.Kind == "delay_set" || e.Event.Kind == "choose_helper" {
				continue
			}
			s = e.Next
			found = true
			break
		}
		if !found {
			t.Fatal("candidate path stuck")
		}
	}
	if s.Base.RootStatus != "" || demand(s) == 0 {
		t.Fatal("consumption erased lifetime obligation")
	}
	for _, e := range Successors(c, substitution.Reference, s) {
		if e.Event.Kind == "root_terminal" {
			t.Fatal("root terminal enabled before joining")
		}
	}
}

func TestReportAuthorizationCounterexample(t *testing.T) {
	for _, v := range []substitution.Variant{substitution.Reference, substitution.Local, substitution.Helper} {
		r := Explore(Configurations()[0], v)
		found := false
		for _, w := range r.Witnesses {
			if w.Code != "report_source_closed_pending" {
				continue
			}
			found = true
			commit, terminal, authorize, denial := -1, -1, -1, -1
			for i, e := range w.Events {
				switch e.Kind {
				case "report_commit":
					commit = i
				case "peer_terminal":
					terminal = i
				case "report_authorize":
					if e.Reason == "source_closed" {
						authorize = i
					}
				case "report_deliver":
					if e.Reason == "source_denied:source_closed" {
						denial = i
					}
				}
			}
			if commit < 0 || terminal <= commit || authorize <= terminal || denial <= authorize {
				t.Fatalf("wrong causal counterexample: %+v", w)
			}
			if w.Final.Base.RootStatus != "" || w.Final.Aggregate != "" || w.Final.Base.Closed || w.Final.ReportDelivered {
				t.Fatal("pending report refusal misrepresented")
			}
		}
		if !found {
			t.Fatalf("%s missing fair source-closure counterexample", v)
		}
	}
}
