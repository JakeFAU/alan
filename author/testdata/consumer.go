package main

import (
	"context"
	"fmt"
	"os"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func record(name string) schema.Record { return schema.Record{name: {Kind: "string", MaxBytes: 100}} }
func echo(caller, obligation string) author.Artifact {
	a, err := author.Compile(author.Definition{Name: "echo", Ports: map[string]author.Port{"caller": {Peer: caller, Obligation: obligation, Receive: record("question"), Send: record("answer")}}, Steps: []author.Step{author.Receive("caller", map[string]string{"question": "question"}), author.Send("caller", map[string]author.Expr{"answer": author.Var("question")}), author.Success()}})
	must(err)
	return a
}
func main() {
	ctx := context.Background()
	path := os.Args[2]
	limits := runtime.DefaultLimits()
	switch os.Args[1] {
	case "create":
		caller, err := author.Compile(author.Definition{Name: "caller", Inputs: record("question"), Outputs: record("answer"), Values: map[string]schema.Value{"count": schema.Int(0)}, Templates: map[string]string{"stage": "retained exact template\n"}, Ports: map[string]author.Port{"first": {Peer: "helper-a", Obligation: "q-a", Send: record("question"), Receive: record("answer")}, "second": {Peer: "helper-b", Obligation: "q-b", Send: record("question"), Receive: record("answer")}}, Steps: []author.Step{
			author.Repeat(2, author.Add("count", author.Var("count"), author.Integer(1))),
			author.Send("first", map[string]author.Expr{"question": author.Var("question")}), author.Receive("first", map[string]string{"answer": "answer"}),
			author.Send("second", map[string]author.Expr{"question": author.Var("answer")}), author.Receive("second", map[string]string{"answer": "answer"}),
			author.IfEqual(author.Var("count"), author.Integer(2), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		}})
		must(err)
		app := author.Application{"caller": {Artifact: caller, Inputs: map[string]schema.Value{"question": schema.Text("echo through two helpers")}}, "helper-a": {Artifact: echo("caller", "q-a")}, "helper-b": {Artifact: echo("caller", "q-b")}}
		p, err := app.Provision()
		must(err)
		s, err := runtime.Create(ctx, path, p, limits)
		must(err)
		// Stop after two finite commits. Reopen in another process without recompilation.
		_, err = s.Run(ctx, runtime.RunOptions{MaxSteps: 2, MaxDispatch: 100})
		if err == nil {
			panic("expected bounded first run")
		}
		must(s.Close())
	case "resume":
		s, err := runtime.Open(ctx, path, limits)
		must(err)
		report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100})
		must(err)
		if len(report.OpenRuns) != 0 {
			panic("open runs remain")
		}
		must(s.Close())
	case "replay":
		snapshot, err := runtime.Inspect(ctx, path, limits)
		must(err)
		if len(snapshot.Tables["programs"]) != 3 {
			panic("missing retained artifacts")
		}
		report, err := runtime.Replay(ctx, path, limits)
		must(err)
		if report.IntentCount != 4 {
			panic(fmt.Sprintf("intent count %d", report.IntentCount))
		}
		for _, run := range report.Runs {
			if run.Projection.Status != "succeeded" {
				panic("unsuccessful run")
			}
			if run.ID.Subject == "caller" && run.Projection.Data["answer"].Text != "echo through two helpers" {
				panic("wrong final answer")
			}
		}
		fmt.Println("public external consumer: compiled, resumed, replayed; 4 ordinary messages, 3 successful runs")
	default:
		panic("unknown mode")
	}
}
