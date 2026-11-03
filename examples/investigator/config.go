package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

const officialEndpoint = "https://api.openai.com/v1/responses"
const maxProviderResponseBytes = 128 << 10 // Fits the default retained observation envelope.
const controlledCredential = "alan-controlled-credential"

type providerConfig struct {
	Model            string `json:"model"`
	Endpoint         string `json:"endpoint"`
	MaxOutputTokens  int    `json:"max_output_tokens"`
	MaxResponseBytes int    `json:"max_response_bytes"`
}

type spendingLimits struct {
	MaxEffects           int64 `json:"max_effects"`
	MaxInFlight          int64 `json:"max_in_flight"`
	MaxTotalOutputTokens int64 `json:"max_total_output_tokens"`
}

// deployment is retained in the original parent input. Credentials are supplied
// only to the local adapter when a runner explicitly enables the live endpoint.
type deployment struct {
	Provider providerConfig `json:"provider"`
	Limits   spendingLimits `json:"limits"`
}

type documentSpec struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type investigationSpec struct {
	deployment
	Question  string         `json:"question"`
	Queries   []string       `json:"queries"`
	Documents []documentSpec `json:"documents"`
}

func (cfg providerConfig) adapter(key string) (*openai.Adapter, error) {
	return openai.New(openai.Config{
		APIKey: key, Model: cfg.Model, Endpoint: cfg.Endpoint,
		MaxOutputTokens: cfg.MaxOutputTokens, MaxResponseBytes: cfg.MaxResponseBytes,
	})
}

func (d deployment) validate() error {
	p := d.Provider
	if strings.TrimSpace(p.Model) == "" || len(p.Model) > 256 || len(p.Endpoint) > 512 {
		return errors.New("specify a nonempty model (up to 256 bytes) and a bounded endpoint")
	}
	if p.MaxResponseBytes > maxProviderResponseBytes {
		return fmt.Errorf("max_response_bytes exceeds the investigator limit of %d", maxProviderResponseBytes)
	}
	if p.Endpoint != officialEndpoint {
		u, err := url.Parse(p.Endpoint)
		if err != nil || u.Scheme != "http" || u.EscapedPath() != "/v1/responses" || u.ForceQuery {
			return errors.New("choose the official Responses endpoint or a literal loopback HTTP /v1/responses endpoint")
		}
	}
	// The shared adapter also checks the endpoint authority, response bound and
	// per-request output token bound. Construction performs no network I/O.
	if _, err := p.adapter(controlledCredential); err != nil {
		return err
	}
	l := d.Limits
	if l.MaxEffects < 1 || l.MaxEffects > 4 || l.MaxInFlight < 1 || l.MaxInFlight > 2 {
		return errors.New("max_effects must be 1..4 and max_in_flight 1..2")
	}
	if l.MaxTotalOutputTokens < 2*int64(p.MaxOutputTokens) {
		return errors.New("max_total_output_tokens must cover both workers' per-call output allowances")
	}
	return nil
}

func (d deployment) resources() runtime.ResourceLimits {
	return runtime.ResourceLimits{MaxInvocations: 2, MaxCreatedRuns: 2, MaxDepth: 1,
		MaxEffects: d.Limits.MaxEffects, MaxInFlight: d.Limits.MaxInFlight}
}

// The standard decoder accepts repeated fields and case-insensitive aliases.
// Spending limits must resolve to a single explicit value at every depth.
func checkConfigKeys(dec *json.Decoder, depth int) error {
	if depth > 8 {
		return errors.New("configuration is too deeply nested")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	seen := make(map[string]bool)
	for dec.More() {
		if delim == '{' {
			k, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := k.(string)
			if !ok || key == "" || strings.ContainsFunc(key, func(r rune) bool {
				return r != '_' && (r < 'a' || r > 'z')
			}) {
				return fmt.Errorf("configuration keys must use lower-case ASCII: %q", key)
			}
			if seen[key] {
				return fmt.Errorf("repeated configuration key %q", key)
			}
			seen[key] = true
		}
		if err := checkConfigKeys(dec, depth+1); err != nil {
			return err
		}
	}
	_, err = dec.Token()
	return err
}

func decodeSettings(data []byte, dst any) error {
	if !utf8.Valid(data) {
		return errors.New("configuration must be UTF-8")
	}
	if err := checkConfigKeys(json.NewDecoder(bytes.NewReader(data)), 0); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON configuration object")
	}
	return nil
}

func loadSpec(path string) (investigationSpec, []effects.Asset, error) {
	var spec investigationSpec
	f, err := os.Open(path)
	if err != nil {
		return spec, nil, err
	}
	defer f.Close()
	const maxSpecBytes = 64 << 10
	b, err := io.ReadAll(io.LimitReader(f, maxSpecBytes+1))
	if err != nil {
		return spec, nil, err
	}
	if len(b) > maxSpecBytes {
		return spec, nil, errors.New("investigation specification exceeds 64 KiB")
	}
	if err := decodeSettings(b, &spec); err != nil {
		return spec, nil, fmt.Errorf("investigation specification: %w", err)
	}
	if spec.Provider.Endpoint == "" {
		spec.Provider.Endpoint = officialEndpoint
	}
	if err := spec.deployment.validate(); err != nil {
		return spec, nil, err
	}
	if len(spec.Question) > 4096 || strings.TrimSpace(spec.Question) == "" || len(spec.Queries) != 2 {
		return spec, nil, errors.New("supply a nonempty question up to 4096 bytes and exactly two lookup queries")
	}
	for _, query := range spec.Queries {
		tokens := strings.FieldsFunc(query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		if len(query) > 512 || len(tokens) == 0 || len(tokens) > 32 {
			return spec, nil, errors.New("each lookup query must contain 1..32 tokens and fit in 512 bytes")
		}
	}
	if len(spec.Documents) == 0 || len(spec.Documents) > effects.MaxAssets {
		return spec, nil, fmt.Errorf("supply 1..%d documents", effects.MaxAssets)
	}
	assets := make([]effects.Asset, 0, len(spec.Documents))
	total := 0
	for _, doc := range spec.Documents {
		src := doc.Path
		if !filepath.IsAbs(src) {
			src = filepath.Join(filepath.Dir(path), src)
		}
		asset, err := effects.Snapshot(doc.ID, src)
		if err != nil {
			return spec, nil, fmt.Errorf("snapshot document %q: %w", doc.ID, err)
		}
		asset.Source = doc.Path
		total += len(asset.Content)
		if total > effects.MaxCorpusBytes {
			return spec, nil, errors.New("document set exceeds the retained corpus byte limit")
		}
		assets = append(assets, asset)
	}
	return spec, assets, effects.CheckAssets(assets)
}
