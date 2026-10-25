package durabletest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func startResource(t *testing.T, paused bool) (*child, string, string) {
	t.Helper()
	dir := binaries(t)
	path := filepath.Join(dir, fmt.Sprintf("resource-%d.sqlite", time.Now().UnixNano()))
	token := fmt.Sprintf("endpoint-%d", time.Now().UnixNano())
	config := barrier{Scenario: t.Name(), Token: token, Store: path, Boundary: "resource.applied", Phase: "post-commit", Subject: "client", Generation: 1, Epoch: 2, Sequence: 1, Intent: "", Attempt: `["authorization","[\"client\",1,1,0]"]`}
	read, write, _ := os.Pipe()
	resumeRead, resumeWrite, _ := os.Pipe()
	cmd := exec.Command(filepath.Join(dir, "alan-resource"), "--db", path, "--token", token, "--listen", "127.0.0.1:0")
	if paused {
		raw, _ := json.Marshal(config)
		cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	}
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.CreateTemp(dir, "resource-stderr-*")
	if err != nil {
		t.Fatal(err)
	}
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
	type ready struct {
		URL, Token, Store string
		PID               int
	}
	done := make(chan ready, 1)
	go func() { var value ready; json.NewDecoder(stdout).Decode(&value); done <- value }()
	select {
	case value := <-done:
		if value.PID != cmd.Process.Pid || value.Token != token || value.Store != path || value.URL == "" {
			t.Fatalf("resource identity %+v", value)
		}
		return c, value.URL, path
	case <-time.After(30 * time.Second):
		t.Fatal("no endpoint readiness")
	}
	return nil, "", ""
}
func TestSeparateResourceHTTPRetainsReceiptAcrossDuplicate(t *testing.T) {
	_, url, path := startResource(t, false)
	request := resource.Request{Key: "public-key", Profile: "idempotent", Delta: 9}
	raw, _ := json.Marshal(request)
	var first resource.Receipt
	for n := 0; n < 2; n++ {
		response, err := http.Post(url, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		var receipt resource.Receipt
		err = json.NewDecoder(response.Body).Decode(&receipt)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			first = receipt
		} else if receipt != first {
			t.Fatal("changed original outcome")
		}
	}
	snapshot, err := resource.Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Total != 9 || len(snapshot.Applications) != 1 || snapshot.Contacts != 2 {
		t.Fatal(snapshot)
	}
}

func resourceProvision(t *testing.T, url, profile string) executor.Provision {
	t.Helper()
	p := executor.Provision{}
	addProgram(t, &p, "client", []executor.Instruction{
		{Op: "emit", Dst: "pending", Effect: &executor.Effect{Kind: "external", Target: executor.Lit(executor.Text(url)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("receipt")), Payload: map[string]executor.Operand{"profile": executor.Lit(executor.Text(profile)), "delta": executor.Lit(executor.Int(5))}}},
		{Op: "receive", Match: &executor.Match{Kind: "receipt", Provenance: "resource-receipt", Source: executor.Lit(executor.Text(url)), Generation: executor.Lit(executor.Int(1)), Obligation: executor.Lit(executor.Text("receipt")), Operation: executor.Ref("pending")}, Bind: map[string]string{"total": "total"}},
		{Op: "end", Outcome: "succeeded"}}, map[string]executor.Value{})
	addProgram(t, &p, "ready-peer", []executor.Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]executor.Value{})
	p.Permissions = []executor.Permission{{Kind: "external", Source: "client", Target: url, Operation: "apply", Revision: 1}}
	return p
}
func addProgram(t *testing.T, p *executor.Provision, id string, code []executor.Instruction, data map[string]executor.Value) {
	t.Helper()
	raw, hash, err := executor.EncodeProgram(executor.Program{Semantics: executor.SemanticVersion, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, executor.Artifact{ID: hash, Semantics: executor.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, executor.Subject{ID: id, Generation: 1, Owner: "owner:" + id, Revision: 1})
	p.Runs = append(p.Runs, executor.InitialRun{ID: executor.RunID{Subject: id, Generation: 1}, ProgramID: hash, Projection: executor.Projection{Status: "open", Data: data}})
}
func TestAutomaticExternalAuthorizationCrash(t *testing.T) {
	_, url, resourcePath := startResource(t, false)
	p := resourceProvision(t, url, "idempotent")
	db := provisionCLI(t, p)
	initial := inspect(t, db)
	c := startPaused(t, db, "external.authorized", "pre-transport", "client", 1)
	ack := c.wait(t)
	c.kill(t)
	stopped := inspect(t, db)
	endpoint, err := resource.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Contacts != 0 || len(stopped.Tables["external_attempts"]) != 1 || len(stopped.Tables["external_receipts"]) != 0 {
		t.Fatal("authorization cut was not before transport")
	}
	cli(t, "run", "--db", db)
	after := inspect(t, db)
	endpoint, err = resource.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoint.Applications) != 1 {
		t.Fatal(endpoint)
	}
	if failures := checkRecords(stopped, after, endpoint); len(failures) > 0 {
		t.Fatal(failures)
	}
	replayUnchanged(t, db, resourcePath)
	emit(t, map[string]any{"scenario": t.Name(), "provision": p, "initial": initial, "barrier": ack, "stopped": stopped, "after": after, "endpoint": endpoint, "assertions": []string{"authorization before transport", "one application", "original intent retained", "read-only replay untouched endpoint"}})
}
func TestAutomaticApplicationMissingReceiptProfiles(t *testing.T) {
	for _, profile := range []string{"idempotent", "non-idempotent"} {
		t.Run(profile, func(t *testing.T) {
			endpointProcess, url, resourcePath := startResource(t, true)
			p := resourceProvision(t, url, profile)
			db := provisionCLI(t, p)
			initial := inspect(t, db)
			endpointProcess.config.Intent = executor.ExternalKey(initial.Namespace, `["client",1,1,0]`)
			c := startPaused(t, db, "external.response", "pre-record", "client", 1)
			appliedAck := endpointProcess.wait(t)
			applied, err := resource.Inspect(t.Context(), resourcePath)
			if err != nil {
				t.Fatal(err)
			}
			beforeResponse := inspect(t, db)
			if len(applied.Applications) != 1 || len(beforeResponse.Tables["external_receipts"]) != 0 {
				t.Fatal("not durable application/missing receipt")
			}
			if _, err = endpointProcess.release.Write([]byte{'c'}); err != nil {
				t.Fatal(err)
			}
			ack := c.wait(t)
			if executor.ExternalKey(initial.Namespace, ack.Intent) != appliedAck.Intent || ack.Attempt != appliedAck.Attempt {
				t.Fatal("uncorrelated endpoint/executor evidence")
			}
			c.kill(t)
			stopped := inspect(t, db)
			replayUnchanged(t, db, resourcePath)
			cli(t, "run", "--db", db)
			after := inspect(t, db)
			endpoint, err := resource.Inspect(t.Context(), resourcePath)
			if err != nil {
				t.Fatal(err)
			}
			if len(endpoint.Applications) != 1 {
				t.Fatal("duplicate automatic mutation")
			}
			client, _, _ := projection(after, "client", 1)
			peer, _, _ := projection(after, "ready-peer", 1)
			if peer.Status != "succeeded" {
				t.Fatal("unknown external blocked ready peer")
			}
			if profile == "idempotent" {
				if endpoint.Contacts != 2 || client.Status != "succeeded" || client.Data["total"] != executor.Int(5) || len(after.Tables["external_receipts"]) != 1 {
					t.Fatal("idempotent recovery not resolved")
				}
			} else {
				if endpoint.Contacts != 1 || client.Status != "open" || len(after.Tables["external_receipts"]) != 0 {
					t.Fatal("non-idempotent unknown retried/erased")
				}
			}
			if failures := checkRecords(stopped, after, endpoint); len(failures) > 0 {
				t.Fatal(failures)
			}
			emit(t, map[string]any{"scenario": t.Name(), "provision": p, "initial": initial, "endpoint_barrier": appliedAck, "executor_barrier": ack, "application_before_response": applied, "stopped": stopped, "after": after, "endpoint": endpoint, "assertions": []string{"endpoint COMMIT before acknowledgment", "executor SIGKILL before receipt", "original authorization and key", "no automatic duplicate"}})
			if profile == "non-idempotent" {
				raw, _ := json.Marshal(endpoint.Requests[0])
				response, err := http.Post(url, "application/json", bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				unsafe, err := resource.Inspect(t.Context(), resourcePath)
				if err != nil {
					t.Fatal(err)
				}
				if len(unsafe.Applications) != 2 || unsafe.Total != 10 {
					t.Fatal("unsafe repetition not observable")
				}
				emit(t, map[string]any{"scenario": t.Name(), "experiment": "explicit unsafe non-idempotent repetition", "request": endpoint.Requests[0], "before": endpoint, "after": unsafe})
			}
		})
	}
}
