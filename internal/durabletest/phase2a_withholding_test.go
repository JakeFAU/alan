//go:build integration

package durabletest

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/runtime"
)

// A5 companions. The first Run really launches and applies. After it returns
// without a result, an authorized operator closes the actual source before a
// new Run sees exactly the same allowed reservation.
func TestPhase2AClosedWithholding(t *testing.T) {
	for _, route := range []string{"non-idempotent", "adapter"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			world := filepath.Join(dir, "world.sqlite")
			responseMode := "drop-after-commit"
			if route == "adapter" {
				responseMode = "normal"
			}
			endpoint, url, journal := phaseResource(t, world, "", responseMode, nil)
			p := phaseProvision(t, route, url, false)
			addOperator(t, &p, "operator", "CancelRun", "client")
			db := filepath.Join(dir, "executor.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			opts := runtime.RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, Adapters: effectboundary.Registry(url, "unknown")}
			initial, err := s.Run(t.Context(), opts)
			if err != nil {
				t.Fatal(err)
			}
			before := inspect(t, db)
			source, _, seq := projection(before, "client", 1)
			if source.Status != "open" || seq != 1 || phaseResultCount(before, route) != 0 {
				t.Fatal("missing source await", source)
			}
			key, descriptor := phaseRequest(t, before, route)
			firstWorld := worldAt(t, world)
			if firstWorld.Contacts != 1 || len(firstWorld.Applications) != 1 || firstWorld.Applications[0].Key != key {
				t.Fatal("first application not bound", firstWorld)
			}
			ingress, err := os.ReadFile(journal)
			if err != nil {
				t.Fatal(err)
			}
			closure := manualControl(t, s, "operator", "client")
			if closure.Reason != "closed" || closure.Before.Projection.Status != "open" || closure.After.Projection.Status != "cancelled" {
				t.Fatal(closure)
			}
			closed := inspect(t, db)
			later, err := s.Run(t.Context(), opts)
			if err != nil {
				t.Fatal(err)
			}
			after := inspect(t, db)
			nextWorld := worldAt(t, world)
			nextIngress, err := os.ReadFile(journal)
			if err != nil {
				t.Fatal(err)
			}
			_, retained := phaseRequest(t, after, route)
			terminal, _, endSeq := projection(after, "client", 1)
			missing := later.UnknownExternal
			if route == "adapter" {
				missing = later.UnknownAdapters
			}
			if len(missing) != 1 || descriptor != retained || phaseResultCount(after, route) != 0 || terminal.Status != "cancelled" || seq != endSeq || !reflect.DeepEqual(firstWorld, nextWorld) || !bytes.Equal(ingress, nextIngress) {
				t.Fatal("closed operation was repeated, erased, or consumed", later)
			}
			captureAuthority(t, map[string]any{
				"route": route, "key": key, "before": before, "closed": closed, "after": after, "closure": closure, "first_report": initial, "second_report": later,
				"world_before": firstWorld, "world_after": nextWorld, "journal_before": string(ingress), "journal_after": string(nextIngress),
				"endpoint_pid": endpoint.cmd.Process.Pid, "endpoint_argv": endpoint.cmd.Args, "endpoint_sha256": binaryHash(endpoint.cmd.Path), "race": raceChildren,
			})
		})
	}
}
