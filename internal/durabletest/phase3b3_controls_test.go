//go:build integration

package durabletest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

// Mutations are input programs, never live-store edits or host-written grants.
func reciprocalEdit(t *testing.T, p *runtime.Provision, subject string, edit func(map[string]any)) {
	t.Helper()
	for ri := range p.Runs {
		if p.Runs[ri].ID.Subject != subject {
			continue
		}
		old := p.Runs[ri].ProgramID
		for ai := range p.Programs {
			if p.Programs[ai].ID != old {
				continue
			}
			var program map[string]any
			if err := json.Unmarshal(p.Programs[ai].Bytes, &program); err != nil {
				t.Fatal(err)
			}
			edit(program)
			raw, err := json.Marshal(program)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			id := hex.EncodeToString(sum[:])
			p.Programs[ai].Bytes = raw
			p.Programs[ai].ID = id
			p.Runs[ri].ProgramID = id
			return
		}
	}
	t.Fatal("missing subject", subject)
}
func reciprocalInstruction(p map[string]any, index int) map[string]any {
	return p["Code"].([]any)[index].(map[string]any)
}
func reciprocalClaim(t *testing.T, p *runtime.Provision, edit func(*runtime.DelegationClaim)) {
	reciprocalEdit(t, p, "b", func(program map[string]any) {
		payload := reciprocalInstruction(program, 1)["Effect"].(map[string]any)["Payload"].(map[string]any)
		value := payload["claim"].(map[string]any)["Literal"].(map[string]any)
		var claim runtime.DelegationClaim
		if err := json.Unmarshal([]byte(value["Text"].(string)), &claim); err != nil {
			t.Fatal(err)
		}
		edit(&claim)
		raw, _ := json.Marshal(claim)
		value["Text"] = string(raw)
	})
}
func reciprocalUse(t *testing.T, p *runtime.Provision, edit func(*runtime.AuthorityUse)) {
	reciprocalEdit(t, p, "a", func(program map[string]any) {
		effect := reciprocalInstruction(program, 3)["Effect"].(map[string]any)
		raw, _ := json.Marshal(effect["Authority"])
		var use runtime.AuthorityUse
		if err := json.Unmarshal(raw, &use); err != nil {
			t.Fatal(err)
		}
		edit(&use)
		effect["Authority"] = use
	})
}
func reciprocalOrdinaryControl(t *testing.T, name string, mutate func(*runtime.Provision, delegation.Manifest)) {
	t.Helper()
	dir := reciprocalDirectory(t, "control-"+name)
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	mutate(&p, m)
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "create.json", map[string]string{"Error": fmt.Sprint(err)})
	if err != nil {
		t.Fatal("control must be an executable retained input", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	report, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
	ea, eb := phaseSnapshot(t, pa), phaseSnapshot(t, pb)
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", map[string]any{"a": ea, "b": eb})
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
	if runErr != nil || ea.Contacts != 0 || eb.Contacts != 0 {
		t.Fatal("adversarial input reached endpoint or errored", runErr, ea, eb)
	}
	for _, e := range snap.Effects {
		if e.ExternalAttempt != nil && e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 0 || e.ExternalReceipt != nil {
			t.Fatal("unauthorized service", e)
		}
	}
	// Fresh owner must not repair a refused original by reconstructing authority.
	s, err = runtime.Open(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2})
	s.Close()
	baseline3b3Write(t, dir, "reopened-run.json", map[string]any{"Report": reopened, "Error": fmt.Sprint(err)})
	if err != nil {
		t.Fatal(err)
	}
	reciprocalCapture(t, db, filepath.Join(dir, "reopened"))
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "control-" + name, Schedule: "automatic-run", Directory: dir, Classification: "safe-refusal", Manifest: m})
	baseline3b3Manifest(t, dir)
}
func TestPhase3B3ReciprocalBindings(t *testing.T) {
	cases := []struct {
		name string
		edit func(*runtime.Provision, delegation.Manifest)
	}{
		{"removed-original-tuple", func(p *runtime.Provision, m delegation.Manifest) { p.Permissions = p.Permissions[1:] }},
		{"issuer-generation", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.Root.Run.Generation = 2 })
		}},
		{"issuer-root", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.Root.RootRun = m.A })
		}},
		{"issuer-revision", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.Root.Permission.Revision = 2 })
		}},
		{"recipient-generation", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.Recipient.Generation = 2 })
		}},
		{"recipient-root", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.RecipientRoot = m.B })
		}},
		{"recipient-revision", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.RecipientRevision = 2 })
		}},
		{"other-request", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.OriginalRequestID = m.ABRequest })
		}},
		{"request-generation", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.OriginalRequestID = `["b",2,1,0]` })
		}},
		{"service-sequence", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = `["a",1,5,0]` })
		}},
		{"service-generation", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalClaim(t, p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = `["a",2,4,0]` })
		}},
		{"service-resource", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalUse(t, p, func(u *runtime.AuthorityUse) { u.Root.Permission.Target = m.Options.EndpointA })
		}},
		{"result-as-grant", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalUse(t, p, func(u *runtime.AuthorityUse) { u.GrantID = `["lifecycle-outcome","[\"b\",1,2,0]"]` })
		}},
		{"invalid-selected-no-own-fallback", func(p *runtime.Provision, m delegation.Manifest) {
			own := m.BAServiceRoot.Permission
			own.Source = "a"
			p.Permissions = append(p.Permissions, own)
			reciprocalUse(t, p, func(u *runtime.AuthorityUse) { u.GrantID = m.BARequest })
		}},
		{"wrong-prior-selection", func(p *runtime.Provision, m delegation.Manifest) {
			reciprocalEdit(t, p, "a", func(program map[string]any) { code := program["Code"].([]any); code[1], code[2] = code[2], code[1] })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { reciprocalOrdinaryControl(t, tc.name, tc.edit) })
	}
}

func TestPhase3B3ActualResultLaundering(t *testing.T) {
	dir := reciprocalDirectory(t, "actual-result-laundering")
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
	snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
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
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "actual-prior-service-result-as-grant", Schedule: "automatic-run-reversed-service-policy", Directory: dir, Classification: "safe-refusal", Manifest: m})
	baseline3b3Manifest(t, dir)
}
func TestPhase3B3SerialSerial(t *testing.T) {
	dir := reciprocalDirectory(t, "serial-serial")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	reciprocalEdit(t, &p, "a", func(program map[string]any) { code := program["Code"].([]any); code[2], code[7] = code[7], code[2] })
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 128, MaxDispatch: 128, MaxInFlight: 2})
	s.Close()
	if err != nil || !report.Quiescent {
		t.Fatal("serial wait", report, err)
	}
	baseline3b3Write(t, dir, "run.json", report)
	ea, eb := phaseSnapshot(t, pa), phaseSnapshot(t, pb)
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", map[string]any{"a": ea, "b": eb})
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
	a, _ := snap.Run(m.A)
	b, _ := snap.Run(m.B)
	if a.Sequence != 2 || b.Sequence != 5 || a.Wait == nil || a.Wait.Obligation != "ab-reply" || b.Wait == nil || b.Wait.Obligation != "ba-reply" || len(snap.Effects) != 0 || a.Projection.Data["paid"].Flag || b.Projection.Data["paid"].Flag || ea.Contacts != 0 || eb.Contacts != 0 {
		t.Fatal("not genuine serial deadlock", a, b)
	}
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "serial-serial", Schedule: "automatic-run-serial-serial-policy", Directory: dir, Classification: "scheduler-deadlock", Manifest: m})
	baseline3b3Manifest(t, dir)
}
