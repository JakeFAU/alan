// Package delegation preserves pre-delegation public-runtime authority probes.
package delegation

import (
	"fmt"
	"sort"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type BaselineScenario string

const (
	BaselineLocal         BaselineScenario = "local"
	BaselineMissing       BaselineScenario = "missing"
	BaselineClaim         BaselineScenario = "claim"
	BaselineResultLaunder BaselineScenario = "result_launder"
	BaselineCartesian     BaselineScenario = "cartesian"
	BaselineCounterARed                    = "phase3b3.counter-A.red"
	BaselineCounterBBlue                   = "phase3b3.counter-B.blue"
	BaselineCounterABlue                   = "phase3b3.counter-A.blue"
)

// Baseline provisions static principal-local complete permission tuples. Local is
// the A-servicing/B-serial S1 control; Missing, Claim and Cartesian preserve its
// ordering and attempt an unheld tuple. ResultLaunder is a separate reversed
// ordering negative control: B services first so A can consume a genuine B result
// before its refused attempt. It does not qualify the positive service policy.
// Capability endpoint adapters are supplied separately by trusted deployment.
func Baseline(s BaselineScenario) (runtime.Provision, error) {
	switch s {
	case BaselineLocal, BaselineMissing, BaselineClaim, BaselineResultLaunder, BaselineCartesian:
	default:
		return runtime.Provision{}, fmt.Errorf("unknown authority scenario %q", s)
	}
	aCapability := BaselineCounterARed
	if s == BaselineMissing || s == BaselineClaim || s == BaselineResultLaunder {
		aCapability = BaselineCounterBBlue
	}
	if s == BaselineCartesian {
		aCapability = BaselineCounterABlue
	}
	a, err := baselineRoot("a", "b", aCapability, s != BaselineResultLaunder, s)
	if err != nil {
		return runtime.Provision{}, err
	}
	b, err := baselineRoot("b", "a", BaselineCounterBBlue, s == BaselineResultLaunder, s)
	if err != nil {
		return runtime.Provision{}, err
	}
	claim := "ordinary-data"
	if s == BaselineClaim {
		claim = "grant:issuer=b;grantee=a;parent=b-root;permission=increment/counter-B/blue"
	}
	p, err := (author.Application{
		"a": {Artifact: a, Inputs: map[string]schema.Value{"own_id": schema.Text("a-request"), "claim": schema.Text(claim)}},
		"b": {Artifact: b, Inputs: map[string]schema.Value{"own_id": schema.Text("b-request"), "claim": schema.Text(claim)}},
	}).Provision()
	if err != nil {
		return p, err
	}
	// The author automatically grants declared effects. Remove the unwanted grant
	// through the public Provision before Create; never edit the positive database.
	filtered := p.Permissions[:0]
	for _, v := range p.Permissions {
		if v.Kind == "adapter" && v.Source == "a" {
			continue
		}
		filtered = append(filtered, v)
	}

	// Authoring declares the attempted operation; deployment alone fixes authority.
	// Keep permission ordering deterministic and never broaden A for S2 attempts.
	filtered = append(filtered, runtime.Permission{Kind: "adapter", Source: "a", Target: BaselineCounterARed, Operation: "increment", Revision: 1, Obligation: "increment"})
	sort.Slice(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Target != b.Target {
			return a.Target < b.Target
		}
		return a.Obligation < b.Obligation
	})
	p.Permissions = filtered
	return p, nil
}

func baselineRoot(subject, peer, capability string, serviceFirst bool, s BaselineScenario) (author.Artifact, error) {
	request := schema.Record{"request_id": {Kind: "string", MaxBytes: 32}, "claim": {Kind: "string", MaxBytes: 256}}
	response := schema.Record{"response_id": {Kind: "string", MaxBytes: 32}, "mutation": {Kind: "int"}, "total": {Kind: "int"}, "service_operation": {Kind: "string", MaxBytes: 256}}
	hold := func() author.Step { return author.Receive("hold", map[string]string{"release": "release"}) }
	claim := author.Var("peer_claim")
	if s == BaselineResultLaunder && subject == "a" {
		claim = author.Var("result_claim")
	}
	serve := []author.Step{
		author.Receive("incoming", map[string]string{"request_id": "peer_id", "claim": "peer_claim"}),
		author.Request("operation", "increment", map[string]author.Expr{"delta": author.Integer(1), "claim": claim, "receipt_mutation": author.Var("receipt_mutation")}, nil),
		author.Await("increment", "operation", map[string]string{"valid": "effect_valid", "disposition": "effect_disposition", "mutation": "mutation", "total": "total"}),
		author.IfEqual(author.Var("effect_valid"), author.Boolean(true), []author.Step{author.Send("incoming", map[string]author.Expr{"response_id": author.Var("peer_id"), "mutation": author.Var("mutation"), "total": author.Var("total"), "service_operation": author.Var("operation")})}, []author.Step{author.Set("denied", author.Boolean(true)), hold()}),
	}
	consume := []author.Step{
		author.Receive("outgoing", map[string]string{"response_id": "reply_id", "mutation": "reply_mutation", "total": "reply_total", "service_operation": "reply_operation"}),
		author.IfEqual(author.Var("reply_id"), author.Var("own_id"), []author.Step{author.Set("paid", author.Boolean(true))}, []author.Step{author.Failure()}),
		// This value is derived only after consuming the authentic B result. It
		// remains ordinary authored data and cannot confer B's adapter permission.
		author.Set("result_claim", author.String("result-from-b-is-not-authority")),
		author.Set("receipt_mutation", author.Var("reply_mutation")),
	}
	steps := []author.Step{author.Send("outgoing", map[string]author.Expr{"request_id": author.Var("own_id"), "claim": author.Var("claim")})}
	if serviceFirst {
		steps = append(steps, serve...)
		steps = append(steps, consume...)
	} else {
		steps = append(steps, consume...)
		steps = append(steps, serve...)
	}
	steps = append(steps, hold(), author.Success())
	return author.Compile(author.Definition{
		Name:    "phase3b3-baseline-" + string(s) + "-" + subject,
		Inputs:  schema.Record{"own_id": {Kind: "string", MaxBytes: 32}, "claim": {Kind: "string", MaxBytes: 256}},
		Values:  map[string]schema.Value{"paid": schema.Bool(false), "denied": schema.Bool(false), "receipt_mutation": schema.Int(0)},
		Effects: map[string]author.Effect{"increment": {Capability: capability, Input: schema.Record{"delta": {Kind: "int"}, "claim": {Kind: "string", MaxBytes: 256}, "receipt_mutation": {Kind: "int"}}, Output: schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}}}},
		Ports: map[string]author.Port{
			"outgoing": {Peer: peer, Obligation: "phase3b3_baseline_" + subject + "_to_" + peer, Send: request, Receive: response},
			"incoming": {Peer: peer, Obligation: "phase3b3_baseline_" + peer + "_to_" + subject, Send: response, Receive: request},
			"hold":     {Peer: "local", Local: true, Obligation: "phase3b3_baseline_hold", Receive: schema.Record{"release": {Kind: "bool"}}},
		}, Steps: steps,
	})
}
