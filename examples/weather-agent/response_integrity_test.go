package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// A public caller can register an adapter with the authentic Prepare method
// but return a different response. This Execute performs no network I/O.
type substitutedWeatherResponse struct {
	*openmeteo.Adapter
	candidate effects.Candidate
}

func (a substitutedWeatherResponse) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	c := a.candidate
	c.ID, c.Key = p.Request.ID, p.Request.Key
	return c
}

func TestWeatherReadersRejectImpossibleResponses(t *testing.T) {
	const endpoint = "http://127.0.0.1:1/v1/forecast"
	adapter, err := openmeteo.New(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	// These are the correct typed fields for morningWeather. laterWeather has
	// a different observation time, temperature and humidity.
	morning := map[string]schema.Value{
		openmeteo.Source:          schema.Text("controlled Open-Meteo fixture"),
		openmeteo.ObservedAt:      schema.Text("2026-11-03T09:45:00Z"),
		openmeteo.GridLatitude:    schema.Text("26.94"),
		openmeteo.GridLongitude:   schema.Text("-80.08"),
		openmeteo.Temperature:     schema.Text("75.4"),
		openmeteo.TemperatureUnit: schema.Text("°F"),
		openmeteo.Humidity:        schema.Text("80"),
		openmeteo.HumidityUnit:    schema.Text("%"),
	}
	for _, tc := range []struct {
		name      string
		candidate effects.Candidate
	}{
		{"negative_HTTP_status", effects.Candidate{
			Disposition: "http_rejected", HTTPStatus: -17, Response: "no network call",
		}},
		{"HTTP_success_described_as_rejected", effects.Candidate{
			Disposition: "http_rejected", HTTPStatus: http.StatusOK, Response: morningWeather,
		}},
		{"foreign_usage_on_failure", effects.Candidate{
			Disposition: "http_rejected", HTTPStatus: http.StatusTooManyRequests,
			Usage: &effects.Usage{TotalTokens: new(int64(7))},
		}},
		{"unavailable_with_complete_measurements", effects.Candidate{
			Disposition: "unavailable", HTTPStatus: http.StatusOK, Response: morningWeather,
		}},
		{"successful_fields_from_another_response", effects.Candidate{
			Disposition: "validated", HTTPStatus: http.StatusOK, Response: laterWeather, Payload: morning,
		}},
		{"success_with_missing_fields_normalized_by_runtime", effects.Candidate{
			Disposition: "validated", HTTPStatus: http.StatusOK, Response: morningWeather,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := filepath.Join(t.TempDir(), "weather.sqlite")
			appCommand(t, provisioning(db, endpoint, 1)...)
			owner, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			_, err = owner.Run(t.Context(), runtime.RunOptions{
				Adapters: map[string]effects.Adapter{capability: substitutedWeatherResponse{Adapter: adapter, candidate: tc.candidate}},
				MaxSteps: 128, MaxDispatch: 32, MaxInFlight: 1,
			})
			if err := errors.Join(err, owner.Close()); err != nil {
				t.Fatal(err)
			}
			s := weatherSnapshot(t, db)
			e := weatherEffect(t, s, "weather-1")
			if e.DispatchAdmission == nil || e.AdapterResult == nil || selectedWeather(t, s, e) == nil {
				t.Fatal("generic runtime did not retain and select the substituted response")
			}
			if _, err := runtime.Replay(t.Context(), db, runtime.DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"inspect", "replay"} {
				var out bytes.Buffer
				err := run(t.Context(), []string{action, "--db", db, "--json"}, &out)
				if err == nil || out.Len() != 0 {
					t.Errorf("%s accepted impossible weather evidence or emitted a partial report (err=%v, bytes=%d)", action, err, out.Len())
				}
			}
		})
	}
}
