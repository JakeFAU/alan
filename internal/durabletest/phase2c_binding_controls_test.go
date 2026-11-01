//go:build integration

package durabletest

import (
	"strings"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func bindingInput(t *testing.T, s *rt.Store, run rt.RunID, port string, values map[string]rt.Value) string {
	t.Helper()
	key := run.Subject + "-" + port
	a, e := s.AdmitInput(t.Context(), rt.Input{TransportID: key, LogicalID: key, Target: run.Subject, Generation: run.Generation, Obligation: port, Payload: values})
	if e != nil || !a.Accepted {
		t.Fatal(a, e)
	}
	return a.ObservationID
}

// These static controls hand a local selector to an ordinary reader. Tests
// manipulate service order or provisioned grants, never application evidence.
func TestPhase2COriginalBindingControls(t *testing.T) {
	for _, name := range []string{"unknown-policy", "existing-selected-error", "missing-authorization", "missing-reservation", "denied-authorization", "denied-grant", "wrong-selector", "wrong-ledger-pin"} {
		t.Run(name, func(t *testing.T) {
			h := newHistory2c(t, name)
			ep, ready := h.endpoint("", nil)
			h.apply = ready.URL
			o := rt.RunID{Subject: "zzz-original", Generation: 1}
			r := rt.RunID{Subject: "aaa-reader", Generation: 1}
			original, e := author.Compile(author.Definition{Name: "controlled-original", Ports: map[string]author.Port{"end": {Peer: "local", Local: true, Obligation: "end", Receive: schema.Record{}}},
				Effects: map[string]author.Effect{"increment": {Capability: rq.CounterCapability, Input: rq.CounterInput(), Output: rq.CounterOutput()}},
				Steps:   []author.Step{author.Request("original", "increment", map[string]author.Expr{"delta": author.Integer(1)}, nil), author.Receive("end", nil), author.Success()}})
			if e != nil {
				t.Fatal(e)
			}
			reader, e := author.Compile(author.Definition{Name: "controlled-reader", Ports: map[string]author.Port{"start": {Peer: "local", Local: true, Obligation: "start", Receive: rq.ReadInput()}},
				Effects: map[string]author.Effect{"lookup": {Capability: rq.ReadCapability, Input: rq.ReadInput(), Output: rq.ReadOutput()}},
				Steps:   []author.Step{author.Receive("start", map[string]string{"target": "selector"}), author.Request("query", "lookup", map[string]author.Expr{"target": author.Var("selector")}, nil), author.Await("lookup", "query", map[string]string{"status": "status", "adoption": "decision", "adopted": "adopted", "disposition": "outcome"}), author.Success()}})
			if e != nil {
				t.Fatal(e)
			}
			p, e := (author.Application{o.Subject: {Artifact: original}, r.Subject: {Artifact: reader}}).Provision()
			if e != nil {
				t.Fatal(e)
			}
			p.Composition = &rt.CompositionProvision{Version: "alan/composition/v1", Roots: []rt.ResourceRoot{
				{Run: o, Members: []rt.RunID{o}, Capabilities: []string{rq.CounterCapability}, Limits: rt.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}},
				{Run: r, Members: []rt.RunID{r}, Capabilities: []string{rq.ReadCapability}, Limits: rt.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}}}
			grant := rt.ReceiptGrant{Original: o, Resolver: r, Sequence: 1, Endpoint: ready.URL, Adopt: true}
			if name == "wrong-ledger-pin" {
				grant.Endpoint = strings.Replace(ready.URL, ready.LedgerID, strings.Repeat("f", 64), 1)
			}
			p.Reconciliation = &rt.ReconciliationProfile{Grants: []rt.ReceiptGrant{grant}}
			if name == "denied-grant" {
				p.Reconciliation.Grants = nil
			}
			phaseCreate(t, p, h.db)
			s, e := rt.Open(t.Context(), h.db, rt.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			mode := "unknown"
			if name == "existing-selected-error" {
				mode = "selected-error"
			}
			opts := rt.RunOptions{MaxSteps: 80, MaxDispatch: 80, MaxInFlight: 1, ExternalTimeout: 4 * time.Second, Adapters: map[string]effects.Adapter{rq.ReadCapability: rq.Reader{Endpoint: ready.LookupURL}, rq.CounterCapability: rq.Counter{Endpoint: ready.URL, Mode: mode}}}
			issued := publicStep(t, s, o, "")
			h.original = issued.Projection.Data["original"].Text
			h.source = o
			if h.original == "" {
				t.Fatal("missing authentic ID")
			}
			early := strings.HasPrefix(name, "missing-") || name == "denied-authorization"
			if name == "missing-reservation" {
				auth, err := s.AuthorizeIntent(t.Context(), h.original)
				if err != nil || !auth.Allowed {
					t.Fatal(auth, err)
				}
			}
			if !early {
				if _, e = s.Run(t.Context(), opts); e != nil {
					t.Fatal(e)
				}
				h.expectWorld(1, 1, 0)
			}
			publicStep(t, s, o, bindingInput(t, s, o, "end", map[string]rt.Value{}))
			closed := publicStep(t, s, o, "")
			if closed.Projection.Status != "succeeded" {
				t.Fatal(closed)
			}
			if e = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, o, false)); e != nil {
				t.Fatal(e)
			}
			if name == "denied-authorization" {
				auth, err := s.AuthorizeIntent(t.Context(), h.original)
				if err != nil || auth.Allowed {
					t.Fatal(auth, err)
				}
			}
			selector := h.original
			if name == "wrong-selector" {
				selector = `["zzz-original",2,1,0]`
			}
			publicStep(t, s, r, bindingInput(t, s, r, "start", map[string]rt.Value{"target": schema.Text(selector)}))
			query := publicStep(t, s, r, "").Projection.Data["query"].Text
			var before string
			if name == "existing-selected-error" {
				before = h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", h.original)
				if h.data(h.db, "SELECT json_extract(descriptor,'$.Disposition') FROM adapter_results WHERE intent_id=?", h.original) != "selected_error" {
					t.Fatal(before)
				}
			}
			h.snapshot("before-read")
			if _, e = s.Run(t.Context(), opts); e != nil {
				t.Fatal(e)
			}
			got, e := s.Snapshot(t.Context(), r)
			if e != nil || got.Projection.Status != "succeeded" {
				t.Fatal(got, e)
			}
			data := got.Projection.Data
			switch name {
			case "unknown-policy", "existing-selected-error":
				action := "unsupported_policy"
				if name == "existing-selected-error" {
					action = "existing"
					if h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", h.original) != before {
						t.Fatal("changed original selected error")
					}
				} else if h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 {
					t.Fatal("reconstructed an unknown-policy adapter")
				}
				if data["decision"].Text != action || data["status"].Text != "found" || data["adopted"].Flag {
					t.Fatal(data)
				}
				h.expectWorld(1, 1, 1)
			default:
				want := "read_denied"
				switch name {
				case "missing-authorization", "missing-reservation":
					want = "attempt_missing"
				case "denied-authorization":
					want = "original_denied"
				case "wrong-ledger-pin":
					want = "endpoint_mismatch"
				}
				if data["outcome"].Text != want || data["adopted"].Flag || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", query) != 0 {
					t.Fatal(want, data)
				}
				if early {
					h.expectWorld(0, 0, 0)
				} else {
					h.expectWorld(1, 1, 0)
				}
			}
			h.event("ordinary reader selected", got)
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}
