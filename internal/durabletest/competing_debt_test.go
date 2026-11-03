//go:build integration

package durabletest

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Both resolvers have received a genuine response to the same missing original.
// An abort during the winner's accounting leaves the duplicate free to settle
// its own query. Only the adopter may repay the original on recovery.
func TestCompetingReadCannotSettleAdopterDebt(t *testing.T) {
	for _, fault := range []string{"original", "adopter query"} {
		t.Run(fault, func(t *testing.T) {
			h := newHistory2c(t)
			endpoint, ready := h.endpoint("", nil)
			h.apply = ready.URL
			hostileSeed(t, h, true)
			h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
			s := liveSuccessor(t, h)
			startRead(t, s, "resolver", map[string]rt.Value{})
			startRead(t, s, "twin", map[string]rt.Value{"target": schema.Text(h.original)})
			returned := make(chan boundary.Event, 2)
			release := make(chan struct{})
			done := runReaders(t, s, ready, 2, func(ctx context.Context, e boundary.Event) error {
				if e.Boundary != "adapter.response" {
					return nil
				}
				returned <- e
				if e.Subject == resolver2c.Subject {
					select {
					case <-release:
					case <-ctx.Done():
					}
				} else {
					<-ctx.Done()
				}
				return nil
			})
			reads := map[string]string{}
			for range 2 {
				e := recvReadBoundary(t, returned, done)
				reads[e.Subject] = e.Intent
			}
			adopter, duplicate := reads[resolver2c.Subject], reads["twin"]
			if adopter == "" || duplicate == "" || adopter == duplicate {
				t.Fatal("expected two distinct admitted reads", reads)
			}
			h.expectWorld(1, 1, 2)
			admitControl(t, s, "operator", resolver2c.Subject)
			if state := untilTerminal(t, s, resolver2c); state.Projection.Status != "cancelled" {
				t.Fatal("adopter was not revoked", state)
			}
			blocked, originalSettlements := h.original, 0
			if fault == "adopter query" {
				blocked, originalSettlements = adopter, 1
			}
			db, err := sql.Open("sqlite3", h.db+"?_busy_timeout=5000")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			quoted := h.data(h.db, "SELECT quote(?)", blocked)
			if _, err = db.ExecContext(t.Context(), "CREATE TRIGGER abort_adopter BEFORE INSERT ON settlements WHEN NEW.intent_id="+quoted+" BEGIN SELECT RAISE(ABORT,'adopter settlement aborted'); END"); err != nil {
				t.Fatal(err)
			}
			close(release)
			if err = <-done; err == nil || !strings.Contains(err.Error(), "adopter settlement aborted") {
				t.Fatalf("Run did not report the accounting failure: %v", err)
			}
			decision := func(id string) string {
				return h.data(h.db, "SELECT json_extract(descriptor,'$.Read.Adoption') FROM adapter_results WHERE intent_id=?", id)
			}
			settlements := func(id string) int {
				return h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", id)
			}
			if decision(adopter) != "adopted" || decision(duplicate) != "duplicate" ||
				settlements(duplicate) != 1 || settlements(adopter) != 0 || settlements(h.original) != originalSettlements {
				t.Fatal("duplicate changed the adopter's original or its settlement")
			}
			published, err := s.SettleInvocation(t.Context(), `["parent",1,1,0]`)
			if err != nil || !published.Accepted || settlements(h.original) != originalSettlements || settlements(adopter) != 0 {
				t.Fatal("publication changed outstanding accounting", published, err)
			}
			h.replay("published with outstanding adopter debt")
			if _, err = db.ExecContext(t.Context(), "DROP TRIGGER abort_adopter"); err != nil {
				t.Fatal(err)
			}
			q := dispatchTarget(s, resolver2c, false)
			for range 2 {
				if err = s.AwaitDispatchQuiescence(t.Context(), q); err == nil || !strings.Contains(err.Error(), "adopter settlement aborted") {
					t.Fatal("same owner forgot the accounting error", err)
				}
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			successor, err := rt.Open(t.Context(), h.db, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer successor.Close()
			if err = successor.AwaitDispatchQuiescence(t.Context(), q); err != nil ||
				settlements(h.original) != 1 || settlements(adopter) != 1 || settlements(duplicate) != 1 {
				t.Fatal("new owner did not settle exactly once", err)
			}
			if h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals") != 2 ||
				h.count(h.db, "SELECT count(*) FROM composition_reservations") != 3 {
				t.Fatal("paying the debt spent another physical read or semantic reservation")
			}
			h.expectWorld(1, 1, 2)
			h.replay("recovered adopter debt")
			phaseStopEndpoint(t, endpoint)
		})
	}
}
