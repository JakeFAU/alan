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
	communication "github.com/JakeFAU/alan/research/communication"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

func startCommunication(t *testing.T, path, cut, profile, order string) *process {
	t.Helper()
	ar, aw, e := os.Pipe()
	must(t, e)
	rr, rw, e := os.Pipe()
	must(t, e)
	log, e := os.CreateTemp(t.TempDir(), "communication-child")
	must(t, e)
	binary, e := os.Executable()
	must(t, e)
	cmd := exec.Command(binary, "-test.run=^TestCommunicationProcessChild$", "-test.v", "-test.timeout="+childTimeout().String())
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "ALAN_I_") && !strings.HasPrefix(v, "ALAN_J_") && !strings.HasPrefix(v, "ALAN_K_") && !strings.HasPrefix(v, "ALAN_BARRIER=") && !strings.HasPrefix(v, "OPENAI_API_KEY=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "ALAN_K_DB="+path, "ALAN_K_CUT="+cut, "ALAN_K_PROFILE="+profile, "ALAN_K_ORDER="+order)
	if dir := os.Getenv("ALAN_I_ARTIFACTS"); dir != "" {
		cmd.Env = append(cmd.Env, "ALAN_I_ARTIFACTS="+dir)
	}
	f := boundary.Event{Store: path, Scenario: t.Name(), Token: fmt.Sprintf("%s/%s/%d", t.Name(), cut, time.Now().UnixNano()), Boundary: "communication.checked-cut", Phase: cut, Subject: "peer"}
	if profile == "automatic" {
		f.Boundary = "execution.committed"
		f.Phase = "post-commit"
		f.Generation = 1
		f.Sequence = 37
	}
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
	env := selectedEnv(cmd.Env)
	for _, v := range cmd.Env {
		if strings.HasPrefix(v, "ALAN_K_") {
			env = append(env, v)
		}
	}
	retain(t, fmt.Sprintf("start-%d", cmd.Process.Pid), map[string]any{"pid": cmd.Process.Pid, "argv": cmd.Args, "environment": env, "filter": f, "profile": profile, "binary_sha256": fmt.Sprintf("%x", shaBytes(raw)), "race_instrumented": raceEnabled})
	return p
}
func TestCommunicationProcessChild(t *testing.T) {
	path := os.Getenv("ALAN_K_DB")
	if path == "" {
		t.Skip("subprocess entry; not a qualification case")
	}
	ctx, e := boundary.FromEnvironment(t.Context(), path)
	must(t, e)
	cut := os.Getenv("ALAN_K_CUT")
	if os.Getenv("ALAN_K_PROFILE") == "automatic" {
		s, e := runtime.Open(ctx, path, runtime.DefaultLimits())
		must(t, e)
		defer s.Close()
		r, e := s.Run(ctx, runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
		must(t, e)
		if !r.Quiescent {
			t.Fatal("automatic operational bound", r)
		}
		retain(t, fmt.Sprintf("automatic-child-%d", os.Getpid()), r)
		return
	}
	d := newCommunicationDriver(t, path)
	s := d.drive(os.Getenv("ALAN_K_ORDER"))
	if cut != "" {
		if !communicationCut(s, cut) {
			t.Fatal("safe cut predicate absent", cut)
		}
		peer, e := d.store.Snapshot(t.Context(), runtime.RunID{Subject: "peer", Generation: 1})
		must(t, e)
		d.record(map[string]any{"kind": "safe_checked_cut", "cut": cut, "inspection_hash": hash(s), "peer": peer})
		must(t, boundary.Reached(ctx, boundary.Event{Boundary: "communication.checked-cut", Phase: cut, Subject: "peer", Generation: 1, Epoch: peer.Epoch, Sequence: peer.Sequence, Intent: communicationReport(s).ID}))
	} else if len(d.repeats) == 0 {
		t.Fatal("public recovery did not repeat original authorization")
	}
	retain(t, fmt.Sprintf("public-child-%d", os.Getpid()), map[string]any{"inspection": s, "reauthorizations": d.repeats, "readmissions": d.readmissions})
}
func communicationEpoch(s runtime.Inspection, subject string) int64 {
	for _, r := range s.Tables["runs"] {
		if r["subject"] == subject {
			n, _ := evidenceNumber(r["epoch"])
			return n
		}
	}
	return 0
}
func TestCommunicationRecovery(t *testing.T) {
	for _, cut := range []string{"report-no-authorization", "report-authorized-peer-terminal", "report-refused-peer-terminal"} {
		t.Run(cut, func(t *testing.T) {
			path := communicationFixture(t)
			profile, order := "public-cut", cut
			if cut == "report-no-authorization" {
				profile = "automatic"
			}
			p := startCommunication(t, path, cut, profile, order)
			ack := p.acknowledgment(t)
			before := inspect(t, path)
			if !communicationCut(before, cut) {
				t.Fatal("acknowledged exact cut absent", cut)
			}
			tr, e := decodeCommunication(before, model.Request{ID: 1, Value: 0, Mode: "answer"}, communication.J1)
			must(t, e)
			queue := communicationQueue(before)
			if cut == "report-no-authorization" {
				want := []string{communicationIdentity("message-delivery", communicationReport(before).ID)}
				if !reflect.DeepEqual(queue["pending_internal_keys"], want) || len(queue["unfinished_invocations"].([]string)) != 0 {
					t.Fatal("sole-report queue predicate absent", queue)
				}
			}
			retain(t, "before", map[string]any{"cut": cut, "profile": profile, "inspection": before, "trace": tr, "ack": ack, "queue": queue})
			p.wait(t, true)
			resume := "public-resume"
			if cut == "report-no-authorization" {
				resume = "automatic"
			}
			next := startCommunication(t, path, "", resume, "resume")
			if next.cmd.Process.Pid == ack.PID {
				t.Fatal("fresh PID absent")
			}
			next.wait(t, false)
			after := inspect(t, path)
			stablePrefix(t, before, after)
			joinIdentityPrefix(t, before, after)
			if communicationEpoch(after, "root") <= communicationEpoch(before, "root") {
				t.Fatal("activation not renewed")
			}
			if !communicationImmutable([]string{"permissions", "composition_permissions", "composition_charges", "composition_admission_decisions"}, before, after) {
				t.Fatal("original permissions/charges/admissions changed")
			}
			tr, e = decodeCommunication(after, model.Request{ID: 1, Value: 0, Mode: "answer"}, communication.J1)
			must(t, e)
			in := communicationReport(before)
			if cut == "report-refused-peer-terminal" {
				if !communicationCut(after, cut) || !communicationImmutable([]string{"attempts", "admissions", "message_acknowledgments", "observations", "history", "intents", "composition_facts", "composition_root_publications"}, before, after) {
					t.Fatal("refused exact retry changed original immutable rows or pending state")
				}
			} else if tr.Aggregate != "succeeded" || tr.Messages[0].Consumption == nil {
				t.Fatal("resumed original report did not complete")
			}
			if cut == "report-authorized-peer-terminal" && !reflect.DeepEqual(communicationRow(before, "attempts", "intent_id", in.ID), communicationRow(after, "attempts", "intent_id", in.ID)) {
				t.Fatal("allowed auth snapshot changed")
			}
			retain(t, "after", map[string]any{"inspection": after, "trace": tr, "before_epoch": communicationEpoch(before, "root"), "after_epoch": communicationEpoch(after, "root"), "resume_profile": resume})
			replay := start(t, path, "", "", "", true)
			replay.wait(t, false)
			if !reflect.DeepEqual(after, inspect(t, path)) {
				t.Fatal("fresh replay mutated ledger")
			}
		})
	}
}
