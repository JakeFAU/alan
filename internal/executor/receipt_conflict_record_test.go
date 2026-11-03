package executor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/mattn/go-sqlite3"
)

func TestConcurrentDispatchRetainsConflictingReceipts(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	var contacts atomic.Int32
	requests := make(chan resource.Request, 2)
	mutations := make(chan int64, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contacts.Add(1) > 2 {
			http.Error(w, "unexpected extra contact", http.StatusConflict)
			return
		}
		var req resource.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		requests <- req
		select {
		case n := <-mutations:
			receipt := receiptFor(req)
			receipt.Mutation, receipt.Total = n, n*req.Delta
			if err := json.NewEncoder(w).Encode(receipt); err != nil {
				t.Error(err)
			}
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	s, path := messageStore(t, externalProvision(t, server.URL, "idempotent"))
	state, err := s.Commit(ctx, current(t, s, "client").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := state.Projection.Data["pending"].Text
	type result struct {
		settled bool
		err     error
	}
	done := make(chan result, 2)
	for range 2 {
		go func() {
			settled, err := s.DispatchExternal(ctx, id)
			done <- result{settled, err}
		}()
	}
	// Both calls reached the endpoint for the same original before either
	// response. Return different receipts in a controlled commit order.
	first, err := receiveRequest(ctx, requests)
	if err != nil {
		t.Fatal(err)
	}
	second, err := receiveRequest(ctx, requests)
	if err != nil || second != first || first.Key != ExternalKey(s.Namespace(), id) {
		t.Fatalf("dispatch requests = %+v and %+v, %v", first, second, err)
	}
	for _, n := range []int64{1, 2} {
		mutations <- n
		select {
		case got := <-done:
			if (n == 1 && (!got.settled || got.err != nil)) || (n == 2 && (got.settled || !errors.Is(got.err, ErrConflict))) {
				t.Fatalf("return %d: settled=%v err=%v", n, got.settled, got.err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	o, err := s.externalOutcome(ctx, id)
	if err != nil || o.Receipt == nil || o.Receipt.Mutation != 1 || count(t, s, "settlements") != 1 || count(t, s, "receipt_conflicts") != 1 {
		t.Fatalf("retained winner and settlement: %+v, %v", o, err)
	}
	replayed, err := Replay(ctx, path, DefaultLimits())
	if err != nil || len(replayed.ReceiptEvidence) != 1 || replayed.ReceiptEvidence[0].IntentID != id || replayed.ReceiptEvidence[0].State != "conflict" || contacts.Load() != 2 {
		t.Fatalf("replay evidence = %+v, err=%v, contacts=%d", replayed.ReceiptEvidence, err, contacts.Load())
	}
}

func recordedRawReceipt(t *testing.T) (*Store, string, ExternalAttempt, resource.Receipt) {
	t.Helper()
	ctx := t.Context()
	s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
	state, err := s.Commit(ctx, current(t, s, "client").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := s.prepareExternal(ctx, state.Projection.Data["pending"].Text)
	if err != nil {
		t.Fatal(err)
	}
	first := receiptFor(a.Request)
	if err := s.recordExternalReceipt(ctx, a, first); err != nil {
		t.Fatal(err)
	}
	return s, path, a, first
}

func TestReceiptConflictCommitFailureRollsBackPromotion(t *testing.T) {
	ctx := t.Context()
	s, path, a, first := recordedRawReceipt(t)
	before, err := Inspect(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	hook := func(f func() int) {
		t.Helper()
		if err := s.conn.Raw(func(c any) error {
			c.(*sqlite3.SQLiteConn).RegisterCommitHook(f)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	loser := first
	loser.Total++
	hook(func() int { return 1 })
	err = s.recordExternalReceipt(ctx, a, loser)
	hook(nil)
	var commitErr sqlite3.Error
	if errors.Is(err, ErrConflict) || !errors.As(err, &commitErr) || commitErr.ExtendedCode != sqlite3.ErrConstraintCommitHook {
		t.Fatalf("failed commit returned %v, want the SQLite commit hook error", err)
	}
	after, err := Inspect(ctx, path, DefaultLimits())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed promotion changed the journal: %v", err)
	}
	if err := s.recordExternalReceipt(ctx, a, loser); !errors.Is(err, ErrConflict) || count(t, s, "receipt_conflicts") != 1 {
		t.Fatalf("retry after rollback: %v", err)
	}
}

func TestForensicRawReceiptCannotSettle(t *testing.T) {
	ctx := t.Context()
	s, _, a, first := recordedRawReceipt(t)
	loser := first
	loser.Total++
	if err := s.recordExternalReceipt(ctx, a, loser); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	foreign := a
	foreign.AuthorizationID += "/other"
	for _, forged := range []ExternalOutcome{
		{Attempt: a, Disposition: "applied", Receipt: &loser},
		{Attempt: a, Disposition: "source_denied"},
		{Attempt: foreign, Disposition: "applied", Receipt: &first},
	} {
		if err := s.settleExternal(ctx, forged); !errors.Is(err, ErrConflict) {
			t.Fatalf("settled %+v: %v", forged, err)
		}
	}
	if count(t, s, "settlements") != 0 || count(t, s, "observations") != 0 {
		t.Fatal("rejected settlement delivered an observation")
	}
	if _, err := s.Run(ctx, RunOptions{MaxSteps: 8, MaxDispatch: 8}); err != nil {
		t.Fatal(err)
	}
	if selected := current(t, s, "client").Projection.Data["total"]; selected != Int(first.Total) {
		t.Fatalf("selected total = %v, want %d", selected, first.Total)
	}
}

func TestForensicReceiptRequiresOriginalAuthorityAndSchema(t *testing.T) {
	for _, fault := range []string{"foreign_authority", "ignored_promotion"} {
		t.Run(fault, func(t *testing.T) {
			ctx := t.Context()
			s, _, a, first := recordedRawReceipt(t)
			switch fault {
			case "foreign_authority":
				auth, err := loadAuthorization(ctx, s.conn, a.Intent.ID)
				if err != nil {
					t.Fatal(err)
				}
				auth.Source.Run.Subject += "/other"
				raw, err := json.Marshal(auth)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.conn.ExecContext(ctx, "UPDATE attempts SET result=? WHERE id=?", raw, auth.ID); err != nil {
					t.Fatal(err)
				}
			case "ignored_promotion":
				if _, err := s.conn.ExecContext(ctx, "CREATE TEMP TRIGGER skip BEFORE UPDATE ON metadata BEGIN SELECT RAISE(IGNORE); END"); err != nil {
					t.Fatal(err)
				}
			}
			loser := first
			loser.Total++
			if err := s.recordExternalReceipt(ctx, a, loser); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("acknowledged damaged binding: %v", err)
			}
			if v, err := storedSchemaNumber(ctx, s.conn); err != nil || v != SchemaVersion {
				t.Fatalf("failed offer promoted the store: %d, %v", v, err)
			}
		})
	}
}

func TestForensicInsertMustRetainExactBytes(t *testing.T) {
	ctx := t.Context()
	s, path, a, first := recordedRawReceipt(t)
	lose := func(total int64) error {
		offer := first
		offer.Total = total
		return s.recordExternalReceipt(ctx, a, offer)
	}
	if err := lose(10); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	for name, trigger := range map[string]string{
		"ignored":   "BEFORE INSERT ON receipt_conflicts BEGIN SELECT RAISE(IGNORE); END",
		"discarded": "AFTER INSERT ON receipt_conflicts BEGIN DELETE FROM receipt_conflicts WHERE rowid=NEW.rowid; END",
		"rewritten": "AFTER INSERT ON receipt_conflicts BEGIN UPDATE receipt_conflicts SET descriptor='{}' WHERE rowid=NEW.rowid; END",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := s.conn.ExecContext(ctx, "CREATE TEMP TRIGGER injected "+trigger); err != nil {
				t.Fatal(err)
			}
			err := lose(15)
			if err == nil || errors.Is(err, ErrConflict) {
				t.Fatalf("acknowledged an offer absent from the journal: %v", err)
			}
			if _, err := s.conn.ExecContext(ctx, "DROP TRIGGER injected"); err != nil {
				t.Fatal(err)
			}
			if _, err := Replay(ctx, path, DefaultLimits()); err != nil || count(t, s, "receipt_conflicts") != 1 {
				t.Fatalf("failed insert changed prior evidence: %v", err)
			}
		})
	}
}
