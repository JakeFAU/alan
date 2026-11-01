package receiptquery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

func CounterInput() schema.Record { return schema.Record{"delta": {Kind: "int"}} }
func CounterOutput() schema.Record {
	return schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}}
}
func ReadInput() schema.Record { return schema.Record{"target": {Kind: "string", MaxBytes: 1024}} }
func ReadOutput() schema.Record {
	return schema.Record{"status": {Kind: "string", MaxBytes: 32}, "target": {Kind: "string", MaxBytes: 1024}, "mutation": {Kind: "int"}, "total": {Kind: "int"}, "adopted": {Kind: "bool"}, "adoption": {Kind: "string", MaxBytes: 128}}
}

type Counter struct{ Endpoint, Mode string }
type Reader struct{ Endpoint string }

func checkRequest(r effects.Request, capability string, input, output schema.Record) error {
	in, _ := input.Retain()
	out, _ := output.Retain()
	if r.Capability != capability || namespaceOf(r.Key, r.ID) == "" || !reflect.DeepEqual(r.Input, in) || !reflect.DeepEqual(r.Output, out) || input.Validate(r.Payload) != nil {
		return errors.New("invalid controlled request declaration or identity")
	}
	return nil
}
func (a Counter) Prepare(r effects.Request) (effects.Prepared, error) {
	if err := checkRequest(r, CounterCapability, CounterInput(), CounterOutput()); err != nil {
		return effects.Prepared{}, err
	}
	if _, err := EndpointID(a.Endpoint, "apply"); err != nil {
		return effects.Prepared{}, err
	}
	mode := a.Mode
	if mode == "" {
		mode = "success"
	}
	delta := r.Payload["delta"].Number
	if (mode != "success" && mode != "selected-error" && mode != "unknown") || (delta != 1 && delta != 2) {
		return effects.Prepared{}, errors.New("unsupported counter mode or delta")
	}
	b, _ := json.Marshal(shortMutation{Key: r.Key, Profile: "idempotent", Delta: delta})
	return effects.Prepared{Request: r, Protocol: CounterProtocol, Endpoint: a.Endpoint, Model: mode, MaxResponseBytes: MaxMutationBytes, Wire: string(b)}, nil
}
func CounterPrepared(p effects.Prepared) error {
	expected, err := (Counter{Endpoint: p.Endpoint, Mode: p.Model}).Prepare(p.Request)
	if err != nil {
		return err
	}
	expected = effects.Freeze(expected)
	if !reflect.DeepEqual(p, expected) {
		return errors.New("counter preparation differs from frozen policy")
	}
	return nil
}
func (a Counter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	if p.Digest == "" {
		p = effects.Freeze(p)
	}
	mode := a.Mode
	if mode == "" {
		mode = "success"
	}
	if a.Endpoint != p.Endpoint || mode != p.Model || CounterPrepared(p) != nil {
		return refusal(p)
	}
	c := post(ctx, p, p.Wire)
	if c.Disposition != "protocol_invalid" || c.HTTPStatus != http.StatusOK {
		return c
	}
	var receipt Receipt
	m, _ := DecodeMutation(p.Wire)
	if !canonical([]byte(c.Response), &receipt) || !receipt.matches(m) {
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
func (a Reader) Prepare(r effects.Request) (effects.Prepared, error) {
	if err := checkRequest(r, ReadCapability, ReadInput(), ReadOutput()); err != nil {
		return effects.Prepared{}, err
	}
	if _, err := EndpointID(a.Endpoint, "lookup"); err != nil {
		return effects.Prepared{}, err
	}
	return effects.Prepared{Request: r, Protocol: ReadProtocol, Endpoint: a.Endpoint, MaxResponseBytes: MaxSnapshotBytes}, nil
}
func refusal(p effects.Prepared) effects.Candidate {
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "protocol_invalid"}
}

// Execute refuses ordinary direct execution. Only an authenticated binding and
// a distinct ordinal reserved by the dispatcher can enter ExecuteRead.
func (Reader) Execute(_ context.Context, p effects.Prepared) effects.Candidate { return refusal(p) }
func (a Reader) ExecuteRead(ctx context.Context, p effects.Prepared, q Query) effects.Candidate {
	expected, err := a.Prepare(p.Request)
	if err != nil || !reflect.DeepEqual(p, effects.Freeze(expected)) {
		return refusal(p)
	}
	ledger, _ := EndpointID(p.Endpoint, "lookup")
	if q.Check() != nil || q.LedgerID != ledger || q.QueryID != p.Request.ID || q.QueryKey != p.Request.Key || q.TargetID != p.Request.Payload["target"].Text {
		return refusal(p)
	}
	b, _ := json.Marshal(q)
	c := post(ctx, p, string(b))
	if c.Disposition != "protocol_invalid" || c.HTTPStatus != http.StatusOK {
		return c
	}
	s, err := DecodeSnapshot(q, c.Response)
	if err != nil {
		return c
	}
	c.Disposition = "validated"
	var mutation, total int64
	if s.Receipt != nil {
		mutation, total = s.Receipt.Mutation, s.Receipt.Total
	}
	c.Payload = map[string]schema.Value{"status": schema.Text(s.Status), "target": schema.Text(q.TargetID), "mutation": schema.Int(mutation), "total": schema.Int(total), "adopted": schema.Bool(false), "adoption": schema.Text("unassessed")}
	return c
}
func post(ctx context.Context, p effects.Prepared, body string) effects.Candidate {
	c := refusal(p)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, strings.NewReader(body))
	if err != nil {
		return c
	}
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = nil
	tr := &http.Transport{DisableKeepAlives: true, DisableCompression: true}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		c.Disposition = "transport_error"
		c.Response = err.Error()
		return c
	}
	defer resp.Body.Close()
	c.HTTPStatus = resp.StatusCode
	b, err := io.ReadAll(io.LimitReader(resp.Body, int64(p.MaxResponseBytes)+1))
	if len(b) <= p.MaxResponseBytes && utf8.Valid(b) {
		c.Response = string(b)
	}
	if err != nil {
		c.Disposition = "transport_error"
	}
	return c
}
