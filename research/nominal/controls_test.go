package nominal_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/schema"
)

const operationLiteral = `["probeA",1,2,0]`

func compiled(t *testing.T, d author.Definition) author.Artifact {
	t.Helper()
	p, err := author.Compile(d)
	if err != nil {
		t.Fatalf("ordinary compile: %v", err)
	}
	return p
}

func template(t *testing.T, p author.Artifact, initial map[string]schema.Value) author.Template {
	t.Helper()
	tmpl, err := author.NewTemplate(author.Application{"entry": {Artifact: p, Inputs: initial}}, "entry", "public")
	if err != nil {
		t.Fatalf("ordinary template: %v", err)
	}
	return tmpl
}

func publicPort(field schema.Field) map[string]author.Port {
	if field.Kind == "string" {
		field.MaxBytes = 4096
	}
	return map[string]author.Port{"public": {
		Peer: "caller", Obligation: "work", Receive: schema.Record{"value": field}, Send: schema.Record{"value": field},
	}}
}

func echo(t *testing.T, kind string) author.Template {
	t.Helper()
	return template(t, compiled(t, author.Definition{
		Name: "phase1q-handle-observation-helper", Ports: publicPort(schema.Field{Kind: kind}),
		Steps: []author.Step{
			author.Receive("public", map[string]string{"value": "input_value"}),
			author.Send("public", map[string]author.Expr{"value": author.Var("input_value")}),
			author.Success(),
		},
	}), map[string]schema.Value{})
}

func await(name, h, dst string) author.Step {
	return author.AwaitInvocation(name, h, author.InvocationOutcomeBindings{
		Status: "child_status", Reason: "child_reason", ResponsePresent: "child_present",
		Response: map[string]string{"value": dst},
	})
}

func invoke(h string, v author.Expr) author.Step {
	return author.Invoke(h, "helper", map[string]author.Expr{"value": v}, nil)
}

// Reconstruct the ordinary artifact and pin its *bytes* to the unchanged Q
// execution. The twin edits only the append operand and the Await operand.
func TestNominalQAndLiteralTwin(t *testing.T) {
	helper := echo(t, "int")
	for _, literal := range []bool{false, true} {
		value, pending := author.Var("original"), "saved"
		if literal {
			value, pending = author.String(operationLiteral), "original"
		}
		p := compiled(t, author.Definition{
			Name:        "phase1q-handle-observation-parent",
			Invocations: map[string]author.Template{"helper": helper},
			Collections: map[string]schema.Field{"handles": {Kind: "list", MaxItems: 1, MaxBytes: 8192,
				Element: schema.Record{"operation": {Kind: "string", MaxBytes: 4096}}}},
			Steps: []author.Step{
				invoke("original", author.Integer(7)),
				author.Append("handles", map[string]author.Expr{"operation": value}),
				author.ForEach(author.Var("handles"), map[string]string{"operation": "saved"},
					await("helper", pending, "child_value"),
					author.IfEqual(author.Var("saved"), author.String(operationLiteral),
						[]author.Step{author.Set("branch", author.String("literal_equal"))},
						[]author.Step{author.Set("branch", author.String("literal_different"))})),
				author.Success(),
			},
		})
		cert, err := nominal.CheckArtifact(p)
		if literal {
			if err != nil || cert.Version == "" || cert.Digest == "" {
				t.Fatalf("Q data twin: cert=%+v err=%v", cert, err)
			}
		} else {
			if sum := fmt.Sprintf("%x", sha256.Sum256(p.Bytes())); sum != "4adac49ee03e94e4edc155a6935a7ec57e5550d2cff184aad65d013e7afadded" {
				t.Fatalf("changed Q input: %s", sum)
			}
			if err == nil || !strings.Contains(err.Error(), "pc 2 (append)") || !strings.Contains(err.Error(), "operation") || !strings.Contains(err.Error(), "handle[helper]") {
				t.Fatalf("Q refusal must identify the first handle escape: %v", err)
			}
		}
		t.Logf("artifact=%s literal_twin=%v cert=%+v err=%v", p.ID(), literal, cert, err)
	}
}

func TestNominalCandidateCertificates(t *testing.T) {
	for _, tc := range []struct {
		build   func() (author.Template, error)
		id      string
		members int
	}{
		{a.New, "a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5", 1},
		{b.New, "2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266", 2},
	} {
		tmpl, err := tc.build()
		if err != nil {
			t.Fatal(err)
		}
		cert, err := nominal.CheckTemplate(tmpl)
		if err != nil || tmpl.ID() != tc.id || cert.ID != tc.id || cert.Scope != "template" ||
			cert.Programs != tc.members || cert.Templates != tc.members || cert.Version == "" || cert.Digest == "" {
			t.Fatalf("%s: %+v %v", tmpl.ID(), cert, err)
		}
		t.Logf("original template=%s cert=%+v", tmpl.ID(), cert)
	}
}

func TestNominalDataFlowAttacks(t *testing.T) {
	txt := author.String(operationLiteral)
	handle := author.Var("original")
	copyOf := author.Var("copy")
	choice := func(a, b author.Expr) author.Step {
		return author.IfEqual(author.Var("choose"), author.Boolean(true),
			[]author.Step{author.Set("copy", a)}, []author.Step{author.Set("copy", b)})
	}
	out := func(expr author.Expr) author.Step {
		return author.Send("public", map[string]author.Expr{"value": expr})
	}
	helper := echo(t, "string")
	// All cases start from the same genuine commit and otherwise valid port.
	for _, tc := range []struct {
		name                            string
		steps                           []author.Step
		ordinaryRefusal, nominalRefusal bool
	}{
		{"genuine-copy", []author.Step{author.Set("copy", handle), await("helper", "copy", "data")}, false, false},
		{"literal-forgery", []author.Step{author.Set("copy", txt), await("helper", "copy", "data")}, false, true},
		{"wrong-declaration", []author.Step{author.Set("copy", handle), await("other", "copy", "data")}, true, false},
		{"payload-data", []author.Step{author.Set("copy", txt), out(copyOf)}, false, false},
		{"alias-payload-escape", []author.Step{author.Set("copy", handle), out(copyOf)}, false, true},
		{"same-sort-join", []author.Step{choice(handle, handle), await("helper", "copy", "data")}, false, false},
		{"mixed-join", []author.Step{choice(handle, txt), await("helper", "copy", "data")}, true, false},
		{"mixed-unused-join", []author.Step{choice(handle, txt)}, false, true},
		{"overwrite-with-data", []author.Step{author.Set("copy", handle), author.Set("copy", txt), await("helper", "copy", "data")}, false, true},
		{"optional-response-rebinds-handle", []author.Step{await("helper", "original", "original"), out(handle)}, false, true},
		{"invoke-update-escape", []author.Step{author.Invoke("second", "helper", map[string]author.Expr{"value": txt}, map[string]author.Expr{"copy": handle}), out(copyOf)}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps := append([]author.Step{invoke("original", txt)}, tc.steps...)
			d := author.Definition{Inputs: schema.Record{"choose": {Kind: "bool"}}, Ports: publicPort(schema.Field{Kind: "string"}),
				Invocations: map[string]author.Template{"helper": helper, "other": helper}, Steps: append(steps, author.Success())}
			p, err := author.Compile(d)
			if tc.ordinaryRefusal {
				if err == nil {
					t.Fatal("ordinary author admitted bad expression")
				}
				t.Logf("ordinary refusal: %v", err)
				return
			}
			if err != nil {
				t.Fatalf("ordinary compile: %v", err)
			}
			_, err = nominal.CheckArtifact(p)
			if (err != nil) != tc.nominalRefusal {
				t.Fatalf("nominal refusal=%v want %v: %v", err != nil, tc.nominalRefusal, err)
			}
			t.Logf("artifact=%s nominal_err=%v", p.ID(), err)
		})
	}
}

func TestNominalTransitiveIdentity(t *testing.T) {
	certify := func(value int64, replacement bool) nominal.Certificate {
		steps := []author.Step{author.Receive("public", map[string]string{"value": "request"})}
		if replacement {
			steps = append(steps, author.Set("request", author.Integer(0)))
		}
		steps = append(steps, author.Send("public", map[string]author.Expr{"value": author.Var("default_value")}), author.Success())
		deps := template(t, compiled(t, author.Definition{Inputs: schema.Record{"default_value": {Kind: "int"}},
			Ports: publicPort(schema.Field{Kind: "int"}), Steps: steps}), map[string]schema.Value{"default_value": schema.Int(value)})
		root := compiled(t, author.Definition{Ports: publicPort(schema.Field{Kind: "int"}), Invocations: map[string]author.Template{"helper": deps},
			Steps: []author.Step{author.Receive("public", map[string]string{"value": "request"}), invoke("original", author.Var("request")),
				await("helper", "original", "answer"), author.Send("public", map[string]author.Expr{"value": author.Var("answer")}), author.Success()}})
		cert, err := nominal.CheckTemplate(template(t, root, map[string]schema.Value{}))
		if err != nil {
			t.Fatal(err)
		}
		return cert
	}
	original := certify(7, false)
	if original != certify(7, false) || original.Digest == certify(8, false).Digest || original.Digest == certify(7, true).Digest {
		t.Fatal("certificate failed to bind helper code and actual initializer")
	}
}

// The first iteration can send literal data while the second can leak an
// earlier commit. A forward-only pass over the lowered loop misses that edge.
func TestNominalIterationBackedge(t *testing.T) {
	helper := echo(t, "string")
	rows := schema.Field{Kind: "list", MaxItems: 2, MaxBytes: 4096, Element: schema.Record{"n": {Kind: "int"}}}
	for _, launder := range []bool{false, true} {
		last := author.String(operationLiteral)
		if launder {
			last = author.Var("original")
		}
		p := compiled(t, author.Definition{Ports: publicPort(schema.Field{Kind: "string"}),
			Invocations: map[string]author.Template{"helper": helper}, Collections: map[string]schema.Field{"rows": rows},
			Values: map[string]schema.Value{"rows": schema.List(map[string]schema.Value{"n": schema.Int(0)}, map[string]schema.Value{"n": schema.Int(1)})},
			Steps: []author.Step{invoke("original", author.String(operationLiteral)), author.Set("carry", author.String(operationLiteral)),
				author.ForEach(author.Var("rows"), map[string]string{"n": "index"},
					author.Send("public", map[string]author.Expr{"value": author.Var("carry")}), author.Set("carry", last)), author.Success()}})
		_, err := nominal.CheckArtifact(p)
		if (err != nil) != launder {
			t.Fatalf("second-iteration leak=%v artifact=%s nominal_err=%v", launder, p.ID(), err)
		}
	}
}

// Keep the outer boundary and its own instructions honest. A local check of
// the caller alone cannot confer a claim over the actual child receiving it.
func TestNominalDependencyLeak(t *testing.T) {
	txt := author.String(operationLiteral)
	helper := echo(t, "string")
	public := publicPort(schema.Field{Kind: "string"})
	for _, leak := range []bool{false, true} {
		operand := txt
		if leak {
			operand = author.Var("original")
		}
		nested := compiled(t, author.Definition{Ports: public, Invocations: map[string]author.Template{"helper": helper},
			Steps: []author.Step{author.Receive("public", map[string]string{"value": "request"}),
				invoke("original", author.Var("request")), author.Send("public", map[string]author.Expr{"value": operand}), author.Success()}})
		parent := compiled(t, author.Definition{Ports: public, Invocations: map[string]author.Template{"helper": template(t, nested, map[string]schema.Value{})},
			Steps: []author.Step{author.Receive("public", map[string]string{"value": "request"}),
				invoke("original", author.Var("request")), await("helper", "original", "data"),
				author.Send("public", map[string]author.Expr{"value": txt}), author.Success()}})
		if _, err := nominal.CheckArtifact(parent); err != nil {
			t.Fatalf("local scope should admit honest parent: %v", err)
		}
		_, err := nominal.CheckTemplate(template(t, parent, map[string]schema.Value{}))
		if (err != nil) != leak {
			t.Fatalf("dependency leak=%v nominal_err=%v", leak, err)
		}
	}
}
