package evidence_test

import (
	"github.com/JakeFAU/alan/examples/evidence"
	"testing"
	"time"
)

func TestProvisionRejectsUnboundedOrImplicitConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*evidence.Config)
	}{
		{"implicit model", func(c *evidence.Config) { c.Model = "" }},
		{"negative model budget", func(c *evidence.Config) { c.MaxModelRequests = -1 }},
		{"huge lookup budget", func(c *evidence.Config) { c.MaxLookupRequests = 11 }},
		{"zero output", func(c *evidence.Config) { c.MaxOutputTokens = 0 }},
		{"oversize response", func(c *evidence.Config) { c.MaxResponseBytes = 65537 }},
		{"zero timeout", func(c *evidence.Config) { c.RequestTimeout = 0 }},
		{"huge timeout", func(c *evidence.Config) { c.RequestTimeout = time.Hour }},
		{"arbitrary endpoint", func(c *evidence.Config) { c.Endpoint = "https://untrusted.example/responses" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := evidence.DefaultConfig()
			c.Model = "controlled-model"
			tc.mutate(&c)
			if _, err := evidence.Provision("question", c, nil); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
