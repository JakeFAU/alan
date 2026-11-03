package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// Read the CLI response as an independent client. The runtime inspector below
// provides the operation, authorization, response, delivery and selection chain.
type weatherReadView struct {
	Number                                                    int `json:"read"`
	State, Report, Source, Temperature, Humidity, Disposition string
	Operation                                                 string `json:"operation_id"`
	Previous                                                  string `json:"previous_operation_id"`
	ApprovalReason                                            string `json:"approval_reason"`
	ObservedAt                                                string `json:"observed_at"`
	TemperatureUnit                                           string `json:"temperature_unit"`
	HumidityUnit                                              string `json:"humidity_unit"`
	Origin                                                    string `json:"result_origin"`
}
type weatherView struct {
	Namespace, Latitude, Longitude, Units, Endpoint string
	Allowance                                       int
	Spent                                           int             `json:"opportunities_spent"`
	Cost                                            json.RawMessage `json:"provider_cost_usd"`
	Reads                                           []weatherReadView
	History                                         []runtime.ExecutionStep
	Pending                                         []runtime.PendingObligation
	ReplayedCommits                                 *int `json:"replayed_commits"`
}

func decodeWeather(t *testing.T, s string) weatherView {
	t.Helper()
	var w weatherView
	if err := json.Unmarshal([]byte(s), &w); err != nil {
		t.Fatalf("weather command did not return its report: %v\n%s", err, s)
	}
	if w.Namespace == "" || w.Allowance < 0 || string(w.Cost) != "null" {
		t.Fatalf("missing durable namespace or finite allowance, or made-up cost: %s", s)
	}
	return w
}

func (w weatherView) read(t *testing.T, n int) weatherReadView {
	t.Helper()
	for _, r := range w.Reads {
		if r.Number == n {
			return r
		}
	}
	t.Fatalf("no visible read %d in %+v", n, w)
	return weatherReadView{}
}

func weatherSnapshot(t *testing.T, db string) *runtime.ExecutionSnapshot {
	t.Helper()
	s, err := runtime.InspectExecution(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func weatherEffect(t *testing.T, s *runtime.ExecutionSnapshot, subject string) *runtime.ExecutionEffect {
	t.Helper()
	for _, e := range s.Effects {
		if e.Source.Subject == subject {
			if e.Source.Generation != 1 || e.Capability != "open-meteo" || e.Name != "forecast" || e.Kind != "adapter" {
				t.Fatalf("unexpected weather operation: %+v", e)
			}
			return e
		}
	}
	t.Fatalf("no retained original weather effect for %s", subject)
	return nil
}

func selectedWeather(t *testing.T, s *runtime.ExecutionSnapshot, e *runtime.ExecutionEffect) *runtime.ReductionRef {
	t.Helper()
	if e.Settlement == nil {
		return nil
	}
	observation, ok := s.Observation(e.Settlement.ObservationID)
	if !ok || observation.Operation != e.OperationID || observation.SourceOperationID != e.OperationID || observation.Run != e.Source {
		t.Fatalf("settlement failed to identify the original weather observation: %+v", e)
	}
	if observation.SelectedAt != nil {
		step, ok := s.Reduction(*observation.SelectedAt)
		if !ok || step.SelectedObservationID != observation.ID || step.Run != e.Source {
			t.Fatalf("weather selection did not come from the original source run: %+v", observation)
		}
	}
	return observation.SelectedAt
}

func weatherDBBytes(t *testing.T, db string) [2][32]byte {
	t.Helper()
	var digests [2][32]byte
	for i, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(db + suffix)
		if err != nil && !(i == 1 && errors.Is(err, os.ErrNotExist)) {
			t.Fatal(err)
		}
		digests[i] = sha256.Sum256(b)
	}
	return digests
}

func compareWeatherContacts(t *testing.T, s *runtime.ExecutionSnapshot, p *weatherProvider, allowance, spent int) {
	t.Helper()
	if !s.Features.DispatchAdmissions {
		t.Fatal("the weather store cannot distinguish a paid reservation from local dispatch admission")
	}
	var charged, limit int64
	for _, r := range s.Roots {
		charged += r.Charges.Effects
		limit += r.Limits.MaxEffects
	}
	if charged != int64(spent) || limit != int64(allowance) {
		t.Fatalf("durable allowance: charged=%d limit=%d, want %d/%d", charged, limit, spent, allowance)
	}
	contacts := map[string]weatherContact{}
	for _, c := range p.contacts() {
		if _, repeated := contacts[c.operation]; repeated {
			t.Fatalf("unapproved retransmission of original operation %s", c.operation)
		}
		contacts[c.operation] = c
		e, ok := s.Effect(c.operation)
		if !ok || e.Authorization == nil || !e.Authorization.Allowed || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed ||
			e.DispatchAdmission == nil || e.DispatchAdmission.AfterFact < 1 || e.DispatchAdmission.OperationID != c.operation ||
			e.DispatchAdmission.PreparedDigest != e.AdapterAttempt.Prepared.Digest ||
			e.AdapterAttempt.AuthorizationID != e.Authorization.ID || e.Authorization.IntentID != c.operation || e.AdapterAttempt.Prepared.Request.ID != c.operation {
			t.Fatalf("provider accepted contact %d without its original grant/reservation: %s", c.index, c.operation)
		}
		prepared := e.AdapterAttempt.Prepared
		if prepared.Endpoint != p.endpoint() || prepared.Wire != p.server.URL+c.uri || c.clientID != fmt.Sprintf("alan-%x", sha256.Sum256([]byte(prepared.Request.Key))) {
			t.Fatalf("contact %d endpoint or namespace-bound key differs from original: %+v", c.index, c)
		}
		if e.AdapterResult != nil {
			result := e.AdapterResult.Candidate
			if !c.written || result.ID != c.operation || result.Key != prepared.Request.Key || result.HTTPStatus != c.status || result.Response != c.body {
				t.Fatalf("retained response differs from independent provider log for original %s: %+v", c.operation, result)
			}
		}
	}
	nAllowed := 0
	for _, e := range s.Effects {
		if e.AdapterAttempt != nil && e.AdapterAttempt.Allowed {
			nAllowed++
		}
		_, contacted := contacts[e.OperationID]
		actual := 0
		if contacted {
			actual = 1
		}
		if e.Dispatch.Maximum == nil || e.Dispatch.Minimum > actual || actual > *e.Dispatch.Maximum {
			t.Fatalf("independent physical contact=%d outside retained bounds %+v for %s", actual, e.Dispatch, e.OperationID)
		}
		if e.AdapterResult != nil && e.AdapterResult.Candidate.HTTPStatus > 0 && !contacted {
			t.Fatalf("fabricated HTTP result for %s", e.OperationID)
		}
		if e.Usage != nil {
			t.Fatalf("invented provider usage %+v", e.Usage)
		}
	}
	if nAllowed != spent {
		t.Fatalf("%d allowed original opportunities for %d debits", nAllowed, spent)
	}
}

func inspectAndReplayWeather(t *testing.T, bin, db string, p *weatherProvider, allowance, spent int) (*runtime.ExecutionSnapshot, weatherView) {
	t.Helper()
	digests, contacts := weatherDBBytes(t, db), p.contacts()
	s := weatherSnapshot(t, db)
	compareWeatherContacts(t, s, p, allowance, spent)
	view := decodeWeather(t, weatherCommand(t, bin, "inspect", "--db", db))
	replayed := decodeWeather(t, weatherCommand(t, bin, "replay", "--db", db))
	if view.Namespace != s.Namespace || view.Spent != spent || view.Allowance != allowance || view.Latitude != "26.93" ||
		view.Longitude != "-80.08" || view.Units != "fahrenheit" || view.Endpoint != p.endpoint() ||
		replayed.Namespace != view.Namespace || replayed.Spent != view.Spent || !reflect.DeepEqual(replayed.Reads, view.Reads) {
		t.Fatalf("public inspect/replay disagree with typed snapshot: %+v / %+v", view, replayed)
	}
	replay, err := runtime.Replay(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	history, err := s.History(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Namespace != s.Namespace || replay.HistoryCount != history.Total || replay.IntentCount != len(s.Operations) {
		t.Fatalf("replay did not reconstruct the original namespace and history: %+v", replay)
	}
	if len(view.History) != history.Total || replayed.ReplayedCommits == nil || *replayed.ReplayedCommits != history.Total ||
		len(view.Pending) != len(s.Pending) || len(view.Pending) > 0 && !reflect.DeepEqual(view.Pending, s.Pending) {
		t.Fatal("visible history, pending obligations or replay count differ from original execution")
	}
	if digests != weatherDBBytes(t, db) || !slices.Equal(contacts, p.contacts()) {
		t.Fatal("offline inspection or replay changed database/WAL bytes or contacted the provider")
	}
	return s, view
}

func authenticWeather(t *testing.T, s *runtime.ExecutionSnapshot, v weatherReadView, n int, contact weatherContact) {
	t.Helper()
	e := weatherEffect(t, s, fmt.Sprintf("weather-%d", n))
	if e.OperationID != contact.operation || e.OperationID != v.Operation || e.State != runtime.EffectSettled || e.ResultDisposition != "validated" || e.AdapterResult == nil || selectedWeather(t, s, e) == nil {
		t.Fatalf("report lacks its selected original successful weather observation: %+v / %+v", e, v)
	}
	wantTime, temperature, humidity := "2026-11-03T09:45:00Z", "75.4", "80"
	if contact.body == laterWeather {
		wantTime, temperature, humidity = "2026-11-03T10:00:00Z", "76.1", "77"
	}
	payload := e.AdapterResult.Candidate.Payload
	if payload["observed_at"].Text != wantTime || payload["temperature"].Text != temperature || payload["humidity"].Text != humidity ||
		payload["temperature_unit"].Text != "°F" || payload["humidity_unit"].Text != "%" || payload["grid_latitude"].Text != "26.94" ||
		payload["grid_longitude"].Text != "-80.08" || payload["source"].Text != "controlled Open-Meteo fixture" {
		t.Fatalf("typed weather measurements disagree with the independently served response: %+v", payload)
	}
	run, ok := s.Run(e.Source)
	if !ok || run.Projection.Status != "succeeded" || v.ObservedAt != wantTime || v.Temperature != temperature || v.Humidity != humidity || v.TemperatureUnit != "°F" || v.HumidityUnit != "%" || v.Source != "controlled Open-Meteo fixture" ||
		v.Report == "" || !strings.Contains(v.Report, temperature) || !strings.Contains(v.Report, humidity) {
		t.Fatalf("terminal report differs from the selected original measurements: %+v", v)
	}
}

func unresolvedWeather(t *testing.T, s *runtime.ExecutionSnapshot, v weatherReadView, id string) {
	t.Helper()
	e, ok := s.Effect(id)
	wantState := "unresolved"
	if ok && e.DispatchAdmission == nil {
		wantState = "spent_before_dispatch"
	}
	if !ok || e.AdapterAttempt == nil || !e.AdapterAttempt.Allowed || e.AdapterResult != nil || e.Settlement != nil || e.ResultDisposition != "" ||
		v.Operation != id || v.State != wantState || v.ObservedAt != "" || v.Temperature != "" || v.Report != "" {
		t.Fatalf("unresolved original became a weather result: %+v / %+v", e, v)
	}
	if !slices.ContainsFunc(s.Pending, func(p runtime.PendingObligation) bool { return p.OperationID == id }) {
		t.Fatalf("unresolved original obligation %s missing", id)
	}
}
