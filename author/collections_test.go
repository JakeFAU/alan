package author_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func listField(n int) schema.Field {
	return schema.Field{Kind: "list", MaxItems: n, MaxBytes: 8192, UniqueKey: "id", Element: schema.Record{"id": {Kind: "string", MaxBytes: 20}, "payload": {Kind: "string", MaxBytes: 20}}}
}
func collectionDefinition(n int) author.Definition {
	f := listField(n)
	return author.Definition{Collections: map[string]schema.Field{"results": f}, Outputs: schema.Record{"results": f}, Ports: map[string]author.Port{"work": {Local: true, Peer: "local", Obligation: "work", Receive: schema.Record{"items": f}}}, Steps: []author.Step{
		author.Receive("work", map[string]string{"items": "items"}),
		author.ForEach(author.Var("items"), map[string]string{"id": "id", "payload": "payload"}, author.Append("results", map[string]author.Expr{"id": author.Var("id"), "payload": author.Var("payload")})), author.Success()}}
}
func TestCollectionsPostInitializationReopenReplay(t *testing.T) {
	artifact, err := author.Compile(collectionDefinition(8))
	if err != nil {
		t.Fatal(err)
	}
	var program executor.Program
	if err = json.Unmarshal(artifact.Bytes(), &program); err != nil {
		t.Fatal(err)
	}
	if program.Semantics != "alan/1g/v1" || artifact.Metadata().Compiler != "alan/author/v2" {
		t.Fatal("missing collection version")
	}
	for _, n := range []int{0, 1, 3, 7, 8, 9} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			p, err := (author.Application{"collector": {Artifact: artifact}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "collection.db")
			s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { s.Close() }()
			// Initialization has already committed before the operator chooses work.
			state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "collector", Generation: 1})
			if err != nil {
				t.Fatal(err)
			}
			state, err = s.Commit(t.Context(), state.Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			rows := make([]map[string]schema.Value, n)
			for i := range rows {
				rows[i] = map[string]schema.Value{"id": schema.Text(fmt.Sprint(i)), "payload": schema.Text("same")}
			}
			admitted, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "transport", LogicalID: "work", Target: "collector", Obligation: "work", Payload: map[string]schema.Value{"items": schema.List(rows...)}})
			if err != nil {
				t.Fatal(err)
			}
			if n == 9 {
				if admitted.Accepted {
					t.Fatal("over-limit input admitted")
				}
				return
			}
			if !admitted.Accepted {
				t.Fatalf("input rejected: %+v", admitted)
			}
			// Reopen at every instruction boundary, including snapshot and element binding.
			for steps := 0; steps < 100; steps++ {
				if _, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 1, MaxDispatch: 10}); err != nil && !errors.Is(err, runtime.ErrLimit) {
					t.Fatal(err)
				}
				state, err = s.Snapshot(t.Context(), state.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err = s.Close(); err != nil {
					t.Fatal(err)
				}
				s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if state.Projection.Status != "open" {
					break
				}
			}
			if state.Projection.Status != "succeeded" {
				t.Fatal("loop did not terminate")
			}
			got, err := state.Projection.Data["results"].Records()
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != n {
				t.Fatalf("got %d results want %d", len(got), n)
			}
			for i, row := range got {
				if row["id"] != schema.Text(fmt.Sprint(i)) || row["payload"] != schema.Text("same") {
					t.Fatalf("order or correspondence lost: %+v", got)
				}
			}
			replay, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if replay.Runs[0].Projection.Status != "succeeded" {
				t.Fatal("replay lost terminal")
			}
		})
	}
}
func TestCollectionCompilerSingleBodyAndZeroAssignment(t *testing.T) {
	a, err := author.Compile(collectionDefinition(1))
	if err != nil {
		t.Fatal(err)
	}
	b, err := author.Compile(collectionDefinition(8))
	if err != nil {
		t.Fatal(err)
	}
	var x, y executor.Program
	json.Unmarshal(a.Bytes(), &x)
	json.Unmarshal(b.Bytes(), &y)
	if len(x.Code) != len(y.Code) {
		t.Fatal("loop body unrolled by bound")
	}
	d := collectionDefinition(8)
	d.Outputs = schema.Record{"id": {Kind: "string", MaxBytes: 20}}
	if _, err = author.Compile(d); err == nil {
		t.Fatal("zero iterations established body binding")
	}
	d = collectionDefinition(8)
	d.Steps = []author.Step{author.Label("again"), author.ForEach(author.Var("results"), nil, author.Goto("again")), author.Success()}
	if _, err = author.Compile(d); err == nil {
		t.Fatal("public backward goto accepted")
	}
}

func TestNestedCollectionSnapshotsReopen(t *testing.T) {
	f := listField(8)
	groups := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 20000, Element: schema.Record{"children": f}}
	d := author.Definition{Inputs: schema.Record{"groups": groups}, Collections: map[string]schema.Field{"results": f}, Outputs: schema.Record{"results": f}, Steps: []author.Step{
		author.ForEach(author.Var("groups"), map[string]string{"children": "children"},
			author.ForEach(author.Var("children"), map[string]string{"id": "id", "payload": "payload"},
				author.Set("groups", author.Value(schema.List())), author.Set("children", author.Value(schema.List())), author.Append("results", map[string]author.Expr{"id": author.Var("id"), "payload": author.Var("payload")}))), author.Success()}}
	// Source-list replacement is declared as an input schema, and must still be typed.
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	one := func(id string) map[string]schema.Value {
		return map[string]schema.Value{"id": schema.Text(id), "payload": schema.Text("same")}
	}
	input := schema.List(map[string]schema.Value{"children": schema.List(one("a"), one("b"))}, map[string]schema.Value{"children": schema.List(one("c"))})
	p, err := (author.Application{"nested": {Artifact: a, Inputs: map[string]schema.Value{"groups": input}}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "nested.db")
	s, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	id := runtime.RunID{Subject: "nested", Generation: 1}
	for i := 0; i < 100; i++ {
		state, err := s.Snapshot(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if state.Projection.Status == "succeeded" {
			rows, _ := state.Projection.Data["results"].Records()
			if len(rows) != 3 || rows[0]["id"] != schema.Text("a") || rows[1]["id"] != schema.Text("b") || rows[2]["id"] != schema.Text("c") {
				t.Fatalf("snapshots changed: %+v", rows)
			}
			if _, err = runtime.Replay(t.Context(), path, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			return
		}
		if _, err = s.Commit(t.Context(), state.Authority(), ""); err != nil {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		s, err = runtime.Open(t.Context(), path, runtime.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("nested loops did not terminate")
}
func TestCollectionLoopPreservesPendingRequest(t *testing.T) {
	d := collectionDefinition(8)
	d.Ports = nil
	d.Inputs = schema.Record{"items": listField(8)}
	d.Effects = map[string]author.Effect{"work": {Capability: "controlled", Input: schema.Record{"id": {Kind: "string", MaxBytes: 20}}, Output: schema.Record{"reply": {Kind: "string", MaxBytes: 20}}}}
	d.Steps = []author.Step{author.ForEach(author.Var("items"), map[string]string{"id": "id", "payload": "payload"}, author.Request("pending", "work", map[string]author.Expr{"id": author.Var("id")}, nil), author.Await("work", "pending", map[string]string{"reply": "reply", "valid": "valid"}), author.Append("results", map[string]author.Expr{"id": author.Var("id"), "payload": author.Var("reply")})), author.Success()}
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	state := executor.Projection{Status: "open", Data: map[string]schema.Value{"items": schema.List(map[string]schema.Value{"id": schema.Text("a"), "payload": schema.Text("same")}, map[string]schema.Value{"id": schema.Text("b"), "payload": schema.Text("same")})}}
	id := executor.RunID{Subject: "loop", Generation: 1}
	pending := ""
	emitted := 0
	for seq := int64(1); seq < 40 && state.Status == "open"; seq++ {
		var obs *executor.Observation
		if p.Code[state.PC].Op == "receive" {
			if state.Data["pending"].Text != pending || pending == "" {
				t.Fatal("cursor erased pending identity")
			}
			obs = &executor.Observation{Run: id, Kind: "adapter", Provenance: "runtime-adapter", Source: "controlled", SourceGeneration: 1, Obligation: "work", Operation: pending, Payload: map[string]schema.Value{"reply": schema.Text("same"), "valid": schema.Bool(true), "disposition": schema.Text("validated")}}
		}
		r, err := executor.Reduce(p, state, obs, id, seq, executor.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Intents) > 0 {
			emitted++
			pending = r.Intents[0].ID
			if state.Data["pending"].Text == pending {
				t.Fatal("input projection mutated")
			}
		}
		raw, _ := json.Marshal(r.Projection)
		var reopened executor.Projection
		if err = json.Unmarshal(raw, &reopened); err != nil {
			t.Fatal(err)
		}
		state = reopened
	}
	n, _ := state.Data["results"].Len()
	if state.Status != "succeeded" || emitted != 2 || n != 2 {
		t.Fatalf("loop did not preserve requests: %+v", state)
	}
}

func TestCollectionReducerStructuralBranchAndMissingCursor(t *testing.T) {
	f := listField(8)
	d := author.Definition{Inputs: schema.Record{"left": f, "right": f}, Steps: []author.Step{author.IfEqual(author.Var("left"), author.Var("right"), []author.Step{author.Success()}, []author.Step{author.Failure()})}}
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	json.Unmarshal(a.Bytes(), &p)
	row := map[string]schema.Value{"id": schema.Text("same"), "payload": schema.Text("same")}
	s := executor.Projection{Status: "open", Data: map[string]schema.Value{"left": schema.List(row), "right": schema.List(row)}}
	r, err := executor.Reduce(p, s, nil, executor.RunID{Subject: "compare", Generation: 1}, 1, executor.DefaultLimits())
	if err != nil || p.Code[r.Projection.PC].Outcome != "succeeded" {
		t.Fatalf("structural branch: %+v %v", r, err)
	}
	a, err = author.Compile(collectionDefinition(8))
	if err != nil {
		t.Fatal(err)
	}
	p = executor.Program{}
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	for pc, in := range p.Code {
		if in.Op == "append" {
			s = executor.Projection{PC: pc, Status: "open", Data: map[string]schema.Value{"results": schema.List(), "id": schema.Text("a"), "payload": schema.Text("same")}}
			if _, err = executor.Reduce(p, s, nil, executor.RunID{Subject: "bad", Generation: 1}, 1, executor.DefaultLimits()); err == nil {
				t.Fatal("loop body resumed without durable snapshot")
			} else {
				t.Logf("rejection: %v", err)
			}
			break
		}
	}
}

func TestCollectionLiteralsAtTypedBoundaries(t *testing.T) {
	child := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 512, Element: schema.Record{"x": {Kind: "int"}}}
	outer := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 2048, Element: schema.Record{"children": child}}
	d := author.Definition{Collections: map[string]schema.Field{"outer": outer, "child": child}, Steps: []author.Step{author.Append("outer", map[string]author.Expr{"children": author.Value(schema.List(map[string]schema.Value{"x": schema.Int(1)}))}), author.IfEqual(author.Var("child"), author.Value(schema.List()), []author.Step{author.Success()}, []author.Step{author.Failure()})}}
	if _, err := author.Compile(d); err != nil {
		t.Fatal(err)
	}
}

func TestCollectionClosedProjectionRetainsSnapshot(t *testing.T) {
	d := collectionDefinition(8)
	d.Ports = nil
	d.Inputs = schema.Record{"items": listField(8)}
	d.Steps = d.Steps[1:]
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	s := executor.Projection{Status: "open", Data: map[string]schema.Value{"items": schema.List(map[string]schema.Value{"id": schema.Text("a"), "payload": schema.Text("same")})}}
	id := executor.RunID{Subject: "closed", Generation: 1}
	for seq := int64(1); seq <= 3; seq++ {
		r, err := executor.Reduce(p, s, nil, id, seq, executor.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		s = r.Projection
	}
	for _, status := range []string{"cancelled", "abandoned"} {
		s.Status = status
		if _, err = executor.Reduce(p, s, nil, id, 4, executor.DefaultLimits()); !errors.Is(err, executor.ErrClosed) {
			t.Fatalf("closed mid-loop projection became corrupt: %v", err)
		}
	}
}
func TestCollectionCompilerRejectsCyclicStepGraph(t *testing.T) {
	if os.Getenv("ALAN_TEST_CYCLIC_STEPS") == "1" {
		debug.SetMaxStack(1 << 20)
		steps := make([]author.Step, 1)
		steps[0] = author.Repeat(1, steps...)
		if _, err := author.Compile(author.Definition{Steps: steps}); err == nil {
			t.Fatal("cyclic author graph accepted")
		}
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCollectionCompilerRejectsCyclicStepGraph$")
	cmd.Env = append(os.Environ(), "ALAN_TEST_CYCLIC_STEPS=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cyclic graph did not reject within compiler bounds: %v\n%.1500s", err, out)
	}
}

func TestCollectionAppendOverflowLeavesInputUnchanged(t *testing.T) {
	f := listField(1)
	full := schema.List(map[string]schema.Value{"id": schema.Text("a"), "payload": schema.Text("same")})
	d := author.Definition{Collections: map[string]schema.Field{"items": f}, Values: map[string]schema.Value{"items": full}, Steps: []author.Step{author.Append("items", map[string]author.Expr{"id": author.String("b"), "payload": author.String("same")}), author.Success()}}
	a, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	r, err := executor.Reduce(p, executor.Projection{Status: "open", Data: map[string]schema.Value{}}, nil, executor.RunID{Subject: "full", Generation: 1}, 1, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	state := r.Projection
	before, _ := json.Marshal(state)
	if _, err = executor.Reduce(p, state, nil, executor.RunID{Subject: "full", Generation: 1}, 2, executor.DefaultLimits()); err == nil {
		t.Fatal("overflow append accepted")
	}
	after, _ := json.Marshal(state)
	if string(before) != string(after) || !state.Data["items"].Equal(full) {
		t.Fatal("failed append changed input projection")
	}
}
