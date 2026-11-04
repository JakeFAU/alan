package executor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This genuine v1 Store was produced by 3f830766, before the v2 extension.
// The archive contains every closed Store component. Never open the archive's
// original: each test extracts a complete disposable copy without SQL edits.
func TestRequestAuthorityDormantPriorStore(t *testing.T) {
	raw, err := os.ReadFile("testdata/v1-dormant.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	tr := tar.NewReader(z)
	dir := t.TempDir()
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != h.Name {
			t.Fatal("invalid fixture entry", h.Name)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, h.Name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "v1-dormant.sqlite")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal("Inspect prior v1:", err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal("Replay prior v1:", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("read-only validation changed prior Store")
	}
	s, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal("Open prior v1:", err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	// Open may renew activation epochs, but programs and their dormant text stay exact.
	recovered, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot.Tables["programs"], recovered.Tables["programs"]) {
		t.Fatal("Open rewrote prior programs")
	}
}

func TestRequestAuthorityDormantClaimBoundary(t *testing.T) {
	for _, tc := range []struct{ name, claim string }{
		{"malformed", "uninterpreted dormant text"},
		{"wrong_version", `{"Version":"alan/request-authority/v2"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := requestAuthorityFixture(t, nil)
			authorityReplaceProgram(t, &p, "B", func(p *Program) { p.Code[1].Effect.Payload["claim"] = Lit(Text(tc.claim)) })
			s, path := messageStore(t, p) // v1 accepts even a not-yet-executed malformed literal.
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			before := current(t, s, "B")
			if _, err := s.Commit(t.Context(), before.Authority(), ""); err == nil {
				t.Fatal("malformed interpreted claim committed")
			}
			if !reflect.DeepEqual(before, current(t, s, "B")) {
				t.Fatal("rejected claim changed projection")
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			// v2 retains the new literal-claim preflight.
			v2 := bindingV2Fixture(t, nil)
			authorityReplaceProgram(t, &v2, "B", func(p *Program) {
				p.Code[1].Effect.Payload["claim"] = Lit(Text(strings.ReplaceAll(tc.claim, "v2", "v1")))
			})
			rejected, err := Create(t.Context(), filepath.Join(t.TempDir(), "v2.sqlite"), v2, DefaultLimits())
			if rejected != nil {
				rejected.Close()
			}
			if err == nil {
				t.Fatal("v2 invalid literal accepted")
			}
		})
	}
}
