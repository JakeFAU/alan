package receiptquery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

func sampleQuery() Query {
	ns := strings.Repeat("1", 64)
	key := func(id string) string { b, _ := json.Marshal([]any{2, ns, id}); return string(b) }
	q := Query{QueryID: "read", QueryKey: key("read"), TargetID: "mutation", TargetKey: key("mutation"), LedgerID: strings.Repeat("2", 64), Ordinal: 1}
	b, _ := json.Marshal(struct {
		Key, Profile string
		Delta        int64
	}{q.TargetKey, "idempotent", 1})
	q.Wire = string(b)
	q.WireSHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
	return q
}
func TestSnapshotRequiresExactStoredWireAndCoherentReceipt(t *testing.T) {
	q := sampleQuery()
	good := Snapshot{Query: q, Number: 2, Status: "found", Count: 1, AppliedWire: q.Wire, Receipt: &Receipt{Key: q.TargetKey, Profile: "idempotent", Delta: 1, Mutation: 1, Total: 1, Process: "original", PID: 17}}
	for _, tc := range []struct {
		name string
		edit func(*Snapshot)
	}{
		{"ordinal", func(s *Snapshot) { s.Query.Ordinal++ }}, {"response-target", func(s *Snapshot) { s.Query.TargetID = "peer" }},
		{"wire-preimage", func(s *Snapshot) { s.AppliedWire += " " }}, {"multiplicity", func(s *Snapshot) { s.Count = 2 }},
		{"receipt-delta", func(s *Snapshot) { s.Receipt.Delta = 2 }}, {"receipt-key", func(s *Snapshot) { s.Receipt.Key = q.QueryKey }},
		{"absence-with-receipt", func(s *Snapshot) { s.Status = "not_found" }}, {"unidentified", func(s *Snapshot) { s.Number = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := effects.Copy(good)
			tc.edit(&s)
			b, _ := json.Marshal(s)
			if _, err := DecodeSnapshot(q, string(b)); err == nil {
				t.Fatal("unbound receipt accepted")
			}
		})
	}
	b, _ := json.Marshal(good)
	if _, err := DecodeSnapshot(q, string(b)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{string(b[:len(b)-1]), strings.Repeat(" ", 65537), strings.Replace(string(b), `"Count":1`, `"Count":2,"Count":1`, 1)} {
		if _, err := DecodeSnapshot(q, bad); err == nil {
			t.Fatal("ambiguous or incomplete bytes accepted")
		}
	}
}

func TestReaderCannotEnterApplyOrFollowARedirect(t *testing.T) {
	q := sampleQuery()
	visits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visits[r.URL.Path]++
		http.Redirect(w, r, "/phase2c/"+q.LedgerID+"/apply", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	lookup := server.URL + "/phase2c/" + q.LedgerID + "/lookup"
	apply := server.URL + "/phase2c/" + q.LedgerID + "/apply"
	input, _ := ReadInput().Retain()
	output, _ := ReadOutput().Retain()
	req := effects.Request{ID: q.QueryID, Key: q.QueryKey, Capability: ReadCapability, Effect: "read", Input: input, Output: output, Payload: map[string]schema.Value{"target": schema.Text(q.TargetID)}}
	reader := Reader{Endpoint: lookup}
	p, err := reader.Prepare(req)
	if err != nil {
		t.Fatal(err)
	}
	p = effects.Freeze(p)
	if c := reader.Execute(t.Context(), p); c.Disposition == "validated" || len(visits) != 0 {
		t.Fatal(c, visits)
	}
	if c := reader.ExecuteRead(t.Context(), p, q); c.HTTPStatus != 307 || c.Disposition != "protocol_invalid" {
		t.Fatal(c)
	}
	for _, bad := range []string{apply, lookup + "?", lookup + "#", strings.Replace(lookup, "lookup", "%6cookup", 1)} {
		reader.Endpoint = bad
		if _, err := reader.Prepare(req); err == nil {
			t.Fatalf("accepted %q", bad)
		}
		if c := reader.ExecuteRead(t.Context(), p, q); c.HTTPStatus != 0 {
			t.Fatal(c)
		}
	}
	if len(visits) != 1 || visits["/phase2c/"+q.LedgerID+"/lookup"] != 1 {
		t.Fatal(visits)
	}
}

func TestPhysicalQueryRejectsDamagedHTTPResponses(t *testing.T) {
	q := sampleQuery()
	for _, tc := range []struct {
		name, disposition string
		body              string
		truncated         bool
	}{
		{"malformed", "protocol_invalid", "{", false}, {"oversized", "protocol_invalid", strings.Repeat("x", MaxSnapshotBytes+1), false}, {"truncated", "transport_error", "{", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.truncated {
					w.Header().Set("Content-Length", "8")
				}
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			reader := Reader{Endpoint: server.URL + "/phase2c/" + q.LedgerID + "/lookup"}
			in, _ := ReadInput().Retain()
			out, _ := ReadOutput().Retain()
			p, err := reader.Prepare(effects.Request{ID: q.QueryID, Key: q.QueryKey, Capability: ReadCapability, Effect: "query", Input: in, Output: out, Payload: map[string]schema.Value{"target": schema.Text(q.TargetID)}})
			if err != nil {
				t.Fatal(err)
			}
			c := reader.ExecuteRead(t.Context(), effects.Freeze(p), q)
			if c.ID != q.QueryID || c.Key != q.QueryKey || c.Disposition != tc.disposition || c.HTTPStatus != 200 || len(c.Payload) != 0 || len(c.Response) > MaxSnapshotBytes {
				t.Fatal(c)
			}
			if _, err = DecodeSnapshot(q, c.Response); err == nil {
				t.Fatal("damaged response admitted as snapshot")
			}
		})
	}
}

func TestUnboundReconstructedCounterRejected(t *testing.T) {
	q := sampleQuery()
	in, _ := CounterInput().Retain()
	out, _ := CounterOutput().Retain()
	counter := Counter{Endpoint: "http://127.0.0.1:5555/phase2c/" + q.LedgerID + "/apply"}
	p, err := counter.Prepare(effects.Request{ID: q.TargetID, Key: q.TargetKey, Capability: CounterCapability, Effect: "increment", Input: in, Output: out, Payload: map[string]schema.Value{"delta": schema.Int(1)}})
	if err != nil {
		t.Fatal(err)
	}
	p = effects.Freeze(p)
	if err = CounterPrepared(p); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*effects.Prepared)
	}{
		{"legacy-protocol", func(p *effects.Prepared) { p.Protocol = "phase2a/counter/v1" }}, {"false-delta", func(p *effects.Prepared) { p.Request.Payload["delta"] = schema.Int(2) }},
		{"request-key", func(p *effects.Prepared) { p.Request.Key = q.QueryKey }}, {"unknown-model", func(p *effects.Prepared) { p.Model = "any-receipt-is-success" }},
		{"different-wire", func(p *effects.Prepared) { p.Wire += " " }}, {"read-route", func(p *effects.Prepared) { p.Endpoint = strings.TrimSuffix(p.Endpoint, "apply") + "lookup" }},
		{"schema", func(p *effects.Prepared) { p.Request.Output, _ = (schema.Record{"mutation": {Kind: "int"}}).Retain() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corrupt := effects.Copy(p)
			tc.edit(&corrupt)
			if CounterPrepared(effects.Freeze(corrupt)) == nil {
				t.Fatal("accepted wrong reconstruction profile")
			}
		})
	}
}

func TestQueryTupleAndMutationValidation(t *testing.T) {
	q := sampleQuery()
	for _, tc := range []struct {
		name string
		edit func(*Query)
	}{
		{"wrong-namespace", func(q *Query) {
			q.TargetKey = strings.Replace(q.TargetKey, strings.Repeat("1", 64), strings.Repeat("2", 64), 1)
		}},
		{"target-id", func(q *Query) { q.TargetID += "x" }}, {"same-op", func(q *Query) { q.QueryID = q.TargetID; q.QueryKey = q.TargetKey }},
		{"wrong-key", func(q *Query) { q.TargetKey = q.QueryKey }}, {"wire-digest", func(q *Query) { q.WireSHA256 = strings.Repeat("f", 64) }},
		{"wire-key", func(q *Query) {
			q.Wire = strings.ReplaceAll(q.Wire, "mutation", "other")
			q.WireSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(q.Wire)))
		}},
		{"ordinal-limit", func(q *Query) { q.Ordinal = 3 }}, {"ledger", func(q *Query) { q.LedgerID = "unknown" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := q
			tc.edit(&bad)
			if bad.Check() == nil {
				t.Fatal("invalid query identity accepted")
			}
		})
	}
}
