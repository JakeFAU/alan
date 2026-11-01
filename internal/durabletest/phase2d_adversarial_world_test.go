//go:build integration

package durabletest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	frame "github.com/JakeFAU/alan/examples/effectfulcomposition"
	x "github.com/JakeFAU/alan/internal/executor"
)

// Extra external submissions use the real committed L key with different
// (non-idempotent) wire. Both applications belong to those external packets.
func TestPhase2DMultiplicity(t *testing.T) {
	h := newHistory2c(t, "external-multiplicity")
	ep, ready := h.endpoint("", nil)
	h.apply = ready.URL
	p := adverseFrame(t, ready.URL, frame.RoleOptions{}, frame.RoleOptions{})
	phaseCreate(t, p, h.db)
	s := adverseOpen(t, h.db)
	parent := publicStep(t, s, staticRun("parent"), "")
	left := invokeEntry(t, s, h.db, parent.Projection.Data["l_invocation"].Text)
	state := publicStep(t, s, left, "")
	id := state.Projection.Data["original"].Text
	if id == "" {
		t.Fatal("missing authentic operation")
	}
	request := rq.Mutation{Key: x.ExternalKey(s.Namespace(), id), Profile: "non-idempotent", Delta: 1}
	wire, e := json.Marshal(request)
	if e != nil {
		t.Fatal(e)
	}
	client := http.Client{Timeout: 5 * time.Second}
	for i := 1; i <= 2; i++ {
		res, e := client.Post(ready.URL, "application/json", bytes.NewReader(wire))
		if e != nil {
			t.Fatal(e)
		}
		receipt, e := io.ReadAll(res.Body)
		res.Body.Close()
		var r rq.Receipt
		if e != nil || res.StatusCode != http.StatusOK || json.Unmarshal(receipt, &r) != nil || r.Mutation != int64(i) || r.Key != request.Key || r.Profile != "non-idempotent" {
			t.Fatal("independent submission", res.Status, e, string(receipt))
		}
		h.event("external non-idempotent submission", map[string]any{"Wire": string(wire), "Receipt": r})
	}
	h.expectWorld(2, 2, 0)
	opts := adverseOptions(ready, "success")
	adverseRun(t, s, opts)
	ids := adverseQ(t, h, s)
	if ids["l"] != id {
		t.Fatal(ids)
	}
	roleService(t, s, "l", "start")
	roleService(t, s, "r", "start")
	adverseRun(t, s, opts)
	l := roleState(t, s, "resolver_l").Projection.Data
	par := roleState(t, s, "parent")
	if l["query1_status"].Text != "conflict" || l["query1_adoption"].Text != "conflict" || !l["conflict"].Flag || l["witness"].Flag || par.Projection.Status != "failed" || par.Projection.Data["decision"].Text != "invalid_evidence" {
		t.Fatal("multiplicity accepted", par.Projection.Status, par.Projection.Data["decision"], l["query1_status"])
	}
	if n := h.count(h.world, "SELECT count(*) FROM application_wires WHERE key=?", request.Key); n != 2 {
		t.Fatal(n)
	}
	h.expectWorld(3, 3, 2) // three committed mutation contacts; L's incompatible POST is separately journaled.
	h.replay("final")
	phaseStopEndpoint(t, ep)
}

// Trust-boundary mutation. The real upstream sends its immutable first receipt
// twice. The interposed service lies about the second receipt's numeric fields.
// No ALAN row is edited; the report must preserve the first selected witness.
func TestPhase2DIncompatiblePositive(t *testing.T) {
	h := newHistory2c(t, "lying-second-snapshot")
	ep, ready := h.endpoint("", nil)
	dest, e := url.Parse(ready.URL)
	if e != nil {
		t.Fatal(e)
	}
	type substitution struct{ Upstream, Delivered string }
	mutations := make(chan substitution, 1)
	client := http.Client{Timeout: 8 * time.Second}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, e := io.ReadAll(r.Body)
		if e != nil {
			http.Error(w, e.Error(), 502)
			return
		}
		resp, e := client.Post(dest.Scheme+"://"+dest.Host+r.URL.RequestURI(), "application/json", bytes.NewReader(body))
		if e != nil {
			http.Error(w, e.Error(), 502)
			return
		}
		b, e := io.ReadAll(resp.Body)
		resp.Body.Close()
		if e != nil {
			http.Error(w, e.Error(), 502)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/lookup") && resp.StatusCode == http.StatusOK {
			var res rq.Snapshot
			if e = json.Unmarshal(b, &res); e != nil {
				http.Error(w, e.Error(), 502)
				return
			}
			// Prefixing the test's origin instruction sequence would duplicate
			// the program compiler. Use the actual second selected role request.
			if res.Number == 2 && res.Receipt != nil {
				before := string(b)
				res.Receipt.Mutation += 3
				res.Receipt.Total += 3
				b, e = json.Marshal(res)
				if e != nil {
					http.Error(w, e.Error(), 502)
					return
				}
				mutations <- substitution{before, string(b)}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(b)
	}))
	defer proxy.Close()
	h.apply = proxy.URL + dest.Path
	proxied := ready
	proxied.URL = h.apply
	proxied.LookupURL, e = rq.LookupEndpoint(h.apply)
	if e != nil {
		t.Fatal(e)
	}
	p := adverseFrame(t, h.apply, frame.RoleOptions{Queries: 2}, frame.RoleOptions{})
	phaseCreate(t, p, h.db)
	s := adverseOpen(t, h.db)
	opts := adverseOptions(proxied, "success")
	adverseRun(t, s, opts)
	ids := adverseQ(t, h, s)
	roleService(t, s, "l", "start")
	adverseRun(t, s, opts)
	first := roleState(t, s, "resolver_l").Projection.Data
	if !first["witness"].Flag || first["witness_target"].Text != ids["l"] || first["witness_query"].Text != first["query1"].Text || first["conflict"].Flag {
		t.Fatal("invalid first witness")
	}
	h.snapshot("honest-first")
	roleService(t, s, "l", "refresh")
	adverseRun(t, s, opts)
	select {
	case m := <-mutations:
		h.event("untrusted intermediary substituted snapshot", m)
	default:
		t.Fatal("substitution did not occur")
	}
	left := roleState(t, s, "resolver_l").Projection.Data
	if left["witness_mutation"] != first["witness_mutation"] || left["witness_total"] != first["witness_total"] || left["witness_query"] != first["witness_query"] || !left["conflict"].Flag || left["query2_status"].Text != "found" || !left["query2_valid"].Flag || left["query2_mutation"].Number != left["witness_mutation"].Number+3 {
		t.Fatal("incompatible found erased or became first")
	}
	roleService(t, s, "r", "start")
	adverseRun(t, s, opts)
	par := roleState(t, s, "parent")
	if par.Projection.Status != "failed" || par.Projection.Data["decision"].Text != "invalid_evidence" || !par.Projection.Data["l_report_conflict"].Flag {
		t.Fatal("parent accepted inconsistent role evidence", par.Projection.Status, par.Projection.Data["decision"])
	}
	h.expectWorld(2, 2, 3)
	h.replay("final")
	phaseStopEndpoint(t, ep)
}
