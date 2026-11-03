package openai

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
)

// ErrUnverifiableResponseEvidence marks a compatible negative with an incomplete
// evidentiary record. Scrubbing or clipping can hide decoded fields. The transport
// can also retain a nonstandard status below 100, including the zero value.
var ErrUnverifiableResponseEvidence = errors.New("retained Responses body cannot verify all decoded evidence")

// VerifyResponseEvidence checks consistency between a prepared Responses request,
// its retained HTTP status and sanitized body, and the adapter's disposition,
// proposal, response ID, returned model and usage. It performs no I/O. It cannot
// authenticate the provider, HTTP status, request ID header, or discarded bytes.
// For a response_limit result the adapter never decoded the body. For an unknown
// outcome it never retained a response. Neither can carry body-derived fields.
func VerifyResponseEvidence(p effects.Prepared, c effects.Candidate) error {
	if p.Digest == "" || p.Protocol != "openai/responses/v1" || effects.CheckPrepared(p) != nil || p.Request.Output.Check() != nil ||
		c.ID != p.Request.ID || c.Key != p.Request.Key {
		return errors.New("provider evidence does not match the prepared request")
	}
	if c.HTTPStatus < 0 || c.HTTPStatus > 999 || c.HTTPStatus >= 100 && c.HTTPStatus < 200 && c.HTTPStatus != 101 ||
		!utf8.ValidString(c.Response) || len(c.Response) > p.MaxResponseBytes ||
		!utf8.ValidString(c.RequestID) || len(c.RequestID) > 512 || !utf8.ValidString(c.ResponseID) || len(c.ResponseID) > 512 ||
		!utf8.ValidString(c.ReturnedModel) || len(c.ReturnedModel) > 512 {
		return errors.New("provider evidence exceeds the adapter bounds")
	}
	if c.Disposition == "unknown" || c.Disposition == "response_limit" {
		if c.Payload != nil || c.ResponseID != "" || c.ReturnedModel != "" || c.Usage != nil {
			return errors.New("unparsed Responses result contains decoded evidence")
		}
		if c.Disposition == "unknown" {
			if c.HTTPStatus != 0 || c.RequestID != "" || c.Response != "" {
				return errors.New("unknown Responses outcome contains a retained HTTP response")
			}
			return nil
		}
		if !possibleLimitedBody(c.Response, p.MaxResponseBytes) {
			return errors.New("short unchanged body cannot be a Responses wire-limit prefix")
		}
		return checkHTTPStatus(c.HTTPStatus)
	}
	if c.Disposition != "validated" && c.Payload != nil {
		return errors.New("unsuccessful Responses result contains a proposal")
	}
	body := effects.Candidate{HTTPStatus: c.HTTPStatus, Response: c.Response}
	want := decodeResponse(p.Request.Output.Record, body)
	if len(want.ResponseID) > 512 || len(want.ReturnedModel) > 512 {
		want.Disposition, want.Payload = "protocol_invalid", nil
	} else if want.Disposition == "validated" && c.Disposition == "credential_redacted" {
		// A scrubbed header alone can suppress an otherwise valid proposal.
		want.Disposition, want.Payload = "credential_redacted", nil
	}
	if want.Disposition == "protocol_invalid" && c.HTTPStatus >= 200 && c.HTTPStatus < 300 &&
		!json.Valid([]byte(c.Response)) && p.MaxResponseBytes-len(c.Response) < utf8.UTFMax {
		if fields, last, ok := clippedObject(c.Response); ok {
			return verifyClippedResponse(p, c, fields, last)
		}
	}
	if want.Disposition != c.Disposition || !reflect.DeepEqual(want.Payload, c.Payload) {
		return errors.New("provider disposition or proposal differs from the retained body")
	}
	if err := checkResponseMetadata(want, c, true, true, true); err != nil {
		return err
	}
	return checkHTTPStatus(c.HTTPStatus)
}

func checkHTTPStatus(status int) error {
	if status < 100 {
		// Go accepts raw peer codes "000" through "099" as final responses.
		return ErrUnverifiableResponseEvidence
	}
	return nil
}

func possibleLimitedBody(s string, limit int) bool {
	if s == "" || len(s) == limit {
		return true
	}
	// Execute cuts the wire at exactly limit bytes. Only sanitization can
	// shorten it to nonempty text, by re-encoding JSON and bounding it again.
	if value, ok := evidenceJSON(s); ok {
		b, err := json.Marshal(value)
		return err == nil && string(b) == s
	}
	return limit-len(s) < utf8.UTFMax && incompleteJSON(s)
}

func incompleteJSON(s string) bool {
	var raw json.RawMessage
	err := json.NewDecoder(strings.NewReader(s)).Decode(&raw)
	return errors.Is(err, io.ErrUnexpectedEOF)
}

// When sanitizing JSON changes it, JSON encoding may expand otherwise harmless
// characters, e.g. '<'. Execute decodes that full sanitized body, then caps it.
// Only a prefix of a compact, key-sorted JSON object can arise this way.
func clippedObject(s string) (map[string]json.RawMessage, string, bool) {
	if !strings.HasPrefix(s, "{") {
		return nil, "", false
	}
	d := json.NewDecoder(strings.NewReader(s))
	_, _ = d.Token()
	fields := make(map[string]json.RawMessage)
	var last string
	for {
		token, err := d.Token()
		if err != nil {
			return fields, last, errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		}
		key, ok := token.(string)
		if !ok || key <= last && len(fields) > 0 {
			return nil, "", false
		}
		last = key
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return fields, last, errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		}
		fields[key] = value
	}
}

func verifyClippedResponse(p effects.Prepared, c effects.Candidate, fields map[string]json.RawMessage, last string) error {
	switch c.Disposition {
	case "protocol_invalid":
		// A malformed raw body may also reach the exact cap with no decoded data.
		if c.ResponseID == "" && c.ReturnedModel == "" && c.Usage == nil {
			return nil
		}
	case "credential_redacted", "provider_failed", "incomplete", "schema_invalid", "refusal":
	default:
		return errors.New("provider disposition cannot come from a clipped sanitized body")
	}
	known := func(key string) bool { return fields[key] != nil || key < last }
	partial, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	want := decodeResponse(p.Request.Output.Record, effects.Candidate{Response: string(partial), HTTPStatus: c.HTTPStatus})
	err = checkResponseMetadata(want, c, known("id"), known("model"), known("usage"))
	if err != nil && !errors.Is(err, ErrUnverifiableResponseEvidence) {
		return err
	}
	// If all protocol fields are known, a remaining optional extension cannot
	// produce a different parsed outcome. A later bad protocol alias could only
	// have caused an early protocol_invalid with no decoded fields (handled above).
	if known("error") && known("status") && known("output") && known("id") && known("model") {
		if len(want.ResponseID) > 512 || len(want.ReturnedModel) > 512 {
			want.Disposition = "protocol_invalid"
		} else if want.Disposition == "validated" {
			want.Disposition = "credential_redacted"
		}
		if c.Disposition != want.Disposition {
			return errors.New("provider disposition contradicts the retained body prefix")
		}
	}
	return ErrUnverifiableResponseEvidence
}

func checkResponseMetadata(want, c effects.Candidate, checkID, checkModel, checkUsage bool) error {
	var incomplete bool
	for _, field := range []struct {
		check          bool
		body, retained string
	}{
		{checkID, want.ResponseID, c.ResponseID},
		{checkModel, want.ReturnedModel, c.ReturnedModel},
	} {
		if field.check && bounded(field.body, 512) != field.retained {
			if c.Disposition == "validated" || !couldScrubText(field.body, field.retained, 512) {
				return errors.New("provider model or response ID differs from the retained body")
			}
			incomplete = true
		}
	}
	if checkUsage {
		if (want.Usage == nil) != (c.Usage == nil) {
			return errors.New("provider usage presence differs from the retained body")
		}
		if want.Usage != nil {
			if !reflect.DeepEqual(want.Usage.InputTokens, c.Usage.InputTokens) ||
				!reflect.DeepEqual(want.Usage.OutputTokens, c.Usage.OutputTokens) ||
				!reflect.DeepEqual(want.Usage.TotalTokens, c.Usage.TotalTokens) {
				return errors.New("provider token counts differ from the retained body")
			}
			body, validBody := evidenceJSON(string(want.Usage.Details))
			retained, validRetained := evidenceJSON(string(c.Usage.Details))
			if !validBody || !validRetained && c.Usage.Details != nil {
				return errors.New("invalid Responses usage details")
			}
			if !reflect.DeepEqual(body, retained) {
				if c.Disposition == "validated" || !couldScrubValue(body, retained) {
					return errors.New("provider usage details differ from the retained body")
				}
				incomplete = true
			}
		}
	}
	if incomplete {
		return ErrUnverifiableResponseEvidence
	}
	return nil
}

// Keep integer precision, reject duplicate keys and ignore JSON formatting,
// escapes and object order changed by serializing a Candidate for storage.
func evidenceJSON(s string) (any, bool) {
	if uniqueJSON([]byte(s)) != nil {
		return nil, false
	}
	d := json.NewDecoder(strings.NewReader(s))
	d.UseNumber()
	var value any
	err := d.Decode(&value)
	return value, err == nil
}

// A second scrub of metadata can remove strings, keys or values when its work
// budget is exhausted. Such omissions cannot independently confirm a negative
// result. A different scalar or an added object key is still a contradiction.
func couldScrubValue(body, retained any) bool {
	if retained == nil || reflect.DeepEqual(body, retained) {
		return true
	}
	switch original := body.(type) {
	case string:
		value, ok := retained.(string)
		return ok && couldScrubText(original, value, len(value)+1)
	case map[string]any:
		object, ok := retained.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range object {
			prior, present := original[key]
			if !present || !couldScrubValue(prior, value) {
				return false
			}
		}
		return true
	case []any:
		items, ok := retained.([]any)
		if !ok || len(items) != len(original) {
			return false
		}
		for i, item := range items {
			if !couldScrubValue(original[i], item) {
				return false
			}
		}
		return true
	}
	return false
}

func couldScrubText(original, retained string, limit int) bool {
	if retained == "" || bounded(original, limit) == retained {
		return true
	}
	body, ok := evidenceJSON(original)
	if !ok {
		return false // non-JSON text is kept intact or omitted wholesale
	}
	after, ok := evidenceJSON(retained)
	if !ok {
		// A structured metadata string can be shortened by scrubbing and
		// then cut mid-token at the separate correlation-field limit.
		if limit-len(retained) >= utf8.UTFMax || !incompleteJSON(retained) {
			return false
		}
		switch body.(type) {
		case map[string]any:
			return strings.HasPrefix(retained, "{")
		case []any:
			return strings.HasPrefix(retained, "[")
		case string:
			return strings.HasPrefix(retained, `"`)
		}
		return false
	}
	return couldScrubValue(body, after)
}
