// Package storetest provides disposable copies of historical SQLite inputs for compatibility tests.
package storetest

import (
	"compress/gzip"
	"embed"
	"io"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/*.sqlite.gz
var stores embed.FS

// Copy expands the named frozen store into a new test directory.
func Copy(t testing.TB, name string) string {
	t.Helper()
	source, err := stores.Open("testdata/" + name + ".sqlite.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	reader, err := gzip.NewReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	path := filepath.Join(t.TempDir(), name+".sqlite")
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(out, reader)
	closeErr := out.Close()
	if err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
	return path
}
