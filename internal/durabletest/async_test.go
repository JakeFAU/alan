package durabletest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

func startAsync(t *testing.T, db string, filters []barrier) *child {
	t.Helper()
	token := fmt.Sprintf("async-%d", time.Now().UnixNano())
	for i := range filters {
		filters[i].Scenario = t.Name()
		filters[i].Token = token
		filters[i].Store = db
		filters[i].Generation = 1
		filters[i].Epoch = 2
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resumeRead, resumeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(binaries(t), "alan"), "run", "--db", db, "--external-timeout", "30s")
	raw, _ := json.Marshal(filters)
	cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	log, err := os.CreateTemp(binaries(t), "async-child-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	write.Close()
	resumeRead.Close()
	c := &child{cmd: cmd, ack: read, release: resumeWrite, config: filters[0], stderr: log}
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
func asyncRelease(t *testing.T, c *child, command byte) {
	t.Helper()
	if _, err := c.release.Write([]byte{command}); err != nil {
		t.Fatal(err)
	}
}
func asyncNext(c *child, name, phase, subject string, sequence int64) {
	c.config.Boundary = name
	c.config.Phase = phase
	c.config.Subject = subject
	c.config.Sequence = sequence
	c.config.Intent = ""
	c.config.Attempt = ""
}

// The sequence is a causal transcript: controller handshakes append events in
// observation order; timestamps and final-state sorting cannot satisfy this oracle.
type asyncEvent struct {
	Name     string
	Boundary barrier
}

func checkAsyncTrace(events []asyncEvent, endpoint, runner barrier, released bool) error {
	names := []string{"endpoint-held", "peer-committed", "peer-delivered", "held-live"}
	if released {
		names = append(names, "response-released", "result-settled")
	}
	if len(events) != len(names) {
		return fmt.Errorf("wrong event count")
	}
	for i, name := range names {
		e := events[i]
		if e.Name != name {
			return fmt.Errorf("causal order: %s at %d", e.Name, i)
		}
		want := runner
		if name == "endpoint-held" || name == "held-live" || name == "response-released" {
			want = endpoint
		}
		b := e.Boundary
		if b.Scenario != want.Scenario || b.Token != want.Token || b.PID != want.PID || b.Store != want.Store || b.Generation != 1 || b.Epoch != 2 {
			return fmt.Errorf("identity mismatch %s", name)
		}
		if name == "endpoint-held" || name == "held-live" || name == "response-released" || name == "result-settled" {
			if b.Intent != endpoint.Intent || b.Attempt != endpoint.Attempt || b.Subject != "client" || b.Sequence != 1 {
				return fmt.Errorf("wrong request %s", name)
			}
		}
		if name == "peer-committed" && (b.Subject != "ready-peer" || b.Sequence != 1 || b.Boundary != "execution.committed") {
			return fmt.Errorf("wrong peer commit")
		}
		if name == "peer-delivered" && (b.Subject != "caller" || b.Sequence != 1 || b.Boundary != "execution.committed") {
			return fmt.Errorf("no ordinary caller consumption")
		}
		if name == "peer-committed" || name == "peer-delivered" {
			found := false
			for _, attempt := range b.InFlight {
				if attempt == endpoint.Attempt {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("held transport already ended before %s", name)
			}
		}
		if name == "held-live" && b.Phase != "holding" {
			return fmt.Errorf("no live hold probe")
		}
	}
	return nil
}
func TestAutomaticAsyncHeldProgressAndConcurrentCrash(t *testing.T) {
	for _, profile := range []string{"idempotent", "non-idempotent"} {
		for _, cut := range []string{"settle", "held-kill", "received-kill"} {
			t.Run(profile+"/"+cut, func(t *testing.T) {
				endpoint, url, resourcePath := startResource(t, true)
				p := resourceProvision(t, url, profile)
				addProgram(t, &p, "caller", []executor.Instruction{{Op: "receive", Match: &executor.Match{Kind: "terminal", Provenance: "runtime-terminal", Source: executor.Lit(executor.Text("ready-peer")), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("done")), Operation: executor.Lit(executor.Text(""))}, Bind: map[string]string{"outcome": "result"}}, {Op: "end", Outcome: "succeeded"}}, map[string]executor.Value{})
				p.Permissions = append(p.Permissions, executor.Permission{Kind: "observation", Source: "ready-peer", Target: "caller", Operation: "terminal", Revision: 1, Obligation: "done"})
				db := provisionCLI(t, p)
				initial := inspect(t, db)
				filters := []barrier{{Boundary: "execution.staged", Phase: "terminal", Subject: "ready-peer", Sequence: 1}, {Boundary: "execution.committed", Phase: "terminal", Subject: "ready-peer", Sequence: 1}, {Boundary: "execution.committed", Phase: "post-commit", Subject: "caller", Sequence: 1}}
				if cut == "received-kill" {
					filters = append(filters, barrier{Boundary: "external.response", Phase: "pre-record", Subject: "client", Sequence: 1, Intent: `["client",1,1,0]`, Attempt: `["authorization","[\"client\",1,1,0]"]`})
				}
				if cut == "settle" {
					filters = append(filters, barrier{Boundary: "external.settled", Phase: "post-record", Subject: "client", Sequence: 1, Intent: `["client",1,1,0]`, Attempt: `["authorization","[\"client\",1,1,0]"]`})
				}
				c := startAsync(t, db, filters)
				// The real peer transaction can be staged while the transport starts, but
				// cannot commit until receipt/hold of this exact request is acknowledged.
				staged := c.wait(t)
				held := endpoint.wait(t)
				events := []asyncEvent{{"endpoint-held", held}}
				asyncRelease(t, c, 'c')
				asyncNext(c, "execution.committed", "terminal", "ready-peer", 1)
				peer := c.wait(t)
				events = append(events, asyncEvent{"peer-committed", peer})
				asyncRelease(t, c, 'c')
				asyncNext(c, "execution.committed", "post-commit", "caller", 1)
				delivered := c.wait(t)
				events = append(events, asyncEvent{"peer-delivered", delivered})
				beforeRelease := inspect(t, db)
				caller, _, _ := projection(beforeRelease, "caller", 1)
				if caller.Data["result"] != executor.Text("succeeded") || len(beforeRelease.Tables["external_receipts"]) != 0 {
					t.Fatal("not independent ordinary delivery before receipt")
				}
				asyncRelease(t, endpoint, 'p')
				endpoint.config.Phase = "holding"
				live := endpoint.wait(t)
				events = append(events, asyncEvent{"held-live", live})
				if err := checkAsyncTrace(events, held, peer, false); err != nil {
					t.Fatal(err)
				}
				if cut == "held-kill" {
					c.kill(t)
				} else {
					asyncRelease(t, c, 'c')
					asyncRelease(t, endpoint, 'c')
					if cut == "received-kill" {
						asyncNext(c, "external.response", "pre-record", "client", 1)
						c.config.Intent = held.Intent
						c.config.Attempt = held.Attempt
						received := c.wait(t)
						if len(inspect(t, db).Tables["external_receipts"]) != 0 {
							t.Fatal("volatile receipt mistaken for durable evidence")
						}
						emit(t, map[string]any{"scenario": t.Name(), "received_before_record": received})
						c.kill(t)
					} else {
						asyncNext(c, "external.settled", "post-record", "client", 1)
						c.config.Intent = held.Intent
						c.config.Attempt = held.Attempt
						settled := c.wait(t)
						events = append(events, asyncEvent{"response-released", live}, asyncEvent{"result-settled", settled})
						asyncRelease(t, c, 'c')
						if err := c.cmd.Wait(); err != nil {
							raw, _ := os.ReadFile(c.stderr.Name())
							t.Fatalf("run: %v %s", err, raw)
						}

					}
				}
				stopped := inspect(t, db)
				if cut != "settle" {
					if len(stopped.Tables["external_attempts"]) != 1 || len(stopped.Tables["external_receipts"]) != 0 {
						t.Fatal("crash lost uncertainty")
					}
					replayUnchanged(t, db, resourcePath)
					if cut == "held-kill" {
						asyncRelease(t, endpoint, 'c')
					}
					cli(t, "run", "--db", db)
				}
				after := inspect(t, db)
				endpointState, err := resource.Inspect(t.Context(), resourcePath)
				if err != nil {
					t.Fatal(err)
				}
				client, _, _ := projection(after, "client", 1)
				if len(endpointState.Applications) != 1 {
					t.Fatal("logical mutation repeated")
				}
				if profile == "non-idempotent" && cut != "settle" {
					if endpointState.Contacts != 1 || client.Status != "open" || len(after.Tables["external_receipts"]) != 0 {
						t.Fatal("uncertain non-idempotent automatically repeated")
					}
				} else {
					if client.Status != "succeeded" || client.Data["total"] != executor.Int(5) || len(after.Tables["external_receipts"]) != 1 {
						t.Fatal("correlated result not consumed")
					}
					if cut != "settle" && endpointState.Contacts != 2 {
						t.Fatal("idempotent original-key recovery missing")
					}
				}
				if cut == "settle" {
					if err := checkAsyncTrace(events, held, peer, true); err != nil {
						t.Fatal(err)
					}
				}
				if failures := checkRecords(stopped, after, endpointState); len(failures) > 0 {
					t.Fatal(failures)
				}
				replayUnchanged(t, db, resourcePath)
				emit(t, map[string]any{"type": "ASYNC", "scenario": t.Name(), "profile": profile, "cut": cut, "provision": p, "initial": initial, "peer_staged": staged, "causal_events": events, "before_release": beforeRelease, "stopped": stopped, "after": after, "endpoint": endpointState, "race_children": raceChildren, "executor_binary_sha256": binaryHash(c.cmd.Path), "resource_binary_sha256": binaryHash(endpoint.cmd.Path)})
			})
		}
	}
}
func TestAsyncOracleRejectsFalseOrderAndIdentity(t *testing.T) {
	endpoint := barrier{Scenario: "scenario", Token: "endpoint", PID: 1, Store: "resource", Subject: "client", Generation: 1, Epoch: 2, Sequence: 1, Intent: "op", Attempt: "auth"}
	runner := endpoint
	runner.Token = "runner"
	runner.PID = 2
	runner.Store = "runtime"
	peer := runner
	peer.InFlight = []string{endpoint.Attempt}
	peer.Subject = "ready-peer"
	peer.Boundary = "execution.committed"
	peer.Intent = ""
	peer.Attempt = ""
	caller := peer
	caller.Subject = "caller"
	live := endpoint
	live.Phase = "holding"
	original := []asyncEvent{{"endpoint-held", endpoint}, {"peer-committed", peer}, {"peer-delivered", caller}, {"held-live", live}}
	if err := checkAsyncTrace(original, endpoint, runner, false); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func([]asyncEvent){
		func(e []asyncEvent) { e[0], e[1] = e[1], e[0] },
		func(e []asyncEvent) { e[0].Boundary.Intent = "different" },
		func(e []asyncEvent) { e[3].Boundary.Attempt = "other" },
		func(e []asyncEvent) { e[2].Boundary.PID = 99 },
		func(e []asyncEvent) { e[1].Boundary.Boundary = "controller.commit" },
		func(e []asyncEvent) { e[3].Name = "transport-timeout" },
		func(e []asyncEvent) { e[2].Boundary.InFlight = nil },
		func(e []asyncEvent) { e[2].Name = "final-state-only" },
	} {
		events := append([]asyncEvent(nil), original...)
		mutate(events)
		if checkAsyncTrace(events, endpoint, runner, false) == nil {
			t.Fatal("accepted false causal evidence", events)
		}
	}
}
