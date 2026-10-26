package author

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

type compiler struct {
	program  executor.Program
	labels   map[string]int
	jumps    map[int]string
	records  map[int]schema.Record
	requests map[int]string
	await    map[int]string
	visits   int
	fields   map[string]schema.Field
}

// Compile validates and lowers a definition. Its result retains all recovery inputs.
func Compile(d Definition) (Artifact, error) {
	c := compiler{program: executor.Program{Semantics: SemanticVersion}, labels: map[string]int{}, jumps: map[int]string{}, records: map[int]schema.Record{}, requests: map[int]string{}, await: map[int]string{}, fields: map[string]schema.Field{}}
	m := &executor.ProgramMetadata{Compiler: CompilerVersion, Name: d.Name, Templates: map[string]string{}, Effects: map[string]executor.EffectDeclaration{}, Ports: map[string]executor.PortDeclaration{}, Receives: map[int]schema.Retained{}}
	for name, value := range d.Templates {
		m.Templates[name] = value
	}
	m.Labels = c.labels
	c.program.Author = m
	var err error
	if m.Inputs, err = d.Inputs.Retain(); err != nil {
		return Artifact{}, fmt.Errorf("inputs: %w", err)
	}
	if m.Outputs, err = d.Outputs.Retain(); err != nil {
		return Artifact{}, fmt.Errorf("outputs: %w", err)
	}
	for _, name := range keys(d.Effects) {
		e := d.Effects[name]
		if name == "" || e.Capability == "" {
			return Artifact{}, fmt.Errorf("effect %q: missing capability", name)
		}
		input, err := e.Input.Retain()
		if err != nil {
			return Artifact{}, fmt.Errorf("effect %q input: %w", name, err)
		}
		_, err = schema.ResultRecord(e.Output)
		if err != nil {
			return Artifact{}, fmt.Errorf("effect %q output: %w", name, err)
		}
		output, err := e.Output.Retain()
		if err != nil {
			return Artifact{}, fmt.Errorf("effect %q output: %w", name, err)
		}
		m.Effects[name] = executor.EffectDeclaration{Capability: e.Capability, Input: input, Output: output}
	}
	for _, name := range keys(d.Ports) {
		p := d.Ports[name]
		if name == "" || p.Peer == "" || p.Obligation == "" || p.Local && (p.Peer != "local" || len(p.Send) > 0) {
			return Artifact{}, fmt.Errorf("port %q: invalid boundary", name)
		}
		send, err := p.Send.Retain()
		if err != nil {
			return Artifact{}, err
		}
		receive, err := p.Receive.Retain()
		if err != nil {
			return Artifact{}, err
		}
		m.Ports[name] = executor.PortDeclaration{Peer: p.Peer, Obligation: p.Obligation, Send: send, Receive: receive, Local: p.Local}
	}
	usesCollections, err := definitionHasCollections(d)
	if err != nil {
		return Artifact{}, err
	}
	if usesCollections {
		c.program.Semantics = executor.CollectionSemanticVersion
		m.Compiler = executor.CollectionCompilerVersion
	}
	if len(d.Collections) > 0 {
		m.Collections = map[string]schema.Field{}
	}
	for _, name := range keys(d.Collections) {
		if err := schema.CheckIdentifier(name); err != nil {
			return Artifact{}, err
		}
		f, err := d.Collections[name].Retain()
		if err != nil || f.Kind != "list" {
			return Artifact{}, fmt.Errorf("collection %q: invalid list declaration: %v", name, err)
		}
		if _, ok := d.Inputs[name]; ok {
			return Artifact{}, fmt.Errorf("collection duplicates input %q", name)
		}
		m.Collections[name] = f
		if _, ok := d.Values[name]; !ok {
			if err := c.append(executor.Instruction{Op: "set", Dst: name, A: executor.Lit(schema.List())}, "collection:"+name); err != nil {
				return Artifact{}, err
			}
		}
	}
	for _, name := range keys(d.Values) {
		if err := schema.CheckIdentifier(name); err != nil {
			return Artifact{}, fmt.Errorf("value: %w", err)
		}
		if _, exists := d.Inputs[name]; exists {
			return Artifact{}, fmt.Errorf("value %q duplicates input", name)
		}
		if err := c.append(executor.Instruction{Op: "set", Dst: name, A: executor.Lit(d.Values[name])}, "value:"+name); err != nil {
			return Artifact{}, err
		}
	}
	if err = c.expand(d.Steps, "steps", nil, 0); err != nil {
		return Artifact{}, err
	}
	for pc, label := range c.jumps {
		target, ok := c.labels[label]
		if !ok {
			return Artifact{}, c.err(pc, "missing label %q", label)
		}
		if target <= pc {
			return Artifact{}, c.err(pc, "unbounded backward branch to %q", label)
		}
		if target >= len(c.program.Code) {
			return Artifact{}, c.err(pc, "label %q has no step", label)
		}
		c.program.Code[pc].Target = target
	}
	if err = c.validate(); err != nil {
		return Artifact{}, err
	}
	raw, id, err := executor.EncodeProgram(c.program)
	if err != nil {
		return Artifact{}, err
	}
	if len(raw) > executor.DefaultLimits().MaxProgramBytes {
		return Artifact{}, fmt.Errorf("compiled artifact byte limit")
	}
	return Artifact{id: id, raw: string(raw)}, nil
}
func keys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }
func (c *compiler) err(pc int, format string, args ...any) error {
	return fmt.Errorf("%s: %s", c.program.Author.Positions[pc], fmt.Sprintf(format, args...))
}
func (c *compiler) append(in executor.Instruction, name string) error {
	if len(c.program.Code) >= MaxInstructions {
		return fmt.Errorf("instruction limit")
	}
	c.program.Code = append(c.program.Code, in)
	c.program.Author.Positions = append(c.program.Author.Positions, name)
	return nil
}
func (c *compiler) expand(steps []Step, path string, scopes []map[string]string, depth int) error {
	if depth > MaxNesting {
		return fmt.Errorf("nesting limit")
	}
	local := map[string]string{}
	for _, s := range steps {
		if s.kind == "label" {
			if err := schema.CheckIdentifier(s.name); err != nil {
				return fmt.Errorf("%s label: %w", path, err)
			}
			if s.name == "" || local[s.name] != "" {
				return fmt.Errorf("%s: duplicate or empty label %q", path, s.name)
			}
			local[s.name] = path + ":" + s.name
		}
	}
	scopes = append(slices.Clone(scopes), local)
	for i, s := range steps {
		c.visits++
		if c.visits > MaxInstructions*10 {
			return fmt.Errorf("author expansion limit")
		}
		name := fmt.Sprintf("%s/%d:%s", path, i, s.kind)
		if err := checkStepIdentifiers(s); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		switch s.kind {
		case "label":
			c.labels[local[s.name]] = len(c.program.Code)
		case "goto":
			label := s.name
			for j := len(scopes) - 1; j >= 0; j-- {
				if resolved := scopes[j][s.name]; resolved != "" {
					label = resolved
					break
				}
			}
			c.jumps[len(c.program.Code)] = label
			if err := c.append(executor.Instruction{Op: "jump"}, name+":"+s.name); err != nil {
				return err
			}
		case "repeat":
			if s.count < 0 {
				return fmt.Errorf("%s: negative repeat", name)
			}
			if s.count > MaxInstructions {
				return fmt.Errorf("%s: repeat limit", name)
			}
			for n := 0; n < s.count; n++ {
				c.visits++
				if c.visits > MaxInstructions*10 {
					return fmt.Errorf("author expansion limit")
				}
				if err := c.expand(s.yes, fmt.Sprintf("%s/%d", name, n), scopes, depth+1); err != nil {
					return err
				}
			}
		case "foreach":
			start := len(c.program.Code)
			if err := c.append(executor.Instruction{Op: "foreach", A: s.a.operand}, name); err != nil {
				return err
			}
			if err := c.append(executor.Instruction{Op: "next", Target: start, Bind: maps.Clone(s.bindings)}, name+"/next"); err != nil {
				return err
			}
			if err := c.expand(s.yes, name+"/body", scopes, depth+1); err != nil {
				return err
			}
			if err := c.append(executor.Instruction{Op: "loop", Target: start}, name+"/loop"); err != nil {
				return err
			}
			c.program.Code[start].Otherwise = len(c.program.Code)
		case "if":
			pc := len(c.program.Code)
			if err := c.append(executor.Instruction{Op: "branch", A: s.a.operand, B: s.b.operand, Target: pc + 1}, name); err != nil {
				return err
			}
			if err := c.expand(s.yes, name+"/then", scopes, depth+1); err != nil {
				return err
			}
			jump := -1
			if c.reachesBoundary(pc+1, len(c.program.Code)) {
				jump = len(c.program.Code)
				if err := c.append(executor.Instruction{Op: "jump"}, name+"/join"); err != nil {
					return err
				}
			}
			c.program.Code[pc].Otherwise = len(c.program.Code)
			if err := c.expand(s.no, name+"/else", scopes, depth+1); err != nil {
				return err
			}
			if jump >= 0 {
				c.program.Code[jump].Target = len(c.program.Code)
			}
		default:
			if err := c.step(s, name); err != nil {
				return err
			}
		}
	}
	return nil
}
func operands(m map[string]Expr) map[string]executor.Operand {
	r := map[string]executor.Operand{}
	for k, v := range m {
		r[k] = v.operand
	}
	return r
}
func (c *compiler) step(s Step, name string) error {
	in := executor.Instruction{Op: s.kind, Dst: s.destination, A: s.a.operand, B: s.b.operand}
	pc := len(c.program.Code)
	switch s.kind {
	case "set", "add":
	case "append":
		f, ok := c.program.Author.Collections[s.destination]
		if !ok {
			return fmt.Errorf("%s: append needs declared collection %q", name, s.destination)
		}
		in.Updates = operands(s.values)
		c.records[pc] = f.Element
	case "end":
		in.Outcome = s.name
	case "send", "receive":
		p, ok := c.program.Author.Ports[s.declaration]
		if !ok {
			return fmt.Errorf("%s: unknown port %q", name, s.declaration)
		}
		if s.kind == "send" {
			p.SendUsed = true
			c.program.Author.Ports[s.declaration] = p
			if p.Local {
				return fmt.Errorf("%s: cannot send on local input port", name)
			}
			in.Op = "emit"
			in.Effect = &executor.Effect{Kind: "message", Target: executor.Lit(executor.Text(p.Peer)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(p.Obligation)), Payload: operands(s.values)}
			c.records[pc] = p.Send.Record
		} else {
			p.ReceiveUsed = true
			c.program.Author.Ports[s.declaration] = p
			kind, provenance := "message", "ordinary-message"
			if p.Local {
				kind, provenance = "input", "local-input"
			}
			in.Op = "receive"
			in.Match = &executor.Match{Kind: kind, Provenance: provenance, Source: executor.Lit(executor.Text(p.Peer)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(p.Obligation)), Operation: executor.Lit(executor.Text(""))}
			in.Bind = maps.Clone(s.bindings)
			c.records[pc] = p.Receive.Record
			c.program.Author.Receives[pc] = p.Receive
		}
	case "request", "await":
		e, ok := c.program.Author.Effects[s.declaration]
		if !ok {
			return fmt.Errorf("%s: unsupported effect %q", name, s.declaration)
		}
		if s.kind == "request" {
			if s.destination == "" {
				return fmt.Errorf("%s: request requires name", name)
			}
			in.Op = "emit"
			in.Effect = &executor.Effect{Kind: "adapter", Target: executor.Lit(executor.Text(e.Capability)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(s.declaration)), Payload: operands(s.values)}
			in.Updates = operands(s.updates)
			c.records[pc] = e.Input.Record
			c.requests[pc] = s.declaration
		} else {
			in.Op = "receive"
			in.Match = &executor.Match{Kind: "adapter", Provenance: "runtime-adapter", Source: executor.Lit(executor.Text(e.Capability)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(s.declaration)), Operation: executor.Ref(s.request)}
			in.Bind = maps.Clone(s.bindings)
			result, err := schema.ResultRecord(e.Output.Record)
			if err != nil {
				return err
			}
			retained, err := result.Retain()
			if err != nil {
				return err
			}
			c.records[pc] = result
			c.program.Author.Receives[pc] = retained
			c.await[pc] = s.declaration
		}
	default:
		return fmt.Errorf("%s: unsupported step", name)
	}
	return c.append(in, name+":"+s.declaration+s.destination)
}

// validate computes definite assignment over the forward-only control-flow graph.
// At joins a value is available only when every reachable predecessor defines it.
func (c *compiler) validate() error {
	if len(c.program.Code) == 0 {
		return fmt.Errorf("missing terminal step")
	}
	incoming := make([]map[string]string, len(c.program.Code))
	incoming[0] = map[string]string{}
	types := map[string]string{}
	for name, f := range c.program.Author.Inputs.Record {
		incoming[0][name] = c.kind(f)
		types[name] = c.kind(f)
	}
	for pc, in := range c.program.Code {
		state := incoming[pc]
		if state == nil {
			continue
		}
		next := maps.Clone(state)
		read := func(o executor.Operand) (string, error) {
			if (o.Ref == "") == (o.Literal == nil) {
				return "", c.err(pc, "invalid expression")
			}
			var kind string
			if o.Ref != "" {
				kind = state[o.Ref]
				if kind == "" {
					return "", c.err(pc, "value %q is not definitely available", o.Ref)
				}
			} else {
				if !o.Literal.Valid() {
					return "", c.err(pc, "invalid literal")
				}
				kind = o.Literal.Kind
				if kind == "list" {
					f, ok := c.program.Author.Collections[in.Dst]
					if !ok {
						f, ok = c.fields[state[in.Dst]]
					}
					if !ok {
						return "", c.err(pc, "list literal requires declared collection or typed destination")
					}
					if err := f.Validate(*o.Literal); err != nil {
						return "", c.err(pc, "list literal: %v", err)
					}
					kind = c.kind(f)
				}
			}
			if o.Offset != nil && kind != "int" {
				return "", c.err(pc, "offset requires int")
			}
			return kind, nil
		}
		readAs := func(o executor.Operand, f schema.Field) (string, error) {
			if o.Literal != nil && o.Literal.Kind == "list" && f.Kind == "list" {
				if o.Ref != "" || o.Offset != nil {
					return "", c.err(pc, "invalid list expression")
				}
				if err := f.Validate(*o.Literal); err != nil {
					return "", c.err(pc, "list literal: %v", err)
				}
				return c.kind(f), nil
			}
			return read(o)
		}
		write := func(name, kind string) error {
			if name == "" {
				return c.err(pc, "empty value name")
			}
			base := func(s string) string {
				if strings.HasPrefix(s, "request:") {
					return "string"
				}
				return s
			}
			if old := types[name]; old != "" && base(old) != base(kind) {
				return c.err(pc, "value %q type changes from %s to %s", name, old, kind)
			}
			types[name] = kind
			next[name] = kind
			return nil
		}
		switch in.Op {
		case "set", "add":
			kind, err := read(in.A)
			if err != nil {
				return err
			}
			if in.Op == "add" {
				other, err := read(in.B)
				if err != nil {
					return err
				}
				if kind != "int" || other != "int" {
					return c.err(pc, "add requires int")
				}
			}
			if f, ok := c.program.Author.Collections[in.Dst]; ok && kind != c.kind(f) {
				return c.err(pc, "collection assignment schema mismatch")
			}
			if f, ok := c.fields[kind]; ok {
				field := f
				c.program.Code[pc].Collection = &field
			}
			if err := write(in.Dst, kind); err != nil {
				return err
			}
		case "foreach":
			kind, err := read(in.A)
			if err != nil {
				return err
			}
			f, ok := c.fields[kind]
			if !ok {
				return c.err(pc, "foreach requires typed list")
			}
			field := f
			c.program.Code[pc].Collection = &field
		case "next":
			f := c.program.Code[in.Target].Collection
			if f == nil {
				return c.err(pc, "missing loop schema")
			}
			seen := map[string]bool{}
			for _, field := range keys(in.Bind) {
				name := in.Bind[field]
				bound, ok := f.Element[field]
				if !ok || seen[name] {
					return c.err(pc, "unknown or duplicate loop binding")
				}
				seen[name] = true
				if err := write(name, c.kind(bound)); err != nil {
					return err
				}
			}
		case "append":
			kind, err := read(executor.Ref(in.Dst))
			if err != nil {
				return err
			}
			f := c.program.Author.Collections[in.Dst]
			if kind != c.kind(f) {
				return c.err(pc, "append schema mismatch")
			}
			if len(in.Updates) != len(f.Element) {
				return c.err(pc, "append fields differ")
			}
			for field, o := range in.Updates {
				want, ok := f.Element[field]
				if !ok {
					return c.err(pc, "unknown append field")
				}
				got, err := readAs(o, want)
				if err != nil {
					return err
				}
				if got != c.kind(want) {
					return c.err(pc, "append field type mismatch")
				}
				if o.Literal != nil {
					if err := want.Validate(*o.Literal); err != nil {
						return c.err(pc, "append field: %v", err)
					}
				}
			}
		case "loop":
			continue
		case "branch":
			expected := c.fields[state[in.B.Ref]]
			a, err := readAs(in.A, expected)
			if err != nil {
				return err
			}
			b, err := readAs(in.B, c.fields[a])
			if err != nil {
				return err
			}
			if a != b {
				return c.err(pc, "branch type mismatch: %s and %s", a, b)
			}
		case "emit":
			record := c.records[pc]
			if len(record) != len(in.Effect.Payload) {
				return c.err(pc, "payload fields differ from schema")
			}
			for _, name := range keys(in.Effect.Payload) {
				o := in.Effect.Payload[name]
				f, ok := record[name]
				if !ok {
					return c.err(pc, "unknown payload field %q", name)
				}
				kind, err := readAs(o, f)
				if err != nil {
					return err
				}
				if kind != c.kind(f) {
					return c.err(pc, "payload %q expects %s", name, f.Kind)
				}
				if o.Literal != nil {
					if err = f.Validate(*o.Literal); err != nil {
						return c.err(pc, "payload %q: %v", name, err)
					}
				}
			}
			for _, name := range keys(in.Updates) {
				if name == in.Dst {
					return c.err(pc, "update conflicts with request %q", name)
				}
				kind, err := readAs(in.Updates[name], c.fields[state[name]])
				if err != nil {
					return err
				}
				if err = write(name, kind); err != nil {
					return err
				}
			}
			if in.Dst != "" {
				if err := write(in.Dst, "request:"+c.requests[pc]); err != nil {
					return err
				}
			}
		case "receive":
			if effect := c.await[pc]; effect != "" {
				kind, err := read(in.Match.Operation)
				if err != nil {
					return err
				}
				if kind != "request:"+effect {
					return c.err(pc, "await requires request for effect %q", effect)
				}
			}
			seen := map[string]bool{}
			for _, field := range keys(in.Bind) {
				name := in.Bind[field]
				f, ok := c.records[pc][field]
				if !ok {
					return c.err(pc, "unknown binding field %q", field)
				}
				if seen[name] {
					return c.err(pc, "duplicate binding %q", name)
				}
				seen[name] = true
				if err := write(name, c.kind(f)); err != nil {
					return err
				}
			}
		case "end":
			if in.Outcome == "succeeded" {
				for _, name := range keys(c.program.Author.Outputs.Record) {
					if next[name] != c.kind(c.program.Author.Outputs.Record[name]) {
						return c.err(pc, "output %q missing or wrong type", name)
					}
				}
			}
			continue
		}
		successors := []int{pc + 1}
		if in.Op == "foreach" {
			successors = append(successors, in.Otherwise)
		}
		if in.Op == "jump" {
			successors = []int{in.Target}
		}
		if in.Op == "branch" {
			successors = []int{in.Target, in.Otherwise}
		}
		for _, target := range successors {
			if target <= pc || target >= len(incoming) {
				return c.err(pc, "control requires later terminal step")
			}
			if incoming[target] == nil {
				incoming[target] = maps.Clone(next)
			} else {
				for name, kind := range incoming[target] {
					if next[name] != kind {
						delete(incoming[target], name)
					}
				}
			}
		}
	}
	return nil
}

// reachesBoundary asks whether the lexical body can continue at its end. A
// trailing label can reach that boundary even when the last emitted instruction
// is a jump or terminal. Nested bodies are fully lowered before this traversal;
// unresolved labels outside the body are exits, not lexical fallthrough.
func (c *compiler) reachesBoundary(start, end int) bool {
	pending := []int{start}
	seen := map[int]bool{}
	for len(pending) > 0 {
		pc := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if pc == end {
			return true
		}
		if pc < start || pc > end || seen[pc] {
			continue
		}
		seen[pc] = true
		in := c.program.Code[pc]
		switch in.Op {
		case "end":
		case "jump":
			target := in.Target
			if label, ok := c.jumps[pc]; ok {
				var exists bool
				target, exists = c.labels[label]
				if !exists {
					continue
				}
			}
			pending = append(pending, target)
		case "branch":
			pending = append(pending, in.Target, in.Otherwise)
		default:
			pending = append(pending, pc+1)
		}
	}
	return false
}

func (c *compiler) kind(f schema.Field) string {
	if f.Kind != "list" {
		return f.Kind
	}
	retained, _ := (schema.Record{"value": f}).Retain()
	key := "list:" + retained.ID
	c.fields[key] = f
	return key
}
func definitionHasCollections(d Definition) (bool, error) {
	if len(d.Collections) > 0 || d.Inputs.HasCollections() || d.Outputs.HasCollections() {
		return true, nil
	}
	for _, v := range d.Values {
		if v.Kind == "list" {
			return true, nil
		}
	}
	for _, e := range d.Effects {
		if e.Input.HasCollections() || e.Output.HasCollections() {
			return true, nil
		}
	}
	for _, p := range d.Ports {
		if p.Send.HasCollections() || p.Receive.HasCollections() {
			return true, nil
		}
	}
	visits := 0
	var walk func([]Step, int) (bool, error)
	walk = func(steps []Step, depth int) (bool, error) {
		if depth > MaxNesting {
			return false, fmt.Errorf("nesting limit")
		}
		for _, s := range steps {
			visits++
			if visits > MaxInstructions*10 {
				return false, fmt.Errorf("author expansion limit")
			}
			if s.kind == "append" || s.kind == "foreach" || s.a.operand.Literal != nil && s.a.operand.Literal.Kind == "list" {
				return true, nil
			}
			for _, body := range [][]Step{s.yes, s.no} {
				found, err := walk(body, depth+1)
				if err != nil || found {
					return found, err
				}
			}
		}
		return false, nil
	}
	return walk(d.Steps, 0)
}
