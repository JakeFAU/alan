//go:build integration

package author_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalConsumerCompileRunReopenReplay(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source, err := os.ReadFile("testdata/consumer.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "/internal/") {
		t.Fatal("consumer has internal import")
	}
	if err = os.WriteFile(filepath.Join(dir, "main.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	mod := "module external.example/consumer\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + root + "\n"
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "consumer")
	args := []string{"build", "-mod=mod", "-o", binary}
	if raceChildren {
		args = append(args, "-race")
	}
	args = append(args, ".")
	t.Logf("external child go %v", args)
	build := exec.Command("go", args...)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("external build: %v\n%s", err, out)
	}
	path := filepath.Join(dir, "consumer.db")
	for _, mode := range []string{"create", "resume", "replay"} {
		cmd := exec.Command(binary, mode, path)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", mode, err, out)
		} else {
			t.Logf("%s: %s", mode, out)
		}
	}
}
