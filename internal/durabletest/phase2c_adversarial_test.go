//go:build integration

package durabletest

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/reconciliation"
	"github.com/JakeFAU/alan/internal/boundary"
	x "github.com/JakeFAU/alan/internal/executor"
	rt "github.com/JakeFAU/alan/runtime"
)

// The primary target follows the ordinary three-actor handoff. Only the
// optional competing reader receives its local operation selector from input.
func hostileSeed(t *testing.T, h *history2c, twin bool) {
	t.Helper()
	p, err := reconciliation.New(reconciliation.Options{})
	if err != nil {
		t.Fatal(err)
	}
	grant := rt.ReceiptGrant{Resolver: resolver2c, Invocation: `["parent",1,1,0]`, Member: "source", Sequence: 2, Endpoint: h.apply, Adopt: true}
	p.Reconciliation = &rt.ReconciliationProfile{Grants: []rt.ReceiptGrant{grant}}
	addOperator(t, &p, "operator", "CancelRun", "resolver")
	addProgram(t, &p, "ordinary", []x.Instruction{gateInstruction(), {Op: "end", Outcome: "succeeded"}}, map[string]x.Value{})
	p.Permissions = append(p.Permissions, rt.Permission{Kind: "input", Source: "local", Target: "ordinary", Operation: "input", Obligation: "control"})
	if twin {
		twinID := rt.RunID{Subject: "twin", Generation: 1}
		def := author.Definition{Name: "phase2c-competing-reader", Ports: map[string]author.Port{"start": {Peer: "local", Local: true, Obligation: "start", Receive: rq.ReadInput()}},
			Effects: map[string]author.Effect{"lookup": {Capability: rq.ReadCapability, Input: rq.ReadInput(), Output: rq.ReadOutput()}},
			Steps:   []author.Step{author.Receive("start", map[string]string{"target": "selector"}), author.Request("other", "lookup", map[string]author.Expr{"target": author.Var("selector")}, nil), author.Await("lookup", "other", map[string]string{"status": "status", "adoption": "decision", "mutation": "mutation", "target": "original"}), author.Success()}}
		a, e := author.Compile(def)
		if e != nil {
			t.Fatal(e)
		}
		other, e := (author.Application{"twin": {Artifact: a}}).Provision()
		if e != nil {
			t.Fatal(e)
		}
		p.Subjects = append(p.Subjects, other.Subjects...)
		p.Programs = append(p.Programs, other.Programs...)
		p.Runs = append(p.Runs, other.Runs...)
		p.Permissions = append(p.Permissions, other.Permissions...)
		p.Composition.Roots = append(p.Composition.Roots, rt.ResourceRoot{Run: twinID, Members: []rt.RunID{twinID}, Capabilities: []string{rq.ReadCapability}, Limits: rt.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
		grant.Resolver = twinID
		p.Reconciliation.Grants = append(p.Reconciliation.Grants, grant)
	}
	phaseCreate(t, p, h.db)
}

func liveSuccessor(t *testing.T, h *history2c) *rt.Store {
	t.Helper()
	s, e := rt.Open(t.Context(), h.db, rt.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
	})
	if e = s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, h.source, false)); e != nil {
		t.Fatal(e)
	}
	h.event("successor original Q", map[string]any{"PID": os.Getpid(), "Source": h.source, "Namespace": s.Namespace()})
	return s
}

func startRead(t *testing.T, s *rt.Store, subject string, payload map[string]rt.Value) {
	t.Helper()
	a, e := s.AdmitInput(t.Context(), rt.Input{TransportID: "service-" + subject, LogicalID: "service-" + subject, Target: subject, Generation: 1, Obligation: "start", Payload: payload})
	if e != nil || !a.Accepted {
		t.Fatal(a, e)
	}
}

func runReaders(t *testing.T, s *rt.Store, ready ready2c, capacity int, observe func(context.Context, boundary.Event) error) <-chan error {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	done := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		_, e := s.Run(boundary.WithObserver(ctx, observe), rt.RunOptions{MaxSteps: 240, MaxDispatch: 240, MaxInFlight: capacity, ExternalTimeout: 18 * time.Second, Adapters: map[string]effects.Adapter{rq.ReadCapability: rq.Reader{Endpoint: ready.LookupURL}, rq.CounterCapability: rq.Counter{Endpoint: ready.URL, Mode: "success"}}})
		done <- e
	}()
	t.Cleanup(func() { cancel(); <-stopped })
	return done
}

func recvReadBoundary(t *testing.T, reached <-chan boundary.Event, done <-chan error) boundary.Event {
	t.Helper()
	select {
	case e := <-reached:
		return e
	case e := <-done:
		t.Fatalf("Run exited before causal acknowledgment: %v", e)
	case <-time.After(15 * time.Second):
		t.Fatal("missing acknowledgment")
	}
	return boundary.Event{}
}

// The socket and endpoint remain live through the successful query. R and a
// peer's local work occur while MaxInFlight=1 is held by the query. Injecting a
// SQLite abort changes no authoritative row; dropping the trigger restores I/O.
func TestPhase2CResolverAuthority(t *testing.T) {
	for _, tc := range []string{"before-admission", "held-read", "sticky-original-settlement", "sticky-query-settlement"} {
		t.Run(tc, func(t *testing.T) {
			h := newHistory2c(t, tc)
			var endpointCut *barrier
			if tc != "before-admission" {
				endpointCut = cut2c("resource.lookup", "pre-snapshot", "")
			}
			ep, ready := h.endpoint("", endpointCut)
			h.apply = ready.URL
			hostileSeed(t, h, false)
			h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
			sticky := strings.HasPrefix(tc, "sticky-")
			var fault *sql.DB
			if sticky {
				var e error
				fault, e = sql.Open("sqlite3", h.db+"?_busy_timeout=5000")
				if e != nil {
					t.Fatal(e)
				}
				defer fault.Close()
				operation := `["resolver",1,4,0]`
				if tc == "sticky-original-settlement" {
					operation = h.original
				}
				selector := "NEW.intent_id=" + h.data(h.db, "SELECT quote(?)", operation)
				_, e = fault.ExecContext(t.Context(), "CREATE TRIGGER abort_settlement BEFORE INSERT ON settlements WHEN "+selector+" BEGIN SELECT RAISE(ABORT,'directed receipt settlement abort'); END")
				if e != nil {
					t.Fatal(e)
				}
			}
			s := liveSuccessor(t, h)
			startRead(t, s, "resolver", map[string]rt.Value{})
			reached := make(chan boundary.Event, 1)
			release := make(chan struct{})
			released := false
			t.Cleanup(func() {
				if !released {
					close(release)
				}
			})
			done := runReaders(t, s, ready, 1, func(ctx context.Context, e boundary.Event) error {
				if tc == "before-admission" && e.Boundary == "dispatch.consider" && e.Subject == "resolver" {
					reached <- e
					select {
					case <-release:
					case <-ctx.Done():
					}
					return ctx.Err()
				}
				return nil
			})
			target := dispatchTarget(s, resolver2c, false)
			if tc == "before-admission" {
				ack := recvReadBoundary(t, reached, done)
				result := manualControl(t, s, "operator", "resolver")
				h.event("resolver R before A", map[string]any{"Hold": ack, "Result": result})
				if e := s.AwaitDispatchQuiescence(t.Context(), target); e != nil {
					t.Fatal(e)
				}
				close(release)
				released = true
				if e := <-done; e != nil {
					t.Fatal(e)
				}
				h.expectWorld(1, 1, 0)
				if h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals") != 0 || h.count(h.db, "SELECT count(*) FROM adapter_results") != 0 {
					t.Fatal("closed reader spent or imported")
				}
				h.replay("final")
				phaseStopEndpoint(t, ep)
				return
			}
			ack := h.await(ep)
			if ack.Intent != `["resolver",1,4,0]` || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals WHERE intent_id=?", ack.Intent) != 1 {
				t.Fatal("unbound lookup", ack)
			}
			h.expectWorld(1, 1, 0)
			admitControl(t, s, "ordinary", "peer-work")
			admitControl(t, s, "operator", "resolver")
			peer := untilTerminal(t, s, rt.RunID{Subject: "ordinary", Generation: 1})
			closed := untilTerminal(t, s, resolver2c)
			if peer.Projection.Status != "succeeded" || closed.Projection.Status != "cancelled" {
				t.Fatal(peer, closed)
			}
			ctx, cancel := context.WithCancel(t.Context())
			waiting := false
			e := s.AwaitDispatchQuiescence(boundary.WithObserver(ctx, func(_ context.Context, ev boundary.Event) error {
				if ev.Boundary == "dispatch.waiting" {
					waiting = true
					cancel()
				}
				return ctx.Err()
			}), target)
			cancel()
			if !waiting || !errors.Is(e, context.Canceled) {
				t.Fatal("Q skipped active read", e)
			}
			h.event("peer committed and resolver R while endpoint held; Q waited", map[string]any{"Peer": peer, "Resolver": closed, "Hold": ack})
			if _, e = ep.release.Write([]byte{'c'}); e != nil {
				t.Fatal(e)
			}
			err := <-done
			if sticky {
				if err == nil || !strings.Contains(err.Error(), "directed receipt settlement abort") {
					t.Fatal("missing Run accounting error", err)
				}
				if h.count(h.db, "SELECT count(*) FROM adapter_results") != 2 || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", ack.Intent) != 0 {
					t.Fatal("missing retained proof or unexpected lookup settlement", err, ack, h.count(h.db, "SELECT count(*) FROM adapter_results"), h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", ack.Intent))
				}
				h.snapshot("accounting-rejected")
				if _, e = fault.ExecContext(t.Context(), "DROP TRIGGER abort_settlement"); e != nil {
					t.Fatal(e)
				}
				for i := 0; i < 2; i++ {
					err = s.AwaitDispatchQuiescence(t.Context(), target)
					if err == nil || !strings.Contains(err.Error(), "directed receipt settlement abort") {
						t.Fatal("lost same-owner accounting failure", i, err)
					}
					h.event("same-owner Q retained SQLite error", err.Error())
				}
				if e = s.Close(); e != nil {
					t.Fatal(e)
				}
				s, e = rt.Open(t.Context(), h.db, rt.DefaultLimits())
				if e != nil {
					t.Fatal(e)
				}
				defer s.Close()
			} else if err != nil {
				t.Fatal(err)
			}
			if e = s.AwaitDispatchQuiescence(t.Context(), target); e != nil {
				t.Fatal(e)
			}
			h.expectWorld(1, 1, 1)
			if h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=? AND json_type(descriptor,'$.Reconciled')='object'", h.original) != 1 || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id IN (?,?)", h.original, ack.Intent) != 2 || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals") != 1 {
				t.Fatal("late evidence/accounting incorrect")
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}

func TestPhase2CCompetingResolvers(t *testing.T) {
	for _, first := range []string{"resolver", "twin"} {
		t.Run(first+"-first", func(t *testing.T) {
			h := newHistory2c(t, first+"-first")
			ep, ready := h.endpoint("", nil)
			h.apply = ready.URL
			hostileSeed(t, h, true)
			h.originalCut(cut2c("adapter.response", "pre-record", ""), nil, false)
			s := liveSuccessor(t, h)
			startRead(t, s, "resolver", map[string]rt.Value{})
			startRead(t, s, "twin", map[string]rt.Value{"target": x.Text(h.original)})
			reached := make(chan boundary.Event, 2)
			settled := make(chan boundary.Event, 2)
			releases := map[string]chan struct{}{"resolver": make(chan struct{}), "twin": make(chan struct{})}
			done := runReaders(t, s, ready, 2, func(ctx context.Context, e boundary.Event) error {
				if c := releases[e.Subject]; c != nil {
					switch e.Boundary {
					case "adapter.response":
						reached <- e
						select {
						case <-c:
						case <-ctx.Done():
						}
						return ctx.Err()
					case "adapter.settled":
						settled <- e
					}
				}
				return nil
			})
			one, two := recvReadBoundary(t, reached, done), recvReadBoundary(t, reached, done)
			if one.Subject == two.Subject || one.Intent == two.Intent {
				t.Fatal("same read delivered twice", one, two)
			}
			h.expectWorld(1, 1, 2)
			if h.count(h.db, "SELECT count(*) FROM adapter_results") != 0 || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals") != 2 {
				t.Fatal("in-flight boundary")
			}
			h.event("two held responses", []boundary.Event{one, two})
			h.snapshot("both-responses")
			second := "resolver"
			if first == "resolver" {
				second = "twin"
			}
			close(releases[first])
			win := recvReadBoundary(t, settled, done)
			if win.Subject != first {
				t.Fatal(win)
			}
			h.event("winning response settled", win)
			before := h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", h.original)
			close(releases[second])
			if e := <-done; e != nil {
				t.Fatal(e)
			}
			if before != h.data(h.db, "SELECT descriptor FROM adapter_results WHERE intent_id=?", h.original) || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", h.original) != 1 || h.count(h.db, "SELECT count(*) FROM composition_reservations") != 3 || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE json_extract(descriptor,'$.Read.Adoption')='duplicate' AND json_extract(intent_id,'$[0]')=?", second) != 1 || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE json_extract(descriptor,'$.Read.Adoption')='adopted' AND json_extract(intent_id,'$[0]')=?", first) != 1 {
				t.Fatal("duplicate receipt changed the original or accounting")
			}
			h.expectWorld(1, 1, 2)
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}
