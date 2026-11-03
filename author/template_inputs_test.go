package author_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

func TestTemplateInitialInputs(t *testing.T) {
	worker := func(inputs schema.Record) author.Artifact {
		t.Helper()
		a, err := author.Compile(author.Definition{
			Name: "template-inputs", Inputs: inputs,
			Ports: map[string]author.Port{"work": {Peer: "caller", Obligation: "research"}},
			Steps: []author.Step{author.Receive("work", nil), author.Send("work", nil), author.Success()},
		})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	create := func(a author.Artifact, inputs map[string]schema.Value) (author.Template, error) {
		return author.NewTemplate(author.Application{"worker": {Artifact: a, Inputs: inputs}}, "worker", "work")
	}
	zeroInputs := worker(nil)
	explicit, err := create(zeroInputs, map[string]schema.Value{})
	if err != nil {
		t.Fatal(err)
	}
	// Public template and bundle identity on the extracted runtime before
	// omitted zero-input records were accepted.
	const templateID = "4d6cff4429fe02870453dac90ad4ed69512073bbe05a8cb5cb5f5d19e03fca80"
	const bundleSHA = "f2a97e3bb872d6f1b12c0610272eeb760d747bd84b9c5a068a6849f83863418a"
	if explicit.ID() != templateID || fmt.Sprintf("%x", sha256.Sum256(explicit.Bytes())) != bundleSHA {
		t.Fatal("established template identity changed")
	}
	t.Run("omitted zero inputs", func(t *testing.T) {
		implicit, err := create(zeroInputs, nil)
		if err != nil {
			t.Fatal(err)
		}
		if implicit.ID() != explicit.ID() || !bytes.Equal(implicit.Bytes(), explicit.Bytes()) {
			t.Fatal("omitting empty inputs changed the retained template")
		}
	})
	t.Run("required input remains required", func(t *testing.T) {
		requiresInput := worker(schema.Record{"n": {Kind: "int"}})
		for _, missing := range []map[string]schema.Value{nil, {}} {
			if _, err := create(requiresInput, missing); err == nil {
				t.Fatal("accepted a missing required input")
			}
		}
		if _, err := create(requiresInput, map[string]schema.Value{"n": schema.Int(1)}); err != nil {
			t.Fatal(err)
		}
	})
}
