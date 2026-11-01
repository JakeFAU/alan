//go:build integration

package durabletest

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	frame "github.com/JakeFAU/alan/examples/effectfulcomposition"
	"github.com/JakeFAU/alan/internal/boundary"
	rt "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func adverseFrame(t *testing.T, endpoint string, l, r frame.RoleOptions) rt.Provision {
	t.Helper()
	tree, e := frame.PureFork("A", "B", "")
	if e != nil {
		t.Fatal(e)
	}
	d, e := frame.New(frame.Options{Tree: tree, Endpoint: endpoint, Left: l, Right: r, Inputs: map[string]schema.Value{"l_mode": schema.Text("answer"), "r_mode": schema.Text("answer"), "l_id": schema.Int(1), "r_id": schema.Int(1), "l_value": schema.Int(0), "r_value": schema.Int(1)}})
	if e != nil {
		t.Fatal(e)
	}
	return d.Provision
}

func adverseOpen(t *testing.T, path string) *rt.Store {
	t.Helper()
	s, e := rt.Open(t.Context(), path, rt.DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
	})
	return s
}

func adverseOptions(ready ready2c, mode string) rt.RunOptions {
	return rt.RunOptions{MaxSteps: 2048, MaxDispatch: 2048, MaxInFlight: 2, ExternalTimeout: 60 * time.Second, Adapters: map[string]effects.Adapter{rq.CounterCapability: rq.Counter{Endpoint: ready.URL, Mode: mode}, rq.ReadCapability: rq.Reader{Endpoint: ready.LookupURL}}}
}
func adverseRun(t *testing.T, s *rt.Store, opts rt.RunOptions) {
	t.Helper()
	if _, e := s.Run(t.Context(), opts); e != nil {
		t.Fatal(e)
	}
}
func roleRun(role string) rt.RunID   { return rt.RunID{Subject: "resolver_" + role, Generation: 1} }
func staticRun(name string) rt.RunID { return rt.RunID{Subject: name, Generation: 1} }
func roleState(t *testing.T, s *rt.Store, who string) rt.RunState {
	t.Helper()
	v, e := s.Snapshot(t.Context(), staticRun(who))
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func roleService(t *testing.T, s *rt.Store, role, port string) {
	t.Helper()
	bindingInput(t, s, roleRun(role), port, map[string]rt.Value{})
}
func competingActor(t *testing.T, p *rt.Provision) {
	t.Helper()
	a, e := author.Compile(author.Definition{Name: "competing-left-adopter", Ports: map[string]author.Port{"start": {Peer: "local", Local: true, Obligation: "start", Receive: rq.ReadInput()}},
		Effects: map[string]author.Effect{"lookup": {Capability: rq.ReadCapability, Input: rq.ReadInput(), Output: rq.ReadOutput()}},
		Steps:   []author.Step{author.Receive("start", map[string]string{"target": "selector"}), author.Request("query", "lookup", map[string]author.Expr{"target": author.Var("selector")}, nil), author.Await("lookup", "query", map[string]string{"status": "status"}), author.Success()}})
	if e != nil {
		t.Fatal(e)
	}
	x, e := (author.Application{"twin": {Artifact: a}}).Provision()
	if e != nil {
		t.Fatal(e)
	}
	p.Subjects = append(p.Subjects, x.Subjects...)
	p.Programs = append(p.Programs, x.Programs...)
	p.Runs = append(p.Runs, x.Runs...)
	p.Permissions = append(p.Permissions, x.Permissions...)
	id := staticRun("twin")
	p.Composition.Roots = append(p.Composition.Roots, rt.ResourceRoot{Run: id, Members: []rt.RunID{id}, Capabilities: []string{rq.ReadCapability}, Limits: rt.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}})
	g := p.Reconciliation.Grants[0]
	g.Resolver = id
	p.Reconciliation.Grants = append(p.Reconciliation.Grants, g)
	addOperator(t, p, "operator", "CancelRun", "twin")
}

// Both genuine issuing responses are acknowledged separately before result
// retention. The independent selector waits for both sources' real End commits
// before killing their owner. It does not close either issuer itself.
func adverseOriginalLoss(t *testing.T, h *history2c, mode string) {
	t.Helper()
	input, keep := pipe2c(t)
	defer input.Close()
	defer keep.Close()
	read, output := pipe2c(t)
	defer read.Close()
	c := phaseTestProcess(t, h.db, h.apply, mode, nil, "TestPhase2DOwner", func(cmd *exec.Cmd) {
		cmd.Stdin = input
		cmd.Stdout = output
		cmd.Env = append(cmd.Env, "ALAN_2D_OWNER_DB="+h.db, "ALAN_2D_CUT=adapter.response", "ALAN_2D_ENDPOINT="+h.apply, "ALAN_2D_CAPACITY=2", "ALAN_2D_POLICY="+mode)
	})
	output.Close()
	profile := map[string]string{}
	for _, pair := range c.cmd.Environ() {
		k, v, _ := strings.Cut(pair, "=")
		if strings.HasPrefix(k, "ALAN_2D_") {
			profile[k] = v
		}
	}
	h.event("owner invocation", map[string]any{"PID": c.cmd.Process.Pid, "Options": profile, "Argv": c.cmd.Args})
	if err := read.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		t.Fatal(err)
	}
	scan := bufio.NewScanner(read)
	seen := map[string]bool{}
	for len(seen) < 2 && scan.Scan() {
		var a struct {
			PID            int
			Namespace, Cut string
			Event          boundary.Event
		}
		if !strings.HasPrefix(scan.Text(), "{") {
			continue
		}
		if e := json.Unmarshal(scan.Bytes(), &a); e != nil {
			t.Fatal(e)
		}
		if a.PID != c.cmd.Process.Pid || a.Namespace != h.data(h.db, "SELECT namespace FROM metadata") || a.Cut != "adapter.response" || a.Event.Boundary != a.Cut || a.Event.Phase != "pre-record" || seen[a.Event.Intent] || h.count(h.db, "SELECT count(*) FROM adapter_attempts WHERE intent_id=? AND json_extract(descriptor,'$.Intent.Run.Subject')=? AND json_extract(descriptor,'$.Allowed')", a.Event.Intent, a.Event.Subject) != 1 {
			t.Fatal("wrong original response", a)
		}
		seen[a.Event.Intent] = true
		h.event("original response held", a)
	}
	if len(seen) != 2 {
		t.Fatal("missing original responses", scan.Err())
	}
	deadline := time.Now().Add(30 * time.Second)
	for h.count(h.db, "SELECT count(*) FROM runs WHERE json_type(projection,'$.Data.original')='object' AND json_extract(projection,'$.Status')='succeeded'") != 2 {
		if time.Now().After(deadline) {
			t.Fatal("issuers did not both end")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if h.count(h.db, "SELECT count(*) FROM adapter_results") != 0 {
		t.Fatal("original already handled")
	}
	h.expectWorld(2, 2, 0)
	c.acknowledged = true
	c.kill(t)
	h.event("original owner SIGKILL", map[string]any{"PID": c.cmd.Process.Pid, "Wait": c.cmd.ProcessState.String(), "Binary": binaryHash(c.cmd.Path)})
	h.snapshot("two-missing")
}

func adverseQ(t *testing.T, h *history2c, s *rt.Store) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, role := range []string{"l", "r"} {
		m := roleState(t, s, "resolver_"+role)
		id := m.Projection.Data["received"].Text
		if id == "" {
			t.Fatal("no handed-off id", m)
		}
		var x []any
		if e := json.Unmarshal([]byte(id), &x); e != nil || len(x) != 4 {
			t.Fatal(id, e)
		}
		source := staticRun(x[0].(string))
		if e := s.AwaitDispatchQuiescence(t.Context(), dispatchTarget(s, source, false)); e != nil {
			t.Fatal(e)
		}
		result[role] = id
	}
	if result["l"] == result["r"] {
		t.Fatal(result)
	}
	h.event("both original Q", map[string]any{"Targets": result, "PID": os.Getpid(), "Namespace": s.Namespace(), "ResolverEpoch": roleState(t, s, "resolver_l").Epoch})
	return result
}

func TestPhase2DCompetingAccounting(t *testing.T) {
	for _, fault := range []string{"original", "adopter-query"} {
		t.Run(fault, func(t *testing.T) { competingAccounting(t, fault) })
	}
}
func competingAccounting(t *testing.T, fault string) {
	t.Helper()
	h := newHistory2c(t, "competing-"+fault)
	ep, ready := h.endpoint("", nil)
	h.apply = ready.URL
	p := adverseFrame(t, h.apply, frame.RoleOptions{}, frame.RoleOptions{})
	competingActor(t, &p)
	phaseCreate(t, p, h.db)
	adverseOriginalLoss(t, h, "success")
	target := h.data(h.db, "SELECT json_extract(projection,'$.Data.received.Text') FROM runs WHERE subject='resolver_l'")
	// Install the failure only between owners. It changes no semantic row.
	db, e := sql.Open("sqlite3", h.db+"?_busy_timeout=5000")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	faultID := target
	wantOriginalSettlement := 0
	if fault == "adopter-query" {
		faultID = `["twin",1,2,0]`
		wantOriginalSettlement = 1
	}
	quoted := h.data(h.db, "SELECT quote(?)", faultID)
	if _, e = db.ExecContext(t.Context(), "CREATE TRIGGER inject_left_abort BEFORE INSERT ON settlements WHEN NEW.intent_id="+quoted+" BEGIN SELECT RAISE(ABORT,'phase2d left settlement aborted'); END"); e != nil {
		t.Fatal(e)
	}
	s := adverseOpen(t, h.db)
	ids := adverseQ(t, h, s)
	if target != ids["l"] {
		t.Fatal(ids)
	}
	roleService(t, s, "r", "start")
	adverseRun(t, s, adverseOptions(ready, "success"))
	if h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", ids["r"]) != 1 {
		t.Fatal("right debt")
	}
	roleService(t, s, "l", "start")
	startRead(t, s, "twin", map[string]rt.Value{"target": schema.Text(target)})
	reached := make(chan boundary.Event, 2)
	winner := make(chan struct{})
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	observed := boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "adapter.response" && (e.Subject == "twin" || e.Subject == "resolver_l") {
			reached <- e
			if e.Subject == "twin" {
				select {
				case <-winner:
				case <-ctx.Done():
				}
			} else {
				// Cleanup cancels workers after the winning accounting failure.
				// The actual candidate has already returned and must be accounted.
				<-ctx.Done()
			}
		}
		return nil
	})
	done := make(chan error, 1)
	go func() { _, err := s.Run(observed, adverseOptions(ready, "success")); done <- err }()
	first, second := recvReadBoundary(t, reached, done), recvReadBoundary(t, reached, done)
	if first.Subject == second.Subject {
		t.Fatal(first, second)
	}
	twin, duplicate := first.Intent, second.Intent
	if first.Subject != "twin" {
		twin, duplicate = second.Intent, first.Intent
	}
	h.expectWorld(2, 2, 3)
	h.event("two preadmitted returns for L", []boundary.Event{first, second})
	admitControl(t, s, "operator", "twin")
	if untilTerminal(t, s, staticRun("twin")).Projection.Status != "cancelled" {
		t.Fatal("operator failed to revoke twin")
	}
	h.snapshot("returns-held")
	close(winner)
	if e = <-done; e == nil || !strings.Contains(e.Error(), "phase2d left settlement aborted") {
		t.Fatal("wrong accounting failure", e)
	}
	decision := func(id string) string {
		return h.data(h.db, "SELECT json_extract(descriptor,'$.Read.Adoption') FROM adapter_results WHERE intent_id=?", id)
	}
	if decision(twin) != "adopted" || decision(duplicate) != "duplicate" || (h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", target) != wantOriginalSettlement || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", twin) != 0) || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", duplicate) != 1 {
		t.Fatal("cleanup did not separate duplicate and original settlements")
	}
	// Use supported manual service while the adopter's genuine debt survives.
	pub, e := s.SettleInvocation(t.Context(), `["parent",1,1,0]`)
	if e != nil || !pub.Accepted {
		t.Fatal(pub, e)
	}
	obs := h.data(h.db, "SELECT observation_id FROM settlements WHERE intent_id=?", duplicate)
	publicStep(t, s, roleRun("l"), obs)
	for steps := 0; h.count(h.db, "SELECT count(*) FROM intents WHERE subject='resolver_l' AND json_extract(descriptor,'$.Obligation')='l_report'") == 0; steps++ {
		if steps > 96 {
			t.Fatal("resolver never reported")
		}
		publicStep(t, s, roleRun("l"), "")
	}
	report := h.data(h.db, "SELECT id FROM intents WHERE subject='resolver_l' AND json_extract(descriptor,'$.Obligation')='l_report'")
	del, e := s.DeliverMessage(t.Context(), report, "phase2d-duplicate-report")
	if e != nil || !del.Accepted {
		t.Fatal(del, e)
	}
	if e = s.AcknowledgeMessage(t.Context(), report, "phase2d-duplicate-report"); e != nil {
		t.Fatal(e)
	}
	h.event("duplicate report and L publication while adopter owes settlement", map[string]any{"Query": duplicate, "Adopter": twin, "Publication": pub, "Report": del, "FaultedOperation": faultID, "OriginalSettled": wantOriginalSettlement == 1})
	if h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", target) != wantOriginalSettlement || h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id=?", twin) != 0 {
		t.Fatal("manual publication changed accounting")
	}
	h.replay("published-unpaid")
	if _, e = db.ExecContext(t.Context(), "DROP TRIGGER inject_left_abort"); e != nil {
		t.Fatal(e)
	}
	q := dispatchTarget(s, staticRun("twin"), false)
	for i := 0; i < 2; i++ {
		e = s.AwaitDispatchQuiescence(t.Context(), q)
		if e == nil || !strings.Contains(e.Error(), "phase2d left settlement aborted") {
			t.Fatal("forgot SQLite failure", e)
		}
		h.event("same-owner twin Q", e.Error())
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s = adverseOpen(t, h.db)
	if e = s.AwaitDispatchQuiescence(t.Context(), q); e != nil {
		t.Fatal(e)
	}
	if h.count(h.db, "SELECT count(*) FROM settlements WHERE intent_id IN (?,?)", target, twin) != 2 {
		t.Fatal("fresh owner left debt")
	}
	h.expectWorld(2, 2, 3)
	h.snapshot("debt-paid")
	adverseRun(t, s, adverseOptions(ready, "success"))
	state := roleState(t, s, "parent")
	if state.Projection.Data["decision"].Text != "applied" || state.Projection.Data["l_report_witness_adoption"].Text != "duplicate" || state.Projection.Data["l_report_witness_query"].Text != duplicate {
		t.Fatal(state.Projection.Status, state.Projection.Data["decision"])
	}
	h.replay("final")
	phaseStopEndpoint(t, ep)
}

func TestPhase2DBindingControls(t *testing.T) {
	for _, name := range []string{"unsupported", "wrong-grant", "read-only", "ledger-replacement", "wrong-role"} {
		t.Run(name, func(t *testing.T) {
			h := newHistory2c(t, name)
			ep, ready := h.endpoint("", nil)
			h.apply = ready.URL
			l, r := frame.RoleOptions{}, frame.RoleOptions{}
			if name == "read-only" {
				r.ReadOnly = true
			}
			if name == "wrong-role" {
				r.Delta = 1
			}
			p := adverseFrame(t, h.apply, l, r)
			if name == "wrong-grant" {
				p.Reconciliation.Grants[1].Invocation = p.Reconciliation.Grants[0].Invocation
			}
			phaseCreate(t, p, h.db)
			mode := "success"
			if name == "unsupported" {
				mode = "unknown"
			}
			if name != "wrong-role" {
				adverseOriginalLoss(t, h, mode)
			}
			s := adverseOpen(t, h.db)
			opts := adverseOptions(ready, mode)
			if name == "wrong-role" {
				adverseRun(t, s, opts)
			}
			ids := adverseQ(t, h, s)
			if name == "ledger-replacement" {
				h.snapshot("old-ledger")
				phaseStopEndpoint(t, ep)
				h.world = filepath.Join(h.work, "replacement.sqlite")
				var changed ready2c
				ep, changed = h.endpoint(phaseAddress(t, h.apply), nil)
				if changed.LedgerID == ready.LedgerID {
					t.Fatal("same ledger")
				}
				h.event("same address new ledger", changed)
			}
			roleService(t, s, "r", "start")
			adverseRun(t, s, opts)
			right := roleState(t, s, "resolver_r")
			if name == "wrong-role" {
				// The actual right actor delivered its own authentic found report.
				// P already selected L, R and T; its pending receive requires L.
				parent := roleState(t, s, "parent")
				if parent.Projection.Data["tree_status"].Text == "" || parent.Projection.Data["l_report_target"].Text != "" || !right.Projection.Data["witness"].Flag {
					t.Fatal("not at L report receive", parent.Projection.PC)
				}
				obs := h.data(h.db, "SELECT id FROM observations WHERE subject='parent' AND json_extract(descriptor,'$.Obligation')='r_report' AND consumed_sequence IS NULL")
				if _, e := s.Commit(t.Context(), parent.Authority(), obs); e == nil {
					t.Fatal("wrong-role report accepted")
				} else {
					h.event("R report rejected at L receive", map[string]any{"Parent": parent.Authority(), "RightObservation": obs, "Error": e.Error(), "RightTarget": ids["r"]})
				}
				next := roleState(t, s, "parent")
				if next.Sequence != parent.Sequence || h.count(h.db, "SELECT count(*) FROM observations WHERE id=? AND consumed_sequence IS NULL", obs) != 1 {
					t.Fatal("wrong-role selection changed history")
				}
			}
			roleService(t, s, "l", "start")
			adverseRun(t, s, opts)
			left := roleState(t, s, "resolver_l")
			parent := roleState(t, s, "parent")
			switch name {
			case "unsupported":
				if left.Projection.Data["witness_adoption"].Text != "unsupported_policy" || !left.Projection.Data["witness"].Flag || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", ids["l"]) != 0 {
					t.Fatal("unsupported policy resolved L")
				}
				h.expectWorld(2, 2, 2)
			case "wrong-grant":
				if right.Projection.Data["query1_disposition"].Text != "read_denied" || right.Projection.Data["witness"].Flag || h.count(h.db, "SELECT count(*) FROM receipt_read_ordinals WHERE json_extract(descriptor,'$.Resolver.Run.Subject')='resolver_r'") != 0 {
					t.Fatal(right.Projection.Data["query1_disposition"])
				}
				h.expectWorld(2, 2, 1)
			case "read-only":
				if right.Projection.Data["witness_adoption"].Text != "denied" || !right.Projection.Data["witness"].Flag || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id=?", ids["r"]) != 0 {
					t.Fatal("read-only imported")
				}
				h.expectWorld(2, 2, 2)
			case "ledger-replacement":
				if left.Projection.Data["query1_valid"].Flag || right.Projection.Data["query1_valid"].Flag || h.count(h.db, "SELECT count(*) FROM adapter_results WHERE intent_id IN (?,?)", ids["l"], ids["r"]) != 0 {
					t.Fatal("replacement supplied evidence")
				}
				h.expectWorld(0, 0, 0)
			case "wrong-role":
				if parent.Projection.Status != "succeeded" || parent.Projection.Data["decision"].Text != "applied" || parent.Projection.Data["l_report_witness_target"].Text != ids["l"] || parent.Projection.Data["r_report_witness_target"].Text != ids["r"] {
					t.Fatal("correct role report failed")
				}
				h.expectWorld(2, 2, 2)
			}
			if name != "wrong-role" && (parent.Projection.Status != "open" || parent.Projection.Data["decision"].Text != "") {
				t.Fatal("false two-role decision")
			}
			h.replay("final")
			phaseStopEndpoint(t, ep)
		})
	}
}
