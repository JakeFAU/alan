package main

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// These are independent provider wire responses. The second read changes both
// the reported observation time and the measurements at the same coordinates.
const morningWeather = `{"latitude":26.94,"longitude":-80.08,"utc_offset_seconds":0,"timezone":"GMT","current_units":{"time":"iso8601","temperature_2m":"°F","relative_humidity_2m":"%"},"current":{"time":"2026-11-03T09:45","temperature_2m":75.4,"relative_humidity_2m":80}}`
const laterWeather = `{"latitude":26.94,"longitude":-80.08,"utc_offset_seconds":0,"timezone":"GMT","current_units":{"time":"iso8601","temperature_2m":"°F","relative_humidity_2m":"%"},"current":{"time":"2026-11-03T10:00","temperature_2m":76.1,"relative_humidity_2m":77}}`

type weatherReply struct {
	status int
	body   string
	hold   bool
	close  bool
}

type weatherContact struct {
	index                    int
	operation, clientID, uri string
	status                   int
	body                     string
	written                  bool
}

type weatherProvider struct {
	server   *httptest.Server
	mu       sync.Mutex
	log      []weatherContact
	accepted chan weatherContact
	release  chan struct{}
}

// The fixture owns its acceptance gate and append-only log. It receives no
// database path, ALAN object, or adapter. An extra request is recorded even
// when the fixture has exhausted its response script.
func newWeatherProvider(t *testing.T, replies ...weatherReply) *weatherProvider {
	t.Helper()
	p := &weatherProvider{accepted: make(chan weatherContact, 16), release: make(chan struct{})}
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := url.Values{
			"latitude": {"26.93"}, "longitude": {"-80.08"}, "temperature_unit": {"fahrenheit"},
			"current": {"temperature_2m,relative_humidity_2m"}, "timezone": {"GMT"}, "timeformat": {"iso8601"}, "cell_selection": {"nearest"},
		}
		if r.Method != http.MethodGet || r.URL.Path != "/v1/forecast" || r.URL.Query().Encode() != want.Encode() {
			t.Errorf("provider got unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
		rawID, err := base64.RawURLEncoding.DecodeString(r.Header.Get("X-Alan-Operation-Id"))
		if err != nil || len(rawID) == 0 || r.Header.Get("X-Client-Request-Id") == "" {
			t.Errorf("missing or malformed provider correlation headers: %v", err)
		}
		p.mu.Lock()
		index := len(p.log) + 1
		reply := weatherReply{status: http.StatusTooManyRequests, body: `{"error":true,"reason":"unplanned repeated lookup"}`}
		if index <= len(replies) {
			reply = replies[index-1]
		} else {
			t.Errorf("unplanned provider contact %d operation %s", index, rawID)
		}
		c := weatherContact{index: index, operation: string(rawID), clientID: r.Header.Get("X-Client-Request-Id"), uri: r.URL.RequestURI(), status: reply.status, body: reply.body}
		p.log = append(p.log, c)
		p.mu.Unlock()
		t.Logf("HTTP accepted index=%d operation=%s client=%s request=%q hold=%t disconnect=%t", c.index, c.operation, c.clientID, c.uri, reply.hold, reply.close)
		select {
		case p.accepted <- c:
		case <-r.Context().Done():
			return
		}
		if reply.hold {
			select {
			case <-p.release:
			case <-r.Context().Done():
				t.Logf("HTTP ended index=%d without writing a response", index)
				return
			}
		}
		if reply.close {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.status)
		if _, err := io.WriteString(w, reply.body); err != nil {
			t.Errorf("sending scripted response for contact %d: %v", c.index, err)
			return
		}
		p.mu.Lock()
		p.log[index-1].written = true
		p.mu.Unlock()
		t.Logf("HTTP response written index=%d status=%d sha256=%x", index, reply.status, sha256.Sum256([]byte(reply.body)))
	}))
	t.Cleanup(func() {
		p.server.CloseClientConnections()
		p.server.Close()
	})
	return p
}

func (p *weatherProvider) endpoint() string { return p.server.URL + "/v1/forecast" }

func (p *weatherProvider) contacts() []weatherContact {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.log)
}

func (p *weatherProvider) waitAccepted(t *testing.T) weatherContact {
	t.Helper()
	select {
	case c := <-p.accepted:
		return c
	case <-time.After(weatherProcessTimeout):
		t.Fatal("original HTTP request never reached the independent endpoint")
	}
	return weatherContact{}
}

func (p *weatherProvider) requireContacts(t *testing.T, n int) {
	t.Helper()
	if got := p.contacts(); len(got) != n {
		t.Fatalf("want %d endpoint contacts, got %+v", n, got)
	}
}

func wireFaults() map[string]weatherReply {
	return map[string]weatherReply{
		"socket_close":             {close: true},
		"malformed_json":           {status: 200, body: "{"},
		"missing_observation_time": {status: 200, body: strings.Replace(morningWeather, `"time":"2026-11-03T09:45",`, "", 1)},
		"missing_measurement":      {status: 200, body: strings.Replace(morningWeather, `,"relative_humidity_2m":80`, "", 1)},
		"wrong_unit":               {status: 200, body: strings.Replace(morningWeather, "°F", "°C", 1)},
		"invalid_location":         {status: 200, body: strings.Replace(morningWeather, `"latitude":26.94`, `"latitude":117`, 1)},
		"incompatible_schema":      {status: 200, body: strings.Replace(morningWeather, `"current":{`, `"current_weather":{`, 1)},
		"refused":                  {status: 403, body: `{"error":true,"reason":"access refused"}`},
		"rate_limit":               {status: 429, body: `{"error":true,"reason":"rate limited"}`},
	}
}
