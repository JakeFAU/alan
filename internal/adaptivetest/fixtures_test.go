//go:build integration

// Package adaptivetest inspects evidence; automatic Store.Run owns application work.
package adaptivetest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

const paddingItems = 128

// pureLeaf performs bounded ordinary local computation, with no scheduling hook.
func pureLeaf(t *testing.T) author.Template {
	t.Helper()
	tokens := make([]map[string]schema.Value, paddingItems)
	for i := range tokens {
		tokens[i] = map[string]schema.Value{"n": schema.Int(int64(i))}
	}
	a, err := author.Compile(author.Definition{Name: "qualification pure bounded leaf", Values: map[string]schema.Value{"padding": schema.List(tokens...), "scratch": schema.Int(0)}, Collections: map[string]schema.Field{"padding": {Kind: "list", MaxItems: paddingItems, MaxBytes: 262144, Element: schema.Record{"n": {Kind: "int"}}}}, Ports: map[string]author.Port{"public": {Peer: "caller", Obligation: "work", Receive: schema.Record{"value": {Kind: "int"}}, Send: schema.Record{"observation": {Kind: "int"}}}}, Steps: []author.Step{author.Receive("public", map[string]string{"value": "value"}), author.IfEqual(author.Var("value"), author.Integer(1), []author.Step{author.ForEach(author.Var("padding"), map[string]string{"n": "n"}, author.Set("scratch", author.Var("n")))}, nil), author.Send("public", map[string]author.Expr{"observation": author.Var("value")}), author.Success()}})
	must(t, err)
	leaf, err := author.NewTemplate(author.Application{"entry": {Artifact: a, Inputs: map[string]schema.Value{}}}, "entry", "public")
	must(t, err)
	return leaf
}
func component(t *testing.T, controlled bool) author.Template {
	t.Helper()
	leaf := pureLeaf(t)
	if controlled {
		var err error
		leaf, err = adaptive.ScalarLeaf(true)
		must(t, err)
	}
	v, err := adaptive.NewScalar(leaf)
	must(t, err)
	return v
}
func primary() map[string]schema.Value {
	return map[string]schema.Value{"seeds": schema.List(map[string]schema.Value{"work_id": schema.Int(1), "value": schema.Int(0)}, map[string]schema.Value{"work_id": schema.Int(2), "value": schema.Int(1)}, map[string]schema.Value{"work_id": schema.Int(3), "value": schema.Int(2)}), "max_rounds": schema.Int(3), "max_work": schema.Int(8)}
}
func provision(t *testing.T, c author.Template, nested bool, capacity int, limits *runtime.ResourceLimits) (adaptive.Deployment, string) {
	t.Helper()
	l := adaptive.DefaultLimits(int64(capacity))
	if limits != nil {
		l = *limits
	}
	d, err := adaptive.New(c, nested, l)
	must(t, err)
	return d, filepath.Join(t.TempDir(), "adaptive.sqlite")
}
func create(t *testing.T, d adaptive.Deployment, path string, submit bool) {
	t.Helper()
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, err)
	if submit {
		a, err := adaptive.Submit(t.Context(), s, "frozen-primary", primary())
		must(t, err)
		if !a.Accepted {
			t.Fatal(a)
		}
	}
	must(t, s.Close())
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type httpAdapter struct{ endpoint string }

func (a httpAdapter) Prepare(r effects.Request) (effects.Prepared, error) {
	raw, err := json.Marshal(r.Payload)
	return effects.Prepared{Request: r, Protocol: "phase1h-controlled/v1", Endpoint: a.endpoint, Wire: string(raw), MaxResponseBytes: 8192}, err
}
func (a httpAdapter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewBufferString(p.Wire))
	if err != nil {
		return c
	}
	r.Header.Set("X-Alan-Operation", p.Request.ID)
	r.Header.Set("X-Alan-Key", p.Request.Key)
	res, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(r)
	if err != nil {
		return c
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8193))
	if err != nil {
		return c
	}
	c.Response = string(raw)
	c.HTTPStatus = res.StatusCode
	c.Disposition = "protocol_invalid"
	var v map[string]schema.Value
	if res.StatusCode == 200 && len(raw) <= 8192 && json.Unmarshal(raw, &v) == nil {
		c.Disposition = "validated"
		c.Payload = v
	}
	return c
}

type contact struct {
	Order                 int
	Operation, Key, Phase string
	Value                 int64
}
type provider struct {
	mu                           sync.Mutex
	ledger                       []contact
	active, peak                 int
	hold                         int64
	paired, failBeta             bool
	entered, release, ended      chan struct{}
	once, releaseOnce, endedOnce sync.Once
}

func newProvider(t *testing.T, hold int64, paired, failBeta bool) (*provider, *httptest.Server) {
	t.Helper()
	p := &provider{hold: hold, paired: paired, failBeta: failBeta, entered: make(chan struct{}), release: make(chan struct{}), ended: make(chan struct{})}
	s := httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(func() { p.unblock(); s.Close() })
	return p, s
}
func (p *provider) unblock() { p.releaseOnce.Do(func() { close(p.release) }) }
func (p *provider) add(v int64, op, key, phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ledger = append(p.ledger, contact{len(p.ledger) + 1, op, key, phase, v})
}
func (p *provider) serve(w http.ResponseWriter, r *http.Request) {
	var payload map[string]schema.Value
	if json.NewDecoder(r.Body).Decode(&payload) != nil {
		http.Error(w, "invalid", 400)
		return
	}
	v := payload["value"].Number
	op, key := r.Header.Get("X-Alan-Operation"), r.Header.Get("X-Alan-Key")
	p.mu.Lock()
	p.active++
	if p.active > p.peak {
		p.peak = p.active
	}
	p.mu.Unlock()
	defer func() { p.mu.Lock(); p.active--; p.mu.Unlock() }()
	p.add(v, op, key, "entered")
	if v == p.hold {
		p.once.Do(func() { close(p.entered) })
		select {
		case <-p.release:
		case <-r.Context().Done():
			p.add(v, op, key, "connection_closed")
			p.endedOnce.Do(func() { close(p.ended) })
			return
		}
	}
	if p.failBeta && v == 1 {
		http.Error(w, "known controlled failure", 400)
		p.add(v, op, key, "response")
		return
	}
	n := v
	if p.paired && v == 2 {
		n = 0
	}
	_ = json.NewEncoder(w).Encode(map[string]schema.Value{"observation": schema.Int(n)})
	p.add(v, op, key, "response")
}
func (p *provider) snapshot() ([]contact, int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]contact{}, p.ledger...), p.active, p.peak
}
func contacts(ledger []contact) int {
	n := 0
	for _, r := range ledger {
		if r.Phase == "entered" {
			n++
		}
	}
	return n
}
func contactMonitor(s runtime.Inspection, ledger []contact) error {
	seen := map[string]bool{}
	for _, c := range ledger {
		if c.Phase != "entered" {
			continue
		}
		if c.Operation == "" || c.Key == "" || seen[c.Operation] {
			return fmt.Errorf("missing/repeated provider operation")
		}
		seen[c.Operation] = true
		found := false
		for _, r := range s.Tables["adapter_attempts"] {
			a := decode[runtime.AdapterAttempt](r, "descriptor")
			if a.Intent.ID == c.Operation && a.Prepared.Request.ID == c.Operation && a.Prepared.Request.Key == c.Key && a.Prepared.Request.Payload["value"].Number == c.Value && a.Allowed {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("contact lacks original allowed attempt: %s", c.Operation)
		}
	}
	return nil
}

// addProgressWitness adds one fixed public-author run to the existing root.
// Environmental input, sent only after HTTP entry, drives ordinary local work.
func addProgressWitness(t *testing.T, p *runtime.Provision) {
	t.Helper()
	a, err := author.Compile(author.Definition{Name: "held transport progress witness", Values: map[string]schema.Value{"marker": schema.Int(0)}, Ports: map[string]author.Port{"start": {Local: true, Peer: "local", Obligation: "start", Receive: schema.Record{}}}, Steps: []author.Step{author.Receive("start", map[string]string{}), author.Set("marker", author.Integer(1)), author.Success()}})
	must(t, err)
	q, err := (author.Application{"progress": {Artifact: a, Inputs: map[string]schema.Value{}}}).Provision()
	must(t, err)
	p.Programs = append(p.Programs, q.Programs...)
	p.Subjects = append(p.Subjects, q.Subjects...)
	p.Runs = append(p.Runs, q.Runs...)
	p.Permissions = append(p.Permissions, q.Permissions...)
	p.Composition.Roots[0].Members = append(p.Composition.Roots[0].Members, runtime.RunID{Subject: "progress", Generation: 1})
}

func progressMonitor(s runtime.Inspection) error {
	q := projection(s, "progress")
	if q.Status != "succeeded" || q.Data["marker"].Number != 1 {
		return fmt.Errorf("post-contact ordinary progress witness incomplete")
	}
	for _, o := range s.Tables["observations"] {
		if o["subject"] != "progress" || o["consumed_sequence"] == nil {
			continue
		}
		for _, h := range s.Tables["history"] {
			if h["subject"] == "progress" && h["generation"] == o["generation"] && h["sequence"] == o["consumed_sequence"] && h["selected_observation"] == o["id"] {
				return nil
			}
		}
	}
	return fmt.Errorf("progress witness input lacks original committed consumption")
}
