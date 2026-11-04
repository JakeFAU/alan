//go:build integration

package durabletest

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This is an explicitly larger public parent outside the frozen two-service C
// domain. Equal Total values do not make the distinct receipt lineages equal.
func TestPhase3B4CampaignAuxiliaryServicedParent(t *testing.T) {
	dir := bindingDirectory(t, "auxiliary-serviced-equal-total")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m := bindingFixture(t, ua, ub, 0, nil, false)
	bindingQ2(t, &p, &m, true)
	p.Composition.Roots[0].Limits.MaxEffects = 2
	q2 := `["b",1,6,0]`
	g2 := `["b",1,7,0]`
	ready2 := `["b",1,10,0]`
	e2 := `["a",1,13,0]`
	reply2 := `["a",1,17,0]`
	claim := m.Reciprocal.Claim
	claim.OriginalRequestID = q2
	claim.ServiceOperationID = e2
	claimRaw, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	clone := func(v any) map[string]any {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err = json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	reciprocalEdit(t, &p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		reciprocalShift(code, 6, 4)
		grant := clone(code[1])
		grant["Dst"] = "grant2"
		grant["Effect"].(map[string]any)["Payload"].(map[string]any)["claim"] = reciprocalLiteral(schema.Text(string(claimRaw)))
		result := clone(code[2])
		result["Match"].(map[string]any)["Operation"] = map[string]any{"Ref": "grant2"}
		result["Bind"] = map[string]string{"accepted": "grant2_accepted"}
		branch := map[string]any{"Op": "branch", "A": map[string]any{"Ref": "grant2_accepted"}, "B": reciprocalLiteral(schema.Bool(true)), "Target": 9, "Otherwise": len(code) + 3}
		ready := clone(code[4])
		ready["Dst"] = "ready2"
		ready["Effect"].(map[string]any)["Obligation"] = reciprocalLiteral(schema.Text("aux-grant-ready"))
		ready["Effect"].(map[string]any)["Payload"].(map[string]any)["grant"] = map[string]any{"Ref": "grant2"}
		program["Code"] = append(append(append([]any{}, code[:6]...), grant, result, branch, ready), code[6:]...)
	})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "message", Source: "b", Target: "a", Operation: "message", Revision: 1, Obligation: "aux-grant-ready"})
	reciprocalEdit(t, &p, "a", func(program map[string]any) {
		code := program["Code"].([]any)
		reciprocalShift(code, 7, 9)
		code[6].(map[string]any)["Target"] = 7
		var inserted []any
		for _, key := range []string{"service_operation", "mutation", "total", "service_accepted"} {
			inserted = append(inserted, map[string]any{"Op": "set", "Dst": "q_saved_" + key, "A": map[string]any{"Ref": key}})
		}
		ready := clone(code[2])
		ready["Match"].(map[string]any)["Obligation"] = reciprocalLiteral(schema.Text("aux-grant-ready"))
		ready["Bind"] = map[string]string{"grant": "grant2_data"}
		service := clone(code[4])
		service["Effect"].(map[string]any)["Authority"] = runtime.AuthorityUse{Version: runtime.RequestAuthorityBindingVersion, Path: "delegated", Root: claim.Root, GrantID: g2}
		service["Effect"].(map[string]any)["Payload"].(map[string]any)["delta"] = reciprocalLiteral(schema.Int(0))
		result := clone(code[5])
		branch := map[string]any{"Op": "branch", "A": map[string]any{"Ref": "service_accepted"}, "B": reciprocalLiteral(schema.Bool(true)), "Target": 15, "Otherwise": len(code) + 8}
		inserted = append(inserted, ready, service, result, branch)
		qReply := clone(code[7])
		payload := qReply["Effect"].(map[string]any)["Payload"].(map[string]any)
		for _, key := range []string{"service_operation", "mutation", "total"} {
			payload[key] = map[string]any{"Ref": "q_saved_" + key}
		}
		payload["accepted"] = map[string]any{"Ref": "q_saved_service_accepted"}
		auxReply := clone(code[7])
		auxReply["Dst"] = "aux_reply"
		auxReply["Effect"].(map[string]any)["Payload"].(map[string]any)["request_id"] = reciprocalLiteral(schema.Text(q2))
		next := append(append([]any{}, code[:7]...), inserted...)
		next = append(next, qReply, auxReply)
		program["Code"] = append(next, code[8:]...)
	})
	m.Reciprocal.BAReply = `["a",1,16,0]`
	m.ReplySequence = 16
	m.Reciprocal.ABService = `["b",1,14,0]`
	m.Reciprocal.ABReply = `["b",1,17,0]`
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	report, err := bindingRun(t, s, dir, 256)
	s.Close()
	if err != nil {
		t.Fatal(err)
	}
	endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
	snap := bindingCapture(t, db, filepath.Join(dir, "final"))
	first, _ := snap.Effect(m.Reciprocal.BAService)
	second, _ := snap.Effect(e2)
	if first == nil || second == nil || first.ExternalReceipt == nil || second.ExternalReceipt == nil || !second.ExternalAttempt.Allowed || second.ExternalAttempt.Authority.OriginalRequestID != q2 || second.ExternalAttempt.Authority.RequestConsumedSequence != 4 || second.ExternalAttempt.Authority.Use.GrantID != g2 {
		t.Fatal(first, second)
	}
	if first.ExternalReceipt.Total != 1 || second.ExternalReceipt.Total != 1 || first.ExternalReceipt.Mutation != 1 || second.ExternalReceipt.Mutation != 2 || second.ExternalAttempt.Request.Delta != 0 || endpoints["b"].Contacts != 2 || len(endpoints["b"].Applications) != 2 {
		t.Fatal("equal Total/distinct real lineage", first, second, endpoints)
	}
	for _, x := range []struct {
		reply, request, service string
		mutation                int64
	}{{m.Reciprocal.BAReply, m.Reciprocal.BARequest, m.Reciprocal.BAService, 1}, {reply2, q2, e2, 2}} {
		reply, ok := snap.Operation(x.reply)
		if !ok || reply.Payload["request_id"].Text != x.request || reply.Payload["service_operation"].Text != x.service || reply.Payload["mutation"].Number != x.mutation || reply.Payload["total"].Number != 1 {
			t.Fatal("authentic reply lineage", reply)
		}
	}

	auxiliaryResult, ok := snap.Observation(second.Settlement.ObservationID)
	if !ok || auxiliaryResult.SelectedAt == nil || auxiliaryResult.SelectedAt.Sequence != 14 {
		t.Fatal("auxiliary result not genuinely selected", auxiliaryResult)
	}
	reciprocalSelectedHistory(t, snap, auxiliaryResult)
	auxiliaryReply := bindingSelected(t, snap, m.Reciprocal.B, reply2)
	if auxiliaryReply.SelectedAt != nil {
		t.Fatal("unexpected auxiliary payment", auxiliaryReply)
	}
	ab, ok := snap.Effect(m.Reciprocal.ABService)
	if !ok || ab.ExternalReceipt == nil || ab.Settlement == nil {
		t.Fatal("principal AB service", ab)
	}
	abResult, ok := snap.Observation(ab.Settlement.ObservationID)
	if !ok || abResult.SelectedAt == nil {
		t.Fatal("principal AB result", abResult)
	}
	reciprocalSelectedHistory(t, snap, abResult)
	abReply := bindingSelected(t, snap, m.Reciprocal.A, m.Reciprocal.ABReply)
	if abReply.SelectedAt == nil || abReply.Payload["request_id"].Text != m.Reciprocal.ABRequest || abReply.Payload["service_operation"].Text != m.Reciprocal.ABService || abReply.Payload["mutation"] != abResult.Payload["mutation"] || abReply.Payload["total"] != abResult.Payload["total"] {
		t.Fatal("principal AB reply custody", abReply)
	}
	reciprocalSelectedHistory(t, snap, abReply)
	for _, id := range []runtime.RunID{m.Reciprocal.A, m.Reciprocal.B} {
		run, _ := snap.Run(id)
		if run.Projection.Status != "succeeded" || !run.Projection.Data["paid"].Flag {
			t.Fatal("principal unpaid", run)
		}
	}
	originalReply := bindingSelected(t, snap, m.Reciprocal.B, m.Reciprocal.BAReply)
	if originalReply.SelectedAt == nil {
		t.Fatal("principal q reply unconsumed")
	}
	reciprocalSelectedHistory(t, snap, originalReply)
	if !report.Quiescent || len(report.OpenRuns) != 0 || len(snap.Pending) != 0 {
		t.Fatal(report, snap.Pending)
	}
	baseline3b3Write(t, dir, "auxiliary-lineage.json", map[string]any{"Original": q2, "Grant": g2, "Ready": ready2, "Service": e2, "Reply": reply2, "Claim": claim, "EqualField": "Total", "EqualValue": 1, "DistinctMutation": []int{1, 2}, "OutsideRestrictedC": true, "MaxEffectsA": 2, "AuxiliaryReplyConsumed": false})
	bindingIndex(t, dir, bindingCase{Name: "auxiliary-serviced-equal-total", Schedule: "automatic-run-extra-genuine-grant-service", Classification: "genuine-q2-authority-equal-total-distinct-lineage", Manifest: m, AuxiliaryRequests: []string{q2}, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}}}, p)
}
