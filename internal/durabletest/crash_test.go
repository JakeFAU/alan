package durabletest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/executor"
)

var buildOnce sync.Once
var binaryDir, buildFailure string

func binaries(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		var err error
		binaryDir, err = os.MkdirTemp("", "alan-task4-evidence-")
		if err != nil {
			buildFailure = err.Error()
			return
		}
		for _, name := range []string{"alan", "alan-resource"} {
			args := []string{"build", "-buildvcs=false", "-o", filepath.Join(binaryDir, name)}
			if raceChildren {
				args = append(args, "-race")
			}
			args = append(args, "./cmd/"+name)
			cmd := exec.Command("go", args...)
			cmd.Dir = "../.."
			out, err := cmd.CombinedOutput()
			if err != nil {
				buildFailure = fmt.Sprintf("%v %s", err, out)
				return
			}
		}
	})
	if buildFailure != "" {
		t.Fatal(buildFailure)
	}
	return binaryDir
}
func cli(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(filepath.Join(binaries(t), "alan"), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v %s", args, err, out)
	}
	emit(t, map[string]any{"type": "process", "command": cmd.Args, "pid": cmd.ProcessState.Pid(), "exit_code": cmd.ProcessState.ExitCode(), "stdout": string(out), "binary_sha256": binaryHash(cmd.Path), "race_children": raceChildren, "environment": map[string]string{"GOCACHE": os.Getenv("GOCACHE")}})
	return out
}

type barrier struct {
	Scenario, Token, Store, Boundary, Phase, Subject string
	PID                                              int
	Generation, Epoch, Sequence                      int64
	Intent, Attempt                                  string
	InFlight                                         []string
}
type child struct {
	acknowledged bool
	cmd          *exec.Cmd
	ack          *os.File
	release      *os.File
	config       barrier
	stderr       *os.File
}

func startPaused(t *testing.T, db, boundaryName, phase, subject string, sequence int64) *child {
	t.Helper()
	dir := binaries(t)
	token := fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
	config := expectedBarrier(t, db, boundaryName, phase, subject, sequence)
	config.Scenario = t.Name()
	config.Token = token
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resumeRead, resumeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(dir, "alan"), "run", "--db", db)
	raw, _ := json.Marshal(config)
	cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	stderr, err := os.CreateTemp(dir, "child-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	write.Close()
	resumeRead.Close()
	c := &child{cmd: cmd, ack: read, release: resumeWrite, config: config, stderr: stderr}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		read.Close()
		resumeWrite.Close()
		stderr.Close()
	})
	return c
}
func (c *child) wait(t *testing.T) barrier {
	t.Helper()
	type result struct {
		b   barrier
		err error
	}
	ready := make(chan result, 1)
	go func() { b, err := readAcknowledgment(c.ack, c.config, c.cmd.Process.Pid); ready <- result{b, err} }()
	var got result
	select {
	case got = <-ready:
	case <-time.After(30 * time.Second):
		t.Fatal("missing acknowledged boundary")
	}
	if got.err != nil {
		raw, _ := os.ReadFile(c.stderr.Name())
		t.Fatalf("no barrier: %v child=%s", got.err, raw)
	}
	b := got.b
	want := c.config
	if !matchingBarrier(b, want, c.cmd.Process.Pid) {
		t.Fatalf("wrong barrier %+v want %+v", b, want)
	}
	validateBarrierFacts(t, b)
	c.acknowledged = true
	return b
}
func (c *child) kill(t *testing.T) {
	t.Helper()
	if !c.acknowledged {
		t.Fatal("refuse claimed crash without validated acknowledgment")
	}
	if err := c.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := c.cmd.Wait()
	status, ok := c.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("not SIGKILL: %v", err)
	}
	emit(t, map[string]any{"type": "process", "command": c.cmd.Args, "pid": c.cmd.Process.Pid, "exit_signal": status.Signal().String(), "barrier_config": c.config, "binary_sha256": binaryHash(c.cmd.Path), "race_children": raceChildren})
}
func inspect(t *testing.T, db string) executor.Inspection {
	t.Helper()
	snapshot, err := executor.Inspect(context.Background(), db, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
func TestAutomaticCLINestedCommitCrashBoundaries(t *testing.T) {
	for _, boundaryName := range []string{"execution.staged", "execution.committed"} {
		t.Run(boundaryName, func(t *testing.T) {
			db := filepath.Join(binaries(t), fmt.Sprintf("%s-%d.sqlite", boundaryName, time.Now().UnixNano()))
			cli(t, "init", "--db", db, "--demo", "nested")
			before := inspect(t, db)
			c := startPaused(t, db, boundaryName, "", "inner", 7)
			ack := c.wait(t)
			c.kill(t)
			stopped := inspect(t, db)
			expected := 0
			if boundaryName == "execution.committed" {
				expected = 1
			}
			n := 0
			for _, row := range stopped.Tables["intents"] {
				var intent executor.Intent
				json.Unmarshal([]byte(row["descriptor"].(string)), &intent)
				if intent.Kind == "lifecycle" {
					n++
				}
			}
			if n != expected {
				t.Fatalf("partial intent: %d want %d", n, expected)
			}
			cli(t, "run", "--db", db)
			after := inspect(t, db)
			if len(after.Tables["lifecycle_results"]) != 1 || len(after.Tables["runs"]) != 4 {
				t.Fatal("nested recovery failed")
			}
			if failures := checkRecords(before, after, resource.Snapshot{}); len(failures) > 0 {
				t.Fatal(failures)
			}
			if err := checkNested(after, "outer", "inner", "worker"); err != nil {
				t.Fatal(err)
			}
			record := map[string]any{"scenario": t.Name(), "command": c.cmd.Args, "barrier": ack, "exit": "SIGKILL", "before": before, "stopped": stopped, "after": after, "race_children": raceChildren}
			emit(t, record)
		})
	}
}

func binaryHash(path string) string {
	raw, _ := os.ReadFile(path)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func emit(t *testing.T, record any) {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(binaries(t), "evidence-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(raw); err != nil {
		t.Fatal(err)
	}
	file.Close()
	t.Logf("DURABLE %s", raw)
}
func replayUnchanged(t *testing.T, db, resourcePath string) {
	t.Helper()
	before := inspect(t, db)
	var endpointBefore resource.Snapshot
	var err error
	if resourcePath != "" {
		endpointBefore, err = resource.Inspect(t.Context(), resourcePath)
		if err != nil {
			t.Fatal(err)
		}
	}
	cli(t, "replay", "--db", db)
	after := inspect(t, db)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only replay mutated executor")
	}
	if resourcePath != "" {
		endpointAfter, err := resource.Inspect(t.Context(), resourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(endpointBefore, endpointAfter) {
			t.Fatal("replay contacted/mutated resource")
		}
	}
}
func TestAutomaticNestedAuthorizationDecisionTerminalAndRepeatedCrash(t *testing.T) {
	for _, scenario := range []struct {
		name, boundary, phase, subject string
		sequence                       int64
		twice                          bool
	}{
		{"authorized", "intent.authorized", "authorized", "inner", 7, false},
		{"accepted", "lifecycle.decided", "accepted", "inner", 7, false},
		{"second-recovery-crash", "lifecycle.decided", "accepted", "inner", 7, true},
		{"terminal", "execution.committed", "terminal", "worker", 3, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := filepath.Join(binaries(t), fmt.Sprintf("nested-%d.sqlite", time.Now().UnixNano()))
			cli(t, "init", "--db", db, "--demo", "nested")
			initial := inspect(t, db)
			c := startPaused(t, db, scenario.boundary, scenario.phase, scenario.subject, scenario.sequence)
			ack := c.wait(t)
			c.kill(t)
			stopped := inspect(t, db)
			if scenario.subject == "inner" {
				p, _, _ := projection(stopped, "inner", 1)
				if p.Data["spent"] != executor.Int(1) || p.Data["pending"] != executor.Text(ack.Intent) {
					t.Fatal("lost pending/reservation at boundary")
				}
			}
			if scenario.name == "authorized" && len(stopped.Tables["lifecycle_results"]) != 0 {
				t.Fatal("target decided before authorization barrier")
			}
			if scenario.boundary == "lifecycle.decided" && len(stopped.Tables["lifecycle_results"]) != 1 {
				t.Fatal("result not committed")
			}
			if scenario.twice {
				second := startPaused(t, db, "lifecycle.decided", "accepted", "inner", 7)
				ack2 := second.wait(t)
				second.kill(t)
				twice := inspect(t, db)
				if ack2.Epoch != ack.Epoch+1 || ack2.Intent != ack.Intent || ack2.Generation != ack.Generation {
					t.Fatal("recovery changed execution/operation")
				}
				p, _, _ := projection(twice, "inner", 1)
				if p.Data["spent"] != executor.Int(1) || p.Data["pending"] != executor.Text(ack.Intent) {
					t.Fatal("second crash reset policy")
				}
				if failures := checkRecords(stopped, twice, resource.Snapshot{}); len(failures) > 0 {
					t.Fatal(failures)
				}
				emit(t, map[string]any{"scenario": t.Name(), "second_barrier": ack2, "second_stop": twice})
			}
			cli(t, "run", "--db", db)
			after := inspect(t, db)
			if failures := checkRecords(stopped, after, resource.Snapshot{}); len(failures) > 0 {
				t.Fatal(failures)
			}
			if err := checkNested(after, "outer", "inner", "worker"); err != nil {
				t.Fatal(err)
			}
			replayUnchanged(t, db, "")
			emit(t, map[string]any{"scenario": t.Name(), "initial": initial, "barrier": ack, "stopped": stopped, "after": after, "assertions": []string{"retained original operation", "one budget charge", "same supervisor generation", "actual successor computation", "replay unchanged"}})
		})
	}
}
func TestAutomaticCLINormalAndRenamedNested(t *testing.T) {
	for _, names := range [][3]string{{"outer", "inner", "worker"}, {"alpha", "beta", "gamma"}} {
		t.Run(names[0], func(t *testing.T) {
			p, err := executor.NestedProvision(executor.NestedConfig{Outer: names[0], Inner: names[1], Worker: names[2], RetryBudget: 1})
			if err != nil {
				t.Fatal(err)
			}
			db := provisionCLI(t, p)
			before := inspect(t, db)
			cli(t, "run", "--db", db)
			after := inspect(t, db)
			if err = checkNested(after, names[0], names[1], names[2]); err != nil {
				t.Fatal(err)
			}
			if failures := checkRecords(before, after, resource.Snapshot{}); len(failures) > 0 {
				t.Fatal(failures)
			}
			replayUnchanged(t, db, "")
			emit(t, map[string]any{"scenario": t.Name(), "provision": p, "before": before, "after": after})
		})
	}
}
func provisionCLI(t *testing.T, p executor.Provision) string {
	t.Helper()
	dir := binaries(t)
	file, err := os.CreateTemp(dir, "provision-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.NewEncoder(file).Encode(p); err != nil {
		t.Fatal(err)
	}
	file.Close()
	db := file.Name() + ".sqlite"
	cli(t, "init", "--db", db, "--provision", file.Name())
	return db
}

func matchingBarrier(b, want barrier, pid int) bool {
	return b.Scenario == want.Scenario && b.Token == want.Token && b.PID == pid && b.Store == want.Store && b.Boundary == want.Boundary && b.Phase == want.Phase && b.Subject == want.Subject && b.Generation == want.Generation && b.Epoch == want.Epoch && b.Sequence == want.Sequence && b.Intent == want.Intent && b.Attempt == want.Attempt
}
func expectedBarrier(t *testing.T, db, name, phase, subject string, sequence int64) barrier {
	t.Helper()
	snapshot := inspect(t, db)
	var generation, epoch int64
	for _, row := range snapshot.Tables["subjects"] {
		if row["id"] == subject {
			generation = row["generation"].(int64)
		}
	}
	for _, row := range snapshot.Tables["runs"] {
		if row["subject"] == subject && row["generation"] == generation {
			epoch = row["epoch"].(int64) + 1
		}
	}
	if generation <= 0 || epoch <= 1 {
		t.Fatal("no declared source execution")
	}
	if phase == "" {
		if name == "execution.staged" {
			phase = "pre-commit"
		} else {
			phase = "post-commit"
		}
	}
	b := barrier{Store: db, Boundary: name, Phase: phase, Subject: subject, Generation: generation, Epoch: epoch, Sequence: sequence}
	if phase != "terminal" {
		raw, _ := json.Marshal([]any{subject, generation, sequence, 0})
		b.Intent = string(raw)
	}
	if name == "lifecycle.decided" || name == "intent.authorized" || name == "external.authorized" || name == "external.response" {
		raw, _ := json.Marshal([]any{"authorization", b.Intent})
		b.Attempt = string(raw)
	}
	return b
}
func validateBarrierFacts(t *testing.T, b barrier) {
	t.Helper()
	if b.Boundary == "resource.applied" {
		endpoint, err := resource.Inspect(t.Context(), b.Store)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, request := range endpoint.Requests {
			if request.Key == b.Intent && request.Authorization == b.Attempt && request.Subject == b.Subject && request.Generation == b.Generation && request.Epoch == b.Epoch && request.Sequence == b.Sequence {
				found = true
			}
		}
		if !found {
			t.Fatal("endpoint acknowledgment lacks committed request")
		}
		return
	}
	snapshot := inspect(t, b.Store)
	_, epoch, sequence := projection(snapshot, b.Subject, b.Generation)
	expectedSequence := b.Sequence
	if b.Boundary == "execution.staged" {
		expectedSequence--
	}
	if epoch != b.Epoch || sequence != expectedSequence {
		t.Fatalf("acknowledgment differs from retained execution epoch/sequence: %d/%d vs %+v", epoch, sequence, b)
	}
	if b.Intent != "" && b.Boundary != "execution.staged" {
		found := false
		for _, row := range snapshot.Tables["intents"] {
			if row["id"] == b.Intent {
				found = true
			}
		}
		if !found {
			t.Fatal("acknowledged operation is not committed")
		}
	}
	if b.Attempt != "" {
		found := false
		for _, row := range snapshot.Tables["attempts"] {
			if row["id"] == b.Attempt && row["intent_id"] == b.Intent {
				found = true
			}
		}
		if !found {
			t.Fatal("acknowledged attempt not committed")
		}
	}
}

func readAcknowledgment(reader io.Reader, want barrier, pid int) (barrier, error) {
	var b barrier
	if err := json.NewDecoder(reader).Decode(&b); err != nil {
		return b, err
	}
	if !matchingBarrier(b, want, pid) {
		return b, fmt.Errorf("wrong barrier %+v expected %+v", b, want)
	}
	return b, nil
}

func TestSecondExecutorAndAliasesCannotRenewEpochBeforeExclusion(t *testing.T) {
	db := filepath.Join(binaries(t), fmt.Sprintf("exclusion-%d.sqlite", time.Now().UnixNano()))
	cli(t, "init", "--db", db, "--demo", "nested")
	c := startPaused(t, db, "lifecycle.decided", "accepted", "inner", 7)
	ack := c.wait(t)
	before := inspect(t, db)
	alias := db + ".symlink"
	if err := os.Symlink(db, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{db, alias} {
		cmd := exec.Command(filepath.Join(binaries(t), "alan"), "run", "--db", path)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("second executor accepted")
		}
		after := inspect(t, db)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("exclusion came after epoch/store mutation")
		}
		emit(t, map[string]any{"scenario": t.Name(), "rejected_command": cmd.Args, "exit_code": cmd.ProcessState.ExitCode(), "output": string(out), "barrier": ack})
	}
	hardlink := db + ".hardlink"
	if err := os.Link(db, hardlink); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(binaries(t), "alan"), "run", "--db", hardlink)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("hardlink alias accepted")
	}
	if err = os.Remove(hardlink); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("hardlink exclusion mutated store")
	}
	emit(t, map[string]any{"scenario": t.Name(), "rejected_hardlink_command": cmd.Args, "output": string(out), "before": before, "after": after})
	c.kill(t)
}
