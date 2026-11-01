//go:build integration

package durabletest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/executor"
	resource "github.com/JakeFAU/alan/internal/fixture"
	publicruntime "github.com/JakeFAU/alan/runtime"
)

// TestPhase2ExecutorProcess is an actual freshly exec'd process. In race runs the
// same instrumented test binary is used, rather than a non-race helper binary.
func TestPhase2ExecutorProcess(t *testing.T) {
	if os.Getenv("ALAN_PHASE2_CHILD") != "1" {
		return
	}
	db := os.Getenv("ALAN_PHASE2_DB")
	ctx, err := boundary.FromEnvironment(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	s, err := publicruntime.Open(ctx, db, publicruntime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	report, err := s.Run(ctx, publicruntime.RunOptions{Adapters: effectboundary.Registry(os.Getenv("ALAN_PHASE2_ENDPOINT"), os.Getenv("ALAN_PHASE2_MODE")), MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1, ExternalTimeout: 30 * time.Second})
	raw, _ := json.Marshal(map[string]any{"PID": os.Getpid(), "Report": report, "Error": fmt.Sprint(err)})
	fmt.Println("PHASE2_RESULT " + string(raw))
	if err != nil {
		t.Fatal(err)
	}
}

func phaseEvidenceDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("ALAN_PHASE2_EVIDENCE")
	if dir == "" {
		dir = filepath.Join(binaries(t), "phase2")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}
func phaseName(t *testing.T) string { return strings.ReplaceAll(t.Name(), "/", "_") }

func phaseProvenance(t *testing.T, cmd *exec.Cmd) map[string]any {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(cwd, "../.."))
	sources := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "evidence" || entry.Name() == "consumer") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !(strings.HasSuffix(path, ".go") || filepath.Base(path) == "go.mod" || filepath.Base(path) == "go.sum") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sources[relative] = fmt.Sprintf("%x", sha256.Sum256(raw))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(hash, "%s  %s\n", sources[key], key)
	}
	digest := fmt.Sprintf("%x", hash.Sum(nil))
	phaseEmit(t, "sources-"+digest, sources)
	build, _ := debug.ReadBuildInfo()
	environment := map[string]string{"CGO_ENABLED": os.Getenv("CGO_ENABLED"), "GOCACHE": os.Getenv("GOCACHE"), "GOMODCACHE": os.Getenv("GOMODCACHE"), "GOPROXY": os.Getenv("GOPROXY"), "GOFLAGS": os.Getenv("GOFLAGS"), "ALAN_PHASE2_EVIDENCE": os.Getenv("ALAN_PHASE2_EVIDENCE")}
	for _, entry := range cmd.Env {
		key, value, found := strings.Cut(entry, "=")
		if found && (strings.HasPrefix(key, "ALAN_PHASE2_") || key == "ALAN_BARRIER") {
			environment[key] = value
		}
	}
	return map[string]any{"PID": cmd.Process.Pid, "cwd": cwd, "argv": cmd.Args, "executable_sha256": binaryHash(cmd.Path), "source_manifest_sha256": digest, "build_info": build, "environment": environment}
}
func phaseEmit(t *testing.T, suffix string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(phaseEvidenceDir(t), phaseName(t)+"-"+suffix+".json")
	if err = os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("PHASE2_EVIDENCE %s sha256=%x", path, sha256.Sum256(append(raw, '\n')))
}

// phaseRawStore takes a coherent SQLite backup, then gzip archives its exact
// bytes. It neither checkpoints nor changes the source connection or rows.
func phaseRawStore(t *testing.T, path, label string) string {
	t.Helper()
	out := filepath.Join(phaseEvidenceDir(t), phaseName(t)+"-"+label+".sqlite.gz")
	script := `import sqlite3,sys,tempfile,gzip,os
source=sqlite3.connect('file:'+sys.argv[1]+'?mode=ro',uri=True)
fd,path=tempfile.mkstemp();os.close(fd)
dest=sqlite3.connect(path);source.backup(dest);dest.close();source.close()
with open(path,'rb') as f: raw=f.read()
with open(sys.argv[2],'wb') as f: f.write(gzip.compress(raw,mtime=0))
os.unlink(path)
`
	cmd := exec.Command("python3", "-c", script, path, out)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("backup: %v %s", err, data)
	}
	return out
}

func phaseResource(t *testing.T, path, address, mode string, filter *barrier) (*child, string, string) {
	t.Helper()
	if address == "" {
		address = "127.0.0.1:0"
	}
	if mode == "" {
		mode = "normal"
	}
	token := fmt.Sprintf("phase2-endpoint-%d", time.Now().UnixNano())
	journal := path + ".ingress.jsonl"
	cmd := exec.Command(filepath.Join(binaries(t), "alan-resource"), "--db", path, "--token", token, "--listen", address, "--journal", journal, "--response-mode", mode)
	config := barrier{Scenario: t.Name(), Token: token, Store: path}
	if filter != nil {
		config = *filter
		config.Scenario = t.Name()
		config.Token = token
		config.Store = path
		raw, _ := json.Marshal(config)
		cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resumeRead, resumeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.CreateTemp(binaries(t), "phase2-endpoint-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	write.Close()
	resumeRead.Close()
	c := &child{cmd: cmd, ack: read, release: resumeWrite, config: config, stderr: log}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		read.Close()
		resumeWrite.Close()
		log.Close()
	})
	type ready struct {
		URL, Token, Store string
		PID               int
	}
	done := make(chan ready, 1)
	go func() { var r ready; json.NewDecoder(stdout).Decode(&r); done <- r }()
	select {
	case r := <-done:
		if r.PID != cmd.Process.Pid || r.Token != token || r.Store != path || r.URL == "" {
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("endpoint identity %+v %s", r, data)
		}
		phaseEmit(t, token, map[string]any{"command": cmd.Args, "ready": r, "binary_sha256": binaryHash(cmd.Path), "race_children": raceChildren})
		phaseEmit(t, token+"-provenance", phaseProvenance(t, cmd))
		return c, r.URL, journal
	case <-time.After(30 * time.Second):
		t.Fatal("endpoint readiness missing")
	}
	return nil, "", ""
}
func phaseStopEndpoint(t *testing.T, c *child) {
	t.Helper()
	if err := c.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err := c.cmd.Wait()
	status, ok := c.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("endpoint stop %v", err)
	}
	phaseEmit(t, fmt.Sprintf("endpoint-stopped-%d", c.cmd.Process.Pid), map[string]any{"command": c.cmd.Args, "PID": c.cmd.Process.Pid, "signal": status.Signal().String(), "binary_sha256": binaryHash(c.cmd.Path)})
}
func phaseProcess(t *testing.T, db, endpoint, mode string, filter *barrier) *child {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestPhase2ExecutorProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "ALAN_PHASE2_CHILD=1", "ALAN_PHASE2_DB="+db, "ALAN_PHASE2_ENDPOINT="+endpoint, "ALAN_PHASE2_MODE="+mode)
	config := barrier{}
	if filter != nil {
		config = *filter
		config.Store = db
		config.Scenario = t.Name()
		config.Token = fmt.Sprintf("phase2-runner-%d", time.Now().UnixNano())
		raw, _ := json.Marshal(config)
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resumeRead, resumeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{write, resumeRead}
	log, err := os.CreateTemp(binaries(t), "phase2-runner-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	phaseEmit(t, fmt.Sprintf("process-start-%d", cmd.Process.Pid), phaseProvenance(t, cmd))
	write.Close()
	resumeRead.Close()
	c := &child{cmd: cmd, ack: read, release: resumeWrite, config: config, stderr: log}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		read.Close()
		resumeWrite.Close()
		log.Close()
	})
	return c
}
func phaseWait(t *testing.T, c *child) barrier {
	t.Helper()
	type result struct {
		b   barrier
		err error
	}
	done := make(chan result, 1)
	go func() { var b barrier; err := json.NewDecoder(c.ack).Decode(&b); done <- result{b, err} }()
	select {
	case got := <-done:
		if got.err != nil {
			raw, _ := os.ReadFile(c.stderr.Name())
			t.Fatalf("boundary missing: %v %s", got.err, raw)
		}
		b := got.b
		want := c.config
		if b.PID != c.cmd.Process.Pid || b.Token != want.Token || b.Store != want.Store || b.Scenario != want.Scenario || b.Boundary != want.Boundary || b.Phase != want.Phase || (want.Subject != "" && want.Subject != b.Subject) {
			t.Fatalf("boundary mismatch %+v want %+v", b, want)
		}
		if strings.HasPrefix(b.Boundary, "resource.") {
			if b.Boundary == "resource.applied" {
				snapshot, err := resource.Inspect(t.Context(), b.Store)
				if err != nil || len(snapshot.Applications) == 0 {
					t.Fatalf("missing committed application: %v", err)
				}
			}
		} else {
			validateBarrierFacts(t, b)
		}
		c.acknowledged = true
		return b
	case <-time.After(30 * time.Second):
		raw, _ := os.ReadFile(c.stderr.Name())
		t.Fatalf("boundary timed out %s", raw)
	}
	return barrier{}
}
func phaseFinish(t *testing.T, c *child) {
	t.Helper()
	err := c.cmd.Wait()
	raw, _ := os.ReadFile(c.stderr.Name())
	phaseEmit(t, fmt.Sprintf("process-%d", c.cmd.Process.Pid), map[string]any{"command": c.cmd.Args, "PID": c.cmd.Process.Pid, "exit_code": c.cmd.ProcessState.ExitCode(), "output": string(raw), "binary_sha256": binaryHash(c.cmd.Path), "race_children": raceChildren})
	if err != nil {
		t.Fatalf("fresh executor %v %s", err, raw)
	}
}
func phaseReplay(t *testing.T, db, world, label string) {
	t.Helper()
	before := inspect(t, db)
	beforeLocalStore := phaseRawStore(t, db, "replay-"+label+"-before-local")
	beforeWorldStore := phaseRawStore(t, world, "replay-"+label+"-before-world")
	beforeIngress, ingressErr := os.ReadFile(world + ".ingress.jsonl")
	endpoint, err := resource.Inspect(t.Context(), world)
	if err != nil {
		t.Fatal(err)
	}
	report, err := executor.Replay(t.Context(), db, executor.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	after := inspect(t, db)
	end, err := resource.Inspect(t.Context(), world)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(endpoint, end) {
		t.Fatal("replay mutated local or external ledger")
	}
	afterIngress, afterIngressErr := os.ReadFile(world + ".ingress.jsonl")
	if !reflect.DeepEqual(beforeIngress, afterIngress) || (ingressErr == nil) != (afterIngressErr == nil) {
		t.Fatal("replay added endpoint ingress")
	}
	phaseEmit(t, "replay-"+label, map[string]any{"before": before, "after": after, "endpoint_before": endpoint, "endpoint_after": end, "report": report, "before_local_store": beforeLocalStore, "before_world_store": beforeWorldStore, "after_local_store": phaseRawStore(t, db, "replay-"+label+"-after-local"), "after_world_store": phaseRawStore(t, world, "replay-"+label+"-after-world"), "ingress_before": string(beforeIngress), "ingress_after": string(afterIngress)})
}
func phaseCopy(t *testing.T, source, dest string) {
	t.Helper()
	in, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
}
func phaseAddress(t *testing.T, endpoint string) string {
	t.Helper()
	u, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}
