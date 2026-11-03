package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
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
	source := func(t *testing.T) *runtime.ExecutionSnapshot {
		t.Helper()
		s, err := runtime.InspectExecution(t.Context(), db, runtime.DefaultLimits())
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
		edit func(*testing.T, map[string]schema.Value)
	}{
		{"agreement claimed for opposite verdicts", func(_ *testing.T, d map[string]schema.Value) {
			d["completion"] = schema.Text("completed")
		}},
		{"disagreement gap omitted", func(_ *testing.T, d map[string]schema.Value) {
			d["gaps"] = schema.List()
		}},
		{"invented question assigned to both workers", func(t *testing.T, d map[string]schema.Value) {
			gaps, err := d["gaps"].Records()
			if err != nil || len(gaps) == 0 {
				t.Fatal("missing genuine disagreement gap", err)
			}
			gaps[len(gaps)-1]["question"] = schema.Text("An invented question?")
			d["gaps"] = schema.List(gaps...)
		}},
		{"counter worker suppressed as a plausible partial answer", func(t *testing.T, d map[string]schema.Value) {
			rows, err := d["findings"].Records()
			if err != nil || len(rows) != 2 {
				t.Fatal("missing genuine findings", err)
			}
			d["counter_accepted"], d["counter_assessment"] = schema.Bool(false), schema.Text("insufficient")
			d["completion"], d["findings"] = schema.Text("partial"), schema.List(rows[0])
			d["gaps"] = schema.List(map[string]schema.Value{
				"role": schema.Text("counter"), "reason": schema.Text("insufficient"), "question": d["question"],
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := source(t)
			parent, ok := s.Run(runtime.RunID{Subject: "investigator", Generation: 1})
			if !ok {
				t.Fatal("missing parent")
			}
			tc.edit(t, parent.Projection.Data)
			if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
				t.Fatalf("contradictory authored answer escaped verification: %+v, %v", r, err)
			}
		})
	}
	t.Run("invented follow-up copied into worker outcome and parent answer", func(t *testing.T) {
		s := source(t)
		parent, ok := s.Run(runtime.RunID{Subject: "investigator", Generation: 1})
		if !ok {
			t.Fatal("missing parent")
		}
		data := parent.Projection.Data
		call, ok := s.Invocation(data["counter_call"].Text)
		if !ok || call.Outcome == nil || call.Outcome.Response["unresolved"].Text != "" {
			t.Fatal("expected counter worker's original response without a follow-up")
		}
		const question = "Who invented this new question?"
		value := schema.Text(question)
		call.Outcome.Response["unresolved"] = value
		data["counter_unresolved"], data["counter_question"] = value, value
		gaps, err := data["gaps"].Records()
		if err != nil || len(gaps) != 1 {
			t.Fatal("expected only disagreement gap", err)
		}
		data["gaps"] = schema.List(map[string]schema.Value{
			"role": schema.Text("counter"), "reason": schema.Text("model_question"), "question": value,
		}, gaps[0])
		if r, err := reportSnapshot(s); err == nil || !strings.Contains(err.Error(), "field unresolved differs from the original model result") || len(r.Findings) != 0 {
			t.Fatalf("model did not return the purported follow-up: %+v, %v", r, err)
		}
	})
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
			parent, ok := s.Run(runtime.RunID{Subject: "investigator", Generation: 1})
			if !ok {
				t.Fatal("missing parent")
			}
			f, err := parent.Projection.Data["findings"].Records()
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(f)
			parent.Projection.Data["findings"] = schema.List(f...)
			if r, err := reportSnapshot(s); err == nil || len(r.Findings) != 0 {
				t.Fatalf("unsupported finding escaped independent check: %+v, %v", r.Findings, err)
			}
		})
	}
	// These are protocol relationships now owned by the runtime. Attack the
	// actual durable evidence on isolated copies, before a typed view is built.
	for _, mutation := range []string{"publication not selected", "other published observation"} {
		t.Run(mutation, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mutant.sqlite")
			for _, suffix := range []string{"", "-wal"} {
				b, err := os.ReadFile(db + suffix)
				if os.IsNotExist(err) && suffix == "-wal" {
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path+suffix, b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			writer, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()
			tx, err := writer.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if mutation == "publication not selected" {
				result, err := tx.Exec("UPDATE observations SET consumed_sequence=NULL WHERE id=(SELECT observation_id FROM composition_outcome_deliveries ORDER BY operation_id LIMIT 1)")
				if err != nil {
					t.Fatal(err)
				}
				if n, err := result.RowsAffected(); err != nil || n != 1 {
					t.Fatal("failed to remove one selection", n, err)
				}
			} else {
				// Exchange the real deliveries while preserving both unique IDs.
				if _, err := tx.Exec(`CREATE TEMP TABLE original_deliveries AS SELECT * FROM composition_outcome_deliveries;
					DELETE FROM composition_outcome_deliveries;
					INSERT INTO composition_outcome_deliveries(operation_id, observation_id)
					SELECT a.operation_id, b.observation_id FROM original_deliveries a
					JOIN original_deliveries b ON a.operation_id != b.operation_id`); err != nil {
					t.Fatal(err)
				}
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if report, err := readReport(t.Context(), path, runtime.DefaultLimits()); !errors.Is(err, runtime.ErrCorrupt) || len(report.Findings) != 0 {
				t.Fatal("expected corrupt parent selection without findings", err)
			}
		})
	}
	s := source(t)
	parent, ok := s.Run(runtime.RunID{Subject: "investigator", Generation: 1})
	if !ok {
		t.Fatal("missing parent")
	}
	parent.Projection.Data["findings"] = schema.List()
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
	// The inspector supplies the adapter execution bounds. These independent
	// values exercise the application's distinct provider-contact presentation.
	report := func(id string, a *runtime.AdapterAttempt, r *runtime.AdapterResult, min, max int) effectReport {
		e := &runtime.ExecutionEffect{OperationID: id, AdapterAttempt: a, AdapterResult: r}
		e.Dispatch.Minimum, e.Dispatch.Maximum = min, new(max)
		if r != nil {
			e.Usage = r.Candidate.Usage
		}
		return describeEffect(e)
	}
	account := accounting{}
	known := report("model-1", attempt, failed, 1, 1)
	account.countModel(known)
	account.countModel(report("model-2", attempt, nil, 0, 1))
	account.countModel(report("model-3", &runtime.AdapterAttempt{Reason: "effect_limit"}, nil, 0, 0))
	account.countModel(report("model-4", nil, nil, 0, 0))
	if account.ModelExecutions != (bounds{Minimum: 1, Maximum: 2}) ||
		account.ProviderContacts != (bounds{Minimum: 1, Maximum: 2}) {
		t.Fatalf("retained decisions misreported as contact: %+v", account)
	}
	if known.Usage == nil || known.Usage.OutputTokens == nil || *known.Usage.OutputTokens != 0 ||
		known.Usage.TotalTokens != nil || known.Usage.InputTokens != nil {
		t.Fatalf("partial usage lost or invented: %+v", known.Usage)
	}
	if report("unknown", attempt, nil, 0, 1).Usage != nil {
		t.Fatal("unknown attempt acquired reported usage")
	}
	local := &runtime.AdapterResult{Candidate: candidate, Disposition: invalidInterpretation}
	if report("model-5", attempt, local, 1, 1).State != "local_validation_failed" {
		t.Fatal("local proposal rejection indistinguishable from provider failure")
	}
}
