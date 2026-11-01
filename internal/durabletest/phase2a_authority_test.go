//go:build integration

package durabletest

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	eb "github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/boundary"
	x "github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

type authorityCase struct {
	Route, Cut, Target, Operation string
	Member                        bool
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

func addOperator(t *testing.T, p *runtime.Provision, name, operation, target string) {
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
func admitControl(t *testing.T, s *runtime.Store, name, target string) string {
	t.Helper()
	a, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: name, LogicalID: name, Target: name, Generation: 1, Obligation: "control", Payload: map[string]x.Value{"target": x.Text(target)}})
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
	return a.ObservationID
}
func stepRun(t *testing.T, s *runtime.Store, name, observation string) x.RunState {
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
func manualControl(t *testing.T, s *runtime.Store, name, target string) x.LifecycleResult {
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
func worldAt(t *testing.T, path string) resource.Snapshot {
	t.Helper()
	world, err := resource.Inspect(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return world
}
func captureAuthority(t *testing.T, data map[string]any) {
	t.Helper()
	dir := os.Getenv("ALAN_PHASE2A_AUTHORITY")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "-")+".json.gz")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if err := errors.Join(json.NewEncoder(gz).Encode(data), gz.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
}

type eventTape struct {
	mu      sync.Mutex
	ordered []any
}

func (e *eventTape) add(v any) { e.mu.Lock(); defer e.mu.Unlock(); e.ordered = append(e.ordered, v) }

func TestPhase2AAuthority(t *testing.T) {
	for _, route := range []string{"external", "adapter"} {
		for _, v := range []authorityCase{
			{Cut: "before-permission", Operation: "CancelRun"}, {Cut: "before-permission", Operation: "RetireSubject"},
			{Cut: "after-permission"}, {Cut: "after-permission", Member: true}, {Cut: "after-permission", Member: true, Target: "scope"}, {Cut: "after-permission", Member: true, Target: "scope", Operation: "RetireSubject"},
			{Cut: "worker"}, {Cut: "worker", Member: true}, {Cut: "worker", Member: true, Target: "scope"}, {Cut: "worker", Operation: "RetireSubject"},
			{Cut: "ingress"}, {Cut: "response"}, {Cut: "context"}, {Cut: "budget", Member: true},
		} {
			v.Route = route
			if v.Target == "" {
				v.Target = "client"
			}
			if v.Operation == "" {
				v.Operation = "CancelRun"
			}
			name := fmt.Sprintf("%s/%s/%s/%s/member-%v", route, v.Cut, v.Operation, v.Target, v.Member)
			t.Run(name, func(t *testing.T) { runAuthority(t, v) })
		}
	}
	t.Run("external/relaunch", func(t *testing.T) {
		runAuthority(t, authorityCase{"external", "relaunch", "client", "CancelRun", false})
	})
	t.Run("external/generation-retry", func(t *testing.T) { runAuthority(t, authorityCase{"external", "retry", "client", "CancelRun", false}) })
}

func runAuthority(t *testing.T, c authorityCase) {
	t.Helper()
	ctx, stop := context.WithTimeout(t.Context(), 25*time.Second)
	defer stop()
	path := filepath.Join(t.TempDir(), "resource.sqlite")
	var filter *barrier
	if c.Cut == "ingress" {
		filter = &barrier{Boundary: "resource.ingress", Phase: "pre-apply"}
	}
	if c.Cut == "relaunch" {
		filter = &barrier{Boundary: "resource.applied", Phase: "post-commit"}
	}
	endpoint, url, journal := phaseResource(t, path, "127.0.0.1:0", "normal", filter)
	var p runtime.Provision
	var adapters map[string]effects.Adapter
	pending := "pending"
	if c.Route == "external" {
		p = resourceProvision(t, url, "idempotent")
	} else {
		d, err := eb.New(eb.Config{Delta: 1})
		if err != nil {
			t.Fatal(err)
		}
		p = d.Provision
		adapters = eb.Registry(url, "success")
		pending = "operation"
	}
	addOperator(t, &p, "operator", c.Operation, c.Target)
	if c.Cut == "retry" {
		addOperator(t, &p, "retry", "Retry", "client")
		for i := range p.Subjects {
			if p.Subjects[i].ID == "client" {
				p.Subjects[i].RetryCancelled = true
			}
		}
	}
	if c.Member {
		addProgram(t, &p, "scope", []x.Instruction{gateInstruction(), {Op: "end", Outcome: "succeeded"}}, map[string]x.Value{})
		root := x.RunID{Subject: "scope", Generation: 1}
		maxEffects := int64(1)
		if c.Cut == "budget" {
			maxEffects = 0
		}
		p.Composition = &runtime.CompositionProvision{Roots: []runtime.ResourceRoot{{Run: root, Members: []x.RunID{root, {Subject: "client", Generation: 1}}, Capabilities: []string{eb.Capability}, Limits: runtime.ResourceLimits{MaxEffects: maxEffects, MaxInFlight: 1}}}}
	}
	db := filepath.Join(t.TempDir(), "authority.sqlite")
	s, err := runtime.Create(ctx, db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	issued := stepRun(t, s, "client", "")
	op := issued.Projection.Data[pending].Text
	if op == "" {
		t.Fatal("missing original operation", issued)
	}
	var tape eventTape
	rec := map[string]any{"case": c, "initial": inspect(t, db), "operation": op, "endpoint_argv": endpoint.cmd.Args, "endpoint_binary": binaryHash(endpoint.cmd.Path), "executor_pid": os.Getpid(), "race": raceChildren}
	captureCut := func(name string) { rec[name] = map[string]any{"local": inspect(t, db), "world": worldAt(t, path)} }
	opts := runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, ExternalTimeout: 15 * time.Second, Adapters: adapters}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	pause := ""
	switch c.Cut {
	case "worker", "context":
		pause = c.Route + ".authorized"
	case "response", "retry":
		pause = c.Route + ".response"
	}
	held := make(chan struct{})
	release := make(chan struct{})
	closed := make(chan struct{})
	var observed sync.Once
	observedCtx := boundary.WithObserver(runCtx, func(ctx context.Context, e boundary.Event) error {
		tape.add(e)
		if e.Boundary == "lifecycle.decided" && e.Subject == "operator" && e.Phase == "accepted" {
			observed.Do(func() { close(closed) })
		}
		if e.Boundary == pause && e.Intent == op {
			close(held)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	type end struct {
		r runtime.RunReport
		e error
	}
	done := make(chan end, 1)
	start := func() { go func() { r, e := s.Run(observedCtx, opts); done <- end{r, e} }() }
	wait := func(ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case v := <-done:
			t.Fatalf("runner ended before acknowledged cut: %v %+v", v.e, v.r)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	var result end
	if c.Cut == "before-permission" || c.Cut == "after-permission" || c.Cut == "budget" {
		if c.Cut == "after-permission" {
			a, e := s.AuthorizeIntent(ctx, op)
			if e != nil || !a.Allowed {
				t.Fatal(a, e)
			}
			rec["first_permission"] = a
		}
		if c.Cut != "budget" {
			rec["closure"] = manualControl(t, s, "operator", c.Target)
			captureCut("closed_before_run")
		}
		start()
	} else {
		start()
		if filter != nil {
			// Read the selected acknowledgment; its exact identity is checked against the
			// original operation and the independent world here, including zero adapter metadata.
			ack := phaseWait(t, endpoint)
			if ack.Boundary != filter.Boundary || ack.Phase != filter.Phase || ack.Intent != x.ExternalKey(inspect(t, db).Namespace, op) {
				t.Fatalf("bad endpoint identity %+v", ack)
			}
			tape.add(ack)
			rec["endpoint_barrier"] = ack
		} else {
			wait(held)
		}
		captureCut("held")
		if c.Cut == "context" {
			tape.add("cancel Run context")
			cancel()
		} else if c.Cut == "relaunch" {
			// The endpoint has committed. Cancelling just the runner loses its response,
			// and the same endpoint stays up through the subsequent Run.
			cancel()
			result = <-done
			rec["first_report"] = result.r
			if !errors.Is(result.e, context.Canceled) {
				t.Fatal("expected cancelled first Run", result.e)
			}
			if _, e := endpoint.release.Write([]byte("c")); e != nil {
				t.Fatal(e)
			}
			rec["closure"] = manualControl(t, s, "operator", c.Target)
			captureCut("closed_before_relaunch")
			r, e := s.Run(ctx, opts)
			done <- end{r, e}
		} else {
			admitControl(t, s, "operator", c.Target)
			wait(closed)
			captureCut("closed_held")
			tape.add("release after committed closure")
			if filter != nil {
				if _, e := endpoint.release.Write([]byte("c")); e != nil {
					t.Fatal(e)
				}
			} else {
				close(release)
			}
		}
	}
	select {
	case result = <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if c.Cut == "context" {
		if !errors.Is(result.e, context.Canceled) {
			t.Fatal(result.e)
		}
	} else if result.e != nil {
		t.Fatal(result.e)
	}
	rec["report"] = result.r
	if c.Cut == "retry" {
		captureCut("before_retry")
		decision := manualControl(t, s, "retry", "client")
		if decision.Successor != 2 {
			t.Fatal(decision)
		}
		rec["retry"] = decision
		r, e := s.Run(ctx, opts)
		if e != nil {
			t.Fatal(e)
		}
		rec["successor_report"] = r
	}
	final := inspect(t, db)
	world := worldAt(t, path)
	local, _, _ := projection(final, "client", 1)
	wantApps := 1
	if c.Cut == "before-permission" || c.Cut == "budget" || c.Cut == "context" || c.Cut == "after-permission" {
		wantApps = 0
	}
	if c.Cut == "retry" {
		wantApps = 2
	}
	if len(world.Applications) != wantApps {
		t.Fatalf("applications %d want %d, final %s", len(world.Applications), wantApps, local.Status)
	}
	if c.Cut == "context" && local.Status != "open" {
		t.Fatal("runner context closed logical source")
	}
	if c.Target == "client" && c.Cut != "budget" && c.Cut != "context" && local.Status != "cancelled" {
		t.Fatal("source closure missing", local)
	}
	if c.Cut == "relaunch" && world.Contacts != 1 {
		t.Fatal("post-cancellation re-exposure reached endpoint", world)
	}
	if c.Cut == "retry" && (world.Contacts != 2 || world.Requests[0].Key == world.Requests[1].Key || world.Requests[1].Generation != 2) {
		t.Fatal("retry identities", world)
	}
	if c.Member && len(final.Tables["composition_charges"]) != wantApps {
		t.Fatal("wrong member charges", final.Tables["composition_charges"])
	}
	raw, e := os.ReadFile(journal)
	if e != nil {
		t.Fatal(e)
	}
	rec["ingress_journal"] = string(raw)
	rec["world"] = world
	rec["final"] = final
	rec["order"] = tape.ordered
	if _, e = runtime.Replay(ctx, db, runtime.DefaultLimits()); e != nil {
		t.Fatal(e)
	}
	later, e := os.ReadFile(journal)
	if e != nil || !bytes.Equal(raw, later) || !reflect.DeepEqual(world, worldAt(t, path)) {
		t.Fatal("replay changed world", e)
	}
	rec["replay_world_unchanged"] = true
	captureAuthority(t, rec)
}
