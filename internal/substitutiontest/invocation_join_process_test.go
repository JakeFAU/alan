//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

func startInvocationJoin(t *testing.T, path string, cut bool) *process {
	t.Helper()
	ar, aw, e := os.Pipe()
	must(t, e)
	rr, rw, e := os.Pipe()
	must(t, e)
	log, e := os.CreateTemp(t.TempDir(), "invocation-join-child")
	must(t, e)
	binary, e := os.Executable()
	must(t, e)
	cmd := exec.Command(binary, "-test.run=^TestInvocationJoinProcessChild$", "-test.v", "-test.timeout="+childTimeout().String())
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "ALAN_") && !strings.HasPrefix(v, "OPENAI_API_KEY=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "ALAN_L_DB="+path)
	if dir := os.Getenv("ALAN_I_ARTIFACTS"); dir != "" {
		cmd.Env = append(cmd.Env, "ALAN_I_ARTIFACTS="+dir)
	}
	f := boundary.Event{Store: path, Scenario: t.Name(), Token: fmt.Sprintf("%s/%d", t.Name(), time.Now().UnixNano()), Boundary: "invocation-join.checked-cut", Phase: "candidate-consumed-s-unproduced", Subject: "root", Generation: 1, Sequence: 3}
	if cut {
		cmd.Env = append(cmd.Env, "ALAN_L_CUT=1")
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
	env := selectedEnv(cmd.Env)
	for _, v := range cmd.Env {
		if strings.HasPrefix(v, "ALAN_L_") {
			env = append(env, v)
		}
	}
	retain(t, fmt.Sprintf("start-%d", cmd.Process.Pid), map[string]any{"pid": cmd.Process.Pid, "argv": cmd.Args, "environment": env, "filter": f, "same_os_executable": binary, "binary_sha256": fmt.Sprintf("%x", shaBytes(raw)), "race_instrumented": raceEnabled})
	return p
}
func TestInvocationJoinProcessChild(t *testing.T) {
	path := os.Getenv("ALAN_L_DB")
	if path == "" {
		t.Skip("subprocess entry, not a qualification case")
	}
	c := model.JoinConfig{Request: model.Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
	if os.Getenv("ALAN_L_CUT") == "1" {
		ctx, e := boundary.FromEnvironment(t.Context(), path)
		must(t, e)
		d := newCommunicationDriver(t, path)
		s := invocationJoinPrefix(t, d, c, "candidate-first")
		if !invocationJoinCut(s, c) {
			t.Fatal("checked physical cut absent")
		}
		root, e := d.store.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
		must(t, e)
		tr := invocationJoinOriginal(t, s, c)
		d.record(map[string]any{"kind": "checked_unfinished_s_cut", "inspection_hash": hash(s), "handles": tr.Handles, "root": root})
		// Store calls have returned. This gate neither holds Store.mu nor reenters a hook.
		must(t, boundary.Reached(ctx, boundary.Event{Boundary: "invocation-join.checked-cut", Phase: "candidate-consumed-s-unproduced", Subject: "root", Generation: 1, Epoch: root.Epoch, Sequence: root.Sequence, Intent: tr.Handles["s"], Root: s.Roots[0].RootID}))
		t.Fatal("cut child was released; expected actual SIGKILL")
	}
	s, e := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, e)
	defer s.Close()
	report, e := s.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
	must(t, e)
	if !report.Quiescent {
		t.Fatal(report)
	}
	t.Logf("fresh-process original work completed: %+v", report)
}

// The single recovery predicate is exercised in each normal/race source-bound run.
func TestInvocationJoinRecovery(t *testing.T) {
	_, path, c := invocationJoinFixture(t, "b", "answer", "answer", 3)
	p := startInvocationJoin(t, path, true)
	ack := p.acknowledgment(t)
	before := inspect(t, path)
	tr := invocationJoinOriginal(t, before, c)
	if !invocationJoinCut(before, c) || ack.Phase != p.filter.Phase || ack.Subject != "root" || ack.Generation != 1 || ack.Sequence != 3 || ack.Intent != tr.Handles["s"] || ack.Root != before.Roots[0].RootID || ack.Epoch != communicationEpoch(before, "root") {
		t.Fatal("ACK does not identify exact retained predicate", ack)
	}
	retain(t, "before", map[string]any{"inspection": before, "trace": tr, "ack": ack, "outstanding": "admitted S unstepped; original request unconsumed; outcome unproduced", "fixed_runs": 1})
	p.wait(t, true)
	next := startInvocationJoin(t, path, false)
	if next.cmd.Process.Pid == ack.PID {
		t.Fatal("fresh PID absent")
	}
	next.wait(t, false)
	after := inspect(t, path)
	final := invocationJoinOriginal(t, after, c)
	if len(final.Component) != 0 || len(final.Guarantee) != 0 {
		t.Fatal(final)
	}
	stablePrefix(t, before, after)
	joinIdentityPrefix(t, before, after)
	if !reflect.DeepEqual(tr.Handles, final.Handles) || !communicationImmutable([]string{"composition_charges", "composition_admission_decisions", "composition_bindings", "composition_members", "composition_requests", "permissions", "composition_permissions"}, before, after) || len(before.Tables["intents"])+1 != len(after.Tables["intents"]) || len(before.Tables["runs"]) != len(after.Tables["runs"]) {
		t.Fatal("replacement work or identity/spending change")
	}
	// Only S's original reply intent is new. Its invocation and run already existed.
	for _, old := range before.Tables["observations"] {
		if old["consumed_sequence"] != nil {
			row := communicationRow(after, "observations", "id", old["id"])
			if !reflect.DeepEqual(old, row) {
				t.Fatal("original consumed evidence changed")
			}
		}
	}
	for _, op := range []string{tr.Handles["e"], tr.Handles["s"]} {
		count := 0
		for _, row := range after.Tables["history"] {
			selected, _ := row["selected_observation"].(string)
			if selected == "" {
				continue
			}
			obs := communicationRow(after, "observations", "id", selected)
			o := decode[observation](obs, "descriptor")
			if o.Operation == op && o.Kind == "invocation" {
				count++
			}
		}
		if count != 1 {
			t.Fatal("original outcome selected more/less than once", op, count)
		}
	}
	if after.Roots[0].Charges.Invocations != 3 || after.Roots[0].Charges.CreatedRuns != 3 || after.Roots[0].Charges.Effects != 0 || communicationEpoch(after, "root") <= ack.Epoch {
		t.Fatal("recovery accounting/activation", after.Roots)
	}
	retain(t, "after", map[string]any{"inspection": after, "trace": final, "before_root_epoch": ack.Epoch, "after_root_epoch": communicationEpoch(after, "root"), "fixed_runs": 1})
	replayUnchanged(t, path)
}
