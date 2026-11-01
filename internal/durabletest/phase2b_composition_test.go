//go:build integration

package durabletest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/boundary"
	x "github.com/JakeFAU/alan/internal/executor"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type compositionProbe struct {
	held    chan boundary.Event
	release func()
	done    chan struct{}
	events  eventTape
	report  rt.RunReport
	err     error
}

// The observer holds only the specified unlocked dispatch hook. Other events
// are recorded without entering the Store or waiting for scheduler service.
func runCompositionProbe(t *testing.T, store *rt.Store, endpoint, hold string) *compositionProbe {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Second)
	p := &compositionProbe{held: make(chan boundary.Event, 1), done: make(chan struct{})}
	release := make(chan struct{})
	var once, first sync.Once
	p.release = func() { once.Do(func() { close(release) }) }
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		p.events.add(e)
		if e.Boundary != hold {
			return nil
		}
		first.Do(func() {
			p.held <- e
			select {
			case <-release:
			case <-ctx.Done():
			}
		})
		return ctx.Err()
	})
	go func() {
		defer close(p.done)
		p.report, p.err = store.Run(ctx, rt.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1, Adapters: effectboundary.Registry(endpoint, "success")})
	}()
	t.Cleanup(func() { p.release(); cancel(); <-p.done })
	return p
}
func (p *compositionProbe) cut(t *testing.T) boundary.Event {
	t.Helper()
	select {
	case e := <-p.held:
		return e
	case <-p.done:
		t.Fatalf("missing dispatch hold: %v", p.err)
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	return boundary.Event{}
}
func (p *compositionProbe) finish(t *testing.T) {
	t.Helper()
	p.release()
	<-p.done
	if p.err != nil {
		t.Fatal(p.err)
	}
}
func compositionStore(t *testing.T, p rt.Provision) (*rt.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "local.sqlite")
	s, err := rt.Create(t.Context(), path, p, rt.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, path
}
func publicStep(t *testing.T, s *rt.Store, run rt.RunID, obs string) rt.RunState {
	t.Helper()
	state, err := s.Snapshot(t.Context(), run)
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.Commit(t.Context(), state.Authority(), obs)
	if err != nil {
		t.Fatal(err)
	}
	return state
}
func invokeEntry(t *testing.T, s *rt.Store, path, operation string) rt.RunID {
	t.Helper()
	a, err := s.AdmitInvocation(t.Context(), operation)
	if err != nil || !a.Accepted || len(a.Members) != 1 {
		t.Fatal(a, err)
	}
	entry := a.Members[0]
	for _, o := range inspect(t, path).Tables["observations"] {
		if o["subject"] == entry.Subject && o["generation"] == entry.Generation && o["consumed_sequence"] == nil {
			publicStep(t, s, entry, o["id"].(string))
			return entry
		}
	}
	t.Fatal("invocation request missing", a)
	return rt.RunID{}
}
func dispatchTarget(s *rt.Store, r rt.RunID, scope bool) rt.DispatchAuthority {
	return rt.DispatchAuthority{Namespace: s.Namespace(), Run: r, OwningScope: scope}
}
func untilTerminal(t *testing.T, s *rt.Store, run rt.RunID) rt.RunState {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		state, err := s.Snapshot(ctx, run)
		if err != nil {
			t.Fatal(err)
		}
		if state.Projection.Status != "open" {
			return state
		}
		runtime.Gosched()
	}
	t.Fatal("terminal transition not observed", run)
	return rt.RunState{}
}

// Outer explicitly reports issuance. Its reply has no field for application.
func nestedDispatchFixture(t *testing.T, fail bool) rt.Provision {
	t.Helper()
	leaf, err := effectboundary.New(effectboundary.Config{Parent: true, Delta: 1})
	if err != nil {
		t.Fatal(err)
	}
	in := schema.Record{"delta": {Kind: "int"}}
	out := schema.Record{"issued": {Kind: "bool"}}
	steps := []author.Step{author.Receive("caller", map[string]string{"delta": "delta"}), author.Invoke("inner", "leaf", map[string]author.Expr{"delta": author.Var("delta")}, nil)}
	if fail {
		steps = append(steps, author.Failure())
	} else {
		steps = append(steps, author.Send("caller", map[string]author.Expr{"issued": author.Boolean(true)}), author.Success())
	}
	frame, err := author.Compile(author.Definition{Name: "phase2b-frame", Ports: map[string]author.Port{"caller": {Peer: "caller", Obligation: "frame", Receive: in, Send: out}}, Invocations: map[string]author.Template{"leaf": leaf.Child}, Steps: steps})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := author.NewTemplate(author.Application{"entry": {Artifact: frame, Inputs: map[string]schema.Value{}}}, "entry", "caller")
	if err != nil {
		t.Fatal(err)
	}
	root, err := author.Compile(author.Definition{Name: "phase2b-nested-parent", Invocations: map[string]author.Template{"frame": outer}, Steps: []author.Step{
		author.Invoke("outer", "frame", map[string]author.Expr{"delta": author.Integer(1)}, nil),
		author.AwaitInvocation("frame", "outer", author.InvocationOutcomeBindings{Status: "status", Reason: "reason", ResponsePresent: "present", Response: map[string]string{"issued": "issued"}}),
		author.Success(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"root": {Artifact: root, Inputs: map[string]schema.Value{}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	rootID := rt.RunID{Subject: "root", Generation: 1}
	p.Composition.Roots = []rt.ResourceRoot{{Run: rootID, Members: []rt.RunID{rootID}, Templates: []string{outer.ID(), leaf.Child.ID()}, Capabilities: []string{effectboundary.Capability}, Limits: rt.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 2, MaxEffects: 1, MaxInFlight: 1}}}
	return p
}

func TestPhase2BNestedAuthority(t *testing.T) {
	for _, status := range []string{"succeeded", "failed"} {
		t.Run(status, func(t *testing.T) {
			fail := status == "failed"
			world := filepath.Join(t.TempDir(), "counter.sqlite")
			endpoint, url, journal := phaseResource(t, world, "", "normal", nil)
			s, path := compositionStore(t, nestedDispatchFixture(t, fail))
			root := rt.RunID{Subject: "root", Generation: 1}
			outerOp := publicStep(t, s, root, "").Projection.Data["outer"].Text
			frame := invokeEntry(t, s, path, outerOp)
			leafOp := publicStep(t, s, frame, "").Projection.Data["inner"].Text
			leaf := invokeEntry(t, s, path, leafOp)
			issued := publicStep(t, s, leaf, "")
			probe := runCompositionProbe(t, s, url, "dispatch.consider")
			cut := probe.cut(t)
			if cut.Intent != issued.Projection.Data["operation"].Text {
				t.Fatal(cut, issued)
			}
			if !fail {
				publicStep(t, s, frame, "")
			}
			terminal := publicStep(t, s, frame, "")
			prepared, err := s.PrepareInvocation(t.Context(), outerOp)
			if err != nil || prepared.Freeze.Status != status || terminal.Projection.Status != status {
				t.Fatal(prepared, err)
			}
			before := inspect(t, path)
			if len(before.Tables["composition_publications"]) != 0 {
				t.Fatal("outer already published")
			}
			scope := dispatchTarget(s, frame, true)
			q := s.AwaitDispatchQuiescence(t.Context(), scope)
			if fail && q != nil || !fail && !errors.Is(q, rt.ErrDispatchAuthorityOpen) {
				t.Fatal("wrong owning-scope observation", q)
			}
			if err := s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, frame, false)); err != nil {
				t.Fatal("terminal frame source", err)
			}
			probe.finish(t)
			after := inspect(t, path)
			leafState, _, _ := projection(after, leaf.Subject, 1)
			rootState, _, _ := projection(after, "root", 1)
			remote := worldAt(t, world)
			if rootState.Status != "succeeded" || rootState.Data["status"].Text != status || rootState.Data["present"].Flag == fail {
				t.Fatal(rootState)
			}
			if fail {
				if remote.Contacts != 0 || leafState.Status != "open" || len(probe.report.UnknownAdapters) != 1 {
					t.Fatal(remote, leafState, probe.report)
				}
			} else if remote.Contacts != 1 || len(remote.Applications) != 1 || leafState.Status != "succeeded" || !leafState.Data["valid"].Flag || leafState.Data["total"].Number != 1 || !rootState.Data["issued"].Flag {
				t.Fatal(remote, leafState, rootState)
			}
			if len(after.Roots) != 1 || after.Roots[0].Charges.Effects != 1 {
				t.Fatal(after.Roots)
			}
			raw, err := os.ReadFile(journal)
			if err != nil && !(fail && os.IsNotExist(err)) {
				t.Fatal(err)
			}
			captureAuthority(t, map[string]any{"case": status, "target": scope, "cut": cut, "prepared": prepared, "before": before, "after": after, "q_open": errors.Is(q, rt.ErrDispatchAuthorityOpen), "events": probe.events.ordered, "report": probe.report, "world": remote, "journal": string(raw), "pid": endpoint.cmd.Process.Pid})
		})
	}
}

func TestPhase2BPrivateFailureOrders(t *testing.T) {
	for _, hold := range []string{"dispatch.consider", "dispatch.admitted"} {
		t.Run(hold, func(t *testing.T) {
			world := filepath.Join(t.TempDir(), "counter.sqlite")
			_, url, journal := phaseResource(t, world, "", "normal", nil)
			d, err := effectboundary.New(effectboundary.Config{Parent: true, FailAfterRequest: true})
			if err != nil {
				t.Fatal(err)
			}
			s, path := compositionStore(t, d.Provision)
			p := runCompositionProbe(t, s, url, hold)
			cut := p.cut(t)
			child := rt.RunID{Subject: cut.Subject, Generation: cut.Generation}
			root := rt.RunID{Subject: "root", Generation: 1}
			if hold == "dispatch.consider" {
				closed := publicStep(t, s, child, "")
				if closed.Projection.Status != "failed" {
					t.Fatal(closed)
				}
				if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, child, false)); err != nil {
					t.Fatal(err)
				}
			} else {
				// Run itself performs private failure, publication and original selection.
				parent := untilTerminal(t, s, root)
				if parent.Projection.Data["child_reason"].Text != "member_failed" {
					t.Fatal(parent)
				}
			}
			before := inspect(t, path)
			if len(before.Tables["adapter_results"]) != 0 || worldAt(t, world).Contacts != 0 {
				t.Fatal("effect already completed at hold")
			}
			p.finish(t)
			if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, child, false)); err != nil {
				t.Fatal(err)
			}
			after := inspect(t, path)
			parent, _, _ := projection(after, "root", 1)
			if parent.Status != "succeeded" || parent.Data["child_status"].Text != "failed" || parent.Data["child_reason"].Text != "member_failed" || parent.Data["child_response_present"].Flag || parent.Data["knowledge"].Text != "local_failure_world_unresolved" {
				t.Fatal(parent)
			}
			n := 0
			if hold == "dispatch.admitted" {
				n = 1
			}
			remote := worldAt(t, world)
			if len(remote.Applications) != n || len(after.Tables["adapter_results"]) != n {
				t.Fatal(remote, p.report)
			}
			raw, err := os.ReadFile(journal)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			captureAuthority(t, map[string]any{"case": hold, "cut": cut, "before": before, "after": after, "world": remote, "journal": string(raw), "report": p.report, "events": p.events.ordered})
		})
	}
}

func TestPhase2BSlowEffectAndPeers(t *testing.T) {
	world := filepath.Join(t.TempDir(), "counter.sqlite")
	endpoint, url, journal := phaseResource(t, world, "", "normal", &barrier{Boundary: "resource.ingress", Phase: "pre-apply"})
	d, err := effectboundary.New(effectboundary.Config{Peer: true})
	if err != nil {
		t.Fatal(err)
	}
	addOperator(t, &d.Provision, "operator", "AbandonRun", "client")
	addProgram(t, &d.Provision, "ordinary", []x.Instruction{gateInstruction(), {Op: "end", Outcome: "succeeded"}}, map[string]x.Value{})
	d.Provision.Permissions = append(d.Provision.Permissions, rt.Permission{Kind: "input", Source: "local", Target: "ordinary", Operation: "input", Obligation: "control"})
	s, path := compositionStore(t, d.Provision)
	p := runCompositionProbe(t, s, url, "")
	ack := phaseWait(t, endpoint)
	if ack.Intent != x.ExternalKey(s.Namespace(), `["client",1,1,0]`) {
		t.Fatal("wrong original key at ingress", ack)
	}
	admitControl(t, s, "operator", "client")
	admitControl(t, s, "ordinary", "work")
	target := rt.RunID{Subject: "client", Generation: 1}
	if result := untilTerminal(t, s, target); result.Projection.Status != "abandoned" {
		t.Fatal(result)
	}
	untilTerminal(t, s, rt.RunID{Subject: "ordinary", Generation: 1})
	before := inspect(t, path)
	if worldAt(t, world).Contacts != 0 || len(before.Tables["adapter_results"]) != 0 {
		t.Fatal("capacity hold ended early")
	}
	wait, cancel := context.WithCancel(t.Context())
	wait = boundary.WithObserver(wait, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.waiting" {
			cancel()
		}
		return ctx.Err()
	})
	err = s.AwaitDispatchQuiescence(wait, dispatchTarget(s, target, false))
	cancel()
	if !errors.Is(err, context.Canceled) {
		t.Fatal("Q acknowledged an in-flight worker", err)
	}
	if _, err = endpoint.release.Write([]byte("c")); err != nil {
		t.Fatal(err)
	}
	p.finish(t)
	if err = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, target, false)); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, path)
	peer, _, _ := projection(after, "peer", 1)
	remote := worldAt(t, world)
	if peer.Status != "succeeded" || !peer.Data["valid"].Flag || len(remote.Applications) != 2 || remote.Applications[0].Delta != 1 || remote.Applications[1].Delta != 2 {
		t.Fatal(peer, remote)
	}
	raw, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	captureAuthority(t, map[string]any{"ingress_ack": ack, "before": before, "after": after, "q_interrupted": true, "report": p.report, "events": p.events.ordered, "world": remote, "journal": string(raw)})
}

func TestPhase2BNestedDrainCoversDescendant(t *testing.T) {
	world := filepath.Join(t.TempDir(), "counter.sqlite")
	_, url, journal := phaseResource(t, world, "", "normal", nil)
	s, path := compositionStore(t, nestedDispatchFixture(t, true))
	root := rt.RunID{Subject: "root", Generation: 1}
	frame := invokeEntry(t, s, path, publicStep(t, s, root, "").Projection.Data["outer"].Text)
	leaf := invokeEntry(t, s, path, publicStep(t, s, frame, "").Projection.Data["inner"].Text)
	original := publicStep(t, s, leaf, "").Projection.Data["operation"].Text
	p := runCompositionProbe(t, s, url, "dispatch.admitted")
	cut := p.cut(t)
	if cut.Intent != original {
		t.Fatal(cut)
	}
	if state := untilTerminal(t, s, frame); state.Projection.Status != "failed" {
		t.Fatal(state)
	}
	if err := s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, frame, false)); err != nil {
		t.Fatal("frame source alone has no worker", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	observed := false
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "dispatch.waiting" {
			observed = true
			cancel()
		}
		return ctx.Err()
	})
	scope := dispatchTarget(s, frame, true)
	err := s.AwaitDispatchQuiescence(ctx, scope)
	cancel()
	if !observed || !errors.Is(err, context.Canceled) {
		t.Fatal("outer-scope Q missed inner descendant", err)
	}
	before := inspect(t, path)
	p.finish(t)
	if err = s.AwaitDispatchQuiescence(t.Context(), scope); err != nil {
		t.Fatal(err)
	}
	after := inspect(t, path)
	state, _, _ := projection(after, leaf.Subject, leaf.Generation)
	remote := worldAt(t, world)
	if state.Status != "succeeded" || !state.Data["valid"].Flag || state.Data["total"].Number != 1 || len(remote.Applications) != 1 {
		t.Fatal(state, remote)
	}
	raw, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	captureAuthority(t, map[string]any{"scope": scope, "source": leaf, "original": original, "cut": cut, "before": before, "after": after, "scope_pending": observed, "world": remote, "journal": string(raw), "report": p.report, "events": p.events.ordered})
}
