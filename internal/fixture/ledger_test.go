package fixture

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/adapters/receiptquery"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func queryForLedger(ledger, id, profile string) receiptquery.Query {
	k := func(id string) string { b, _ := json.Marshal([]any{2, strings.Repeat("4", 64), id}); return string(b) }
	q := receiptquery.Query{QueryID: "read-" + id, QueryKey: k("read-" + id), TargetID: id, TargetKey: k(id), LedgerID: ledger, Ordinal: 1}
	raw, _ := json.Marshal(Request{Key: k(id), Profile: profile, Delta: 3})
	q.Wire = " " + string(raw) + "\n"
	q.WireSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(q.Wire)))
	return q
}
func TestIndependentLedgerStoresWireAndSnapshotWithoutBusinessMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.db")
	world, err := OpenLedger(t.Context(), path, "first")
	if err != nil {
		t.Fatal(err)
	}
	q := queryForLedger(world.LedgerID(), "original", "idempotent")
	before, err := world.Lookup(t.Context(), q)
	if err != nil || before.Status != "not_found" || before.Count != 0 {
		t.Fatal(before, err)
	}
	if _, err = world.ApplyWire(t.Context(), q.Wire); err != nil {
		t.Fatal(err)
	}
	if _, err = world.ApplyWire(t.Context(), q.Wire); err != nil {
		t.Fatal(err)
	}
	plain := queryForLedger(world.LedgerID(), "other", "non-idempotent")
	for i := 0; i < 2; i++ {
		if _, err = world.ApplyWire(t.Context(), plain.Wire); err != nil {
			t.Fatal(err)
		}
	}
	if err = world.Close(); err != nil {
		t.Fatal(err)
	}
	if legacy, err := Open(t.Context(), path, "wrong-format"); err == nil {
		legacy.Close()
		t.Fatal("opened ledger as legacy")
	}
	world, err = OpenLedger(t.Context(), path, "second")
	if err != nil {
		t.Fatal(err)
	}
	defer world.Close()
	for _, v := range []struct {
		q      receiptquery.Query
		status string
		count  int64
	}{{q, "found", 1}, {plain, "conflict", 2}} {
		s, err := world.Lookup(t.Context(), v.q)
		if err != nil || s.Status != v.status || s.Count != v.count || s.Number <= before.Number {
			t.Fatal(s, err)
		}
		if v.status == "found" && (s.AppliedWire != q.Wire || s.Receipt == nil || s.Receipt.Total != 3 || s.Receipt.Process != "first") {
			t.Fatal(s)
		}
		if v.status == "conflict" && s.Receipt != nil {
			t.Fatal(s)
		}
	}
	changed := q
	changed.Wire = strings.TrimSpace(q.Wire)
	changed.WireSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(changed.Wire)))
	if _, err := world.ApplyWire(t.Context(), changed.Wire); err != ErrConflict {
		t.Fatal(err)
	}
	s, err := world.Lookup(t.Context(), changed)
	if err != nil || s.Status != "conflict" || s.Count != 1 || s.Receipt != nil {
		t.Fatal(s, err)
	}
	snap, err := Inspect(t.Context(), path)
	if err != nil || len(snap.Applications) != 3 || snap.Contacts != 4 || snap.Total != 9 {
		t.Fatal(snap, err)
	}
}

func TestLegacyResourceCannotAcquireLedgerWireBindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	world, err := Open(t.Context(), path, "old")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = world.Apply(t.Context(), Request{Key: "old-key", Profile: "idempotent", Delta: 1}); err != nil {
		t.Fatal(err)
	}
	if err = world.Close(); err != nil {
		t.Fatal(err)
	}
	if mistaken, err := OpenLedger(t.Context(), path, "new"); err == nil {
		mistaken.Close()
		t.Fatal("old re-encoded bytes treated as original wire")
	}
	s, err := Inspect(t.Context(), path)
	if err != nil || s.Total != 1 || s.LedgerID != "" || s.Contacts != 1 {
		t.Fatal(s, err)
	}
}

// Each independently committed application has the same inputs, process/PID,
// and previous business history. Only the observed branch services queries.
func TestCoupledQueriesPreserveEveryBusinessRow(t *testing.T) {
	open := func() *Resource {
		r, err := OpenLedger(t.Context(), filepath.Join(t.TempDir(), "world.sqlite"), "coupled")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { r.Close() })
		return r
	}
	observed, control := open(), open()
	q := queryForLedger(observed.LedgerID(), "original", "non-idempotent")
	peer := queryForLedger(observed.LedgerID(), "peer", "idempotent")
	applyBoth := func(wire string) {
		t.Helper()
		for _, r := range []*Resource{observed, control} {
			if _, err := r.ApplyWire(t.Context(), wire); err != nil {
				t.Fatal(err)
			}
		}
	}
	business := func(r *Resource) map[string][][]any {
		t.Helper()
		out := make(map[string][][]any)
		for _, table := range []string{"keys", "applications", "contacts", "application_wires"} {
			rows, err := r.conn.QueryContext(t.Context(), "SELECT * FROM "+table+" ORDER BY rowid")
			if err != nil {
				t.Fatal(err)
			}
			cols, err := rows.Columns()
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				v := make([]any, len(cols))
				dest := make([]any, len(cols))
				for i := range v {
					dest[i] = &v[i]
				}
				if err := rows.Scan(dest...); err != nil {
					t.Fatal(err)
				}
				out[table] = append(out[table], v)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if err := rows.Close(); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	for i, step := range []struct{ wire, status string }{{"", "not_found"}, {peer.Wire, "not_found"}, {q.Wire, "found"}, {peer.Wire, "found"}, {q.Wire, "conflict"}} {
		if step.wire != "" {
			applyBoth(step.wire)
		}
		s, err := observed.Lookup(t.Context(), q)
		if err != nil || s.Status != step.status || s.Number != int64(i+1) {
			t.Fatal(s, err)
		}
		if !reflect.DeepEqual(business(observed), business(control)) {
			t.Fatalf("business projections diverge after %s at %d", step.status, i)
		}
	}
}
