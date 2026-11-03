package executor

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
)

// The first retained losing offer promotes the physical schema in the same
// transaction. Versions 6, 7 and 8 retain the protocols of 3, 4 and 5.
const receiptConflictSchemaOffset = 3

const dispatchAdmissionSchema = 9
const dispatchAdmissionConflictSchema = 10

const continuationSchema = 11
const continuationConflictSchema = 12

func hasContinuationSchema(version int) bool {
	return version == continuationSchema || version == continuationConflictSchema
}

func hasDispatchAdmissionSchema(version int) bool {
	return hasContinuationSchema(version) || version == dispatchAdmissionSchema || version == dispatchAdmissionConflictSchema
}

func hasReceiptConflictSchema(version int) bool {
	switch version {
	case SchemaVersion + receiptConflictSchemaOffset,
		compositionSchemaVersion + receiptConflictSchemaOffset,
		reconciliationSchemaVersion + receiptConflictSchemaOffset, dispatchAdmissionConflictSchema, continuationConflictSchema:
		return true
	default:
		return false
	}
}

func protocolSchemaNumber(version int) int {
	if hasDispatchAdmissionSchema(version) {
		return compositionSchemaVersion
	}
	if hasReceiptConflictSchema(version) {
		return version - receiptConflictSchemaOffset
	}
	return version
}

func ensureReceiptConflictSchema(ctx context.Context, tx *sql.Tx) error {
	version, err := storedSchemaNumber(ctx, tx)
	if err != nil {
		return err
	}
	var userVersion int
	if err = tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&userVersion); err != nil {
		return err
	}
	if userVersion != version {
		return ErrVersion
	}
	if hasReceiptConflictSchema(version) {
		return nil
	}
	if version != SchemaVersion && !isCompositionSchema(protocolSchemaNumber(version)) {
		return ErrVersion
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE receipt_conflicts(
		intent_id TEXT NOT NULL REFERENCES intents(id),
		winner_digest TEXT NOT NULL,
		offer_digest TEXT NOT NULL,
		descriptor BLOB NOT NULL,
		PRIMARY KEY(intent_id,offer_digest))`); err != nil {
		return err
	}
	switch version {
	case dispatchAdmissionSchema:
		version = dispatchAdmissionConflictSchema
	case continuationSchema:
		version = continuationConflictSchema
	default:
		version += receiptConflictSchemaOffset
	}
	if _, err = tx.ExecContext(ctx, "UPDATE metadata SET schema_version=? WHERE singleton=1", version); err != nil {
		return err
	}
	promoted, err := storedSchemaNumber(ctx, tx)
	if err != nil {
		return err
	}
	if promoted != version {
		return corrupt("receipt evidence schema promotion was not retained")
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", version))
	return err
}

// Under the store's single-owner exclusion, bind the exact validated offer
// to the immutable first result. Only that first result is selectable.
func (s *Store) commitReceiptConflict(ctx context.Context, tx *sql.Tx, id string, winner, offer []byte) error {
	auth, err := validatedAuthorization(ctx, tx, identity("authorization", id))
	if err != nil {
		return err
	}
	if !auth.Allowed {
		return corrupt("conflicting receipt without original authorization")
	}
	if err := ensureReceiptConflictSchema(ctx, tx); err != nil {
		return err
	}
	winnerDigest, offerDigest := programHash(winner), programHash(offer)
	_, err = tx.ExecContext(ctx, `INSERT INTO receipt_conflicts(intent_id,winner_digest,offer_digest,descriptor)
		VALUES(?,?,?,?) ON CONFLICT(intent_id,offer_digest) DO NOTHING`, id, winnerDigest, offerDigest, offer)
	if err != nil {
		return err
	}
	// A successful statement or a matching digest alone cannot attest that
	// the exact bytes survived insertion. Read them back before committing.
	var retainedWinner string
	var retainedOffer []byte
	if err = tx.QueryRowContext(ctx, "SELECT winner_digest,descriptor FROM receipt_conflicts WHERE intent_id=? AND offer_digest=?", id, offerDigest).Scan(&retainedWinner, &retainedOffer); err != nil {
		return err
	}
	if retainedWinner != winnerDigest || !bytes.Equal(retainedOffer, offer) {
		return corrupt("retained conflicting receipt differs")
	}
	if err = s.commitOwnedTx(ctx, tx); err != nil {
		return err
	}
	return ErrConflict
}
