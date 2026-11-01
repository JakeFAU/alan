package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/fixture"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("alan-resource", flag.ContinueOnError)
	path := flags.String("db", "", "independent SQLite store")
	token := flags.String("token", "", "process-start identity")
	address := flags.String("listen", "127.0.0.1:0", "loopback listen address")
	journalPath := flags.String("journal", "", "optional passive ingress/result JSONL journal")
	responseMode := flags.String("response-mode", "normal", "normal, drop-after-commit, or finish-accepted")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *path == "" || *token == "" || flags.NArg() != 0 {
		return fmt.Errorf("db and token required")
	}
	if *responseMode != "normal" && *responseMode != "drop-after-commit" && *responseMode != "finish-accepted" {
		return fmt.Errorf("unsupported response mode")
	}
	var journal *os.File
	if *journalPath != "" {
		var err error
		journal, err = os.OpenFile(*journalPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer journal.Close()
	}
	var journalMu sync.Mutex
	var contactNumber int64
	observe := func(record map[string]any) error {
		if journal == nil {
			return nil
		}
		journalMu.Lock()
		defer journalMu.Unlock()
		record["PID"] = os.Getpid()
		record["Process"] = *token
		if err := json.NewEncoder(journal).Encode(record); err != nil {
			return err
		}
		return journal.Sync()
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("literal loopback address required")
	}
	ctx, err := boundary.FromEnvironment(context.Background(), *path)
	if err != nil {
		return err
	}
	resource, err := fixture.Open(ctx, *path, *token)
	if err != nil {
		return err
	}
	defer resource.Close()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	defer listener.Close()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var contact int64
		var body io.Reader
		if journal == nil {
			// Preserve the original uninstrumented protocol, including its limited
			// streaming decoder and immediate method/path rejection.
			if r.Method != http.MethodPost || r.URL.Path != "/apply" {
				http.Error(w, "unsupported protocol", http.StatusNotFound)
				return
			}
			defer r.Body.Close()
			body = io.LimitReader(r.Body, 1<<20)
		} else {
			journalMu.Lock()
			contactNumber++
			contact = contactNumber
			journalMu.Unlock()
			defer r.Body.Close()
			raw, readErr := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			if err := observe(map[string]any{"Event": "ingress", "Contact": contact, "Method": r.Method, "Path": r.URL.Path, "Body": string(raw)}); err != nil {
				http.Error(w, "journal unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Method != http.MethodPost || r.URL.Path != "/apply" {
				http.Error(w, "unsupported protocol", http.StatusNotFound)
				return
			}
			// The explicitly instrumented profile retains a complete bounded body.
			if readErr != nil || len(raw) > 1<<20 {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			body = bytes.NewReader(raw)
		}
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		var request fixture.Request
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		event := boundary.Event{Boundary: "resource.ingress", Phase: "pre-apply", Subject: request.Subject, Generation: request.Generation, Epoch: request.Epoch, Sequence: request.Sequence, Intent: request.Key, Attempt: request.Authorization}
		// In this opt-in world the accepted packet belongs to the endpoint.
		// Client death cannot withdraw it after validation.
		applyCtx := r.Context()
		if *responseMode == "finish-accepted" {
			applyCtx = ctx
		}
		if err := boundary.Reached(boundary.Inherit(applyCtx, ctx), event); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		receipt, err := resource.Apply(applyCtx, request)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, fixture.ErrConflict) {
				status = http.StatusConflict
			}
			observe(map[string]any{"Event": "rejected", "Contact": contact, "Status": status, "Error": err.Error()})
			http.Error(w, err.Error(), status)
			return
		}
		if err := observe(map[string]any{"Event": "committed", "Contact": contact, "Receipt": receipt}); err != nil {
			http.Error(w, "journal unavailable", http.StatusServiceUnavailable)
			return
		}
		event.Boundary = "resource.applied"
		event.Phase = "post-commit"
		if err = boundary.Reached(boundary.Inherit(r.Context(), ctx), event); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if *responseMode == "drop-after-commit" {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				connection.Close()
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipt)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"URL": "http://" + listener.Addr().String() + "/apply", "Token": *token, "Store": *path, "PID": os.Getpid()}); err != nil {
		return err
	}
	return server.Serve(listener)
}
