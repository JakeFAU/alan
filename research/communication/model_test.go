package communication

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/research/substitution"
)

func emit(t *testing.T, name string, v any) {
	t.Helper()
	if dir := os.Getenv("ALAN_COMMUNICATION_MODEL_EVIDENCE"); dir != "" {
		if !filepath.IsAbs(dir) {
			t.Fatal("evidence directory must be absolute")
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func TestReferenceCensus(t *testing.T) {
	r := Explore(PrimaryConfig(), substitution.Reference)
	if r.States < 1 || r.Successful != 2 || r.Pending != 2 || r.MaximalStates != 4 || r.Stuck != 0 || r.Failed != 0 {
		t.Fatalf("unexpected maximal-state census: %+v", r)
	}
	if r.RankFailures+r.DomainFailures+r.ResourceFailures+r.LifetimeFailures+r.ProtocolFailures+r.Cycles+r.Cutoffs != 0 {
		t.Fatalf("invariant failure: %+v", r)
	}
	if r.MaxRank > RankBound || r.MaximalExecutions == "" {
		t.Fatal("unbounded or incomplete enumeration")
	}
	if len(r.Maxima) != r.MaximalStates {
		t.Fatal("not all maximal states retained")
	}
	for _, s := range r.Maxima {
		if s.Join.Base.E != 0 || s.Join.Base.I != s.Join.Base.R || (s.Join.Base.I != 2 && s.Join.Base.I != 3) {
			t.Fatalf("unexpected resource debit: %+v", s)
		}
		if s.Join.Base.RootStatus != "succeeded" {
			t.Fatal("root did not reach actual local Success")
		}
		if s.Join.Aggregate == "" && (s.Join.Base.PeerStatus != "" || s.AckAllowed || s.AckDelivered || s.AckStage != 3) {
			t.Fatal("pending category not exact ACK refusal")
		}
	}
	t.Logf("K1 states=%d edges=%d maxima=%d success=%d pending=%d rank=%d..%d paths=%s", r.States, r.Transitions, r.MaximalStates, r.Successful, r.Pending, r.MinRank, r.MaxRank, r.MaximalExecutions)
	emit(t, "census-K1.json", r)
}
func eventIndex(events []join.Event, kind string) int {
	for i, e := range events {
		if e.Kind == kind {
			return i
		}
	}
	return -1
}
func TestMovedRaceAndPairedControl(t *testing.T) {
	r := Explore(PrimaryConfig(), substitution.Reference)
	found := map[string]bool{}
	for _, w := range r.Witnesses {
		if w.Code != "ack_source_closed_pending" && w.Code != "authorized_then_terminal_then_admission" {
			continue
		}
		found[w.Code] = true
		report, commit, term, auth, admit := eventIndex(w.Events, "report_consume"), eventIndex(w.Events, "ack_commit"), eventIndex(w.Events, "root_terminal"), eventIndex(w.Events, "ack_authorize"), eventIndex(w.Events, "ack_deliver")
		if report < 0 || commit <= report || term <= commit || auth <= commit || admit <= auth {
			t.Fatalf("invalid causal trace: %+v", w)
		}
		if w.Code == "ack_source_closed_pending" {
			if auth <= term || w.Events[auth].Reason != "source_closed" || w.Events[admit].Reason != "source_denied:source_closed" || eventIndex(w.Events, "ack_consume") >= 0 {
				t.Fatal("moved-race witness invalid")
			}
		} else if auth >= term || admit <= term || w.Final.Join.Aggregate != "succeeded" {
			t.Fatal("authorization-before-closure control invalid")
		}
	}
	if len(found) != 2 {
		t.Fatalf("missing paired witnesses: %v", found)
	}
}

func TestFirstAuthorizationLifetimeProduct(t *testing.T) {
	states := []LifetimeState{{}}
	seen := map[LifetimeState]bool{{}: true}
	maxima := map[LifetimeState]bool{}
	edges := 0
	rank := func(s LifetimeState) int {
		r := 0
		if s.SourceClosed {
			r++
		}
		if s.First != 0 {
			r++
		}
		if s.Admission != 0 {
			r++
		}
		return r
	}
	for i := 0; i < len(states); i++ {
		s := states[i]
		if s.Reauthorize() != s {
			t.Fatal("retry changed first disposition")
		}
		es := LifetimeSuccessors(s)
		if len(es) == 0 {
			maxima[s] = true
		}
		for _, n := range es {
			edges++
			if rank(n) <= rank(s) {
				t.Fatal("rank failure")
			}
			if s.First != 0 && n.First != s.First {
				t.Fatal("first authorization changed")
			}
			if n.Admission != 0 && n.Admission != n.First {
				t.Fatal("closure changed admission for retained disposition")
			}
			if !seen[n] {
				seen[n] = true
				states = append(states, n)
			}
		}
	}
	if len(maxima) != 2 || !maxima[LifetimeState{SourceClosed: true, First: 1, Admission: 1}] || !maxima[LifetimeState{SourceClosed: true, First: 2, Admission: 2}] {
		t.Fatalf("unexpected maximal outcomes: %v", maxima)
	}
	t.Logf("source-lifetime states=%d edges=%d maxima=%d", len(states), edges, len(maxima))
	emit(t, "source-lifetime.json", struct {
		States      []LifetimeState
		Transitions int
	}{states, edges})
}
