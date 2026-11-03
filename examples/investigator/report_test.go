package main

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestVerifyCitationChecksVersionAndExactUTF8Bytes(t *testing.T) {
	asset, err := effects.NewAsset("minutes", "minutes.txt", []byte("é\nCafé won.\nOther line."))
	if err != nil {
		t.Fatal(err)
	}
	valid := citation{Document: asset.ID, Version: asset.Version, Start: 3, End: 13, Quote: "Café won."}
	if err := verifySpan(asset, valid); err != nil {
		t.Fatalf("valid byte span rejected: %v", err)
	}
	for _, test := range []struct {
		name string
		edit func(*effects.Asset, *citation)
	}{
		{"other document", func(_ *effects.Asset, c *citation) { c.Document = "other" }},
		{"stale version", func(_ *effects.Asset, c *citation) { c.Version = "stale" }},
		{"bytes changed outside span", func(a *effects.Asset, _ *citation) { a.Content += "!" }},
		{"different quote", func(_ *effects.Asset, c *citation) { c.Quote = "Other line." }},
		{"negative start", func(_ *effects.Asset, c *citation) { c.Start = -1 }},
		{"past the end", func(_ *effects.Asset, c *citation) { c.End = math.MaxInt64 }},
		{"empty span", func(_ *effects.Asset, c *citation) { c.End = c.Start; c.Quote = "" }},
		{"splits code point", func(a *effects.Asset, c *citation) {
			c.Start, c.End = 1, 2
			c.Quote = a.Content[1:2]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			doc, span := asset, valid
			test.edit(&doc, &span)
			if err := verifySpan(doc, span); err == nil {
				t.Fatal("unsupported citation accepted")
			}
		})
	}
}

type wrongRangeLookup struct{ *lookup.Adapter }

func (l wrongRangeLookup) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := l.Adapter.Execute(ctx, p)
	c.Payload["end"] = schema.Int(c.Payload["end"].Number + 1)
	return c
}

// A deployment can supply a schema-valid but incorrect lookup. Its result,
// model prompt, model quote, worker reply, and parent finding all agree. Only
// the independent check against the original document bytes rejects it.
func TestAnswerRejectsFalseLookupAfterSuccessfulReplay(t *testing.T) {
	const quote = "Alpha votes were counted."
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, body := fixtureResponse("primary", "supports", quote)
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	p, d := priorDeployment(t, server.URL, [2]string{"alpha", "counted"})
	model, err := d.Provider.adapter("test-credential")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "false-range.sqlite")
	limits := runtime.DefaultLimits()
	store, err := runtime.Create(t.Context(), path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Run(t.Context(), runtime.RunOptions{
		MaxSteps: 256, MaxDispatch: 128, MaxInFlight: 2,
		Adapters: map[string]effects.Adapter{"corpus": wrongRangeLookup{lookup.New()}, "model": model},
	}); err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(t.Context(), path, limits)
	if err != nil || replay.Roots[0].Outcome == nil || replay.Roots[0].Outcome.Status != "succeeded" {
		t.Fatalf("expected a consistent, completed runtime history: %+v, %v", replay.Roots, err)
	}
	if r, err := readReport(t.Context(), path, limits); err == nil || !strings.Contains(err.Error(), "citation outside retained document bytes") || len(r.Findings) != 0 {
		t.Fatalf("schema-valid false range entered the answer: %+v, %v", r, err)
	}
}

func TestAnswerRequiresOriginalSelectedEvidence(t *testing.T) {
	server := responsesServer(t, "opposes")
	db := initializeCase(t, server, 4)
	if err := run(t.Context(), []string{"run", db}, io.Discard); err != nil {
		t.Fatal(err)
	}
	source := func(t *testing.T) runtime.Inspection {
		t.Helper()
		s, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	report, err := reportSnapshot(source(t))
	if err != nil || report.Status != "disagreement" || len(report.Findings) != 2 {
		t.Fatalf("original selected evidence: %+v, %v", report.Findings, err)
	}
	for i, role := range []string{"primary", "counter"} {
		f := report.Findings[i]
		if f.Role != role || f.Citation.Document != caseFacts[role].document || f.Citation.Quote != caseFacts[role].quote {
			t.Fatalf("returned a different original source: %+v", f)
		}
	}
	for _, tc := range []struct {
		name string
		edit func([]map[string]schema.Value)
	}{
		{"other worker's authentic document span", func(f []map[string]schema.Value) {
			for _, key := range []string{"document", "version", "start", "end", "quote"} {
				f[0][key] = f[1][key]
			}
		}},
		{"invocation digest used as operation", func(f []map[string]schema.Value) {
			f[0]["invocation_operation_id"] = schema.Text(report.Workers[0].InvocationID)
		}},
		{"other worker's lookup", func(f []map[string]schema.Value) { f[0]["lookup_id"] = f[1]["lookup_id"] }},
		{"other worker's model", func(f []map[string]schema.Value) { f[0]["model_id"] = f[1]["model_id"] }},
		{"invented interpretation", func(f []map[string]schema.Value) { f[0]["interpretation"] = schema.Text("invented") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := source(t)
			for _, row := range s.Tables["runs"] {
				if row["subject"] != "investigator" {
					continue
				}
				var p runtime.Projection
				if err := json.Unmarshal([]byte(row["projection"].(string)), &p); err != nil {
					t.Fatal(err)
				}
				f, err := p.Data["findings"].Records()
				if err != nil {
					t.Fatal(err)
				}
				tc.edit(f)
				p.Data["findings"] = schema.List(f...)
				raw, err := json.Marshal(p)
				if err != nil {
					t.Fatal(err)
				}
				row["projection"] = string(raw)
			}
			if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
				t.Fatalf("unsupported finding escaped independent check: %+v, %v", r.Findings, err)
			}
		})
	}
	s := source(t)
	observed := s.Tables["composition_outcome_deliveries"][0]["observation_id"]
	for _, row := range s.Tables["observations"] {
		if row["id"] == observed {
			row["consumed_sequence"] = nil
		}
	}
	if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
		t.Fatal("published but unselected outcome counted toward the final answer")
	}
	s = source(t)
	deliveries := s.Tables["composition_outcome_deliveries"]
	deliveries[0]["observation_id"], deliveries[1]["observation_id"] = deliveries[1]["observation_id"], deliveries[0]["observation_id"]
	if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
		t.Fatal("the other selected invocation authenticated the claimed worker")
	}
	s = source(t)
	for _, row := range s.Tables["runs"] {
		if row["subject"] == "investigator" {
			var p runtime.Projection
			if err := json.Unmarshal([]byte(row["projection"].(string)), &p); err != nil {
				t.Fatal(err)
			}
			p.Data["findings"] = schema.List()
			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			row["projection"] = string(raw)
		}
	}
	if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
		t.Fatal("a completed disagreement omitted its authored findings")
	}
}

func TestAccountingSeparatesReservedFromProviderContact(t *testing.T) {
	attempt := &runtime.AdapterAttempt{Allowed: true}
	candidate := effects.Candidate{HTTPStatus: 400, Disposition: "http_rejected"}
	zero := int64(0)
	candidate.Usage = &effects.Usage{OutputTokens: &zero}
	failed := &runtime.AdapterResult{Disposition: "http_rejected", Candidate: candidate}
	account := accounting{}
	known := describeEffect("model-1", attempt, failed)
	account.countModel(known)
	account.countModel(describeEffect("model-2", attempt, nil))
	account.countModel(describeEffect("model-3", &runtime.AdapterAttempt{Reason: "effect_limit"}, nil))
	account.countModel(describeEffect("model-4", nil, nil))
	if account.ModelExecutions != (bounds{Minimum: 1, Maximum: 2}) ||
		account.ProviderContacts != (bounds{Minimum: 1, Maximum: 2}) {
		t.Fatalf("retained decisions misreported as contact: %+v", account)
	}
	if known.Usage == nil || known.Usage.OutputTokens == nil || *known.Usage.OutputTokens != 0 ||
		known.Usage.TotalTokens != nil || known.Usage.InputTokens != nil {
		t.Fatalf("partial usage lost or invented: %+v", known.Usage)
	}
	if describeEffect("unknown", attempt, nil).Usage != nil {
		t.Fatal("unknown attempt acquired reported usage")
	}
	local := &runtime.AdapterResult{Candidate: candidate, Disposition: invalidInterpretation}
	if describeEffect("model-5", attempt, local).State != "local_validation_failed" {
		t.Fatal("local proposal rejection indistinguishable from provider failure")
	}
}
