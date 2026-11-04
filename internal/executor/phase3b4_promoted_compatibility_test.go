//go:build integration

package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Ancillary compatibility only: the parent is an already completed genuine
// public K01/K02 Run. This appends deliberately supplied rejected conflict
// evidence through the existing internal writer on a disposable copy. It does
// not claim a public conflict-production schedule or invoke transport again.
func TestPhase3B4PromotedPublicParents(t *testing.T) {
	root := os.Getenv("ALAN_PHASE3B4_PUBLIC_PARENTS")
	out := os.Getenv("ALAN_PHASE3B4_PROMOTED_OUTPUT")
	if root == "" || out == "" {
		t.Skip("requires freshly exported exact public K01/K02 parents and fresh output root")
	}
	write := func(dir, name string, v any) {
		t.Helper()
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyDB := func(from, to string) map[string]string {
		t.Helper()
		hashes := map[string]string{}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			raw, err := os.ReadFile(from + suffix)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			hashes[suffix] = hex.EncodeToString(sum[:])
			if err = os.WriteFile(to+suffix, raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		return hashes
	}
	for _, name := range []string{"K01", "K02"} {
		t.Run(name, func(t *testing.T) {
			parents, err := filepath.Glob(filepath.Join(root, name+"-*", "final", "runtime.sqlite"))
			if err != nil || len(parents) != 1 {
				t.Fatal("need unique exact public parent", parents, err)
			}
			if err = os.MkdirAll(out, 0700); err != nil {
				t.Fatal(err)
			}
			dir, err := os.MkdirTemp(out, name+"-promoted-")
			if err != nil {
				t.Fatal(err)
			}
			db := filepath.Join(t.TempDir(), "copy.sqlite")
			hashes := copyDB(parents[0], db)
			write(dir, "parent.json", map[string]any{"Path": parents[0], "SHA256": hashes, "Scope": "ancillary existing-internal-writer compatibility; no public conflict-production schedule"})
			before, err := Inspect(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			write(dir, "parent-inspect.json", before)
			s, err := Open(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			opened, err := Inspect(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			write(dir, "opened-inspect.json", opened)
			id := `["a",1,3,0]`
			attempt, err := loadAdapterAttempt(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			first, err := loadAdapterResult(t.Context(), s.conn, id)
			if err != nil {
				t.Fatal(err)
			}
			candidate := first.Candidate
			candidate.Response = "distinct deliberately supplied rejected conflict evidence for exact public K parent"
			_, err = s.recordAdapterResult(t.Context(), attempt, candidate)
			if !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			after, err := Inspect(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			write(dir, "promoted-inspect.json", after)
			for table, rows := range opened.Tables {
				if table == "metadata" || table == "receipt_conflicts" {
					continue
				}
				if !reflect.DeepEqual(rows, after.Tables[table]) {
					t.Fatal("writer changed unrelated retained bytes", table)
				}
			}
			if len(after.Tables["receipt_conflicts"]) != 1 {
				t.Fatal("missing exact losing evidence")
			}
			typed, err := InspectExecution(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			want := 14
			if name == "K02" {
				want = 16
			}
			if typed.StorageVersion != want {
				t.Fatal(typed.StorageVersion)
			}
			write(dir, "typed-inspection.json", typed)
			e, ok := typed.Effect(id)
			if !ok || e.AdapterResult == nil || !reflect.DeepEqual(*e.AdapterResult, first) || e.Settlement == nil {
				t.Fatal("winner changed", e)
			}
			o, ok := typed.Observation(e.Settlement.ObservationID)
			if !ok || o.SelectedAt == nil {
				t.Fatal("public consumed winner lost", o)
			}
			report, err := Replay(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			write(dir, "replay.json", report)
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			copyDB(db, filepath.Join(dir, "runtime.sqlite"))
			s, err = Open(t.Context(), db, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			if _, err = Replay(t.Context(), db, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			write(dir, "assertions.json", map[string]any{"StorageVersion": want, "ParentProgramHistoryResultRowsUnchanged": true, "OriginalResultAlreadyConsumed": true, "AdditionalEndpointApplications": 0, "InternalWriter": "Store.recordAdapterResult", "PublicConflictProduction": "not established; no public adapter result writer; concurrent Run rejected by ErrRunActive"})
		})
	}
}
