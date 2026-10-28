//go:build integration

package adaptivetest

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	_ "github.com/mattn/go-sqlite3"
)

// probe is explicitly test-only and source-bound. Commit's observer still holds
// Store.mu, so calling public Snapshot there would deadlock. This connection can
// only inspect candidate projections; full public Inspect validates every cut.
type probe struct {
	db                *sql.DB
	template, subject string
}

func newProbe(path, template string) (*probe, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro&_query_only=1&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var q int
	if err = db.QueryRow("PRAGMA query_only").Scan(&q); err != nil || q != 1 {
		db.Close()
		return nil, fmt.Errorf("read-only probe query_only=%d: %v", q, err)
	}
	return &probe{db: db, template: template}, nil
}
func (p *probe) candidate(ctx context.Context, e boundary.Event, cut string) (bool, error) {
	if p.subject == "" && e.Boundary == "invocation.admitted" {
		err := p.db.QueryRowContext(ctx, "SELECT m.subject FROM composition_members m JOIN composition_invocations i ON m.invocation_id=i.id WHERE json_extract(i.descriptor,'$.TemplateID')=? ORDER BY m.rowid LIMIT 1", p.template).Scan(&p.subject)
		if err != nil && err != sql.ErrNoRows {
			return false, err
		}
	}
	if e.Boundary == "runner.waiting" {
		return cut == "capacity" || cut == "unknown", nil
	}
	if e.Boundary != "execution.committed" || e.Subject != p.subject || p.subject == "" {
		return false, nil
	}
	var raw string
	var seq, epoch int64
	if err := p.db.QueryRowContext(ctx, "SELECT sequence,epoch,projection FROM runs WHERE subject=? AND generation=?", e.Subject, e.Generation).Scan(&seq, &epoch, &raw); err != nil {
		return false, err
	}
	if seq != e.Sequence || epoch != e.Epoch {
		return false, fmt.Errorf("candidate source moved")
	}
	var q runtime.Projection
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		return false, err
	}
	switch cut {
	case "decision":
		d := rows(q, "decision")
		return len(d) == 1 && d[0]["round"].Number == 1 && reflect.DeepEqual(valueIDs(d[0]["candidate"]), []int64{7, 10, 11}) && q.Data["rounds"].Number == 1, nil
	case "intents":
		return e.Intent != "" && q.Data["rounds"].Number == 1 && q.Data["work_id"].Number == 7 && reflect.DeepEqual(ids(q, "current"), []int64{7, 10, 11}), nil
	case "partial":
		return reflect.DeepEqual(ids(q, "results"), []int64{1, 2, 3, 7}) && len(rows(q, "round_results")) == 1, nil
	case "denial":
		return reflect.DeepEqual(ids(q, "results"), []int64{1, 2, 3, 7, 10}), nil
	case "unknown":
		return len(rows(q, "results")) == 2 && rows(q, "results")[1]["status"].Text == "failed", nil
	}
	return false, nil
}
func cutMonitor(s runtime.Inspection, template, cut string, e boundary.Event) error {
	_, subject := adaptiveMember(s, template)
	q := projection(s, subject)
	ops := itemOperations(s, template)
	switch cut {
	case "decision":
		d := rows(q, "decision")
		if len(d) != 1 || d[0]["round"].Number != 1 || !reflect.DeepEqual(valueIDs(d[0]["candidate"]), []int64{7, 10, 11}) || len(ops) != 3 || q.Data["work_count"].Number != 3 || !reflect.DeepEqual(ids(q, "results"), []int64{1, 2, 3}) {
			return fmt.Errorf("wrong decision cut")
		}
		for _, op := range ops {
			if !consumed(s, op, subject) {
				return fmt.Errorf("round input not consumed")
			}
		}
	case "intents":
		if len(ops) != 4 || ops[7] != e.Intent || ops[10] != "" || ops[11] != "" || q.Data["work_count"].Number != 6 || q.Data["trial"].Number != 6 || !reflect.DeepEqual(ids(q, "current"), []int64{7, 10, 11}) {
			return fmt.Errorf("wrong partial-intent/whole-count cut")
		}
	case "partial":
		if !reflect.DeepEqual(ids(q, "results"), []int64{1, 2, 3, 7}) || len(ops) != 6 || !consumed(s, ops[7], subject) {
			return fmt.Errorf("partial input not consumed")
		}
		unfinished := 0
		for _, id := range []int64{10, 11} {
			if _, ok := outcome(s, ops[id]); !ok && admitted(s, ops[id]) {
				unfinished++
			}
		}
		if unfinished == 0 {
			return fmt.Errorf("required sibling already finished")
		}
		d := rows(q, "decision")
		if len(d) != 1 || d[0]["round"].Number != 1 {
			return fmt.Errorf("premature second decision")
		}
	case "denial":
		if q.Data["work_count"].Number != 6 || len(rows(q, "results")) != 5 || rows(q, "results")[4]["status"].Text != "rejected" || rows(q, "results")[4]["reason"].Text != "invocation_limit" {
			return fmt.Errorf("wrong next-frontier denial cut")
		}
	case "unknown":
		return pendingMonitor(s, template)
	case "capacity":
		if err := progressMonitor(s); err != nil {
			return err
		}
		var limit struct{ Limits struct{ MaxInFlight int64 } }
		if len(s.Tables["composition_roots"]) != 1 {
			return fmt.Errorf("root profile absent")
		}
		if err := jsonUnmarshal(s.Tables["composition_roots"][0]["descriptor"], &limit); err != nil {
			return err
		}
		if limit.Limits.MaxInFlight == 2 {
			known := 0
			for _, op := range ops {
				if _, ok := outcome(s, op); ok {
					known++
				}
			}
			if known != 2 {
				return fmt.Errorf("independent siblings not completed yet")
			}
		}
		if len(ops) != 3 || len(rows(q, "decision")) != 0 {
			return fmt.Errorf("logical frontier did not progress independently")
		}
		for _, op := range ops {
			if !admitted(s, op) {
				return fmt.Errorf("frontier admission absent")
			}
		}
	default:
		return fmt.Errorf("unknown cut %q", cut)
	}
	return origins(s)
}
func TestAdaptiveProcessChild(t *testing.T) {
	path := os.Getenv("ALAN_H_DB")
	if path == "" {
		t.Skip("subprocess entry")
	}
	ctx, err := boundary.FromEnvironment(t.Context(), path)
	must(t, err)
	if os.Getenv("ALAN_H_REPLAY") == "1" {
		_, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
		must(t, err)
		return
	}
	template, cut := os.Getenv("ALAN_H_TEMPLATE"), os.Getenv("ALAN_H_CUT")
	if cut != "" {
		p, err := newProbe(path, template)
		must(t, err)
		defer p.db.Close()
		inherited := ctx
		ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
			// Forward tracking events even when they are not candidate pause boundaries.
			if e.Boundary == "adapter.authorized" || e.Boundary == "adapter.finished" || e.Boundary == "external.authorized" || e.Boundary == "external.finished" {
				return boundary.Reached(inherited, e)
			}
			ok, err := p.candidate(ctx, e, cut)
			if err != nil || !ok {
				return err
			}
			snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
			if err != nil {
				return err
			}
			if cutMonitor(snapshot, template, cut, e) != nil {
				return nil
			}
			return boundary.Reached(inherited, e)
		})
	}
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	must(t, err)
	defer s.Close()
	// Only the live Store owner admits the parent-triggered environmental input.
	// This goroutine cannot reduce, dispatch or derive application work.
	var inputDone chan error
	if cut == "capacity" {
		inputDone = make(chan error, 1)
		trigger := os.NewFile(5, "post-contact-progress")
		defer trigger.Close()
		go func() {
			var signal [1]byte
			_, err := io.ReadFull(trigger, signal[:])
			if err == nil && signal[0] != 'p' {
				err = fmt.Errorf("invalid progress trigger")
			}
			if err == nil {
				var a runtime.AdmissionResult
				a, err = s.AdmitInput(ctx, runtime.Input{TransportID: "post-contact-progress", LogicalID: "post-contact-progress", Target: "progress", Generation: 1, Obligation: "start", Payload: map[string]schema.Value{}})
				if err == nil && !a.Accepted {
					err = fmt.Errorf("progress input rejected: %v", a)
				}
			}
			inputDone <- err
		}()
	}
	registry := map[string]effects.Adapter{}
	if ep := os.Getenv("ALAN_H_ENDPOINT"); ep != "" {
		registry[adaptive.ScalarCapability] = httpAdapter{ep}
	}
	report, err := s.Run(ctx, runtime.RunOptions{Adapters: registry, MaxSteps: 30000, MaxDispatch: 3000, MaxInFlight: 2, ExternalTimeout: raceDuration(10 * time.Minute)})
	must(t, err)
	if inputDone != nil {
		must(t, <-inputDone)
	}
	emit(t, map[string]any{"kind": "child_run", "report": report})
}

type process struct {
	cmd               *exec.Cmd
	ack, release, log *os.File
	progress          *os.File
	filter            boundary.Event
	stopped           bool
	network           string
}

func start(t *testing.T, path, template, cut, endpoint string, replay bool, binary string) *process {
	t.Helper()
	ackR, ackW, err := os.Pipe()
	must(t, err)
	relR, relW, err := os.Pipe()
	must(t, err)
	log, err := os.CreateTemp(t.TempDir(), "child-")
	must(t, err)
	if binary == "" {
		binary, err = os.Executable()
		must(t, err)
	}
	exe := binary
	args := []string{"-test.run=^TestAdaptiveProcessChild$", "-test.v", "-test.timeout=" + raceDuration(12*time.Minute).String()}
	network := "not requested"
	if replay {
		sandbox, err := exec.LookPath("sandbox-exec")
		if err != nil {
			t.Skip("required OS network denial unavailable: sandbox-exec absent")
		}
		binary = sandbox
		args = append([]string{"-p", "(version 1)(allow default)(deny network*)", exe}, args...)
		network = "macOS sandbox-exec deny network*"
	}
	cmd := exec.Command(binary, args...)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "ALAN_H_") && !strings.HasPrefix(v, "ALAN_BARRIER=") && !strings.HasPrefix(v, "OPENAI_API_KEY=") {
			cmd.Env = append(cmd.Env, v)
		}
	}
	cmd.Env = append(cmd.Env, "ALAN_H_DB="+path, "ALAN_H_TEMPLATE="+template, "ALAN_H_CUT="+cut, "ALAN_H_ENDPOINT="+endpoint)
	f := boundary.Event{Store: path, Scenario: t.Name(), Token: fmt.Sprintf("%s/%s/%d", t.Name(), cut, time.Now().UnixNano()), Boundary: "execution.committed", Phase: "post-commit"}
	if cut == "capacity" {
		f.Boundary = "runner.waiting"
		f.Phase = "in-flight"
	}
	if cut != "" {
		raw, _ := json.Marshal(f)
		cmd.Env = append(cmd.Env, "ALAN_BARRIER="+string(raw))
	}
	if replay {
		cmd.Env = append(cmd.Env, "ALAN_H_REPLAY=1", "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1")
	}
	cmd.ExtraFiles = []*os.File{ackW, relR}
	var progressR, progressW *os.File
	if cut == "capacity" {
		progressR, progressW, err = os.Pipe()
		must(t, err)
		cmd.ExtraFiles = append(cmd.ExtraFiles, progressR)
	}

	cmd.Stdout = log
	cmd.Stderr = log
	must(t, cmd.Start())
	ackW.Close()
	relR.Close()
	if progressR != nil {
		progressR.Close()
	}
	p := &process{cmd: cmd, ack: ackR, release: relW, log: log, filter: f, network: network, progress: progressW}
	t.Cleanup(func() {
		if !p.stopped {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		ackR.Close()
		relW.Close()
		log.Close()
		if progressW != nil {
			progressW.Close()
		}
	})
	raw, err := os.ReadFile(exe)
	must(t, err)
	emit(t, map[string]any{"kind": "process_start", "pid": cmd.Process.Pid, "argv": cmd.Args, "binary_sha256": rawHash(raw), "filter": f, "network_isolation": network, "credential_absent": true, "race_instrumented": raceEnabled(), "watchdog_scale": int64(raceDuration(time.Second) / time.Second), "child_timeout": raceDuration(12 * time.Minute).String()})
	return p
}
func (p *process) acknowledgment(t *testing.T) boundary.Event {
	t.Helper()
	var e boundary.Event
	done := make(chan error, 1)
	go func() { done <- json.NewDecoder(p.ack).Decode(&e) }()
	select {
	case err := <-done:
		if err != nil {
			raw, _ := os.ReadFile(p.log.Name())
			t.Fatal(err, string(raw))
		}
	case <-time.After(raceDuration(5 * time.Minute)):
		t.Fatal("watchdog: boundary absent")
	}
	must(t, validAck(e, p.filter, p.cmd.Process.Pid, map[string]bool{}))
	return e
}
func (p *process) wait(t *testing.T, kill bool) {
	t.Helper()
	if kill {
		must(t, p.cmd.Process.Signal(syscall.SIGKILL))
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case err := <-done:
		p.stopped = true
		raw, _ := os.ReadFile(p.log.Name())
		status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		emit(t, map[string]any{"kind": "process_exit", "pid": p.cmd.Process.Pid, "exit_code": p.cmd.ProcessState.ExitCode(), "signal": status.Signal().String(), "output": string(raw)})
		if kill {
			if err == nil || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatal("not SIGKILL", err)
			}
		} else {
			must(t, err)
		}
	case <-time.After(raceDuration(12 * time.Minute)):
		t.Fatal("watchdog: child did not exit")
	}
}
func (p *process) resume(t *testing.T) {
	t.Helper()
	_, err := p.release.Write([]byte{'c'})
	must(t, err)
}
func retain(t *testing.T, name string, v any) {
	t.Helper()
	folder := os.Getenv("ALAN_H_ARTIFACTS")
	if folder == "" {
		folder = t.TempDir()
	}
	folder = filepath.Join(folder, strings.NewReplacer("/", "_", "=", "-").Replace(t.Name()))
	must(t, os.MkdirAll(folder, 0755))
	path := filepath.Join(folder, name+".json.gz")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	must(t, err)
	z := gzip.NewWriter(f)
	must(t, json.NewEncoder(z).Encode(v))
	must(t, z.Close())
	must(t, f.Close())
	raw, err := os.ReadFile(path)
	must(t, err)
	emit(t, map[string]any{"kind": "retained_artifact", "path": path, "gzip_sha256": rawHash(raw), "value_sha256": digest(v)})
}
