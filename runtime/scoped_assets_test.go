//go:build integration

package runtime_test

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type assetRecorder struct {
	prepared []effects.Request
	executed int
}

func (a *assetRecorder) Prepare(r effects.Request) (effects.Prepared, error) {
	a.prepared = append(a.prepared, r)
	lines := make([]string, 0, len(r.Assets))
	for _, doc := range r.Assets {
		lines = append(lines, doc.Content)
	}
	return effects.Prepared{Request: r, Protocol: "fixture/join-lines", Wire: strings.Join(lines, "\n"), MaxResponseBytes: 128}, nil
}
func (a *assetRecorder) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	a.executed++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated",
		Payload: map[string]schema.Value{"text": schema.Text(p.Wire)}}
}

type assetSizer struct {
	prepared []effects.Request
	executed int
}

func (a *assetSizer) Prepare(r effects.Request) (effects.Prepared, error) {
	a.prepared = append(a.prepared, r)
	size := 0
	for _, doc := range r.Assets {
		size += len(doc.Content)
	}
	return effects.Prepared{Request: r, Protocol: "fixture/byte-count", Wire: strconv.Itoa(size), MaxResponseBytes: 128}, nil
}
func (a *assetSizer) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	a.executed++
	size, err := strconv.ParseInt(p.Wire, 10, 64)
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "protocol_invalid"}
	if err == nil {
		c.Disposition = "validated"
		c.Payload = map[string]schema.Value{"bytes": schema.Int(size)}
	}
	return c
}

func scopeProvision(t *testing.T, scopes map[string][]string, assets []effects.Asset) runtime.Provision {
	t.Helper()
	app := author.Application{}
	for _, capability := range []string{"first", "second"} {
		field := "text"
		output := schema.Record{"text": {Kind: "string", MaxBytes: 128}}
		if capability == "second" {
			field = "bytes"
			output = schema.Record{"bytes": {Kind: "int"}}
		}
		prog, err := author.Compile(author.Definition{
			Name: capability, Effects: map[string]author.Effect{"read": {Capability: capability, Output: output}},
			Steps: []author.Step{
				author.Request("op", "read", nil, nil),
				author.Await("read", "op", map[string]string{field: "answer", schema.ResultValid: "ok"}),
				author.IfEqual(author.Var("ok"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()}),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		app[capability] = author.Instance{Artifact: prog}
	}
	p, err := app.Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = assets
	p.AssetScopes = scopes
	return p
}

func retainedAsset(t *testing.T, id, data string) effects.Asset {
	t.Helper()
	a, err := effects.NewAsset(id, "test", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestScopedAssetBindings(t *testing.T) {
	a, b, c := retainedAsset(t, "a", "red"), retainedAsset(t, "b", "green"), retainedAsset(t, "c", "blue")
	for _, tt := range []struct {
		name          string
		scopes        map[string][]string
		first, second []effects.Asset
		text          string
		bytes         int64
	}{
		{"none", nil, nil, nil, "", 0},
		{"disjoint", map[string][]string{"first": {"a"}, "second": {"b"}}, []effects.Asset{a}, []effects.Asset{b}, "red", 5},
		{"overlap", map[string][]string{"first": {"b", "a"}, "second": {"c", "b"}}, []effects.Asset{a, b}, []effects.Asset{b, c}, "red\ngreen", 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), "case.sqlite")
			s, err := runtime.Create(ctx, path, scopeProvision(t, tt.scopes, []effects.Asset{c, b, a}), runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = runtime.Open(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			r1, r2 := new(assetRecorder), new(assetSizer)
			_, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 24, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"first": r1, "second": r2}})
			if err = errors.Join(err, s.Close()); err != nil {
				t.Fatal(err)
			}
			if len(r1.prepared) != 1 || r1.executed != 1 || !slices.Equal(r1.prepared[0].Assets, tt.first) ||
				len(r2.prepared) != 1 || r2.executed != 1 || !slices.Equal(r2.prepared[0].Assets, tt.second) {
				t.Fatalf("wrong invocation or selected documents: first=%+v second=%+v", r1, r2)
			}
			snap, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
			if err != nil || len(snap.Effects) != 2 {
				t.Fatalf("inspect: %v", err)
			}
			for _, e := range snap.Effects {
				want, protocol, wire, answer := tt.first, "fixture/join-lines", tt.text, schema.Text(tt.text)
				if e.Capability == "second" {
					want, protocol, wire, answer = tt.second, "fixture/byte-count", strconv.FormatInt(tt.bytes, 10), schema.Int(tt.bytes)
				}
				prepared := e.AdapterAttempt.Prepared
				if !slices.Equal(prepared.Request.Assets, want) || prepared.Protocol != protocol || prepared.Wire != wire {
					t.Fatalf("wrong frozen selection or preparation: %+v", prepared)
				}
				source, ok := snap.Run(e.Source)
				if !ok || source.Projection.Status != "succeeded" || !source.Projection.Data["answer"].Equal(answer) {
					t.Fatalf("source %s did not consume its adapter's answer", e.Source.Subject)
				}
			}
			if _, err = runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestScopedAssetSelectionIsHostAuthority(t *testing.T) {
	doc := retainedAsset(t, "doc", "private")
	path := filepath.Join(t.TempDir(), "case.sqlite")
	ctx := t.Context()
	limits := runtime.DefaultLimits()
	for _, ids := range [][]string{{"missing"}, {"doc", "doc"}} {
		_, err := runtime.Create(ctx, path, scopeProvision(t, map[string][]string{"first": ids}, []effects.Asset{doc}), limits)
		if err == nil {
			t.Fatalf("accepted invalid binding %q", ids)
		}
	}
	p := scopeProvision(t, map[string][]string{"first": {"doc"}}, []effects.Asset{doc})
	p.Permissions = slices.DeleteFunc(p.Permissions, func(g runtime.Permission) bool { return g.Target == "first" })
	s, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	first, second := new(assetRecorder), new(assetSizer)
	_, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 32, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"first": first, "second": second}})
	if err = errors.Join(err, s.Close()); err != nil {
		t.Fatal(err)
	}
	if len(first.prepared) != 0 || first.executed != 0 || len(second.prepared) != 1 || len(second.prepared[0].Assets) != 0 {
		t.Fatalf("denied or unrelated capability received data: first=%+v second=%+v", first, second)
	}
	snap, err := runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, effect := range snap.Effects {
		if effect.Capability == "first" && (effect.AdapterAttempt.Allowed || effect.AdapterAttempt.Reason != "grant_denied" || effect.AdapterResult.Disposition != "grant_denied") {
			t.Fatalf("expected retained grant refusal: %+v", effect)
		}
	}
	if _, err = runtime.Replay(ctx, path, limits); err != nil {
		t.Fatal(err)
	}
}

func TestScopedAssetNoninterference(t *testing.T) {
	original := retainedAsset(t, "a", "original")
	unrelated := retainedAsset(t, "b", "outside scope")
	run := func(assets []effects.Asset) effects.Prepared {
		t.Helper()
		path := filepath.Join(t.TempDir(), "case.sqlite")
		s, err := runtime.Create(t.Context(), path, scopeProvision(t, map[string][]string{"first": {"a"}}, assets), runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		first := new(assetRecorder)
		_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 32, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"first": first, "second": new(assetSizer)}})
		if err = errors.Join(err, s.Close()); err != nil {
			t.Fatal(err)
		}
		snap, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		for _, effect := range snap.Effects {
			if effect.Capability == "first" {
				if !effect.AdapterAttempt.Allowed || first.executed != 1 {
					t.Fatal("selected operation ineligible")
				}
				p := effect.AdapterAttempt.Prepared
				p.Request.Key = "" // independent stores have unique namespace-bound external keys
				return effects.Freeze(p)
			}
		}
		t.Fatal("no first effect")
		return effects.Prepared{}
	}
	before := run([]effects.Asset{original})
	for _, outside := range []effects.Asset{unrelated, retainedAsset(t, "b", "modified")} {
		after := run([]effects.Asset{outside, original})
		if before.Digest != after.Digest {
			t.Fatal("unrelated asset changed prepared request")
		}
	}
	changed := retainedAsset(t, "a", "changed")
	after := run([]effects.Asset{changed, unrelated})
	if after.Digest == before.Digest || !slices.Equal(after.Request.Assets, []effects.Asset{changed}) {
		t.Fatal("selected asset version absent from prepared request")
	}
}

type scopedWeather struct {
	*openmeteo.Adapter
	executed int
}

func (w *scopedWeather) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	w.executed++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "fixture_declines"}
}

func TestScopedWeatherWithUnrelatedDocument(t *testing.T) {
	prog, err := author.Compile(author.Definition{Name: "weather", Effects: map[string]author.Effect{"read": {Capability: "weather", Input: openmeteo.InputSchema(), Output: openmeteo.OutputSchema()}}, Steps: []author.Step{
		author.Request("op", "read", map[string]author.Expr{openmeteo.Latitude: author.String("26.93"), openmeteo.Longitude: author.String("-80.08"), openmeteo.Units: author.String("fahrenheit")}, nil),
		author.Await("read", "op", nil), author.Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"wx": {Artifact: prog}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = []effects.Asset{retainedAsset(t, "doc", "unrelated")}
	actual, err := openmeteo.New(openmeteo.DefaultEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	w := &scopedWeather{Adapter: actual}
	path := filepath.Join(t.TempDir(), "weather.sqlite")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Run(t.Context(), runtime.RunOptions{Adapters: map[string]effects.Adapter{"weather": w}, MaxSteps: 8, MaxDispatch: 8})
	if err = errors.Join(err, s.Close()); err != nil {
		t.Fatal(err)
	}
	if w.executed != 1 {
		t.Fatal("weather adapter did not pass its real preparation")
	}
	if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestScopedAssetDeniedReceiptProtocols(t *testing.T) {
	endpoint := "http://127.0.0.1:1/phase2c/" + strings.Repeat("a", 64)
	for _, tc := range []struct {
		effect   author.Effect
		payload  map[string]author.Expr
		adapter  effects.Adapter
		protocol string
	}{
		{author.Effect{Capability: receiptquery.ReadCapability, Input: receiptquery.ReadInput(), Output: receiptquery.ReadOutput()}, map[string]author.Expr{"target": author.String("missing")}, receiptquery.Reader{Endpoint: endpoint + "/lookup"}, receiptquery.ReadProtocol},
		{author.Effect{Capability: receiptquery.CounterCapability, Input: receiptquery.CounterInput(), Output: receiptquery.CounterOutput()}, map[string]author.Expr{"delta": author.Integer(1)}, receiptquery.Counter{Endpoint: endpoint + "/apply"}, receiptquery.CounterProtocol},
	} {
		t.Run(tc.effect.Capability, func(t *testing.T) {
			capability := tc.effect.Capability
			program, err := author.Compile(author.Definition{Effects: map[string]author.Effect{"call": tc.effect}, Steps: []author.Step{
				author.Request("op", "call", tc.payload, nil), author.Await("call", "op", nil), author.Success(),
			}})
			if err != nil {
				t.Fatal(err)
			}
			p, err := (author.Application{"agent": {Artifact: program}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			agent := runtime.RunID{Subject: "agent", Generation: 1}
			p.Composition = &runtime.CompositionProvision{Roots: []runtime.ResourceRoot{{Run: agent, Members: []runtime.RunID{agent}, Capabilities: []string{capability}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}}}
			p.Reconciliation = &runtime.ReconciliationProfile{}
			p.Permissions = nil
			doc := retainedAsset(t, "doc", "selected")
			p.Assets = []effects.Asset{doc}
			p.AssetScopes = map[string][]string{capability: {"doc"}}
			path := filepath.Join(t.TempDir(), "receipt.sqlite")
			ctx, limits := t.Context(), runtime.DefaultLimits()
			store, err := runtime.Create(ctx, path, p, limits)
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.Run(ctx, runtime.RunOptions{MaxSteps: 8, MaxDispatch: 8, Adapters: map[string]effects.Adapter{capability: tc.adapter}})
			if err = errors.Join(err, store.Close()); err != nil {
				t.Fatal(err)
			}
			cut, err := runtime.InspectExecution(ctx, path, limits)
			if err != nil {
				t.Fatal(err)
			}
			if len(cut.Effects) != 1 {
				t.Fatal(cut.Effects)
			}
			attempt := cut.Effects[0].AdapterAttempt
			if attempt == nil || attempt.Allowed || attempt.Reason != "grant_denied" || attempt.Prepared.Protocol != tc.protocol || !slices.Equal(attempt.Prepared.Request.Assets, []effects.Asset{doc}) {
				t.Fatalf("unexpected controlled refusal %+v", attempt)
			}
			if _, err := runtime.Replay(ctx, path, limits); err != nil {
				t.Fatal(err)
			}
			store, err = runtime.Open(ctx, path, limits)
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
