package runtime_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func ExampleStore_RunProgress() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "alan-progress-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "example.sqlite")

	program, err := author.Compile(author.Definition{
		Name: "wait-for-input",
		Ports: map[string]author.Port{
			"start": {Peer: "local", Local: true, Obligation: "start", Receive: schema.Record{}},
		},
		Steps: []author.Step{
			author.Set("status", author.String("ready")),
			author.Receive("start", nil),
			author.Success(),
		},
	})
	if err != nil {
		panic(err)
	}
	provision, err := (author.Application{"worker": {Artifact: program}}).Provision()
	if err != nil {
		panic(err)
	}
	limits := runtime.DefaultLimits()
	store, err := runtime.Create(ctx, path, provision, limits)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// The counters describe this call. Inspect explicitly to see why it stopped.
	progress, err := store.RunProgress(ctx, runtime.RunOptions{MaxSteps: 10, MaxDispatch: 10})
	if err != nil {
		panic(err)
	}
	fmt.Printf("steps=%d dispatches=%d quiescent=%t\n", progress.Steps, progress.Dispatches, progress.Quiescent)
	cut, err := runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		panic(err)
	}
	worker, found := cut.Run(runtime.RunID{Subject: "worker", Generation: 1})
	if !found || worker.Wait == nil {
		panic("worker is not waiting")
	}
	fmt.Printf("worker=%s waiting=%s\n", worker.Projection.Status, worker.Wait.Kind)

	// Output:
	// steps=1 dispatches=0 quiescent=true
	// worker=open waiting=input
}

var _ func(*runtime.Store, context.Context, runtime.RunOptions) (runtime.RunProgress, error) = (*runtime.Store).RunProgress
