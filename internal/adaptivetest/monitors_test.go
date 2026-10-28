//go:build integration

package adaptivetest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func emit(t *testing.T, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	must(t, err)
	t.Log("ADAPTIVE " + string(raw))
}
func decode[T any](r map[string]any, k string) T {
	var v T
	if err := json.Unmarshal([]byte(r[k].(string)), &v); err != nil {
		panic(err)
	}
	return v
}
func inspect(t *testing.T, path string) runtime.Inspection {
	t.Helper()
	s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	must(t, err)
	return s
}
func rawHash(v []byte) string { return fmt.Sprintf("%x", sha256.Sum256(v)) }
func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func rows(p runtime.Projection, name string) []map[string]schema.Value {
	r, _ := p.Data[name].Records()
	return r
}
func valueIDs(v schema.Value) []int64 {
	r, _ := v.Records()
	out := []int64{}
	for _, x := range r {
		out = append(out, x["work_id"].Number)
	}
	return out
}
func ids(p runtime.Projection, name string) []int64 { return valueIDs(p.Data[name]) }
func projection(s runtime.Inspection, subject string) runtime.Projection {
	for _, r := range s.Tables["runs"] {
		if r["subject"] == subject {
			return decode[runtime.Projection](r, "projection")
		}
	}
	return runtime.Projection{}
}
func adaptiveMember(s runtime.Inspection, template string) (runtime.InvocationDescriptor, string) {
	for _, r := range s.Tables["composition_invocations"] {
		d := decode[runtime.InvocationDescriptor](r, "descriptor")
		if d.TemplateID != template {
			continue
		}
		for _, m := range s.Tables["composition_members"] {
			if m["invocation_id"] == d.ID {
				return d, m["subject"].(string)
			}
		}
	}
	return runtime.InvocationDescriptor{}, ""
}

type retainedIntent struct {
	ID, Kind string
	Payload  map[string]schema.Value
}

func itemOperations(s runtime.Inspection, template string) map[int64]string {
	_, subject := adaptiveMember(s, template)
	out := map[int64]string{}
	for _, h := range s.Tables["history"] {
		if h["subject"] != subject {
			continue
		}
		p := decode[runtime.Projection](h, "successor")
		for _, i := range decode[[]retainedIntent](h, "intents") {
			if i.Kind == "invocation" {
				out[p.Data["work_id"].Number] = i.ID
			}
		}
	}
	return out
}
func outcome(s runtime.Inspection, op string) (runtime.InvocationOutcome, bool) {
	for _, table := range []string{"composition_publications", "composition_results"} {
		for _, r := range s.Tables[table] {
			v := decode[runtime.InvocationOutcome](r, "descriptor")
			if v.OperationID == op {
				return v, true
			}
		}
	}
	return runtime.InvocationOutcome{}, false
}
func admitted(s runtime.Inspection, op string) bool {
	for _, r := range s.Tables["composition_invocations"] {
		if decode[runtime.InvocationDescriptor](r, "descriptor").OperationID == op {
			var a struct{ Accepted bool }
			if err := json.Unmarshal([]byte(r["admission"].(string)), &a); err != nil {
				panic(err)
			}
			return a.Accepted
		}
	}
	return false
}
func consumed(s runtime.Inspection, op, subject string) bool {
	for _, d := range s.Tables["composition_outcome_deliveries"] {
		if d["operation_id"] != op {
			continue
		}
		for _, o := range s.Tables["observations"] {
			if o["id"] != d["observation_id"] || o["consumed_sequence"] == nil || o["subject"] != subject {
				continue
			}
			for _, h := range s.Tables["history"] {
				if h["subject"] == subject && h["generation"] == o["generation"] && h["sequence"] == o["consumed_sequence"] && h["selected_observation"] == o["id"] {
					return true
				}
			}
		}
	}
	return false
}
func origins(s runtime.Inspection) error {
	for _, r := range s.Tables["composition_invocations"] {
		v := decode[runtime.InvocationDescriptor](r, "descriptor")
		found := false
		for _, h := range s.Tables["history"] {
			if h["subject"] != v.Caller.Subject || h["generation"] != v.Caller.Generation || h["sequence"] != v.SourceSequence {
				continue
			}
			for _, i := range decode[[]retainedIntent](h, "intents") {
				if i.ID == v.OperationID && i.Kind == "invocation" {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("invocation lacks original committed source: %s", v.OperationID)
		}
	}
	return nil
}
func preserve(before, after runtime.Inspection) error {
	for _, original := range before.Tables["observations"] {
		found := false
		for _, later := range after.Tables["observations"] {
			if later["id"] == original["id"] && later["subject"] == original["subject"] && later["generation"] == original["generation"] && later["descriptor"] == original["descriptor"] && (original["consumed_sequence"] == nil || later["consumed_sequence"] == original["consumed_sequence"]) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("original observation identity/content/consumption changed")
		}
	}
	for _, table := range []string{"history", "intents", "composition_roots", "composition_static_members", "composition_templates", "composition_bindings", "composition_members", "composition_facts", "composition_invocations", "composition_admission_decisions", "composition_publications", "composition_results", "composition_charges", "composition_requests", "adapter_attempts", "adapter_results", "composition_reservations"} {
		for _, r := range before.Tables[table] {
			found := false
			for _, later := range after.Tables[table] {
				if reflect.DeepEqual(r, later) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("original %s row replaced", table)
			}
		}
	}
	return nil
}

type fixtureCase struct {
	Case           string    `json:"case"`
	Rounds         [][]int64 `json:"rounds"`
	Observations   [][]int64 `json:"round_observations"`
	OrderedResults []struct {
		ID          int64  `json:"work_id"`
		Value       int64  `json:"value"`
		Observation int64  `json:"observation"`
		Round       int64  `json:"round"`
		Parent      *int64 `json:"predecessor"`
	} `json:"ordered_results"`
	IDs     []int64 `json:"ordered_result_ids"`
	Inputs  []int64 `json:"ordered_result_inputs"`
	Count   int64   `json:"application_work_count"`
	Stop    string  `json:"stop_reason"`
	Charges struct {
		Direct, Nested struct {
			Invocations int64 `json:"accepted_invocations"`
			Runs        int64 `json:"created_runs_scalar"`
		}
	} `json:"charges"`
}

func frozen(t *testing.T, paired bool) fixtureCase {
	t.Helper()
	raw, err := os.ReadFile("../../docs/foundational/phase1h/evidence/fixtures/expected-work.json")
	must(t, err)
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "4fc58131e849fc15fd45dee5426c087fb7db4f99cb9539667da7b075f9cbff97" {
		t.Fatal("frozen oracle drift")
	}
	var ledger struct{ Cases []fixtureCase }
	must(t, json.Unmarshal(raw, &ledger))
	for _, c := range ledger.Cases {
		if (!paired && c.Case == "primary") || (paired && c.Case == "paired_gamma_zero") {
			return c
		}
	}
	t.Fatal("missing frozen case")
	return fixtureCase{}
}

// applicationOracle uses literal independently frozen expectations, never the
// application builder's derivation/classification helpers.
func applicationOracle(s runtime.Inspection, template string, f fixtureCase, nested bool, effects int64) error {
	p := projection(s, "root")
	got := rows(p, "results")
	wantIDs, wantValues := append([]int64{}, f.IDs...), append([]int64{}, f.Inputs...)
	obs := []int64{}
	for _, r := range f.Observations {
		obs = append(obs, r...)
	}
	for _, r := range f.OrderedResults {
		wantIDs = append(wantIDs, r.ID)
		wantValues = append(wantValues, r.Value)
	}
	if !reflect.DeepEqual(ids(p, "results"), wantIDs) {
		return fmt.Errorf("ordered work differs: got %v want %v", ids(p, "results"), wantIDs)
	}
	if len(obs) != len(got) {
		return fmt.Errorf("observation expectation count")
	}
	_, subject := adaptiveMember(s, template)
	ops := itemOperations(s, template)
	seen := map[string]bool{}
	position := 0
	for ri, frontier := range f.Rounds {
		for _, id := range frontier {
			r := got[position]
			response, _ := r["responses"].Records()
			op := r["operation"].Text
			parent := int64(0)
			edge := "seed"
			if ri > 0 {
				switch id {
				case 7:
					parent = 2
					edge = "check"
				case 10:
					parent = 3
					edge = "left"
				case 11:
					parent = 3
					edge = "right"
				case 31:
					parent = 10
					edge = "check"
				default:
					return fmt.Errorf("no literal predecessor for %d", id)
				}
			}
			if r["work_id"].Number != id || r["value"].Number != wantValues[position] || r["parent_id"].Number != parent || r["edge"].Text != edge || r["round"].Number != int64(ri+1) || r["status"].Text != "succeeded" || len(response) != 1 || response[0]["observation"].Number != obs[position] || op == "" || seen[op] || ops[id] != op {
				return fmt.Errorf("literal result/provenance mismatch at %d", position)
			}
			o, ok := outcome(s, op)
			if !ok || !o.ResponsePresent || o.Status != "succeeded" || o.Response["observation"].Number != obs[position] || o.Reason != r["reason"].Text || !consumed(s, op, subject) {
				return fmt.Errorf("original authenticated outcome/consumption mismatch")
			}
			seen[op] = true
			position++
		}
	}
	if len(ops) != len(got) {
		return fmt.Errorf("extra emitted work operation")
	}
	if p.Data["stop_reason"].Text != f.Stop || p.Data["work_count"].Number != f.Count || p.Data["rounds"].Number != int64(len(f.Rounds)) || !p.Data["complete"].Flag || len(rows(p, "remaining")) != 0 {
		return fmt.Errorf("completion envelope mismatch")
	}
	ch := f.Charges.Direct
	if nested {
		ch = f.Charges.Nested
	}
	if len(s.Roots) != 1 || s.Roots[0].Charges.Invocations != ch.Invocations || s.Roots[0].Charges.CreatedRuns != ch.Runs || s.Roots[0].Charges.Effects != effects || s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != "succeeded" {
		return fmt.Errorf("root charges/protocol mismatch")
	}
	// Each retained decision must contain exactly the next literal frontier.
	decisions := map[int64]bool{}
	for _, h := range s.Tables["history"] {
		if h["subject"] != subject {
			continue
		}
		q := decode[runtime.Projection](h, "successor")
		d := rows(q, "decision")
		if len(d) != 1 {
			continue
		}
		round := d[0]["round"].Number
		if decisions[round] {
			continue
		}
		want := []int64{}
		if int(round) < len(f.Rounds) {
			want = f.Rounds[round]
		}
		if !reflect.DeepEqual(valueIDs(d[0]["candidate"]), want) {
			return fmt.Errorf("decision %d differs", round)
		}
		decisions[round] = true
	}
	if len(decisions) != len(f.Rounds) {
		return fmt.Errorf("missing retained decision")
	}
	return origins(s)
}
func validAck(got, want boundary.Event, pid int, used map[string]bool) error {
	if got.PID != pid || got.Token != want.Token || got.Scenario != want.Scenario || got.Store != want.Store || got.Boundary != want.Boundary || got.Phase != want.Phase || used[got.Token] {
		return fmt.Errorf("foreign/reused acknowledgment")
	}
	if got.Boundary != "runner.waiting" && (got.Subject == "" || got.Generation != 1 || got.Sequence <= 0) {
		return fmt.Errorf("invalid source identity")
	}
	used[got.Token] = true
	return nil
}
func durableAck(s runtime.Inspection, e boundary.Event) error {
	if e.Boundary == "execution.committed" && e.Root != "" {
		return fmt.Errorf("execution hook does not emit a root identity")
	}
	if e.Boundary == "runner.waiting" {
		if e.Subject != "" || e.Sequence != 0 || e.Epoch != 0 {
			return fmt.Errorf("fabricated waiting source")
		}
		return nil
	}
	for _, h := range s.Tables["history"] {
		if h["subject"] != e.Subject || h["generation"] != e.Generation || h["sequence"] != e.Sequence {
			continue
		}
		if e.Boundary == "execution.committed" && h["epoch"] != e.Epoch {
			return fmt.Errorf("execution epoch mismatch")
		}
		if e.Boundary != "execution.committed" && e.Epoch != 0 {
			return fmt.Errorf("invocation hook epoch must be zero")
		}
		intents := decode[[]retainedIntent](h, "intents")
		first := ""
		if len(intents) > 0 {
			first = intents[0].ID
		}
		if first != e.Intent {
			return fmt.Errorf("source intent mismatch")
		}
		return nil
	}
	return fmt.Errorf("acknowledged source not retained")
}
func monitorControls(t *testing.T, s runtime.Inspection, e, want boundary.Event, pid int) {
	t.Helper()
	for name, change := range map[string]func(*boundary.Event){"pid": func(e *boundary.Event) { e.PID++ }, "token": func(e *boundary.Event) { e.Token += "foreign" }, "scenario": func(e *boundary.Event) { e.Scenario += "foreign" }, "store": func(e *boundary.Event) { e.Store += "foreign" }, "phase": func(e *boundary.Event) { e.Phase += "foreign" }, "subject": func(e *boundary.Event) { e.Subject += "foreign" }, "generation": func(e *boundary.Event) { e.Generation++ }, "sequence": func(e *boundary.Event) { e.Sequence++ }, "epoch": func(e *boundary.Event) { e.Epoch++ }, "intent": func(e *boundary.Event) { e.Intent += "foreign" }} {
		m := e
		change(&m)
		err := validAck(m, want, pid, map[string]bool{})
		if err == nil {
			err = durableAck(s, m)
		}
		if err == nil {
			t.Fatal("ACK mutant survived", name)
		}
		emit(t, map[string]any{"kind": "ack_control", "fault": name, "original": e, "mutant": m, "rejection": err.Error()})
	}
	used := map[string]bool{}
	must(t, validAck(e, want, pid, used))
	if validAck(e, want, pid, used) == nil {
		t.Fatal("reused ACK accepted")
	}
}
