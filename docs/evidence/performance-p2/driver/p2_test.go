package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestHTTPProofRejectsJournalMutations(t *testing.T) {
	supportedMetrics()
	r := report{Bound: 8, Sample: "http-proof", Transport: frozenTransport("http", "127.0.0.1:0"), PrefixEnabled: true, Prefixes: make([]prefixRecord, 0, 8)}
	dir := t.TempDir()
	if err := run(context.Background(), &r, dir, 0); err != nil {
		t.Fatal(err)
	}
	decl, err := author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: 8, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
	if err != nil {
		t.Fatal(err)
	}
	cut, err := aruntime.InspectExecution(context.Background(), filepath.Join(dir, "store.sqlite"), aruntime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err := decl.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		change func(*[]contact)
	}{
		{"operation", func(c *[]contact) { (*c)[0].OperationID = "foreign" }},
		{"key", func(c *[]contact) { (*c)[0].Key = "foreign" }},
		{"capability", func(c *[]contact) { (*c)[0].Capability = "foreign" }},
		{"effect", func(c *[]contact) { (*c)[0].Effect = "foreign" }},
		{"protocol", func(c *[]contact) { (*c)[0].Protocol = "foreign" }},
		{"endpoint", func(c *[]contact) { (*c)[0].Endpoint = "http://127.0.0.1:1/foreign" }},
		{"prepared digest", func(c *[]contact) { (*c)[0].PreparedDigest = "foreign" }},
		{"prepared field unchanged digest", func(c *[]contact) { (*c)[0].Prepared.Request.Capability = "foreign" }},
		{"request bytes", func(c *[]contact) { (*c)[0].RequestWire += " " }},
		{"request digest", func(c *[]contact) { (*c)[0].RequestSHA256 = "foreign" }},
		{"response identity", func(c *[]contact) { (*c)[0].ResponseID = "foreign" }},
		{"response bytes", func(c *[]contact) { (*c)[0].ResponseWire += " " }},
		{"response digest", func(c *[]contact) { (*c)[0].ResponseSHA256 = "foreign" }},
		{"incomplete response", func(c *[]contact) { (*c)[0].ResponseComplete = false }},
		{"missing", func(c *[]contact) { *c = (*c)[1:] }},
		{"duplicate", func(c *[]contact) { *c = append(*c, (*c)[0]) }},
		{"reordered", func(c *[]contact) { (*c)[0], (*c)[1] = (*c)[1], (*c)[0] }},
	}
	for _, journal := range []string{"adapter", "server"} {
		for _, tt := range mutations {
			t.Run(journal+"/"+tt.name, func(t *testing.T) {
				m := &meter{mode: "http", endpoint: r.Transport.Endpoint, contacts: effects.Copy(r.Contacts), journal: serverJournal{contacts: effects.Copy(r.ServerContacts)}}
				if journal == "adapter" {
					tt.change(&m.contacts)
				} else {
					tt.change(&m.journal.contacts)
				}
				if _, err := verify(cut, view, m, 8); err == nil {
					t.Fatal("accepted changed journal")
				}
			})
		}
	}
	if err := r.verifyPrefixes(); err != nil {
		t.Fatal(err)
	}
}

func TestStableActionWireCarriesOriginalInEnvelope(t *testing.T) {
	m := &meter{mode: "http", endpoint: "http://127.0.0.1:1234/measure"}
	request := effects.Request{ID: "operation-a", Key: "key-a", Capability: "meter", Effect: "read", Payload: map[string]schema.Value{"station": schema.Text("north")}}
	a, err := m.Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	request.ID, request.Key = "operation-b", "key-b"
	b, err := m.Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	if a.Wire != b.Wire {
		t.Fatal("action wire changed, invalidating continuation scope")
	}
	a, b = effects.Freeze(a), effects.Freeze(b)
	raw, _ := json.Marshal(a)
	if !strings.Contains(string(raw), "operation-a") || !strings.Contains(string(raw), "key-a") || a.Digest == b.Digest {
		t.Fatal("envelope lost original identity")
	}
}

func TestPrefixTogglePreservesWorkload(t *testing.T) {
	supportedMetrics()
	for _, mode := range []string{"http", "direct"} {
		for _, enabled := range []bool{true, false} {
			r := report{Bound: 1, Sample: "toggle", Transport: frozenTransport(mode, "127.0.0.1:0"), PrefixEnabled: enabled, Prefixes: make([]prefixRecord, 0, 1)}
			if err := run(context.Background(), &r, t.TempDir(), 0); err != nil {
				t.Fatal(err)
			}
			if !r.InvariantsOK || len(r.Contacts) != 1 || r.Selected != 1 || r.Spent != 1 || r.Reductions != 3 || r.ProtectedApprovals != 0 {
				t.Fatal("toggle changed workload")
			}
			if enabled && len(r.Prefixes) != 1 || !enabled && len(r.Prefixes) != 0 {
				t.Fatal("toggle changed wrong records")
			}
		}
	}
}
func TestSnapshotArithmeticRejectsReset(t *testing.T) {
	a := resourceSnapshot{at: time.Unix(0, 100), alloc: 10, mallocs: 10, gc: 2, forced: 1, pause: 10, user: 10, system: 10, rss: 10, counters: map[string]uint64{"x": 10}, runtimeMetrics: map[string]float64{"cpu": 1}}
	b := a
	b.at = time.Unix(0, 200)
	mutations := []func(*resourceSnapshot){
		func(s *resourceSnapshot) { s.alloc = 9 }, func(s *resourceSnapshot) { s.mallocs = 9 }, func(s *resourceSnapshot) { s.gc = 1 }, func(s *resourceSnapshot) { s.forced = 0 },
		func(s *resourceSnapshot) { s.pause = 9 }, func(s *resourceSnapshot) { s.user = 9 }, func(s *resourceSnapshot) { s.system = 9 }, func(s *resourceSnapshot) { s.at = time.Unix(0, 0) },
		func(s *resourceSnapshot) { s.counters = map[string]uint64{"x": 9} }, func(s *resourceSnapshot) { s.counters = map[string]uint64{} }, func(s *resourceSnapshot) { s.runtimeMetrics = map[string]float64{"cpu": .5} },
	}
	for i, mutate := range mutations {
		c := b
		mutate(&c)
		if _, err := snapshotDelta(a, c); err == nil {
			t.Fatalf("accepted reset %d", i)
		}
	}
	c := b
	c.counters = map[string]uint64{"x": 10, "new": 2}
	d, err := snapshotDelta(a, c)
	if err != nil || d.Counters["new"] != 2 {
		t.Fatal("new diagnostic counter not zero-based", err)
	}
}
func TestPrefixProofRejectsMissingDuplicateReorderedAndArithmetic(t *testing.T) {
	zero := stage{RuntimeMetrics: map[string]float64{}, Counters: map[string]uint64{}, ProcessMaxRSS: 100}
	first := zero
	first.WallNS, first.TotalAlloc = 10, 10
	second := zero
	second.WallNS, second.TotalAlloc = 20, 20
	outer := zero
	outer.Name, outer.WallNS, outer.TotalAlloc = "execute_window", 25, 25
	v, err := newCutVerifier([]aruntime.RootCompletion{fixtureRoot(1, false), fixtureRoot(2, false)}, 2)
	if err != nil {
		t.Fatal(err)
	}
	firstReport := aruntime.RunReport{Steps: 3, Dispatches: 1, Quiescent: true, Roots: []aruntime.RootCompletion{fixtureRoot(1, true), fixtureRoot(2, false)}, OpenRuns: []aruntime.RunID{{Subject: "series-2", Generation: 1}}}
	cut1, err := v.verify(firstReport, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	secondReport := aruntime.RunReport{Steps: 4, Dispatches: 1, Quiescent: true, Roots: []aruntime.RootCompletion{fixtureRoot(1, true), fixtureRoot(2, true)}}
	cut2, err := v.verify(secondReport, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	base := report{RootMembership: v.membership, Bound: 2, Transport: frozenTransport("http", "127.0.0.1:0"), PrefixEnabled: true, Stages: []stage{outer}, Prefixes: []prefixRecord{
		{Position: 1, DeclarationBound: 2, RetainedPositions: 1, Contacts: 1, ServerContacts: 1, Cut: cut1, Direct: first, Cumulative: first},
		{Position: 2, DeclarationBound: 2, RetainedPositions: 2, Contacts: 2, ServerContacts: 2, Cut: cut2, Direct: first, Cumulative: second}}}
	if err := base.verifyPrefixes(); err != nil {
		t.Fatal(err)
	}
	tests := []func(*report){
		func(r *report) { r.Prefixes = r.Prefixes[:1] }, func(r *report) { r.Prefixes[1] = r.Prefixes[0] },
		func(r *report) { r.Prefixes[0], r.Prefixes[1] = r.Prefixes[1], r.Prefixes[0] },
		func(r *report) { r.Prefixes[1].Cumulative.TotalAlloc = 9 }, func(r *report) { r.Prefixes[1].Direct.WallNS = -1 },
		func(r *report) { r.Prefixes[1].Contacts = 3 }, func(r *report) { r.Prefixes[1].Cumulative.WallNS = 30 },
	}
	for i, mutate := range tests {
		r := effects.Copy(base)
		mutate(&r)
		if err := r.verifyPrefixes(); err == nil {
			t.Fatalf("accepted invalid prefixes %d", i)
		}
	}
}

func fixtureRoot(position int, complete bool) aruntime.RootCompletion {
	run := aruntime.RunID{Subject: fmt.Sprintf("series-%d", position), Generation: 1}
	root := aruntime.RootCompletion{RootID: fmt.Sprintf("root-%d", position), Run: run, LocalStatus: "open", Pending: "members", Limits: aruntime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}
	root.Charges.RootID = root.RootID
	if complete {
		root.LocalStatus, root.Pending = "succeeded", ""
		root.Charges.Effects = 1
		root.Outcome = &aruntime.InvocationOutcome{ID: fmt.Sprintf("outcome-%d", position), InvocationID: "root:" + root.RootID, OperationID: "root:" + root.RootID, Caller: run, Status: "succeeded", Reason: "member_succeeded", DecisionOrder: int64(position)}
	}
	return root
}
func TestReturnedRunReportRejectsUnprovedPrefix(t *testing.T) {
	initial := []aruntime.RootCompletion{fixtureRoot(1, false), fixtureRoot(2, false)}
	good := aruntime.RunReport{Steps: 3, Dispatches: 1, Quiescent: true, Roots: []aruntime.RootCompletion{fixtureRoot(1, true), fixtureRoot(2, false)}, OpenRuns: []aruntime.RunID{{Subject: "series-2", Generation: 1}}}
	mutations := []struct {
		name    string
		change  func(*aruntime.RunReport)
		ordinal int
	}{
		{"forged loop ordinal", func(*aruntime.RunReport) {}, 2},
		{"missing root", func(r *aruntime.RunReport) { r.Roots = r.Roots[:1] }, 1},
		{"duplicate root", func(r *aruntime.RunReport) { r.Roots[1] = r.Roots[0] }, 1},
		{"foreign root", func(r *aruntime.RunReport) { r.Roots[0].RootID = "foreign" }, 1},
		{"wrong generation", func(r *aruntime.RunReport) { r.Roots[0].Run.Generation = 2 }, 1},
		{"still open", func(r *aruntime.RunReport) { r.Roots[0].LocalStatus = "open" }, 1},
		{"pending root", func(r *aruntime.RunReport) { r.Roots[0].Pending = "members" }, 1},
		{"missing outcome", func(r *aruntime.RunReport) { r.Roots[0].Outcome = nil }, 1},
		{"missing debit", func(r *aruntime.RunReport) { r.Roots[0].Charges.Effects = 0 }, 1},
		{"changed result status", func(r *aruntime.RunReport) { r.Roots[0].Outcome.Status = "failed" }, 1},
		{"unjustified completed future", func(r *aruntime.RunReport) { r.Roots[1] = fixtureRoot(2, true) }, 1},
		{"duplicate open", func(r *aruntime.RunReport) { r.OpenRuns = append(r.OpenRuns, r.OpenRuns[0]) }, 1},
		{"wrong open", func(r *aruntime.RunReport) { r.OpenRuns[0].Generation = 2 }, 1},
		{"not quiescent", func(r *aruntime.RunReport) { r.Quiescent = false }, 1},
		{"wrong steps", func(r *aruntime.RunReport) { r.Steps++ }, 1},
		{"wrong dispatches", func(r *aruntime.RunReport) { r.Dispatches++ }, 1},
		{"uncertain adapter", func(r *aruntime.RunReport) { r.UnknownAdapters = []string{"unknown"} }, 1},
	}
	for _, tt := range mutations {
		t.Run(tt.name, func(t *testing.T) {
			v, err := newCutVerifier(initial, 2)
			if err != nil {
				t.Fatal(err)
			}
			r := effects.Copy(good)
			tt.change(&r)
			if _, err := v.verify(r, tt.ordinal, true); err == nil {
				t.Fatal("accepted unproved cut")
			}
		})
	}
	v, err := newCutVerifier(initial, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.verify(good, 1, true); err != nil {
		t.Fatal(err)
	}
	next := aruntime.RunReport{Steps: 4, Dispatches: 1, Quiescent: true, Roots: []aruntime.RootCompletion{fixtureRoot(1, true), fixtureRoot(2, true)}}
	next.Roots[0].Outcome.ID = "replacement-original"
	if _, err := v.verify(next, 2, true); err == nil {
		t.Fatal("accepted changed prior original outcome")
	}
}
func TestFailureReportCopiesJournalAfterHandlerDrain(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	entered, release, shutdown := make(chan struct{}), make(chan struct{}), make(chan struct{})
	m := &meter{mode: "http", transport: &http.Transport{}, serverDone: make(chan error, 1)}
	m.server = &http.Server{ConnState: m.trackConnection, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		m.journal.mu.Lock()
		m.journal.contacts = append(m.journal.contacts, contact{Ordinal: 1, OperationID: "accepted-during-drain", ResponseComplete: true})
		m.journal.mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
	})}
	m.server.RegisterOnShutdown(func() { close(shutdown) })
	go func() { m.serverDone <- m.server.Serve(listener) }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", "http://"+listener.Addr().String()+"/measure", nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: m.transport}
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		response, _ := client.Do(req)
		if response != nil {
			response.Body.Close()
		}
	}()
	<-entered
	cancel()
	<-clientDone
	original := errors.New("original Run failure")
	var r report
	finished := make(chan error, 1)
	go func() {
		finished <- func() (err error) { defer func() { err = errors.Join(err, m.finishReport(&r)) }(); return original }()
	}()
	<-shutdown
	// The handler has not populated its journal yet; Shutdown must drain it
	// before the actual deferred finishReport path copies failure evidence.
	close(release)
	if err := <-finished; !errors.Is(err, original) {
		t.Fatalf("lost original failure: %v", err)
	}
	if len(r.ServerContacts) != 1 || r.ServerContacts[0].OperationID != "accepted-during-drain" || !r.ServerContacts[0].ResponseComplete {
		t.Fatal("failure journal copied before handler drained")
	}
}
