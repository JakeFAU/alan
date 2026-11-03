package runtime_test

import (
	"database/sql"
	"errors"
	"testing"

	rt "github.com/JakeFAU/alan/runtime"
)

// The first input in the present schedule binds to the open entry at its
// requested generation. It was refused because the static route is absent.
func TestCausalRecoveryAuthenticatesBoundTransport(t *testing.T) {
	for _, reason := range []string{"target_closed", "target_retired", "target_generation_changed", "source_denied:source_closed", "schema_invalid"} {
		t.Run(reason, func(t *testing.T) {
			path := recoveryCopy(t, "present")
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			row, err := db.ExecContext(t.Context(), `UPDATE admissions SET result=CAST(json_set(result,'$.Reason',?) AS BLOB)
				WHERE transport_id='presence' AND domain='input' AND json_extract(result,'$.Reason')='route_denied'
				AND json_extract(result,'$.Bound.Subject')=? AND json_extract(result,'$.Bound.Generation')=1`, reason, recoveryEntry)
			if err != nil {
				t.Fatal(err)
			}
			if n, err := row.RowsAffected(); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			requireRecoveryRejection(t, path, rt.ErrCorrupt)
		})
	}
}

func TestCausalRecoveryRejectsHiddenComposition(t *testing.T) {
	path := recoveryCopy(t, "ready")
	cut := recoveryCut(t, path)
	if cut.StorageVersion != 4 || len(cut.Roots) != 1 || cut.Roots[0].Charges.Effects != 1 || len(cut.Roots[0].UnknownExternal) != 1 || cut.Roots[0].UnknownExternal[0] != recoveryWrite {
		t.Fatal("fixture needs the original charged, unresolved write", cut.Roots)
	}
	// Keep the entire composition deployment and its append-only facts. Only
	// the two version anchors now claim an ordinary store without composition.
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(t.Context(), "UPDATE metadata SET schema_version=3 WHERE schema_version=4; PRAGMA user_version=3"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	requireRecoveryRejection(t, path, rt.ErrVersion)
}

// Keep the genuine binding and route refusal intact. Each isolated edit
// makes the input ineligible to reach the writer's target lookup at all.
func TestCausalRecoveryRejectsMalformedPresenceWitness(t *testing.T) {
	for _, tc := range []struct{ name, fields string }{
		{"empty transport ID", `id='["admission","input",""]', transport_id='',
			descriptor=CAST(json_set(descriptor,'$.TransportID','') AS BLOB),
			result=CAST(json_set(result,'$.TransportID','') AS BLOB)`},
		{"empty payload key", `descriptor=CAST(json_set(descriptor,'$.Payload',json('{"":{"Kind":"string","Number":0,"Text":"","Flag":false}}')) AS BLOB)`},
		{"invalid typed payload", `descriptor=CAST(json_set(descriptor,'$.Payload',json('{"v":{"Kind":"string","Number":1,"Text":"","Flag":false}}')) AS BLOB)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := recoveryCopy(t, "present")
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			r, err := db.ExecContext(t.Context(), "UPDATE admissions SET "+tc.fields+" WHERE transport_id='presence' AND domain='input'")
			if err != nil {
				t.Fatal(err)
			}
			if n, err := r.RowsAffected(); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			requireRecoveryRejection(t, path, rt.ErrCorrupt)
		})
	}
}

func requireRecoveryRejection(t *testing.T, path string, want error) {
	t.Helper()
	ctx, limits := t.Context(), rt.DefaultLimits()
	before := recoveryImage(t, path)
	cut, inspect := rt.InspectExecution(ctx, path, limits)
	_, replay := rt.Replay(ctx, path, limits)
	owner, open := rt.Open(ctx, path, limits)
	if owner != nil {
		if err := owner.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if cut != nil || !errors.Is(inspect, want) || !errors.Is(replay, want) || owner != nil || !errors.Is(open, want) {
		t.Errorf("want %v: InspectExecution=%v Replay=%v Open=%v owned=%t", want, inspect, replay, open, owner != nil)
	}
	if recoveryImage(t, path) != before {
		t.Fatal("rejected history caused a durable change")
	}
}
