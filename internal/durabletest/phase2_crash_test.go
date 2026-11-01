//go:build integration

package durabletest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

func phaseProvision(t *testing.T, route, endpoint string, peer bool) executor.Provision {
	t.Helper()
	if route == "adapter" {
		d, err := effectboundary.New(effectboundary.Config{Delta: 1, Peer: peer})
		if err != nil {
			t.Fatal(err)
		}
		return d.Provision
	}
	p := resourceProvision(t, endpoint, route)
	if !peer {
		p.Programs = p.Programs[:1]
		p.Subjects = p.Subjects[:1]
		p.Runs = p.Runs[:1]
	}
	return p
}
func phaseCreate(t *testing.T, p executor.Provision, path string) {
	t.Helper()
	s, err := executor.Create(t.Context(), path, p, executor.DefaultLimits())
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
func phaseResultCount(s executor.Inspection, route string) int {
	if route == "adapter" {
		return len(s.Tables["adapter_results"])
	}
	return len(s.Tables["external_receipts"])
}
func phaseAttempt(s executor.Inspection, route string) (table, key string) {
	if route == "adapter" {
		return "adapter_attempts", "adapter_results"
	}
	return "external_attempts", "external_receipts"
}
func phaseRequest(t *testing.T, s executor.Inspection, route string) (string, string) {
	t.Helper()
	table, _ := phaseAttempt(s, route)
	if len(s.Tables[table]) != 1 {
		t.Fatal("expected one retained attempt")
	}
	raw := s.Tables[table][0]["descriptor"].(string)
	if route == "adapter" {
		var a executor.AdapterAttempt
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			t.Fatal(err)
		}
		if !a.Allowed {
			t.Fatal(a)
		}
		return a.Prepared.Request.Key, raw
	}
	var a executor.ExternalAttempt
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		t.Fatal(err)
	}
	if !a.Allowed {
		t.Fatal(a)
	}
	return a.Request.Key, raw
}
func phaseJournal(t *testing.T, path, label string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(phaseEvidenceDir(t), phaseName(t)+"-"+label+".jsonl")
	if err = os.WriteFile(out, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPhase2PairedCrashCuts(t *testing.T) {
	for _, route := range []string{"idempotent", "non-idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			endpoint, endpointURL, journal := phaseResource(t, filepath.Join(dir, "world-U0.sqlite"), "", "", nil)
			address := phaseAddress(t, endpointURL)
			base := filepath.Join(dir, "base.sqlite")
			phaseCreate(t, phaseProvision(t, route, endpointURL, false), base)
			initial := inspect(t, base)
			dbs := []string{filepath.Join(dir, "U0.sqlite"), filepath.Join(dir, "U1.sqlite")}
			for _, db := range dbs {
				phaseCopy(t, base, db)
			}
			var cuts [2]executor.Inspection
			for branch := 0; branch < 2; branch++ {
				label := fmt.Sprintf("U%d", branch)
				world := filepath.Join(dir, "world-"+label+".sqlite")
				if branch == 1 {
					endpoint, _, journal = phaseResource(t, world, address, "", nil)
				}
				beforeWorld := phaseSnapshot(t, world)
				if beforeWorld.Contacts != 0 || len(beforeWorld.Applications) != 0 {
					t.Fatal("branch world not initially empty")
				}
				prefix := "external"
				if route == "adapter" {
					prefix = "adapter"
				}
				cut := barrier{Boundary: prefix + ".authorized", Phase: "pre-transport", Subject: "client"}
				if branch == 1 {
					cut.Boundary = prefix + ".response"
					cut.Phase = "pre-record"
				}
				c := phaseProcess(t, dbs[branch], endpointURL, "success", &cut)
				ack := phaseWait(t, c)
				c.kill(t)
				cuts[branch] = inspect(t, dbs[branch])
				cutWorld := phaseSnapshot(t, world)
				key, request := phaseRequest(t, cuts[branch], route)
				if phaseResultCount(cuts[branch], route) != 0 || len(cuts[branch].Tables["settlements"]) != 0 || len(cutWorld.Applications) != branch {
					t.Fatal("not required unknown/world cut")
				}
				if branch == 1 && cutWorld.Applications[0].Key != key {
					t.Fatal("application key mismatch")
				}
				stoppedLocal := phaseRawStore(t, dbs[branch], label+"-cut-local")
				stoppedWorld := phaseRawStore(t, world, label+"-cut-world")
				phaseReplay(t, dbs[branch], world, label+"-cut")
				recovered := phaseProcess(t, dbs[branch], endpointURL, "success", nil)
				phaseFinish(t, recovered)
				if recovered.cmd.Process.Pid == c.cmd.Process.Pid {
					t.Fatal("recovery reused process")
				}
				after := inspect(t, dbs[branch])
				end := phaseSnapshot(t, world)
				client, epoch, _ := projection(after, "client", 1)
				if epoch != ack.Epoch+1 {
					t.Fatal("recovery did not renew epoch")
				}
				transmissions := int(end.Contacts - cutWorld.Contacts)
				if route == "idempotent" {
					if transmissions != 1 || len(end.Applications) != 1 || phaseResultCount(after, route) != 1 || client.Status != "succeeded" {
						t.Fatal("supported idempotent progress absent")
					}
					if branch == 1 && !reflect.DeepEqual(cutWorld.Applications, end.Applications) {
						t.Fatal("duplicate changed original result")
					}
				} else if transmissions != 0 || len(end.Applications) != branch || phaseResultCount(after, route) != 0 || client.Status != "open" {
					t.Fatal("uncertain operation repeated or fabricated")
				}
				phaseReplay(t, dbs[branch], world, label+"-recovered")
				phaseEmit(t, label, map[string]any{"route": route, "branch": label, "initial": initial, "endpoint_initial": beforeWorld, "barrier": ack, "crash_PID": c.cmd.Process.Pid, "recovery_PID": recovered.cmd.Process.Pid, "signal": "SIGKILL", "source_request": request, "key": key, "cut": cuts[branch], "world_cut": cutWorld, "after": after, "world_after": end, "recovery_transmissions": transmissions, "consumer_status": client.Status, "cut_local_store": stoppedLocal, "cut_world_store": stoppedWorld, "after_local_store": phaseRawStore(t, dbs[branch], label+"-after-local"), "after_world_store": phaseRawStore(t, world, label+"-after-world"), "journal": phaseJournal(t, journal, label+"-ingress"), "executor_binary_sha256": binaryHash(c.cmd.Path), "endpoint_binary_sha256": binaryHash(endpoint.cmd.Path), "race_children": raceChildren})
				phaseStopEndpoint(t, endpoint)
			}
			if !reflect.DeepEqual(cuts[0], cuts[1]) {
				phaseEmit(t, "REJECTED-pair", cuts)
				t.Fatal("full retained cut inspections differ; no abstraction permitted")
			}
			phaseEmit(t, "pair", map[string]any{"U0": cuts[0], "U1": cuts[1], "comparison": "full Inspection deep equality; no field erased or renamed", "equal": true})
		})
	}
}

func TestPhase2KnownEvidenceCrashCuts(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		for _, stage := range []string{"recorded", "settled"} {
			t.Run(route+"/"+stage, func(t *testing.T) {
				dir := t.TempDir()
				endpoint, endpointURL, journal := phaseResource(t, filepath.Join(dir, "world.sqlite"), "", "", nil)
				db := filepath.Join(dir, "local.sqlite")
				phaseCreate(t, phaseProvision(t, route, endpointURL, false), db)
				prefix := "external"
				if route == "adapter" {
					prefix = "adapter"
				}
				cut := barrier{Boundary: prefix + ".result-recorded", Phase: "pre-settlement", Subject: "client"}
				if stage == "settled" {
					cut.Boundary = prefix + ".settled"
					cut.Phase = "post-record"
					if route == "adapter" {
						cut.Phase = "pre-consumption"
					}
				}
				c := phaseProcess(t, db, endpointURL, "success", &cut)
				ack := phaseWait(t, c)
				c.kill(t)
				stopped := inspect(t, db)
				worldCut := phaseSnapshot(t, filepath.Join(dir, "world.sqlite"))
				client, _, _ := projection(stopped, "client", 1)
				wantSettlements := 0
				if stage == "settled" {
					wantSettlements = 1
				}
				if phaseResultCount(stopped, route) != 1 || len(stopped.Tables["settlements"]) != wantSettlements || client.Status != "open" || len(worldCut.Applications) != 1 {
					t.Fatal("known cut invalid")
				}
				cutLocal := phaseRawStore(t, db, "cut-local")
				cutWorld := phaseRawStore(t, filepath.Join(dir, "world.sqlite"), "cut-world")
				phaseReplay(t, db, filepath.Join(dir, "world.sqlite"), "known-cut")
				c2 := phaseProcess(t, db, endpointURL, "success", nil)
				phaseFinish(t, c2)
				after := inspect(t, db)
				end := phaseSnapshot(t, filepath.Join(dir, "world.sqlite"))
				client, _, _ = projection(after, "client", 1)
				if client.Status != "succeeded" || phaseResultCount(after, route) != 1 || len(after.Tables["settlements"]) != 1 || !reflect.DeepEqual(worldCut, end) {
					t.Fatal("known result recovery contacted endpoint or failed consumption")
				}
				phaseReplay(t, db, filepath.Join(dir, "world.sqlite"), "known-after")
				phaseEmit(t, "known", map[string]any{"route": route, "stage": stage, "barrier": ack, "crash_PID": c.cmd.Process.Pid, "recovery_PID": c2.cmd.Process.Pid, "signal": "SIGKILL", "stopped": stopped, "after": after, "world_cut": worldCut, "world_after": end, "cut_local_store": cutLocal, "cut_world_store": cutWorld, "after_local_store": phaseRawStore(t, db, "after-local"), "after_world_store": phaseRawStore(t, filepath.Join(dir, "world.sqlite"), "after-world"), "journal": phaseJournal(t, journal, "ingress"), "executor_binary_sha256": binaryHash(c.cmd.Path), "endpoint_binary_sha256": binaryHash(endpoint.cmd.Path), "race_children": raceChildren})
			})
		}
	}
}

func phasePost(t *testing.T, endpoint string, request any) (resource.Receipt, int) {
	t.Helper()
	raw, _ := json.Marshal(request)
	response, err := http.Post(endpoint, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var receipt resource.Receipt
	if response.StatusCode == http.StatusOK {
		if err = json.NewDecoder(response.Body).Decode(&receipt); err != nil {
			t.Fatal(err)
		}
	}
	return receipt, response.StatusCode
}
func TestPhase2EndpointRestartBindingAndHistoricalReceipt(t *testing.T) {
	dir := t.TempDir()
	world := filepath.Join(dir, "world.sqlite")
	endpoint, endpointURL, journal := phaseResource(t, world, "", "", nil)
	request := resource.Request{Key: "original-business-attempt", Profile: "idempotent", Delta: 5}
	first, status := phasePost(t, endpointURL, request)
	if status != 200 {
		t.Fatal(status)
	}
	phaseStopEndpoint(t, endpoint)
	endpoint, secondURL, journal := phaseResource(t, world, phaseAddress(t, endpointURL), "", nil)
	if secondURL != endpointURL {
		t.Fatal("restart changed endpoint")
	}
	duplicate, status := phasePost(t, secondURL, request)
	if status != 200 || first != duplicate {
		t.Fatal("endpoint restart lost original result")
	}
	altered := request
	altered.Delta = 6
	_, status = phasePost(t, secondURL, altered)
	if status != 409 {
		t.Fatal("same-key changed payload accepted")
	}
	next := request
	next.Key = "new-business-attempt"
	later, status := phasePost(t, secondURL, next)
	if status != 200 || later.Total != 10 || later.Mutation != 2 {
		t.Fatal("new business attempt comparator failed")
	}
	retained, status := phasePost(t, secondURL, request)
	if status != 200 || retained != first {
		t.Fatal("historical receipt rewritten")
	}
	endpointState := phaseSnapshot(t, world)
	if endpointState.Total != 10 || len(endpointState.Applications) != 2 || endpointState.Contacts != 4 {
		t.Fatal("incorrect contacts/applications")
	}
	phaseEmit(t, "endpoint-controls", map[string]any{"request": request, "first": first, "after_restart": duplicate, "altered_request": altered, "altered_status": 409, "new_business_request": next, "later_receipt": later, "historical_receipt_after_interference": retained, "world": endpointState, "world_store": phaseRawStore(t, world, "world"), "journal": phaseJournal(t, journal, "ingress"), "endpoint_binary_sha256": binaryHash(endpoint.cmd.Path), "race_children": raceChildren})
}

func TestPhase2AdapterRealErrorAndSelectedUnsuccessful(t *testing.T) {
	for _, mode := range []string{"success", "selected-error", "unknown", "drop-after-commit"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			endpointMode := "normal"
			adapterMode := mode
			if mode == "drop-after-commit" {
				endpointMode = mode
				adapterMode = "success"
			}
			_, endpointURL, journal := phaseResource(t, filepath.Join(dir, "world.sqlite"), "", endpointMode, nil)
			db := filepath.Join(dir, "local.sqlite")
			phaseCreate(t, phaseProvision(t, "adapter", endpointURL, false), db)
			c := phaseProcess(t, db, endpointURL, adapterMode, nil)
			phaseFinish(t, c)
			after := inspect(t, db)
			world := phaseSnapshot(t, filepath.Join(dir, "world.sqlite"))
			client, _, _ := projection(after, "client", 1)
			if len(world.Applications) != 1 || world.Contacts != 1 {
				t.Fatal("no real application")
			}
			results := phaseResultCount(after, "adapter")
			if mode == "unknown" {
				if results != 0 || client.Status != "open" {
					t.Fatal("unknown fabricated result")
				}
			} else if results != 1 || client.Status == "open" {
				t.Fatal("known candidate not consumed")
			}
			phaseReplay(t, db, filepath.Join(dir, "world.sqlite"), mode)
			phaseEmit(t, "result", map[string]any{"adapter_mode": adapterMode, "endpoint_mode": endpointMode, "after": after, "world": world, "local_store": phaseRawStore(t, db, "local"), "world_store": phaseRawStore(t, filepath.Join(dir, "world.sqlite"), "world"), "journal": phaseJournal(t, journal, "ingress"), "process_PID": c.cmd.Process.Pid, "executor_binary_sha256": binaryHash(c.cmd.Path), "race_children": raceChildren})
		})
	}
}

func TestPhase2ParentAdapterCrash(t *testing.T) {
	for _, stage := range []string{"unknown", "recorded"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			worldPath := filepath.Join(dir, "world.sqlite")
			_, endpointURL, journal := phaseResource(t, worldPath, "", "", nil)
			deployment, err := effectboundary.New(effectboundary.Config{Parent: true, Delta: 1})
			if err != nil {
				t.Fatal(err)
			}
			db := filepath.Join(dir, "local.sqlite")
			phaseCreate(t, deployment.Provision, db)
			cut := barrier{Boundary: "adapter.authorized", Phase: "pre-transport"}
			if stage == "recorded" {
				cut.Boundary = "adapter.result-recorded"
				cut.Phase = "pre-settlement"
			}
			c := phaseProcess(t, db, endpointURL, "success", &cut)
			ack := phaseWait(t, c)
			c.kill(t)
			stopped := inspect(t, db)
			worldCut := phaseSnapshot(t, worldPath)
			cutLocal := phaseRawStore(t, db, "cut-local")
			cutWorld := phaseRawStore(t, worldPath, "cut-world")
			phaseReplay(t, db, worldPath, "parent-cut")
			fresh := phaseProcess(t, db, endpointURL, "success", nil)
			phaseFinish(t, fresh)
			after := inspect(t, db)
			worldAfter := phaseSnapshot(t, worldPath)
			root, _, _ := projection(after, "root", 1)
			if !reflect.DeepEqual(worldCut, worldAfter) {
				t.Fatal("parent recovery repeated physical request")
			}
			if stage == "unknown" {
				if len(worldAfter.Applications) != 0 || worldAfter.Contacts != 0 || len(after.Tables["adapter_attempts"]) != 1 || len(after.Tables["adapter_results"]) != 0 || root.Status != "open" || root.Data["knowledge"] != executor.Text("may_have_applied") {
					t.Fatal("parent unknown crash boundary invalid")
				}
				if _, exists := root.Data["child_status"]; exists {
					t.Fatal("fabricated child outcome")
				}
			} else if len(worldAfter.Applications) != 1 || len(after.Tables["adapter_results"]) != 1 || root.Status != "succeeded" || root.Data["knowledge"] != executor.Text("historical_application_receipt") {
				t.Fatal("parent known crash recovery did not complete")
			}
			phaseReplay(t, db, worldPath, "parent-after")
			journalPath := ""
			if _, err := os.Stat(journal); err == nil {
				journalPath = phaseJournal(t, journal, "ingress")
			}
			phaseEmit(t, "parent-crash", map[string]any{"stage": stage, "barrier": ack, "crash_PID": c.cmd.Process.Pid, "recovery_PID": fresh.cmd.Process.Pid, "signal": "SIGKILL", "stopped": stopped, "after": after, "world_cut": worldCut, "world_after": worldAfter, "cut_local_store": cutLocal, "cut_world_store": cutWorld, "after_local_store": phaseRawStore(t, db, "after-local"), "after_world_store": phaseRawStore(t, worldPath, "after-world"), "journal": journalPath, "executor_binary_sha256": binaryHash(c.cmd.Path), "race_children": raceChildren})
		})
	}
}

// The journal flag selects a separate bounded observation profile. With no
// journal, preserve the historical streaming decoder's treatment of padding
// beyond its 1MiB window rather than silently hardening the baseline protocol.
func TestPhase2EndpointLegacyPaddedJSON(t *testing.T) {
	_, endpointURL, worldPath := startResource(t, false)
	request := resource.Request{Key: "legacy-padding", Profile: "idempotent", Delta: 1}
	raw, _ := json.Marshal(request)
	raw = append(raw, bytes.Repeat([]byte{' '}, (1<<20)+1)...)
	response, err := http.Post(endpointURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("legacy padded request status %d", response.StatusCode)
	}
	snapshot := phaseSnapshot(t, worldPath)
	if len(snapshot.Applications) != 1 || snapshot.Total != 1 {
		t.Fatal("legacy request not applied")
	}
	phaseEmit(t, "legacy-padding", map[string]any{"body_bytes": len(raw), "http_status": response.StatusCode, "world": snapshot})
}

func TestPhase2EndpointLegacyMethodRejectBeforeBody(t *testing.T) {
	_, endpointURL, worldPath := startResource(t, false)
	connection, err := net.Dial("tcp", phaseAddress(t, endpointURL))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err = connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// Deliberately omit the declared body. A handler that reads it before method
	// validation cannot reply; Connection: close avoids server-side body draining.
	if _, err = fmt.Fprint(connection, "GET /unsupported HTTP/1.1\r\nHost: localhost\r\nContent-Length: 4\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("legacy rejection status %d", response.StatusCode)
	}
	snapshot := phaseSnapshot(t, worldPath)
	if snapshot.Contacts != 0 || len(snapshot.Applications) != 0 {
		t.Fatal("unsupported request applied")
	}
	phaseEmit(t, "legacy-method", map[string]any{"http_status": response.StatusCode, "declared_body_bytes": 4, "sent_body_bytes": 0, "world": snapshot})
}
