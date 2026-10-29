package join_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
)

// TestJoinArtifactExport writes only public artifact bytes, when explicitly asked.
// These are correspondence evidence, never an oracle for the independent model.
func TestJoinArtifactExport(t *testing.T) {
	dir := os.Getenv("ALAN_JOIN_EXPORT")
	if dir == "" {
		t.Skip("ALAN_JOIN_EXPORT is unset")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	hashes := map[string]string{}
	write := func(name string, raw []byte) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(dir, name), raw, 0644); e != nil {
			t.Fatal(e)
		}
		h := sha256.Sum256(raw)
		hashes[name] = hex.EncodeToString(h[:])
	}
	export := func(name string, d substitution.Deployment) {
		t.Helper()
		write(name+"-candidate-template.json", d.Candidate.Bytes())
		write(name+"-s-template.json", d.Peer.Bytes())
		for _, r := range d.Provision.Runs {
			for _, p := range d.Provision.Programs {
				if p.ID == r.ProgramID {
					write(name+"-"+r.ID.Subject+"-program.json", p.Bytes)
				}
			}
		}
		raw, e := json.MarshalIndent(d.Provision, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		write(name+"-provision.json", raw)
	}
	for _, c := range []struct {
		name string
		new  func() (author.Template, error)
	}{{"a", a.New}, {"b", b.New}} {
		candidate, e := c.new()
		if e != nil {
			t.Fatal(e)
		}
		d, e := substitution.DelayedParallel(candidate, config(), 32)
		if e != nil {
			t.Fatal(e)
		}
		export("c0-"+c.name, d)
		d, e = join.New(candidate, config(), join.Options{})
		if e != nil {
			t.Fatal(e)
		}
		export("join-"+c.name, d)
		if c.name == "a" {
			for _, control := range []struct {
				name   string
				option join.Options
			}{{"s-fail", join.Options{SMode: "fail"}}, {"s-absent", join.Options{SMode: "absent"}}, {"s-pending", join.Options{SMode: "pending"}}, {"premature", join.Options{PrematureReport: true}}, {"wrong-request", join.Options{WrongRequest: true}}} {
				d, e = join.New(candidate, config(), control.option)
				if e != nil {
					t.Fatal(e)
				}
				export(control.name, d)
			}
		}
	}
	raw, e := json.MarshalIndent(hashes, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "sha256.json"), append(raw, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
