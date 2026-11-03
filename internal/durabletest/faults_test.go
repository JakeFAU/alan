package durabletest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

func retryProbe(t *testing.T) executor.Provision {
	p := executor.Provision{}
	retry := executor.Instruction{Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "lifecycle", Target: executor.Lit(executor.Text("worker")), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("probe")), Payload: map[string]executor.Operand{"operation": executor.Lit(executor.Text("Retry")), "revision": executor.Lit(executor.Int(1))}}}
	result := executor.Instruction{Op: "receive", Match: &executor.Match{Kind: "lifecycle", Provenance: "runtime-lifecycle", Source: executor.Lit(executor.Text("worker")), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("probe")), Operation: executor.Ref("pending")}, Bind: map[string]string{"accepted": "accepted", "reason": "reason"}}
	input := executor.Instruction{Op: "receive", Match: &executor.Match{Kind: "input", Provenance: "local-input", Source: executor.Lit(executor.Text("local")), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("input")), Operation: executor.Lit(executor.Text(""))}, Bind: map[string]string{"go": "go"}}
	addProgram(t, &p, "a-probe", []executor.Instruction{retry, result, {Op: "end", Outcome: "succeeded"}}, map[string]executor.Value{})
	addProgram(t, &p, "new-probe", []executor.Instruction{input, retry, result, {Op: "end", Outcome: "succeeded"}}, map[string]executor.Value{})
	addProgram(t, &p, "worker", []executor.Instruction{input, {Op: "end", Outcome: "failed"}}, map[string]executor.Value{})
	for _, source := range []string{"a-probe", "new-probe"} {
		p.Permissions = append(p.Permissions, executor.Permission{Kind: "lifecycle", Source: source, Target: "worker", Operation: "Retry", Revision: 1})
	}
	for _, target := range []string{"worker", "new-probe"} {
		p.Permissions = append(p.Permissions, executor.Permission{Kind: "input", Source: "local", Target: target, Operation: "input", Obligation: "input"})
	}
	return p
}
func inputCLI(t *testing.T, db, target, key string) executor.AdmissionResult {
	t.Helper()
	input := executor.Input{TransportID: key, LogicalID: key, Target: target, Generation: 1, Obligation: "input", Payload: map[string]executor.Value{"go": executor.Text("yes")}}
	file, err := os.CreateTemp(binaries(t), "input-*.json")
	if err != nil {
		t.Fatal(err)
	}
	json.NewEncoder(file).Encode(input)
	file.Close()
	raw := cli(t, "input", "--db", db, "--request", file.Name())
	var result executor.AdmissionResult
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	emit(t, map[string]any{"scenario": t.Name(), "initial_input": input, "result": result})
	return result
}
func failWorkerThroughProgram(t *testing.T, db, observation string) {
	t.Helper()
	s, err := executor.Open(t.Context(), db, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Snapshot(t.Context(), executor.RunID{Subject: "worker", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.Commit(t.Context(), state.Authority(), observation)
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.Commit(t.Context(), state.Authority(), "")
	if err != nil || state.Projection.Status != "failed" {
		t.Fatalf("actual target program failure %+v %v", state, err)
	}
}

// injectLostRejection is a separate TEST ONLY database fault, never a failpoint.
func injectLostRejection(t *testing.T, path, id string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM lifecycle_results WHERE id=?", id)
	if err != nil {
		t.Fatal(err)
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		t.Fatal("fault did not delete original rejection")
	}
}
func TestRejectedResultCrashAndLostRejectionControl(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprint("lost-rejection=", fault), func(t *testing.T) {
			p := retryProbe(t)
			db := provisionCLI(t, p)
			initial := inspect(t, db)
			c := startPaused(t, db, "lifecycle.decided", "rejected", "a-probe", 1)
			ack := c.wait(t)
			c.kill(t)
			stopped := inspect(t, db)
			var original executor.LifecycleResult
			json.Unmarshal([]byte(stopped.Tables["lifecycle_results"][0]["result"].(string)), &original)
			if original.Accepted || original.Reason != "not_retryable" || original.Before.Run.Generation != 1 || original.Before.Revision != 1 || original.Before.Projection.Status != "open" {
				t.Fatal("rejection not genuine active-target eligibility")
			}
			if fault {
				injectLostRejection(t, db, ack.Intent)
			}
			input := inputCLI(t, db, "worker", "fail-worker")
			failWorkerThroughProgram(t, db, input.ObservationID)
			eligible := inspect(t, db)
			worker, _, _ := projection(eligible, "worker", 1)
			if worker.Status != "failed" {
				t.Fatal("target not eligible")
			}
			cli(t, "run", "--db", db)
			after := inspect(t, db)
			failures := checkRecords(stopped, after, resource.Snapshot{})
			if fault {
				if len(failures) == 0 {
					t.Fatal("unchanged oracle missed lost rejection")
				}
				var now executor.LifecycleResult
				json.Unmarshal([]byte(after.Tables["lifecycle_results"][0]["result"].(string)), &now)
				if !now.Accepted || now.Successor != 2 {
					t.Fatal("fault masked: same op did not now accept")
				}
			} else {
				if len(failures) > 0 {
					t.Fatal(failures)
				}
				var now executor.LifecycleResult
				json.Unmarshal([]byte(after.Tables["lifecycle_results"][0]["result"].(string)), &now)
				if !reflect.DeepEqual(original, now) {
					t.Fatal("original rejection changed")
				}
				probe, _, _ := projection(after, "a-probe", 1)
				if probe.Status != "succeeded" || probe.Data["accepted"] != (executor.Value{Kind: "bool", Flag: false}) {
					t.Fatal("rejection not consumed by ordinary program")
				}
				inputCLI(t, db, "new-probe", "new-request")
				cli(t, "run", "--db", db)
				newAfter := inspect(t, db)
				if len(newAfter.Tables["lifecycle_results"]) != 2 {
					t.Fatal("new operation did not decide")
				}
				var accepted executor.LifecycleResult
				json.Unmarshal([]byte(newAfter.Tables["lifecycle_results"][1]["result"].(string)), &accepted)
				if !accepted.Accepted || accepted.Successor != 2 {
					t.Fatal("new authorized operation not eligible")
				}
				replayUnchanged(t, db, "")
				emit(t, map[string]any{"new_operation_result": accepted, "after_new_request": newAfter})
			}
			emit(t, map[string]any{"scenario": t.Name(), "fault": fault, "provision": p, "initial": initial, "barrier": ack, "stopped": stopped, "eligible_before_retransmit": eligible, "after": after, "unchanged_oracle_failures": failures, "assertions": []string{"initial matching grant/gen/revision", "target failed through input and ordinary commits", "same ID retransmitted", "fault acceptance observable"}})
		})
	}
}

// injectRecoveredPolicyReset mutates only test state after a successful live Open.
// The oracle reads history independently; it does not ask runtime replay to diagnose it.
func injectRecoveredPolicyReset(t *testing.T, path string) {
	t.Helper()
	s, err := executor.Open(t.Context(), path, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Snapshot(t.Context(), executor.RunID{Subject: "inner", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	state.Projection.Data["spent"] = executor.Int(0)
	state.Projection.Data["pending"] = executor.Text("")
	raw, _ := json.Marshal(state.Projection)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE runs SET projection=? WHERE subject='inner' AND generation=1", raw); err != nil {
		t.Fatal(err)
	}
	db.Close()
	report, err := s.Run(t.Context(), executor.RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.OpenRuns) == 0 {
		t.Fatal("fault not observable in live runner")
	}
}
func TestRecoveredPolicyResetControl(t *testing.T) {
	db := filepath.Join(binaries(t), fmt.Sprintf("reset-%d.sqlite", time.Now().UnixNano()))
	cli(t, "init", "--db", db, "--demo", "nested")
	c := startPaused(t, db, "lifecycle.decided", "accepted", "inner", 7)
	ack := c.wait(t)
	c.kill(t)
	before := inspect(t, db)
	injectRecoveredPolicyReset(t, db)
	after := inspect(t, db)
	failures := checkRecords(before, after, resource.Snapshot{})
	if len(failures) == 0 {
		t.Fatal("unchanged oracle missed recovered policy reset")
	}
	emit(t, map[string]any{"scenario": t.Name(), "fault": "after validated Open, reset same-run spent/pending; then actual Run", "barrier": ack, "before": before, "after": after, "unchanged_oracle_failures": failures})
}
func TestSpeculativeDispatchControlAndCorrectRollback(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprint("speculative=", fault), func(t *testing.T) {
			_, url, resourcePath := startResource(t, false)
			p := resourceProvision(t, url, "idempotent")
			db := provisionCLI(t, p)
			before := inspect(t, db)
			var c *child
			if !fault {
				c = startPaused(t, db, "execution.staged", "pre-commit", "client", 1)
			} else {
				c = startFaultChild(t, db)
			}
			ack := c.wait(t)
			c.kill(t)
			after := inspect(t, db)
			endpoint, err := resource.Inspect(t.Context(), resourcePath)
			if err != nil {
				t.Fatal(err)
			}
			failures := checkRecords(before, after, endpoint)
			if len(after.Tables["intents"]) != 0 || len(after.Tables["history"]) != 0 {
				t.Fatal("local transaction did not roll back")
			}
			if fault {
				if len(endpoint.Applications) != 1 || len(failures) == 0 {
					t.Fatal("speculative fault not detected")
				}
			} else {
				if len(endpoint.Applications) != 0 || len(failures) != 0 {
					t.Fatal("uncommitted correct path dispatched")
				}
				cli(t, "run", "--db", db)
				recovered := inspect(t, db)
				endpointAfter, err := resource.Inspect(t.Context(), resourcePath)
				if err != nil {
					t.Fatal(err)
				}
				if len(endpointAfter.Applications) != 1 {
					t.Fatal("retry did not apply once")
				}
				if f := checkRecords(after, recovered, endpointAfter); len(f) > 0 {
					t.Fatal(f)
				}
				emit(t, map[string]any{"recovered": recovered, "endpoint_after": endpointAfter})
			}
			emit(t, map[string]any{"scenario": t.Name(), "fault": fault, "provision": p, "barrier": ack, "before": before, "after": after, "endpoint": endpoint, "unchanged_oracle_failures": failures})
		})
	}
}
func startFaultChild(t *testing.T, db string) *child {
	t.Helper()
	binary := filepath.Join(binaries(t), "executor-fault.test")
	args := []string{"test", "-c", "-o", binary}
	if raceChildren {
		args = append(args, "-race")
	}
	args = append(args, "./internal/executor")
	build := exec.Command("go", args...)
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	config := expectedBarrier(t, db, "execution.staged", "pre-commit", "client", 1)
	config.Scenario = t.Name()
	config.Token = fmt.Sprintf("fault-%d", time.Now().UnixNano())
	read, write, _ := os.Pipe()
	resumeRead, resumeWrite, _ := os.Pipe()
	cmd := exec.Command(binary, "-test.run=^TestTask4FaultProcess$")
	raw, _ := json.Marshal(config)
	cmd.Env = append(os.Environ(), "ALAN_TEST_FAULT=speculative-dispatch", "ALAN_TEST_STORE="+db, "ALAN_BARRIER="+string(raw))
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	stderr, err := os.CreateTemp(binaries(t), "fault-stderr-*")
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
