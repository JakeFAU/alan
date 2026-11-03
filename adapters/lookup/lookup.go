// Package lookup performs deterministic bounded token matching over retained bytes.
package lookup

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const MaxSnippetBytes = 2048

func InputSchema() schema.Record { return schema.Record{"query": {Kind: "string", MaxBytes: 512}} }

// ExcludingInputSchema is an optional profile with two exact span exclusions.
// Each excludeN slot contains document, version (content SHA-256), start and end.
// An all-zero slot is unused. Unknown well-formed identities are ignored.
func ExcludingInputSchema() schema.Record {
	r := InputSchema()
	for i := 1; i <= 2; i++ {
		prefix := fmt.Sprintf("exclude%d_", i)
		r[prefix+"document"] = schema.Field{Kind: "string", MaxBytes: 512}
		r[prefix+"version"] = schema.Field{Kind: "string", MaxBytes: 64}
		r[prefix+"start"] = schema.Field{Kind: "int"}
		r[prefix+"end"] = schema.Field{Kind: "int"}
	}
	return r
}

func OutputSchema() schema.Record {
	return schema.Record{"document": {Kind: "string", MaxBytes: 512}, "version": {Kind: "string", MaxBytes: 64}, "snippet": {Kind: "string", MaxBytes: MaxSnippetBytes}, "start": {Kind: "int"}, "end": {Kind: "int"}}
}

type Adapter struct{}

func New() *Adapter { return &Adapter{} }
func (*Adapter) Prepare(r effects.Request) (effects.Prepared, error) {
	protocol, _, err := validateRequest(r)
	return effects.Prepared{Request: effects.Copy(r), Protocol: protocol, MaxResponseBytes: 8192}, err
}

type excludedSpan struct {
	document, version string
	start, end        int64
}

func validateRequest(r effects.Request) (string, []excludedSpan, error) {
	legacy, _ := InputSchema().Retain()
	excluding, _ := ExcludingInputSchema().Retain()
	protocol := "lookup/tokens/v1"
	input := legacy.Record
	// The original public adapter accepted an omitted Input on direct legacy
	// calls. Preserve that exact query-only convenience; new profiles must declare
	// their immutable schema, and any nonzero declaration is checked strictly.
	undeclaredLegacy := r.Input.ID == "" && r.Input.Record == nil
	if !undeclaredLegacy {
		if err := r.Input.Check(); err != nil {
			return protocol, nil, fmt.Errorf("lookup input schema: %w", err)
		}
		switch r.Input.ID {
		case legacy.ID:
		case excluding.ID:
			protocol = "lookup/tokens-excluding/v1"
			input = excluding.Record
		default:
			return protocol, nil, errors.New("lookup input schema mismatch")
		}
	}
	if err := input.Validate(r.Payload); err != nil {
		return protocol, nil, err
	}
	if err := effects.CheckAssets(r.Assets); err != nil {
		return protocol, nil, err
	}
	expected, _ := OutputSchema().Retain()
	if r.Output.ID != expected.ID {
		return protocol, nil, errors.New("lookup output schema mismatch")
	}
	if err := r.Output.Check(); err != nil {
		return protocol, nil, fmt.Errorf("lookup output schema: %w", err)
	}
	if protocol == "lookup/tokens/v1" {
		return protocol, nil, nil
	}
	spans := make([]excludedSpan, 0, 2)
	for i := 1; i <= 2; i++ {
		prefix := fmt.Sprintf("exclude%d_", i)
		span := excludedSpan{
			document: r.Payload[prefix+"document"].Text, version: r.Payload[prefix+"version"].Text,
			start: r.Payload[prefix+"start"].Number, end: r.Payload[prefix+"end"].Number,
		}
		if span == (excludedSpan{}) {
			continue
		}
		decoded, err := hex.DecodeString(span.version)
		validIdentity := span.document != "" && len(decoded) == 32 && err == nil && strings.ToLower(span.version) == span.version
		validRange := span.start >= 0 && span.end > span.start && span.end <= effects.MaxAssetBytes
		if !validIdentity || !validRange {
			return protocol, nil, fmt.Errorf("invalid excluded span %d identity or range", i)
		}
		for _, a := range r.Assets {
			if a.ID != span.document || a.Version != span.version {
				continue
			}
			if span.end > int64(len(a.Content)) {
				return protocol, nil, fmt.Errorf("excluded span %d outside document", i)
			}
			startBoundary := utf8.RuneStart(a.Content[span.start])
			endBoundary := span.end == int64(len(a.Content)) || utf8.RuneStart(a.Content[span.end])
			if !startBoundary || !endBoundary {
				return protocol, nil, fmt.Errorf("excluded span %d splits UTF-8 encoding", i)
			}
		}
		spans = append(spans, span)
	}
	return protocol, spans, nil
}

func isExcluded(spans []excludedSpan, a effects.Asset, start, end int) bool {
	for _, span := range spans {
		if span.document == a.ID && span.version == a.Version && span.start == int64(start) && span.end == int64(end) {
			return true
		}
	}
	return false
}

func tokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}
func (*Adapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "no_match"}
	protocol, excluded, err := validateRequest(p.Request)
	if err != nil {
		c.Disposition = "request_invalid"
		return c
	}
	if p.Protocol != protocol {
		c.Disposition = "protocol_invalid"
		return c
	}
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
			if score > best && !isExcluded(excluded, a, off, stop) {
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
