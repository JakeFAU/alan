// Package owningtree qualifies a finite protocol projection. It imports no ALAN implementation.
package owningtree

import (
	"fmt"
	"strings"
)

type request struct {
	ID    int
	Value int
	Mode  string
}

type scenario struct {
	Shape      string
	Assignment string
	Requests   []request
}

// A static path indexes one original operation in this fresh, generation-one tree.
// The root O is separately provisioned. S and e are different calls even at equal inputs.
type member struct {
	role, kind string
	input      request
	children   []*member
}

func (s scenario) root() *member {
	index := 0
	var build func(string, string) *member
	build = func(role, shape string) *member {
		n := &member{role: role, kind: shape}
		child := func(name, form string) { n.children = append(n.children, build(role+"/"+name, form)) }
		switch shape {
		case "outer":
			child("tree", s.Shape)
		case "fork":
			child("left", "leaf")
			child("right", "leaf")
		case "left-nested":
			child("left", "fork")
			child("right", "leaf")
		case "right-nested":
			child("left", "leaf")
			child("right", "fork")
		case "relay-above":
			child("child", "fork")
		case "relay-left":
			child("left", "relay-leaf")
			child("right", "leaf")
		case "relay-leaf":
			child("child", "leaf")
		case "leaf":
			n.input = s.Requests[index]
			child("e", s.Assignment[index:index+1])
			child("S", "S")
			index++
		case "B":
			child("h", "h")
		case "A", "S", "h":
		default:
			panic("unknown shape " + shape)
		}
		if shape == "A" || shape == "B" || shape == "S" || shape == "h" {
			n.input = s.Requests[index]
		}
		return n
	}
	return build("O", "outer")
}

type event struct {
	ID         string
	Owner      string
	Facts      []string // In-transaction ordered word.
	SnapshotOf string   `json:",omitempty"` // First decision reads this source in its prestate.
	After      []string // Conjunction. All incoming edges are required.
}

type graph map[string]event

func key(role, action string) string { return role + ":" + action }

type observation struct {
	end, envelope string // empty end is the lawful pending case
	payload       string // canonical role-indexed application data in this summary, empty if absent
}

func product(parts []observation) observation {
	status := "succeeded"
	var fields []string
	for _, p := range parts {
		if p.end == "" {
			return observation{}
		}
		if p.end == "failed" {
			status = "failed"
		}
		fields = append(fields, p.payload)
	}
	return observation{status, "member_" + status, strings.Join(fields, ";")}
}

// Build by executing a local causal script for each role. Forking the script does not
// wait for the child, and Send launches a concurrent two-step disposition service.
func submitted(root *member) graph {
	g := graph{}
	var run func(*member, string, string) observation
	run = func(n *member, caller, start string) observation {
		role := n.role
		add := func(target, kind, owner string, facts []string, deps ...string) string {
			id := key(target, kind)
			e := event{ID: id, Owner: owner, Facts: facts, After: deps}
			switch kind {
			case "allow":
				e.SnapshotOf = caller
			case "reply_allow":
				e.SnapshotOf = role
			}
			g[id] = e
			return id
		}
		pc := start
		if caller != "" {
			auth := add(role, "allow", "store", []string{"first_invoke_allow"}, start)
			admit := add(role, "admit", "store", []string{"admit", "create_member", "request_deliver", "charge_I1_R1"}, auth)
			pc = add(role, "receive", role, []string{"request_consume", inputs(n)}, admit)
		}
		parts := make([]observation, len(n.children))
		for i, c := range n.children {
			var preds []string
			if pc != "" {
				preds = []string{pc}
			}
			pc = add(c.role, "invoke", role, []string{"invoke_commit", inputs(c)}, preds...)
			parts[i] = run(c, role, pc)
		}
		order := make([]int, len(parts))
		for i := range order {
			order[i] = i
		}
		if n.kind == "leaf" {
			order = []int{1, 0}
		}
		for _, i := range order {
			if parts[i].end == "" {
				return observation{}
			}
			c := n.children[i]
			pc = add(c.role, "select", role, []string{"outcome_consume", parts[i].envelope, parts[i].payload}, pc, key(c.role, "publish"))
		}
		result := observation{"succeeded", "member_succeeded", ""}
		switch n.kind {
		case "S":
			result.payload = fmt.Sprintf("%d,0", n.input.ID)
		case "h":
			result.payload = fmt.Sprintf("%d,%d", n.input.ID, 1-n.input.Value)
		case "A", "B":
			switch n.input.Mode {
			case "pending":
				add(role, "unmatched", role, []string{"pending_receive"}, pc)
				return observation{}
			case "fail":
				result.end = "failed"
				result.envelope = "member_failed"
			case "absent":
				result.envelope = "missing_reply"
			case "answer":
				if n.kind == "B" {
					result.payload = parts[0].payload
				} else {
					result.payload = fmt.Sprintf("%d,%d", n.input.ID, 1-n.input.Value)
				}
			default:
				panic("unknown mode")
			}
		case "leaf":
			candidate := parts[0]
			tag := "answer"
			switch candidate.envelope {
			case "member_failed":
				tag = "candidate_failed"
			case "missing_reply":
				tag = "candidate_missing_reply"
			}
			if tag != "answer" {
				result.end = "failed"
				result.envelope = "member_failed"
			}
			result.payload = fmt.Sprintf("%s=%d,%s,[%s]", role, n.input.ID, tag, candidate.payload)
		default:
			result = product(parts)
		}
		var disposition string
		if caller != "" && result.payload != "" {
			pc = add(role, "send", role, []string{"reply_commit", result.payload}, pc)
			a := add(role, "reply_allow", "store", []string{"first_reply_allow"}, pc)
			disposition = add(role, "record", "store", []string{"recorded_reply"}, a)
		}
		if caller == "" {
			decision := "answer"
			if result.end == "failed" {
				decision = "diagnosed"
			}
			pc = add(role, "decide", role, []string{decision}, pc)
		}
		end := add(role, "end", role, []string{"end_" + result.end, "freeze"}, pc)
		if caller == "" {
			add(role, "publish", "store", []string{"root_publish", result.end}, end)
		} else {
			deps := []string{end}
			if disposition != "" {
				deps = append(deps, disposition)
			}
			add(role, "publish", "store", []string{"invocation_publish", result.envelope, result.payload, "outcome_fact", "outcome_deliver"}, deps...)
		}
		return result
	}
	run(root, "", "")
	return g
}

// Role-ordered record view. Preserve numeric literals and the distinct S/h input schemas.
func inputs(n *member) string {
	q := n.input
	switch n.kind {
	case "S":
		return fmt.Sprintf("%d,%d", q.ID, q.Value)
	case "h":
		q.Mode = "answer"
	case "A", "B", "leaf":
	default:
		parts := []string{}
		for _, c := range n.children {
			parts = append(parts, inputs(c))
		}
		return strings.Join(parts, ";")
	}
	return fmt.Sprintf("%d,%d,%s", q.ID, q.Value, q.Mode)
}
