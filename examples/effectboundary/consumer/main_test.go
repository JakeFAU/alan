//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/examples/effectboundary"
	"github.com/JakeFAU/alan/runtime"
)

func TestSeparateConsumerUnknownAndPeer(t *testing.T) {
	var contacts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request effectboundary.WireRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		n := contacts.Add(1)
		if err := json.NewEncoder(w).Encode(effectboundary.Receipt{Key: request.Key, Profile: request.Profile, Delta: request.Delta, Mutation: n, Total: n, Process: "consumer-test", PID: 1}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "store.db")
	var out bytes.Buffer
	for _, args := range [][]string{{"init", "-db", path}, {"run", "-db", path, "-endpoint", server.URL, "-mode", "unknown"}} {
		out.Reset()
		if err := execute(t.Context(), args, &out); err != nil {
			t.Fatal(err)
		}
	}
	var report runtime.RunReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.UnknownAdapters) != 1 || contacts.Load() != 2 {
		t.Fatalf("report=%+v contacts=%d", report, contacts.Load())
	}
	out.Reset()
	if err := execute(t.Context(), []string{"recover", "-db", path, "-endpoint", server.URL}, &out); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != 2 {
		t.Fatal("consumer retransmitted unknown adapter")
	}
	out.Reset()
	if err := execute(t.Context(), []string{"replay", "-db", path}, &out); err != nil {
		t.Fatal(err)
	}
	if contacts.Load() != 2 {
		t.Fatal("consumer replay contacted endpoint")
	}
}
