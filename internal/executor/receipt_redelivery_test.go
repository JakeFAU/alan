//go:build integration

package executor

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/effects"
)

func openReceiptCut(t *testing.T, family, label string) (*Store, string) {
	t.Helper()
	fixture := filepath.Join("..", "..", "docs", "foundational", "phase2c", "evidence", "process-normal", family, label+"-local.sqlite.gz")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	raw, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "genuine.sqlite")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
	})
	return s, path
}

// This exercises the private recording boundary with a genuinely retained
// candidate from the separate-process campaign. The controlled endpoint is
// absent. A closed resolver can redeliver the identical authorized return
// without adding a second proof, expense or observation.
func TestPhase2CRedeliveryOfAdmittedReturn(t *testing.T) {
	s, path := openReceiptCut(t, "atomic-import", "final")
	const query = `["resolver",1,4,0]`
	a, err := loadAdapterAttempt(t.Context(), s.conn, query)
	if err != nil {
		t.Fatal(err)
	}
	o, err := readOrdinal(t.Context(), s.conn, query, 1)
	if err != nil {
		t.Fatal(err)
	}
	r, err := loadAdapterResult(t.Context(), s.conn, query)
	if err != nil || r.Read == nil || r.Read.Adoption != "adopted" {
		t.Fatal(r, err)
	}
	before, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.recordReadResult(t.Context(), a, o, r.Candidate)
	if err != nil || !reflect.DeepEqual(got, r) {
		t.Fatal("equivalent redelivery", got, err)
	}
	bad := effects.Candidate{ID: r.Candidate.ID, Key: r.Candidate.Key, Disposition: "transport_error", Response: "unavailable"}
	if _, err = s.recordReadResult(t.Context(), a, o, bad); !errors.Is(err, ErrConflict) {
		t.Fatal("late incompatible candidate overwrote the first", err)
	}
	after, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("redelivery changed retained accounting", err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

// Corruption injection only. Use a disposable copy after the first authorized
// read was observed by the real endpoint, but before its result was retained.
// A grant denial cannot explain a pre-entry local refusal for this immutable
// already-granted adapter attempt. Recompute the normal envelopes and facts
// to attack the semantic checker rather than testing a stale hash.
func TestPhase2CRejectImpossibleGrantDenial(t *testing.T) {
	for _, reason := range []string{"grant_denied", "source_denied", "read_denied", "read_exhausted", "attempt_missing", "target_missing", "original_denied", "endpoint_mismatch", "unsupported_original", "original_open", "original_busy", "original_unsettled"} {
		t.Run(reason, func(t *testing.T) { rejectForgedRefusal(t, reason) })
	}
}
func rejectForgedRefusal(t *testing.T, reason string) {
	t.Helper()
	s, path := openReceiptCut(t, "held-lost", "lost-absence")
	const id = `["resolver",1,4,0]`
	a, e := loadAdapterAttempt(t.Context(), s.conn, id)
	if e != nil || !a.Allowed {
		t.Fatal(a, e)
	}
	if _, e = readOrdinal(t.Context(), s.conn, id, 1); e != nil {
		t.Fatal(e)
	}
	fake := effects.Candidate{ID: id, Key: a.Prepared.Request.Key, Disposition: reason}
	if _, e = s.recordAdapterResult(t.Context(), a, fake); e == nil {
		t.Error("generic recorder accepted a fabricated allowed-read refusal")
	} else {
		// Bypass the generic recorder in the disposable copy. Regenerate both
		// the row digest and its genuine ordered fact to attack replay itself.
		values, disposition := effects.Result(a.Prepared.Request.Output.Record, fake)
		r := AdapterResult{Candidate: fake, Values: values, Disposition: disposition}
		if reason == "original_busy" || reason == "original_unsettled" {
			r.RefusalWitness = a.Prepared.Request.Payload["target"].Text
		}
		r.Digest = adapterResultDigest(r)
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := s.conn.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(t.Context(), "INSERT INTO adapter_results VALUES(?,?)", id, raw); err != nil {
			t.Fatal(err)
		}
		if err = s.recordCompositionSource(t.Context(), tx, compositionExternalResultFact, id); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if _, e = Replay(t.Context(), path, DefaultLimits()); e == nil {
		t.Fatal("replay accepted a fabricated refusal for an allowed attempt with an admitted read and matching immutable grant")
	}
}
