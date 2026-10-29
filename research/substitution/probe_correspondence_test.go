package substitution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActualProbeMappingCoverage(t *testing.T) {
	type record struct {
		Name         string
		ArtifactPath string `json:"artifact_path"`
		Digest       string `json:"artifact_bytes_sha256"`
		Instructions []struct {
			PC          int
			Model       string
			Instruction json.RawMessage
		}
	}
	var manifest struct {
		SourcePath   string `json:"source_path"`
		SourceDigest string `json:"source_sha256"`
		Templates    map[string]string
		Records      []record
	}
	root := filepath.Join("..", "..")
	dir := filepath.Join(root, "docs/foundational/phase1i/evidence/model/probe-artifacts")
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if err := json.Unmarshal(read(filepath.Join(dir, "correspondence.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	digest := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	if digest(read(filepath.Join(root, manifest.SourcePath))) != manifest.SourceDigest {
		t.Fatal("public probe source differs from inspected mapping")
	}
	expected := map[string]string{"weak-a": "receive set set emit end", "weak-b": "receive emit receive branch jump end emit end", "p": "receive emit receive branch jump end emit end", "q": "receive set emit receive branch jump end emit end", "helper-zero-false": "receive branch set jump set emit end", "helper-zero-true": "receive set emit end"}
	if len(manifest.Records) != len(expected) || len(manifest.Templates) != 4 {
		t.Fatal("missing actual probe artifact")
	}
	for filename, want := range manifest.Templates {
		if digest(read(filepath.Join(dir, filename))) != want {
			t.Fatalf("template changed %s", filename)
		}
	}
	count := 0
	for _, r := range manifest.Records {
		t.Run(r.Name, func(t *testing.T) {
			raw := read(filepath.Join(root, r.ArtifactPath))
			if digest(raw) != r.Digest {
				t.Fatal("program bytes changed")
			}
			var artifact struct{ Code []json.RawMessage }
			if err := json.Unmarshal(raw, &artifact); err != nil {
				t.Fatal(err)
			}
			short := strings.TrimPrefix(r.Name, "phase1i-probe-")
			ops := strings.Fields(expected[short])
			if len(ops) != len(artifact.Code) || len(r.Instructions) != len(ops) {
				t.Fatalf("incomplete mapping %s", short)
			}
			for pc, ins := range r.Instructions {
				count++
				if ins.PC != pc || ins.Model == "" {
					t.Fatalf("missing PC%d", pc)
				}
				var actual, recorded any
				if err := json.Unmarshal(artifact.Code[pc], &actual); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(ins.Instruction, &recorded); err != nil {
					t.Fatal(err)
				}
				a, _ := json.Marshal(actual)
				b, _ := json.Marshal(recorded)
				if string(a) != string(b) {
					t.Fatalf("lost operands PC%d", pc)
				}
				var op struct {
					Op                string
					Target, Otherwise int
					Dst               string
					Match             *struct {
						Kind, Provenance                          string
						Source, Generation, Obligation, Operation struct {
							Ref     string
							Literal *struct {
								Kind, Text string
								Number     int
							}
						}
					}
				}
				if err := json.Unmarshal(artifact.Code[pc], &op); err != nil {
					t.Fatal(err)
				}
				if op.Op != ops[pc] {
					t.Fatalf("unexpected opcode PC%d", pc)
				}
				if op.Op == "receive" {
					m := op.Match
					if m == nil || m.Generation.Literal == nil || m.Generation.Literal.Number != 1 {
						t.Fatalf("receive generation changed PC%d", pc)
					}
					if pc == 0 {
						if m.Kind != "message" || m.Provenance != "ordinary-message" || m.Source.Literal.Text != "caller" || m.Obligation.Literal.Text != "substitution" || m.Operation.Literal.Text != "" {
							t.Fatal("public receive guard changed")
						}
					} else {
						if m.Kind != "invocation" || m.Provenance != "runtime-invocation" || m.Operation.Ref != "helper_handle" || m.Obligation.Literal.Text != "helper" {
							t.Fatal("original helper guard changed")
						}
					}
				}
				if op.Op == "branch" {
					target, otherwise := 4, 5
					if short == "q" {
						target, otherwise = 5, 6
					}
					if short == "helper-zero-false" {
						target, otherwise = 2, 4
					}
					if op.Target != target || op.Otherwise != otherwise {
						t.Fatalf("branch arm omitted PC%d", pc)
					}
				}
			}
		})
	}
	if count != 41 {
		t.Fatalf("mapped%d want41", count)
	}
}

func TestProbeHelperBothBranchArms(t *testing.T) {
	// Instruction-level checks of both retained arms; the product census stays input0.
	for _, value := range []int{0, 1} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			c := Config{Shape: "bare", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 2}
			s := Initial(c)
			s.Ops[helper].Admission = 1
			s.Ops[helper].PC = mode
			s.PrivatePC[helper] = 1
			s.PrivateRegisters[helper][0] = 1
			s.PrivateRegisters[helper][1] = value
			g := actualProbeGraph("p")
			branch := State{}
			found := false
			for _, e := range g(c, s) {
				if e.Event.Kind == "private_step" && e.Event.Operation == "h" {
					branch = e.Next
					found = true
				}
			}
			wantPC := 2
			if value == 1 {
				wantPC = 4
			}
			if !found || branch.PrivatePC[helper] != wantPC {
				t.Fatal("incorrect helper branch")
			}
			found = false
			for _, e := range g(c, branch) {
				if e.Event.Kind == "private_step" && e.Event.Operation == "h" {
					found = true
					if e.Next.PrivateRegisters[helper][2] != 1-value {
						t.Fatal("incorrect helper Set")
					}
				}
			}
			if !found {
				t.Fatal("missing helper Set")
			}
		})
	}
}
