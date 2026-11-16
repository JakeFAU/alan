package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlannerRequiresExplicitTravelMinutes(t *testing.T) {
	c := fixtureConfiguration()
	c.MaxTravelMinutes = 0
	c.Parks = c.Parks[:1]
	c.Parks[0].TravelMinutes = 0
	raw, err := c.serialized()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, old, replacement, required string
	}{
		{name: "explicit zero"},
		{name: "omitted maximum", old: `"max_travel_minutes":0,`, required: "max_travel_minutes"},
		{name: "null maximum", old: `"max_travel_minutes":0`,
			replacement: `"max_travel_minutes":null`, required: "max_travel_minutes"},
		{name: "omitted park travel", old: `"travel_minutes":0,`, required: "travel_minutes"},
		{name: "null park travel", old: `"travel_minutes":0`,
			replacement: `"travel_minutes":null`, required: "travel_minutes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := string(raw)
			if tc.old != "" {
				data = strings.Replace(data, tc.old, tc.replacement, 1)
				if data == string(raw) {
					t.Fatal("fixture did not exercise the missing travel input")
				}
			}
			path := filepath.Join(t.TempDir(), "outing.json")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := readConfiguration(path, time.Time{})
			if tc.required != "" {
				if err == nil || !strings.Contains(err.Error(), tc.required) {
					t.Fatalf("expected error for %s, got %v", tc.required, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			plan, err := got.normalize()
			if err != nil || got.MaxTravelMinutes != 0 || len(plan.Candidates) != 1 ||
				!plan.Candidates[0].Eligible || plan.Candidates[0].TravelMinutes != 0 {
				t.Fatalf("explicit zero travel did not yield an eligible walk: %+v %v", plan, err)
			}
		})
	}
}
