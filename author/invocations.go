package author

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Template retains ordinary member artifacts, defaults and one caller boundary.
// All accessors return independent copies; no runtime author callback is stored.
type Template struct{ raw string }
type templateBundle struct {
	Descriptor    executor.TemplateDescriptor
	Programs      []runtime.Artifact
	Dependencies  []executor.TemplateDescriptor
	Input, Output schema.Retained
}

func (t Template) bundle() (templateBundle, error) {
	var b templateBundle
	if t.raw == "" {
		return b, fmt.Errorf("empty template")
	}
	err := json.Unmarshal([]byte(t.raw), &b)
	return b, err
}
func (t Template) ID() string    { b, _ := t.bundle(); return b.Descriptor.ID }
func (t Template) Bytes() []byte { return []byte(t.raw) }

func NewTemplate(a Application, entryMember, entryPort string) (Template, error) {
	t := executor.TemplateDescriptor{Members: map[string]executor.TemplateMember{}, EntryMember: entryMember, EntryPort: entryPort}
	entry, ok := a[entryMember]
	if !ok {
		return Template{}, fmt.Errorf("missing entry member")
	}
	port, ok := entry.Artifact.Metadata().Ports[entryPort]
	if !ok || port.Local {
		return Template{}, fmt.Errorf("missing public port")
	}
	t.CallerPeer = port.Peer
	b := templateBundle{Input: port.Receive, Output: port.Send}
	dependencies := map[string]executor.TemplateDescriptor{}
	programs := map[string]runtime.Artifact{}
	children := map[string]bool{}
	capabilities := map[string]bool{}
	for name, instance := range a {
		if instance.Artifact.ID() == "" {
			return Template{}, fmt.Errorf("empty artifact")
		}
		t.Members[name] = executor.TemplateMember{ProgramID: instance.Artifact.ID(),
			Initial: runtime.Projection{Status: "open",
				Data: schema.CloneValues(instance.Inputs)}}
		programs[instance.Artifact.ID()] = runtime.Artifact{ID: instance.Artifact.ID(),
			Semantics: artifactSemantics(instance.Artifact),
			Bytes:     instance.Artifact.Bytes()}
		for _, e := range instance.Artifact.Metadata().Effects {
			capabilities[e.Capability] = true
		}
		for _, e := range instance.Artifact.Metadata().Invocations {
			children[e.TemplateID] = true
		}
		var nested templateCatalogue
		if instance.Artifact.catalogue != "" {
			if err := json.Unmarshal([]byte(instance.Artifact.catalogue), &nested); err != nil {
				return Template{}, err
			}
		}
		for _, d := range nested.Templates {
			dependencies[d.ID] = d
		}
		for _, p := range nested.Programs {
			programs[p.ID] = p
		}
	}
	t.Templates = keys(children)
	t.Capabilities = keys(capabilities)
	for _, id := range keys(programs) {
		b.Programs = append(b.Programs, programs[id])
	}
	var err error
	b.Descriptor, err = executor.RetainTemplate(t, b.Programs)
	if err != nil {
		return Template{}, err
	}
	for _, id := range keys(dependencies) {
		b.Dependencies = append(b.Dependencies, dependencies[id])
	}
	raw, err := json.Marshal(b)
	return Template{raw: string(raw)}, err
}

type InvocationOutcomeBindings struct {
	Status, Reason, ResponsePresent string
	Response                        map[string]string
}

func Invoke(pending, template string, values, updates map[string]Expr) Step {
	return Step{kind: "invoke", destination: pending, declaration: template, values: values, updates: updates}
}
func AwaitInvocation(template, pending string, b InvocationOutcomeBindings) Step {
	bindings := map[string]string{"status": b.Status, "reason": b.Reason, "response_present": b.ResponsePresent}
	for field, dst := range b.Response {
		bindings["response."+field] = dst
	}
	return Step{kind: "await-invocation", declaration: template, request: pending, bindings: bindings}
}

type templateCatalogue struct {
	Templates []executor.TemplateDescriptor
	Programs  []runtime.Artifact
}

func retainInvocations(d Definition, m *executor.ProgramMetadata) (string, error) {
	if len(d.Invocations) == 0 {
		return "", nil
	}
	m.Invocations = map[string]executor.InvocationDeclaration{}
	templates := map[string]executor.TemplateDescriptor{}
	programs := map[string]runtime.Artifact{}
	for _, name := range keys(d.Invocations) {
		if err := schema.CheckIdentifier(name); err != nil {
			return "", err
		}
		b, err := d.Invocations[name].bundle()
		if err != nil {
			return "", err
		}
		m.Invocations[name] = executor.InvocationDeclaration{TemplateID: b.Descriptor.ID, Input: b.Input, Output: b.Output}
		templates[b.Descriptor.ID] = b.Descriptor
		for _, t := range b.Dependencies {
			templates[t.ID] = t
		}
		for _, p := range b.Programs {
			programs[p.ID] = p
		}
	}
	c := templateCatalogue{}
	for _, id := range keys(templates) {
		c.Templates = append(c.Templates, templates[id])
	}
	for _, id := range keys(programs) {
		c.Programs = append(c.Programs, programs[id])
	}
	raw, err := json.Marshal(c)
	return string(raw), err
}

// Schemas returns independent copies of the retained public request and response
// schemas. A zero Template returns an error rather than an empty contract.
func (t Template) Schemas() (input, output schema.Retained, err error) {
	b, err := t.bundle()
	if err != nil {
		return input, output, err
	}
	if err = b.Input.Check(); err != nil {
		return input, output, err
	}
	if err = b.Output.Check(); err != nil {
		return input, output, err
	}
	return b.Input, b.Output, nil
}
