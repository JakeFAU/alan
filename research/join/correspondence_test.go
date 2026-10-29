package join

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// These types and evaluator do not import the compiler, reducer or schema.
// They inspect independently frozen instruction arrays against the J1 oracle.
type scalar struct {
	Kind   string
	Number int
	Text   string
	Flag   bool
}
type operand struct {
	Ref     string
	Literal *scalar
}
type route struct {
	Kind, Provenance                                  string
	Source, Target, Generation, Obligation, Operation operand
	Payload                                           map[string]operand
}
type instruction struct {
	Op, Dst, Outcome  string
	A, B              operand
	Target, Otherwise int
	Match, Effect     *route
	Bind              map[string]string
}
type program struct{ Code []instruction }

func artifactBytes(t *testing.T, name string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join("../../docs/foundational/phase1j/evidence/context/artifacts", name+".json"))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func artifact(t *testing.T, name string) program {
	t.Helper()
	var p program
	if e := json.Unmarshal(artifactBytes(t, name), &p); e != nil {
		t.Fatal(e)
	}
	return p
}
func number(n int) scalar  { return scalar{Kind: "int", Number: n} }
func word(s string) scalar { return scalar{Kind: "string", Text: s} }
func flag(b bool) scalar   { return scalar{Kind: "bool", Flag: b} }
func eval(t *testing.T, o operand, vars map[string]scalar) scalar {
	t.Helper()
	if o.Literal != nil {
		return *o.Literal
	}
	v, ok := vars[o.Ref]
	if !ok {
		t.Fatalf("undefined scalar %q", o.Ref)
	}
	return v
}
func suffix(t *testing.T, p program, pc int, vars map[string]scalar) (string, map[string]scalar) {
	t.Helper()
	var payload map[string]scalar
	for {
		i := p.Code[pc]
		switch i.Op {
		case "set":
			vars[i.Dst] = eval(t, i.A, vars)
			pc++
		case "branch":
			if eval(t, i.A, vars) == eval(t, i.B, vars) {
				pc = i.Target
			} else {
				pc = i.Otherwise
			}
		case "jump":
			pc = i.Target
		case "emit":
			if i.Effect == nil || i.Effect.Kind != "message" || payload != nil {
				t.Fatal("unexpected suffix effect")
			}
			payload = map[string]scalar{}
			for k, v := range i.Effect.Payload {
				payload[k] = eval(t, v, vars)
			}
			pc++
		case "receive":
			return "pending", payload
		case "end":
			return i.Outcome, payload
		default:
			t.Fatalf("unexpected suffix instruction %q", i.Op)
		}
	}
}
func putOutcome(vars map[string]scalar, status, present, id, value string, o Outcome) {
	vars[status] = word(o.Status)
	vars[present] = flag(o.Present)
	if o.Present {
		vars[id] = number(o.ID)
		vars[value] = number(o.Value)
	}
}
func outcomes() []Outcome {
	var all []Outcome
	for _, s := range []string{"succeeded", "failed", "cancelled", "abandoned", "rejected"} {
		for _, p := range []bool{false, true} {
			for id := 1; id <= 2; id++ {
				for v := 0; v <= 1; v++ {
					o := Outcome{Status: s, Reason: "exact retained reason", Present: p, ID: id, Value: v}
					if !p {
						o.ID = 0
						o.Value = 0
					}
					all = append(all, o)
				}
			}
		}
	}
	return all
}
func TestContextInstructionCorrespondence(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		t.Run(name, func(t *testing.T) {
			root := artifact(t, "join-"+name+"-root-program")
			peer := artifact(t, "join-"+name+"-peer-program")
			oldRoot := artifact(t, "c0-"+name+"-root-program")
			oldPeer := artifact(t, "c0-"+name+"-peer-program")
			if len(root.Code) != 28 || len(peer.Code) != 40 {
				t.Fatal("unmapped instruction count")
			}
			// Exact unchanged semantic prefixes, including every original handle operand.
			for _, pair := range []struct{ a, b []instruction }{{root.Code[:2], oldRoot.Code[:2]}, {peer.Code[:34], oldPeer.Code[:34]}} {
				a, _ := json.Marshal(pair.a)
				b, _ := json.Marshal(pair.b)
				if !bytes.Equal(a, b) {
					t.Fatal("primary prefix changed")
				}
			}
			for _, kind := range []string{"candidate-template", "s-template"} {
				if !bytes.Equal(artifactBytes(t, "c0-"+name+"-"+kind), artifactBytes(t, "join-"+name+"-"+kind)) {
					t.Fatal("frozen component changed")
				}
			}
			for pc, i := range root.Code {
				if i.Op == "branch" && (i.Target <= pc || i.Otherwise <= pc) || i.Op == "jump" && i.Target <= pc {
					t.Fatalf("root suffix not forward at %d", pc)
				}
			}
			for pc, i := range peer.Code {
				if i.Op == "branch" && (i.Target <= pc || i.Otherwise <= pc) || i.Op == "jump" && i.Target <= pc {
					t.Fatalf("peer suffix not forward at %d", pc)
				}
			}
			r := root.Code[2]
			if r.Op != "receive" || r.Match.Kind != "message" || r.Match.Provenance != "ordinary-message" || r.Match.Source.Literal.Text != "peer" || r.Match.Generation.Literal.Number != 1 || r.Match.Obligation.Literal.Text != Obligation {
				t.Fatal("mandatory join matching changed")
			}
			wantBind := map[string]string{"reason": "join_reason", "request_id": "join_request_id", "response_id": "join_id", "response_present": "join_present", "response_value": "join_value", "s_operation": "join_s_operation", "status": "join_status"}
			a, _ := json.Marshal(r.Bind)
			b, _ := json.Marshal(wantBind)
			if !bytes.Equal(a, b) {
				t.Fatal("join selected bindings changed")
			}
			for _, c := range Configurations() {
				if c.Depth == 3 {
					continue
				}
				for _, candidate := range outcomes() {
					for _, s := range outcomes() {
						vars := map[string]scalar{"request_id": number(c.Request.ID), "request_value": number(c.Request.Value), "join_request_id": number(c.Request.ID)}
						putOutcome(vars, "child_status", "child_present", "answer_id", "answer_value", candidate)
						putOutcome(vars, "join_status", "join_present", "join_id", "join_value", s)
						got, _ := suffix(t, root, 3, vars)
						want := "failed"
						if validCandidate(c.Request, candidate) && validS(c.Request, s) {
							want = "succeeded"
						}
						if got != want {
							t.Fatalf("root actual decision %s, J1 %s", got, want)
						}
					}
				}
			}
			for _, o := range outcomes() {
				vars := map[string]scalar{"request_id": number(1), "peer_pending": word("original-operation"), "peer_reason": word(o.Reason)}
				putOutcome(vars, "peer_status", "peer_present", "peer_id", "peer_value", o)
				end, payload := suffix(t, peer, 34, vars)
				if end != "succeeded" {
					t.Fatal("reporting peer not successful")
				}
				want := map[string]scalar{"request_id": number(1), "s_operation": word("original-operation"), "status": word(o.Status), "reason": word(o.Reason), "response_present": flag(o.Present), "response_id": number(o.ID), "response_value": number(o.Value)}
				a, _ := json.Marshal(payload)
				b, _ := json.Marshal(want)
				if !bytes.Equal(a, b) {
					t.Fatalf("wrong exact report %s != %s", a, b)
				}
			}
			got, _ := suffix(t, root, 3, map[string]scalar{"request_id": number(1), "join_request_id": number(2)})
			if got != "pending" {
				t.Fatal("wrong request discharged obligation")
			}
		})
	}
}
func TestOriginalFrozenComponentHashes(t *testing.T) {
	// These C0 bytes bind the reused A/B maps and unchanged original S. The old
	// Phase1I correspondence test additionally checks all A/B/helper operands.
	for name, want := range map[string]string{
		"join-a-candidate-template": "db5453b6e08abe5ecbf9614d472626c228675ba197abdaa36cad4aa3c5b04409",
		"join-b-candidate-template": "8693f675a4ec4be6d14a859157fbbf9d21b3608e4485d32b900421c0d7d7b5aa",
		"join-a-s-template":         "786e6c6a2122962469a1e263c57d30b81481faeafdbe9f395d0c669763874241",
	} {
		if got := fmt.Sprintf("%x", sha256.Sum256(artifactBytes(t, name))); got != want {
			t.Fatalf("%s frozen bytes changed %s", name, got)
		}
	}
}

func TestCorrespondenceManifest(t *testing.T) {
	var manifest struct {
		Records []struct {
			Artifact, SHA256 string
			Instructions     []struct {
				PC          int
				Model       string
				Instruction json.RawMessage
			}
		}
	}
	b, err := os.ReadFile("../../docs/foundational/phase1j/evidence/model/artifact-correspondence.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Records) != 4 {
		t.Fatal("missing actual context records")
	}
	for _, r := range manifest.Records {
		raw, err := os.ReadFile(filepath.Join("../..", r.Artifact))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(raw)) != r.SHA256 {
			t.Fatal("mapped artifact changed")
		}
		var p struct{ Code []json.RawMessage }
		if err = json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		if len(p.Code) != len(r.Instructions) {
			t.Fatal("missing instruction")
		}
		for pc, i := range r.Instructions {
			var a, b any
			if err = json.Unmarshal(p.Code[pc], &a); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(i.Instruction, &b); err != nil {
				t.Fatal(err)
			}
			aa, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			if i.PC != pc || i.Model == "" || !bytes.Equal(aa, bb) {
				t.Fatalf("mapping mismatch %s pc%d", r.Artifact, pc)
			}
		}
	}
}
