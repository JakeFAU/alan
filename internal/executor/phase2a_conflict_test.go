//go:build integration

package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	publicschema "github.com/JakeFAU/alan/schema"
)

// Challenge the recorder directly. The public parent has no corresponding
// receipt injection method. Only the first candidate supplies the settlement.
func TestPhase2ARecordConflict(t *testing.T) {
	s, path, id := committedAdapter(t)
	work, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || work == nil {
		t.Fatal(work, err)
	}
	first := transportAdapter(t.Context(), *work).candidate
	retained, err := s.recordAdapterResult(t.Context(), work.attempt, first)
	if err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"foreign-key", "contradictory-result"} {
		t.Run(control, func(t *testing.T) {
			changed := effects.Copy(first)
			if control == "foreign-key" {
				changed.Key += "-other"
			} else {
				changed.Payload["answer"] = Text("incompatible")
			}
			_, err := s.recordAdapterResult(t.Context(), work.attempt, changed)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("unexpected recorder disposition %v", err)
			}
			inspection, inspectErr := Inspect(t.Context(), path, DefaultLimits())
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			rows := inspection.Tables["adapter_results"]
			if len(rows) != 1 {
				t.Fatal("lost or duplicated original result")
			}
			var actual AdapterResult
			if e := json.Unmarshal([]byte(rows[0]["descriptor"].(string)), &actual); e != nil || !reflect.DeepEqual(actual, retained) {
				t.Fatal("changed original", e)
			}
			if (len(inspection.Tables["receipt_conflicts"]) == 1) != (control == "contradictory-result") {
				t.Fatalf("wrong forensic classification for %s", control)
			}
			raw, e := json.Marshal(map[string]any{"original": first, "offered": changed, "error": err.Error(), "retained": actual, "attempt": work.attempt})
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("RECORDER %s", raw)
		})
	}
}

func TestBaselineRecorderOfferOrder(t *testing.T) {
	for _, order := range [][2]int64{{5, 10}, {10, 5}} {
		name := fmt.Sprintf("%d_then_%d", order[0], order[1])
		t.Run("raw/"+name, func(t *testing.T) {
			s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
			run, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), run.Projection.Data["pending"].Text)
			if err != nil {
				t.Fatal(err)
			}
			checkConflictingOffers(t, path, a.Intent.ID, order, func(total int64) error {
				receipt := receiptFor(a.Request)
				receipt.Total = total
				return s.recordExternalReceipt(t.Context(), a, receipt)
			})
			got, err := externalOutcome(t.Context(), s.conn, a.Intent.ID)
			if err != nil || got.Receipt == nil || got.Receipt.Total != order[0] {
				t.Fatalf("retained raw result = %+v, %v", got, err)
			}
		})
		t.Run("adapter/"+name, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			work, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
			if err != nil || work == nil {
				t.Fatal(work, err)
			}
			base := transportAdapter(t.Context(), *work).candidate
			checkConflictingOffers(t, path, id, order, func(value int64) error {
				c := effects.Copy(base)
				c.Payload["answer"] = Text(fmt.Sprint(value))
				_, err := s.recordAdapterResult(t.Context(), work.attempt, c)
				return err
			})
			got, err := loadAdapterResult(t.Context(), s.conn, id)
			if err != nil || got.Candidate.Payload["answer"] != Text(fmt.Sprint(order[0])) {
				t.Fatalf("retained adapter result = %+v, %v", got, err)
			}
		})
	}
}

func checkConflictingOffers(t *testing.T, path, id string, order [2]int64, offer func(int64) error) {
	t.Helper()
	if err := offer(order[0]); err != nil {
		t.Fatal(err)
	}
	before, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := offer(order[1]); !errors.Is(err, ErrConflict) {
		t.Fatalf("second offer returned %v, want ErrConflict", err)
	}
	after, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	rows := after.Tables["receipt_conflicts"]
	if len(rows) != 1 || rows[0]["intent_id"] != id {
		t.Fatalf("acknowledged conflict for %s has no unique durable witness: %v", id, rows)
	}
	for table, original := range before.Tables {
		if table != "metadata" && !reflect.DeepEqual(original, after.Tables[table]) {
			t.Fatalf("conflict changed %s", table)
		}
	}
	if err := offer(order[1]); !errors.Is(err, ErrConflict) {
		t.Fatalf("repeat losing offer = %v, want ErrConflict", err)
	}
	if err := offer(order[0]); err != nil {
		t.Fatalf("repeat winning offer = %v", err)
	}
	again, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil || !reflect.DeepEqual(after, again) {
		t.Fatalf("duplicate offers changed the inspection: %v", err)
	}
	replay, err := Replay(t.Context(), path, DefaultLimits())
	if err != nil || len(replay.ReceiptEvidence) != 1 || replay.ReceiptEvidence[0].IntentID != id ||
		replay.ReceiptEvidence[0].State != "conflict" || !reflect.DeepEqual(after.ReceiptEvidence, replay.ReceiptEvidence) {
		t.Fatalf("receipt evidence for %s = %+v, %v", id, replay.ReceiptEvidence, err)
	}
}

// An unsuccessful return supplies a selectable failure, even when later valid
// receipts establish more about the same original operation.
func TestLateAdapterReceiptsAfterUnsuccessfulSettlement(t *testing.T) {
	s, path, id := committedAdapter(t)
	ctx := t.Context()
	w, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || w == nil {
		t.Fatal(w, err)
	}
	first := effects.Candidate{ID: id, Key: w.attempt.Prepared.Request.Key, Disposition: "http_error"}
	result, err := s.recordAdapterResult(ctx, w.attempt, first)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.settleAdapter(ctx, w.attempt, result); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, RunOptions{MaxSteps: 4, MaxDispatch: 4}); err != nil {
		t.Fatal(err)
	}
	settled := current(t, s, "agent")
	if settled.Projection.Status != "succeeded" || settled.Projection.Data["valid"] != publicschema.Bool(false) {
		t.Fatalf("expected a consumed unsuccessful result: %+v", settled)
	}
	for _, tc := range []struct {
		answer, state string
		rows          int
	}{{"one", "known", 1}, {"one", "known", 1}, {"two", "conflict", 2}} {
		offer := effects.Candidate{ID: id, Key: first.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text(tc.answer)}}
		if _, err := s.recordAdapterResult(ctx, w.attempt, offer); !errors.Is(err, ErrConflict) {
			t.Fatalf("late offer: %v", err)
		}
		replayed, err := Replay(ctx, path, DefaultLimits())
		if err != nil || len(replayed.ReceiptEvidence) != 1 {
			t.Fatalf("late offer did not survive replay: %+v, %v", replayed.ReceiptEvidence, err)
		}
		ev := replayed.ReceiptEvidence[0]
		if ev.IntentID != id || ev.State != tc.state || count(t, s, "receipt_conflicts") != tc.rows {
			t.Fatalf("after %q: %+v, want %s and %d forensic rows", tc.answer, ev, tc.state, tc.rows)
		}
		if tc.state == "known" {
			var digest string
			if err := s.conn.QueryRowContext(ctx, "SELECT offer_digest FROM receipt_conflicts WHERE intent_id=?", id).Scan(&digest); err != nil || ev.ReceiptDigest != digest {
				t.Fatalf("known receipt digest = %q, want %q: %v", ev.ReceiptDigest, digest, err)
			}
		} else if ev.ReceiptDigest != "" {
			t.Fatal("conflict projection chose a receipt")
		}
	}
	if !reflect.DeepEqual(current(t, s, "agent"), settled) || count(t, s, "settlements") != 1 || count(t, s, "observations") != 1 {
		t.Fatal("forensic receipts changed an already settled continuation")
	}
}
