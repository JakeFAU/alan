package openai_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

func evidenceFixture(t *testing.T, disposition, envelope string) (effects.Prepared, effects.Candidate) {
	t.Helper()
	output, err := (schema.Record{"answer": {Kind: "string", MaxBytes: 128}}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	p := effects.Freeze(effects.Prepared{Request: effects.Request{ID: "local", Key: "key", Output: output}, Protocol: "openai/responses/v1", MaxResponseBytes: 4096})
	details := `{ "output_tokens": 3, "extra": {"x":"<flag>", "large":9007199254740993} }`
	c := effects.Candidate{ID: "local", Key: "key", Disposition: disposition, HTTPStatus: 200, RequestID: "http-id", ResponseID: "resp-id", ReturnedModel: "fixture",
		Response: `{"id":"resp-id","model":"fixture","usage":` + details + `,` + envelope + `}`,
		Usage:    &effects.Usage{OutputTokens: new(int64(3)), Details: json.RawMessage(details)}}
	if disposition == "validated" {
		c.Payload = map[string]schema.Value{"answer": schema.Text("original")}
	}
	return p, effects.Copy(c)
}

const validEnvelope = `"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"answer\":\"original\"}"}]}]`

func TestVerifyResponseEvidence(t *testing.T) {
	for _, tc := range []struct{ disposition, envelope string }{
		{"validated", validEnvelope},
		{"incomplete", `"status":"incomplete"`},
		{"provider_failed", `"status":"failed"`},
		{"refusal", `"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"refusal","refusal":"no"}]}]`},
		{"schema_invalid", strings.Replace(validEnvelope, `{\"answer\":\"original\"}`, `{}`, 1)},
		{"protocol_invalid", `"status":"in_progress"`},
	} {
		t.Run(tc.disposition, func(t *testing.T) {
			p, c := evidenceFixture(t, tc.disposition, tc.envelope)
			// The runtime's JSON round trip already compacted and HTML-escaped Details.
			c.Usage.Details = json.RawMessage(`{"extra":{"large":9007199254740993,"x":"\u003cflag\u003e"},"output_tokens":3}`)
			if err := openai.VerifyResponseEvidence(p, c); err != nil {
				t.Fatalf("genuine evidence failed: %v", err)
			}
			for _, change := range []struct {
				name string
				edit func(*effects.Candidate)
			}{
				{"reported response ID", func(c *effects.Candidate) { c.ResponseID = "another" }},
				{"reported model", func(c *effects.Candidate) { c.ReturnedModel = "another" }},
				{"reported usage", func(c *effects.Candidate) { c.Usage.OutputTokens = new(int64(22)) }},
				{"unknown is not zero", func(c *effects.Candidate) { c.Usage.InputTokens = new(int64(0)) }},
				{"missing usage", func(c *effects.Candidate) { c.Usage = nil }},
				{"reported details", func(c *effects.Candidate) {
					c.Usage.Details = json.RawMessage(`{"extra":{"large":9007199254740992,"x":"<flag>"},"output_tokens":3}`)
				}},
				{"duplicate details", func(c *effects.Candidate) {
					c.Usage.Details = json.RawMessage(`{"output_tokens":22,"output_tokens":3,"extra":{"x":"<flag>","large":9007199254740993}}`)
				}},
				{"body metadata", func(c *effects.Candidate) { c.Response = strings.Replace(c.Response, "resp-id", "altered", 1) }},
				{"proposal", func(c *effects.Candidate) { c.Payload = map[string]schema.Value{"answer": schema.Text("inserted")} }},
				{"foreign request", func(c *effects.Candidate) { c.Key = "other" }},
				{"disposition", func(c *effects.Candidate) { c.Disposition = "http_rejected" }},
			} {
				t.Run(change.name, func(t *testing.T) {
					candidate := effects.Copy(c)
					change.edit(&candidate)
					err := openai.VerifyResponseEvidence(p, candidate)
					if err == nil || errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
						t.Fatalf("wanted contradiction, got %v", err)
					}
				})
			}
		})
	}
}

func TestVerifyResponseEvidenceLimits(t *testing.T) {
	p, full := evidenceFixture(t, "validated", validEnvelope)
	for _, tc := range []struct {
		name, disposition, response string
		http                        int
	}{
		{"transport unknown", "unknown", "", 0},
		{"HTTP rejection before decoding", "http_rejected", full.Response, 429},
		{"size limit before decoding", "response_limit", strings.Repeat("x", p.MaxResponseBytes), 200},
		{"malformed", "protocol_invalid", "{", 200},
		{"duplicate omitted by sanitizer", "protocol_invalid", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := effects.Candidate{ID: full.ID, Key: full.Key, Disposition: tc.disposition, Response: tc.response, HTTPStatus: tc.http}
			if err := openai.VerifyResponseEvidence(p, c); err != nil {
				t.Fatal(err)
			}
			c.Usage = full.Usage
			if err := openai.VerifyResponseEvidence(p, c); err == nil || errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
				t.Fatalf("unparsed response gained usage: %v", err)
			}
		})
	}
	long := strings.Repeat("é", 270)
	c := effects.Copy(full)
	c.Response = strings.Replace(c.Response, "resp-id", long, 1)
	c.ResponseID, c.Payload, c.Disposition = long[:512], nil, "protocol_invalid"
	if err := openai.VerifyResponseEvidence(p, c); err != nil {
		t.Fatalf("historical ID cap: %v", err)
	}
	c = effects.Copy(full)
	c.Payload, c.Disposition = nil, "credential_redacted"
	if err := openai.VerifyResponseEvidence(p, c); err != nil {
		t.Fatalf("a scrubbed header can suppress a valid body: %v", err)
	}
	c.Usage.OutputTokens = new(int64(22))
	if err := openai.VerifyResponseEvidence(p, c); err == nil || errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
		t.Fatalf("redaction hid a contradictory counter: %v", err)
	}
}

func TestVerifyResponseEvidenceRequiresRetainedRequest(t *testing.T) {
	p, c := evidenceFixture(t, "validated", validEnvelope)
	p.Digest = ""
	if err := openai.VerifyResponseEvidence(p, c); err == nil {
		t.Fatal("unfrozen request accepted")
	}
}

func TestVerifyResponseLimitRejectsShortUnchangedBody(t *testing.T) {
	p, full := evidenceFixture(t, "validated", validEnvelope)
	c := effects.Candidate{ID: full.ID, Key: full.Key, HTTPStatus: 200, Response: full.Response, Disposition: "response_limit"}
	if err := openai.VerifyResponseEvidence(p, c); err == nil || errors.Is(err, openai.ErrUnverifiableResponseEvidence) {
		t.Fatalf("a short unsanitized body cannot be the retained wire-limit prefix: %v", err)
	}
}
