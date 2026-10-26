//go:build integration

package runtime_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

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
