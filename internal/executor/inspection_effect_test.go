//go:build integration

package executor

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

func TestInspectEffectRetainsFirstOutcomeAndLaterOffers(t *testing.T) {
	s, path, id := committedAdapter(t)
	ctx := t.Context()
	view := func() *ExecutionEffect {
		t.Helper()
		snapshot, err := InspectExecution(ctx, path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		e, ok := snapshot.Effect(id)
		if !ok {
			t.Fatal("missing original effect")
		}
		return e
	}
	if e := view(); e.State != "committed" || e.Consequence != "no_dispatch" || e.Authorization != nil || e.Dispatch.Minimum != 0 {
		t.Fatal(e)
	}
	if _, err := s.AuthorizeIntent(ctx, id); err != nil {
		t.Fatal(err)
	}
	if e := view(); e.State != "authorized" || e.AdapterAttempt != nil {
		t.Fatal(e)
	}
	w, err := s.prepareAdapterWork(ctx, id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || w == nil {
		t.Fatal(err)
	}
	if e := view(); e.State != "reserved_unknown" || e.AdapterResult != nil || e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 1 {
		t.Fatal(e)
	}
	c := effects.Candidate{ID: id, Key: w.attempt.Prepared.Request.Key, Disposition: "http_error"}
	first, err := s.recordAdapterResult(ctx, w.attempt, c)
	if err != nil {
		t.Fatal(err)
	}
	if e := view(); e.State != "result_unsettled" || e.ResultDisposition != "http_error" || e.Consequence != "unresolved" || e.Settlement != nil {
		t.Fatal(e)
	}
	if err := s.settleAdapter(ctx, w.attempt, first); err != nil {
		t.Fatal(err)
	}
	if e := view(); e.State != "settled" || e.Consequence != "unresolved" {
		t.Fatalf("local settlement overstated remote application: %+v", e)
	}
	for _, answer := range []string{"late", "late", "different"} {
		candidate := effects.Candidate{ID: id, Key: c.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text(answer)}}
		if _, err := s.recordAdapterResult(ctx, w.attempt, candidate); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
	}
	e := view()
	if e.State != "settled" || e.Settlement == nil || e.AdapterResult.Digest != first.Digest || e.ResultDisposition != "http_error" || len(e.AdditionalReceipts) != 2 || e.ReceiptEvidenceState != "conflict" || e.Consequence != "conflicting_receipts" {
		t.Fatalf("first and later evidence: %+v", e)
	}
	if _, err := s.conn.ExecContext(ctx, "UPDATE settlements SET observation_id=NULL WHERE intent_id=?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectExecution(ctx, path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("unrelated settlement accepted: %v", err)
	}
}

// Start from an actual post-SIGKILL ordinal. Keep both the modified record
// and its ordered digest canonical, so a checksum alone cannot reject it.
func TestInspectReceiptReadRejectsReboundAuthority(t *testing.T) {
	for _, damage := range []string{"swapped_operations", "wrong_original_generation", "wrong_authorization", "wrong_resolver_epoch", "forged_adoption"} {
		t.Run(damage, func(t *testing.T) {
			s, path := openReceiptCut(t, "unrecorded-read")
			const query = `["resolver",1,4,0]`
			o, err := readOrdinal(t.Context(), s.conn, query, 1)
			if err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "swapped_operations":
				o.Query.QueryID, o.Query.TargetID = o.Query.TargetID, o.Query.QueryID
				o.Query.QueryKey, o.Query.TargetKey = o.Query.TargetKey, o.Query.QueryKey
			case "wrong_original_generation":
				o.OriginalRun.Generation++
			case "wrong_authorization":
				o.OriginalAuthorization = identity("authorization", query)
			case "wrong_resolver_epoch":
				o.Resolver.Epoch = 0
			case "forged_adoption":
				o.Adopt = !o.Adopt
			}
			fid := compositionIdentity("alan/fact/receipt-read/v1", []any{query, 1})
			var original []byte
			if err = s.conn.QueryRowContext(t.Context(), "SELECT descriptor FROM composition_facts WHERE id=?", fid).Scan(&original); err != nil {
				t.Fatal(err)
			}
			var f compositionFact
			if err = json.Unmarshal(original, &f); err != nil {
				t.Fatal(err)
			}
			f.RecordDigest = compositionIdentity("alan/receipt-read-ordinal/v1", o)
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE receipt_read_ordinals SET descriptor=CAST(? AS BLOB) WHERE intent_id=? AND ordinal=1", compositionJSON(o), query); err != nil {
				t.Fatal(err)
			}
			if _, err = s.conn.ExecContext(t.Context(), "UPDATE composition_facts SET descriptor=CAST(? AS BLOB) WHERE id=?", compositionJSON(f), fid); err != nil {
				t.Fatal(err)
			}
			if cut, err := InspectExecution(t.Context(), path, DefaultLimits()); cut != nil || !errors.Is(err, ErrCorrupt) {
				t.Fatalf("returned a graph with forged original read authority: %v", err)
			}
		})
	}
}
