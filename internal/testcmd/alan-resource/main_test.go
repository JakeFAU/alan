package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/internal/fixture"
)

func TestLedgerCLIHelper(t *testing.T) {
	if os.Getenv("ALAN_INTERNAL_LEDGER_CLI_TEST") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"alan-resource"}, os.Args[i+1:]...)
			break
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

type ledgerServer struct {
	URL, LookupURL, LedgerID, Token, Store string
	PID                                    int
	cmd                                    *exec.Cmd
}

func startLedgerCLI(t *testing.T, path, addr, token string) ledgerServer {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLedgerCLIHelper$", "--", "--receipt-ledger", "--listen", addr, "--db", path, "--token", token)
	cmd.Env = append(os.Environ(), "ALAN_INTERNAL_LEDGER_CLI_TEST=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	var s ledgerServer
	if err = json.NewDecoder(out).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if cmd.Process.Pid != s.PID || s.Token != token || s.Store != path || len(s.LedgerID) != 64 {
		t.Fatalf("wrong startup identity %+v", s)
	}
	s.cmd = cmd
	return s
}
func stopLedgerCLI(t *testing.T, s ledgerServer) {
	t.Helper()
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := s.cmd.Wait(); err != nil {
		t.Fatalf("endpoint did not close cleanly: %v", err)
	}
}
func TestLedgerCLIRestartReplacementAndRouteSeparation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "original.sqlite")
	s := startLedgerCLI(t, path, "127.0.0.1:0", "first")
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	key := func(id string) string { b, _ := json.Marshal([]any{2, strings.Repeat("b", 64), id}); return string(b) }
	q := receiptquery.Query{QueryID: "read", QueryKey: key("read"), TargetID: "operation", TargetKey: key("operation"), LedgerID: s.LedgerID, Ordinal: 1}
	b, _ := json.Marshal(fixture.Request{Key: q.TargetKey, Profile: "non-idempotent", Delta: 5})
	q.Wire = string(b)
	q.WireSHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
	query, _ := json.Marshal(q)
	call := func(method, target string, body []byte, status int) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != status {
			t.Fatalf("%s %s: %d %s %v", method, target, resp.StatusCode, b, err)
		}
		return string(b)
	}
	call("POST", s.URL, b, 200)
	call("GET", s.LookupURL, query, 404)
	call("POST", s.LookupURL+"?x=1", query, 404)
	call("POST", strings.Replace(s.LookupURL, "lookup", "%6cookup", 1), query, 404)
	call("POST", s.URL, query, 400)
	call("POST", s.LookupURL, b, 400)
	call("POST", "http://"+u.Host+"/apply", b, 404)
	stopLedgerCLI(t, s)
	successor := startLedgerCLI(t, path, u.Host, "second")
	if successor.LedgerID != s.LedgerID || successor.URL != s.URL || successor.LookupURL != s.LookupURL || successor.PID == s.PID {
		t.Fatal("restart failed identity check")
	}
	raw := call("POST", s.LookupURL, query, 200)
	result, err := receiptquery.DecodeSnapshot(q, raw)
	if err != nil || result.Status != "found" || result.Receipt.Process != "first" || result.Receipt.PID != s.PID || result.Receipt.Total != 5 {
		t.Fatal(result, err)
	}
	stopLedgerCLI(t, successor)
	world, err := fixture.Inspect(t.Context(), path)
	if err != nil || world.Contacts != 1 || len(world.Applications) != 1 || world.ReadContacts != 1 {
		t.Fatal(world, err)
	}
	fresh := startLedgerCLI(t, filepath.Join(dir, "different.sqlite"), u.Host, "replacement")
	if fresh.LedgerID == s.LedgerID {
		t.Fatal("new ledger reused identity")
	}
	call("POST", s.LookupURL, query, 404)
	call("POST", fresh.LookupURL, query, 400)
	stopLedgerCLI(t, fresh)
}
