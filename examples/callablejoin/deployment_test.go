package callablejoin_test

import (
	"bytes"
	"encoding/json"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/callablejoin"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/examples/substitution/protocol"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"math"
	"testing"
)

// The old root-only consuming context cannot satisfy this callable boundary.
func TestPublicCallableConstruction(t *testing.T) {
	c, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	d, err := callablejoin.New(c, substitution.Config{Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 4, MaxCreatedRuns: 4, MaxDepth: 3, MaxInFlight: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Join.ID() == "" {
		t.Fatal("missing new public callable J template")
	}
	input, output, err := d.Join.Schemas()
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Record) != 3 || len(output.Record) != 3 || output.Record["answers"].MaxItems != 1 {
		t.Fatal("public request/report schema", input, output)
	}
	if d.Outer.ID() == "" || len(d.Provision.Runs) != 1 {
		t.Fatal("missing sole fixed O")
	}
}

// Literal historical template identities pin A/B/helper; S also matches the
// original public constructor byte-for-byte. Shared code normalization removes
// only the intended binding ID, never policy or schema instructions.
func TestOriginalBindingsAndCommonPolicy(t *testing.T) {
	var commonJ, commonO []byte
	for _, tc := range []struct {
		name, id string
		build    func() (author.Template, error)
	}{
		{"a", "a03135203f8b728a6aa7b732d06952cc87b70b3aced2c0833293e333760eece5", a.New},
		{"b", "2e10975004e0351d07b237e0360a2b96b50d5884b651cfa5bfc8e88af62da266", b.New},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			if c.ID() != tc.id {
				t.Fatal("historical candidate changed", c.ID())
			}
			cfg := substitution.Config{Shape: "sequential", Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 4, MaxCreatedRuns: 4, MaxDepth: 3, MaxInFlight: 1}}
			d, err := callablejoin.New(c, cfg)
			if err != nil {
				t.Fatal(err)
			}
			original, err := substitution.New(c, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(c.Bytes(), d.Candidate.Bytes()) || !bytes.Equal(original.Peer.Bytes(), d.Successor.Bytes()) || d.Successor.ID() != "a6bfb72f823541b21e8780faf1eaab779254f5a563e5352cd631521c7ab6c61c" {
				t.Fatal("original template bytes changed")
			}
			if tc.name == "b" {
				var bundle struct{ Dependencies []struct{ ID string } }
				if err := json.Unmarshal(c.Bytes(), &bundle); err != nil {
					t.Fatal(err)
				}
				if len(bundle.Dependencies) != 1 || bundle.Dependencies[0].ID != "1806cf3a9e1e67ec0ed25a665b7a8a9be2dd7cf37585c5b1cd3e274b12595e3d" {
					t.Fatal("original helper changed", bundle)
				}
			}
			j := bytes.ReplaceAll(d.JoinArtifact.Bytes(), []byte(c.ID()), []byte("CANDIDATE"))
			o := bytes.ReplaceAll(d.Outer.Bytes(), []byte(d.Join.ID()), []byte("JOIN"))
			if commonJ == nil {
				commonJ, commonO = j, o
			} else if !bytes.Equal(j, commonJ) || !bytes.Equal(o, commonO) {
				t.Fatal("policy differs beyond binding")
			}
			for _, artifact := range []author.Artifact{d.JoinArtifact, d.Outer} {
				var program struct {
					Code []struct {
						Op, Dst string
						A, B    struct {
							Ref     string
							Literal *schema.Value
						}
						Match *struct{ Operation struct{ Ref string } }
					}
				}
				if err := json.Unmarshal(artifact.Bytes(), &program); err != nil {
					t.Fatal(err)
				}
				for _, in := range program.Code {
					if in.Op == "branch" && (in.A.Ref == "request_mode" || in.B.Ref == "request_mode") {
						t.Fatal("mode-based diagnosis")
					}
				}
				if artifact.ID() == d.JoinArtifact.ID() {
					code := program.Code
					if code[0].Op != "set" || code[0].Dst != "answers" {
						t.Fatal("missing declared empty list initialization")
					}
					code = code[1:]
					if code[0].Op != "receive" || code[1].Dst != "candidate_operation" || code[2].Dst != "s_operation" || code[3].Match.Operation.Ref != "s_operation" || code[4].Match.Operation.Ref != "candidate_operation" {
						t.Fatal("original commitments/consumptions changed")
					}
				} else {
					seenTrue, seenFalse, presence := false, false, false
					for _, in := range program.Code {
						if in.Op == "branch" && in.B.Literal != nil {
							if in.A.Ref == "answer_seen" {
								if in.B.Literal.Flag {
									seenTrue = true
								} else {
									seenFalse = true
								}
							}
							if in.A.Ref == "join_present" && in.B.Literal.Flag {
								presence = true
							}
						}
					}
					if !seenTrue || !seenFalse || !presence {
						t.Fatal("missing exact-one/zero cardinality or presence guards")
					}
				}
			}
		})
	}
}

func TestSchemaBoundsDoNotImplyCardinalityOrFiniteProfile(t *testing.T) {
	report := callablejoin.Report()
	empty := map[string]schema.Value{"id": schema.Int(1), "classification": schema.Text("answer"), "answers": schema.List()}
	// Empty answers are schema-valid even with answer: O must reject semantically.
	if err := report.Validate(empty); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int64{math.MinInt64, math.MaxInt64} {
		one := schema.List(map[string]schema.Value{"id": schema.Int(n), "value": schema.Int(n)})
		if err := report["answers"].Validate(one); err != nil {
			t.Fatal("192 bytes must include full original integer domain", err)
		}
	}
	two := schema.List(map[string]schema.Value{"id": schema.Int(1), "value": schema.Int(1)}, map[string]schema.Value{"id": schema.Int(1), "value": schema.Int(1)})
	if err := report["answers"].Validate(two); err == nil {
		t.Fatal("two answers accepted")
	}
	if err := protocol.Request().Validate(map[string]schema.Value{"id": schema.Int(99), "value": schema.Int(99), "mode": schema.Text("answer")}); err != nil {
		t.Fatal("original scalar schema accidentally finite", err)
	}
	c, err := a.New()
	if err != nil {
		t.Fatal(err)
	}
	for _, req := range []substitution.Request{{ID: 99, Value: 0, Mode: "answer"}, {ID: 1, Value: 2, Mode: "answer"}, {ID: 1, Value: 0, Mode: "invalid"}} {
		if _, err := callablejoin.New(c, substitution.Config{Request: req}); err == nil {
			t.Fatal("invalid concrete profile accepted", req)
		}
	}
}
