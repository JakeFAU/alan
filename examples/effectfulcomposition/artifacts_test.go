package effectfulcomposition_test

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	frame "github.com/JakeFAU/alan/examples/effectfulcomposition"
	"github.com/JakeFAU/alan/schema"
)

// This source inventory has no endpoint service. The schema-5 URL is a literal
// configuration pin; no receipt or target identity is supplied to the programs.
func TestFrameArtifactInventory(t *testing.T) {
	type item struct {
		Shape                                                   string
		Left, Right                                             frame.RoleOptions
		TreeID, ParentID, LeftID, RightID, ResolverL, ResolverR string
		Programs, Templates                                     []string
		Input                                                   map[string]schema.Value
	}
	manifest := map[string]item{}
	blobs := map[string][]byte{}
	add := func(prefix, id string, b []byte) {
		name := prefix + "/" + id
		if p, ok := blobs[name]; ok && string(p) != string(b) {
			t.Fatal("different bytes under one identity", name)
		}
		blobs[name] = b
	}
	for _, c := range []struct {
		name, l, r, relay string
		equal, observed   bool
	}{
		{"fork-AA", "A", "A", "", false, false}, {"fork-AB", "A", "B", "", false, false},
		{"fork-BA", "B", "A", "", false, false}, {"fork-BB-equal", "B", "B", "", true, false},
		{"relay-fork-AB-mixed", "A", "B", "above", false, true},
		{"fork-relayA-B", "A", "B", "left", false, false}, {"fork-A-relayB", "A", "B", "right", false, false},
	} {
		tree, err := frame.PureFork(c.l, c.r, c.relay)
		if err != nil {
			t.Fatal(err)
		}
		input := map[string]schema.Value{"l_id": schema.Int(1), "r_id": schema.Int(1), "l_value": schema.Int(0), "r_value": schema.Int(1), "l_mode": schema.Text("answer"), "r_mode": schema.Text("answer")}
		o := frame.Options{Tree: tree, Inputs: input, Left: frame.RoleOptions{Queries: 2, Delta: 1}, Right: frame.RoleOptions{Queries: 1, Delta: 2, ObserveOriginal: c.observed}, Endpoint: "http://127.0.0.1:1234/phase2c/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/apply", Capacity: 2}
		if c.equal {
			o.Right.Delta = 1
		}
		if c.observed {
			o.Inputs["r_mode"] = schema.Text("absent")
		}
		d, err := frame.New(o)
		if err != nil {
			t.Fatal(c.name, err)
		}
		row := item{Shape: c.name, Left: o.Left, Right: o.Right, Input: o.Inputs, TreeID: d.Tree.ID(), ParentID: d.Parent.ID(), LeftID: d.Left.ID(), RightID: d.Right.ID(), ResolverL: d.Resolvers["l"].ID(), ResolverR: d.Resolvers["r"].ID()}
		for _, p := range d.Provision.Programs {
			h := sha256.Sum256(p.Bytes)
			if hex.EncodeToString(h[:]) != p.ID {
				t.Fatalf("%s program digest %s", c.name, p.ID)
			}
			add("program", p.ID, p.Bytes)
			row.Programs = append(row.Programs, p.ID)
		}
		for _, tplate := range d.Provision.Composition.Templates {
			raw, err := json.Marshal(tplate)
			if err != nil {
				t.Fatal(err)
			}
			add("template", tplate.ID, raw)
			row.Templates = append(row.Templates, tplate.ID)
		}
		manifest[c.name] = row
	}
	t.Logf("%d configurations, %d unique retained program/template records", len(manifest), len(blobs))
	dir := os.Getenv("ALAN_PHASE2D_AUTHORED")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "configurations.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "retained-bytes.json.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z := gzip.NewWriter(f)
	if err = json.NewEncoder(z).Encode(blobs); err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
}
