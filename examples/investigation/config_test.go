package investigation_test

import (
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/investigation"
)

func TestProvisionRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*investigation.Config)
	}{
		{"missing model", func(c *investigation.Config) { c.Model = "" }},
		{"wrong application", func(c *investigation.Config) { c.Application = "other" }},
		{"negative revisions", func(c *investigation.Config) { c.MaxRevisions = -1 }},
		{"two revisions", func(c *investigation.Config) { c.MaxRevisions = 2 }},
		{"zero tokens", func(c *investigation.Config) { c.MaxOutputTokens = 0 }},
		{"large response", func(c *investigation.Config) { c.MaxResponseBytes = 65537 }},
		{"zero timeout", func(c *investigation.Config) { c.RequestTimeout = 0 }},
		{"large timeout", func(c *investigation.Config) { c.RequestTimeout = time.Hour }},
		{"arbitrary endpoint", func(c *investigation.Config) { c.Endpoint = "https://not-a-provider.example/" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := investigation.DefaultConfig()
			c.Model = "controlled"
			tc.change(&c)
			if _, err := investigation.Provision("question", c, nil); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	c := investigation.DefaultConfig()
	c.Model = "controlled"
	for _, question := range []string{"", " \t\n"} {
		if _, err := investigation.Provision(question, c, nil); err == nil {
			t.Fatal("empty question accepted")
		}
	}
}

// Branch order must not vary with Go map iteration: artifacts are durable IDs.
func TestFactoryArtifactIdentityIsStable(t *testing.T) {
	first, err := investigation.New("study", "caller", "job", investigation.Budgets{MaxRevisions: 1})
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		next, err := investigation.New("study", "caller", "job", investigation.Budgets{MaxRevisions: 1})
		if err != nil {
			t.Fatal(err)
		}
		for name, instance := range first.Instances {
			if next.Instances[name].Artifact.ID() != instance.Artifact.ID() {
				t.Fatal("nondeterministic artifact", name)
			}
		}
	}
}
