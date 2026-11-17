//go:build integration

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

type rawModelFixture struct {
	dir        string
	config     configuration
	provisions []runtime.Provision
	options    deploymentOptions
}

func rawModelCommand(t *testing.T, args ...string) []byte {
	t.Helper()
	command := exec.CommandContext(t.Context(), "python3", append([]string{"-I", "-B"}, args...)...)
	output, err := command.CombinedOutput()
	recordOutingCommand(t, command.Args, output, err)
	if err != nil {
		t.Fatalf("independent Python command: %v\n%s", err, output)
	}
	return output
}

func newRawModelFixture(t *testing.T, count int, options deploymentOptions, codes []int, unknown bool) rawModelFixture {
	return newRawModelFixtureEndpoint(t, count, options, codes, unknown, "")
}

// A process fixture uses the independently acknowledged endpoint while freezing
// the same native response bytes for the raw checker before creating any Store.
func newRawModelFixtureEndpoint(t *testing.T, count int, options deploymentOptions, codes []int, unknown bool, endpoint string) rawModelFixture {
	t.Helper()
	dir := t.TempDir()
	if root := os.Getenv("ALAN_RAW_CAPTURE_DIR"); root != "" {
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		var err error
		dir, err = os.MkdirTemp(root, strings.ReplaceAll(t.Name(), "/", "-")+"-")
		if err != nil {
			t.Fatal(err)
		}
	}
	c := fixtureConfiguration()
	if count == 0 {
		c.Slots = nil
	} else if count == 1 {
		c.Parks = c.Parks[1:]
	}
	plan, err := c.normalize()
	if err != nil {
		t.Fatal(err)
	}
	entries := []fixtureEntry{}
	for index, candidate := range plan.Candidates {
		if !candidate.Eligible {
			continue
		}
		code := 1
		if index < len(codes) {
			code = codes[index]
		}
		body := fmt.Sprintf(`{"latitude":%s,"longitude":%s,"utc_offset_seconds":0,"timezone":"GMT","hourly_units":{"time":"iso8601","temperature_2m":"°C","weather_code":"wmo code"},"hourly":{"time":["2026-10-11T15:00","2026-10-11T16:00"],"temperature_2m":[17.2,18.3],"weather_code":[%d,2]}}`, candidate.Latitude, candidate.Longitude, code)
		entries = append(entries, fixtureEntry{fixtureTuple: fixtureTuple{Latitude: candidate.Latitude, Longitude: candidate.Longitude, StartHour: candidate.Start, EndHour: candidate.End}, Status: 200, Headers: map[string]string{"Content-Type": "application/json"}, Body: &body})
	}
	fixtureBytes, err := json.Marshal(struct {
		Version   int            `json:"version"`
		Reference string         `json:"reference_time"`
		Entries   []fixtureEntry `json:"entries"`
	}{1, c.ReferenceTime, entries})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "" {
		fixtureBytes, err = os.ReadFile("testdata/fixture.json")
		if err != nil {
			t.Fatal(err)
		}
	}
	fixturePathname := filepath.Join(dir, "fixture.json")
	if err := os.WriteFile(fixturePathname, fixtureBytes, 0600); err != nil {
		t.Fatal(err)
	}
	fixture, _, err := readFixture(fixturePathname)
	if err != nil {
		t.Fatal(err)
	}
	handler := newFixtureHandler(fixture, c.Units)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unknown {
			if _, err := handler.lookup(r); err != nil {
				t.Error(err)
			}
			connection, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				err = connection.Close()
			}
			if err != nil {
				t.Error(err)
			}
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	c.Endpoint = server.URL + fixturePath
	if endpoint != "" {
		c.Endpoint = endpoint
	}
	configBytes, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "derived-configuration.json"), configBytes, 0600); err != nil {
		t.Fatal(err)
	}
	baseline, err := c.provision(assessFirst)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := c.provisionDeployment(options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := freezeDeploymentManifest(configBytes, plan, baseline, candidate, options)
	if err := writeDeploymentManifest(dir, manifest); err != nil {
		t.Fatal(err)
	}
	t.Logf("raw capture: %s", dir)
	return rawModelFixture{dir: dir, config: c, provisions: []runtime.Provision{baseline, candidate}, options: options}
}

func (f rawModelFixture) create(t *testing.T, candidate bool) (*runtime.Store, string) {
	t.Helper()
	index, name := 0, "baseline"
	if candidate {
		index, name = 1, "candidate"
	}
	path := filepath.Join(f.dir, name+"-live.db")
	owner, err := runtime.Create(t.Context(), path, f.provisions[index], runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := owner.Close(); err != nil {
			t.Error(err)
		}
	})
	return owner, path
}

func (f rawModelFixture) run(ctx context.Context, owner *runtime.Store) error {
	adapter, err := openmeteo.NewHourly(f.config.Endpoint)
	if err != nil {
		return err
	}
	_, err = owner.RunProgress(ctx, runtime.RunOptions{Adapters: map[string]effects.Adapter{capability: adapter}, MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1})
	return err
}

func (f rawModelFixture) capture(t *testing.T, path, name, want string, report bool) string {
	t.Helper()
	captured := filepath.Join(f.dir, name+".db")
	rawModelCommand(t, "model/test_evidence.py", "capture", path, captured)
	args := []string{"model/run.py", "check", "--store", captured, "--configuration", filepath.Join(f.dir, "derived-configuration.json"), "--fixture", filepath.Join(f.dir, "fixture.json"), "--manifest", filepath.Join(f.dir, "deployment-manifest.json"), "--output", filepath.Join(f.dir, name+"-check.json")}
	if report {
		output := plannerOutput(t, "inspect", "--db", path, "--json")
		reportPath := filepath.Join(f.dir, name+"-report.json")
		if err := os.WriteFile(reportPath, []byte(output), 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--report", reportPath)
	}
	output := rawModelCommand(t, args...)
	var checked struct {
		Accepted       bool   `json:"accepted"`
		Classification string `json:"classification"`
	}
	if err := json.Unmarshal(output, &checked); err != nil {
		t.Fatal(err)
	}
	if !checked.Accepted || checked.Classification != want {
		t.Fatalf("independent check: %s, want %s", output, want)
	}
	return captured
}

func TestDelegatedRawModel(t *testing.T) {
	for _, tc := range []struct {
		name          string
		count         int
		allowance     int64
		codes         []int
		unknown       bool
		variant, want string
	}{
		{"empty", 0, 1, nil, false, delegatedWeather, "complete"},
		{"one", 1, 1, nil, false, delegatedWeather, "complete"},
		{"two", 2, 1, []int{1, 61}, false, delegatedWeather, "complete"},
		{"no_match", 2, 1, []int{61, 61}, false, delegatedWeather, "complete"},
		{"unavailable", 1, 1, []int{999}, false, delegatedWeather, "diagnostic_terminal"},
		{"mixed_unavailable_suitable", 2, 1, []int{999, 1}, false, delegatedWeather, "complete"},
		{"refused", 2, 0, nil, false, delegatedWeather, "refused"},
		{"refused_one", 1, 0, nil, false, delegatedWeather, "refused"},
		{"refused_empty", 0, 0, nil, false, delegatedWeather, "refused"},
		{"unknown", 2, 1, nil, true, delegatedWeather, "unknown_forecast"},
		{"confirm_first", 2, 0, []int{1, 61}, false, confirmFirst, "confirm_first_cycle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRawModelFixture(t, tc.count, deploymentOptions{Weather: tc.variant, HelperAdmission: tc.allowance}, tc.codes, tc.unknown)
			baseline, basePath := f.create(t, false)
			f.capture(t, basePath, "baseline-initial", "pending", true)
			if err := f.run(t.Context(), baseline); err != nil {
				t.Fatal(err)
			}
			baseWant := tc.want
			if tc.want == "refused" || tc.want == "confirm_first_cycle" {
				baseWant = "complete"
			}
			f.capture(t, basePath, "baseline", baseWant, true)
			owner, path := f.create(t, true)
			f.capture(t, path, "candidate-initial", "pending", true)
			if err := f.run(t.Context(), owner); err != nil {
				t.Fatal(err)
			}
			f.capture(t, path, "candidate", tc.want, true)
			rawModelCommand(t, "model/run.py", "compare", "--baseline", filepath.Join(f.dir, "baseline.db"), "--candidate", filepath.Join(f.dir, "candidate.db"), "--configuration", filepath.Join(f.dir, "derived-configuration.json"), "--fixture", filepath.Join(f.dir, "fixture.json"), "--manifest", filepath.Join(f.dir, "deployment-manifest.json"), "--output", filepath.Join(f.dir, "comparison.json"))
		})
	}
}

var errRawModelCut = errors.New("captured raw model cut")

func rawModelPause(t *testing.T, f rawModelFixture, owner *runtime.Store, path string, match func(boundary.Event, *runtime.ExecutionSnapshot) bool) {
	t.Helper()
	observed := boundary.WithObserver(t.Context(), func(ctx context.Context, event boundary.Event) error {
		// These are the only event shapes the declared capture predicates can
		// match. Preserve all predicates and inspect the complete Store there.
		// New cut kinds must extend this filter together with their predicate.
		switch event.Boundary {
		case "execution.committed":
			if event.Intent == "" && event.Phase != "terminal" {
				return nil
			}
		case "execution.staged":
			if event.Subject != weatherSubject {
				return nil
			}
		case "invocation.admitted", "invocation.result-recorded", "adapter.result-recorded", "adapter.settled", "intent.authorized":
		default:
			return nil
		}
		cut, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
		if err != nil {
			return err
		}
		if match(event, cut) {
			return errRawModelCut
		}
		return nil
	})
	if err := f.run(observed, owner); !errors.Is(err, errRawModelCut) {
		t.Fatalf("missing requested cut: %v", err)
	}
}

func TestDelegatedRawCuts(t *testing.T) {
	for _, name := range []string{"proposal", "invocation", "request", "result", "settled", "second_forecast", "helper_terminal", "published", "assessment", "decision", "decision_authorized", "decision_admitted", "refused_unpublished", "refused_unselected"} {
		t.Run(name, func(t *testing.T) {
			options := defaultDeploymentOptions(delegatedWeather)
			if strings.HasPrefix(name, "refused_") {
				options.HelperAdmission = 0
			}
			f := newRawModelFixture(t, 2, options, []int{1, 61}, false)
			owner, path := f.create(t, true)
			rawModelPause(t, f, owner, path, func(event boundary.Event, cut *runtime.ExecutionSnapshot) bool {
				op, _ := cut.Operation(event.Intent)
				switch name {
				case "proposal":
					return event.Boundary == "execution.committed" && op != nil && op.Obligation == proposalPort
				case "invocation":
					return event.Boundary == "execution.committed" && op != nil && op.Kind == "invocation"
				case "request", "refused_unpublished":
					return event.Boundary == "invocation.admitted"
				case "result":
					return event.Boundary == "adapter.result-recorded"
				case "settled":
					return event.Boundary == "adapter.settled"
				case "second_forecast":
					return event.Boundary == "execution.committed" && op != nil && op.Kind == "adapter" && len(cut.Effects) == 2
				case "helper_terminal":
					return event.Boundary == "execution.committed" && event.Phase == "terminal" && event.Subject != outingSubject && event.Subject != weatherSubject
				case "published", "refused_unselected":
					return event.Boundary == "invocation.result-recorded"
				case "assessment":
					return event.Boundary == "execution.committed" && op != nil && op.Obligation == assessmentPort
				case "decision":
					return event.Boundary == "execution.committed" && op != nil && op.Obligation == decisionPort
				case "decision_authorized":
					return event.Boundary == "intent.authorized" && op != nil && op.Obligation == decisionPort
				case "decision_admitted":
					if event.Boundary != "execution.staged" || event.Subject != weatherSubject {
						return false
					}
					weather, _ := cut.Run(weatherRun)
					return weather.Wait != nil && weather.Wait.Obligation == decisionPort && len(weather.Wait.MatchingObservationIDs) == 1
				}
				return false
			})
			want := "pending"
			if strings.HasPrefix(name, "refused_") {
				want = "refused"
			}
			f.capture(t, path, "cut", want, true)
			if err := f.run(t.Context(), owner); err != nil {
				t.Fatal(err)
			}
			want = "complete"
			if options.HelperAdmission == 0 {
				want = "refused"
			}
			f.capture(t, path, "resumed", want, true)
		})
	}
}

func rawModelFinishLocal(t *testing.T, owner *runtime.Store, run runtime.RunID) {
	t.Helper()
	for range 100 {
		state, err := owner.Snapshot(t.Context(), run)
		if err != nil {
			t.Fatal(err)
		}
		if state.Projection.Status != "open" {
			return
		}
		if _, err = owner.Commit(t.Context(), state.Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("local terminal bound exceeded")
}

func TestDelegatedRawPublicDecisionControls(t *testing.T) {
	for _, late := range []bool{true, false} {
		name := "allowed_before_End_admitted_after_End"
		if late {
			name = "late_first_authorization"
		}
		t.Run(name, func(t *testing.T) {
			f := newRawModelFixture(t, 0, defaultDeploymentOptions(delegatedWeather), nil, false)
			owner, path := f.create(t, true)
			var original string
			rawModelPause(t, f, owner, path, func(event boundary.Event, cut *runtime.ExecutionSnapshot) bool {
				op, _ := cut.Operation(event.Intent)
				if event.Boundary == "execution.committed" && op != nil && op.Obligation == decisionPort {
					original = op.ID
					return true
				}
				return false
			})
			f.capture(t, path, "decision-commit", "pending", true)
			if !late {
				auth, err := owner.AuthorizeIntent(t.Context(), original)
				if err != nil || !auth.Allowed {
					t.Fatalf("pre-End authorization: %+v %v", auth, err)
				}
				f.capture(t, path, "allowed-open", "pending", true)
			}
			rawModelFinishLocal(t, owner, outingRun)
			auth, err := owner.AuthorizeIntent(t.Context(), original)
			if err != nil || auth.Allowed == late || late && auth.Reason != "source_closed" {
				t.Fatalf("retained authorization: %+v %v", auth, err)
			}
			transport := "raw-model-original-decision"
			admission, err := owner.DeliverMessage(t.Context(), original, transport)
			if err != nil || admission.Accepted == late {
				t.Fatalf("post-End original admission: %+v %v", admission, err)
			}
			if err := owner.AcknowledgeMessage(t.Context(), original, transport); err != nil {
				t.Fatal(err)
			}
			again, err := owner.AuthorizeIntent(t.Context(), original)
			if err != nil || again != auth {
				t.Fatalf("first disposition changed: %+v %v", again, err)
			}
			want := "pending"
			if late {
				want = "refused"
			}
			f.capture(t, path, "after-End-admission", want, false)
			if err := f.run(t.Context(), owner); err != nil {
				t.Fatal(err)
			}
			if !late {
				want = "complete"
			}
			f.capture(t, path, "final", want, false)
		})
	}
}

func TestDelegatedRawFrozenReplyControl(t *testing.T) {
	f := newRawModelFixture(t, 0, defaultDeploymentOptions(delegatedWeather), nil, false)
	owner, path := f.create(t, true)
	var reply runtime.Intent
	var call string
	rawModelPause(t, f, owner, path, func(event boundary.Event, cut *runtime.ExecutionSnapshot) bool {
		op, _ := cut.Operation(event.Intent)
		if event.Boundary == "execution.committed" && op != nil && op.Kind == "message" && op.Obligation == "assess" {
			reply = *op
			call = cut.Invocations[0].OperationID
			return true
		}
		return false
	})
	rawModelFinishLocal(t, owner, reply.Run)
	prepared, err := owner.PrepareInvocation(t.Context(), call)
	if err != nil || prepared.Ready || prepared.Pending != "reply_admission" {
		t.Fatalf("unadmitted frozen reply: %+v %v", prepared, err)
	}
	settled, err := owner.SettleInvocation(t.Context(), call)
	if err != nil || settled.Accepted {
		t.Fatalf("premature publication: %+v %v", settled, err)
	}
	f.capture(t, path, "frozen-unadmitted", "pending", true)
	auth, err := owner.AuthorizeIntent(t.Context(), reply.ID)
	if err != nil || !auth.Allowed {
		t.Fatalf("bound post-End authorization: %+v %v", auth, err)
	}
	admission, err := owner.DeliverMessage(t.Context(), reply.ID, "raw-model-bound-reply")
	if err != nil || !admission.Accepted || admission.Reason != "recorded_reply" || admission.ObservationID != "" {
		t.Fatalf("bound reply admission: %+v %v", admission, err)
	}
	f.capture(t, path, "recorded-unacknowledged", "pending", true)
	if err := owner.AcknowledgeMessage(t.Context(), reply.ID, "raw-model-bound-reply"); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.SettleInvocation(t.Context(), call); err != nil {
		t.Fatal(err)
	}
	f.capture(t, path, "published-unselected", "pending", true)
	if err := f.run(t.Context(), owner); err != nil {
		t.Fatal(err)
	}
	f.capture(t, path, "final", "complete", true)
}
