// Package openai implements one nonstreaming Responses HTTP request per effect.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Config struct {
	APIKey, Model, Endpoint           string
	MaxOutputTokens, MaxResponseBytes int
}
type Adapter struct{ config Config }

func New(c Config) (*Adapter, error) {
	if c.Endpoint == "" {
		c.Endpoint = "https://api.openai.com/v1/responses"
	}
	u, e := url.Parse(c.Endpoint)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid provider endpoint")
	}
	production := u.Scheme == "https" && u.Host == "api.openai.com" && u.Path == "/v1/responses"
	ip := net.ParseIP(u.Hostname())
	controlled := u.Scheme == "http" && ip != nil && ip.IsLoopback()
	if !production && !controlled {
		return nil, errors.New("provider requires official HTTPS or literal loopback test endpoint")
	}
	if c.APIKey == "" || strings.ContainsAny(c.APIKey, "\r\n") || !utf8.ValidString(c.APIKey) || c.Model == "" || !utf8.ValidString(c.Model) || c.MaxOutputTokens < 1 || c.MaxOutputTokens > 32768 || c.MaxResponseBytes < 1 || c.MaxResponseBytes > effects.MaxWireBytes {
		return nil, errors.New("explicit model, runtime credential and bounded limits required")
	}
	return &Adapter{config: c}, nil
}
func (a *Adapter) Prepare(r effects.Request) (effects.Prepared, error) {
	p := effects.Prepared{Request: effects.Copy(r), Protocol: "openai/responses/v1", Endpoint: a.config.Endpoint, Model: a.config.Model, MaxOutputTokens: a.config.MaxOutputTokens, MaxResponseBytes: a.config.MaxResponseBytes}
	if e := r.Input.Check(); e != nil {
		return p, e
	}
	if e := r.Input.Record.Validate(r.Payload); e != nil {
		return p, e
	}
	instructions, ok := r.Payload["instructions"]
	if !ok || instructions.Kind != "string" || len(r.Payload) < 2 {
		return p, errors.New("Responses requires instructions string and at least one input field")
	}
	inputText := ""
	if input, ok := r.Payload["input"]; ok && len(r.Payload) == 2 && input.Kind == "string" {
		inputText = input.Text
	} else {
		record := map[string]any{}
		for name, value := range r.Payload {
			if name == "instructions" {
				continue
			}
			switch value.Kind {
			case "string":
				record[name] = value.Text
			case "int":
				record[name] = value.Number
			case "bool":
				record[name] = value.Flag
			}
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			return p, errors.New("record input encoding failed")
		}
		inputText = string(encoded)
	}
	if e := r.Output.Check(); e != nil {
		return p, e
	}
	props := map[string]any{}
	required := []string{}
	for k, f := range r.Output.Record {
		prop := map[string]any{"type": map[string]string{"string": "string", "int": "integer", "bool": "boolean"}[f.Kind]}
		if len(f.Enum) > 0 {
			prop["enum"] = f.Enum
		}
		props[k] = prop
		required = append(required, k)
	}
	sort.Strings(required)
	body := map[string]any{"model": p.Model, "instructions": instructions.Text, "input": []map[string]string{{"role": "user", "content": inputText}}, "max_output_tokens": p.MaxOutputTokens, "store": false, "truncation": "disabled", "text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "alan_" + r.Output.ID, "strict": true, "schema": map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}}}}
	// Provider schema name is at most 64 characters; full schema identity is retained in Request.
	body["text"].(map[string]any)["format"].(map[string]any)["name"] = "alan_" + r.Output.ID[:59]
	raw, e := json.Marshal(body)
	if e != nil {
		return p, errors.New("request encoding failed")
	}
	p.Wire = string(raw)
	if strings.Contains(p.Wire, a.config.APIKey) {
		return p, errors.New("credential occurs in retained request")
	}
	p = effects.Freeze(p)
	return p, effects.CheckPrepared(p)
}
func (a *Adapter) Execute(ctx context.Context, p effects.Prepared) (c effects.Candidate) {
	c = effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	sanitizer := evidenceSanitizer{secret: a.config.APIKey, bytesLeft: 8*p.MaxResponseBytes + 8192, nodesLeft: 65536}
	evidenceChanged := false
	cleanText := func(value string) string {
		clean, changed := sanitizer.text(value, 0)
		evidenceChanged = evidenceChanged || changed
		return clean
	}
	defer func() {
		if len(c.ResponseID) > 512 || len(c.ReturnedModel) > 512 {
			c.Disposition = "protocol_invalid"
			c.Payload = nil
		}
		c.RequestID = bounded(cleanText(c.RequestID), 512)
		c.ResponseID = bounded(cleanText(c.ResponseID), 512)
		c.ReturnedModel = bounded(cleanText(c.ReturnedModel), 512)
		c.Response = bounded(c.Response, p.MaxResponseBytes) // already passed the same sanitizer before decoding
		for key, value := range c.Payload {
			_ = cleanText(key)
			if value.Kind == "string" {
				_ = cleanText(value.Text)
			}
		}
		if c.Usage != nil && c.Usage.Details != nil {
			details := cleanText(string(c.Usage.Details))
			if json.Valid([]byte(details)) {
				c.Usage.Details = json.RawMessage(details)
			} else {
				c.Usage.Details = nil
			}
		}
		if evidenceChanged {
			c.Payload = nil
			if c.Disposition == "validated" {
				c.Disposition = "credential_redacted"
			}
		}
	}()
	// Reconstructing from retained bytes prevents configuration from changing model/input.
	if p.Protocol != "openai/responses/v1" || p.Endpoint != a.config.Endpoint || effects.CheckPrepared(p) != nil {
		return c
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewBufferString(p.Wire))
	if e != nil {
		return c
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	// Use a fixed ASCII digest-sized identifier, without interpreting it as deduplication.
	req.Header.Set("X-Client-Request-Id", clientID(p.Request.Key))
	// A fresh transport and non-rewindable body disable net/http's connection retry path.
	req.GetBody = nil
	transport := &http.Transport{DisableKeepAlives: true, TLSHandshakeTimeout: 10 * time.Second}
	defer transport.CloseIdleConnections()
	client := http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(req)
	if e != nil {
		return c
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, int64(p.MaxResponseBytes)+1))
	if e != nil {
		return c
	}
	c.HTTPStatus = response.StatusCode
	c.RequestID = response.Header.Get("x-request-id")
	if len(raw) > p.MaxResponseBytes {
		raw = raw[:p.MaxResponseBytes]
		c.Response = cleanText(string(raw))
		c.Disposition = "response_limit"
		return c
	}
	c.Response = cleanText(string(raw))
	if !utf8.ValidString(c.Response) {
		c.Response = strings.ToValidUTF8(c.Response, "�")
		c.Disposition = "protocol_invalid"
		return c
	}
	if c.HTTPStatus < 200 || c.HTTPStatus >= 300 {
		c.Disposition = "http_rejected"
		return c
	}
	var body struct {
		ID, Model, Status string
		Error             json.RawMessage
		Output            []struct {
			Type, Role string
			Status     json.RawMessage
			Content    []struct{ Type, Text, Refusal string }
		}
		Usage *struct {
			Input  *int64 `json:"input_tokens"`
			Output *int64 `json:"output_tokens"`
			Total  *int64 `json:"total_tokens"`
		}
	}
	if uniqueJSON([]byte(c.Response)) != nil || exactProtocolKeys([]byte(c.Response)) != nil || json.Unmarshal([]byte(c.Response), &body) != nil {
		c.Disposition = "protocol_invalid"
		return c
	}
	c.ResponseID = body.ID
	c.ReturnedModel = body.Model
	if body.Usage != nil {
		var full map[string]json.RawMessage
		_ = json.Unmarshal([]byte(c.Response), &full)
		c.Usage = &effects.Usage{InputTokens: body.Usage.Input, OutputTokens: body.Usage.Output, TotalTokens: body.Usage.Total, Details: full["usage"]}
	}
	if body.Status == "failed" || len(body.Error) > 0 && string(body.Error) != "null" {
		c.Disposition = "provider_failed"
		return c
	}
	if body.Status == "incomplete" {
		c.Disposition = "incomplete"
		return c
	}
	if body.Status != "completed" {
		c.Disposition = "protocol_invalid"
		return c
	}
	var texts []string
	invalid := false
	refusal := false
	for _, item := range body.Output {
		// Omission is supported for compatibility; present status must be terminal.
		// RawMessage distinguishes an omitted field from explicit null.
		if len(item.Status) > 0 {
			var status string
			if json.Unmarshal(item.Status, &status) != nil || status != "completed" {
				c.Disposition = "protocol_invalid"
				return c
			}
		}
		switch item.Type {
		case "reasoning":
		case "message":
			if item.Role != "assistant" {
				invalid = true
			}
			for _, part := range item.Content {
				switch part.Type {
				case "refusal":
					refusal = true
				case "output_text":
					texts = append(texts, part.Text)
				default:
					invalid = true
				}
			}
		default:
			invalid = true
		}
	}
	if refusal {
		c.Disposition = "refusal"
		return c
	}
	if invalid || len(texts) != 1 {
		c.Disposition = "protocol_invalid"
		return c
	}
	payload, e := parsePayload(texts[0], p.Request.Output.Record)
	if e != nil {
		c.Disposition = "schema_invalid"
		return c
	}
	if string(raw) != c.Response {
		c.Disposition = "credential_redacted"
		return c
	}
	c.Payload = payload
	c.Disposition = "validated"
	return c
}
func parsePayload(text string, record schema.Record) (map[string]schema.Value, error) {
	var raw map[string]json.RawMessage
	if uniqueJSON([]byte(text)) != nil || json.Unmarshal([]byte(text), &raw) != nil || len(raw) != len(record) {
		return nil, errors.New("invalid output")
	}
	values := map[string]schema.Value{}
	for k, f := range record {
		b, ok := raw[k]
		if !ok || string(b) == "null" {
			return nil, errors.New("missing output field")
		}
		v := schema.Value{Kind: f.Kind}
		var e error
		switch f.Kind {
		case "string":
			e = json.Unmarshal(b, &v.Text)
		case "int":
			e = json.Unmarshal(b, &v.Number)
		case "bool":
			e = json.Unmarshal(b, &v.Flag)
		}
		if e != nil {
			return nil, e
		}
		values[k] = v
	}
	return values, record.Validate(values)
}

func bounded(s string, max int) string {
	if max < 0 {
		return ""
	}
	s = strings.ToValidUTF8(s, "�")
	if len(s) <= max {
		return s
	}
	end := max
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end]
}
