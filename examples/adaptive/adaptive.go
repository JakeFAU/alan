// Package adaptive authors bounded, result-dependent work with ordinary durable
// programs. Go constructs immutable artifacts; it never orchestrates a frontier.
package adaptive

import (
	"fmt"
	"reflect"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

func build(leaf author.Template, p profile) (author.Template, error) {
	in, out, err := leaf.Schemas()
	if err != nil {
		return author.Template{}, err
	}
	if p.payload == "value" && (!reflect.DeepEqual(in.Record, schema.Record{"value": integer()}) || !reflect.DeepEqual(out.Record, schema.Record{"observation": integer()})) {
		return author.Template{}, fmt.Errorf("scalar leaf must have exact value/observation int contract")
	}
	frontier := schema.Record{"work_id": integer(), "parent_id": integer(), "edge": text(8), "round": integer(), p.payload: p.payloadField}
	response := list(1, schema.Record{p.observation: p.responseField})
	result := schema.Record{}
	for k, v := range frontier {
		result[k] = v
	}
	result["operation"] = text(4096)
	result["status"] = text(32)
	result["reason"] = text(4096)
	result["responses"] = response
	request := schema.Record{"seeds": list(8, schema.Record{"work_id": integer(), p.payload: p.payloadField}), "max_rounds": integer(), "max_work": integer()}
	reply := schema.Record{"complete": {Kind: "bool"}, "stop_reason": text(32), "rounds": integer(), "work_count": integer(), "results": list(8, result), "remaining": list(16, frontier)}
	collections := map[string]schema.Field{
		"tokens":  list(3, schema.Record{"round": integer()}),
		"current": list(16, frontier), "next": list(16, frontier), "handles": list(8, schema.Record{"work_id": integer(), "operation": text(4096)}),
		"history": list(8, schema.Record{"work_id": integer()}), "round_results": list(8, result), "results": reply["results"], "responses": response,
		"decision": list(1, schema.Record{"round": integer(), "candidate": list(16, frontier), "stop_reason": text(32), "complete": {Kind: "bool"}}),
	}
	values := map[string]schema.Value{"rounds": schema.Int(0), "work_count": schema.Int(0), "trial": schema.Int(0), "complete": schema.Bool(false), "stop_reason": schema.Text(""), "hard": schema.Bool(false), "active": schema.Bool(true), "tokens": schema.List(map[string]schema.Value{"round": schema.Int(1)}, map[string]schema.Value{"round": schema.Int(2)}, map[string]schema.Value{"round": schema.Int(3)})}
	for _, n := range []string{"duplicate", "overflow", "empty_frontier"} {
		values[n] = schema.Bool(false)
	}

	steps := []author.Step{author.Receive("public", bindings("seeds", "max_rounds", "max_work")), setBool("valid_rounds", false), setBool("valid_work", false), setBool("valid_ids", true)}
	for i := int64(1); i <= 3; i++ {
		steps = append(steps, eq("max_rounds", author.Integer(i), setBool("valid_rounds", true)))
	}
	for i := int64(1); i <= 8; i++ {
		steps = append(steps, eq("max_work", author.Integer(i), setBool("valid_work", true)))
	}
	seedBody := []author.Step{setBool("valid_id", false)}
	for i := int64(1); i <= 8; i++ {
		seedBody = append(seedBody, eq("work_id", author.Integer(i), setBool("valid_id", true)))
	}
	seedBody = append(seedBody, eq("valid_id", author.Boolean(false), setBool("valid_ids", false)), author.Append("current", map[string]author.Expr{"work_id": author.Var("work_id"), "parent_id": author.Integer(0), "edge": author.String("seed"), "round": author.Integer(1), p.payload: author.Var(p.payload)}))
	steps = append(steps, author.ForEach(author.Var("seeds"), bindings("work_id", p.payload), seedBody...))
	for _, name := range []string{"valid_rounds", "valid_work", "valid_ids"} {
		steps = append(steps, eq(name, author.Boolean(false), stop("invalid_request", true)...))
	}
	steps = append(steps, eq("active", author.Boolean(true), preflight("current", false)...),
		eq("active", author.Boolean(true), eq("duplicate", author.Boolean(true), stop("duplicate_work_id", true)...)),
		eq("active", author.Boolean(true), eq("overflow", author.Boolean(true), stop("work_limit", false)...)),
		eq("active", author.Boolean(true), eq("empty_frontier", author.Boolean(true), stop("exhausted", false)...)))
	steps = append(steps, author.ForEach(author.Var("tokens"), map[string]string{"round": "round"}, eq("active", author.Boolean(true), roundBody(p)...)))
	steps = append(steps, author.Send("public", map[string]author.Expr{"complete": author.Var("complete"), "stop_reason": author.Var("stop_reason"), "rounds": author.Var("rounds"), "work_count": author.Var("work_count"), "results": author.Var("results"), "remaining": author.Var("current")}), author.IfEqual(author.Var("hard"), author.Boolean(true), []author.Step{author.Failure()}, []author.Step{author.Success()}))
	a, err := author.Compile(author.Definition{Name: "bounded adaptive " + p.payload, Invocations: map[string]author.Template{"leaf": leaf}, Collections: collections, Values: values, Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: request, Send: reply}}, Steps: steps})
	if err != nil {
		return author.Template{}, err
	}
	return author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
}

func stop(reason string, hard bool) []author.Step {
	return []author.Step{author.Set("stop_reason", author.String(reason)), setBool("hard", hard), setBool("active", false), setBool("complete", reason == "exhausted")}
}

// Equality-before-increment checks capacity without comparison primitives. Pair
// positions avoid self-comparison; history retains only previously admitted IDs.
func preflight(name string, history bool) []author.Step {
	body := []author.Step{setBool("empty_frontier", false), eq("trial", author.Var("max_work"), setBool("overflow", true)), eq("overflow", author.Boolean(false), increment("trial")), increment("position"), setInt("other_position", 0),
		author.ForEach(author.Var(name), map[string]string{"work_id": "other_id"}, increment("other_position"), author.IfEqual(author.Var("position"), author.Var("other_position"), nil, []author.Step{eq("check_id", author.Var("other_id"), setBool("duplicate", true))})),
	}
	if history {
		body = append(body, author.ForEach(author.Var("history"), map[string]string{"work_id": "history_id"}, eq("check_id", author.Var("history_id"), setBool("duplicate", true))))
	}
	return []author.Step{setBool("duplicate", false), setBool("overflow", false), setBool("empty_frontier", true), author.Set("trial", author.Var("work_count")), setInt("position", 0), author.ForEach(author.Var(name), map[string]string{"work_id": "check_id"}, body...)}
}
func roundBody(p profile) []author.Step {
	fields := []string{"work_id", "parent_id", "edge", "round", p.payload}
	// history tracks the prefix whose item-emission loop has begun. The full
	// current frontier and work_count retain application acceptance; history is
	// complete before the next candidate collision check. It carries no charges.
	invoke := []author.Step{author.Append("history", exprs("work_id")), author.Invoke("pending", "leaf", map[string]author.Expr{p.payload: author.Var(p.payload)}, nil), author.Append("handles", map[string]author.Expr{"work_id": author.Var("work_id"), "operation": author.Var("pending")})}
	row := exprs(fields...)
	row["operation"] = author.Var("pending")
	row["status"] = author.Var("leaf_status")
	row["reason"] = author.Var("leaf_reason")
	row["responses"] = author.Var("responses")
	collect := []author.Step{
		author.AwaitInvocation("leaf", "pending", author.InvocationOutcomeBindings{Status: "leaf_status", Reason: "leaf_reason", ResponsePresent: "leaf_present", Response: map[string]string{p.observation: "leaf_observation"}}),
		empty("responses"), eq("leaf_present", author.Boolean(true), author.Append("responses", map[string]author.Expr{p.observation: author.Var("leaf_observation")})),
		author.Append("round_results", row), author.Append("results", row),
		author.IfEqual(author.Var("leaf_status"), author.String("succeeded"), nil, []author.Step{eq("failure_reason", author.String(""), author.Set("failure_reason", author.String("leaf_non_success")), eq("leaf_status", author.String("rejected"), eq("leaf_reason", author.String("invocation_limit"), author.Set("failure_reason", author.String("root_budget_denied"))), eq("leaf_reason", author.String("created_run_limit"), author.Set("failure_reason", author.String("root_budget_denied")))))}),
	}
	resultBindings := bindings(fields...)
	resultBindings["status"] = "derive_status"
	resultBindings["responses"] = "derive_responses"
	one := append([]author.Step{setBool("valid_response", true)}, child(p, 1, "check", 0)...)
	two := append([]author.Step{setBool("valid_response", true)}, child(p, 1, "left", 1)...)
	two = append(two, child(p, 2, "right", 0)...)
	classify := append(p.classify(),
		eq("observation", author.Integer(0), setBool("valid_response", true)),
		eq("observation", author.Integer(1), one...),
		eq("observation", author.Integer(2), two...),
	)
	derive := []author.Step{eq("derive_status", author.String("succeeded"),
		setBool("valid_response", false),
		author.ForEach(author.Var("derive_responses"), map[string]string{p.observation: "leaf_observation"}, classify...),
		eq("valid_response", author.Boolean(false), setBool("invalid_observation", true)),
	)}

	// Preflight computed the whole accepted total. Retain it before any item
	// intent; runtime admission/effect charges remain separate actual facts.
	steps := []author.Step{author.Set("work_count", author.Var("trial")), empty("handles"), empty("round_results"), empty("next"), setBool("invalid_observation", false), author.Set("failure_reason", author.String("")), author.ForEach(author.Var("current"), bindings(fields...), invoke...),
		author.ForEach(author.Var("current"), bindings(fields...), author.ForEach(author.Var("handles"), map[string]string{"work_id": "handle_id", "operation": "pending"}, eq("work_id", author.Var("handle_id"), collect...))),
		increment("rounds"), author.ForEach(author.Var("round_results"), resultBindings, derive...),
	}
	steps = append(steps, preflight("next", true)...)
	// Assign from lowest to highest priority; the final matching condition wins.
	steps = append(steps, eq("round", author.Var("max_rounds"), stop("round_limit", false)...), eq("overflow", author.Boolean(true), stop("work_limit", false)...), eq("empty_frontier", author.Boolean(true), stop("exhausted", false)...), eq("duplicate", author.Boolean(true), stop("duplicate_work_id", true)...), eq("invalid_observation", author.Boolean(true), stop("invalid_observation", true)...), author.IfEqual(author.Var("failure_reason"), author.String(""), nil, []author.Step{author.Set("stop_reason", author.Var("failure_reason")), setBool("hard", true), setBool("active", false), setBool("complete", false)}),
		empty("decision"), author.Append("decision", map[string]author.Expr{"round": author.Var("round"), "candidate": author.Var("next"), "stop_reason": author.Var("stop_reason"), "complete": author.Var("complete")}), author.Set("current", author.Var("next")))
	return steps
}
func child(p profile, slot int64, edge string, value int64) []author.Step {
	payload := author.Var(p.payload)
	if p.payload == "value" {
		payload = author.Integer(value)
	}
	return []author.Step{author.Add("child_id", author.Var("work_id"), author.Var("work_id")), author.Add("child_id", author.Var("child_id"), author.Var("work_id")), author.Add("child_id", author.Var("child_id"), author.Integer(slot)), author.Append("next", map[string]author.Expr{"work_id": author.Var("child_id"), "parent_id": author.Var("work_id"), "edge": author.String(edge), "round": author.Offset("round", 1), p.payload: payload})}
}
