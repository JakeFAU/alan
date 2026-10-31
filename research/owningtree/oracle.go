package owningtree

import (
	"fmt"
	"slices"
	"strings"
)

// Predict the caller-visible packet directly from the supplied component hypothesis.
// Do not use submitted child packets or a mode test in the parent forwarding machine.
func expectedPacket(n *member) (end, reason, data string) {
	q := n.input
	switch n.kind {
	case "S", "h":
		v := 0
		if n.kind == "h" {
			v = 1 - q.Value
		}
		return "succeeded", "member_succeeded", fmt.Sprintf("%d,%d", q.ID, v)
	case "A", "B", "leaf":
		if q.Mode == "pending" {
			return "", "", ""
		}
		tag := "answer"
		value := fmt.Sprintf("%d,%d", q.ID, 1-q.Value)
		end, reason = "succeeded", "member_succeeded"
		switch q.Mode {
		case "fail":
			end, reason, tag, value = "failed", "member_failed", "candidate_failed", ""
		case "absent":
			reason, tag, value = "missing_reply", "candidate_missing_reply", ""
		}
		if n.kind != "leaf" {
			return end, reason, value
		}
		if q.Mode != "answer" {
			end, reason = "failed", "member_failed"
		}
		return end, reason, fmt.Sprintf("%s=%d,%s,[%s]", n.role, q.ID, tag, value)
	default:
		fields := []string{}
		end = "succeeded"
		for _, c := range n.children {
			status, _, field := expectedPacket(c)
			if status == "" {
				return "", "", ""
			}
			if status != "succeeded" {
				end = "failed"
			}
			fields = append(fields, field)
		}
		return end, "member_" + end, strings.Join(fields, ";")
	}
}

type boundary struct {
	Waits               []string
	DeliveredUnselected []string
	Decision            string
	Product             string
}

// Construct constraints by superimposing a serial instruction stream for each run and
// a separate Store guard for each action. No submitted event or predecessor is input.
func expected(root *member) (graph, boundary) {
	rules := graph{}
	front := boundary{}
	var visit func(*member)
	visit = func(n *member) {
		path := n.role
		end, reason, payload := expectedPacket(n)
		var local []string
		declare := func(role, action, owner string, word []string, deps ...string) string {
			id := key(role, action)
			e := event{ID: id, Owner: owner, Facts: word, After: deps}
			if action == "reply_allow" {
				e.SnapshotOf = role
			}
			if action == "allow" {
				e.SnapshotOf = role[:strings.LastIndex(role, "/")]
			}
			rules[id] = e
			return id
		}
		instruction := func(role, action string, word ...string) {
			local = append(local, declare(role, action, path, word))
		}
		depend := func(target, prior string) {
			e := rules[target]
			e.After = append(e.After, prior)
			rules[target] = e
		}
		if n != root {
			declare(path, "allow", "store", []string{"first_invoke_allow"}, key(path, "invoke"))
			declare(path, "admit", "store", []string{"admit", "create_member", "request_deliver", "charge_I1_R1"}, key(path, "allow"))
			instruction(path, "receive", "request_consume", inputs(n))
			depend(key(path, "receive"), key(path, "admit"))
		}
		for _, child := range n.children {
			instruction(child.role, "invoke", "invoke_commit", inputs(child))
			visit(child)
		}
		waitOrder := slices.Clone(n.children)
		if n.kind == "leaf" {
			slices.Reverse(waitOrder)
		}
		blocked := false
		for _, child := range waitOrder {
			state, why, report := expectedPacket(child)
			if state == "" {
				if !blocked {
					front.Waits = append(front.Waits, path+" -> "+child.role)
				}
				blocked = true
			} else if blocked {
				front.DeliveredUnselected = append(front.DeliveredUnselected, child.role)
			} else {
				instruction(child.role, "select", "outcome_consume", why, report)
				depend(key(child.role, "select"), key(child.role, "publish"))
			}
		}
		if end == "" && !blocked {
			instruction(path, "unmatched", "pending_receive")
			front.Waits = append(front.Waits, path+" -> second-entry")
		}
		if end != "" {
			if n == root {
				front.Decision = "answer"
				if end != "succeeded" {
					front.Decision = "diagnosed"
				}
				front.Product = payload
				instruction(path, "decide", front.Decision)
			} else if payload != "" {
				instruction(path, "send", "reply_commit", payload)
				declare(path, "reply_allow", "store", []string{"first_reply_allow"}, key(path, "send"))
				declare(path, "record", "store", []string{"recorded_reply"}, key(path, "reply_allow"))
			}
			instruction(path, "end", "end_"+end, "freeze")
			if n == root {
				declare(path, "publish", "store", []string{"root_publish", end}, key(path, "end"))
			} else {
				declare(path, "publish", "store", []string{"invocation_publish", reason, payload, "outcome_fact", "outcome_deliver"}, key(path, "end"))
				if payload != "" {
					depend(key(path, "publish"), key(path, "record"))
				}
			}
		}
		for i := 1; i < len(local); i++ {
			depend(local[i], local[i-1])
		}
	}
	visit(root)
	slices.Sort(front.Waits)
	slices.Sort(front.DeliveredUnselected)
	return rules, front
}
