// Package nominal implements a research-only opaque invocation-handle judgment.
// Callers recheck the exact artifact or template before relying on a certificate.
package nominal

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

const Version = "alan/research/nomcert/v1"

// Certificate identifies the checked scope. ID is the retained program or root
// template ID. Digest additionally binds the judgment version, scope and exact
// public Bytes(), including the entire template bundle when Scope is "template".
type Certificate struct {
	Version, Scope, ID, Digest string
	Programs, Templates        int
}

func certificate(scope, id string, raw []byte, programs, templates int) Certificate {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00", Version, scope)
	h.Write(raw)
	return Certificate{Version: Version, Scope: scope, ID: id, Digest: fmt.Sprintf("%x", h.Sum(nil)), Programs: programs, Templates: templates}
}

// CheckArtifact checks lowered operands with literal declared input values,
// empty cursors and no ambient registers. Dependencies, actual initial values,
// authenticity of supplied observations and resource adequacy are separate.
func CheckArtifact(a author.Artifact) (Certificate, error) {
	raw := a.Bytes()
	p, err := program(a.ID(), raw)
	if err != nil {
		return Certificate{}, err
	}
	if err := checkOperands(p); err != nil {
		return Certificate{}, fmt.Errorf("program %s: %w", a.ID(), err)
	}
	return certificate("artifact", a.ID(), raw, 1, 0), nil
}

func program(id string, raw []byte) (executor.Program, error) {
	var p executor.Program
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	if p.Author == nil {
		return p, fmt.Errorf("requires an author program")
	}
	// These pairs are part of the judgment. Learning a newer executor or
	// compiler version must not automatically expand the v1 proof's domain.
	if !(p.Semantics == "alan/1e/v1" && p.Author.Compiler == "alan/author/v1" ||
		p.Semantics == "alan/1g/v1" && p.Author.Compiler == "alan/author/v2") {
		return p, fmt.Errorf("unsupported semantics %q and compiler %q", p.Semantics, p.Author.Compiler)
	}
	canonical, got, err := executor.EncodeProgram(p)
	if err != nil {
		return p, err
	}
	if got != id || !bytes.Equal(raw, canonical) {
		return p, fmt.Errorf("program bytes or identity differ from retained encoding")
	}
	return p, nil
}

// Mirrors the public Template.Bytes() encoding. The round-trip check below
// refuses extensions or other encodings that this version does not understand.
type bundle struct {
	Descriptor    executor.TemplateDescriptor
	Programs      []executor.Artifact
	Dependencies  []executor.TemplateDescriptor
	Input, Output schema.Retained
}

// CheckTemplate checks an entire closed catalogue of singleton, caller-only
// templates and their actual literal initializers. It checks the binding of each
// declaration to its callee's entry record schemas, including unused declarations.
func CheckTemplate(t author.Template) (Certificate, error) {
	raw := t.Bytes()
	var b bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return Certificate{}, err
	}
	canonical, err := json.Marshal(b)
	if err != nil || !bytes.Equal(raw, canonical) {
		return Certificate{}, fmt.Errorf("unsupported template bundle encoding")
	}
	descriptors := map[string]executor.TemplateDescriptor{}
	for _, d := range append(b.Dependencies, b.Descriptor) {
		if _, ok := descriptors[d.ID]; ok || d.ID == "" {
			return Certificate{}, fmt.Errorf("duplicate or empty template identity %q", d.ID)
		}
		descriptors[d.ID] = d
	}
	programs := map[string]executor.Program{}
	artifacts := map[string]executor.Artifact{}
	for _, a := range b.Programs {
		if _, ok := programs[a.ID]; ok {
			return Certificate{}, fmt.Errorf("duplicate program %s", a.ID)
		}
		p, err := program(a.ID, a.Bytes)
		if err != nil {
			return Certificate{}, err
		}
		if err = checkOperands(p); err != nil {
			return Certificate{}, fmt.Errorf("program %s: %w", a.ID, err)
		}
		programs[a.ID], artifacts[a.ID] = p, a
	}
	ports := map[string]executor.PortDeclaration{}
	usedPrograms := map[string]bool{}
	for _, id := range slices.Sorted(maps.Keys(descriptors)) {
		d := descriptors[id]
		if d.Version != "alan/template/v1" {
			return Certificate{}, fmt.Errorf("template %s: unsupported version %q", id, d.Version)
		}
		member, ok := d.Members[d.EntryMember]
		if !ok || len(d.Members) != 1 || len(d.Capabilities) != 0 {
			return Certificate{}, fmt.Errorf("template %s requires one entry member and no capabilities", id)
		}
		p, ok := programs[member.ProgramID]
		if !ok {
			return Certificate{}, fmt.Errorf("missing program %s", member.ProgramID)
		}
		// Reuse the runtime's identity, complete initial schema/projection and
		// endpoint checks. Additionally exclude any ambient cursor or peer.
		retained, err := executor.RetainTemplate(d, []executor.Artifact{artifacts[member.ProgramID]})
		if err != nil {
			return Certificate{}, fmt.Errorf("template %s: %w", id, err)
		}
		port, ok := p.Author.Ports[d.EntryPort]
		if !reflect.DeepEqual(retained, d) || member.Initial.PC != 0 || member.Initial.Status != "open" ||
			len(member.Initial.Cursors) != 0 || !ok || port.Peer != d.CallerPeer || port.Local || len(p.Author.Ports) != 1 {
			return Certificate{}, fmt.Errorf("template %s requires canonical initialization and one bound caller port", id)
		}
		ports[id], usedPrograms[member.ProgramID] = port, true
	}
	root := ports[b.Descriptor.ID]
	if !reflect.DeepEqual(b.Input, root.Receive) || !reflect.DeepEqual(b.Output, root.Send) {
		return Certificate{}, fmt.Errorf("root entry schema differs from its retained caller port")
	}

	visited := map[string]bool{}
	pending := []string{b.Descriptor.ID}
	for len(pending) > 0 {
		id := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[id] {
			continue
		}
		visited[id] = true
		d := descriptors[id]
		p := programs[d.Members[d.EntryMember].ProgramID]
		children := map[string]bool{}
		for _, name := range slices.Sorted(maps.Keys(p.Author.Invocations)) {
			inv := p.Author.Invocations[name]
			port, ok := ports[inv.TemplateID]
			if !ok || port.Receive.ID != inv.Input.ID || port.Send.ID != inv.Output.ID {
				return Certificate{}, fmt.Errorf("template %s declaration %q: missing dependency or mismatched caller schema", id, name)
			}
			children[inv.TemplateID] = true
		}
		if !slices.Equal(slices.Sorted(maps.Keys(children)), d.Templates) {
			return Certificate{}, fmt.Errorf("template %s catalogue differs from its invocation declarations", id)
		}
		pending = append(pending, d.Templates...)
	}
	if len(visited) != len(descriptors) || len(usedPrograms) != len(programs) {
		return Certificate{}, fmt.Errorf("bundle contains an unowned template or program")
	}
	return certificate("template", b.Descriptor.ID, raw, len(programs), len(descriptors)), nil
}
