//go:build integration

package durabletest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

const reciprocalR1 = "phase3b3.resource-one"
const reciprocalR2 = "phase3b3.resource-two"

// Each capability is immutable deployment; Effect selects a real fixed operation.
// The untrusted delta input deliberately cannot change the selected operation.
type reciprocalAdapter struct{ Endpoint, Capability string }

func (a reciprocalAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	delta := int64(0)
	switch r.Effect {
	case "add-one":
		delta = 1
	case "add-ten":
		delta = 10
	default:
		return effects.Prepared{}, fmt.Errorf("unknown physical operation")
	}
	if r.Capability != a.Capability || r.Key == "" || len(r.Payload) != 1 || r.Payload["delta"].Kind != "int" {
		return effects.Prepared{}, fmt.Errorf("fixed capability/input")
	}
	wire, err := json.Marshal(resource.Request{Key: r.Key, Profile: "idempotent", Delta: delta})
	return effects.Prepared{Request: r, Protocol: "phase3b3/fixed-counter/v1", Endpoint: a.Endpoint, Model: a.Capability + "/" + r.Effect, Wire: string(wire), MaxResponseBytes: 8192}, err
}
func (a reciprocalAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "transport_error"}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewBufferString(p.Wire))
	if err != nil {
		c.Response = err.Error()
		return c
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		c.Response = err.Error()
		return c
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8193))
	c.HTTPStatus = response.StatusCode
	c.Response = string(raw)
	c.Disposition = "protocol_invalid"
	if err != nil || len(raw) > 8192 || response.StatusCode != http.StatusOK {
		return c
	}
	var receipt resource.Receipt
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	delta := int64(1)
	if p.Request.Effect == "add-ten" {
		delta = 10
	}
	if d.Decode(&receipt) != nil || d.Decode(new(any)) != io.EOF || receipt.Key != p.Request.Key || receipt.Profile != "idempotent" || receipt.Delta != delta || receipt.Mutation < 1 || receipt.PID < 1 || receipt.Process == "" {
		return c
	}
	c.Disposition = "validated"
	c.Payload = map[string]schema.Value{"mutation": schema.Int(receipt.Mutation), "total": schema.Int(receipt.Total)}
	return c
}
func reciprocalAdapters(config []reciprocalAdapter) map[string]effects.Adapter {
	r := map[string]effects.Adapter{}
	for _, a := range config {
		r[a.Capability] = a
	}
	return r
}
func reciprocalAdapterProvision(t *testing.T, ua, ub, capability, operation string) (runtime.Provision, delegation.Manifest) {
	t.Helper()
	p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake})
	if err != nil {
		t.Fatal(err)
	}
	record := schema.Record{"request_id": {Kind: "string", MaxBytes: 256}, "claim": {Kind: "string", MaxBytes: 8192}}
	artifact, err := author.Compile(author.Definition{Name: "request-bound-fixed-operation", Inputs: schema.Record{}, Effects: map[string]author.Effect{operation: {Capability: capability, Input: schema.Record{"delta": {Kind: "int"}}, Output: schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}}}}, Ports: map[string]author.Port{"ready": {Peer: "b", Obligation: "grant-ready", Receive: schema.Record{"grant": {Kind: "string", MaxBytes: 256}}}, "request": {Peer: "b", Obligation: "ba-request", Receive: record}}, Steps: []author.Step{author.Receive("ready", map[string]string{"grant": "grant_data"}), author.Receive("request", map[string]string{"request_id": "peer_request", "claim": "peer_claim"}), author.Request("service_operation", operation, map[string]author.Expr{"delta": author.Integer(999)}, nil), author.Await(operation, "service_operation", map[string]string{"valid": "accepted", "disposition": "disposition", "mutation": "mutation", "total": "total"}), author.IfEqual(author.Var("accepted"), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()})}})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs[0] = runtime.Artifact{ID: artifact.ID(), Semantics: author.SemanticVersion, Bytes: artifact.Bytes()}
	p.Runs[0].ProgramID = artifact.ID()
	p.Runs[0].Projection.Data = map[string]schema.Value{}
	root := runtime.AuthorityRoot{Run: m.B, RootRun: m.B, Permission: runtime.Permission{Kind: "adapter", Source: "b", Target: capability, Operation: operation, Revision: 1, Obligation: operation}}
	m.BAServiceRoot = root
	m.BAService = `["a",1,3,0]`
	m.ABRequest = ""
	m.ABService = ""
	m.ABReply = ""
	m.BAReply = ""
	m.Claim.Root = root
	m.Claim.ServiceOperationID = m.BAService
	reciprocalClaim(t, &p, func(c *runtime.DelegationClaim) { *c = m.Claim })
	reciprocalEdit(t, &p, "a", func(program map[string]any) {
		effect := reciprocalInstruction(program, 2)["Effect"].(map[string]any)
		effect["Authority"] = runtime.AuthorityUse{Version: runtime.RequestAuthorityVersion, Path: "delegated", Root: root, GrantID: m.Grant}
	})
	reciprocalEdit(t, &p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		code[3].(map[string]any)["Otherwise"] = 6
		program["Code"] = append(code[:5], reciprocalInput("hold"), map[string]any{"Op": "end", "Outcome": "failed"})
	})
	p.Permissions = []runtime.Permission{{Kind: "adapter", Source: "b", Target: reciprocalR1, Operation: "add-one", Revision: 1, Obligation: "add-one"}, {Kind: "adapter", Source: "b", Target: reciprocalR2, Operation: "add-ten", Revision: 1, Obligation: "add-ten"}, {Kind: "message", Source: "b", Target: "a", Operation: "message", Revision: 1, Obligation: "ba-request"}, {Kind: "message", Source: "b", Target: "a", Operation: "message", Revision: 1, Obligation: "grant-ready"}}
	p.Composition.Roots[0].Capabilities = []string{capability}
	return p, m
}
func TestPhase3B3CartesianOperations(t *testing.T) {
	for _, tc := range []struct {
		name, capability, operation string
		allowed                     bool
		delta                       int64
	}{{"r1-add-one", reciprocalR1, "add-one", true, 1}, {"r2-add-ten", reciprocalR2, "add-ten", true, 10}, {"r1-add-ten", reciprocalR1, "add-ten", false, 0}, {"r2-add-one", reciprocalR2, "add-one", false, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := reciprocalDirectory(t, "cartesian-"+tc.name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m := reciprocalAdapterProvision(t, ua, ub, tc.capability, tc.operation)
			config := []reciprocalAdapter{{ua, reciprocalR1}, {ub, reciprocalR2}}
			baseline3b3Write(t, dir, "deployment-adapters.json", config)
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, runErr := s.Run(t.Context(), runtime.RunOptions{Adapters: reciprocalAdapters(config), MaxSteps: 128, MaxDispatch: 128, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			s.Close()
			baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
			if runErr != nil {
				t.Fatal(runErr)
			}
			endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
			phaseStopEndpoint(t, ca)
			phaseStopEndpoint(t, cb)
			baseline3b3Write(t, dir, "endpoints.json", endpoints)
			reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
			reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
			snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
			raw, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			var grant runtime.LifecycleResult
			if len(raw.Tables["lifecycle_results"]) != 1 {
				t.Fatal("grant missing")
			}
			if err = json.Unmarshal([]byte(raw.Tables["lifecycle_results"][0]["result"].(string)), &grant); err != nil || grant.Accepted != tc.allowed {
				t.Fatal("tuple issuance", grant, err)
			}
			classification := "safe-refusal"
			if tc.allowed {
				classification = "safe-complete-service"
				e, ok := snap.Effect(m.BAService)
				if !ok || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.AdapterAttempt.Authority == nil || len(e.AuthorityAdmissions) != 1 || !e.AuthorityAdmissions[0].Allowed || e.AdapterResult == nil || e.ResultDisposition != "validated" || e.Settlement == nil {
					t.Fatal("full tuple path missing", e)
				}
				result, ok := snap.Observation(e.Settlement.ObservationID)
				if !ok || result.Run != m.A || result.Operation != m.BAService || result.SelectedAt == nil {
					t.Fatal("original adapter result", result)
				}
				reciprocalSelectedHistory(t, snap, result)
				if e.AdapterAttempt.Prepared.Request.Effect != tc.operation || e.AdapterAttempt.Prepared.Request.Capability != tc.capability || e.AdapterAttempt.Prepared.Request.Payload["delta"].Number != 999 {
					t.Fatal("declaration/wire mismatch", e)
				}
				var wire resource.Request
				json.Unmarshal([]byte(e.AdapterAttempt.Prepared.Wire), &wire)
				if wire.Delta != tc.delta {
					t.Fatal("payload overrode operation", wire)
				}
				var receipt resource.Receipt
				if err = json.Unmarshal([]byte(e.AdapterResult.Candidate.Response), &receipt); err != nil {
					t.Fatal(err)
				}
				target := "a"
				other := "b"
				if tc.capability == reciprocalR2 {
					target, other = other, target
				}
				ledger := endpoints[target]
				if ledger.Total != tc.delta || ledger.Contacts != 1 || len(ledger.Applications) != 1 || ledger.Applications[0] != receipt || len(ledger.Requests) != 1 || ledger.Requests[0] != wire || endpoints[other].Contacts != 0 {
					t.Fatal("physical operation mismatch", endpoints, receipt)
				}
			} else {
				if len(snap.Effects) != 0 || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 || len(raw.Tables["composition_charges"]) != 0 {
					t.Fatal("Cartesian authority escaped")
				}
			}
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "cartesian-" + tc.name, Schedule: "automatic-run", Directory: dir, Classification: classification, Manifest: m})
			baseline3b3Manifest(t, dir)
		})
	}
}
func TestPhase3B3GenericUnknown(t *testing.T) {
	dir := reciprocalDirectory(t, "generic-unknown")
	ca, ua, pa := reciprocalEndpoint(t, dir, "a", &barrier{Boundary: "resource.applied", Phase: "post-commit"})
	cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
	p, m := reciprocalAdapterProvision(t, ua, ub, reciprocalR1, "add-one")
	config := []reciprocalAdapter{{ua, reciprocalR1}, {ub, reciprocalR2}}
	baseline3b3Write(t, dir, "deployment-adapters.json", config)
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "manifest.json", m)
	db := filepath.Join(t.TempDir(), "runtime.sqlite")
	s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	owner := func(out string, filter *barrier) *child {
		path := filepath.Join(dir, out)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		return phaseTestProcess(t, db, "", "", filter, "TestPhase3B3Owner", func(cmd *exec.Cmd) {
			raw, _ := json.Marshal(config)
			cmd.Env = append(cmd.Env, "ALAN_PHASE3B3_CHILD=1", "ALAN_PHASE3B3_OUTPUT="+path, "ALAN_PHASE3B3_ADAPTERS="+string(raw))
		})
	}
	first := owner("first-owner", &barrier{Boundary: "runner.waiting", Phase: "in-flight"})
	endpointAck := reciprocalAwait(t, ca)
	ack := reciprocalAwait(t, first)
	baseline3b3Write(t, dir, "acknowledgments.json", map[string]any{"Runtime": ack, "Endpoint": endpointAck})
	first.kill(t)
	baseline3b3Write(t, dir, "kill.json", map[string]any{"PID": first.cmd.Process.Pid, "Wait": first.cmd.ProcessState.String()})
	stopped := reciprocalCapture(t, db, filepath.Join(dir, "stopped"))
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
	final := reciprocalCapture(t, db, filepath.Join(dir, "final"))
	e, _ := final.Effect(m.BAService)
	if e == nil || e.AdapterResult != nil || e.Settlement != nil || len(e.AuthorityAdmissions) != 1 || endpoints["a"].Contacts != 1 || len(endpoints["a"].Applications) != 1 || endpoints["b"].Contacts != 0 {
		t.Fatal("generic unknown retried or invented result", e, endpoints)
	}
	a, _ := final.Run(m.A)
	if a.Projection.Status != "open" || a.Sequence != 3 {
		t.Fatal("unknown did not remain pending", a)
	}
	reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "generic-unknown", Schedule: "automatic-run", Directory: dir, Classification: "transport-unknown", Manifest: m, Cut: &ack})
	baseline3b3Manifest(t, dir)
}
