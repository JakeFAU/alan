//go:build integration

package durabletest

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/effectboundary"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// End after Request is legal. The root captured its original operation before
// admission was refused. A successful source can leave the aggregate pending.
func TestPhase2BSuccessfulSourceCapturesMissingEffect(t *testing.T) {
	source, err := author.Compile(author.Definition{Name: "phase2b-issue-and-end",
		Effects: map[string]author.Effect{"increment": {Capability: effectboundary.Capability, Input: schema.Record{"delta": {Kind: "int"}}, Output: schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}}}},
		Steps:   []author.Step{author.Request("original", "increment", map[string]author.Expr{"delta": author.Integer(1)}, nil), author.Success()},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: source, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	root := rt.RunID{Subject: "root", Generation: 1}
	p.Composition = &rt.CompositionProvision{Roots: []rt.ResourceRoot{{Run: root, Members: []rt.RunID{root}, Capabilities: []string{effectboundary.Capability}, Limits: rt.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	world := filepath.Join(t.TempDir(), "counter.sqlite")
	_, url, _ := phaseResource(t, world, "", "normal", nil)
	s, path := compositionStore(t, p)
	probe := runCompositionProbe(t, s, url, "dispatch.consider")
	cut := probe.cut(t)
	state := publicStep(t, s, root, "")
	if state.Projection.Status != "succeeded" || state.Projection.Data["original"].Text != cut.Intent {
		t.Fatal(state, cut)
	}
	if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, root, true)); err != nil {
		t.Fatal(err)
	}
	probe.finish(t)
	after := inspect(t, path)
	if len(after.Roots) != 1 {
		t.Fatal(after.Roots)
	}
	result := after.Roots[0]
	if result.Pending != "external_outcome" || result.Charges.Effects != 1 || !reflect.DeepEqual(result.UnknownExternal, []string{cut.Intent}) || result.Outcome != nil || worldAt(t, world).Contacts != 0 {
		t.Fatal(result)
	}
	captureAuthority(t, map[string]any{"cut": cut, "after": after, "report": probe.report, "world": worldAt(t, world), "events": probe.events.ordered})
}

func TestPhase2BExactPriorStores(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			source := filepath.Join("..", "..", "docs", "foundational", "phase2a", "evidence", "process-normal", "TestPhase2KnownEvidenceCrashCuts_"+route+"_recorded-cut-local.sqlite.gz")
			archive, err := os.Open(source)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			data, err := gzip.NewReader(archive)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "retained.sqlite")
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(f, data); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			if err = data.Close(); err != nil {
				t.Fatal(err)
			}
			before := inspect(t, path)
			originalKey, originalAttempt := phaseRequest(t, before, route)
			beforeReplay, err := rt.Replay(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, inspect(t, path)) {
				t.Fatal("replay changed pinned earlier state")
			}
			s, err := rt.Open(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var options = rt.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}
			// Neither this known raw receipt nor the recorded adapter result requires a live endpoint/registry.
			report, err := s.Run(t.Context(), options)
			if err != nil {
				t.Fatal(err)
			}
			root := rt.RunID{Subject: "client", Generation: 1}
			if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, root, false)); err != nil {
				t.Fatal(err)
			}
			after := inspect(t, path)
			key, attempt := phaseRequest(t, after, route)
			state, _, _ := projection(after, "client", 1)
			if state.Status != "succeeded" || key != originalKey || attempt != originalAttempt || phaseResultCount(before, route) != 1 || phaseResultCount(after, route) != 1 {
				t.Fatal(state, report)
			}
			afterReplay, err := rt.Replay(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, inspect(t, path)) {
				t.Fatal("replay changed recovered state")
			}
			captureAuthority(t, map[string]any{"archive": source, "before": before, "after": after, "report": report, "replay_before": beforeReplay, "replay_after": afterReplay})
		})
	}
}
