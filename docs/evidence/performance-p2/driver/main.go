// performance-p2 measures one fresh bounded-series execution per process.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"runtime/pprof"
	"runtime/trace"
	"syscall"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Assigned only by an explicitly generated instrumentation hook in a scratch
// copy of this driver. Normal public-package builds have no counter hook.
var counterSnapshot func() map[string]uint64

type stage struct {
	GCCount        uint64             `json:"gc_count"`
	ForcedGCCount  uint64             `json:"forced_gc_count"`
	GCPauseNS      uint64             `json:"gc_pause_ns"`
	RuntimeMetrics map[string]float64 `json:"runtime_metric_deltas"`
	Name           string             `json:"name"`
	Position       int                `json:"position,omitempty"`
	WallNS         int64              `json:"wall_ns"`
	UserCPUNS      int64              `json:"user_cpu_ns"`
	SystemCPUNS    int64              `json:"system_cpu_ns"`
	TotalAlloc     uint64             `json:"total_alloc_bytes"`
	Mallocs        uint64             `json:"mallocs"`
	ProcessMaxRSS  int64              `json:"process_maxrss_cumulative"`
	Counters       map[string]uint64  `json:"counter_deltas,omitempty"`
	Error          string             `json:"error,omitempty"`
}
type contact struct {
	Prepared         effects.Prepared `json:"prepared"`
	RequestWire      string           `json:"request_wire"`
	RequestSHA256    string           `json:"request_sha256"`
	ResponseWire     string           `json:"response_wire"`
	ResponseSHA256   string           `json:"response_sha256"`
	ResponseComplete bool             `json:"response_complete"`
	Ordinal          int              `json:"ordinal"`
	OperationID      string           `json:"operation_id"`
	Key              string           `json:"key"`
	PreparedDigest   string           `json:"prepared_digest"`
	Capability       string           `json:"capability"`
	Effect           string           `json:"effect"`
	Protocol         string           `json:"protocol"`
	Endpoint         string           `json:"endpoint"`
	ResponseID       string           `json:"response_id"`
}
type evidence struct {
	RootOutcome    *aruntime.InvocationOutcome `json:"root_outcome"`
	Prepared       effects.Prepared            `json:"prepared"`
	Candidate      effects.Candidate           `json:"candidate"`
	Index          int                         `json:"index"`
	Run            aruntime.RunID              `json:"run"`
	RootID         string                      `json:"root_id"`
	OperationID    string                      `json:"operation_id"`
	PreparedDigest string                      `json:"prepared_digest"`
	ResultDigest   string                      `json:"result_digest"`
	ResponseID     string                      `json:"response_id"`
	ObservationID  string                      `json:"observation_id"`
	SelectedAt     *aruntime.ReductionRef      `json:"selected_at"`
	Approval       *aruntime.ContinuationView  `json:"approval,omitempty"`
}
type report struct {
	RootMembership      []rootIdentity      `json:"root_membership"`
	PrefixConfiguration prefixConfiguration `json:"prefix_configuration"`
	Transport           transportConfig     `json:"transport"`
	PrefixEnabled       bool                `json:"prefix_enabled"`
	Prefixes            []prefixRecord      `json:"prefixes"`
	PrefixResidual      stage               `json:"prefix_residual"`
	ServerContacts      []contact           `json:"server_contacts"`
	MetricNames         []string            `json:"runtime_metric_names"`
	ResourceScope       string              `json:"resource_scope"`
	Reductions          int                 `json:"reductions"`
	ProcessWallNS       int64               `json:"process_wall_ns"`
	Version             string              `json:"version"`
	Bound               int                 `json:"bound"`
	Sample              string              `json:"sample"`
	SourceLabel         string              `json:"source_label"`
	Store               string              `json:"store"`
	Schedule            string              `json:"schedule"`
	ReopenEvery         int                 `json:"reopen_every"`
	GoVersion           string              `json:"go_version"`
	GOOS                string              `json:"goos"`
	GOARCH              string              `json:"goarch"`
	GOMAXPROCS          int                 `json:"gomaxprocs"`
	MaxRSSUnit          string              `json:"maxrss_unit"`
	MemProfileRate      int                 `json:"mem_profile_rate"`
	Profiled            bool                `json:"profiled"`
	Instrumented        bool                `json:"instrumented"`
	StartedUTC          string              `json:"started_utc"`
	Stages              []stage             `json:"stages"`
	ExecuteTotals       stage               `json:"execute_totals"`
	Contacts            []contact           `json:"contacts"`
	Evidence            []evidence          `json:"evidence"`
	Namespace           string              `json:"namespace"`
	StorageVersion      int                 `json:"storage_version"`
	Selected            int                 `json:"selected"`
	Completed           int                 `json:"completed"`
	Spent               int                 `json:"spent"`
	ProtectedApprovals  int                 `json:"protected_approvals"`
	ReplayHistory       int                 `json:"replay_history"`
	ReplayIntents       int                 `json:"replay_intents"`
	InvariantsOK        bool                `json:"invariants_ok"`
	Error               string              `json:"error,omitempty"`
}

func definition() author.Definition {
	return author.Definition{Name: "measurement", Inputs: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 64}}}}, Steps: []author.Step{
		author.Request("operation", "read", map[string]author.Expr{"station": author.Var("station")}, nil), author.Await("read", "operation", map[string]string{"sample": "sample"}), author.Success()}}
}
func usage() (syscall.Rusage, error) {
	var r syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &r)
	return r, err
}
func cpu(t syscall.Timeval) int64 { return t.Sec*1e9 + int64(t.Usec)*1000 }
func (r *report) measure(name string, position int, f func() error) error {
	fmt.Fprintf(os.Stderr, "sample=%s bound=%d stage=%s position=%d start\n", r.Sample, r.Bound, name, position)
	before, err := takeSnapshot()
	if err != nil {
		return err
	}
	start := time.Now()
	var callErr error
	if r.Profiled {
		pprof.Do(context.Background(), pprof.Labels("stage", name), func(context.Context) { callErr = f() })
	} else {
		callErr = f()
	}
	elapsed := time.Since(start)
	after, err := takeSnapshot()
	if err != nil {
		return err
	}
	s, err := snapshotDelta(before, after)
	if err != nil {
		return err
	}
	s.Name, s.Position, s.WallNS = name, position, elapsed.Nanoseconds()
	if callErr != nil {
		s.Error = callErr.Error()
	}
	r.Stages = append(r.Stages, s)
	if name == "run" || name == "approval_snapshot_basis" || name == "approval_establishment" {
		t := &r.ExecuteTotals
		t.Name = "execute_sum"
		t.WallNS += s.WallNS
		t.UserCPUNS += s.UserCPUNS
		t.SystemCPUNS += s.SystemCPUNS
		t.TotalAlloc += s.TotalAlloc
		t.Mallocs += s.Mallocs
		t.GCCount += s.GCCount
		t.ForcedGCCount += s.ForcedGCCount
		t.GCPauseNS += s.GCPauseNS
		if t.RuntimeMetrics == nil {
			t.RuntimeMetrics = map[string]float64{}
		}
		for k, v := range s.RuntimeMetrics {
			t.RuntimeMetrics[k] += v
		}
		t.ProcessMaxRSS = s.ProcessMaxRSS
		if s.Counters != nil {
			if t.Counters == nil {
				t.Counters = map[string]uint64{}
			}
			for k, v := range s.Counters {
				t.Counters[k] += v
			}
		}
	}
	fmt.Fprintf(os.Stderr, "sample=%s bound=%d stage=%s position=%d done wall_ns=%d error=%v\n", r.Sample, r.Bound, name, position, s.WallNS, callErr)
	return callErr
}

func verify(cut *aruntime.ExecutionSnapshot, v author.SeriesView, m *meter, n int) ([]evidence, error) {
	fail := func(f string, a ...any) ([]evidence, error) { return nil, fmt.Errorf(f, a...) }
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(cut.Runs) != n || len(cut.Roots) != n || len(cut.Effects) != n || len(cut.Continuations) != n-1 || len(m.contacts) != n || (m.mode == "http" && m.serverCount() != n) {
		return fail("counts runs=%d roots=%d effects=%d approvals=%d contacts=%d want=%d", len(cut.Runs), len(cut.Roots), len(cut.Effects), len(cut.Continuations), len(m.contacts), n)
	}
	if v.Bound != n || len(v.Positions) != n || v.Completed != n || v.Selected != n || v.Spent != n || v.Failed != 0 || v.Pending != 0 || v.Unresolved != 0 || !v.BoundExhausted {
		return fail("series aggregate mismatch: %+v", v)
	}
	ids, keys, observations, results, roots := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	out := make([]evidence, 0, n)
	for j, p := range v.Positions {
		want := aruntime.RunID{Subject: fmt.Sprintf("series-%d", j+1), Generation: 1}
		if p.Index != j+1 || p.Run == nil || p.Run.ID != want || p.Run.Projection.Status != "succeeded" || p.State != author.SeriesCompleted || p.Root == nil || p.Root.Run != want || p.Root.Charges.Effects != 1 || p.Root.Charges.Invocations != 0 || p.Root.Charges.CreatedRuns != 0 || p.Root.Outcome == nil {
			return fail("position %d run/root/charge/state mismatch", j+1)
		}
		e, o := p.Effect, p.Observation
		if e == nil || e.Source != want || e.OperationID == "" || ids[e.OperationID] || len(p.Run.EffectIDs) != 1 || p.Run.EffectIDs[0] != e.OperationID || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.AdapterResult == nil || e.ResultDisposition != "validated" || e.State != aruntime.EffectSettled || e.Settlement == nil || e.DispatchAdmission == nil || e.Dispatch.Minimum != 1 || e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 1 {
			return fail("position %d effect identity/reservation/result mismatch", j+1)
		}
		a, res, c := e.AdapterAttempt, e.AdapterResult, m.contacts[j]
		if a.Prepared.Request.ID != e.OperationID || a.Prepared.Request.Key == "" || keys[a.Prepared.Request.Key] || res.Candidate.ID != e.OperationID || res.Candidate.Key != a.Prepared.Request.Key || res.Digest == "" || results[res.Digest] || c.Ordinal != j+1 || c.OperationID != e.OperationID || c.Key != a.Prepared.Request.Key || c.PreparedDigest != a.Prepared.Digest || c.ResponseID != res.Candidate.ResponseID {
			return fail("position %d adapter contact provenance mismatch", j+1)
		}
		if c.Capability != a.Prepared.Request.Capability || c.Effect != a.Prepared.Request.Effect || c.Protocol != a.Prepared.Protocol || c.Endpoint != a.Prepared.Endpoint {
			return fail("position %d adapter contact request mismatch", j+1)
		}
		if err := verifyTransport(a.Prepared, res.Candidate, c, m, j); err != nil {
			return fail("position %d transport proof: %v", j+1, err)
		}
		if o == nil || o.ID == "" || observations[o.ID] || o.SourceOperationID != e.OperationID || o.SelectedAt == nil || o.SelectedAt.Run != want || e.Settlement.ObservationID != o.ID {
			return fail("position %d selection provenance mismatch", j+1)
		}
		reduction, ok := cut.Reduction(*o.SelectedAt)
		if !ok || reduction.SelectedObservationID != o.ID {
			return fail("position %d missing selected reduction", j+1)
		}
		if j == 0 {
			if p.Approval != nil {
				return fail("first position unexpectedly has protected approval")
			}
		} else {
			prev := out[j-1]
			if p.Approval == nil || p.Approval.Approval.Target != want || p.Approval.Approval.PredecessorID != prev.OperationID || p.Approval.Approval.PreparedDigest != prev.PreparedDigest || p.Approval.Approval.ResultDigest != prev.ResultDigest || !p.Approval.Approval.Admitted || p.Approval.Approval.Reason != "operator" || p.Approval.Approval.ScopeDigest == "" || p.Approval.SelectedAt == nil || p.Approval.SelectedAt.Run != want || p.Approval.SuccessorOperationID != e.OperationID || p.Approval.SelectedAt.Sequence >= e.Sequence {
				return fail("position %d protected predecessor/selection causality mismatch", j+1)
			}
			approvalObs, ok := cut.Observation(p.Approval.ObservationID)
			if !ok || approvalObs.SelectedAt == nil || *approvalObs.SelectedAt != *p.Approval.SelectedAt {
				return fail("position %d approval selection mismatch", j+1)
			}
		}
		if roots[p.Root.RootID] {
			return fail("duplicate root")
		}
		ids[e.OperationID], keys[a.Prepared.Request.Key], observations[o.ID], results[res.Digest], roots[p.Root.RootID] = true, true, true, true, true
		out = append(out, evidence{RootOutcome: p.Root.Outcome, Prepared: a.Prepared, Candidate: res.Candidate, Index: j + 1, Run: want, RootID: p.Root.RootID, OperationID: e.OperationID, PreparedDigest: a.Prepared.Digest, ResultDigest: res.Digest, ResponseID: res.Candidate.ResponseID, ObservationID: o.ID, SelectedAt: o.SelectedAt, Approval: p.Approval})
	}
	return out, nil
}

func run(ctx context.Context, r *report, dir string, reopenEvery int) (err error) {
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "store.sqlite")
	r.Store = path
	if _, e := os.Stat(path); !errors.Is(e, os.ErrNotExist) {
		return fmt.Errorf("fresh store required: %s", path)
	}
	var decl author.Series
	var provision aruntime.Provision
	if err = r.measure("construct_provision", 0, func() error {
		var e error
		decl, e = author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: r.Bound, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
		if e != nil {
			return e
		}
		provision, e = decl.Provision()
		return e
	}); err != nil {
		return err
	}
	var store *aruntime.Store
	defer func() {
		if store != nil {
			err = errors.Join(err, store.Close())
		}
	}()
	if err = r.measure("create", 0, func() error {
		var e error
		store, e = aruntime.Create(ctx, path, provision, aruntime.DefaultLimits())
		return e
	}); err != nil {
		return err
	}
	var cuts *cutVerifier
	if err = r.measure("original_state_verification", 0, func() error {
		initial, e := aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		if len(initial.Runs) != r.Bound || len(initial.Roots) != r.Bound || len(initial.Effects) != 0 || len(initial.Continuations) != 0 {
			return errors.New("initial provisioned state mismatch")
		}
		for i := 1; i <= r.Bound; i++ {
			run, ok := initial.Run(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
			if !ok || len(run.EffectIDs) != 0 {
				return errors.New("initial original identity mismatch")
			}
		}
		cuts, e = newCutVerifier(initial.Roots, r.Bound)
		if e != nil {
			return e
		}
		r.RootMembership = append([]rootIdentity{}, cuts.membership...)
		return nil
	}); err != nil {
		return err
	}
	if r.Transport.Mode == "" {
		r.Transport = frozenTransport("direct", "")
	}
	m, err := newMeter(r.Transport)
	if err != nil {
		return err
	}
	r.Transport.Endpoint = m.endpoint
	defer func() { err = errors.Join(err, m.finishReport(r)) }()
	opts := aruntime.RunOptions{MaxSteps: 16*r.Bound + 32, MaxDispatch: r.Bound + 8, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"meter": m}}
	windowName := "execute_window"
	if reopenEvery > 0 {
		windowName = "execution_window_including_reopen"
	}
	if err = r.measure(windowName, 0, func() error {
		var base, previous resourceSnapshot
		if r.PrefixEnabled {
			base, err = takeSnapshot()
			if err != nil {
				return err
			}
			previous = base
		}
		for i := 1; i <= r.Bound; i++ {
			if i > 1 {
				var basis aruntime.ContinuationBasis
				if err = r.measure("approval_snapshot_basis", i, func() error {
					cut, e := aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())
					if e != nil {
						return e
					}
					basis, e = cut.ContinuationBasis(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
					return e
				}); err != nil {
					return err
				}
				if err = r.measure("approval_establishment", i, func() error { _, e := store.ApproveContinuation(ctx, basis, "operator"); return e }); err != nil {
					return err
				}
			}
			var rr aruntime.RunReport
			if err = r.measure("run", i, func() error { var e error; rr, e = store.Run(ctx, opts); return e }); err != nil {
				return err
			}
			m.mu.Lock()
			calls := len(m.contacts)
			m.mu.Unlock()
			if calls != i {
				return fmt.Errorf("position %d adapter calls=%d", i, calls)
			}
			cut, e := cuts.verify(rr, i, true)
			if e != nil {
				return e
			}
			if reopenEvery > 0 && i < r.Bound && i%reopenEvery == 0 {
				if err = r.measure("interval_close_reopen_resume", i, func() error {
					if e := store.Close(); e != nil {
						return e
					}
					store = nil
					var e error
					store, e = aruntime.Open(ctx, path, aruntime.DefaultLimits())
					if e != nil {
						return e
					}
					resumed, e := store.Run(ctx, opts)
					if e != nil {
						return e
					}
					_, e = cuts.verify(resumed, i, false)
					return e
				}); err != nil {
					return err
				}
				m.mu.Lock()
				calls = len(m.contacts)
				m.mu.Unlock()
				if calls != i {
					return fmt.Errorf("reopen duplicated contact at %d", i)
				}
			}
			serverCalls := m.serverCount()
			if m.mode == "http" && serverCalls != calls {
				return fmt.Errorf("position %d server contacts=%d adapter contacts=%d", i, serverCalls, calls)
			}
			if r.PrefixEnabled {
				current, e := takeSnapshot()
				if e != nil {
					return e
				}
				direct, e := snapshotDelta(previous, current)
				if e != nil {
					return e
				}
				cumulative, e := snapshotDelta(base, current)
				if e != nil {
					return e
				}
				r.Prefixes = append(r.Prefixes, prefixRecord{Position: i, DeclarationBound: r.Bound, RetainedPositions: cut.Completed, Cut: cut, Contacts: calls, ServerContacts: serverCalls, Direct: direct, Cumulative: cumulative})
				previous = current
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if r.PrefixEnabled {
		if err := r.verifyPrefixes(); err != nil {
			return err
		}
	}
	var cut *aruntime.ExecutionSnapshot
	var view author.SeriesView
	if err = r.measure("final_inspection", 0, func() error {
		var e error
		cut, e = aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())
		return e
	}); err != nil {
		return err
	}
	if err = r.measure("final_series_view", 0, func() error { var e error; view, e = decl.Inspect(cut); return e }); err != nil {
		return err
	}
	if err = r.measure("final_series_proof", 0, func() error { var e error; r.Evidence, e = verify(cut, view, m, r.Bound); return e }); err != nil {
		return err
	}
	for i, prefix := range r.Prefixes {
		if prefix.Cut.NewRoot.RootID != r.Evidence[i].RootID || !reflect.DeepEqual(prefix.Cut.NewRoot.Outcome, r.Evidence[i].RootOutcome) {
			return errors.New("prefix original root outcome changed by final cut")
		}
	}
	history, historyErr := cut.History(0, 1)
	if historyErr != nil {
		return historyErr
	}
	r.Reductions = history.Total
	if r.Reductions != 4*r.Bound-1 {
		return fmt.Errorf("reductions=%d want=%d", r.Reductions, 4*r.Bound-1)
	}
	r.Namespace, r.StorageVersion, r.Selected, r.Completed, r.Spent, r.ProtectedApprovals = cut.Namespace, cut.StorageVersion, view.Selected, view.Completed, view.Spent, len(cut.Continuations)
	before := r.Evidence
	if err = r.measure("close_reopen_resume", 0, func() error {
		if e := store.Close(); e != nil {
			return e
		}
		store = nil
		var e error
		store, e = aruntime.Open(ctx, path, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		_, e = store.Run(ctx, opts)
		return e
	}); err != nil {
		return err
	}
	if err = r.measure("resume_inspection_view", 0, func() error {
		var e error
		cut, e = aruntime.InspectExecution(ctx, path, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		view, e = decl.Inspect(cut)
		return e
	}); err != nil {
		return err
	}
	after, e := verify(cut, view, m, r.Bound)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("completed resume changed original evidence")
	}
	if err = r.measure("close", 0, func() error { e := store.Close(); store = nil; return e }); err != nil {
		return err
	}
	if err = r.measure("replay", 0, func() error {
		replayed, e := aruntime.Replay(ctx, path, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		if replayed.Namespace != r.Namespace || len(replayed.Runs) != r.Bound || len(replayed.Roots) != r.Bound {
			return fmt.Errorf("replay identity/count mismatch")
		}
		for _, run := range replayed.Runs {
			if run.Projection.Status != "succeeded" {
				return fmt.Errorf("replay non-success: %v", run.ID)
			}
		}
		r.ReplayHistory, r.ReplayIntents = replayed.HistoryCount, replayed.IntentCount
		return nil
	}); err != nil {
		return err
	}
	r.InvariantsOK = true
	return nil
}

func main() {
	processStart := time.Now()
	n := flag.Int("bound", 1, "finite series bound (1,8,16,32,64,128,256)")
	dir := flag.String("dir", "", "new sample artifact directory (required)")
	sample := flag.String("sample", "1", "sample identifier")
	label := flag.String("source-label", "unspecified", "source/build identifier recorded verbatim")
	reopen := flag.Int("reopen-every", 0, "separate recovery schedule; reopen after every k selected positions (0 disables)")
	cp := flag.String("cpu-profile", "", "optional CPU profile path; profiling campaign only")
	ap := flag.String("alloc-profile", "", "optional allocation profile path; profiling campaign only")
	mr := flag.Int("mem-profile-rate", goruntime.MemProfileRate, "allocation profile sampling rate in bytes; 1 samples all allocations (diagnostic only)")
	tr := flag.String("trace", "", "optional execution trace path; profiling campaign only")
	mode := flag.String("mode", "http", "http (headline) or direct (historical bridge only)")
	endpoint := flag.String("endpoint", "127.0.0.1:0", "loopback listener address; exact host:port for same-seed comparisons")
	prefix := flag.Bool("prefix", true, "collect adjacent prefix snapshots; false is calibration/bridge only")
	timeout := flag.Duration("timeout", 30*time.Minute, "single sample deadline; no automatic retries")
	flag.Parse()
	allowed := map[int]bool{1: true, 8: true, 16: true, 32: true, 64: true, 128: true, 256: true}
	if (*mode != "http" && *mode != "direct") || !allowed[*n] || *dir == "" || *reopen < 0 || *mr < 1 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "required: -bound {1,8,16,32,64,128,256} -dir NEW_DIRECTORY; reopen-every >= 0; mem-profile-rate >= 1")
		os.Exit(2)
	}
	if err := os.MkdirAll(*dir, 0700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	goruntime.MemProfileRate = *mr
	r := report{MemProfileRate: goruntime.MemProfileRate, Version: "alan/performance-p2/v1", Bound: *n, Sample: *sample, SourceLabel: *label, Schedule: "ordinary", ReopenEvery: *reopen, GoVersion: goruntime.Version(), GOOS: goruntime.GOOS, GOARCH: goruntime.GOARCH, GOMAXPROCS: goruntime.GOMAXPROCS(0), MaxRSSUnit: "platform-dependent", Profiled: *cp != "" || *ap != "" || *tr != "", Instrumented: counterSnapshot != nil, StartedUTC: time.Now().UTC().Format(time.RFC3339Nano)}
	r.Transport = frozenTransport(*mode, *endpoint)
	r.PrefixEnabled = *prefix
	r.PrefixConfiguration = frozenPrefixes()
	r.Prefixes = make([]prefixRecord, 0, *n)
	r.MetricNames = supportedMetrics()
	r.ResourceScope = "process-wide including same-process HTTP server; GC/CPU asynchronous; RSS cumulative high-water gauge"
	if goruntime.GOOS == "darwin" {
		r.MaxRSSUnit = "bytes"
	} else if goruntime.GOOS == "linux" {
		r.MaxRSSUnit = "KiB"
	}
	if *reopen > 0 {
		r.Schedule = "interval-reopen"
	}
	var stops []func()
	startErr := func() error {
		if *cp != "" {
			f, e := os.Create(*cp)
			if e != nil {
				return e
			}
			if e = pprof.StartCPUProfile(f); e != nil {
				f.Close()
				return e
			}
			stops = append(stops, func() { pprof.StopCPUProfile(); f.Close() })
		}
		if *tr != "" {
			f, e := os.Create(*tr)
			if e != nil {
				return e
			}
			if e = trace.Start(f); e != nil {
				f.Close()
				return e
			}
			stops = append(stops, func() { trace.Stop(); f.Close() })
		}
		return nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	err := startErr
	if err == nil {
		err = run(ctx, &r, *dir, *reopen)
	}
	cancel()
	for i := len(stops) - 1; i >= 0; i-- {
		stops[i]()
	}
	if *ap != "" {
		f, e := os.Create(*ap)
		if e == nil {
			e = pprof.Lookup("allocs").WriteTo(f, 0)
			e = errors.Join(e, f.Close())
		}
		err = errors.Join(err, e)
	}
	if err != nil {
		r.Error = err.Error()
	}
	r.ProcessWallNS = time.Since(processStart).Nanoseconds()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if e := encoder.Encode(r); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
