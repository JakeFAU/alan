//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/boundary"
	x "github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
)

// All cases have one immutable member effect debit. The synchronous consider
// hook needs a separate lifecycle driver; the held worker leaves Run free.
func TestPhase2BLiveOrders(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		for _, c := range []struct{ label, stage, operation, target, reason string }{
			{"R-source", "dispatch.consider", "CancelRun", "client", "source_closed"},
			{"R-scope", "dispatch.consider", "CancelRun", "scope", "scope_closed"},
			{"R-abandon", "dispatch.consider", "AbandonRun", "client", "source_closed"},
			{"R-retire", "dispatch.consider", "RetireSubject", "client", "source_retired"},
			{"A-source", "dispatch.admitted", "CancelRun", "client", ""},
			{"A-scope", "dispatch.admitted", "CancelRun", "scope", ""},
			{"returned-unknown", "dispatch.admitted", "CancelRun", "client", ""},
			{"drop-after-commit", "dispatch.admitted", "CancelRun", "client", ""},
			{"queued-result", "runner.received", "CancelRun", "client", ""},
			{"recorded-result", "record", "CancelRun", "client", ""},
		} {
			if c.label == "returned-unknown" && route != "adapter" {
				continue
			}
			t.Run(route+"/"+c.label, func(t *testing.T) {
				ctx, stop := context.WithTimeout(t.Context(), 30*time.Second)
				defer stop()
				dir := t.TempDir()
				world, db := filepath.Join(dir, "world.sqlite"), filepath.Join(dir, "local.sqlite")
				endpointMode, adapterMode := "normal", "success"
				if c.label == "returned-unknown" {
					adapterMode = "unknown"
				}
				if c.label == "drop-after-commit" {
					endpointMode = c.label
				}
				_, url, journal := phaseResource(t, world, "", endpointMode, nil)
				p := phaseProvision(t, route, url, false)
				addOperator(t, &p, "operator", c.operation, c.target)
				addProgram(t, &p, "scope", []x.Instruction{gateInstruction(), {Op: "end", Outcome: "succeeded"}}, map[string]x.Value{})
				root := runtime.RunID{Subject: "scope", Generation: 1}
				p.Composition = &runtime.CompositionProvision{Roots: []runtime.ResourceRoot{{Run: root, Members: []runtime.RunID{root, clientOne}, Capabilities: []string{effectboundary.Capability}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}}}
				s, err := runtime.Create(ctx, db, p, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer s.Close()
				held, release := make(chan boundary.Event, 1), make(chan struct{})
				decision := make(chan struct{}, 1)
				var tape eventTape
				stage := c.stage
				if stage == "record" {
					prefix := "external"
					if route == "adapter" {
						prefix = route
					}
					stage = prefix + ".result-recorded"
				}
				observed := boundary.WithObserver(ctx, func(callCtx context.Context, e boundary.Event) error {
					tape.add(e)
					if e.Boundary == "lifecycle.decided" && e.Subject == "operator" && e.Phase == "accepted" {
						select {
						case decision <- struct{}{}:
						default:
						}
					}
					if e.Boundary == stage && e.Subject == "client" {
						held <- e
						select {
						case <-release:
							return nil
						case <-callCtx.Done():
							return callCtx.Err()
						}
					}
					return nil
				})
				finished := make(chan error, 1)
				go func() {
					_, err := s.Run(observed, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, ExternalTimeout: 20 * time.Second, Adapters: effectboundary.Registry(url, adapterMode)})
					finished <- err
				}()
				var reached boundary.Event
				select {
				case reached = <-held:
				case e := <-finished:
					t.Fatal("missing cut before exit", e)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
				initial := inspect(t, db)
				key, descriptor := phaseRequest(t, initial, route)
				initialWorld := worldAt(t, world)
				if len(initial.Tables["composition_charges"]) != 1 {
					t.Fatal("missing reservation charge")
				}
				if stage == "dispatch.consider" || stage == "dispatch.admitted" {
					if initialWorld.Contacts != 0 || phaseResultCount(initial, route) != 0 {
						t.Fatal("pre-call cut already executed")
					}
				} else {
					want := 0
					if c.stage == "record" {
						want = 1
					}
					if initialWorld.Contacts != 1 || phaseResultCount(initial, route) != want || len(initial.Tables["settlements"]) != 0 {
						t.Fatal("bad accounting cut")
					}
				}
				if stage == "dispatch.admitted" {
					admitControl(t, s, "operator", c.target)
					select {
					case <-decision:
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				} else {
					v := manualControl(t, s, "operator", c.target)
					if v.Before.Projection.Status != "open" || v.After.Projection.Status == "open" {
						t.Fatal("no effective closure", v)
					}
				}
				closed := inspect(t, db)
				target := runtime.DispatchAuthority{Namespace: s.Namespace(), Run: clientOne, OwningScope: c.target == "scope"}
				source, _, _ := projection(closed, "client", 1)
				if (c.target == "scope") != (source.Status == "open") {
					t.Fatal("wrong closure domain", source)
				}
				tape.add("R confirmed against durable source/scope")
				var pendingErr string
				if stage == "dispatch.consider" {
					if err := s.AwaitDispatchQuiescence(ctx, target); err != nil {
						t.Fatal(err)
					}
					tape.add("successful Q while consideration held before A")
				} else {
					waiting := make(chan struct{}, 1)
					waitCtx, cancel := context.WithCancel(boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
						if e.Boundary == "dispatch.waiting" && e.Phase == "in-flight" {
							tape.add(e)
							select {
							case waiting <- struct{}{}:
							default:
							}
						}
						return nil
					}))
					woke := make(chan error, 1)
					go func() { woke <- s.AwaitDispatchQuiescence(waitCtx, target) }()
					select {
					case <-waiting:
					case early := <-woke:
						t.Fatalf("Q passed held %s: %v", stage, early)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
					cancel()
					err := <-woke
					if !errors.Is(err, context.Canceled) {
						t.Fatal("unsuccessful Q result", err)
					}
					pendingErr = err.Error()
				}
				tape.add("release held opportunity")
				close(release)
				if err := <-finished; err != nil {
					t.Fatal(err)
				}
				if err := s.AwaitDispatchQuiescence(ctx, target); err != nil {
					t.Fatal(err)
				}
				tape.add("successful Q after Run handling")
				final := inspect(t, db)
				end := worldAt(t, world)
				_, afterDescriptor := phaseRequest(t, final, route)
				if descriptor != afterDescriptor || !reflect.DeepEqual(initial.Tables["composition_charges"], final.Tables["composition_charges"]) || !sameOriginalAuthorization(initial, final, reached.Intent) {
					t.Fatal("original attempt or semantic charge changed")
				}
				want := 1
				if stage == "dispatch.consider" {
					want = 0
				}
				wantResult := want
				if c.label == "returned-unknown" || (c.label == "drop-after-commit" && route == "idempotent") {
					wantResult = 0
				}
				if c.label == "drop-after-commit" && route == "adapter" {
					var result x.AdapterResult
					rows := final.Tables["adapter_results"]
					if len(rows) != 1 {
						t.Fatal("missing authentic error")
					}
					if err := json.Unmarshal([]byte(rows[0]["descriptor"].(string)), &result); err != nil {
						t.Fatal(err)
					}
					if result.Candidate.Disposition != "transport_error" {
						t.Fatal("wrong returned evidence", result)
					}
				}
				if len(end.Applications) != want || int(end.Contacts) != want || phaseResultCount(final, route) != wantResult {
					t.Fatalf("world/results disagree: wanted %d got %+v", want, end)
				}
				refused, admitted := 0, 0
				for _, item := range tape.ordered {
					e, ok := item.(boundary.Event)
					if !ok || e.Intent != reached.Intent {
						continue
					}
					if e.Boundary == "dispatch.admitted" {
						admitted++
					}
					if e.Boundary == "dispatch.refused" {
						if e.Phase != c.reason {
							t.Fatalf("reason %s expected %s", e.Phase, c.reason)
						}
						refused++
					}
				}
				if admitted != want || refused != 1-want {
					t.Fatal("gate order", admitted, refused, tape.ordered)
				}
				ingress, err := os.ReadFile(journal)
				if err != nil {
					t.Fatal(err)
				}
				phaseBStoreCut(t, db, world, "final")
				phaseEmit(t, "live", map[string]any{"route": route, "target": target, "operation": reached.Intent, "key": key, "events": tape.ordered, "held": initial, "world_held": initialWorld, "revoked": closed, "wait_error": pendingErr, "final": final, "world_final": end, "ingress": string(ingress)})
			})
		}
	}
}

// Retry is eligible on the ordinary unscoped source only. The old Q does
// not certify the later generation, whose distinct operation really applies.
func TestPhase2BSupersededOpportunity(t *testing.T) {
	ctx, stop := context.WithTimeout(t.Context(), 20*time.Second)
	defer stop()
	dir := t.TempDir()
	world, db := filepath.Join(dir, "world.sqlite"), filepath.Join(dir, "local.sqlite")
	_, url, _ := phaseResource(t, world, "", "", nil)
	p := phaseProvision(t, "idempotent", url, false)
	p.Subjects[0].RetryCancelled = true
	addOperator(t, &p, "operator", "CancelRun", "client")
	addOperator(t, &p, "retry", "Retry", "client")
	s, err := runtime.Create(ctx, db, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var tape eventTape
	held, release := make(chan struct{}), make(chan struct{})
	observed := boundary.WithObserver(ctx, func(c context.Context, e boundary.Event) error {
		tape.add(e)
		if e.Boundary == "dispatch.consider" && e.Subject == "client" && e.Generation == 1 {
			close(held)
			select {
			case <-release:
			case <-c.Done():
				return c.Err()
			}
		}
		return nil
	})
	finished := make(chan error, 1)
	go func() {
		_, err := s.Run(observed, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
		finished <- err
	}()
	select {
	case <-held:
	case err := <-finished:
		t.Fatal("missing gate cut", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	before := inspect(t, db)
	oldKey, _ := phaseRequest(t, before, "idempotent")
	cancel := manualControl(t, s, "operator", "client")
	retry := manualControl(t, s, "retry", "client")
	if cancel.Reason != "closed" || retry.Successor != 2 {
		t.Fatal(cancel, retry)
	}
	old := runtime.DispatchAuthority{Namespace: s.Namespace(), Run: clientOne}
	if err := s.AwaitDispatchQuiescence(ctx, old); err != nil {
		t.Fatal(err)
	}
	successor := old
	successor.Run.Generation = 2
	if err := s.AwaitDispatchQuiescence(ctx, successor); !errors.Is(err, runtime.ErrDispatchAuthorityOpen) {
		t.Fatal("old Q incorrectly covered successor", err)
	}
	tape.add("Q on G1 only; G2 still eligible; release G1 consideration")
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	end := inspect(t, db)
	w := worldAt(t, world)
	found := false
	for _, item := range tape.ordered {
		e, ok := item.(boundary.Event)
		if ok && e.Boundary == "dispatch.refused" && e.Generation == 1 && e.Phase == "source_generation_changed" {
			found = true
		}
	}
	client, _, _ := projection(end, "client", 2)
	if !found || client.Status != "succeeded" || w.Contacts != 1 || len(w.Requests) != 1 || w.Requests[0].Key == oldKey || w.Requests[0].Generation != 2 || len(end.Tables["external_receipts"]) != 1 {
		t.Fatal("wrong generation made progress", w, end.Tables["external_receipts"])
	}
	phaseBStoreCut(t, db, world, "final")
	phaseEmit(t, "generations", map[string]any{"old_key": oldKey, "before": before, "after": end, "events": tape.ordered, "cancel": cancel, "retry": retry, "world": w})
}

func sameOriginalAuthorization(before, after runtime.Inspection, intent string) bool {
	find := func(s runtime.Inspection) map[string]any {
		for _, r := range s.Tables["attempts"] {
			if r["intent_id"] == intent {
				return r
			}
		}
		return nil
	}
	original := find(before)
	return original != nil && reflect.DeepEqual(original, find(after))
}
func TestPhase2BOpenSuccess(t *testing.T) {
	for _, route := range []string{"idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			world, db := filepath.Join(dir, "world.sqlite"), filepath.Join(dir, "local.sqlite")
			_, url, _ := phaseResource(t, world, "", "", nil)
			phaseCreate(t, phaseProvision(t, route, url, false), db)
			owner := phaseProcess(t, db, url, "success", nil)
			phaseFinish(t, owner)
			client := inspect(t, db)
			end := worldAt(t, world)
			key, _ := phaseRequest(t, client, route)
			state, _, _ := projection(client, "client", 1)
			if state.Status != "succeeded" || end.Contacts != 1 || len(end.Applications) != 1 || end.Applications[0].Key != key || len(client.Tables["settlements"]) != 1 {
				t.Fatal("open progress missing", state, end)
			}
			settlement := client.Tables["settlements"][0]
			selected := false
			for _, obs := range client.Tables["observations"] {
				if obs["subject"] == "client" && obs["id"] == settlement["observation_id"] && obs["consumed_sequence"] != nil {
					selected = true
				}
			}
			if !selected {
				t.Fatal("no original result selection", settlement)
			}
			phaseBStoreCut(t, db, world, "final")
			phaseEmit(t, "success", map[string]any{"key": key, "client": state, "selected": settlement["observation_id"], "world": end})
		})
	}
}
