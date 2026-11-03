package runtime_test

import (
	"database/sql"
	"testing"

	rt "github.com/JakeFAU/alan/runtime"
)

// An unsupported journal mode must be rejected before a writable SQLite
// connection can convert the main header back to WAL.
func TestCausalRecoveryRejectsUnsupportedJournalBeforeOpeningWriter(t *testing.T) {
	path := recoveryCopy(t, "present")
	ctx, limits := t.Context(), rt.DefaultLimits()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode=DELETE").Scan(&mode); err != nil || mode != "delete" {
		t.Fatal(mode, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := recoveryImage(t, path)
	if _, err := rt.InspectExecution(ctx, path, limits); err == nil {
		t.Error("inspection accepted unsupported journal")
	}
	if _, err := rt.Replay(ctx, path, limits); err == nil {
		t.Error("Replay accepted unsupported journal")
	}
	owner, err := rt.Open(ctx, path, limits)
	if owner != nil {
		owner.Close()
	}
	if owner != nil || err == nil || recoveryImage(t, path) != before {
		t.Fatal("Open converted an unsupported journal or changed its contents", err)
	}
}
