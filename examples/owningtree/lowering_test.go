package owningtree_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/owningtree"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Decode the executable fields of the retained artifact, independently of the
// author's source-position table. All targets below are relative to current PC.
type operand struct {
	Ref     string
	Literal *schema.Value
	Offset  *int64
}

func (v operand) String() string {
	if v.Offset != nil {
		return fmt.Sprintf("offset(%s,%d)", v.Ref, *v.Offset)
	}
	if v.Ref != "" && v.Literal == nil {
		return v.Ref
	}
	if v.Ref == "" && v.Literal != nil {
		switch v.Literal.Kind {
		case "int":
			return strconv.FormatInt(v.Literal.Number, 10)
		case "bool":
			return strconv.FormatBool(v.Literal.Flag)
		case "string":
			return strconv.Quote(v.Literal.Text)
		}
	}
	return "bad operand"
}

type instruction struct {
	Op, Dst, Outcome  string
	A, B              operand
	Target, Otherwise int
	Match             *struct {
		Kind, Provenance                          string
		Source, Generation, Obligation, Operation operand
	}
	Effect *struct {
		Kind                           string
		Target, Generation, Obligation operand
		Payload                        map[string]operand
	}
	Bind       map[string]string
	Updates    map[string]operand
	Collection *schema.Field
}
type termRole struct {
	name, prefix string
	template     author.Template
	paths        []string
}

func renderCode(t *testing.T, code []instruction) []string {
	t.Helper()
	var out []string
	for pc, in := range code {
		var line string
		switch in.Op {
		case "set":
			line = fmt.Sprintf("set %s %s", in.Dst, in.A)
		case "branch":
			line = fmt.Sprintf("if %s %s %d %d", in.A, in.B, in.Target-pc, in.Otherwise-pc)
		case "jump", "loop":
			line = fmt.Sprintf("%s %d", in.Op, in.Target-pc)
		case "foreach":
			if in.Collection == nil || !reflect.DeepEqual(*in.Collection, schema.Field{Kind: "list", MaxItems: 1, MaxBytes: 192, Element: schema.Record{"id": {Kind: "int"}, "value": {Kind: "int"}}}) {
				t.Fatalf("PC %d collection %+v", pc, in.Collection)
			}
			line = fmt.Sprintf("foreach %s %d", in.A, in.Otherwise-pc)
		case "next":
			line = fmt.Sprintf("next %d %v", in.Target-pc, in.Bind)
		case "end":
			line = "end " + in.Outcome
		case "receive":
			m := in.Match
			line = fmt.Sprintf("receive %s %s %s %s %s %s %v", m.Kind, m.Provenance, m.Source, m.Generation, m.Obligation, m.Operation, in.Bind)
		case "emit":
			e := in.Effect
			line = fmt.Sprintf("emit %s %s %s %s %s %v", e.Kind, e.Target, e.Generation, e.Obligation, in.Dst, e.Payload)
		default:
			t.Fatalf("unexpected instruction at %d: %+v", pc, in)
		}
		if len(in.Updates) != 0 {
			t.Fatalf("extra updates at %d", pc)
		}
		out = append(out, line)
	}
	return out
}

func appendGuard(dst []string, a, b string) []string {
	return append(dst, "if "+a+" "+b+" 1 2", "jump 2", "set tree_valid false")
}

// The interval covers PCs [0,35) relative to a role's first id comparison.
// Scratch registers are reused only between completed role intervals.
func roleCode(path, name string) []string {
	code := appendGuard(nil, "report_"+path+"id", path+"id")
	code = append(code, "if "+path+"value 0 1 3", "set expected 1", "jump 2", "set expected 0", "set seen false",
		"foreach report_"+path+"answers 13", "next -1 map[id:row_id value:row_value]")
	code = appendGuard(code, "seen", "false")
	code = append(code, "set seen true")
	code = appendGuard(code, "row_id", path+"id")
	code = appendGuard(code, "row_value", "expected")
	tag := "report_" + path + "classification"
	code = append(code, "loop -12", "if "+tag+" \"answer\" 1 5")
	code = appendGuard(code, "seen", "true")
	code = append(code, "jump 10", "set "+name+"_answer false")
	code = appendGuard(code, "seen", "false")
	code = append(code, "if "+tag+" \"candidate_failed\" 1 2", "jump 4")
	return appendGuard(code, tag, "\"candidate_missing_reply\"")
}

func checkCode(t *testing.T, raw []byte, outer bool, roles ...termRole) {
	t.Helper()
	var program struct {
		Code   []instruction
		Author author.Metadata
	}
	check(t, json.Unmarshal(raw, &program))
	got := renderCode(t, program.Code)
	var want, invokes, selects []string
	receive := map[string]string{}
	forward := map[string]string{}
	if len(program.Author.Invocations) != len(roles) || len(program.Author.Effects) != 0 {
		t.Fatal("declarations")
	}
	for _, r := range roles {
		in, out, err := r.template.Schemas()
		check(t, err)
		d := program.Author.Invocations[r.name]
		if d.TemplateID != r.template.ID() || d.Input.ID != in.ID || d.Output.ID != out.ID {
			t.Fatal("changed child contract", r.name)
		}
		args := map[string]string{}
		bind := map[string]string{"status": r.name + "_status", "reason": r.name + "_reason", "response_present": r.name + "_present"}
		for _, path := range r.paths {
			for _, field := range []string{"id", "value", "mode"} {
				k := path + field
				wire := r.prefix + k
				args[k] = wire
				receive[wire] = wire
			}
			for _, field := range []string{"id", "classification", "answers"} {
				k := path + field
				wire := r.prefix + k
				bind["response."+k] = "report_" + wire
				forward[wire] = "report_" + wire
			}
		}
		invokes = append(invokes, fmt.Sprintf("emit invocation %q 1 %q %s_operation %v", r.template.ID(), r.name, r.name, args))
		selects = append(selects, fmt.Sprintf("receive invocation runtime-invocation %q 1 %q %s_operation %v", r.template.ID(), r.name, r.name, bind),
			"set "+r.name+"_answer true", fmt.Sprintf("if %s_present true 1 %d", r.name, 35*len(r.paths)+17))
		for _, path := range r.paths {
			selects = append(selects, roleCode(r.prefix+path, r.name)...)
		}
		selects = append(selects, "if "+r.name+"_answer true 1 8")
		selects = appendGuard(selects, r.name+"_status", "\"succeeded\"")
		selects = appendGuard(selects, r.name+"_reason", "\"member_succeeded\"")
		selects = append(selects, "jump 8")
		selects = appendGuard(selects, r.name+"_status", "\"failed\"")
		selects = appendGuard(selects, r.name+"_reason", "\"member_failed\"")
		selects = append(selects, "set tree_answer false", "jump 2", "set tree_valid false")
	}
	if !outer {
		want = append(want, fmt.Sprintf("receive message ordinary-message \"caller\" 1 \"work\" \"\" %v", receive))
	}
	want = append(want, invokes...)
	want = append(want, "set tree_valid true", "set tree_answer true")
	want = append(want, selects...)
	if !outer {
		want = append(want, "if tree_valid true 1 7", fmt.Sprintf("emit message \"caller\" 1 \"work\"  %v", forward))
	} else {
		want = append(want, "if tree_valid true 1 6")
	}
	want = append(want, "if tree_answer true 1 3", "set decision \"answer\"", "end succeeded", "set decision \"diagnosed\"", "end failed", "set decision \"invalid_tree_outcome\"", "end failed")
	if len(want) != len(got) {
		t.Fatalf("instructions got %d want %d", len(got), len(want))
	}
	for pc, line := range want {
		if got[pc] != line {
			t.Fatalf("pc %d got %q want %q", pc, got[pc], line)
		}
	}
}

func checkMember(t *testing.T, term author.Template, roles ...termRole) {
	t.Helper()
	var bundle struct {
		Descriptor struct {
			ID, EntryMember, EntryPort, CallerPeer string
			Members                                map[string]struct {
				ProgramID string
				Initial   runtime.Projection
			}
			Templates []string
		}
		Programs []runtime.Artifact
	}
	check(t, json.Unmarshal(term.Bytes(), &bundle))
	ds := bundle.Descriptor
	member := ds.Members["entry"]
	if ds.ID != term.ID() || ds.EntryMember != "entry" || ds.EntryPort != "public" || ds.CallerPeer != "caller" || len(ds.Members) != 1 || member.Initial.Status != "open" || member.Initial.PC != 0 || len(member.Initial.Data)+len(member.Initial.Cursors) != 0 {
		t.Fatal("changed singleton initializer", ds)
	}
	children := map[string]bool{}
	for _, r := range roles {
		children[r.template.ID()] = true
	}
	if !slices.Equal(slices.Sorted(maps.Keys(children)), ds.Templates) {
		t.Fatal("direct dependencies", ds.Templates)
	}
	for _, p := range bundle.Programs {
		if p.ID == member.ProgramID {
			checkCode(t, p.Bytes, false, roles...)
			return
		}
	}
	t.Fatal("missing entry program")
}

func TestActualConstructorInstructions(t *testing.T) {
	ac, err := a.New()
	check(t, err)
	bc, err := b.New()
	check(t, err)
	la, err := owningtree.Leaf(ac)
	check(t, err)
	lb, err := owningtree.Leaf(bc)
	check(t, err)
	leaf := []string{""}
	pair := []string{"l_", "r_"}
	for _, as := range []struct {
		name        string
		left, right author.Template
	}{{"AA", la, la}, {"AB", la, lb}, {"BA", lb, la}, {"BB", lb, lb}} {
		t.Run(as.name, func(t *testing.T) {
			f, err := owningtree.Fork(as.left, as.right)
			check(t, err)
			checkMember(t, f, termRole{"left", "l_", as.left, leaf}, termRole{"right", "r_", as.right, leaf})
			r, err := owningtree.Relay(f)
			check(t, err)
			checkMember(t, r, termRole{"child", "", f, pair})
			for _, side := range []string{"left", "right"} {
				l, rh := as.left, r
				lp, rp := leaf, pair
				if side == "left" {
					l, rh = r, as.right
					lp, rp = pair, leaf
				}
				tree, err := owningtree.Fork(l, rh)
				check(t, err)
				checkMember(t, tree, termRole{"left", "l_", l, lp}, termRole{"right", "r_", rh, rp})
				var paths []string
				for _, p := range lp {
					paths = append(paths, "l_"+p)
				}
				for _, p := range rp {
					paths = append(paths, "r_"+p)
				}
				o, err := owningtree.Outer(tree)
				check(t, err)
				checkCode(t, o.Bytes(), true, termRole{"tree", "", tree, paths})
				t.Logf("%s %s tree=%s outer=%s", side, strings.Join(paths, ","), tree.ID(), o.ID())
			}
		})
	}
}
