package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/runtime"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		panic("one disposable store path required")
	}
	ctx := context.Background()
	path := os.Args[1]
	cut, e := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	check(e)
	replayed, e := runtime.Replay(ctx, path, runtime.DefaultLimits())
	check(e)
	before := make(map[runtime.RunID]runtime.RunState)
	for _, r := range cut.Runs {
		before[r.ID] = r.RunState
	}
	s, e := runtime.Open(ctx, path, runtime.DefaultLimits())
	check(e)
	check(s.Close())
	after, e := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	check(e)
	if len(after.Runs) != len(cut.Runs) || len(after.Effects) != len(cut.Effects) {
		panic("reopen changed logical identities")
	}
	for _, r := range after.Runs {
		old := before[r.ID]
		if old.ID != r.ID || old.ProgramID != r.ProgramID || old.Sequence != r.Sequence {
			panic("reopen changed retained run")
		}
	}
	out := map[string]any{"storage_version": cut.StorageVersion, "namespace": cut.Namespace, "runs": len(cut.Runs), "effects": len(cut.Effects), "continuations": len(cut.Continuations), "replay_runs": len(replayed.Runs), "open_close": "valid", "provider_contacts": 0}
	raw, e := json.MarshalIndent(out, "", "  ")
	check(e)
	fmt.Println(string(raw))
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}
