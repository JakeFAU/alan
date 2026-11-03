//go:build integration

package runtime_test

import (
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This ordinary author intentionally emits before its approval receive. Call
// Commit directly to bypass every scheduler and CLI check.
func TestContinuationRawCommitRequiresPriorSelection(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) {
		target, err := author.Compile(author.Definition{Name: "unguarded ordering", Ports: map[string]author.Port{
			"consent": author.ContinuationPort("approve"),
		}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 16}}}},
			Steps: []author.Step{author.Request("h", "read", map[string]author.Expr{"station": author.String("north")}, nil), author.Receive("consent", nil), author.Success()},
		})
		if err != nil {
			t.Fatal(err)
		}
		p.Programs = append(p.Programs, runtime.Artifact{ID: target.ID(), Semantics: author.SemanticVersion, Bytes: target.Bytes()})
		for i := range p.Runs {
			if p.Runs[i].ID == meterNext {
				p.Runs[i].ProgramID = target.ID()
			}
		}
	})
	ctx := t.Context()
	before, err := f.owner.Snapshot(ctx, meterNext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.owner.Commit(ctx, before.Authority(), ""); err == nil {
		t.Fatal("raw Commit created a protected operation with no approval")
	}
	after, err := f.owner.Snapshot(ctx, meterNext)
	if err != nil || after.Authority() != before.Authority() {
		t.Fatalf("refused commit changed authority: %v, %v", after, err)
	}
	cut := meterCut(t, f.path)
	if len(cut.Effects) != 0 || meterSpent(cut) != 0 || f.meter.calls.Load() != 0 {
		t.Fatal("failed raw commit changed effect or expenditure")
	}
	if _, err = runtime.Replay(ctx, f.path, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
