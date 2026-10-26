// Package effects defines one-request deployment capabilities. Implementations are
// trusted infrastructure, not programs or model-selected callbacks.
package effects

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/schema"
	"io"
	"os"
	"unicode/utf8"
)

const MaxAssets = 64
const MaxAssetBytes = 1 << 20
const MaxCorpusBytes = 4 << 20
const MaxWireBytes = 1 << 20

// Asset is a retained UTF-8 document. Source is provenance only, never read on recovery.
type Asset struct{ ID, Source, Version, Content string }

func NewAsset(id, source string, content []byte) (Asset, error) {
	a := Asset{ID: id, Source: source, Content: string(content)}
	sum := sha256.Sum256(content)
	a.Version = hex.EncodeToString(sum[:])
	return a, a.Check()
}
func (a Asset) Check() error {
	sum := sha256.Sum256([]byte(a.Content))
	if schema.CheckIdentifier(a.ID) != nil || len(a.ID) > 512 || len(a.Source) > 4096 || !utf8.ValidString(a.Source) || !utf8.ValidString(a.Content) || len(a.Content) > MaxAssetBytes || a.Version != hex.EncodeToString(sum[:]) {
		return errors.New("invalid corpus asset")
	}
	return nil
}

// Snapshot reads an explicitly selected file once, before Create/Run.
func Snapshot(id, path string) (Asset, error) {
	f, e := os.Open(path)
	if e != nil {
		return Asset{}, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, MaxAssetBytes+1))
	if e != nil {
		return Asset{}, e
	}
	return NewAsset(id, path, b)
}
func CheckAssets(assets []Asset) error {
	if len(assets) > MaxAssets {
		return errors.New("too many corpus documents")
	}
	n := 0
	seen := map[string]bool{}
	for _, a := range assets {
		if e := a.Check(); e != nil {
			return e
		}
		if seen[a.ID] {
			return errors.New("duplicate document identity")
		}
		seen[a.ID] = true
		n += len(a.Content)
	}
	if n > MaxCorpusBytes {
		return errors.New("corpus byte limit")
	}
	return nil
}

type Request struct {
	ID, Key, Capability, Effect string
	Input, Output               schema.Retained
	Payload                     map[string]schema.Value
	Assets                      []Asset
}

// Prepared contains all selected non-secret settings and exact wire bytes.
// Endpoint is deployment configuration, never taken from Payload.
type Prepared struct {
	Request                           Request
	Digest                            string
	Protocol, Endpoint, Model         string
	MaxOutputTokens, MaxResponseBytes int
	Wire                              string
}
type Usage struct {
	InputTokens, OutputTokens, TotalTokens *int64
	Details                                json.RawMessage
}
type Candidate struct {
	ID, Key, Disposition                           string
	Payload                                        map[string]schema.Value
	HTTPStatus                                     int
	RequestID, ResponseID, ReturnedModel, Response string
	Usage                                          *Usage
}

// Adapter.Prepare must be pure, bounded, nonblocking serialization and config
// validation: no I/O, goroutines, Store access or policy. Execute is called only
// in a bounded worker after authorization and durable attempt reservation. It
// performs one effect, has no Store, and must respect context cancellation.
// Implementations are trusted deployment code; this is not a sandbox.
type Adapter interface {
	Prepare(Request) (Prepared, error)
	Execute(context.Context, Prepared) Candidate
}

// Copy gives exclusive ownership of nested schema, payload and asset containers.
func Copy[T any](v T) T { b, _ := json.Marshal(v); var out T; _ = json.Unmarshal(b, &out); return out }
func Freeze(p Prepared) Prepared {
	p.Digest = ""
	b, _ := json.Marshal(p)
	sum := sha256.Sum256(b)
	p.Digest = hex.EncodeToString(sum[:])
	return p
}
func CheckPrepared(p Prepared) error {
	if p.Digest != "" && Freeze(p).Digest != p.Digest {
		return errors.New("prepared digest mismatch")
	}
	if p.Protocol == "" || !utf8.ValidString(p.Protocol) || len(p.Wire) > MaxWireBytes || !utf8.ValidString(p.Wire) || p.MaxResponseBytes < 1 || p.MaxResponseBytes > MaxWireBytes || p.MaxOutputTokens < 0 || !utf8.ValidString(p.Model) || !utf8.ValidString(p.Endpoint) {
		return errors.New("invalid prepared effect")
	}
	return nil
}

// Result normalizes unsuccessful candidates to typed zeros. The runtime owns the
// envelope and independently validates successful payloads.
func Result(output schema.Record, c Candidate) (map[string]schema.Value, string) {
	d := c.Disposition
	if d == "" || len(d) > 128 || !utf8.ValidString(d) {
		d = "protocol_invalid"
	}
	valid := d == "validated"
	if valid && output.Validate(c.Payload) != nil {
		valid = false
		d = "schema_invalid"
	}
	v := map[string]schema.Value{}
	for k, f := range output {
		if valid {
			v[k] = c.Payload[k]
		} else {
			v[k] = schema.Value{Kind: f.Kind}
		}
	}
	v["valid"] = schema.Bool(valid)
	v["disposition"] = schema.Text(d)
	return v, d
}
