//go:build integration

package investigationtest

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"

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
		var wire struct {
			Input []struct{ Content string }
			Text  struct {
				Format struct {
					Schema struct{ Properties map[string]json.RawMessage }
				}
			}
		}
		if err := json.Unmarshal(raw, &wire); err != nil {
			panic(err)
		}
		if len(wire.Input) != 1 {
			panic("missing model input")
		}
		var in map[string]any
		if err := json.Unmarshal([]byte(wire.Input[0].Content), &in); err != nil {
			panic(err)
		}
		props := wire.Text.Format.Schema.Properties
		kind := ""
		payload := map[string]any{}
		switch {
		case props["query"] != nil:
			kind = "plan"
			payload = map[string]any{"decision": "lookup", "tool": "lookup", "query": in["question"], "message": ""}
		case props["document"] != nil:
			kind = "synthesize"
			payload["answer"] = in["snippet"]
			for _, k := range []string{"document", "version", "snippet", "start", "end"} {
				payload[k] = in[k]
			}
		case props["candidate"] != nil:
			kind = "compose"
			candidate := initialCandidate
			if in["e1_snippet"] != "alpha release supports Linux." || in["e2_snippet"] != "beta rollback requires version 2." {
				candidate = "INVALID INPUT"
			}
			if candidate != "INVALID INPUT" && in["e3_snippet"] == "gamma approval is recorded." && in["e4_snippet"] == "delta window is seven minutes." {
				candidate = finalCandidate
			}
			payload = map[string]any{"decision": "answer", "candidate": candidate}
		case props["reason"] != nil:
			kind = "review"
			decision := "revise"
			if in["candidate"] == finalCandidate && in["e1_snippet"] == "alpha release supports Linux." && in["e2_snippet"] == "beta rollback requires version 2." && in["e3_snippet"] == "gamma approval is recorded." && in["e4_snippet"] == "delta window is seven minutes." {
				decision = "accept"
			}
			payload = map[string]any{"decision": decision, "reason": "Checked candidate against supplied evidence.", "subquestion1": "gamma", "subquestion2": "delta"}
		case props["subquestion1"] != nil:
			kind = "decompose"
			payload = map[string]any{"decision": "investigate", "subquestion1": "alpha", "subquestion2": "beta", "message": ""}
		default:
			panic("unknown output schema")
		}
		mu.Lock()
		ordinal++
		n := ordinal
		hold := make(chan struct{})
		held := os.Getenv("ALAN_PROVIDER_HOLD") == "alpha" && kind == "plan" && in["question"] == "alpha"
		if held {
			releases[n] = hold
		}
		emit(providerEvent{Kind: "accepted", Ordinal: n, ClientID: r.Header.Get("X-Client-Request-Id"), Wire: string(raw), WireDigest: fmt.Sprintf("%x", sha256.Sum256(raw))})
		// The controlled provider has applied its deterministic computation and
		// fsynced the produced payload before withholding the HTTP response.
		applied, _ := json.Marshal(payload)
		emit(providerEvent{Kind: "applied", Ordinal: n, Response: string(applied)})
		mu.Unlock()
		if held {
			<-hold
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
