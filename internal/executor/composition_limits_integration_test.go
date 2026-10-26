//go:build integration

package executor

import (
	"context"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	recordschema "github.com/JakeFAU/alan/schema"
	"reflect"
	"testing"
	"time"
)

func scopedAdapters(t *testing.T, maxEffects int64, await bool) (*Store, map[string]string) {
	t.Helper()
	p := Provision{}
	for _, name := range []string{"a", "b"} {
		addAdapterProgram(t, &p, name, await)
	}
	c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"a", 1}, Members: []RunID{{"a", 1}, {"b", 1}}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: maxEffects, MaxInFlight: 1}}}}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/effects.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ids := map[string]string{}
	for _, name := range []string{"a", "b"} {
		r, err := s.Commit(t.Context(), current(t, s, name).Authority(), "")
		if err != nil {
			t.Fatal(err)
		}
		ids[name] = r.Projection.Data["pending"].Text
	}
	return s, ids
}
func TestCompositionReservationDebitsOnceAndDenies(t *testing.T) {
	s, ids := scopedAdapters(t, 1, true)
	registry := map[string]effects.Adapter{"model": callbackAdapter{}}
	w, err := s.prepareAdapterWork(t.Context(), ids["a"], registry, time.Second)
	if err != nil || w == nil {
		t.Fatal(w, err)
	}
	if again, err := s.prepareAdapterWork(t.Context(), ids["a"], registry, time.Second); err != nil || again != nil {
		t.Fatal(again, err)
	}
	denied, err := s.prepareAdapterWork(t.Context(), ids["b"], registry, time.Second)
	if err != nil || denied != nil {
		t.Fatal("ceiling must retain denial", denied, err)
	}
	attempt, err := loadAdapterAttempt(t.Context(), s.conn, ids["b"])
	if err != nil || attempt.Allowed || attempt.Reason != "effect_limit" {
		t.Fatal(attempt, err)
	}
	if count(t, s, "composition_charges") != 1 || count(t, s, "adapter_results") != 1 {
		t.Fatal("debit or denial missing")
	}
}
func TestCompositionReservationClosureAfterSourceAuthorization(t *testing.T) {
	s, ids := scopedAdapters(t, 4, false)
	auth, err := s.AuthorizeIntent(t.Context(), ids["b"])
	if err != nil || !auth.Allowed {
		t.Fatal(auth, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "a").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	w, err := s.prepareAdapterWork(t.Context(), ids["b"], map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || w != nil {
		t.Fatal("closed root launched", w, err)
	}
	attempt, err := loadAdapterAttempt(t.Context(), s.conn, ids["b"])
	if err != nil || attempt.Allowed || attempt.Reason != "scope_closed" {
		t.Fatal(attempt, err)
	}
	retained, err := loadAuthorization(t.Context(), s.conn, ids["b"])
	if err != nil || !retained.Allowed {
		t.Fatal(retained, err)
	}
	if count(t, s, "composition_charges") != 0 || count(t, s, "adapter_results") != 1 {
		t.Fatal("closure charged or lost denial")
	}
}

func TestCompositionPhysicalCapacitySkipsSaturatedRoot(t *testing.T) {
	p := Provision{}
	for _, name := range []string{"a", "b", "c"} {
		addAdapterProgram(t, &p, name, true)
	}
	roots := []compositionRoot{}
	for _, names := range [][]string{{"a", "b"}, {"c"}} {
		r := compositionRoot{Run: RunID{names[0], 1}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 8, MaxInFlight: 1}}
		for _, name := range names {
			r.Members = append(r.Members, RunID{name, 1})
		}
		roots = append(roots, r)
	}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/capacity.sqlite", p, compositionProvision{Version: compositionProtocolVersion, Roots: roots}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{"a", "b", "c"} {
		if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	held := make(chan struct{})
	release := make(chan struct{})
	independent := make(chan struct{})
	forbidden := make(chan struct{}, 1)
	adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		switch p.Request.ID {
		case identity("a", 1, 1, 0):
			close(held)
			select {
			case <-release:
			case <-ctx.Done():
			}
		case identity("b", 1, 1, 0):
			forbidden <- struct{}{}
		case identity("c", 1, 1, 0):
			close(independent)
		}
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text("ok")}}
	}}
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(t.Context(), RunOptions{Adapters: map[string]effects.Adapter{"model": adapter}, MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 2, ExternalTimeout: 10 * time.Second})
		done <- err
	}()
	<-held
	select {
	case <-independent:
	case err := <-done:
		t.Fatal("independent root did not run", err)
	}
	select {
	case <-forbidden:
		t.Error("same root exceeded persisted physical ceiling")
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCompositionUnknownIndependentAndLateEvidenceReconstruction(t *testing.T) {
	s, ids := scopedAdapters(t, 4, true)
	registry := map[string]effects.Adapter{"model": callbackAdapter{}}
	w, err := s.prepareAdapterWork(t.Context(), ids["a"], registry, time.Second)
	if err != nil || w == nil {
		t.Fatal(w, err)
	}
	unknown := adapterCompletion{attempt: w.attempt, candidate: effects.Candidate{ID: ids["a"], Key: w.attempt.Prepared.Request.Key, Disposition: "unknown"}}
	if err = s.accountAdapter(t.Context(), unknown); err != nil {
		t.Fatal(err)
	}
	report, err := s.Run(t.Context(), RunOptions{Adapters: registry, MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil || len(report.UnknownAdapters) != 1 || report.UnknownAdapters[0] != ids["a"] {
		t.Fatal(report, err)
	}
	if current(t, s, "b").Projection.Status != "succeeded" {
		t.Fatal("independent member did not finish")
	}
	root, err := compositionRootFor(t.Context(), s.conn, RunID{"a", 1})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareRoot(t.Context(), root.ID)
	if err != nil || prepared.Ready {
		t.Fatal(prepared, err)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal("unknown replay", err)
	}
	if err = s.accountAdapter(t.Context(), transportAdapter(t.Context(), *w)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(t.Context(), RunOptions{Adapters: registry, MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal("late replay", err)
	}
}
func TestCompositionReservationAndResultOrdering(t *testing.T) {
	for _, kind := range []compositionFactKind{compositionReservationFact, compositionExternalResultFact} {
		t.Run(string(kind), func(t *testing.T) {
			s, ids := scopedAdapters(t, 2, true)
			w, err := s.prepareAdapterWork(t.Context(), ids["a"], map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
			if err != nil || w == nil {
				t.Fatal(w, err)
			}
			if err = s.accountAdapter(t.Context(), transportAdapter(t.Context(), *w)); err != nil {
				t.Fatal(err)
			}
			if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
				t.Fatal("baseline", err)
			}
			facts, err := readCompositionFacts(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			index := -1
			for i, f := range facts {
				if f.Kind == kind {
					index = i
					break
				}
			}
			if index < 1 {
				t.Fatal("missing target fact")
			}
			changed := append([]compositionFact{facts[index]}, append(append([]compositionFact{}, facts[:index]...), facts[index+1:]...)...)
			if _, err = s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			if _, err = s.conn.ExecContext(t.Context(), "DELETE FROM composition_facts"); err != nil {
				t.Fatal(err)
			}
			for i, f := range changed {
				f.Order = int64(i + 1)
				if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_facts VALUES(?,?,?,?)", f.Order, f.ID, f.Kind, compositionJSON(f)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=ON"); err != nil {
				t.Fatal(err)
			}
			if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
				t.Fatal("accepted reordered external evidence")
			}
		})
	}
}

func TestCompositionNestedCapacityOneExecutesLeafTransport(t *testing.T) {
	p, c := nestedInvocationFixture(t, false)
	artifactProgram := func(id string) Program {
		for _, a := range p.Programs {
			if a.ID == id {
				var program Program
				if err := decode(a.Bytes, &program); err != nil {
					t.Fatal(err)
				}
				return program
			}
		}
		t.Fatal("missing program")
		return Program{}
	}
	retainProgram := func(program Program) string {
		raw, id, err := EncodeProgram(program)
		if err != nil {
			t.Fatal(err)
		}
		p.Programs = append(p.Programs, Artifact{ID: id, Semantics: program.Semantics, Bytes: raw})
		return id
	}
	leaf, outer := c.Templates[0], c.Templates[1]
	entry := artifactProgram(leaf.Members["entry"].ProgramID)
	input, _ := (recordschema.Record{"input": {Kind: "string", MaxBytes: 10}}).Retain()
	output, _ := (recordschema.Record{"answer": {Kind: "string", MaxBytes: 20}}).Retain()
	result, _ := recordschema.ResultRecord(output.Record)
	received, _ := result.Retain()
	entry.Author.Effects = map[string]EffectDeclaration{"call": {Capability: "model", Input: input, Output: output}}
	call := Instruction{Op: "emit", Dst: "external", Effect: &Effect{Kind: "adapter", Target: Lit(Text("model")), Generation: Lit(Int(1)), Obligation: Lit(Text("call")), Payload: map[string]Operand{"input": Lit(Text("question"))}}}
	await := Instruction{Op: "receive", Match: &Match{Kind: "adapter", Provenance: "runtime-adapter", Source: Lit(Text("model")), Generation: Lit(Int(1)), Obligation: Lit(Text("call")), Operation: Ref("external")}, Bind: map[string]string{"answer": "answer"}}
	entry.Code = append(entry.Code[:1], append([]Instruction{call, await}, entry.Code[1:]...)...)
	entry.Author.Positions = make([]string, len(entry.Code))
	entry.Author.Receives[2] = received
	leaf.ID = ""
	leaf.Capabilities = []string{"model"}
	m := leaf.Members["entry"]
	m.ProgramID = retainProgram(entry)
	leaf.Members["entry"] = m
	var err error
	leaf, err = RetainTemplate(leaf, p.Programs)
	if err != nil {
		t.Fatal(err)
	}
	outerEntry := artifactProgram(outer.Members["entry"].ProgramID)
	d := outerEntry.Author.Invocations["nested"]
	d.TemplateID = leaf.ID
	outerEntry.Author.Invocations["nested"] = d
	outerEntry.Code[1].Effect.Target = Lit(Text(leaf.ID))
	outer.ID = ""
	outer.Templates = []string{leaf.ID}
	m = outer.Members["entry"]
	m.ProgramID = retainProgram(outerEntry)
	outer.Members["entry"] = m
	outer, err = RetainTemplate(outer, p.Programs)
	if err != nil {
		t.Fatal(err)
	}
	root := artifactProgram(p.Runs[0].ProgramID)
	d = root.Author.Invocations["child"]
	d.TemplateID = outer.ID
	root.Author.Invocations["child"] = d
	for i := range 2 {
		root.Code[i].Effect.Target = Lit(Text(outer.ID))
	}
	p.Runs[0].ProgramID = retainProgram(root)
	awaitingInvocationRoot(t, &p)
	c.Templates = []compositionTemplate{leaf, outer}
	c.Roots[0].Templates = []string{leaf.ID, outer.ID}
	c.Roots[0].Capabilities = []string{"model"}
	addProgram(t, &p, "independent", []Instruction{{Op: "set", Dst: "proof", A: Lit(Int(42))}, {Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	c.Roots[0].Members = append(c.Roots[0].Members, RunID{"independent", 1})
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/nested-capacity.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	waiting := make(chan struct{}, 1)
	adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text("ok")}}
	}}
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, event boundary.Event) error {
		if event.Boundary == "runner.waiting" {
			select {
			case waiting <- struct{}{}:
			default:
			}
		}
		return nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, RunOptions{Adapters: map[string]effects.Adapter{"model": adapter}, MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, ExternalTimeout: 10 * time.Second})
		done <- err
	}()
	select {
	case <-started:
	case err := <-done:
		t.Fatal("nested leaf transport never executed", err)
	}
	<-waiting
	independent := current(t, s, "independent")
	if independent.Projection.Status != "succeeded" || independent.Projection.Data["proof"].Number != 42 {
		t.Error("pure computation blocked by held transport")
	}
	if current(t, s, "root").Projection.Status != "open" {
		t.Error("root completed before nested leaf")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if current(t, s, "root").Projection.Data["result"].Text != "ok" {
		t.Fatal("nested result missing")
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestCompositionReservedAdapterMayLaunchAfterClosure(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "root", true)
	addInvocationController(t, &p, "CancelRun")
	c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"root", 1}, Members: []RunID{{"root", 1}}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/late-launch.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	next, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	launches := 0
	adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		launches++
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]Value{"answer": Text("ok")}}
	}}
	work, err := s.prepareAdapterWork(t.Context(), next.Projection.Data["pending"].Text, map[string]effects.Adapter{"model": adapter}, time.Second)
	if err != nil || work == nil {
		t.Fatal(work, err)
	}
	if result := authorizedDecision(t, s, committedRequest(t, s, "controller")); !result.Accepted {
		t.Fatal(result)
	}
	root, err := compositionRootFor(t.Context(), s.conn, RunID{"root", 1})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.SettleRoot(t.Context(), root.ID)
	if err != nil || !before.Ready || before.Outcome.Status != "cancelled" {
		t.Fatal(before, err)
	}
	if err = s.accountAdapter(t.Context(), transportAdapter(t.Context(), *work)); err != nil {
		t.Fatal(err)
	}
	after, err := s.PrepareRoot(t.Context(), root.ID)
	if err != nil || !reflect.DeepEqual(before.Outcome, after.Outcome) || launches != 1 {
		t.Fatal(before, after, launches, err)
	}
	if count(t, s, "composition_charges") != 1 || count(t, s, "adapter_results") != 1 {
		t.Fatal("lost late result or changed debit")
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
func TestCompositionEffectCeilingSharedAcrossAdapterAndLegacy(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "root", true)
	addExternalClient(t, &p, "peer", "http://127.0.0.1:1", "non-idempotent", 5)
	c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"root", 1}, Members: []RunID{{"root", 1}, {"peer", 1}}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/shared.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{"root", "peer"} {
		if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if w, err := s.prepareAdapterWork(t.Context(), identity("root", 1, 1, 0), map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second); err != nil || w == nil {
		t.Fatal(w, err)
	}
	denied, _, err := s.prepareExternal(t.Context(), identity("peer", 1, 1, 0))
	if err != nil || denied.Allowed || denied.Reason != "effect_limit" {
		t.Fatal(denied, err)
	}
	if count(t, s, "composition_charges") != 1 {
		t.Fatal("external category reset root budget")
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestCompositionUnknownReopenKeepsOriginalAttemptAndBudget(t *testing.T) {
	p := Provision{}
	for _, name := range []string{"root", "peer"} {
		addAdapterProgram(t, &p, name, true)
	}
	c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"root", 1}, Members: []RunID{{"root", 1}, {"peer", 1}}, Capabilities: []string{"model"}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	p.Composition = &c
	path := t.TempDir() + "/unknown-reopen.sqlite"
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"root", "peer"} {
		if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	before := current(t, s, "root")
	operation := before.Projection.Data["pending"].Text
	work, err := s.prepareAdapterWork(t.Context(), operation, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || work == nil {
		t.Fatal(work, err)
	}
	if err = s.accountAdapter(t.Context(), adapterCompletion{attempt: work.attempt, candidate: effects.Candidate{Disposition: "unknown"}}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	after := current(t, s, "root")
	if after.Epoch != before.Epoch+1 || !reflect.DeepEqual(after.Projection, before.Projection) {
		t.Fatal(before, after)
	}
	calls := 0
	adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		calls++
		return effects.Candidate{Disposition: "unknown"}
	}}
	report, err := s.Run(t.Context(), RunOptions{Adapters: map[string]effects.Adapter{"model": adapter}, MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 9})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !reflect.DeepEqual(report.UnknownAdapters, []string{operation}) || len(report.Roots) != 1 || report.Roots[0].Charges.Effects != 1 || report.Roots[0].Limits.MaxInFlight != 1 {
		t.Fatal("recovery repeated uncertainty or raised limits", calls, report)
	}
	denied, err := loadAdapterAttempt(t.Context(), s.conn, identity("peer", 1, 1, 0))
	if err != nil || denied.Allowed || denied.Reason != "effect_limit" {
		t.Fatal(denied, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
