//go:build integration

package executor

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

// A deployment that opts into read accounting must never run with the legacy
// schema, even when its grant list is empty (deny every target).
func TestPhase2CExplicitProfileNeverSilentlyDowngrades(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	p.Composition = &c
	if err := json.Unmarshal([]byte(`{"Reconciliation":{"Grants":[]}}`), &p); err != nil {
		t.Fatal(err)
	}
	s, err := Create(t.Context(), filepath.Join(t.TempDir(), "store.sqlite"), p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	version, err := schemaNumber(t.Context(), s.conn)
	if err != nil || version != 5 {
		t.Fatalf("explicit receipt-read profile opened as schema %d, error %v", version, err)
	}
}

// An empty opted-in deployment has no read result to reveal a version
// downgrade. Its durable capability tables must still exclude legacy opening.
func TestPhase2CRejectDowngradeBeforeAnyRead(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	p.Composition = &c
	p.Reconciliation = &ReconciliationProfile{}
	path := filepath.Join(t.TempDir(), "store.sqlite")
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "UPDATE metadata SET schema_version=4; PRAGMA user_version=4"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	old, err := Open(t.Context(), path, DefaultLimits())
	if old != nil {
		old.Close()
	}
	if !errors.Is(err, ErrVersion) {
		t.Fatalf("downgraded profile reopened before first read: %v", err)
	}
}
