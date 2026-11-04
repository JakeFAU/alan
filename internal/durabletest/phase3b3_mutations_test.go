//go:build integration

package durabletest

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
)

// Raw SQL dumping intentionally has no dependency on runtime validation. All
// callers supply disposable copies; archived originals are never opened.
func reciprocalRawDump(t *testing.T, path, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite3", (&url.URL{Scheme: "file", Path: path}).String()+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(t.Context(), "SELECT name FROM sqlite_schema WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err = rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	tables := map[string][]map[string]any{}
	for _, name := range names {
		query := `SELECT * FROM "` + strings.ReplaceAll(name, `"`, `""`) + `" ORDER BY rowid`
		rows, err = db.QueryContext(t.Context(), query)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range targets {
				targets[i] = &values[i]
			}
			if err = rows.Scan(targets...); err != nil {
				t.Fatal(err)
			}
			row := map[string]any{}
			for i, name := range columns {
				v := values[i]
				if b, ok := v.([]byte); ok {
					v = string(b)
				}
				row[name] = v
			}
			tables[name] = append(tables[name], row)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	baseline3b3Write(t, dir, "raw-tables.json", map[string]any{"Tables": tables})
}

type reciprocalFact struct {
	AuthorityAdmission        *runtime.AuthorityAdmission `json:",omitempty"`
	Version, ID, Kind         string
	Order                     int64
	RootIDs                   []string
	Run                       runtime.RunID
	Sequence                  int64
	OperationID, RecordDigest string
}

func TestPhase3B3RetainedMutations(t *testing.T) {
	baseDir := reciprocalDirectory(t, "mutation-base")
	ca, ua, pa := reciprocalEndpoint(t, baseDir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, baseDir, "b", nil)
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "base.sqlite")
	s, err := runtime.Create(t.Context(), base, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	s.Close()
	if err != nil {
		t.Fatal(err)
	}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	reciprocalCopyDB(t, pa, filepath.Join(baseDir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(baseDir, "endpoint-b.sqlite"))
	baseline3b3Write(t, baseDir, "provision.json", p)
	baseline3b3Write(t, baseDir, "manifest.json", m)
	reciprocalCapture(t, base, filepath.Join(baseDir, "final"))
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "mutation-base", Schedule: "automatic-run", Directory: baseDir, Classification: "safe-complete", Manifest: m})
	baseline3b3Manifest(t, baseDir)
	for _, name := range []string{"missing-grant", "forged-grant-origin", "effect-use-result", "gate-attempt-digest", "result-link"} {
		t.Run(name, func(t *testing.T) {
			dir := reciprocalDirectory(t, "mutation-"+name)
			path := filepath.Join(t.TempDir(), "mutated.sqlite")
			reciprocalCopyDB(t, base, path)
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			var before, after []byte
			query := ""
			var args []any
			switch name {
			case "missing-grant":
				query = "DELETE FROM lifecycle_results WHERE id=?"
				args = []any{m.Grant}
			case "forged-grant-origin":
				if err = db.QueryRow("SELECT result FROM lifecycle_results WHERE id=?", m.Grant).Scan(&before); err != nil {
					t.Fatal(err)
				}
				var r runtime.LifecycleResult
				if err = json.Unmarshal(before, &r); err != nil {
					t.Fatal(err)
				}
				r.Request.Intent.Run.Generation = 2
				after, err = json.Marshal(r)
				query = "UPDATE lifecycle_results SET result=? WHERE id=?"
				args = []any{after, m.Grant}
			case "effect-use-result":
				if err = db.QueryRow("SELECT descriptor FROM external_attempts WHERE intent_id=?", m.BAService).Scan(&before); err != nil {
					t.Fatal(err)
				}
				var a runtime.ExternalAttempt
				if err = json.Unmarshal(before, &a); err != nil {
					t.Fatal(err)
				}
				a.Authority.Use.GrantID = m.ABService
				after, err = json.Marshal(a)
				query = "UPDATE external_attempts SET descriptor=? WHERE intent_id=?"
				args = []any{after, m.BAService}
			case "gate-attempt-digest":
				if err = db.QueryRow("SELECT descriptor FROM composition_facts WHERE kind='authority_admission' ORDER BY decision_order LIMIT 1").Scan(&before); err != nil {
					t.Fatal(err)
				}
				var f reciprocalFact
				if err = json.Unmarshal(before, &f); err != nil {
					t.Fatal(err)
				}
				f.AuthorityAdmission.AttemptDigest = "forged"
				raw, _ := json.Marshal([]any{"alan/authority-admission/v1", f.AuthorityAdmission})
				sum := sha256.Sum256(raw)
				f.RecordDigest = hex.EncodeToString(sum[:])
				after, err = json.Marshal(f)
				query = "UPDATE composition_facts SET descriptor=? WHERE id=?"
				args = []any{after, f.ID}
			case "result-link":
				if err = db.QueryRow("SELECT descriptor FROM external_receipts WHERE intent_id=?", m.BAService).Scan(&before); err != nil {
					t.Fatal(err)
				}
				var r runtime.ExternalReceipt
				if err = json.Unmarshal(before, &r); err != nil {
					t.Fatal(err)
				}
				r.Key = "substituted-equal-payload-operation"
				after, err = json.Marshal(r)
				query = "UPDATE external_receipts SET descriptor=? WHERE intent_id=?"
				args = []any{after, m.BAService}
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := db.Exec(query, args...)
			if err != nil {
				t.Fatal(err)
			}
			n, err := result.RowsAffected()
			if err != nil || n != 1 {
				t.Fatal("mutation boundary", n, err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "mutation.json", map[string]any{"Name": name, "SQL": query, "OriginalBytes": string(before), "MutatedBytes": string(after), "Rows": n, "SourceCase": baseDir})
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			reciprocalRawDump(t, path, dir)
			reciprocalCopyDB(t, path, filepath.Join(dir, "runtime.sqlite"))
			outcomes := map[string]string{}
			for _, api := range []string{"Inspect", "InspectExecution", "Replay", "Open"} {
				probe := filepath.Join(t.TempDir(), "probe.sqlite")
				reciprocalCopyDB(t, path, probe)
				switch api {
				case "Inspect":
					_, err = runtime.Inspect(t.Context(), probe, runtime.DefaultLimits())
				case "InspectExecution":
					_, err = runtime.InspectExecution(t.Context(), probe, runtime.DefaultLimits())
				case "Replay":
					_, err = runtime.Replay(t.Context(), probe, runtime.DefaultLimits())
				case "Open":
					var opened *runtime.Store
					opened, err = runtime.Open(t.Context(), probe, runtime.DefaultLimits())
					if opened != nil {
						opened.Close()
					}
				}
				outcomes[api] = fmt.Sprint(err)
				if err == nil {
					t.Errorf("%s accepted %s", api, name)
				}
			}
			baseline3b3Write(t, dir, "outcomes.json", outcomes)
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "mutation-" + name, Schedule: "disposable-store-mutation", Directory: dir, Classification: "rejected-invalid-history", Manifest: m})
			baseline3b3Manifest(t, dir)
		})
	}
}
