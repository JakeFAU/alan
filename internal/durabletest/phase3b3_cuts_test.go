//go:build integration

package durabletest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

func reciprocalEndpoint(t *testing.T, dir, name string, cut *barrier) (*child, string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".sqlite")
	token := fmt.Sprintf("reciprocal-%s-%d", name, time.Now().UnixNano())
	config := barrier{Scenario: t.Name(), Token: token, Store: path}
	if cut != nil {
		config.Boundary = cut.Boundary
		config.Phase = cut.Phase
		config.Subject = cut.Subject
		config.Sequence = cut.Sequence
	}
	read, write := pipe2c(t)
	resumeRead, resumeWrite := pipe2c(t)
	cmd := exec.Command(filepath.Join(binaries(t), "alan-resource"), "--db", path, "--token", token, "--listen", "127.0.0.1:0", "--journal", filepath.Join(dir, "endpoint-"+name+".jsonl"))
	if cut != nil {
		raw, _ := json.Marshal(config)
		cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	}
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(dir, "endpoint-"+name+".log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = log
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
	var ready struct {
		URL, Token, Store string
		PID               int
	}
	if err = json.NewDecoder(stdout).Decode(&ready); err != nil {
		raw, _ := os.ReadFile(log.Name())
		t.Fatal(err, string(raw))
	}
	if ready.PID != cmd.Process.Pid || ready.Token != token || ready.Store != path {
		t.Fatal("endpoint identity", ready)
	}
	baseline3b3Write(t, dir, "endpoint-"+name+"-ready.json", ready)
	return c, ready.URL, path
}

type reciprocalCut struct {
	name, boundary, phase, subject string
	sequence                       int64
	intent                         string
	endpoint                       bool
}

func TestPhase3B3ReciprocalCuts(t *testing.T) {
	cuts := []reciprocalCut{
		{"01-before-grant", "execution.staged", "pre-commit", "b", 2, `["b",1,2,0]`, false},
		{"02-retained-grant", "lifecycle.decided", "accepted", "b", 2, `["b",1,2,0]`, false},
		{"03-before-authorization", "execution.committed", "post-commit", "a", 4, `["a",1,4,0]`, false},
		{"04-complete-decision", "dispatch.consider", "pre-admission", "a", 0, `["a",1,4,0]`, false},
		{"05-before-admission", "dispatch.consider", "pre-admission", "a", 0, `["a",1,4,0]`, false},
		{"06a-admitted-no-contact", "dispatch.admitted", "pre-call", "a", 0, `["a",1,4,0]`, false},
		{"06b-applied-no-receipt", "resource.applied", "post-commit", "a", 4, "", true},
		{"07-retained-result", "external.result-recorded", "pre-settlement", "a", 0, `["a",1,4,0]`, false},
		{"08-before-selection", "external.settled", "post-record", "a", 0, `["a",1,4,0]`, false},
		{"09-mutual-wait", "runner.waiting", "in-flight", "", 0, "", true},
		{"10-one-complete", "execution.committed", "terminal", "a", 10, "", false},
	}
	for _, cut := range cuts {
		t.Run(cut.name, func(t *testing.T) {
			dir := reciprocalDirectory(t, "cut-"+cut.name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			var endpointCut *barrier
			if cut.endpoint {
				endpointCut = &barrier{Boundary: "resource.applied", Phase: "post-commit", Subject: "a", Sequence: 4}
			}
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", endpointCut)
			p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
			if err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			store, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			initial := reciprocalCapture(t, db, filepath.Join(dir, "initial"))
			filter := &barrier{Boundary: cut.boundary, Phase: cut.phase, Subject: cut.subject, Sequence: cut.sequence, Intent: cut.intent}
			if cut.name == "06b-applied-no-receipt" {
				filter = &barrier{Boundary: "runner.waiting", Phase: "in-flight"}
			}
			first := reciprocalOwner(t, db, filepath.Join(dir, "first-owner"), filter)
			var endpointAck *boundary.Event
			if cut.endpoint {
				e := reciprocalAwait(t, cb)
				endpointAck = &e
			}
			ack := reciprocalAwait(t, first)
			baseline3b3Write(t, dir, "acknowledgments.json", map[string]any{"Runtime": ack, "Endpoint": endpointAck})
			first.kill(t)
			baseline3b3Write(t, dir, "kill.json", map[string]any{"PID": first.cmd.Process.Pid, "Wait": first.cmd.ProcessState.String(), "Command": first.cmd.Args})
			stopped := reciprocalCapture(t, db, filepath.Join(dir, "stopped"))
			before := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
			baseline3b3Write(t, dir, "endpoints-stopped.json", before)
			reciprocalCheckCut(t, cut, stopped, m, before, ack)
			if cut.endpoint {
				if _, err = cb.release.Write([]byte("c")); err != nil {
					t.Fatal(err)
				}
			}
			next := reciprocalOwner(t, db, filepath.Join(dir, "fresh-owner"), nil)
			phaseFinish(t, next)
			baseline3b3Write(t, dir, "owner-exit.json", map[string]any{"PID": next.cmd.Process.Pid, "Wait": next.cmd.ProcessState.String(), "Command": next.cmd.Args})
			if first.cmd.Process.Pid == next.cmd.Process.Pid {
				t.Fatal("not fresh process")
			}
			endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
			phaseStopEndpoint(t, ca)
			phaseStopEndpoint(t, cb)
			baseline3b3Write(t, dir, "endpoints-final.json", endpoints)
			reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
			reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
			final := reciprocalCapture(t, db, filepath.Join(dir, "final"))
			reciprocalComplete(t, final, m, endpoints)
			for _, id := range []runtime.RunID{m.A, m.B} {
				a, _ := stopped.Run(id)
				b, _ := final.Run(id)
				want := a.Epoch + 1
				if a.Projection.Status != "open" {
					want = a.Epoch
				}
				if b.Epoch != want || b.ProgramID != a.ProgramID || b.ID != a.ID {
					t.Fatal("activation/identity drift", a, b)
				}
			}
			if initial.Namespace != final.Namespace || len(final.Runs) != 2 {
				t.Fatal("namespace/roots changed")
			}
			if old, ok := stopped.Effect(m.BAService); ok && old.ExternalAttempt != nil {
				new, _ := final.Effect(m.BAService)
				if !reflect.DeepEqual(old.ExternalAttempt, new.ExternalAttempt) {
					t.Fatal("original attempt changed")
				}
				if old.ExternalReceipt != nil && len(old.AuthorityAdmissions) != len(new.AuthorityAdmissions) {
					t.Fatal("known result reissued")
				}
				if old.ExternalReceipt == nil && len(old.AuthorityAdmissions) > 0 {
					last := new.AuthorityAdmissions[len(new.AuthorityAdmissions)-1]
					a, _ := final.Run(m.A)
					if last.Source.Epoch != a.Epoch || last.Ordinal != int64(len(old.AuthorityAdmissions)+1) {
						t.Fatal("fresh gate not current", last)
					}
				}
			}
			raw, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if len(raw.Tables["lifecycle_results"]) != 1 || len(raw.Tables["composition_charges"]) != 2 {
				t.Fatal("duplicate grant/charge")
			}
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "cut-" + cut.name, Schedule: "automatic-run", Directory: dir, Classification: "safe-complete", Manifest: m, Cut: &ack})
			baseline3b3Manifest(t, dir)
		})
	}
}

func reciprocalCheckCut(t *testing.T, c reciprocalCut, s *runtime.ExecutionSnapshot, m delegation.Manifest, endpoints map[string]resource.Snapshot, ack boundary.Event) {
	t.Helper()
	a, _ := s.Run(m.A)
	b, _ := s.Run(m.B)
	e, _ := s.Effect(m.BAService)
	switch c.name {
	case "01-before-grant":
		if b.Sequence != 1 {
			t.Fatal("staged grant retained", b)
		}
		if _, ok := s.Operation(m.Grant); ok {
			t.Fatal("uncommitted grant retained")
		}
	case "02-retained-grant":
		if b.Sequence != 2 || a.Sequence != 1 {
			t.Fatal("grant cut advanced", a, b)
		}
	case "03-before-authorization":
		if e == nil || e.Authorization != nil || e.ExternalAttempt != nil {
			t.Fatal("before-source authorization", e)
		}
	case "04-complete-decision", "05-before-admission":
		if e == nil || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || e.ExternalAttempt.Authority == nil || len(e.AuthorityAdmissions) != 0 {
			t.Fatal("decision/pre-gate cut", e)
		}
	case "06a-admitted-no-contact":
		if e == nil || len(e.AuthorityAdmissions) != 1 || ack.AuthorityAdmissionOrdinal != 1 || endpoints["b"].Contacts != 0 || e.ExternalReceipt != nil {
			t.Fatal("post-gate no-contact", e, ack, endpoints)
		}
	case "06b-applied-no-receipt", "09-mutual-wait":
		if e == nil || len(e.AuthorityAdmissions) != 1 || e.ExternalReceipt != nil || e.Settlement != nil || endpoints["b"].Contacts != 1 || len(endpoints["b"].Applications) != 1 || a.Sequence != 4 || b.Sequence != 5 || a.Projection.PC != 4 || b.Projection.PC != 5 {
			t.Fatal("genuine mutual waits/unknown", a, b, e, endpoints)
		}
	case "07-retained-result":
		if e == nil || e.ExternalReceipt == nil || e.Settlement != nil {
			t.Fatal("retained but unsettled", e)
		}
	case "08-before-selection":
		if e == nil || e.ExternalReceipt == nil || e.Settlement == nil {
			t.Fatal("settled missing", e)
		}
		o, ok := s.Observation(e.Settlement.ObservationID)
		if !ok || o.SelectedAt != nil {
			t.Fatal("already selected", o)
		}
	case "10-one-complete":
		if a.Projection.Status != "succeeded" || b.Projection.Status != "open" || b.Sequence != 12 || !a.Projection.Data["paid"].Flag || !b.Projection.Data["paid"].Flag {
			t.Fatal("terminal observation pending", a, b)
		}
	}
	if c.name != "10-one-complete" && (a.Projection.Data["paid"].Flag || b.Projection.Data["paid"].Flag) {
		t.Fatal("invented original payment")
	}
	if c.name != "06b-applied-no-receipt" && c.name != "09-mutual-wait" && c.name != "07-retained-result" && c.name != "08-before-selection" && c.name != "10-one-complete" && (endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0) {
		t.Fatal("contact before cut")
	}
}
