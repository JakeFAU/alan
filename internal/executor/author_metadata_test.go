package executor_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

func TestLegacyProgramEncodingUnchanged(t *testing.T) {
	p := executor.Program{Semantics: executor.SemanticVersion, Code: []executor.Instruction{{Op: "end", Outcome: "succeeded"}}}
	raw, _, err := executor.EncodeProgram(p)
	if err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{"Semantics":"alan/1d/v1","Code":[{"Op":"end","Dst":"","A":{"Ref":"","Literal":null,"Offset":null},"B":{"Ref":"","Literal":null,"Offset":null},"Target":0,"Otherwise":0,"Match":null,"Bind":null,"Effect":null,"Updates":null,"Outcome":"succeeded"}]}`)
	if !bytes.Equal(raw, legacy) {
		t.Fatalf("legacy bytes changed: %s", raw)
	}
	p.Author = &executor.ProgramMetadata{}
	if _, _, err = executor.EncodeProgram(p); !errors.Is(err, executor.ErrVersion) {
		t.Fatalf("metadata attached to old semantics: %v", err)
	}
}
func TestAuthorMetadataRejectsMissingReceiveContract(t *testing.T) {
	a, err := author.Compile(author.Definition{Ports: map[string]author.Port{"input": {Peer: "local", Local: true, Obligation: "q", Receive: schema.Record{"q": {Kind: "int"}}}}, Steps: []author.Step{author.Receive("input", map[string]string{"q": "q"}), author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	var p executor.Program
	if err = json.Unmarshal(a.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	p.Author.Receives = nil
	if _, _, err = executor.EncodeProgram(p); err == nil {
		t.Fatal("missing retained receive contract accepted")
	}
}
func TestAuthorUnknownVersionsRejectedOnRecovery(t *testing.T) {
	ctx := context.Background()
	for _, field := range []string{"compiler", "semantics"} {
		t.Run(field, func(t *testing.T) {
			a, err := author.Compile(author.Definition{Steps: []author.Step{author.Success()}})
			if err != nil {
				t.Fatal(err)
			}
			provision, err := (author.Application{"worker": {Artifact: a}}).Provision()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "version.db")
			store, err := executor.Create(ctx, path, provision, executor.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			var program executor.Program
			if err = json.Unmarshal(a.Bytes(), &program); err != nil {
				t.Fatal(err)
			}
			sem := author.SemanticVersion
			if field == "compiler" {
				program.Author.Compiler = "future/compiler"
			} else {
				program.Semantics = "future/semantics"
				sem = program.Semantics
			}
			raw, err := json.Marshal(program)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			id := hex.EncodeToString(sum[:])
			db, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec("UPDATE programs SET id=?,semantics=?,bytes=?", id, sem, raw); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec("UPDATE runs SET program_id=?", id); err != nil {
				t.Fatal(err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err = executor.Replay(ctx, path, executor.DefaultLimits()); !errors.Is(err, executor.ErrVersion) {
				t.Fatalf("Replay: %v", err)
			}
			if opened, err := executor.Open(ctx, path, executor.DefaultLimits()); !errors.Is(err, executor.ErrVersion) {
				if opened != nil {
					opened.Close()
				}
				t.Fatalf("Open: %v", err)
			}
		})
	}
}
