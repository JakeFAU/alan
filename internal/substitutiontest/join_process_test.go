//go:build integration

package substitutiontest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	joinapp "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/internal/boundary"
	joinmodel "github.com/JakeFAU/alan/research/join"
	"github.com/JakeFAU/alan/runtime"
)

// startJoin reuses the reviewed ACK/SIGKILL/wait machinery, selecting only a new
// child entrypoint. Pipes are observation gates, never protocol participants.
func startJoin(t *testing.T, path, cut string) *process {
	t.Helper()
	ar, aw, e := os.Pipe()
	must(t, e)
	rr, rw, e := os.Pipe()
	must(t, e)
	log, e := os.CreateTemp(t.TempDir(), "join-child")
	must(t, e)
	binary, e := os.Executable()
	must(t, e)
	cmd := exec.Command(binary, "-test.run=^TestJoinProcessChild$", "-test.v", "-test.timeout="+childTimeout().String())
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "ALAN_I_") && !strings.HasPrefix(v, "ALAN_J_") && !strings.HasPrefix(v, "ALAN_BARRIER=") && !strings.HasPrefix(v, "OPENAI_API_KEY=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	f := boundary.Event{Store: path, Scenario: t.Name(), Token: fmt.Sprintf("%s/%s/%d", t.Name(), cut, time.Now().UnixNano()), Boundary: "execution.committed"}
	cmd.Env = append(cmd.Env, "ALAN_J_DB="+path, "ALAN_J_CUT="+cut)
	if cut != "" {
		raw, e := json.Marshal(f)
		must(t, e)
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
	}
	cmd.ExtraFiles = []*os.File{aw, rr}
	cmd.Stdout = log
	cmd.Stderr = log
	must(t, cmd.Start())
	aw.Close()
	rr.Close()
	p := &process{cmd: cmd, ack: ar, release: rw, log: log, filter: f}
	t.Cleanup(func() {
		if !p.stopped {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		ar.Close()
		rw.Close()
		log.Close()
	})
	raw, e := os.ReadFile(binary)
	must(t, e)
	retain(t, fmt.Sprintf("start-%d", cmd.Process.Pid), map[string]any{"pid": cmd.Process.Pid, "argv": cmd.Args, "filter": f, "binary_sha256": fmt.Sprintf("%x", shaBytes(raw)), "race_instrumented": raceEnabled})
	return p
}
func TestJoinProcessChild(t *testing.T) {
	path := os.Getenv("ALAN_J_DB")
	if path == "" {
		t.Skip("subprocess entry")
	}
	ctx, e := boundary.FromEnvironment(t.Context(), path)
	must(t, e)
	if cut := os.Getenv("ALAN_J_CUT"); cut != "" {
		inherited := ctx
		ctx = boundary.WithObserver(ctx, func(ctx context.Context, event boundary.Event) error {
			if event.Boundary != "execution.committed" {
				return nil
			}
			s, e := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if e != nil {
				return e
			}
			if joinCut(s, cut) {
				return boundary.Reached(inherited, event)
			}
			return nil
		})
	}
	s, e := runtime.Open(ctx, path, runtime.DefaultLimits())
	must(t, e)
	defer s.Close()
	r, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
	must(t, e)
	if !r.Quiescent {
		t.Fatal("operational bound", r)
	}
	t.Logf("run %+v", r)
}
func TestJoinRecovery(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, cut := range []string{"candidate-consumed", "report-committed"} {
			t.Run(name+"/"+cut, func(t *testing.T) {
				q := app.Request{ID: 1, Value: 0, Mode: "answer"}
				d, e := joinapp.New(candidate(t, name), app.Config{Shape: "parallel", Request: q, Limits: limits(3, 3, 2)}, joinapp.Options{})
				must(t, e)
				path := filepath.Join(t.TempDir(), "retained.db")
				create(t, d, path)
				p := startJoin(t, path, cut)
				ack := p.acknowledgment(t)
				before := inspect(t, path)
				if !joinCut(before, cut) {
					t.Fatal("ACK predicate not retained")
				}
				trace, e := decodeJoin(before, q)
				must(t, e)
				if v := joinmodel.CheckTrace(trace); len(v) > 0 {
					t.Fatal(v)
				}
				retain(t, "before", map[string]any{"inspection": before, "trace": trace, "cut": cut, "ack": ack})
				p.wait(t, true)
				next := startJoin(t, path, "")
				if next.cmd.Process.Pid == ack.PID {
					t.Fatal("PID reused")
				}
				next.wait(t, false)
				after := inspect(t, path)
				stablePrefix(t, before, after)
				joinIdentityPrefix(t, before, after)
				trace, e = decodeJoin(after, q)
				must(t, e)
				if v := joinmodel.CheckTrace(trace); len(v) > 0 {
					t.Fatal(v)
				}
				if projection(after, "root").Status != "succeeded" || projection(after, "peer").Status != "succeeded" || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "succeeded" {
					t.Fatal("resumed completion absent")
				}
				var eb, ea int64
				for _, r := range before.Tables["runs"] {
					if r["subject"] == "root" {
						eb, _ = evidenceNumber(r["epoch"])
					}
				}
				for _, r := range after.Tables["runs"] {
					if r["subject"] == "root" {
						ea, _ = evidenceNumber(r["epoch"])
					}
				}
				if ea <= eb {
					t.Fatal("activation epoch not renewed")
				}
				retain(t, "after", map[string]any{"inspection": after, "trace": trace, "before_root_epoch": eb, "after_root_epoch": ea})
				replay := start(t, path, "", "", "", true)
				replay.wait(t, false)
				if !reflect.DeepEqual(after, inspect(t, path)) {
					t.Fatal("fresh-process replay mutated original ledger")
				}
			})
		}
	}
}
func joinIdentityPrefix(t *testing.T, before, after runtime.Inspection) {
	t.Helper()
	for _, table := range []string{"programs", "composition_bindings", "composition_members", "attempts", "admissions", "message_acknowledgments", "history"} {
		for _, old := range before.Tables[table] {
			found := false
			for _, row := range after.Tables[table] {
				if reflect.DeepEqual(old, row) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("original %s row changed", table)
			}
		}
	}
	for _, old := range before.Tables["runs"] {
		found := false
		for _, row := range after.Tables["runs"] {
			if row["subject"] == old["subject"] && row["generation"] == old["generation"] && row["program_id"] == old["program_id"] && row["initial"] == old["initial"] {
				found = true
			}
		}
		if !found {
			t.Fatal("original run/generation/artifact replaced")
		}
	}
}
func TestJoinCancellation(t *testing.T) {
	q := app.Request{ID: 1, Value: 0, Mode: "answer"}
	d, e := joinapp.New(candidate(t, "b"), app.Config{Shape: "parallel", Request: q, Limits: limits(3, 3, 2)}, joinapp.Options{})
	must(t, e)
	addOperator(t, &d.Provision)
	path := filepath.Join(t.TempDir(), "retained.db")
	create(t, d, path)
	p := startJoin(t, path, "candidate-consumed")
	p.acknowledgment(t)
	before := inspect(t, path)
	p.wait(t, true)
	decision := cancelStopped(t, path)
	run(t, path)
	after := inspect(t, path)
	stablePrefix(t, before, after)
	closed, e := validateRetained(after)
	must(t, e)
	if !closed || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" {
		t.Fatal("authorized interruption absent")
	}
	trace, e := decodeJoin(after, q)
	must(t, e)
	retain(t, "result", map[string]any{"before": before, "inspection": after, "decision": decision, "trace": trace, "fixed_runs": 3, "operator_outside_root": true})
	replayUnchanged(t, path)
}
