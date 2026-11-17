//go:build integration

package main

import (
	"bytes"
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
	if err := runCLI(ctx, []string{os.Getenv("ALAN_OUTING_CUT_ACTION"), "--db", path, "--json"}, os.Stdout); err != nil {
		t.Fatal(err)
	}
}

type outingCutOwner struct {
	cmd     *exec.Cmd
	done    chan error
	log     string
	release *os.File
}

func launchOutingOwner(t *testing.T, binary, action, db string, filter *boundary.Event) (*outingCutOwner, boundary.Event) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), outingProcessTimeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, binary, action, "--db", db, "--json")
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0")
	dir := t.TempDir()
	var ack, release *os.File
	if filter != nil {
		filter.Scenario, filter.Store, filter.Token = t.Name(), db, dir
		raw, err := json.Marshal(filter)
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOutingBoundaryOwner$")
		cmd.Env = append(os.Environ(), "ALAN_OUTING_CUT_DB="+db, "ALAN_OUTING_CUT_ACTION="+action, "ALAN_BARRIER="+string(raw), "GORACE=atexit_sleep_ms=0")
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
		release = releaseWrite
		t.Cleanup(func() { _ = release.Close() })
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
	owner := &outingCutOwner{cmd: cmd, done: make(chan error, 1), log: log, release: release}
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
			event.Boundary != filter.Boundary || event.Phase != filter.Phase ||
			filter.Subject != "" && event.Subject != filter.Subject ||
			filter.Generation != 0 && event.Generation != filter.Generation ||
			filter.Epoch != 0 && event.Epoch != filter.Epoch ||
			filter.Sequence != 0 && event.Sequence != filter.Sequence ||
			filter.Intent != "" && event.Intent != filter.Intent ||
			filter.Attempt != "" && event.Attempt != filter.Attempt ||
			filter.Root != "" && event.Root != filter.Root {
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

// The existing barrier accepts 'c' to continue the staged transaction.
func (owner *outingCutOwner) continueAndRead(t *testing.T) plannerReport {
	t.Helper()
	if _, err := owner.release.Write([]byte{'c'}); err != nil {
		t.Fatal(err)
	}
	waitErr := <-owner.done
	owner.done = nil
	raw, err := os.ReadFile(owner.log)
	if err != nil {
		t.Fatal(err)
	}
	if waitErr != nil {
		t.Fatalf("released owner failed: %v\n%s", waitErr, raw)
	}
	var report plannerReport
	// The test harness prints its own exit status after the CLI's JSON report.
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&report); err != nil {
		t.Fatalf("released owner did not report its result: %v\n%s", err, raw)
	}
	return report
}
