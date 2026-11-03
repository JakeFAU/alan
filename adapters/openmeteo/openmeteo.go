// Package openmeteo reads two current, model-based weather measurements from
// Open-Meteo's non-commercial forecast API, or from an explicit loopback fixture.
package openmeteo

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const (
	DefaultEndpoint     = "https://api.open-meteo.com/v1/forecast"
	Protocol            = "open-meteo/current/v1"
	MaxResponseBytes    = 16 << 10
	ClientRequestHeader = "X-Client-Request-Id"
	// OperationHeader contains the base64url-encoded original ALAN operation ID
	// on loopback fixtures only. The namespace-scoped digest goes to both endpoints.
	OperationHeader = "X-Alan-Operation-Id"
	Attribution     = "Weather data by Open-Meteo (https://open-meteo.com/), CC BY 4.0 (https://creativecommons.org/licenses/by/4.0/). Values as supplied. Time formatted in UTC."

	Latitude        = "latitude"
	Longitude       = "longitude"
	Units           = "units"
	Source          = "source"
	ObservedAt      = "observed_at"
	GridLatitude    = "grid_latitude"
	GridLongitude   = "grid_longitude"
	Temperature     = "temperature"
	TemperatureUnit = "temperature_unit"
	Humidity        = "humidity"
	HumidityUnit    = "humidity_unit"
)

// InputSchema describes the exact authored location and temperature unit.
func InputSchema() schema.Record {
	return schema.Record{
		Latitude:  {Kind: "string", MaxBytes: maxNumberBytes},
		Longitude: {Kind: "string", MaxBytes: maxNumberBytes},
		Units:     {Kind: "string", MaxBytes: 10, Enum: []string{"celsius", "fahrenheit"}},
	}
}

// OutputSchema describes the validated current conditions. Measurements use
// decimal strings so ALAN can retain their exact provider representation.
func OutputSchema() schema.Record {
	return schema.Record{
		Source:          {Kind: "string", MaxBytes: 32},
		ObservedAt:      {Kind: "string", MaxBytes: 20},
		GridLatitude:    {Kind: "string", MaxBytes: maxNumberBytes},
		GridLongitude:   {Kind: "string", MaxBytes: maxNumberBytes},
		Temperature:     {Kind: "string", MaxBytes: maxNumberBytes},
		TemperatureUnit: {Kind: "string", MaxBytes: 3},
		Humidity:        {Kind: "string", MaxBytes: maxNumberBytes},
		HumidityUnit:    {Kind: "string", MaxBytes: 1},
	}
}

type Adapter struct{ endpoint string }

// New accepts the official HTTPS endpoint or the same path on a literal
// HTTP loopback IP and explicit port. No endpoint comes from authored input.
func New(endpoint string) (*Adapter, error) {
	if endpoint == DefaultEndpoint {
		return &Adapter{endpoint}, nil
	}
	if len(endpoint) <= 128 {
		host := strings.TrimSuffix(strings.TrimPrefix(endpoint, "http://"), "/v1/forecast")
		ap, err := netip.ParseAddrPort(host)
		if err == nil && ap.Port() != 0 && ap.Addr().IsLoopback() && ap.Addr().Zone() == "" &&
			endpoint == "http://"+ap.String()+"/v1/forecast" {
			return &Adapter{endpoint}, nil
		}
	}
	return nil, errors.New("weather endpoint must be the official HTTPS URL or /v1/forecast on a literal HTTP loopback address and port")
}

// ClientRequestID correlates provider contacts with a retained namespace-scoped
// operation key. It confers no provider deduplication or retry entitlement.
func ClientRequestID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "alan-" + hex.EncodeToString(sum[:])
}

func (a *Adapter) Prepare(r effects.Request) (effects.Prepared, error) {
	if a == nil || a.endpoint == "" || len(r.Assets) != 0 {
		return effects.Prepared{}, errors.New("weather requires an explicit endpoint and no corpus")
	}
	for _, id := range []string{r.ID, r.Key, r.Capability, r.Effect} {
		if len(id) == 0 || len(id) > 4096 || !utf8.ValidString(id) {
			return effects.Prepared{}, errors.New("invalid weather request identity")
		}
	}
	input, _ := InputSchema().Retain()
	output, _ := OutputSchema().Retain()
	if !reflect.DeepEqual(input, r.Input) || !reflect.DeepEqual(output, r.Output) || len(r.Payload) != len(input.Record) {
		return effects.Prepared{}, errors.New("weather schema mismatch")
	}
	for name, field := range input.Record {
		v := r.Payload[name]
		if len(v.Text) > field.MaxBytes || v != schema.Text(v.Text) {
			return effects.Prepared{}, errors.New("invalid weather request field")
		}
	}
	lat, lon, units := r.Payload[Latitude].Text, r.Payload[Longitude].Text, r.Payload[Units].Text
	if err := ValidateLocation(lat, lon, units); err != nil {
		return effects.Prepared{}, err
	}
	query := url.Values{
		"latitude":         {lat},
		"longitude":        {lon},
		"current":          {"temperature_2m,relative_humidity_2m"},
		"temperature_unit": {units},
		"timezone":         {"GMT"},
		"timeformat":       {"iso8601"},
		"cell_selection":   {"nearest"},
	}
	return effects.Freeze(effects.Prepared{
		Request: effects.Copy(r), Protocol: Protocol, Endpoint: a.endpoint, Wire: a.endpoint + "?" + query.Encode(), MaxResponseBytes: MaxResponseBytes,
	}), nil
}

func (a *Adapter) matches(p effects.Prepared) bool {
	expected, err := a.Prepare(p.Request)
	return err == nil && reflect.DeepEqual(expected, p)
}

func (a *Adapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	if !a.matches(p) {
		c.Disposition = "request_invalid"
		return c
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Wire, nil)
	if err != nil {
		return c
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", "ALAN-weather-agent/1 (+https://github.com/JakeFAU/alan)")
	req.Header.Set(ClientRequestHeader, ClientRequestID(p.Request.Key))
	if a.endpoint != DefaultEndpoint {
		req.Header.Set(OperationHeader, base64.RawURLEncoding.EncodeToString([]byte(p.Request.ID)))
	}
	// RoundTrip cannot follow redirects. A new HTTP/1 transport per operation
	// rules out net/http's retries on reused connections and HTTP/2 streams.
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	transport := &http.Transport{
		Protocols:              &protocols,
		DisableKeepAlives:      true,
		DisableCompression:     true,
		MaxResponseHeaderBytes: 16 << 10,
		DialContext:            (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
	}
	defer transport.CloseIdleConnections()
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 599 {
		return c // This GET requires a final HTTP response, with no protocol switch.
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return c // Partial or lost response leaves the original unresolved.
	}
	c.HTTPStatus = resp.StatusCode
	if len(body) > MaxResponseBytes {
		c.Disposition = "response_limit"
		body = body[:MaxResponseBytes]
	}
	if utf8.Valid(body) {
		c.Response = string(body)
	}
	if c.Disposition == "response_limit" {
		return c
	}
	if c.HTTPStatus != http.StatusOK {
		c.Disposition = "http_rejected"
		return c
	}
	if !weatherContent(resp.Header) {
		c.Disposition = "protocol_invalid"
		return c
	}
	c.Payload, c.Disposition = observation(p, c.Response)
	return c
}

func weatherContent(header http.Header) bool {
	if len(header.Values("Content-Type")) != 1 || len(header.Values("Content-Range")) != 0 {
		return false
	}
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	charset := params["charset"]
	if err != nil || mediaType != "application/json" || (charset != "" && !strings.EqualFold(charset, "utf-8")) {
		return false
	}
	encodings := header.Values("Content-Encoding")
	return len(encodings) == 0 || (len(encodings) == 1 && strings.EqualFold(encodings[0], "identity"))
}

// VerifyResponseEvidence checks a native result against the frozen request,
// HTTP status and retained body. Response headers are checked during Execute.
// A protocol refusal may therefore have a readable body and unsuitable headers.
func VerifyResponseEvidence(p effects.Prepared, c effects.Candidate) error {
	a, err := New(p.Endpoint)
	if err != nil || !a.matches(p) || c.ID != p.Request.ID || c.Key != p.Request.Key ||
		c.HTTPStatus < 200 || c.HTTPStatus > 599 || len(c.Response) > MaxResponseBytes || !utf8.ValidString(c.Response) ||
		c.Usage != nil || c.RequestID != "" || c.ResponseID != "" || c.ReturnedModel != "" {
		return errors.New("weather evidence differs from the prepared request or expected response")
	}
	if c.Disposition != "validated" && c.Payload != nil {
		return errors.New("unsuccessful weather response contains measurements")
	}
	if (c.HTTPStatus == http.StatusNoContent || c.HTTPStatus == http.StatusNotModified) && (c.Response != "" || c.Disposition != "http_rejected") {
		return errors.New("body evidence contradicts a bodyless HTTP status")
	}
	switch c.Disposition {
	case "response_limit":
		// Non-UTF-8 and truncated multibyte prefixes are not retained as text.
		if len(c.Response) != MaxResponseBytes && c.Response != "" {
			return errors.New("weather response does not reach its stated limit")
		}
		return nil
	case "http_rejected":
		if c.HTTPStatus == http.StatusOK {
			return errors.New("HTTP 200 contradicts the weather refusal")
		}
		return nil
	}
	if c.HTTPStatus != http.StatusOK {
		return errors.New("weather disposition requires HTTP 200")
	}
	if c.Disposition == "protocol_invalid" {
		return nil
	}
	payload, disposition := observation(p, c.Response)
	if disposition != c.Disposition || !reflect.DeepEqual(payload, c.Payload) {
		return errors.New("weather fields differ from the retained provider body")
	}
	return nil
}
