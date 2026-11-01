//go:build integration

package durabletest

import "testing"

// The parent's two-query contract checks the report the resolver actually sent.
// First adoption is durable even when the final duplicate fails that guard.
func TestPhase2CRefreshAfterAdoption(t *testing.T) {
	h := newHistory2c(t, "final-duplicate")
	ep, ready := h.endpoint("", nil)
	h.apply = ready.URL
	h.seed(2)
	h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
	o := h.successor(nil)
	first := h.read(o, "start")
	if first.Resolver.Projection.Data["query1_adoption"].Text != "adopted" {
		t.Fatal("first real read did not adopt", first.Resolver)
	}
	h.snapshot("adopted")
	second := h.read(o, "refresh")
	h.stop(o)
	if second.Resolver.Projection.Data["query2_adoption"].Text != "duplicate" || second.Resolver.Projection.Data["query2_adopted"].Flag {
		t.Fatal("final read did not preserve duplicate evidence", second.Resolver)
	}
	if h.count(h.db, `SELECT count(*) FROM runs WHERE subject='parent' AND json_extract(projection,'$.Status')='failed' AND json_extract(projection,'$.Data.source_status.Text')='succeeded' AND json_extract(projection,'$.Data.receipt_adoption.Text')='duplicate' AND json_extract(projection,'$.Data.receipt_target.Text')=?`, h.original) != 1 {
		t.Fatal("caller did not reject its final selected duplicate")
	}
	if h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 1 || h.count(h.db, "SELECT count(*) FROM composition_reservations") != 3 {
		t.Fatal("adoption or semantic charge changed on refresh")
	}
	h.expectWorld(1, 1, 2)
	h.replay("final")
	phaseStopEndpoint(t, ep)
}
