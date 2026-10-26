//go:build integration

package evidencetest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// TestRunnerChild is a test-only generic owner host, used for additional static
// mixed-transport programs. Application crash tests use the actual CLI instead.
func TestRunnerChild(t *testing.T) {
	if os.Getenv("ALAN_TEST_RUNNER") != "1" {
		t.Skip("child entrypoint")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	db := os.Getenv("ALAN_RUNNER_DB")
	ctx, err := boundary.FromEnvironment(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Open(ctx, db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	model, err := openai.New(openai.Config{APIKey: "synthetic-process-fixture", Model: "controlled-requested-model", Endpoint: os.Getenv("ALAN_RUNNER_ENDPOINT"), MaxOutputTokens: 128, MaxResponseBytes: 65536})
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := strconv.Atoi(os.Getenv("ALAN_RUNNER_CAPACITY"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 1000, MaxDispatch: 1000, MaxInFlight: capacity, ExternalTimeout: 2 * time.Minute, Adapters: map[string]effects.Adapter{"model": model, "lookup": lookup.New()}})
	raw, _ := json.Marshal(map[string]any{"report": report, "error": fmt.Sprint(err)})
	fmt.Println(string(raw))
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
}
func compile(t *testing.T, d author.Definition) author.Artifact {
	t.Helper()
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func mixedProvision(t *testing.T, endpoint string, closeModel bool) runtime.Provision {
	t.Helper()
	str := schema.Field{Kind: "string", MaxBytes: 4096}
	decision := schema.Record{"decision": str, "tool": str, "query": str, "message": str}
	steps := []author.Step{author.Request("pending", "model", map[string]author.Expr{"instructions": author.String("Controlled process fixture."), "input": author.String("WAL")}, nil)}
	if !closeModel {
		steps = append(steps, author.Await("model", "pending", map[string]string{"decision": "decision"}))
	}
	steps = append(steps, author.Success())
	model := compile(t, author.Definition{Name: "held model", Effects: map[string]author.Effect{"model": {Capability: "model", Input: schema.Record{"instructions": str, "input": str}, Output: decision}}, Steps: steps})
	msg := schema.Record{"value": str}
	peer := compile(t, author.Definition{Name: "ordinary peer", Ports: map[string]author.Port{"out": {Peer: "caller", Obligation: "independent", Send: msg}}, Steps: []author.Step{author.Send("out", map[string]author.Expr{"value": author.String("peer advanced while model held")}), author.Success()}})
	caller := compile(t, author.Definition{Name: "ordinary caller", Outputs: msg, Ports: map[string]author.Port{"in": {Peer: "peer", Obligation: "independent", Receive: msg}}, Steps: []author.Step{author.Receive("in", map[string]string{"value": "value"}), author.Success()}})
	tool := compile(t, author.Definition{Name: "independent lookup", Effects: map[string]author.Effect{"lookup": {Capability: "lookup", Input: lookup.InputSchema(), Output: lookup.OutputSchema()}}, Steps: []author.Step{author.Request("pending", "lookup", map[string]author.Expr{"query": author.String("WAL")}, nil), author.Await("lookup", "pending", map[string]string{"snippet": "snippet"}), author.Success()}})
	p, err := (author.Application{"amodel": {Artifact: model}, "peer": {Artifact: peer}, "caller": {Artifact: caller}, "zlookup": {Artifact: tool}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	asset, err := effects.NewAsset("source", "fixture:retained", []byte("WAL retains a durable log.\n"))
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = []effects.Asset{asset}
	// Retain the old fixture transport as a separate ordinary program. This is a
	// static test fixture, not a new instruction or application runtime branch.
	target := endpoint + "/fixture"
	prog := executor.Program{Semantics: executor.SemanticVersion, Code: []executor.Instruction{
		{Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "external", Target: executor.Lit(executor.Text(target)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("receipt")), Payload: map[string]executor.Operand{"profile": executor.Lit(executor.Text("idempotent")), "delta": executor.Lit(executor.Int(5))}}},
		{Op: "receive", Match: &executor.Match{Kind: "receipt", Provenance: "resource-receipt", Source: executor.Lit(executor.Text(target)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("receipt")), Operation: executor.Ref("pending")}, Bind: map[string]string{"total": "total"}},
		{Op: "end", Outcome: "succeeded"}}}
	raw, id, err := executor.EncodeProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: executor.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, runtime.Subject{ID: "zresource", Generation: 1, Owner: "owner:zresource", Revision: 1})
	p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: "zresource", Generation: 1}, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}})
	p.Permissions = append(p.Permissions, runtime.Permission{Kind: "external", Source: "zresource", Target: target, Operation: "apply", Revision: 1})
	return p
}
func startRunner(t *testing.T, p *provider, capacity int, closed bool, filters []boundary.Event) (*process, string) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "mixed.sqlite")
	provision := mixedProvision(t, p.endpoint, closed)
	s, err := runtime.Create(t.Context(), db, provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for i := range filters {
		filters[i].Store = db
		filters[i].Scenario = t.Name()
		filters[i].Token = fmt.Sprintf("%s-%d", t.Name(), i)
	}
	raw, _ := json.Marshal(filters)
	emit(t, map[string]any{"type": "mixed-provision", "provision": provision, "barriers": filters, "capacity": capacity})
	child := launch(t, filepath.Join(binaries(t), "provider"), []string{"-test.run=^TestRunnerChild$", "-test.v"}, []string{"ALAN_TEST_RUNNER=1", "ALAN_RUNNER_DB=" + db, "ALAN_RUNNER_ENDPOINT=" + p.endpoint, "ALAN_RUNNER_CAPACITY=" + strconv.Itoa(capacity), "ALAN_BARRIER=" + string(raw)})
	return child, db
}
func snapshot(t *testing.T, db string) runtime.Inspection {
	t.Helper()
	s, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func projection(t *testing.T, s runtime.Inspection, subject string) runtime.Projection {
	t.Helper()
	for _, r := range s.Tables["runs"] {
		if r["subject"] == subject {
			var p runtime.Projection
			if err := json.Unmarshal([]byte(r["projection"].(string)), &p); err != nil {
				t.Fatal(err)
			}
			return p
		}
	}
	t.Fatal("missing subject " + subject)
	return runtime.Projection{}
}
func TestHeldModelOrdinaryProgressAndSharedCapacity(t *testing.T) {
	for _, capacity := range []int{1, 2} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			p := startProvider(t, true)
			filters := []boundary.Event{{Boundary: "execution.staged", Subject: "peer", Sequence: 1}, {Boundary: "execution.committed", Subject: "peer", Sequence: 1}, {Boundary: "execution.committed", Subject: "caller", Sequence: 1}}
			if capacity == 2 {
				filters = append(filters, boundary.Event{Boundary: "adapter.settled", Subject: "zlookup"}, boundary.Event{Boundary: "external.settled", Subject: "zresource"})
			}
			child, db := startRunner(t, p, capacity, false, filters)
			staged := ackBoundary(t, child, "execution.staged", "peer")
			accepted := p.accepted(t, 1)
			// Only after the independent provider has durably accepted and withheld the
			// model response may the peer's staged transaction commit.
			child.send(t, "c")
			committed := ackBoundary(t, child, "execution.committed", "peer")
			child.send(t, "c")

			var delivered boundary.Event
			var during runtime.Inspection
			completed := map[string]boundary.Event{}
			needed := 1
			if capacity == 2 {
				needed = 3
			}
			for len(completed) < needed {
				var event boundary.Event
				child.next(t, &event)
				if err := validateAcknowledgment(event, child.expected, child.cmd.Process.Pid, child.used); err != nil {
					t.Fatal(err)
				}
				if event.PID != child.cmd.Process.Pid || event.Scenario != t.Name() || event.Epoch <= 0 {
					t.Fatal("wrong mixed event identity")
				}
				key := event.Boundary + ":" + event.Subject
				if _, exists := completed[key]; exists {
					t.Fatal("duplicate mixed event")
				}
				switch key {
				case "execution.committed:caller":
					delivered = event
					during = snapshot(t, db)
					if projection(t, during, "caller").Data["value"].Text != "peer advanced while model held" {
						t.Fatal("caller did not consume peer")
					}
					if capacity == 1 && (len(during.Tables["adapter_attempts"]) != 1 || len(during.Tables["external_attempts"]) != 0) {
						t.Fatal("shared capacity bypassed")
					}
				case "adapter.settled:zlookup", "external.settled:zresource":
					if capacity != 2 || len(event.InFlight) == 0 {
						t.Fatal("fast result lacked held model")
					}
				default:
					t.Fatalf("unexpected mixed event %+v", event)
				}
				completed[key] = event
				emit(t, map[string]any{"type": "mixed-completion-boundary", "event": event})
				for _, providerEvent := range p.events(t) {
					if providerEvent.Ordinal == 1 && providerEvent.Kind == "responding" {
						t.Fatal("model responded before causal release")
					}
				}
				child.send(t, "c")
			}
			if capacity == 2 {
				held := snapshot(t, db)
				if len(held.Tables["adapter_results"]) != 1 || len(held.Tables["external_receipts"]) != 1 {
					t.Fatal("fast completions not durable before model release")
				}
				emit(t, map[string]any{"type": "fast-mixed-completions", "events": completed, "snapshot": held})
			}
			heldSnapshot := snapshot(t, db)
			var heldAttempt runtime.AdapterAttempt
			for _, row := range heldSnapshot.Tables["adapter_attempts"] {
				var a runtime.AdapterAttempt
				if err := json.Unmarshal([]byte(row["descriptor"].(string)), &a); err != nil {
					t.Fatal(err)
				}
				if a.Intent.Run.Subject == "amodel" {
					heldAttempt = a
				}
			}
			if accepted.PID != p.p.cmd.Process.Pid {
				t.Fatal("wrong held provider PID")
			}
			assertContact(t, heldAttempt, accepted)
			for key, event := range completed {
				if key != "execution.committed:caller" {
					found := false
					for _, id := range event.InFlight {
						if id == heldAttempt.AuthorizationID {
							found = true
						}
					}
					if !found {
						t.Fatal("exact held model missing from active set")
					}
				}
			}
			p.p.send(t, "1\n")
			child.finish(t)
			after := snapshot(t, db)
			for _, subject := range []string{"amodel", "caller", "peer", "zlookup", "zresource"} {
				if projection(t, after, subject).Status != "succeeded" {
					t.Fatal("incomplete " + subject)
				}
			}
			emit(t, map[string]any{"scenario": t.Name(), "capacity": capacity, "causal_order": []string{"provider durable accepted withheld", "peer committed", "caller consumed peer", "release model"}, "accepted": accepted, "peer_staged": staged, "peer_committed": committed, "caller_consumed": delivered, "during": during, "after": after, "provider_contacts": p.events(t)})
		})
	}
}
func TestRealModelLateEvidenceAfterSourceClosure(t *testing.T) {
	p := startProvider(t, true)
	child, db := startRunner(t, p, 2, true, []boundary.Event{{Boundary: "execution.staged", Subject: "amodel", Phase: "terminal"}, {Boundary: "execution.committed", Subject: "amodel", Phase: "terminal"}, {Boundary: "adapter.result-recorded", Subject: "amodel"}})
	staged := ackBoundary(t, child, "execution.staged", "amodel")
	accepted := p.accepted(t, 1)
	child.send(t, "c")
	closed := ackBoundary(t, child, "execution.committed", "amodel")
	if projection(t, snapshot(t, db), "amodel").Status != "succeeded" {
		t.Fatal("source not closed")
	}
	child.send(t, "c")
	p.p.send(t, "1\n")
	late := ackBoundary(t, child, "adapter.result-recorded", "amodel")
	known := snapshot(t, db)
	if late.Epoch != closed.Epoch || late.Sequence >= closed.Sequence || late.Intent == "" || late.Attempt == "" {
		t.Fatal("late evidence lost original correlation")
	}
	if projection(t, known, "amodel").Status != "succeeded" {
		t.Fatal("late result reopened source")
	}
	child.send(t, "c")
	child.finish(t)
	after := snapshot(t, db)
	if projection(t, after, "amodel").Status != "succeeded" {
		t.Fatal("settlement reopened source")
	}
	if _, err := runtime.Replay(t.Context(), db, runtime.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	emit(t, map[string]any{"scenario": t.Name(), "closure": "ordinary terminal success, no supervisory policy claim", "staged": staged, "accepted": accepted, "closed": closed, "late": late, "known_unsettled": known, "after": after, "provider_contacts": p.events(t)})
}
