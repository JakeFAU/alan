//go:build integration

package durabletest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/runtime"
)

type history2c struct {
	t                      *testing.T
	db, world, apply, work string
	source                 runtime.RunID
	original               string
	ownerPID               int
	ownerEpoch             int64
	readers                map[string]*sql.DB
}

func newHistory2c(t *testing.T) *history2c {
	t.Helper()
	work := t.TempDir()
	return &history2c{t: t, work: work, db: filepath.Join(work, "local.sqlite"), world: filepath.Join(work, "world.sqlite"), readers: map[string]*sql.DB{}}
}

func (h *history2c) event(name string, detail any) {
	h.t.Helper()
	h.t.Logf("%s: %+v", name, detail)
}

func (h *history2c) query(path, stmt string, args ...any) *sql.Row {
	h.t.Helper()
	conn := h.readers[path]
	if conn == nil {
		u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_busy_timeout=5000"}
		var err error
		conn, err = sql.Open("sqlite3", u.String())
		if err != nil {
			h.t.Fatal(err)
		}
		h.t.Cleanup(func() { conn.Close() })
		h.readers[path] = conn
	}
	return conn.QueryRowContext(h.t.Context(), stmt, args...)
}
func (h *history2c) count(path, stmt string, args ...any) int {
	h.t.Helper()
	var n int
	if err := h.query(path, stmt, args...).Scan(&n); err != nil {
		h.t.Fatal(stmt, err)
	}
	return n
}
func (h *history2c) data(path, stmt string, args ...any) string {
	h.t.Helper()
	var s string
	if err := h.query(path, stmt, args...).Scan(&s); err != nil {
		h.t.Fatal(stmt, err)
	}
	return s
}

type ready2c struct {
	PID                                    int
	URL, LookupURL, LedgerID, Store, Token string
}

func (h *history2c) endpoint(address string, cut *barrier) (*child, ready2c) {
	t := h.t
	t.Helper()
	if address == "" {
		address = "127.0.0.1:0"
	}
	token := fmt.Sprintf("phase2c-%d", time.Now().UnixNano())
	mode := "normal"
	if cut != nil {
		mode = "finish-accepted"
	}
	cmd := exec.Command(filepath.Join(binaries(t), "alan-resource"), "--receipt-ledger", "--db", h.world, "--token", token, "--listen", address, "--journal", h.world+".ingress.jsonl", "--response-mode", mode)
	conf := barrier{Store: h.world, Token: token, Scenario: t.Name()}
	if cut != nil {
		conf.Boundary, conf.Phase = cut.Boundary, cut.Phase
		raw, _ := json.Marshal(conf)
		cmd.Env = append(os.Environ(), "ALAN_BARRIER="+string(raw))
	}
	ackR, ackW := pipe2c(t)
	resumeR, resumeW := pipe2c(t)
	cmd.ExtraFiles = []*os.File{ackW, resumeR}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.CreateTemp(h.work, "endpoint-log-")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ackW.Close()
	resumeR.Close()
	c := &child{cmd: cmd, ack: ackR, release: resumeW, config: conf, stderr: log}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		ackR.Close()
		resumeW.Close()
		log.Close()
	})
	done := make(chan ready2c, 1)
	go func() { var r ready2c; json.NewDecoder(out).Decode(&r); done <- r }()
	var r ready2c
	select {
	case r = <-done:
	case <-time.After(35 * time.Second):
		t.Fatal("endpoint startup timed out")
	}
	pin, e := rq.EndpointID(r.URL, "apply")
	lookup, le := rq.LookupEndpoint(r.URL)
	if e != nil || le != nil || r.PID != cmd.Process.Pid || r.Token != token || r.Store != h.world || pin != r.LedgerID || r.LookupURL != lookup {
		raw, _ := os.ReadFile(log.Name())
		t.Fatal("invalid ledger readiness", r, string(raw))
	}
	h.event("endpoint ready", r)
	return c, r
}

func (h *history2c) seed(queries int, allowAdoption bool) {
	t := h.t
	t.Helper()
	p, err := reconciliationProvision(queries)
	if err != nil {
		t.Fatal(err)
	}
	p.Reconciliation = &runtime.ReconciliationProfile{Grants: []runtime.ReceiptGrant{{Resolver: resolver2c, Invocation: `["parent",1,1,0]`, Member: "source", Sequence: 2, Endpoint: h.apply, Adopt: allowAdoption}}}
	phaseCreate(t, p, h.db)
}

func (h *history2c) replay(label string) {
	t := h.t
	t.Helper()
	journal := func() []byte {
		data, err := os.ReadFile(h.world + ".ingress.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := inspect(t, h.db)
	world := phaseSnapshot(t, h.world)
	ingress := journal()
	report, err := runtime.Replay(t.Context(), h.db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, inspect(t, h.db)) || !reflect.DeepEqual(world, phaseSnapshot(t, h.world)) || !bytes.Equal(ingress, journal()) {
		t.Fatal("historical replay changed records", label)
	}
	t.Logf("replay at %s, %d commits", label, report.HistoryCount)
}
