//go:build integration

package adaptivetest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/runtime"
)

func jsonUnmarshal(raw any, out any) error { return json.Unmarshal([]byte(raw.(string)), out) }
func TestAdaptiveReadOnlyReplay(t *testing.T) {
	c := component(t, true)
	provider, server := newProvider(t, -1, false, false)
	d, path := provision(t, c, true, 2, nil)
	create(t, d, path, true)
	p := start(t, path, c.ID(), "", server.URL, false, "")
	p.wait(t, false)
	before := inspect(t, path)
	must(t, applicationOracle(before, c.ID(), frozen(t, false), true, 7))
	ledger, _, _ := provider.snapshot()
	replay := start(t, path, c.ID(), "", "", true, "")
	replay.wait(t, false)
	after := inspect(t, path)
	last, _, _ := provider.snapshot()
	if digest(before) != digest(after) || digest(ledger) != digest(last) {
		t.Fatal("replay changed ledger/contact")
	}
	retain(t, "replayed", after)
	emit(t, map[string]any{"kind": "read_only_replay", "before_ledger_sha256": digest(before), "after_ledger_sha256": digest(after), "before_contacts": contacts(ledger), "after_contacts": contacts(last), "network_isolation": replay.network, "credential_absent": true, "adapter_registry_present": false, "runtime_verdict": "Replay passed without mutation", "application_verdict": "frozen primary accepted independently"})
}
func TestAdaptiveFaultChild(t *testing.T) {
	path := os.Getenv("ALAN_H_FAULT_DB")
	if path == "" {
		t.Skip("fault subprocess only")
	}
	c := component(t, true)
	d, _ := provision(t, c, true, 2, nil)
	create(t, d, path, true)
	s, err := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, err)
	report, err := s.Run(t.Context(), runtime.RunOptions{Adapters: map[string]effects.Adapter{adaptive.ScalarCapability: httpAdapter{os.Getenv("ALAN_H_FAULT_ENDPOINT")}}, MaxSteps: 30000, MaxDispatch: 3000, MaxInFlight: 2, ExternalTimeout: time.Minute})
	must(t, err)
	must(t, s.Close())
	replay, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	must(t, err)
	emit(t, map[string]any{"kind": "fault_child_protocol", "adaptive_template": c.ID(), "runtime_result": report, "replay_result": replay, "runtime_verdict": "Run and Replay valid"})
}
func TestFaultyAdaptiveApplicationOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	must(t, err)
	path := filepath.Join(t.TempDir(), "fault.sqlite")
	source := filepath.Join(root, "examples/adaptive/adaptive.go")
	original, err := os.ReadFile(source)
	must(t, err)
	needle := "classify := append(p.classify(),"
	if strings.Count(string(original), needle) != 1 {
		t.Fatal("fault overlay insertion no longer exact")
	}
	replacement := "classify := append(append(p.classify(), eq(\"work_id\", author.Integer(3), eq(\"leaf_observation\", author.Integer(0), author.Set(\"observation\", author.Integer(2))))),"
	mutated := strings.Replace(string(original), needle, replacement, 1)
	folder := t.TempDir()
	modified := filepath.Join(folder, "adaptive.go")
	must(t, os.WriteFile(modified, []byte(mutated), 0644))
	overlay := map[string]any{"Replace": map[string]string{source: modified}}
	overlayPath := filepath.Join(folder, "overlay.json")
	raw, err := json.Marshal(overlay)
	must(t, err)
	must(t, os.WriteFile(overlayPath, raw, 0644))
	binary := filepath.Join(folder, "adaptive-fault.test")
	args := []string{"test", "-tags", "integration", "-c", "-o", binary, "-overlay", overlayPath}
	race := false
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, v := range info.Settings {
			if v.Key == "-race" && v.Value == "true" {
				race = true
			}
		}
	}
	if race {
		args = append(args, "-race")
	}
	args = append(args, "./internal/adaptivetest")
	cmd := exec.CommandContext(t.Context(), "go", args...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	retain(t, "overlay", map[string]any{"original_source": string(original), "fault_source": mutated, "overlay": overlay, "build_argv": cmd.Args, "build_output": string(output), "race_instrumented": race})
	must(t, err)
	provider, server := newProvider(t, -1, true, false)
	child := exec.CommandContext(context.WithoutCancel(t.Context()), binary, "-test.run=^TestAdaptiveFaultChild$", "-test.v", "-test.timeout=5m")
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "OPENAI_API_KEY=") && !strings.HasPrefix(v, "ALAN_H_") && !strings.HasPrefix(v, "ALAN_BARRIER=") {
			child.Env = append(child.Env, v)
		}
	}
	child.Env = append(child.Env, "ALAN_H_FAULT_DB="+path, "ALAN_H_FAULT_ENDPOINT="+server.URL)
	out, err := child.CombinedOutput()
	emit(t, map[string]any{"kind": "fault_child_exit", "argv": child.Args, "output": string(out), "race_instrumented": race})
	must(t, err)
	s := inspect(t, path)
	var template string
	for _, r := range s.Tables["composition_invocations"] {
		d := decode[runtime.InvocationDescriptor](r, "descriptor")
		if len(d.Inputs["seeds"].Kind) > 0 && d.Depth == 2 {
			template = d.TemplateID
		}
	}
	if template == "" || template == component(t, true).ID() {
		t.Fatal("authentic changed adaptive artifact absent")
	}
	ops := itemOperations(s, template)
	for _, id := range []int64{10, 11, 31} {
		if !admitted(s, ops[id]) {
			t.Fatal("fault never reached forbidden successor admission", id)
		}
	}
	gamma, ok := outcome(s, ops[3])
	if !ok || gamma.Response["observation"].Number != 0 {
		t.Fatal("original gamma0 was not authenticated")
	}
	if s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "succeeded" {
		t.Fatal("control failed runtime before semantic oracle")
	}
	rejection := applicationOracle(s, template, frozen(t, true), true, 4)
	if rejection == nil {
		t.Fatal("unchanged oracle accepted faulty application")
	}
	ledger, _, _ := provider.snapshot()
	must(t, contactMonitor(s, ledger))
	retain(t, "faulty_execution", s)
	emit(t, map[string]any{"kind": "faulty_application_oracle", "actual_adaptive_template": template, "ordinary_fault": "authenticated gamma0 classification overwritten with normal Set2", "original_gamma_outcome": gamma, "forbidden_admitted": []int64{10, 11, 31}, "operations": ops, "runtime_verdict": "Run and Replay passed", "application_verdict": "rejected by unchanged paired oracle", "oracle_rejection": fmt.Sprint(rejection), "race_instrumented": race, "provider_ledger": ledger})
}
