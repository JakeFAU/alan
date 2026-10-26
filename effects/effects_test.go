package effects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotRetainsBytesAfterSourceDeletion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "source.txt")
	if e := os.WriteFile(p, []byte("retained"), 0600); e != nil {
		t.Fatal(e)
	}
	a, e := Snapshot("doc", p)
	if e != nil {
		t.Fatal(e)
	}
	os.Remove(p)
	if a.Content != "retained" || a.Check() != nil {
		t.Fatal(a)
	}
	a.Content = "changed"
	if a.Check() == nil {
		t.Fatal("changed bytes accepted")
	}
}
func TestPreparedDigestRejectsChangedWire(t *testing.T) {
	p := Prepared{Protocol: "test", MaxResponseBytes: 1024, Wire: "original"}
	p = Freeze(p)
	if CheckPrepared(p) != nil {
		t.Fatal("frozen rejected")
	}
	p.Wire = "changed"
	if CheckPrepared(p) == nil {
		t.Fatal("changed wire accepted")
	}
}
func TestAssetIdentityIsBounded(t *testing.T) {
	if _, e := NewAsset(string(make([]byte, 513)), "source", []byte("x")); e == nil {
		t.Fatal("unbounded document identity accepted")
	}
}
