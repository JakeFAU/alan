package executor

import (
	"testing"

	"github.com/JakeFAU/alan/effects"
)

func TestContinuationScopeSeparatesIdentityFromApprovedAction(t *testing.T) {
	original := effects.Freeze(effects.Prepared{
		Request:  effects.Request{ID: "first", Key: "namespace-first", Capability: "weather", Effect: "forecast"},
		Protocol: "http/v1", Endpoint: "first.example", Wire: "GET /forecast", MaxResponseBytes: 512,
	})
	next := original
	next.Request.ID, next.Request.Key = "second", "namespace-second"
	if continuationScope(next) != continuationScope(original) {
		t.Fatal("a separately metered request may have a new original identity")
	}
	for _, change := range []func(*effects.Prepared){
		func(p *effects.Prepared) { p.Endpoint = "other.example" },
		func(p *effects.Prepared) { p.Request.Capability = "other" },
		func(p *effects.Prepared) { p.Wire += "?other=1" },
		func(p *effects.Prepared) { p.MaxResponseBytes++ },
	} {
		changed := next
		change(&changed)
		if continuationScope(changed) == continuationScope(original) {
			t.Fatal("fresh identity widened the approved action")
		}
	}
}
