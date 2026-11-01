package effectfulcomposition

import (
	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

// All five required selections precede any completed decision or final ACK.
// On a pending pure T both original envelopes may have been consumed while
// each resolver's independently delivered report remains unselected.
func parentProgram(o Options, l, r author.Template) (author.Artifact, error) {
	in, out, err := o.Tree.template.Schemas()
	if err != nil {
		return author.Artifact{}, err
	}
	d := author.Definition{Name: "phase2d-parent", Inputs: in.Record, Ports: map[string]author.Port{}, Invocations: map[string]author.Template{"l_issue": l, "r_issue": r, "tree": o.Tree.template}}
	args := map[string]author.Expr{}
	for f := range in.Record {
		args[f] = author.Var(f)
	}
	d.Steps = []author.Step{author.Invoke("l_invocation", "l_issue", nil, nil), author.Invoke("r_invocation", "r_issue", nil, nil), author.Invoke("tree_invocation", "tree", args, nil)}
	for _, role := range []string{"l", "r"} {
		h, relay, rep := role+"_handoff", role+"_relay", role+"_report"
		d.Ports[h] = duplex(runtime.HandoffPeer, role+"_identity", rq.ReadInput())
		d.Ports[relay] = duplex("resolver_"+role, relay, rq.ReadInput())
		d.Ports[rep] = author.Port{Peer: "resolver_" + role, Obligation: rep, Send: rq.ReadInput(), Receive: reportSchema()}
		d.Steps = append(d.Steps, receiveTarget(h, role+"_target"), sendTarget(relay, role+"_target"), receiveTarget(relay, role+"_resolver_ack"), eq(role+"_resolver_ack", author.Var(role+"_target"), nil, []author.Step{author.Failure()}), sendTarget(h, role+"_resolver_ack"))
	}
	for _, role := range []string{"l", "r"} {
		b := map[string]string{}
		for f := range issuance() {
			b[f] = role + "_" + f
		}
		b["target"] = role + "_issued_target"
		d.Steps = append(d.Steps, author.AwaitInvocation(role+"_issue", role+"_invocation", author.InvocationOutcomeBindings{Status: role + "_source_status", Reason: role + "_source_reason", ResponsePresent: role + "_source_present", Response: b}))
	}
	treeResponse := map[string]string{}
	for f := range out.Record {
		treeResponse[f] = "tree_" + f
	}
	d.Steps = append(d.Steps, author.AwaitInvocation("tree", "tree_invocation", author.InvocationOutcomeBindings{Status: "tree_status", Reason: "tree_reason", ResponsePresent: "tree_present", Response: treeResponse}))
	for _, role := range []string{"l", "r"} {
		b := map[string]string{}
		for f := range reportSchema() {
			b[f] = role + "_report_" + f
		}
		d.Steps = append(d.Steps, author.Receive(role+"_report", b))
	}
	d.Steps = append(d.Steps, setBool("accepted", true))
	d.Steps = append(d.Steps, roleChecks("l")...)
	d.Steps = append(d.Steps, roleChecks("r")...)
	d.Steps = append(d.Steps, treeChecks(o.Tree.paths)...)
	bad := []author.Step{setBool("accepted", false)}
	sum := author.Integer(o.Left.Delta + o.Right.Delta)
	order := func(first, second string, delta int64) []author.Step {
		return []author.Step{
			require(first+"_report_witness_mutation", author.Integer(1), bad), require(first+"_report_witness_total", author.Integer(delta), bad),
			require(second+"_report_witness_mutation", author.Integer(2), bad), require(second+"_report_witness_total", sum, bad),
		}
	}
	d.Steps = append(d.Steps, eq("l_target", author.Var("r_target"), bad, nil), eq("l_report_witness_mutation", author.Integer(1), order("l", "r", o.Left.Delta), order("r", "l", o.Right.Delta)))
	d.Steps = append(d.Steps, eq("accepted", author.Boolean(true), []author.Step{eq("tree_class", author.String("answer"), []author.Step{setText("decision", "applied")}, []author.Step{setText("decision", "applied_tree_diagnosed")})}, []author.Step{setText("decision", "invalid_evidence")}))
	// Deliberately no ACK-of-ACK. The automatic service argument and the
	// manual counterexample apply to these exact final ordinary sends.
	d.Steps = append(d.Steps, sendTarget("l_report", "l_target"), sendTarget("r_report", "r_target"), eq("decision", author.String("applied"), []author.Step{author.Success()}, []author.Step{author.Failure()}))
	return author.Compile(d)
}

func roleChecks(role string) []author.Step {
	p := role + "_"
	w := p + "report_"
	bad := []author.Step{setBool("accepted", false), setText(p+"class", "invalid")}
	verify := func(q string) []author.Step {
		z := w + q + "_"
		v := []author.Step{require(z+"id", author.Var(w+"witness_query"), bad), require(z+"valid", author.Boolean(true), bad), require(z+"status", author.String("found"), bad)}
		for _, k := range []string{"target", "mutation", "total", "adoption"} {
			v = append(v, require(z+k, author.Var(w+"witness_"+k), bad))
		}
		return v
	}
	issued := []author.Step{setText(p+"class", "applied_issued"), require(p+"interaction", author.String("issued"), bad), require(p+"valid", author.Boolean(false), bad), require(p+"source_status", author.String("succeeded"), bad), require(p+"source_reason", author.String("member_succeeded"), bad)}
	direct := []author.Step{setText(p+"class", "applied_direct"), require(p+"interaction", author.String("validated"), bad), require(p+"source_status", author.String("succeeded"), bad), require(p+"source_reason", author.String("member_succeeded"), bad), require(p+"mutation", author.Var(w+"witness_mutation"), bad), require(p+"total", author.Var(w+"witness_total"), bad)}
	failed := []author.Step{setText(p+"class", "applied_after_original_failure"), require(p+"interaction", author.String("transport_error"), bad), require(p+"source_status", author.String("failed"), bad), require(p+"source_reason", author.String("member_failed"), bad), require(p+"mutation", author.Integer(0), bad), require(p+"total", author.Integer(0), bad)}
	present := []author.Step{
		eq(p+"observed", author.Boolean(true), []author.Step{eq(p+"valid", author.Boolean(true), direct, failed)}, issued),
		require(p+"issued", author.Boolean(true), bad), require(p+"issued_target", author.Var(p+"target"), bad),
		require(w+"target", author.Var(p+"target"), bad), require(w+"witness", author.Boolean(true), bad), require(w+"witness_target", author.Var(p+"target"), bad), require(w+"conflict", author.Boolean(false), bad),
		eq(w+"witness_query", author.String(""), bad, nil),
		eq(w+"witness_query", author.Var(w+"query1_id"), verify("query1"), verify("query2")),
		eq(w+"witness_adoption", author.String("adopted"), nil, []author.Step{eq(w+"witness_adoption", author.String("duplicate"), nil, []author.Step{eq(w+"witness_adoption", author.String("existing"), nil, bad)})}),
	}
	return []author.Step{eq(p+"source_present", author.Boolean(true), present, bad)}
}

func treeChecks(paths []string) []author.Step {
	bad := []author.Step{setBool("tree_valid", false), setBool("accepted", false)}
	present := []author.Step{setBool("tree_valid", true), setText("tree_class", "answer")}
	for _, p := range paths {
		tag := "tree_" + p + "classification"
		diagnosed := []author.Step{
			setText("tree_class", "diagnosed"),
			require("tree_seen", author.Boolean(false), bad),
			eq(tag, author.String("candidate_failed"), nil, []author.Step{
				require(tag, author.String("candidate_missing_reply"), bad),
			}),
		}
		present = append(present,
			require("tree_"+p+"id", author.Var(p+"id"), bad),
			setBool("tree_seen", false),
			eq(p+"value", author.Integer(0), []author.Step{author.Set("expected_value", author.Integer(1))}, []author.Step{author.Set("expected_value", author.Integer(0))}),
			author.ForEach(author.Var("tree_"+p+"answers"), map[string]string{"id": "row_id", "value": "row_value"},
				require("tree_seen", author.Boolean(false), bad), setBool("tree_seen", true),
				require("row_id", author.Var(p+"id"), bad), require("row_value", author.Var("expected_value"), bad)),
			eq(tag, author.String("answer"), []author.Step{require("tree_seen", author.Boolean(true), bad)}, diagnosed),
		)
	}
	envelope := func(status, reason string) []author.Step {
		return []author.Step{
			require("tree_status", author.String(status), bad), require("tree_reason", author.String(reason), bad),
		}
	}
	present = append(present, eq("tree_class", author.String("answer"), envelope("succeeded", "member_succeeded"), envelope("failed", "member_failed")),
		eq("tree_valid", author.Boolean(false), []author.Step{setText("tree_class", "invalid")}, nil))
	absent := []author.Step{setBool("accepted", false), setText("tree_class", "invalid")}
	return []author.Step{eq("tree_present", author.Boolean(true), present, absent)}
}
