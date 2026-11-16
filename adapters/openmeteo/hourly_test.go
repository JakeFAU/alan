package openmeteo

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const hourlyFixture = `{"latitude":26.940,"longitude":-80.08,"utc_offset_seconds":0,"timezone":"GMT","timezone_abbreviation":"GMT","elevation":7,"hourly_units":{"time":"iso8601","temperature_2m":"°F","weather_code":"wmo code"},"hourly":{"time":["2026-11-07T23:00","2026-11-08T00:00"],"temperature_2m":[75.40,7.2e1],"weather_code":[0,61]}}`

func hourlyTestRequest(t *testing.T) effects.Request {
	t.Helper()
	input, err := HourlyInputSchema().Retain()
	if err != nil {
		t.Fatal(err)
	}
	output, err := HourlyOutputSchema().Retain()
	if err != nil {
		t.Fatal(err)
	}
	return effects.Request{
		ID: `["forecast",1,4,0]`, Key: "forecast/key", Capability: "weather", Effect: "forecast",
		Input: input, Output: output, Payload: map[string]schema.Value{
			Latitude: schema.Text("26.930"), Longitude: schema.Text("-80.08"), Units: schema.Text("fahrenheit"),
			StartHour: schema.Text("2026-11-07T23:00:00Z"), EndHour: schema.Text("2026-11-08T00:00:00Z"),
		},
	}
}

func hourlyTestAdapter(t *testing.T, endpoint string) (*HourlyAdapter, effects.Prepared) {
	t.Helper()
	a, err := NewHourly(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Prepare(hourlyTestRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	return a, p
}

func TestHourlyRequestAndEvidence(t *testing.T) {
	body := strings.Replace(hourlyFixture, `"weather_code":[0,61]`, `"weather_code":[0,97]`, 1)
	var contacts atomic.Int32
	seen := make(chan *http.Request, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contacts.Add(1) == 1 {
			seen <- r.Clone(r.Context())
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	a, p := hourlyTestAdapter(t, srv.URL+"/v1/forecast")
	const path = "/v1/forecast?cell_selection=nearest&end_hour=2026-11-08T00%3A00&hourly=temperature_2m%2Cweather_code&latitude=26.930&longitude=-80.08&start_hour=2026-11-07T23%3A00&temperature_unit=fahrenheit&timeformat=iso8601&timezone=GMT"
	if p.Protocol != "open-meteo/hourly/v1" || p.Endpoint != srv.URL+"/v1/forecast" || p.Wire != srv.URL+path ||
		p.MaxResponseBytes != MaxResponseBytes || p.Digest == "" || effects.CheckPrepared(p) != nil ||
		!reflect.DeepEqual(p.Request, hourlyTestRequest(t)) || contacts.Load() != 0 {
		t.Fatalf("incorrect preparation %+v", p)
	}
	if again, err := a.Prepare(effects.Copy(p.Request)); err != nil || !reflect.DeepEqual(again, p) {
		t.Fatalf("non-deterministic preparation: %+v, %v", again, err)
	}
	c := a.Execute(t.Context(), p)
	if c.Disposition != "validated" || c.ID != p.Request.ID || c.Key != p.Request.Key || c.Response != body ||
		c.HTTPStatus != http.StatusOK || contacts.Load() != 1 {
		t.Fatalf("unexpected result %+v, contacts %d", c, contacts.Load())
	}
	req := <-seen
	if req.Method != http.MethodGet || req.Host != strings.TrimPrefix(srv.URL, "http://") || req.URL.RequestURI() != path ||
		req.Header.Get(ClientRequestHeader) != ClientRequestID(p.Request.Key) ||
		req.Header.Get(OperationHeader) != base64.RawURLEncoding.EncodeToString([]byte(p.Request.ID)) ||
		req.Header.Get("Accept-Encoding") != "identity" {
		t.Fatalf("actual request %s %s %v", req.Method, req.URL, req.Header)
	}
	want := map[string]schema.Value{
		Source: schema.Text("controlled Open-Meteo fixture"), GridLatitude: schema.Text("26.940"), GridLongitude: schema.Text("-80.08"),
		TemperatureUnit: schema.Text("°F"), StartHour: schema.Text("2026-11-07T23:00:00Z"), EndHour: schema.Text("2026-11-08T00:00:00Z"),
		StartTemperature: schema.Text("75.40"), EndTemperature: schema.Text("7.2e1"), StartWeatherCode: schema.Int(0), EndWeatherCode: schema.Int(97),
	}
	if !reflect.DeepEqual(c.Payload, want) || HourlyOutputSchema().Validate(c.Payload) != nil {
		t.Fatalf("payload %+v, want %+v", c.Payload, want)
	}
	if err := VerifyHourlyResponseEvidence(effects.Copy(p), effects.Copy(c)); err != nil || contacts.Load() != 1 {
		t.Fatalf("offline evidence error %v, contacts %d", err, contacts.Load())
	}
}

func TestHourlyInvalidEvidence(t *testing.T) {
	tests := []struct{ name, old, replacement, want string }{
		{"malformed JSON", hourlyFixture, `{`, "protocol_invalid"},
		{"trailing JSON", hourlyFixture, hourlyFixture + ` {}`, "protocol_invalid"},
		{"duplicate nested key", `"weather_code":[0,61]`, `"weather_code":[1,1],"weather_code":[0,61]`, "protocol_invalid"},
		{"duplicate metadata key", `"elevation":7`, `"elevation":8,"elevation":7`, "protocol_invalid"},
		{"wrong hour", `T23:00`, `T22:00`, "protocol_invalid"},
		{"missing sample", `,"2026-11-08T00:00"`, ``, "protocol_invalid"},
		{"missing times", `"time":["2026-11-07T23:00","2026-11-08T00:00"],`, ``, "protocol_invalid"},
		{"duplicate hour", `2026-11-08T00:00`, `2026-11-07T23:00`, "protocol_invalid"},
		{"extra hour", `"2026-11-08T00:00"],"temperature_2m":[75.40,7.2e1],"weather_code":[0,61]`,
			`"2026-11-08T00:00","2026-11-08T01:00"],"temperature_2m":[75.40,7.2e1,73.0],"weather_code":[0,61,0]`, "protocol_invalid"},
		{"mismatched temperatures", `[75.40,7.2e1]`, `[75.40]`, "protocol_invalid"},
		{"mismatched codes", `[0,61]`, `[0,61,61]`, "protocol_invalid"},
		{"null time", `"2026-11-07T23:00"`, `null`, "protocol_invalid"},
		{"null temperature", `75.40`, `null`, "unavailable"},
		{"null code", `[0,61]`, `[0,null]`, "unavailable"},
		{"unknown WMO code", `[0,61]`, `[0,98]`, "unavailable"},
		{"fractional WMO code", `[0,61]`, `[0,61.5]`, "protocol_invalid"},
		{"numeric string", `75.40`, `"75.40"`, "protocol_invalid"},
		{"temperature range", `75.40`, `302.0000000000000001`, "protocol_invalid"},
		{"wrong temperature unit", `°F`, `°C`, "protocol_invalid"},
		{"wrong code unit", `"wmo code"`, `"percent"`, "protocol_invalid"},
		{"wrong time unit", `"iso8601"`, `"unixtime"`, "protocol_invalid"},
		{"wrong time zone", `"timezone":"GMT"`, `"timezone":"America/New_York"`, "protocol_invalid"},
		{"wrong time zone abbreviation", `"timezone_abbreviation":"GMT"`, `"timezone_abbreviation":"PST"`, "protocol_invalid"},
		{"wrong offset", `"utc_offset_seconds":0`, `"utc_offset_seconds":3600`, "protocol_invalid"},
		{"far grid", `26.940`, `27.940`, "protocol_invalid"},
		{"coordinate range", `-80.08`, `-180.01`, "protocol_invalid"},
		{"provider error", `{"latitude"`, `{"error":true,"latitude"`, "protocol_invalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(hourlyFixture, tc.old, tc.replacement, 1)
			if body == hourlyFixture {
				t.Fatal("fixture mutation had no effect")
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, body)
			}))
			t.Cleanup(srv.Close)
			a, p := hourlyTestAdapter(t, srv.URL+"/v1/forecast")
			c := a.Execute(t.Context(), p)
			if c.Disposition != tc.want || c.HTTPStatus != http.StatusOK || c.Payload != nil || c.Response != body {
				t.Fatalf("unexpected refusal %+v, want disposition %s", c, tc.want)
			}
			if err := VerifyHourlyResponseEvidence(p, c); err != nil {
				t.Fatal("genuine refusal failed offline verification", err)
			}
		})
	}
}

func TestHourlyTimeRange(t *testing.T) {
	a, _ := hourlyTestAdapter(t, DefaultEndpoint)
	for _, tc := range []struct{ name, start, end string }{
		{"missing", "", "2026-11-08T00:00:00Z"},
		{"UTC offset spelling", "2026-11-07T23:00:00+00:00", "2026-11-08T00:00:00Z"},
		{"seconds", "2026-11-07T23:00:01Z", "2026-11-08T00:00:01Z"},
		{"fractional seconds", "2026-11-07T23:00:00.0Z", "2026-11-08T00:00:00Z"},
		{"minutes", "2026-11-07T23:01:00Z", "2026-11-08T00:01:00Z"},
		{"wrong date", "2026-11-31T23:00:00Z", "2026-12-01T00:00:00Z"},
		{"equal", "2026-11-08T00:00:00Z", "2026-11-08T00:00:00Z"},
		{"reversed", "2026-11-08T01:00:00Z", "2026-11-08T00:00:00Z"},
		{"too long", "2026-11-07T22:00:00Z", "2026-11-08T00:00:00Z"},
		{"year zero", "0000-12-31T23:00:00Z", "0001-01-01T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := hourlyTestRequest(t)
			r.Payload[StartHour], r.Payload[EndHour] = schema.Text(tc.start), schema.Text(tc.end)
			if _, err := a.Prepare(r); err == nil {
				t.Fatalf("accepted time range %s to %s", tc.start, tc.end)
			}
		})
	}
}

func TestHourlyReportBindsSamplesAndPreparedRequest(t *testing.T) {
	// The two arrays remain paired by index even if the hours are returned in reverse order.
	body := strings.NewReplacer(`"2026-11-07T23:00","2026-11-08T00:00"`, `"2026-11-08T00:00","2026-11-07T23:00"`,
		`[75.40,7.2e1]`, `[7.2e1,75.40]`, `[0,61]`, `[61,0]`).Replace(hourlyFixture)
	var contacts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	a, p := hourlyTestAdapter(t, srv.URL+"/v1/forecast")
	c := a.Execute(t.Context(), p)
	if c.Disposition != "validated" || c.Payload[StartTemperature] != schema.Text("75.40") ||
		c.Payload[EndTemperature] != schema.Text("7.2e1") || c.Payload[StartWeatherCode] != schema.Int(0) || c.Payload[EndWeatherCode] != schema.Int(61) {
		t.Fatalf("incorrectly paired samples %+v", c)
	}
	for _, mutate := range []func(*effects.Candidate){
		func(c *effects.Candidate) { c.Payload[StartTemperature] = schema.Text("75.4") },
		func(c *effects.Candidate) { c.Payload[EndWeatherCode] = schema.Int(0) },
		func(c *effects.Candidate) { c.Response = strings.Replace(c.Response, `T23:00`, `T22:00`, 1) },
		func(c *effects.Candidate) { c.Key += "different" },
	} {
		changed := effects.Copy(c)
		mutate(&changed)
		if VerifyHourlyResponseEvidence(p, changed) == nil {
			t.Fatalf("accepted forged candidate %+v", changed)
		}
	}
	for _, mutate := range []func(*effects.Prepared){
		func(p *effects.Prepared) { p.Wire += "&latitude=0" },
		func(p *effects.Prepared) { p.Protocol = Protocol },
		func(p *effects.Prepared) { p.MaxResponseBytes++ },
		func(p *effects.Prepared) { p.Request.Payload[Latitude] = schema.Text("26.94") },
	} {
		changed := effects.Copy(p)
		mutate(&changed)
		changed = effects.Freeze(changed)
		if VerifyHourlyResponseEvidence(changed, c) == nil {
			t.Fatalf("accepted forged preparation %+v", changed)
		}
		if rejected := a.Execute(t.Context(), changed); rejected.Disposition != "request_invalid" || contacts.Load() != 1 {
			t.Fatalf("forged preparation yielded %+v with total contacts %d", rejected, contacts.Load())
		}
	}
	if VerifyResponseEvidence(p, c) == nil {
		t.Fatal("current verifier accepted hourly evidence")
	}
}

func TestHourlyTransportBounds(t *testing.T) {
	var contacts atomic.Int32
	for _, tc := range []struct {
		name, disposition string
		status, size      int
		handler           http.HandlerFunc
	}{
		{"redirect", "http_rejected", http.StatusTemporaryRedirect, -1, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/v1/forecast", http.StatusTemporaryRedirect)
		}},
		{"truncated", "unknown", 0, 0, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "9000")
			io.WriteString(w, hourlyFixture)
		}},
		{"oversized", "response_limit", http.StatusOK, MaxResponseBytes, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, hourlyFixture+strings.Repeat(" ", MaxResponseBytes))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := contacts.Load()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				tc.handler(w, r)
			}))
			t.Cleanup(srv.Close)
			a, p := hourlyTestAdapter(t, srv.URL+"/v1/forecast")
			c := a.Execute(t.Context(), p)
			if c.Disposition != tc.disposition || c.HTTPStatus != tc.status || (tc.size >= 0 && len(c.Response) != tc.size) || c.Payload != nil || contacts.Load()-before != 1 {
				t.Fatalf("disposition=%q HTTP=%d bytes=%d contacts=%d", c.Disposition, c.HTTPStatus, len(c.Response), contacts.Load()-before)
			}
			if c.HTTPStatus != 0 && VerifyHourlyResponseEvidence(p, c) != nil {
				t.Fatal("failed to verify retained transport refusal")
			}
		})
	}
}

func TestHourlyWeatherCodeSet(t *testing.T) {
	want := map[int64]bool{
		0: true, 1: true, 2: true, 3: true, 45: true, 48: true,
		51: true, 53: true, 55: true, 56: true, 57: true,
		61: true, 63: true, 65: true, 66: true, 67: true,
		71: true, 73: true, 75: true, 77: true, 80: true,
		81: true, 82: true, 85: true, 86: true, 95: true, 96: true, 97: true, 99: true,
	}
	for i := int64(-1); i <= 100; i++ {
		if got := ValidWeatherCode(i); got != want[i] {
			t.Errorf("ValidWeatherCode(%d) = %t, want %t", i, got, want[i])
		}
	}
}
