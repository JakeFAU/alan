package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/runtime"
)

const weatherProcessTimeout = 60 * time.Second

// Compile the example as an independent consumer. Go enforces the public
// import boundary here, even for a new production file added to the example.
func publicWeatherBinary(t *testing.T) string {
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
	module := "module example.org/independent-weather-agent\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + strconv.Quote(root) + "\n"
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
	binary := filepath.Join(dir, "weather-agent")
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
		t.Fatalf("independent developer could not build weather-agent: %v\n%s", err, output)
	}
	return binary
}

func weatherCommand(t *testing.T, binary string, args ...string) string {
	t.Helper()
	return weatherOutput(t, binary, append(args, "--json")...)
}

func weatherOutput(t *testing.T, binary string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), weatherProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", args[0], err, out)
	}
	return string(out)
}

func startWeather(t *testing.T, binary, endpoint string, allowance int) string {
	t.Helper()
	db := filepath.Join(t.TempDir(), "weather.sqlite")
	weatherCommand(t, binary, "start", "--db", db, "--lat", "26.93", "--lon", "-80.08", "--units", "fahrenheit", "--endpoint", endpoint, "--allowance", fmt.Sprint(allowance))
	return db
}

func deniedWeatherCommand(t *testing.T, binary string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), weatherProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), "ALAN_BARRIER=", "GORACE=atexit_sleep_ms=0")
	out, err := cmd.CombinedOutput()
	var e *exec.ExitError
	if !errors.As(err, &e) || e.ExitCode() < 1 || !strings.Contains(strings.ToLower(string(out)), "allowance") {
		t.Fatalf("approval beyond durable allowance should be refused, got %v: %s", err, out)
	}
}

type weatherPath struct {
	operation                    string
	request, selection, terminal int64
}

// Learn the policy's sequence numbers from a successful independent reference
// run. The test at each cut separately checks which state actually committed.
func measuredWeatherPath(t *testing.T, binary string) weatherPath {
	t.Helper()
	p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather, hold: true})
	db := startWeather(t, binary, p.endpoint(), 1)
	p.requireContacts(t, 0)
	child := launchWeather(t, binary, db, "run", nil)
	accepted := p.waitAccepted(t)
	before := weatherSnapshot(t, db)
	e, ok := before.Effect(accepted.operation)
	if !ok || e.AdapterResult != nil || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed {
		t.Fatalf("endpoint's acknowledged but withheld response is already known: %+v", e)
	}
	close(p.release)
	fresh := decodeWeather(t, child.wait(t)).read(t, 1)
	s, visible := inspectAndReplayWeather(t, binary, db, p, 1, 1)
	authenticWeather(t, s, fresh, 1, accepted)
	if fresh.Origin != "read_during_this_run" || visible.read(t, 1).Origin != "previously_retained" {
		t.Fatalf("fresh and retained provenance: %+v/%+v", fresh, visible)
	}
	e = weatherEffect(t, s, "weather-1")
	selected := selectedWeather(t, s, e)
	run, _ := s.Run(e.Source)
	t.Logf("reference: original=%s report=%q", e.OperationID, fresh.Report)
	return weatherPath{e.OperationID, e.Sequence, selected.Sequence, run.Sequence}
}

type weatherCrashCase struct {
	name, boundary, phase    string
	sequence                 int64
	stage                    runtime.EffectStage
	contacts, spent          int
	selected, terminal, lost bool
}

func (c weatherCrashCase) filter(path weatherPath) weatherCut {
	f := weatherCut{Boundary: c.boundary, Phase: c.phase, Subject: "weather-1", Generation: 1, Sequence: c.sequence}
	if c.sequence == 0 || c.sequence == path.request {
		f.Intent = path.operation
	}
	return f
}

func (c weatherCrashCase) checkStop(t *testing.T, path weatherPath, s *runtime.ExecutionSnapshot, v weatherView, p *weatherProvider) {
	t.Helper()
	p.requireContacts(t, c.contacts)
	run, ok := s.Run(runtime.RunID{Subject: "weather-1", Generation: 1})
	if !ok || (run.Projection.Status == "succeeded") != c.terminal {
		t.Fatalf("incorrect stopped run at %s: %+v", c.name, run)
	}
	if c.stage == "" {
		if len(s.Effects) != 0 || len(s.Operations) != 0 || run.Sequence >= path.request {
			t.Fatal("uncommitted request fabricated after death")
		}
		return
	}
	e := weatherEffect(t, s, "weather-1")
	if e.OperationID != path.operation || e.State != c.stage {
		t.Fatalf("stopped original=%+v, want stage %s for %s", e, c.stage, path.operation)
	}
	if (e.Authorization != nil) != (c.stage != runtime.EffectCommitted) {
		t.Fatalf("incorrect historical permission at %s", c.name)
	}
	if (e.AdapterAttempt != nil) != (c.spent == 1) {
		t.Fatalf("incorrect physical reservation at %s", c.name)
	}
	admitted := c.boundary == "dispatch.admitted" || c.contacts != 0
	if (e.DispatchAdmission != nil) != admitted {
		t.Fatalf("%s: retained local dispatch admission=%+v, want present=%t", c.name, e.DispatchAdmission, admitted)
	}
	if admitted && (e.DispatchAdmission.OperationID != e.OperationID || e.DispatchAdmission.PreparedDigest != e.AdapterAttempt.Prepared.Digest ||
		e.Dispatch.Maximum == nil || *e.Dispatch.Maximum != 1 || e.DispatchAdmission.AfterFact < 1) {
		t.Fatalf("physical admission does not identify one original, frozen request: %+v", e.DispatchAdmission)
	}
	if c.boundary == "dispatch.consider" {
		if e.Dispatch.Maximum == nil {
			t.Fatal("missing physical dispatch bound before admission")
		}
		if *e.Dispatch.Maximum != 0 {
			t.Fatalf("stopped before local admission with a spent reservation: dispatch upper bound %d, want zero", *e.Dispatch.Maximum)
		}
	}
	if (e.AdapterResult != nil) != (c.stage == runtime.EffectResultUnsettled || c.stage == runtime.EffectSettled) {
		t.Fatalf("incorrect response retention at %s", c.name)
	}
	if (e.Settlement != nil) != (c.stage == runtime.EffectSettled) || (selectedWeather(t, s, e) != nil) != c.selected {
		t.Fatalf("incorrect settlement or selection at %s", c.name)
	}
	if (v.read(t, 1).Report != "") != c.terminal {
		t.Fatalf("terminal report visibility at %s: %+v", c.name, v)
	}
	if c.lost {
		unresolvedWeather(t, s, v.read(t, 1), path.operation)
	}
}

func TestWeatherProcessRecovery(t *testing.T) {
	binary := publicWeatherBinary(t)
	path := measuredWeatherPath(t, binary)
	retainedCut := weatherCrashCase{name: "after_response_retention", boundary: "adapter.result-recorded", phase: "pre-settlement", stage: runtime.EffectResultUnsettled, contacts: 1, spent: 1}
	selectedCut := weatherCrashCase{name: "after_selection", boundary: "execution.committed", phase: "post-commit", sequence: path.selection, stage: runtime.EffectSettled, contacts: 1, spent: 1, selected: true}
	terminalCut := weatherCrashCase{name: "before_terminal", boundary: "execution.staged", phase: "terminal", sequence: path.terminal, stage: runtime.EffectSettled, contacts: 1, spent: 1, selected: true}
	cases := []weatherCrashCase{
		{name: "before_request_commit", boundary: "execution.staged", phase: "pre-commit", sequence: path.request},
		{name: "after_request_commit", boundary: "execution.committed", phase: "post-commit", sequence: path.request, stage: runtime.EffectCommitted},
		{name: "after_authorization", boundary: "intent.authorized", phase: "authorized", stage: runtime.EffectAuthorized},
		{name: "reserved_before_dispatch_admission", boundary: "dispatch.consider", phase: "pre-admission", stage: runtime.EffectReservedUnknown, spent: 1, lost: true},
		{name: "physical_dispatch_admitted_before_call", boundary: "dispatch.admitted", phase: "pre-call", stage: runtime.EffectReservedUnknown, spent: 1, lost: true},
		{name: "HTTP_response_before_retention", boundary: "adapter.response", phase: "pre-record", stage: runtime.EffectReservedUnknown, contacts: 1, spent: 1, lost: true},
		retainedCut,
		{name: "after_settlement", boundary: "adapter.settled", phase: "pre-consumption", stage: runtime.EffectSettled, contacts: 1, spent: 1},
		{name: "before_selection", boundary: "execution.staged", phase: "pre-commit", sequence: path.selection, stage: runtime.EffectSettled, contacts: 1, spent: 1},
		selectedCut,
		terminalCut,
		{name: "after_terminal", boundary: "execution.committed", phase: "terminal", sequence: path.terminal, stage: runtime.EffectSettled, contacts: 1, spent: 1, selected: true, terminal: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather})
			db := startWeather(t, binary, p.endpoint(), 1)
			f := c.filter(path)
			owner := launchWeather(t, binary, db, "run", &f)
			event := owner.awaitCut(t)
			owner.kill(t, event.witness())
			stopped, v := inspectAndReplayWeather(t, binary, db, p, 1, c.spent)
			c.checkStop(t, path, stopped, v, p)
			if c.name == "after_terminal" {
				status := weatherOutput(t, binary, "status", "--db", db)
				requireWeatherText(t, status, []string{"Previously retained", "Observation selected", "Local completion remains pending", "weather-agent resume"},
					[]string{"Resume may contact the provider", "weather-agent refresh"})
				p.requireContacts(t, 1)
			}
			pause := weatherInterruption(t, event.witness(), event.PID, c.contacts, stopped)
			if event.Attempt != "" {
				original := weatherEffect(t, stopped, "weather-1")
				if original.Authorization == nil || original.Authorization.ID != event.Attempt {
					t.Fatal("barrier acknowledged another authority")
				}
			}
			view := decodeWeather(t, weatherCommand(t, binary, "resume", "--db", db))
			after, retained := inspectAndReplayWeather(t, binary, db, p, 1, 1)
			if after.Namespace != stopped.Namespace {
				t.Fatal("recovery replaced the original computation")
			}
			e := weatherEffect(t, after, "weather-1")
			if e.OperationID != path.operation || e.OperationID != view.read(t, 1).Operation {
				t.Fatalf("recovery lost original operation %s", e.OperationID)
			}
			if c.lost {
				unresolvedWeather(t, after, retained.read(t, 1), path.operation)
				p.requireContacts(t, c.contacts)
				deniedWeatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
				p.requireContacts(t, c.contacts)
			} else {
				p.requireContacts(t, 1)
				authenticWeather(t, after, view.read(t, 1), 1, p.contacts()[0])
				authenticWeather(t, after, retained.read(t, 1), 1, p.contacts()[0])
				if c.contacts == 1 && view.read(t, 1).Origin != "previously_retained" {
					t.Fatal("recovered local data presented as a new physical observation")
				}
			}
			t.Logf("recovered original=%s stage=%s local_admission=%t contacts=%d charged=1/1 report=%q", e.OperationID, e.State, e.DispatchAdmission != nil, len(p.contacts()), retained.read(t, 1).Report)
			recordWeatherCase(t, after, p, pause)
		})
	}
	t.Run("two_interrupted_recoveries", func(t *testing.T) {
		p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather})
		db := startWeather(t, binary, p.endpoint(), 1)
		var priorEpoch int64
		var pauses []map[string]any
		for i, c := range []weatherCrashCase{retainedCut, selectedCut, terminalCut} {
			f := c.filter(path)
			action := "resume"
			if i == 0 {
				action = "run"
			}
			owner := launchWeather(t, binary, db, action, &f)
			event := owner.awaitCut(t)
			owner.kill(t, event.witness())
			s, v := inspectAndReplayWeather(t, binary, db, p, 1, 1)
			c.checkStop(t, path, s, v, p)
			pauses = append(pauses, weatherInterruption(t, event.witness(), event.PID, 1, s))
			run, _ := s.Run(runtime.RunID{Subject: "weather-1", Generation: 1})
			if run.Epoch <= priorEpoch {
				t.Fatal("owner reopening failed to advance activation")
			}
			priorEpoch = run.Epoch
		}
		_ = weatherCommand(t, binary, "resume", "--db", db)
		s, v := inspectAndReplayWeather(t, binary, db, p, 1, 1)
		p.requireContacts(t, 1)
		authenticWeather(t, s, v.read(t, 1), 1, p.contacts()[0])
		recordWeatherCase(t, s, p, pauses...)
	})
	t.Run("accepted_and_lost_then_explicitly_reobserved", func(t *testing.T) { weatherRefresh(t, binary, true) })
	t.Run("completed_then_explicitly_refreshed", func(t *testing.T) { weatherRefresh(t, binary, false) })
	t.Run("zero_allowance", func(t *testing.T) {
		p := newWeatherProvider(t)
		db := startWeather(t, binary, p.endpoint(), 0)
		_ = weatherCommand(t, binary, "run", "--db", db)
		s, v := inspectAndReplayWeather(t, binary, db, p, 0, 0)
		e := weatherEffect(t, s, "weather-1")
		if e.AdapterAttempt == nil || e.AdapterAttempt.Allowed || e.AdapterAttempt.Reason != "effect_limit" || v.read(t, 1).Report != "" || v.read(t, 1).State != "unavailable" {
			t.Fatalf("zero allowance did not retain an explicit refusal: %+v", e)
		}
		deniedWeatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
		p.requireContacts(t, 0)
		recordWeatherCase(t, s, p)
	})
	for name, reply := range wireFaults() {
		t.Run(name, func(t *testing.T) {
			p := newWeatherProvider(t, reply)
			db := startWeather(t, binary, p.endpoint(), 1)
			_ = weatherCommand(t, binary, "run", "--db", db)
			s, v := inspectAndReplayWeather(t, binary, db, p, 1, 1)
			p.requireContacts(t, 1)
			e := weatherEffect(t, s, "weather-1")
			if reply.close {
				unresolvedWeather(t, s, v.read(t, 1), p.contacts()[0].operation)
			} else {
				run, ok := s.Run(e.Source)
				disposition := "protocol_invalid"
				if reply.status != 200 {
					disposition = "http_rejected"
				}
				if !ok || run.Projection.Status != "failed" || e.State != runtime.EffectSettled || e.ResultDisposition != disposition || selectedWeather(t, s, e) == nil ||
					v.read(t, 1).State != "unavailable" || v.read(t, 1).Report != "" || v.read(t, 1).ObservedAt != "" || v.read(t, 1).Temperature != "" {
					t.Fatalf("bad external response produced weather or lost the original diagnostic: %+v / %+v", e, v)
				}
			}
			_ = weatherCommand(t, binary, "resume", "--db", db)
			p.requireContacts(t, 1)
			t.Logf("original=%s disposition=%s contacts=1 charged=1/1", e.OperationID, e.ResultDisposition)
			recordWeatherCase(t, s, p)
		})
	}
}

func weatherRefresh(t *testing.T, binary string, lose bool) {
	t.Helper()
	p := newWeatherProvider(t, weatherReply{status: 200, body: morningWeather, hold: lose}, weatherReply{status: 200, body: laterWeather})
	db := startWeather(t, binary, p.endpoint(), 2)
	first := launchWeather(t, binary, db, "run", nil)
	contact := p.waitAccepted(t)
	if lose {
		first.kill(t, fmt.Sprintf("independent endpoint accepted index=%d operation=%s with response withheld", contact.index, contact.operation))
	} else {
		_ = first.wait(t)
	}
	stopped, view := inspectAndReplayWeather(t, binary, db, p, 2, 1)
	var pauses []map[string]any
	if lose {
		pauses = append(pauses, weatherInterruption(t, "endpoint accepted index=1 and withheld response", first.cmd.Process.Pid, 1, stopped))
	}
	original := weatherEffect(t, stopped, "weather-1")
	if lose {
		before := weatherDBBytes(t, db)
		status := weatherOutput(t, binary, "status", "--db", db)
		requireWeatherText(t, status, []string{original.OperationID, "provider response is unknown", "Optional separate approval for read 2"},
			[]string{"weather-agent resume"})
		if status != weatherOutput(t, binary, "status", "--db", db) || before != weatherDBBytes(t, db) {
			t.Fatal("repeated fresh status altered the unknown cut")
		}
		p.requireContacts(t, 1)
	}
	for i := 0; i < 2; i++ {
		_ = weatherCommand(t, binary, "resume", "--db", db)
		next, v := inspectAndReplayWeather(t, binary, db, p, 2, 1)
		e := weatherEffect(t, next, "weather-1")
		if next.Namespace != stopped.Namespace || e.OperationID != contact.operation || !reflect.DeepEqual(original, e) {
			t.Fatal("reopening replaced the original result or authority")
		}
		if lose {
			unresolvedWeather(t, next, v.read(t, 1), contact.operation)
		} else {
			authenticWeather(t, next, v.read(t, 1), 1, contact)
		}
		p.requireContacts(t, 1)
	}
	for i := 0; i < 2; i++ {
		_ = weatherCommand(t, binary, "refresh", "--db", db, "--read", "2")
	}
	p.requireContacts(t, 1)
	if lose {
		requireWeatherText(t, weatherOutput(t, binary, "status", "--db", db),
			[]string{original.OperationID, "Resume will not retry this original", "Read 2. Approved", "Resume may contact the provider", "weather-agent resume"},
			[]string{"weather-agent refresh"})
	}
	served := decodeWeather(t, weatherCommand(t, binary, "resume", "--db", db))
	s, v := inspectAndReplayWeather(t, binary, db, p, 2, 2)
	p.requireContacts(t, 2)
	purpose := "refresh_after_report"
	if lose {
		purpose = "new_read_after_unknown_response"
	}
	if s.Namespace != stopped.Namespace || v.read(t, 2).Previous != contact.operation || v.read(t, 2).Operation == contact.operation || served.read(t, 2).Origin != "read_during_this_run" ||
		v.read(t, 2).ApprovalReason != purpose || !reflect.DeepEqual(weatherEffect(t, s, "weather-1"), original) || v.read(t, 1).Report != view.read(t, 1).Report {
		t.Fatal("fresh authorization lost or overwrote the original evidence")
	}
	authenticWeather(t, s, v.read(t, 2), 2, p.contacts()[1])
	authority, ok := s.Continuation(runtime.RunID{Subject: "weather-2", Generation: 1})
	if !ok || authority.SelectedAt == nil || authority.Approval.PredecessorID != contact.operation || authority.SuccessorOperationID != p.contacts()[1].operation {
		t.Fatalf("new read has no selected authority from the independently observed original: %+v", authority)
	}
	obs, ok := s.Observation(authority.ObservationID)
	if !ok || obs.Kind != "input" || obs.Provenance != "local-input" || obs.SelectedAt == nil || *obs.SelectedAt != *authority.SelectedAt || obs.Payload["previous_operation_id"].Text != contact.operation {
		t.Fatalf("continuation approval does not match the original selected local input: %+v", obs)
	}
	if lose {
		unresolvedWeather(t, s, v.read(t, 1), contact.operation)
		requireWeatherText(t, weatherOutput(t, binary, "status", "--db", db), []string{contact.operation, v.read(t, 2).Operation,
			"provider response is unknown", "does not settle it", "No remaining read slot"}, []string{"weather-agent refresh", "weather-agent resume"})
	} else {
		authenticWeather(t, s, v.read(t, 1), 1, contact)
	}
	for _, cmd := range [][]string{{"refresh", "--db", db, "--read", "2"}, {"resume", "--db", db}} {
		_ = weatherCommand(t, binary, cmd...)
	}
	deniedWeatherCommand(t, binary, "refresh", "--db", db, "--read", "3")
	p.requireContacts(t, 2)
	t.Logf("approved new read original=%s original_state=%s original_time=%q later=%s later_time=%q contacts=2 charged=2/2", contact.operation, v.read(t, 1).State, v.read(t, 1).ObservedAt, v.read(t, 2).Operation, v.read(t, 2).ObservedAt)
	recordWeatherCase(t, s, p, pauses...)
}
