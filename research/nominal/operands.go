package nominal

import (
	"fmt"
	"maps"
	"slices"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

// Each register has one lifetime sort, even across mutually exclusive paths,
// repeated loop iterations and absent response bindings. The scan checks all
// PCs, including unreachable code. A known sort does not assert that a value is
// currently defined. Missing values disable the same step on both sides.
type registers map[string]string

func handle(declaration string) string { return "handle[" + declaration + "]" }

func (r registers) write(dst, kind string) error {
	if dst == "" {
		return fmt.Errorf("empty register destination")
	}
	if previous := r[dst]; previous != "" && previous != kind {
		return fmt.Errorf("register %q changes sort from %s to %s", dst, previous, kind)
	}
	r[dst] = kind
	return nil
}

func (r registers) read(o executor.Operand) (string, error) {
	if (o.Ref == "") == (o.Literal == nil) {
		return "", fmt.Errorf("expected exactly one literal or reference")
	}
	var kind string
	if o.Literal != nil {
		if !o.Literal.Valid() {
			return "", fmt.Errorf("invalid literal")
		}
		kind = o.Literal.Kind
	} else {
		kind = r[o.Ref]
		if kind == "" {
			return "", fmt.Errorf("register %q has no checked input or writer", o.Ref)
		}
	}
	if o.Offset != nil && kind != "int" {
		return "", fmt.Errorf("offset requires int, got %s", kind)
	}
	return kind, nil
}

func (r registers) expect(o executor.Operand, want string) error {
	got, err := r.read(o)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("operand %q requires %s, got %s", o.Ref, want, got)
	}
	return nil
}

func (r registers) payload(operands map[string]executor.Operand, record schema.Record) error {
	if len(operands) != len(record) {
		return fmt.Errorf("payload fields differ from declared record")
	}
	for _, name := range slices.Sorted(maps.Keys(operands)) {
		f, ok := record[name]
		if !ok {
			return fmt.Errorf("undeclared payload field %q", name)
		}
		if err := r.expect(operands[name], f.Kind); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
	}
	return nil
}

func (r registers) bind(bindings map[string]string, record schema.Record) error {
	for _, field := range slices.Sorted(maps.Keys(bindings)) {
		f, ok := record[field]
		if !ok {
			return fmt.Errorf("undeclared binding %q", field)
		}
		if err := r.write(bindings[field], f.Kind); err != nil {
			return fmt.Errorf("binding %q: %w", field, err)
		}
	}
	return nil
}

func checkOperands(p executor.Program) error {
	if len(p.Author.Effects) > 0 {
		return fmt.Errorf("external capability declarations outside profile")
	}
	r := registers{}
	for name, f := range p.Author.Inputs.Record {
		r[name] = f.Kind
	}
	for pc, in := range p.Code {
		err := checkInstruction(p, pc, in, r)
		if err != nil {
			return fmt.Errorf("pc %d (%s) %s: %w", pc, in.Op, p.Author.Positions[pc], err)
		}
	}
	return nil
}

func checkInstruction(p executor.Program, pc int, in executor.Instruction, r registers) error {
	switch in.Op {
	case "set":
		kind, err := r.read(in.A)
		if err != nil {
			return err
		}
		if (in.Collection != nil || p.Author.Collections[in.Dst].Kind == "list") && kind != "list" {
			return fmt.Errorf("collection assignment requires literal data, got %s", kind)
		}
		return r.write(in.Dst, kind)
	case "add":
		if err := r.expect(in.A, "int"); err != nil {
			return err
		}
		if err := r.expect(in.B, "int"); err != nil {
			return err
		}
		return r.write(in.Dst, "int")
	case "branch":
		kind, err := r.read(in.A)
		if err != nil {
			return err
		}
		return r.expect(in.B, kind)
	case "jump", "loop":
		// These only change the PC. EncodeProgram validates structured
		// foreach/next/loop boundaries including the back edge to next.
		return nil
	case "append":
		if err := r.expect(executor.Ref(in.Dst), "list"); err != nil {
			return err
		}
		return r.payload(in.Updates, p.Author.Collections[in.Dst].Element)
	case "foreach":
		return r.expect(in.A, "list")
	case "next":
		// A row binds only data from foreach's immutable data-only snapshot.
		// The exhausted edge writes nothing; existing registers keep their sort.
		return r.bind(in.Bind, p.Code[in.Target].Collection.Element)
	case "emit":
		return checkEmit(p.Author, in, r)
	case "receive":
		return checkReceive(p.Author, pc, in, r)
	case "end":
		if in.Outcome == "succeeded" {
			for _, name := range slices.Sorted(maps.Keys(p.Author.Outputs.Record)) {
				if err := r.expect(executor.Ref(name), p.Author.Outputs.Record[name].Kind); err != nil {
					return fmt.Errorf("declared output %q: %w", name, err)
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported opcode")
	}
}

func literal(o executor.Operand, want schema.Value) bool {
	return o.Ref == "" && o.Offset == nil && o.Literal != nil && *o.Literal == want
}

func textLiteral(o executor.Operand) (string, bool) {
	if o.Literal == nil {
		return "", false
	}
	s := o.Literal.Text
	return s, literal(o, schema.Text(s))
}

func checkEmit(m *executor.ProgramMetadata, in executor.Instruction, r registers) error {
	e := in.Effect
	decl, ok := textLiteral(e.Obligation)
	if !ok || !literal(e.Generation, schema.Int(1)) {
		return fmt.Errorf("requires literal declaration and generation one")
	}
	switch e.Kind {
	case "invocation":
		inv, ok := m.Invocations[decl]
		if !ok || !literal(e.Target, schema.Text(inv.TemplateID)) || in.Dst == "" {
			return fmt.Errorf("invalid original invocation endpoint or destination")
		}
		if err := r.payload(e.Payload, inv.Input.Record); err != nil {
			return err
		}
		// All operands are evaluated against the pre-commit data, including
		// when one update references another update's destination.
		writes := registers{}
		for _, name := range slices.Sorted(maps.Keys(in.Updates)) {
			kind, err := r.read(in.Updates[name])
			if err != nil {
				return err
			}
			if name == in.Dst || (m.Collections[name].Kind == "list" && kind != "list") {
				return fmt.Errorf("conflicting or non-list update %q", name)
			}
			writes[name] = kind
		}
		for _, name := range slices.Sorted(maps.Keys(writes)) {
			if err := r.write(name, writes[name]); err != nil {
				return err
			}
		}
		return r.write(in.Dst, handle(decl))
	case "message":
		if in.Dst != "" || len(in.Updates) != 0 {
			return fmt.Errorf("message must have no result register or updates")
		}
		for _, port := range m.Ports {
			if port.SendUsed && !port.Local && literal(e.Target, schema.Text(port.Peer)) && port.Obligation == decl {
				return r.payload(e.Payload, port.Send.Record)
			}
		}
		return fmt.Errorf("undeclared send endpoint")
	default:
		return fmt.Errorf("unsupported effect %q", e.Kind)
	}
}

func checkReceive(m *executor.ProgramMetadata, pc int, in executor.Instruction, r registers) error {
	match := in.Match
	decl, ok := textLiteral(match.Obligation)
	if !ok || !literal(match.Generation, schema.Int(1)) {
		return fmt.Errorf("requires literal declaration and generation one")
	}
	observed := m.Receives[pc]
	switch match.Kind {
	case "invocation":
		inv, ok := m.Invocations[decl]
		if !ok || !literal(match.Source, schema.Text(inv.TemplateID)) || match.Provenance != "runtime-invocation" {
			return fmt.Errorf("invalid original invocation outcome endpoint")
		}
		if err := r.expect(match.Operation, handle(decl)); err != nil {
			return fmt.Errorf("original Await: %w", err)
		}
		expected, err := executor.InvocationOutcomeRecord(inv.Output.Record).Retain()
		if err != nil || expected.ID != observed.ID {
			return fmt.Errorf("invocation outcome schema differs from declaration")
		}
		// The reducer deletes every bound response field on absent responses.
		// Its optional register can consequently contain only data or be absent.
	case "input", "message":
		local := match.Kind == "input"
		provenance := "ordinary-message"
		if local {
			provenance = "local-input"
		}
		if match.Provenance != provenance || !literal(match.Operation, schema.Text("")) {
			return fmt.Errorf("invalid port receive provenance or operation")
		}
		found := false
		for _, port := range m.Ports {
			if port.ReceiveUsed && port.Local == local && literal(match.Source, schema.Text(port.Peer)) &&
				port.Obligation == decl && port.Receive.ID == observed.ID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("undeclared receive endpoint")
		}
	default:
		return fmt.Errorf("unsupported receive %q", match.Kind)
	}
	return r.bind(in.Bind, observed.Record)
}
