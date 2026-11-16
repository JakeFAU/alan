package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

func fixtureConfiguration() configuration {
	return configuration{
		Version: 1, ReferenceTime: "2026-10-10T16:30:00Z", Date: "tomorrow",
		Timezone: "America/New_York", Units: "celsius", Endpoint: openmeteo.DefaultEndpoint,
		MaxTravelMinutes: 30,
		Availability:     []timeWindow{{Start: "08:00", End: "15:00"}},
		Slots:            []string{"11:00"},
		Parks: []park{
			{ID: "hemlock", Name: "Hemlock Park", Latitude: "40.7", Longitude: "-74", TravelMinutes: 12,
				Opening: []timeWindow{{Start: "09:00", End: "14:00"}}},
			{ID: "cedar", Name: "Cedar Park", Latitude: "40.8", Longitude: "-74", TravelMinutes: 8,
				Opening: []timeWindow{{Start: "09:00", End: "14:00"}}},
		},
	}
}

func TestPlannerCalendarBounds(t *testing.T) {
	c := fixtureConfiguration()
	plan, err := c.normalize()
	if err != nil || plan.Date != "2026-10-11" {
		t.Fatalf("normalize tomorrow: %+v %v", plan, err)
	}
	if len(plan.Candidates) != 2 || plan.Candidates[0].ParkID != "cedar" ||
		plan.Candidates[0].Start != "2026-10-11T15:00:00Z" || !plan.Candidates[1].Eligible {
		t.Fatalf("candidate ranking or timezone: %+v", plan.Candidates)
	}
	c.Availability[0].Start = "10:50"
	plan, err = c.normalize()
	if err != nil || !plan.Candidates[0].Eligible || plan.Candidates[1].Eligible {
		t.Fatalf("outbound travel: %+v %v", plan, err)
	}

	for _, tc := range []struct {
		name, zone, date, input string
		ok                      bool
	}{
		{"fold", "America/New_York", "2026-11-01", "01:00", false},
		{"fold first", "America/New_York", "2026-11-01", "2026-11-01T01:00:00-04:00", true},
		{"fold second", "America/New_York", "2026-11-01", "2026-11-01T01:00:00-05:00", true},
		{"bogus offset", "America/New_York", "2026-11-01", "2026-11-01T01:00:00-03:00", false},
		{"malformed offset", "America/New_York", "2026-10-11", "2026-10-11T10:00:00-03:60", false},
		{"gap", "America/New_York", "2026-03-08", "02:00", false},
		{"fractional aligned", "Asia/Kathmandu", "2026-10-11", "11:45", true},
		{"fractional unaligned", "Asia/Kathmandu", "2026-10-11", "11:00", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fixtureConfiguration()
			c.Date, c.Timezone, c.Slots = tc.date, tc.zone, []string{tc.input}
			_, err := c.normalize()
			if (err == nil) != tc.ok {
				t.Fatalf("valid=%v, error=%v", tc.ok, err)
			}
		})
	}

	c = fixtureConfiguration()
	for i := range 2 {
		p := c.Parks[i]
		p.ID += "-other"
		c.Parks = append(c.Parks, p)
	}
	c.Slots = []string{"11:00", "09:00", "12:00", "10:00"}
	plan, err = c.normalize()
	if err != nil || len(plan.Candidates) != 16 {
		t.Fatalf("maximum accepted plan: %+v %v", plan, err)
	}
	const firstSlot = "2026-10-11T13:00:00Z"
	first, second := plan.Candidates[0], plan.Candidates[1]
	if first.ParkID != "cedar" || second.ParkID != "cedar-other" || first.Start != firstSlot || second.Start != firstSlot {
		t.Fatalf("travel, start and park ID ordering: %+v %+v", first, second)
	}
	c.Parks = append(c.Parks, park{ID: "extra"})
	if _, err := c.normalize(); err == nil {
		t.Fatal("accepted five parks")
	}
	c.Parks = c.Parks[:4]
	c.Slots = append(c.Slots, "13:00")
	if _, err := c.normalize(); err == nil {
		t.Fatal("accepted five slots")
	}
}

func fixtureServer(t *testing.T, cedarCode, hemlockCode string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	calls := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		code := cedarCode
		if q.Get("latitude") == "40.7" {
			code = hemlockCode
		}
		startCode, endCode, customEnd := strings.Cut(code, "/")
		if !customEnd {
			endCode = "2"
		}
		if r.Method != http.MethodGet || q.Get("temperature_unit") != "celsius" ||
			q.Get("start_hour") != "2026-10-11T15:00" || q.Get("end_hour") != "2026-10-11T16:00" {
			t.Errorf("wrong retained forecast query: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"latitude":%s,"longitude":-74,"utc_offset_seconds":0,"timezone":"GMT",
			"hourly_units":{"time":"iso8601","temperature_2m":"°C","weather_code":"wmo code"},
			"hourly":{"time":["2026-10-11T15:00","2026-10-11T16:00"],
			"temperature_2m":[17.2,18.3],"weather_code":[%s,%s]}}`, q.Get("latitude"), startCode, endCode)
	}))
	t.Cleanup(server.Close)
	return server, calls
}

func executeFixture(t *testing.T, c configuration, variant string) (*runtime.ExecutionSnapshot, string) {
	t.Helper()
	p, err := c.provision(variant)
	if err != nil {
		t.Fatal(err)
	}
	ctx, limits := t.Context(), runtime.DefaultLimits()
	path := filepath.Join(t.TempDir(), "outing.db")
	store, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = runtime.Open(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := openmeteo.NewHourly(c.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RunProgress(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{"open-meteo-hourly": adapter}, MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1,
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
	retained, retainedVariant, err := retainedConfiguration(cut)
	if err != nil || retainedVariant != variant || !reflect.DeepEqual(retained, c) {
		t.Fatalf("retained configuration: %+v %q %v", retained, retainedVariant, err)
	}
	if err := verifyPlannerEvidence(cut, retained, retainedVariant); err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range replay.Runs {
		inspected, ok := cut.Run(r.ID)
		if !ok || !reflect.DeepEqual(inspected.Projection, r.Projection) {
			t.Fatalf("replay differs for %s", r.ID.Subject)
		}
	}
	return cut, path
}

func requireDecision(t *testing.T, cut *runtime.ExecutionSnapshot, outcome, park string) {
	t.Helper()
	var decision *runtime.Intent
	for _, op := range cut.Operations {
		if op.Kind == "message" && op.Obligation == "outing-decision" {
			if decision != nil {
				t.Fatal("multiple decisions")
			}
			decision = &op
		}
	}
	if decision == nil || decision.Payload["outcome"].Text != outcome {
		t.Fatalf("decision %#v, want %s", decision, outcome)
	}
	selection, err := decision.Payload["selection"].Records()
	if err != nil {
		t.Fatal(err)
	}
	if park == "" && len(selection) != 0 || park != "" && (len(selection) != 1 || selection[0]["park_id"].Text != park) {
		t.Fatalf("unexpected selection: %v", selection)
	}
	if park != "" {
		effect, ok := cut.Effect(selection[0]["forecast_operation"].Text)
		if !ok || effect.Settlement == nil || effect.AdapterResult.Candidate.Payload[openmeteo.TemperatureUnit].Text != "°C" {
			t.Fatalf("selected row has no original successful Celsius evidence: %+v", effect)
		}
	}
	consumed := 0
	for _, o := range cut.Observations {
		if o.SourceOperationID == decision.ID && o.SelectedAt != nil && o.SelectedAt.Run.Subject == "weather" {
			consumed++
		}
	}
	if consumed != 1 {
		t.Fatalf("decision selected %d times by Weather", consumed)
	}
	want := "succeeded"
	if outcome == "unavailable" {
		want = "failed"
	}
	for _, run := range cut.Runs {
		if run.Projection.Status != want {
			t.Fatalf("%s status %s at PC %d, want %s", run.ID.Subject, run.Projection.Status, run.Projection.PC, want)
		}
	}
}

func TestPlannerRecommendationAndNone(t *testing.T) {
	for _, tc := range []struct {
		name, cedar, hemlock, outcome, park string
		empty                               bool
		calls                               int32
	}{
		{"recommendation", "1", "0", "recommended", "cedar", false, 2},
		{"exhaustive none", "61", "45", "no_match", "", false, 2},
		{"disallowed ending sample", "0/95", "0", "recommended", "hemlock", false, 2},
		{"empty list still completes protocol", "0", "0", "no_match", "", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := fixtureServer(t, tc.cedar, tc.hemlock)
			c := fixtureConfiguration()
			c.Endpoint = server.URL + "/v1/forecast"
			if tc.empty {
				c.Availability = nil
			}
			cut, _ := executeFixture(t, c, "assess-first")
			requireDecision(t, cut, tc.outcome, tc.park)
			if calls.Load() != tc.calls {
				t.Fatalf("native contact count %d, want %d", calls.Load(), tc.calls)
			}
		})
	}
}

func TestPlannerPartialEvidence(t *testing.T) {
	for _, tc := range []struct{ later, outcome, park string }{
		{"2", "recommended", "hemlock"},
		{"61", "unavailable", ""},
		{"null", "unavailable", ""},
	} {
		t.Run(tc.later, func(t *testing.T) {
			server, calls := fixtureServer(t, "999", tc.later)
			c := fixtureConfiguration()
			c.Endpoint = server.URL + "/v1/forecast"
			cut, _ := executeFixture(t, c, "assess-first")
			requireDecision(t, cut, tc.outcome, tc.park)
			if calls.Load() != 2 {
				t.Fatalf("native contact count %d, want 2", calls.Load())
			}
		})
	}
}

func TestPlannerConfirmationFirstWaits(t *testing.T) {
	c := fixtureConfiguration()
	bad, path := executeFixture(t, c, "confirm-first")
	for _, r := range bad.Runs {
		want := "outing-assessment"
		if r.ID.Subject == "weather" {
			want = "outing-decision"
		}
		if r.Wait == nil || r.Wait.Obligation != want || len(r.Wait.MatchingObservationIDs) != 0 || r.Projection.Status != "open" {
			t.Fatalf("%s wait: %+v", r.ID.Subject, r.Wait)
		}
	}
	if len(bad.Operations) != 1 || len(bad.Effects) != 0 {
		t.Fatalf("unexpected operation: %+v", bad.Operations)
	}
	p, err := c.provision("assess-first")
	if err != nil {
		t.Fatal(err)
	}
	if bad.Deployment.Runs[0].ProgramID != p.Runs[0].ProgramID ||
		bad.Deployment.Runs[1].ProgramID == p.Runs[1].ProgramID {
		t.Fatal("expected changed Weather artifact and stable Outing")
	}
	if err := runCLI(t.Context(), []string{"resume", "--db", path, "--json"}, new(strings.Builder)); err != nil {
		t.Fatal(err)
	}
}
