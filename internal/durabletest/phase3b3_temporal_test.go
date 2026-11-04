//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func reciprocalLiteral(v schema.Value) map[string]any { return map[string]any{"Literal": v} }
func reciprocalInput(obligation string) map[string]any {
	return map[string]any{"Op": "receive", "Match": map[string]any{"Kind": "input", "Provenance": "local-input", "Source": reciprocalLiteral(schema.Text("local")), "Generation": reciprocalLiteral(schema.Int(1)), "Obligation": reciprocalLiteral(schema.Text(obligation)), "Operation": reciprocalLiteral(schema.Text(""))}}
}
func reciprocalShift(code []any, start, delta int) {
	for _, v := range code {
		in := v.(map[string]any)
		if in["Op"] == "branch" || in["Op"] == "jump" {
			for _, key := range []string{"Target", "Otherwise"} {
				if n, ok := in[key].(float64); ok && int(n) >= start {
					in[key] = int(n) + delta
				}
			}
		}
	}
}
func reciprocalCommit(t *testing.T, s *runtime.Store, id runtime.RunID, observation string) {
	t.Helper()
	state, err := s.Snapshot(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), state.Authority(), observation); err != nil {
		t.Fatal(err)
	}
}
func reciprocalInputCommit(t *testing.T, s *runtime.Store, id runtime.RunID, obligation string) {
	t.Helper()
	r, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: obligation, LogicalID: obligation, Target: id.Subject, Generation: id.Generation, Obligation: obligation, Payload: map[string]schema.Value{}})
	if err != nil || !r.Accepted {
		t.Fatal(r, err)
	}
	reciprocalCommit(t, s, id, r.ObservationID)
}
func reciprocalRevokeProgram(t *testing.T, p *runtime.Provision, m *delegation.Manifest) {
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: "b", Operation: "input", Obligation: "revoke-trigger"})
	claim, _ := json.Marshal(runtime.RevocationClaim{Version: runtime.RequestAuthorityVersion, Root: m.BAServiceRoot})
	reciprocalEdit(t, p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		reciprocalShift(code, 5, 3)
		emit := map[string]any{"Op": "emit", "Dst": "revoke", "Effect": map[string]any{"Kind": "lifecycle", "Target": reciprocalLiteral(schema.Text("b")), "Generation": reciprocalLiteral(schema.Int(1)), "Obligation": reciprocalLiteral(schema.Text("revocation-result")), "Payload": map[string]any{"operation": reciprocalLiteral(schema.Text("RevokeAuthority")), "revision": reciprocalLiteral(schema.Int(1)), "claim": reciprocalLiteral(schema.Text(string(claim)))}}}
		receive := map[string]any{"Op": "receive", "Match": map[string]any{"Kind": "lifecycle", "Provenance": "runtime-lifecycle", "Source": reciprocalLiteral(schema.Text("b")), "Generation": reciprocalLiteral(schema.Int(1)), "Obligation": reciprocalLiteral(schema.Text("revocation-result")), "Operation": map[string]any{"Ref": "revoke"}}, "Bind": map[string]string{"accepted": "revoked"}}
		next := append([]any{}, code[:5]...)
		next = append(next, reciprocalInput("revoke-trigger"), emit, receive)
		program["Code"] = append(next, code[5:]...)
	})
	m.ABService = `["b",1,12,0]`
	m.ABReply = `["b",1,15,0]`
}
func reciprocalRevoke(t *testing.T, s *runtime.Store, m delegation.Manifest, dir string) {
	t.Helper()
	reciprocalInputCommit(t, s, m.B, "revoke-trigger")
	reciprocalCommit(t, s, m.B, "")
	id := `["b",1,7,0]`
	auth, err := s.AuthorizeIntent(t.Context(), id)
	if err != nil || !auth.Allowed {
		t.Fatal(auth, err)
	}
	r, err := s.LoadLifecycleRequest(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := s.DecideLifecycle(t.Context(), r)
	if err != nil || !decision.Accepted {
		t.Fatal(decision, err)
	}
	admission, err := s.SettleLifecycle(t.Context(), id, "revocation-delivery")
	if err != nil || !admission.Accepted {
		t.Fatal(admission, err)
	}
	reciprocalCommit(t, s, m.B, admission.ObservationID)
	baseline3b3Write(t, dir, "controlled-revocation.json", map[string]any{"Authorization": auth, "Decision": decision, "Admission": admission})
}
func TestPhase3B3ReciprocalRevocation(t *testing.T) {
	for _, cut := range []struct{ name, boundary, phase string }{{"before-gate", "dispatch.consider", "pre-admission"}, {"after-gate", "authority.admission", "allowed"}, {"after-result", "external.result-recorded", "pre-settlement"}} {
		t.Run(cut.name, func(t *testing.T) {
			dir := reciprocalDirectory(t, "revoke-"+cut.name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
			if err != nil {
				t.Fatal(err)
			}
			reciprocalRevokeProgram(t, &p, &m)
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			did := false
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
				if e.Boundary == cut.boundary && e.Phase == cut.phase && e.Intent == m.BAService && !did {
					did = true
					baseline3b3Write(t, dir, "temporal-boundary.json", e)
					reciprocalRevoke(t, s, m, dir)
				}
				return nil
			})
			report, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			s.Close()
			baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
			if runErr != nil || !did {
				t.Fatal("controlled schedule", runErr, did)
			}
			endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
			phaseStopEndpoint(t, ca)
			phaseStopEndpoint(t, cb)
			baseline3b3Write(t, dir, "endpoints.json", endpoints)
			reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
			reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
			snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
			e, ok := snap.Effect(m.BAService)
			if !ok || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 1 {
				t.Fatal("lost original allow", e)
			}
			classification := "safe-complete"
			if cut.name == "before-gate" {
				classification = "permission-blocked"
				if e.AuthorityAdmissions[0].Allowed || e.ExternalReceipt != nil || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal("revoked gate contacted", e, endpoints)
				}
			} else {
				if !e.AuthorityAdmissions[0].Allowed {
					t.Fatal("historical allow lost")
				}
				reciprocalComplete(t, snap, m, endpoints)
			}
			s, err = runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			reciprocalCapture(t, db, filepath.Join(dir, "reopened"))
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "revoke-" + cut.name, Schedule: "controlled-retained-protocol", Directory: dir, Classification: classification, Manifest: m})
			baseline3b3Manifest(t, dir)
		})
	}
}
func TestPhase3B3ReciprocalLateGrant(t *testing.T) {
	dir := reciprocalDirectory(t, "late-grant")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	reciprocalEdit(t, &p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		reciprocalShift(code, 1, 1)
		next := []any{code[0], reciprocalInput("issue-grant")}
		program["Code"] = append(next, code[1:]...)
	})
	reciprocalEdit(t, &p, "a", func(program map[string]any) {
		code := program["Code"].([]any)
		code[1] = reciprocalInput("attempt-without-grant")
		code[10] = reciprocalInput("hold-refused-original")
	})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: "a", Operation: "input", Obligation: "attempt-without-grant"}, runtime.Permission{Kind: "input", Source: "local", Target: "b", Operation: "input", Obligation: "issue-grant"})
	m.Grant = `["b",1,3,0]`
	m.Ready = `["b",1,6,0]`
	m.ABService = `["b",1,10,0]`
	m.ABReply = `["b",1,13,0]`
	reciprocalUse(t, &p, func(u *runtime.AuthorityUse) { u.GrantID = m.Grant })
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.AdmitInput(t.Context(), runtime.Input{TransportID: "attempt", LogicalID: "attempt", Target: "a", Generation: 1, Obligation: "attempt-without-grant", Payload: map[string]schema.Value{}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2})
	if err != nil {
		t.Fatal(err)
	}
	baseline3b3Write(t, dir, "before-grant-run.json", report)
	s.Close()
	before := reciprocalCapture(t, db, filepath.Join(dir, "before-grant"))
	e, ok := before.Effect(m.BAService)
	if !ok || e.ExternalAttempt == nil || e.ExternalAttempt.Allowed {
		t.Fatal("absent grant did not refuse", e)
	}
	raw, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil || len(raw.Tables["lifecycle_results"]) != 0 {
		t.Fatal("grant already existed", err)
	}
	s, err = runtime.Open(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AdmitInput(t.Context(), runtime.Input{TransportID: "issue", LogicalID: "issue", Target: "b", Generation: 1, Obligation: "issue-grant", Payload: map[string]schema.Value{}})
	if err != nil {
		t.Fatal(err)
	}
	report, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2})
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	baseline3b3Write(t, dir, "after-grant-run.json", report)
	after := reciprocalCapture(t, db, filepath.Join(dir, "after-grant"))
	later, _ := after.Effect(m.BAService)
	if !reflect.DeepEqual(e.ExternalAttempt, later.ExternalAttempt) || len(later.AuthorityAdmissions) != 0 {
		t.Fatal("late grant repaired original")
	}
	raw, err = runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil || len(raw.Tables["lifecycle_results"]) != 1 {
		t.Fatal("late grant missing", err)
	}
	var result runtime.LifecycleResult
	if err = json.Unmarshal([]byte(raw.Tables["lifecycle_results"][0]["result"].(string)), &result); err != nil || !result.Accepted {
		t.Fatal("late grant not accepted", result, err)
	}
	endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", endpoints)
	if endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
		t.Fatal("late grant contacted")
	}
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "first-refused-then-grant", Schedule: "controlled-local-inputs-automatic-run", Directory: dir, Classification: "safe-refusal", Manifest: m})
	baseline3b3Manifest(t, dir)
}

func TestPhase3B3AppliedUnknownRevoked(t *testing.T) {
	dir := reciprocalDirectory(t, "applied-unknown-revoked")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", &barrier{Boundary: "resource.applied", Phase: "post-commit", Subject: "a", Sequence: 4})
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	reciprocalRevokeProgram(t, &p, &m)
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	first := reciprocalOwner(t, db, filepath.Join(dir, "first-owner"), &barrier{Boundary: "runner.waiting", Phase: "in-flight"})
	endpointAck := reciprocalAwait(t, cb)
	ack := reciprocalAwait(t, first)
	baseline3b3Write(t, dir, "acknowledgments.json", map[string]any{"Runtime": ack, "Endpoint": endpointAck})
	first.kill(t)
	baseline3b3Write(t, dir, "kill.json", map[string]any{"PID": first.cmd.Process.Pid, "Wait": first.cmd.ProcessState.String()})
	stopped := reciprocalCapture(t, db, filepath.Join(dir, "stopped"))
	before, _ := stopped.Effect(m.BAService)
	if before == nil || before.ExternalReceipt != nil || len(before.AuthorityAdmissions) != 1 {
		t.Fatal("unknown prefix", before)
	}
	if _, err = cb.release.Write([]byte("c")); err != nil {
		t.Fatal(err)
	}
	s, err = runtime.Open(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	backup := filepath.Join(t.TempDir(), "opened.sqlite")
	reciprocalBackup(t, db, backup)
	reciprocalCapture(t, backup, filepath.Join(dir, "reopened"))
	reciprocalRevoke(t, s, m, dir)
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	s.Close()
	baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(err)})
	if len(report.UnknownExternal) != 1 || report.UnknownExternal[0] != m.BAService {
		t.Fatal("unknown original missing from report", report)
	}
	if err != nil {
		t.Fatal(err)
	}
	endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", endpoints)
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
	e, _ := snap.Effect(m.BAService)
	if e == nil || !reflect.DeepEqual(before.ExternalAttempt, e.ExternalAttempt) || e.ExternalReceipt != nil || e.Settlement != nil || len(e.AuthorityAdmissions) != 2 || e.AuthorityAdmissions[1].Allowed || e.AuthorityAdmissions[1].Source.Epoch != before.AuthorityAdmissions[0].Source.Epoch+1 || endpoints["b"].Contacts != 1 || len(endpoints["b"].Applications) != 1 || endpoints["a"].Contacts != 0 {
		t.Fatal("revoked unknown re-exposed or fabricated", e, endpoints)
	}
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "applied-unknown-revoked-before-fresh-gate", Schedule: "process-cut-then-controlled-retained-protocol", Directory: dir, Classification: "permission-blocked-transport-unknown", Manifest: m, Cut: &ack})
	baseline3b3Manifest(t, dir)
}
