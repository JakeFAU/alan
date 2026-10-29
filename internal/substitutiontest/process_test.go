//go:build integration

package substitutiontest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/effects"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func childTimeout() time.Duration {
	if raceEnabled {
		return 600 * time.Second
	}
	return 100 * time.Second
}

type process struct {
	cmd                                    *exec.Cmd
	ack, release, log, control, controlAck *os.File
	filter                                 boundary.Event
	stopped                                bool
}

func start(t *testing.T, path, template, cut, endpoint string, replay bool, controlled ...bool) *process {
	t.Helper()
	ar, aw, e := os.Pipe()
	must(t, e)
	rr, rw, e := os.Pipe()
	must(t, e)
	log, e := os.CreateTemp(t.TempDir(), "child")
	must(t, e)
	binary, e := os.Executable()
	must(t, e)
	args := []string{"-test.run=^TestSubstitutionProcessChild$", "-test.v", "-test.timeout=" + childTimeout().String()}
	exe := binary
	network := "not requested"
	if replay {
		if sandbox, e := exec.LookPath("sandbox-exec"); e == nil {
			exe = sandbox
			args = append([]string{"-p", "(version 1)(allow default)(deny network*)", binary}, args...)
			network = "macOS sandbox-exec deny network*"
		} else {
			network = "unavailable; absent registry/credential only"
		}
	}
	cmd := exec.Command(exe, args...)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "ALAN_I_") && !strings.HasPrefix(v, "ALAN_BARRIER=") && !strings.HasPrefix(v, "OPENAI_API_KEY=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "ALAN_I_DB="+path, "ALAN_I_TEMPLATE="+template, "ALAN_I_CUT="+cut, "ALAN_I_ENDPOINT="+endpoint)
	f := boundary.Event{Store: path, Scenario: t.Name(), Token: fmt.Sprintf("%s/%s/%d", t.Name(), cut, time.Now().UnixNano()), Boundary: "execution.committed"}
	switch cut {
	case "entry-admitted":
		f.Boundary = "invocation.admitted"
	case "outcome-retained", "published":
		f.Boundary = "invocation.result-recorded"
	case "effect-attempt":
		f.Boundary = "runner.waiting"
	case "effect-reserved":
		f.Boundary = "adapter.authorized"
	case "effect-commit":
		f.Boundary = "execution.committed"
	case "effect-response":
		f.Boundary = "adapter.response"
	}
	if cut != "" {
		raw, _ := json.Marshal(f)
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
	}
	if replay {
		cmd.Env = append(cmd.Env, "ALAN_I_REPLAY=1", "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1")
	}
	cmd.ExtraFiles = []*os.File{aw, rr}
	var controlW, controlR, closedR, closedW *os.File
	if len(controlled) > 0 && controlled[0] {
		controlR, controlW, e = os.Pipe()
		must(t, e)
		closedR, closedW, e = os.Pipe()
		must(t, e)
		cmd.ExtraFiles = append(cmd.ExtraFiles, controlR, closedW)
		cmd.Env = append(cmd.Env, "ALAN_I_CONTROL=1")
	}
	cmd.Stdout = log
	cmd.Stderr = log
	must(t, cmd.Start())
	aw.Close()
	rr.Close()
	p := &process{cmd: cmd, ack: ar, release: rw, log: log, filter: f, control: controlW, controlAck: closedR}
	if controlR != nil {
		controlR.Close()
		closedW.Close()
	}
	t.Cleanup(func() {
		if !p.stopped {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		ar.Close()
		rw.Close()
		log.Close()
		if controlW != nil {
			controlW.Close()
			closedR.Close()
		}
	})
	raw, e := os.ReadFile(binary)
	must(t, e)
	retain(t, fmt.Sprintf("start-%d", cmd.Process.Pid), map[string]any{"pid": cmd.Process.Pid, "argv": cmd.Args, "environment": selectedEnv(cmd.Env), "binary_sha256": fmt.Sprintf("%x", shaBytes(raw)), "filter": f, "network": network, "race_instrumented": raceEnabled})
	return p
}
func shaBytes(b []byte) [32]byte { return sha256.Sum256(b) }
func (p *process) acknowledgment(t *testing.T) boundary.Event {
	t.Helper()
	var e boundary.Event
	done := make(chan error, 1)
	go func() { done <- json.NewDecoder(p.ack).Decode(&e) }()
	select {
	case err := <-done:
		if err != nil {
			raw, _ := os.ReadFile(p.log.Name())
			t.Fatal(err, string(raw))
		}
	case <-time.After(childTimeout()):
		t.Fatal("boundary watchdog")
	}
	if e.PID != p.cmd.Process.Pid || e.Store != p.filter.Store || e.Token != p.filter.Token || e.Scenario != p.filter.Scenario || e.Boundary != p.filter.Boundary {
		t.Fatal("wrong acknowledgment", e)
	}
	retain(t, fmt.Sprintf("ack-%d", e.PID), e)
	return e
}
func (p *process) resume(t *testing.T) { _, e := p.release.Write([]byte{'c'}); must(t, e) }
func (p *process) wait(t *testing.T, kill bool) {
	t.Helper()
	if kill {
		must(t, p.cmd.Process.Signal(syscall.SIGKILL))
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case e := <-done:
		p.stopped = true
		status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		raw, _ := os.ReadFile(p.log.Name())
		retain(t, fmt.Sprintf("exit-%d", p.cmd.Process.Pid), map[string]any{"pid": p.cmd.Process.Pid, "exit_code": p.cmd.ProcessState.ExitCode(), "signal": status.Signal().String(), "output": string(raw)})
		if kill {
			if e == nil || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatal("not actual SIGKILL", e, status)
			}
		} else if e != nil {
			t.Fatal(e, string(raw))
		}
	case <-time.After(childTimeout()):
		t.Fatal("child watchdog")
	}
}
func hasCut(s runtime.Inspection, template, cut string, e boundary.Event) bool {
	subjects := map[string]bool{}
	for _, r := range s.Tables["composition_admission_decisions"] {
		a := decode[runtime.InvocationAdmission](r, "descriptor")
		if a.Invocation.TemplateID == template {
			for _, m := range a.Members {
				subjects[m.Subject] = true
			}
		}
	}
	switch cut {
	case "entry-admitted":
		return e.Subject == "root"
	case "outcome-retained", "published":
		return e.Subject == "root"
	case "effect-attempt", "effect-reserved", "effect-response":
		return true
	}
	if e.Boundary != "execution.committed" || e.Intent == "" {
		return false
	}
	for _, r := range s.Tables["intents"] {
		i := decode[intent](r, "descriptor")
		if i.ID != e.Intent {
			continue
		}
		switch cut {
		case "entry-commit":
			return i.Run.Subject == "root" && i.Kind == "invocation"
		case "helper-commit":
			return subjects[i.Run.Subject] && i.Kind == "invocation"
		case "reply-commit":
			return subjects[i.Run.Subject] && i.Kind == "message"
		case "effect-commit":
			return i.Kind == "adapter"
		}
	}
	return false
}
func TestSubstitutionProcessChild(t *testing.T) {
	path := os.Getenv("ALAN_I_DB")
	if path == "" {
		t.Skip("subprocess entry")
	}
	ctx, e := boundary.FromEnvironment(t.Context(), path)
	must(t, e)
	if os.Getenv("ALAN_I_REPLAY") == "1" {
		r, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
		must(t, e)
		t.Logf("replay %+v", r)
		return
	}
	if os.Getenv("ALAN_I_CONTROL") == "1" {
		inherited := ctx
		closed := os.NewFile(6, "closure-ack")
		defer closed.Close()
		ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			if e.Boundary == "lifecycle.decided" {
				e.PID = os.Getpid()
				if err := json.NewEncoder(closed).Encode(e); err != nil {
					return err
				}
			}
			return boundary.Reached(inherited, e)
		})
	}
	cut, template := os.Getenv("ALAN_I_CUT"), os.Getenv("ALAN_I_TEMPLATE")
	if cut != "" {
		inherited := ctx
		ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			if e.Boundary == "lifecycle.decided" {
				return boundary.Reached(inherited, e)
			}
			if e.Boundary == "adapter.authorized" || e.Boundary == "adapter.finished" {
				if cut != "effect-reserved" {
					return boundary.Reached(inherited, e)
				}
			}
			if e.Boundary != processBoundary(cut) {
				return nil
			}
			s, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				return err
			}
			if hasCut(s, template, cut, e) {
				return boundary.Reached(inherited, e)
			}
			return nil
		})
	}
	s, e := runtime.Open(ctx, path, runtime.DefaultLimits())
	must(t, e)
	defer s.Close()
	if os.Getenv("ALAN_I_CONTROL") == "1" {
		trigger := os.NewFile(5, "cancel-input")
		defer trigger.Close()
		go func() {
			var signal [1]byte
			if _, err := trigger.Read(signal[:]); err == nil && signal[0] == 'c' {
				_, _ = s.AdmitInput(ctx, runtime.Input{TransportID: "cancel-operator", LogicalID: "cancel-operator", Target: "operator", Generation: 1, Obligation: "cancel", Payload: map[string]schema.Value{}})
			}
		}()
	}
	registry := map[string]effects.Adapter{}
	if endpoint := os.Getenv("ALAN_I_ENDPOINT"); endpoint != "" {
		registry[app.EffectCapability] = httpAdapter{endpoint}
	}
	r, e := s.Run(ctx, runtime.RunOptions{Adapters: registry, MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1, ExternalTimeout: childTimeout()})
	must(t, e)
	if !r.Quiescent {
		t.Fatal("bounds exhausted", r)
	}
	t.Logf("run %+v", r)
}
func processBoundary(cut string) string {
	switch cut {
	case "entry-admitted":
		return "invocation.admitted"
	case "published", "outcome-retained":
		return "invocation.result-recorded"
	case "effect-attempt":
		return "runner.waiting"
	case "effect-reserved":
		return "adapter.authorized"
	case "effect-response":
		return "adapter.response"
	default:
		return "execution.committed"
	}
}
func stablePrefix(t *testing.T, before, after runtime.Inspection) {
	t.Helper()
	for _, table := range []string{"intents", "composition_charges", "composition_admission_decisions", "composition_replies", "composition_publications", "composition_reservations", "adapter_attempts"} {
		for _, old := range before.Tables[table] {
			found := false
			for _, row := range after.Tables[table] {
				if reflect.DeepEqual(old, row) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("original %s row lost", table)
			}
		}
	}
	for _, old := range before.Tables["observations"] {
		found := false
		for _, row := range after.Tables["observations"] {
			if row["id"] == old["id"] && row["descriptor"] == old["descriptor"] {
				found = true
			}
		}
		if !found {
			t.Fatal("observation identity lost")
		}
	}
}
func runRecovery(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, cut := range []string{"entry-commit", "outcome-retained"} {
			t.Run(name+"/"+cut, func(t *testing.T) {
				q := app.Request{ID: 2, Value: 1, Mode: "answer"}
				l := limits(3, 3, 3)
				d, path := provision(t, name, "bare", q, l)
				create(t, d, path)
				p := start(t, path, d.Candidate.ID(), cut, "", false)
				ack := p.acknowledgment(t)
				before := inspect(t, path)
				retain(t, "before", before)
				if cut == "entry-commit" && len(before.Tables["composition_admission_decisions"]) != 0 {
					t.Fatal("entry already admitted")
				}
				if cut == "outcome-retained" && (len(before.Tables["composition_publications"]) == 0 || projection(before, "root").Status != "open") {
					t.Fatal("wrong retained outcome cut")
				}
				p.wait(t, true)
				next := start(t, path, d.Candidate.ID(), "", "", false)
				if next.cmd.Process.Pid == ack.PID {
					t.Fatal("PID reused")
				}
				next.wait(t, false)
				after := inspect(t, path)
				stablePrefix(t, before, after)
				n := checked(t, after, d, q, "bare", name, l)
				if projection(after, "root").Status != "succeeded" {
					t.Fatal("no successful continuation")
				}
				var eb, ea int64
				for _, r := range before.Tables["runs"] {
					if r["subject"] == "root" {
						eb = r["epoch"].(int64)
					}
				}
				for _, r := range after.Tables["runs"] {
					if r["subject"] == "root" {
						ea = r["epoch"].(int64)
					}
				}
				if ea <= eb {
					t.Fatal("activation epoch not renewed")
				}
				retain(t, "after", map[string]any{"inspection": after, "normalized": n, "before_root_epoch": eb, "after_root_epoch": ea})
				replay := start(t, path, "", "", "", true)
				replay.wait(t, false)
				if !reflect.DeepEqual(after, inspect(t, path)) {
					t.Fatal("replay mutated ledger")
				}
			})
		}
	}
}
func selectedEnv(env []string) []string {
	out := []string{}
	for _, v := range env {
		for _, prefix := range []string{"ALAN_I_", "ALAN_BARRIER=", "GOTOOLCHAIN=", "GOPROXY=", "GOSUMDB=", "GOFLAGS=", "GOCACHE="} {
			if strings.HasPrefix(v, prefix) {
				out = append(out, v)
				break
			}
		}
	}
	return out
}
func (p *process) cancelLive(t *testing.T) {
	t.Helper()
	_, e := p.control.Write([]byte{'c'})
	must(t, e)
	var ack boundary.Event
	done := make(chan error, 1)
	go func() { done <- json.NewDecoder(p.controlAck).Decode(&ack) }()
	select {
	case e := <-done:
		must(t, e)
	case <-time.After(childTimeout()):
		t.Fatal("live closure ACK absent")
	}
	if ack.PID != p.cmd.Process.Pid || ack.Boundary != "lifecycle.decided" || ack.Subject != "operator" {
		t.Fatal("wrong live lifecycle acknowledgment", ack)
	}
	retain(t, "live-closure-ack", ack)
}
