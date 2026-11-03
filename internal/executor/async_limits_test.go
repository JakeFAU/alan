package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

func addExternalClient(t *testing.T, p *Provision, name, target, profile string, delta int64) {
	t.Helper()
	addProgram(t, p, name, []Instruction{
		{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text(profile)), "delta": Lit(Int(delta))}}},
		{Op: "receive", Match: &Match{Kind: "receipt", Provenance: "resource-receipt", Source: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Operation: Ref("pending")}, Bind: map[string]string{"total": "total"}},
		{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = append(p.Permissions, Permission{Kind: "external", Source: name, Target: target, Operation: "apply", Revision: 1})
}
func receiptFor(req resource.Request) resource.Receipt {
	return resource.Receipt{Key: req.Key, Profile: req.Profile, Delta: req.Delta, Mutation: req.Delta, Total: req.Delta, Process: "held-test", PID: 1}
}
func receiveRequest(ctx context.Context, requests <-chan resource.Request) (resource.Request, error) {
	select {
	case req := <-requests:
		return req, nil
	case <-ctx.Done():
		return resource.Request{}, ctx.Err()
	}
}

func TestAsyncSaturationLeavesBacklogAndDeliversPeer(t *testing.T) {
	for _, capacity := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(capacity), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			requests := make(chan resource.Request, 10)
			releases := map[string]chan struct{}{}
			for i := 0; i < capacity+1; i++ {
				releases[fmt.Sprintf("client%d", i)] = make(chan struct{})
			}
			var mu sync.Mutex
			seen := map[string]int{}
			active, peak := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req resource.Request
				json.NewDecoder(r.Body).Decode(&req)
				mu.Lock()
				seen[req.Key]++
				active++
				if active > peak {
					peak = active
				}
				mu.Unlock()
				defer func() { mu.Lock(); active--; mu.Unlock() }()
				requests <- req
				select {
				case <-releases[req.Subject]:
					json.NewEncoder(w).Encode(receiptFor(req))
				case <-r.Context().Done():
				}
			}))
			defer server.Close()
			defer cancel()
			p := Provision{}
			for i := 0; i < capacity+1; i++ {
				addExternalClient(t, &p, fmt.Sprintf("client%d", i), server.URL, "idempotent", int64(i+1))
			}
			code := []Instruction{}
			for i := 0; i < 12; i++ {
				code = append(code, Instruction{Op: "set", Dst: "work", A: Lit(Int(int64(i)))})
			}
			code = append(code, Instruction{Op: "end", Outcome: "succeeded"})
			addProgram(t, &p, "zpeer", code, map[string]Value{})
			addProgram(t, &p, "zcaller", []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text("zpeer")), Generation: Lit(Int(1)), Obligation: Lit(Text("done")), Operation: Lit(Text(""))}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "zpeer", Target: "zcaller", Operation: "terminal", Revision: 1, Obligation: "done"})
			s, _ := messageStore(t, p)
			held := []resource.Request{}
			ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
				if e.Subject == "zpeer" && e.Sequence == 1 && e.Boundary == "execution.staged" {
					for i := 0; i < capacity; i++ {
						req, err := receiveRequest(ctx, requests)
						if err != nil {
							return err
						}
						held = append(held, req)
					}
				}
				if e.Subject == "zcaller" && e.Sequence == 1 && e.Boundary == "execution.committed" {
					mu.Lock()
					n := len(seen)
					mu.Unlock()
					if n != capacity {
						return fmt.Errorf("extra transport before slot release: %d", n)
					}
					close(releases[held[0].Subject])
				}
				return nil
			})
			// The controller waits for the first newly admitted backlog request before
			// releasing the remaining slots. It never commits application work.
			controller := make(chan error, 1)
			delivered := make(chan struct{})
			priorCtx := ctx
			ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
				if err := boundary.Reached(priorCtx, e); err != nil {
					return err
				}
				if e.Subject == "zcaller" && e.Sequence == 1 && e.Boundary == "execution.committed" {
					close(delivered)
				}
				return nil
			})
			go func() {
				select {
				case <-delivered:
				case <-ctx.Done():
					controller <- ctx.Err()
					return
				}
				req, err := receiveRequest(ctx, requests)
				if err != nil {
					controller <- err
					return
				}
				close(releases[req.Subject])
				for _, h := range held[1:] {
					close(releases[h.Subject])
				}
				controller <- nil
			}()
			report, err := s.Run(ctx, RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: capacity, ExternalTimeout: 5 * time.Second})
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			if err = <-controller; err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if peak > capacity || len(seen) != capacity+1 || report.Dispatches != capacity+2 || len(report.UnresolvedIntents) != 0 {
				t.Fatalf("capacity/backlog %+v peak=%d seen=%v", report, peak, seen)
			}
			for key, n := range seen {
				if n != 1 {
					t.Fatalf("duplicate in-flight claim %s: %d", key, n)
				}
			}
		})
	}
}

func TestAsyncReorderedReceiptsConsumeWithoutHeadOfLineAndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	requests := make(chan resource.Request, 2)
	firstRelease := make(chan struct{})
	var once sync.Once
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		contacts.Add(1)
		requests <- req
		if req.Subject == "a" {
			select {
			case <-firstRelease:
			case <-r.Context().Done():
				return
			}
		}
		json.NewEncoder(w).Encode(receiptFor(req))
	}))
	defer server.Close()
	defer once.Do(func() { close(firstRelease) })
	p := Provision{}
	addExternalClient(t, &p, "a", server.URL, "idempotent", 5)
	addExternalClient(t, &p, "b", server.URL, "idempotent", 9)
	s, path := messageStore(t, p)
	order := []string{}
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Subject == "b" && e.Boundary == "external.authorized" {
			for i := 0; i < 1; i++ {
				req, err := receiveRequest(ctx, requests)
				if err != nil {
					return err
				}
				if req.Subject != "a" {
					return fmt.Errorf("second operation arrived first")
				}
			}
		}
		if e.Sequence == 2 && e.Boundary == "execution.committed" {
			order = append(order, e.Subject)
			if e.Subject == "b" {
				select {
				case <-firstRelease:
					return fmt.Errorf("first released before second consumed")
				default:
				}
				once.Do(func() { close(firstRelease) })
			}
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 30, MaxDispatch: 10, MaxInFlight: 2})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "b,a" || report.Dispatches != 2 || current(t, s, "a").Projection.Data["total"] != Int(5) || current(t, s, "b").Projection.Data["total"] != Int(9) {
		t.Fatalf("correlation/order %v %+v", order, report)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	before := contacts.Load()
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != before {
		t.Fatal("replay contacted endpoint")
	}
}

func TestAsyncCancelAccountsReceivedCandidateAndPreservesUnknown(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	held := make(chan struct{})
	received := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		if req.Subject == "a" {
			close(held)
			<-r.Context().Done()
			return
		}
		json.NewEncoder(w).Encode(receiptFor(req))
	}))
	defer server.Close()
	p := Provision{}
	addExternalClient(t, &p, "a", server.URL, "non-idempotent", 5)
	addExternalClient(t, &p, "b", server.URL, "non-idempotent", 9)
	addProgram(t, &p, "zpeer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, path := messageStore(t, p)
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Subject == "b" && e.Boundary == "external.response" {
			close(received)
			<-ctx.Done()
			return ctx.Err()
		}
		if e.Subject == "zpeer" && e.Boundary == "execution.staged" {
			select {
			case <-held:
			case <-time.After(5 * time.Second):
				return fmt.Errorf("hold watchdog")
			}
			select {
			case <-received:
			case <-time.After(5 * time.Second):
				return fmt.Errorf("response watchdog")
			}
			cancel()
			return ctx.Err()
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 20, MaxDispatch: 20, MaxInFlight: 2})
	if !errors.Is(err, context.Canceled) || report.Quiescent || len(report.UnknownExternal) != 1 || count(t, s, "external_receipts") != 1 {
		t.Fatalf("cleanup lost received outcome %+v %v", report, err)
	}
	if current(t, s, "a").Projection.Status != "open" {
		t.Fatal("physical cancellation logically closed run")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	report, err = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 1})
	if err != nil || report.Dispatches != 0 || len(report.UnknownExternal) != 1 || current(t, s, "b").Projection.Status != "succeeded" {
		t.Fatalf("reopen charged Unknown or lost receipt %+v %v", report, err)
	}
}

func TestAsyncOverlappingRunRejectedAndCloseJoins(t *testing.T) {
	held := make(chan struct{})
	ended := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		close(held)
		<-r.Context().Done()
		close(ended)
	}))
	defer server.Close()
	s, path := messageStore(t, externalProvision(t, server.URL, "non-idempotent"))
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20, ExternalTimeout: 10 * time.Second})
		done <- err
	}()
	select {
	case <-held:
	case <-time.After(5 * time.Second):
		t.Fatal("hold watchdog")
	}
	if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); !errors.Is(err, ErrRunActive) {
		t.Fatalf("overlap %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Close did not cancel owner: %v", err)
	}
	<-ended
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	report, err := reopened.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil || len(report.UnknownExternal) != 1 {
		t.Fatalf("retained uncertainty %+v %v", report, err)
	}
}

func TestAsyncLimitsWithHeldWork(t *testing.T) {
	for _, limit := range []string{"steps", "dispatch"} {
		t.Run(limit, func(t *testing.T) {
			held := make(chan struct{})
			var contacts atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				if contacts.Add(1) == 1 {
					close(held)
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			p := Provision{}
			addExternalClient(t, &p, "a", server.URL, "non-idempotent", 5)
			addExternalClient(t, &p, "b", server.URL, "non-idempotent", 9)
			addProgram(t, &p, "zpeer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			s, _ := messageStore(t, p)
			ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
				if e.Subject == "b" && e.Boundary == "execution.staged" {
					select {
					case <-held:
						return nil
					case <-time.After(5 * time.Second):
						return fmt.Errorf("hold watchdog")
					}
				}
				return nil
			})
			options := RunOptions{MaxSteps: 2, MaxDispatch: 10, MaxInFlight: 2}
			if limit == "dispatch" {
				options.MaxSteps = 20
				options.MaxDispatch = 1
			}
			report, err := s.Run(ctx, options)
			if !errors.Is(err, ErrLimit) || report.Quiescent || report.Dispatches > options.MaxDispatch || report.Steps > options.MaxSteps || len(report.UnknownExternal) == 0 {
				t.Fatalf("budget %+v %v", report, err)
			}
			if limit == "dispatch" && contacts.Load() != 1 {
				t.Fatal("overspent dispatch")
			}
		})
	}
}

func TestAsyncReceiptPersistenceFailureIsExplicitAndRecoverable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(receiptFor(req))
	}))
	defer server.Close()
	s, path := messageStore(t, externalProvision(t, server.URL, "idempotent"))
	if _, err := s.conn.ExecContext(t.Context(), `CREATE TEMP TRIGGER reject_receipt BEFORE INSERT ON external_receipts BEGIN SELECT RAISE(ABORT,'test receipt persistence failure'); END`); err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err == nil || !strings.Contains(err.Error(), "test receipt persistence failure") || len(report.UnknownExternal) != 1 || count(t, s, "external_receipts") != 0 {
		t.Fatalf("swallowed storage failure %+v %v", report, err)
	}
	s.Close()
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncLateReceiptValidatesRetainedEpochAndRejectsForgery(t *testing.T) {
	s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
	state, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := s.prepareExternal(t.Context(), state.Projection.Data["pending"].Text)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	bad := a
	bad.Request.Epoch++
	if err = s.recordExternalReceipt(t.Context(), bad, receiptFor(a.Request)); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("forged candidate %v", err)
	}
	if err = s.recordExternalReceipt(t.Context(), a, receiptFor(a.Request)); err != nil {
		t.Fatalf("authentic old epoch rejected %v", err)
	}
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncOrdinarySupervisorAdvancesWhileExternalHeld(t *testing.T) {
	held := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		close(held)
		select {
		case <-release:
			json.NewEncoder(w).Encode(receiptFor(req))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer once.Do(func() { close(release) })
	p, err := NestedProvision(NestedConfig{RetryBudget: 1, SuccessorInput: "success"})
	if err != nil {
		t.Fatal(err)
	}
	addExternalClient(t, &p, "a", server.URL, "idempotent", 5)
	s, _ := messageStore(t, p)
	advanced := false
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Subject == "inner" && e.Sequence == 1 && e.Boundary == "execution.staged" {
			select {
			case <-held:
			case <-time.After(5 * time.Second):
				return fmt.Errorf("hold watchdog")
			}
		}
		if e.Boundary == "lifecycle.decided" && e.Phase == "accepted" {
			advanced = true
			once.Do(func() { close(release) })
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 200, MaxDispatch: 200})
	if err != nil || !advanced || len(report.UnknownExternal) != 0 {
		t.Fatalf("ordinary supervision blocked %+v %v", report, err)
	}
}

func TestAsyncTimeoutAndUnavailableAllowProgressBeforeFailure(t *testing.T) {
	for _, mode := range []string{"timeout", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			held := make(chan struct{})
			peer := make(chan struct{})
			finished := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				close(held)
				defer close(finished)
				if mode == "timeout" {
					<-r.Context().Done()
				} else {
					select {
					case <-peer:
						w.WriteHeader(http.StatusServiceUnavailable)
					case <-r.Context().Done():
					}
				}
			}))
			defer server.Close()
			p := externalProvision(t, server.URL, "non-idempotent")
			addProgram(t, &p, "peer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			s, _ := messageStore(t, p)
			ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
				if e.Subject == "peer" && e.Boundary == "execution.staged" {
					select {
					case <-held:
					case <-time.After(5 * time.Second):
						return fmt.Errorf("hold watchdog")
					}
				}
				if e.Subject == "peer" && e.Boundary == "execution.committed" {
					select {
					case <-finished:
						return fmt.Errorf("failure preceded eligible peer commit")
					default:
					}
					close(peer)
				}
				return nil
			})
			report, err := s.Run(ctx, RunOptions{MaxSteps: 20, MaxDispatch: 20, ExternalTimeout: 500 * time.Millisecond})
			if err != nil || len(report.UnknownExternal) != 1 || current(t, s, "peer").Projection.Status != "succeeded" || current(t, s, "client").Projection.Status != "open" {
				t.Fatalf("failure escaped application %+v %v", report, err)
			}
		})
	}
}

func TestAsyncAuthorizedTransportFinishesAfterOrdinarySourceClosure(t *testing.T) {
	held := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		close(held)
		select {
		case <-release:
			json.NewEncoder(w).Encode(receiptFor(req))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer once.Do(func() { close(release) })
	p := Provision{}
	addProgram(t, &p, "client", []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(server.URL)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(5))}}}, {Op: "end", Outcome: "failed"}}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "external", Source: "client", Target: server.URL, Operation: "apply", Revision: 1}}
	s, path := messageStore(t, p)
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Subject == "client" && e.Sequence == 2 && e.Boundary == "execution.staged" {
			select {
			case <-held:
			case <-time.After(5 * time.Second):
				return fmt.Errorf("hold watchdog")
			}
		}
		if e.Subject == "client" && e.Sequence == 2 && e.Boundary == "execution.committed" {
			once.Do(func() { close(release) })
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 20, MaxDispatch: 20})
	if err != nil || len(report.UnresolvedIntents) != 0 || count(t, s, "external_receipts") != 1 || current(t, s, "client").Projection.Status != "failed" {
		t.Fatalf("late receipt %+v %v", report, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncAdmittedInputWakesIdleOwnerWhileTransportHeld(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	held := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		json.NewDecoder(r.Body).Decode(&req)
		close(held)
		select {
		case <-release:
			json.NewEncoder(w).Encode(receiptFor(req))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer once.Do(func() { close(release) })
	p := inputFixture(t)
	addExternalClient(t, &p, "a", server.URL, "idempotent", 5)
	s, _ := messageStore(t, p)
	waiting := make(chan struct{})
	admitted := make(chan struct{})
	waits := 0
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "runner.waiting" {
			waits++
			if waits == 2 {
				close(waiting)
				select {
				case <-admitted:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		if e.Subject == "receiver" && e.Boundary == "execution.committed" && e.Sequence == 1 {
			once.Do(func() { close(release) })
		}
		return nil
	})
	type result struct {
		report RunReport
		err    error
	}
	done := make(chan result, 1)
	go func() {
		report, err := s.Run(ctx, RunOptions{MaxSteps: 30, MaxDispatch: 30, ExternalTimeout: 10 * time.Second})
		done <- result{report, err}
	}()
	select {
	case <-held:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case <-waiting:
	case <-ctx.Done():
		t.Fatal("owner never reached idle wait")
	}
	admission, err := s.AdmitInput(ctx, Input{TransportID: "wake-input", LogicalID: "wake-logical", Target: "receiver", Obligation: "work", Payload: map[string]Value{"number": Int(37)}})
	if err != nil || !admission.Accepted {
		t.Fatalf("admission %+v %v", admission, err)
	}
	close(admitted)
	got := <-done
	if got.err != nil || len(got.report.UnresolvedIntents) != 0 || current(t, s, "receiver").Projection.Data["number"] != Int(37) {
		t.Fatalf("input did not wake owner %+v %v", got.report, got.err)
	}
}

func TestAsyncBudgetAccountsReceivedOutcomeWhileAnotherRequestIsHeld(t *testing.T) {
	for _, budget := range []string{"steps", "dispatch"} {
		t.Run(budget, func(t *testing.T) {
			held := make(chan struct{})
			received := make(chan struct{})
			queue := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req resource.Request
				json.NewDecoder(r.Body).Decode(&req)
				if req.Subject == "a" {
					close(held)
					<-r.Context().Done()
					return
				}
				json.NewEncoder(w).Encode(receiptFor(req))
			}))
			defer server.Close()
			p := Provision{}
			addExternalClient(t, &p, "a", server.URL, "non-idempotent", 5)
			addExternalClient(t, &p, "b", server.URL, "non-idempotent", 9)
			addProgram(t, &p, "zpeer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			addProgram(t, &p, "zzcaller", []Instruction{{Op: "receive", Match: &Match{Kind: "terminal", Provenance: "runtime-terminal", Source: Lit(Text("zpeer")), Generation: Lit(Int(1)), Obligation: Lit(Text("done")), Operation: Lit(Text(""))}}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			p.Permissions = append(p.Permissions, Permission{Kind: "observation", Source: "zpeer", Target: "zzcaller", Operation: "terminal", Revision: 1, Obligation: "done"})
			s, path := messageStore(t, p)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
				if e.Subject == "b" && e.Boundary == "external.response" {
					close(received)
					select {
					case <-queue:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				if e.Subject == "zpeer" && e.Boundary == "execution.staged" {
					select {
					case <-held:
					case <-ctx.Done():
						return ctx.Err()
					}
					select {
					case <-received:
					case <-ctx.Done():
						return ctx.Err()
					}
					close(queue)
				}
				return nil
			})
			options := RunOptions{MaxSteps: 3, MaxDispatch: 20, MaxInFlight: 2, ExternalTimeout: 10 * time.Second}
			if budget == "dispatch" {
				options.MaxSteps = 20
				options.MaxDispatch = 2
			}
			report, err := s.Run(ctx, options)
			if !errors.Is(err, ErrLimit) || report.Quiescent || len(report.UnknownExternal) != 1 || count(t, s, "external_receipts") != 1 || report.Steps > options.MaxSteps || report.Dispatches > options.MaxDispatch {
				t.Fatalf("budget lost queued outcome %+v %v", report, err)
			}
			s.Close()
			s, err = Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
				t.Fatal(err)
			}
			if current(t, s, "b").Projection.Status != "succeeded" || current(t, s, "zzcaller").Projection.Status != "succeeded" {
				t.Fatal("recovery failed to consume retained receipt/delivery")
			}
		})
	}
}

func TestAsyncBudgetCountsCommitEvenIfPostCommitObserverFails(t *testing.T) {
	p := Provision{}
	addProgram(t, &p, "peer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, _ := messageStore(t, p)
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "execution.committed" {
			return context.Canceled
		}
		return nil
	})
	report, err := s.Run(ctx, RunOptions{MaxSteps: 1, MaxDispatch: 1})
	if !errors.Is(err, context.Canceled) || report.Steps != 1 || current(t, s, "peer").Projection.Status != "succeeded" {
		t.Fatalf("durable commit not charged %+v %v", report, err)
	}
}
