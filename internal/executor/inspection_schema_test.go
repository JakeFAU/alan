package executor

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestInspectMixedCaseReceiptJournal(t *testing.T) {
	ctx := t.Context()
	s, path, attempt, first := recordedRawReceipt(t)
	other := first
	other.Total++
	if err := s.recordExternalReceipt(ctx, attempt, other); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	// SQLite resolves both spellings to the same table. A second name is
	// required to change only the capitalization with ALTER TABLE.
	if _, err := s.conn.ExecContext(ctx, `ALTER TABLE receipt_conflicts RENAME TO held_receipts;
		ALTER TABLE held_receipts RENAME TO ReCeIpT_CoNfLiCtS;
		CREATE TRIGGER runs AFTER UPDATE ON runs BEGIN SELECT 1; END`); err != nil {
		t.Fatal(err)
	}
	cut, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	effect, ok := cut.Effect(attempt.Intent.ID)
	if !cut.Features.ConflictJournal || !ok || effect.ExternalReceipt == nil || *effect.ExternalReceipt != first ||
		len(effect.AdditionalReceipts) != 1 || effect.AdditionalReceipts[0].External == nil || *effect.AdditionalReceipts[0].External != other {
		t.Fatalf("missing original receipt or losing offer: %+v", effect)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE metadata SET schema_version=3; PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(ctx, path, DefaultLimits()); cut != nil || !errors.Is(err, ErrVersion) {
		t.Fatalf("downgrade with retained receipt evidence: %v, %v", cut, err)
	}
}

func TestInspectUnversionedMixedCaseObjects(t *testing.T) {
	for _, tc := range []struct{ name, ddl string }{
		{"receipt_view", "CREATE VIEW ReCeIpT_CoNfLiCtS AS SELECT 1 AS intent_id"},
		{"composition_table", "CREATE TABLE CoMpOsItIoN_FaCtS(id TEXT)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path := messageStore(t, inputFixture(t))
			if _, err := s.conn.ExecContext(t.Context(), tc.ddl); err != nil {
				t.Fatal(err)
			}
			if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrVersion) {
				t.Fatalf("unversioned %s: %v, %v", tc.name, cut, err)
			}
		})
	}
}

func TestInspectCompleteProfileProgramLimit(t *testing.T) {
	ctx := t.Context()
	p, c := invocationFixture(t, "succeeded", 1)
	template := c.Templates[0]
	for i := range 32 {
		template.Members[fmt.Sprintf("helper%d", i)] = template.Members["helper"]
	}
	replaceInvocationTemplate(t, &p, &c, template)
	path := filepath.Join(t.TempDir(), "profile.sqlite")
	s, err := createInvocationStore(ctx, path, p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var profileBytes, largestPart int
	err = s.conn.QueryRowContext(ctx, `SELECT length(descriptor),
		max((SELECT max(length(bytes)) FROM programs), (SELECT max(length(descriptor)) FROM composition_templates))
		FROM composition_profile`).Scan(&profileBytes, &largestPart)
	if err != nil || profileBytes <= largestPart {
		t.Fatalf("profile size %d, largest program or template %d: %v", profileBytes, largestPart, err)
	}
	limits := DefaultLimits()
	limits.MaxProgramBytes = profileBytes
	if _, err := InspectExecution(ctx, path, limits); err != nil {
		t.Fatalf("profile at byte limit: %v", err)
	}
	limits.MaxProgramBytes--
	if cut, err := InspectExecution(ctx, path, limits); cut != nil || !errors.Is(err, ErrLimit) {
		t.Fatalf("complete profile exceeds limit, individual parts fit: %v, %v", cut, err)
	}
}

func TestInspectInvocationWithoutCompositionProtocol(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "invocation.sqlite")
	p, profile := invocationFixture(t, "succeeded", 1)
	s, err := createInvocationStore(ctx, path, p, profile, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	run, err := s.Commit(ctx, current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	cut, err := InspectExecution(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := cut.Invocation(run.Projection.Data["pending"].Text); !ok || v.State != InvocationCommitted || !cut.Features.Composition {
		t.Fatalf("missing committed invocation: %+v", v)
	}
	// Leave the emitting program and its history intact. Even with every
	// composition table removed, the operation still requires this protocol.
	if _, err := s.conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		t.Fatal(err)
	}
	for _, table := range compositionInspectionTables {
		if _, err := s.conn.ExecContext(ctx, "DROP TABLE "+table); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE metadata SET schema_version=3; PRAGMA user_version=3; PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	if cut, err := InspectExecution(ctx, path, DefaultLimits()); cut != nil || !errors.Is(err, ErrVersion) {
		t.Fatalf("invocation in ordinary store: %v, %v", cut, err)
	}
}
