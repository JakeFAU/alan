package fixture

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestDurableResourceRetainsKeyParametersAndOriginalReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.sqlite")
	r, err := Open(context.Background(), path, "process-one")
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Key: "operation", Profile: "idempotent", Delta: 7}
	first, err := r.Apply(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 7 || first.Mutation != 1 || first.Process != "process-one" {
		t.Fatal(first)
	}
	r.Close()
	r, err = Open(t.Context(), path, "process-two")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	second, err := r.Apply(t.Context(), request)
	if err != nil || second != first {
		t.Fatalf("changed original receipt: %+v %v", second, err)
	}
	request.Delta = 8
	if _, err = r.Apply(t.Context(), request); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	snapshot, err := Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Applications) != 1 || snapshot.Total != 7 {
		t.Fatal(snapshot)
	}
}
func TestNonIdempotentUnsafeRepetitionIsTwoApplications(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.sqlite")
	r, err := Open(t.Context(), path, "unsafe-experiment")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	request := Request{Key: "same-key", Profile: "non-idempotent", Delta: 3}
	first, err := r.Apply(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Apply(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Mutation != 1 || second.Mutation != 2 || second.Total != 6 {
		t.Fatalf("unsafe repetition not observable: %+v %+v", first, second)
	}
}

func TestResourceReadOnlyInspectionRejectsMissingAndUnsupportedStores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.sqlite")
	if _, err := Inspect(t.Context(), path); err == nil {
		t.Fatal("missing resource inspection succeeded")
	}
	r, err := Open(t.Context(), path, "version-test")
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE metadata SET version=99"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err = Inspect(t.Context(), path); err == nil {
		t.Fatal("unsupported resource inspection succeeded")
	}
	if reopened, err := Open(t.Context(), path, "next"); err == nil {
		reopened.Close()
		t.Fatal("unsupported resource open succeeded")
	}
}

func TestResourceInspectionFailsClosedOnNonWALStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resource.sqlite")
	r, err := Open(t.Context(), path, "settings-test")
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err = Inspect(t.Context(), path); err == nil {
		t.Fatal("read-only resource inspection accepted non-WAL store")
	}
}
