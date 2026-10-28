package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

// scalarHTTP is a controlled deployment transport for the exact {value:int}
// capability. Endpoint validation happens before construction in runCLI.
type scalarHTTP struct{ endpoint string }

func (a scalarHTTP) Prepare(r effects.Request) (effects.Prepared, error) {
	if len(r.Payload) != 1 || r.Payload["value"].Kind != "int" {
		return effects.Prepared{}, fmt.Errorf("scalar input contract")
	}
	raw, err := json.Marshal(map[string]int64{"value": r.Payload["value"].Number})
	return effects.Prepared{Request: r, Protocol: "adaptive/controlled-scalar/v1", Endpoint: a.endpoint, Wire: string(raw), MaxResponseBytes: 8192}, err
}
func (a scalarHTTP) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewBufferString(p.Wire))
	if err != nil {
		return c
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return c
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8193))
	if err != nil {
		return c
	}
	c.HTTPStatus = response.StatusCode
	c.Response = string(raw)
	c.Disposition = "protocol_invalid"
	if response.StatusCode != 200 || len(raw) > 8192 {
		return c
	}
	var body struct {
		Observation *int64 `json:"observation"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&body) != nil || body.Observation == nil || d.Decode(new(any)) != io.EOF {
		return c
	}
	c.Disposition = "validated"
	c.Payload = map[string]schema.Value{"observation": schema.Int(*body.Observation)}
	return c
}
