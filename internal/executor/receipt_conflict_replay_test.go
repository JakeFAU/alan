package executor

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestUnversionedReceiptConflictTableRejected(t *testing.T) {
	s, path := messageStore(t, inputFixture(t))
	if _, err := s.conn.ExecContext(t.Context(), "CREATE TABLE receipt_conflicts(intent_id TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Errorf("Replay with unversioned conflict table: %v", err)
	}
	if _, err := Inspect(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Errorf("Inspect with unversioned conflict table: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(t.Context(), path, DefaultLimits())
	if owner != nil {
		owner.Close()
	}
	if !errors.Is(err, ErrVersion) {
		t.Errorf("Open with unversioned conflict table: %v", err)
	}
}

func TestRawReceiptEvidenceRejectsDamagedOrMisboundOffers(t *testing.T) {
	for _, damage := range []string{"missing table", "missing offer", "missing winner", "missing grant", "foreign authorization source", "wrong authorization request", "losing receipt settled", "wrong winner digest", "wrong offer digest", "winner offered again", "foreign key", "invalid mutation", "noncanonical encoding"} {
		t.Run(damage, func(t *testing.T) {
			s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
			run, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), run.Projection.Data["pending"].Text)
			if err != nil {
				t.Fatal(err)
			}
			first := receiptFor(a.Request)
			if err = s.recordExternalReceipt(t.Context(), a, first); err != nil {
				t.Fatal(err)
			}
			other := first
			other.Total++
			if err = s.recordExternalReceipt(t.Context(), a, other); !errors.Is(err, ErrConflict) {
				t.Fatalf("second receipt: %v", err)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatalf("uncorrupted ledger: %v", err)
			}
			var query string
			switch damage {
			case "missing table":
				query = "DROP TABLE receipt_conflicts"
			case "missing offer":
				query = "DELETE FROM receipt_conflicts"
			case "missing winner":
				query = "DELETE FROM external_receipts"
			case "missing grant":
				query = "DELETE FROM permissions WHERE kind='external'"
			case "foreign authorization source":
				query = `UPDATE attempts SET result=CAST(json_set(result,'$.Source.Run.Subject','foreign') AS BLOB)`
			case "wrong authorization request":
				query = "UPDATE attempts SET descriptor='{}'"
			case "losing receipt settled":
				outcome := ExternalOutcome{Attempt: a, Disposition: "applied", Receipt: &first}
				if err := s.settleExternal(t.Context(), outcome); err != nil {
					t.Fatal(err)
				}
				outcome.Receipt = &other
				raw, err := json.Marshal(outcome)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.conn.ExecContext(t.Context(), "UPDATE settlements SET descriptor=? WHERE intent_id=?", raw, a.Intent.ID); err != nil {
					t.Fatal(err)
				}
			case "wrong winner digest":
				query = "UPDATE receipt_conflicts SET winner_digest='foreign'"
			case "wrong offer digest":
				query = "UPDATE receipt_conflicts SET offer_digest='foreign'"
			case "winner offered again", "foreign key", "invalid mutation", "noncanonical encoding":
				// Rehash the forgery. A matching checksum cannot authenticate
				// the original operation or make its first receipt a second one.
				forged := other
				switch damage {
				case "winner offered again":
					forged = first
				case "foreign key":
					forged.Key += "/other"
				case "invalid mutation":
					forged.Mutation = 0
				}
				raw, e := json.Marshal(forged)
				if e != nil {
					t.Fatal(e)
				}
				if damage == "noncanonical encoding" {
					raw = append(raw, '\n')
				}
				replaceReceiptOffer(t, s, raw)
			}
			if query != "" {
				if _, err := s.conn.ExecContext(t.Context(), query); err != nil {
					t.Fatal(err)
				}
			}
			assertReceiptEvidenceRejected(t, s, path)
		})
	}
}

func replaceReceiptOffer(t *testing.T, s *Store, raw []byte) {
	t.Helper()
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE receipt_conflicts SET descriptor=?,offer_digest=?", raw, digest); err != nil {
		t.Fatal(err)
	}
}

func assertReceiptEvidenceRejected(t *testing.T, s *Store, path string) {
	t.Helper()
	if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
		t.Error("Replay accepted damaged receipt evidence")
	}
	if inspection, err := Inspect(t.Context(), path, DefaultLimits()); err == nil || len(inspection.ReceiptEvidence) != 0 || inspection.Tables["receipt_conflicts"] != nil {
		t.Errorf("Inspect exported receipt evidence for a damaged ledger: %+v, %v", inspection.ReceiptEvidence, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	owner, err := Open(t.Context(), path, DefaultLimits())
	if err == nil {
		owner.Close()
		t.Fatal("Open accepted damaged receipt evidence")
	}
}
