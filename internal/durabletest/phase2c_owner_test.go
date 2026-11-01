//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

var resolver2c = runtime.RunID{Subject: "resolver", Generation: 1}

type command2c struct {
	Op     string
	Source runtime.RunID
}

type reply2c struct {
	command2c
	PID       int
	Namespace string
	Resolver  runtime.RunState
	Original  *runtime.RunState
	Report    *runtime.RunReport
}

// TestPhase2COwner is the actual process serving the retained Store. Its command
// pipe can schedule an empty input or ask for a current observation/Q.
func TestPhase2COwner(t *testing.T) {
	if os.Getenv("ALAN_PHASE2_CHILD") != "1" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	path := os.Getenv("ALAN_PHASE2_DB")
	hooks, err := boundary.FromEnvironment(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	apply := os.Getenv("ALAN_PHASE2_ENDPOINT")
	lookup, err := rq.LookupEndpoint(apply)
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]effects.Adapter{rq.CounterCapability: rq.Counter{Endpoint: apply, Mode: "success"}, rq.ReadCapability: rq.Reader{Endpoint: lookup}}
	park := make(chan struct{}, 1)
	runCtx := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "runner.waiting" {
			select {
			case park <- struct{}{}:
			default:
			}
		}
		if os.Getenv("ALAN_BARRIER") == "" {
			return nil
		}
		return boundary.Reached(boundary.Inherit(ctx, hooks), e)
	})
	type result struct {
		report runtime.RunReport
		err    error
	}
	var running chan result
	control := json.NewDecoder(os.Stdin)
	acks := os.NewFile(5, "acknowledgments")
	defer acks.Close()
	encode := json.NewEncoder(acks)
	request := command2c{Op: "open"}
	for {
		reply := reply2c{command2c: request, PID: os.Getpid(), Namespace: s.Namespace()}
		switch request.Op {
		case "open", "exit":
		case "run":
			if running != nil {
				t.Fatal("duplicate active runner")
			}
			done := make(chan result, 1)
			running = done
			go func() {
				r, e := s.Run(runCtx, runtime.RunOptions{MaxInFlight: 1, MaxSteps: 240, MaxDispatch: 240, ExternalTimeout: 75 * time.Second, Adapters: registry})
				done <- result{r, e}
			}()
		case "finish":
			if running == nil {
				t.Fatal("no runner")
			}
			select {
			case r := <-running:
				if r.err != nil {
					t.Fatal(r.err)
				}
				reply.Report = &r.report
				running = nil
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		case "closed":
			select {
			case <-park:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		case "Q":
			if err := s.AwaitDispatchQuiescence(ctx, runtime.DispatchAuthority{Namespace: s.Namespace(), Run: request.Source}); err != nil {
				t.Fatal(err)
			}
		case "start", "refresh":
			id := "service-" + request.Op
			got, err := s.AdmitInput(ctx, runtime.Input{TransportID: id, LogicalID: id, Target: resolver2c.Subject, Generation: 1, Obligation: request.Op, Payload: map[string]runtime.Value{}})
			if err != nil || !got.Accepted {
				t.Fatal(got, err)
			}
		default:
			t.Fatal("unknown host command", request)
		}
		if request.Source.Subject != "" {
			state, e := s.Snapshot(ctx, request.Source)
			if e != nil || state.Projection.Status != "succeeded" {
				t.Fatal("expected original closure", state, e)
			}
			reply.Original = &state
		}
		reply.Resolver, err = s.Snapshot(ctx, resolver2c)
		if err != nil {
			t.Fatal(err)
		}
		if err = encode.Encode(reply); err != nil {
			t.Fatal(err)
		}
		if request.Op == "exit" {
			return
		}
		request = command2c{}
		if err = control.Decode(&request); err != nil {
			t.Fatal(err)
		}
	}
}

type owner2c struct {
	*child
	commands *os.File
	replies  *os.File
}

func pipe2c(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	return r, w
}

func (h *history2c) owner(cut *barrier) *owner2c {
	t := h.t
	t.Helper()
	reqR, reqW := pipe2c(t)
	ackR, ackW := pipe2c(t)
	c := phaseTestProcess(t, h.db, h.apply, "success", cut, "TestPhase2COwner", func(cmd *exec.Cmd) {
		cmd.Stdin = reqR
		cmd.ExtraFiles = append(cmd.ExtraFiles, ackW)
	})
	reqR.Close()
	ackW.Close()
	t.Cleanup(func() { reqW.Close(); ackR.Close() })
	o := &owner2c{child: c, commands: reqW, replies: ackR}
	open := h.command(o, "open")
	if h.ownerPID != 0 && (open.PID == h.ownerPID || open.Resolver.Epoch != h.ownerEpoch+1) {
		t.Fatal("successor activation", h.ownerPID, h.ownerEpoch, open)
	}
	h.ownerPID, h.ownerEpoch = open.PID, open.Resolver.Epoch
	return o
}

func (h *history2c) command(o *owner2c, op string) reply2c {
	t := h.t
	t.Helper()
	req := command2c{Op: op}
	if op == "closed" || op == "Q" {
		req.Source = h.source
	}
	if op != "open" {
		if e := json.NewEncoder(o.commands).Encode(req); e != nil {
			t.Fatal(e)
		}
	}
	var reply reply2c
	if e := o.replies.SetReadDeadline(time.Now().Add(45 * time.Second)); e != nil {
		t.Fatal(e)
	}
	if e := json.NewDecoder(o.replies).Decode(&reply); e != nil {
		raw, _ := os.ReadFile(o.stderr.Name())
		t.Fatalf("%s: %v %s", op, e, raw)
	}
	if reply.command2c != req || reply.PID != o.cmd.Process.Pid || reply.Namespace == "" || reply.Resolver.ID != resolver2c {
		t.Fatal("wrong owner reply", reply)
	}
	if req.Source.Subject != "" && (reply.Original == nil || reply.Original.ID != req.Source || reply.Original.Projection.Status != "succeeded") {
		t.Fatal("wrong Q/closure target", reply)
	}
	h.event("command", reply)
	return reply
}

func (h *history2c) kill(o *owner2c) {
	o.kill(h.t)
	h.event("SIGKILL", map[string]any{"PID": o.cmd.Process.Pid, "Wait": o.cmd.ProcessState.String(), "Binary": binaryHash(o.cmd.Path)})
}
func (h *history2c) finish(o *owner2c) reply2c {
	result := h.command(o, "finish")
	h.command(o, "exit")
	phaseFinish(h.t, o.child)
	return result
}

func cut2c(name, phase, subject string) *barrier {
	return &barrier{Boundary: name, Phase: phase, Subject: subject}
}

// Worker events retain the emission sequence while the source can legitimately
// reach its terminal sequence on another scheduler turn.
func (h *history2c) await(c *child) barrier {
	t := h.t
	t.Helper()
	if e := c.ack.SetReadDeadline(time.Now().Add(40 * time.Second)); e != nil {
		t.Fatal(e)
	}
	var b barrier
	if e := json.NewDecoder(c.ack).Decode(&b); e != nil {
		raw, _ := os.ReadFile(c.stderr.Name())
		t.Fatalf("cut %s: %v %s", c.config.Boundary, e, raw)
	}
	f := c.config
	if b.Token != f.Token || b.Scenario != f.Scenario || b.Store != f.Store || b.PID != c.cmd.Process.Pid || b.Boundary != f.Boundary || b.Phase != f.Phase || (f.Subject != "" && b.Subject != f.Subject) {
		t.Fatal("uncorrelated barrier", b, f)
	}
	if b.Store == h.db {
		var subject, authorization string
		var sequence, generation int64
		e := h.query(h.db, `SELECT json_extract(i.descriptor,'$.Run.Subject'),json_extract(i.descriptor,'$.Run.Generation'),json_extract(i.descriptor,'$.Sequence'),a.id FROM intents i JOIN attempts a ON a.intent_id=i.id WHERE i.id=? AND json_extract(a.result,'$.Allowed')=1`, b.Intent).Scan(&subject, &generation, &sequence, &authorization)
		if e != nil || subject != b.Subject || generation != b.Generation || sequence != b.Sequence || authorization != b.Attempt {
			t.Fatal("uncorrelated retained intent/authorization", b, e)
		}
		if b.Subject == resolver2c.Subject {
			var ordinal int
			var epoch int64
			var target string
			err := h.query(h.db, `SELECT ordinal,json_extract(descriptor,'$.Resolver.Epoch'),json_extract(descriptor,'$.Query.TargetID') FROM receipt_read_ordinals WHERE intent_id=? ORDER BY ordinal DESC LIMIT 1`, b.Intent).Scan(&ordinal, &epoch, &target)
			if err != nil || ordinal < 1 || ordinal > 2 || target != h.original || epoch != h.ownerEpoch || b.Epoch != epoch {
				t.Fatal("unbound physical opportunity", b, ordinal, epoch, target, err)
			}
		} else if int64(h.count(h.db, `SELECT json_extract(result,'$.Source.Epoch') FROM attempts WHERE id=?`, b.Attempt)) != b.Epoch {
			t.Fatal("wrong original activation", b)
		}
	}
	c.acknowledged = true
	h.event("barrier", b)
	return b
}

func copyFile2c(t *testing.T, src, dst string) {
	t.Helper()
	in, e := os.Open(src)
	if e != nil {
		t.Fatal(e)
	}
	defer in.Close()
	out, e := os.Create(dst)
	if e != nil {
		t.Fatal(e)
	}
	_, e = io.Copy(out, in)
	if e = errors.Join(e, out.Close()); e != nil {
		t.Fatal(e)
	}
}
