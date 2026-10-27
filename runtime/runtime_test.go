//go:build integration

package runtime_test

import (
	"context"
	"encoding/json"
	"path/filepath"
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
