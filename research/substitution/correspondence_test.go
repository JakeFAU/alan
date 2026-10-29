package substitution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFrozenInstructionCorrespondenceCoverage(t *testing.T) {
	type instruction struct {
		PC          int
		Instruction json.RawMessage
		Model       string
	}
	type record struct {
		Name         string
		ArtifactPath string `json:"artifact_path"`
		Digest       string `json:"artifact_bytes_sha256"`
		Instructions []instruction
	}
	var manifest struct{ Records []record }
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "docs/foundational/phase1i/evidence/model/artifact-correspondence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Records) != 3 {
		t.Fatal("expected A, B entry, B helper coverage")
	}
	for _, r := range manifest.Records {
		t.Run(r.Name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, r.ArtifactPath))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != r.Digest {
				t.Fatalf("frozen artifact changed: %s", got)
			}
			var p struct{ Code []json.RawMessage }
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			if len(p.Code) != len(r.Instructions) {
				t.Fatal("instruction omitted")
			}
			for pc, ins := range r.Instructions {
				if ins.PC != pc || ins.Model == "" {
					t.Fatalf("missing instruction mapping %d", pc)
				}
				var actual, recorded any
				if err := json.Unmarshal(p.Code[pc], &actual); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(ins.Instruction, &recorded); err != nil {
					t.Fatal(err)
				}
				a, _ := json.Marshal(actual)
				b, _ := json.Marshal(recorded)
				if string(a) != string(b) {
					t.Fatalf("instruction mapping lost operands at PC%d", pc)
				}
			}
		})
	}
}
