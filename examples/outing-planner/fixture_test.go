package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlannerFixtureRejectsUnexpectedRequests(t *testing.T) {
	f, _, err := readFixture("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	const query = "?latitude=40.8&longitude=-74&start_hour=2026-10-11T15%3A00&end_hour=2026-10-11T16%3A00" +
		"&hourly=temperature_2m%2Cweather_code&temperature_unit=celsius&timezone=GMT&timeformat=iso8601&cell_selection=nearest"
	for _, tc := range []struct{ method, request, want string }{
		{http.MethodGet, "/other" + query, "unexpected fixture request"},
		{http.MethodPost, fixturePath + query, "unexpected fixture request"},
		{http.MethodGet, fixturePath + query + "&latitude=40.8", "unexpected fixture request"},
		{http.MethodGet, fixturePath + strings.Replace(query, "latitude=40.8", "latitude=40.9", 1), "(40.9, -74,"},
	} {
		h := newFixtureHandler(f, "celsius")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.request, nil))
		count, first := h.results()
		if w.Code != http.StatusBadRequest || count != 1 || first == nil || !strings.Contains(first.Error(), tc.want) {
			t.Fatalf("%s %s: status %d, %d calls, %v", tc.method, tc.request, w.Code, count, first)
		}
	}
	h := newFixtureHandler(f, "celsius")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fixturePath+query, nil))
	result := w.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	entry := f.Entries[fixtureTuple{
		Latitude: "40.8", Longitude: "-74", StartHour: "2026-10-11T15:00:00Z", EndHour: "2026-10-11T16:00:00Z",
	}]
	if err != nil || result.StatusCode != entry.Status || result.Header.Get("Content-Type") != entry.Headers["Content-Type"] ||
		string(body) != *entry.Body {
		t.Fatalf("fixture changed its native response: status %d, headers %v, body %q, %v", result.StatusCode, result.Header, body, err)
	}
}
