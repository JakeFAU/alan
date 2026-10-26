//go:build integration

package compositiontest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func effectInput() schema.Value {
	rows := []map[string]schema.Value{}
	for i, q := range []string{"fast", "hold", "later"} {
		rows = append(rows, map[string]schema.Value{"item_id": schema.Text([]string{"item-a", "item-b", "item-c"}[i]), "question": schema.Text(q)})
	}
	return schema.List(rows...)
}

type contact struct {
	Order                     int
	Question, Phase, ClientID string
}
type provider struct {
	ended            chan struct{}
	mu               sync.Mutex
	ledger           []contact
	active, peak     int
	entered, release chan struct{}
	once             sync.Once
}

func (p *provider) record(q, phase, key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ledger = append(p.ledger, contact{Order: len(p.ledger) + 1, Question: q, Phase: phase, ClientID: key})
}
func (p *provider) serve(w http.ResponseWriter, r *http.Request) {
	var req struct{ Input []struct{ Content string } }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Input) != 1 {
		http.Error(w, "bad controlled request", 400)
		return
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(req.Input[0].Content), &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	q, _ := input["question"].(string)
	p.mu.Lock()
	p.active++
	if p.active > p.peak {
		p.peak = p.active
	}
	p.mu.Unlock()
	defer func() { p.mu.Lock(); p.active--; p.mu.Unlock() }()
	key := r.Header.Get("X-Client-Request-Id")
	p.record(q, "entered", key)
	if q == "hold" {
		p.once.Do(func() { close(p.entered) })
		select {
		case <-p.release:
		case <-r.Context().Done():
			p.record(q, "connection_closed_after_kill", key)
			close(p.ended)
			return
		}
	}
	result, _ := json.Marshal(map[string]any{"decision": "clarification", "tool": "", "query": "", "message": "clarify:" + q})
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "controlled", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(result)}}}}})
	p.record(q, "response_written", key)
}
func (p *provider) snapshot() ([]contact, int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]contact{}, p.ledger...), p.active, p.peak
}
func TestProcessPartialWithOutstandingTransport(t *testing.T) {
	p := &provider{ended: make(chan struct{}), ledger: []contact{}, entered: make(chan struct{}), release: make(chan struct{})}
	server := httptest.NewServer(http.HandlerFunc(p.serve))
	defer server.Close()
	defer close(p.release)
	leaf, err := worktree.Evidence(evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
	if err != nil {
		t.Fatal(err)
	}
	d, err := worktree.New(leaf, true, worktree.DefaultLimits(2))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "effect.sqlite")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	a, err := worktree.Submit(t.Context(), s, "effect-three", effectInput())
	if err != nil || !a.Accepted {
		t.Fatal(a, err)
	}
	s.Close()
	child := startEndpoint(t, path, d.Collection.ID(), "partial", false, server.URL)
	select {
	case <-p.entered:
	case <-time.After(300 * time.Second):
		t.Fatal("watchdog: no held transport")
	}
	ack := child.acknowledgment(t)
	before := inspect(t, path)
	if err = durableAck(before, ack); err != nil {
		t.Fatal(err)
	}
	accepted, err := acceptanceItems(before, d.Collection.ID(), effectInput())
	if err != nil {
		t.Fatal(err)
	}
	ledger, active, peak := p.snapshot()
	if active != 1 || peak > 2 {
		t.Fatal("not physically held", active, peak)
	}
	inv, subject := collection(before, d.Collection.ID())
	completed := rows(projection(before, subject), "results")
	if len(completed) != 1 || completed[0]["item_id"].Text != "item-a" {
		t.Fatal("partial collection not reached", completed)
	}
	var heldOperation string
	for _, r := range before.Tables["composition_invocations"] {
		v := decode[runtime.InvocationDescriptor](r, "descriptor")
		if v.ParentInvocationID == inv.ID && v.Inputs["question"].Text == "hold" {
			heldOperation = v.OperationID
		}
	}
	if heldOperation == "" {
		t.Fatal("held item invocation absent")
	}
	for _, r := range before.Tables["composition_results"] {
		if decode[runtime.InvocationOutcome](r, "descriptor").OperationID == heldOperation {
			t.Fatal("held operation already complete")
		}
	}
	if before.Roots[0].Outcome != nil {
		t.Fatal("premature root outcome")
	}
	premature := before
	premature.Roots = append(before.Roots[:0:0], before.Roots...)
	premature.Roots[0].Outcome = &runtime.InvocationOutcome{Status: "succeeded"}
	if err := pendingMonitor(premature, heldOperation); err == nil {
		t.Fatal("premature success mutant survived")
	}
	emit(t, map[string]any{"kind": "effect_partial_cut", "ack": ack, "acceptance": accepted, "held_operation": heldOperation, "completed": completed, "provider_ledger": ledger, "physical_active": active, "peak": peak, "snapshot": before})
	child.wait(t, true)
	select {
	case <-p.ended:
	case <-time.After(300 * time.Second):
		t.Fatal("watchdog: killed transport did not close")
	}
	recovered := startEndpoint(t, path, d.Collection.ID(), "", false, server.URL)
	recovered.wait(t, false)
	after := inspect(t, path)
	if err = pendingMonitor(after, heldOperation); err != nil {
		t.Fatal(err)
	}
	if after.Roots[0].Charges.Invocations != 5 || after.Roots[0].Charges.CreatedRuns != 8 || after.Roots[0].Charges.Effects != 3 || len(after.Roots[0].UnknownExternal) != 1 {
		t.Fatal("unknown identity/budget not retained", after.Roots)
	}
	unknown := after.Roots[0].UnknownExternal[0]
	finalLedger, _, finalPeak := p.snapshot()
	if err := contactBindings(after, finalLedger); err != nil {
		t.Fatal(err)
	}
	entries := 0
	heldEntries := 0
	for _, c := range finalLedger {
		if c.Phase == "entered" {
			entries++
			if c.Question == "hold" {
				heldEntries++
			}
		}
	}
	if entries != 3 || heldEntries != 1 || finalPeak > 2 {
		t.Fatal("repeated or excess physical dispatch", finalLedger)
	}
	replay := start(t, path, d.Collection.ID(), "", true)
	replay.wait(t, false)
	afterReplay := inspect(t, path)
	lastLedger, _, _ := p.snapshot()
	if !reflect.DeepEqual(finalLedger, lastLedger) || !reflect.DeepEqual(after, afterReplay) {
		t.Fatal("Replay changed contacts or store")
	}
	emit(t, map[string]any{"kind": "effect_recovered_unknown", "old_pid": child.cmd.Process.Pid, "new_pid": recovered.cmd.Process.Pid, "original_unknown": unknown, "held_invocation_operation": heldOperation, "provider_ledger": lastLedger, "after": after, "premature_success_mutant_detected": true, "replay_unchanged": true})
}
func pendingMonitor(s runtime.Inspection, operation string) error {
	known := false
	for _, r := range s.Tables["composition_results"] {
		if decode[runtime.InvocationOutcome](r, "descriptor").OperationID == operation {
			known = true
		}
	}
	if known {
		return fmt.Errorf("held/unknown item was reported complete")
	}
	if s.Roots[0].Outcome != nil && s.Roots[0].Outcome.Status == "succeeded" {
		return fmt.Errorf("success despite required unresolved item")
	}
	return nil
}

// The adapter sends a client correlation ID; this is not provider deduplication.
func contactBindings(s runtime.Inspection, ledger []contact) error {
	seen := map[string]bool{}
	for _, c := range ledger {
		if c.Phase != "entered" {
			continue
		}
		if c.ClientID == "" || seen[c.ClientID] {
			return fmt.Errorf("missing or repeated client correlation")
		}
		seen[c.ClientID] = true
		matches := 0
		for _, row := range s.Tables["adapter_attempts"] {
			a := decode[runtime.AdapterAttempt](row, "descriptor")
			expected := fmt.Sprintf("alan-%x", sha256.Sum256([]byte(a.Prepared.Request.Key)))
			if expected == c.ClientID && a.Intent.ID == a.Prepared.Request.ID && a.Intent.Payload["question"].Text == c.Question {
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("contact has %d exact prepared operation matches", matches)
		}
	}
	if len(seen) != len(s.Tables["adapter_attempts"]) {
		return fmt.Errorf("attempt/contact conservation differs")
	}
	return nil
}
