//go:build integration

package substitutiontest

import (
	"path/filepath"
	"testing"

	join "github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestRetainedBClosedHelperReply(t *testing.T) {
	b := candidate(t, "b")
	cert, err := nominal.CheckTemplate(b)
	must(t, err)
	d, err := join.New(b, substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: limits(4, 4, 3)})
	must(t, err)
	addOperator(t, &d.Provision)
	path := filepath.Join(t.TempDir(), "closed-helper.db")
	store, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, err)
	must(t, store.Close())
	driver := newCommunicationDriver(t, path)
	step := func(run runtime.RunID, n int) {
		t.Helper()
		for range n {
			if !driver.step(run) {
				t.Fatalf("original step in %v not enabled", run)
			}
		}
	}
	admit := func(caller runtime.RunID, register, template string, depth int64) runtime.InvocationAdmission {
		t.Helper()
		op := projection(inspect(t, path), caller.Subject).Data[register].Text
		a, err := driver.store.AdmitInvocation(t.Context(), op)
		must(t, err)
		if !a.Accepted || len(a.Members) != 1 || a.Invocation.TemplateID != template || a.Invocation.OperationID != op || a.Invocation.Caller != caller || a.Invocation.Depth != depth {
			t.Fatal("wrong original admission", a)
		}
		return a
	}
	root := runtime.RunID{Subject: "root", Generation: 1}
	step(root, 1)
	j := admit(root, "join_operation", d.Join.ID(), 1).Members[0]
	step(j, 4)
	e := admit(j, "candidate_operation", nominalB, 2).Members[0]
	admit(j, "s_operation", d.Successor.ID(), 2)
	step(e, 2)
	h := admit(e, "helper_operation", nominalH, 3)
	step(h.Members[0], 4)
	before := inspect(t, path)
	var sent intent
	for _, row := range before.Tables["intents"] {
		in := decode[intent](row, "descriptor")
		if in.Run == h.Members[0] && in.Kind == "message" {
			sent = in
		}
	}
	if sent.ID == "" || sent.Sequence != 4 || sent.Target != e.Subject || sent.Generation != e.Generation || sent.Obligation != protocol.Obligation ||
		sent.Payload["id"] != schema.Int(1) || sent.Payload["value"] != schema.Int(1) || projection(before, h.Members[0].Subject).PC != 6 ||
		communicationRow(before, "attempts", "intent_id", sent.ID) != nil || len(before.Tables["composition_replies"]) != 0 {
		t.Fatal("wrong pre-closure send cut", sent)
	}
	operator := runtime.RunID{Subject: "operator", Generation: 1}
	admitCancel(t, driver.store)
	step(operator, 2)
	request, err := driver.store.LoadLifecycleRequest(t.Context(), projection(inspect(t, path), "operator").Data["pending"].Text)
	must(t, err)
	opAuth, err := driver.store.AuthorizeIntent(t.Context(), request.Intent.ID)
	must(t, err)
	if !opAuth.Allowed {
		t.Fatal("operator authorization", opAuth)
	}
	closed, err := driver.store.DecideLifecycle(t.Context(), request)
	must(t, err)
	if !closed.Accepted || closed.Reason != "closed" || closed.After.Run != root || closed.After.Projection.Status != "cancelled" {
		t.Fatal(closed)
	}
	cut := inspect(t, path)
	if len(cut.Tables["composition_replies"]) != 0 || communicationRow(cut, "attempts", "intent_id", sent.ID) != nil || projection(cut, e.Subject).Status != "open" {
		t.Fatal("expected undecided helper reply with open parent")
	}
	step(h.Members[0], 1)
	auth := driver.authorize(sent.ID)
	if !auth.Allowed || auth.SourceStatus != "succeeded" || auth.Generation != h.Members[0].Generation || auth.Source != h.Members[0].Subject {
		t.Fatal(auth)
	}
	driver.deliver(sent.ID)
	after := inspect(t, path)
	if len(after.Tables["composition_replies"]) != 1 {
		t.Fatal("expected sole helper disposition")
	}
	original := decode[reply](after.Tables["composition_replies"][0], "descriptor")
	if original.InvocationID != h.Invocation.ID || original.Intent.ID != sent.ID || original.Intent.Run != h.Members[0] ||
		!original.Accepted || original.Reason != "recorded_reply" || len(after.Tables["history"]) != 14 || after.Roots[0].Charges.Invocations != 4 ||
		after.Roots[0].Charges.CreatedRuns != 4 || after.Roots[0].Charges.Effects != 0 || projection(after, root.Subject).Status != "cancelled" {
		t.Fatal("bound helper reply after root cancellation", original)
	}
	stablePrefix(t, before, after)
	t.Logf("B=%s h=%s reply=%s after granted root cancel: first allowed=%t actual=%s; K1 closed guard=refused/scope_closed", b.ID(), h.Invocation.TemplateID, sent.ID, auth.Allowed, original.Reason)
	retain(t, "closure", map[string]any{"certificate": cert, "inspection": after, "original_reply": sent.ID, "cancel_operation": request.Intent.ID})
}
