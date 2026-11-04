package author_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func seriesDefinition() author.Definition {
	return author.Definition{Name: "sample", Inputs: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 64}}}}, Steps: []author.Step{
		author.Request("operation", "read", map[string]author.Expr{"station": author.Var("station")}, nil),
		author.Await("read", "operation", map[string]string{"sample": "sample"}), author.Success(),
	}}
}
func seriesOptions(n int) author.SeriesOptions {
	return author.SeriesOptions{ID: "sample", Count: n, First: author.ApproveAtProvision, Inputs: map[string]schema.Value{"station": schema.Text("north")}, ApprovalReasons: []string{"observe again"}}
}
func TestBoundedSeriesConstruction(t *testing.T) {
	d := seriesDefinition()
	original, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{1, 8, 32, 256} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, err := author.BoundedSeries(d, seriesOptions(n))
			if err != nil {
				t.Fatal(err)
			}
			p, err := s.Provision()
			if err != nil {
				t.Fatal(err)
			}
			if s.ID() != "sample" || s.Count() != n || len(p.Runs) != n || len(p.Composition.Roots) != n || len(p.Composition.Continuations) != n-1 {
				t.Fatal("wrong finite expansion")
			}
			run, _ := s.Run(1)
			if run != (runtime.RunID{Subject: "sample-1", Generation: 1}) {
				t.Fatal(run)
			}
			for _, r := range p.Runs {
				if r.ID == run && r.ProgramID != original.ID() {
					t.Fatal("first artifact changed")
				}
			}
			if _, err := s.Run(0); err == nil {
				t.Fatal("zero index")
			}
			if _, err := s.Run(n + 1); err == nil {
				t.Fatal("out of range")
			}
		})
	}
	for _, n := range []int{0, -1, math.MaxInt} {
		if _, err := author.BoundedSeries(d, seriesOptions(n)); err == nil {
			t.Fatalf("accepted count %d", n)
		}
	}
	options := seriesOptions(2)
	options.First = 0
	if _, err := author.BoundedSeries(d, options); err == nil {
		t.Fatal("implicit first authorization")
	}
	for _, id := range []string{"", string([]byte{0xff})} {
		options = seriesOptions(2)
		options.ID = id
		if _, err := author.BoundedSeries(d, options); err == nil {
			t.Fatal("invalid id")
		}
	}
	var zero author.Series
	if _, err := zero.Provision(); err == nil {
		t.Fatal("zero provision")
	}
	if _, err := zero.Run(1); err == nil {
		t.Fatal("zero run")
	}
	if _, err := zero.Inspect(nil); err == nil {
		t.Fatal("zero inspect")
	}
}
func TestBoundedSeriesRejectsSkippedConsumptionAndRepeatedOperations(t *testing.T) {
	base := seriesDefinition()
	cases := map[string][]author.Step{
		"skip await":       {base.Steps[0], author.Goto("done"), base.Steps[1], author.Label("done"), author.Success()},
		"conditional skip": {base.Steps[0], author.IfEqual(author.String("a"), author.String("a"), []author.Step{author.Success()}, []author.Step{base.Steps[1], author.Success()})},
		"repeat once":      {author.Repeat(1, base.Steps[0], base.Steps[1]), author.Success()},
		"two alternatives": {author.IfEqual(author.String("a"), author.String("a"), base.Steps, base.Steps)},
		"no await":         {base.Steps[0], author.Success()},
	}
	for name, steps := range cases {
		t.Run(name, func(t *testing.T) {
			d := base
			d.Steps = steps
			if _, err := author.BoundedSeries(d, seriesOptions(2)); err == nil {
				t.Fatal("accepted non-observation")
			}
		})
	}
	d := base
	d.Ports = map[string]author.Port{"unused": author.ContinuationPort("other")}
	if _, err := author.BoundedSeries(d, seriesOptions(2)); err == nil {
		t.Fatal("extra port")
	}
}
func TestBoundedSeriesOwnsDeclaration(t *testing.T) {
	d := seriesDefinition()
	o := seriesOptions(2)
	s, err := author.BoundedSeries(d, o)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.Provision()
	o.Inputs["station"] = schema.Text("south")
	o.ApprovalReasons[0] = "changed"
	d.Inputs["station"] = schema.Field{Kind: "int"}
	d.Steps[0] = author.Success()
	p, _ := s.Provision()
	p.Runs[0].Projection.Data["station"] = schema.Text("west")
	p.Programs[0].Bytes[0] = 'x'
	p.Composition.Roots[0].Capabilities[0] = "other"
	after, _ := s.Provision()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("declaration aliased caller data")
	}
}

type seriesMeter struct{ calls int }

func (m *seriesMeter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "sample/v1", Endpoint: "local-meter", Wire: r.Payload["station"].Text, MaxResponseBytes: 256}, nil
}
func (m *seriesMeter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}}
}
func TestBoundedSeriesProtectedExecution(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(3))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.Provision()
	path := filepath.Join(t.TempDir(), "series.db")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	cut := func() *runtime.ExecutionSnapshot {
		v, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	initial, err := s.Inspect(cut())
	if err != nil {
		t.Fatal(err)
	}
	if initial.Positions[0].State != author.SeriesAwaitingRequest || initial.Positions[1].State != author.SeriesNotEligible {
		t.Fatal(initial)
	}
	meter := new(seriesMeter)
	run := func() {
		_, err := owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 64, MaxDispatch: 16, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"meter": meter}})
		if err != nil {
			t.Fatal(err)
		}
	}
	run()
	if meter.calls != 1 {
		t.Fatal(meter.calls)
	}
	snapshot := cut()
	v, err := s.Inspect(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if v.Completed != 1 || v.Selected != 1 || v.Spent != 1 || v.Pending != 2 || v.Unresolved != 0 || v.BoundExhausted || v.Positions[1].State != author.SeriesAwaitingApproval {
		t.Fatal(v)
	}
	third, _ := s.Run(3)
	if _, err := snapshot.ContinuationBasis(third); err == nil {
		t.Fatal("third eligible before second paid")
	}
	second, _ := s.Run(2)
	basis, err := snapshot.ContinuationBasis(second)
	if err != nil {
		t.Fatal(err)
	}
	first, err := owner.ApproveContinuation(t.Context(), basis, "observe again")
	if err != nil || !first.Accepted {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := owner.ApproveContinuation(t.Context(), basis, "observe again")
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatalf("duplicate changed decision: %+v %v", again, err)
	}
	run()
	run()
	if meter.calls != 2 {
		t.Fatal(meter.calls)
	}
	v, err = s.Inspect(cut())
	if err != nil || v.Completed != 2 || v.Spent != 2 {
		t.Fatalf("%+v %v", v, err)
	}
	other, _ := author.BoundedSeries(seriesDefinition(), func() author.SeriesOptions { o := seriesOptions(3); o.ID = "foreign"; return o }())
	if _, err := other.Inspect(cut()); err == nil {
		t.Fatal("accepted different series")
	}
	snapshot = cut()
	snapshot.Deployment.Permissions[0].Target = "wrong"
	if _, err := s.Inspect(snapshot); err == nil {
		t.Fatal("accepted wrong grants")
	}
	snapshot = cut()
	snapshot.Composition.Roots[0].Limits.MaxEffects = 2
	if _, err := s.Inspect(snapshot); err == nil {
		t.Fatal("accepted wrong root")
	}
}

func TestBoundedSeriesInspectionIntermediateEvidence(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Inspect(nil); err == nil {
		t.Fatal("nil snapshot accepted")
	}
	p, _ := s.Provision()
	path := filepath.Join(t.TempDir(), "cuts.db")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	inspect := func() author.SeriesView {
		cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		view, err := s.Inspect(cut)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	id, _ := s.Run(1)
	state, err := owner.Snapshot(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = owner.Commit(t.Context(), state.Authority(), ""); err != nil {
		t.Fatal(err)
	}
	view := inspect()
	if view.Positions[0].State != author.SeriesCommitted || view.Positions[0].Observation != nil {
		t.Fatal(view)
	}
	if _, err = owner.AuthorizeIntent(t.Context(), view.Positions[0].Effect.OperationID); err != nil {
		t.Fatal(err)
	}
	view = inspect()
	if view.Positions[0].State != author.SeriesAuthorized {
		t.Fatal(view)
	}
	// One reduction consumes the result but cannot also execute Success. The
	// successful observation remains pending until its later local terminal work.
	meter := new(seriesMeter)
	_, err = owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 1, MaxDispatch: 8, Adapters: map[string]effects.Adapter{"meter": meter}})
	if err != nil && !errors.Is(err, runtime.ErrLimit) {
		t.Fatal(err)
	}
	view = inspect()
	if view.Selected != 1 || view.Completed != 0 || view.Pending != 1 || view.Positions[0].State != author.SeriesSelected {
		t.Fatal(view)
	}
	_, err = owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 8, MaxDispatch: 8, Adapters: map[string]effects.Adapter{"meter": meter}})
	if err != nil {
		t.Fatal(err)
	}
	view = inspect()
	if view.Completed != 1 || !view.BoundExhausted || meter.calls != 1 {
		t.Fatal(view)
	}
	cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	original, _ := cut.Run(id)
	original.Projection.PC = 0
	if _, err = s.Inspect(cut); err == nil {
		t.Fatal("accepted mismatched current projection")
	}
}

type seriesUnknownMeter struct{ seriesMeter }

func (m *seriesUnknownMeter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
}
func TestBoundedSeriesPaidUnknownAllowsDistinctApproval(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(2))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.Provision()
	path := filepath.Join(t.TempDir(), "unknown.db")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if owner != nil {
			owner.Close()
		}
	}()
	meter := new(seriesUnknownMeter)
	run := func() {
		_, err := owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 16, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"meter": meter}})
		if err != nil {
			t.Fatal(err)
		}
	}
	run()
	cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	if view.Spent != 1 || view.Unresolved != 1 || view.Pending != 2 || view.Selected != 0 || view.Positions[0].State != author.SeriesOutcomeUnknown || view.Positions[1].State != author.SeriesAwaitingApproval || view.Positions[0].Effect.Usage != nil || view.Positions[0].Effect.Settlement != nil || view.Positions[0].Observation != nil {
		t.Fatal(view)
	}
	operation := view.Positions[0].Effect.OperationID
	for range 2 {
		if err = owner.Close(); err != nil {
			t.Fatal(err)
		}
		owner, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		run()
	}
	if meter.calls != 1 {
		t.Fatal("reopened unknown contacted again")
	}
	cut, err = runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	next, _ := s.Run(2)
	basis, err := cut.ContinuationBasis(next)
	if err != nil {
		t.Fatal(err)
	}
	if basis.ResultDigest != "" || basis.PredecessorID != operation {
		t.Fatal(basis)
	}
	approval, err := owner.ApproveContinuation(t.Context(), basis, "observe again")
	if err != nil || !approval.Accepted {
		t.Fatalf("%+v %v", approval, err)
	}
	run()
	cut, err = runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err = s.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	if view.Completed != 0 || view.Spent != 2 || !view.BoundExhausted || view.Unresolved != 2 || view.Positions[0].Effect.OperationID != operation || meter.calls != 2 {
		t.Fatal(view)
	}
}

func TestBoundedSeriesRetainedResultIsNotSelection(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(1))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.Provision()
	path := filepath.Join(t.TempDir(), "unselected.db")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	meter := new(seriesMeter)
	// The first reduction commits the request. Dispatch is allowed, but the next
	// ready reduction exceeds the step budget, retaining an unselected response.
	_, err = owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 1, MaxDispatch: 8, Adapters: map[string]effects.Adapter{"meter": meter}})
	if !errors.Is(err, runtime.ErrLimit) {
		t.Fatalf("expected bounded stop: %v", err)
	}
	cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	if view.Positions[0].Effect.AdapterResult == nil || view.Positions[0].Observation != nil || view.Selected != 0 || view.Completed != 0 || view.Positions[0].State != author.SeriesResultUnselected {
		t.Fatal(view)
	}
	cut.Runs[0].Projection.Status = "succeeded"
	if _, err = s.Inspect(cut); err == nil {
		t.Fatal("fabricated completion accepted")
	}
}

func TestBoundedSeriesRejectsDifferentInitialDeployment(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(2))
	if err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*runtime.Provision){
		"input":             func(p *runtime.Provision) { p.Runs[0].Projection.Data["station"] = schema.Text("south") },
		"owner":             func(p *runtime.Provision) { p.Subjects[0].Owner = "another-owner" },
		"root allowance":    func(p *runtime.Provision) { p.Composition.Roots[0].Limits.MaxEffects = 2 },
		"approval identity": func(p *runtime.Provision) { p.Composition.Continuations[0].ApprovalID = "other-approval" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			p, _ := s.Provision()
			change(&p)
			path := filepath.Join(t.TempDir(), "other.db")
			owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Inspect(cut); err == nil {
				t.Fatal("different coherent deployment accepted")
			}
		})
	}
}

func ExampleBoundedSeries() {
	observation := author.Definition{Name: "read sample", Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{}, Output: schema.Record{}}}, Steps: []author.Step{author.Request("request", "read", nil, nil), author.Await("read", "request", nil), author.Success()}}
	series, err := author.BoundedSeries(observation, author.SeriesOptions{ID: "samples", Count: 8, First: author.ApproveAtProvision, ApprovalReasons: []string{"observe again"}})
	if err != nil {
		panic(err)
	}
	provision, err := series.Provision()
	if err != nil {
		panic(err)
	}
	first, _ := series.Run(1)
	fmt.Println(first.Subject, first.Generation)
	fmt.Println(len(provision.Runs), len(provision.Composition.Continuations))
	// Output:
	// samples-1 1
	// 8 7
}

type seriesFailedMeter struct{ seriesMeter }

func (m *seriesFailedMeter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.calls++
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "transport_error"}
}
func TestBoundedSeriesLocalFailurePreservesExternalUncertainty(t *testing.T) {
	s, err := author.BoundedSeries(seriesDefinition(), seriesOptions(1))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.Provision()
	path := filepath.Join(t.TempDir(), "failed.db")
	owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	meter := new(seriesFailedMeter)
	_, err = owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 16, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"meter": meter}})
	if err != nil {
		t.Fatal(err)
	}
	cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	if view.Completed != 0 || view.Failed != 1 || view.Selected != 1 || view.Unresolved != 1 || view.Pending != 0 || view.Positions[0].State != author.SeriesFailed || view.Positions[0].Observation == nil || !view.BoundExhausted {
		t.Fatal(view)
	}
	view.Positions[0].Observation.Payload["sample"] = schema.Text("invented")
	if _, err = s.Inspect(cut); err == nil {
		t.Fatal("mismatched selected typed result accepted")
	}
}

func TestBoundedSeriesOwnsNestedSchemasAndValues(t *testing.T) {
	d := seriesDefinition()
	element := schema.Record{"name": {Kind: "string", MaxBytes: 16, Enum: []string{"north"}}}
	d.Inputs["stations"] = schema.Field{Kind: "list", Element: element, MaxItems: 2, MaxBytes: 256}
	o := seriesOptions(2)
	o.Inputs["stations"] = schema.List(map[string]schema.Value{"name": schema.Text("north")})
	s, err := author.BoundedSeries(d, o)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.Provision()
	element["name"].Enum[0] = "changed"
	replacement := schema.List(map[string]schema.Value{"name": schema.Text("south")})
	*o.Inputs["stations"].List = *replacement.List
	p, _ := s.Provision()
	*p.Runs[0].Projection.Data["stations"].List = *replacement.List
	after, _ := s.Provision()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("nested values or schemas alias declaration")
	}
}

func TestBoundedSeriesTerminalFailureBeforeSelection(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		name := "retained_result"
		if unknown {
			name = "unknown_result"
		}
		t.Run(name, func(t *testing.T) {
			d := seriesDefinition()
			d.Steps = []author.Step{d.Steps[0], author.Failure(), d.Steps[1], author.Success()}
			s, err := author.BoundedSeries(d, seriesOptions(3))
			if err != nil {
				t.Fatal(err)
			}
			p, err := s.Provision()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "failed-before-selection.db")
			owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if owner != nil {
					owner.Close()
				}
			}()
			knownMeter, unknownMeter := new(seriesMeter), new(seriesUnknownMeter)
			var adapter effects.Adapter = knownMeter
			if unknown {
				adapter = unknownMeter
			}
			_, err = owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 16, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"meter": adapter}})
			if err != nil {
				t.Fatal(err)
			}
			var operation string
			for _, phase := range []string{"original", "reopened"} {
				if phase == "reopened" {
					if err = owner.Close(); err != nil {
						t.Fatal(err)
					}
					owner, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
					if err != nil {
						t.Fatal(err)
					}
				}
				cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				view, err := s.Inspect(cut)
				if err != nil {
					t.Fatal(err)
				}
				first := view.Positions[0]
				if first.Run.Projection.Status != "failed" || first.Effect == nil || first.Observation != nil || view.Selected != 0 || view.Spent != 1 {
					t.Fatalf("%s: invalid discriminating failure cut: %+v", phase, view)
				}
				if operation == "" {
					operation = first.Effect.OperationID
				}
				if first.Effect.OperationID != operation || (first.Effect.AdapterResult == nil) != unknown || (first.Effect.Settlement == nil) != unknown {
					t.Fatalf("%s: original effect/result evidence changed: %+v", phase, first.Effect)
				}
				wantUnresolved := 0
				if unknown {
					wantUnresolved = 1
				}
				if first.State != author.SeriesFailed || view.Failed != 1 || view.Pending != 2 || view.Completed != 0 || view.Unresolved != wantUnresolved {
					t.Errorf("%s: terminal failure state=%s failed=%d pending=%d completed=%d unresolved=%d; want failed/1/2/0/%d", phase, first.State, view.Failed, view.Pending, view.Completed, view.Unresolved, wantUnresolved)
				}
			}
			if knownMeter.calls+unknownMeter.calls != 1 {
				t.Fatal("expected one original adapter contact")
			}
		})
	}
}

func TestBoundedSeriesContinuationIdentityLimits(t *testing.T) {
	for _, id := range []string{strings.Repeat("x", 4100), strings.Repeat("\x00", 700), strings.Repeat("x", 4085), strings.Repeat("\x00", 680) + "xxxxx", strings.Repeat("<", 681), strings.Repeat("\u2028", 681)} {
		o := seriesOptions(2)
		o.ID = id
		if _, err := author.BoundedSeries(seriesDefinition(), o); !errors.Is(err, runtime.ErrLimit) {
			t.Errorf("unapprovable identity: raw bytes=%d want ErrLimit, got %v", len(id), err)
		}
		o.Count = 1
		if _, err := author.BoundedSeries(seriesDefinition(), o); err != nil {
			t.Errorf("single observation has no continuation identity restriction: %v", err)
		}
	}
	// Position 10 is a predecessor at bound 11, adding one identity byte.
	o := seriesOptions(11)
	o.ID = strings.Repeat("x", 4084)
	if _, err := author.BoundedSeries(seriesDefinition(), o); err == nil {
		t.Error("ignored predecessor index digit growth")
	}
	// The second predecessor adds an approval receive before an already
	// nine-reduction request prefix, growing the sequence from 9 to 10.
	d := seriesDefinition()
	prefix := make([]author.Step, 8)
	for i := range prefix {
		prefix[i] = author.Set("local", author.Integer(1))
	}
	d.Steps = append(prefix, d.Steps...)
	o = seriesOptions(3)
	o.ID = strings.Repeat("x", 4084)
	if _, err := author.BoundedSeries(d, o); err == nil {
		t.Error("ignored protected predecessor sequence digit growth")
	}
}

func TestBoundedSeriesContinuationIdentityBoundaryExecutes(t *testing.T) {
	for name, id := range map[string]string{
		"plain":           strings.Repeat("x", 4084),
		"escaped":         strings.Repeat("\x00", 680) + "xxxx",
		"html escaped":    strings.Repeat("<", 680) + "xxxx",
		"unicode escaped": strings.Repeat("\u2028", 680) + "xxxx",
	} {
		t.Run(name, func(t *testing.T) {
			o := seriesOptions(2)
			o.ID = id
			s, err := author.BoundedSeries(seriesDefinition(), o)
			if err != nil {
				t.Fatal(err)
			}
			p, err := s.Provision()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "identity-boundary.db")
			owner, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			meter := new(seriesMeter)
			run := func() {
				_, err := owner.Run(t.Context(), runtime.RunOptions{MaxSteps: 16, MaxDispatch: 16, Adapters: map[string]effects.Adapter{"meter": meter}})
				if err != nil {
					t.Fatal(err)
				}
			}
			run()
			cut, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			next, _ := s.Run(2)
			basis, err := cut.ContinuationBasis(next)
			if err != nil {
				t.Fatal(err)
			}
			if len(basis.PredecessorID) != 4096 {
				t.Fatalf("not the field boundary: %d", len(basis.PredecessorID))
			}
			approval, err := owner.ApproveContinuation(t.Context(), basis, "observe again")
			if err != nil || !approval.Accepted {
				t.Fatalf("boundary approval: %+v %v", approval, err)
			}
			run()
			cut, err = runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			view, err := s.Inspect(cut)
			if err != nil {
				t.Fatal(err)
			}
			if meter.calls != 2 || view.Completed != 2 || view.Spent != 2 {
				t.Fatalf("calls=%d view=%+v", meter.calls, view)
			}
		})
	}
}

func TestBoundedSeriesContinuationSequenceWidth(t *testing.T) {
	// With no collection loop, the first PC bound includes nine pure reductions.
	d := seriesDefinition()
	prefix := make([]author.Step, 9)
	for i := range prefix {
		prefix[i] = author.Set("local", author.Integer(1))
	}
	d.Steps = append(prefix, d.Steps...)
	o := seriesOptions(2)
	o.ID = strings.Repeat("x", 4084)
	if _, err := author.BoundedSeries(d, o); !errors.Is(err, runtime.ErrLimit) {
		t.Fatalf("first request sequence width: %v", err)
	}
	o.ID = strings.Repeat("x", 4083)
	if _, err := author.BoundedSeries(d, o); err != nil {
		t.Fatalf("two-digit sequence boundary: %v", err)
	}

	// A legal pure collection prefix can require many more reductions than its
	// static code length. Construction reserves int64 width without running it.
	d = seriesDefinition()
	d.Collections = map[string]schema.Field{"items": {Kind: "list", MaxItems: 1, MaxBytes: 128, Element: schema.Record{"value": {Kind: "int"}}}}
	d.Steps = append([]author.Step{author.ForEach(author.Var("items"), nil, author.Set("local", author.Integer(1)))}, d.Steps...)
	o.ID = strings.Repeat("x", 4067)
	if _, err := author.BoundedSeries(d, o); !errors.Is(err, runtime.ErrLimit) {
		t.Fatalf("loop sequence width: %v", err)
	}
	o.ID = strings.Repeat("x", 4066)
	if _, err := author.BoundedSeries(d, o); err != nil {
		t.Fatalf("full int64-width boundary: %v", err)
	}

	// The final position is never a predecessor; its extra index digit does not
	// require additional continuation payload space until capacity reaches 11.
	o = seriesOptions(10)
	o.ID = strings.Repeat("x", 4084)
	if _, err := author.BoundedSeries(seriesDefinition(), o); err != nil {
		t.Fatalf("last position mistaken for predecessor: %v", err)
	}
}
