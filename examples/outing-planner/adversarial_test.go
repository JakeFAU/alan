package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestPlannerConfigurationDecoding(t *testing.T) {
	c := fixtureConfiguration()
	c.ReferenceTime = ""
	raw, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "outing.json")
	now := time.Date(2026, 10, 10, 1, 2, 3, 42, time.UTC)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	frozen, err := readConfiguration(path, now)
	if err != nil || frozen.ReferenceTime != now.Format(time.RFC3339Nano) {
		t.Fatalf("reference not frozen: %+v %v", frozen, err)
	}
	data := string(raw)
	for _, invalid := range []string{
		strings.Replace(data, `"version":1`, `"version":2,"version":1`, 1),
		strings.Replace(data, `"id":"hemlock"`, `"id":"fake","id":"hemlock"`, 1),
		strings.Replace(data, `"version":1`, `"version":1,"unknown":0`, 1),
		strings.Replace(data, `"latitude":"40.7"`, `"latitude":40.7`, 1),
		strings.Replace(data, `"id":"hemlock"`, `"id":"cedar"`, 1),
		strings.Replace(data, `"slots":["11:00"]`, `"slots":["11:00","2026-10-10T11:00:00-04:00"]`, 1),
		strings.Replace(data, `"travel_minutes":12`, `"travel_minutes":9223372036854775807`, 1),
		strings.Replace(data, `"allowed_weather_codes":null`, `"allowed_weather_codes":[4]`, 1),
		strings.Replace(data, `"allowed_weather_codes":null`, `"allowed_weather_codes":[1,1]`, 1),
		data + "{}",
		data + strings.Repeat(" ", maxConfiguration),
	} {
		if err := os.WriteFile(path, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readConfiguration(path, now); err == nil {
			t.Fatalf("accepted invalid configuration prefix %.140q", invalid)
		}
	}
}

func TestPlannerRejectsMismatchedBatch(t *testing.T) {
	c := fixtureConfiguration()
	plan, err := c.normalize()
	if err != nil {
		t.Fatal(err)
	}
	makeRows := func() []map[string]schema.Value {
		rows := make([]map[string]schema.Value, 0, len(plan.Candidates))
		for i, candidate := range plan.Candidates {
			row := candidateRecord(candidate)
			row["forecast_operation"] = schema.Text(string(rune('a' + i)))
			row["weather_status"] = schema.Text(suitable)
			rows = append(rows, row)
		}
		return rows
	}
	for _, name := range []string{"wrong proposal", "omission", "changed coordinates", "changed time", "reordered", "duplicate forecast"} {
		t.Run(name, func(t *testing.T) {
			rows := makeRows()
			d := weatherDefinition(assessFirst)
			claimedID := author.Var(proposalID)
			switch name {
			case "wrong proposal":
				claimedID = author.String("another")
			case "omission":
				rows = rows[:1]
			case "changed coordinates":
				rows[0]["latitude"] = schema.Text("40.9")
			case "changed time":
				rows[0]["end_hour"] = schema.Text("2026-10-11T17:00:00Z")
			case "reordered":
				rows[0], rows[1] = rows[1], rows[0]
			case "duplicate forecast":
				rows[1]["forecast_operation"] = rows[0]["forecast_operation"]
			}
			d.Outputs = nil
			d.Steps = []author.Step{
				author.Receive(proposalPort, nil),
				author.Send(assessmentPort, map[string]author.Expr{
					proposalID: claimedID, assessments: author.Value(schema.List(rows...)),
				}),
				author.Receive(decisionPort, nil),
				author.Success(),
			}
			cut := executeAltered(t, c, assessFirst, weatherSubject, d)
			outing, _ := cut.Run(outingRun)
			if outing.Projection.Status != "failed" {
				t.Fatalf("Outing accepted the malformed batch: %+v", outing.Projection)
			}
			for _, op := range cut.Operations {
				if op.Obligation == decisionPort {
					t.Fatal("a malformed batch reached a committed decision")
				}
			}
		})
	}
	rows := makeRows()
	rows[1] = rows[0]
	if err := assessmentList.Validate(schema.List(rows...)); err == nil {
		t.Fatal("typed batch schema accepted a duplicate candidate ID")
	}
}

func TestPlannerWeatherRejectsWrongDecision(t *testing.T) {
	c := fixtureConfiguration()
	c.Slots = nil
	d := outingDefinition()
	// Preserve the genuine empty proposal and assessment reception but claim a
	// recommendation with no selection on the final send.
	payload := recordExpressions(decisionSchema, "")
	payload[outcome] = author.String(recommended)
	d.Steps[len(d.Steps)-2] = author.Send(decisionPort, payload)
	cut := executeAltered(t, c, assessFirst, outingSubject, d)
	weather, _ := cut.Run(weatherRun)
	if weather.Projection.Status != "failed" {
		t.Fatalf("Weather accepted forged decision data: %+v", weather.Projection)
	}
}

func TestPlannerConfirmationFirstRejectsMismatchesBeforeForecast(t *testing.T) {
	for _, name := range []string{"wrong proposal", "wrong candidates"} {
		t.Run(name, func(t *testing.T) {
			server, calls := fixtureServer(t, "0", "0")
			c := fixtureConfiguration()
			c.Endpoint = server.URL + "/v1/forecast"
			d := outingDefinition()
			// An alternate peer sends a typed but mismatched decision before
			// waiting for the assessment. The production owner rejects this peer.
			payload := map[string]author.Expr{
				proposalID: author.Var(proposalID), candidates: author.Var(candidates),
				outcome: author.String(noMatch), selection: author.Value(schema.List()),
			}
			if name == "wrong proposal" {
				payload[proposalID] = author.String("another")
			} else {
				payload[candidates] = author.Value(schema.List())
			}
			d.Outputs = nil
			d.Steps = append(d.Steps[:2],
				author.Send(decisionPort, payload),
				author.Receive(assessmentPort, nil),
				author.Success(),
			)
			cut := executeAltered(t, c, confirmFirst, outingSubject, d)
			weather, _ := cut.Run(weatherRun)
			if weather.Projection.Status != "failed" {
				t.Fatalf("Weather accepted an incorrect confirmation: %+v", weather.Projection)
			}
			var selectedDecision bool
			for _, o := range cut.Observations {
				if o.Obligation == decisionPort && o.Run == weatherRun && o.SelectedAt != nil {
					selectedDecision = true
				}
			}
			if !selectedDecision {
				t.Fatal("Weather failed before selecting the injected decision")
			}
			if calls.Load() != 0 || len(cut.Effects) != 0 {
				t.Fatalf("mismatched confirmation caused %d committed forecast operations and %d provider contacts",
					len(cut.Effects), calls.Load())
			}
		})
	}
}

func executeAltered(t *testing.T, c configuration, variant, subject string, def author.Definition) *runtime.ExecutionSnapshot {
	t.Helper()
	p, err := c.provision(variant)
	if err != nil {
		t.Fatal(err)
	}
	a, err := author.Compile(def)
	if err != nil {
		t.Fatal(err)
	}
	for i := range p.Runs {
		if p.Runs[i].ID.Subject == subject {
			p.Programs[i].ID = a.ID()
			p.Programs[i].Bytes = a.Bytes()
			p.Runs[i].ProgramID = a.ID()
			break
		}
	}
	path := filepath.Join(t.TempDir(), "altered.db")
	ctx, limits := t.Context(), runtime.DefaultLimits()
	store, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := openmeteo.NewHourly(c.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RunProgress(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{capability: adapter},
		MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1,
	})
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	cut, err := runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := retainedConfiguration(cut); err == nil {
		t.Fatal("accepted an altered application artifact")
	}
	return cut
}

func TestPlannerRefusesChangedDeployment(t *testing.T) {
	server, contacts := fixtureServer(t, "1", "2")
	for _, name := range []string{"grant", "root allowance", "input"} {
		t.Run(name, func(t *testing.T) {
			c := fixtureConfiguration()
			c.Endpoint = server.URL + fixturePath
			p, err := c.provision(assessFirst)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "grant":
				p.Permissions = append(p.Permissions, runtime.Permission{
					Kind: "input", Source: "local", Target: weatherSubject, Operation: "input", Obligation: "outside",
				})
			case "root allowance":
				p.Composition.Roots[1].Limits.MaxEffects++
			case "input":
				p.Runs[1].Projection.Data["allowed_weather_codes"] = schema.List()
			}
			path := filepath.Join(t.TempDir(), "altered.db")
			store, err := runtime.Create(t.Context(), path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			if err := runCLI(t.Context(), []string{"resume", "--db", path}, new(strings.Builder)); err == nil {
				t.Fatal("owner accepted a changed deployment")
			}
		})
	}
	if contacts.Load() != 0 {
		t.Fatalf("refused grant, scope or input reached transport %d times", contacts.Load())
	}
}

func TestPlannerUnknownForecastStaysOriginal(t *testing.T) {
	var contacts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	first, path := executeFixture(t, c, assessFirst)
	if len(first.Effects) != 1 {
		t.Fatalf("expected one original forecast, got %d", len(first.Effects))
	}
	f := first.Effects[0]
	if f.DispatchAdmission == nil || f.AdapterResult != nil || f.Settlement != nil {
		t.Fatalf("unexpected forecast evidence: %+v", f)
	}
	weather, _ := first.Run(weatherRun)
	if weather.Wait == nil || weather.Wait.OperationID != f.OperationID {
		t.Fatalf("Weather did not await its lost forecast: %+v", weather.Wait)
	}
	resumed, progress, err := operatePlanner(t.Context(), path, first.Namespace, c, assessFirst)
	if err != nil || !progress.Quiescent {
		t.Fatalf("resume of original wait: %+v %v", progress, err)
	}
	if contacts.Load() != 1 || len(resumed.Effects) != 1 || !reflect.DeepEqual(resumed.Effects[0], f) {
		t.Fatalf("reopen changed original evidence or contacted provider again: %d", contacts.Load())
	}
}

func TestPlannerReportsCommittedForecastRequest(t *testing.T) {
	ctx := t.Context()
	c := fixtureConfiguration()
	c.Endpoint = "http://127.0.0.1:1/v1/forecast"
	path := filepath.Join(t.TempDir(), "outing.db")
	if err := createPlanner(ctx, path, c, assessFirst); err != nil {
		t.Fatal(err)
	}
	initial, _, _, err := inspectPlanner(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("stop after forecast request commitment")
	var committed runtime.Intent
	observed := boundary.WithObserver(ctx, func(ctx context.Context, event boundary.Event) error {
		if event.Boundary != "execution.committed" || event.Subject != weatherSubject {
			return nil
		}
		cut, _, _, err := inspectPlanner(ctx, path)
		if err != nil {
			return err
		}
		if len(cut.Effects) == 0 {
			return nil
		}
		if len(cut.Effects) != 1 {
			t.Fatalf("expected one committed forecast, got %d", len(cut.Effects))
		}
		forecast := cut.Effects[0]
		if forecast.AdapterAttempt != nil || forecast.DispatchAdmission != nil || forecast.AdapterResult != nil {
			t.Fatal("forecast reached adapter preparation or dispatch before the cut")
		}
		op, ok := cut.Operation(forecast.OperationID)
		if !ok || op.Sequence != event.Sequence {
			t.Fatal("forecast lacks its original operation at the committed sequence")
		}
		committed = *op
		return stop
	})
	if _, _, err := operatePlanner(observed, path, initial.Namespace, c, assessFirst); !errors.Is(err, stop) {
		t.Fatalf("did not stop at the original request commit: %v", err)
	}
	output := plannerOutput(t, "inspect", "--db", path, "--json")
	plannerOutcome(t, output, pendingOutcome)
	var report struct {
		Operations []runtime.Intent `json:"operations"`
	}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatal(err)
	}
	for _, op := range report.Operations {
		if op.ID != committed.ID {
			continue
		}
		wantPayload := map[string]schema.Value{
			"latitude": schema.Text("40.8"), "longitude": schema.Text("-74"),
			"start_hour": schema.Text("2026-10-11T15:00:00Z"), "end_hour": schema.Text("2026-10-11T16:00:00Z"),
			"units": schema.Text("celsius"),
		}
		if !reflect.DeepEqual(op, committed) || op.Run != weatherRun || op.Ordinal != 0 ||
			op.Kind != "adapter" || op.Target != capability || op.Obligation != effectName ||
			!reflect.DeepEqual(op.Payload, wantPayload) {
			t.Fatalf("report changed the committed forecast request: %+v", op)
		}
		return
	}
	t.Fatalf("report omits committed forecast operation %s before adapter preparation", committed.ID)
}

func TestPlannerReopensAtFinalDecisionCommit(t *testing.T) {
	c := fixtureConfiguration()
	c.Slots = nil
	path := filepath.Join(t.TempDir(), "outing.db")
	ctx := t.Context()
	if err := createPlanner(ctx, path, c, assessFirst); err != nil {
		t.Fatal(err)
	}
	cut, _, _, err := inspectPlanner(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	stopAfterDecision := errors.New("test stopped after decision commitment")
	var decisionID string
	observed := boundary.WithObserver(ctx, func(ctx context.Context, event boundary.Event) error {
		if event.Boundary != "execution.committed" || event.Subject != outingSubject {
			return nil
		}
		current, _, _, err := inspectPlanner(ctx, path)
		if err != nil {
			return err
		}
		for _, op := range current.Operations {
			if op.Obligation == decisionPort {
				decisionID = op.ID
				return stopAfterDecision
			}
		}
		return nil
	})
	if _, _, err := operatePlanner(observed, path, cut.Namespace, c, assessFirst); !errors.Is(err, stopAfterDecision) {
		t.Fatalf("runner did not stop at committed decision: %v", err)
	}
	pending, _, _, err := inspectPlanner(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	outing, _ := pending.Run(outingRun)
	if outing.Projection.Status != "open" || decisionID == "" {
		t.Fatalf("incorrect pre-delivery cut: %v, %s", outing.Projection.Status, decisionID)
	}
	for _, o := range pending.Observations {
		if o.SourceOperationID == decisionID {
			t.Fatal("decision was delivered before the cut")
		}
	}
	committed := plannerOutput(t, "status", "--db", path, "--json")
	plannerOutcome(t, committed, pendingOutcome)
	resumed, _, err := operatePlanner(ctx, path, cut.Namespace, c, assessFirst)
	if err != nil {
		t.Fatal(err)
	}
	requireDecision(t, resumed, noMatch, "")
	if _, found := resumed.Operation(decisionID); !found {
		t.Fatal("reopen did not preserve the original decision operation")
	}
}

func TestPlannerInitialCLI(t *testing.T) {
	server, contacts := fixtureServer(t, "1", "2")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	raw, _ := c.serialized()
	dir := t.TempDir()
	configPath, path := filepath.Join(dir, "outing.json"), filepath.Join(dir, "outing.db")
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := runCLI(t.Context(), []string{"start", "--db", path, "--config", configPath, "--json"}, &out); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != 0 || !json.Valid([]byte(out.String())) {
		t.Fatalf("start contacted provider or returned malformed JSON: %d", contacts.Load())
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"status", "run", "resume", "status"} {
		out.Reset()
		if err := runCLI(t.Context(), []string{action, "--db", path, "--json"}, &out); err != nil {
			t.Fatal(err)
		}
		if !json.Valid([]byte(out.String())) {
			t.Fatalf("%s returned malformed JSON", action)
		}
	}
	cut, _, _, err := inspectPlanner(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	requireDecision(t, cut, recommended, "cedar")
	if contacts.Load() != 2 {
		t.Fatalf("CLI contacted fixture %d times", contacts.Load())
	}
}

type corruptForecast struct{ *openmeteo.HourlyAdapter }

func (a corruptForecast) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	result := a.HourlyAdapter.Execute(ctx, p)
	if result.Disposition == "validated" {
		result.Payload[openmeteo.StartWeatherCode] = schema.Int(0)
	}
	return result
}

func TestPlannerRefusesNativeMismatch(t *testing.T) {
	server, contacts := fixtureServer(t, "61", "2")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + "/v1/forecast"
	path := filepath.Join(t.TempDir(), "outing.db")
	ctx, limits := t.Context(), runtime.DefaultLimits()
	if err := createPlanner(ctx, path, c, assessFirst); err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := openmeteo.NewHourly(c.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RunProgress(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{capability: corruptForecast{adapter}},
		MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1,
	})
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"status", "resume"} {
		err := runCLI(ctx, []string{action, "--db", path, "--json"}, new(strings.Builder))
		if err == nil || !strings.Contains(err.Error(), "retained provider body") {
			t.Fatalf("%s accepted typed fields that disagreed with the native body: %v", action, err)
		}
	}
	if contacts.Load() != 2 {
		t.Fatalf("refused CLI operation contacted the fixture, %d contacts", contacts.Load())
	}
}
