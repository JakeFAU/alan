//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
)

// The stopped owner uses the real CLI and inherits the existing test barrier.
func TestOutingBoundaryOwner(t *testing.T) {
	path, active := os.LookupEnv("ALAN_OUTING_CUT_DB")
	if !active {
		return
	}
	ctx, err := boundary.FromEnvironment(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := runCLI(ctx, []string{"run", "--db", path, "--json"}, os.Stdout); err != nil {
		t.Fatal(err)
	}
}

type outingCutOwner struct {
	cmd  *exec.Cmd
	done chan error
	log  string
}

func launchOutingOwner(t *testing.T, binary, db string, filter *boundary.Event) (*outingCutOwner, boundary.Event) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), outingProcessTimeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, binary, "run", "--db", db, "--json")
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0")
	dir := t.TempDir()
	var ack *os.File
	if filter != nil {
		filter.Scenario, filter.Store, filter.Token = t.Name(), db, dir
		raw, err := json.Marshal(filter)
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOutingBoundaryOwner$")
		cmd.Env = append(os.Environ(), "ALAN_OUTING_CUT_DB="+db, "ALAN_BARRIER="+string(raw), "GORACE=atexit_sleep_ms=0")
		ackRead, ackWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		ack = ackRead
		defer ackRead.Close()
		defer ackWrite.Close()
		releaseRead, releaseWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer releaseRead.Close()
		t.Cleanup(func() { _ = releaseWrite.Close() })
		cmd.ExtraFiles = []*os.File{ackWrite, releaseRead}
	}
	log := filepath.Join(dir, "owner.txt")
	output, err := os.Create(log)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for _, file := range cmd.ExtraFiles {
		_ = file.Close()
	}
	owner := &outingCutOwner{cmd: cmd, done: make(chan error, 1), log: log}
	go func() { owner.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if owner.done != nil {
			cancel()
			<-owner.done
		}
	})
	var event boundary.Event
	if filter != nil {
		if err := ack.SetReadDeadline(time.Now().Add(outingProcessTimeout)); err != nil {
			t.Fatal(err)
		}
		if err := json.NewDecoder(ack).Decode(&event); err != nil {
			raw, _ := os.ReadFile(log)
			t.Fatalf("missing acknowledgement for %+v: %v\n%s", filter, err, raw)
		}
		if event.PID != cmd.Process.Pid || event.Store != db || event.Token != dir || event.Scenario != t.Name() ||
			event.Boundary != filter.Boundary || event.Phase != filter.Phase || event.Subject != filter.Subject ||
			event.Generation != filter.Generation || event.Epoch != filter.Epoch || event.Sequence != filter.Sequence {
			t.Fatalf("wrong acknowledgement: got %+v, want %+v", event, filter)
		}
	}
	return owner, event
}

func (owner *outingCutOwner) kill(t *testing.T) {
	t.Helper()
	if err := owner.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := <-owner.done
	owner.done = nil
	status, ok := owner.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		raw, _ := os.ReadFile(owner.log)
		t.Fatalf("wanted SIGKILL, got %v (%v)\n%s", err, status, raw)
	}
	t.Logf("SIGKILL pid=%d", owner.cmd.Process.Pid)
}
