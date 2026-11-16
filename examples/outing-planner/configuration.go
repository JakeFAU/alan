package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JakeFAU/alan/adapters/openmeteo"
)

const (
	maxParks         = 4
	maxSlots         = 4
	maxCandidates    = maxParks * maxSlots
	maxWindows       = 16
	maxConfiguration = 48 << 10
	maxParkName      = 128
	dateLayout       = "2006-01-02"
)

type configuration struct {
	Version             int          `json:"version"`
	ReferenceTime       string       `json:"reference_time"`
	Date                string       `json:"date"`
	Timezone            string       `json:"timezone"`
	Units               string       `json:"units"`
	Endpoint            string       `json:"endpoint"`
	MaxTravelMinutes    int          `json:"max_travel_minutes"`
	AllowedWeatherCodes []int64      `json:"allowed_weather_codes"`
	Availability        []timeWindow `json:"availability"`
	Slots               []string     `json:"slots"`
	Parks               []park       `json:"parks"`
}

type timeWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type park struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Latitude      string       `json:"latitude"`
	Longitude     string       `json:"longitude"`
	TravelMinutes int          `json:"travel_minutes"`
	Opening       []timeWindow `json:"opening"`
}

type normalizedPlan struct {
	Date       string
	Candidates []candidate
}

type candidate struct {
	ID            string `json:"id"`
	ParkID        string `json:"park_id"`
	Name          string `json:"name"`
	Latitude      string `json:"latitude"`
	Longitude     string `json:"longitude"`
	Start         string `json:"start_hour"`
	End           string `json:"end_hour"`
	TravelMinutes int    `json:"travel_minutes"`
	Eligible      bool   `json:"eligible"`
	Reason        string `json:"reason"`
}

func readConfiguration(path string, now time.Time) (configuration, error) {
	raw, err := readBoundedFile(path, maxConfiguration)
	if err != nil {
		return configuration{}, err
	}
	return configurationAt(raw, now.UTC().Format(time.RFC3339Nano))
}

func configurationAt(raw []byte, reference string) (configuration, error) {
	c, err := decodeConfiguration(raw)
	if err != nil {
		return c, err
	}
	if c.ReferenceTime == "" {
		c.ReferenceTime = reference
	}
	_, err = c.normalize()
	return c, err
}

func readBoundedFile(path string, maxBytes int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("%s exceeds the %d-byte input limit", path, maxBytes)
	}
	return raw, nil
}

func decodeConfiguration(raw []byte) (configuration, error) {
	var c configuration
	if len(raw) > maxConfiguration {
		return c, errors.New("outing configuration exceeds its byte bound")
	}
	// Decode required numbers as pointers to distinguish explicit zero from
	// an omitted or null value before constructing the retained inputs.
	type configurationFields configuration
	type parkFields park
	type parkInput struct {
		parkFields
		TravelMinutes *int `json:"travel_minutes"`
	}
	input := struct {
		*configurationFields
		MaxTravelMinutes *int        `json:"max_travel_minutes"`
		Parks            []parkInput `json:"parks"`
	}{configurationFields: (*configurationFields)(&c)}
	// JSON v2 rejects duplicate names, trailing documents and malformed UTF-8.
	if err := jsonv2.Unmarshal(raw, &input, jsonv2.RejectUnknownMembers(true)); err != nil {
		return c, fmt.Errorf("outing configuration: %w", err)
	}
	if input.MaxTravelMinutes == nil {
		return c, errors.New("outing configuration requires max_travel_minutes")
	}
	c.MaxTravelMinutes = *input.MaxTravelMinutes
	if input.Parks != nil {
		c.Parks = make([]park, 0, len(input.Parks))
	}
	for i, p := range input.Parks {
		if p.TravelMinutes == nil {
			return c, fmt.Errorf("outing configuration requires parks[%d].travel_minutes", i)
		}
		row := park(p.parkFields)
		row.TravelMinutes = *p.TravelMinutes
		c.Parks = append(c.Parks, row)
	}
	return c, nil
}

func (c configuration) allowedCodes() []int64 {
	if c.AllowedWeatherCodes == nil {
		return []int64{0, 1, 2, 3}
	}
	return c.AllowedWeatherCodes
}

type interval struct{ start, end time.Time }

var (
	parkIDPattern       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,31}$`)
	clockPattern        = regexp.MustCompile(`^[0-9]{2}:[0-9]{2}$`)
	explicitTimePattern = regexp.MustCompile(
		`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`,
	)
)

func (c configuration) normalize() (normalizedPlan, error) {
	if c.Version != 1 {
		return normalizedPlan{}, errors.New("outing configuration requires version 1")
	}
	if len(c.Parks) > maxParks || len(c.Slots) > maxSlots {
		return normalizedPlan{}, errors.New("at most four parks and four slots are supported")
	}
	if !validTravelMinutes(c.MaxTravelMinutes) {
		return normalizedPlan{}, errors.New("maximum travel minutes must be a nonnegative representable duration")
	}
	if c.Units != "celsius" && c.Units != "fahrenheit" {
		return normalizedPlan{}, errors.New("units must be celsius or fahrenheit")
	}
	if _, err := openmeteo.NewHourly(c.Endpoint); err != nil {
		return normalizedPlan{}, err
	}
	codes := map[int64]bool{}
	for _, code := range c.allowedCodes() {
		if codes[code] || !openmeteo.ValidWeatherCode(code) {
			return normalizedPlan{}, fmt.Errorf("duplicate or unsupported weather code %d", code)
		}
		codes[code] = true
	}
	ref, err := parseExplicitTime(c.ReferenceTime)
	if err != nil {
		return normalizedPlan{}, fmt.Errorf("reference_time: %w", err)
	}
	zone, err := time.LoadLocation(c.Timezone)
	if err != nil || c.Timezone == "" || c.Timezone == "Local" {
		return normalizedPlan{}, fmt.Errorf("expected an IANA timezone, got %q", c.Timezone)
	}
	date := c.Date
	if date == "tomorrow" {
		// Use the local calendar date, then advance its Gregorian label in UTC.
		// Adding 24 elapsed hours in the named zone can cross zero or two dates.
		y, m, d := ref.In(zone).Date()
		date = time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC).Format(dateLayout)
	}
	day, err := time.Parse(dateLayout, date)
	if err != nil || day.Year() < 1 || day.Format(dateLayout) != date {
		return normalizedPlan{}, fmt.Errorf("invalid target date %q", date)
	}

	available, err := resolveWindows(c.Availability, date, zone)
	if err != nil {
		return normalizedPlan{}, fmt.Errorf("availability: %w", err)
	}
	slots := make([]time.Time, len(c.Slots))
	seenSlots := map[time.Time]bool{}
	for i, input := range c.Slots {
		start, err := resolveClock(input, date, zone)
		if err != nil {
			return normalizedPlan{}, fmt.Errorf("slot %q: %w", input, err)
		}
		wholeHour := start.Equal(start.Truncate(time.Hour))
		if !wholeHour || start.Year() < 1 || start.Add(time.Hour).Year() > 9999 ||
			start.In(zone).Format(dateLayout) != date || seenSlots[start] {
			return normalizedPlan{}, fmt.Errorf("slot %q must be a unique whole UTC hour starting on %s", input, date)
		}
		slots[i] = start
		seenSlots[start] = true
	}

	rows := make([]candidate, 0, len(c.Parks)*len(slots))
	seenParks := map[string]bool{}
	for _, p := range c.Parks {
		if !parkIDPattern.MatchString(p.ID) || seenParks[p.ID] {
			return normalizedPlan{}, fmt.Errorf("invalid or duplicate park ID %q", p.ID)
		}
		seenParks[p.ID] = true
		if strings.TrimSpace(p.Name) == "" || len(p.Name) > maxParkName || !utf8.ValidString(p.Name) ||
			!validTravelMinutes(p.TravelMinutes) {
			return normalizedPlan{}, fmt.Errorf("park %q has an invalid name or travel duration", p.ID)
		}
		if err := openmeteo.ValidateLocation(p.Latitude, p.Longitude, c.Units); err != nil {
			return normalizedPlan{}, fmt.Errorf("park %q: %w", p.ID, err)
		}
		open, err := resolveWindows(p.Opening, date, zone)
		if err != nil {
			return normalizedPlan{}, fmt.Errorf("park %q opening: %w", p.ID, err)
		}
		for _, start := range slots {
			end := start.Add(time.Hour)
			travel := time.Duration(p.TravelMinutes) * time.Minute
			row := candidate{
				ParkID: p.ID, Name: p.Name, Latitude: p.Latitude, Longitude: p.Longitude,
				Start: start.Format(time.RFC3339), End: end.Format(time.RFC3339), TravelMinutes: p.TravelMinutes,
			}
			row.ID = digestText(strings.Join([]string{row.ParkID, row.Latitude, row.Longitude, row.Start, row.End}, "|"))
			switch {
			case p.TravelMinutes > c.MaxTravelMinutes:
				row.Reason = "travel_limit"
			case !covers(open, start, end):
				row.Reason = "outside_opening_hours"
			case !covers(available, start.Add(-travel), end.Add(travel)):
				row.Reason = "outside_availability_with_travel"
			default:
				row.Eligible, row.Reason = true, "eligible"
			}
			rows = append(rows, row)
		}
	}
	slices.SortFunc(rows, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(a.TravelMinutes, b.TravelMinutes), cmp.Compare(a.Start, b.Start), cmp.Compare(a.ParkID, b.ParkID))
	})
	return normalizedPlan{Date: date, Candidates: rows}, nil
}

func validTravelMinutes(minutes int) bool {
	return minutes >= 0 && int64(minutes) <= math.MaxInt64/int64(time.Minute)
}

func parseExplicitTime(text string) (time.Time, error) {
	if !explicitTimePattern.MatchString(text) || strings.HasSuffix(text, "-00:00") {
		return time.Time{}, fmt.Errorf("expected RFC3339 with a known, valid offset, got %q", text)
	}
	t, err := time.Parse(time.RFC3339Nano, text)
	if err != nil || t.Year() < 1 {
		return time.Time{}, fmt.Errorf("invalid RFC3339 time %q", text)
	}
	return t, nil
}

func resolveClock(text, date string, zone *time.Location) (time.Time, error) {
	if !clockPattern.MatchString(text) {
		t, err := parseExplicitTime(text)
		if err != nil {
			return time.Time{}, err
		}
		_, statedOffset := t.Zone()
		_, localOffset := t.In(zone).Zone()
		if statedOffset != localOffset {
			return time.Time{}, fmt.Errorf("%q has an offset inconsistent with %s at that instant", text, zone)
		}
		return t.UTC(), nil
	}
	// Interpret the wall-clock fields as a comparison value. Check every
	// possible zone offset around it against actual UTC-to-local conversion.
	wall, err := time.Parse(dateLayout+" 15:04", date+" "+text)
	if err != nil {
		return time.Time{}, err
	}
	var matches []time.Time
	const searchRadius = 48 * time.Hour
	offsets := map[int]bool{}
	end := wall.Add(searchRadius)
	for cursor := wall.Add(-searchRadius); !cursor.After(end); {
		local := cursor.In(zone)
		_, offset := local.Zone()
		if !offsets[offset] {
			offsets[offset] = true
			instant := wall.Add(-time.Duration(offset) * time.Second)
			if instant.In(zone).Format(dateLayout+" 15:04:05") == wall.Format(dateLayout+" 15:04:05") {
				matches = append(matches, instant)
			}
		}
		_, next := local.ZoneBounds()
		if next.IsZero() || !next.After(cursor) {
			break
		}
		cursor = next
	}
	if len(matches) != 1 {
		return time.Time{}, fmt.Errorf("clock time %s %s is ambiguous or nonexistent in %s", date, text, zone)
	}
	return matches[0], nil
}

func resolveWindows(inputs []timeWindow, date string, zone *time.Location) ([]interval, error) {
	if len(inputs) > maxWindows {
		return nil, fmt.Errorf("at most %d intervals are supported", maxWindows)
	}
	windows := make([]interval, 0, len(inputs))
	for _, input := range inputs {
		start, err := resolveClock(input.Start, date, zone)
		if err != nil {
			return nil, err
		}
		end, err := resolveClock(input.End, date, zone)
		if err != nil {
			return nil, err
		}
		if !end.After(start) {
			return nil, fmt.Errorf("interval %q to %q must have positive duration", input.Start, input.End)
		}
		windows = append(windows, interval{start: start, end: end})
	}
	slices.SortFunc(windows, func(a, b interval) int { return a.start.Compare(b.start) })
	merged := make([]interval, 0, len(windows))
	for _, window := range windows {
		if len(merged) == 0 || window.start.After(merged[len(merged)-1].end) {
			merged = append(merged, window)
			continue
		}
		last := &merged[len(merged)-1]
		if window.end.After(last.end) {
			last.end = window.end
		}
	}
	return merged, nil
}

func covers(windows []interval, start, end time.Time) bool {
	return slices.ContainsFunc(windows, func(w interval) bool {
		return !start.Before(w.start) && !end.After(w.end)
	})
}

func (c configuration) serialized() ([]byte, error) {
	return json.Marshal(c)
}

func digestText(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}
