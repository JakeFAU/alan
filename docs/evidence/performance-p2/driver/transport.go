package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const httpProtocol = "alan-p2-http/v1"
const responseCap = 4096
const requestCap = 65536

type transportConfig struct {
	HTTPVersion             string `json:"http_version"`
	DialTimeoutNS           int64  `json:"dial_timeout_ns"`
	TCPKeepAliveNS          int64  `json:"tcp_keepalive_ns"`
	IdleTimeoutNS           int64  `json:"idle_timeout_ns"`
	ResponseHeaderTimeoutNS int64  `json:"response_header_timeout_ns"`
	ServerReadTimeoutNS     int64  `json:"server_read_timeout_ns"`
	ServerWriteTimeoutNS    int64  `json:"server_write_timeout_ns"`
	MaxConnsPerHost         int    `json:"max_conns_per_host"`
	MaxHeaderBytes          int    `json:"max_header_bytes"`
	ExpectContinueTimeoutNS int64  `json:"expect_continue_timeout_ns"`
	Proxy                   bool   `json:"proxy"`

	Mode             string `json:"mode"`
	ListenAddress    string `json:"listen_address"`
	Endpoint         string `json:"endpoint"`
	Protocol         string `json:"protocol"`
	RequestFormat    string `json:"request_format"`
	ResponseFormat   string `json:"response_format"`
	MaxRequestBytes  int    `json:"max_request_bytes"`
	MaxResponseBytes int    `json:"max_response_bytes"`
	TimeoutNS        int64  `json:"timeout_ns"`
	ReuseConnections bool   `json:"reuse_connections"`
	Redirects        bool   `json:"redirects"`
	ProviderRetries  int    `json:"provider_retries"`
	POSTRetries      bool   `json:"post_retries"`
	ResourceScope    string `json:"resource_scope"`
}

func frozenTransport(mode, address string) transportConfig {
	c := transportConfig{HTTPVersion: "HTTP/1.1", DialTimeoutNS: int64(30 * time.Second), TCPKeepAliveNS: int64(30 * time.Second),
		IdleTimeoutNS: int64(90 * time.Second), ResponseHeaderTimeoutNS: int64(30 * time.Second), ServerReadTimeoutNS: int64(30 * time.Second),
		ServerWriteTimeoutNS: int64(30 * time.Second), MaxConnsPerHost: 1, MaxHeaderBytes: 8192, ExpectContinueTimeoutNS: int64(time.Second), Mode: mode, ListenAddress: address, Protocol: httpProtocol,
		RequestFormat: "json-encoding-go-effects.Prepared-v1", ResponseFormat: "json-operation-key-response-sample-v1",
		MaxRequestBytes: requestCap, MaxResponseBytes: responseCap, TimeoutNS: int64(30 * time.Second),
		ReuseConnections: true, ResourceScope: "same-process-server-included"}
	if mode == "direct" {
		c.HTTPVersion = "none"
		c.DialTimeoutNS = 0
		c.TCPKeepAliveNS = 0
		c.IdleTimeoutNS = 0
		c.ResponseHeaderTimeoutNS = 0
		c.ServerReadTimeoutNS = 0
		c.ServerWriteTimeoutNS = 0
		c.MaxConnsPerHost = 0
		c.MaxHeaderBytes = 0
		c.ExpectContinueTimeoutNS = 0
		c.Endpoint, c.Protocol = "deterministic-local-no-http", "measurement/v1"
		c.ListenAddress = ""
		c.RequestFormat, c.ResponseFormat = "station-text", "historical-candidate-v1"
		c.MaxRequestBytes, c.MaxResponseBytes = 64, 256
		c.TimeoutNS, c.ReuseConnections = 0, false
		c.ResourceScope = "process-no-server"
	}
	return c
}

type serverJournal struct {
	mu       sync.Mutex
	contacts []contact
}
type meter struct {
	mu          sync.Mutex
	contacts    []contact
	mode        string
	endpoint    string
	client      *http.Client
	transport   *http.Transport
	server      *http.Server
	serverDone  chan error
	connections sync.WaitGroup
	journal     serverJournal
}

type httpResult struct {
	OperationID string `json:"operation_id"`
	Key         string `json:"key"`
	ResponseID  string `json:"response_id"`
	Sample      string `json:"sample"`
}

func digestBytes(b []byte) string { d := sha256.Sum256(b); return hex.EncodeToString(d[:]) }
func resultBytes(p effects.Prepared) []byte {
	b, _ := json.Marshal(httpResult{OperationID: p.Request.ID, Key: p.Request.Key,
		ResponseID: "local-response:" + p.Request.ID, Sample: "present"})
	return b
}
func preparedWire(p effects.Prepared) (string, error) {
	// Continuation scope requires identical action bytes; the transport envelope
	// separately carries the full original ID/key and frozen Prepared digest.
	request := effects.Copy(p.Request)
	request.ID, request.Key = "", ""
	b, e := json.Marshal(struct {
		Request  effects.Request `json:"request"`
		Protocol string          `json:"protocol"`
		Endpoint string          `json:"endpoint"`
	}{Request: request, Protocol: p.Protocol, Endpoint: p.Endpoint})
	return string(b), e
}
func newMeter(c transportConfig) (*meter, error) {
	m := &meter{mode: c.Mode, endpoint: c.Endpoint, contacts: make([]contact, 0), journal: serverJournal{contacts: make([]contact, 0)}}
	if c.Mode == "direct" {
		return m, nil
	}
	if c.Mode != "http" {
		return nil, fmt.Errorf("unknown transport mode %q", c.Mode)
	}
	host, _, err := net.SplitHostPort(c.ListenAddress)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("numeric loopback-only listen address required")
	}
	listener, err := net.Listen("tcp", c.ListenAddress)
	if err != nil {
		return nil, err
	}
	m.endpoint = "http://" + listener.Addr().String() + "/measure"
	m.transport = &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		DisableKeepAlives: false, MaxConnsPerHost: 1, MaxIdleConns: 1, MaxIdleConnsPerHost: 1,
		IdleConnTimeout: 90 * time.Second, ResponseHeaderTimeout: 30 * time.Second, ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2: false}
	m.client = &http.Client{Transport: m.transport, Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	m.server = &http.Server{ConnState: m.trackConnection, Handler: http.HandlerFunc(m.handle), ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 8192}
	m.serverDone = make(chan error, 1)
	go func() { m.serverDone <- m.server.Serve(listener) }()
	// The synchronously bound listener is ready before Execute; there is no warmup contact.
	return m, nil
}
func (m *meter) trackConnection(_ net.Conn, state http.ConnState) {
	if state == http.StateNew {
		m.connections.Add(1)
	}
	if state == http.StateClosed || state == http.StateHijacked {
		m.connections.Done()
	}
}
func (m *meter) close() error {
	if m.server == nil {
		return nil
	}
	m.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := m.server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, m.server.Close())
	}
	serveErr := <-m.serverDone
	// Serve has stopped accepting: no new StateNew registrations can race Wait.
	// StateClosed follows handler return, including after a forced Close.
	m.connections.Wait()
	if !errors.Is(serveErr, http.ErrServerClosed) {
		err = errors.Join(err, serveErr)
	}
	return err
}

// finishReport is also the error-path boundary: active handlers must finish
// before either journal is copied into the preserved report.
func (m *meter) finishReport(r *report) error {
	err := m.close()
	m.mu.Lock()
	r.Contacts = append([]contact{}, m.contacts...)
	m.mu.Unlock()
	r.ServerContacts = m.serverCopy()
	return err
}
func (m *meter) serverCount() int {
	m.journal.mu.Lock()
	defer m.journal.mu.Unlock()
	return len(m.journal.contacts)
}
func (m *meter) serverCopy() []contact {
	m.journal.mu.Lock()
	defer m.journal.mu.Unlock()
	return append([]contact{}, m.journal.contacts...)
}
func (m *meter) Prepare(r effects.Request) (effects.Prepared, error) {
	if m.mode == "direct" {
		return effects.Prepared{Request: r, Protocol: "measurement/v1", Endpoint: "deterministic-local-no-http", Wire: r.Payload["station"].Text, MaxResponseBytes: 256}, nil
	}
	p := effects.Prepared{Request: r, Protocol: httpProtocol, Endpoint: m.endpoint, MaxResponseBytes: responseCap}
	var err error
	p.Wire, err = preparedWire(p)
	if len(p.Wire) > requestCap/2 {
		return effects.Prepared{}, errors.New("prepared wire too large")
	}
	return p, err
}
func contactFor(p effects.Prepared, ordinal int, request, response []byte, complete bool) contact {
	return contact{Ordinal: ordinal, OperationID: p.Request.ID, Key: p.Request.Key, PreparedDigest: p.Digest,
		Capability: p.Request.Capability, Effect: p.Request.Effect, Protocol: p.Protocol, Endpoint: p.Endpoint,
		ResponseID: "local-response:" + p.Request.ID, Prepared: p, RequestWire: string(request), RequestSHA256: digestBytes(request),
		ResponseWire: string(response), ResponseSHA256: digestBytes(response), ResponseComplete: complete}
}
func (m *meter) handle(w http.ResponseWriter, r *http.Request) {
	// Every value in this journal is decoded/derived from the actual incoming HTTP
	// request. No client Prepared object, endpoint configuration or contact is read.
	if r.Method != "POST" || r.URL.Path != "/measure" || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "protocol", 400)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, requestCap+1))
	if err != nil || len(raw) > requestCap {
		http.Error(w, "request bound", 400)
		return
	}
	var p effects.Prepared
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&p); err != nil {
		http.Error(w, "request", 400)
		return
	}
	canonical, _ := json.Marshal(p)
	wire, err := preparedWire(p)
	valid := err == nil && string(canonical) == string(raw) && p.Wire == wire && effects.CheckPrepared(p) == nil &&
		p.Digest != "" && p.Protocol == httpProtocol && p.Endpoint == "http://"+r.Host+r.URL.RequestURI() && p.MaxResponseBytes == responseCap
	if !valid {
		http.Error(w, "provenance", 400)
		return
	}
	response := resultBytes(p)
	w.Header().Set("Content-Type", "application/json")
	m.journal.mu.Lock()
	defer m.journal.mu.Unlock()
	c := contactFor(p, len(m.journal.contacts)+1, raw, response, false)
	m.journal.contacts = append(m.journal.contacts, c)
	n, writeErr := w.Write(response)
	m.journal.contacts[len(m.journal.contacts)-1].ResponseComplete = writeErr == nil && n == len(response)
}
func (m *meter) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	if m.mode == "direct" {
		m.mu.Lock()
		defer m.mu.Unlock()
		response := "local-response:" + p.Request.ID
		m.contacts = append(m.contacts, contact{Ordinal: len(m.contacts) + 1, OperationID: p.Request.ID, Key: p.Request.Key, PreparedDigest: p.Digest, Capability: p.Request.Capability, Effect: p.Request.Effect, Protocol: p.Protocol, Endpoint: p.Endpoint, ResponseID: response})
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}, ResponseID: response}
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "invalid"}
	}
	m.mu.Lock()
	ordinal := len(m.contacts) + 1
	m.contacts = append(m.contacts, contactFor(p, ordinal, raw, nil, false))
	m.mu.Unlock()
	// NopCloser deliberately hides strings.Reader's rewind interface. GetBody is
	// nil, no idempotency header is sent, and Go cannot retry this nonempty POST.
	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint, io.NopCloser(strings.NewReader(string(raw))))
	if err != nil {
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "invalid"}
	}
	req.ContentLength = int64(len(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "unknown"}
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, responseCap+1))
	closeErr := resp.Body.Close()
	complete := readErr == nil && closeErr == nil && len(body) <= responseCap && resp.StatusCode == 200
	m.mu.Lock()
	m.contacts[ordinal-1] = contactFor(p, ordinal, raw, body, complete)
	m.mu.Unlock()
	if !complete || string(body) != string(resultBytes(p)) {
		return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "invalid"}
	}
	return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}, HTTPStatus: 200, RequestID: p.Request.ID, ResponseID: "local-response:" + p.Request.ID, Response: string(body)}
}
func verifyTransport(p effects.Prepared, candidate effects.Candidate, c contact, m *meter, j int) error {
	if effects.CheckPrepared(p) != nil || p.Digest == "" {
		return errors.New("invalid retained Prepared digest")
	}
	expected := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}, ResponseID: "local-response:" + p.Request.ID}
	if m.mode != "http" {
		if p.Protocol != "measurement/v1" || p.Endpoint != "deterministic-local-no-http" || p.Wire != p.Request.Payload["station"].Text || p.MaxResponseBytes != 256 || !reflect.DeepEqual(candidate, expected) {
			return errors.New("direct protocol/result mismatch")
		}
		return nil
	}
	raw, _ := json.Marshal(p)
	response := resultBytes(p)
	wire, err := preparedWire(p)
	if err != nil || p.Protocol != httpProtocol || p.Endpoint != m.endpoint || p.Wire != wire || p.MaxResponseBytes != responseCap {
		return errors.New("HTTP retained request mismatch")
	}
	expected.HTTPStatus, expected.RequestID, expected.Response = 200, p.Request.ID, string(response)
	if !reflect.DeepEqual(candidate, expected) {
		return errors.New("HTTP retained Candidate mismatch")
	}
	want := contactFor(p, j+1, raw, response, true)
	if !reflect.DeepEqual(c, want) {
		return errors.New("adapter request/response mismatch")
	}
	m.journal.mu.Lock()
	defer m.journal.mu.Unlock()
	if j >= len(m.journal.contacts) || !reflect.DeepEqual(m.journal.contacts[j], want) {
		return errors.New("server request/response bijection mismatch")
	}
	return nil
}
