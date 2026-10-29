//go:build integration

package substitutiontest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/effects"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
)

type httpAdapter struct{ endpoint string }

func (a httpAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	b, e := json.Marshal(r.Payload)
	return effects.Prepared{Request: r, Protocol: "phase1i-controlled/v1", Endpoint: a.endpoint, Wire: string(b), MaxResponseBytes: 8192}, e
}
func (a httpAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	r, e := http.NewRequestWithContext(ctx, "POST", p.Endpoint, bytes.NewBufferString(p.Wire))
	if e != nil {
		return c
	}
	r.Header.Set("X-Alan-Operation", p.Request.ID)
	r.Header.Set("X-Alan-Key", p.Request.Key)
	resp, e := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(r)
	if e != nil {
		return c
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if e != nil {
		return c
	}
	c.Response = string(raw)
	c.HTTPStatus = resp.StatusCode
	c.Disposition = "protocol_invalid"
	var payload map[string]schema.Value
	if resp.StatusCode == 200 && len(raw) <= 8192 && json.Unmarshal(raw, &payload) == nil {
		c.Disposition = "validated"
		c.Payload = payload
	}
	return c
}

type contact struct {
	Operation, Key, Phase string
	Value                 int64
}
type endpoint struct {
	mu                         sync.Mutex
	ledger                     []contact
	entered, release, ended    chan struct{}
	once, releaseOnce, endOnce sync.Once
	hold                       bool
}

func newEndpoint(t *testing.T, hold bool) (*endpoint, *httptest.Server) {
	p := &endpoint{entered: make(chan struct{}), release: make(chan struct{}), ended: make(chan struct{}), hold: hold}
	server := httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(func() { p.unblock(); server.Close() })
	return p, server
}
func (p *endpoint) add(c contact) { p.mu.Lock(); defer p.mu.Unlock(); p.ledger = append(p.ledger, c) }
func (p *endpoint) serve(w http.ResponseWriter, r *http.Request) {
	var payload map[string]schema.Value
	if json.NewDecoder(r.Body).Decode(&payload) != nil {
		http.Error(w, "invalid", 400)
		return
	}
	c := contact{r.Header.Get("X-Alan-Operation"), r.Header.Get("X-Alan-Key"), "applied", payload["value"].Number}
	p.add(c)
	p.once.Do(func() { close(p.entered) })
	if p.hold {
		select {
		case <-p.release:
		case <-r.Context().Done():
			c.Phase = "connection_closed"
			p.add(c)
			p.endOnce.Do(func() { close(p.ended) })
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]schema.Value{"observation": schema.Int(1 - c.Value)})
	c.Phase = "response"
	p.add(c)
}
func (p *endpoint) unblock() { p.releaseOnce.Do(func() { close(p.release) }) }
func (p *endpoint) snapshot() []contact {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]contact{}, p.ledger...)
}
func contacts(p *endpoint) int {
	n := 0
	for _, c := range p.snapshot() {
		if c.Phase == "applied" {
			n++
		}
	}
	return n
}
func await(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(childTimeout()):
		t.Fatal("watchdog", name)
	}
}
func effectDeployment(t *testing.T, twice, operator bool) (app.Deployment, string) {
	c, e := app.EffectCompanion(twice)
	must(t, e)
	l := limits(1, 1, 2)
	l.MaxEffects = 1
	d, e := app.New(c, app.Config{Shape: "bare", Request: app.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: l})
	must(t, e)
	d.Provision.Composition.Roots[0].Capabilities = []string{app.EffectCapability}
	if operator {
		addOperator(t, &d.Provision)
	}
	return d, filepathJoin(t.TempDir(), "retained.db")
}
func contactMonitor(t *testing.T, s runtime.Inspection, p *endpoint) {
	t.Helper()
	seen := map[string]bool{}
	for _, c := range p.snapshot() {
		if c.Phase != "applied" {
			continue
		}
		if c.Operation == "" || c.Key == "" || seen[c.Operation] {
			t.Fatal("invalid/repeated endpoint effect", c)
		}
		seen[c.Operation] = true
		found := false
		for _, r := range s.Tables["adapter_attempts"] {
			a := decode[runtime.AdapterAttempt](r, "descriptor")
			if a.Intent.ID == c.Operation && a.Prepared.Request.Key == c.Key && a.Allowed {
				found = true
			}
		}
		if !found {
			t.Fatal("endpoint contact lacks original allowed attempt")
		}
	}
}
func runEffects(t *testing.T) {
	t.Run("known", func(t *testing.T) {
		d, path := effectDeployment(t, false, false)
		create(t, d, path)
		ep, server := newEndpoint(t, false)
		p := start(t, path, d.Candidate.ID(), "", server.URL, false)
		p.wait(t, false)
		s := inspect(t, path)
		contactMonitor(t, s, ep)
		if projection(s, "root").Status != "succeeded" || s.Roots[0].Charges.Effects != 1 || len(s.Tables["adapter_results"]) != 1 || contacts(ep) != 1 {
			t.Fatal("known effect mismatch", s.Roots)
		}
		retain(t, "known", map[string]any{"inspection": s, "endpoint": ep.snapshot()})
		replay := start(t, path, "", "", "", true)
		replay.wait(t, false)
		if !reflect.DeepEqual(s, inspect(t, path)) || contacts(ep) != 1 {
			t.Fatal("replay recontact/mutation")
		}
	})
	t.Run("unknown-applied-reopen", func(t *testing.T) {
		d, path := effectDeployment(t, false, false)
		create(t, d, path)
		ep, server := newEndpoint(t, true)
		p := start(t, path, d.Candidate.ID(), "effect-attempt", server.URL, false)
		p.acknowledgment(t)
		await(t, ep.entered, "endpoint apply")
		before := inspect(t, path)
		if len(before.Tables["adapter_results"]) != 0 || before.Roots[0].Charges.Effects != 1 {
			t.Fatal("no missing receipt")
		}
		retain(t, "before", map[string]any{"inspection": before, "endpoint": ep.snapshot()})
		p.wait(t, true)
		await(t, ep.ended, "original connection close")
		next := start(t, path, d.Candidate.ID(), "", server.URL, false)
		next.wait(t, false)
		after := inspect(t, path)
		stablePrefix(t, before, after)
		contactMonitor(t, after, ep)
		if contacts(ep) != 1 || len(after.Tables["adapter_attempts"]) != 1 || len(after.Tables["adapter_results"]) != 0 || after.Roots[0].Charges.Effects != 1 || len(after.Roots[0].UnknownExternal) != 1 || projection(after, "root").Status != "open" {
			t.Fatal("unknown conservation failed", after.Roots, ep.snapshot())
		}
		retain(t, "after", map[string]any{"inspection": after, "endpoint": ep.snapshot(), "zero_recontact": true})
		replay := start(t, path, "", "", "", true)
		replay.wait(t, false)
		if !reflect.DeepEqual(after, inspect(t, path)) || contacts(ep) != 1 {
			t.Fatal("unknown replay changed state")
		}
	})
	t.Run("closure-before-reservation", func(t *testing.T) {
		d, path := effectDeployment(t, false, true)
		create(t, d, path)
		ep, server := newEndpoint(t, false)
		p := start(t, path, d.Candidate.ID(), "effect-commit", server.URL, false)
		p.acknowledgment(t)
		before := inspect(t, path)
		p.wait(t, true)
		decision := cancelStopped(t, path)
		next := start(t, path, d.Candidate.ID(), "", server.URL, false)
		next.wait(t, false)
		s := inspect(t, path)
		stablePrefix(t, before, s)
		if contacts(ep) != 0 || s.Roots[0].Charges.Effects != 0 {
			t.Fatal("closed scope contacted endpoint")
		}
		found := false
		for _, row := range s.Tables["composition_reservations"] {
			var r struct {
				Allowed bool
				Reason  string
			}
			r = decode[struct {
				Allowed bool
				Reason  string
			}](row, "descriptor")
			if !r.Allowed && r.Reason == "scope_closed" {
				found = true
			}
		}
		if !found {
			t.Fatal("no final scope reservation denial")
		}
		retain(t, "closed", map[string]any{"before": before, "inspection": s, "decision": decision, "endpoint": ep.snapshot(), "fixed_runs": 2})
	})
	runLiveEffects(t)
}
func filepathJoin(a, b string) string { return a + "/" + b }

var _ = fmt.Sprint
