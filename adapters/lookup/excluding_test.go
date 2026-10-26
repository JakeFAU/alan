package lookup_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

type repeatedSpan struct {
	LegacyRequest   effects.Request           `json:"legacy_request"`
	LegacyCandidate effects.Candidate         `json:"legacy_candidate"`
	InitialSpans    []map[string]schema.Value `json:"initial_spans"`
}

// Removing filtering, weakening tuple identity, or changing lexical scoring
// fails these independently hand-specified spans from the retained live input.
func TestRetainedRepeatedSpanCanBeExcluded(t *testing.T) {
	raw, err := os.ReadFile("testdata/repeated-span.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture repeatedSpan
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	legacy := runLookup(t, fixture.LegacyRequest)
	if !reflect.DeepEqual(legacy.Payload, fixture.LegacyCandidate.Payload) {
		t.Fatal("legacy changed retained result")
	}
	extended := excludingRequest(t, fixture.LegacyRequest)
	for i, span := range fixture.InitialSpans {
		setExclusion(extended.Payload, i+1, span)
	}
	if err = lookup.InputSchema().Validate(extended.Payload); err == nil {
		t.Fatal("legacy schema unexpectedly expresses exclusions")
	}
	got := runLookup(t, extended)
	if got.Disposition != "validated" || got.Payload["document"].Text != "phase1f-cedar" || got.Payload["start"].Number != 244 || got.Payload["end"].Number != 390 {
		t.Fatalf("next eligible span=%+v", got)
	}
	if err = exclusionMonitor(extended, got); err != nil {
		t.Fatal(err)
	}
	// This mutant ignores the declared input but still returns a valid ordinary
	// citation. The SAME input/output monitor must reject it for exclusion alone.
	mutant := legacy
	if err = lookup.OutputSchema().Validate(mutant.Payload); err != nil {
		t.Fatal("mutant masked by output schema", err)
	}
	rejection := exclusionMonitor(extended, mutant)
	if rejection == nil {
		t.Fatal("input/output monitor accepted an excluded exact span")
	}
	record, _ := json.Marshal(map[string]any{"case": "ignored-exclusions-mutant", "input": extended, "output": mutant, "rejection": rejection.Error(), "positive": got})
	t.Logf("INVESTIGATION %s", record)
}

func TestExclusionIdentityAndBounds(t *testing.T) {
	a, err := effects.NewAsset("doc", "fixture", []byte("α match\nβ match\nγ match\n"))
	if err != nil {
		t.Fatal(err)
	}
	legacy := effects.Request{ID: "i", Key: "k", Input: mustRetain(t, lookup.InputSchema()), Output: mustRetain(t, lookup.OutputSchema()), Payload: map[string]schema.Value{"query": schema.Text("match")}, Assets: []effects.Asset{a}}
	initial := runLookup(t, legacy)
	if initial.Payload["start"].Number != 0 || initial.Payload["end"].Number != 8 {
		t.Fatal(initial)
	}
	for _, tc := range []struct {
		name            string
		edit            func(map[string]schema.Value)
		wantStart       int64
		wantDisposition string
	}{
		{"zero", func(map[string]schema.Value) {}, 0, "validated"},
		{"one", func(p map[string]schema.Value) { setExclusion(p, 1, initial.Payload) }, 9, "validated"},
		{"duplicate", func(p map[string]schema.Value) {
			setExclusion(p, 1, initial.Payload)
			setExclusion(p, 2, initial.Payload)
		}, 9, "validated"},
		{"two", func(p map[string]schema.Value) {
			setExclusion(p, 1, initial.Payload)
			setExclusion(p, 2, span(a, 9, 17))
		}, 18, "validated"},
		{"unknown-document", func(p map[string]schema.Value) {
			setExclusion(p, 1, initial.Payload)
			p["exclude1_document"] = schema.Text("absent")
		}, 0, "validated"},
		{"unknown-version", func(p map[string]schema.Value) {
			setExclusion(p, 1, initial.Payload)
			p["exclude1_version"] = schema.Text(strings.Repeat("0", 64))
		}, 0, "validated"},
		{"noncandidate-subspan", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 0, 2)) }, 0, "validated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := excludingRequest(t, legacy)
			tc.edit(r.Payload)
			got := runLookup(t, r)
			if got.Disposition != tc.wantDisposition || got.Payload["start"].Number != tc.wantStart {
				t.Fatalf("got %+v", got)
			}
			if err := exclusionMonitor(r, got); err != nil {
				t.Fatal(err)
			}
			if tc.name == "zero" && !reflect.DeepEqual(got.Payload, initial.Payload) {
				t.Fatal("zero slots changed legacy result")
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(map[string]schema.Value)
	}{
		{"partial-document", func(p map[string]schema.Value) { p["exclude1_document"] = schema.Text("doc") }},
		{"partial-version", func(p map[string]schema.Value) { p["exclude1_version"] = schema.Text(a.Version) }},
		{"partial-offset", func(p map[string]schema.Value) { p["exclude1_end"] = schema.Int(8) }},
		{"bad-version", func(p map[string]schema.Value) {
			setExclusion(p, 1, initial.Payload)
			p["exclude1_version"] = schema.Text("not-a-hash")
		}},
		{"negative", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, -1, 8)) }},
		{"empty", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 0, 0)) }},
		{"reversed", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 9, 8)) }},
		{"out-of-range", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 0, 999)) }},
		{"unknown-oversize", func(p map[string]schema.Value) {
			setExclusion(p, 1, span(a, 0, effects.MaxAssetBytes+1))
			p["exclude1_document"] = schema.Text("absent")
		}},
		{"mid-rune-start", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 1, 8)) }},
		{"mid-rune-end", func(p map[string]schema.Value) { setExclusion(p, 1, span(a, 0, 1)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := excludingRequest(t, legacy)
			tc.edit(r.Payload)
			if _, err := lookup.New().Prepare(r); err == nil {
				t.Fatal("invalid tuple accepted")
			}
		})
	}
	t.Run("all-excluded", func(t *testing.T) {
		short, _ := effects.NewAsset("short", "fixture", []byte("match\nmatch\n"))
		r := excludingRequest(t, legacy)
		r.Assets = []effects.Asset{short}
		setExclusion(r.Payload, 1, span(short, 0, 5))
		setExclusion(r.Payload, 2, span(short, 6, 11))
		got := runLookup(t, r)
		if got.Disposition != "no_match" {
			t.Fatal(got)
		}
		if err := exclusionMonitor(r, got); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDeclaredSchemaAndProtocol(t *testing.T) {
	r := effects.Request{ID: "i", Key: "k", Input: mustRetain(t, lookup.InputSchema()), Output: mustRetain(t, lookup.OutputSchema()), Payload: map[string]schema.Value{"query": schema.Text("match")}}
	for _, tc := range []struct {
		name     string
		request  effects.Request
		protocol string
	}{
		{"legacy", r, "lookup/tokens/v1"},
		{"excluding", excludingRequest(t, r), "lookup/tokens-excluding/v1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := lookup.New().Prepare(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if p.Protocol != tc.protocol {
				t.Fatal("wrong protocol", p.Protocol)
			}
			wrongProfile := "lookup/tokens/v1"
			if tc.protocol == wrongProfile {
				wrongProfile = "lookup/tokens-excluding/v1"
			}
			p.Protocol = wrongProfile
			if got := lookup.New().Execute(t.Context(), p); got.Disposition != "protocol_invalid" {
				t.Fatal("accepted another recognized profile's protocol", got)
			}
			p.Protocol = "lookup/unrecognized/v1"
			if got := lookup.New().Execute(t.Context(), p); got.Disposition != "protocol_invalid" {
				t.Fatal("accepted wrong protocol", got)
			}
			changed := effects.Copy(tc.request)
			changed.Input.ID = "forged"
			if _, err = lookup.New().Prepare(changed); err == nil {
				t.Fatal("forged input ID accepted")
			}
			changed = effects.Copy(tc.request)
			changed.Input.Record["query"] = schema.Field{Kind: "string", MaxBytes: 511}
			if _, err = lookup.New().Prepare(changed); err == nil {
				t.Fatal("mutated immutable input accepted")
			}
			changed = effects.Copy(tc.request)
			changed.Output.Record["document"] = schema.Field{Kind: "string", MaxBytes: 511}
			if _, err = lookup.New().Prepare(changed); err == nil {
				t.Fatal("mutated immutable output accepted")
			}
		})
	}
	extended := excludingRequest(t, r)
	extended.Input = r.Input
	if _, err := lookup.New().Prepare(extended); err == nil {
		t.Fatal("legacy declaration accepted excluding payload")
	}
	extended = excludingRequest(t, r)
	extended.Input = schema.Retained{}
	if _, err := lookup.New().Prepare(extended); err == nil {
		t.Fatal("excluding payload accepted without declaration")
	}
}

func TestExcludingRetainsUTF8TruncatedSpanBoundary(t *testing.T) {
	a, _ := effects.NewAsset("long", "fixture", []byte("hit "+strings.Repeat("é", 1023)+"\nhit next\n"))
	r := effects.Request{ID: "i", Key: "k", Input: mustRetain(t, lookup.InputSchema()), Output: mustRetain(t, lookup.OutputSchema()), Payload: map[string]schema.Value{"query": schema.Text("hit")}, Assets: []effects.Asset{a}}
	first := runLookup(t, r)
	if first.Payload["end"].Number != 2048 {
		t.Fatal("legacy snippet boundary changed", first)
	}
	extended := excludingRequest(t, r)
	setExclusion(extended.Payload, 1, first.Payload)
	got := runLookup(t, extended)
	if got.Payload["snippet"].Text != "hit next" || got.Payload["start"].Number != 2051 {
		t.Fatal(got)
	}
	if err := exclusionMonitor(extended, got); err != nil {
		t.Fatal(err)
	}
}

func span(a effects.Asset, start, end int64) map[string]schema.Value {
	return map[string]schema.Value{"document": schema.Text(a.ID), "version": schema.Text(a.Version), "start": schema.Int(start), "end": schema.Int(end)}
}
func setExclusion(p map[string]schema.Value, n int, s map[string]schema.Value) {
	for _, k := range []string{"document", "version", "start", "end"} {
		p[fmt.Sprintf("exclude%d_%s", n, k)] = s[k]
	}
}
func excludingRequest(t *testing.T, r effects.Request) effects.Request {
	t.Helper()
	r = effects.Copy(r)
	r.Input = mustRetain(t, lookup.ExcludingInputSchema())
	for k, f := range lookup.ExcludingInputSchema() {
		if k != "query" {
			r.Payload[k] = schema.Value{Kind: f.Kind}
		}
	}
	return r
}
func runLookup(t *testing.T, r effects.Request) effects.Candidate {
	t.Helper()
	p, err := lookup.New().Prepare(r)
	if err != nil {
		t.Fatal(err)
	}
	return lookup.New().Execute(context.Background(), p)
}

// Independently compares declared tuples to public output; it does not call an
// adapter helper or infer exclusions from participant names or history.
func exclusionMonitor(r effects.Request, c effects.Candidate) error {
	if c.Disposition != "validated" {
		return nil
	}
	if err := lookup.OutputSchema().Validate(c.Payload); err != nil {
		return err
	}
	for i := 1; i <= 2; i++ {
		prefix := fmt.Sprintf("exclude%d_", i)
		if r.Payload[prefix+"document"].Text == "" {
			continue
		}
		if r.Payload[prefix+"document"] == c.Payload["document"] && r.Payload[prefix+"version"] == c.Payload["version"] && r.Payload[prefix+"start"] == c.Payload["start"] && r.Payload[prefix+"end"] == c.Payload["end"] {
			return fmt.Errorf("output repeats excluded span %d", i)
		}
	}
	return nil
}

func TestExcludingPreservesLexicalTermLimit(t *testing.T) {
	request := effects.Request{ID: "i", Key: "k", Input: mustRetain(t, lookup.InputSchema()),
		Output: mustRetain(t, lookup.OutputSchema()), Payload: map[string]schema.Value{"query": schema.Text(strings.Repeat("match ", 33))}}
	for _, r := range []effects.Request{request, excludingRequest(t, request)} {
		if got := runLookup(t, r); got.Disposition != "query_invalid" {
			t.Fatal("term limit changed", got)
		}
	}
}
