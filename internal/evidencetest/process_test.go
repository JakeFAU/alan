//go:build integration

package evidencetest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
)

var buildOnce sync.Once
var buildDir, buildError string

func binaries(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		var err error
		buildDir, err = os.MkdirTemp("", "alan-phase1e-process-")
		if err != nil {
			buildError = err.Error()
			return
		}
		for _, target := range []string{"alan-evidence", "provider"} {
			args := []string{"build", "-buildvcs=false", "-o", filepath.Join(buildDir, target)}
			if target == "provider" {
				args = []string{"test", "-c", "-tags", "integration", "-buildvcs=false", "-o", filepath.Join(buildDir, target)}
			}
			if raceChildren {
				args = append(args, "-race")
			}
			if target == "provider" {
				args = append(args, "./internal/evidencetest")
			} else {
				args = append(args, "./cmd/alan-evidence")
			}
			cmd := exec.CommandContext(t.Context(), "go", args...)
			cmd.Dir = "../.."
			out, err := cmd.CombinedOutput()
			if err != nil {
				buildError = fmt.Sprintf("%v: %v %s", args, err, out)
				return
			}
			emit(t, map[string]any{"type": "build", "command": cmd.Args, "race_children": raceChildren, "binary_sha256": fileHash(filepath.Join(buildDir, target))})
		}
	})
	if buildError != "" {
		t.Fatal(buildError)
	}
	return buildDir
}
func emit(t *testing.T, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("EVIDENCE " + string(raw))
}
func fileHash(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func cleanEnv() []string {
	var out []string
	for _, s := range os.Environ() {
		if !strings.HasPrefix(s, "OPENAI_API_KEY=") && !strings.HasPrefix(s, "ALAN_BARRIER=") && !strings.HasPrefix(s, "ALAN_TEST_PROVIDER=") {
			out = append(out, s)
		}
	}
	return out
}
func invoke(t *testing.T, secret bool, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), filepath.Join(binaries(t), "alan-evidence"), args...)
	cmd.Env = cleanEnv()
	if secret {
		cmd.Env = append(cmd.Env, "OPENAI_API_KEY=synthetic-process-fixture")
	}
	out, err := cmd.CombinedOutput()
	emit(t, map[string]any{"type": "cli", "command": cmd.Args, "pid": cmd.ProcessState.Pid(), "exit_code": cmd.ProcessState.ExitCode(), "output": string(out), "race_children": raceChildren, "credential": "synthetic fixture supplied only when running"})
	if err != nil {
		t.Fatalf("CLI %v: %v %s", args, err, out)
	}
	return out
}

type process struct {
	expected          []boundary.Event
	used              map[string]bool
	cmd               *exec.Cmd
	ack, release, log *os.File
	decoder           *json.Decoder
	stopped           bool
}

func launch(t *testing.T, binary string, args, env []string) *process {
	t.Helper()
	ackR, ackW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	relR, relW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.CreateTemp(binaries(t), "process-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, args...)
	cmd.Env = append(cleanEnv(), env...)
	cmd.ExtraFiles = []*os.File{ackW, relR}
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ackW.Close()
	relR.Close()
	p := &process{cmd: cmd, ack: ackR, release: relW, log: log, decoder: json.NewDecoder(ackR)}
	p.used = map[string]bool{}
	for _, item := range env {
		if raw, ok := strings.CutPrefix(item, "ALAN_BARRIER="); ok {
			if err := json.Unmarshal([]byte(raw), &p.expected); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Cleanup(func() {
		if !p.stopped {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		ackR.Close()
		relW.Close()
		log.Close()
	})
	emit(t, map[string]any{"type": "process-start", "command": cmd.Args, "pid": cmd.Process.Pid, "binary_sha256": fileHash(binary), "race_children": raceChildren})
	return p
}
func (p *process) next(t *testing.T, v any) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- p.decoder.Decode(v) }()
	select {
	case err := <-done:
		if err != nil {
			raw, _ := os.ReadFile(p.log.Name())
			t.Fatalf("ack: %v %s", err, raw)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("watchdog: missing process acknowledgment")
	}
}
func (p *process) kill(t *testing.T, ack any) {
	t.Helper()
	if err := p.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := p.cmd.Wait()
	p.stopped = true
	status, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("not SIGKILL: %v", err)
	}
	raw, _ := os.ReadFile(p.log.Name())
	emit(t, map[string]any{"type": "process-exit", "pid": p.cmd.Process.Pid, "signal": status.Signal().String(), "ack": ack, "output": string(raw)})
}
func (p *process) finish(t *testing.T) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case err := <-done:
		p.stopped = true
		raw, _ := os.ReadFile(p.log.Name())
		emit(t, map[string]any{"type": "process-exit", "pid": p.cmd.Process.Pid, "exit_code": p.cmd.ProcessState.ExitCode(), "output": string(raw)})
		if err != nil {
			t.Fatalf("child: %v %s", err, raw)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("watchdog: child did not exit")
	}
}
func (p *process) send(t *testing.T, s string) {
	t.Helper()
	if _, err := io.WriteString(p.release, s); err != nil {
		t.Fatal(err)
	}
}

type provider struct {
	p                *process
	endpoint, ledger string
}

func startProvider(t *testing.T, hold bool) *provider {
	t.Helper()
	dir := t.TempDir()
	ledger := filepath.Join(dir, "contacts.jsonl")
	env := []string{"ALAN_TEST_PROVIDER=1", "ALAN_PROVIDER_LEDGER=" + ledger}
	if hold {
		env = append(env, "ALAN_PROVIDER_HOLD=first")
	}
	p := launch(t, filepath.Join(binaries(t), "provider"), []string{"-test.run=^TestProviderChild$", "-test.v"}, env)
	var e providerEvent
	p.next(t, &e)
	if e.Kind != "ready" || e.PID != p.cmd.Process.Pid {
		t.Fatal("bad provider ready")
	}
	emit(t, map[string]any{"type": "provider-ready", "event": e})
	return &provider{p, e.Endpoint, ledger}
}
func (p *provider) events(t *testing.T) []providerEvent {
	t.Helper()
	raw, err := os.ReadFile(p.ledger)
	if err != nil {
		t.Fatal(err)
	}
	var out []providerEvent
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	for {
		var e providerEvent
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}
func (p *provider) accepted(t *testing.T, n int) providerEvent {
	t.Helper()
	for {
		var e providerEvent
		p.p.next(t, &e)
		emit(t, map[string]any{"type": "provider-event", "event": e})
		if e.Kind == "accepted" && e.Ordinal == n {
			return e
		}
	}
}
func contacts(es []providerEvent) int {
	n := 0
	for _, e := range es {
		if e.Kind == "accepted" {
			n++
		}
	}
	return n
}
func inspect(t *testing.T, db string) evidence.View {
	t.Helper()
	v, err := evidence.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func initialize(t *testing.T, p *provider) (string, string) {
	t.Helper()
	dir := t.TempDir()
	db, corpus := filepath.Join(dir, "app.sqlite"), filepath.Join(dir, "source.txt")
	if err := os.WriteFile(corpus, []byte("WAL retains a durable log.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	invoke(t, false, "init", "--db", db, "--question", "What does WAL retain?", "--model", "controlled-requested-model", "--endpoint", p.endpoint, "--corpus", corpus, "--request-timeout", "2m")
	return db, corpus
}
func paused(t *testing.T, db string, filters ...boundary.Event) *process {
	t.Helper()
	for i := range filters {
		filters[i].Store = db
		filters[i].Scenario = t.Name()
		filters[i].Token = fmt.Sprintf("%s-%d", t.Name(), i)
	}
	raw, _ := json.Marshal(filters)
	emit(t, map[string]any{"type": "barrier-config", "filters": filters})
	return launch(t, filepath.Join(binaries(t), "alan-evidence"), []string{"run", "--db", db}, []string{"OPENAI_API_KEY=synthetic-process-fixture", "ALAN_BARRIER=" + string(raw)})
}
func ackBoundary(t *testing.T, p *process, name, subject string) boundary.Event {
	t.Helper()
	var e boundary.Event
	p.next(t, &e)
	if err := validateAcknowledgment(e, p.expected, p.cmd.Process.Pid, p.used); err != nil {
		t.Fatal(err)
	}
	if e.PID != p.cmd.Process.Pid || e.Boundary != name || e.Subject != subject || e.Scenario != t.Name() || e.Store == "" || e.Token == "" || e.Epoch <= 0 || e.Generation != 1 || e.Sequence <= 0 {
		t.Fatalf("mismatched barrier %+v", e)
	}
	emit(t, map[string]any{"type": "boundary", "event": e})
	return e
}
func attempt(t *testing.T, v evidence.View, id string) runtime.AdapterAttempt {
	t.Helper()
	for _, row := range v.Snapshot.Tables["adapter_attempts"] {
		var a runtime.AdapterAttempt
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &a); err != nil {
			t.Fatal(err)
		}
		if a.Prepared.Request.ID == id {
			return a
		}
	}
	t.Fatalf("missing attempt %s", id)
	return runtime.AdapterAttempt{}
}
func assertContact(t *testing.T, a runtime.AdapterAttempt, e providerEvent) {
	t.Helper()
	digest := sha256.Sum256([]byte(a.Prepared.Request.Key))
	want := fmt.Sprintf("alan-%x", digest)
	if e.Wire != a.Prepared.Wire || e.WireDigest != fmt.Sprintf("%x", sha256.Sum256([]byte(a.Prepared.Wire))) || e.ClientID != want || !a.Allowed {
		t.Fatalf("contact correlation: %+v requestID=%s digest=%s client=%s want=%s", e, a.Prepared.Request.ID, a.Prepared.Digest, e.ClientID, want)
	}
}
func assertSuccess(t *testing.T, v evidence.View) {
	t.Helper()
	if v.Result["answer"].Text != "WAL retains a durable log." || v.ModelRequests != 2 || v.LookupRequests != 1 || v.Attempts != 3 || len(v.Unknown) != 0 {
		t.Fatalf("bad result %+v", v)
	}
	for _, r := range v.Runs {
		if r.Projection.Status != "succeeded" {
			t.Fatalf("unfinished %s", r.ID.Subject)
		}
	}
}
func assertOffline(t *testing.T, p *provider, db string, before evidence.View) {
	t.Helper()
	events := p.events(t)
	p.p.kill(t, map[string]any{"reason": "offline replay provider unavailable"})
	var replay runtime.ReplayReport
	if err := json.Unmarshal(invoke(t, false, "replay", "--db", db), &replay); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, p.events(t)) {
		t.Fatal("offline replay changed store or contacts")
	}
	for _, r := range replay.Runs {
		for _, want := range before.Runs {
			if r.ID == want.ID && !reflect.DeepEqual(r.Projection, want.Projection) {
				t.Fatal("replay projection mismatch")
			}
		}
	}
	emit(t, map[string]any{"type": "offline-replay", "before": before, "after": after, "replay": replay, "provider_contacts": events, "credentials": false, "provider_available": false, "source_available": false, "registry": false, "adapter_invocations_inferred": 0, "zero_basis": "registry-free replay API; unchanged complete store and independent provider log; no dynamic tool invocation counter"})
}

func TestRecordedModelAndLookupCrashRecovery(t *testing.T) {
	for _, tc := range []struct{ name, subject string }{{"C-model", "assistant"}, {"D-lookup", "assistant/retrieval"}} {
		t.Run(tc.name, func(t *testing.T) {
			p := startProvider(t, false)
			db, corpus := initialize(t, p)
			before := inspect(t, db)
			filters := []boundary.Event{{Boundary: "adapter.result-recorded", Phase: "pre-settlement", Subject: tc.subject}}
			if tc.name == "D-lookup" {
				filters = append(filters, boundary.Event{Boundary: "adapter.finished", Phase: "transport-ended", Subject: tc.subject})
			}
			c := paused(t, db, filters...)
			if tc.name == "D-lookup" {
				executed := ackBoundary(t, c, "adapter.finished", tc.subject)
				emit(t, map[string]any{"type": "lookup-executed", "event": executed})
				c.send(t, "c")
			}
			ack := ackBoundary(t, c, "adapter.result-recorded", tc.subject)
			stopped := inspect(t, db)
			if ack.Intent == "" || ack.Attempt == "" {
				t.Fatal("missing operation identity")
			}
			a := attempt(t, stopped, ack.Intent)
			if err := validateDurableBoundary(stopped.Snapshot, ack); err != nil {
				t.Fatal(err)
			}
			if a.AuthorizationID != ack.Attempt {
				t.Fatal("wrong authorization")
			}
			if stopped.ModelRequests != 1 || (tc.name == "C-model" && stopped.LookupRequests != 0) || (tc.name == "D-lookup" && stopped.LookupRequests != 1) {
				t.Fatal("successor effect advanced before boundary")
			}
			found := false
			for _, r := range stopped.Results {
				if r.Candidate.ID == ack.Intent {
					found = true
				}
			}
			if !found {
				t.Fatal("result not durable")
			}
			for _, row := range stopped.Snapshot.Tables["settlements"] {
				if row["intent_id"] == ack.Intent {
					t.Fatal("result consumed before selected boundary")
				}
			}
			c.kill(t, ack)
			if err := os.Remove(corpus); err != nil {
				t.Fatal(err)
			}
			// A barrier on lookup authorization independently counts its actual Execute
			// entry on C; D already executed it and must never hit this barrier again.
			if tc.name == "C-model" {
				r := paused(t, db, boundary.Event{Boundary: "adapter.finished", Phase: "transport-ended", Subject: "assistant/retrieval"}, boundary.Event{Boundary: "adapter.authorized", Intent: ack.Intent})
				lookupAck := ackBoundary(t, r, "adapter.finished", "assistant/retrieval")
				r.send(t, "c")
				r.finish(t)
				emit(t, map[string]any{"type": "lookup-executed", "event": lookupAck})
			} else {
				r := paused(t, db, boundary.Event{Boundary: "adapter.authorized", Intent: ack.Intent})
				r.finish(t)
				var unexpected boundary.Event
				if err := r.decoder.Decode(&unexpected); err != io.EOF {
					t.Fatalf("recorded lookup reinvoked: %v %+v", err, unexpected)
				}
				emit(t, map[string]any{"type": "lookup-recovery-invocations", "intent": ack.Intent, "count": 0})
			}
			after := inspect(t, db)
			assertSuccess(t, after)
			if before.Snapshot.Namespace != after.Snapshot.Namespace {
				t.Fatal("namespace changed on recovery")
			}
			pending := "pending_plan"
			if tc.name == "D-lookup" {
				pending = "pending_lookup"
			}
			for _, run := range after.Runs {
				if run.ID.Subject == tc.subject && run.Projection.Data[pending].Text != ack.Intent {
					t.Fatal("pending operation identity reset")
				}
			}
			if !reflect.DeepEqual(a, attempt(t, after, ack.Intent)) {
				t.Fatal("recorded attempt changed")
			}
			assertContacts(t, after, p.events(t))
			if contacts(p.events(t)) != 2 {
				t.Fatal("historical model repeated")
			}
			assertConsumed(t, after, ack.Intent)
			emit(t, map[string]any{"scenario": t.Name(), "before": before, "stopped": stopped, "after": after, "barrier": ack, "provider_contacts": p.events(t), "source_deleted": corpus, "fixture": "durable-controlled-http", "live": "NOT RUN"})
			assertOffline(t, p, db, after)
		})
	}
}

func TestAcceptedUnknownCrashDoesNotRepeat(t *testing.T) {
	p := startProvider(t, true)
	db, corpus := initialize(t, p)
	c := paused(t, db, boundary.Event{Boundary: "adapter.authorized", Phase: "pre-transport", Subject: "assistant"})
	ack := ackBoundary(t, c, "adapter.authorized", "assistant")
	c.send(t, "c")
	accepted := p.accepted(t, 1)
	if accepted.PID != p.p.cmd.Process.Pid {
		t.Fatal("provider process mismatch")
	}
	stopped := inspect(t, db)
	if err := validateDurableBoundary(stopped.Snapshot, ack); err != nil {
		t.Fatal(err)
	}
	a := attempt(t, stopped, ack.Intent)
	assertContact(t, a, accepted)
	if len(stopped.Results) != 0 || len(stopped.Unknown) != 1 || stopped.Unknown[0] != ack.Intent {
		t.Fatal("accepted outcome must be unknown")
	}
	c.kill(t, accepted)
	if err := os.Remove(corpus); err != nil {
		t.Fatal(err)
	}
	invoke(t, true, "recover", "--db", db)
	after := inspect(t, db)
	if after.ModelRequests != 1 || after.LookupRequests != 0 || after.Attempts != 1 || !reflect.DeepEqual(stopped.Unknown, after.Unknown) || !reflect.DeepEqual(a, attempt(t, after, ack.Intent)) || contacts(p.events(t)) != 1 {
		t.Fatal("unknown operation repeated or reset")
	}
	for _, want := range stopped.Runs {
		for _, got := range after.Runs {
			if want.ID == got.ID && !reflect.DeepEqual(want.Projection, got.Projection) {
				t.Fatal("unknown projection/budget/pending changed")
			}
		}
	}
	emit(t, map[string]any{"scenario": t.Name(), "barrier": ack, "accepted": accepted, "stopped": stopped, "after": after, "provider_contacts": p.events(t), "disposition": "Unknown; no automatic repeat"})
	assertOffline(t, p, db, after)
}

func assertContacts(t *testing.T, v evidence.View, events []providerEvent) {
	t.Helper()
	seen := map[string]bool{}
	for _, e := range events {
		if e.Kind != "accepted" {
			continue
		}
		if seen[e.ClientID] {
			t.Fatal("duplicate provider client request identity")
		}
		seen[e.ClientID] = true
		found := false
		for _, row := range v.Snapshot.Tables["adapter_attempts"] {
			var a runtime.AdapterAttempt
			if err := json.Unmarshal([]byte(row["descriptor"].(string)), &a); err != nil {
				t.Fatal(err)
			}
			if a.Prepared.Request.Capability != evidence.ModelCapability {
				continue
			}
			sum := sha256.Sum256([]byte(a.Prepared.Request.Key))
			if e.ClientID == fmt.Sprintf("alan-%x", sum) {
				assertContact(t, a, e)
				found = true
			}
		}
		if !found {
			t.Fatal("provider contact lacks exact durable attempt")
		}
	}
}
func assertConsumed(t *testing.T, v evidence.View, intent string) {
	t.Helper()
	obs := ""
	for _, r := range v.Snapshot.Tables["settlements"] {
		if r["intent_id"] == intent {
			obs, _ = r["observation_id"].(string)
		}
	}
	if obs == "" {
		t.Fatal("recorded result not settled")
	}
	for _, r := range v.Snapshot.Tables["observations"] {
		if r["id"] == obs && r["consumed_sequence"] != nil {
			return
		}
	}
	t.Fatal("recorded result not ordinarily consumed")
}

func TestCancellationRetainsRealReceivedModelCandidate(t *testing.T) {
	p := startProvider(t, false)
	db, corpus := initialize(t, p)
	child := paused(t, db, boundary.Event{Boundary: "adapter.response", Phase: "pre-record", Subject: "assistant"})
	ack := ackBoundary(t, child, "adapter.response", "assistant")
	before := inspect(t, db)
	if len(before.Results) != 0 || len(before.Unknown) != 1 {
		t.Fatal("candidate must not already be recorded")
	}
	if err := child.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- child.cmd.Wait() }()
	select {
	case err := <-done:
		child.stopped = true
		if err == nil || child.cmd.ProcessState.ExitCode() != 1 {
			t.Fatalf("expected canceled CLI exit 1: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("watchdog: cancellation did not drain child")
	}
	raw, err := os.ReadFile(child.log.Name())
	if err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	if len(after.Results) != 1 || after.Results[0].Candidate.ID != ack.Intent || len(after.Unknown) != 0 || len(after.Snapshot.Tables["settlements"]) != 1 {
		t.Fatal("received candidate lost during owner cancellation")
	}
	if attempt(t, after, ack.Intent).AuthorizationID != ack.Attempt {
		t.Fatal("cancellation lost original authorization")
	}
	if err := os.Remove(corpus); err != nil {
		t.Fatal(err)
	}
	invoke(t, true, "recover", "--db", db)
	final := inspect(t, db)
	assertSuccess(t, final)
	assertContacts(t, final, p.events(t))
	if contacts(p.events(t)) != 2 {
		t.Fatal("cancel recovery repeated original model")
	}
	emit(t, map[string]any{"scenario": t.Name(), "injection": "external controller SIGTERM; no supervisory policy claim", "barrier": ack, "pid": child.cmd.Process.Pid, "signal": "SIGTERM", "exit_code": 1, "output": string(raw), "before": before, "after_cancel": after, "after_recovery": final, "provider_contacts": p.events(t)})
}

// validateAcknowledgment authenticates controller correlation, not a security
// signature. Every selected field, token, process and store must match exactly.
func validateAcknowledgment(got boundary.Event, filters []boundary.Event, pid int, used map[string]bool) error {
	for _, want := range filters {
		if got.Token != want.Token {
			continue
		}
		if used[got.Token] || got.PID != pid || got.Store != want.Store || got.Scenario != want.Scenario || got.Boundary != want.Boundary || got.Epoch <= 0 || got.Generation <= 0 || got.Sequence <= 0 {
			return fmt.Errorf("wrong acknowledgment identity: %+v", got)
		}
		if (want.Subject != "" && got.Subject != want.Subject) || (want.Phase != "" && got.Phase != want.Phase) || (want.Intent != "" && got.Intent != want.Intent) || (want.Attempt != "" && got.Attempt != want.Attempt) || (want.Epoch != 0 && got.Epoch != want.Epoch) || (want.Generation != 0 && got.Generation != want.Generation) || (want.Sequence != 0 && got.Sequence != want.Sequence) {
			return fmt.Errorf("wrong acknowledgment operation: %+v", got)
		}
		used[got.Token] = true
		return nil
	}
	return fmt.Errorf("unrequested acknowledgment token: %+v", got)
}

// validateDurableBoundary binds wildcard observation fields to the retained
// authorization and exact committed adapter operation, including its sequence.
func validateDurableBoundary(s runtime.Inspection, e boundary.Event) error {
	authorized := false
	for _, row := range s.Tables["attempts"] {
		var a executor.Authorization
		if err := json.Unmarshal([]byte(row["result"].(string)), &a); err != nil {
			return err
		}
		if a.ID != e.Attempt {
			continue
		}
		if !a.Allowed || a.IntentID != e.Intent || a.Source.Run.Subject != e.Subject || a.Source.Run.Generation != e.Generation || a.Source.Epoch != e.Epoch {
			return fmt.Errorf("boundary not bound to durable source authority")
		}
		authorized = true
		break
	}
	if !authorized {
		return fmt.Errorf("boundary authorization absent")
	}
	var retained *executor.Intent
	for _, row := range s.Tables["adapter_attempts"] {
		var a runtime.AdapterAttempt
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &a); err != nil {
			return err
		}
		if a.Intent.ID != e.Intent {
			continue
		}
		if a.AuthorizationID != e.Attempt || a.Prepared.Request.ID != e.Intent || a.Intent.Run.Subject != e.Subject || a.Intent.Run.Generation != e.Generation {
			return fmt.Errorf("boundary not bound to durable adapter attempt")
		}
		if a.Intent.Sequence != e.Sequence {
			return fmt.Errorf("boundary sequence %d differs from retained adapter intent sequence %d", e.Sequence, a.Intent.Sequence)
		}
		retained = &a.Intent
		break
	}
	if retained == nil {
		return fmt.Errorf("boundary adapter attempt absent")
	}
	for _, row := range s.Tables["intents"] {
		var intent executor.Intent
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &intent); err != nil {
			return err
		}
		if intent.ID != e.Intent {
			continue
		}
		if intent.Sequence != e.Sequence {
			return fmt.Errorf("boundary sequence %d differs from committed intent sequence %d", e.Sequence, intent.Sequence)
		}
		if !reflect.DeepEqual(intent, *retained) {
			return fmt.Errorf("boundary adapter attempt differs from committed intent")
		}
		return nil
	}
	return fmt.Errorf("boundary committed intent absent")
}
