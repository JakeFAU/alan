//go:build integration

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

const outingProcessTimeout = 90 * time.Second

// Compile the example as an independent consumer. Go enforces the public
// import boundary here, even for a new production file added to the example.
func publicOutingBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	module := "module example.org/independent-outing-planner\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + strconv.Quote(root) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "outing-planner")
	args := []string{"build", "-buildvcs=false", "-mod=mod", "-o", binary}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "-race" && setting.Value == "true" {
				args = append(args, "-race")
			}
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", append(args, ".")...)
	build.Dir, build.Env = dir, append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("independent developer could not build outing-planner: %v\n%s", err, output)
	}
	return binary
}

func outingProcessJSON(t *testing.T, binary, action, db string, extra ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), outingProcessTimeout)
	defer cancel()
	args := append([]string{action, "--db", db, "--json"}, extra...)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh %s: %v\n%s", action, err, output)
	}
	return output
}

func startOutingProcess(t *testing.T, binary, endpoint, variant string, dryPolicy bool) string {
	t.Helper()
	c := fixtureConfiguration()
	c.Endpoint = endpoint
	if dryPolicy {
		c.AllowedWeatherCodes = []int64{}
	}
	raw, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path, config := filepath.Join(dir, "outing.db"), filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, raw, 0600); err != nil {
		t.Fatal(err)
	}
	outingProcessJSON(t, binary, "start", path, "--config", config, "--weather", variant)
	return path
}

type outingContact struct {
	operation string
	latitude  string
	clientID  string
}

type outingEndpoint struct {
	url      string
	accepted chan outingContact
	mu       sync.Mutex
	contacts []outingContact
	firstErr string
}

func newOutingEndpoint(t *testing.T, holdFirst bool) *outingEndpoint {
	t.Helper()
	fixture, _, err := readFixture("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	handler := newFixtureHandler(fixture, "celsius")
	endpoint := &outingEndpoint{accepted: make(chan outingContact, 1)}
	listener, err := listenForFixture()
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encodedID := r.Header.Get(openmeteo.OperationHeader)
			id, decodeErr := base64.RawURLEncoding.DecodeString(encodedID)
			contact := outingContact{
				operation: string(id),
				latitude:  r.URL.Query().Get("latitude"),
				clientID:  r.Header.Get(openmeteo.ClientRequestHeader),
			}
			_, lookupErr := handler.lookup(r)
			endpoint.mu.Lock()
			endpoint.contacts = append(endpoint.contacts, contact)
			first := len(endpoint.contacts) == 1
			if (decodeErr != nil || len(id) == 0 || contact.clientID == "" || lookupErr != nil) && endpoint.firstErr == "" {
				endpoint.firstErr = r.Method + " " + r.URL.RequestURI() + " operation=" + encodedID
			}
			endpoint.mu.Unlock()
			if first && holdFirst {
				endpoint.accepted <- contact
				select {
				case <-r.Context().Done():
				case <-release:
				}
				return
			}
			handler.ServeHTTP(w, r)
		}),
	}
	endpoint.url = "http://" + listener.Addr().String() + fixturePath
	t.Cleanup(func() {
		close(release)
		_ = server.Close()
	})
	go func() { _ = server.Serve(listener) }()
	return endpoint
}

func (e *outingEndpoint) requireContacts(t *testing.T, s *runtime.ExecutionSnapshot, latitudes ...string) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.firstErr != "" || len(e.contacts) != len(latitudes) {
		t.Fatalf("endpoint accepted %v, want latitudes %v, invalid request %q", e.contacts, latitudes, e.firstErr)
	}
	seen := make(map[string]bool)
	for i, c := range e.contacts {
		effect, ok := s.Effect(c.operation)
		op, opOK := s.Operation(c.operation)
		if c.latitude != latitudes[i] || seen[c.operation] || !ok || !opOK ||
			op.Payload[openmeteo.Latitude].Text != c.latitude || effect.DispatchAdmission == nil ||
			effect.DispatchAdmission.OperationID != c.operation || effect.AdapterAttempt == nil ||
			effect.AdapterAttempt.Prepared.Request.ID != c.operation ||
			openmeteo.ClientRequestID(effect.AdapterAttempt.Prepared.Request.Key) != c.clientID {
			t.Fatalf("endpoint contact %d does not identify the original candidate operation: %+v", i+1, c)
		}
		seen[c.operation] = true
	}
}

func outingSnapshot(t *testing.T, path string) *runtime.ExecutionSnapshot {
	t.Helper()
	s, err := runtime.InspectExecution(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func outingHistory(t *testing.T, s *runtime.ExecutionSnapshot) []runtime.ExecutionStep {
	t.Helper()
	page, err := s.History(0, runtime.MaxHistoryPage)
	if err != nil || page.More {
		t.Fatalf("unable to compare full process witness history: %v (more=%t)", err, page.More)
	}
	return page.Steps
}

func requireOutingRecovery(t *testing.T, cut, final *runtime.ExecutionSnapshot) {
	t.Helper()
	if cut.Namespace != final.Namespace || !reflect.DeepEqual(cut.Deployment, final.Deployment) ||
		!reflect.DeepEqual(cut.Composition, final.Composition) || !reflect.DeepEqual(cut.Permissions, final.Permissions) ||
		len(cut.Runs) != 2 || len(final.Runs) != 2 {
		t.Fatal("fresh owner changed namespace, deployment, scope, permission, or generation count")
	}
	for _, rc := range cut.Runs {
		rf, ok := final.Run(rc.ID)
		epoch := rc.Epoch
		if rc.Projection.Status == "open" {
			epoch++
		}
		if !ok || rc.ID.Generation != 1 || rf.Epoch != epoch || rf.ProgramID != rc.ProgramID ||
			rf.Revision != rc.Revision || !reflect.DeepEqual(rf.Initial, rc.Initial) || rf.Sequence < rc.Sequence {
			t.Fatalf("run %v failed identity/epoch recovery from epoch %d to %d: %+v", rc.ID, rc.Epoch, epoch, rf)
		}
	}
	before, after := outingHistory(t, cut), outingHistory(t, final)
	if len(after) < len(before) || !reflect.DeepEqual(before, after[:len(before)]) {
		t.Fatal("fresh owner rewrote the retained reduction history prefix")
	}
	for _, op := range cut.Operations {
		saved, ok := final.Operation(op.ID)
		if !ok || !reflect.DeepEqual(op, *saved) {
			t.Fatalf("fresh owner lost or rewrote original operation %s", op.ID)
		}
	}
	for _, e := range cut.Effects {
		if e.Settlement != nil {
			saved, ok := final.Effect(e.OperationID)
			if !ok || !reflect.DeepEqual(e, saved) {
				t.Fatalf("fresh owner changed original settled forecast %s", e.OperationID)
			}
		}
	}
	for _, o := range cut.Observations {
		saved, ok := final.Observation(o.ID)
		if !ok {
			t.Fatalf("fresh owner lost original observation %s", o.ID)
		}
		copy := *saved
		if o.SelectedAt == nil {
			copy.SelectedAt = nil
		}
		if !reflect.DeepEqual(copy, *o) {
			t.Fatalf("fresh owner changed original observation %s", o.ID)
		}
	}
}

func verifyOutingOfflineReaders(t *testing.T, binary, path string, s *runtime.ExecutionSnapshot, endpoint *outingEndpoint, want string, contacts ...string) {
	t.Helper()
	inspect := outingProcessJSON(t, binary, "inspect", path)
	var report plannerReport
	if err := json.Unmarshal(inspect, &report); err != nil {
		t.Fatal(err)
	}
	if report.Namespace != s.Namespace || report.Outcome != want || !reflect.DeepEqual(report.Runs, s.Runs) ||
		!slices.EqualFunc(report.Operations, s.Operations, func(a, b runtime.Intent) bool {
			return reflect.DeepEqual(a, b)
		}) ||
		!reflect.DeepEqual(report.History, outingHistory(t, s)) {
		t.Fatalf("fresh inspect does not expose the retained run/operation history or outcome %s", want)
	}
	endpoint.requireContacts(t, s, contacts...)
	if !slices.EqualFunc(report.Effects, s.Effects, func(a, b *runtime.ExecutionEffect) bool { return reflect.DeepEqual(a, b) }) {
		t.Fatal("fresh inspect changed retained effects")
	}
	if !slices.EqualFunc(report.Observations, s.Observations, func(a, b *runtime.ObservationView) bool { return reflect.DeepEqual(a, b) }) {
		t.Fatal("fresh inspect changed retained observations")
	}
	if status := outingProcessJSON(t, binary, "status", path); string(status) != string(inspect) {
		t.Fatal("fresh status does not match the inspected cut")
	}
	endpoint.requireContacts(t, s, contacts...)
	replay := outingProcessJSON(t, binary, "replay", path)
	var replayReport plannerReport
	if err := json.Unmarshal(replay, &replayReport); err != nil || replayReport.Replay == nil ||
		replayReport.Replay.Namespace != s.Namespace || len(replayReport.Replay.Runs) != 2 {
		t.Fatalf("offline replay did not report the original two runs: %v", err)
	}
	replayed := replayReport.Replay
	replayReport.Replay = nil
	if !reflect.DeepEqual(report, replayReport) {
		t.Fatal("offline replay report does not match inspection")
	}
	for _, r := range replayed.Runs {
		got, ok := s.Run(r.ID)
		if !ok || !reflect.DeepEqual(r, got.RunState) {
			t.Fatalf("offline replay differs for %v from the public snapshot", r.ID)
		}
	}
	endpoint.requireContacts(t, s, contacts...)
	if after := outingSnapshot(t, path); !reflect.DeepEqual(s, after) {
		t.Fatal("fresh offline readers mutated the retained cut")
	}
}

func requireOutingSelection(t *testing.T, s *runtime.ExecutionSnapshot, operationID string, run runtime.RunID) *runtime.ReductionRef {
	t.Helper()
	var chosen *runtime.ReductionRef
	for _, o := range s.Observations {
		if o.SourceOperationID != operationID || o.Run != run || o.SelectedAt == nil {
			continue
		}
		step, ok := s.Reduction(*o.SelectedAt)
		if chosen != nil || o.SelectedAt.Run != run || !ok || step.SelectedObservationID != o.ID {
			t.Fatalf("operation %s has an invalid or duplicate selection in %s", operationID, run.Subject)
		}
		chosen = o.SelectedAt
	}
	if chosen == nil {
		t.Fatalf("%s did not consume original operation %s", run.Subject, operationID)
	}
	return chosen
}

func outingOperation(t *testing.T, s *runtime.ExecutionSnapshot, obligation string) runtime.Intent {
	t.Helper()
	var op runtime.Intent
	for _, i := range s.Operations {
		if i.Obligation == obligation {
			if op.ID != "" {
				t.Fatalf("multiple %s operations", obligation)
			}
			op = i
		}
	}
	if op.ID == "" {
		t.Fatalf("missing %s operation", obligation)
	}
	return op
}

func outingForecast(t *testing.T, s *runtime.ExecutionSnapshot, latitude string) *runtime.ExecutionEffect {
	t.Helper()
	for _, i := range s.Operations {
		if i.Kind == "adapter" && i.Payload[openmeteo.Latitude].Text == latitude {
			e, ok := s.Effect(i.ID)
			if ok {
				return e
			}
		}
	}
	t.Fatalf("missing original forecast for latitude %s", latitude)
	return nil
}

func outingCutAt(s *runtime.ExecutionSnapshot, run runtime.RunID, sequence int64, at, phase string) boundary.Event {
	r, _ := s.Run(run)
	return boundary.Event{
		Boundary: at, Phase: phase, Subject: run.Subject,
		Generation: run.Generation, Epoch: r.Epoch, Sequence: sequence,
	}
}

func requireOutingCommit(t *testing.T, s *runtime.ExecutionSnapshot, event boundary.Event, op runtime.Intent) {
	t.Helper()
	run := runtime.RunID{Subject: event.Subject, Generation: event.Generation}
	step, ok := s.Reduction(runtime.ReductionRef{Run: run, Sequence: event.Sequence})
	r, runOK := s.Run(run)
	if !ok || !runOK || r.Sequence != event.Sequence || step.Epoch != event.Epoch ||
		op.ID != event.Intent || op.Run != run || op.Sequence != step.Sequence ||
		!slices.Equal(step.OperationIDs, []string{op.ID}) {
		t.Fatalf("acknowledged commit %+v is inconsistent with original operation %+v", event, op)
	}
}

func TestOutingProcessRecovery(t *testing.T) {
	binary := publicOutingBinary(t)
	endpoint := newOutingEndpoint(t, false)
	baselinePath := startOutingProcess(t, binary, endpoint.url, assessFirst, false)
	outingProcessJSON(t, binary, "run", baselinePath)
	baseline := outingSnapshot(t, baselinePath)
	endpoint.requireContacts(t, baseline, "40.8", "40.7")
	requireDecision(t, baseline, recommended, "cedar")
	baseOuting, ok := baseline.Run(outingRun)
	if !ok {
		t.Fatal("baseline lacks the original Outing run")
	}
	first := outingForecast(t, baseline, "40.8")
	second := outingForecast(t, baseline, "40.7")

	for _, tc := range []struct {
		name     string
		filter   boundary.Event
		before   []string
		decision string
	}{
		{"proposal committed", outingCutAt(baseline, outingRun, outingOperation(t, baseline, proposalPort).Sequence, "execution.committed", "post-commit"), nil, noMatch},
		{"result recorded", outingCutAt(baseline, weatherRun, first.Sequence, "adapter.result-recorded", "pre-settlement"), []string{"40.8"}, recommended},
		{"decision committed", outingCutAt(baseline, outingRun, outingOperation(t, baseline, decisionPort).Sequence, "execution.committed", "post-commit"), []string{"40.8", "40.7"}, recommended},
		{"next candidate committed", outingCutAt(baseline, weatherRun, second.Sequence, "execution.committed", "post-commit"), []string{"40.8"}, recommended},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newOutingEndpoint(t, false)
			db := startOutingProcess(t, binary, p.url, assessFirst, tc.decision == noMatch)
			initial := outingSnapshot(t, db)
			p.requireContacts(t, initial)
			owner, event := launchOutingOwner(t, binary, "run", db, &tc.filter)
			cut := outingSnapshot(t, db)
			requireOutingRecovery(t, initial, cut)
			p.requireContacts(t, cut, tc.before...)
			var retainedResult *runtime.ExecutionEffect
			switch tc.name {
			case "proposal committed":
				requireOutingCommit(t, cut, event, outingOperation(t, cut, proposalPort))
				if len(cut.Operations) != 1 || len(cut.Observations) != 0 {
					t.Fatal("proposal cut already admitted or consumed the proposal")
				}
			case "result recorded":
				retainedResult = outingForecast(t, cut, "40.8")
				if retainedResult.OperationID != event.Intent || retainedResult.Source != weatherRun ||
					retainedResult.Sequence != event.Sequence || retainedResult.AdapterAttempt == nil ||
					retainedResult.AdapterAttempt.AuthorizationID != event.Attempt ||
					retainedResult.AdapterResult == nil || retainedResult.ResultDisposition != "validated" ||
					retainedResult.Settlement != nil || len(cut.Effects) != 1 {
					t.Fatalf("acknowledged pre-settlement cut has wrong effect: %+v", retainedResult)
				}
				for _, obs := range cut.Observations {
					if obs.SourceOperationID == event.Intent {
						t.Fatal("newly recorded forecast had already reached Weather's inbox")
					}
				}
			case "decision committed":
				requireOutingCommit(t, cut, event, outingOperation(t, cut, decisionPort))
				for _, a := range cut.Admissions {
					if a.SourceOperationID == event.Intent {
						t.Fatal("decision was already delivered at acknowledged commit")
					}
				}
				for _, o := range cut.Observations {
					if o.SourceOperationID == event.Intent {
						t.Fatal("decision was already in the inbox at acknowledged commit")
					}
				}
			case "next candidate committed":
				prior := outingForecast(t, cut, "40.8")
				next := outingForecast(t, cut, "40.7")
				intent, ok := cut.Operation(next.OperationID)
				if !ok {
					t.Fatal("next candidate has no original intent")
				}
				requireOutingCommit(t, cut, event, *intent)
				if len(cut.Effects) != 2 || prior.AdapterResult == nil || prior.Settlement == nil ||
					next.AdapterAttempt != nil || next.DispatchAdmission != nil || next.AdapterResult != nil || next.State != runtime.EffectCommitted {
					t.Fatalf("prior forecast or pending original changed: prior=%+v next=%+v", prior, next)
				}
				if at := requireOutingSelection(t, cut, prior.OperationID, weatherRun); at.Sequence >= next.Sequence {
					t.Fatal("Weather committed the next forecast before selecting the first")
				}
				retainedResult = prior
			}
			t.Logf("acknowledged %s/%s at %s/%d epoch=%d sequence=%d operation=%s attempt=%s accepted=%d",
				event.Boundary, event.Phase, event.Subject, event.Generation, event.Epoch, event.Sequence, event.Intent, event.Attempt, len(tc.before))
			owner.kill(t)
			verifyOutingOfflineReaders(t, binary, db, cut, p, pendingOutcome, tc.before...)
			var admittedDecision string
			if tc.name == "decision committed" {
				original := outingOperation(t, cut, decisionPort)
				filter := outingCutAt(cut, outingRun, baseOuting.Sequence, "execution.staged", "terminal")
				filter.Epoch++
				resumed, staged := launchOutingOwner(t, binary, "resume", db, &filter)
				beforeTerminal := outingSnapshot(t, db)
				requireOutingRecovery(t, cut, beforeTerminal)
				admittedDecision = requireDecisionBeforeOutingTerminal(t, beforeTerminal, staged, original)
				p.requireContacts(t, beforeTerminal, tc.before...)
				if report := resumed.continueAndRead(t); report.Outcome != tc.decision {
					t.Fatalf("released resume reported %q, want %s", report.Outcome, tc.decision)
				}
			} else {
				raw := outingProcessJSON(t, binary, "resume", db)
				plannerOutcome(t, string(raw), tc.decision)
			}
			final := outingSnapshot(t, db)
			requireOutingRecovery(t, cut, final)
			park := "cedar"
			if tc.decision == noMatch {
				park = ""
			}
			requireDecision(t, final, tc.decision, park)
			decision := outingOperation(t, final, decisionPort)
			at := requireOutingSelection(t, final, decision.ID, weatherRun)
			if admittedDecision != "" {
				step, ok := final.Reduction(*at)
				if !ok || step.SelectedObservationID != admittedDecision {
					t.Fatal("Weather did not select the decision admitted before Outing terminated")
				}
			}
			if retainedResult != nil {
				got, ok := final.Effect(retainedResult.OperationID)
				if !ok || got.Settlement == nil || !reflect.DeepEqual(got.AdapterResult, retainedResult.AdapterResult) ||
					!reflect.DeepEqual(got.AdapterAttempt, retainedResult.AdapterAttempt) ||
					!reflect.DeepEqual(got.DispatchAdmission, retainedResult.DispatchAdmission) {
					t.Fatal("recovery replaced earlier native forecast evidence")
				}
				requireOutingSelection(t, final, got.OperationID, weatherRun)
			}
			verifyOutingOfflineReaders(t, binary, db, final, p, tc.decision, "40.8", "40.7")
			t.Logf("completed %s, decision=%s selected by Weather at sequence %d, endpoint contacts=2", tc.decision, decision.ID, at.Sequence)
		})
	}

	t.Run("provider accepted response lost", func(t *testing.T) {
		p := newOutingEndpoint(t, true)
		db := startOutingProcess(t, binary, p.url, assessFirst, false)
		initial := outingSnapshot(t, db)
		owner, _ := launchOutingOwner(t, binary, "run", db, nil)
		var contact outingContact
		select {
		case contact = <-p.accepted:
		case <-time.After(outingProcessTimeout):
			raw, _ := os.ReadFile(owner.log)
			t.Fatalf("endpoint did not acknowledge an accepted GET:\n%s", raw)
		}
		cut := outingSnapshot(t, db)
		requireOutingRecovery(t, initial, cut)
		original := requireUnknownOuting(t, cut, contact.operation)
		p.requireContacts(t, cut, "40.8")
		t.Logf("endpoint acknowledged GET at %s with response withheld for original=%s", contact.latitude, contact.operation)
		owner.kill(t)
		verifyOutingOfflineReaders(t, binary, db, cut, p, unknownForecast, "40.8")
		for i := range 2 {
			raw := outingProcessJSON(t, binary, "resume", db)
			plannerOutcome(t, string(raw), unknownForecast)
			next := outingSnapshot(t, db)
			requireOutingRecovery(t, cut, next)
			requireUnadvancedOuting(t, cut, next)
			if got := requireUnknownOuting(t, next, contact.operation); !reflect.DeepEqual(got, original) {
				t.Fatal("resume changed the withheld original's retained effect")
			}
			verifyOutingOfflineReaders(t, binary, db, next, p, unknownForecast, "40.8")
			t.Logf("fresh resume %d preserved the unknown original and one endpoint contact", i+1)
			cut = next
		}
	})

	t.Run("confirmation first remains blocked", func(t *testing.T) {
		p := newOutingEndpoint(t, false)
		measure := startOutingProcess(t, binary, p.url, confirmFirst, false)
		raw := outingProcessJSON(t, binary, "run", measure)
		plannerOutcome(t, string(raw), pendingOutcome)
		blocked := outingSnapshot(t, measure)
		requireOutingWaits(t, blocked)
		history := outingHistory(t, blocked)
		last := history[len(history)-1]
		filter := outingCutAt(blocked, last.Run, last.Sequence, "execution.committed", "post-commit")

		path := startOutingProcess(t, binary, p.url, confirmFirst, false)
		initial := outingSnapshot(t, path)
		owner, event := launchOutingOwner(t, binary, "run", path, &filter)
		cut := outingSnapshot(t, path)
		requireOutingRecovery(t, initial, cut)
		requireOutingWaits(t, cut)
		step, ok := cut.Reduction(runtime.ReductionRef{Run: last.Run, Sequence: event.Sequence})
		if !ok || step.Epoch != event.Epoch || step.AfterPC != last.AfterPC || len(step.OperationIDs) != 0 || event.Intent != "" {
			t.Fatalf("wrong acknowledged reciprocal wait cut: %+v, step=%+v", event, step)
		}
		p.requireContacts(t, cut)
		t.Logf("acknowledged blocked cut at %s/%d epoch=%d sequence=%d", event.Subject, event.Generation, event.Epoch, event.Sequence)
		owner.kill(t)
		verifyOutingOfflineReaders(t, binary, path, cut, p, pendingOutcome)
		raw = outingProcessJSON(t, binary, "resume", path)
		plannerOutcome(t, string(raw), pendingOutcome)
		next := outingSnapshot(t, path)
		requireOutingRecovery(t, cut, next)
		requireUnadvancedOuting(t, cut, next)
		requireOutingWaits(t, next)
		verifyOutingOfflineReaders(t, binary, path, next, p, pendingOutcome)
	})
}

// A staged terminal reduction is still uncommitted. The accepted admission
// and inbox observation must already be visible to a separate snapshot reader.
func requireDecisionBeforeOutingTerminal(t *testing.T, s *runtime.ExecutionSnapshot, event boundary.Event, decision runtime.Intent) string {
	t.Helper()
	outing, ok := s.Run(outingRun)
	terminal := runtime.ReductionRef{Run: outingRun, Sequence: event.Sequence}
	_, committed := s.Reduction(terminal)
	if !ok || outing.Projection.Status != "open" || outing.Epoch != event.Epoch ||
		outing.Sequence+1 != event.Sequence || committed {
		t.Fatalf("terminal acknowledgement %+v did not witness an open Outing projection: %+v", event, outing)
	}
	var accepted *runtime.AdmissionView
	for _, a := range s.Admissions {
		if a.SourceOperationID != decision.ID {
			continue
		}
		if accepted != nil || !a.Accepted || a.Domain != "message" || a.Source != outingRun ||
			a.LogicalID != decision.ID || a.Target != weatherRun.Subject || a.RequestedGeneration != decision.Generation ||
			a.Obligation != decisionPort || a.Bound == nil || *a.Bound != weatherRun || a.ObservationID == "" {
			t.Fatalf("wrong admission for original decision %s: %+v", decision.ID, a)
		}
		accepted = a
	}
	if accepted == nil {
		t.Fatalf("Outing staged its terminal reduction without delivering decision %s", decision.ID)
	}
	obs, ok := s.Observation(accepted.ObservationID)
	producer := runtime.ReductionRef{Run: decision.Run, Sequence: decision.Sequence}
	if !ok || obs.Run != weatherRun || obs.Kind != "message" || obs.SourceOperationID != decision.ID ||
		obs.ProducerReduction == nil || *obs.ProducerReduction != producer || obs.Obligation != decisionPort ||
		!slices.Contains(obs.Admissions, accepted.ID) || !reflect.DeepEqual(obs.Payload, decision.Payload) {
		t.Fatalf("accepted decision %s did not create the exact Weather observation: %+v", decision.ID, obs)
	}
	t.Logf("acknowledged %s/%s pid=%d outing/1 epoch=%d sequence=%d; Outing open at sequence %d with accepted decision=%s admission=%s observation=%s",
		event.Boundary, event.Phase, event.PID, event.Epoch, event.Sequence, outing.Sequence, decision.ID, accepted.ID, obs.ID)
	return obs.ID
}

func requireUnknownOuting(t *testing.T, s *runtime.ExecutionSnapshot, operationID string) *runtime.ExecutionEffect {
	t.Helper()
	effect, ok := s.Effect(operationID)
	if !ok || len(s.Effects) != 1 || len(s.Operations) != 2 || effect.Source != weatherRun ||
		effect.DispatchAdmission == nil || effect.AdapterAttempt == nil || !effect.AdapterAttempt.Allowed ||
		effect.AdapterResult != nil || effect.Settlement != nil || effect.ResultDisposition != "" {
		t.Fatalf("original %s did not remain admitted and unresolved: %+v", operationID, effect)
	}
	for _, o := range s.Observations {
		if o.SourceOperationID == operationID {
			t.Fatal("withheld response produced an observation")
		}
	}
	for _, r := range s.Runs {
		if r.Projection.Status != "open" {
			t.Fatalf("%s gave a terminal answer without a forecast result", r.ID.Subject)
		}
	}
	return effect
}

func requireUnadvancedOuting(t *testing.T, cut, next *runtime.ExecutionSnapshot) {
	t.Helper()
	if len(outingHistory(t, cut)) != len(outingHistory(t, next)) ||
		len(cut.Operations) != len(next.Operations) || len(cut.Observations) != len(next.Observations) ||
		!reflect.DeepEqual(cut.Admissions, next.Admissions) || !reflect.DeepEqual(cut.Effects, next.Effects) ||
		!reflect.DeepEqual(cut.Pending, next.Pending) {
		t.Fatal("resume advanced a blocked cut")
	}
	for _, r := range cut.Runs {
		reopened, _ := next.Run(r.ID)
		if !reflect.DeepEqual(r.Projection, reopened.Projection) || !reflect.DeepEqual(r.Wait, reopened.Wait) {
			t.Fatalf("blocked projection or wait changed for %s", r.ID.Subject)
		}
	}
}

func requireOutingWaits(t *testing.T, s *runtime.ExecutionSnapshot) {
	t.Helper()
	proposal := outingOperation(t, s, proposalPort)
	requireOutingSelection(t, s, proposal.ID, weatherRun)
	if len(s.Operations) != 1 || len(s.Effects) != 0 {
		t.Fatal("confirmation-first realized a forecast or a reciprocal reply")
	}
	for _, r := range s.Runs {
		wantSource, wantObligation := weatherSubject, assessmentPort
		if r.ID == weatherRun {
			wantSource, wantObligation = outingSubject, decisionPort
		}
		w := r.Wait
		if r.Projection.Status != "open" || w == nil || !w.Resolved || w.Kind != "message" ||
			w.Source != wantSource || w.SourceGeneration != 1 || w.Obligation != wantObligation || len(w.MatchingObservationIDs) != 0 {
			t.Fatalf("wrong retained reciprocal wait for %s: %+v", r.ID.Subject, w)
		}
	}
}
