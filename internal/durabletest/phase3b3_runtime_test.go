//go:build integration

package durabletest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	sqlite "github.com/mattn/go-sqlite3"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

type reciprocalCase struct {
	Version, Name, Schedule, Directory, Classification string
	Interpretation                                     string `json:",omitempty"`
	Manifest                                           delegation.Manifest
	Cut                                                *boundary.Event `json:",omitempty"`
}

var reciprocalIndexMu sync.Mutex

func reciprocalDirectory(t *testing.T, name string) string {
	t.Helper()
	root := os.Getenv("ALAN_PHASE3B3_EVIDENCE")
	if root == "" {
		return t.TempDir()
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, name+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("raw evidence:", dir)
	return dir
}
func reciprocalIndex(t *testing.T, c reciprocalCase) {
	t.Helper()
	switch c.Name {
	case "control-wrong-prior-selection":
		c.Interpretation = "Authentic BA request selected, then ready selected, then bound A4 service refused. Immediate-selection constraint is a runtime narrowing; unchanged model permits delayed service."
	case "ordinary-s3-missing":
		c.Interpretation = "An accepted grant exists, but A selects no authority. This is missing-selector refusal; before-grant prefix of first-refused-then-grant supplies genuine absent-grant evidence."
	case "ordinary-s3-claim":
		c.Interpretation = "An accepted grant exists, but A uses the actual request ID as GrantID. Request-shaped data supplies no authority."
	case "first-refused-then-grant":
		c.Interpretation = "Original effect refused while no grant exists; later accepted program grant and reopen do not repair that original."
	case "cut-04-complete-decision", "cut-05-before-admission":
		c.Interpretation = "These two requested labels coincide at dispatch.consider/pre-admission; they are not distinct physical boundaries."
	case "ordinary-s2-handshake":
		c.Interpretation = "Separate grant/ready handshake and distinct BA/AB action tuples. Mapping to unchanged LateGrant model requires an explicit whole-action abstraction."
	case "actual-prior-service-result-as-grant":
		c.Interpretation = "A consumes authentic B7 service result via B10 reply before A5 attempts to use that actual retained result operation as its grant. Reversed service policy control."
	}
	baseline3b3Write(t, c.Directory, "case.json", c)
	reciprocalSources(t, c.Directory)
	root := os.Getenv("ALAN_PHASE3B3_EVIDENCE")
	if root == "" {
		return
	}
	reciprocalIndexMu.Lock()
	defer reciprocalIndexMu.Unlock()
	path := filepath.Join(root, "case-index.json")
	var cases []reciprocalCase
	if raw, err := os.ReadFile(path); err == nil {
		if err = json.Unmarshal(raw, &cases); err != nil {
			t.Fatal(err)
		}
	}
	cases = append(cases, c)
	baseline3b3Write(t, root, "case-index.json", cases)
}
func reciprocalCopyDB(t *testing.T, source, dest string) {
	t.Helper()
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if _, err := os.Stat(source + suffix); err == nil {
			phaseCopy(t, source+suffix, dest+suffix)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
}
func reciprocalCapture(t *testing.T, path, dir string) *runtime.ExecutionSnapshot {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// The source owner has closed or been SIGKILLed. Inspect a complete disposable
	// copy, then archive exact stopped bytes without ever opening that archive.
	probe := filepath.Join(t.TempDir(), "probe.sqlite")
	reciprocalCopyDB(t, path, probe)
	raw, rawErr := runtime.Inspect(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "inspect.json", raw)
	s, typedErr := runtime.InspectExecution(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "typed-inspection.json", s)
	replay, replayErr := runtime.Replay(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "replay.json", map[string]any{"Report": replay, "Error": fmt.Sprint(replayErr)})
	baseline3b3Write(t, dir, "outcomes.json", map[string]string{"Inspect": fmt.Sprint(rawErr), "InspectExecution": fmt.Sprint(typedErr), "Replay": fmt.Sprint(replayErr)})
	reciprocalCopyDB(t, path, filepath.Join(dir, "runtime.sqlite"))
	if rawErr != nil || typedErr != nil || replayErr != nil {
		t.Fatal("snapshot validation", rawErr, typedErr, replayErr)
	}
	reciprocalAccounting(t, raw, s)
	cursor := 0
	for n := 0; ; n++ {
		page, err := s.History(cursor, 64)
		if err != nil {
			t.Fatal(err)
		}
		baseline3b3Write(t, dir, fmt.Sprintf("history-%03d.json", n), page)
		if !page.More {
			break
		}
		cursor = page.Next
	}
	return s
}

func reciprocalComplete(t *testing.T, s *runtime.ExecutionSnapshot, m delegation.Manifest, endpoints map[string]resource.Snapshot) {
	t.Helper()
	for _, id := range []runtime.RunID{m.A, m.B} {
		r, ok := s.Run(id)
		if !ok || r.Projection.Status != "succeeded" || !r.Projection.Data["paid"].Flag {
			t.Fatalf("original not complete %+v", r)
		}
	}
	for _, x := range []struct {
		request, service, reply string
		server, requester       runtime.RunID
		endpoint                string
	}{{m.BARequest, m.BAService, m.BAReply, m.A, m.B, "b"}, {m.ABRequest, m.ABService, m.ABReply, m.B, m.A, "a"}} {
		e, ok := s.Effect(x.service)
		if !ok || e.Source != x.server || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || e.ExternalReceipt == nil || e.Settlement == nil || len(e.AuthorityAdmissions) == 0 {
			t.Fatal("missing exact service", e)
		}
		w := e.ExternalAttempt.Authority
		server, _ := s.Run(x.server)
		if w == nil || !w.Allowed || w.Use == nil || w.ServingRootID != server.RootID {
			t.Fatal("wrong serving-root authority", w)
		}
		if x.server == m.A && m.Options.Scenario == delegation.S2Handshake {
			if w.Use.Path != "delegated" || w.Use.GrantID != m.Grant || w.Use.Root != m.BAServiceRoot || w.OriginalRequestID != m.BARequest || w.ServiceOperationID != m.BAService || w.RequestConsumedSequence != e.Sequence-1 || w.GrantDecisionSequence < 1 {
				t.Fatal("delegated custody witness", w)
			}
		}
		for i, g := range e.AuthorityAdmissions {
			if !g.Allowed || g.Ordinal != int64(i+1) || g.Source.Run != x.server || g.Namespace != s.Namespace || g.AttemptDigest == "" {
				t.Fatal("gate identity", g)
			}
		}
		var selected *runtime.ObservationView
		for _, o := range s.Observations {
			if o.Run == x.server && o.SourceOperationID == x.request && o.Kind == "message" {
				if selected != nil {
					t.Fatal("duplicate request")
				}
				selected = o
			}
		}
		if selected == nil || selected.ProducerRun == nil || *selected.ProducerRun != x.requester || selected.SelectedAt == nil || selected.SelectedAt.Run != x.server || selected.SelectedAt.Sequence != reciprocalServiceSequence(t, x.service)-1 || e.Sequence != selected.SelectedAt.Sequence+1 {
			t.Fatal("original request selection", selected, e)
		}
		result, ok := s.Observation(e.Settlement.ObservationID)
		if !ok || result.Run != x.server || result.Operation != x.service || result.SourceOperationID != x.service || result.SelectedAt == nil || result.SelectedAt.Run != x.server || result.SelectedAt.Sequence != e.Sequence+1 || result.Payload["mutation"].Number != e.ExternalReceipt.Mutation || result.Payload["total"].Number != e.ExternalReceipt.Total {
			t.Fatal("original result selection", result)
		}
		found := false
		for _, o := range s.Observations {
			if o.Run == x.requester && o.SourceOperationID == x.reply && o.Kind == "message" {
				if o.ProducerRun == nil || *o.ProducerRun != x.server || o.SelectedAt == nil || o.SelectedAt.Run != x.requester || o.Payload["request_id"].Text != x.request || o.Payload["service_operation"].Text != x.service || o.Payload["mutation"] != result.Payload["mutation"] || o.Payload["total"] != result.Payload["total"] || !o.Payload["accepted"].Flag {
					t.Fatal("exact result-bearing reply", o)
				}
				reply, ok := s.Operation(x.reply)
				if !ok || reply.Run != x.server || reply.Sequence <= result.SelectedAt.Sequence {
					t.Fatal("reply before custody")
				}
				reciprocalSelectedHistory(t, s, o)
				found = true
			}
		}
		reciprocalSelectedHistory(t, s, result)
		reciprocalSelectedHistory(t, s, selected)
		if !found {
			t.Fatal("original reply not selected")
		}
		ledger := endpoints[x.endpoint]
		if len(ledger.Applications) != 1 || ledger.Applications[0] != *e.ExternalReceipt || ledger.Total != 1 || ledger.Contacts < 1 {
			t.Fatal("endpoint exact result", ledger, e)
		}
		for _, req := range ledger.Requests {
			if req != e.ExternalAttempt.Request {
				t.Fatal("original wire changed", req, e.ExternalAttempt.Request)
			}
		}
	}
}

func TestPhase3B3ReciprocalOrdinary(t *testing.T) {
	for _, scenario := range []delegation.Scenario{delegation.S1Own, delegation.S2Handshake, delegation.S3Missing, delegation.S3Claim} {
		t.Run(string(scenario), func(t *testing.T) {
			dir := reciprocalDirectory(t, "ordinary-"+string(scenario))
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: scenario, TerminalHandshake: true})
			if err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			var events []boundary.Event
			ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
				reciprocalIndexMu.Lock()
				defer reciprocalIndexMu.Unlock()
				events = append(events, e)
				return nil
			})
			report, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
			baseline3b3Write(t, dir, "boundaries.json", events)
			if runErr != nil {
				t.Fatal(runErr)
			}
			endpoints := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
			phaseStopEndpoint(t, ca)
			phaseStopEndpoint(t, cb)
			baseline3b3Write(t, dir, "endpoints.json", endpoints)
			reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
			reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
			snapshot := reciprocalCapture(t, db, filepath.Join(dir, "final"))
			if scenario == delegation.S1Own && len(snapshot.LifecycleDecisions) != 0 {
				t.Fatal("own completion unnecessarily issued a grant")
			}
			classification := "safe-complete"
			if scenario == delegation.S1Own || scenario == delegation.S2Handshake {
				reciprocalComplete(t, snapshot, m, endpoints)
			} else {
				classification = "safe-refusal"
				e, ok := snapshot.Effect(m.BAService)
				if !ok || e.ExternalAttempt == nil || e.ExternalAttempt.Allowed || e.ExternalReceipt != nil || len(e.AuthorityAdmissions) != 0 || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal("ambient authority escaped", e)
				}
			}
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "ordinary-" + string(scenario), Schedule: "automatic-run", Directory: dir, Classification: classification, Manifest: m})
			baseline3b3Manifest(t, dir)
		})
	}
}

// TestPhase3B3Owner is a separate OS process; it uses only public Open and Run.
func TestPhase3B3Owner(t *testing.T) {
	if os.Getenv("ALAN_PHASE3B3_CHILD") != "1" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	path := os.Getenv("ALAN_PHASE2_DB")
	out := os.Getenv("ALAN_PHASE3B3_OUTPUT")
	hooks, err := boundary.FromEnvironment(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		baseline3b3Write(t, out, "open.json", map[string]string{"Error": err.Error()})
		t.Fatal(err)
	}
	defer s.Close()
	states := map[string]runtime.RunState{}
	for _, id := range []string{"a", "b"} {
		states[id], err = s.Snapshot(ctx, runtime.RunID{Subject: id, Generation: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	baseline3b3Write(t, out, "open.json", map[string]any{"PID": os.Getpid(), "Namespace": s.Namespace(), "Runs": states, "Error": "<nil>"})
	backup := filepath.Join(t.TempDir(), "post-open.sqlite")
	reciprocalBackup(t, path, backup)
	reciprocalCapture(t, backup, filepath.Join(out, "post-open"))
	f, err := os.OpenFile(filepath.Join(out, "boundaries.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var mu sync.Mutex
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		mu.Lock()
		err := json.NewEncoder(f).Encode(e)
		if err == nil {
			err = f.Sync()
		}
		mu.Unlock()
		if err != nil {
			return err
		}
		if os.Getenv("ALAN_BARRIER") == "" {
			return nil
		}
		return boundary.Reached(boundary.Inherit(ctx, hooks), e)
	})
	var config []reciprocalAdapter
	if raw := os.Getenv("ALAN_PHASE3B3_ADAPTERS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			t.Fatal(err)
		}
	}
	report, runErr := s.Run(ctx, runtime.RunOptions{Adapters: reciprocalAdapters(config), MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 75 * time.Second})
	baseline3b3Write(t, out, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func reciprocalOwner(t *testing.T, db, dir string, cut *barrier) *child {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return phaseTestProcess(t, db, "", "", cut, "TestPhase3B3Owner", func(cmd *exec.Cmd) { cmd.Env = append(cmd.Env, "ALAN_PHASE3B3_CHILD=1", "ALAN_PHASE3B3_OUTPUT="+dir) })
}

// This reader preserves the authority ordinal and avoids legacy fixture-specific
// assertions about terminal events, attempt presence and first activation epochs.
func reciprocalAwait(t *testing.T, c *child) boundary.Event {
	t.Helper()
	if err := c.ack.SetReadDeadline(time.Now().Add(40 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var e boundary.Event
	if err := json.NewDecoder(c.ack).Decode(&e); err != nil {
		raw, _ := os.ReadFile(c.stderr.Name())
		t.Fatal("ack", err, string(raw))
	}
	w := c.config
	if e.Scenario != w.Scenario || e.Token != w.Token || e.Store != w.Store || e.PID != c.cmd.Process.Pid || e.Boundary != w.Boundary || (w.Phase != "" && e.Phase != w.Phase) || (w.Subject != "" && e.Subject != w.Subject) || (w.Sequence != 0 && e.Sequence != w.Sequence) || (w.Intent != "" && e.Intent != w.Intent) || (w.Attempt != "" && e.Attempt != w.Attempt) || (w.Generation != 0 && e.Generation != w.Generation) || (w.Epoch != 0 && e.Epoch != w.Epoch) {
		t.Fatal("wrong process/boundary", e, w)
	}
	c.acknowledged = true
	return e
}

func reciprocalServiceSequence(t *testing.T, id string) int64 {
	t.Helper()
	var fields []json.RawMessage
	if err := json.Unmarshal([]byte(id), &fields); err != nil || len(fields) != 4 {
		t.Fatal("locator", id, err)
	}
	var n int64
	if err := json.Unmarshal(fields[2], &n); err != nil {
		t.Fatal(err)
	}
	return n
}
func reciprocalSelectedHistory(t *testing.T, s *runtime.ExecutionSnapshot, o *runtime.ObservationView) {
	t.Helper()
	if o.SelectedAt == nil || o.SelectedAt.Run != o.Run {
		t.Fatal("selection logical run mismatch", o)
	}
	cursor := 0
	found := 0
	for {
		page, err := s.History(cursor, 64)
		if err != nil {
			t.Fatal(err)
		}
		for _, step := range page.Steps {
			if step.Run == o.Run && step.Sequence == o.SelectedAt.Sequence {
				if step.Instruction != "receive" || step.SelectedObservationID != o.ID {
					t.Fatal("selection history mismatch", step, o)
				}
				found++
			}
		}
		if !page.More {
			break
		}
		cursor = page.Next
	}
	if found != 1 {
		t.Fatal("selection missing exact history", o)
	}
}

// SQLite's online backup captures committed pages coherently while the owner is
// alive. This is used before Run; plain live-file copying is never a snapshot.
func reciprocalBackup(t *testing.T, source, destination string) {
	t.Helper()
	src, err := sql.Open("sqlite3", (&url.URL{Scheme: "file", Path: source}).String()+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := sql.Open("sqlite3", destination)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	sc, err := src.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer sc.Close()
	dc, err := dst.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer dc.Close()
	err = sc.Raw(func(s any) error {
		return dc.Raw(func(d any) error {
			b, e := d.(*sqlite.SQLiteConn).Backup("main", s.(*sqlite.SQLiteConn), "main")
			if e != nil {
				return e
			}
			done, e := b.Step(-1)
			finish := b.Finish()
			if e != nil {
				return e
			}
			if !done {
				return fmt.Errorf("incomplete SQLite backup")
			}
			return finish
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func reciprocalSources(t *testing.T, dir string) {
	t.Helper()
	_, source, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("source identity unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	head, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, pattern := range []string{"examples/delegation/reciprocal*.go", "internal/durabletest/phase3b3_*test.go"} {
		paths, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			relative, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatal(err)
			}
			files[relative] = hex.EncodeToString(sum[:])
		}
	}
	baseline3b3Write(t, dir, "source.json", map[string]any{"HEAD": string(head), "FilesSHA256": files, "GoVersion": goruntime.Version(), "GOOS": goruntime.GOOS, "GOARCH": goruntime.GOARCH, "Command": os.Args, "Test": t.Name(), "RaceChildren": raceChildren, "Environment": map[string]string{"GOCACHE": os.Getenv("GOCACHE"), "GOTOOLCHAIN": os.Getenv("GOTOOLCHAIN"), "GOPROXY": os.Getenv("GOPROXY"), "GOSUMDB": os.Getenv("GOSUMDB")}})
	raw, err := os.ReadFile(filepath.Join(dir, "provision.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p runtime.Provision
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	programDir := filepath.Join(dir, "programs")
	if err = os.MkdirAll(programDir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, program := range p.Programs {
		if err = os.WriteFile(filepath.Join(programDir, program.ID+".json"), program.Bytes, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func reciprocalAccounting(t *testing.T, raw runtime.Inspection, s *runtime.ExecutionSnapshot) {
	t.Helper()
	if s.StorageVersion != 13 && s.StorageVersion != 14 {
		t.Fatal("request authority profile", s.StorageVersion)
	}
	for _, e := range s.Effects {
		allowed := false
		if e.ExternalAttempt != nil {
			allowed = e.ExternalAttempt.Allowed
		}
		if e.AdapterAttempt != nil {
			allowed = e.AdapterAttempt.Allowed
		}
		count := 0
		for _, row := range raw.Tables["composition_charges"] {
			if row["operation_id"] != e.OperationID {
				continue
			}
			var c struct {
				RootID, OperationID, Kind         string
				Effects, Invocations, CreatedRuns int64
			}
			if err := json.Unmarshal([]byte(row["descriptor"].(string)), &c); err != nil {
				t.Fatal(err)
			}
			run, _ := s.Run(e.Source)
			if c.RootID != run.RootID || row["root_id"] != run.RootID || c.OperationID != e.OperationID || c.Kind != "effect" || c.Effects != 1 || c.Invocations != 0 || c.CreatedRuns != 0 {
				t.Fatal("misattributed debit", c, e)
			}
			count++
		}
		want := 0
		if allowed {
			want = 1
		}
		if count != want {
			t.Fatal("original debit count", e.OperationID, count, want)
		}
	}
}
