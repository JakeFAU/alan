package semantics

import (
	"encoding/json"
	"errors"
	"fmt"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	ex "github.com/JakeFAU/alan/internal/executor"
)

// This bridge runs the preserved step+inspect oracle and independently authored
// ordinary runtime programs. It never uses runtime Reduce or Replay as an oracle.
// It is a selected-boundary relation, not a compiler or whole-machine bisimulation.
const bridgeObligation = "inner/g1/declared-obligation/init2"

type bridgeRunView struct {
	Status   string
	Terminal bool
}
type bridgeRequestView struct {
	Source, Target, Obligation, Operation          string
	SourceGeneration, ExpectedGeneration, Revision int64
	Authorized, Decided, Accepted                  bool
	Reason                                         string
	Successor                                      int64
}
type bridgeView struct {
	Generations                         map[string]int64
	Retired                             map[string]bool
	Runs                                map[string]bridgeRunView
	Expected, Spent, Limit              int64
	Pending, Obligation, Classification string
	ResultAccepted                      bool
	ResultSuccessor                     int64
	Requests                            map[string]bridgeRequestView
}
type bridge struct {
	t            *testing.T
	spec         compositionSpec
	model        machine
	monitor      monitor
	trace        []event
	store        *ex.Store
	path         string
	provision    ex.Provision
	ids          map[string]string // exact model ID -> exact runtime ID, never string stripping
	observations map[string]string
	actions      []any
	checkpoints  []any
}

func newBridge(t *testing.T, name string, expectedSeed int, provision ex.Provision) *bridge {
	t.Helper()
	var spec compositionSpec
	for _, candidate := range supervisionScenarios() {
		if candidate.Name == name {
			spec = candidate
		}
	}
	if spec.Name == "" || len(spec.Seed) != expectedSeed {
		t.Fatalf("fixture identity changed: %s seed=%d want=%d", name, len(spec.Seed), expectedSeed)
	}
	b := &bridge{t: t, spec: spec, model: newMachine(spec.Config), monitor: freshMonitor(), provision: provision, ids: map[string]string{}, observations: map[string]string{}}
	b.path = filepath.Join(t.TempDir(), "runtime.sqlite")
	var err error
	b.store, err = ex.Create(t.Context(), b.path, provision, ex.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := b.store.Close(); err != nil {
			t.Error(err)
		}
	})
	return b
}
func (b *bridge) modelEvents(events ...event) {
	b.t.Helper()
	for _, e := range events {
		next, why := step(b.model, e)
		if why != "" {
			b.t.Fatalf("model event %d %+v rejected: %s", len(b.trace)+1, e, why)
		}
		mon, violation := inspect(b.model, next, e, b.monitor)
		if violation != "" {
			b.t.Fatalf("model event %+v safety violation: %s", e, violation)
		}
		b.model, b.monitor = next, mon
		b.trace = append(b.trace, e)
	}
}
func (b *bridge) seedTo(n int) {
	b.t.Helper()
	if n < len(b.trace) || n > len(b.spec.Seed) {
		b.t.Fatal("invalid seed boundary")
	}
	b.modelEvents(b.spec.Seed[len(b.trace):n]...)
}
func (b *bridge) localModel(subject string, g int, obs string) {
	b.modelEvents(event{Kind: prepare, A: subject, G: g, C: 1, Observation: obs}, event{Kind: commit})
}
func (b *bridge) state(subject string, g int64) ex.RunState {
	b.t.Helper()
	s, err := b.store.Snapshot(b.t.Context(), ex.RunID{Subject: subject, Generation: g})
	if err != nil {
		b.t.Fatal(err)
	}
	return s
}
func (b *bridge) commit(subject string, g int64, obs string) ex.RunState {
	b.t.Helper()
	before := b.state(subject, g)
	after, err := b.store.Commit(b.t.Context(), before.Authority(), obs)
	if err != nil {
		b.t.Fatal(err)
	}
	b.actions = append(b.actions, map[string]any{"action": "Commit", "authority": before.Authority(), "selected": obs, "after": after})
	return after
}
func (b *bridge) commits(subject string, g int64, n int) {
	for range n {
		b.commit(subject, g, "")
	}
}
func (b *bridge) inspection() ex.Inspection {
	b.t.Helper()
	s, err := ex.Inspect(b.t.Context(), b.path, ex.DefaultLimits())
	if err != nil {
		b.t.Fatal(err)
	}
	return s
}
func bridgeDecode[T any](t *testing.T, raw any) T {
	t.Helper()
	var v T
	s, ok := raw.(string)
	if !ok {
		t.Fatalf("inspection JSON type %T", raw)
	}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func bridgeEqual(t *testing.T, label string, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s\nwant %s\ngot  %s", label, canonical(want), canonical(got))
	}
}
func (b *bridge) reopen() {
	b.t.Helper()
	before := b.inspection()
	if err := b.store.Close(); err != nil {
		b.t.Fatal(err)
	}
	var err error
	b.store, err = ex.Open(b.t.Context(), b.path, ex.DefaultLimits())
	if err != nil {
		b.t.Fatal(err)
	}
	after := b.inspection()
	// Open renews every open run, including fresh successors. Closed runs stay frozen.
	for i, row := range before.Tables["runs"] {
		prior := bridgeDecode[ex.Projection](b.t, row["projection"])
		expected := row["epoch"].(int64)
		if prior.Status == "open" {
			expected++
		}
		if after.Tables["runs"][i]["epoch"] != expected {
			b.t.Fatal("Open epoch renewal mismatch")
		}
		after.Tables["runs"][i]["epoch"] = row["epoch"]
	}
	bridgeEqual(b.t, "reopen changed durable evidence beyond open-run epochs", before.Tables, after.Tables)
	b.actions = append(b.actions, map[string]any{"action": "Close/Open", "epoch_rule": "each open run +1; terminal epochs unchanged"})
}
func (b *bridge) request(modelID string) ex.LifecycleRequest {
	b.t.Helper()
	id := b.state("inner", 1).Projection.Data["pending"].Text
	if old, ok := b.ids[modelID]; ok && old != id {
		b.t.Fatal("operation ID changed")
	}
	b.ids[modelID] = id
	q, err := b.store.LoadLifecycleRequest(b.t.Context(), id)
	if err != nil {
		b.t.Fatal(err)
	}
	return q
}
func (b *bridge) authorize(q ex.LifecycleRequest) {
	b.t.Helper()
	a, err := b.store.AuthorizeIntent(b.t.Context(), q.Intent.ID)
	if err != nil || !a.Allowed {
		b.t.Fatalf("source authorization %+v %v", a, err)
	}
	b.actions = append(b.actions, map[string]any{"action": "AuthorizeIntent", "result": a})
}
func (b *bridge) decide(q ex.LifecycleRequest) ex.LifecycleResult {
	b.t.Helper()
	r, err := b.store.DecideLifecycle(b.t.Context(), q)
	if err != nil {
		b.t.Fatal(err)
	}
	b.actions = append(b.actions, map[string]any{"action": "DecideLifecycle", "request": q, "result": r})
	return r
}
func (b *bridge) settle(modelID string) string {
	b.t.Helper()
	r, err := b.store.SettleLifecycle(b.t.Context(), b.ids[modelID], "bridge-result:"+modelID)
	if err != nil || !r.Accepted {
		b.t.Fatalf("settlement %+v %v", r, err)
	}
	b.observations[resultID(modelID, "")] = r.ObservationID
	b.actions = append(b.actions, map[string]any{"action": "SettleLifecycle", "id": b.ids[modelID], "result": r})
	return r.ObservationID
}
func (b *bridge) terminal() string {
	b.t.Helper()
	r, err := b.store.DeliverTerminal(b.t.Context(), ex.RunID{Subject: "worker", Generation: 1}, ex.TerminalDestination{Target: "inner", Obligation: bridgeObligation}, "bridge-terminal")
	if err != nil || !r.Accepted {
		b.t.Fatalf("terminal %+v %v", r, err)
	}
	b.observations[terminalID("worker", 1, "")] = r.ObservationID
	b.actions = append(b.actions, map[string]any{"action": "DeliverTerminal", "result": r})
	return r.ObservationID
}
func (b *bridge) modelView() bridgeView {
	v := bridgeView{Generations: map[string]int64{}, Retired: map[string]bool{}, Runs: map[string]bridgeRunView{}, Requests: map[string]bridgeRequestView{}}
	for _, name := range []string{"inner", "worker"} {
		a := b.model.Subjects[name]
		v.Generations[name] = int64(a.Current)
		v.Retired[name] = a.Retired
		for g := 1; g <= a.Current; g++ {
			r := b.model.Runs[runID(name, g)]
			status := r.Projection.Status
			if status == "active" {
				status = "open"
			}
			v.Runs[runID(name, g)] = bridgeRunView{status, r.Outcome != nil}
		}
	}
	d := b.model.Runs[runID("inner", 1)].Projection.Local
	if d != nil {
		v.Expected = int64(d.Values[childGeneration])
		v.Spent = int64(d.Values[spent])
		v.Limit = int64(d.Values[limit])
		v.Obligation = d.Names[1]
		v.Pending = d.Pending
		v.ResultAccepted = d.Values[acceptedResult] == 1
		v.ResultSuccessor = int64(d.Values[successor])
		v.Classification = map[int]string{0: "", 1: "succeeded", 2: "failed", 3: "abandoned", 4: "cancelled", 5: "suspicion"}[d.Values[classification]]
	}
	for id, in := range b.model.Intents {
		if in.Descriptor.Lifecycle == "" {
			continue
		}
		q := requestValue(in.Descriptor.Lifecycle)
		view := bridgeRequestView{Source: q.Requester, SourceGeneration: int64(q.RequesterG), Target: q.Target, ExpectedGeneration: int64(q.ExpectedG), Obligation: q.Obligation, Operation: string(q.Operation), Revision: int64(q.W)}
		if b.model.Lifecycle != nil {
			_, view.Authorized = b.model.Lifecycle.Authorized[id]
			r, ok := b.model.Lifecycle.Results[id]
			view.Decided = ok
			view.Accepted = r.Accepted
			view.Successor = int64(r.Successor)
			view.Reason = r.Reason
		}
		v.Requests[id] = view
	}
	return v
}
func (b *bridge) runtimeView(s ex.Inspection) bridgeView {
	b.t.Helper()
	v := bridgeView{Generations: map[string]int64{}, Retired: map[string]bool{}, Runs: map[string]bridgeRunView{}, Requests: map[string]bridgeRequestView{}}
	for _, row := range s.Tables["subjects"] {
		name := row["id"].(string)
		if name != "inner" && name != "worker" {
			continue
		}
		v.Generations[name] = row["generation"].(int64)
		v.Retired[name] = row["retired"].(int64) != 0
	}
	for _, row := range s.Tables["runs"] {
		name := row["subject"].(string)
		if name != "inner" && name != "worker" {
			continue
		}
		g := row["generation"].(int64)
		p := bridgeDecode[ex.Projection](b.t, row["projection"])
		terminal := false
		for _, end := range s.Tables["terminal_obligations"] {
			if end["subject"] == name && end["generation"] == g {
				terminal = true
				frozen := bridgeDecode[ex.Projection](b.t, end["outcome"])
				bridgeEqual(b.t, "frozen terminal outcome status", p.Status, frozen.Status)
			}
		}
		v.Runs[runID(name, int(g))] = bridgeRunView{p.Status, terminal}
		if name == "inner" && g == 1 {
			v.Expected = p.Data["expected"].Number
			v.Spent = p.Data["spent"].Number
			v.Limit = p.Data["limit"].Number
			v.Obligation = p.Data["obligation"].Text
			v.Classification = p.Data["outcome"].Text
			v.ResultAccepted = p.Data["accepted"].Flag
			v.ResultSuccessor = p.Data["successor"].Number
			pending := p.Data["pending"].Text
			if pending != "" {
				found := false
				for mid, rid := range b.ids {
					if rid == pending {
						v.Pending = mid
						found = true
					}
				}
				if !found {
					b.t.Fatalf("unmapped pending ID %s", pending)
				}
			}
		}
	}
	for mid, rid := range b.ids {
		if b.model.Intents[mid] == nil || b.model.Intents[mid].Descriptor.Lifecycle == "" {
			continue
		}
		q, err := b.store.LoadLifecycleRequest(b.t.Context(), rid)
		if err != nil {
			b.t.Fatal(err)
		}
		view := bridgeRequestView{Source: q.Intent.Run.Subject, SourceGeneration: q.Intent.Run.Generation, Target: q.Intent.Target, ExpectedGeneration: q.Intent.Generation, Obligation: q.Intent.Obligation, Operation: q.Operation, Revision: q.ExpectedRevision}
		if q.SourceRevision != 7 {
			b.t.Fatal("source revision changed")
		}
		for _, row := range s.Tables["attempts"] {
			if row["intent_id"] == rid {
				a := bridgeDecode[ex.Authorization](b.t, row["result"])
				view.Authorized = a.Allowed
			}
		}
		for _, row := range s.Tables["lifecycle_results"] {
			if row["id"] == rid {
				r := bridgeDecode[ex.LifecycleResult](b.t, row["result"])
				bridgeEqual(b.t, "immutable result request", q, r.Request)
				view.Decided = true
				view.Accepted = r.Accepted
				view.Successor = r.Successor
				reasons := map[string]string{"retried": "", "not_retryable": "retry policy", "grant_denied": "authority"}
				reason, ok := reasons[r.Reason]
				if !ok {
					b.t.Fatalf("unmapped reason %q", r.Reason)
				}
				view.Reason = reason
			}
		}
		v.Requests[mid] = view
	}
	return v
}
func (b *bridge) check(label string) {
	b.t.Helper()
	snapshot := b.inspection()
	want, got := b.modelView(), b.runtimeView(snapshot)
	lifecycleIntents := 0
	for _, row := range snapshot.Tables["intents"] {
		in := bridgeDecode[ex.Intent](b.t, row["descriptor"])
		if in.Kind == "lifecycle" {
			lifecycleIntents++
		}
	}
	bridgeEqual(b.t, "no extra lifecycle operations", len(want.Requests), lifecycleIntents)
	decisions := 0
	for _, q := range want.Requests {
		if q.Decided {
			decisions++
		}
	}
	bridgeEqual(b.t, "one retained target decision per decided operation", decisions, len(snapshot.Tables["lifecycle_results"]))
	bridgeEqual(b.t, label, want, got)
	// Notice binding and one-time consumption are compared independently of local fields.
	for mid, rid := range b.observations {
		var mo observation
		var bound string
		for id, r := range b.model.Runs {
			if o, ok := r.Inbox[mid]; ok {
				mo = o
				bound = id
			}
		}
		found := false
		for _, row := range snapshot.Tables["observations"] {
			if row["id"] != rid {
				continue
			}
			found = true
			o := bridgeDecode[ex.Observation](b.t, row["descriptor"])
			bridgeEqual(b.t, "notice bound run", bound, runID(o.Run.Subject, int(o.Run.Generation)))
			bridgeEqual(b.t, "notice consumption", mo.Consumed, row["consumed_sequence"] != nil)
			n := noticeValue(mo.Notice)
			bridgeEqual(b.t, "notice source", n.Subject, o.Source)
			bridgeEqual(b.t, "notice source generation", int64(n.G), o.SourceGeneration)
			bridgeEqual(b.t, "notice obligation", bridgeObligation, o.Obligation)
			if n.Kind == "lifecycle" {
				bridgeEqual(b.t, "result provenance", "runtime-lifecycle", o.Provenance)
				bridgeEqual(b.t, "result operation", b.ids[n.Logical], o.Operation)
			} else {
				bridgeEqual(b.t, "terminal provenance", "runtime-terminal", o.Provenance)
			}
		}
		if !found {
			b.t.Fatal("missing notice")
		}
	}
	b.checkpoints = append(b.checkpoints, map[string]any{"label": label, "model_event_count": len(b.trace), "expected": want, "actual": got, "runtime": snapshot, "assertions": "equal public projection; immutable requests/results; notice binding/correlation/consumption"})
}
func (b *bridge) emitRecord(limits string) {
	b.t.Helper()
	record := map[string]any{"test": b.t.Name(), "fixture": b.spec.Name, "model_config": b.spec.Config, "model_seed": b.spec.Seed, "model_trace": b.trace, "runtime_provision": b.provision, "id_map": b.ids, "observation_map": b.observations, "runtime_actions": b.actions, "checkpoints": b.checkpoints, "limits": limits, "passed": true, "mapping": "active=open; exact operation table; retry policy=not_retryable; authority=grant_denied; accepted empty reason=retried; five ordinary init commits; selected failure policy only; receive+clear maps to model receive; no PC/sequence/epoch equality across physical reopen"}
	b.t.Log("CONFORMANCE " + canonical(record))
}
func bridgeAdd(t *testing.T, p *ex.Provision, name string, code []ex.Instruction, data map[string]ex.Value) {
	t.Helper()
	raw, id, err := ex.EncodeProgram(ex.Program{Semantics: ex.SemanticVersion, Code: code})
	if err != nil {
		t.Fatal(err)
	}
	p.Programs = append(p.Programs, ex.Artifact{ID: id, Semantics: ex.SemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, ex.Subject{ID: name, Generation: 1, Owner: "owner", Revision: 7})
	p.Runs = append(p.Runs, ex.InitialRun{ID: ex.RunID{Subject: name, Generation: 1}, ProgramID: id, Projection: ex.Projection{Status: "open", Data: data}})
}
func bridgeRetry(reserve bool) ex.Instruction {
	in := ex.Instruction{Op: "emit", Dst: "pending", Effect: &ex.Effect{Kind: "lifecycle", Target: ex.Ref("child"), Generation: ex.Ref("expected"), Obligation: ex.Ref("obligation"), Payload: map[string]ex.Operand{"operation": ex.Lit(ex.Text("Retry")), "revision": ex.Ref("revision")}}}
	if reserve {
		delta := int64(1)
		in.Updates = map[string]ex.Operand{"spent": {Ref: "spent", Offset: &delta}}
	}
	return in
}
func bridgeReceive(kind, provenance string, operation ex.Operand, bind map[string]string) ex.Instruction {
	return ex.Instruction{Op: "receive", Match: &ex.Match{Kind: kind, Provenance: provenance, Source: ex.Ref("child"), Generation: ex.Ref("expected"), Obligation: ex.Ref("obligation"), Operation: operation}, Bind: bind}
}
func bridgeProvision(t *testing.T, raw, grant bool, worker []ex.Instruction) ex.Provision {
	t.Helper()
	p := ex.Provision{}
	code := []ex.Instruction{
		{Op: "set", Dst: "child", A: ex.Lit(ex.Text("worker"))},
		{Op: "set", Dst: "obligation", A: ex.Lit(ex.Text(bridgeObligation))},
		{Op: "set", Dst: "expected", A: ex.Lit(ex.Int(1))},
		{Op: "set", Dst: "revision", A: ex.Lit(ex.Int(7))},
		{Op: "set", Dst: "limit", A: ex.Lit(ex.Int(1))},
	}
	resultReceive := bridgeReceive("lifecycle", "runtime-lifecycle", ex.Ref("pending"), map[string]string{"accepted": "accepted", "successor": "successor", "reason": "reason"})
	clear := ex.Instruction{Op: "set", Dst: "pending", A: ex.Lit(ex.Text(""))}
	if raw {
		code = append(code, bridgeRetry(false), resultReceive, clear, ex.Instruction{Op: "end", Outcome: "failed"})
	} else {
		code = append(code,
			bridgeReceive("terminal", "runtime-terminal", ex.Lit(ex.Text("")), map[string]string{"outcome": "outcome"}), //5
			ex.Instruction{Op: "branch", A: ex.Ref("outcome"), B: ex.Lit(ex.Text("failed")), Target: 7, Otherwise: 14},
			ex.Instruction{Op: "branch", A: ex.Ref("spent"), B: ex.Ref("limit"), Target: 15, Otherwise: 8},
			bridgeRetry(true), resultReceive, clear, //8,9,10
			ex.Instruction{Op: "branch", A: ex.Ref("accepted"), B: ex.Lit(ex.Value{Kind: "bool", Flag: true}), Target: 12, Otherwise: 15},
			ex.Instruction{Op: "set", Dst: "expected", A: ex.Ref("successor")},
			ex.Instruction{Op: "jump", Target: 5},
			ex.Instruction{Op: "end", Outcome: "succeeded"}, ex.Instruction{Op: "end", Outcome: "failed"})
	}
	bridgeAdd(t, &p, "inner", code, map[string]ex.Value{"spent": ex.Int(0), "pending": ex.Text("")})
	bridgeAdd(t, &p, "worker", worker, map[string]ex.Value{})
	p.Permissions = []ex.Permission{{Kind: "observation", Source: "worker", Target: "inner", Operation: "terminal", Revision: 7, Obligation: bridgeObligation}}
	if grant {
		p.Permissions = append(p.Permissions, ex.Permission{Kind: "lifecycle", Source: "inner", Target: "worker", Operation: "Retry", Revision: 7})
	}
	return p
}
func (b *bridge) standardPrefix() ex.LifecycleRequest {
	b.seedTo(10)
	b.commits("inner", 1, 5)
	b.check("ordinary initialization")
	b.seedTo(12)
	b.commit("worker", 1, "")
	b.check("actual worker failure")
	b.seedTo(13)
	obs := b.terminal()
	b.check("terminal admitted, not consumed")
	b.seedTo(15)
	b.commit("inner", 1, obs)
	b.check("terminal consumed once")
	b.seedTo(21)
	b.commits("inner", 1, 2)
	b.check("failure and budget branches before emit")
	b.seedTo(23)
	b.commit("inner", 1, "")
	q := b.request(intentID("inner", 1, 10))
	b.check("atomic pending and spent reservation")
	return q
}
func TestRuntimeConformanceAcceptedLossAndSuccessor(t *testing.T) {
	b := newBridge(t, "B_accepted_lost_ack_and_successor", 25, bridgeProvision(t, false, true, []ex.Instruction{{Op: "end", Outcome: "failed"}}))
	q := b.standardPrefix()
	b.seedTo(24)
	b.authorize(q)
	b.check("source authorized; no target result")
	b.seedTo(25)
	original := b.decide(q)
	b.check("accepted successor without result admission")
	b.reopen()
	b.check("accepted decision survives reopen")
	b.modelEvents(event{Kind: decideLifecycle, ID: intentID("inner", 1, 10)})
	bridgeEqual(t, "accepted retransmission", original, b.decide(q))
	b.check("lost result retransmitted")
	b.reopen()
	b.check("retransmission survives reopen")
	b.localModel("worker", 2, "")
	b.commit("worker", 2, "")
	b.check("successor executes unchanged worker artifact")
	b.emitRecord("No result admission before successor work. Physical reopen epochs are implementation stutters, checked separately. Worker succeeds in doing its declared Fault, not a business success claim.")
}
func TestRuntimeConformanceInterruptedCommittedRequest(t *testing.T) {
	b := newBridge(t, "C_committed_intent_interrupted", 23, bridgeProvision(t, false, true, []ex.Instruction{{Op: "end", Outcome: "failed"}}))
	q := b.standardPrefix()
	id := intentID("inner", 1, 10)
	b.modelEvents(event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7})
	b.authorize(q)
	b.check("authorized after committed request")
	b.modelEvents(event{Kind: decideLifecycle, ID: id})
	b.decide(q)
	b.check("accepted before interruption")
	b.modelEvents(event{Kind: admitLifecycle, ID: id})
	obs := b.settle(id)
	b.check("result admitted before speculative receive")
	old := b.state("inner", 1)
	b.modelEvents(event{Kind: prepare, A: "inner", G: 1, C: 1, Observation: resultID(id, "")}, event{Kind: interrupt}, life(reactivate, "inner", 1))
	b.reopen()
	b.check("same-execution interruption then reactivation")
	before := b.inspection()
	if _, err := b.store.Commit(t.Context(), old.Authority(), obs); !errors.Is(err, ex.ErrStale) {
		t.Fatalf("old epoch committed: %v", err)
	}
	bridgeEqual(t, "stale commit changed evidence", before, b.inspection())
	fresh := b.state("inner", 1)
	if fresh.Epoch != int64(b.model.Runs[runID("inner", 1)].E) || fresh.Sequence != old.Sequence {
		t.Fatal("interrupted epoch/sequence mismatch")
	}
	b.localModel("inner", 1, resultID(id, ""))
	b.consumeResult(obs, true)
	b.check("retained result consumed and pending cleared")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.check("ordinary successor expectation updated")
	b.emitRecord("Runtime has no speculative proposal API. Model Prepare/Interrupt maps to omitted uncommitted runtime work. Runtime receive and pending clear require two commits; only combined boundary matches model atomic receive. Not a process kill or transaction crash claim.")
}
func TestRuntimeConformanceRejectionSurvivesFailure(t *testing.T) {
	p := bridgeProvision(t, true, true, []ex.Instruction{{Op: "end", Outcome: "failed"}})
	// A second statically declared requester is a separately labeled runtime extension.
	bridgeAdd(t, &p, "probe", []ex.Instruction{bridgeRetry(false), {Op: "end", Outcome: "succeeded"}}, map[string]ex.Value{"child": ex.Text("worker"), "expected": ex.Int(1), "revision": ex.Int(7), "obligation": ex.Text("fresh-probe")})
	p.Permissions = append(p.Permissions, ex.Permission{Kind: "lifecycle", Source: "probe", Target: "worker", Operation: "Retry", Revision: 7})
	b := newBridge(t, "B_rejected_then_failed_retransmission", 14, p)
	b.seedTo(10)
	b.commits("inner", 1, 5)
	b.check("active worker initialized")
	b.seedTo(12)
	b.commit("inner", 1, "")
	q := b.request(intentID("inner", 1, 6))
	b.check("unbudgeted ordinary probe emitted")
	b.seedTo(13)
	b.authorize(q)
	b.check("source authorized independently of eligibility")
	b.seedTo(14)
	original := b.decide(q)
	b.check("active target rejection retained")
	b.reopen()
	b.check("rejection reopened")
	b.localModel("worker", 1, "")
	b.commit("worker", 1, "")
	b.check("actual failure changes target eligibility")
	b.reopen()
	b.check("failure reopened before retransmission")
	id := intentID("inner", 1, 6)
	b.modelEvents(event{Kind: decideLifecycle, ID: id})
	bridgeEqual(t, "immutable rejected result", original, b.decide(q))
	b.check("same operation remains rejected after failure")
	b.modelEvents(event{Kind: admitLifecycle, ID: id})
	obs := b.settle(id)
	b.localModel("inner", 1, resultID(id, ""))
	b.consumeResult(obs, false)
	b.check("original rejection consumed")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.check("ordinary rejection escalation")
	b.commit("probe", 1, "")
	freshID := b.state("probe", 1).Projection.Data["pending"].Text
	fresh, err := b.store.LoadLifecycleRequest(t.Context(), freshID)
	if err != nil {
		t.Fatal(err)
	}
	b.authorize(fresh)
	accepted := b.decide(fresh)
	if !accepted.Accepted || accepted.Successor != 2 || freshID == q.Intent.ID {
		t.Fatal("distinct eligible operation did not accept")
	}
	bridgeEqual(t, "original rejection retained after new acceptance", original, b.decide(q))
	b.commit("worker", 2, "")
	b.checkpoints = append(b.checkpoints, map[string]any{"label": "runtime extension: distinct statically declared requester accepts and successor executes", "original": original, "fresh": accepted, "runtime": b.inspection()})
	b.emitRecord("Original model seed and suffix remain unchanged. The final distinct probe and successor work are a separate runtime contract extension; model raw requester has no second Emit. This is not failure-driven supervisory policy.")
}
func TestRuntimeConformanceObservationDoesNotGrantAuthority(t *testing.T) {
	b := newBridge(t, "I_observation_without_lifecycle_authority", 23, bridgeProvision(t, false, false, []ex.Instruction{{Op: "end", Outcome: "failed"}}))
	q := b.standardPrefix()
	id := intentID("inner", 1, 10)
	b.modelEvents(event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7})
	b.authorize(q)
	b.check("source authority succeeds")
	b.modelEvents(event{Kind: decideLifecycle, ID: id})
	b.decide(q)
	b.check("target authority rejects despite observation grant")
	b.modelEvents(event{Kind: admitLifecycle, ID: id})
	obs := b.settle(id)
	b.localModel("inner", 1, resultID(id, ""))
	b.consumeResult(obs, false)
	b.check("authority rejection consumed")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.localModel("inner", 1, "")
	b.commit("inner", 1, "")
	b.check("ordinary authority rejection escalation")
	b.emitRecord("Source authorization is successful; target grant denial is independently retained. Observation permission conveys no lifecycle permission.")
}

func (b *bridge) consumeResult(obs string, reopen bool) {
	b.t.Helper()
	before := b.state("inner", 1)
	after := b.commit("inner", 1, obs)
	bridgeEqual(b.t, "runtime receive must retain pending until ordinary clear", before.Projection.Data["pending"], after.Projection.Data["pending"])
	snapshot := b.inspection()
	found := false
	for _, row := range snapshot.Tables["observations"] {
		if row["id"] == obs {
			found = row["consumed_sequence"] != nil
		}
	}
	if !found {
		b.t.Fatal("runtime receive failed to consume")
	}
	b.checkpoints = append(b.checkpoints, map[string]any{"label": "unmatched intermediate: runtime consumed result but pending not yet cleared", "relation": "not equivalent and not a stutter of the selected public projection", "model": b.modelView(), "runtime": b.runtimeView(snapshot), "snapshot": snapshot})
	if reopen {
		b.reopen()
		bridgeEqual(b.t, "residual receive continuation survived Open", after.Projection, b.state("inner", 1).Projection)
	}
	b.commit("inner", 1, "")
}

func bridgeExternalWorker(endpoint string) []ex.Instruction {
	return []ex.Instruction{
		{Op: "emit", Dst: "pending-effect", Effect: &ex.Effect{Kind: "external", Target: ex.Lit(ex.Text(endpoint)), Generation: ex.Lit(ex.Int(1)), Obligation: ex.Lit(ex.Text("business:order-7")), Payload: map[string]ex.Operand{"profile": ex.Lit(ex.Text("idempotent")), "delta": ex.Lit(ex.Int(1))}}},
		{Op: "receive", Match: &ex.Match{Kind: "input", Provenance: "local-input", Source: ex.Lit(ex.Text("local")), Generation: ex.Lit(ex.Int(1)), Obligation: ex.Lit(ex.Text("control")), Operation: ex.Lit(ex.Text(""))}, Bind: map[string]string{"mode": "mode"}},
		{Op: "branch", A: ex.Ref("mode"), B: ex.Lit(ex.Text("fail")), Target: 5, Otherwise: 3},
		{Op: "receive", Match: &ex.Match{Kind: "receipt", Provenance: "resource-receipt", Source: ex.Lit(ex.Text(endpoint)), Generation: ex.Lit(ex.Int(1)), Obligation: ex.Lit(ex.Text("business:order-7")), Operation: ex.Ref("pending-effect")}, Bind: map[string]string{"total": "total"}},
		{Op: "end", Outcome: "succeeded"},
		{Op: "end", Outcome: "failed"},
	}
}
func (b *bridge) control(g int64, mode string) {
	b.t.Helper()
	input := ex.Input{TransportID: fmt.Sprintf("control-%d", g), LogicalID: fmt.Sprintf("control-%d", g), Target: "worker", Generation: g, Obligation: "control", Payload: map[string]ex.Value{"mode": ex.Text(mode)}}
	admission, err := b.store.AdmitInput(b.t.Context(), input)
	if err != nil || !admission.Accepted {
		b.t.Fatalf("control %+v %v", admission, err)
	}
	b.actions = append(b.actions, map[string]any{"action": "runtime-only ordinary input", "input": input, "admission": admission})
	b.commit("worker", g, admission.ObservationID)
	b.commit("worker", g, "")
	if mode == "fail" {
		b.commit("worker", g, "")
	}
}
func (b *bridge) dispatchExternal(id string, want bool) {
	b.t.Helper()
	settled, err := b.store.DispatchExternal(b.t.Context(), id)
	if err != nil || settled != want {
		b.t.Fatalf("external settled=%t want=%t: %v", settled, want, err)
	}
	b.actions = append(b.actions, map[string]any{"action": "DispatchExternal", "id": id, "settled": settled})
}
func (b *bridge) externalCheck(label, resourcePath string) {
	b.t.Helper()
	b.check(label)
	snapshot := b.inspection()
	world, err := resourcefixture.Inspect(b.t.Context(), resourcePath)
	if err != nil {
		b.t.Fatal(err)
	}
	bridgeEqual(b.t, "external mutation count", b.model.World.Mutations, len(world.Applications))
	views := map[string]any{}
	for mid, in := range b.model.Intents {
		if in.Descriptor.Lifecycle != "" {
			continue
		}
		rid, ok := b.ids[mid]
		if !ok {
			b.t.Fatalf("unmapped external %s", mid)
		}
		var intent ex.Intent
		found := false
		for _, row := range snapshot.Tables["intents"] {
			if row["id"] == rid {
				intent = bridgeDecode[ex.Intent](b.t, row["descriptor"])
				found = true
			}
		}
		if !found {
			b.t.Fatal("missing external intent")
		}
		bridgeEqual(b.t, "external source run", ex.RunID{Subject: in.Descriptor.A, Generation: int64(in.Descriptor.G)}, intent.Run)
		bridgeEqual(b.t, "declared business identity", in.Descriptor.Tag, intent.Obligation)
		bridgeEqual(b.t, "declared counter delta", int64(in.Descriptor.Value), intent.Payload["delta"].Number)
		authorized, receipt, settled := false, false, false
		for _, row := range snapshot.Tables["external_attempts"] {
			if row["intent_id"] == rid {
				a := bridgeDecode[ex.ExternalAttempt](b.t, row["descriptor"])
				bridgeEqual(b.t, "immutable attempt intent", intent, a.Intent)
				authorized = a.Allowed
				bridgeEqual(b.t, "external request key", ex.ExternalKey(snapshot.Namespace, rid), a.Request.Key)
			}
		}
		for _, row := range snapshot.Tables["external_receipts"] {
			if row["intent_id"] == rid {
				receipt = true
				r := bridgeDecode[resourcefixture.Receipt](b.t, row["descriptor"])
				bridgeEqual(b.t, "receipt key", ex.ExternalKey(snapshot.Namespace, rid), r.Key)
			}
		}
		for _, row := range snapshot.Tables["settlements"] {
			if row["intent_id"] == rid {
				settled = true
			}
		}
		bridgeEqual(b.t, "external authorization", b.model.hasAttempt(mid), authorized)
		_, modelReceipt := b.model.Receipts[attemptID(mid, 1)]
		bridgeEqual(b.t, "external receipt", modelReceipt, receipt)
		bridgeEqual(b.t, "external settlement", in.Settled, settled)
		views[mid] = map[string]any{"runtime_id": rid, "authorized": authorized, "receipt": receipt, "settled": settled, "model_suppressed": in.Suppressed}
	}
	b.checkpoints = append(b.checkpoints, map[string]any{"label": label + " external common projection", "external": views, "independent_resource": world, "limits": "in-process loopback protocol endpoint; endpoint inspection is test evidence only; no resource DB reads by ordinary executor"})
}
func TestRuntimeConformanceExternalGenerationsAndLateReceipt(t *testing.T) {
	resourcePath := filepath.Join(t.TempDir(), "resource.sqlite")
	endpoint, err := resourcefixture.Open(t.Context(), resourcePath, "conformance-endpoint")
	if err != nil {
		t.Fatal(err)
	}
	defer endpoint.Close()
	var reply atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request resourcefixture.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		receipt, err := endpoint.Apply(r.Context(), request)
		if err != nil {
			t.Error(err)
			return
		}
		if !reply.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if err := json.NewEncoder(w).Encode(receipt); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	p := bridgeProvision(t, false, true, bridgeExternalWorker(server.URL))
	p.Permissions = append(p.Permissions, ex.Permission{Kind: "external", Source: "worker", Target: server.URL, Operation: "apply", Revision: 7}, ex.Permission{Kind: "input", Source: "local", Target: "worker", Operation: "input", Obligation: "control"})
	b := newBridge(t, "J_unknown_effect_recovery_late_settlement", 38, p)
	b.seedTo(10)
	b.commits("inner", 1, 5)
	b.check("J initialized")
	oldModel := intentID("worker", 1, 1)
	b.seedTo(12)
	b.commit("worker", 1, "")
	oldID := b.state("worker", 1).Projection.Data["pending-effect"].Text
	b.ids[oldModel] = oldID
	b.externalCheck("old intent committed without attempt", resourcePath)
	b.reopen()
	b.externalCheck("committed external intent survives reopen without dispatch", resourcePath)
	// DispatchExternal combines an observable source authorization, durable attempt,
	// and HTTP call. Compare after model AuthorizeAttempt+ApplyExternal, not between.
	b.seedTo(14)
	b.dispatchExternal(oldID, false)
	b.externalCheck("application with missing receipt", resourcePath)
	b.reopen()
	b.externalCheck("unknown application survives reopen", resourcePath)
	b.seedTo(16)
	b.control(1, "fail")
	b.externalCheck("old execution fails while effect remains unknown", resourcePath)
	beforeClosed := b.inspection()
	if _, err := b.store.Commit(t.Context(), b.state("worker", 1).Authority(), ""); !errors.Is(err, ex.ErrClosed) {
		t.Fatalf("closed run advanced %v", err)
	}
	bridgeEqual(t, "closed advancement changed evidence", beforeClosed, b.inspection())
	b.seedTo(17)
	notice := b.terminal()
	b.check("J terminal admission")
	b.seedTo(19)
	b.commit("inner", 1, notice)
	b.check("J terminal consumed")
	b.seedTo(25)
	b.commits("inner", 1, 2)
	b.check("J failure/budget branch")
	b.seedTo(27)
	b.commit("inner", 1, "")
	q := b.request(intentID("inner", 1, 10))
	b.check("J retry reservation")
	b.seedTo(28)
	b.authorize(q)
	b.seedTo(29)
	b.decide(q)
	b.externalCheck("new worker execution exists; old effect unresolved", resourcePath)
	b.seedTo(30)
	obs := b.settle(intentID("inner", 1, 10))
	b.seedTo(32)
	b.consumeResult(obs, false)
	b.check("J lifecycle result consumed")
	b.seedTo(38)
	b.commits("inner", 1, 3)
	b.externalCheck("inner waits for current generation two", resourcePath)
	b.reopen()
	b.externalCheck("retry and old uncertainty survive reopen", resourcePath)
	freshModel := intentID("worker", 2, 1)
	b.localModel("worker", 2, "")
	b.commit("worker", 2, "")
	freshID := b.state("worker", 2).Projection.Data["pending-effect"].Text
	b.ids[freshModel] = freshID
	if freshID == oldID {
		t.Fatal("generation reused execution key")
	}
	b.externalCheck("actual successor emits same declared business mutation with distinct key", resourcePath)
	b.modelEvents(event{Kind: authorize, ID: freshModel, Attempt: 1, Owner: "owner", W: 7}, event{Kind: applyExternal, ID: freshModel, Attempt: 1})
	b.dispatchExternal(freshID, false)
	b.externalCheck("two execution keys permit two business mutations", resourcePath)
	// Runtime-only inputs position the successor at a real receipt receive. The
	// preserved model worker instead has Fault next; whole-trace equivalence ends.
	b.control(2, "wait")
	newBefore := b.state("worker", 2)
	innerBefore := b.state("inner", 1)
	if newBefore.Projection.PC != 3 || newBefore.Projection.Data["pending-effect"].Text != freshID {
		t.Fatal("successor not waiting for its actual effect")
	}
	b.modelEvents(event{Kind: recordReceipt, ID: oldModel, Attempt: 1})
	reply.Store(true)
	b.dispatchExternal(oldID, true)
	b.externalCheck("late old receipt settles only old execution", resourcePath)
	bridgeEqual(t, "late old receipt changed successor", newBefore, b.state("worker", 2))
	bridgeEqual(t, "late receipt changed supervisor pending/budget", innerBefore, b.state("inner", 1))
	snapshot := b.inspection()
	var oldObservation string
	for _, row := range snapshot.Tables["observations"] {
		o := bridgeDecode[ex.Observation](t, row["descriptor"])
		if o.Kind == "receipt" && o.Operation == oldID {
			oldObservation = o.ID
			bridgeEqual(t, "late receipt binding", ex.RunID{Subject: "worker", Generation: 1}, o.Run)
			bridgeEqual(t, "late receipt provenance", "resource-receipt", o.Provenance)
			bridgeEqual(t, "late receipt obligation", "business:order-7", o.Obligation)
			if row["consumed_sequence"] != nil {
				t.Fatal("closed source consumed late evidence")
			}
		}
	}
	if oldObservation == "" {
		t.Fatal("missing old receipt observation")
	}
	if _, err := b.store.Commit(t.Context(), newBefore.Authority(), oldObservation); !errors.Is(err, ex.ErrIneligible) {
		t.Fatalf("old receipt consumed by successor %v", err)
	}
	bridgeEqual(t, "wrong old receipt changed evidence", snapshot, b.inspection())
	b.reopen()
	bridgeEqual(t, "pending effect survived reopen", newBefore.Projection, b.state("worker", 2).Projection)
	// Separately prove there was eligible *new* work, not merely an unrelated guard.
	b.dispatchExternal(freshID, true)
	var freshObservation string
	for _, row := range b.inspection().Tables["observations"] {
		o := bridgeDecode[ex.Observation](t, row["descriptor"])
		if o.Kind == "receipt" && o.Operation == freshID {
			freshObservation = o.ID
		}
	}
	if freshObservation == "" {
		t.Fatal("missing current receipt")
	}
	after := b.commit("worker", 2, freshObservation)
	if after.Projection.Data["total"] != ex.Int(2) {
		t.Fatal("current receipt not consumed")
	}
	b.commit("worker", 2, "")
	world, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if world.Total != 2 || len(world.Applications) != 2 || world.Contacts != 4 {
		t.Fatalf("business mutation evidence %+v", world)
	}
	b.checkpoints = append(b.checkpoints, map[string]any{"label": "runtime-only current receipt consumption after rejection of old receipt", "runtime": b.inspection(), "resource": world, "new_receipt_observation": freshObservation})
	b.emitRecord("Common J checkpoints compare lifecycle/public status and effect authorization/uncertainty/settlement. Runtime worker adds explicit local input receive and branch to choose failure versus receipt wait; model worker has Emit then Fault. These extra transitions and the final new receipt consumption are a separate runtime contract, not whole-trace equivalence. Original late receipt is retrieved by idempotent HTTP retransmission after prior application; the original endpoint receipt is retained. Distinct execution keys permit two business mutations. In-process endpoint; Task4 owns process isolation.")
}

func TestRuntimeConformanceClosureBeforeExternalAuthorization(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	code := bridgeExternalWorker(server.URL)
	p := bridgeProvision(t, false, true, []ex.Instruction{code[0], {Op: "end", Outcome: "failed"}})
	p.Permissions = append(p.Permissions, ex.Permission{Kind: "external", Source: "worker", Target: server.URL, Operation: "apply", Revision: 7})
	b := newBridge(t, "J_unknown_effect_recovery_late_settlement", 38, p)
	b.seedTo(10)
	b.commits("inner", 1, 5)
	b.seedTo(12)
	b.commit("worker", 1, "")
	mid := intentID("worker", 1, 1)
	rid := b.state("worker", 1).Projection.Data["pending-effect"].Text
	b.ids[mid] = rid
	b.localModel("worker", 1, "")
	b.commit("worker", 1, "")
	b.check("terminal closure before any external authorization")
	if !b.model.Intents[mid].Suppressed {
		t.Fatal("model did not suppress unauthorized closed intent")
	}
	deniedEvent := event{Kind: authorize, ID: mid, Attempt: 1, Owner: "owner", W: 7}
	next, why := step(b.model, deniedEvent)
	if why != "closed" {
		t.Fatalf("model closure rejection %q", why)
	}
	bridgeEqual(t, "model rejected authorization changed state", b.model, next)
	mon, violation := inspect(b.model, next, deniedEvent, b.monitor)
	if violation != "" {
		t.Fatal(violation)
	}
	b.monitor = mon
	b.trace = append(b.trace, deniedEvent)
	before := b.state("worker", 1)
	auth, err := b.store.AuthorizeIntent(t.Context(), rid)
	if err != nil || auth.Allowed || auth.Reason != "source_closed" {
		t.Fatalf("new source authority after closure %+v %v", auth, err)
	}
	b.actions = append(b.actions, map[string]any{"action": "closure-before-authorization", "model_event": deniedEvent, "model_rejection": why, "runtime_authorization": auth})
	b.dispatchExternal(rid, true)
	bridgeEqual(t, "denial changed frozen run", before, b.state("worker", 1))
	snapshot := b.inspection()
	if len(snapshot.Tables["external_receipts"]) != 0 || contacts.Load() != 0 {
		t.Fatal("suppressed work contacted endpoint or fabricated receipt")
	}
	if len(snapshot.Tables["settlements"]) != 1 {
		t.Fatal("missing explicit source-denial settlement")
	}
	outcome := bridgeDecode[ex.ExternalOutcome](t, snapshot.Tables["settlements"][0]["descriptor"])
	if outcome.Disposition != "source_denied" || outcome.Attempt.Allowed || outcome.Receipt != nil {
		t.Fatalf("invalid suppressed outcome %+v", outcome)
	}
	b.reopen()
	b.check("closed state and suppression survive reopen")
	b.checkpoints = append(b.checkpoints, map[string]any{"label": "suppression mapping", "model_suppressed": true, "runtime_outcome": outcome, "runtime": b.inspection(), "endpoint_contacts": contacts.Load(), "relation": "model rejects authorization and retains suppressed bit; runtime retains denied source authorization plus explicit denial settlement; ledger shapes are not equal"})
	b.emitRecord("Uses unchanged J seed prefix 1-12 followed by original Fault and a rejected AuthorizeAttempt. The rejected event is explicitly recorded with reason closed. Model suppression maps to runtime source_closed/source_denied; runtime retains denial evidence and an unconsumed closed-run observation, whereas model has only suppression. Neither contacts endpoint. No receipt is fabricated.")
}
