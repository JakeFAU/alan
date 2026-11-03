package openmeteo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
)

func TestFinalHTTPResponseOnWire(t *testing.T) {
	for _, tc := range []struct {
		name, preamble, body, disposition string
		wireStatus, retainedStatus        int
	}{
		{"interim_only", "", "", "unknown", 103, 0},
		{"unsolicited_switch", "", "", "unknown", 101, 0},
		{"out_of_range", "", "", "unknown", 600, 0},
		{"no_content", "", "unexpected content", "http_rejected", 204, 204},
		{"not_modified", "", "unexpected content", "http_rejected", 304, 304},
		{"non_utf8_refusal", "", "\xff", "http_rejected", 429, 429},
		{"interim_then_success", "HTTP/1.1 103 Early Hints\r\n\r\n", fixture, "validated", 200, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var contacts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				contacts.Add(1)
				// A normal ResponseWriter suppresses illegal bodies and supplies final headers.
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				_, err = fmt.Fprintf(conn, "%sHTTP/1.1 %d Test\r\nContent-Length: %d\r\nContent-Type: application/json\r\n\r\n%s",
					tc.preamble, tc.wireStatus, len(tc.body), tc.body)
				if err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			a, p := testAdapter(t, server.URL+"/v1/forecast")
			result := a.Execute(t.Context(), p)
			if contacts.Load() != 1 || result.Disposition != tc.disposition || result.HTTPStatus != tc.retainedStatus {
				t.Fatalf("contacts=%d disposition=%q status=%d", contacts.Load(), result.Disposition, result.HTTPStatus)
			}
			wantBody := ""
			if tc.disposition == "validated" {
				wantBody = tc.body
			}
			if result.Response != wantBody || (VerifyResponseEvidence(p, result) == nil) != (tc.retainedStatus != 0) {
				t.Fatalf("unexpected native evidence: %+v", result)
			}
		})
	}
}

func TestRejectImpossibleNegativeHTTPEvidence(t *testing.T) {
	_, p := testAdapter(t, "http://127.0.0.1:1/v1/forecast")
	for _, tc := range []struct {
		status            int
		disposition, body string
	}{
		{101, "http_rejected", ""}, {103, "http_rejected", ""}, {600, "http_rejected", ""},
		{204, "http_rejected", "body"}, {304, "http_rejected", "body"},
		{204, "response_limit", ""}, {304, "response_limit", strings.Repeat("x", MaxResponseBytes)},
		{429, "http_rejected", "\xff"}, {200, "protocol_invalid", "\xff"},
		{200, "response_limit", strings.Repeat("x", MaxResponseBytes-1) + "\xff"},
	} {
		t.Run(strconv.Itoa(tc.status)+"_"+tc.disposition, func(t *testing.T) {
			c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, HTTPStatus: tc.status, Disposition: tc.disposition, Response: tc.body}
			if err := VerifyResponseEvidence(p, c); err == nil {
				t.Fatalf("verified impossible %s with status %d and %d retained bytes", tc.disposition, tc.status, len(tc.body))
			}
		})
	}
}
