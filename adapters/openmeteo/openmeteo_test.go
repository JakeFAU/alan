package openmeteo

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const fixture = `{"latitude":26.94,"longitude":-80.08,"utc_offset_seconds":0,"timezone":"GMT","timezone_abbreviation":"GMT","elevation":7,"current_units":{"time":"iso8601","interval":"seconds","temperature_2m":"°F","relative_humidity_2m":"%"},"current":{"time":"2026-11-03T09:45","interval":900,"temperature_2m":75.40,"relative_humidity_2m":80}}`

func testRequest(t *testing.T) effects.Request {
	t.Helper()
	input, err := InputSchema().Retain()
	if err != nil {
		t.Fatal(err)
	}
	output, err := OutputSchema().Retain()
	if err != nil {
		t.Fatal(err)
	}
	return effects.Request{ID: `["weather",1,4,0]`, Key: "external/key", Capability: "weather", Effect: "observe",
		Input: input, Output: output, Payload: map[string]schema.Value{
			Latitude: schema.Text("26.930"), Longitude: schema.Text("-80.08"), Units: schema.Text("fahrenheit"),
		}}
}

func testAdapter(t *testing.T, endpoint string) (*Adapter, effects.Prepared) {
	t.Helper()
	a, err := New(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Prepare(testRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	return a, p
}

func TestPreparedRequestAndObservation(t *testing.T) {
	var contacts atomic.Int32
	seen := make(chan *http.Request, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		seen <- r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, fixture)
	}))
	t.Cleanup(srv.Close)
	a, p := testAdapter(t, srv.URL+"/v1/forecast")
	again, err := a.Prepare(effects.Copy(p.Request))
	if err != nil || !reflect.DeepEqual(p, again) || p.Digest == "" || effects.CheckPrepared(p) != nil || contacts.Load() != 0 {
		t.Fatalf("non-deterministic preparation: %+v %v", again, err)
	}
	c := a.Execute(t.Context(), p)
	if c.Disposition != "validated" || c.ID != p.Request.ID || c.Key != p.Request.Key || c.Response != fixture || c.HTTPStatus != 200 || c.Usage != nil || contacts.Load() != 1 {
		t.Fatalf("result %+v, contacts %d", c, contacts.Load())
	}
	req := <-seen
	if req.Method != "GET" || req.Host != strings.TrimPrefix(srv.URL, "http://") || srv.URL+req.URL.RequestURI() != p.Wire ||
		req.Header.Get(ClientRequestHeader) != ClientRequestID(p.Request.Key) || req.Header.Get(OperationHeader) != base64.RawURLEncoding.EncodeToString([]byte(p.Request.ID)) ||
		req.Header.Get("Accept-Encoding") != "identity" {
		t.Fatalf("actual request %s %s %v, prepared %+v", req.Method, req.URL, req.Header, p)
	}
	params := url.Values{"latitude": {"26.930"}, "longitude": {"-80.08"}, "current": {"temperature_2m,relative_humidity_2m"}, "temperature_unit": {"fahrenheit"}, "timezone": {"GMT"}, "timeformat": {"iso8601"}, "cell_selection": {"nearest"}}
	if !reflect.DeepEqual(req.URL.Query(), params) {
		t.Fatalf("query %v, want %v", req.URL.Query(), params)
	}
	want := map[string]schema.Value{
		Source: schema.Text("controlled Open-Meteo fixture"), ObservedAt: schema.Text("2026-11-03T09:45:00Z"),
		GridLatitude: schema.Text("26.94"), GridLongitude: schema.Text("-80.08"),
		Temperature: schema.Text("75.40"), TemperatureUnit: schema.Text("°F"), Humidity: schema.Text("80"), HumidityUnit: schema.Text("%"),
	}
	if !reflect.DeepEqual(c.Payload, want) || OutputSchema().Validate(c.Payload) != nil || VerifyResponseEvidence(p, c) != nil {
		t.Fatalf("validated weather %+v", c)
	}
	changed := effects.Copy(c)
	changed.Payload[Temperature] = schema.Text("75.4")
	if VerifyResponseEvidence(p, changed) == nil {
		t.Fatal("accepted changed measurement lexeme")
	}
	changed = effects.Copy(c)
	changed.Response = strings.ReplaceAll(fixture, "09:45", "09:30")
	if VerifyResponseEvidence(p, changed) == nil {
		t.Fatal("accepted changed provider time")
	}
}

func TestRequestBoundsAndEndpoint(t *testing.T) {
	for _, endpoint := range []string{"", "http://api.open-meteo.com/v1/forecast", DefaultEndpoint + "?", DefaultEndpoint + "#", DefaultEndpoint + "?apikey=secret",
		"https://token@api.open-meteo.com/v1/forecast", "https://example.com/v1/forecast", "http://localhost:5000/v1/forecast",
		"http://192.0.2.1:81/v1/forecast", "http://127.0.0.1:5000/a", "http://127.0.0.1:0/v1/forecast", "http://127.0.0.1:65536/v1/forecast"} {
		if _, err := New(endpoint); err == nil {
			t.Errorf("accepted endpoint %q", endpoint)
		}
	}
	for _, endpoint := range []string{DefaultEndpoint, "http://[::1]:1234/v1/forecast", "http://127.0.0.1:5000/v1/forecast"} {
		if _, err := New(endpoint); err != nil {
			t.Errorf("rejected endpoint %q: %v", endpoint, err)
		}
	}
	for _, lat := range []string{"", " 1", "NaN", "+5", "1,3", "1e1", "-90.00000000000000000001", "91", "26&latitude=1", strings.Repeat("1", 40)} {
		if ValidateLocation(lat, "180", "fahrenheit") == nil {
			t.Errorf("accepted latitude %q", lat)
		}
	}
	if ValidateLocation("-90", "-180.00", "celsius") != nil || ValidateLocation("1", "180.01", "celsius") == nil || ValidateLocation("1", "2", "kelvin") == nil {
		t.Fatal("coordinate or unit bounds")
	}
	a, p := testAdapter(t, DefaultEndpoint)
	for _, corrupt := range []func(*effects.Request){
		func(r *effects.Request) { r.Input.ID = "" }, func(r *effects.Request) { r.Output.Record[Temperature] = schema.Field{Kind: "int"} },
		func(r *effects.Request) { r.Payload["endpoint"] = schema.Text(DefaultEndpoint) }, func(r *effects.Request) { r.Payload[Latitude] = schema.Int(26) },
		func(r *effects.Request) { r.Payload[Units] = schema.Text("metric") }, func(r *effects.Request) { r.Key = "" },
	} {
		r := effects.Copy(p.Request)
		corrupt(&r)
		if _, err := a.Prepare(r); err == nil {
			t.Errorf("accepted changed request %+v", r)
		}
	}
	original := testRequest(t)
	p, err := a.Prepare(original)
	if err != nil {
		t.Fatal(err)
	}
	original.Payload[Latitude] = schema.Text("41")
	if p.Request.Payload[Latitude].Text != "26.930" {
		t.Fatal("prepared request aliases caller")
	}
}

func TestInvalidOrUnavailableResponse(t *testing.T) {
	tests := []struct {
		name, body, mime, coding, want string
		status                         int
	}{
		{"syntax", "{", "application/json", "", "protocol_invalid", 200},
		{"trailing", fixture + " {}", "application/json", "", "protocol_invalid", 200},
		{"duplicate", strings.ReplaceAll(fixture, `"temperature_2m":75.40`, `"temperature_2m":20,"temperature_2m":75.40`), "application/json", "", "protocol_invalid", 200},
		{"spelling", strings.ReplaceAll(fixture, `"temperature_2m":75.40`, `"TEMPERATURE_2M":75.40`), "application/json", "", "protocol_invalid", 200},
		{"missing", strings.ReplaceAll(fixture, `,"relative_humidity_2m":80`, ""), "application/json", "", "protocol_invalid", 200},
		{"unavailable", strings.ReplaceAll(fixture, `"relative_humidity_2m":80`, `"relative_humidity_2m":null`), "application/json", "", "unavailable", 200},
		{"non_number", strings.ReplaceAll(fixture, `:75.40`, `:"75.40"`), "application/json", "", "protocol_invalid", 200},
		{"range", strings.ReplaceAll(fixture, `:80}`, `:100.000000000000000001}`), "application/json", "", "protocol_invalid", 200},
		{"impossible_temperature", strings.ReplaceAll(fixture, `:75.40`, `:-1000`), "application/json", "", "protocol_invalid", 200},
		{"unexpected_unit", strings.ReplaceAll(fixture, "°F", "°C"), "application/json", "", "protocol_invalid", 200},
		{"humidity_unit", strings.ReplaceAll(fixture, `"%"`, `"ratio"`), "application/json", "", "protocol_invalid", 200},
		{"time", strings.ReplaceAll(fixture, "11-03T09", "02-30T09"), "application/json", "", "protocol_invalid", 200},
		{"offset", strings.ReplaceAll(fixture, "seconds\":0", "seconds\":3600"), "application/json", "", "protocol_invalid", 200},
		{"timezone", strings.ReplaceAll(fixture, `"GMT"`, `"America/New_York"`), "application/json", "", "protocol_invalid", 200},
		{"missing_time_units", strings.ReplaceAll(fixture, `"time":"iso8601",`, ""), "application/json", "", "protocol_invalid", 200},
		{"wrong_grid", strings.ReplaceAll(fixture, "26.94", "27.94"), "application/json", "", "protocol_invalid", 200},
		{"wrong_grid_range", strings.ReplaceAll(fixture, "-80.08", "-181"), "application/json", "", "protocol_invalid", 200},
		{"mixed_error", strings.Replace(fixture, "{", `{"error":true,`, 1), "application/json", "", "protocol_invalid", 200},
		{"refusal", `{"error":true,"reason":"rate limit"}`, "application/json", "", "http_rejected", 429},
		{"partial_status", fixture, "application/json", "", "http_rejected", 206},
		{"content_type", fixture, "text/html", "", "protocol_invalid", 200},
		{"content_encoding", fixture, "application/json", "gzip", "protocol_invalid", 200},
		{"limit", strings.Repeat("x", MaxResponseBytes+1), "application/json", "", "response_limit", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var contacts atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				contacts.Add(1)
				w.Header().Set("Content-Type", tc.mime)
				if tc.coding != "" {
					w.Header().Set("Content-Encoding", tc.coding)
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(srv.Close)
			a, p := testAdapter(t, srv.URL+"/v1/forecast")
			c := a.Execute(t.Context(), p)
			if c.Disposition != tc.want || c.Payload != nil || c.HTTPStatus != tc.status || c.Usage != nil || len(c.Response) > MaxResponseBytes || contacts.Load() != 1 {
				t.Fatalf("disposition=%q HTTP=%d contacts=%d size=%d", c.Disposition, c.HTTPStatus, contacts.Load(), len(c.Response))
			}
			if err := VerifyResponseEvidence(p, c); err != nil {
				t.Fatal("genuine provider failure did not verify offline", err)
			}
		})
	}
}

func TestLostResponsesAndRedirect(t *testing.T) {
	var contacts atomic.Int32
	redirected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(100)
		fmt.Fprint(w, fixture)
	}))
	t.Cleanup(redirected.Close)
	for _, tc := range []struct{ name, want string }{{"closed", "unknown"}, {"partial_body", "unknown"}, {"redirect", "http_rejected"}} {
		t.Run(tc.name, func(t *testing.T) {
			before := contacts.Load()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				if tc.name == "redirect" {
					http.Redirect(w, r, redirected.URL+"/v1/forecast", http.StatusTemporaryRedirect)
					return
				}
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if tc.name == "partial_body" {
					fmt.Fprint(conn, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 9000\r\n\r\n", fixture)
				}
				conn.Close()
			}))
			t.Cleanup(srv.Close)
			a, p := testAdapter(t, srv.URL+"/v1/forecast")
			c := a.Execute(t.Context(), p)
			if c.Disposition != tc.want || c.Payload != nil || contacts.Load()-before != 1 {
				t.Fatalf("%+v contacts %d", c, contacts.Load()-before)
			}
		})
	}
}

func TestCancellationAtAcceptedRequest(t *testing.T) {
	accepted := make(chan struct{})
	var contacts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		contacts.Add(1)
		close(accepted)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	a, p := testAdapter(t, srv.URL+"/v1/forecast")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan effects.Candidate, 1)
	go func() { done <- a.Execute(ctx, p) }()
	<-accepted
	cancel()
	c := <-done
	if c.Disposition != "unknown" || c.Response != "" || c.Payload != nil || contacts.Load() != 1 {
		t.Fatalf("withheld response %+v calls=%d", c, contacts.Load())
	}
	if c = a.Execute(ctx, p); c.Disposition != "unknown" || contacts.Load() != 1 {
		t.Fatalf("canceled request contacted provider: %+v", c)
	}
}

func TestPreparedConfigurationCannotRedirect(t *testing.T) {
	var contacts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { contacts.Add(1); io.WriteString(w, fixture) }))
	t.Cleanup(srv.Close)
	a, p := testAdapter(t, srv.URL+"/v1/forecast")
	for _, mutate := range []func(*effects.Prepared){func(p *effects.Prepared) { p.Wire += "&latitude=0" }, func(p *effects.Prepared) { p.Endpoint = DefaultEndpoint }, func(p *effects.Prepared) { p.Protocol = "other" }, func(p *effects.Prepared) { p.MaxResponseBytes++ }, func(p *effects.Prepared) { p.Digest = "" }} {
		changed := effects.Copy(p)
		mutate(&changed)
		if changed.Digest != "" {
			changed = effects.Freeze(changed)
		}
		c := a.Execute(t.Context(), changed)
		if c.Disposition == "validated" || contacts.Load() != 0 {
			t.Fatalf("contact with corrupted preparation %+v", c)
		}
	}
}

func TestConcurrentRequestsKeepTheirOwnEvidence(t *testing.T) {
	seen := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get(ClientRequestHeader)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, fixture)
	}))
	t.Cleanup(srv.Close)
	a, p := testAdapter(t, srv.URL+"/v1/forecast")
	want := map[string]bool{}
	results := make(chan effects.Candidate, cap(seen))
	for i := 0; i < cap(seen); i++ {
		r := p.Request
		r.ID, r.Key = "op"+strconv.Itoa(i), "key"+strconv.Itoa(i)
		want[ClientRequestID(r.Key)] = true
		p, err := a.Prepare(r)
		if err != nil {
			t.Fatal(err)
		}
		go func() { results <- a.Execute(t.Context(), p) }()
	}
	for i := 0; i < cap(seen); i++ {
		c := <-results
		if c.Disposition != "validated" || c.ID != "op"+strings.TrimPrefix(c.Key, "key") {
			t.Fatalf("unexpected candidate %+v", c)
		}
		h := <-seen
		if !want[h] {
			t.Fatalf("unexpected contact %q", h)
		}
		delete(want, h)
	}
}

func TestCelsiusEvidenceAcrossAntimeridian(t *testing.T) {
	a, p := testAdapter(t, DefaultEndpoint)
	r := p.Request
	r.Payload[Latitude] = schema.Text("89.9")
	r.Payload[Longitude] = schema.Text("179.9")
	r.Payload[Units] = schema.Text("celsius")
	p, err := a.Prepare(r)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.NewReplacer("26.94", "89.88", "-80.08", "-179.9", "°F", "°C", "75.40", "-2.1e1").Replace(fixture)
	payload, disposition := observation(p, body)
	if disposition != "validated" || payload[Temperature] != schema.Text("-2.1e1") || payload[Source] != schema.Text("Open-Meteo") {
		t.Fatalf("valid Celsius grid near dateline: %q %+v", disposition, payload)
	}
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: disposition, HTTPStatus: 200, Response: body, Payload: payload}
	// This is an offline protocol fixture for the official endpoint. Never send it.
	if err := VerifyResponseEvidence(p, c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*effects.Candidate){
		func(c *effects.Candidate) { c.Key += "different" }, func(c *effects.Candidate) { c.HTTPStatus = 299 },
		func(c *effects.Candidate) { c.Usage = &effects.Usage{} }, func(c *effects.Candidate) { c.ResponseID = "invented" },
		func(c *effects.Candidate) { c.Payload[GridLatitude] = schema.Text("89.9") },
	} {
		changed := effects.Copy(c)
		change(&changed)
		if VerifyResponseEvidence(p, changed) == nil {
			t.Fatalf("verified changed evidence: %+v", changed)
		}
	}
}

func TestConflictingContentMetadata(t *testing.T) {
	for _, headers := range []http.Header{
		{"Content-Type": {"application/json", "text/plain"}},
		{"Content-Type": {"application/json; charset=iso-8859-1"}},
		{"Content-Type": {"application/json"}, "Content-Encoding": {"identity", "gzip"}},
		{"Content-Type": {"application/json"}, "Content-Range": {"bytes 0-99/1000"}},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			for k, values := range headers {
				for _, value := range values {
					w.Header().Add(k, value)
				}
			}
			fmt.Fprint(w, fixture)
		}))
		a, p := testAdapter(t, srv.URL+"/v1/forecast")
		c := a.Execute(t.Context(), p)
		srv.Close()
		if c.Disposition != "protocol_invalid" || c.Payload != nil {
			t.Errorf("accepted headers %v: %q", headers, c.Disposition)
		}
	}
}
