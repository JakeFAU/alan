//go:build integration

package durabletest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// This adapter is trusted fixture deployment configuration. Each capability
// denotes one complete resource/scope tuple and has its own separate endpoint.
// Authored claims and prior receipts never select a resource or configuration.
type baseline3b3Counter struct{ endpoint, capability, resource, scope string }

func (a baseline3b3Counter) Prepare(r effects.Request) (effects.Prepared, error) {
	u, err := url.Parse(a.endpoint)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return effects.Prepared{}, fmt.Errorf("fixed loopback endpoint required")
	}
	if r.Capability != a.capability || r.Key == "" || len(r.Payload) != 3 || r.Payload["delta"] != schema.Int(1) || r.Payload["claim"].Kind != "string" || r.Payload["receipt_mutation"].Kind != "int" {
		return effects.Prepared{}, fmt.Errorf("fixed tuple/input required")
	}
	wire, err := json.Marshal(resource.Request{Key: r.Key, Profile: "idempotent", Delta: 1})
	return effects.Prepared{Request: r, Protocol: "phase3b3/baseline-counter/v1", Endpoint: a.endpoint, Model: a.resource + "/" + a.scope, Wire: string(wire), MaxResponseBytes: 8192}, err
}
func (a baseline3b3Counter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
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
	if d.Decode(&receipt) != nil || d.Decode(new(any)) != io.EOF || receipt.Key != p.Request.Key || receipt.Profile != "idempotent" || receipt.Delta != 1 || receipt.Mutation < 1 || receipt.PID < 1 || receipt.Process == "" {
		return c
	}
	c.Disposition = "validated"
	c.Payload = map[string]schema.Value{"mutation": schema.Int(receipt.Mutation), "total": schema.Int(receipt.Total)}
	return c
}

type baseline3b3Evidence struct {
	directory string
	snapshot  *runtime.ExecutionSnapshot
	history   runtime.HistoryPage
	endpoints map[string]resource.Snapshot
}

func baseline3b3Write(t *testing.T, dir, name string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func baseline3b3Manifest(t *testing.T, dir string) {
	t.Helper()
	sums := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() == "digests.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sums[relative] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	baseline3b3Write(t, dir, "digests.json", sums)
}

// All execution, inspection and replay use public runtime APIs. Endpoint ledgers
// are independent separate processes using the existing local counter fixture.
func baseline3b3Exercise(t *testing.T, label string, p runtime.Provision) baseline3b3Evidence {
	t.Helper()
	dir := t.TempDir()
	if root := os.Getenv("ALAN_PHASE3B3_EVIDENCE"); root != "" {
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		var err error
		dir, err = os.MkdirTemp(root, label+"-")
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("raw evidence:", dir)
	configurations := []baseline3b3Counter{
		{capability: delegation.BaselineCounterARed, resource: "counter-A", scope: "red"},
		{capability: delegation.BaselineCounterBBlue, resource: "counter-B", scope: "blue"},
		{capability: delegation.BaselineCounterABlue, resource: "counter-A", scope: "blue"},
	}
	registry := map[string]effects.Adapter{}
	children := map[string]*child{}
	paths := map[string]string{}
	for i := range configurations {
		c, endpoint, path := startResource(t, false)
		configurations[i].endpoint = endpoint
		a := configurations[i]
		registry[a.capability] = a
		children[a.capability] = c
		paths[a.capability] = path
	}
	baseline3b3Write(t, dir, "provision.json", p)
	baseline3b3Write(t, dir, "configuration.json", map[string]any{"RuntimeBaseCommit": "af6943a5805a9a6c0e8fe77f804264f4765d07ec", "Label": label, "Capabilities": map[string]string{delegation.BaselineCounterARed: configurations[0].endpoint, delegation.BaselineCounterBBlue: configurations[1].endpoint, delegation.BaselineCounterABlue: configurations[2].endpoint}})
	path := filepath.Join(dir, "runtime.sqlite")
	store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	report, runErr := store.Run(ctx, runtime.RunOptions{Adapters: registry, MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 2 * time.Second})
	baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	baseline3b3Write(t, dir, "inspect.json", raw)
	snapshot, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	baseline3b3Write(t, dir, "typed-inspection.json", snapshot)
	history, err := snapshot.History(0, runtime.MaxHistoryPage)
	if err != nil || history.More {
		t.Fatal("history bounds", err)
	}
	baseline3b3Write(t, dir, "history.json", history)
	endpoints := map[string]resource.Snapshot{}
	for capability, child := range children {
		endpoints[capability] = phaseSnapshot(t, paths[capability])
		phaseStopEndpoint(t, child)
		// Keep exact closed-process database bytes and any remaining WAL sidecars.
		for _, suffix := range []string{"", "-wal", "-shm"} {
			source := paths[capability] + suffix
			if _, err = os.Stat(source); err == nil {
				phaseCopy(t, source, filepath.Join(dir, capability+".sqlite"+suffix))
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
	}
	baseline3b3Write(t, dir, "endpoint-ledgers-before-replay.json", endpoints)
	replay, replayErr := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "replay.json", map[string]any{"Report": replay, "Error": fmt.Sprint(replayErr)})
	// The endpoint processes are stopped before Replay: replay must need no network.
	baseline3b3Write(t, dir, "endpoint-processes-stopped-before-replay.json", true)
	baseline3b3Manifest(t, dir)
	if runErr != nil || !report.Quiescent || replayErr != nil {
		t.Fatal("run/replay", report, runErr, replayErr)
	}
	return baseline3b3Evidence{dir, snapshot, history, endpoints}
}

type baseline3b3Binding struct {
	RequestOperation                 string
	Requester, Server                runtime.RunID
	RequestObservation               string
	RequestSelection                 runtime.ReductionRef
	ServiceOperation                 string
	ServiceSequence                  int64
	ResultObservation                string
	ResultSelection                  *runtime.ReductionRef
	ReplyOperation, ReplyObservation string
	ReplySelection                   *runtime.ReductionRef
}

// Follow the authentic original envelope, not merely the equal authored request
// string. Every positive custody link is checked against the retained producer,
// original service operation, generation and actual receive selection.
func baseline3b3Bindings(t *testing.T, e baseline3b3Evidence) []baseline3b3Binding {
	t.Helper()
	s := e.snapshot
	bindings := []baseline3b3Binding{}
	for _, effect := range s.Effects {
		if effect.Source.Generation != 1 || effect.Authorization == nil || !effect.Authorization.Allowed || effect.AdapterAttempt == nil || effect.AdapterResult == nil || effect.Settlement == nil {
			t.Fatalf("incomplete original effect: %+v", effect)
		}
		var request *runtime.ObservationView
		for _, o := range s.Observations {
			if o.Kind == "message" && o.Run == effect.Source && o.Payload["request_id"].Kind == "string" && o.SelectedAt != nil {
				if request != nil {
					t.Fatal("multiple original request selections")
				}
				request = o
			}
		}
		if request == nil || request.ProducerRun == nil || request.SourceOperationID == "" || request.ProducerRun.Generation != 1 || request.SelectedAt.Run != effect.Source || request.SelectedAt.Sequence >= effect.Sequence {
			t.Fatal("no exact request precedes service", effect, request)
		}
		original, ok := s.Operation(request.SourceOperationID)
		if !ok || original.Run != *request.ProducerRun || original.Target != effect.Source.Subject || !reflect.DeepEqual(original.Payload, request.Payload) {
			t.Fatal("unauthentic request producer")
		}
		result, ok := s.Observation(effect.Settlement.ObservationID)
		if !ok || result.SelectedAt == nil || result.Run != effect.Source || result.Operation != effect.OperationID || result.SourceOperationID != effect.OperationID || result.SelectedAt.Sequence <= effect.Sequence {
			t.Fatal("wrong original effect result selection", effect, result)
		}
		b := baseline3b3Binding{RequestOperation: original.ID, Requester: original.Run, Server: effect.Source, RequestObservation: request.ID, RequestSelection: *request.SelectedAt, ServiceOperation: effect.OperationID, ServiceSequence: effect.Sequence, ResultObservation: result.ID, ResultSelection: result.SelectedAt}
		for _, reply := range s.Observations {
			if reply.Kind != "message" || reply.Payload["service_operation"].Text != effect.OperationID {
				continue
			}
			if effect.ResultDisposition != "validated" || reply.ProducerRun == nil || *reply.ProducerRun != effect.Source || reply.Run != original.Run || reply.SelectedAt == nil || reply.Payload["response_id"] != original.Payload["request_id"] || reply.Payload["mutation"] != effect.AdapterResult.Values["mutation"] || reply.Payload["total"] != effect.AdapterResult.Values["total"] {
				t.Fatal("reply not bound to exact original result", reply)
			}
			producing, ok := s.Operation(reply.SourceOperationID)
			if !ok || producing.Run != effect.Source || producing.Sequence <= result.SelectedAt.Sequence || reply.ProducerReduction == nil || reply.ProducerReduction.Sequence != producing.Sequence {
				t.Fatal("reply preceded original result custody")
			}
			b.ReplyOperation = producing.ID
			b.ReplyObservation = reply.ID
			b.ReplySelection = reply.SelectedAt
		}
		if effect.ResultDisposition == "validated" && b.ReplySelection == nil {
			t.Fatal("authentic service result not consumed by requester")
		}
		bindings = append(bindings, b)
	}
	baseline3b3Write(t, e.directory, "original-bindings.json", bindings)
	baseline3b3Manifest(t, e.directory)
	return bindings
}

func baseline3b3CheckContacts(t *testing.T, e baseline3b3Evidence, want map[string]int64) {
	t.Helper()
	for capability, ledger := range e.endpoints {
		if ledger.Contacts != want[capability] || int64(len(ledger.Requests)) != want[capability] || int64(len(ledger.Applications)) != want[capability] {
			t.Fatalf("%s: contacts/applications %+v want %d", capability, ledger, want[capability])
		}
		for _, request := range ledger.Requests {
			found := false
			for _, effect := range e.snapshot.Effects {
				attempt := effect.AdapterAttempt
				if attempt != nil && attempt.Allowed && effect.Capability == capability && attempt.Prepared.Request.Key == request.Key && effect.AdapterResult != nil && effect.ResultDisposition == "validated" {
					var receipt resource.Receipt
					if err := json.Unmarshal([]byte(effect.AdapterResult.Candidate.Response), &receipt); err != nil {
						t.Fatal(err)
					}
					if receipt != ledger.Applications[0] || receipt.Key != request.Key || request.Delta != 1 {
						t.Fatal("endpoint result differs from selected original")
					}
					found = true
				}
			}
			if !found {
				t.Fatal("unauthorized or unaccounted physical contact", capability, request)
			}
		}
	}
}

func TestPhase3B3Baseline(t *testing.T) {
	for _, scenario := range []delegation.BaselineScenario{delegation.BaselineLocal, delegation.BaselineMissing, delegation.BaselineClaim, delegation.BaselineCartesian, delegation.BaselineResultLaunder} {
		t.Run(string(scenario), func(t *testing.T) {
			p, err := delegation.Baseline(scenario)
			if err != nil {
				t.Fatal(err)
			}
			e := baseline3b3Exercise(t, string(scenario), p)
			bindings := baseline3b3Bindings(t, e)
			a, _ := e.snapshot.Run(runtime.RunID{Subject: "a", Generation: 1})
			b, _ := e.snapshot.Run(runtime.RunID{Subject: "b", Generation: 1})
			want := map[string]int64{}
			if scenario == delegation.BaselineLocal {
				want[delegation.BaselineCounterARed] = 1
				want[delegation.BaselineCounterBBlue] = 1
				if len(bindings) != 2 || !a.Projection.Data["paid"].Flag || !b.Projection.Data["paid"].Flag {
					t.Fatal("S1 did not consume two authentic results")
				}
				var aReplyToB *runtime.ReductionRef
				var bServiceSequence int64
				for _, binding := range bindings {
					if binding.ReplySelection == nil {
						t.Fatal("missing positive custody")
					}
					if binding.Server.Subject == "a" {
						aReplyToB = binding.ReplySelection
					}
					if binding.Server.Subject == "b" {
						bServiceSequence = binding.ServiceSequence
					}
				}
				if aReplyToB == nil || aReplyToB.Run.Subject != "b" || aReplyToB.Sequence >= bServiceSequence {
					t.Fatal("positive fixture lost A-servicing/B-serial policy")
				}
			} else {
				operation := a.Projection.Data["operation"].Text
				effect, ok := e.snapshot.Effect(operation)
				if !ok || effect.Source != a.ID || effect.AdapterAttempt.Allowed || effect.AdapterAttempt.Reason != "grant_denied" || effect.ResultDisposition != "grant_denied" || effect.Consequence != runtime.ConsequenceNoDispatch || effect.Dispatch.Minimum != 0 || effect.Dispatch.Maximum == nil || *effect.Dispatch.Maximum != 0 || !a.Projection.Data["denied"].Flag || b.Projection.Data["paid"].Flag {
					t.Fatal("expected original authority refusal", effect)
				}
				if scenario == delegation.BaselineResultLaunder {
					want[delegation.BaselineCounterBBlue] = 1
					if !a.Projection.Data["paid"].Flag || len(bindings) != 2 || effect.AdapterAttempt.Prepared.Request.Payload["receipt_mutation"].Number != 1 || effect.AdapterAttempt.Prepared.Request.Payload["claim"].Text != "result-from-b-is-not-authority" {
						t.Fatal("no prior authentic B result laundering attempt")
					}
					// This reversed-order negative control is outside the positive service policy.
					var prior *baseline3b3Binding
					for i := range bindings {
						if bindings[i].Server.Subject == "b" {
							prior = &bindings[i]
						}
					}
					if prior == nil || prior.ReplySelection == nil || prior.ReplySelection.Run != effect.Source || prior.ReplySelection.Sequence >= effect.Sequence {
						t.Fatal("A attempted before consuming authentic B result")
					}
				} else if len(bindings) != 1 || a.Projection.Data["paid"].Flag {
					t.Fatal("unexpected completion before refusal")
				}
				if scenario == delegation.BaselineClaim && effect.AdapterAttempt.Prepared.Request.Payload["claim"].Text != "grant:issuer=b;grantee=a;parent=b-root;permission=increment/counter-B/blue" {
					t.Fatal("claim-shaped data did not reach actual attempt")
				}
				if scenario == delegation.BaselineCartesian && effect.Capability != delegation.BaselineCounterABlue {
					t.Fatal("Cartesian attempt did not select unheld resource/scope tuple")
				}
			}
			baseline3b3CheckContacts(t, e, want)
		})
	}
}

// Opt-in RED demonstration. The ordinary suite checks the honest refusal and
// stays green. This separate command asserts the desired S2 completion on the
// unchanged runtime and must fail only after the S1 apparatus control passes.
func TestPhase3B3BaselineS2CompletionExpectedFailure(t *testing.T) {
	if os.Getenv("ALAN_PHASE3B3_EXPECT_S2_COMPLETION") != "1" {
		t.Skip("opt-in historical S2 completion falsifier")
	}
	control, err := delegation.Baseline(delegation.BaselineLocal)
	if err != nil {
		t.Fatal(err)
	}
	c := baseline3b3Exercise(t, "expected-failure-s1-control", control)
	baseline3b3Bindings(t, c)
	for _, run := range c.snapshot.Runs {
		if !run.Projection.Data["paid"].Flag {
			t.Fatal("S1 apparatus failed")
		}
	}
	p, err := delegation.Baseline(delegation.BaselineMissing)
	if err != nil {
		t.Fatal(err)
	}
	e := baseline3b3Exercise(t, "expected-failure-s2-missing", p)
	baseline3b3Bindings(t, e)
	baseline3b3CheckContacts(t, e, map[string]int64{})
	a, _ := e.snapshot.Run(runtime.RunID{Subject: "a", Generation: 1})
	effect, ok := e.snapshot.Effect(a.Projection.Data["operation"].Text)
	if !ok || !effect.Authorization.Allowed || effect.AdapterAttempt.Allowed || effect.AdapterAttempt.Reason != "grant_denied" {
		t.Fatal("unexpected falsifier cause", effect)
	}
	for _, run := range e.snapshot.Runs {
		if !run.Projection.Data["paid"].Flag {
			t.Fatalf("EXPECTED FAIL: desired S2 completion unavailable on unchanged public API: original %s run=%+v authorization=allowed adapter=grant_denied; both authentic results required", effect.OperationID, effect.Source)
		}
	}
}

// Scope-only labels do not exercise the operation coordinate. This additional
// two-root control fixes the complete global permission set to
// P={(counter-A/red,op1),(counter-B/blue,op2)} and actually attempts the absent
// (counter-A/red,op2). It is a tuple lookup control, not reciprocal S1/S2 service.
func TestPhase3B3BaselineCartesianOperation(t *testing.T) {
	input := schema.Record{"delta": {Kind: "int"}, "claim": {Kind: "string", MaxBytes: 256}, "receipt_mutation": {Kind: "int"}}
	output := schema.Record{"mutation": {Kind: "int"}, "total": {Kind: "int"}}
	compile := func(subject, capability string, names []string) author.Artifact {
		effects := map[string]author.Effect{}
		steps := []author.Step{}
		for _, name := range names {
			effects[name] = author.Effect{Capability: capability, Input: input, Output: output}
			steps = append(steps, author.Request(name, name, map[string]author.Expr{"delta": author.Integer(1), "claim": author.String("tuple-control"), "receipt_mutation": author.Integer(0)}, nil), author.Await(name, name, map[string]string{"valid": name + "_valid", "disposition": name + "_disposition"}))
		}
		steps = append(steps, author.Receive("hold", map[string]string{"release": "release"}), author.Success())
		artifact, err := author.Compile(author.Definition{Name: "phase3b3-operation-tuple-" + subject, Effects: effects, Ports: map[string]author.Port{"hold": {Peer: "local", Local: true, Obligation: "hold", Receive: schema.Record{"release": {Kind: "bool"}}}}, Steps: steps})
		if err != nil {
			t.Fatal(err)
		}
		return artifact
	}
	p, err := (author.Application{
		"a": {Artifact: compile("a", delegation.BaselineCounterARed, []string{"op1", "op2"})},
		"b": {Artifact: compile("b", delegation.BaselineCounterBBlue, []string{"op2"})},
	}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	filtered := p.Permissions[:0]
	for _, permission := range p.Permissions {
		if permission.Kind == "adapter" && permission.Source == "a" && permission.Operation == "op2" {
			continue
		}
		filtered = append(filtered, permission)
	}
	p.Permissions = filtered
	e := baseline3b3Exercise(t, "cartesian-operation", p)
	if len(e.snapshot.Effects) != 3 {
		t.Fatal("missing actual whole-tuple operations")
	}
	allowed := map[string]bool{}
	for _, effect := range e.snapshot.Effects {
		if effect.Authorization == nil || !effect.Authorization.Allowed || effect.AdapterAttempt == nil || effect.AdapterResult == nil || effect.Settlement == nil {
			t.Fatal("missing actual permission decision", effect)
		}
		selected, ok := e.snapshot.Observation(effect.Settlement.ObservationID)
		if !ok || selected.SelectedAt == nil || selected.Operation != effect.OperationID {
			t.Fatal("missing original result selection")
		}
		if effect.Source.Subject == "a" && effect.Name == "op2" {
			if effect.AdapterAttempt.Allowed || effect.AdapterAttempt.Reason != "grant_denied" || effect.ResultDisposition != "grant_denied" || effect.Dispatch.Maximum == nil || *effect.Dispatch.Maximum != 0 {
				t.Fatal("operation cross product invented", effect)
			}
		} else {
			if !effect.AdapterAttempt.Allowed || effect.ResultDisposition != "validated" {
				t.Fatal("positive tuple control failed", effect)
			}
			allowed[effect.Capability+"/"+effect.Name] = true
		}
	}
	if !reflect.DeepEqual(allowed, map[string]bool{delegation.BaselineCounterARed + "/op1": true, delegation.BaselineCounterBBlue + "/op2": true}) {
		t.Fatal("wrong whole-tuple control", allowed)
	}
	baseline3b3CheckContacts(t, e, map[string]int64{delegation.BaselineCounterARed: 1, delegation.BaselineCounterBBlue: 1})
}
