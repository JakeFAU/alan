//go:build integration

package investigationtest

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

	"github.com/JakeFAU/alan/examples/investigation"
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
		buildDir, err = os.MkdirTemp("", "alan-phase1f-process-")
		if err != nil {
			buildError = err.Error()
			return
		}
		for _, target := range []string{"alan-investigate", "provider"} {
			args := []string{"build", "-buildvcs=false", "-o", filepath.Join(buildDir, target)}
			if target == "provider" {
				args = []string{"test", "-c", "-tags", "integration", "-buildvcs=false", "-o", filepath.Join(buildDir, target)}
			}
			if raceChildren {
				args = append(args, "-race")
			}
			if target == "provider" {
				args = append(args, "./internal/investigationtest")
			} else {
				args = append(args, "./cmd/alan-investigate")
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
	t.Log("INVESTIGATION " + string(raw))
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
	cmd := exec.CommandContext(t.Context(), filepath.Join(binaries(t), "alan-investigate"), args...)
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
	case <-time.After(watchdog()):
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
	case <-time.After(watchdog()):
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
		env = append(env, "ALAN_PROVIDER_HOLD=alpha")
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
func inspect(t *testing.T, db string) investigation.View {
	t.Helper()
	v, err := investigation.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

const corpusText = "alpha release supports Linux.\nbeta rollback requires version 2.\ngamma approval is recorded.\ndelta window is seven minutes.\n"
const initialCandidate = "Linux supported; rollback version 2."
const finalCandidate = "Linux supported; rollback version 2; approval recorded; seven minutes."

// Watchdogs bound infrastructure waits, not semantic progress. Race instrumented
// SQLite runs cost substantially more; every causal assertion still uses a pipe ack.
func watchdog() time.Duration {
	if raceChildren {
		return 3 * time.Minute
	}
	return 30 * time.Second
}
func initialize(t *testing.T, p *provider) (string, string) {
	t.Helper()
	dir := t.TempDir()
	db := filepath.Join(dir, "app.sqlite")
	corpus := filepath.Join(dir, "corpus")
	if err := os.Mkdir(corpus, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(corpus, "source.txt")
	if err := os.WriteFile(source, []byte(corpusText), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{"schema": "phase1f-corpus/v1", "documents": []any{map[string]any{"ID": "packet", "Path": "corpus/source.txt", "SHA256": fileHash(source), "Bytes": len(corpusText)}}}
	raw, _ := json.Marshal(manifest)
	path := filepath.Join(corpus, "manifest.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	invoke(t, false, "init", "--db", db, "--question", "What is supported and when do we roll back?", "--model", "controlled-requested-model", "--endpoint", p.endpoint, "--corpus-manifest", path, "--nested", "--max-revisions", "1", "--request-timeout", "2m")
	emit(t, map[string]any{"type": "fixture", "schema": "phase1f-process/v1", "fixture": "four-facts-review-revise-accept/v1", "manifest": manifest, "corpus": corpusText, "config": inspect(t, db).Config, "watchdog": watchdog().String()})
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
	return launch(t, filepath.Join(binaries(t), "alan-investigate"), []string{"run", "--db", db}, []string{"OPENAI_API_KEY=synthetic-process-fixture", "ALAN_BARRIER=" + string(raw)})
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
func attempt(t *testing.T, v investigation.View, id string) runtime.AdapterAttempt {
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
func assertSuccess(t *testing.T, v investigation.View) {
	t.Helper()
	if v.Result["answer"].Text != finalCandidate || v.Result["report_outcome"].Text != "publish" || v.Result["revision_count"].Number != 1 || v.ModelRequests != 13 || v.LookupRequests != 4 || v.Attempts != 17 || len(v.Unknown) != 0 {
		t.Fatalf("bad result: %v models=%d lookups=%d unknown=%v", v.Result, v.ModelRequests, v.LookupRequests, v.Unknown)
	}
}
func assertOffline(t *testing.T, p *provider, db string, before investigation.View) {
	t.Helper()
	events := p.events(t)
	ledgerSHA := fileHash(p.ledger)
	p.p.kill(t, map[string]any{"reason": "offline replay provider unavailable"})
	var replay runtime.ReplayReport
	if err := json.Unmarshal(invoke(t, false, "replay", "--db", db), &replay); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, p.events(t)) || ledgerSHA != fileHash(p.ledger) {
		t.Fatal("offline replay changed store or contacts")
	}
	if len(replay.Runs) != len(before.Runs) {
		t.Fatal("replay omitted or added participants")
	}
	seen := map[runtime.RunID]bool{}
	for _, r := range replay.Runs {
		if seen[r.ID] {
			t.Fatal("duplicate replay participant")
		}
		seen[r.ID] = true
		matched := false
		for _, want := range before.Runs {
			if r.ID == want.ID {
				matched = true
				if !reflect.DeepEqual(r.Projection, want.Projection) {
					t.Fatal("replay projection mismatch")
				}
			}
		}
		if !matched {
			t.Fatal("unknown replay participant")
		}
	}

	emit(t, map[string]any{"type": "offline-replay", "before": compact(before), "after": compact(after), "replay": replay, "provider_contacts": events, "provider_ledger_sha256_before": ledgerSHA, "provider_ledger_sha256_after": fileHash(p.ledger), "credentials": false, "provider_available": false, "source_available": false, "registry": false, "adapter_invocations_inferred": 0, "zero_basis": "registry-free replay API; unchanged complete store and independent provider log; no dynamic tool invocation counter"})
}

func assertContacts(t *testing.T, v investigation.View, events []providerEvent) {
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
			if a.Prepared.Request.Capability != investigation.ModelCapability {
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
func assertConsumed(t *testing.T, v investigation.View, intent string) {
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
