package evidence_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/schema"
)

func TestExcludingHelperIsAnExternalPublicConsumer(t *testing.T) {
	// Legacy callers cannot silently supply the new fields. The new factory's
	// distinct port contract must be chosen explicitly.
	request := map[string]schema.Value{}
	for k, f := range evidence.ExcludingRequestSchema() {
		request[k] = schema.Value{Kind: f.Kind}
	}
	request["question"] = schema.Text("match")
	if evidence.RequestSchema().Validate(request) == nil {
		t.Fatal("legacy helper accepted optional fields implicitly")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("testdata/excluding_consumer.go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	module := "module external.example/excludingconsumer\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + root + "\n"
	for name, data := range map[string][]byte{"main.go": source, "go.mod": []byte(module)} {
		if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(t.Context(), "go", "run", "-mod=mod", ".")
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("external excluding helper: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "public exclusion reached next span") {
		t.Fatal(string(output))
	}
}

func TestLegacyFactoryRetainsDevelopmentArtifactIdentity(t *testing.T) {
	component, err := evidence.New("investigation/helper1", "investigation", "helper1", evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
	if err != nil {
		t.Fatal(err)
	}
	// IDs retained in the original development-v2 store, before this extension.
	for name, id := range map[string]string{
		"investigation/helper1":           "9c8ec36dacdccde46c960b0d4eef835d2a36cafa4b766c20635a537650a1ff66",
		"investigation/helper1/retrieval": "cfa4f77a2253555ed9eee8469744812ff42d4d65b5b3ae6ce21c1a6bc9cb4df2",
	} {
		if component.Instances[name].Artifact.ID() != id {
			t.Fatal("legacy artifact changed", name, component.Instances[name].Artifact.ID())
		}
	}
}
