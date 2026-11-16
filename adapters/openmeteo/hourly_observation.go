package openmeteo

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

// ValidWeatherCode reports whether code is in Open-Meteo's documented WMO subset.
func ValidWeatherCode(code int64) bool {
	switch code {
	case 0, 1, 2, 3, 45, 48, 51, 53, 55, 56, 57, 61, 63, 65, 66, 67,
		71, 73, 75, 77, 80, 81, 82, 85, 86, 95, 96, 97, 99:
		return true
	default:
		return false
	}
}

type hourlyProviderResponse struct {
	Error        jsontext.Value `json:"error"`
	Latitude     jsontext.Value `json:"latitude"`
	Longitude    jsontext.Value `json:"longitude"`
	Offset       *int           `json:"utc_offset_seconds"`
	Timezone     string         `json:"timezone"`
	Abbreviation *string        `json:"timezone_abbreviation"`
	Units        struct {
		Time        string `json:"time"`
		Temperature string `json:"temperature_2m"`
		WeatherCode string `json:"weather_code"`
	} `json:"hourly_units"`
	Hourly struct {
		Time        []string         `json:"time"`
		Temperature []jsontext.Value `json:"temperature_2m"`
		WeatherCode []*int64         `json:"weather_code"`
	} `json:"hourly"`
}

func hourlyObservation(p effects.Prepared, body string) (map[string]schema.Value, string) {
	var b hourlyProviderResponse
	// JSON v2 rejects duplicate keys, malformed Unicode and trailing documents.
	if err := json.Unmarshal([]byte(body), &b); err != nil || (len(b.Error) > 0 && string(b.Error) != "false") {
		return nil, "protocol_invalid"
	}
	if b.Offset == nil || *b.Offset != 0 || !utcZone(b.Timezone) || (b.Abbreviation != nil && !utcZone(*b.Abbreviation)) ||
		b.Units.Time != "iso8601" || b.Units.WeatherCode != "wmo code" {
		return nil, "protocol_invalid"
	}
	lat, latOK := boundedDecimal(string(b.Latitude), -90, 90)
	lon, lonOK := boundedDecimal(string(b.Longitude), -180, 180)
	if !latOK || !lonOK || !nearbyGrid(p, lat, lon) {
		return nil, "protocol_invalid"
	}
	unit, low, high := "°C", int64(-150), int64(150)
	if p.Request.Payload[Units].Text == "fahrenheit" {
		unit, low, high = "°F", -238, 302
	}
	times, temps, codes := b.Hourly.Time, b.Hourly.Temperature, b.Hourly.WeatherCode
	if b.Units.Temperature != unit || len(times) != 2 || len(temps) != 2 || len(codes) != 2 {
		return nil, "protocol_invalid"
	}
	// Preparation has checked both canonical UTC hours. At offset zero, the
	// provider's ISO minute has the same date and hour without seconds or zone.
	start, end := p.Request.Payload[StartHour].Text, p.Request.Payload[EndHour].Text
	startMinute, endMinute := strings.TrimSuffix(start, ":00Z"), strings.TrimSuffix(end, ":00Z")
	first, last := 0, 1
	if times[0] == endMinute {
		first, last = 1, 0
	}
	if times[first] != startMinute || times[last] != endMinute {
		return nil, "protocol_invalid"
	}
	unavailable := false
	for i, t := range temps {
		temp := string(t)
		if _, ok := boundedDecimal(temp, low, high); !ok && temp != "null" {
			return nil, "protocol_invalid"
		}
		if temp == "null" || codes[i] == nil || !ValidWeatherCode(*codes[i]) {
			unavailable = true
		}
	}
	if unavailable {
		return nil, "unavailable"
	}
	source := "Open-Meteo"
	if p.Endpoint != DefaultEndpoint {
		source = "controlled Open-Meteo fixture"
	}
	return map[string]schema.Value{
		Source: schema.Text(source), GridLatitude: schema.Text(string(b.Latitude)), GridLongitude: schema.Text(string(b.Longitude)),
		TemperatureUnit: schema.Text(unit), StartHour: schema.Text(start), EndHour: schema.Text(end),
		StartTemperature: schema.Text(string(temps[first])), EndTemperature: schema.Text(string(temps[last])),
		StartWeatherCode: schema.Int(*codes[first]), EndWeatherCode: schema.Int(*codes[last]),
	}, "validated"
}
