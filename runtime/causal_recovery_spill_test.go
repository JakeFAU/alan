package runtime_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestCausalRecoveryFailedRenewalPreservesEntireWAL(t *testing.T) {
	ctx, limits := t.Context(), rt.DefaultLimits()
	artifact, err := author.Compile(author.Definition{
		Inputs: schema.Record{"payload": {Kind: "string", MaxBytes: 1000}},
		Steps:  []author.Step{author.Success()},
	})
	if err != nil {
		t.Fatal(err)
	}
	const count = 768
	worker := author.Instance{
		Artifact: artifact, Inputs: map[string]schema.Value{"payload": schema.Text(strings.Repeat("x", 1000))},
	}
	app := author.Application{"zz-last": worker}
	for i := range count - 1 {
		app[fmt.Sprintf("worker-%03d", i)] = worker
	}
	p, err := app.Provision()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.sqlite")
	store, err := rt.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// The renewal touches more leaf pages than fit in the default SQLite cache.
	// Abort the last update, after all earlier open runs staged their renewals.
	// Keep the connection open so last-close checkpointing cannot hide a spill.
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER reject_last_renewal BEFORE UPDATE OF epoch ON runs
		WHEN NEW.subject='zz-last' BEGIN SELECT raise(ABORT,'last renewal rejected'); END;`); err != nil {
		t.Fatal(err)
	}
	before := recoveryImage(t, path)
	store, err = rt.Open(ctx, path, limits)
	if store != nil {
		store.Close()
	}
	if store != nil || err == nil || !strings.Contains(err.Error(), "last renewal rejected") {
		t.Fatalf("Open returned an owner or missed the final renewal: %v", err)
	}
	var unchanged int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM runs WHERE epoch=1").Scan(&unchanged); err != nil || unchanged != count {
		t.Fatalf("rolled-back runs = %d, want %d: %v", unchanged, count, err)
	}
	if after := recoveryImage(t, path); after != before {
		t.Fatalf("failed renewal changed database or WAL bytes: before %x after %x", before, after)
	}
}
