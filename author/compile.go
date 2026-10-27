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
	catalogue, err := retainInvocations(d, m)
	if err != nil {
		return Artifact{}, err
	}
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
	if usesCollections || len(d.Invocations) > 0 {
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
	return Artifact{id: id, raw: string(raw), catalogue: catalogue}, nil
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
	case "invoke", "await-invocation":
		e, ok := c.program.Author.Invocations[s.declaration]
		if !ok {
			return fmt.Errorf("unknown invocation %q", s.declaration)
		}
		if s.kind == "invoke" {
			in.Op = "emit"
			in.Effect = &executor.Effect{Kind: "invocation", Target: executor.Lit(executor.Text(e.TemplateID)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(s.declaration)), Payload: operands(s.values)}
			in.Updates = operands(s.updates)
			c.records[pc] = e.Input.Record
			c.requests[pc] = "invocation:" + s.declaration
		} else {
			in.Op = "receive"
			in.Match = &executor.Match{Kind: "invocation", Provenance: "runtime-invocation", Source: executor.Lit(executor.Text(e.TemplateID)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text(s.declaration)), Operation: executor.Ref(s.request)}
			in.Bind = maps.Clone(s.bindings)
			record := executor.InvocationOutcomeRecord(e.Output.Record)
			retained, err := record.Retain()
			if err != nil {
				return err
			}
			c.records[pc] = record
			c.program.Author.Receives[pc] = retained
			c.await[pc] = "invocation:" + s.declaration
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
	reaching := c.reachingStates()
	for pc, in := range c.program.Code {
		state := incoming[pc]
		if state == nil {
			continue
		}
		rstate := reaching[pc]
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
			kind, err := read(o)
			// Operation identities are runtime strings at typed payload/list
			// boundaries. Serializing one loses its static declaration proof;
			// AwaitInvocation authenticates the retained operation at runtime.
			if f.Kind == "string" && strings.HasPrefix(kind, "request:") {
				kind = "string"
			}
			return kind, err
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
			if in.Op == "set" {
				if f, ok := c.program.Author.Collections[in.Dst]; ok {
					if err := validateOperandValues(rstate, in.A, f); err != nil {
						return c.err(pc, "collection %q: %v", in.Dst, err)
					}
				}
				if outField, ok := c.program.Author.Outputs.Record[in.Dst]; ok && !strings.HasPrefix(c.program.Author.Positions[pc], "value:") {
					if err := validateOperandValues(rstate, in.A, outField); err != nil {
						return c.err(pc, "output %q: %v", in.Dst, err)
					}
				}
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
				if err := validateOperandValues(rstate, o, want); err != nil {
					return c.err(pc, "append field: %v", err)
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
				if err = validateOperandValues(rstate, o, f); err != nil {
					return c.err(pc, "payload %q: %v", name, err)
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
				if f, ok := c.program.Author.Collections[name]; ok && kind != c.kind(f) {
					return c.err(pc, "collection update schema mismatch")
				}
				if f, ok := c.program.Author.Collections[name]; ok {
					if err := validateOperandValues(rstate, in.Updates[name], f); err != nil {
						return c.err(pc, "update %q: %v", name, err)
					}
				}
				if outField, ok := c.program.Author.Outputs.Record[name]; ok {
					if err := validateOperandValues(rstate, in.Updates[name], outField); err != nil {
						return c.err(pc, "output %q: %v", name, err)
					}
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
				if kind != "request:"+effect && !(kind == "string" && strings.HasPrefix(effect, "invocation:")) {
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
					want := c.program.Author.Outputs.Record[name]
					if next[name] != c.kind(want) {
						return c.err(pc, "output %q missing or wrong type", name)
					}
					if err := validateOperandValues(rstate, executor.Ref(name), want); err != nil {
						return c.err(pc, "output %q: %v", name, err)
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

const maxTrackedIntConstants = 64

type reachingState struct {
	consts      map[string][]schema.Value
	dynamic     map[string]bool
	elemConsts  map[string]map[string][]schema.Value
	elemDynamic map[string]bool
	fields      map[string][]schema.Field
}

func newReachingState() *reachingState {
	return &reachingState{
		consts:      map[string][]schema.Value{},
		dynamic:     map[string]bool{},
		elemConsts:  map[string]map[string][]schema.Value{},
		elemDynamic: map[string]bool{},
		fields:      map[string][]schema.Field{},
	}
}

func cloneValues(vals []schema.Value) []schema.Value {
	if len(vals) == 0 {
		return nil
	}
	out := make([]schema.Value, len(vals))
	for i, v := range vals {
		out[i] = v.Clone()
	}
	return out
}

func cloneField(f schema.Field) schema.Field {
	f.Enum = slices.Clone(f.Enum)
	if f.Element != nil {
		el := make(schema.Record, len(f.Element))
		for k, v := range f.Element {
			el[k] = cloneField(v)
		}
		f.Element = el
	}
	return f
}

func equalField(a, b schema.Field) bool {
	if a.Kind != b.Kind || a.MaxBytes != b.MaxBytes || a.MaxItems != b.MaxItems || a.UniqueKey != b.UniqueKey || !slices.Equal(a.Enum, b.Enum) || len(a.Element) != len(b.Element) {
		return false
	}
	for k, av := range a.Element {
		bv, ok := b.Element[k]
		if !ok || !equalField(av, bv) {
			return false
		}
	}
	return true
}

func addField(list []schema.Field, f schema.Field) ([]schema.Field, bool) {
	for _, existing := range list {
		if equalField(existing, f) {
			return list, false
		}
	}
	return append(list, cloneField(f)), true
}

func (s *reachingState) clone() *reachingState {
	out := newReachingState()
	for k, vals := range s.consts {
		out.consts[k] = cloneValues(vals)
	}
	for k, d := range s.dynamic {
		if d {
			out.dynamic[k] = true
		}
	}
	for k, m := range s.elemConsts {
		cp := make(map[string][]schema.Value, len(m))
		for f, vals := range m {
			cp[f] = cloneValues(vals)
		}
		out.elemConsts[k] = cp
	}
	for k, d := range s.elemDynamic {
		if d {
			out.elemDynamic[k] = true
		}
	}
	for k, flist := range s.fields {
		cp := make([]schema.Field, len(flist))
		for i, f := range flist {
			cp[i] = cloneField(f)
		}
		out.fields[k] = cp
	}
	return out
}

func addConstant(list []schema.Value, v schema.Value) ([]schema.Value, bool) {
	for _, existing := range list {
		if existing.Equal(v) {
			return list, false
		}
	}
	if v.Kind == "int" && len(list) >= maxTrackedIntConstants {
		return list, false
	}
	if len(list) >= MaxInstructions {
		return list, false
	}
	return append(list, v.Clone()), true
}

func checkedAddInt64(a, b int64) (int64, bool) {
	c := a + b
	if (b > 0 && c < a) || (b < 0 && c > a) {
		return 0, false
	}
	return c, true
}

func evalConsts(s *reachingState, o executor.Operand) []schema.Value {
	if s == nil || (o.Ref == "") == (o.Literal == nil) {
		return nil
	}
	if o.Literal != nil {
		if !o.Literal.Valid() || o.Offset != nil {
			return nil
		}
		return []schema.Value{o.Literal.Clone()}
	}
	src := s.consts[o.Ref]
	if len(src) == 0 {
		return nil
	}
	if o.Offset == nil {
		return cloneValues(src)
	}
	var out []schema.Value
	for _, v := range src {
		if v.Kind != "int" {
			return nil
		}
		sum, ok := checkedAddInt64(v.Number, *o.Offset)
		if !ok {
			return nil
		}
		out, _ = addConstant(out, schema.Int(sum))
	}
	return out
}

func evalDynamic(s *reachingState, o executor.Operand) bool {
	if s == nil {
		return true
	}
	if o.Literal != nil {
		return false
	}
	if o.Ref == "" {
		return true
	}
	if s.dynamic[o.Ref] {
		return true
	}
	if o.Offset != nil {
		for _, v := range s.consts[o.Ref] {
			if v.Kind != "int" {
				return true
			}
			if _, ok := checkedAddInt64(v.Number, *o.Offset); !ok {
				return true
			}
		}
	}
	return len(s.consts[o.Ref]) == 0
}

func extractListElemConsts(v schema.Value) map[string][]schema.Value {
	rows, err := v.Records()
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := map[string][]schema.Value{}
	for _, row := range rows {
		for k, val := range row {
			out[k], _ = addConstant(out[k], val)
		}
	}
	return out
}

func validateOperandValues(s *reachingState, o executor.Operand, want schema.Field) error {
	for _, v := range evalConsts(s, o) {
		if err := want.Validate(v); err != nil {
			return err
		}
	}
	if s != nil && o.Ref != "" && o.Literal == nil && o.Offset == nil && want.Kind == "string" {
		for _, srcField := range s.fields[o.Ref] {
			if srcField.Kind != "string" || len(srcField.Enum) == 0 {
				continue
			}
			compatible := false
			var lastErr error
			for _, candidate := range srcField.Enum {
				val := schema.Text(candidate)
				if srcField.Validate(val) != nil {
					continue
				}
				if err := want.Validate(val); err == nil {
					compatible = true
					break
				} else {
					lastErr = err
				}
			}
			if !compatible {
				if lastErr != nil {
					return lastErr
				}
				return fmt.Errorf("enum domain is incompatible with target field")
			}
		}
	}
	return nil
}

func refineBranchState(thenState, elseState *reachingState, left, right executor.Operand) {
	if left.Ref == "" || left.Literal != nil || left.Offset != nil || right.Literal == nil || right.Ref != "" || right.Offset != nil {
		return
	}
	lit := *right.Literal
	if !lit.Valid() {
		return
	}
	name := left.Ref
	vals := elseState.consts[name]
	dyn := elseState.dynamic[name]

	// Refine elseState (where left != lit).
	if len(vals) > 0 {
		var kept []schema.Value
		for _, v := range vals {
			if !v.Equal(lit) {
				kept = append(kept, v.Clone())
			}
		}
		if len(kept) > 0 {
			elseState.consts[name] = kept
		} else if dyn {
			delete(elseState.consts, name)
		} else {
			// Contradicted else branch: preserve purely static constants so dead
			// branches cannot launder invalid constants into nil.
			elseState.consts[name] = cloneValues(vals)
		}
	}
	if lit.Kind == "string" {
		if flist := elseState.fields[name]; len(flist) > 0 {
			var updated []schema.Field
			for _, f := range flist {
				if f.Kind == "string" && len(f.Enum) > 0 {
					var keptEnum []string
					for _, e := range f.Enum {
						if e != lit.Text {
							keptEnum = append(keptEnum, e)
						}
					}
					if len(keptEnum) > 0 {
						f.Enum = keptEnum
					}
				}
				updated = append(updated, f)
			}
			elseState.fields[name] = updated
		}
	}

	// Refine thenState (where left == lit).
	inVals := false
	for _, v := range thenState.consts[name] {
		if v.Equal(lit) {
			inVals = true
			break
		}
	}
	if inVals || thenState.dynamic[name] {
		thenState.consts[name] = []schema.Value{lit.Clone()}
		thenState.dynamic[name] = false
		var incompatible []schema.Field
		for _, f := range thenState.fields[name] {
			if f.Validate(lit) != nil {
				incompatible = append(incompatible, f)
			}
		}
		if len(incompatible) > 0 {
			thenState.fields[name] = incompatible
		} else {
			delete(thenState.fields, name)
		}
	} else {
		// Contradicted then branch: keep existing static constants and add lit
		// so neither a prior invalid constant nor an invalid branch literal escapes.
		list := cloneValues(thenState.consts[name])
		list, _ = addConstant(list, lit)
		thenState.consts[name] = list
	}
}

func (c *compiler) reachingStates() []*reachingState {
	n := len(c.program.Code)
	incoming := make([]*reachingState, n)
	if n == 0 {
		return incoming
	}
	initState := newReachingState()
	for name, f := range c.program.Author.Inputs.Record {
		initState.dynamic[name] = true
		if f.Kind == "list" {
			initState.elemDynamic[name] = true
			initState.fields[name] = []schema.Field{cloneField(f)}
		} else if f.Kind == "string" && len(f.Enum) > 0 {
			initState.fields[name] = []schema.Field{cloneField(f)}
		}
	}
	for name, f := range c.program.Author.Collections {
		initState.fields[name] = []schema.Field{cloneField(f)}
	}
	incoming[0] = initState
	queue := []int{0}
	queued := make([]bool, n)
	queued[0] = true

	mergeInto := func(target int, state *reachingState) {
		if target < 0 || target >= n {
			return
		}
		dest := incoming[target]
		changed := false
		if dest == nil {
			incoming[target] = state.clone()
			changed = true
		} else {
			for name, vals := range state.consts {
				list := dest.consts[name]
				for _, v := range vals {
					var added bool
					list, added = addConstant(list, v)
					if added {
						changed = true
					}
				}
				if len(list) > 0 {
					dest.consts[name] = list
				}
			}
			for name, d := range state.dynamic {
				if d && !dest.dynamic[name] {
					dest.dynamic[name] = true
					changed = true
				}
			}
			for name, m := range state.elemConsts {
				destMap := dest.elemConsts[name]
				if destMap == nil && len(m) > 0 {
					destMap = map[string][]schema.Value{}
					dest.elemConsts[name] = destMap
				}
				for f, vals := range m {
					list := destMap[f]
					for _, v := range vals {
						var added bool
						list, added = addConstant(list, v)
						if added {
							changed = true
						}
					}
					if len(list) > 0 {
						destMap[f] = list
					}
				}
			}
			for name, d := range state.elemDynamic {
				if d && !dest.elemDynamic[name] {
					dest.elemDynamic[name] = true
					changed = true
				}
			}
			for name, flist := range state.fields {
				list := dest.fields[name]
				for _, f := range flist {
					var added bool
					list, added = addField(list, f)
					if added {
						changed = true
					}
				}
				if len(list) > 0 {
					dest.fields[name] = list
				}
			}
		}
		if changed && !queued[target] {
			queued[target] = true
			queue = append(queue, target)
		}
	}

	assign := func(next, state *reachingState, dst string, o executor.Operand) {
		vals := evalConsts(state, o)
		if len(vals) > 0 {
			next.consts[dst] = vals
		} else {
			delete(next.consts, dst)
		}
		if evalDynamic(state, o) {
			next.dynamic[dst] = true
		} else {
			delete(next.dynamic, dst)
		}
		if f, ok := c.program.Author.Collections[dst]; ok {
			next.fields[dst] = []schema.Field{cloneField(f)}
		} else if o.Ref != "" && o.Literal == nil && o.Offset == nil && len(state.fields[o.Ref]) > 0 {
			cp := make([]schema.Field, len(state.fields[o.Ref]))
			for i, f := range state.fields[o.Ref] {
				cp[i] = cloneField(f)
			}
			next.fields[dst] = cp
		} else {
			delete(next.fields, dst)
		}
		if o.Literal != nil && o.Literal.Kind == "list" {
			if m := extractListElemConsts(*o.Literal); len(m) > 0 {
				next.elemConsts[dst] = m
			} else {
				delete(next.elemConsts, dst)
			}
			delete(next.elemDynamic, dst)
		} else if o.Ref != "" && o.Literal == nil && o.Offset == nil {
			if m := state.elemConsts[o.Ref]; len(m) > 0 {
				cp := make(map[string][]schema.Value, len(m))
				for f, v := range m {
					cp[f] = cloneValues(v)
				}
				next.elemConsts[dst] = cp
			} else {
				delete(next.elemConsts, dst)
			}
			if state.elemDynamic[o.Ref] {
				next.elemDynamic[dst] = true
			} else {
				delete(next.elemDynamic, dst)
			}
		} else {
			delete(next.elemConsts, dst)
			delete(next.elemDynamic, dst)
		}
	}

	for len(queue) > 0 {
		pc := queue[0]
		queue = queue[1:]
		queued[pc] = false
		state := incoming[pc]
		if state == nil {
			continue
		}
		next := state.clone()
		in := c.program.Code[pc]
		switch in.Op {
		case "set":
			assign(next, state, in.Dst, in.A)
		case "add":
			aVals := evalConsts(state, in.A)
			bVals := evalConsts(state, in.B)
			aDyn := evalDynamic(state, in.A)
			bDyn := evalDynamic(state, in.B)
			var sums []schema.Value
			overflow := false
			if len(aVals) > 0 && len(bVals) > 0 {
				for _, a := range aVals {
					for _, b := range bVals {
						if a.Kind != "int" || b.Kind != "int" {
							overflow = true
							break
						}
						s, ok := checkedAddInt64(a.Number, b.Number)
						if !ok {
							overflow = true
							break
						}
						sums, _ = addConstant(sums, schema.Int(s))
					}
				}
			}
			if !overflow && len(sums) > 0 {
				next.consts[in.Dst] = sums
			} else {
				delete(next.consts, in.Dst)
			}
			if aDyn || bDyn || overflow || len(sums) == 0 {
				next.dynamic[in.Dst] = true
			} else {
				delete(next.dynamic, in.Dst)
			}
			delete(next.fields, in.Dst)
			delete(next.elemConsts, in.Dst)
			delete(next.elemDynamic, in.Dst)
		case "next":
			var elemMap map[string][]schema.Value
			listDyn := true
			var listFields []schema.Field
			if in.Target >= 0 && in.Target < n {
				forIn := c.program.Code[in.Target]
				if forIn.A.Literal != nil && forIn.A.Literal.Kind == "list" {
					elemMap = extractListElemConsts(*forIn.A.Literal)
					listDyn = false
				} else if forIn.A.Ref != "" {
					elemMap = state.elemConsts[forIn.A.Ref]
					listDyn = state.elemDynamic[forIn.A.Ref]
					for _, f := range state.fields[forIn.A.Ref] {
						if f.Kind == "list" {
							listFields = append(listFields, f)
						}
					}
					if len(listFields) == 0 {
						if f, ok := c.program.Author.Collections[forIn.A.Ref]; ok {
							listFields = []schema.Field{f}
						} else if f, ok := c.program.Author.Inputs.Record[forIn.A.Ref]; ok && f.Kind == "list" {
							listFields = []schema.Field{f}
						}
					}
				}
			}
			for _, field := range keys(in.Bind) {
				name := in.Bind[field]
				var scalarVals []schema.Value
				var nestedElem map[string][]schema.Value
				for _, v := range elemMap[field] {
					if v.Kind == "list" {
						for k, kvals := range extractListElemConsts(v) {
							if nestedElem == nil {
								nestedElem = map[string][]schema.Value{}
							}
							list := nestedElem[k]
							for _, kv := range kvals {
								list, _ = addConstant(list, kv)
							}
							if len(list) > 0 {
								nestedElem[k] = list
							}
						}
					} else {
						scalarVals = append(scalarVals, v.Clone())
					}
				}
				if len(scalarVals) > 0 {
					next.consts[name] = scalarVals
				} else {
					delete(next.consts, name)
				}
				if listDyn || len(scalarVals) == 0 {
					next.dynamic[name] = true
				} else {
					delete(next.dynamic, name)
				}
				delete(next.fields, name)
				for _, lf := range listFields {
					if bound, ok := lf.Element[field]; ok {
						if bound.Kind == "list" || (bound.Kind == "string" && len(bound.Enum) > 0) {
							next.fields[name], _ = addField(next.fields[name], bound)
						}
					}
				}
				if len(nestedElem) > 0 {
					next.elemConsts[name] = nestedElem
				} else {
					delete(next.elemConsts, name)
				}
				if listDyn || len(nestedElem) == 0 {
					next.elemDynamic[name] = true
				} else {
					delete(next.elemDynamic, name)
				}
			}
		case "append":
			delete(next.consts, in.Dst)
			destMap := next.elemConsts[in.Dst]
			if destMap == nil {
				destMap = map[string][]schema.Value{}
				next.elemConsts[in.Dst] = destMap
			}
			for field, o := range in.Updates {
				list := destMap[field]
				for _, v := range evalConsts(state, o) {
					list, _ = addConstant(list, v)
				}
				if o.Ref != "" && o.Literal == nil && o.Offset == nil {
					for k, kvals := range state.elemConsts[o.Ref] {
						for _, kv := range kvals {
							list, _ = addConstant(list, schema.List(map[string]schema.Value{k: kv.Clone()}))
						}
					}
					if state.elemDynamic[o.Ref] {
						next.elemDynamic[in.Dst] = true
					}
				}
				if len(list) > 0 {
					destMap[field] = list
				}
				if evalDynamic(state, o) {
					next.elemDynamic[in.Dst] = true
				}
			}
		case "emit":
			for _, name := range keys(in.Updates) {
				assign(next, state, name, in.Updates[name])
			}
			if in.Dst != "" {
				delete(next.consts, in.Dst)
				next.dynamic[in.Dst] = true
				delete(next.fields, in.Dst)
				delete(next.elemConsts, in.Dst)
				delete(next.elemDynamic, in.Dst)
			}
		case "receive":
			rec := c.records[pc]
			for _, field := range keys(in.Bind) {
				name := in.Bind[field]
				delete(next.consts, name)
				next.dynamic[name] = true
				delete(next.fields, name)
				delete(next.elemConsts, name)
				delete(next.elemDynamic, name)
				if f, ok := rec[field]; ok {
					if f.Kind == "list" {
						next.elemDynamic[name] = true
						next.fields[name] = []schema.Field{cloneField(f)}
					} else if f.Kind == "string" && len(f.Enum) > 0 {
						next.fields[name] = []schema.Field{cloneField(f)}
					}
				}
			}
		case "loop":
			mergeInto(in.Target, next)
			continue
		case "end":
			continue
		case "branch":
			thenState := next.clone()
			elseState := next.clone()
			refineBranchState(thenState, elseState, in.A, in.B)
			refineBranchState(thenState, elseState, in.B, in.A)
			mergeInto(in.Target, thenState)
			mergeInto(in.Otherwise, elseState)
			continue
		}
		if in.Op == "jump" {
			mergeInto(in.Target, next)
			continue
		}
		mergeInto(pc+1, next)
		if in.Op == "foreach" {
			mergeInto(in.Otherwise, next)
		}
	}
	return incoming
}
