package author

import (
	"slices"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

type guardKey struct {
	name           string
	literal        string
	stringEquality bool
}

type boolGuard struct {
	key   guardKey
	value bool
}

type stringSource struct {
	guard boolGuard
	row   bool
}

// Each alternative covers values on paths consistent with its guard.
// A row schema alone does not capture relationships between columns.
type stringBounds map[stringSource]schema.Field

func literalStringBounds(value string, guard boolGuard) stringBounds {
	return stringBounds{{guard: guard}: {Kind: "string", MaxBytes: max(1, len(value)), Enum: []string{value}}}
}

func (b stringBounds) fits(want schema.Field) bool {
	for _, src := range b {
		if len(src.Enum) == 0 {
			if src.MaxBytes > want.MaxBytes || len(want.Enum) != 0 {
				return false
			}
			continue
		}
		allowed := false
		for _, value := range src.Enum {
			if want.Validate(schema.Text(value)) == nil {
				allowed = true
			} else {
				return false
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func (b stringBounds) mayEqual(value schema.Value) bool {
	if len(b) == 0 {
		return true
	}
	for _, field := range b {
		if field.Validate(value) == nil {
			return true
		}
	}
	return false
}

func joinStringBound(dst stringBounds, source stringSource, field schema.Field) {
	if old, ok := dst[source]; ok {
		field.MaxBytes = max(field.MaxBytes, old.MaxBytes)
		if len(old.Enum) == 0 || len(field.Enum) == 0 {
			field.Enum = nil
		} else if !slices.Equal(old.Enum, field.Enum) {
			values := append(slices.Clone(old.Enum), field.Enum...)
			slices.Sort(values)
			field.Enum = slices.Compact(values)
		}
	}
	dst[source] = field
}

func (b stringBounds) under(guard boolGuard) stringBounds {
	next := make(stringBounds, len(b))
	for source, field := range b {
		if source.guard.key.name == "" {
			source.guard = guard
		} else if source.guard.key == guard.key && source.guard.value != guard.value {
			continue
		}
		joinStringBound(next, source, field)
	}
	return next
}

// forget removes iteration-local predicates when control leaves their loop.
func (b stringBounds) forget(names map[string]bool) stringBounds {
	result := stringBounds{}
	for source, field := range b {
		if names[source.guard.key.name] {
			source.guard = boolGuard{}
		}
		joinStringBound(result, source, field)
	}
	return result
}

// Track guards whose values have one definition and cannot be rebound.
// A receive inside a loop can bind a different validity value each iteration.
func (c *compiler) immutableBoolGuards() map[string]bool {
	candidates := map[string]bool{}
	writes := map[string]int{}
	for name, field := range c.program.Author.Inputs.Record {
		if field.Kind == "bool" {
			candidates[name] = true
			writes[name]++
		}
	}
	loopExit := 0
	for pc, in := range c.program.Code {
		if in.Op == "foreach" {
			loopExit = max(loopExit, in.Otherwise)
		}
		if in.Op == "receive" && in.Match.Kind == "adapter" && pc >= loopExit {
			if dst := in.Bind[schema.ResultValid]; dst != "" {
				candidates[dst] = true
			}
		}
		writes[in.Dst]++
		if in.Op == "emit" {
			for dst := range in.Updates {
				writes[dst]++
			}
		}
		for _, dst := range in.Bind {
			writes[dst]++
		}
	}
	for name := range candidates {
		if writes[name] != 1 {
			delete(candidates, name)
		}
	}
	return candidates
}

// Loop booleans may change across iterations, but a single unmodified binding
// denotes the current row inside its loop body.
func (c *compiler) singleRowBoolBindings() map[string]bool {
	candidates := map[string]bool{}
	writes := map[string]int{}
	for name := range c.program.Author.Inputs.Record {
		writes[name]++
	}
	for _, in := range c.program.Code {
		if in.Dst != "" {
			writes[in.Dst]++
		}
		for name := range in.Updates {
			writes[name]++
		}
		for field, name := range in.Bind {
			writes[name]++
			if in.Op == "next" || in.Op == "receive" && in.Match.Kind == "adapter" && field == schema.ResultValid {
				candidates[name] = true
			}
		}
	}
	for name := range candidates {
		if writes[name] != 1 {
			delete(candidates, name)
		}
	}
	return candidates
}

// Only input strings that no instruction can overwrite are stable across tests.
func (c *compiler) immutableStringInputs() map[string]bool {
	inputs := map[string]bool{}
	for name, field := range c.program.Author.Inputs.Record {
		if field.Kind == "string" {
			inputs[name] = true
		}
	}
	for _, in := range c.program.Code {
		delete(inputs, in.Dst)
		for dst := range in.Updates {
			delete(inputs, dst)
		}
		for _, dst := range in.Bind {
			delete(inputs, dst)
		}
	}
	return inputs
}

func comparedLiteral(in executor.Instruction, kind string) (string, schema.Value, bool) {
	ref, lit := in.A, in.B
	if ref.Literal != nil {
		ref, lit = lit, ref
	}
	if ref.Ref == "" || ref.Offset != nil || lit.Literal == nil || lit.Offset != nil || lit.Literal.Kind != kind {
		return "", schema.Value{}, false
	}
	return ref.Ref, *lit.Literal, true
}

// loopStringInvariants conservatively unions each loop-written string's entry
// values with every literal or copied value written in the body. Any other
// writer, unresolved source, or recursive dependency remains unknown.
func (c *compiler) loopStringInvariants(start, end int, state map[string]string, incoming map[string]stringBounds) map[string]stringBounds {
	type writes struct {
		operands []executor.Operand
		fields   []schema.Field
		unknown  bool
	}
	all := map[string]*writes{}
	get := func(name string) *writes {
		if all[name] == nil {
			all[name] = &writes{}
		}
		return all[name]
	}
	for pc := start + 1; pc < end; pc++ {
		in := c.program.Code[pc]
		if in.Dst != "" {
			if in.Op == "set" {
				get(in.Dst).operands = append(get(in.Dst).operands, in.A)
			} else {
				get(in.Dst).unknown = true
			}
		}
		if in.Op == "emit" {
			for name, o := range in.Updates {
				get(name).operands = append(get(name).operands, o)
			}
		}
		for field, name := range in.Bind {
			bound, ok := c.records[pc][field]
			if ok && bound.Kind == "string" && field != schema.ResultDisposition && c.invalidAdapterResultStops(pc, end) {
				get(name).fields = append(get(name).fields, bound)
			} else {
				get(name).unknown = true
			}
		}
	}
	memo := map[string]stringBounds{}
	failed := map[string]bool{}
	visiting := map[string]bool{}
	var resolve func(string) (stringBounds, bool)
	resolve = func(name string) (stringBounds, bool) {
		if b, ok := memo[name]; ok {
			return b, true
		}
		if failed[name] || visiting[name] {
			return nil, false
		}
		visiting[name] = true
		defer delete(visiting, name)
		result := stringBounds{}
		add := func(b stringBounds) {
			for source, f := range b {
				source.guard = boolGuard{}
				joinStringBound(result, source, f)
			}
		}
		if state[name] != "" {
			if state[name] != "string" || len(incoming[name]) == 0 {
				failed[name] = true
				return nil, false
			}
			add(incoming[name])
		}
		if w := all[name]; w != nil {
			if w.unknown {
				failed[name] = true
				return nil, false
			}
			for _, f := range w.fields {
				add(stringBounds{{}: f})
			}
			for _, o := range w.operands {
				if o.Offset != nil {
					failed[name] = true
					return nil, false
				}
				if o.Literal != nil && o.Ref == "" && o.Literal.Kind == "string" {
					add(literalStringBounds(o.Literal.Text, boolGuard{}))
					continue
				}
				if o.Ref != "" && o.Literal == nil {
					b, ok := resolve(o.Ref)
					if !ok {
						failed[name] = true
						return nil, false
					}
					add(b)
					continue
				}
				failed[name] = true
				return nil, false
			}
		}
		if len(result) == 0 {
			failed[name] = true
			return nil, false
		}
		memo[name] = result
		return result, true
	}
	for name := range all {
		resolve(name)
	}
	return memo
}

// A result can contribute its success schema to a loop invariant only if an
// invalid result immediately reaches a failure terminal before any loop-back.
func (c *compiler) invalidAdapterResultStops(pc, end int) bool {
	if pc+1 >= end {
		return false
	}
	in := c.program.Code[pc]
	if in.Op != "receive" || in.Match.Kind != "adapter" {
		return false
	}
	valid := in.Bind[schema.ResultValid]
	if valid == "" {
		return false
	}
	branch := c.program.Code[pc+1]
	if branch.Op != "branch" {
		return false
	}
	name, lit, ok := comparedLiteral(branch, "bool")
	if !ok || name != valid {
		return false
	}
	target := branch.Target
	if lit.Flag {
		target = branch.Otherwise
	}
	return target > pc+1 && target < end && c.program.Code[target].Op == "end" && c.program.Code[target].Outcome == "failed"
}
