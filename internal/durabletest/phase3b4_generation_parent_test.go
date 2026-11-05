//go:build integration

package durabletest

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// M03 needs a real target-generation-two intent, not a modified claim or an
// invented generation-two run. This public prefix deliberately stops unpaid.
func TestPhase3B4CampaignAuxiliaryTargetGeneration(t *testing.T) {
	const name = "M03-auxiliary-target-generation-two"
	const q2 = `["b",1,6,0]`
	dir := bindingDirectory(t, name)
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m := bindingFixture(t, ua, ub, 0, nil, false)
	bindingQ2(t, &p, &m, true)
	reciprocalEdit(t, &p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		q2 := code[5].(map[string]any)["Effect"].(map[string]any)
		q2["Generation"] = reciprocalLiteral(schema.Int(2))
	})
	baseline3b3Write(t, dir, "provision.json", p)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	bindingInitial(t, s, m.Reciprocal)
	bindingGrant(t, s, m.Reciprocal)
	reciprocalCommit(t, s, m.Reciprocal.B, "")
	before := bindingCaptureLive(t, db, filepath.Join(dir, "emitted-before-authorization"))
	auth, err := s.AuthorizeIntent(t.Context(), q2)
	if err != nil || !auth.Allowed || auth.Reason != "authorized" || auth.IntentID != q2 || auth.Source.Run != m.Reciprocal.B {
		t.Fatal("auxiliary source authorization", auth, err)
	}
	const transport = "M03 auxiliary target generation two"
	result, err := s.DeliverMessage(t.Context(), q2, transport)
	if err != nil || result.Accepted || result.Reason != "target_generation_changed" || result.ObservationID != "" || result.Bound == nil || *result.Bound != m.Reciprocal.A {
		t.Fatal("auxiliary generation refusal", result, err)
	}
	baseline3b3Write(t, dir, "auxiliary-delivery.json", map[string]any{"Original": q2, "SourceAuthorization": auth, "AdmissionResult": result, "OutsideRestrictedC": true, "ClaimedCompletion": false})
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	snap := bindingCapture(t, db, filepath.Join(dir, "final"))
	intent, ok := snap.Operation(q2)
	original, originalOK := snap.Operation(m.Reciprocal.BARequest)
	if !ok || !originalOK || intent.Kind != "message" || intent.Run != m.Reciprocal.B || intent.Target != "a" || intent.Generation != 2 || intent.Obligation != original.Obligation || !reflect.DeepEqual(intent.Payload, original.Payload) {
		t.Fatal("exact genuine auxiliary original", intent, original)
	}
	step, ok := snap.Reduction(runtime.ReductionRef{Run: m.Reciprocal.B, Sequence: 6})
	if !ok || step.Instruction != "emit" || step.BeforePC != 5 || len(step.OperationIDs) != 1 || step.OperationIDs[0] != q2 {
		t.Fatal("auxiliary was not emitted by ordinary B6", step)
	}
	matched := 0
	for _, a := range snap.Admissions {
		if a.SourceOperationID != q2 {
			continue
		}
		matched++
		if a.Domain != "message" || a.LogicalID != q2 || a.Source != m.Reciprocal.B || a.Target != "a" || a.RequestedGeneration != 2 || !reflect.DeepEqual(a.AdmissionResult, result) {
			t.Fatal("retained auxiliary admission", a)
		}
	}
	if matched != 1 {
		t.Fatal("auxiliary admission count", matched)
	}
	for _, o := range snap.Observations {
		if o.SourceOperationID == q2 {
			t.Fatal("refused auxiliary acquired an observation", o)
		}
	}
	if len(snap.Runs) != 2 || len(snap.Composition.Roots) != 2 || !reflect.DeepEqual(snap.Composition.Roots, before.Composition.Roots) || !reflect.DeepEqual(snap.Deployment.Subjects, p.Subjects) {
		t.Fatal("principal/root set changed", snap.Runs, snap.Composition.Roots, snap.Deployment)
	}
	for _, r := range snap.Runs {
		if r.ID != m.Reciprocal.A && r.ID != m.Reciprocal.B || r.Projection.Status != "open" || r.Projection.Data["paid"].Flag {
			t.Fatal("unexpected run or completion", r)
		}
	}
	for _, id := range []string{m.Reciprocal.ABService, m.Reciprocal.BAService} {
		if _, exists := snap.Operation(id); exists {
			t.Fatal("prefix unexpectedly emitted service", id)
		}
	}
	endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
	for _, endpoint := range endpoints {
		if endpoint.Contacts != 0 || len(endpoint.Applications) != 0 {
			t.Fatal("prefix exposed an external service", endpoint)
		}
	}
	bindingIndex(t, dir, bindingCase{Name: name, Schedule: "controlled-public-commit-authorize-deliver-prefix", Classification: "source-authorized-target-generation-refusal-outside-C", Manifest: m, AuxiliaryRequests: []string{q2}, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "emitted-before-authorization", Directory: "emitted-before-authorization"}, {Ordinal: 2, Predecessor: "emitted-before-authorization", Label: "final", Directory: "final"}}}, p)
}
