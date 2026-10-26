// Package evidence builds a statically composed, one-question evidence assistant.
// All planning, retrieval and citation policy runs as ordinary authored steps.
package evidence

import (
	"fmt"
	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
)

const ModelCapability = "evidence-model"
const LookupCapability = "evidence-lookup"
const planningPrompt = `Choose lookup, clarification, or insufficient_evidence. For lookup set tool to exactly "lookup" and supply a nonempty query of at most 512 UTF-8 bytes AND at most 32 lexical terms. A lexical term is a run of letters or numbers: punctuation splits terms, including in version numbers, and repeated terms still count toward the limit. The tool retrieves one best-matching line by case-insensitive exact term overlap, without stemming. Use a few concise, discriminative entity and fact/action keywords, not a verbose natural-language restatement of the question. Include only entity, version or deployment qualifiers needed to distinguish the requested facts; broad context can match a header instead of an answer-bearing line. Include useful action/status vocabulary and relevant alternate word forms when needed to target the fact itself. Never propose another tool or endpoint. Use message for clarification or insufficient evidence. Return all schema fields; unused fields are empty strings.`
const synthesisPrompt = `Answer the question using only the supplied evidence. Return a nonempty answer plus exactly one citation: copy document, version, snippet, start and end exactly from the evidence. Do not invent evidence or alter the quoted snippet. The citation is mechanically checked, not proof of every sentence.`

func str(n int) schema.Field { return schema.Field{Kind: "string", MaxBytes: n} }

// RequestSchema is the helper's one-question public request contract.
func RequestSchema() schema.Record { return schema.Record{"question": str(4096)} }

// ExcludingRequestSchema adds two exact source-span exclusions to the public
// question contract. See lookup.ExcludingInputSchema for tuple validation.
func ExcludingRequestSchema() schema.Record {
	r := lookup.ExcludingInputSchema()
	delete(r, "query")
	r["question"] = str(4096)
	return r
}

// ResultSchema is the public result contract. Non-answer results carry no citation.
func ResultSchema() schema.Record {
	return schema.Record{"disposition": {Kind: "string", MaxBytes: 128, Enum: []string{"answer", "clarification", "insufficient_evidence", "no_match", "invalid_result", "budget_exhausted"}}, "answer": str(8192), "document": str(512), "version": str(64), "snippet": str(2048), "start": {Kind: "int"}, "end": {Kind: "int"}}
}

// Budgets are logical effect ceilings. Zero explicitly disables that effect.
// At most ten requests of either kind may be configured; this version uses at
// most two model requests and one lookup and does not spend remaining capacity.
type Budgets struct{ MaxModelRequests, MaxLookupRequests int }

// Component contains static instances and the port a caller can use without
// knowing any internal subject names. Add Instances to the caller's Application.
type Component struct {
	Instances author.Application
	Port      author.Port
}

// New compiles an ordinary planner/synthesizer and retrieval worker. name is the
// public endpoint; caller and obligation select its statically wired peer.
func New(name, caller, obligation string, b Budgets) (Component, error) {
	return newComponent(name, caller, obligation, b, false)
}

// NewExcluding uses the same ordinary helper program with an explicitly chosen
// request/lookup profile. Exclusions are caller data forwarded by ordinary
// messages; model output cannot change them. New retains its legacy contract.
func NewExcluding(name, caller, obligation string, b Budgets) (Component, error) {
	return newComponent(name, caller, obligation, b, true)
}

func newComponent(name, caller, obligation string, b Budgets, excluding bool) (Component, error) {
	var out Component
	if b.MaxModelRequests < 0 || b.MaxModelRequests > 10 || b.MaxLookupRequests < 0 || b.MaxLookupRequests > 10 {
		return out, fmt.Errorf("logical budgets must be between 0 and 10")
	}
	retrieval := name + "/retrieval"
	retrievalInput := schema.Record{"query": str(512), "execute": {Kind: "bool"}}
	requestSchema := RequestSchema()
	lookupSchema := lookup.InputSchema()
	if excluding {
		requestSchema = ExcludingRequestSchema()
		lookupSchema = lookup.ExcludingInputSchema()
		for k, f := range lookupSchema {
			retrievalInput[k] = f
		}
	}
	retrievalRequest := map[string]author.Expr{"query": author.Var("plan_query"), "execute": author.Boolean(true)}
	skippedRequest := map[string]author.Expr{"query": author.String(""), "execute": author.Boolean(false)}
	for k := range lookupSchema {
		if k == "query" {
			continue
		}
		retrievalRequest[k] = author.Var(k)
		skippedRequest[k] = author.Var(k)
	}

	retrievalOutput, _ := schema.ResultRecord(lookup.OutputSchema())
	retrievalOutput["budget_exhausted"] = schema.Field{Kind: "bool"}
	workerValues := map[string]schema.Value{"lookup_spent": schema.Int(0), "lookup_limit": schema.Int(int64(b.MaxLookupRequests)), "pending_lookup": schema.Text(""), "budget_exhausted": schema.Bool(false), "valid": schema.Bool(false), "disposition": schema.Text("skipped")}
	for k, f := range lookup.OutputSchema() {
		workerValues[k] = schema.Value{Kind: f.Kind}
	}
	workerSteps := []author.Step{
		author.Receive("assistant", bindings(retrievalInput, "")),
		author.IfEqual(author.Var("execute"), author.Boolean(false), []author.Step{author.Goto("reply")}, nil),
		author.IfEqual(author.Var("lookup_spent"), author.Var("lookup_limit"), []author.Step{author.Set("budget_exhausted", author.Boolean(true)), author.Goto("reply")}, nil),
		author.Label("lookup"), author.Request("pending_lookup", "lookup", expressions(lookupSchema, ""), map[string]author.Expr{"lookup_spent": author.Offset("lookup_spent", 1)}),
		author.Await("lookup", "pending_lookup", bindings(lookupResultSchema(), "")),
		author.Label("reply"), author.Send("assistant", expressions(retrievalOutput, "")), author.Success(),
	}
	worker, err := author.Compile(author.Definition{Name: "evidence retrieval", Values: workerValues, Ports: map[string]author.Port{"assistant": {Peer: name, Obligation: "retrieval", Receive: retrievalInput, Send: retrievalOutput}}, Effects: map[string]author.Effect{"lookup": {Capability: LookupCapability, Input: lookupSchema, Output: lookup.OutputSchema()}}, Steps: workerSteps})
	if err != nil {
		return out, err
	}
	decision := schema.Record{"decision": {Kind: "string", MaxBytes: 64, Enum: []string{"lookup", "clarification", "insufficient_evidence"}}, "tool": str(512), "query": str(512), "message": str(8192)}
	synthesis := schema.Record{"answer": str(8192), "document": str(512), "version": str(64), "snippet": str(2048), "start": {Kind: "int"}, "end": {Kind: "int"}}
	planInput := schema.Record{"instructions": str(4096), "question": str(4096)}
	synthInput := lookup.OutputSchema()
	synthInput["instructions"] = str(4096)
	synthInput["question"] = str(4096)
	values := map[string]schema.Value{"model_spent": schema.Int(0), "model_limit": schema.Int(int64(b.MaxModelRequests)), "pending_plan": schema.Text(""), "pending_synthesis": schema.Text(""), "retrieval_done": schema.Bool(false), "result_disposition": schema.Text("invalid_result")}
	for k, f := range ResultSchema() {
		if k != "disposition" {
			values["result_"+k] = schema.Value{Kind: f.Kind}
		}
	}
	finish := func(disposition string) []author.Step {
		return []author.Step{author.Set("result_disposition", author.String(disposition)), author.Goto("reply")}
	}
	guard := func(a, b author.Expr) author.Step { return author.IfEqual(a, b, nil, finish("invalid_result")) }
	nonempty := func(name string) author.Step {
		return author.IfEqual(author.Var(name), author.String(""), finish("invalid_result"), nil)
	}
	budget := func() author.Step {
		return author.IfEqual(author.Var("model_spent"), author.Var("model_limit"), finish("budget_exhausted"), nil)
	}
	planEnvelope, _ := schema.ResultRecord(decision)
	synthEnvelope, _ := schema.ResultRecord(synthesis)
	steps := []author.Step{
		author.Receive("caller", bindings(requestSchema, "")),
		author.Label("planning"), budget(),
		author.Request("pending_plan", "plan", map[string]author.Expr{"instructions": author.String(planningPrompt), "question": author.Var("question")}, map[string]author.Expr{"model_spent": author.Offset("model_spent", 1)}),
		author.Await("plan", "pending_plan", bindings(planEnvelope, "plan_")),
		guard(author.Var("plan_valid"), author.Boolean(true)),
		author.IfEqual(author.Var("plan_decision"), author.String("clarification"), append([]author.Step{nonempty("plan_message"), author.Set("result_answer", author.Var("plan_message"))}, finish("clarification")...), nil),
		author.IfEqual(author.Var("plan_decision"), author.String("insufficient_evidence"), append([]author.Step{nonempty("plan_message"), author.Set("result_answer", author.Var("plan_message"))}, finish("insufficient_evidence")...), nil),
		guard(author.Var("plan_decision"), author.String("lookup")), guard(author.Var("plan_tool"), author.String("lookup")), nonempty("plan_query"),
		author.Label("retrieval"), author.Send("retrieval", retrievalRequest),
		author.Receive("retrieval", bindings(retrievalOutput, "evidence_")), author.Set("retrieval_done", author.Boolean(true)),
		author.IfEqual(author.Var("evidence_budget_exhausted"), author.Boolean(true), finish("budget_exhausted"), nil),
		author.IfEqual(author.Var("evidence_disposition"), author.String("no_match"), finish("no_match"), nil),
		guard(author.Var("evidence_valid"), author.Boolean(true)),
		author.Label("synthesis"), budget(),
	}
	input := expressions(lookup.OutputSchema(), "evidence_")
	input["question"] = author.Var("question")
	input["instructions"] = author.String(synthesisPrompt)
	steps = append(steps, author.Request("pending_synthesis", "synthesize", input, map[string]author.Expr{"model_spent": author.Offset("model_spent", 1)}), author.Await("synthesize", "pending_synthesis", bindings(synthEnvelope, "synth_")), guard(author.Var("synth_valid"), author.Boolean(true)), nonempty("synth_answer"))
	for _, k := range []string{"document", "version", "snippet", "start", "end"} {
		steps = append(steps, guard(author.Var("synth_"+k), author.Var("evidence_"+k)))
	}
	for _, k := range []string{"answer", "document", "version", "snippet", "start", "end"} {
		steps = append(steps, author.Set("result_"+k, author.Var("synth_"+k)))
	}
	steps = append(steps, author.Set("result_disposition", author.String("answer")), author.Label("reply"),
		author.IfEqual(author.Var("retrieval_done"), author.Boolean(false), []author.Step{author.Send("retrieval", skippedRequest), author.Receive("retrieval", map[string]string{"disposition": "skipped_retrieval"})}, nil),
		author.Send("caller", expressions(ResultSchema(), "result_")), author.IfEqual(author.Var("result_disposition"), author.String("invalid_result"), []author.Step{author.Failure()}, []author.Step{author.Success()}))
	assistant, err := author.Compile(author.Definition{Name: "evidence assistant", Values: values, Templates: map[string]string{"planning": planningPrompt, "synthesis": synthesisPrompt}, Ports: map[string]author.Port{"caller": {Peer: caller, Obligation: obligation, Receive: requestSchema, Send: ResultSchema()}, "retrieval": {Peer: retrieval, Obligation: "retrieval", Send: retrievalInput, Receive: retrievalOutput}}, Effects: map[string]author.Effect{"plan": {Capability: ModelCapability, Input: planInput, Output: decision}, "synthesize": {Capability: ModelCapability, Input: synthInput, Output: synthesis}}, Steps: steps})
	if err != nil {
		return out, err
	}
	out.Instances = author.Application{name: {Artifact: assistant}, retrieval: {Artifact: worker}}
	out.Port = author.Port{Peer: name, Obligation: obligation, Send: requestSchema, Receive: ResultSchema()}
	return out, nil
}
func lookupResultSchema() schema.Record { r, _ := schema.ResultRecord(lookup.OutputSchema()); return r }
func bindings(r schema.Record, prefix string) map[string]string {
	v := map[string]string{}
	for k := range r {
		v[k] = prefix + k
	}
	return v
}
func expressions(r schema.Record, prefix string) map[string]author.Expr {
	v := map[string]author.Expr{}
	for k := range r {
		v[k] = author.Var(prefix + k)
	}
	return v
}
