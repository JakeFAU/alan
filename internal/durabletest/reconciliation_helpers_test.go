//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	x "github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
	rt "github.com/JakeFAU/alan/runtime"
)

func phaseStopEndpoint(t *testing.T, c *child) {
	t.Helper()
	if err := c.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := c.cmd.Wait()
	status, ok := c.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("endpoint stop %v", err)
	}
}

func phaseTestProcess(t *testing.T, db, endpoint, mode string, filter *barrier, testName string, setup func(*exec.Cmd)) *child {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^"+testName+"$", "-test.v")
	cmd.Env = append(os.Environ(), "ALAN_PHASE2_CHILD=1", "ALAN_PHASE2_DB="+db, "ALAN_PHASE2_ENDPOINT="+endpoint, "ALAN_PHASE2_MODE="+mode)
	config := barrier{}
	if filter != nil {
		config = *filter
		config.Store = db
		config.Scenario = t.Name()
		config.Token = fmt.Sprintf("phase2-runner-%d", time.Now().UnixNano())
		raw, _ := json.Marshal(config)
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resumeRead, resumeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	log, err := os.CreateTemp(binaries(t), "phase2-runner-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if setup != nil {
		setup(cmd)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	write.Close()
	resumeRead.Close()
	c := &child{cmd: cmd, ack: read, release: resumeWrite, config: config, stderr: log}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		read.Close()
		resumeWrite.Close()
		log.Close()
	})
	return c
}

func phaseFinish(t *testing.T, c *child) {
	t.Helper()
	err := c.cmd.Wait()
	raw, _ := os.ReadFile(c.stderr.Name())
	if err != nil {
		t.Fatalf("fresh executor %v %s", err, raw)
	}
}

func phaseCopy(t *testing.T, source, dest string) {
	t.Helper()
	in, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
}

func phaseAddress(t *testing.T, endpoint string) string {
	t.Helper()
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}

func phaseCreate(t *testing.T, p x.Provision, path string) {
	t.Helper()
	s, err := x.Create(t.Context(), path, p, x.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}

func phaseSnapshot(t *testing.T, path string) resource.Snapshot {
	t.Helper()
	s, err := resource.Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func phaseResultCount(s x.Inspection, route string) int {
	if route == "adapter" {
		return len(s.Tables["adapter_results"])
	}
	return len(s.Tables["external_receipts"])
}

func phaseAttempt(s x.Inspection, route string) (table, key string) {
	if route == "adapter" {
		return "adapter_attempts", "adapter_results"
	}
	return "external_attempts", "external_receipts"
}

func phaseRequest(t *testing.T, s x.Inspection, route string) (string, string) {
	t.Helper()
	table, _ := phaseAttempt(s, route)
	if len(s.Tables[table]) != 1 {
		t.Fatal("expected one retained attempt")
	}
	raw := s.Tables[table][0]["descriptor"].(string)
	if route == "adapter" {
		var a x.AdapterAttempt
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			t.Fatal(err)
		}
		if !a.Allowed {
			t.Fatal(a)
		}
		return a.Prepared.Request.Key, raw
	}
	var a x.ExternalAttempt
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatal(err)
	}
	if !a.Allowed {
		t.Fatal(a)
	}
	return a.Request.Key, raw
}

func gateInstruction() x.Instruction {
	return x.Instruction{
		Op: "receive",
		Match: &x.Match{
			Kind: "input", Provenance: "local-input", Source: x.Lit(x.Text("local")),
			Generation: x.Lit(x.Int(1)), Obligation: x.Lit(x.Text("control")),
			Operation: x.Lit(x.Text("")),
		},
		Bind: map[string]string{"target": "target"},
	}
}

func addOperator(t *testing.T, p *rt.Provision, name, operation, target string) {
	t.Helper()
	lifecycle := x.Effect{
		Kind: "lifecycle", Target: x.Ref("target"), Generation: x.Lit(x.Int(1)),
		Obligation: x.Lit(x.Text("control")),
		Payload: map[string]x.Operand{
			"operation": x.Lit(x.Text(operation)), "revision": x.Lit(x.Int(1)),
		},
	}
	addProgram(t, p, name, []x.Instruction{
		gateInstruction(), {Op: "emit", Dst: "control_op", Effect: &lifecycle},
		{Op: "end", Outcome: "succeeded"},
	}, map[string]x.Value{})
	p.Permissions = append(p.Permissions,
		x.Permission{Kind: "input", Source: "local", Target: name, Operation: "input", Obligation: "control"},
		x.Permission{Kind: "lifecycle", Source: name, Target: target, Operation: operation, Revision: 1},
	)
}

func admitControl(t *testing.T, s *rt.Store, name, target string) string {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), rt.Input{TransportID: name, LogicalID: name, Target: name, Generation: 1, Obligation: "control", Payload: map[string]x.Value{"target": x.Text(target)}})
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
	return a.ObservationID
}

func stepRun(t *testing.T, s *rt.Store, name, observation string) x.RunState {
	t.Helper()
	r, err := s.Snapshot(t.Context(), x.RunID{Subject: name, Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.Commit(t.Context(), r.Authority(), observation)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func manualControl(t *testing.T, s *rt.Store, name, target string) x.LifecycleResult {
	t.Helper()
	stepRun(t, s, name, admitControl(t, s, name, target))
	issued := stepRun(t, s, name, "")
	id := issued.Projection.Data["control_op"].Text
	a, err := s.AuthorizeIntent(t.Context(), id)
	if err != nil || !a.Allowed {
		t.Fatal(a, err)
	}
	req, err := s.LoadLifecycleRequest(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.DecideLifecycle(t.Context(), req)
	if err != nil || !res.Accepted {
		t.Fatal(res, err)
	}
	return res
}

func publicStep(t *testing.T, s *rt.Store, run rt.RunID, obs string) rt.RunState {
	t.Helper()
	state, err := s.Snapshot(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.Commit(t.Context(), state.Authority(), obs)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func invokeEntry(t *testing.T, s *rt.Store, path, operation string) rt.RunID {
	t.Helper()
	a, err := s.AdmitInvocation(t.Context(), operation)
	if err != nil || !a.Accepted || len(a.Members) != 1 {
		t.Fatal(a, err)
	}
	entry := a.Members[0]
	for _, o := range inspect(t, path).Tables["observations"] {
		if o["subject"] == entry.Subject && o["generation"] == entry.Generation && o["consumed_sequence"] == nil {
			publicStep(t, s, entry, o["id"].(string))
			return entry
		}
	}
	t.Fatal("invocation request missing", a)
	return rt.RunID{}
}
func dispatchTarget(s *rt.Store, r rt.RunID, scope bool) rt.DispatchAuthority {
	return rt.DispatchAuthority{Namespace: s.Namespace(), Run: r, OwningScope: scope}
}

func untilTerminal(t *testing.T, s *rt.Store, run rt.RunID) rt.RunState {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		state, err := s.Snapshot(ctx, run)
		if err != nil {
			t.Fatal(err)
		}
		if state.Projection.Status != "open" {
			return state
		}
		runtime.Gosched()
	}
	t.Fatal("terminal transition not observed", run)
	return rt.RunState{}
}
