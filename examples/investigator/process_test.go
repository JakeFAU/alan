package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

// A subprocess runs the same command handler as the installed executable. Only
// the test executable inherits the boundary observer and its pause/ack pipes.
func TestInvestigatorProcess(t *testing.T) {
	raw := os.Getenv("ALAN_INVESTIGATOR_TEST_ARGS")
	if raw == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(raw), &args); err != nil || len(args) < 2 {
		t.Fatal("invalid child command", err)
	}
	ctx, err := boundary.FromEnvironment(t.Context(), args[1])
	if err == nil {
		err = run(ctx, args, os.Stdout)
	}
	if err != nil {
		t.Fatal(err)
	}
}

const processDeadline = 60 * time.Second

// The fresh owner and every read-only command use the actual production binary.
// The process being stopped alone needs the test-only inherited observer.
var binaryDir string
var investigatorBinary = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "alan-investigator-bin-")
	if err != nil {
		return "", err
	}
	binaryDir = dir
	path := filepath.Join(dir, "investigator")
	args := []string{"build", "-buildvcs=false", "-o", path}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "-race" && setting.Value == "true" {
				args = append(args, "-race")
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", append(args, ".")...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build investigator: %w: %s", err, output)
	}
	return path, nil
})

func TestMain(m *testing.M) {
	code := m.Run()
	if binaryDir != "" {
		os.RemoveAll(binaryDir)
	}
	os.Exit(code)
}

type investigatorChild struct {
	cmd          *exec.Cmd
	out          *os.File
	ack, release *os.File
	decoder      *json.Decoder
	done         chan error
	store, token string
	pid          int
	finished     bool
}

func startInvestigator(t *testing.T, args []string, filters ...boundary.Event) *investigatorChild {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), processDeadline)
	t.Cleanup(cancel)
	var cmd *exec.Cmd
	if len(filters) == 0 {
		bin, err := investigatorBinary()
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.CommandContext(ctx, bin, args...)
	} else {
		cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInvestigatorProcess$")
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	// No live user credential is required by, or passed to, the offline child.
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "OPENAI_API_KEY=", "ALAN_INVESTIGATOR_TEST_ARGS="+string(encoded), "GORACE=atexit_sleep_ms=0")
	out, err := os.CreateTemp(t.TempDir(), "process-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	cmd.Stdout, cmd.Stderr = out, out
	c := &investigatorChild{cmd: cmd, out: out, done: make(chan error, 1), store: args[1], token: filepath.Base(out.Name())}
	var inherited []*os.File
	if len(filters) != 0 {
		for i := range filters {
			filters[i].Scenario = t.Name()
			filters[i].Token = c.token
			filters[i].Store = c.store
		}
		raw, err := json.Marshal(filters)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		resumeRead, resumeWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { read.Close(); resumeWrite.Close() })
		c.ack, c.release, c.decoder = read, resumeWrite, json.NewDecoder(read)
		inherited = []*os.File{write, resumeRead}
		cmd.ExtraFiles = inherited
	}
	err = cmd.Start()
	for _, f := range inherited {
		f.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	c.pid = cmd.Process.Pid
	go func() { c.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !c.finished {
			cancel()
			<-c.done
		}
	})
	return c
}

func (c *investigatorChild) output(t *testing.T) []byte {
	t.Helper()
	out, err := os.ReadFile(c.out.Name())
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (c *investigatorChild) wait(t *testing.T) []byte {
	t.Helper()
	err := <-c.done
	c.finished = true
	if err != nil {
		t.Fatalf("child %d exited %v: %s", c.pid, err, c.output(t))
	}
	return c.output(t)
}

func (c *investigatorChild) next(t *testing.T) boundary.Event {
	t.Helper()
	if err := c.ack.SetReadDeadline(time.Now().Add(processDeadline)); err != nil {
		t.Fatal(err)
	}
	var e boundary.Event
	if err := c.decoder.Decode(&e); err != nil {
		t.Fatalf("missing acknowledgment from %d: %v; %s", c.pid, err, c.output(t))
	}
	if e.PID != c.pid || e.Store != c.store || e.Scenario != t.Name() || e.Token != c.token || e.Subject == "" || e.Generation != 1 || e.Sequence < 1 {
		t.Fatalf("cross-process or incomplete acknowledgment: %+v", e)
	}
	return e
}

func (c *investigatorChild) proceed(t *testing.T) {
	t.Helper()
	if _, err := io.WriteString(c.release, "c"); err != nil {
		t.Fatal(err)
	}
}

func (c *investigatorChild) kill(t *testing.T, proof string) {
	t.Helper()
	if err := c.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := <-c.done
	c.finished = true
	status, ok := c.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v %v", status, err)
	}
	t.Logf("SIGKILL pid=%d after %s", c.pid, proof)
}

func repeatedCut(name, phase string, count int) []boundary.Event {
	filters := make([]boundary.Event, count)
	for i := range filters {
		filters[i] = boundary.Event{Boundary: name, Phase: phase, Generation: 1}
	}
	return filters
}

func cliOutput(t *testing.T, args ...string) []byte {
	t.Helper()
	return startInvestigator(t, args).wait(t)
}

func retainedBytes(t *testing.T, db string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(db + suffix)
		if suffix == "-wal" && os.IsNotExist(err) {
			result[suffix] = []byte{}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		result[suffix] = data
	}
	return result
}
