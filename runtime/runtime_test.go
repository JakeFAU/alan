//go:build integration

package runtime_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type schemaTestAdapter struct {
	fn func(effects.Request) map[string]schema.Value
}

func (a schemaTestAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "test/v1", MaxResponseBytes: 4096}, nil
}

func (a schemaTestAdapter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: a.fn(p.Request)}
}

func TestRuntimeRejectsInvalidAuthorIngress(t *testing.T) {
	ctx := context.Background()
	record := schema.Record{"question": {Kind: "string", MaxBytes: 10, Enum: []string{"why"}}}
	a, err := author.Compile(author.Definition{Ports: map[string]author.Port{"input": {Peer: "local", Local: true, Obligation: "question", Receive: record}}, Steps: []author.Step{author.Receive("input", map[string]string{"question": "question"}), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"worker": {Artifact: a}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "input.db")
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, tt := range []struct {
		name    string
		payload map[string]schema.Value
	}{
		{"type", map[string]schema.Value{"question": schema.Int(2)}}, {"enum", map[string]schema.Value{"question": schema.Text("no")}}, {"extra", map[string]schema.Value{"question": schema.Text("why"), "extra": schema.Bool(true)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, err := s.AdmitInput(ctx, runtime.Input{TransportID: tt.name, LogicalID: tt.name, Target: "worker", Obligation: "question", Payload: tt.payload})
			if err != nil {
				t.Fatal(err)
			}
			if r.Accepted || r.Reason != "schema_invalid" {
				t.Fatalf("invalid input admitted: %+v", r)
			}
		})
	}
	r, err := s.AdmitInput(ctx, runtime.Input{TransportID: "good", LogicalID: "good", Target: "worker", Obligation: "question", Payload: map[string]schema.Value{"question": schema.Text("why")}})
	if err != nil || !r.Accepted {
		t.Fatalf("valid input: %+v %v", r, err)
	}
	if _, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Runs) != 1 || replay.Runs[0].Projection.Status != "succeeded" {
		t.Fatalf("replay: %+v", replay)
	}
}

func TestRuntimeValidatesInitialAndFinalSchemas(t *testing.T) {
	ctx := context.Background()
	a, err := author.Compile(author.Definition{Inputs: schema.Record{"text": {Kind: "string", MaxBytes: 10}}, Outputs: schema.Record{"answer": {Kind: "string", MaxBytes: 2}}, Steps: []author.Step{author.Set("answer", author.Var("text")), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"worker": {Artifact: a, Inputs: map[string]schema.Value{"text": schema.Text("long")}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Create(ctx, filepath.Join(t.TempDir(), "output.db"), p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20}); err == nil {
		t.Fatal("oversized success output accepted")
	}
	p.Runs[0].Projection.Data["text"] = schema.Int(2)
	if bad, err := runtime.Create(ctx, filepath.Join(t.TempDir(), "initial.db"), p, runtime.DefaultLimits()); err == nil {
		bad.Close()
		t.Fatal("raw provision bypassed initial schema")
	}
}

func TestRuntimeAuthoredControlPaths(t *testing.T) {
	for _, tt := range []struct {
		name   string
		choose bool
		want   int64
		status string
	}{{"then", true, 2, "succeeded"}, {"else", false, 9, "failed"}} {
		t.Run(tt.name, func(t *testing.T) {
			a, err := author.Compile(author.Definition{Values: map[string]schema.Value{"count": schema.Int(0)}, Steps: []author.Step{
				author.Repeat(0, author.Add("count", author.Var("count"), author.Integer(100))),
				author.Repeat(5, author.Add("count", author.Var("count"), author.Integer(1)), author.IfEqual(author.Var("count"), author.Integer(2), []author.Step{author.Goto("done")}, nil)),
				author.Label("done"), author.IfEqual(author.Boolean(tt.choose), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Set("count", author.Integer(9)), author.Failure()}),
			}})
			if err != nil {
				t.Fatal(err)
			}
			p, err := (author.Application{"worker": {Artifact: a}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "control.db")
			ctx := context.Background()
			s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
				s.Close()
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			got := replay.Runs[0].Projection
			if got.Status != tt.status || got.Data["count"].Number != tt.want {
				t.Fatalf("got %+v, want %s count=%d", got, tt.status, tt.want)
			}
		})
	}
}

func TestRuntimeConditionalLexicalContinuations(t *testing.T) {
	tests := []struct {
		name   string
		steps  []author.Step
		status string
	}{
		{"then forward goto to trailing label", []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(true), []author.Step{author.Goto("join"), author.Label("join")}, []author.Step{author.Failure()}), author.Success()}, "succeeded"},
		{"nested trailing labels", []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(true), []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(true), []author.Step{author.Goto("inner"), author.Label("inner")}, []author.Step{author.Failure()}), author.Goto("outer"), author.Label("outer")}, []author.Step{author.Failure()}), author.Success()}, "succeeded"},
		{"repeat exit to arm boundary", []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(true), []author.Step{author.Repeat(2, author.Goto("exit")), author.Label("exit")}, []author.Step{author.Failure()}), author.Success()}, "succeeded"},
		{"terminal arms", []author.Step{author.IfEqual(author.Boolean(true), author.Boolean(true), []author.Step{author.Success()}, []author.Step{author.Failure()})}, "succeeded"},
		{"else still selected", []author.Step{author.IfEqual(author.Boolean(false), author.Boolean(true), []author.Step{author.Goto("join"), author.Label("join")}, []author.Step{author.Failure()}), author.Success()}, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := author.Compile(author.Definition{Steps: tt.steps})
			if err != nil {
				t.Fatal(err)
			}
			p, err := (author.Application{"probe": {Artifact: a}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "continuation.db")
			s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100})
			if err != nil {
				s.Close()
				t.Fatal(err)
			}
			if len(report.OpenRuns) != 0 {
				t.Fatalf("Run left open runs: %+v", report)
			}
			snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				s.Close()
				t.Fatal(err)
			}
			var live runtime.Projection
			projection, ok := snapshot.Tables["runs"][0]["projection"].(string)
			if !ok {
				s.Close()
				t.Fatal("missing live projection")
			}
			if err = json.Unmarshal([]byte(projection), &live); err != nil {
				s.Close()
				t.Fatal(err)
			}
			if live.Status != tt.status {
				s.Close()
				t.Fatalf("Run status=%s want=%s", live.Status, tt.status)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if replay.Runs[0].Projection.Status != tt.status {
				t.Fatalf("Replay status=%s want=%s", replay.Runs[0].Projection.Status, tt.status)
			}
		})
	}
}

func TestRuntimePreservesDistinctUnicodeIdentifiers(t *testing.T) {
	composed, decomposed := "é", "e\u0301"
	a, err := author.Compile(author.Definition{
		Name: "定義", Inputs: schema.Record{"初期": {Kind: "bool"}}, Outputs: schema.Record{"回答": {Kind: "string", MaxBytes: 100}},
		Values: map[string]schema.Value{composed: schema.Int(1), decomposed: schema.Int(2)}, Templates: map[string]string{"説明": "保存する内容"},
		Effects: map[string]author.Effect{"推論": {Capability: "模型"}},
		Ports:   map[string]author.Port{"入口": {Peer: "local", Local: true, Obligation: "問題", Receive: schema.Record{"質問": {Kind: "string", MaxBytes: 100}}}},
		Steps:   []author.Step{author.Receive("入口", map[string]string{"質問": "回答"}), author.IfEqual(author.Var(composed), author.Integer(1), []author.Step{author.IfEqual(author.Var(decomposed), author.Integer(2), []author.Step{author.Goto("完了")}, []author.Step{author.Failure()})}, []author.Step{author.Failure()}), author.Label("完了"), author.Success()},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{"工人": {Artifact: a, Inputs: map[string]schema.Value{"初期": schema.Bool(true)}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "unicode.db")
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admitted, err := s.AdmitInput(ctx, runtime.Input{TransportID: "入力", LogicalID: "質問一", Target: "工人", Obligation: "問題", Payload: map[string]schema.Value{"質問": schema.Text("保持された回答")}})
	if err != nil || !admitted.Accepted {
		t.Fatalf("Unicode input %+v: %v", admitted, err)
	}
	if _, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	run := replay.Runs[0]
	if run.ID.Subject != "工人" || run.Projection.Status != "succeeded" || run.Projection.Data[composed].Number != 1 || run.Projection.Data[decomposed].Number != 2 || run.Projection.Data["回答"].Text != "保持された回答" {
		t.Fatalf("Unicode identity or data changed: %+v", run)
	}
	if a.Metadata().Effects["推論"].Capability != "模型" || a.Metadata().Templates["説明"] != "保存する内容" {
		t.Fatal("Unicode metadata changed")
	}
}

func TestRuntimeEnforcesAuthoredSchemasBeforeCommit(t *testing.T) {
	enumField := schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"fast", "slow"}}
	shortField := schema.Field{Kind: "string", MaxBytes: 4}

	t.Run("rejects schema-invalid adapter request before durable commit", func(t *testing.T) {
		worker, err := author.Compile(author.Definition{
			Values:  map[string]schema.Value{"marker": schema.Text("initial")},
			Ports:   map[string]author.Port{"in": {Peer: "local", Local: true, Obligation: "trigger", Receive: schema.Record{"raw": {Kind: "string", MaxBytes: 30}}}},
			Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": shortField}}},
			Steps: []author.Step{
				author.Receive("in", map[string]string{"raw": "raw"}),
				author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("raw")}, map[string]author.Expr{"marker": author.String("mutated")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		p, err := (author.Application{"worker": {Artifact: worker}}).Provision()
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "invalid-adapter.db")
		s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		admitted, err := s.AdmitInput(ctx, runtime.Input{
			TransportID: "t1", LogicalID: "l1", Target: "worker", Obligation: "trigger",
			Payload: map[string]schema.Value{"raw": schema.Text("NOT_IN_ENUM")},
		})
		if err != nil || !admitted.Accepted {
			t.Fatalf("admit input: %+v %v", admitted, err)
		}
		_, runErr := s.Run(ctx, runtime.RunOptions{
			MaxSteps: 10, MaxDispatch: 10,
			Adapters: map[string]effects.Adapter{
				"model": schemaTestAdapter{fn: func(effects.Request) map[string]schema.Value {
					return map[string]schema.Value{"res": schema.Text("ok")}
				}},
			},
		})
		if runErr == nil {
			t.Fatal("expected schema-invalid adapter request to be rejected")
		}
		snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Tables["intents"]) != 0 {
			t.Fatalf("schema-invalid adapter intent reached durable intents table: %+v", snap.Tables["intents"])
		}
		if len(snap.Tables["history"]) != 2 {
			t.Fatalf("rejected emit reduction appended to durable history: got %d rows", len(snap.Tables["history"]))
		}
		var proj runtime.Projection
		if err := json.Unmarshal([]byte(snap.Tables["runs"][0]["projection"].(string)), &proj); err != nil {
			t.Fatal(err)
		}
		if proj.Data["marker"].Text != "initial" || proj.PC != 2 || proj.Status != "open" {
			t.Fatalf("rejected emit reduction was not atomic: projection=%+v", proj)
		}
		if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
			t.Fatalf("replay after rejected reduction failed: %v", err)
		}
	})

	t.Run("rejects schema-invalid ordinary message before durable commit", func(t *testing.T) {
		sender, err := author.Compile(author.Definition{
			Values: map[string]schema.Value{"marker": schema.Text("initial")},
			Ports: map[string]author.Port{
				"in":  {Peer: "local", Local: true, Obligation: "trigger", Receive: schema.Record{"raw": {Kind: "string", MaxBytes: 30}}},
				"out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}},
			},
			Steps: []author.Step{
				author.Receive("in", map[string]string{"raw": "raw"}),
				author.Send("out", map[string]author.Expr{"mode": author.Var("raw")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		receiver, err := author.Compile(author.Definition{
			Ports: map[string]author.Port{"in": {Peer: "sender", Obligation: "work", Receive: schema.Record{"mode": enumField}}},
			Steps: []author.Step{author.Receive("in", map[string]string{"mode": "got"}), author.Success()},
		})
		if err != nil {
			t.Fatal(err)
		}
		p, err := (author.Application{
			"sender":   {Artifact: sender},
			"receiver": {Artifact: receiver},
		}).Provision()
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "invalid-message.db")
		s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		admitted, err := s.AdmitInput(ctx, runtime.Input{
			TransportID: "t1", LogicalID: "l1", Target: "sender", Obligation: "trigger",
			Payload: map[string]schema.Value{"raw": schema.Text("NOT_IN_ENUM")},
		})
		if err != nil || !admitted.Accepted {
			t.Fatalf("admit input: %+v %v", admitted, err)
		}
		if _, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 10, MaxDispatch: 10}); err == nil {
			t.Fatal("expected schema-invalid message send to be rejected")
		}
		snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Tables["intents"]) != 0 {
			t.Fatalf("schema-invalid message intent reached durable intents table: %+v", snap.Tables["intents"])
		}
		if len(snap.Tables["history"]) != 2 {
			t.Fatalf("rejected message emit appended to durable history: got %d rows", len(snap.Tables["history"]))
		}
		if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
			t.Fatalf("replay after rejected message emit failed: %v", err)
		}
	})

	t.Run("rejects schema-invalid invocation before durable commit", func(t *testing.T) {
		childArt, err := author.Compile(author.Definition{
			Ports: map[string]author.Port{"entry": {Peer: "caller", Obligation: "task", Receive: schema.Record{"mode": enumField}, Send: schema.Record{"ok": shortField}}},
			Steps: []author.Step{author.Receive("entry", map[string]string{"mode": "m"}), author.Send("entry", map[string]author.Expr{"ok": author.String("done")}), author.Success()},
		})
		if err != nil {
			t.Fatal(err)
		}
		childTpl, err := author.NewTemplate(author.Application{"worker": {Artifact: childArt, Inputs: map[string]schema.Value{}}}, "worker", "entry")
		if err != nil {
			t.Fatal(err)
		}
		caller, err := author.Compile(author.Definition{
			Values:      map[string]schema.Value{"marker": schema.Text("initial")},
			Invocations: map[string]author.Template{"sub": childTpl},
			Ports:       map[string]author.Port{"in": {Peer: "local", Local: true, Obligation: "trigger", Receive: schema.Record{"raw": {Kind: "string", MaxBytes: 30}}}},
			Steps: []author.Step{
				author.Receive("in", map[string]string{"raw": "raw"}),
				author.Invoke("inv", "sub", map[string]author.Expr{"mode": author.Var("raw")}, map[string]author.Expr{"marker": author.String("mutated")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		p, err := (author.Application{"caller": {Artifact: caller, Inputs: map[string]schema.Value{}}}).Provision()
		if err != nil {
			t.Fatal(err)
		}
		p.Composition.Roots = []runtime.ResourceRoot{{
			Run:       runtime.RunID{Subject: "caller", Generation: 1},
			Members:   []runtime.RunID{{Subject: "caller", Generation: 1}},
			Templates: []string{childTpl.ID()},
			Limits:    runtime.ResourceLimits{MaxInvocations: 4, MaxCreatedRuns: 8, MaxDepth: 2, MaxEffects: 8, MaxInFlight: 2},
		}}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "invalid-invoke.db")
		s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		admitted, err := s.AdmitInput(ctx, runtime.Input{
			TransportID: "t1", LogicalID: "l1", Target: "caller", Obligation: "trigger",
			Payload: map[string]schema.Value{"raw": schema.Text("NOT_IN_ENUM")},
		})
		if err != nil || !admitted.Accepted {
			t.Fatalf("admit input: %+v %v", admitted, err)
		}
		if _, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 10, MaxDispatch: 10}); err == nil {
			t.Fatal("expected schema-invalid invocation to be rejected")
		}
		snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Tables["intents"]) != 0 {
			t.Fatalf("schema-invalid invocation intent reached durable intents table: %+v", snap.Tables["intents"])
		}
		if len(snap.Tables["history"]) != 2 {
			t.Fatalf("rejected invocation emit appended to durable history: got %d rows", len(snap.Tables["history"]))
		}
		if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
			t.Fatalf("replay after rejected invocation emit failed: %v", err)
		}
	})

	t.Run("positive control valid runtime-derived values execute and replay", func(t *testing.T) {
		sender, err := author.Compile(author.Definition{
			Outputs: schema.Record{"final": shortField},
			Ports: map[string]author.Port{
				"in":  {Peer: "local", Local: true, Obligation: "trigger", Receive: schema.Record{"raw": {Kind: "string", MaxBytes: 30}}},
				"out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": enumField}},
			},
			Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": shortField}}},
			Steps: []author.Step{
				author.Receive("in", map[string]string{"raw": "raw"}),
				author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("raw")}, nil),
				author.Await("tool", "req", map[string]string{"valid": "ok", "res": "final"}),
				author.Send("out", map[string]author.Expr{"mode": author.Var("raw")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		receiver, err := author.Compile(author.Definition{
			Outputs: schema.Record{"got": enumField},
			Ports:   map[string]author.Port{"in": {Peer: "sender", Obligation: "work", Receive: schema.Record{"mode": enumField}}},
			Steps:   []author.Step{author.Receive("in", map[string]string{"mode": "got"}), author.Success()},
		})
		if err != nil {
			t.Fatal(err)
		}
		p, err := (author.Application{
			"sender":   {Artifact: sender},
			"receiver": {Artifact: receiver},
		}).Provision()
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "valid-positive.db")
		s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		admitted, err := s.AdmitInput(ctx, runtime.Input{
			TransportID: "t1", LogicalID: "l1", Target: "sender", Obligation: "trigger",
			Payload: map[string]schema.Value{"raw": schema.Text("fast")},
		})
		if err != nil || !admitted.Accepted {
			t.Fatalf("admit input: %+v %v", admitted, err)
		}
		report, err := s.Run(ctx, runtime.RunOptions{
			MaxSteps: 20, MaxDispatch: 20,
			Adapters: map[string]effects.Adapter{
				"model": schemaTestAdapter{fn: func(req effects.Request) map[string]schema.Value {
					if req.Payload["mode"].Text != "fast" {
						t.Fatalf("unexpected adapter payload: %+v", req.Payload)
					}
					return map[string]schema.Value{"res": schema.Text("done")}
				}},
			},
		})
		if err != nil || !report.Quiescent || len(report.OpenRuns) != 0 {
			t.Fatalf("valid flow failed: report=%+v err=%v", report, err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("replay failed: %v", err)
		}
		if len(replay.Runs) != 2 || replay.Runs[0].Projection.Status != "succeeded" || replay.Runs[1].Projection.Status != "succeeded" {
			t.Fatalf("unexpected replay runs: %+v", replay.Runs)
		}
	})
}

func TestRuntimeEnforcesUnambiguousSendContracts(t *testing.T) {
	strictField := schema.Field{Kind: "string", MaxBytes: 4, Enum: []string{"fast", "slow"}}
	wideField := schema.Field{Kind: "string", MaxBytes: 32}

	t.Run("rejects strict_out vs wide_out bypass at compile and retained metadata boundaries", func(t *testing.T) {
		_, err := author.Compile(author.Definition{
			Name:   "sender",
			Inputs: schema.Record{"dyn": wideField},
			Ports: map[string]author.Port{
				"strict_out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": strictField}},
				"wide_out":   {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": wideField}},
			},
			Steps: []author.Step{
				author.IfEqual(author.Var("dyn"), author.String("unreachable_branch"),
					[]author.Step{
						author.Send("wide_out", map[string]author.Expr{"mode": author.String("wide_literal_ok")}),
						author.Failure(),
					},
					[]author.Step{
						author.Send("strict_out", map[string]author.Expr{"mode": author.Var("dyn")}),
						author.Success(),
					},
				),
			},
		})
		if err == nil || !strings.Contains(err.Error(), "ambiguous send contract") {
			t.Fatalf("expected Compile to reject conflicting SendUsed schemas on same route, got: %v", err)
		}

		// Also verify when wide_out is declared on the same (Peer, Obligation) with SendUsed=false:
		// 1) valid Compile succeeds;
		// 2) tampered retained artifact bytes setting SendUsed=true on wide_out are rejected by runtime.Create;
		// 3) untampered execution sending an invalid value on strict_out is rejected before commit (0 intents, 2 history rows).
		senderArt, err := author.Compile(author.Definition{
			Name:   "sender",
			Inputs: schema.Record{"dyn": wideField},
			Ports: map[string]author.Port{
				"strict_out": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": strictField}},
				"wide_out":   {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": wideField}},
			},
			Steps: []author.Step{
				author.Send("strict_out", map[string]author.Expr{"mode": author.Var("dyn")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatalf("Compile with single SendUsed port failed: %v", err)
		}
		receiverArt, err := author.Compile(author.Definition{
			Name:    "receiver",
			Outputs: schema.Record{"got": strictField},
			Ports: map[string]author.Port{
				"in": {Peer: "sender", Obligation: "work", Receive: schema.Record{"mode": strictField}},
			},
			Steps: []author.Step{
				author.Receive("in", map[string]string{"mode": "got"}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatalf("Compile receiver failed: %v", err)
		}

		tamperedBytes := bytes.ReplaceAll(senderArt.Bytes(), []byte(`"SendUsed":false`), []byte(`"SendUsed":true`))
		sum := sha256.Sum256(tamperedBytes)
		tamperedID := hex.EncodeToString(sum[:])
		ctx := context.Background()
		tamperedProv := runtime.Provision{
			Programs: []runtime.Artifact{
				{ID: tamperedID, Semantics: "alan/1e/v1", Bytes: tamperedBytes},
				{ID: receiverArt.ID(), Semantics: "alan/1e/v1", Bytes: receiverArt.Bytes()},
			},
			Subjects: []runtime.Subject{
				{ID: "sender", Generation: 1, Owner: "owner:sender", Revision: 1},
				{ID: "receiver", Generation: 1, Owner: "owner:receiver", Revision: 1},
			},
			Runs: []runtime.InitialRun{
				{
					ID:         runtime.RunID{Subject: "sender", Generation: 1},
					ProgramID:  tamperedID,
					Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{"dyn": schema.Text("VIOLATES_STRICT_SCHEMA")}},
				},
				{
					ID:         runtime.RunID{Subject: "receiver", Generation: 1},
					ProgramID:  receiverArt.ID(),
					Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}},
				},
			},
			Permissions: []runtime.Permission{
				{Kind: "message", Source: "sender", Target: "receiver", Operation: "message", Obligation: "work", Revision: 1},
			},
		}
		if _, err := runtime.Create(ctx, filepath.Join(t.TempDir(), "tampered.sqlite"), tamperedProv, runtime.DefaultLimits()); err == nil {
			t.Fatal("expected runtime.Create to reject tampered retained artifact with ambiguous SendUsed contract")
		}

		// Untampered execution: Send on strict_out must be validated against strict_out (not wide_out) and rejected before durable commit.
		validProv := tamperedProv
		validProv.Programs[0] = runtime.Artifact{ID: senderArt.ID(), Semantics: "alan/1e/v1", Bytes: senderArt.Bytes()}
		validProv.Runs[0].ProgramID = senderArt.ID()
		path := filepath.Join(t.TempDir(), "strict_enforced.sqlite")
		s, err := runtime.Create(ctx, path, validProv, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("runtime.Create: %v", err)
		}
		defer s.Close()
		if _, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 10, MaxDispatch: 10}); err == nil {
			t.Fatal("expected invalid message on strict_out to be rejected before commit")
		}
		snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if len(snap.Tables["intents"]) != 0 {
			t.Fatalf("invalid message reached durable intents table: %+v", snap.Tables["intents"])
		}
		if len(snap.Tables["history"]) != 0 {
			t.Fatalf("invalid message appended to durable history: got %d rows", len(snap.Tables["history"]))
		}
		if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
			t.Fatalf("Replay failed: %v", err)
		}
	})

	t.Run("positive controls for legitimate multi-port applications", func(t *testing.T) {
		senderArt, err := author.Compile(author.Definition{
			Name: "sender",
			Ports: map[string]author.Port{
				"work_primary": {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": strictField}},
				"work_alias":   {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": strictField}},
				"work_unused":  {Peer: "receiver", Obligation: "work", Send: schema.Record{"mode": wideField}},
				"audit_out":    {Peer: "receiver", Obligation: "audit", Send: schema.Record{"note": wideField}},
			},
			Steps: []author.Step{
				author.Send("work_primary", map[string]author.Expr{"mode": author.String("fast")}),
				author.Send("work_alias", map[string]author.Expr{"mode": author.String("slow")}),
				author.Send("audit_out", map[string]author.Expr{"note": author.String("audit-trail-ok")}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatalf("Compile(sender): %v", err)
		}
		receiverArt, err := author.Compile(author.Definition{
			Name:    "receiver",
			Outputs: schema.Record{"first": strictField, "second": strictField, "note": wideField},
			Ports: map[string]author.Port{
				"work_in":  {Peer: "sender", Obligation: "work", Receive: schema.Record{"mode": strictField}},
				"audit_in": {Peer: "sender", Obligation: "audit", Receive: schema.Record{"note": wideField}},
			},
			Steps: []author.Step{
				author.Receive("work_in", map[string]string{"mode": "first"}),
				author.Receive("work_in", map[string]string{"mode": "second"}),
				author.Receive("audit_in", map[string]string{"note": "note"}),
				author.Success(),
			},
		})
		if err != nil {
			t.Fatalf("Compile(receiver): %v", err)
		}
		prov, err := (author.Application{
			"sender":   {Artifact: senderArt},
			"receiver": {Artifact: receiverArt},
		}).Provision()
		if err != nil {
			t.Fatalf("Provision: %v", err)
		}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "multiport_positive.sqlite")
		s, err := runtime.Create(ctx, path, prov, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer s.Close()
		sum, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 20, MaxDispatch: 20})
		if err != nil || !sum.Quiescent || len(sum.OpenRuns) != 0 {
			t.Fatalf("Run failed: sum=%+v err=%v", sum, err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("Replay: %v", err)
		}
		if len(replay.Runs) != 2 || replay.Runs[0].Projection.Status != "succeeded" || replay.Runs[1].Projection.Status != "succeeded" {
			t.Fatalf("unexpected replay runs: %+v", replay.Runs)
		}
	})
}

func TestRuntimeExecutesFalseRejectionProgramsAndCatchesStaticBypassesBeforeCommit(t *testing.T) {
	enumField := schema.Field{Kind: "string", MaxBytes: 10, Enum: []string{"fast", "slow"}}

	t.Run("all five false-rejection classes compile execute to succeeded and replay", func(t *testing.T) {
		validPrograms := []struct {
			name   string
			def    author.Definition
			inputs map[string]schema.Value
		}{
			{
				name: "FR1_contradicted_IfEqual_branch",
				def: author.Definition{
					Outputs: schema.Record{"mode": enumField, "seen": {Kind: "bool"}},
					Steps: []author.Step{
						author.Set("mode", author.String("fast")),
						author.IfEqual(author.Var("mode"), author.String("NOT_IN_ENUM"),
							[]author.Step{author.Set("seen", author.Boolean(true))},
							[]author.Step{author.Set("seen", author.Boolean(false))},
						),
						author.Success(),
					},
				},
			},
			{
				name:   "FR2_correlated_flag_guard",
				inputs: map[string]schema.Value{"go": schema.Bool(true)},
				def: author.Definition{
					Inputs:  schema.Record{"go": {Kind: "bool"}},
					Outputs: schema.Record{"out": enumField},
					Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": enumField}}},
					Steps: []author.Step{
						author.IfEqual(author.Var("go"), author.Boolean(true),
							[]author.Step{author.Set("mode", author.String("fast"))},
							[]author.Step{author.Set("mode", author.String("UNSET_SENTINEL"))},
						),
						author.IfEqual(author.Var("go"), author.Boolean(true),
							[]author.Step{
								author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("mode")}, nil),
								author.Await("tool", "req", map[string]string{"valid": "ok", "res": "out"}),
								author.Success(),
							},
							[]author.Step{author.Failure()},
						),
					},
				},
			},
			{
				name: "FR3_collection_row_correlation_in_ForEach",
				def: author.Definition{
					Outputs: schema.Record{"last": enumField},
					Collections: map[string]schema.Field{
						"items": {
							Kind: "list", MaxItems: 4, MaxBytes: 512,
							Element: schema.Record{"active": {Kind: "bool"}, "mode": {Kind: "string", MaxBytes: 16}},
						},
					},
					Values: map[string]schema.Value{
						"last": schema.Text("slow"),
						"items": schema.List(
							map[string]schema.Value{"active": schema.Bool(false), "mode": schema.Text("SKIP_ROW")},
							map[string]schema.Value{"active": schema.Bool(true), "mode": schema.Text("fast")},
						),
					},
					Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": enumField}}},
					Steps: []author.Step{
						author.ForEach(author.Var("items"), map[string]string{"active": "act", "mode": "m"},
							author.IfEqual(author.Var("act"), author.Boolean(true),
								[]author.Step{
									author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("m")}, nil),
									author.Await("tool", "req", map[string]string{"valid": "ok", "res": "last"}),
								},
								nil,
							),
						),
						author.Success(),
					},
				},
			},
			{
				name: "FR4_nonempty_ForEach_overwriting_sentinel",
				def: author.Definition{
					Outputs: schema.Record{"x": enumField},
					Collections: map[string]schema.Field{
						"items": {
							Kind: "list", MaxItems: 4, MaxBytes: 512,
							Element: schema.Record{"v": {Kind: "string", MaxBytes: 16}},
						},
					},
					Values: map[string]schema.Value{
						"items": schema.List(map[string]schema.Value{"v": schema.Text("fast")}),
						"x":     schema.Text("UNSET_SENTINEL"),
					},
					Steps: []author.Step{
						author.ForEach(author.Var("items"), map[string]string{"v": "x"}),
						author.Success(),
					},
				},
			},
			{
				name: "FR5_intermediate_Set_on_output_overwritten_before_Success",
				def: author.Definition{
					Outputs: schema.Record{"mode": enumField},
					Steps: []author.Step{
						author.Set("mode", author.String("TEMP_WORKING_VALUE")),
						author.Set("mode", author.String("fast")),
						author.Success(),
					},
				},
			},
		}

		for _, tc := range validPrograms {
			t.Run(tc.name, func(t *testing.T) {
				art, err := author.Compile(tc.def)
				if err != nil {
					t.Fatalf("Compile(%s) failed: %v", tc.name, err)
				}
				prov, err := (author.Application{
					"worker": {Artifact: art, Inputs: tc.inputs},
				}).Provision()
				if err != nil {
					t.Fatalf("Provision(%s): %v", tc.name, err)
				}
				ctx := context.Background()
				path := filepath.Join(t.TempDir(), tc.name+".sqlite")
				s, err := runtime.Create(ctx, path, prov, runtime.DefaultLimits())
				if err != nil {
					t.Fatalf("Create(%s): %v", tc.name, err)
				}
				sum, err := s.Run(ctx, runtime.RunOptions{
					MaxSteps: 20, MaxDispatch: 20,
					Adapters: map[string]effects.Adapter{
						"model": schemaTestAdapter{fn: func(effects.Request) map[string]schema.Value {
							return map[string]schema.Value{"res": schema.Text("fast")}
						}},
					},
				})
				_ = s.Close()
				if err != nil || !sum.Quiescent || len(sum.OpenRuns) != 0 {
					t.Fatalf("Run(%s) failed: sum=%+v err=%v", tc.name, sum, err)
				}
				replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
				if err != nil {
					t.Fatalf("Replay(%s): %v", tc.name, err)
				}
				if len(replay.Runs) != 1 || replay.Runs[0].Projection.Status != "succeeded" {
					t.Fatalf("expected succeeded run in Replay(%s), got: %+v", tc.name, replay.Runs)
				}
			})
		}
	})

	t.Run("static bypass and runtime-dependent invalid programs are rejected before commit", func(t *testing.T) {
		runtimeInvalidCases := []struct {
			name   string
			def    author.Definition
			inputs map[string]schema.Value
		}{
			{
				name:   "bypass_1_IfEqual_var_to_var_comparison",
				inputs: map[string]schema.Value{"dyn": schema.Text("NOT_IN_ENUM")},
				def: author.Definition{
					Inputs:  schema.Record{"dyn": {Kind: "string", MaxBytes: 20}},
					Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
					Steps: []author.Step{
						author.Set("bad", author.String("NOT_IN_ENUM")),
						author.IfEqual(author.Var("dyn"), author.Var("bad"),
							[]author.Step{
								author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("dyn")}, nil),
								author.Success(),
							},
							[]author.Step{author.Failure()},
						),
					},
				},
			},
			{
				name:   "bypass_2_IfEqual_alias_not_refined",
				inputs: map[string]schema.Value{"dyn": schema.Text("NOT_IN_ENUM")},
				def: author.Definition{
					Inputs:  schema.Record{"dyn": {Kind: "string", MaxBytes: 20}},
					Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
					Steps: []author.Step{
						author.Set("alias", author.Var("dyn")),
						author.IfEqual(author.Var("dyn"), author.String("NOT_IN_ENUM"),
							[]author.Step{
								author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("alias")}, nil),
								author.Success(),
							},
							[]author.Step{author.Failure()},
						),
					},
				},
			},
			{
				name:   "branch_dependent_request_with_invalid_branch_selected",
				inputs: map[string]schema.Value{"flag": schema.Bool(false)},
				def: author.Definition{
					Inputs:  schema.Record{"flag": {Kind: "bool"}},
					Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
					Steps: []author.Step{
						author.IfEqual(author.Var("flag"), author.Boolean(true),
							[]author.Step{author.Set("tmp", author.String("fast"))},
							[]author.Step{author.Set("tmp", author.String("NOT_IN_ENUM"))},
						),
						author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("tmp")}, nil),
						author.Success(),
					},
				},
			},
			{
				name: "collection_element_invalid_at_runtime_sink",
				def: author.Definition{
					Collections: map[string]schema.Field{"items": {Kind: "list", MaxItems: 4, MaxBytes: 512, Element: schema.Record{"raw": {Kind: "string", MaxBytes: 30}}}},
					Values:      map[string]schema.Value{"items": schema.List(map[string]schema.Value{"raw": schema.Text("NOT_IN_ENUM")})},
					Effects:     map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}}},
					Steps: []author.Step{
						author.ForEach(author.Var("items"), map[string]string{"raw": "x"},
							author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("x")}, nil),
						),
						author.Success(),
					},
				},
			},
		}

		for _, tc := range runtimeInvalidCases {
			t.Run(tc.name, func(t *testing.T) {
				art, err := author.Compile(tc.def)
				if err != nil {
					t.Fatalf("expected %s to compile under Design C, got: %v", tc.name, err)
				}
				prov, err := (author.Application{
					"worker": {Artifact: art, Inputs: tc.inputs},
				}).Provision()
				if err != nil {
					t.Fatalf("Provision(%s): %v", tc.name, err)
				}
				ctx := context.Background()
				path := filepath.Join(t.TempDir(), tc.name+".sqlite")
				s, err := runtime.Create(ctx, path, prov, runtime.DefaultLimits())
				if err != nil {
					t.Fatalf("Create(%s): %v", tc.name, err)
				}
				_, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 10, MaxDispatch: 10})
				_ = s.Close()
				if runErr == nil {
					t.Fatalf("expected runtime pre-commit rejection for %s, got nil", tc.name)
				}
				snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
				if err != nil {
					t.Fatalf("Inspect(%s): %v", tc.name, err)
				}
				if len(snap.Tables["intents"]) != 0 {
					t.Fatalf("expected 0 committed intents for %s, got %d", tc.name, len(snap.Tables["intents"]))
				}
				if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
					t.Fatalf("Replay(%s): %v", tc.name, err)
				}
			})
		}
	})
}

func TestPR15Adversarial_DifferentialSeed20261027(t *testing.T) {
	rng := rand.New(rand.NewSource(20261027))
	enumField := schema.Field{Kind: "string", MaxBytes: 8, Enum: []string{"fast", "slow"}}
	wideField := schema.Field{Kind: "string", MaxBytes: 16}

	var accepted, rejected, falseAcceptances, falseRejections, runtimeCaught int
	for i := 0; i < 100; i++ {
		pattern := rng.Intn(4)
		var def author.Definition
		var initVal string
		expectValidAtRuntime := false

		switch pattern {
		case 0:
			val := "fast"
			if rng.Intn(2) == 1 {
				val = "BAD_ENUM"
			} else {
				expectValidAtRuntime = true
			}
			initVal = "fast"
			def = author.Definition{
				Inputs:  schema.Record{"in": wideField},
				Outputs: schema.Record{"out": enumField},
				Steps: []author.Step{
					author.Set("tmp", author.String(val)),
					author.Set("out", author.Var("tmp")),
					author.Success(),
				},
			}
		case 1:
			initVal = "fast"
			expectValidAtRuntime = true
			def = author.Definition{
				Inputs:  schema.Record{"in": wideField},
				Outputs: schema.Record{"out": enumField},
				Steps: []author.Step{
					author.IfEqual(author.Var("in"), author.String("fast"),
						[]author.Step{author.Set("tmp", author.String("fast"))},
						[]author.Step{author.Set("tmp", author.String("SENTINEL"))},
					),
					author.IfEqual(author.Var("in"), author.String("fast"),
						[]author.Step{author.Set("out", author.Var("tmp")), author.Success()},
						[]author.Step{author.Failure()},
					),
				},
			}
		case 2:
			if rng.Intn(2) == 0 {
				initVal = "slow"
				expectValidAtRuntime = true
			} else {
				initVal = "INVALID_RUNTIME"
				expectValidAtRuntime = false
			}
			def = author.Definition{
				Inputs:  schema.Record{"in": wideField},
				Outputs: schema.Record{"out": enumField},
				Effects: map[string]author.Effect{"tool": {Capability: "model", Input: schema.Record{"mode": enumField}, Output: schema.Record{"res": enumField}}},
				Steps: []author.Step{
					author.Request("req", "tool", map[string]author.Expr{"mode": author.Var("in")}, nil),
					author.Set("out", author.String("fast")),
					author.Success(),
				},
			}
		case 3:
			initVal = "fast"
			expectValidAtRuntime = true
			def = author.Definition{
				Inputs:  schema.Record{"in": wideField},
				Outputs: schema.Record{"out": enumField},
				Steps: []author.Step{
					author.Set("tmp", author.String("slow")),
					author.IfEqual(author.Var("tmp"), author.String("IMPOSSIBLE"),
						[]author.Step{author.Set("flag", author.Value(schema.Bool(true)))},
						[]author.Step{author.Set("flag", author.Value(schema.Bool(false)))},
					),
					author.Set("out", author.Var("tmp")),
					author.Success(),
				},
			}
		}

		art, err := author.Compile(def)
		if err != nil {
			rejected++
			if expectValidAtRuntime {
				falseRejections++
			}
			continue
		}
		accepted++

		prov, err := (author.Application{
			"worker": {Artifact: art, Inputs: map[string]schema.Value{"in": schema.Text(initVal)}},
		}).Provision()
		if err != nil {
			t.Fatalf("iter %d Provision: %v", i, err)
		}
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), fmt.Sprintf("diff_%d.sqlite", i))
		s, err := runtime.Create(ctx, path, prov, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("iter %d Create: %v", i, err)
		}
		_, runErr := s.Run(ctx, runtime.RunOptions{
			MaxSteps: 10, MaxDispatch: 10,
			Adapters: map[string]effects.Adapter{
				"model": schemaTestAdapter{fn: func(effects.Request) map[string]schema.Value {
					return map[string]schema.Value{"res": schema.Text("fast")}
				}},
			},
		})
		_ = s.Close()
		if !expectValidAtRuntime {
			if runErr == nil {
				falseAcceptances++
				t.Fatalf("iter %d: expected runtime rejection for invalid input %q, got nil", i, initVal)
			}
			runtimeCaught++
			snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				t.Fatalf("iter %d Inspect: %v", i, err)
			}
			if len(snap.Tables["intents"]) != 0 {
				falseAcceptances++
				t.Fatalf("iter %d: invalid intent committed: %+v", i, snap.Tables["intents"])
			}
		} else if runErr != nil {
			t.Fatalf("iter %d: expected valid run, got error: %v", i, runErr)
		}
		if _, err := runtime.Replay(ctx, path, runtime.DefaultLimits()); err != nil {
			t.Fatalf("iter %d Replay: %v", i, err)
		}
		snap, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			t.Fatalf("iter %d Inspect: %v", i, err)
		}
		var proj runtime.Projection
		if err := json.Unmarshal([]byte(snap.Tables["runs"][0]["projection"].(string)), &proj); err != nil {
			t.Fatalf("iter %d unmarshal projection: %v", i, err)
		}
	}
	t.Logf("Differential summary (seed=20261027, N=100): accepted=%d, rejected=%d, falseAcceptances=%d, falseRejections=%d, runtimeCaught=%d",
		accepted, rejected, falseAcceptances, falseRejections, runtimeCaught)
	if falseAcceptances != 0 || falseRejections != 0 {
		t.Fatalf("expected 0 falseAcceptances and 0 falseRejections, got falseAcceptances=%d falseRejections=%d", falseAcceptances, falseRejections)
	}
}

func BenchmarkPR15CompileScaling(b *testing.B) {
	enumVals := make([]string, 32)
	for i := range enumVals {
		enumVals[i] = fmt.Sprintf("opt_%02d", i)
	}
	enumField := schema.Field{Kind: "string", MaxBytes: 16, Enum: enumVals}

	b.Run("SmallStraightLine", func(b *testing.B) {
		def := author.Definition{
			Outputs: schema.Record{"out": enumField},
			Steps: []author.Step{
				author.Set("x", author.String("opt_00")),
				author.Set("out", author.Var("x")),
				author.Success(),
			},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := author.Compile(def); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("BranchJoinChain_60", func(b *testing.B) {
		steps := []author.Step{author.Set("x", author.String("opt_00"))}
		for i := 0; i < 60; i++ {
			opt := enumVals[i%len(enumVals)]
			steps = append(steps, author.IfEqual(author.Var("in"), author.Value(schema.Int(int64(i))),
				[]author.Step{author.Set("x", author.String(opt))},
				[]author.Step{author.Set("x", author.Var("x"))},
			))
		}
		steps = append(steps, author.Set("out", author.Var("x")), author.Success())
		def := author.Definition{
			Inputs:  schema.Record{"in": {Kind: "int"}},
			Outputs: schema.Record{"out": enumField},
			Steps:   steps,
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := author.Compile(def); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NestedForEach_Depth3_WideEnum", func(b *testing.B) {
		innerList := schema.Field{
			Kind: "list", MaxItems: 8, MaxBytes: 1024,
			Element: schema.Record{"v": enumField},
		}
		midList := schema.Field{
			Kind: "list", MaxItems: 8, MaxBytes: 2048,
			Element: schema.Record{"inner": innerList},
		}
		outerList := schema.Field{
			Kind: "list", MaxItems: 8, MaxBytes: 4096,
			Element: schema.Record{"mid": midList},
		}
		def := author.Definition{
			Collections: map[string]schema.Field{"outer": outerList},
			Values: map[string]schema.Value{
				"outer": schema.List(map[string]schema.Value{
					"mid": schema.List(map[string]schema.Value{
						"inner": schema.List(
							map[string]schema.Value{"v": schema.Text("opt_00")},
							map[string]schema.Value{"v": schema.Text("opt_01")},
							map[string]schema.Value{"v": schema.Text("opt_02")},
							map[string]schema.Value{"v": schema.Text("opt_03")},
						),
					}),
				}),
			},
			Outputs: schema.Record{"out": enumField},
			Steps: []author.Step{
				author.Set("out", author.String("opt_00")),
				author.ForEach(author.Var("outer"), map[string]string{"mid": "m"},
					author.ForEach(author.Var("m"), map[string]string{"inner": "in_list"},
						author.ForEach(author.Var("in_list"), map[string]string{"v": "out"}),
					),
				),
				author.Success(),
			},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := author.Compile(def); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LoopWithShiftAndBranches_MaxInstructions", func(b *testing.B) {
		colField := schema.Field{
			Kind: "list", MaxItems: 32, MaxBytes: 4096,
			Element: schema.Record{"v": enumField},
		}
		var body []author.Step
		for i := 15; i > 0; i-- {
			body = append(body, author.Set(fmt.Sprintf("v_%02d", i), author.Var(fmt.Sprintf("v_%02d", i-1))))
		}
		for i := 0; i < 32; i++ {
			opt := enumVals[i%len(enumVals)]
			body = append(body, author.IfEqual(author.Var("in"), author.Value(schema.Int(int64(i))),
				[]author.Step{author.Set("v_00", author.String(opt))},
				[]author.Step{author.Set("v_00", author.Var("v_00"))},
			))
		}
		steps := []author.Step{}
		for i := 0; i < 16; i++ {
			steps = append(steps, author.Set(fmt.Sprintf("v_%02d", i), author.String("opt_00")))
		}
		steps = append(steps,
			author.ForEach(author.Var("items"), map[string]string{"v": "v_00"}, body...),
			author.Set("out", author.Var("v_15")),
			author.Success(),
		)
		def := author.Definition{
			Inputs:      schema.Record{"in": {Kind: "int"}},
			Collections: map[string]schema.Field{"items": colField},
			Outputs:     schema.Record{"out": enumField},
			Steps:       steps,
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := author.Compile(def); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestHistoricalReplayAndArtifactEncodingCompatibility(t *testing.T) {
	ctx := context.Background()
	stores := []string{
		filepath.Join("..", "docs", "foundational", "phase1f", "evidence", "live-phase1e", "store.sqlite"),
		filepath.Join("..", "docs", "foundational", "phase1f", "evidence", "live-development-v2", "dev-01-baseline", "store.sqlite"),
		filepath.Join("..", "docs", "foundational", "phase1f", "evidence", "live-development-v2", "dev-01-investigation", "store.sqlite"),
	}
	for _, storePath := range stores {
		if _, err := runtime.Replay(ctx, storePath, runtime.DefaultLimits()); err != nil {
			t.Fatalf("historical Replay(%s) failed: %v", storePath, err)
		}
	}

	// Verify unchanged valid definitions preserve exact artifact encoding and ID from baseline 4550e0f4.
	d := author.Definition{
		Name:      "answer",
		Inputs:    schema.Record{"question": {Kind: "string", MaxBytes: 200}},
		Outputs:   schema.Record{"answer": {Kind: "string", MaxBytes: 200}},
		Templates: map[string]string{"plan": "verbatim\nplanning", "reply": "answer the question"},
		Steps:     []author.Step{author.Set("answer", author.Var("question")), author.Success()},
	}
	art, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	const wantBaselineID = "7f55108db4bbc18fd058a8a62a4a1fa3e0c71da548d5cce071620f86797c5c94"
	if art.ID() != wantBaselineID {
		t.Fatalf("artifact ID changed for unchanged valid program: got %s, want %s", art.ID(), wantBaselineID)
	}
}
