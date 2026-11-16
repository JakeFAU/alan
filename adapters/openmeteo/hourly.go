package openmeteo

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const (
	HourlyProtocol = "open-meteo/hourly/v1"

	StartHour        = "start_hour"
	EndHour          = "end_hour"
	StartWeatherCode = "start_weather_code"
	EndWeatherCode   = "end_weather_code"
	StartTemperature = "start_temperature"
	EndTemperature   = "end_temperature"

	isoMinute = "2006-01-02T15:04"
)

// HourlyInputSchema describes one location, temperature unit and two UTC hours.
func HourlyInputSchema() schema.Record {
	input := InputSchema()
	input[StartHour] = schema.Field{Kind: "string", MaxBytes: 20}
	input[EndHour] = schema.Field{Kind: "string", MaxBytes: 20}
	return input
}

// HourlyOutputSchema describes the grid and the two forecast samples.
// Temperatures retain their original decimal representation.
func HourlyOutputSchema() schema.Record {
	return schema.Record{
		Source:           {Kind: "string", MaxBytes: 32},
		GridLatitude:     {Kind: "string", MaxBytes: maxNumberBytes},
		GridLongitude:    {Kind: "string", MaxBytes: maxNumberBytes},
		TemperatureUnit:  {Kind: "string", MaxBytes: 3},
		StartHour:        {Kind: "string", MaxBytes: 20},
		EndHour:          {Kind: "string", MaxBytes: 20},
		StartTemperature: {Kind: "string", MaxBytes: maxNumberBytes},
		EndTemperature:   {Kind: "string", MaxBytes: maxNumberBytes},
		StartWeatherCode: {Kind: "int"},
		EndWeatherCode:   {Kind: "int"},
	}
}

// HourlyAdapter reads two consecutive hourly forecasts for one location.
type HourlyAdapter struct{ endpoint string }

var _ effects.Adapter = (*HourlyAdapter)(nil)

// NewHourly accepts the same official and literal loopback endpoints as New.
func NewHourly(endpoint string) (*HourlyAdapter, error) {
	if _, err := New(endpoint); err != nil {
		return nil, err
	}
	return &HourlyAdapter{endpoint: endpoint}, nil
}

func parseHour(text string) (time.Time, bool) {
	hour, err := time.Parse(time.RFC3339, text)
	return hour, err == nil && hour.Year() != 0 && hour.UTC().Format(time.RFC3339) == text && hour.Equal(hour.Truncate(time.Hour))
}

func (a *HourlyAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	if a == nil || a.endpoint == "" || len(r.Assets) != 0 {
		return effects.Prepared{}, errors.New("weather requires an explicit endpoint and no corpus")
	}
	for _, id := range []string{r.ID, r.Key, r.Capability, r.Effect} {
		if len(id) == 0 || len(id) > 4096 || !utf8.ValidString(id) {
			return effects.Prepared{}, errors.New("invalid weather request identity")
		}
	}
	input, _ := HourlyInputSchema().Retain()
	output, _ := HourlyOutputSchema().Retain()
	if !reflect.DeepEqual(input, r.Input) || !reflect.DeepEqual(output, r.Output) {
		return effects.Prepared{}, errors.New("weather schema mismatch")
	}
	if err := input.Record.Validate(r.Payload); err != nil {
		return effects.Prepared{}, err
	}
	lat, lon, units := r.Payload[Latitude].Text, r.Payload[Longitude].Text, r.Payload[Units].Text
	if err := ValidateLocation(lat, lon, units); err != nil {
		return effects.Prepared{}, err
	}
	start, startOK := parseHour(r.Payload[StartHour].Text)
	end, endOK := parseHour(r.Payload[EndHour].Text)
	if !startOK || !endOK || !end.Equal(start.Add(time.Hour)) {
		return effects.Prepared{}, errors.New("forecast requires two consecutive whole hours in canonical UTC RFC3339")
	}
	query := url.Values{
		"latitude":         {lat},
		"longitude":        {lon},
		"hourly":           {"temperature_2m,weather_code"},
		"temperature_unit": {units},
		"start_hour":       {start.Format(isoMinute)},
		"end_hour":         {end.Format(isoMinute)},
		"timezone":         {"GMT"},
		"timeformat":       {"iso8601"},
		"cell_selection":   {"nearest"},
	}
	return effects.Freeze(effects.Prepared{
		Request: effects.Copy(r), Protocol: HourlyProtocol, Endpoint: a.endpoint,
		Wire: a.endpoint + "?" + query.Encode(), MaxResponseBytes: MaxResponseBytes,
	}), nil
}

func (a *HourlyAdapter) matches(p effects.Prepared) bool {
	expected, err := a.Prepare(p.Request)
	return err == nil && reflect.DeepEqual(expected, p)
}

func (a *HourlyAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	return executeWeather(ctx, p, a.matches(p), hourlyObservation)
}

// VerifyHourlyResponseEvidence checks a native result against the frozen hourly
// request, HTTP status and retained body. Execute checks the response headers.
func VerifyHourlyResponseEvidence(p effects.Prepared, c effects.Candidate) error {
	a, err := NewHourly(p.Endpoint)
	return verifyWeatherResponseEvidence(p, c, err == nil && a.matches(p), hourlyObservation)
}
