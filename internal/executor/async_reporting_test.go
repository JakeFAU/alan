package executor

import (
	"context"
	"encoding/json"
	"errors"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Receipt persistence and settlement are intentionally separate commits. A
// non-idempotent receipt surviving a settlement error must settle on reopen
// without another transport launch or business mutation.
func TestAsyncKnownReceiptSurvivesSettlementFailure(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		contacts.Add(1)
		if err := json.NewEncoder(w).Encode(receiptFor(req)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	s, path := messageStore(t, externalProvision(t, server.URL, "non-idempotent"))
	if _, err := s.conn.ExecContext(context.Background(), `CREATE TEMP TRIGGER final_review_reject_settlement BEFORE INSERT ON settlements BEGIN SELECT RAISE(ABORT,'review settlement failure'); END`); err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err == nil || !strings.Contains(err.Error(), "review settlement failure") || report.Quiescent || count(t, s, "external_receipts") != 1 || count(t, s, "settlements") != 0 {
		t.Fatalf("failure boundary: report=%+v err=%v", report, err)
	}
	if current(t, s, "client").Projection.Status != "open" || contacts.Load() != 1 {
		t.Fatal("unexpected source or transport")
	}
	if len(report.UnresolvedIntents) != 1 || len(report.PendingDeliveries) != 1 {
		t.Fatalf("unsettled delivery disappeared: %+v", report)
	}
	if len(report.UnknownExternal) != 0 {
		t.Fatalf("durable valid receipt is known applied; UnknownExternal=%v, UnresolvedIntents=%v, err=%v", report.UnknownExternal, report.UnresolvedIntents, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	report, err = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil || contacts.Load() != 1 || count(t, s, "settlements") != 1 || current(t, s, "client").Projection.Status != "succeeded" || len(report.UnresolvedIntents) != 0 {
		t.Fatalf("recovery: report=%+v err=%v contacts=%d", report, err, contacts.Load())
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != 1 {
		t.Fatal("replay dispatched")
	}
	t.Logf("reopen settled and consumed without another request; report=%+v contacts=%d", report, contacts.Load())
}

func TestAsyncRunReportRejectsMalformedRetainedReceipt(t *testing.T) {
	for _, raw := range []string{`{`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			s, _ := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "non-idempotent"))
			state, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			id := state.Projection.Data["pending"].Text
			attempt, _, err := s.prepareExternal(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			if err = s.recordExternalReceipt(t.Context(), attempt, receiptFor(attempt.Request)); err != nil {
				t.Fatal(err)
			}
			// Deliberate corrupt-record injection must not be treated as known or Unknown.
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE external_receipts SET descriptor=? WHERE intent_id=?", []byte(raw), id); err != nil {
				t.Fatal(err)
			}
			_, validationErr := externalOutcome(t.Context(), s.conn, id)
			if validationErr == nil {
				t.Fatal("invalid fixture receipt passed validation")
			}
			settlementErr := errors.New("settlement failed")
			report, err := s.finishRun(t.Context(), RunReport{}, false, settlementErr)
			if !errors.Is(err, settlementErr) || !strings.Contains(err.Error(), validationErr.Error()) || report.Quiescent || len(report.UnknownExternal) != 0 {
				t.Fatalf("malformed receipt validation hidden: report=%+v err=%v validation=%v", report, err, validationErr)
			}
		})
	}
}
