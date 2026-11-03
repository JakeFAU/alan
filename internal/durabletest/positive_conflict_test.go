//go:build integration

package durabletest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/runtime"
)

// A second positive snapshot may be well-formed yet disagree with the first
// receipt. It must leave the original result and its one settlement intact.
func TestIncompatiblePositiveReadKeepsAdoptedOriginal(t *testing.T) {
	h := newHistory2c(t)
	ep, ready := h.endpoint("", nil)
	ledger, err := url.Parse(ready.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	var altered atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream, err := client.Post(ledger.Scheme+"://"+ledger.Host+r.URL.RequestURI(), "application/json", r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Body.Close()
		raw, err := io.ReadAll(upstream.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if upstream.StatusCode == http.StatusOK && strings.HasSuffix(r.URL.Path, "/lookup") {
			var snap receiptquery.Snapshot
			if err = json.Unmarshal(raw, &snap); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			if snap.Number == 2 && snap.Receipt != nil {
				snap.Receipt.Mutation += 3
				snap.Receipt.Total += 3
				raw, err = json.Marshal(snap)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				altered.Add(1)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(upstream.StatusCode)
		_, _ = w.Write(raw)
	}))
	defer proxy.Close()
	h.apply = proxy.URL + ledger.Path
	h.seed(2, true)
	h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
	owner := h.successor(nil)
	first := h.read(owner, "start").Resolver.Projection.Data
	if first["query1_adoption"].Text != "adopted" || !first["query1_adopted"].Flag {
		t.Fatal("missing first adoption", first)
	}
	original := func() string {
		return h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", h.original)
	}
	before := original()
	second := h.read(owner, "refresh").Resolver.Projection.Data
	h.stop(owner)
	if altered.Load() != 1 || !second["query2_valid"].Flag || second["query2_status"].Text != "found" ||
		second["query2_target"].Text != h.original || second["query2_adoption"].Text != "conflict" ||
		second["query2_adopted"].Flag || second["query2_mutation"].Number == first["query1_mutation"].Number {
		t.Fatal("incompatible positive did not remain distinguishable", second)
	}
	if original() != before || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 1 {
		t.Fatal("conflicting positive replaced the first adopted result or changed settlement")
	}
	cut, err := runtime.InspectExecution(t.Context(), h.db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	e, ok := cut.Effect(h.original)
	if !ok || e.Settlement == nil || e.AdapterResult == nil || e.AdapterResult.Reconciled == nil ||
		e.ReceiptEvidenceState != "conflict" || len(e.ReadsOfOriginal) != 2 || e.Dispatch.Minimum != 0 {
		t.Fatalf("original adopted result or subsequent conflict missing: %+v", e)
	}
	adoptions := map[string]int{}
	for _, id := range e.ReadsOfOriginal {
		query, ok := cut.Effect(id)
		if !ok || len(query.ReadOpportunities) != 1 || query.Dispatch.Minimum != 1 {
			t.Fatalf("missing read opportunity: %+v", query)
		}
		read := query.ReadOpportunities[0]
		if !read.MayAdopt || read.Query.TargetID != e.OperationID || read.Snapshot == nil || read.Snapshot.Status != "found" ||
			(read.Adoption == "adopted") != (id == e.AdapterResult.Reconciled.QueryID) {
			t.Fatalf("read and original adoption disagree: %+v", read)
		}
		adoptions[read.Adoption]++
	}
	if adoptions["adopted"] != 1 || adoptions["conflict"] != 1 {
		t.Fatal(adoptions)
	}
	h.expectWorld(1, 1, 2)
	h.replay("incompatible positive")
	phaseStopEndpoint(t, ep)
}
