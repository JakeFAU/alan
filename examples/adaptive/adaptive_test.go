package adaptive_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/examples/worktree"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type adapter struct {
	result func(effects.Request) effects.Candidate
}

func (a adapter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Freeze(effects.Prepared{Request: r, Protocol: "controlled", MaxResponseBytes: 131072}), nil
}
func (a adapter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	c := a.result(p.Request)
	c.ID = p.Request.ID
	c.Key = p.Request.Key
	return c
}
func scalarAdapter(paired bool) effects.Adapter {
	return adapter{func(r effects.Request) effects.Candidate {
		n := r.Payload["value"].Number
		if paired && n == 2 {
			n = 0
		}
		return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"observation": schema.Int(n)}}
	}}
}
func request(values ...int64) map[string]schema.Value {
	rows := []map[string]schema.Value{}
	for i, v := range values {
		rows = append(rows, map[string]schema.Value{"work_id": schema.Int(int64(i + 1)), "value": schema.Int(v)})
	}
	return map[string]schema.Value{"seeds": schema.List(rows...), "max_rounds": schema.Int(3), "max_work": schema.Int(8)}
}

type execution struct {
	data       map[string]schema.Value
	report     runtime.RunReport
	inspection runtime.Inspection
	path       string
}

func execute(t *testing.T, component author.Template, nested bool, capacity int, req map[string]schema.Value, adapters map[string]effects.Adapter, limits ...runtime.ResourceLimits) execution {
	t.Helper()
	l := adaptive.DefaultLimits(int64(capacity))
	if len(limits) > 0 {
		l = limits[0]
	}
	d, err := adaptive.New(component, nested, l)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "adaptive.sqlite")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admitted, err := adaptive.Submit(t.Context(), s, "work", req)
	if err != nil || !admitted.Accepted {
		t.Fatal(admitted, err)
	}
	report, err := s.Run(t.Context(), runtime.RunOptions{Adapters: adapters, MaxSteps: 20000, MaxDispatch: 2000, MaxInFlight: capacity})
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return execution{state.Projection.Data, report, inspection, path}
}
func scalar(t *testing.T, controlled bool) author.Template {
	t.Helper()
	l, e := adaptive.ScalarLeaf(controlled)
	if e != nil {
		t.Fatal(e)
	}
	c, e := adaptive.NewScalar(l)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func rows(t *testing.T, v schema.Value) []map[string]schema.Value {
	t.Helper()
	r, e := v.Records()
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func ids(t *testing.T, v schema.Value) []int64 {
	out := []int64{}
	for _, r := range rows(t, v) {
		out = append(out, r["work_id"].Number)
	}
	return out
}
func assertStop(t *testing.T, x execution, reason string, count, rounds int64, remaining []int64) {
	t.Helper()
	if x.data["stop_reason"].Text != reason || x.data["work_count"].Number != count || x.data["rounds"].Number != rounds || !reflect.DeepEqual(ids(t, x.data["remaining"]), remaining) {
		t.Fatalf("stop mismatch: reason=%v count=%v rounds=%v remaining=%v report=%+v", x.data["stop_reason"], x.data["work_count"], x.data["rounds"], ids(t, x.data["remaining"]), x.report)
	}
	if x.data["complete"].Flag != (reason == "exhausted") {
		t.Fatal("incorrect application completion")
	}
	if len(x.report.Roots) != 1 || x.report.Roots[0].Outcome == nil {
		t.Fatal("root incomplete", x.report)
	}
}

func TestScalarAdaptivePairedResults(t *testing.T) {
	c := scalar(t, true)
	for _, nested := range []bool{false, true} {
		for _, paired := range []bool{false, true} {
			t.Run(fmt.Sprintf("nested=%v/paired=%v", nested, paired), func(t *testing.T) {
				x := execute(t, c, nested, 2, request(0, 1, 2), map[string]effects.Adapter{adaptive.ScalarCapability: scalarAdapter(paired)})
				expected := []int64{1, 2, 3, 7, 10, 11, 31}
				parents := []int64{0, 0, 0, 2, 3, 3, 10}
				values := []int64{0, 1, 2, 0, 1, 0, 0}
				round := []int64{1, 1, 1, 2, 2, 2, 3}
				edges := []string{"seed", "seed", "seed", "check", "left", "right", "check"}
				observations := []int64{0, 1, 2, 0, 1, 0, 0}
				if paired {
					expected = expected[:4]
					parents = parents[:4]
					values = values[:4]
					round = round[:4]
					edges = edges[:4]
					observations = []int64{0, 1, 0, 0}
				}
				r := rows(t, x.data["results"])
				if !reflect.DeepEqual(ids(t, x.data["results"]), expected) {
					t.Fatal("frozen work IDs", ids(t, x.data["results"]))
				}
				ops := map[string]bool{}
				for i, v := range r {
					response := rows(t, v["responses"])
					op := v["operation"].Text
					if v["parent_id"].Number != parents[i] || v["value"].Number != values[i] || v["round"].Number != round[i] || v["edge"].Text != edges[i] || v["status"].Text != "succeeded" || len(response) != 1 || response[0]["observation"].Number != observations[i] || op == "" || ops[op] {
						t.Fatal("frozen row mismatch", i, v)
					}
					ops[op] = true
				}
				assertStop(t, x, "exhausted", int64(len(expected)), round[len(round)-1], []int64{})
				enclosing := int64(1)
				if nested {
					enclosing = 2
				}
				charges := x.report.Roots[0].Charges
				if charges.Invocations != int64(len(expected))+enclosing || charges.CreatedRuns != int64(len(expected))+enclosing || charges.Effects != int64(len(expected)) {
					t.Fatal(charges)
				}
			})
		}
	}
}
func TestScalarAdaptiveValidationAndBounds(t *testing.T) {
	c := scalar(t, false)
	cases := []struct {
		name          string
		request       map[string]schema.Value
		stop          string
		count, rounds int64
		remaining     []int64
	}{
		{"empty", request(), "exhausted", 0, 0, []int64{}}, {"all-zero", request(0, 0, 0), "exhausted", 3, 1, []int64{}},
		{"eight-equal", request(0, 0, 0, 0, 0, 0, 0, 0), "exhausted", 8, 1, []int64{}},
		{"sixteen", request(2, 2, 2, 2, 2, 2, 2, 2), "duplicate_work_id", 8, 1, []int64{4, 5, 7, 8, 10, 11, 13, 14, 16, 17, 19, 20, 22, 23, 25, 26}},
		{"collision", request(1, 0, 0, 0), "duplicate_work_id", 4, 1, []int64{4}},
	}
	for _, n := range []int64{0, 4} {
		q := request(0)
		q["max_rounds"] = schema.Int(n)
		cases = append(cases, struct {
			name          string
			request       map[string]schema.Value
			stop          string
			count, rounds int64
			remaining     []int64
		}{fmt.Sprint("rounds", n), q, "invalid_request", 0, 0, []int64{1}})
	}
	for _, n := range []int64{0, 9} {
		q := request(0)
		q["max_work"] = schema.Int(n)
		cases = append(cases, struct {
			name          string
			request       map[string]schema.Value
			stop          string
			count, rounds int64
			remaining     []int64
		}{fmt.Sprint("work", n), q, "invalid_request", 0, 0, []int64{1}})
	}
	for _, n := range []int64{0, 9} {
		q := request(0)
		q["seeds"] = schema.List(map[string]schema.Value{"work_id": schema.Int(n), "value": schema.Int(0)})
		cases = append(cases, struct {
			name          string
			request       map[string]schema.Value
			stop          string
			count, rounds int64
			remaining     []int64
		}{fmt.Sprint("id", n), q, "invalid_request", 0, 0, []int64{n}})
	}
	q := request(0)
	q["seeds"] = schema.List(map[string]schema.Value{"work_id": schema.Int(1), "value": schema.Int(0)}, map[string]schema.Value{"work_id": schema.Int(1), "value": schema.Int(1)})
	cases = append(cases, struct {
		name          string
		request       map[string]schema.Value
		stop          string
		count, rounds int64
		remaining     []int64
	}{"duplicate", q, "duplicate_work_id", 0, 0, []int64{1, 1}})
	for _, v := range []struct {
		name              string
		rounds, work      int64
		count, roundCount int64
		remaining         []int64
		stop              string
	}{{"no-prefix", 3, 5, 3, 1, []int64{7, 10, 11}, "work_limit"}, {"round1", 1, 8, 3, 1, []int64{7, 10, 11}, "round_limit"}, {"both", 1, 3, 3, 1, []int64{7, 10, 11}, "work_limit"}, {"round2", 2, 8, 6, 2, []int64{31}, "round_limit"}, {"exact", 3, 7, 7, 3, []int64{}, "exhausted"}, {"seed-limit", 3, 2, 0, 0, []int64{1, 2, 3}, "work_limit"}} {
		q := request(0, 1, 2)
		q["max_rounds"] = schema.Int(v.rounds)
		q["max_work"] = schema.Int(v.work)
		cases = append(cases, struct {
			name          string
			request       map[string]schema.Value
			stop          string
			count, rounds int64
			remaining     []int64
		}{v.name, q, v.stop, v.count, v.roundCount, v.remaining})
	}
	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			x := execute(t, c, false, 1, v.request, nil)
			assertStop(t, x, v.stop, v.count, v.rounds, v.remaining)
			if len(x.inspection.Tables["composition_invocations"]) != int(v.count)+1 {
				t.Fatal("unexpected affected frontier admission")
			}
		})
	}
}

func TestAdaptiveArtifactReuse(t *testing.T) {
	c := scalar(t, false)
	in, out, e := c.Schemas()
	if e != nil {
		t.Fatal(e)
	}
	for _, nested := range []bool{false, true} {
		d, e := adaptive.New(c, nested, adaptive.DefaultLimits(2))
		if e != nil {
			t.Fatal(e)
		}
		ci, co, _ := d.Adaptive.Schemas()
		if d.Adaptive.ID() != c.ID() || ci.ID != in.ID || co.ID != out.ID {
			t.Fatal("artifact/schema changed")
		}
		for _, q := range []map[string]schema.Value{request(), request(0, 1, 2), request(0, 0, 0, 0, 0, 0, 0, 0)} {
			x := execute(t, c, nested, 2, q, nil)
			maxDepth := 0
			for _, r := range x.inspection.Tables["composition_invocations"] {
				var desc struct {
					Depth      int
					TemplateID string
				}
				decode(t, r["descriptor"], &desc)
				limit := 2
				if nested {
					limit = 3
				}
				if desc.Depth > maxDepth {
					maxDepth = desc.Depth
				}
				if desc.Depth > limit {
					t.Fatal("depth", desc)
				}
			}
		}
		expectedTemplates := 2
		if nested {
			expectedTemplates = 3
		}
		if len(d.Provision.Composition.Templates) != expectedTemplates {
			t.Fatal("unexpected round-template invocation")
		}

		for _, a := range d.Provision.Programs {
			if len(a.Bytes) > 1048576 {
				t.Fatal("program bytes", len(a.Bytes))
			}
			t.Logf("artifact=%s bytes=%d", a.ID, len(a.Bytes))
		}
	}
}
func decode(t *testing.T, raw any, out any) {
	t.Helper()
	s, ok := raw.(string)
	if !ok {
		t.Fatalf("expected retained JSON, got %T", raw)
	}
	if e := json.Unmarshal([]byte(s), out); e != nil {
		t.Fatal(e)
	}
}
func helper(t *testing.T) author.Template {
	t.Helper()
	l, e := worktree.Evidence(evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
	if e != nil {
		t.Fatal(e)
	}
	c, e := adaptive.NewEvidence(l)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func questions(n int, q string) map[string]schema.Value {
	r := []map[string]schema.Value{}
	for i := 0; i < n; i++ {
		r = append(r, map[string]schema.Value{"work_id": schema.Int(int64(i + 1)), "question": schema.Text(q)})
	}
	return map[string]schema.Value{"seeds": schema.List(r...), "max_rounds": schema.Int(3), "max_work": schema.Int(8)}
}
func TestAdaptiveEvidenceProfileRejection(t *testing.T) {
	c := helper(t)
	in, _, _ := c.Schemas()
	for _, n := range []int{512, 513, 4096} {
		q := questions(8, strings.Repeat("\x00", n))
		valid := in.Record.Validate(q) == nil
		if valid != (n == 512) {
			t.Fatal("raw byte schema boundary", n)
		}
		if n == 4096 {
			old := evidence.RequestSchema()
			if old.Validate(map[string]schema.Value{"question": schema.Text(strings.Repeat("\x00", n))}) != nil {
				t.Fatal("historical helper schema changed")
			}
		}
		if n == 512 {
			continue
		}
		for _, nested := range []bool{false, true} {
			d, e := adaptive.New(c, nested, adaptive.DefaultLimits(1))
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "reject.sqlite")
			s, e := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
			if e != nil {
				t.Fatal(e)
			}
			a, e := adaptive.Submit(t.Context(), s, "invalid", q)
			s.Close()
			if e != nil || a.Accepted {
				t.Fatal(a, e)
			}
			i, e := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if e != nil || len(i.Tables["composition_invocations"]) != 0 {
				t.Fatal("item work before schema validation", e)
			}
		}
	}
}
func TestAdaptiveWrapperOutcomeMapping(t *testing.T) {
	for _, nested := range []bool{false, true} {
		c := scalar(t, false)
		q := request(1)
		q["max_rounds"] = schema.Int(1)
		x := execute(t, c, nested, 1, q, nil)
		assertStop(t, x, "round_limit", 1, 1, []int64{4})
		if x.report.Roots[0].Outcome.Status != "succeeded" {
			t.Fatal("bounded stop failed protocol")
		}
		c = scalar(t, true)
		a := adapter{func(r effects.Request) effects.Candidate { return effects.Candidate{Disposition: "schema_invalid"} }}
		x = execute(t, c, nested, 1, request(0), map[string]effects.Adapter{adaptive.ScalarCapability: a})
		assertStop(t, x, "leaf_non_success", 1, 1, []int64{})
		if x.report.Roots[0].Outcome.Status != "failed" {
			t.Fatal("failed child mapped to success")
		}
		r := rows(t, x.data["results"])
		if r[0]["status"].Text != "failed" || len(rows(t, r[0]["responses"])) != 0 {
			t.Fatal("missing outcome fabricated", r)
		}
	}
}
func TestScalarExtremesAndInvalidObservation(t *testing.T) {
	for _, v := range []int64{math.MinInt64, math.MaxInt64, -1, 3} {
		x := execute(t, scalar(t, false), false, 1, request(v), nil)
		assertStop(t, x, "invalid_observation", 1, 1, []int64{})
		r := rows(t, x.data["results"])
		if rows(t, r[0]["responses"])[0]["observation"].Number != v {
			t.Fatal("observation truncated")
		}
	}
}

func TestAdaptiveFirstFailureAndRootBudget(t *testing.T) {
	for _, nested := range []bool{false, true} {
		limits := adaptive.DefaultLimits(1)
		limits.MaxInvocations = 3
		if nested {
			limits.MaxInvocations = 4
		}
		x := execute(t, scalar(t, true), nested, 1, request(0, 1, 2), map[string]effects.Adapter{adaptive.ScalarCapability: scalarAdapter(false)}, limits)
		assertStop(t, x, "root_budget_denied", 3, 1, []int64{7})
		r := rows(t, x.data["results"])
		if r[2]["status"].Text != "rejected" || r[2]["reason"].Text != "invocation_limit" || len(rows(t, r[2]["responses"])) != 0 {
			t.Fatal("budget denial correspondence", r)
		}
		a := adapter{func(r effects.Request) effects.Candidate {
			if r.Payload["value"].Number == 0 {
				return effects.Candidate{Disposition: "protocol_invalid"}
			}
			return effects.Candidate{Disposition: "validated", Payload: map[string]schema.Value{"observation": r.Payload["value"]}}
		}}
		x = execute(t, scalar(t, true), nested, 2, request(0, 1, 2), map[string]effects.Adapter{adaptive.ScalarCapability: a})
		assertStop(t, x, "leaf_non_success", 3, 1, []int64{7, 10, 11})
		if len(rows(t, x.data["results"])) != 3 {
			t.Fatal("failure skipped sibling evidence")
		}
	}
}
