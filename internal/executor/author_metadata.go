package executor

import (
	"fmt"
	"unicode/utf8"

	recordschema "github.com/JakeFAU/alan/schema"
)

const AuthorSemanticVersion = "alan/1e/v1"
const AuthorCompilerVersion = "alan/author/v1"

// ProgramMetadata is immutable compiler evidence retained in the program bytes.
type ProgramMetadata struct {
	Compiler        string
	Name            string
	Inputs, Outputs recordschema.Retained
	Templates       map[string]string
	Positions       []string
	Labels          map[string]int
	Effects         map[string]EffectDeclaration
	Ports           map[string]PortDeclaration
	Receives        map[int]recordschema.Retained
}
type EffectDeclaration struct {
	Capability    string
	Input, Output recordschema.Retained
}
type PortDeclaration struct {
	Peer, Obligation      string
	Send, Receive         recordschema.Retained
	Local                 bool
	SendUsed, ReceiveUsed bool
}

func supportedSemantics(s string) bool { return s == SemanticVersion || s == AuthorSemanticVersion }
func validateMetadata(p Program) error {
	if p.Semantics == SemanticVersion {
		if p.Author != nil {
			return ErrVersion
		}
		return nil
	}
	m := p.Author
	if m == nil || m.Compiler != AuthorCompilerVersion {
		return ErrVersion
	}
	if m.Name != "" {
		if err := recordschema.CheckIdentifier(m.Name); err != nil {
			return fmt.Errorf("definition name: %w", err)
		}
	}
	for name, value := range m.Templates {
		if err := recordschema.CheckIdentifier(name); err != nil {
			return fmt.Errorf("template name: %w", err)
		}
		if !utf8.ValidString(name) || !utf8.ValidString(value) {
			return fmt.Errorf("invalid UTF-8 template %q", name)
		}
	}
	if len(m.Positions) != len(p.Code) {
		return fmt.Errorf("author position map length")
	}
	for _, s := range []recordschema.Retained{m.Inputs, m.Outputs} {
		if err := s.Check(); err != nil {
			return err
		}
	}
	for name, e := range m.Effects {
		for _, id := range []string{name, e.Capability} {
			if err := recordschema.CheckIdentifier(id); err != nil {
				return fmt.Errorf("effect: %w", err)
			}
		}
		if _, err := recordschema.ResultRecord(e.Output.Record); err != nil {
			return err
		}
		if name == "" || e.Capability == "" {
			return fmt.Errorf("invalid effect declaration")
		}
		for _, s := range []recordschema.Retained{e.Input, e.Output} {
			if err := s.Check(); err != nil {
				return err
			}
		}
	}
	for name, port := range m.Ports {
		for _, id := range []string{name, port.Peer, port.Obligation} {
			if err := recordschema.CheckIdentifier(id); err != nil {
				return fmt.Errorf("port: %w", err)
			}
		}
		if name == "" || port.Peer == "" || port.Obligation == "" {
			return fmt.Errorf("invalid port declaration")
		}
		for _, s := range []recordschema.Retained{port.Send, port.Receive} {
			if err := s.Check(); err != nil {
				return err
			}
		}
	}
	contracts := map[string]string{}
	for _, port := range m.Ports {
		if port.ReceiveUsed {
			key := fmt.Sprintf("%t/%q/%q", port.Local, port.Peer, port.Obligation)
			if previous := contracts[key]; previous != "" && previous != port.Receive.ID {
				return fmt.Errorf("ambiguous receive contract")
			}
			contracts[key] = port.Receive.ID
		}
	}
	for pc, in := range p.Code {
		if in.Op == "receive" {
			if _, ok := m.Receives[pc]; !ok {
				return fmt.Errorf("missing receive schema at %d", pc)
			}
		}
	}
	for pc, s := range m.Receives {
		if pc < 0 || pc >= len(p.Code) || p.Code[pc].Op != "receive" {
			return fmt.Errorf("invalid receive schema position")
		}
		if err := s.Check(); err != nil {
			return err
		}
	}
	return nil
}

func validateAuthorInitial(p Program, projection Projection) error {
	if p.Author == nil {
		return nil
	}
	if projection.PC != 0 {
		return fmt.Errorf("author initial position must be zero")
	}
	return p.Author.Inputs.Record.Validate(projection.Data)
}
func validateAuthorOutput(p Program, data map[string]Value) error {
	if p.Author == nil {
		return nil
	}
	output := map[string]Value{}
	for name := range p.Author.Outputs.Record {
		v, ok := data[name]
		if !ok {
			return fmt.Errorf("missing output %q", name)
		}
		output[name] = v
	}
	return p.Author.Outputs.Record.Validate(output)
}
func authorAdmissionValid(p Program, d AdmissionDescriptor) bool {
	if p.Author == nil || d.Kind != "input" && d.Kind != "message" {
		return true
	}
	if d.Operation != "" || d.Source.Generation != 1 {
		return false
	}
	for _, port := range p.Author.Ports {
		if port.ReceiveUsed && port.Local == (d.Kind == "input") && port.Peer == d.Source.Subject && port.Obligation == d.Obligation && port.Receive.Record.Validate(d.Payload) == nil {
			return true
		}
	}
	return false
}
