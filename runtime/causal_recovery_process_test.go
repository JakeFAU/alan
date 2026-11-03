package runtime_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"syscall"
	"testing"
	"time"

	rt "github.com/JakeFAU/alan/runtime"
)

const (
	recoveryController = "retirement-controller"
	recoveryLife       = `["retirement-controller",1,2,0]`
	recoveryCall       = `["root",1,1,0]`
	recoveryWrite      = `["writer",1,1,0]`
	recoveryInput      = `["observation","input","control-target"]`
	recoveryNS         = "c35964b3158d492fa0d69dff69892f31b21cee6696720c639b0f2b2adfca3859"
	recoveryInvocation = "79a7f81b35486203d6086198432fc3072fda6e9f41782b769cd96500f2092d2e"
	recoveryEntry      = "137e1522572b02966c069ece4545dbf67fb289781c9be8c144b0015c749ddecc"
	recoveryHelper     = "3a5f694a5e35b05738e808ffe3fb0d7a34d73c2122159c575410ae116a946456"
	recoveryRequest    = "2d8761d3f1c96eb71b7754dd0152b8d687b19ec491791e4444c255a4ee31249c"
)

func recoveryRun(name string) rt.RunID { return rt.RunID{Subject: name, Generation: 1} }

func recoveryCopy(t testing.TB, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ledger.sqlite")
	b, err := os.ReadFile(filepath.Join("testdata", "lifecycle-recovery", name+".sqlite.gz"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// An attempted physical external exchange must fail the process before any network I/O.
type recoveryNoHTTP struct{}

func (recoveryNoHTTP) RoundTrip(*http.Request) (*http.Response, error) {
	panic("external contact during recovery")
}
func watchRecoveryHTTP(t *testing.T) {
	old := http.DefaultTransport
	http.DefaultTransport = recoveryNoHTTP{}
	t.Cleanup(func() { http.DefaultTransport = old })
}

// Include the main database and every byte of the WAL. SQLite's reconstructible
// SHM index and the runtime's OS lock are outside the committed application state.
func recoveryImage(t *testing.T, path string) [2][32]byte {
	t.Helper()
	var result [2][32]byte
	for i, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(path + suffix)
		if err != nil && !(i == 1 && errors.Is(err, os.ErrNotExist)) {
			t.Fatal(err)
		}
		result[i] = sha256.Sum256(b)
	}
	return result
}

func recoveryCut(t *testing.T, path string) *rt.ExecutionSnapshot {
	t.Helper()
	before := recoveryImage(t, path)
	cut, err := rt.InspectExecution(t.Context(), path, rt.DefaultLimits())
	if err != nil {
		t.Fatal("InspectExecution", err)
	}
	report, err := rt.Replay(t.Context(), path, rt.DefaultLimits())
	if err != nil {
		t.Fatal("Replay", err)
	}
	page, err := cut.History(0, rt.MaxHistoryPage)
	if err != nil || page.Total != report.HistoryCount || report.IntentCount != len(cut.Operations) || report.Namespace != cut.Namespace || cut.Namespace != recoveryNS || !reflect.DeepEqual(report.Roots, cut.Roots) || len(cut.Runs) != len(report.Runs) {
		t.Fatal("inspection and replay differ", report, err)
	}
	for _, state := range report.Runs {
		view, ok := cut.Run(state.ID)
		if !ok || !reflect.DeepEqual(view.RunState, state) {
			t.Fatal("replayed generation differs", state.ID)
		}
	}
	if recoveryImage(t, path) != before {
		t.Fatal("readers changed committed database or WAL")
	}
	return cut
}

// Compare the entire detached view including reductions. Only a successful
// activation may increase current epochs; historical witnesses never change.
func checkRecoveryRenewal(t *testing.T, before, after *rt.ExecutionSnapshot) {
	t.Helper()
	for _, run := range before.Runs {
		if run.Projection.Status == "open" {
			run.Epoch++
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("owner activation changed facts beyond the single epoch increment of each open generation")
	}
}

func checkRecoveryEvidence(t *testing.T, cut *rt.ExecutionSnapshot, reason string, created, selected bool) {
	t.Helper()
	effect, ok := cut.Effect(recoveryWrite)
	if !ok || effect.State != rt.EffectReservedUnknown || effect.Consequence != rt.ConsequenceUnresolved || effect.Settlement != nil || effect.ExternalReceipt != nil || effect.ExternalAttempt == nil || effect.ExternalAttempt.Request.Key != `[2,"`+recoveryNS+`","[\"writer\",1,1,0]"]` {
		t.Fatal("lost original unknown reservation", effect)
	}
	if len(cut.Roots) != 1 {
		t.Fatal(cut.Roots)
	}
	root := cut.Roots[0]
	if root.Charges.Effects != 1 || !reflect.DeepEqual(root.UnknownExternal, []string{recoveryWrite}) {
		t.Fatal("lost resource or consequence", root)
	}
	if root.LocalStatus != "open" || root.Outcome != nil {
		t.Fatal("spurious root completion", root)
	}
	ctl, ok := cut.Operation(recoveryLife)
	if !ok || ctl.Target != recoveryEntry || ctl.Run != recoveryRun(recoveryController) {
		t.Fatal("lost controller intent", ctl)
	}
	o, ok := cut.Observation(recoveryInput)
	if !ok || o.SelectedAt == nil || *o.SelectedAt != (rt.ReductionRef{Run: recoveryRun(recoveryController), Sequence: 1}) {
		t.Fatal("controller input selection", o)
	}
	if !slices.ContainsFunc(cut.Pending, func(p rt.PendingObligation) bool {
		return p.Kind == "lifecycle_awaiting_settlement" && p.OperationID == recoveryLife && p.Run == ctl.Run
	}) {
		t.Fatal("lost original unsettled request", cut.Pending)
	}
	result, found := cut.LifecycleDecision(recoveryLife)
	if found != (reason != "") || found && (result.Reason != reason || result.Accepted || result.Request.Intent.ID != recoveryLife || result.DecisionSequence != 1) {
		t.Fatal("wrong target decision", result)
	}
	if reason == "target_missing" && (result.Before != nil || result.After != nil) {
		t.Fatal("supplied a witness for absence", result)
	}
	if reason == "grant_denied" && (result.Before == nil || result.Before.Run != recoveryRun(recoveryEntry) || result.Before.Projection.Status != "open" || !reflect.DeepEqual(result.Before, result.After)) {
		t.Fatal("lost witnessed open generation", result)
	}
	call, ok := cut.Invocation(recoveryCall)
	if ok != created {
		t.Fatal("unexpected invocation", call)
	}
	child, ok := cut.Run(recoveryRun(recoveryEntry))
	if ok != created {
		t.Fatal("unexpected worker", child)
	}
	if !created {
		if root.Charges.Invocations != 0 || root.Charges.CreatedRuns != 0 {
			t.Fatal(root.Charges)
		}
		return
	}
	if root.Charges.Invocations != 1 || root.Charges.CreatedRuns != 2 || !reflect.DeepEqual(root.PendingInvocations, []string{recoveryCall}) || call.State != rt.InvocationAdmitted || call.Admission == nil || call.Admission.Invocation.ID != recoveryInvocation || child.LocalState != rt.RunOpen || child.OwningInvocationID != recoveryInvocation || child.OriginOperationID != recoveryCall {
		t.Fatal("lost admission, resources or worker", root, call, child)
	}
	obs, ok := cut.Observation(recoveryRequest)
	if !ok || call.RequestObservationID != recoveryRequest || (obs.SelectedAt != nil) != selected || selected && *obs.SelectedAt != (rt.ReductionRef{Run: child.ID, Sequence: 1}) {
		t.Fatal("wrong invocation selection", obs)
	}
}

func TestCausalRecoveryRetainedPair(t *testing.T) {
	watchRecoveryHTTP(t)
	for _, tc := range []struct{ name, reason string }{{"missing", "target_missing"}, {"present", "grant_denied"}} {
		t.Run(tc.name, func(t *testing.T) {
			path := recoveryCopy(t, tc.name)
			cut, err := rt.InspectExecution(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal("InspectExecution", err)
			}
			t.Run("readers", func(t *testing.T) { recoveryCut(t, path) })
			checkRecoveryEvidence(t, cut, tc.reason, true, true)
			for name, want := range map[string]int64{recoveryEntry: 1, recoveryHelper: 1, "finished": 2, recoveryController: 2, "root": 2, "writer": 2} {
				r, ok := cut.Run(recoveryRun(name))
				if !ok || r.Epoch != want {
					t.Fatal(name, r)
				}
			}
			before := recoveryImage(t, path)
			s, err := rt.Open(t.Context(), path, rt.DefaultLimits())
			if err != nil {
				t.Fatal("Open", err)
			}
			defer s.Close()
			next := recoveryCut(t, path)
			for _, run := range next.Runs {
				owned, err := s.Snapshot(t.Context(), run.ID)
				if err != nil || !reflect.DeepEqual(owned, run.RunState) {
					t.Fatal("owned generation differs from both readers", run.ID, err)
				}
			}
			checkRecoveryRenewal(t, cut, next)
			if recoveryImage(t, path) == before {
				t.Fatal("accepted owner failed to persist renewal")
			}
		})
	}
}

// Keep a connection alive so last-close checkpointing cannot mask an appended
// frame. The pragma writes a genuine committed frame without changing ALAN rows.
func recoveryWAL(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(fmt.Sprintf("PRAGMA user_version=%d", version)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path + "-wal")
	if err != nil || info.Size() <= 32 {
		t.Fatal("missing durable WAL frames", err)
	}
	return db
}

func TestCausalRecoveryRejectionAtomic(t *testing.T) {
	if path := os.Getenv("ALAN_RECOVERY_WAL_PATH"); path != "" {
		// The subprocess owns the only SQLite connection. Kill it after commit
		// so last-close checkpointing cannot empty the WAL before the test.
		db := recoveryWAL(t, path)
		if os.Getenv("ALAN_RECOVERY_WAL_CASE") == "missing" {
			if _, err := db.Exec("UPDATE runs SET epoch=epoch+? WHERE json_extract(projection,'$.Status')='open'", int64(math.MaxInt64-2)); err != nil {
				t.Fatal(err)
			}
		}
		t.Fatal(syscall.Kill(os.Getpid(), syscall.SIGKILL))
	}
	watchRecoveryHTTP(t)
	for _, tc := range []struct {
		name     string
		overflow bool
	}{{"forged", false}, {"missing", true}} {
		t.Run(tc.name, func(t *testing.T) {
			path := recoveryCopy(t, tc.name)
			child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCausalRecoveryRejectionAtomic$")
			child.Env = append(os.Environ(), "ALAN_RECOVERY_WAL_PATH="+path, "ALAN_RECOVERY_WAL_CASE="+tc.name)
			out, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("WAL writer did not die after commit: %v %s", err, out)
			}
			if info, err := os.Stat(path + "-wal"); err != nil || info.Size() <= 32 {
				t.Fatal("killed writer left no durable WAL frames", err)
			}
			want := rt.ErrCorrupt
			if tc.overflow {
				// The two members sort first. They are one activation younger than
				// the static open generations; both can renew before the first overflow.
				cut := recoveryCut(t, path)
				for _, name := range []string{recoveryEntry, recoveryHelper, recoveryController} {
					r, ok := cut.Run(recoveryRun(name))
					v := int64(math.MaxInt64 - 1)
					if name == recoveryController {
						v++
					}
					if !ok || r.Epoch != v {
						t.Fatal("partial renewal precondition", name, r)
					}
				}
				want = rt.ErrLimit
			}
			before := recoveryImage(t, path)
			if !tc.overflow {
				if v, err := rt.InspectExecution(t.Context(), path, rt.DefaultLimits()); v != nil || !errors.Is(err, rt.ErrCorrupt) {
					t.Errorf("forgery inspection: %v", err)
				}
				if _, err := rt.Replay(t.Context(), path, rt.DefaultLimits()); !errors.Is(err, rt.ErrCorrupt) {
					t.Errorf("forgery replay: %v", err)
				}
			}
			for range 2 {
				s, err := rt.Open(t.Context(), path, rt.DefaultLimits())
				if s != nil {
					s.Close()
				}
				if s != nil || !errors.Is(err, want) {
					t.Fatalf("owner accepted or misclassified history: %v, want %v", err, want)
				}
				if recoveryImage(t, path) != before {
					t.Fatal("failed owner changed database or durable WAL frames")
				}
			}
		})
	}
}

// Acknowledgment is written only after the named production API returns with
// committed state. The child then blocks on a pipe until the parent kills it.
type recoveryAck struct {
	Stage, Path string
	PID         int
}

func TestCausalRecoveryOwnerChild(t *testing.T) {
	stage := os.Getenv("ALAN_RECOVERY_TEST_STAGE")
	if stage == "" {
		return
	}
	watchRecoveryHTTP(t)
	path := os.Getenv("ALAN_RECOVERY_TEST_STORE")
	ctx := t.Context()
	s, err := rt.Open(ctx, path, rt.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	commit := func(name, observation string) rt.RunState {
		state, e := s.Snapshot(ctx, recoveryRun(name))
		if e != nil {
			t.Fatal(e)
		}
		r, e := s.Commit(ctx, state.Authority(), observation)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	switch stage {
	case "authorized":
		input, err := s.AdmitInput(ctx, rt.Input{TransportID: "control-target", LogicalID: "control-target", Target: recoveryController, Generation: 1, Obligation: "control", Payload: map[string]rt.Value{"target": {Kind: "string", Text: recoveryEntry}}})
		if err != nil || !input.Accepted || input.ObservationID != recoveryInput {
			t.Fatal(input, err)
		}
		commit(recoveryController, input.ObservationID)
		r := commit(recoveryController, "")
		a, err := s.AuthorizeIntent(ctx, r.Projection.Data["pending"].Text)
		if err != nil || !a.Allowed || a.IntentID != recoveryLife {
			t.Fatal(a, err)
		}
	case "missing":
		req, err := s.LoadLifecycleRequest(ctx, recoveryLife)
		if err != nil {
			t.Fatal(err)
		}
		d, err := s.DecideLifecycle(ctx, req)
		if err != nil || d.Reason != "target_missing" || d.Before != nil || d.After != nil {
			t.Fatal(d, err)
		}
	case "created":
		r := commit("root", "")
		a, err := s.AdmitInvocation(ctx, r.Projection.Data["pending"].Text)
		if err != nil || !a.Accepted || a.Invocation.ID != recoveryInvocation {
			t.Fatal(a, err)
		}
	case "worker":
		commit(recoveryEntry, recoveryRequest)
	case "activated": // The Open above was the only application operation.
	default:
		t.Fatal("unknown stage", stage)
	}
	ack := os.NewFile(3, "ready")
	defer ack.Close()
	if err = json.NewEncoder(ack).Encode(recoveryAck{stage, path, os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	var release [1]byte
	_, err = io.ReadFull(os.Stdin, release[:])
	t.Fatalf("unexpected child release: %v", err)
}

func killRecoveryOwner(t *testing.T, path, stage string, live func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCausalRecoveryOwnerChild$")
	cmd.Env = append(os.Environ(), "ALAN_RECOVERY_TEST_STAGE="+stage, "ALAN_RECOVERY_TEST_STORE="+path)
	ack, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer ack.Close()
	cmd.ExtraFiles = []*os.File{w}
	hold, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	var log bytes.Buffer
	cmd.Stdout, cmd.Stderr = &log, &log
	err = cmd.Start()
	w.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	var got recoveryAck
	if err = json.NewDecoder(ack).Decode(&got); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("child acknowledgment: %v: %s", err, log.String())
	}
	if got != (recoveryAck{stage, path, cmd.Process.Pid}) {
		t.Fatal("incorrect owner acknowledgment", got)
	}
	live()
	if err = cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if err == nil || !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("missing SIGKILL: %v %v %s", err, status, log.String())
	}
	t.Logf("SIGKILL pid=%d after durable %s", got.PID, stage)
}

func TestCausalRecoveryProcessDeaths(t *testing.T) {
	watchRecoveryHTTP(t)
	path := recoveryCopy(t, "ready")
	// The anchor also prevents last-close WAL checkpointing after each killed owner.
	db := recoveryWAL(t, path)
	var prior *rt.ExecutionSnapshot
	var originalAuth []byte
	stages := []string{"authorized", "missing", "created", "worker", "activated", "activated"}
	for i, stage := range stages {
		var live *rt.ExecutionSnapshot
		killRecoveryOwner(t, path, stage, func() {
			before := recoveryImage(t, path)
			s, err := rt.Open(t.Context(), path, rt.DefaultLimits())
			if s != nil {
				s.Close()
			}
			if s != nil || !errors.Is(err, rt.ErrLocked) {
				t.Fatal("live owner lock", err)
			}
			live = recoveryCut(t, path)
			if recoveryImage(t, path) != before {
				t.Fatal("competing readers or owner changed committed frames")
			}
		})
		cut := recoveryCut(t, path)
		if !reflect.DeepEqual(cut, live) {
			t.Fatal("owner death changed the durable cut", stage)
		}
		reason := "target_missing"
		if i == 0 {
			reason = ""
		}
		checkRecoveryEvidence(t, cut, reason, i >= 2, i >= 3)
		var saved []byte
		if err := db.QueryRow("SELECT result FROM attempts WHERE intent_id=?", recoveryLife).Scan(&saved); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			var a rt.Authorization
			if err := json.Unmarshal(saved, &a); err != nil || !a.Allowed || a.IntentID != recoveryLife || a.Source != (rt.Authority{Run: recoveryRun(recoveryController), Epoch: 3, Sequence: 2, Revision: 1}) {
				t.Fatal("original pending source authorization", a, err)
			}
			originalAuth = saved
		} else if !bytes.Equal(saved, originalAuth) {
			t.Fatal("owner replaced original source authorization")
		}
		if i >= 4 {
			checkRecoveryRenewal(t, prior, cut)
		}
		prior = cut
	}
}
