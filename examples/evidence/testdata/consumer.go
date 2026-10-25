// This file is built as an independent module by the application contract test.
package main

import (
	"context"
	"fmt"
	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"os"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	ctx := context.Background()
	path := os.Args[2]
	limits := runtime.DefaultLimits()
	switch os.Args[1] {
	case "create":
		first, err := evidence.New("first", "customer", "first-question", evidence.Budgets{MaxModelRequests: 3, MaxLookupRequests: 2})
		must(err)
		second, err := evidence.New("second", "customer", "second-question", evidence.Budgets{MaxModelRequests: 3, MaxLookupRequests: 2})
		must(err)
		caller, err := author.Compile(author.Definition{Name: "independent two-helper customer", Outputs: schema.Record{"answer": {Kind: "string", MaxBytes: 8192}}, Ports: map[string]author.Port{"a": first.Port, "b": second.Port}, Steps: []author.Step{
			author.Send("a", map[string]author.Expr{"question": author.String("How does WAL work?")}), author.Receive("a", map[string]string{"disposition": "first_disposition", "answer": "first_answer"}),
			author.IfEqual(author.Var("first_disposition"), author.String("clarification"), nil, []author.Step{author.Failure()}),
			author.Send("b", map[string]author.Expr{"question": author.Var("first_answer")}), author.Receive("b", map[string]string{"disposition": "second_disposition", "answer": "answer"}),
			author.IfEqual(author.Var("second_disposition"), author.String("clarification"), []author.Step{author.Success()}, []author.Step{author.Failure()}),
		}})
		must(err)
		app := author.Application{"customer": {Artifact: caller}}
		for name, instance := range first.Instances {
			app[name] = instance
		}
		for name, instance := range second.Instances {
			app[name] = instance
		}
		p, err := app.Provision()
		must(err)
		s, err := runtime.Create(ctx, path, p, limits)
		must(err)
		must(s.Close())
	case "resume":
		model, err := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Endpoint: os.Args[3], Model: "controlled-external", MaxOutputTokens: 512, MaxResponseBytes: 65536})
		must(err)
		s, err := runtime.Open(ctx, path, limits)
		must(err)
		report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 1000, MaxDispatch: 1000, Adapters: map[string]effects.Adapter{evidence.ModelCapability: model, evidence.LookupCapability: lookup.New()}})
		must(err)
		must(s.Close())
		if len(report.OpenRuns) != 0 {
			panic("open runs remain")
		}
	case "replay":
		report, err := runtime.Replay(ctx, path, limits)
		must(err)
		for _, run := range report.Runs {
			if run.Projection.Status != "succeeded" {
				panic("non-success")
			}
			if run.ID.Subject == "customer" && run.Projection.Data["answer"].Text != "Which durability guarantee do you mean?" {
				panic("wrong public result")
			}
		}
		fmt.Println("two evidence helper instances: independent customer consumed public clarification results; offline replay succeeded")
	default:
		panic("unknown command")
	}
}
