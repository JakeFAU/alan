package main

import (
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/JakeFAU/alan/adapters/openmeteo"
)

const (
	fixturePath = "/v1/forecast"
	wireHour    = "2006-01-02T15:04"
	// JSON escaping can expand each original byte to six fixture bytes.
	maxFixtureBytes = maxCandidates*(6*openmeteo.MaxResponseBytes+4096) + 4096
)

type fixtureTuple struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	StartHour string `json:"start_hour"`
	EndHour   string `json:"end_hour"`
}

func (t fixtureTuple) String() string {
	return fmt.Sprintf("(%s, %s, %s, %s)", t.Latitude, t.Longitude, t.StartHour, t.EndHour)
}

type fixtureEntry struct {
	fixtureTuple
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

type weatherFixture struct {
	ReferenceTime time.Time
	ReferenceText string
	Entries       map[fixtureTuple]fixtureEntry
}

func readFixture(path string) (weatherFixture, []byte, error) {
	raw, err := readBoundedFile(path, maxFixtureBytes)
	if err != nil {
		return weatherFixture{}, nil, err
	}
	var input struct {
		Version       int            `json:"version"`
		ReferenceTime string         `json:"reference_time"`
		Entries       []fixtureEntry `json:"entries"`
	}
	if err := jsonv2.Unmarshal(raw, &input, jsonv2.RejectUnknownMembers(true)); err != nil {
		return weatherFixture{}, nil, fmt.Errorf("weather fixture: %w", err)
	}
	if input.Version != 1 || input.Entries == nil || len(input.Entries) > maxCandidates {
		return weatherFixture{}, nil, errors.New("weather fixture requires version 1 and at most sixteen entries")
	}
	ref, err := parseExplicitTime(input.ReferenceTime)
	if err != nil {
		return weatherFixture{}, nil, fmt.Errorf("fixture reference_time: %w", err)
	}
	f := weatherFixture{
		ReferenceTime: ref, ReferenceText: input.ReferenceTime,
		Entries: make(map[fixtureTuple]fixtureEntry, len(input.Entries)),
	}
	for _, entry := range input.Entries {
		key := entry.fixtureTuple
		if _, exists := f.Entries[key]; exists {
			return weatherFixture{}, nil, fmt.Errorf("duplicate fixture tuple %s", key)
		}
		if err := entry.validate(); err != nil {
			return weatherFixture{}, nil, fmt.Errorf("fixture tuple %s: %w", key, err)
		}
		f.Entries[key] = entry
	}
	return f, raw, nil
}

func (e fixtureEntry) validate() error {
	if err := openmeteo.ValidateLocation(e.Latitude, e.Longitude, "celsius"); err != nil {
		return err
	}
	start, errStart := time.Parse(time.RFC3339, e.StartHour)
	end, errEnd := time.Parse(time.RFC3339, e.EndHour)
	if errStart != nil || errEnd != nil || start.Year() < 1 ||
		start.UTC().Format(time.RFC3339) != e.StartHour || end.UTC().Format(time.RFC3339) != e.EndHour ||
		!start.Equal(start.Truncate(time.Hour)) || !end.Equal(start.Add(time.Hour)) {
		return errors.New("fixture hours must be consecutive whole hours in canonical UTC RFC3339")
	}
	if e.Status < 200 || e.Status > 599 || e.Headers == nil || e.Body == nil || len(*e.Body) > openmeteo.MaxResponseBytes {
		return errors.New("fixture requires a final HTTP status, a header map and a bounded body string")
	}
	if (e.Status == http.StatusNoContent || e.Status == http.StatusNotModified) && *e.Body != "" {
		return errors.New("fixture HTTP status does not permit a response body")
	}
	seen := make(map[string]bool, len(e.Headers))
	for name, value := range e.Headers {
		key := http.CanonicalHeaderKey(name)
		if seen[key] || !fixtureHeaderToken(name) || strings.IndexFunc(value, invalidHeaderByte) >= 0 {
			return fmt.Errorf("invalid or duplicate fixture header %q", name)
		}
		seen[key] = true
		switch key {
		case "Content-Length", "Transfer-Encoding", "Trailer":
			return fmt.Errorf("fixture header %q controls HTTP message framing", name)
		}
	}
	return nil
}

func fixtureHeaderToken(name string) bool {
	if name == "" {
		return false
	}
	return strings.IndexFunc(name, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
			strings.ContainsRune("!#$%&'*+-.^`_|~", r))
	}) < 0
}

func invalidHeaderByte(r rune) bool { return r < ' ' && r != '\t' || r == 127 }

func (f weatherFixture) requireCandidates(plan normalizedPlan) error {
	for _, row := range plan.Candidates {
		key := fixtureTuple{Latitude: row.Latitude, Longitude: row.Longitude, StartHour: row.Start, EndHour: row.End}
		if _, ok := f.Entries[key]; row.Eligible && !ok {
			return fmt.Errorf("missing fixture tuple %s for %s", key, row.ParkID)
		}
	}
	return nil
}

// fixtureHandler has no forwarding path. An unexpected request is remembered
// separately from the response shown to the adapter.
type fixtureHandler struct {
	fixture weatherFixture
	units   string
	mu      sync.Mutex
	calls   int
	first   error
}

func newFixtureHandler(f weatherFixture, units string) *fixtureHandler {
	return &fixtureHandler{fixture: f, units: units}
}

func (h *fixtureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	entry, err := h.lookup(r)
	h.mu.Lock()
	h.calls++
	if err != nil && h.first == nil {
		h.first = err
	}
	h.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for key, value := range entry.Headers {
		w.Header().Set(key, value)
	}
	if _, sent := w.Header()["Content-Type"]; !sent {
		// Avoid synthesizing a Content-Type from fixture bytes.
		w.Header()["Content-Type"] = nil
	}
	if _, sent := w.Header()["Date"]; !sent {
		w.Header()["Date"] = nil
	}
	w.WriteHeader(entry.Status)
	if *entry.Body != "" {
		_, _ = w.Write([]byte(*entry.Body))
	}
}

func (h *fixtureHandler) lookup(r *http.Request) (fixtureEntry, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return fixtureEntry{}, fmt.Errorf("unexpected fixture query %q: %w", r.URL.RawQuery, err)
	}
	key := fixtureTuple{Latitude: q.Get("latitude"), Longitude: q.Get("longitude")}
	start, startErr := time.Parse(wireHour, q.Get("start_hour"))
	end, endErr := time.Parse(wireHour, q.Get("end_hour"))
	key.StartHour, key.EndHour = start.Format(time.RFC3339), end.Format(time.RFC3339)
	want := url.Values{
		"latitude": {key.Latitude}, "longitude": {key.Longitude},
		"start_hour": {start.Format(wireHour)}, "end_hour": {end.Format(wireHour)},
		"hourly": {"temperature_2m,weather_code"}, "temperature_unit": {h.units},
		"timezone": {"GMT"}, "timeformat": {"iso8601"}, "cell_selection": {"nearest"},
	}
	if r.Method != http.MethodGet || r.URL.EscapedPath() != fixturePath ||
		startErr != nil || endErr != nil || want.Encode() != q.Encode() {
		return fixtureEntry{}, fmt.Errorf("unexpected fixture request %s %s", r.Method, r.URL.RequestURI())
	}
	entry, ok := h.fixture.Entries[key]
	if !ok {
		return fixtureEntry{}, fmt.Errorf("unexpected fixture request for tuple %s", key)
	}
	return entry, nil
}

func (h *fixtureHandler) results() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls, h.first
}

func listenForFixture() (net.Listener, error) {
	return net.Listen("tcp4", "127.0.0.1:0")
}
