//go:build integration

package durabletest

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestPhase3B4CampaignActualResultLaundering(t *testing.T) {
	dir := bindingDirectory(t, "actual-result-laundering")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	m.BAService = `["a",1,5,0]`
	m.BAReply = `["a",1,8,0]`
	m.ABService = `["b",1,7,0]`
	m.ABReply = `["b",1,10,0]`
	m.Claim.ServiceOperationID = m.BAService
	reciprocalUse(t, &p, func(u *runtime.AuthorityUse) { u.GrantID = m.ABService })
	reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { *c = m.Claim })
	reciprocalEdit(t, &p, "a", func(program map[string]any) {
		old := program["Code"].([]any)
		var code []any
		for _, n := range []int{0, 1, 7, 2, 3, 4, 5, 6, 8, 9, 10} {
			code = append(code, old[n])
		}
		code[6].(map[string]any)["Target"] = 7
		program["Code"] = code
	})
	reciprocalEdit(t, &p, "b", func(program map[string]any) {
		old := program["Code"].([]any)
		var code []any
		for _, n := range []int{0, 1, 2, 3, 4, 7, 8, 9, 10, 11, 5, 6, 12, 13, 14} {
			code = append(code, old[n])
		}
		code[8].(map[string]any)["Target"] = 9
		code[11].(map[string]any)["Target"] = 12
		program["Code"] = code
	})
	bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
	m.Claim.Version = runtime.RequestAuthorityBindingVersion
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	s.Close()
	baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(err)})
	if err != nil {
		t.Fatal(err)
	}
	endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", endpoints)
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	snap := bindingCapture(t, db, filepath.Join(dir, "final"))
	attempt, _ := snap.Effect(m.BAService)
	prior, _ := snap.Effect(m.ABService)
	if attempt == nil || prior == nil || prior.ExternalReceipt == nil || prior.Settlement == nil || attempt.ExternalAttempt == nil || attempt.ExternalAttempt.Allowed || attempt.ExternalAttempt.Authority.Use.GrantID != m.ABService || len(attempt.AuthorityAdmissions) != 0 || endpoints["a"].Contacts != 1 || endpoints["b"].Contacts != 0 {
		t.Fatal("actual prior result laundering", prior, attempt, endpoints)
	}
	result, ok := snap.Observation(prior.Settlement.ObservationID)
	if !ok || result.Run != m.B || result.Operation != m.ABService {
		t.Fatal("prior original result missing")
	}
	reciprocalSelectedHistory(t, snap, result)
	found := false
	for _, o := range snap.Observations {
		if o.SourceOperationID == m.ABReply && o.Run == m.A {
			if o.SelectedAt == nil || o.SelectedAt.Sequence != 3 || o.Payload["service_operation"].Text != m.ABService || o.Payload["mutation"].Number != prior.ExternalReceipt.Mutation || o.Payload["total"].Number != prior.ExternalReceipt.Total {
				t.Fatal("prior actual B result not selected", o)
			}
			reciprocalSelectedHistory(t, snap, o)
			found = true
		}
	}
	if !found {
		t.Fatal("no prior authentic result")
	}
	bindingAncillaryIndex(t, p, reciprocalCase{Version: "alan/phase3b4-ancillary/v1", Name: "actual-prior-service-result-as-grant", Schedule: "automatic-run-reversed-service-policy", Directory: dir, Classification: "safe-refusal", Manifest: m})
	baseline3b3Manifest(t, dir)
}

func TestPhase3B4CampaignGenericUnknown(t *testing.T) {
	dir := bindingDirectory(t, "generic-unknown")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", &barrier{Boundary: "resource.applied", Phase: "post-commit"})
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m := reciprocalAdapterProvision(t, ua, ub, reciprocalR1, "add-one")
	config := []reciprocalAdapter{{ua, reciprocalR1}, {ub, reciprocalR2}}
	baseline3b3Write(t, dir, "deployment-adapters.json", config)
	bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
	m.Claim.Version = runtime.RequestAuthorityBindingVersion
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	bindingCapture(t, db, filepath.Join(dir, "initial"))
	owner := func(out string, filter *barrier) *child {
		path := filepath.Join(dir, out)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		return phaseTestProcess(t, db, "", "", filter, "TestPhase3B4Owner", func(cmd *exec.Cmd) {
			raw, _ := json.Marshal(config)
			cmd.Env = append(cmd.Env, "ALAN_PHASE3B4_CHILD=1", "ALAN_PHASE3B4_OUTPUT="+path, "ALAN_PHASE3B4_ADAPTERS="+string(raw))
		})
	}
	first := owner("first-owner", &barrier{Boundary: "runner.waiting", Phase: "in-flight"})
	endpointAck := reciprocalAwait(t, ca)
	ack := reciprocalAwait(t, first)
	baseline3b3Write(t, dir, "acknowledgments.json", map[string]any{"Runtime": ack, "Endpoint": endpointAck})
	first.kill(t)
	baseline3b3Write(t, dir, "kill.json", map[string]any{"PID": first.cmd.Process.Pid, "Wait": first.cmd.ProcessState.String()})
	stopped := bindingCapture(t, db, filepath.Join(dir, "stopped"))
	before, _ := stopped.Effect(m.BAService)
	if before == nil || before.AdapterAttempt == nil || !before.AdapterAttempt.Allowed || before.AdapterResult != nil || before.Settlement != nil || len(before.AuthorityAdmissions) != 1 {
		t.Fatal("not admitted unknown", before)
	}
	if _, err = ca.release.Write([]byte("c")); err != nil {
		t.Fatal(err)
	}
	next := owner("fresh-owner", nil)
	phaseFinish(t, next)
	rawReport, err := os.ReadFile(filepath.Join(dir, "fresh-owner", "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reopened struct {
		Report runtime.RunReport
		Error  string
	}
	if err = json.Unmarshal(rawReport, &reopened); err != nil {
		t.Fatal(err)
	}
	if reopened.Error != "<nil>" || len(reopened.Report.UnknownAdapters) != 1 || reopened.Report.UnknownAdapters[0] != m.BAService || len(reopened.Report.UnknownExternal) != 0 || reopened.Report.Dispatches != 0 {
		t.Fatal("unknown report lost original", reopened)
	}

	endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", endpoints)
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	final := bindingCapture(t, db, filepath.Join(dir, "final"))
	e, _ := final.Effect(m.BAService)
	if e == nil || e.AdapterResult != nil || e.Settlement != nil || len(e.AuthorityAdmissions) != 1 || endpoints["a"].Contacts != 1 || len(endpoints["a"].Applications) != 1 || endpoints["b"].Contacts != 0 {
		t.Fatal("generic unknown retried or invented result", e, endpoints)
	}
	a, _ := final.Run(m.A)
	if a.Projection.Status != "open" || a.Sequence != 3 {
		t.Fatal("unknown did not remain pending", a)
	}
	bindingAncillaryIndex(t, p, reciprocalCase{Version: "alan/phase3b4-ancillary/v1", Name: "generic-unknown", Schedule: "automatic-run", Directory: dir, Classification: "transport-unknown", Manifest: m, Cut: &ack})
	baseline3b3Manifest(t, dir)
}

func TestPhase3B4CampaignAppliedUnknownRevoked(t *testing.T) {
	dir := bindingDirectory(t, "applied-unknown-revoked")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", &barrier{Boundary: "resource.applied", Phase: "post-commit", Subject: "a", Sequence: 4})
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	reciprocalRevokeProgram(t, &p, &m)
	bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
	m.Claim.Version = runtime.RequestAuthorityBindingVersion
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	bindingCapture(t, db, filepath.Join(dir, "initial"))
	first := bindingOwner(t, db, filepath.Join(dir, "first-owner"), &barrier{Boundary: "runner.waiting", Phase: "in-flight"}, "", false)
	endpointAck := reciprocalAwait(t, cb)
	ack := reciprocalAwait(t, first)
	baseline3b3Write(t, dir, "acknowledgments.json", map[string]any{"Runtime": ack, "Endpoint": endpointAck})
	first.kill(t)
	baseline3b3Write(t, dir, "kill.json", map[string]any{"PID": first.cmd.Process.Pid, "Wait": first.cmd.ProcessState.String()})
	stopped := bindingCapture(t, db, filepath.Join(dir, "stopped"))
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
	bindingCapture(t, backup, filepath.Join(dir, "reopened"))
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
	snap := bindingCapture(t, db, filepath.Join(dir, "final"))
	e, _ := snap.Effect(m.BAService)
	if e == nil || !reflect.DeepEqual(before.ExternalAttempt, e.ExternalAttempt) || e.ExternalReceipt != nil || e.Settlement != nil || len(e.AuthorityAdmissions) != 2 || e.AuthorityAdmissions[1].Allowed || e.AuthorityAdmissions[1].Source.Epoch != before.AuthorityAdmissions[0].Source.Epoch+1 || endpoints["b"].Contacts != 1 || len(endpoints["b"].Applications) != 1 || endpoints["a"].Contacts != 0 {
		t.Fatal("revoked unknown re-exposed or fabricated", e, endpoints)
	}
	bindingAncillaryIndex(t, p, reciprocalCase{Version: "alan/phase3b4-ancillary/v1", Name: "applied-unknown-revoked-before-fresh-gate", Schedule: "process-cut-then-controlled-retained-protocol", Directory: dir, Classification: "permission-blocked-transport-unknown", Manifest: m, Cut: &ack})
	baseline3b3Manifest(t, dir)
}

func bindingAncillaryIndex(t *testing.T, p runtime.Provision, c reciprocalCase) {
	m := delegation.RequestBindingManifest{Version: "alan/phase3b4-ancillary/v1", Reciprocal: c.Manifest, PrincipalRequests: [2]string{c.Manifest.ABRequest, c.Manifest.BARequest}}
	names := []string{"initial", "first-owner/post-open", "stopped", "fresh-owner/post-open", "reopened", "final"}
	var captures []bindingCaptureRef
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(c.Directory, name, "runtime.sqlite")); err == nil {
			prev := ""
			if len(captures) > 0 {
				prev = captures[len(captures)-1].Directory
			}
			ref := bindingCaptureRef{Ordinal: len(captures) + 1, Predecessor: prev, Label: name, Directory: name}
			if name == "stopped" && c.Cut != nil {
				ref.OwnerPID = c.Cut.PID
				ref.AcknowledgedSIGKILL = true
			}
			if name == "final" && c.Name == "generic-unknown" {
				var opened struct{ PID int }
				raw, err := os.ReadFile(filepath.Join(c.Directory, "fresh-owner", "open.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(raw, &opened); err != nil || opened.PID <= 0 {
					t.Fatal(err)
				}
				ref.OwnerPID = opened.PID
			}
			captures = append(captures, ref)
		}
	}
	bindingIndex(t, c.Directory, bindingCase{Name: c.Name, Schedule: "ancillary-v2-" + c.Schedule, Classification: c.Classification, Manifest: m, Captures: captures}, p)
}
