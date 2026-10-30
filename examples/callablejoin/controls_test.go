package callablejoin_test

import (
	"github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/runtime"
	"testing"
)

func TestCallableJoinControlIdentities(t *testing.T) {
	c, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	d, err := callablejoin.New(c, substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 4, MaxCreatedRuns: 4, MaxDepth: 3, MaxInFlight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{d.Join.ID(): true}
	for _, kind := range []string{"no-report-success", "no-report-failure", "false-candidate-failed"} {
		artifact, join, err := callablejoin.NewControl(kind, c, d.Successor)
		if err != nil {
			t.Fatal(err)
		}
		if seen[join.ID()] || artifact.Metadata().Name != "phase1n-control-"+kind {
			t.Fatal("control lacks separate identity", kind)
		}
		seen[join.ID()] = true
		_, output, err := join.Schemas()
		if err != nil {
			t.Fatal(err)
		}
		_, positiveOutput, err := d.Join.Schemas()
		if err != nil {
			t.Fatal(err)
		}
		if output.ID != positiveOutput.ID {
			t.Fatal("omission changed public reply contract")
		}
		if _, err := callablejoin.NewOuter(join); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := callablejoin.NewControl("unknown", c, d.Successor); err == nil {
		t.Fatal("unknown control accepted")
	}
}
