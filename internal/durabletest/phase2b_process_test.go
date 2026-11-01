//go:build integration

package durabletest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/boundary"
	x "github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
)

var clientOne = runtime.RunID{Subject: "client", Generation: 1}

type controlReply struct {
	PID         int
	AnchorEpoch int64
	Command     string
	Client      runtime.RunState
	Report      *runtime.RunReport
}

// The extra pipe is a host command channel. Its replies observe real Store
// transitions and never write rows or provide evidence to the ordinary program.
func TestPhase2BControlledExecutor(t *testing.T) {
	if os.Getenv("ALAN_PHASE2_CHILD") != "1" {
		return
	}
	db := os.Getenv("ALAN_PHASE2_DB")
	ctx, stop := context.WithTimeout(context.Background(), 90*time.Second)
	defer stop()
	barriers, err := boundary.FromEnvironment(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Open(ctx, db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	parked := make(chan struct{}, 1)
	runCtx := boundary.WithObserver(ctx, func(c context.Context, e boundary.Event) error {
		if e.Boundary == "runner.waiting" {
			select {
			case parked <- struct{}{}:
			default:
			}
		}
		if os.Getenv("ALAN_BARRIER") == "" {
			return nil
		}
		return boundary.Reached(boundary.Inherit(c, barriers), e)
	})
	acks := os.NewFile(5, "command-acks")
	defer acks.Close()
	ack := func(command string, report *runtime.RunReport) {
		client, err := s.Snapshot(ctx, clientOne)
		if err != nil {
			t.Fatal(err)
		}
		anchor, err := s.Snapshot(ctx, runtime.RunID{Subject: "anchor", Generation: 1})
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(acks).Encode(controlReply{PID: os.Getpid(), Command: command, Client: client, Report: report, AnchorEpoch: anchor.Epoch}); err != nil {
			t.Fatal(err)
		}
	}
	ack("open", nil)
	type runEnd struct {
		r   runtime.RunReport
		err error
	}
	var finished chan runEnd
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		command := reader.Text()
		var report *runtime.RunReport
		switch command {
		case "start":
			if finished != nil {
				t.Fatal("duplicate Run")
			}
			finished = make(chan runEnd, 1)
			go func() {
				r, err := s.Run(runCtx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, ExternalTimeout: 60 * time.Second, Adapters: effectboundary.Registry(os.Getenv("ALAN_PHASE2_ENDPOINT"), "success")})
				finished <- runEnd{r, err}
			}()
		case "finish":
			select {
			case result := <-finished:
				if result.err != nil {
					t.Fatal(result.err)
				}
				report = &result.r
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		case "revoke-automatic":
			admitControl(t, s, "operator", "client")
			// Ask the ordinary runner to complete its control work. Inspect only
			// after it has had a scheduler turn, outside its Store transaction.
			for {
				select {
				case <-parked:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				state, err := s.Snapshot(ctx, runtime.RunID{Subject: "operator", Generation: 1})
				if err != nil {
					t.Fatal(err)
				}
				if state.Projection.Status == "succeeded" {
					break
				}
			}
		case "revoke-manual":
			decision := manualControl(t, s, "operator", "client")
			if decision.Reason != "closed" {
				t.Fatal(decision)
			}
		case "arm-crash":
			state, err := s.Snapshot(ctx, clientOne)
			if err != nil {
				t.Fatal(err)
			}
			if state.Projection.Status != "cancelled" {
				t.Fatal("crash cut before revocation")
			}
			err = boundary.Reached(barriers, boundary.Event{Boundary: "fixture.revoked", Phase: "pre-kill", Subject: clientOne.Subject, Generation: clientOne.Generation, Epoch: state.Epoch, Sequence: state.Sequence})
			if err != nil {
				t.Fatal(err)
			}
		case "quiesce":
			if err := s.AwaitDispatchQuiescence(ctx, runtime.DispatchAuthority{Namespace: s.Namespace(), Run: clientOne}); err != nil {
				t.Fatal(err)
			}
		case "exit":
			ack(command, nil)
			return
		default:
			t.Fatal("unknown host command", command)
		}
		ack(command, report)
	}
	t.Fatal("unexpected command pipe EOF", reader.Err())
}

type controlledOwner struct {
	*child
	request, response *os.File
	order             []controlReply
}

func startControlled(t *testing.T, db, endpoint string, cut *barrier) *controlledOwner {
	t.Helper()
	reqR, reqW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ackR, ackW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	c := phaseTestProcess(t, db, endpoint, "success", cut, "TestPhase2BControlledExecutor", func(cmd *exec.Cmd) {
		cmd.Stdin = reqR
		cmd.ExtraFiles = append(cmd.ExtraFiles, ackW)
	})
	reqR.Close()
	ackW.Close()
	t.Cleanup(func() { reqW.Close(); ackR.Close() })
	owner := &controlledOwner{child: c, request: reqW, response: ackR}
	owner.command(t, "open")
	return owner
}
func (c *controlledOwner) command(t *testing.T, command string) controlReply {
	t.Helper()
	if command != "open" {
		if _, err := fmt.Fprintln(c.request, command); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.response.SetReadDeadline(time.Now().Add(35 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var reply controlReply
	if err := json.NewDecoder(c.response).Decode(&reply); err != nil {
		raw, _ := os.ReadFile(c.stderr.Name())
		t.Fatalf("%s acknowledgment %v: %s", command, err, raw)
	}
	if reply.PID != c.cmd.Process.Pid || reply.Command != command || reply.Client.ID != clientOne {
		t.Fatal("command identity", reply)
	}
	c.order = append(c.order, reply)
	return reply
}
func assertClosed(t *testing.T, reply controlReply) {
	t.Helper()
	if reply.Client.Projection.Status != "cancelled" {
		t.Fatal("genuine source cancellation absent", reply)
	}
}
func phaseBStoreCut(t *testing.T, db, world, label string) {
	t.Helper()
	phaseRawStore(t, db, label+"-local")
	phaseRawStore(t, world, label+"-world")
	phaseJournal(t, world+".ingress.jsonl", label+"-ingress")
}

// This disabled, open receiver has no lifecycle or endpoint authority. Its
// activation witnesses an actual new Open even though closed epochs are stable.
func addPhase2BControls(t *testing.T, p *runtime.Provision) {
	addOperator(t, p, "operator", "CancelRun", "client")
	addProgram(t, p, "anchor", []x.Instruction{gateInstruction(), {Op: "end", Outcome: "succeeded"}}, map[string]x.Value{})
}
func distinctOwners(t *testing.T, old, next *controlledOwner) {
	t.Helper()
	if old.cmd.Process.Pid == next.cmd.Process.Pid || next.order[0].AnchorEpoch != old.order[0].AnchorEpoch+1 || next.order[0].Client.Epoch != old.order[0].Client.Epoch {
		t.Fatal("owner and activation identity", old.order, next.order)
	}
}
func TestPhase2BRevokedCrashPair(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.sqlite")
	world := filepath.Join(dir, "U0-world.sqlite")
	endpoint, url, _ := phaseResource(t, world, "", "", nil)
	address := phaseAddress(t, url)
	p := phaseProvision(t, "idempotent", url, false)
	addPhase2BControls(t, &p)
	phaseCreate(t, p, base)
	var cuts, recovered [2]runtime.Inspection
	for branch := range 2 {
		label := fmt.Sprintf("U%d", branch)
		if branch == 1 {
			world = filepath.Join(dir, label+"-world.sqlite")
			endpoint, _, _ = phaseResource(t, world, address, "", nil)
		}
		db := filepath.Join(dir, label+"-local.sqlite")
		phaseCopy(t, base, db)
		cut := barrier{Boundary: "dispatch.admitted", Phase: "pre-call", Subject: "client"}
		if branch == 1 {
			cut.Boundary, cut.Phase = "external.response", "pre-record"
		}
		first := startControlled(t, db, url, &cut)
		first.command(t, "start")
		ack := phaseWait(t, first.child)
		assertClosed(t, first.command(t, "revoke-automatic"))
		first.kill(t)
		cuts[branch] = inspect(t, db)
		before := phaseSnapshot(t, world)
		key, retained := phaseRequest(t, cuts[branch], "idempotent")
		if len(before.Applications) != branch || phaseResultCount(cuts[branch], "idempotent") != 0 {
			t.Fatal("bad closed cut")
		}
		phaseBStoreCut(t, db, world, label+"-cut")
		phaseReplay(t, db, world, label+"-cut")
		next := startControlled(t, db, url, nil)
		distinctOwners(t, first, next)
		assertClosed(t, next.command(t, "quiesce"))
		next.command(t, "start")
		report := next.command(t, "finish").Report
		next.command(t, "quiesce")
		next.command(t, "exit")
		phaseFinish(t, next.child)
		recovered[branch] = inspect(t, db)
		_, afterRequest := phaseRequest(t, recovered[branch], "idempotent")
		after := phaseSnapshot(t, world)
		if first.cmd.Process.Pid == next.cmd.Process.Pid || len(report.UnknownExternal) != 1 || phaseResultCount(recovered[branch], "idempotent") != 0 || retained != afterRequest || !reflect.DeepEqual(before, after) {
			t.Fatal("recovery changed unknown", report, before, after)
		}
		phaseBStoreCut(t, db, world, label+"-after")
		phaseReplay(t, db, world, label+"-after")
		phaseEmit(t, label, map[string]any{"key": key, "barrier": ack, "signal": "SIGKILL", "crashed": first.order, "reopened": next.order, "world_cut": before, "world_after": after})
		phaseStopEndpoint(t, endpoint)
	}
	if !reflect.DeepEqual(cuts[0], cuts[1]) || !reflect.DeepEqual(recovered[0], recovered[1]) {
		t.Fatal("full local Inspections differ")
	}
}

func TestPhase2BRecordedRecovery(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			world, db := filepath.Join(dir, "world.sqlite"), filepath.Join(dir, "local.sqlite")
			_, url, _ := phaseResource(t, world, "", "", nil)
			p := phaseProvision(t, route, url, false)
			addPhase2BControls(t, &p)
			phaseCreate(t, p, db)
			prefix := "external"
			if route == "adapter" {
				prefix = route
			}
			owner := startControlled(t, db, url, &barrier{Boundary: prefix + ".result-recorded", Phase: "pre-settlement", Subject: "client"})
			owner.command(t, "start")
			ack := phaseWait(t, owner.child)
			assertClosed(t, owner.command(t, "revoke-manual"))
			owner.kill(t)
			before := inspect(t, db)
			originalWorld := phaseSnapshot(t, world)
			if phaseResultCount(before, route) != 1 || len(before.Tables["settlements"]) != 0 || len(phaseSnapshot(t, world).Applications) != 1 {
				t.Fatal("bad retained cut")
			}
			phaseBStoreCut(t, db, world, "cut")
			phaseReplay(t, db, world, "cut")
			next := startControlled(t, db, url, nil)
			distinctOwners(t, owner, next)
			if len(inspect(t, db).Tables["settlements"]) != 0 {
				t.Fatal("Open had already settled cut")
			}
			assertClosed(t, next.command(t, "quiesce"))
			next.command(t, "exit")
			phaseFinish(t, next.child)
			after := inspect(t, db)
			_, resultTable := phaseAttempt(before, route)
			if !reflect.DeepEqual(originalWorld, phaseSnapshot(t, world)) || !reflect.DeepEqual(before.Tables[resultTable], after.Tables[resultTable]) || len(after.Tables["settlements"]) != 1 {
				t.Fatal("known evidence not settled exactly once")
			}
			phaseBStoreCut(t, db, world, "after")
			phaseReplay(t, db, world, "after")
			phaseEmit(t, "known", map[string]any{"barrier": ack, "signal": "SIGKILL", "crashed": owner.order, "reopened": next.order})
		})
	}
}

func TestPhase2BDelayedApplication(t *testing.T) {
	dir := t.TempDir()
	world, db := filepath.Join(dir, "world.sqlite"), filepath.Join(dir, "local.sqlite")
	endpoint, url, _ := phaseResource(t, world, "", "finish-accepted", &barrier{Boundary: "resource.ingress", Phase: "pre-apply"})
	p := phaseProvision(t, "idempotent", url, false)
	addPhase2BControls(t, &p)
	phaseCreate(t, p, db)
	first := startControlled(t, db, url, &barrier{Boundary: "fixture.revoked", Phase: "pre-kill", Subject: "client"})
	first.command(t, "start")
	ingress := phaseWait(t, endpoint)
	assertClosed(t, first.command(t, "revoke-automatic"))
	// A second owner is refused while the first still holds the real flock.
	before := inspect(t, db)
	refused, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
	if err == nil {
		refused.Close()
		t.Fatal("second owner acquired live store")
	}
	if !errors.Is(err, runtime.ErrLocked) {
		t.Fatal("unexpected second Open error", err)
	}
	if !reflect.DeepEqual(before, inspect(t, db)) {
		t.Fatal("refused Open changed state")
	}
	key, descriptor := phaseRequest(t, before, "idempotent")
	if key != ingress.Intent || phaseSnapshot(t, world).Contacts != 0 {
		t.Fatal("endpoint cut not before application")
	}
	// A still-held endpoint request blocks this worker. The verified ingress
	// and host command reply jointly identify the live executor being killed.
	if _, err := fmt.Fprintln(first.request, "arm-crash"); err != nil {
		t.Fatal(err)
	}
	crashAck := phaseWait(t, first.child)
	first.kill(t)
	phaseBStoreCut(t, db, world, "cut")
	phaseReplay(t, db, world, "cut")
	next := startControlled(t, db, url, nil)
	distinctOwners(t, first, next)
	assertClosed(t, next.command(t, "quiesce"))
	phaseBStoreCut(t, db, world, "acknowledged")
	if phaseSnapshot(t, world).Contacts != 0 {
		t.Fatal("packet applied before Q")
	}
	if _, err := endpoint.release.Write([]byte{'c'}); err != nil {
		t.Fatal(err)
	}
	limit := time.After(20 * time.Second)
	for phaseSnapshot(t, world).Contacts == 0 {
		select {
		case <-limit:
			t.Fatal("accepted packet did not apply")
		case <-time.After(2 * time.Millisecond):
		}
	}
	applied := phaseSnapshot(t, world)
	if len(applied.Applications) != 1 || applied.Applications[0].Key != key || applied.Applications[0].PID != ingress.PID {
		t.Fatal(applied)
	}
	next.command(t, "start")
	report := next.command(t, "finish").Report
	assertClosed(t, next.command(t, "quiesce"))
	next.command(t, "exit")
	phaseFinish(t, next.child)
	after := inspect(t, db)
	_, retained := phaseRequest(t, after, "idempotent")
	if next.cmd.Process.Pid == first.cmd.Process.Pid || retained != descriptor || len(report.UnknownExternal) != 1 || phaseResultCount(after, "idempotent") != 0 || !reflect.DeepEqual(applied, phaseSnapshot(t, world)) {
		t.Fatal("later receiver acquired impossible knowledge or sent another request", report)
	}
	phaseBStoreCut(t, db, world, "after")
	phaseReplay(t, db, world, "after")
	phaseEmit(t, "delayed", map[string]any{"key": key, "endpoint_ingress": ingress, "executor_barrier": crashAck, "signal": "SIGKILL", "crashed": first.order, "reopened": next.order, "live_open_refusal": err.Error(), "order": []string{"ingress", "actual revocation returned", "second Open refused", "SIGKILL and Wait", "fresh Open", "successful Q", "endpoint release", "original endpoint application", "no recovery transmission"}, "world_after": applied})
}
