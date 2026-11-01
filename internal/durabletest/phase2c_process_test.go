//go:build integration

package durabletest

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

func (h *history2c) originalCut(cut *barrier, endpoint *child, raw bool) {
	t := h.t
	t.Helper()
	o := h.owner(cut)
	h.command(o, "run")
	var ack barrier
	if endpoint != nil {
		ack = h.await(endpoint)
	} else {
		ack = h.await(o.child)
	}
	kind := "adapter"
	if raw {
		kind = "external"
	}
	err := h.query(h.db, `SELECT id,json_extract(descriptor,'$.Run.Subject'),json_extract(descriptor,'$.Run.Generation') FROM intents WHERE json_extract(descriptor,'$.Kind')=?`, kind).Scan(&h.original, &h.source.Subject, &h.source.Generation)
	if err != nil {
		t.Fatal(err)
	}
	closed := h.command(o, "closed")
	if endpoint != nil {
		if ack.Intent != executor.ExternalKey(closed.Namespace, h.original) || phaseSnapshot(t, h.world).Contacts != 0 {
			t.Fatal("held packet identity", ack)
		}
		// The original call cannot return while this endpoint barrier is held.
		// Its owner has independently acknowledged terminal source state via fd5.
		o.acknowledged = true
	} else if ack.Intent != h.original {
		t.Fatal("wrong original cut", ack)
	}
	h.kill(o)
	if h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 0 ||
		h.count(h.db, "SELECT count(*) FROM external_receipts")+h.count(h.db, "SELECT count(*) FROM adapter_results") != 0 {
		t.Fatal("original already handled")
	}
	if !raw {
		if h.count(h.db, `SELECT count(*) FROM composition_freezes f,json_each(f.descriptor,'$.ExternalIntents') i WHERE i.value=? AND json_extract(f.descriptor,'$.Status')='succeeded'`, h.original) != 1 ||
			h.count(h.db, "SELECT count(*) FROM composition_publications") != 0 ||
			h.count(h.db, "SELECT count(*) FROM composition_reservations WHERE operation_id=?", h.original) != 1 {
			t.Fatal("original is not the captured missing result", h.original)
		}
	}
	h.replay("original-cut")
}

func (h *history2c) successor(cut *barrier) *owner2c {
	o := h.owner(cut)
	h.command(o, "Q")
	return o
}
func (h *history2c) read(o *owner2c, service string) reply2c {
	h.command(o, service)
	h.command(o, "run")
	return h.command(o, "finish")
}
func (h *history2c) stop(o *owner2c) { h.command(o, "exit"); phaseFinish(h.t, o.child) }

func (h *history2c) expectWorld(contacts, app, reads int) {
	t := h.t
	t.Helper()
	snap := phaseSnapshot(t, h.world)
	if snap.Contacts != int64(contacts) || len(snap.Applications) != app || snap.ReadContacts != int64(reads) {
		t.Fatalf("world contacts/apps/reads %d/%d/%d, want %d/%d/%d", snap.Contacts, len(snap.Applications), snap.ReadContacts, contacts, app, reads)
	}
}

func (h *history2c) expectFound(queries, ordinals int) {
	t := h.t
	t.Helper()
	for _, r := range []struct {
		stmt string
		args []any
		want int
	}{
		{"SELECT count(*) FROM adapter_results WHERE intent_id=? AND json_type(descriptor,'$.Reconciled')='object'", []any{h.original}, 1},
		{"SELECT count(*) FROM settlements WHERE intent_id=?", []any{h.original}, 1},
		{"SELECT count(*) FROM composition_publications", nil, 1},
		{"SELECT count(*) FROM receipt_read_ordinals", nil, ordinals},
		{"SELECT count(*) FROM composition_reservations", nil, queries + 1},
		{`SELECT count(*) FROM runs WHERE subject='parent' AND json_extract(projection,'$.Status')='succeeded' AND json_extract(projection,'$.Data.decision.Text')='applied' AND json_extract(projection,'$.Data.receipt_target.Text')=? AND json_extract(projection,'$.Data.issued_target.Text')=?`, []any{h.original, h.original}, 1},
		{`SELECT count(*) FROM observations o JOIN history h ON h.selected_observation=o.id WHERE h.subject='parent' AND o.id IN (SELECT observation_id FROM composition_outcome_deliveries)`, nil, 1},
		{`SELECT count(*) FROM observations o JOIN history h ON h.selected_observation=o.id WHERE h.subject='parent' AND json_extract(o.descriptor,'$.Obligation')='receipt' AND json_extract(o.descriptor,'$.Payload.status.Text')='found'`, nil, 1},
	} {
		if n := h.count(h.db, r.stmt, r.args...); n != r.want {
			t.Fatalf("got %d want %d for %s", n, r.want, r.stmt)
		}
	}
}

func TestPhase2CProcessMatched(t *testing.T) {
	shared := t.TempDir()
	empty := filepath.Join(shared, "empty.sqlite")
	initial := filepath.Join(shared, "initial.sqlite")
	ledger, err := fixture.OpenLedger(t.Context(), empty, "fork-initializer")
	if err != nil {
		t.Fatal(err)
	}
	if err = ledger.Close(); err != nil {
		t.Fatal(err)
	}
	var address string
	var cuts, reopened [2]runtime.Inspection
	for i := 0; i < 2; i++ {
		label := fmt.Sprintf("U%d", i)
		t.Run(label, func(t *testing.T) {
			h := newHistory2c(t, label)
			phaseCopy(t, empty, h.world)
			ep, ready := h.endpoint(address, nil)
			h.apply = ready.URL
			cut := cut2c("dispatch.admitted", "pre-call", "")
			if i == 0 {
				address = phaseAddress(t, h.apply)
				h.seed(1)
				phaseCopy(t, h.db, initial)
			} else {
				phaseCopy(t, initial, h.db)
				cut = cut2c("adapter.response", "pre-record", "")
			}
			h.originalCut(cut, nil, false)
			cuts[i] = inspect(t, h.db)
			o := h.successor(nil)
			h.snapshot("Q")
			reopened[i] = inspect(t, h.db)
			h.expectWorld(i, i, 0)
			result := h.read(o, "start")
			h.stop(o)
			h.expectWorld(i, i, 1)
			if i == 0 {
				if result.Resolver.Projection.Data["query1_status"].Text != "not_found" || h.count(h.db, "SELECT count(*) FROM composition_publications") != 0 || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 {
					t.Fatal("absence resolved original", result)
				}
			} else {
				h.expectFound(1, 1)
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
	if !reflect.DeepEqual(cuts[0], cuts[1]) || !reflect.DeepEqual(reopened[0], reopened[1]) {
		t.Fatal("local pre-query cuts differ")
	}
}

func (h *history2c) releasePacket(endpoint *child) {
	t := h.t
	t.Helper()
	h.event("release original packet", map[string]int{"EndpointPID": endpoint.cmd.Process.Pid})
	if _, err := endpoint.release.Write([]byte{'c'}); err != nil {
		t.Fatal(err)
	}
	ctx := time.After(10 * time.Second)
	for h.count(h.world, "SELECT count(*) FROM applications") == 0 {
		select {
		case <-ctx:
			t.Fatal("original packet not applied")
		case <-time.After(2 * time.Millisecond):
		}
	}
	h.event("original application", phaseSnapshot(t, h.world).Applications)
}

func TestPhase2CProcessHeld(t *testing.T) {
	for _, lost := range []bool{false, true} {
		label := "held-retained"
		queries := 2
		if lost {
			label = "held-lost"
			queries = 1
		}
		t.Run(label, func(t *testing.T) {
			h := newHistory2c(t, label)
			ep, ready := h.endpoint("", cut2c("resource.ingress", "pre-apply", ""))
			h.apply = ready.URL
			h.seed(queries)
			h.originalCut(nil, ep, false)
			var queryCut *barrier
			if lost {
				queryCut = cut2c("adapter.response", "pre-record", "resolver")
			}
			o := h.successor(queryCut)
			h.expectWorld(0, 0, 0)
			h.snapshot("Q")
			h.command(o, "start")
			h.command(o, "run")
			if lost {
				ack := h.await(o.child)
				h.expectWorld(0, 0, 1)
				if h.count(h.db, "SELECT count(*) FROM adapter_results") != 0 || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=? AND ordinal=1", ack.Intent) != 1 {
					t.Fatal("unexpected retained answer")
				}
				h.kill(o)
				h.replay("lost-absence")
			} else {
				r := h.command(o, "finish")
				if r.Resolver.Projection.Data["query1_status"].Text != "not_found" {
					t.Fatal("no selected absence", r)
				}
				h.replay("retained-absence")
			}
			h.releasePacket(ep)
			h.expectWorld(1, 1, 1)
			if lost {
				o = h.successor(nil)
			} else {
				h.command(o, "refresh")
			}
			h.command(o, "run")
			h.finish(o)
			h.expectWorld(1, 1, 2)
			h.expectFound(queries, 2)
			var a, b receiptquery.Snapshot
			if err := json.Unmarshal([]byte(h.data(h.world, "SELECT response FROM receipt_reads WHERE number=1")), &a); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(h.data(h.world, "SELECT response FROM receipt_reads WHERE number=2")), &b); err != nil {
				t.Fatal(err)
			}
			if a.Status != "not_found" || b.Status != "found" || a.Query.Ordinal != 1 || (a.Query.QueryID == b.Query.QueryID) != lost || (b.Query.Ordinal == 2) != lost || b.Query.TargetID != h.original || b.Receipt == nil || b.Receipt.PID != ep.cmd.Process.Pid {
				t.Fatal("snapshot interval", a, b)
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}

func TestPhase2CProcessAccounting(t *testing.T) {
	for _, name := range []string{"atomic-import", "exhausted"} {
		t.Run(name, func(t *testing.T) {
			h := newHistory2c(t, name)
			ep, ready := h.endpoint("", nil)
			h.apply = ready.URL
			h.seed(1)
			h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
			cut := cut2c("adapter.result-recorded", "pre-settlement", "resolver")
			if name == "exhausted" {
				cut = cut2c("dispatch.admitted", "pre-call", "resolver")
			}
			o := h.successor(cut)
			h.command(o, "start")
			h.command(o, "run")
			query := h.await(o.child).Intent
			if name == "atomic-import" {
				h.expectWorld(1, 1, 1)
				if h.count(h.db, "SELECT count(*) FROM adapter_results") != 2 || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id IN (?,?)", query, h.original) != 0 || h.count(h.db, "SELECT count(*) FROM composition_publications") != 0 {
					t.Fatal("atomic committed cut already settled or published")
				}
				h.kill(o)
				h.replay("atomic-committed")
				o = h.successor(nil)
				h.command(o, "run")
				h.finish(o)
				h.expectWorld(1, 1, 1)
				h.expectFound(1, 1)
			} else {
				h.kill(o)
				h.replay("ordinal-1")
				o = h.successor(cut)
				h.command(o, "run")
				if ack := h.await(o.child); ack.Intent != query {
					t.Fatal("replaced logical query", ack)
				}
				h.kill(o)
				h.replay("ordinal-2")
				o = h.successor(nil)
				h.command(o, "run")
				result := h.finish(o)
				h.expectWorld(1, 1, 0)
				if h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", query) != 2 || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", h.original) != 0 || h.count(h.db, "SELECT count(*) FROM composition_reservations") != 2 || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", query) != 1 || result.Resolver.Projection.Data["query1_valid"].Flag {
					t.Fatal("exhaustion created evidence/reset allowance", result)
				}
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}
