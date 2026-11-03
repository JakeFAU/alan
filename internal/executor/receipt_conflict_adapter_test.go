//go:build integration

package executor

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

func TestAdapterReceiptEvidenceRejectsRehashedInvalidOffer(t *testing.T) {
	for _, damage := range []string{"foreign key", "unsuccessful", "fabricated values", "read proof"} {
		t.Run(damage, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			w, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
			if err != nil || w == nil {
				t.Fatal(w, err)
			}
			first := transportAdapter(t.Context(), *w).candidate
			if _, err = s.recordAdapterResult(t.Context(), w.attempt, first); err != nil {
				t.Fatal(err)
			}
			other := effects.Copy(first)
			other.Payload["answer"] = Text("later")
			offer, err := s.recordAdapterResult(t.Context(), w.attempt, other)
			if !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			switch damage {
			case "foreign key":
				offer.Candidate.Key += "/foreign"
			case "unsuccessful":
				offer.Candidate.Disposition = "http_error"
				offer.Values, offer.Disposition = effects.Result(w.attempt.Prepared.Request.Output.Record, offer.Candidate)
			case "fabricated values":
				offer.Values["answer"] = Text("never offered")
			case "read proof":
				offer.Read = &receiptReadDecision{Ordinal: 1, Adoption: "adopted"}
			}
			offer.Digest = adapterResultDigest(offer)
			raw, err := json.Marshal(offer)
			if err != nil {
				t.Fatal(err)
			}
			replaceReceiptOffer(t, s, raw)
			assertReceiptEvidenceRejected(t, s, path)
		})
	}
}

// A genuine receipt-query import provides both a read proof and a reconstructed
// original. Neither can anchor the baseline transport recorder's side journal.
func TestReadAndReconstructedResultsCannotAnchorBaselineOffers(t *testing.T) {
	for _, role := range []string{"query", "reconstructed original"} {
		t.Run(role, func(t *testing.T) {
			s, path := openReceiptCut(t, "adopted-handoff")
			ctx := t.Context()
			id := `["resolver",1,4,0]`
			read, err := loadAdapterResult(ctx, s.conn, id)
			if err != nil || read.Read == nil || read.Read.Adoption != "adopted" {
				t.Fatal(read, err)
			}
			payload := effects.Copy(read.Candidate.Payload)
			if role == "reconstructed original" {
				id = payload["target"].Text
				payload = map[string]Value{"total": Int(3), "mutation": Int(2)}
			}
			a, err := loadAdapterAttempt(ctx, s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			candidate := effects.Candidate{ID: id, Key: a.Prepared.Request.Key, Disposition: "validated", Payload: payload}
			if _, err := s.recordAdapterResult(ctx, a, candidate); !errors.Is(err, ErrConflict) {
				t.Fatalf("baseline recording on %s: %v", role, err)
			}
			before, err := Inspect(ctx, path, DefaultLimits())
			if err != nil || before.Tables["receipt_conflicts"] != nil {
				t.Fatalf("ineligible offer promoted the historical store: %v", err)
			}
			values, disposition := effects.Result(a.Prepared.Request.Output.Record, candidate)
			if disposition != "validated" {
				t.Fatal("test offer does not satisfy the output schema")
			}
			offer := AdapterResult{Candidate: candidate, Values: values, Disposition: disposition}
			offer.Digest = adapterResultDigest(offer)
			raw, err := json.Marshal(offer)
			if err != nil {
				t.Fatal(err)
			}
			var first []byte
			if err := s.conn.QueryRowContext(ctx, "SELECT descriptor FROM adapter_results WHERE intent_id=?", id).Scan(&first); err != nil {
				t.Fatal(err)
			}
			// Bypass the recorder and supply matching hashes on this disposable cut.
			tx, err := s.conn.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if err = ensureReceiptConflictSchema(ctx, tx); err != nil {
				t.Fatal(err)
			}
			_, err = tx.ExecContext(ctx, "INSERT INTO receipt_conflicts VALUES(?,?,?,?)", id, fmt.Sprintf("%x", sha256.Sum256(first)), fmt.Sprintf("%x", sha256.Sum256(raw)), raw)
			if err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			assertReceiptEvidenceRejected(t, s, path)
		})
	}
}
