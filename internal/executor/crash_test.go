package executor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
)

// crashBarrier is the reusable test-controller evidence envelope. The controller
// accepts only a matching process, store, scenario and exact execution boundary.
type crashBarrier struct {
	Scenario        string
	PID             int
	Store           string
	Phase           string
	Run             RunID
	Epoch, Sequence int64
	Intent          string
}

func TestSIGKILLExecutionCommitBoundaries(t *testing.T) {
	if phase := os.Getenv("ALAN_CRASH_PHASE"); phase != "" {
		path := os.Getenv("ALAN_CRASH_STORE")
		scenario := os.Getenv("ALAN_CRASH_SCENARIO")
		s, err := Create(t.Context(), path, fixture(t, false), DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		initial := state(t, s)
		acknowledge := func(subject string, generation, epoch, sequence int64, intent string) {
			barrier := crashBarrier{Scenario: scenario, PID: os.Getpid(), Store: path, Phase: phase, Run: RunID{subject, generation}, Epoch: epoch, Sequence: sequence, Intent: intent}
			if err := json.NewEncoder(os.Stdout).Encode(barrier); err != nil {
				t.Fatal(err)
			}
			for {
				time.Sleep(time.Hour)
			}
		}
		if phase == "before-commit" {
			// The driver hook and temporary trigger live entirely in test code. They
			// pause the real Commit after history/intents are staged, before COMMIT.
			err = s.conn.Raw(func(driverConn any) error {
				return driverConn.(*sqlite3.SQLiteConn).RegisterFunc("crash_barrier", func(subject string, generation, epoch, sequence int64, intent string) int {
					acknowledge(subject, generation, epoch, sequence, intent)
					return 0
				}, false)
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.conn.ExecContext(t.Context(), `CREATE TEMP TRIGGER pause_execution BEFORE UPDATE OF projection ON runs BEGIN SELECT crash_barrier(NEW.subject,NEW.generation,NEW.epoch,NEW.sequence,(SELECT id FROM intents WHERE subject=NEW.subject AND generation=NEW.generation AND sequence=NEW.sequence)); END`)
			if err != nil {
				t.Fatal(err)
			}
		}
		after, err := s.Commit(t.Context(), initial.Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		acknowledge(after.ID.Subject, after.ID.Generation, after.Epoch, after.Sequence, after.Projection.Data["pending"].Text)
		return
	}
	for _, phase := range []string{"before-commit", "after-commit"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "executor.sqlite")
			scenario := t.Name() + ":" + filepath.Base(filepath.Dir(path))
			cmd := exec.Command(os.Args[0], "-test.run=^TestSIGKILLExecutionCommitBoundaries$")
			cmd.Env = append(os.Environ(), "ALAN_CRASH_PHASE="+phase, "ALAN_CRASH_STORE="+path, "ALAN_CRASH_SCENARIO="+scenario)
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stderr = os.Stderr
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			})
			type result struct {
				barrier crashBarrier
				err     error
			}
			ready := make(chan result, 1)
			go func() {
				var b crashBarrier
				err := json.NewDecoder(bufio.NewReader(pipe)).Decode(&b)
				ready <- result{b, err}
			}()
			var ack result
			select {
			case ack = <-ready:
			case <-time.After(20 * time.Second):
				t.Fatal("no acknowledged execution barrier")
			}
			if ack.err != nil {
				t.Fatal(ack.err)
			}
			b := ack.barrier
			if b.Scenario != scenario || b.PID != cmd.Process.Pid || b.Store != path || b.Phase != phase || b.Run != (RunID{"worker", 42}) || b.Epoch != 1 || b.Sequence != 1 || b.Intent != `["worker",42,1,0]` {
				t.Fatalf("wrong barrier: %+v", b)
			}
			if err = cmd.Process.Signal(syscall.SIGKILL); err != nil {
				t.Fatal(err)
			}
			err = cmd.Wait()
			if err == nil {
				t.Fatal("child exited normally")
			}
			status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatalf("not killed: %v", err)
			}
			report, err := Replay(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if phase == "after-commit" {
				want = 1
			}
			if report.HistoryCount != want || report.IntentCount != want || len(report.Runs) != 1 || report.Runs[0].Sequence != int64(want) {
				t.Fatalf("partial/lost commit: %+v", report)
			}
			s, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatalf("SIGKILL did not release exclusivity: %v", err)
			}
			defer s.Close()
			resumed := state(t, s)
			if resumed.Epoch != 2 || resumed.ID != b.Run || resumed.Sequence != int64(want) {
				t.Fatalf("wrong recovered execution %+v", resumed)
			}
			t.Logf("barrier=%+v termination=%s SQLite=%s source=%s replay_history=%d", b, fmt.Sprint(status.Signal()), report.SQLiteVersion, report.SQLiteSourceID, report.HistoryCount)
		})
	}
}
