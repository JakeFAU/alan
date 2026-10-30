package callablejoin

import (
	"fmt"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/schema"
)

// NewControl constructs a separately named, deliberately nonconforming J.
// These executable research controls do not change the positive constructor.
func NewControl(kind string, candidate, successor author.Template) (author.Artifact, author.Template, error) {
	if kind != "no-report-success" && kind != "no-report-failure" && kind != "false-candidate-failed" {
		return author.Artifact{}, author.Template{}, fmt.Errorf("unknown callable join control %q", kind)
	}
	steps := []author.Step{
		author.Receive(PublicPort, map[string]string{"id": "request_id", "value": "request_value", "mode": "request_mode"}),
		author.Invoke("candidate_operation", "candidate", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value"), "mode": author.Var("request_mode")}, nil),
		author.Invoke("s_operation", "successor", map[string]author.Expr{"id": author.Var("request_id"), "value": author.Var("request_value")}, nil),
		author.AwaitInvocation("successor", "s_operation", author.InvocationOutcomeBindings{Status: "s_status", Reason: "s_reason", ResponsePresent: "s_present", Response: map[string]string{"id": "s_id", "value": "s_value"}}),
		author.AwaitInvocation("candidate", "candidate_operation", author.InvocationOutcomeBindings{Status: "candidate_status", Reason: "candidate_reason", ResponsePresent: "candidate_present", Response: map[string]string{"id": "candidate_id", "value": "candidate_value"}}),
	}
	if kind == "false-candidate-failed" {
		// The lie is reachable only after real missing_reply evidence and valid S.
		policy := []author.Step{author.Set("classification", author.String("candidate_failed"))}
		guards := [][2]author.Expr{
			{author.Var("s_status"), author.String("succeeded")},
			{author.Var("s_reason"), author.String("member_succeeded")},
			{author.Var("s_present"), author.Boolean(true)},
			{author.Var("s_id"), author.Var("request_id")},
			{author.Var("s_value"), author.Integer(0)},
			{author.Var("candidate_status"), author.String("failed")},
			{author.Var("candidate_reason"), author.String("missing_reply")},
			{author.Var("candidate_present"), author.Boolean(false)},
		}
		for i := len(guards) - 1; i >= 0; i-- {
			policy = []author.Step{author.IfEqual(guards[i][0], guards[i][1], policy, []author.Step{author.Set("classification", author.String("invalid_outcome"))})}
		}
		steps = append(steps, policy...)
		steps = append(steps, author.Send(PublicPort, map[string]author.Expr{"id": author.Var("request_id"), "classification": author.Var("classification"), "answers": author.Var("answers")}), author.Failure())
	} else if kind == "no-report-success" {
		steps = append(steps, author.Success())
	} else {
		steps = append(steps, author.Failure())
	}
	artifact, err := author.Compile(author.Definition{
		Name:        "phase1n-control-" + kind,
		Invocations: map[string]author.Template{"candidate": candidate, "successor": successor},
		Collections: map[string]schema.Field{"answers": Report()["answers"]},
		Ports:       map[string]author.Port{PublicPort: {Peer: "caller", Obligation: "work", Receive: protocol.Request(), Send: Report()}},
		Steps:       steps,
	})
	if err != nil {
		return artifact, author.Template{}, err
	}
	template, err := author.NewTemplate(author.Application{"entry": {Artifact: artifact, Inputs: map[string]schema.Value{}}}, "entry", PublicPort)
	return artifact, template, err
}
