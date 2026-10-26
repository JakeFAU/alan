package executor

import (
	"fmt"
	recordschema "github.com/JakeFAU/alan/schema"
)

// Cursor is the immutable iteration snapshot and the next element index. Binding
// increments Index in its own commit; body instructions never advance it.
type Cursor struct {
	Snapshot Value
	Index    int
}

func cloneCursors(in map[int]Cursor) map[int]Cursor {
	if len(in) == 0 {
		return nil
	}
	out := map[int]Cursor{}
	for k, c := range in {
		c.Snapshot = c.Snapshot.Clone()
		out[k] = c
	}
	return out
}
func validateCollections(p Program) error {
	for pc, in := range p.Code {
		if in.Collection != nil {
			if p.Semantics != CollectionSemanticVersion {
				return ErrVersion
			}
			if in.Collection.Kind != "list" {
				return fmt.Errorf("invalid collection schema")
			}
			if _, err := in.Collection.Retain(); err != nil {
				return err
			}
		}
		operands := []Operand{in.A, in.B}
		for _, o := range in.Updates {
			operands = append(operands, o)
		}
		if in.Effect != nil {
			operands = append(operands, in.Effect.Target, in.Effect.Generation, in.Effect.Obligation)
			for _, o := range in.Effect.Payload {
				operands = append(operands, o)
			}
		}
		if in.Match != nil {
			operands = append(operands, in.Match.Source, in.Match.Generation, in.Match.Obligation, in.Match.Operation)
		}
		for _, o := range operands {
			if p.Semantics == CollectionSemanticVersion && o.Literal != nil && !o.Literal.Valid() {
				return fmt.Errorf("invalid literal")
			}
			if o.Literal != nil && o.Literal.Kind == "list" {
				if p.Semantics != CollectionSemanticVersion {
					return ErrVersion
				}
				if !o.Literal.Valid() {
					return fmt.Errorf("invalid list literal")
				}
			}
		}
		switch in.Op {
		case "foreach":
			if in.Collection == nil || in.Otherwise <= pc+2 || in.Otherwise >= len(p.Code) {
				return fmt.Errorf("invalid structured loop")
			}
			next, back := p.Code[pc+1], p.Code[in.Otherwise-1]
			if next.Op != "next" || next.Target != pc || back.Op != "loop" || back.Target != pc {
				return fmt.Errorf("invalid loop boundaries")
			}
		case "next", "loop":
			if in.Target < 0 || in.Target >= pc || p.Code[in.Target].Op != "foreach" {
				return fmt.Errorf("invalid loop residual target")
			}
			start := p.Code[in.Target]
			if in.Op == "next" && pc != in.Target+1 || in.Op == "loop" && pc != start.Otherwise-1 {
				return fmt.Errorf("unstructured loop edge")
			}
			if in.Op == "next" {
				seen := map[string]bool{}
				for field, dst := range in.Bind {
					if _, ok := start.Collection.Element[field]; !ok || recordschema.CheckIdentifier(dst) != nil || seen[dst] {
						return fmt.Errorf("invalid loop binding")
					}
					seen[dst] = true
				}
			}
		case "append":
			if p.Author == nil || p.Author.Collections[in.Dst].Kind != "list" {
				return fmt.Errorf("append requires declared list")
			}
		}
		// A branch cannot enter a compiler-owned loop residual from outside it.
		if p.Semantics == CollectionSemanticVersion && (in.Op == "jump" || in.Op == "branch") {
			targets := []int{in.Target}
			if in.Op == "branch" {
				targets = append(targets, in.Otherwise)
			}
			for _, target := range targets {
				if target <= pc {
					return fmt.Errorf("unbounded backward branch")
				}
				for start, loop := range p.Code {
					if loop.Op == "foreach" && target > start && target < loop.Otherwise && (pc <= start || pc >= loop.Otherwise) {
						return fmt.Errorf("branch enters loop body")
					}
				}
			}
		}
	}
	// Structured loops must be nested or disjoint, never crossed.
	for start, a := range p.Code {
		if a.Op != "foreach" {
			continue
		}
		for other, b := range p.Code {
			if b.Op == "foreach" && other > start && other < a.Otherwise && b.Otherwise >= a.Otherwise {
				return fmt.Errorf("crossed loop boundaries")
			}
		}
	}
	return nil
}
func validateCursors(p Program, s Projection) error {
	if len(s.Cursors) > 0 && p.Semantics != CollectionSemanticVersion {
		return ErrVersion
	}
	if s.Status == "open" {
		for pc, in := range p.Code {
			if in.Op == "foreach" && s.PC > pc && s.PC < in.Otherwise {
				if _, ok := s.Cursors[pc]; !ok {
					return fmt.Errorf("missing durable cursor")
				}
			}
		}
	}
	for pc, c := range s.Cursors {
		if pc < 0 || pc >= len(p.Code) || p.Code[pc].Op != "foreach" {
			return fmt.Errorf("invalid cursor origin")
		}
		in := p.Code[pc]
		if s.PC <= pc || s.PC >= in.Otherwise || c.Index < 0 {
			return fmt.Errorf("cursor outside loop")
		}
		if err := in.Collection.Validate(c.Snapshot); err != nil {
			return fmt.Errorf("cursor snapshot: %w", err)
		}
		n, _ := c.Snapshot.Len()
		if c.Index > n || s.PC != pc+1 && c.Index == 0 {
			return fmt.Errorf("cursor outside snapshot")
		}
	}
	return nil
}
func cleanCursors(p Program, s *Projection) {
	for pc := range s.Cursors {
		if s.Status != "open" || s.PC <= pc || s.PC >= p.Code[pc].Otherwise {
			delete(s.Cursors, pc)
		}
	}
	if len(s.Cursors) == 0 {
		s.Cursors = nil
	}
}
func reduceCollection(p Program, in Instruction, s Projection, next *Projection) error {
	switch in.Op {
	case "foreach":
		v, err := eval(in.A, s.Data)
		if err != nil {
			return err
		}
		if err = in.Collection.Validate(v); err != nil {
			return err
		}
		if next.Cursors == nil {
			next.Cursors = map[int]Cursor{}
		}
		next.Cursors[s.PC] = Cursor{Snapshot: v.Clone()}
	case "next":
		c, ok := s.Cursors[in.Target]
		if !ok {
			return fmt.Errorf("missing durable cursor")
		}
		n, _ := c.Snapshot.Len()
		if c.Index == n {
			delete(next.Cursors, in.Target)
			next.PC = p.Code[in.Target].Otherwise
			break
		}
		row, err := c.Snapshot.At(c.Index)
		if err != nil {
			return err
		}
		for field, dst := range in.Bind {
			next.Data[dst] = row[field].Clone()
		}
		c.Index++
		c.Snapshot = c.Snapshot.Clone()
		next.Cursors[in.Target] = c
	case "loop":
		if _, ok := s.Cursors[in.Target]; !ok {
			return fmt.Errorf("missing durable cursor")
		}
		next.PC = in.Target + 1
	case "append":
		row := map[string]Value{}
		for field, o := range in.Updates {
			v, err := eval(o, s.Data)
			if err != nil {
				return err
			}
			row[field] = v
		}
		v, err := p.Author.Collections[in.Dst].Append(s.Data[in.Dst], row)
		if err != nil {
			return fmt.Errorf("append: %w", err)
		}
		next.Data[in.Dst] = v
	}
	return nil
}
