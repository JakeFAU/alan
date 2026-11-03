//go:build integration

package durabletest

import "testing"

// A positive lookup after a lost response can inform the resolver even when
// its grant cannot install the missing result on behalf of the closed source.
func TestReceiptReadGrantCannotAdoptOriginal(t *testing.T) {
	h := newHistory2c(t)
	endpoint, ready := h.endpoint("", nil)
	h.apply = ready.URL
	h.seed(1, false)
	h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
	owner := h.successor(nil)
	result := h.read(owner, "start")
	h.stop(owner)
	data := result.Resolver.Projection.Data
	if !data["query1_valid"].Flag || data["query1_status"].Text != "found" ||
		data["query1_target"].Text != h.original || data["query1_adoption"].Text != "denied" ||
		data["query1_adopted"].Flag {
		t.Fatal("read grant did not preserve the found receipt and separate adoption decision", data)
	}
	if h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 ||
		h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 0 {
		t.Fatal("read grant changed the original operation")
	}
	h.expectWorld(1, 1, 1)
	h.replay("after read without adoption")
	phaseStopEndpoint(t, endpoint)
}
