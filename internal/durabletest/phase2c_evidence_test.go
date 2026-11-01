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
	"github.com/JakeFAU/alan/examples/reconciliation"
	"github.com/JakeFAU/alan/runtime"
)

type history2c struct {
	t                           *testing.T
	db, world, apply, dir, work string
	source                      runtime.RunID
	original                    string
	ownerPID                    int
	ownerEpoch                  int64
	readers                     map[string]*sql.DB
	events                      []struct {
		Event  string
		Detail any
	}
}

func newHistory2c(t *testing.T, name string) *history2c {
	t.Helper()
	base := phaseEvidenceDir(t)
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALAN_PHASE2_EVIDENCE", dir)
	work := t.TempDir()
	return &history2c{t: t, dir: dir, work: work, db: filepath.Join(work, "local.sqlite"), world: filepath.Join(work, "world.sqlite"), readers: map[string]*sql.DB{}}
}

func (h *history2c) event(name string, detail any) {
	h.t.Helper()
	h.events = append(h.events, struct {
		Event  string
		Detail any
	}{name, detail})
	raw, err := json.MarshalIndent(h.events, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(h.dir, "transcript.json"), append(raw, '\n'), 0600)
	}
	if err != nil {
		h.t.Fatal(err)
	}
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
	phaseEmit(t, fmt.Sprintf("endpoint-%d", r.PID), phaseProvenance(t, cmd))
	h.event("endpoint ready", r)
	return c, r
}

func (h *history2c) seed(queries int) {
	t := h.t
	t.Helper()
	p, err := reconciliation.New(reconciliation.Options{Queries: queries})
	if err != nil {
		t.Fatal(err)
	}
	p.Reconciliation = &runtime.ReconciliationProfile{Grants: []runtime.ReceiptGrant{{Resolver: resolver2c, Invocation: `["parent",1,1,0]`, Member: "source", Sequence: 2, Endpoint: h.apply, Adopt: true}}}
	phaseCreate(t, p, h.db)
}

func (h *history2c) snapshot(label string) {
	t := h.t
	t.Helper()
	for _, store := range []struct{ kind, path string }{{"local", h.db}, {"world", h.world}} {
		file := phaseRawStore(t, store.path, label+"-"+store.kind)
		if err := os.Rename(file, filepath.Join(h.dir, label+"-"+store.kind+".sqlite.gz")); err != nil {
			t.Fatal(err)
		}
	}
	copyFile2c(t, h.world+".ingress.jsonl", filepath.Join(h.dir, label+"-journal.jsonl"))
	h.event("snapshot", label)
}

func (h *history2c) replay(label string) {
	t := h.t
	t.Helper()
	h.snapshot(label)
	cut := inspect(t, h.db)
	w := phaseSnapshot(t, h.world)
	report, err := runtime.Replay(t.Context(), h.db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	h.snapshot(label + "-replayed")
	journal := func(s string) []byte {
		data, e := os.ReadFile(filepath.Join(h.dir, s+"-journal.jsonl"))
		if e != nil {
			t.Fatal(e)
		}
		return data
	}
	if !reflect.DeepEqual(cut, inspect(t, h.db)) || !reflect.DeepEqual(w, phaseSnapshot(t, h.world)) || !bytes.Equal(journal(label), journal(label+"-replayed")) {
		t.Fatal("historical replay changed records", label)
	}
	h.event("replay", map[string]any{"Cut": label, "HistoryCount": report.HistoryCount})
}
