//go:build integration

package openai_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
)

func TestVerifySanitizedHTTPResponse(t *testing.T) {
	const limit = 4096
	// A literal '<' expands to six bytes when removing a credential forces
	// re-encoding. The response stored after the final cap ends inside x.
	padding := strings.Repeat("<", limit/3)
	unknownFields := `{"a":"` + padding + `","id":"real","model":"fixture","status":"incomplete","usage":{"output_tokens":3},"z":"` + reviewCredential + `"}`
	knownFields := `{"id":"real","model":"fixture","status":"incomplete","usage":{"output_tokens":3},"x":"` + padding + `","z":"` + reviewCredential + `"}`
	for _, tc := range []struct {
		name, body, header, disposition string
		incomplete, known               bool
	}{
		{"body scrub", strings.TrimSuffix(reviewCompleted, "}") + `,"extra":"` + reviewCredential + `"}`, "", "credential_redacted", false, false},
		{"header scrub", reviewCompleted, reviewCredential, "credential_redacted", false, false},
		{"overlong response ID", strings.Replace(reviewCompleted, `"status":`, `"id":"`+strings.Repeat("é", 270)+`","status":`, 1), "", "protocol_invalid", false, false},
		{"unknown fields after clipping", unknownFields, "", "incomplete", true, false},
		{"known fields before clipping", knownFields, "", "incomplete", true, true},
		{"wire response limit", strings.Repeat("z", limit+1), "", "response_limit", false, false},
		{"wire response limit scrubbed to empty", reviewCredential + strings.Repeat("z", limit), "", "response_limit", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Request-Id", tc.header)
				_, _ = io.WriteString(w, tc.body)
			}))
			a, err := openai.New(openai.Config{APIKey: reviewCredential, Model: "fixture", Endpoint: s.URL, MaxOutputTokens: 128, MaxResponseBytes: limit})
			if err != nil {
				t.Fatal(err)
			}
			p, err := a.Prepare(request(t))
			if err != nil {
				t.Fatal(err)
			}
			c := effects.Copy(a.Execute(t.Context(), p))
			s.Close()
			if c.Disposition != tc.disposition {
				t.Fatalf("got disposition %q, want %q", c.Disposition, tc.disposition)
			}
			err = openai.VerifyResponseEvidence(p, c)
			if tc.incomplete {
				if c.ResponseID != "real" || c.Usage == nil || *c.Usage.OutputTokens != 3 || !errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
					t.Fatalf("clipped HTTP response lost its evidence boundary: id %q, usage %v, err %v", c.ResponseID, c.Usage, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if tc.known {
				c.Usage.OutputTokens = new(int64(22))
				if err = openai.VerifyResponseEvidence(p, c); err == nil || errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
					t.Fatalf("contradictory usage in the surviving prefix: %v", err)
				}
			}
		})
	}
}

func TestVerifyMalformedHTTPStatus(t *testing.T) {
	// net/http accepts a three-character nonnegative code including 000 or
	// 099. A raw peer is necessary because ResponseWriter disallows them.
	for _, status := range []string{"000", "099"} {
		t.Run(status, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				conn, rw, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				defer conn.Close()
				_, err = rw.WriteString("HTTP/1.1 " + status + " Bad\r\nContent-Length: 0\r\n\r\n")
				if err == nil {
					err = rw.Flush()
				}
				if err != nil {
					t.Error(err)
				}
			}))
			defer s.Close()
			a, err := openai.New(openai.Config{APIKey: reviewCredential, Model: "fixture", Endpoint: s.URL, MaxOutputTokens: 128, MaxResponseBytes: 4096})
			if err != nil {
				t.Fatal(err)
			}
			p, err := a.Prepare(request(t))
			if err != nil {
				t.Fatal(err)
			}
			c := a.Execute(t.Context(), p)
			if c.Disposition != "http_rejected" || c.HTTPStatus >= 100 {
				t.Fatalf("original candidate: %+v", c)
			}
			t.Logf("raw peer status %s produced %s, integer %d", status, c.Disposition, c.HTTPStatus)
			if err := openai.VerifyResponseEvidence(p, c); !errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
				t.Fatalf("unverifiable nonstandard status: %v", err)
			}
		})
	}
}
