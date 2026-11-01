package fixture

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JakeFAU/alan/adapters/receiptquery"
)

// The opt-in format commits its identity and all tables atomically at creation.
// Reopening an existing ledger never regenerates the identity.
func (r *Resource) openLedger(ctx context.Context) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var tables int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type='table'").Scan(&tables); err != nil {
		return err
	}
	if tables == 0 {
		const ddl = `
 CREATE TABLE metadata(version INTEGER NOT NULL);
 INSERT INTO metadata VALUES(2);
 CREATE TABLE ledger_identity(singleton INTEGER PRIMARY KEY CHECK(singleton=1),id TEXT NOT NULL);
 CREATE TABLE keys(key TEXT PRIMARY KEY,profile TEXT NOT NULL,delta INTEGER NOT NULL,descriptor BLOB NOT NULL);
 CREATE TABLE applications(number INTEGER PRIMARY KEY,receipt BLOB NOT NULL);
 CREATE TABLE contacts(number INTEGER PRIMARY KEY,request BLOB NOT NULL);
 CREATE TABLE application_wires(number INTEGER PRIMARY KEY REFERENCES applications(number),key TEXT NOT NULL REFERENCES keys(key),wire BLOB NOT NULL);
 CREATE INDEX application_wires_key ON application_wires(key);
 CREATE TABLE receipt_reads(number INTEGER PRIMARY KEY,request BLOB NOT NULL,response BLOB NOT NULL);`
		if _, err = tx.ExecContext(ctx, ddl); err != nil {
			return err
		}
		var id [32]byte
		if _, err = rand.Read(id[:]); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO ledger_identity VALUES(1,?)", hex.EncodeToString(id[:])); err != nil {
			return err
		}
	}
	var version, count int
	if err = tx.QueryRowContext(ctx, "SELECT version,(SELECT count(*) FROM metadata) FROM metadata").Scan(&version, &count); err != nil {
		return err
	}
	if version != 2 || count != 1 {
		return errors.New("unsupported receipt ledger version")
	}
	if err = tx.QueryRowContext(ctx, "SELECT id FROM ledger_identity WHERE singleton=1").Scan(&r.ledgerID); err != nil {
		return err
	}
	b, err := hex.DecodeString(r.ledgerID)
	if err != nil || len(b) != 32 || hex.EncodeToString(b) != r.ledgerID {
		return errors.New("invalid receipt ledger identity")
	}
	return tx.Commit()
}
func (r *Resource) LedgerID() string { return r.ledgerID }

// ApplyWire re-decodes the exact bounded bytes that the business commit binds.
func (r *Resource) ApplyWire(ctx context.Context, wire string) (Receipt, error) {
	if r.ledgerID == "" {
		return Receipt{}, errors.New("legacy resource has no wire ledger")
	}
	m, err := receiptquery.DecodeMutation(wire)
	if err != nil {
		return Receipt{}, err
	}
	return r.apply(ctx, Request(m), []byte(wire))
}

// Lookup serializes the observed committed application set and its distinct
// audit insert. There are no writes to keys, applications or business contacts.
func (r *Resource) Lookup(ctx context.Context, q receiptquery.Query) (receiptquery.Snapshot, error) {
	var s receiptquery.Snapshot
	if err := q.Check(); err != nil {
		return s, err
	}
	if q.LedgerID != r.ledgerID {
		return s, errors.New("different ledger")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return s, err
	}
	defer tx.Rollback()
	s.Query = q
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM application_wires WHERE key=?", q.TargetKey).Scan(&s.Count); err != nil {
		return s, err
	}
	s.Status = "conflict"
	switch s.Count {
	case 0:
		s.Status = "not_found"
	case 1:
		var wire, receipt []byte
		if err = tx.QueryRowContext(ctx, `SELECT w.wire,a.receipt FROM application_wires w JOIN applications a ON a.number=w.number WHERE w.key=?`, q.TargetKey).Scan(&wire, &receipt); err != nil {
			return s, err
		}
		if string(wire) == q.Wire {
			var original receiptquery.Receipt
			if err = json.Unmarshal(receipt, &original); err != nil {
				return s, err
			}
			s.Status = "found"
			s.AppliedWire = string(wire)
			s.Receipt = &original
		}
	}
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(number),0)+1 FROM receipt_reads").Scan(&s.Number); err != nil {
		return s, err
	}
	response, err := json.Marshal(s)
	if err != nil || len(response) > receiptquery.MaxSnapshotBytes {
		return s, fmt.Errorf("snapshot exceeds protocol bound")
	}
	request, _ := json.Marshal(q)
	if _, err = tx.ExecContext(ctx, "INSERT INTO receipt_reads(number,request,response) VALUES(?,?,?)", s.Number, request, response); err != nil {
		return s, err
	}
	return s, tx.Commit()
}
