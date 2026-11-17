package author

import (
	"reflect"
	"strings"

	"github.com/JakeFAU/alan/internal/executor"
)

// handleValue is a may-analysis: unknown is ordinary/untrusted data, tags are
// declaration identities, and fields summarize all rows of a list. Empty fields
// are bottom (an empty list), rather than proof about a forged string.
type handleValue struct {
	unknown bool
	tags    map[string]bool
	fields  map[string]handleValue
}

// User variables and loop snapshots occupy disjoint key spaces, including when
// a legal user identifier contains a NUL byte.
type handleKey struct {
	variable string
	snapshot bool
	pc       int
}
type handleState map[handleKey]handleValue

func handleMerge(a, b handleValue) handleValue {
	r := handleValue{unknown: a.unknown || b.unknown}
	for _, x := range []handleValue{a, b} {
		for t := range x.tags {
			if r.tags == nil {
				r.tags = map[string]bool{}
			}
			r.tags[t] = true
		}
		for f, v := range x.fields {
			if r.fields == nil {
				r.fields = map[string]handleValue{}
			}
			r.fields[f] = handleMerge(r.fields[f], v)
		}
	}
	return r
}
func handleCopy(s handleState) handleState {
	r := handleState{}
	for k, v := range s {
		r[k] = v
	}
	return r
}
func handleRead(s handleState, o executor.Operand) handleValue {
	if o.Ref != "" && o.Literal == nil && o.Offset == nil {
		if v, ok := s[handleKey{variable: o.Ref}]; ok {
			return v
		}
		return handleValue{unknown: true}
	}
	if o.Literal != nil && o.Literal.Kind == "list" {
		n, err := o.Literal.Len()
		if err == nil && n == 0 {
			return handleValue{}
		}
	}
	return handleValue{unknown: true}
}

// checkInvocationHandles follows all lowered control-flow paths, including loop
// backedges. It never changes retained instructions or their string encoding.
func (c *compiler) checkInvocationHandles() error {
	code := c.program.Code
	if len(c.program.Author.Invocations) == 0 {
		return nil
	}
	states := make([]handleState, len(code))
	states[0] = handleState{}
	for k := range c.program.Author.Inputs.Record {
		states[0][handleKey{variable: k}] = handleValue{unknown: true}
	}
	queue := []int{0}
	queued := make([]bool, len(code))
	queued[0] = true
	for len(queue) > 0 {
		pc := queue[0]
		queue = queue[1:]
		queued[pc] = false
		in := code[pc]
		s := states[pc]
		n := handleCopy(s)
		switch in.Op {
		case "set":
			n[handleKey{variable: in.Dst}] = handleRead(s, in.A)
		case "add":
			n[handleKey{variable: in.Dst}] = handleValue{unknown: true}
		case "append":
			v := n[handleKey{variable: in.Dst}]
			fields := map[string]handleValue{}
			for k, x := range v.fields {
				fields[k] = x
			}
			for k, o := range in.Updates {
				fields[k] = handleMerge(fields[k], handleRead(s, o))
			}
			v.fields = fields
			n[handleKey{variable: in.Dst}] = v
		case "foreach":
			n[handleKey{snapshot: true, pc: pc}] = handleRead(s, in.A)
		case "next":
			v := s[handleKey{snapshot: true, pc: in.Target}]
			for field, dst := range in.Bind {
				x := v.fields[field]
				if v.unknown {
					x.unknown = true
				}
				n[handleKey{variable: dst}] = x
			}
		case "emit":
			for dst, o := range in.Updates {
				n[handleKey{variable: dst}] = handleRead(s, o)
			}
			if in.Dst != "" {
				if strings.HasPrefix(c.requests[pc], "invocation:") {
					n[handleKey{variable: in.Dst}] = handleValue{tags: map[string]bool{c.requests[pc]: true}}
				} else {
					n[handleKey{variable: in.Dst}] = handleValue{unknown: true}
				}
			}
		case "receive":
			for _, dst := range in.Bind {
				n[handleKey{variable: dst}] = handleValue{unknown: true}
			}
		}
		successors := []int{pc + 1}
		switch in.Op {
		case "end":
			successors = nil
		case "jump":
			successors = []int{in.Target}
		case "branch":
			successors = []int{in.Target, in.Otherwise}
		case "foreach":
			successors = []int{pc + 1, in.Otherwise}
			if c.loopEmptiness[pc] {
				successors = []int{in.Otherwise}
			}
		case "loop":
			successors = []int{in.Target + 1, code[in.Target].Otherwise}
		}
		for _, to := range successors {
			if to < 0 || to >= len(code) {
				continue
			}
			var joined handleState
			if states[to] == nil {
				joined = handleCopy(n)
			} else {
				joined = handleCopy(states[to])
				for k, v := range n {
					if old, ok := joined[k]; ok {
						joined[k] = handleMerge(old, v)
					} else {
						v.unknown = true
						joined[k] = v
					}
				}
				for k, v := range joined {
					if _, ok := n[k]; !ok {
						v.unknown = true
						joined[k] = v
					}
				}
			}
			if !reflect.DeepEqual(joined, states[to]) {
				states[to] = joined
				if !queued[to] {
					queue = append(queue, to)
					queued[to] = true
				}
			}
		}
	}
	for pc, in := range code {
		want := c.await[pc]
		if states[pc] == nil || !strings.HasPrefix(want, "invocation:") {
			continue
		}
		v := handleRead(states[pc], in.Match.Operation)
		if v.unknown || len(v.tags) != 1 || !v.tags[want] {
			return c.err(pc, "await requires proven request for %q", want)
		}
	}
	return nil
}
