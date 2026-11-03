package openmeteo

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const maxNumberBytes = 32

var decimal = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,2})(\.[0-9]+)?([eE][+-]?[0-9]{1,2})?$`)

// boundedDecimal accepts exact JSON decimal numbers in the adapter's supported
// range. Only grid distance uses the rounded binary value. Payloads retain the
// original numeric spelling, including trailing zeroes.
func boundedDecimal(s string, low, high int64) (float64, bool) {
	if len(s) > maxNumberBytes || !decimal.MatchString(s) {
		return 0, false
	}
	n, ok := new(big.Rat).SetString(s)
	if !ok || n.Cmp(big.NewRat(low, 1)) < 0 || n.Cmp(big.NewRat(high, 1)) > 0 {
		return 0, false
	}
	x, _ := n.Float64()
	return x, true
}

// ValidateLocation accepts one WGS84 latitude/longitude pair in ordinary decimal
// notation and either celsius or fahrenheit. The exact input goes into the GET.
func ValidateLocation(latitude, longitude, units string) error {
	if _, ok := boundedDecimal(latitude, -90, 90); !ok || strings.ContainsAny(latitude, "eE") {
		return errors.New("weather latitude must be a decimal between -90 and 90")
	}
	if _, ok := boundedDecimal(longitude, -180, 180); !ok || strings.ContainsAny(longitude, "eE") {
		return errors.New("weather longitude must be a decimal between -180 and 180")
	}
	if units != "celsius" && units != "fahrenheit" {
		return errors.New("weather units must be celsius or fahrenheit")
	}
	return nil
}

type providerResponse struct {
	Error        jsontext.Value `json:"error"`
	Latitude     jsontext.Value `json:"latitude"`
	Longitude    jsontext.Value `json:"longitude"`
	Offset       *int           `json:"utc_offset_seconds"`
	Timezone     string         `json:"timezone"`
	Abbreviation *string        `json:"timezone_abbreviation"`
	Units        struct {
		Time        string `json:"time"`
		Temperature string `json:"temperature_2m"`
		Humidity    string `json:"relative_humidity_2m"`
	} `json:"current_units"`
	Current struct {
		Time        string         `json:"time"`
		Temperature jsontext.Value `json:"temperature_2m"`
		Humidity    jsontext.Value `json:"relative_humidity_2m"`
	} `json:"current"`
}

func utcZone(zone string) bool {
	return zone == "GMT" || zone == "UTC" || zone == "Etc/GMT" || zone == "Etc/UTC"
}

// Open-Meteo returns the model grid-cell centre, which may differ from the
// requested point. With cell_selection=nearest this adapter rejects a grid
// farther than 50 km. This is an application sanity bound, not a provider SLA.
func nearbyGrid(p effects.Prepared, lat, lon float64) bool {
	queryLat, _ := boundedDecimal(p.Request.Payload[Latitude].Text, -90, 90)
	queryLon, _ := boundedDecimal(p.Request.Payload[Longitude].Text, -180, 180)
	const radians = math.Pi / 180
	dlat, dlon := math.Sin((queryLat-lat)*radians/2), math.Sin((queryLon-lon)*radians/2)
	h := dlat*dlat + math.Cos(queryLat*radians)*math.Cos(lat*radians)*dlon*dlon
	maxSin := math.Sin(50.0 / (2 * 6371.0))
	return h <= maxSin*maxSin
}

func observation(p effects.Prepared, body string) (map[string]schema.Value, string) {
	var b providerResponse
	// JSON v2 matches exact field names and rejects duplicate keys, malformed
	// Unicode and trailing documents. Unknown additive metadata is harmless.
	if err := json.Unmarshal([]byte(body), &b); err != nil || (len(b.Error) > 0 && string(b.Error) != "false") {
		return nil, "protocol_invalid"
	}
	if b.Offset == nil || *b.Offset != 0 || !utcZone(b.Timezone) || (b.Abbreviation != nil && !utcZone(*b.Abbreviation)) || b.Units.Time != "iso8601" {
		return nil, "protocol_invalid"
	}
	// With timezone=GMT and offset=0, the provider's ISO minute is UTC.
	const minute = "2006-01-02T15:04"
	at, err := time.Parse(minute, b.Current.Time)
	if err != nil || at.Format(minute) != b.Current.Time || at.Year() == 0 {
		return nil, "protocol_invalid"
	}
	lat, latOK := boundedDecimal(string(b.Latitude), -90, 90)
	lon, lonOK := boundedDecimal(string(b.Longitude), -180, 180)
	if !latOK || !lonOK || !nearbyGrid(p, lat, lon) {
		return nil, "protocol_invalid"
	}
	// A generous supported range for terrestrial 2 m air temperature. Use
	// equivalent bounds in both units rather than changing or clipping values.
	unit, low, high := "°C", int64(-150), int64(150)
	if p.Request.Payload[Units].Text == "fahrenheit" {
		unit, low, high = "°F", -238, 302
	}
	if b.Units.Temperature != unit || b.Units.Humidity != "%" {
		return nil, "protocol_invalid"
	}
	temp, humidity := string(b.Current.Temperature), string(b.Current.Humidity)
	_, tempOK := boundedDecimal(temp, low, high)
	_, humidityOK := boundedDecimal(humidity, 0, 100)
	if (!tempOK && temp != "null") || (!humidityOK && humidity != "null") {
		return nil, "protocol_invalid"
	}
	if temp == "null" || humidity == "null" {
		return nil, "unavailable"
	}
	source := "Open-Meteo"
	if p.Endpoint != DefaultEndpoint {
		source = "controlled Open-Meteo fixture"
	}
	return map[string]schema.Value{
		Source: schema.Text(source), ObservedAt: schema.Text(at.Format(time.RFC3339)),
		GridLatitude: schema.Text(string(b.Latitude)), GridLongitude: schema.Text(string(b.Longitude)),
		Temperature: schema.Text(temp), TemperatureUnit: schema.Text(b.Units.Temperature),
		Humidity: schema.Text(humidity), HumidityUnit: schema.Text(b.Units.Humidity),
	}, "validated"
}
