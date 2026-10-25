// Package lookup performs deterministic bounded token matching over retained bytes.
package lookup

import (
	"context"
	"errors"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxSnippetBytes = 2048

func InputSchema() schema.Record { return schema.Record{"query": {Kind: "string", MaxBytes: 512}} }
func OutputSchema() schema.Record {
	return schema.Record{"document": {Kind: "string", MaxBytes: 512}, "version": {Kind: "string", MaxBytes: 64}, "snippet": {Kind: "string", MaxBytes: MaxSnippetBytes}, "start": {Kind: "int"}, "end": {Kind: "int"}}
}

type Adapter struct{}

func New() *Adapter { return &Adapter{} }
func (*Adapter) Prepare(r effects.Request) (effects.Prepared, error) {
	p := effects.Prepared{Request: effects.Copy(r), Protocol: "lookup/tokens/v1", MaxResponseBytes: 8192}
	if e := InputSchema().Validate(r.Payload); e != nil {
		return p, e
	}
	if e := effects.CheckAssets(r.Assets); e != nil {
		return p, e
	}
	expected, _ := OutputSchema().Retain()
	if r.Output.ID != expected.ID {
		return p, errors.New("lookup output schema mismatch")
	}
	return p, nil
}
func tokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}
func (*Adapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "no_match"}
	query := tokens(p.Request.Payload["query"].Text)
	if len(query) == 0 || len(query) > 32 {
		c.Disposition = "query_invalid"
		return c
	}
	assets := append([]effects.Asset(nil), p.Request.Assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	best := 0
	var chosen effects.Asset
	start, end := 0, 0
	for _, a := range assets {
		for off := 0; off < len(a.Content); {
			if ctx.Err() != nil {
				c.Disposition = "unknown"
				return c
			}
			n := strings.IndexByte(a.Content[off:], '\n')
			if n < 0 {
				n = len(a.Content) - off
			}
			stop := off + n
			if stop-off > MaxSnippetBytes {
				stop = off + MaxSnippetBytes
				for stop > off && !utf8.RuneStart(a.Content[stop]) {
					stop--
				}
			}
			words := map[string]bool{}
			for _, w := range tokens(a.Content[off:stop]) {
				words[w] = true
			}
			score := 0
			seen := map[string]bool{}
			for _, w := range query {
				if words[w] && !seen[w] {
					score++
					seen[w] = true
				}
			}
			if score > best {
				best = score
				chosen = a
				start = off
				end = stop
			}
			off += n + 1
		}
	}
	if best > 0 {
		c.Disposition = "validated"
		c.Payload = map[string]schema.Value{"document": schema.Text(chosen.ID), "version": schema.Text(chosen.Version), "snippet": schema.Text(chosen.Content[start:end]), "start": schema.Int(int64(start)), "end": schema.Int(int64(end))}
	}
	return c
}
