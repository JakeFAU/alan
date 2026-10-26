//go:build integration

package evidencetest

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
)

type providerEvent struct {
	Kind, Endpoint, ClientID, Wire, WireDigest, Response string
	PID, Ordinal                                         int
}

// TestProviderChild is a separate durable HTTP process. It has no runtime-store
// path or database access. Headers (including Authorization) are never retained.
func TestProviderChild(t *testing.T) {
	if os.Getenv("ALAN_TEST_PROVIDER") != "1" {
		t.Skip("child entrypoint")
	}
	ledger, err := os.OpenFile(os.Getenv("ALAN_PROVIDER_LEDGER"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	resource, err := resourcefixture.Open(t.Context(), os.Getenv("ALAN_PROVIDER_LEDGER")+".sqlite", "phase1e-controlled-provider")
	if err != nil {
		t.Fatal(err)
	}
	defer resource.Close()
	ack := os.NewFile(3, "provider-ack")
	defer ack.Close()
	control := os.NewFile(4, "provider-control")
	defer control.Close()
	var mu sync.Mutex
	ordinal := 0
	releases := map[int]chan struct{}{}
	emit := func(e providerEvent) {
		e.PID = os.Getpid()
		if err := json.NewEncoder(ledger).Encode(e); err != nil {
			panic(err)
		}
		if err := ledger.Sync(); err != nil {
			panic(err)
		}
		if err := json.NewEncoder(ack).Encode(e); err != nil {
			panic(err)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	emit(providerEvent{Kind: "ready", Endpoint: "http://" + listener.Addr().String()})
	go func() {
		scanner := bufio.NewScanner(control)
		for scanner.Scan() {
			n, err := strconv.Atoi(scanner.Text())
			if err != nil {
				panic(err)
			}
			mu.Lock()
			ch := releases[n]
			if ch != nil {
				close(ch)
				delete(releases, n)
			}
			mu.Unlock()
		}
		_ = listener.Close()
	}()
	server := http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			panic(err)
		}
		mu.Lock()
		ordinal++
		n := ordinal
		hold := make(chan struct{})
		held := os.Getenv("ALAN_PROVIDER_HOLD") == "first" && n == 1
		if held {
			releases[n] = hold
		}
		emit(providerEvent{Kind: "accepted", Ordinal: n, ClientID: r.Header.Get("X-Client-Request-Id"), Wire: string(raw), WireDigest: fmt.Sprintf("%x", sha256.Sum256(raw))})
		mu.Unlock()
		if held {
			<-hold
		}
		if r.URL.Path == "/fixture" {
			var request resourcefixture.Request
			if err := json.Unmarshal(raw, &request); err != nil {
				panic(err)
			}
			receipt, err := resource.Apply(r.Context(), request)
			if err != nil {
				panic(err)
			}
			response, err := json.Marshal(receipt)
			if err != nil {
				panic(err)
			}
			mu.Lock()
			emit(providerEvent{Kind: "responding", Ordinal: n, Response: string(response)})
			mu.Unlock()
			if _, err := w.Write(response); err != nil {
				return
			}
			return
		}
		var wire struct {
			Model string `json:"model"`
			Input []struct {
				Content string `json:"content"`
			} `json:"input"`
		}
		if err := json.Unmarshal(raw, &wire); err != nil {
			panic(err)
		}
		payload := map[string]any{"decision": "lookup", "tool": "lookup", "query": "WAL", "message": ""}
		var in map[string]any
		if len(wire.Input) > 0 {
			_ = json.Unmarshal([]byte(wire.Input[0].Content), &in)
		}
		if snippet, ok := in["snippet"]; ok {
			payload = map[string]any{"answer": "WAL retains a durable log.", "document": in["document"], "version": in["version"], "snippet": snippet, "start": in["start"], "end": in["end"]}
		}
		text, _ := json.Marshal(payload)
		response, _ := json.Marshal(map[string]any{"id": fmt.Sprintf("controlled-response-%d", n), "model": "controlled-returned-model", "status": "completed", "usage": map[string]int{"input_tokens": 13, "output_tokens": 7, "total_tokens": 20}, "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(text)}}}}})
		mu.Lock()
		emit(providerEvent{Kind: "responding", Ordinal: n, ClientID: r.Header.Get("X-Client-Request-Id"), Response: string(response)})
		mu.Unlock()
		w.Header().Set("X-Request-Id", fmt.Sprintf("controlled-request-%d", n))
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(response); err != nil {
			return
		}
	})}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		t.Log(err)
	}
}
