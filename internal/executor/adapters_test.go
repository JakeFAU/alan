//go:build integration

package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	recordschema "github.com/JakeFAU/alan/schema"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type callbackAdapter struct {
	prepare func(effects.Request) (effects.Prepared, error)
	execute func(context.Context, effects.Prepared) effects.Candidate
}

func (a callbackAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	if a.prepare != nil {
		return a.prepare(r)
	}
	return effects.Prepared{Request: r, Protocol: "test/v1", MaxResponseBytes: 4096}, nil
}
func (a callbackAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	if a.execute != nil {
		return a.execute(ctx, p)
	}
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]recordschema.Value{"answer": Text("ok")}}
}
func addAdapterProgram(t *testing.T, p *Provision, name string, await bool) {
	t.Helper()
	empty, _ := recordschema.Record{}.Retain()
	input, _ := (recordschema.Record{"input": {Kind: "string", MaxBytes: 10}}).Retain()
	output, _ := (recordschema.Record{"answer": {Kind: "string", MaxBytes: 20}}).Retain()
	result, _ := recordschema.ResultRecord(output.Record)
	receive, _ := result.Retain()
	code := []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "adapter", Target: Lit(Text("model")), Generation: Lit(Int(1)), Obligation: Lit(Text("call")), Payload: map[string]Operand{"input": Lit(Text("question"))}}}}
	metadata := &ProgramMetadata{Compiler: AuthorCompilerVersion, Inputs: empty, Outputs: empty, Effects: map[string]EffectDeclaration{"call": {Capability: "model", Input: input, Output: output}}, Receives: map[int]recordschema.Retained{}}
	if await {
		code = append(code, Instruction{Op: "receive", Match: &Match{Kind: "adapter", Provenance: "runtime-adapter", Source: Lit(Text("model")), Generation: Lit(Int(1)), Obligation: Lit(Text("call")), Operation: Ref("pending")}, Bind: map[string]string{"answer": "answer", "valid": "valid"}})
		metadata.Receives[1] = receive
	}
	code = append(code, Instruction{Op: "end", Outcome: "succeeded"})
	metadata.Positions = make([]string, len(code))
	raw, id, e := EncodeProgram(Program{Semantics: AuthorSemanticVersion, Code: code, Author: metadata})
	if e != nil {
		t.Fatal(e)
	}
	p.Programs = append(p.Programs, Artifact{ID: id, Semantics: AuthorSemanticVersion, Bytes: raw})
	p.Subjects = append(p.Subjects, Subject{ID: name, Generation: 1, Revision: 1, Owner: "owner:" + name})
	p.Runs = append(p.Runs, InitialRun{ID: RunID{name, 1}, ProgramID: id, Projection: Projection{Status: "open", Data: map[string]Value{}}})
	p.Permissions = append(p.Permissions, Permission{Kind: "adapter", Source: name, Target: "model", Operation: "call", Revision: 1, Obligation: "call"})
}
func committedAdapter(t *testing.T) (*Store, string, string) {
	t.Helper()
	p := Provision{}
	addAdapterProgram(t, &p, "agent", true)
	s, path := messageStore(t, p)
	next, e := s.Commit(t.Context(), current(t, s, "agent").Authority(), "")
	if e != nil {
		t.Fatal(e)
	}
	return s, path, next.Projection.Data["pending"].Text
}
func TestAdapterCommittedGrantCorrelationAndConflict(t *testing.T) {
	s, _, id := committedAdapter(t)
	registry := map[string]effects.Adapter{"model": callbackAdapter{}}
	if _, e := s.prepareAdapterWork(t.Context(), "uncommitted", registry, time.Second); e == nil {
		t.Fatal("uncommitted work accepted")
	}
	w, e := s.prepareAdapterWork(t.Context(), id, registry, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	c := transportAdapter(t.Context(), *w)
	wrong := effects.Copy(c.candidate)
	wrong.Key = "different"
	if _, e = s.recordAdapterResult(t.Context(), w.attempt, wrong); !errors.Is(e, ErrConflict) {
		t.Fatal("mismatched result accepted", e)
	}
	if count(t, s, "adapter_results") != 0 {
		t.Fatal("mismatched candidate retained")
	}
	r, e := s.recordAdapterResult(t.Context(), w.attempt, c.candidate)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.recordAdapterResult(t.Context(), w.attempt, c.candidate); e != nil {
		t.Fatal("same result conflict", e)
	}
	changed := effects.Copy(c.candidate)
	changed.Payload["answer"] = Text("conflict")
	if _, e = s.recordAdapterResult(t.Context(), w.attempt, changed); !errors.Is(e, ErrConflict) {
		t.Fatal("conflicting result accepted", e)
	}
	if e = s.settleAdapter(t.Context(), w.attempt, r); e != nil {
		t.Fatal(e)
	}
}
func TestAdapterStaticGrantAndInputBeforeLaunch(t *testing.T) {
	for _, kind := range []string{"grant", "input", "capability"} {
		t.Run(kind, func(t *testing.T) {
			p := Provision{}
			addAdapterProgram(t, &p, "agent", true)
			if kind == "grant" {
				p.Permissions = nil
			} else {
				var program Program
				json.Unmarshal(p.Programs[0].Bytes, &program)
				if kind == "input" {
					program.Code[0].Effect.Payload["input"] = Lit(Text("exceeds ten bytes"))
				} else {
					program.Code[0].Effect.Target = Lit(Text("shell"))
				}
				raw, id, e := EncodeProgram(program)
				if e != nil {
					t.Fatal(e)
				}
				p.Programs[0].ID = id
				p.Programs[0].Bytes = raw
				p.Runs[0].ProgramID = id
			}
			s, _ := messageStore(t, p)
			var contacts atomic.Int64
			a := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
				contacts.Add(1)
				return callbackAdapter{}.Execute(ctx, p)
			}}
			r, e := s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a, "shell": a}})
			if contacts.Load() != 0 {
				t.Fatal("unauthorized effect executed")
			}
			if kind == "grant" {
				if e != nil || len(r.UnresolvedIntents) != 0 {
					t.Fatal(r, e)
				}
				if current(t, s, "agent").Projection.Data["valid"].Flag {
					t.Fatal("denial became valid")
				}
			} else if e == nil || count(t, s, "adapter_attempts") != 0 {
				t.Fatal("invalid request spent attempt", e)
			}
		})
	}
}
func TestAdapterWorkerCopyAndLateEvidenceAfterEpoch(t *testing.T) {
	s, path, id := committedAdapter(t)
	w, e := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		p.Request.Payload["input"] = Text("mutated")
		p.Request.Output.Record["answer"] = recordschema.Field{Kind: "bool"}
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]recordschema.Value{"answer": Text("ok")}}
	}}}, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	c := transportAdapter(t.Context(), *w)
	s.Close()
	s, e = Open(t.Context(), path, DefaultLimits())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.accountAdapter(t.Context(), c); e != nil {
		t.Fatal(e)
	}
	a, e := loadAdapterAttempt(t.Context(), s.conn, id)
	if e != nil {
		t.Fatal(e)
	}
	if a.Prepared.Request.Payload["input"].Text != "question" || a.Prepared.Request.Output.Record["answer"].Kind != "string" {
		t.Fatal("worker mutated retained request")
	}
	if _, e = s.Run(t.Context(), RunOptions{MaxSteps: 20, MaxDispatch: 20}); e != nil {
		t.Fatal(e)
	}
	if current(t, s, "agent").Projection.Status != "succeeded" {
		t.Fatal("late evidence not consumed")
	}
}
func TestAdapterMixedTransportCapacityAndPeerProgress(t *testing.T) {
	for _, capacity := range []int{1, 2} {
		t.Run(string(rune('0'+capacity)), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			held := make(chan struct{})
			release := make(chan struct{})
			var released atomic.Bool
			unblock := func() {
				if released.CompareAndSwap(false, true) {
					close(release)
				}
			}
			defer unblock()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req resourcefixture.Request
				json.NewDecoder(r.Body).Decode(&req)
				close(held)
				select {
				case <-release:
					json.NewEncoder(w).Encode(resourcefixture.Receipt{Key: req.Key, Profile: req.Profile, Delta: req.Delta, Total: req.Delta, Mutation: 1, Process: "controlled", PID: 1})
				case <-ctx.Done():
				}
			}))
			defer server.Close()
			p := externalProvision(t, server.URL, "idempotent")
			addAdapterProgram(t, &p, "modelworker", true)
			addProgram(t, &p, "zzpeer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
			s, _ := messageStore(t, p)
			var calls atomic.Int64
			adapter := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
				calls.Add(1)
				return callbackAdapter{}.Execute(ctx, p)
			}}
			peer := false
			ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
				if e.Subject == "zzpeer" && e.Boundary == "execution.committed" {
					select {
					case <-held:
					case <-ctx.Done():
						return ctx.Err()
					}
					peer = true
					if capacity == 1 {
						if calls.Load() != 0 || count(t, s, "adapter_attempts") != 0 {
							unblock()
							return errors.New("adapter bypassed shared capacity")
						}
						unblock()
					}
				}
				if e.Boundary == "adapter.settled" && capacity == 2 {
					if released.Load() {
						return errors.New("fast adapter waited for slow fixture")
					}
					unblock()
				}
				return nil
			})
			r, e := s.Run(ctx, RunOptions{MaxSteps: 40, MaxDispatch: 40, MaxInFlight: capacity, ExternalTimeout: 4 * time.Second, Adapters: map[string]effects.Adapter{"model": adapter}})
			unblock()
			if e != nil || !peer || len(r.OpenRuns) != 0 || calls.Load() != 1 {
				t.Fatalf("mixed capacity report %+v %v peer=%v calls=%d", r, e, peer, calls.Load())
			}
		})
	}
}
func TestAdapterValidLateResultDoesNotReopenClosedSource(t *testing.T) {
	p := Provision{}
	addAdapterProgram(t, &p, "agent", false)
	s, path := messageStore(t, p)
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	a := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return callbackAdapter{}.Execute(ctx, p)
	}}
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if e.Boundary == "execution.committed" && e.Subject == "agent" && e.Sequence == 2 {
			select {
			case <-started:
			case <-ctx.Done():
				return ctx.Err()
			}
			close(release)
		}
		return nil
	})
	r, e := s.Run(ctx, RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if e != nil || len(r.OpenRuns) != 0 || count(t, s, "adapter_results") != 1 || count(t, s, "observations") != 1 {
		t.Fatal(r, e)
	}
	if current(t, s, "agent").Projection.Status != "succeeded" {
		t.Fatal("source reopened")
	}
	if _, e = Replay(t.Context(), path, DefaultLimits()); e != nil {
		t.Fatal(e)
	}
}
func TestAdapterCancellationAccountsReceivedCandidate(t *testing.T) {
	s, _, _ := committedAdapter(t)
	ctx, cancel := context.WithCancel(t.Context())
	a := callbackAdapter{execute: func(ctx context.Context, p effects.Prepared) effects.Candidate {
		cancel()
		<-ctx.Done()
		return callbackAdapter{}.Execute(ctx, p)
	}}
	_, e := s.Run(ctx, RunOptions{MaxSteps: 20, MaxDispatch: 20, Adapters: map[string]effects.Adapter{"model": a}})
	if !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if count(t, s, "adapter_results") != 1 || count(t, s, "settlements") != 1 {
		t.Fatal("candidate lost during cancellation")
	}
}
func TestDeniedAttemptWithoutResultRecoversDeterministically(t *testing.T) {
	s, _, id := committedAdapter(t)
	w, e := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	a := w.attempt
	a.Allowed = false
	a.Reason = "grant_denied"
	raw, _ := json.Marshal(a)
	if _, e = s.conn.ExecContext(t.Context(), "DELETE FROM permissions WHERE kind='adapter'"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.conn.ExecContext(t.Context(), "UPDATE adapter_attempts SET descriptor=? WHERE intent_id=?", raw, id); e != nil {
		t.Fatal(e)
	}
	_, e = s.prepareAdapterWork(t.Context(), id, nil, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if count(t, s, "adapter_results") != 1 || count(t, s, "settlements") != 1 {
		t.Fatal("known denied attempt stranded without result")
	}
}
func TestSchemaV2RejectedBeforeEpochRenewal(t *testing.T) {
	s, path, _ := committedAdapter(t)
	before := current(t, s, "agent").Epoch
	if _, e := s.conn.ExecContext(t.Context(), "UPDATE metadata SET schema_version=2; PRAGMA user_version=2"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if reopened, e := Open(t.Context(), path, DefaultLimits()); !errors.Is(e, ErrVersion) {
		if reopened != nil {
			reopened.Close()
		}
		t.Fatal("v2 accepted", e)
	}
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var after int64
	if e = db.QueryRow("SELECT epoch FROM runs").Scan(&after); e != nil {
		t.Fatal(e)
	}
	if after != before {
		t.Fatal("rejected layout renewed epoch")
	}
}
func TestAdapterResultEvidenceMutationRejectedOffline(t *testing.T) {
	s, path, id := committedAdapter(t)
	w, e := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	c := transportAdapter(t.Context(), *w)
	r, e := s.recordAdapterResult(t.Context(), w.attempt, c.candidate)
	if e != nil {
		t.Fatal(e)
	}
	r.Candidate.Response = "different evidence"
	raw, _ := json.Marshal(r)
	if _, e = s.conn.ExecContext(t.Context(), "UPDATE adapter_results SET descriptor=? WHERE intent_id=?", raw, id); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if _, e = Inspect(t.Context(), path, DefaultLimits()); e == nil {
		t.Fatal("modified result evidence accepted")
	}
	if _, e = Replay(t.Context(), path, DefaultLimits()); e == nil {
		t.Fatal("modified result replayed")
	}
	if reopened, e := Open(t.Context(), path, DefaultLimits()); e == nil {
		reopened.Close()
		t.Fatal("modified result activated")
	}
}
func TestAdapterPrepareCannotChangeCommittedRequest(t *testing.T) {
	s, _, id := committedAdapter(t)
	a := callbackAdapter{prepare: func(r effects.Request) (effects.Prepared, error) {
		r.Payload["input"] = Text("altered")
		return effects.Prepared{Request: r, Protocol: "test", MaxResponseBytes: 4096}, nil
	}}
	if _, e := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": a}, time.Second); e == nil {
		t.Fatal("request reinterpreted")
	}
	if count(t, s, "adapter_attempts") != 0 || count(t, s, "attempts") != 0 {
		t.Fatal("preparation mismatch spent attempt")
	}
}
