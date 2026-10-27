package adaptive_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func plan(disposition, message string) effects.Candidate {
	return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"decision": schema.Text(disposition), "message": schema.Text(message), "tool": schema.Text(""), "query": schema.Text("")}}
}
func evidenceAdapters(disposition, message string) map[string]effects.Adapter {
	return map[string]effects.Adapter{evidence.ModelCapability: adapter{func(effects.Request) effects.Candidate { return plan(disposition, message) }}}
}

// Each row is the exact public helper contract. This ordinary leaf isolates
// succeeded diagnostics and response absence from the unchanged helper policy.
func publicEvidence(t *testing.T, disposition string, present, failed bool) author.Template {
	t.Helper()
	steps := []author.Step{author.Receive("public", map[string]string{"question": "question"})}
	if present {
		response := map[string]author.Expr{}
		for k, f := range evidence.ResultSchema() {
			response[k] = author.Value(schema.Default(f))
		}
		response["disposition"] = author.String(disposition)
		steps = append(steps, author.Send("public", response))
	}
	if failed {
		steps = append(steps, author.Failure())
	} else {
		steps = append(steps, author.Success())
	}
	a, e := author.Compile(author.Definition{Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: evidence.RequestSchema(), Send: evidence.ResultSchema()}}, Steps: steps})
	if e != nil {
		t.Fatal(e)
	}
	l, e := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
	if e != nil {
		t.Fatal(e)
	}
	c, e := adaptive.NewEvidence(l)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestAdaptiveEvidenceSummaryBinding(t *testing.T) {
	for _, n := range []int{6, 8} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			answer := strings.Repeat("\x00", 8192)
			x := execute(t, helper(t), false, 2, questions(n, "q"), evidenceAdapters("clarification", answer))
			assertSummary(t, x, "clarification", n, answer)
		})
	}
	for _, v := range []struct {
		disposition, stop string
		present, failed   bool
	}{{"answer", "exhausted", true, false}, {"no_match", "exhausted", true, false}, {"insufficient_evidence", "round_limit", true, false}, {"budget_exhausted", "invalid_observation", true, false}, {"invalid_result", "leaf_non_success", true, true}, {"answer", "leaf_non_success", false, false}} {
		t.Run(v.disposition+fmt.Sprint(v.present), func(t *testing.T) {
			q := questions(1, "q")
			q["max_rounds"] = schema.Int(1)
			x := execute(t, publicEvidence(t, v.disposition, v.present, v.failed), true, 1, q, nil)
			if x.data["stop_reason"].Text != v.stop {
				t.Fatal(x.data)
			}
			r := rows(t, x.data["results"])
			responses := rows(t, r[0]["responses"])
			if (len(responses) == 1) != v.present {
				t.Fatal("response absence changed", r)
			}
		})
	}
}
func assertSummary(t *testing.T, x execution, disposition string, n int, answer string) {
	t.Helper()
	r := rows(t, x.data["results"])
	if len(r) != n {
		t.Fatal("summary count", len(r))
	}
	for _, row := range r {
		v := rows(t, row["responses"])
		if len(v) != 1 || len(v[0]) != 1 || v[0]["disposition"].Text != disposition {
			t.Fatal("summary must bind only disposition", v)
		}
		found := false
		for _, retained := range x.inspection.Tables["composition_results"] {
			var outcome runtime.InvocationOutcome
			decode(t, retained["descriptor"], &outcome)
			if outcome.OperationID == row["operation"].Text {
				found = true
				if !outcome.ResponsePresent || outcome.Response["answer"].Text != answer || outcome.Response["disposition"].Text != disposition {
					t.Fatal("original helper response changed", outcome)
				}
			}
		}
		if !found {
			t.Fatal("original operation absent")
		}
	}
	for _, h := range x.inspection.Tables["history"] {
		var p runtime.Projection
		decode(t, h["successor"], &p)
		if _, adaptiveScope := p.Data["tokens"]; adaptiveScope {
			for name := range p.Data {
				if strings.Contains(name, "answer") || strings.Contains(name, "citation") || strings.Contains(name, "snippet") {
					t.Fatal("full helper response leaked into adaptive projection", name)
				}
			}
		}
	}
}

func TestControlledEvidenceAdaptive(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			var mu sync.Mutex
			plans := 0
			model := adapter{func(r effects.Request) effects.Candidate {
				if _, synthesis := r.Payload["snippet"]; synthesis {
					out := map[string]schema.Value{"answer": schema.Text("WAL records changes.")}
					for _, k := range []string{"document", "version", "snippet", "start", "end"} {
						out[k] = r.Payload[k]
					}
					return effects.Candidate{Disposition: "validated", Payload: out}
				}
				mu.Lock()
				plans++
				n := plans
				mu.Unlock()
				if n == 1 {
					return plan("clarification", "Clarify WAL")
				}
				c := plan("lookup", "")
				c.Payload["tool"] = schema.Text("lookup")
				c.Payload["query"] = schema.Text("WAL")
				return c
			}}
			lookup := adapter{func(effects.Request) effects.Candidate {
				return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"document": schema.Text("durability"), "version": schema.Text("v1"), "snippet": schema.Text("WAL records changes."), "start": schema.Int(0), "end": schema.Int(20)}}
			}}
			x := execute(t, helper(t), nested, 2, questions(1, "How does the retained WAL record relate to database changes?"), map[string]effects.Adapter{evidence.ModelCapability: model, evidence.LookupCapability: lookup})
			assertStop(t, x, "exhausted", 2, 2, []int64{})
			if !reflect.DeepEqual(ids(t, x.data["results"]), []int64{1, 4}) {
				t.Fatal("frozen evidence work trace")
			}
			r := rows(t, x.data["results"])
			if r[1]["question"].Text != r[0]["question"].Text || r[1]["parent_id"].Number != 1 || r[1]["edge"].Text != "check" {
				t.Fatal("child question policy")
			}
			enclosing := int64(1)
			if nested {
				enclosing = 2
			}
			charges := x.report.Roots[0].Charges
			if charges.Invocations != 2+enclosing || charges.CreatedRuns != 4+enclosing || charges.Effects != 4 {
				t.Fatal(charges)
			}
			for _, v := range x.inspection.Tables["composition_invocations"] {
				var d runtime.InvocationDescriptor
				decode(t, v["descriptor"], &d)
				limit := int64(2)
				if nested {
					limit = 3
				}
				if d.Depth > limit {
					t.Fatal("adapter wrapper added depth", d)
				}
			}
		})
	}
	// Known invalid model data must retain the helper's diagnostic and failure.
	x := execute(t, helper(t), false, 1, questions(1, "q"), map[string]effects.Adapter{evidence.ModelCapability: adapter{func(effects.Request) effects.Candidate { return effects.Candidate{Disposition: "schema_invalid"} }}})
	assertStop(t, x, "leaf_non_success", 1, 1, []int64{})
	r := rows(t, x.data["results"])
	if r[0]["status"].Text != "failed" || rows(t, r[0]["responses"])[0]["disposition"].Text != "invalid_result" {
		t.Fatal("diagnostic converted to success", r)
	}
}

type bytePeak struct {
	Bytes                                int
	Subject, TransitionAddress, Variable string
	Generation, Epoch, HistorySequence   int64
	SuccessorPC                          int
}

func TestAdaptiveWorstCaseBytes(t *testing.T) {
	c := helper(t)
	for _, nested := range []bool{false, true} {
		for _, mode := range []string{"sixteen", "clarification6", "clarification8", "maximum-answer"} {
			t.Run(fmt.Sprintf("nested=%v/%s", nested, mode), func(t *testing.T) {
				n := 8
				q := strings.Repeat("\x00", 512)
				disposition := "insufficient_evidence"
				answer := strings.Repeat("\x00", 8192)
				adapters := evidenceAdapters(disposition, answer)
				if strings.HasPrefix(mode, "clarification") {
					q = "q"
					disposition = "clarification"
					adapters = evidenceAdapters(disposition, answer)
					if mode == "clarification6" {
						n = 6
					}
				}
				if mode == "maximum-answer" {
					n = 1
					disposition = "answer"
					adapters = maximumAnswerAdapters()
				}
				x := execute(t, c, nested, 2, questions(n, q), adapters)
				if mode == "sixteen" {
					assertStop(t, x, "duplicate_work_id", 8, 1, []int64{4, 5, 7, 8, 10, 11, 13, 14, 16, 17, 19, 20, 22, 23, 25, 26})
				}
				assertSummary(t, x, disposition, n, answer)
				measure(t, x)
			})
		}
	}
	// Scalar payloads exercise full int64 encoding without arithmetic on payloads.
	for _, value := range []int64{-9223372036854775808, 9223372036854775807} {
		x := execute(t, scalar(t, false), true, 1, request(value), nil)
		measure(t, x)
	}
}
func maximumAnswerAdapters() map[string]effects.Adapter {
	return map[string]effects.Adapter{
		evidence.ModelCapability: adapter{func(r effects.Request) effects.Candidate {
			if _, ok := r.Payload["snippet"]; ok {
				p := map[string]schema.Value{"answer": schema.Text(strings.Repeat("\x00", 8192))}
				for _, k := range []string{"document", "version", "snippet", "start", "end"} {
					p[k] = r.Payload[k]
				}
				return effects.Candidate{Disposition: "validated", Payload: p}
			}
			c := plan("lookup", "")
			c.Payload["tool"] = schema.Text("lookup")
			c.Payload["query"] = schema.Text("q")
			return c
		}},
		evidence.LookupCapability: adapter{func(effects.Request) effects.Candidate {
			return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"document": schema.Text(strings.Repeat("\x00", 512)), "version": schema.Text(strings.Repeat("\x00", 64)), "snippet": schema.Text(strings.Repeat("\x00", 2048)), "start": schema.Int(-9223372036854775808), "end": schema.Int(9223372036854775807)}}
		}},
	}
}
func measure(t *testing.T, x execution) {
	t.Helper()
	peaks := map[string]bytePeak{}
	variables := map[string]bytePeak{}
	adaptiveKeys := map[string]bool{}
	allowedLists := map[string]bool{}
	for _, k := range []string{"tokens", "seeds", "current", "next", "handles", "history", "round_results", "results", "responses", "decision", "derive_responses"} {
		allowedLists[k] = true
	}
	var transition bytePeak
	var visit func(string, schema.Value)
	visit = func(name string, v schema.Value) {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		if len(b) > 1048576 {
			t.Fatal("native value bound", name, len(b))
		}
		if v.Kind == "list" {
			r := rows(t, v)
			a, _ := json.Marshal(r)
			if len(a) > 262144 {
				t.Fatal("canonical array bound", name, len(a))
			}
			if len(a) > variables[name].Bytes {
				peak := transition
				peak.Bytes = len(a)
				peak.Variable = name
				variables[name] = peak
			}
			for _, row := range r {
				for k, child := range row {
					visit(name+"."+k, child)
				}
			}
		}
	}
	scopes := map[string]string{}
	for _, r := range x.inspection.Tables["runs"] {
		var p runtime.Projection
		decode(t, r["projection"], &p)
		subject := fmt.Sprint(r["subject"])
		scope := "helper"
		if subject == "root" {
			scope = "root"
		} else if _, ok := p.Data["tokens"]; ok {
			scope = "adaptive"
		} else if _, ok := p.Data["request_seeds"]; ok {
			scope = "wrapper"
		}
		scopes[subject] = scope
	}
	allowedKeys := map[string]bool{}
	for _, k := range strings.Fields("active check_id child_id complete current decision derive_responses derive_status duplicate edge empty_frontier failure_reason handle_id handles hard history history_id invalid_observation leaf_observation leaf_present leaf_reason leaf_status max_rounds max_work next observation other_id other_position overflow parent_id pending position question responses results round round_results rounds seeds stop_reason tokens trial valid_id valid_ids valid_response valid_rounds valid_work value work_count work_id") {
		allowedKeys[k] = true
	}

	for _, r := range x.inspection.Tables["history"] {
		var p runtime.Projection
		decode(t, r["successor"], &p)
		subject, _ := r["subject"].(string)
		transition = retainedTransition(t, r, p.PC)
		scope := scopes[subject]
		limit := 1048576
		switch scope {
		case "adaptive":
			limit = 621 * 1024
		case "root", "wrapper":
			limit = 512 * 1024
		}

		b, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		if len(b) > limit {
			t.Fatal("complete projection bound", scope, len(b), limit)
		}
		if len(b) > peaks[scope].Bytes {
			peak := transition
			peak.Bytes = len(b)
			peaks[scope] = peak
		}
		dataCursors := 0
		for _, cursor := range p.Cursors {
			visit("cursor", cursor.Snapshot)
			records := rows(t, cursor.Snapshot)
			if len(records) > 0 {
				if _, control := records[0]["round"]; !(control && len(records[0]) == 1) {
					dataCursors++
				}
			}
		}
		if scope == "adaptive" && dataCursors > 3 {
			t.Fatal("extra data-bearing cursor snapshots", dataCursors)
		}
		if scope == "adaptive" && len(p.Data) > 256 {
			t.Fatal("adaptive named-key inventory")
		}
		for k, v := range p.Data {
			if scope == "adaptive" {
				adaptiveKeys[k] = true
				if !allowedKeys[k] {
					t.Fatal("additional live variable outside frozen inventory", k)
				}
				if len(k) > 64 {
					t.Fatal("key length")
				}
				if v.Kind == "list" && !allowedLists[k] {
					t.Fatal("extra live list outside contract inventory", k)
				}
			}
			visit(scope+"."+k, v)
		}
		raw, ok := r["intents"].(string)
		if !ok || len(raw) > 1048576 {
			t.Fatal("complete intent array bound")
		}
	}
	for _, v := range x.inspection.Tables["observations"] {
		raw, ok := v["descriptor"].(string)
		if !ok || len(raw) > 1048576 {
			t.Fatal("observation bound")
		}
	}
	for _, v := range x.inspection.Tables["programs"] {
		raw, ok := v["bytes"].(string)
		if !ok {
			raw, ok = v["artifact"].(string)
		}
		if !ok {
			t.Fatal("missing actual artifact bytes", v)
		}
		if len(raw) > 1048576 {
			t.Fatal("artifact bound")
		}
		if len(raw) > peaks["artifact"].Bytes {
			peaks["artifact"] = bytePeak{Bytes: len(raw), Subject: fmt.Sprint(v["id"])}
		}
	}
	for _, v := range x.inspection.Tables["composition_results"] {
		var o runtime.InvocationOutcome
		decode(t, v["descriptor"], &o)
		if len(o.OperationID) >= 128 || len(o.Reason) > 64 || !ascii(o.OperationID) || !ascii(o.Reason) {
			t.Fatal("reachable producer assumption", o.OperationID, o.Reason)
		}
	}
	keys := []string{}
	for k := range adaptiveKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raw, _ := json.Marshal(struct {
		Scopes       map[string]bytePeak
		Variables    map[string]bytePeak
		AdaptiveKeys []string
	}{peaks, variables, keys})
	t.Log("ADAPTIVE_BYTES " + string(raw))
}
func ascii(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// This additional witness was authored after the broad9b180f59 run. It keeps a
// previous decision live while the next round scans two full cursor snapshots.
func TestAdaptiveWorstCaseBytesPreviousDecision(t *testing.T) {
	component := helper(t)
	input, _, err := component.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	old := input.Record
	seeds := old["seeds"]
	question := seeds.Element["question"]
	question.MaxBytes = 4096
	seeds.Element["question"] = question
	old["seeds"] = seeds
	if err := old.Validate(questions(8, strings.Repeat("\x00", 4096))); err != nil {
		t.Fatal("historical eight-seed request should fit old proposed schema", err)
	}
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprintf("nested=%v", nested), func(t *testing.T) {
			q := strings.Repeat("\x00", 512)
			answer := strings.Repeat("\x00", 8192)
			req := questions(4, q)
			req["seeds"] = schema.List(
				map[string]schema.Value{"work_id": schema.Int(5), "question": schema.Text(q)},
				map[string]schema.Value{"work_id": schema.Int(6), "question": schema.Text(q)},
				map[string]schema.Value{"work_id": schema.Int(7), "question": schema.Text(q)},
				map[string]schema.Value{"work_id": schema.Int(8), "question": schema.Text(q)},
			)
			var mu sync.Mutex
			count := 0
			model := adapter{func(effects.Request) effects.Candidate {
				mu.Lock()
				count++
				n := count
				mu.Unlock()
				if n <= 4 {
					return plan("clarification", answer)
				}
				return plan("insufficient_evidence", answer)
			}}
			x := execute(t, component, nested, 2, req, map[string]effects.Adapter{evidence.ModelCapability: model})
			assertStop(t, x, "work_limit", 8, 2, []int64{49, 50, 58, 59, 67, 68, 76, 77})
			if x.report.Roots[0].Outcome.Status != "succeeded" {
				t.Fatal("work-bound application stop must succeed at protocol layer")
			}
			if !reflect.DeepEqual(ids(t, x.data["results"]), []int64{5, 6, 7, 8, 16, 19, 22, 25}) {
				t.Fatal("literal two-round work order")
			}
			results := rows(t, x.data["results"])
			for i, r := range results {
				disposition := "clarification"
				if i >= 4 {
					disposition = "insufficient_evidence"
				}
				response := rows(t, r["responses"])
				if len(response) != 1 || len(response[0]) != 1 || response[0]["disposition"].Text != disposition || r["question"].Text != q {
					t.Fatal("two-round summary/payload mismatch", i)
				}
				found := false
				for _, retained := range x.inspection.Tables["composition_results"] {
					var outcome runtime.InvocationOutcome
					decode(t, retained["descriptor"], &outcome)
					if outcome.OperationID == r["operation"].Text {
						found = true
						if outcome.Response["answer"].Text != answer {
							t.Fatal("full original response changed")
						}
					}
				}
				if !found {
					t.Fatal("original operation missing")
				}
			}
			for i, r := range rows(t, x.data["remaining"]) {
				parents := []int64{16, 16, 19, 19, 22, 22, 25, 25}
				edges := []string{"left", "right", "left", "right", "left", "right", "left", "right"}
				if r["question"].Text != q || r["parent_id"].Number != parents[i] || r["edge"].Text != edges[i] || r["round"].Number != 3 {
					t.Fatal("full remaining question/provenance was altered", i)
				}
			}
			enclosing := int64(1)
			if nested {
				enclosing = 2
			}
			charges := x.report.Roots[0].Charges
			if charges.Invocations != 8+enclosing || charges.CreatedRuns != 16+enclosing || charges.Effects != 8 {
				t.Fatal("two-round evidence charges", charges)
			}
			var witness bytePeak
			for _, h := range x.inspection.Tables["history"] {
				var p runtime.Projection
				decode(t, h["successor"], &p)
				if _, ok := p.Data["tokens"]; !ok {
					continue
				}
				if p.Data["rounds"].Number != 2 || len(rows(t, p.Data["current"])) != 4 || len(rows(t, p.Data["next"])) != 8 || len(rows(t, p.Data["decision"])) != 1 || len(rows(t, p.Data["round_results"])) != 4 || len(rows(t, p.Data["results"])) != 8 || len(rows(t, p.Data["handles"])) != 4 || len(p.Cursors) != 3 {
					continue
				}
				decision := rows(t, p.Data["decision"])[0]
				if decision["round"].Number != 1 || !reflect.DeepEqual(ids(t, decision["candidate"]), []int64{16, 19, 22, 25}) {
					continue
				}
				dataSnapshots := 0
				for _, cursor := range p.Cursors {
					r := rows(t, cursor.Snapshot)
					if len(r) > 0 && len(r[0]) > 1 {
						dataSnapshots++
					}
				}
				if dataSnapshots != 2 {
					continue
				}
				b, err := json.Marshal(p)
				if err != nil {
					t.Fatal(err)
				}
				if len(b) > witness.Bytes {
					witness = retainedTransition(t, h, p.PC)
					witness.Bytes = len(b)
				}
			}
			if witness.Bytes == 0 {
				t.Fatal("no actual complete projection retained previous decision plus current/next/round+total/handles and two active data snapshots")
			}
			raw, _ := json.Marshal(witness)
			t.Log("ADAPTIVE_SIMULTANEOUS " + string(raw))
			measure(t, x)
		})
	}
}

func retainedTransition(t *testing.T, row map[string]any, successorPC int) bytePeak {
	t.Helper()
	value := func(name string) int64 {
		n, ok := row[name].(int64)
		if !ok {
			t.Fatalf("retained %s is not int64: %T", name, row[name])
		}
		return n
	}
	p := bytePeak{Subject: fmt.Sprint(row["subject"]), Generation: value("generation"), Epoch: value("epoch"), HistorySequence: value("sequence"), SuccessorPC: successorPC}
	p.TransitionAddress = fmt.Sprintf("%s/generation=%d/epoch=%d/history=%d", p.Subject, p.Generation, p.Epoch, p.HistorySequence)
	return p
}
