package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	recordschema "github.com/JakeFAU/alan/schema"
)

func TestCreateRejectsProvisionBeforeTouchingDestination(t *testing.T) {
	for _, name := range []string{"permission", "schema", "artifact"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "worker.sqlite")
			p := fixture(t, false)
			switch name {
			case "permission":
				p.Permissions = []Permission{{Kind: "adaptr"}}
			case "schema":
				empty, err := (recordschema.Record{}).Retain()
				if err != nil {
					t.Fatal(err)
				}
				program := Program{Semantics: AuthorSemanticVersion, Code: []Instruction{{Op: "end", Outcome: "succeeded"}}, Author: &ProgramMetadata{Compiler: AuthorCompilerVersion, Inputs: empty, Outputs: empty, Positions: []string{"end"}}}
				if _, _, err := EncodeProgram(program); err != nil {
					t.Fatal(err)
				}
				p.Programs[0].Semantics = AuthorSemanticVersion
				program.Author.Inputs.ID = "invalid"
				raw, err := json.Marshal(program)
				if err != nil {
					t.Fatal(err)
				}
				p.Programs[0].Bytes = raw
				p.Programs[0].ID = programHash(raw)
				p.Runs[0].ProgramID = p.Programs[0].ID
			case "artifact":
				p.Programs[0].ID = "invalid"
			}
			if s, err := Create(t.Context(), path, p, DefaultLimits()); !errors.Is(err, os.ErrInvalid) || s != nil || errors.Is(err, ErrCorrupt) {
				t.Fatalf("invalid provision returned store=%v err=%v", s != nil, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("initial validation wrote %v, err=%v", entries, err)
			}
			s, err := Create(t.Context(), path, fixture(t, false), DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			assertReopenCreatedStore(t, path)
		})
	}
}

func assertReopenCreatedStore(t *testing.T, path string) {
	t.Helper()
	report, err := Replay(t.Context(), path, DefaultLimits())
	if err != nil || report.Namespace == "" || len(report.Runs) != 1 || report.Runs[0].Epoch != 1 {
		t.Fatalf("initial replay: %+v, %v", report, err)
	}
	s, err := Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := Replay(t.Context(), path, DefaultLimits())
	if err != nil || after.Namespace != report.Namespace || after.Runs[0].Epoch != 2 || after.HistoryCount != report.HistoryCount {
		t.Fatalf("reopened replay: %+v, %v", after, err)
	}
}

func TestCreateConcurrentAndExistingStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worker.sqlite")
	p := fixture(t, true)
	start := make(chan struct{})
	type result struct {
		owner *Store
		err   error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			s, err := Create(t.Context(), path, p, DefaultLimits())
			results <- result{s, err}
		}()
	}
	close(start)
	a, b := <-results, <-results
	if b.err == nil {
		a, b = b, a
	}
	if a.err != nil || a.owner == nil || b.owner != nil || !errors.Is(b.err, ErrLocked) {
		t.Fatalf("competing results: %v, %v", a.err, b.err)
	}
	if err := a.owner.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Create(t.Context(), path, p, DefaultLimits()); !errors.Is(err, os.ErrExist) {
		t.Fatalf("existing store: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("changed existing store: %v", err)
	}
	assertReopenCreatedStore(t, path)
}

func TestInterruptedCreate(t *testing.T) {
	if phase := os.Getenv("ALAN_TEST_CREATION_CUT"); phase != "" {
		ctx := boundary.WithObserver(t.Context(), func(_ context.Context, e boundary.Event) error {
			if e.Boundary == "store.create" && e.Phase == phase {
				if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
					return err
				}
				select {}
			}
			return nil
		})
		_, err := Create(ctx, os.Getenv("ALAN_TEST_CREATION_PATH"), fixture(t, true), DefaultLimits())
		t.Fatalf("missed %s: %v", phase, err)
	}
	for _, phase := range []string{"validated", "before-commit", "committed"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "worker.sqlite")
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInterruptedCreate$")
			child.Env = append(os.Environ(), "ALAN_TEST_CREATION_CUT="+phase, "ALAN_TEST_CREATION_PATH="+path)
			out, err := child.CombinedOutput()
			var exited *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exited) || exited.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("process missed %s: %v: %s", phase, err, out)
			}
			switch phase {
			case "validated":
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("path exists before reservation", err)
				}
				s, err := Create(t.Context(), path, fixture(t, true), DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			case "before-commit":
				if _, err := Create(t.Context(), path, fixture(t, true), DefaultLimits()); !errors.Is(err, os.ErrExist) {
					t.Fatalf("replaced interrupted file: %v", err)
				}
				if s, err := Open(t.Context(), path, DefaultLimits()); s != nil || err == nil {
					t.Fatal("opened incomplete store", err)
				}
				return
			case "committed":
				if _, err := Create(t.Context(), path, fixture(t, true), DefaultLimits()); !errors.Is(err, os.ErrExist) {
					t.Fatalf("replaced committed file: %v", err)
				}
			}
			assertReopenCreatedStore(t, path)
		})
	}
}

// SQLite reserves these neighbors for its own journals. A store or file that
// happens to have one of these names must not be consumed during creation.
func TestCreatePreservesExistingSQLiteSidecars(t *testing.T) {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run(suffix, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "worker.sqlite")
			neighbor := path + suffix
			other, err := Create(t.Context(), neighbor, fixture(t, true), DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := other.Close(); err != nil {
				t.Fatal(err)
			}
			if store, err := Create(t.Context(), path, fixture(t, true), DefaultLimits()); store != nil || !errors.Is(err, os.ErrExist) {
				if store != nil {
					store.Close()
				}
				t.Fatalf("accepted occupied SQLite sidecar: %v", err)
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("claimed main path despite occupied sidecar: %v", err)
			}
			assertReopenCreatedStore(t, neighbor)
		})
	}
}

// Probe the same directory as the fixture. Filename comparison follows the
// volume's rules, including Unicode aliases on some case-insensitive volumes.
func sidecarAliases(t *testing.T, dir, spelling, suffix string) bool {
	t.Helper()
	exemplar := filepath.Join(dir, "probe"+suffix)
	if err := os.WriteFile(exemplar, nil, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(filepath.Join(dir, "probe"+spelling))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

var sqliteNeighborNames = []struct{ spelling, suffix string }{
	{"-wal", "-wal"}, {"-WAL", "-wal"}, {"-sHm", "-shm"}, {"-ſhm", "-shm"}, {"-JOURNAL", "-journal"},
}

func TestCreateAfterClosedSQLiteNeighbor(t *testing.T) {
	for _, n := range sqliteNeighborNames {
		t.Run(n.spelling, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			path := filepath.Join(dir, "worker.sqlite")
			alias := sidecarAliases(t, dir, n.spelling, n.suffix)
			base, err := Create(ctx, path, fixture(t, true), DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := base.Close(); err != nil {
				t.Fatal(err)
			}
			other, err := Create(ctx, path+n.spelling, fixture(t, true), DefaultLimits())
			if alias {
				if other != nil {
					other.Close()
				}
				if !errors.Is(err, os.ErrExist) || other != nil {
					t.Fatalf("accepted occupied journal alias: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if err := other.Close(); err != nil {
					t.Fatal(err)
				}
				assertReopenCreatedStore(t, path+n.spelling)
			}
			assertReopenCreatedStore(t, path)
		})
	}
}

func TestCreateRacesSQLiteNeighbor(t *testing.T) {
	for _, n := range sqliteNeighborNames {
		t.Run(n.spelling, func(t *testing.T) {
			dir := t.TempDir()
			if sidecarAliases(t, dir, n.spelling, n.suffix) {
				raceSQLiteNeighbor(t, filepath.Join(dir, "worker.sqlite"), n.spelling)
			}
		})
	}
}

func raceSQLiteNeighbor(t *testing.T, path, suffix string) {
	t.Helper()
	ctx := t.Context()
	p := fixture(t, true)
	start := make(chan struct{})
	type result struct {
		path  string
		owner *Store
		err   error
	}
	results := make(chan result, 2)
	for _, target := range []string{path, path + suffix} {
		go func() {
			<-start
			s, err := Create(ctx, target, p, DefaultLimits())
			results <- result{target, s, err}
		}()
	}
	close(start)
	a, b := <-results, <-results
	if b.err == nil {
		a, b = b, a
	}
	if a.owner == nil || a.err != nil || b.owner != nil || !(errors.Is(b.err, ErrLocked) || errors.Is(b.err, os.ErrExist)) {
		t.Fatalf("overlapping SQLite names returned %s: %v, %s: %v", a.path, a.err, b.path, b.err)
	}
	if err := a.owner.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, b.path, p, DefaultLimits()); !errors.Is(err, os.ErrExist) {
		t.Fatalf("second creation after owner close: %v", err)
	}
	assertReopenCreatedStore(t, a.path)
}
