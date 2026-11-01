// Package effectboundary contains the public Phase 2A controlled-effect probes.
package effectboundary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const Capability = "phase2a.counter"

// WireRequest uses only the public retained operation key and authored input.
// No executor authorization, generation or activation field is fabricated.
type WireRequest struct {
	Key, Profile string
	Delta        int64
}
type Receipt struct {
	Key, Profile           string
	Delta, Mutation, Total int64
	Process                string
	PID                    int
}

// Counter is trusted deployment code. Mode is retained in Prepared.Model.
// Modes selected-error and unknown apply only to delta=1, leaving the independent
// delta=2 peer able to complete through the same registry and physical slot.
type Counter struct{ Endpoint, Mode string }

func Registry(endpoint, mode string) map[string]effects.Adapter {
	return map[string]effects.Adapter{Capability: Counter{Endpoint: endpoint, Mode: mode}}
}
func (a Counter) Prepare(r effects.Request) (effects.Prepared, error) {
	u, err := url.Parse(a.Endpoint)
	if err != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return effects.Prepared{}, fmt.Errorf("literal loopback HTTP endpoint required")
	}
	mode := a.Mode
	if mode == "" {
		mode = "success"
	}
	if mode != "success" && mode != "selected-error" && mode != "unknown" {
		return effects.Prepared{}, fmt.Errorf("unsupported adapter mode")
	}
	if len(r.Payload) != 1 || r.Payload["delta"].Kind != "int" || (r.Payload["delta"].Number != 1 && r.Payload["delta"].Number != 2) || r.Key == "" {
		return effects.Prepared{}, fmt.Errorf("bounded counter input required")
	}
	raw, err := json.Marshal(WireRequest{Key: r.Key, Profile: "idempotent", Delta: r.Payload["delta"].Number})
	return effects.Prepared{Request: r, Protocol: "phase2a/counter/v1", Endpoint: a.Endpoint, Model: mode, Wire: string(raw), MaxResponseBytes: 8192}, err
}
func (a Counter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "transport_error"}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewBufferString(p.Wire))
	if err != nil {
		c.Response = err.Error()
		return c
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		c.Response = err.Error()
		return c
	}
	defer response.Body.Close()
	c.HTTPStatus = response.StatusCode
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8193))
	if err != nil {
		c.Response = err.Error()
		return c
	}
	c.Disposition = "protocol_invalid"
	if len(raw) > 8192 {
		return c
	}
	c.Response = string(raw)
	if response.StatusCode != http.StatusOK {
		return c
	}
	var receipt Receipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || decoder.Decode(new(any)) != io.EOF || receipt.Key != p.Request.Key || receipt.Profile != "idempotent" || receipt.Delta != p.Request.Payload["delta"].Number || receipt.Mutation < 1 {
		return c
	}
	c.Disposition = "validated"
	c.Payload = map[string]schema.Value{"mutation": schema.Int(receipt.Mutation), "total": schema.Int(receipt.Total)}
	if receipt.Delta == 1 {
		switch p.Model {
		case "selected-error":
			c.Disposition = "selected_error"
		case "unknown":
			c.Disposition = "unknown"
		}
	}
	return c
}
