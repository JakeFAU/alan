package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/fixture"
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
	ledger := flags.Bool("receipt-ledger", false, "opt in to the version 2 exact-wire application ledger and receipt lookup")
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
	openResource := fixture.Open
	if *ledger {
		openResource = fixture.OpenLedger
	}
	resource, err := openResource(ctx, *path, *token)
	if err != nil {
		return err
	}
	defer resource.Close()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	defer listener.Close()
	applyPath := "/apply"
	lookupPath := ""
	if *ledger {
		prefix := "/phase2c/" + resource.LedgerID()
		applyPath = prefix + "/apply"
		lookupPath = prefix + "/lookup"
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var contact int64
		var body io.Reader
		var wire []byte
		if *ledger {
			defer r.Body.Close()
			if journal != nil {
				journalMu.Lock()
				contactNumber++
				contact = contactNumber
				journalMu.Unlock()
			}
			limit := receiptquery.MaxMutationBytes
			isQuery := r.RequestURI == lookupPath
			if isQuery {
				limit = receiptquery.MaxQueryBytes
			}
			var readErr error
			wire, readErr = io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
			if err := observe(map[string]any{"Event": "ingress", "Contact": contact, "Method": r.Method, "Path": r.RequestURI, "Body": string(wire)}); err != nil {
				http.Error(w, "journal unavailable", http.StatusServiceUnavailable)
				return
			}
			if r.Method != http.MethodPost || (!isQuery && r.RequestURI != applyPath) {
				http.Error(w, "unsupported protocol", http.StatusNotFound)
				return
			}
			if readErr != nil || len(wire) > limit {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if isQuery {
				serveLookup(w, r, ctx, resource, wire, contact, observe)
				return
			}
			if _, err := receiptquery.DecodeMutation(string(wire)); err != nil {
				http.Error(w, "bad mutation wire", http.StatusBadRequest)
				return
			}
			body = bytes.NewReader(wire)
		} else if journal == nil {
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
		var receipt fixture.Receipt
		var err error
		if *ledger {
			receipt, err = resource.ApplyWire(applyCtx, string(wire))
		} else {
			receipt, err = resource.Apply(applyCtx, request)
		}
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
		if *ledger {
			raw, _ := json.Marshal(receipt)
			w.Write(raw)
		} else {
			json.NewEncoder(w).Encode(receipt)
		}
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	base := "http://" + listener.Addr().String()
	ready := map[string]any{"URL": base + applyPath, "Token": *token, "Store": *path, "PID": os.Getpid()}
	if *ledger {
		ready["LookupURL"] = base + lookupPath
		ready["LedgerID"] = resource.LedgerID()
	}
	var stopCtx context.Context
	if *ledger {
		var stop context.CancelFunc
		stopCtx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
	}
	if err = json.NewEncoder(os.Stdout).Encode(ready); err != nil {
		return err
	}
	if !*ledger {
		return server.Serve(listener)
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	select {
	case err := <-served:
		return err
	case <-stopCtx.Done():
		drain, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(drain); err != nil {
			return err
		}
		err := <-served
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func serveLookup(w http.ResponseWriter, r *http.Request, ctx context.Context, resource *fixture.Resource, wire []byte, contact int64, observe func(map[string]any) error) {
	q, err := receiptquery.DecodeQuery(wire)
	if err != nil || q.LedgerID != resource.LedgerID() {
		http.Error(w, "bad query binding", http.StatusBadRequest)
		return
	}
	readCtx := boundary.Inherit(r.Context(), ctx)
	event := boundary.Event{Boundary: "resource.lookup", Phase: "pre-snapshot", Intent: q.QueryID, Attempt: q.QueryKey, Sequence: int64(q.Ordinal)}
	if err = boundary.Reached(readCtx, event); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	s, err := resource.Lookup(readCtx, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	raw, _ := json.Marshal(s)
	if err = observe(map[string]any{"Event": "snapshot", "Contact": contact, "Snapshot": s, "Response": string(raw)}); err != nil {
		http.Error(w, "journal unavailable", http.StatusServiceUnavailable)
		return
	}
	event.Phase = "post-snapshot"
	if err = boundary.Reached(readCtx, event); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw)
}
