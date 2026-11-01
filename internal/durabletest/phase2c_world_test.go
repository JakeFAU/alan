//go:build integration

package durabletest

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	x "github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Raw external operations belong to the lower-level language. This source
// sends the ID emitted by its real operation. Its resolver is an authored
// program that still has to receive, request and select the outcome.
func (h *history2c) seedRaw(profile string) {
	t := h.t
	t.Helper()
	orig := runtime.RunID{Subject: "source", Generation: 1}
	p := runtime.Provision{}
	value := func(v string) x.Operand { return x.Lit(x.Text(v)) }
	one := x.Lit(x.Int(1))
	addProgram(t, &p, "source", []x.Instruction{
		{Op: "emit", Dst: "original", Effect: &x.Effect{Kind: "external", Target: value(h.apply), Generation: one, Obligation: value("mutation"), Payload: map[string]x.Operand{"delta": one, "profile": value(profile)}}},
		{Op: "emit", Effect: &x.Effect{Kind: "message", Target: value("resolver"), Generation: one, Obligation: value("target"), Payload: map[string]x.Operand{"target": x.Ref("original")}}},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]x.Value{})
	resolver, err := author.Compile(author.Definition{Name: "phase2c-raw-resolver",
		Ports:   map[string]author.Port{"fromsource": {Peer: "source", Obligation: "target", Receive: rq.ReadInput()}, "start": {Peer: "local", Local: true, Obligation: "start", Receive: schema.Record{}}},
		Effects: map[string]author.Effect{"lookup": {Capability: rq.ReadCapability, Input: rq.ReadInput(), Output: rq.ReadOutput()}},
		Steps:   []author.Step{author.Receive("fromsource", map[string]string{"target": "received"}), author.Receive("start", nil), author.Request("query1", "lookup", map[string]author.Expr{"target": author.Var("received")}, nil), author.Await("lookup", "query1", map[string]string{"status": "query1_status", "target": "query1_target", "adoption": "query1_adoption", "adopted": "query1_adopted", "valid": "query1_valid", "mutation": "query1_mutation", "total": "query1_total"}), author.Success()},
	})
	if err != nil {
		t.Fatal(err)
	}
	var program x.Program
	if err = json.Unmarshal(resolver.Bytes(), &program); err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, runtime.Artifact{ID: resolver.ID(), Bytes: resolver.Bytes(), Semantics: program.Semantics})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: "resolver", Generation: 1, Owner: "owner:resolver", Revision: 1})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: resolver2c, ProgramID: resolver.ID(), Projection: runtime.Projection{Data: map[string]runtime.Value{}, Status: "open"}})
	p.Permissions = []runtime.Permission{
		{Kind: "external", Source: "source", Target: h.apply, Operation: "apply", Revision: 1},
		{Kind: "message", Source: "source", Target: "resolver", Obligation: "target", Operation: "message", Revision: 1},
		{Kind: "input", Source: "local", Target: "resolver", Obligation: "start", Operation: "input"},
		{Kind: "adapter", Source: "resolver", Target: rq.ReadCapability, Obligation: "lookup", Operation: "lookup", Revision: 1},
	}
	p.Composition = &runtime.CompositionProvision{Version: "alan/composition/v1", Roots: []runtime.ResourceRoot{
		{Run: orig, Members: []runtime.RunID{orig}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}},
		{Run: resolver2c, Members: []runtime.RunID{resolver2c}, Capabilities: []string{rq.ReadCapability}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}},
	}}
	p.Reconciliation = &runtime.ReconciliationProfile{Grants: []runtime.ReceiptGrant{{Resolver: resolver2c, Original: orig, Sequence: 1, Endpoint: h.apply, Adopt: true}}}
	phaseCreate(t, p, h.db)
}

func TestPhase2CProcessRaw(t *testing.T) {
	for _, tc := range []struct {
		name, profile string
		extra         int
	}{{"raw-single", "non-idempotent", 0}, {"raw-ambiguous", "non-idempotent", 1}, {"raw-keyed", "idempotent", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHistory2c(t, tc.name)
			ep, ready := h.endpoint("", nil)
			h.apply = ready.URL
			h.seedRaw(tc.profile)
			h.originalCut(cut2c("external.response", "pre-record", "source"), nil, true)
			h.expectWorld(1, 1, 0)
			o := h.successor(nil)
			h.snapshot("Q")
			if tc.extra == 1 {
				wire := h.data(h.world, "SELECT request FROM contacts WHERE number=1")
				client := http.Client{Timeout: 5 * time.Second}
				resp, e := client.Post(h.apply, "application/json", strings.NewReader(wire))
				if e != nil {
					t.Fatal(e)
				}
				body, e := io.ReadAll(io.LimitReader(resp.Body, 8193))
				resp.Body.Close()
				if e != nil || resp.StatusCode != http.StatusOK {
					t.Fatal("external second exposure", resp.Status, string(body), e)
				}
				h.event("external comparator POST", map[string]any{"Wire": wire, "Response": string(body)})
			}
			n, status, action := 1, "found", "evidence_only"
			if tc.name == "raw-ambiguous" {
				n, status, action = 2, "conflict", "conflict"
			}
			h.expectWorld(1+tc.extra, n, 0)
			h.snapshot("pre-read")
			r := h.read(o, "start")
			h.stop(o)
			h.expectWorld(1+tc.extra, n, 1)
			d := r.Resolver.Projection.Data
			if !d["query1_valid"].Flag || d["query1_status"].Text != status || d["query1_adoption"].Text != action || d["query1_adopted"].Flag || d["query1_target"].Text != h.original || h.count(h.db, "SELECT count(*) FROM external_receipts") != 0 || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 0 {
				t.Fatal("raw application interpretation", r)
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}

func TestPhase2CProcessLedger(t *testing.T) {
	for _, name := range []string{"ledger-restart", "ledger-replacement"} {
		t.Run(name, func(t *testing.T) {
			h := newHistory2c(t, name)
			old, first := h.endpoint("", nil)
			h.apply = first.URL
			h.seed(1)
			h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
			phaseStopEndpoint(t, old)
			h.event("endpoint stopped", first)
			if name == "ledger-replacement" {
				h.world = filepath.Join(h.work, "replacement.sqlite")
			}
			current, second := h.endpoint(phaseAddress(t, first.URL), nil)
			if first.Token == second.Token || first.PID == second.PID || phaseAddress(t, first.URL) != phaseAddress(t, second.URL) {
				t.Fatal(first, second)
			}
			o := h.successor(nil)
			h.snapshot("Q")
			r := h.read(o, "start")
			h.stop(o)
			if name == "ledger-restart" {
				h.expectFound(1, 1)
				h.expectWorld(1, 1, 1)
				var snap rq.Snapshot
				if e := json.Unmarshal([]byte(h.data(h.world, "SELECT response FROM receipt_reads")), &snap); e != nil {
					t.Fatal(e)
				}
				if first.LedgerID != second.LedgerID || snap.Receipt == nil || snap.Receipt.Process != first.Token || snap.Receipt.PID != first.PID {
					t.Fatal("incorrect continuity", snap, first, second)
				}
			} else {
				if first.LedgerID == second.LedgerID || r.Resolver.Projection.Data["query1_valid"].Flag || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 {
					t.Fatal("a replacement supplied evidence", first, second, r)
				}
				h.expectWorld(0, 0, 0)
			}
			h.replay("final")
			phaseStopEndpoint(t, current)
		})
	}
}

// The first retained transport error is already a terminal query outcome. Only
// a separately debited logical request can observe the restarted ledger.
func TestPhase2CProcessUnavailableRefresh(t *testing.T) {
	h := newHistory2c(t, "unavailable-refresh")
	first, ready := h.endpoint("", nil)
	h.apply = ready.URL
	h.seed(2)
	h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
	phaseStopEndpoint(t, first)
	h.event("endpoint stopped", ready)
	o := h.successor(nil)
	h.snapshot("Q")
	r := h.read(o, "start")
	values := r.Resolver.Projection.Data
	id := values["query1"].Text
	if id == "" || values["query1_valid"].Flag || values["query1_disposition"].Text != "transport_error" || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 {
		t.Fatal("unavailable lookup invented evidence", values)
	}
	firstResult := h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", id)
	h.expectWorld(1, 1, 0)
	h.replay("retained-error")
	second, restart := h.endpoint(phaseAddress(t, h.apply), nil)
	if restart.LedgerID != ready.LedgerID || restart.PID == ready.PID || restart.Token == ready.Token {
		t.Fatal("wrong restarted service", ready, restart)
	}
	r = h.read(o, "refresh")
	h.stop(o)
	h.expectWorld(1, 1, 1)
	h.expectFound(2, 2)
	if r.Resolver.Projection.Data["query2"].Text == id || h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", id) != firstResult {
		t.Fatal("retained error was reused or overwritten", id)
	}
	h.replay("final")
	phaseStopEndpoint(t, second)
}
