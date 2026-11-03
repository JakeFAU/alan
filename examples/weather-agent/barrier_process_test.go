package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

// The only instrumented process is the one being stopped at an internal
// transaction boundary. Its command handler is exactly the application's.
// External acceptance and all ordinary run/resume/inspect/replay commands use
// the independently compiled, uninstrumented consumer binary.
func TestWeatherBoundaryOwner(t *testing.T) {
	db, action := os.Getenv("ALAN_WEATHER_CUT_DB"), os.Getenv("ALAN_WEATHER_CUT_COMMAND")
	if db == "" {
		return
	}
	ctx, err := boundary.FromEnvironment(t.Context(), db)
	if err == nil {
		args := []string{action, "--db", db, "--json"}
		if strings.HasPrefix(action, "approve-") {
			n := strings.TrimPrefix(action, "approve-")
			if n == "second" {
				n = "2"
			}
			args = []string{"refresh", "--db", db, "--read", n, "--json"}
		}
		err = run(ctx, args, os.Stdout)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// This is the controller's independent view of the inherited pipe protocol.
// Filters are causal. The deadlines only bound an unresponsive child.
type weatherCut struct {
	Scenario, Token, Store, Boundary, Phase, Subject string
	PID                                              int
	Generation, Epoch, Sequence                      int64
	Intent, Attempt                                  string
}

type weatherOwner struct {
	cmd    *exec.Cmd
	cut    *weatherCut
	ack    *os.File
	output string
	done   chan error
	reaped bool
}

func launchWeather(t *testing.T, binary, db, action string, filter *weatherCut) *weatherOwner {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), weatherProcessTimeout)
	t.Cleanup(cancel)
	dir := t.TempDir()
	out, err := os.Create(filepath.Join(dir, "owner.txt"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	cmd := exec.CommandContext(ctx, binary, action, "--db", db, "--json")
	if filter != nil {
		cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWeatherBoundaryOwner$")
	}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0", "ALAN_WEATHER_CUT_DB="+db, "ALAN_WEATHER_CUT_COMMAND="+action)
	c := &weatherOwner{cmd: cmd, output: out.Name(), done: make(chan error, 1)}
	if filter != nil {
		f := *filter
		f.Store, f.Token, f.Scenario = db, dir, t.Name()
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		releaseRead, releaseWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		c.cut, c.ack = &f, read
		t.Cleanup(func() { read.Close(); releaseWrite.Close() })
		raw, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
		cmd.ExtraFiles = []*os.File{write, releaseRead}
	}
	err = cmd.Start()
	for _, f := range cmd.ExtraFiles {
		f.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	go func() { c.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !c.reaped {
			cancel()
			<-c.done
		}
	})
	return c
}

func (c *weatherOwner) readOutput(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(c.output)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func (c *weatherOwner) wait(t *testing.T) string {
	t.Helper()
	err := <-c.done
	c.reaped = true
	out := c.readOutput(t)
	if err != nil {
		t.Fatalf("fresh executor exited %v: %s", err, out)
	}
	return out
}

func (c *weatherOwner) awaitCut(t *testing.T) weatherCut {
	t.Helper()
	if err := c.ack.SetReadDeadline(time.Now().Add(weatherProcessTimeout)); err != nil {
		t.Fatal(err)
	}
	var e weatherCut
	if err := json.NewDecoder(c.ack).Decode(&e); err != nil {
		t.Fatalf("executor did not acknowledge %s/%s: %v, output %s", c.cut.Boundary, c.cut.Phase, err, c.readOutput(t))
	}
	f := c.cut
	if e.PID != c.cmd.Process.Pid || e.Token != f.Token || e.Scenario != f.Scenario || e.Store != f.Store ||
		e.Subject != f.Subject || e.Generation != f.Generation || e.Boundary != f.Boundary || (f.Phase != "" && e.Phase != f.Phase) ||
		(f.Sequence != 0 && e.Sequence != f.Sequence) || (f.Intent != "" && e.Intent != f.Intent) || (f.Attempt != "" && e.Attempt != f.Attempt) {
		t.Fatalf("acknowledgment does not identify requested boundary: %+v, want %+v", e, f)
	}
	return e
}

func (c *weatherOwner) kill(t *testing.T, witness string) {
	t.Helper()
	if err := c.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := <-c.done
	c.reaped = true
	s, ok := c.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !s.Signaled() || s.Signal() != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v, status %v", err, s)
	}
	t.Logf("SIGKILL pid=%d witness=%s", c.cmd.Process.Pid, witness)
}

func (e weatherCut) witness() string {
	return fmt.Sprintf("%s/%s subject=%s generation=%d sequence=%d operation=%s authorization=%s", e.Boundary, e.Phase, e.Subject, e.Generation, e.Sequence, e.Intent, e.Attempt)
}
